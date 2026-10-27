package app

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Chat is the lead inbox (ported crm/inbox conversation model) plus the
// AI chatbot that answers leads on the website widget and WhatsApp.
type Chat struct {
	*Deps
	auto      *Automation
	limiter   *RateLimiter
	badTokens *RateLimiter
}

const maxChatBody = 2000

func NewChat(d *Deps, auto *Automation) *Chat {
	return &Chat{Deps: d, auto: auto, limiter: NewRateLimiter(d.Clock, time.Minute, 20), badTokens: NewRateLimiter(d.Clock, 10*time.Minute, 10)}
}

// DefaultChatbotSettings is used until an admin saves the settings row.
func DefaultChatbotSettings() domain.ChatbotSettings {
	return domain.ChatbotSettings{Enabled: true, BotName: "Wira", Persona: "Ramah, sigap, dan profesional.", ServicesOffered: "Social media management, SEO, ads management, produksi konten, website, branding.",
		PricingHints: "Harga menyesuaikan kebutuhan; tim sales akan mengirim penawaran resmi.", BusinessHoursStart: 9, BusinessHoursEnd: 18,
		HandoffRules: "Serahkan ke tim sales bila pengguna minta bicara dengan manusia, komplain, atau pertanyaan di luar layanan.", Greeting: "Halo! Saya Wira dari Mixist. Ada yang bisa saya bantu?", MinConfidence: 0.55}
}

func (c *Chat) settings(ctx context.Context) (domain.ChatbotSettings, error) {
	s, err := c.Store.GetChatbotSettings(ctx)
	if err != nil || s == nil {
		return DefaultChatbotSettings(), err
	}
	return *s, nil
}

func (c *Chat) GetSettings(ctx context.Context) (domain.ChatbotSettings, error) {
	return c.settings(ctx)
}

func (c *Chat) SaveSettings(ctx context.Context, actor Actor, s domain.ChatbotSettings) (domain.ChatbotSettings, error) {
	errs := map[string]string{}
	for field, v := range map[string]string{"botName": s.BotName, "persona": s.Persona, "servicesOffered": s.ServicesOffered, "pricingHints": s.PricingHints, "handoffRules": s.HandoffRules, "greeting": s.Greeting} {
		if domain.JSTrim(v) == "" {
			errs[field] = "Wajib diisi"
		}
	}
	if s.BusinessHoursStart < 0 || s.BusinessHoursStart > 23 || s.BusinessHoursEnd < 0 || s.BusinessHoursEnd > 23 {
		errs["businessHoursStart"] = "Jam kerja harus 0-23"
	}
	if s.MinConfidence < 0 || s.MinConfidence > 1 {
		errs["minConfidence"] = "Ambang keyakinan harus 0-1"
	}
	if err := fieldErrs(errs); err != nil {
		return s, err
	}
	s.UpdatedBy, s.UpdatedAt = actor.ref(), c.now()
	return s, c.Store.SaveChatbotSettings(ctx, s)
}

func (c *Chat) WhatsAppEnabled() bool { return c.WhatsApp != nil && c.WhatsApp.Enabled() }

// ─── inbox (internal) ────────────────────────────────────────

func (c *Chat) List(ctx context.Context, f domain.ConversationFilter) ([]domain.Conversation, error) {
	return c.Store.ListConversations(ctx, f)
}

// ConversationDetail is the inbox thread.
type ConversationDetail struct {
	Conversation domain.Conversation
	Messages     []domain.Message
	Notes        []domain.ConversationNote
	Insight      *domain.Insight
}

func (c *Chat) load(ctx context.Context, id string) (*domain.Conversation, error) {
	conv, err := c.Store.GetConversation(ctx, id)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, notFound(msgConvNotFound)
	}
	return conv, nil
}

func (c *Chat) Get(ctx context.Context, id string) (*ConversationDetail, error) {
	conv, err := c.load(ctx, id)
	if err != nil {
		return nil, err
	}
	msgs, err := c.Store.ListMessages(ctx, id, time.Time{}, 500)
	if err != nil {
		return nil, err
	}
	notes, err := c.Store.ListNotes(ctx, id)
	if err != nil {
		return nil, err
	}
	insight, err := c.Store.GetInsight(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ConversationDetail{Conversation: *conv, Messages: msgs, Notes: notes, Insight: insight}, nil
}

func (c *Chat) cleanBody(body string) (string, error) {
	body = domain.JSTrim(body)
	if body == "" {
		return "", apperr.Field("body", "Pesan tidak boleh kosong")
	}
	if len([]rune(body)) > maxChatBody {
		return "", apperr.Field("body", "Pesan maksimal 2000 karakter")
	}
	return body, nil
}

func preview(body string) *string {
	s := domain.SliceUTF16(strings.Join(strings.Fields(body), " "), 120)
	return &s
}

// deliver sends an outbound message on the conversation's channel and
// stores it with the delivery status.
func (c *Chat) deliver(ctx context.Context, conv *domain.Conversation, m domain.Message) (domain.Message, error) {
	m.ID, m.ConversationID, m.Direction, m.CreatedAt, m.Status = ids.New(), conv.ID, "out", c.now(), "sent"
	if conv.Channel == "whatsapp" {
		provider := "wa-gateway"
		m.Provider = &provider
		switch {
		case !c.WhatsAppEnabled():
			m.Status, m.ErrorReason = "failed", ptr("WA gateway belum dikonfigurasi")
		case conv.Phone == nil:
			m.Status, m.ErrorReason = "failed", ptr("nomor tujuan tidak tersedia")
		default:
			msgID, err := c.WhatsApp.Send(ctx, *conv.Phone, m.Body)
			if err != nil {
				m.Status, m.ErrorReason = "failed", ptr(domain.SliceUTF16(err.Error(), 300))
			} else if msgID != "" {
				m.ProviderMessageID = &msgID
			}
		}
	}
	_, err := c.Store.InsertMessage(ctx, m)
	return m, err
}

// Reply is an agent reply: it takes the conversation over from the bot
// and records the first response time.
func (c *Chat) Reply(ctx context.Context, actor Actor, id, body string) (*ConversationDetail, error) {
	conv, err := c.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if body, err = c.cleanBody(body); err != nil {
		return nil, err
	}
	if _, err := c.deliver(ctx, conv, domain.Message{Sender: "agent", Body: body, SentByUserID: actor.ref()}); err != nil {
		return nil, err
	}
	now := c.now()
	if conv.FirstResponseAt == nil && conv.AwaitingSince != nil {
		secs := int32(now.Sub(*conv.AwaitingSince).Seconds())
		conv.FirstResponseAt, conv.FirstResponseSeconds = &now, &secs
	}
	conv.BotActive, conv.AwaitingSince, conv.UnreadCount = false, nil, 0
	if conv.AssignedUserID == nil {
		conv.AssignedUserID = actor.ref()
	}
	if conv.Status == "open" || conv.Status == "resolved" {
		conv.Status = "in_progress"
	}
	conv.LastMessageAt, conv.LastMessagePreview, conv.UpdatedAt = &now, preview(body), now
	if err := c.Store.SaveConversation(ctx, conv); err != nil {
		return nil, err
	}
	return c.Get(ctx, id)
}

// Action applies an inbox action (assign_me, unassign, set_status,
// set_complaint, mark_read, takeover, release).
func (c *Chat) Action(ctx context.Context, actor Actor, id, action, status string, value bool) (*domain.Conversation, error) {
	conv, err := c.load(ctx, id)
	if err != nil {
		return nil, err
	}
	now := c.now()
	switch action {
	case "assign_me":
		conv.AssignedUserID = actor.ref()
	case "unassign":
		conv.AssignedUserID = nil
	case "set_status":
		if !domain.Contains(domain.ConversationStatuses, status) {
			return nil, apperr.Field("status", "Status tidak valid")
		}
		if status == "resolved" && conv.Status != "resolved" {
			conv.ResolvedAt = &now
			secs := int32(now.Sub(conv.CreatedAt).Seconds())
			conv.ResolutionSeconds, conv.AwaitingSince = &secs, nil
		}
		conv.Status = status
	case "set_complaint":
		conv.IsComplaint = value
		if value && (conv.Priority == "low" || conv.Priority == "normal") {
			conv.Priority = "high"
		}
	case "mark_read":
		conv.UnreadCount = 0
	case "takeover":
		conv.BotActive = false
		if conv.AssignedUserID == nil {
			conv.AssignedUserID = actor.ref()
		}
	case "release":
		conv.BotActive, conv.HandoffReason, conv.HandoffAt = true, nil, nil
	default:
		return nil, apperr.Field("action", "Aksi tidak dikenal")
	}
	conv.UpdatedAt = now
	return conv, c.Store.SaveConversation(ctx, conv)
}

func (c *Chat) AddNote(ctx context.Context, actor Actor, id, body string) (*domain.ConversationNote, error) {
	if _, err := c.load(ctx, id); err != nil {
		return nil, err
	}
	body, err := c.cleanBody(body)
	if err != nil {
		return nil, err
	}
	n := domain.ConversationNote{ID: ids.New(), ConversationID: id, AuthorUserID: actor.UserID, Body: body, CreatedAt: c.now()}
	return &n, c.Store.InsertNote(ctx, n)
}

// Analyze stores the AI insight of a conversation; an unchanged transcript
// returns the cached one ("cache"), no customer text returns "empty".
func (c *Chat) Analyze(ctx context.Context, id string) (*domain.Insight, string, error) {
	conv, err := c.load(ctx, id)
	if err != nil {
		return nil, "", err
	}
	msgs, err := c.Store.RecentMessages(ctx, id, domain.MaxTranscriptMessages)
	if err != nil {
		return nil, "", err
	}
	var transcript []domain.TranscriptMessage
	for _, m := range msgs {
		if m.Sender == "system" || domain.JSTrim(m.Body) == "" {
			continue
		}
		transcript = append(transcript, domain.TranscriptMessage{Direction: m.Direction, Body: domain.JSTrim(m.Body)})
	}
	if len(transcript) == 0 {
		return nil, "empty", nil
	}
	fp := domain.Fingerprint(transcript)
	stored, err := c.Store.GetInsight(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if stored != nil && !domain.NeedsAnalysis(&stored.Fingerprint, fp) {
		return stored, "cache", nil
	}
	res, err := c.AI.Complete(ctx, AIRequest{Messages: []AIMessage{{Role: "system", Content: domain.InsightSystemPrompt}, {Role: "user", Content: "Percakapan:\n" + domain.TranscriptText(transcript, domain.MaxTranscriptChars)}}, JSON: true, Temperature: 0, MaxTokens: 600})
	if err != nil {
		return nil, "", err
	}
	in := domain.MockInsight(transcript)
	in.Model = MockModel
	if !res.Mock {
		parsed := domain.ParseInsight(res.Content)
		if parsed == nil {
			return nil, "", apperr.FromProvider("openrouter", apperr.Unknown, "Jawaban AI tidak dapat dibaca. Coba lagi.", nil)
		}
		in = *parsed
		in.Model = res.Model
	}
	in.ConversationID, in.Fingerprint, in.AnalyzedAt = id, fp, c.now()
	if err := c.Store.SaveInsight(ctx, in); err != nil {
		return nil, "", err
	}
	if in.IsComplaint && !conv.IsComplaint {
		conv.IsComplaint, conv.UpdatedAt = true, c.now()
		if err := c.Store.SaveConversation(ctx, conv); err != nil {
			return nil, "", err
		}
	}
	return &in, "analyzed", nil
}

func (c *Chat) ListTemplates(ctx context.Context) ([]domain.ReplyTemplate, error) {
	return c.Store.ListReplyTemplates(ctx, false)
}

func (c *Chat) SaveTemplate(ctx context.Context, actor Actor, t domain.ReplyTemplate) (*domain.ReplyTemplate, error) {
	errs := map[string]string{}
	if domain.JSTrim(t.Title) == "" || len([]rune(t.Title)) > 80 {
		errs["title"] = "Judul wajib diisi (maks. 80 karakter)"
	}
	if domain.JSTrim(t.Body) == "" || len([]rune(t.Body)) > 2000 {
		errs["body"] = "Isi wajib diisi (maks. 2000 karakter)"
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	if t.ID == "" {
		t.ID, t.CreatedBy = ids.New(), actor.ref()
	}
	return &t, c.Store.SaveReplyTemplate(ctx, t)
}

func (c *Chat) DeleteTemplate(ctx context.Context, id string) error {
	return c.Store.DeleteReplyTemplate(ctx, id)
}

// ─── website widget (public) ─────────────────────────────────

// StartInput opens a widget conversation.
type StartInput struct {
	Name, Email, Phone, PageURL string
	Attribution                 domain.Attribution
}

// PublicChat is what the widget renders.
type PublicChat struct {
	Token         string
	BotName       string
	Messages      []domain.Message
	HumanTakeover bool
}

func (c *Chat) StartPublic(ctx context.Context, ip string, in StartInput) (*PublicChat, error) {
	if err := c.limiter.Check("chat-start:" + ip); err != nil {
		return nil, err
	}
	s, err := c.settings(ctx)
	if err != nil {
		return nil, err
	}
	now := c.now()
	token := NewToken()
	utm, _ := json.Marshal(in.Attribution)
	conv := &domain.Conversation{ID: ids.New(), Channel: "web", ExternalID: token, VisitorToken: &token, DisplayName: domain.TrimmedOrNil(domain.SliceUTF16(in.Name, 100)), Status: "open", Priority: "normal", BotActive: s.Enabled, PageURL: domain.TrimmedOrNil(domain.SliceUTF16(in.PageURL, 500)), Utm: utm, CreatedAt: now, UpdatedAt: now}
	if e := strings.ToLower(domain.JSTrim(in.Email)); domain.IsEmail(e) {
		conv.Email = &e
	}
	if p := domain.NormalizePhone(in.Phone); domain.IsValidNormalizedPhone(p) {
		conv.Phone = &p
	}
	if err := c.Store.SaveConversation(ctx, conv); err != nil {
		return nil, err
	}
	greeting := domain.Message{ID: ids.New(), ConversationID: conv.ID, Direction: "out", Sender: "bot", Body: s.Greeting, Status: "sent", CreatedAt: now}
	if !s.Enabled {
		greeting.Sender, greeting.Body = "system", "Terima kasih sudah menghubungi Mixist. Tinggalkan pesan dan tim kami akan membalas di sini."
	}
	if _, err := c.Store.InsertMessage(ctx, greeting); err != nil {
		return nil, err
	}
	conv.LastMessageAt, conv.LastMessagePreview = &now, preview(greeting.Body)
	if err := c.Store.SaveConversation(ctx, conv); err != nil {
		return nil, err
	}
	return &PublicChat{Token: token, BotName: s.BotName, Messages: []domain.Message{greeting}}, nil
}

func (c *Chat) byToken(ctx context.Context, token string) (*domain.Conversation, error) {
	if len(token) < 32 {
		return nil, notFound(msgLinkInvalid)
	}
	conv, err := c.Store.GetConversationByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, notFound(msgLinkInvalid)
	}
	return conv, nil
}

// SendPublic stores the visitor's message and, while the bot is on,
// answers it. The response carries every message after the visitor's.
func (c *Chat) SendPublic(ctx context.Context, ip, token, body string) (*PublicChat, error) {
	if err := c.limiter.Check("chat-send:" + ip); err != nil {
		return nil, err
	}
	conv, err := c.byToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if body, err = c.cleanBody(body); err != nil {
		return nil, err
	}
	m := domain.Message{ID: ids.New(), ConversationID: conv.ID, Direction: "in", Sender: "customer", Body: body, Status: "received", CreatedAt: c.now()}
	if _, err := c.Store.InsertMessage(ctx, m); err != nil {
		return nil, err
	}
	if err := c.inbound(ctx, conv, body); err != nil {
		return nil, err
	}
	return c.poll(ctx, conv.ID, token, m.CreatedAt.Add(-time.Microsecond))
}

func (c *Chat) PollPublic(ctx context.Context, ip, token string, after time.Time) (*PublicChat, error) {
	if err := c.limiter.Check("chat-poll:" + ip); err != nil {
		return nil, err
	}
	conv, err := c.byToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return c.poll(ctx, conv.ID, token, after)
}

func (c *Chat) poll(ctx context.Context, convID, token string, after time.Time) (*PublicChat, error) {
	conv, err := c.load(ctx, convID)
	if err != nil {
		return nil, err
	}
	msgs, err := c.Store.ListMessages(ctx, convID, after, 200)
	if err != nil {
		return nil, err
	}
	s, err := c.settings(ctx)
	if err != nil {
		return nil, err
	}
	return &PublicChat{Token: token, BotName: s.BotName, Messages: msgs, HumanTakeover: !conv.BotActive}, nil
}

// ─── WhatsApp gateway inbound (public) ───────────────────────

// WAInbound is one message of the wa-gateway forward payload.
type WAInbound struct {
	RemoteJid, MessageID, Text, MediaType, PushName string
	FromMe                                          bool
	Timestamp                                       float64
}

// jidToPhone is jidToPhone: the digits before "@" (and ":device"), 10-15
// of them, else "".
func jidToPhone(jid string) string {
	if !strings.HasSuffix(jid, "@s.whatsapp.net") {
		return ""
	}
	user, _, _ := strings.Cut(jid, "@")
	user, _, _ = strings.Cut(user, ":")
	var b strings.Builder
	for _, r := range user {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	if d := b.String(); len(d) >= 10 && len(d) <= 15 {
		return d
	}
	return ""
}

// ReceiveWhatsApp is POST /api/wa/inbound: token checked in constant time
// with a brute-force limiter, personal chats stored (our own messages too),
// and customer messages answered by the bot.
func (c *Chat) ReceiveWhatsApp(ctx context.Context, ip, token string, msgs []WAInbound) (stored, skipped int, err error) {
	if !c.WhatsAppEnabled() {
		return 0, 0, notFound("Integrasi WhatsApp belum diaktifkan")
	}
	if err := c.badTokens.Check("wa:" + ip); err != nil {
		return 0, 0, err
	}
	if !c.WhatsApp.VerifyToken(token) {
		return 0, 0, apperr.New(apperr.Unauthenticated, "Token gateway tidak valid")
	}
	for _, in := range msgs {
		phone := jidToPhone(in.RemoteJid)
		text := domain.SliceUTF16(domain.JSTrim(in.Text), 4000)
		if text == "" && in.MediaType != "" {
			text = "[" + in.MediaType + "]"
		}
		if phone == "" || text == "" {
			skipped++
			continue
		}
		ok, err := c.receiveOne(ctx, phone, text, in)
		if err != nil {
			return stored, skipped, err
		}
		if ok {
			stored++
		} else {
			skipped++
		}
	}
	return stored, skipped, nil
}

func (c *Chat) receiveOne(ctx context.Context, phone, text string, in WAInbound) (bool, error) {
	conv, err := c.Store.GetConversationByExternal(ctx, "whatsapp", phone)
	if err != nil {
		return false, err
	}
	now := c.now()
	if conv == nil {
		s, err := c.settings(ctx)
		if err != nil {
			return false, err
		}
		conv = &domain.Conversation{ID: ids.New(), Channel: "whatsapp", ExternalID: phone, Phone: &phone, DisplayName: domain.TrimmedOrNil(in.PushName), Status: "open", Priority: "normal", BotActive: s.Enabled, Utm: json.RawMessage("{}"), CreatedAt: now, UpdatedAt: now}
		if err := c.Store.SaveConversation(ctx, conv); err != nil {
			return false, err
		}
	}
	at := now
	if in.Timestamp > 0 {
		at = time.UnixMilli(int64(in.Timestamp * 1000)).UTC()
	}
	provider := "wa-gateway"
	m := domain.Message{ID: ids.New(), ConversationID: conv.ID, Direction: "in", Sender: "customer", Body: text, Status: "received", Provider: &provider, ProviderMessageID: domain.TrimmedOrNil(in.MessageID), CreatedAt: at}
	if in.FromMe {
		// Sent from the business phone itself: an agent message.
		m.Direction, m.Sender, m.Status = "out", "agent", "sent"
	}
	inserted, err := c.Store.InsertMessage(ctx, m)
	if err != nil || !inserted {
		return false, err
	}
	if in.FromMe {
		conv.AwaitingSince, conv.LastMessageAt, conv.LastMessagePreview, conv.UpdatedAt = nil, &at, preview(text), now
		return true, c.Store.SaveConversation(ctx, conv)
	}
	return true, c.inbound(ctx, conv, text)
}

// inbound applies the CS rules to a customer message (awaiting_since,
// unread count, reopening) and lets the bot answer.
func (c *Chat) inbound(ctx context.Context, conv *domain.Conversation, body string) error {
	now := c.now()
	if conv.AwaitingSince == nil {
		conv.AwaitingSince = &now
	}
	if conv.Status == "resolved" || conv.Status == "waiting_customer" {
		conv.Status, conv.ResolvedAt = "open", nil
	}
	conv.UnreadCount++
	conv.LastMessageAt, conv.LastMessagePreview, conv.UpdatedAt = &now, preview(body), now
	if err := c.Store.SaveConversation(ctx, conv); err != nil {
		return err
	}
	if conv.LeadID != nil {
		c.auto.Signal(ctx, *conv.LeadID, SignalChatInbound, conv.Channel)
	}
	if !conv.BotActive {
		return nil
	}
	return c.botRespond(ctx, conv, body)
}

// botRespond answers with OpenRouter (or the mock), extracts lead fields
// into a lead, guards prices, and hands off to a human when asked, on a
// complaint, or when confidence is low.
func (c *Chat) botRespond(ctx context.Context, conv *domain.Conversation, last string) error {
	s, err := c.settings(ctx)
	if err != nil {
		return err
	}
	if !s.Enabled {
		return nil
	}
	known := domain.LeadExtract{Name: deref(conv.DisplayName), Phone: deref(conv.Phone), Email: deref(conv.Email)}
	if conv.LeadID != nil {
		if lead, err := c.Store.GetLead(ctx, *conv.LeadID); err == nil && lead != nil {
			known = known.Merge(domain.LeadExtract{Name: lead.PicName, Company: lead.OrgName, Phone: deref(lead.PicPhone), Email: deref(lead.PicEmail)})
		}
	}
	recent, err := c.Store.RecentMessages(ctx, conv.ID, 20)
	if err != nil {
		return err
	}
	for _, m := range recent {
		if m.Sender == "customer" && m.Body != last {
			known = known.Merge(domain.ExtractLeadFields(m.Body))
		}
	}
	now := c.now()
	reply, err := c.botReply(ctx, recent, s, known, last, now)
	if err != nil {
		return err
	}
	if forced, reason := domain.DetectHandoff(last); forced && !reply.Handoff {
		reply.Handoff, reply.HandoffReason, reply.Reply = true, reason, domain.HandoffReply(reason, s, now)
	}
	if !domain.PricesWithinHints(reply.Reply, s.PricingHints) {
		reply.Reply = domain.PriceGuardReply
	}
	if !reply.Handoff && reply.Confidence < s.MinConfidence {
		reply.Handoff, reply.HandoffReason, reply.Reply = true, domain.HandoffLowConfidence, domain.HandoffReply(domain.HandoffLowConfidence, s, now)
	}
	extract := domain.ExtractLeadFields(last).Merge(reply.Lead)
	if err := c.upsertLead(ctx, conv, known.Merge(extract)); err != nil {
		c.warn("chatbot lead upsert failed", "conversation", conv.ID, "err", err)
	}
	if _, err := c.deliver(ctx, conv, domain.Message{Sender: "bot", Body: domain.SliceUTF16(reply.Reply, 2000)}); err != nil {
		return err
	}
	conv.LastMessageAt, conv.LastMessagePreview, conv.UpdatedAt = &now, preview(reply.Reply), now
	if reply.Handoff {
		conv.BotActive, conv.HandoffReason, conv.HandoffAt, conv.AwaitingSince = false, &reply.HandoffReason, &now, &now
		if reply.HandoffReason == domain.HandoffComplaint {
			conv.IsComplaint, conv.Priority = true, "high"
		}
	} else {
		conv.AwaitingSince = nil
	}
	if err := c.Store.SaveConversation(ctx, conv); err != nil {
		return err
	}
	if reply.Handoff {
		c.notifyHandoff(ctx, conv, s, reply.HandoffReason, last)
	}
	return nil
}

func (c *Chat) botReply(ctx context.Context, recent []domain.Message, s domain.ChatbotSettings, known domain.LeadExtract, last string, now time.Time) (domain.BotReply, error) {
	turns := make([]domain.ChatTurn, len(recent))
	for i, m := range recent {
		turns[i] = domain.ChatTurn{Sender: m.Sender, Body: m.Body}
	}
	knownJSON, _ := json.Marshal(known)
	res, err := c.AI.Complete(ctx, AIRequest{Messages: []AIMessage{
		{Role: "system", Content: domain.BotSystemPrompt(s, now)},
		{Role: "user", Content: "Riwayat percakapan:\n" + domain.BotHistory(turns, 20) + "\n\nData lead yang sudah diketahui: " + string(knownJSON) + "\n\nBalas pesan terakhir pengguna."},
	}, JSON: true, Temperature: 0.3, MaxTokens: 600})
	if err != nil {
		c.warn("chatbot completion failed", "err", err)
		return domain.BotReply{Reply: domain.HandoffReply(domain.HandoffLowConfidence, s, now), Handoff: true, HandoffReason: domain.HandoffLowConfidence}, nil
	}
	if res.Mock {
		return domain.MockBotReply(s, known, last, now), nil
	}
	return domain.ParseBotReply(res.Content), nil
}

func (c *Chat) notifyHandoff(ctx context.Context, conv *domain.Conversation, s domain.ChatbotSettings, reason, last string) {
	recipients := append([]string(nil), s.HandoffUserIDs...)
	if conv.AssignedUserID != nil {
		recipients = append(recipients, *conv.AssignedUserID)
	} else if conv.LeadID != nil {
		if lead, err := c.Store.GetLead(ctx, *conv.LeadID); err == nil && lead != nil && lead.OwnerUserID != nil {
			recipients = append(recipients, *lead.OwnerUserID)
		}
	}
	labels := map[string]string{domain.HandoffAskedHuman: "minta bicara dengan tim", domain.HandoffComplaint: "komplain", domain.HandoffLowConfidence: "bot kurang yakin"}
	who := firstNonEmpty(deref(conv.DisplayName), deref(conv.Phone), deref(conv.Email), "Pengunjung")
	c.notify(ctx, Notification{UserIDs: dedupe(recipients), Type: NotifChatHandoff, Title: "Chat perlu dibalas: " + who + " (" + labels[reason] + ")", Body: domain.SliceUTF16(last, 300), Href: "/erp/sales/inbox?conversation=" + conv.ID})
}

func dedupe(ids []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, id := range ids {
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// upsertLead creates or updates the conversation's lead (source chatbot)
// once a phone or email is known; existing values are never overwritten.
func (c *Chat) upsertLead(ctx context.Context, conv *domain.Conversation, l domain.LeadExtract) error {
	if l.Phone == "" && l.Email == "" {
		return nil
	}
	now := c.now()
	var lead *domain.Lead
	var err error
	if conv.LeadID != nil {
		lead, err = c.Store.GetLead(ctx, *conv.LeadID)
	} else {
		lead, err = c.Store.FindLeadByContact(ctx, l.Phone, l.Email)
	}
	if err != nil {
		return err
	}
	notes := leadNotes(l)
	created := lead == nil
	if created {
		org := firstNonEmpty(l.Company, l.Name, "Pengunjung chat")
		lead = &domain.Lead{ID: ids.New(), OrgName: org, OrgType: "lainnya", PicName: firstNonEmpty(l.Name, "Belum diketahui"), Source: "chatbot", Temperature: leadTemperature(l), Status: "baru", Custom: json.RawMessage("{}"), Audit: domain.Audit{CreatedAt: now}}
		if utm := conv.Utm; len(utm) > 2 {
			_ = json.Unmarshal(utm, &lead.Attribution)
		}
		if conv.Channel == "web" && conv.PageURL != nil && lead.LandingPage == nil {
			lead.LandingPage = conv.PageURL
		}
	} else {
		if l.Company != "" && (lead.OrgName == "Pengunjung chat" || lead.OrgName == lead.PicName) {
			lead.OrgName = l.Company
		}
		if l.Name != "" && lead.PicName == "Belum diketahui" {
			lead.PicName = l.Name
		}
		if t := leadTemperature(l); temperatureRank(t) > temperatureRank(lead.Temperature) {
			lead.Temperature = t
		}
	}
	if lead.PicPhone == nil && l.Phone != "" {
		lead.PicPhone = &l.Phone
	}
	if lead.PicEmail == nil && l.Email != "" {
		lead.PicEmail = &l.Email
	}
	if notes != "" && !strings.Contains(deref(lead.Notes), notes) {
		lead.Notes = ptr(domain.JSTrim(deref(lead.Notes) + "\n" + notes))
	}
	lead.UpdatedAt = now
	if err := c.Store.SaveLead(ctx, lead); err != nil {
		return err
	}
	if conv.LeadID == nil {
		conv.LeadID = &lead.ID
	}
	if created {
		c.auto.Emit(ctx, Event{Type: "lead.created", SubjectType: "lead", SubjectID: lead.ID, Payload: map[string]any{"channel": conv.Channel}})
	}
	return nil
}

func leadNotes(l domain.LeadExtract) string {
	var parts []string
	for _, p := range [][2]string{{"Kebutuhan", l.Need}, {"Budget", l.Budget}, {"Timeline", l.Timeline}} {
		if p[1] != "" {
			parts = append(parts, p[0]+": "+p[1])
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "[Chatbot] " + strings.Join(parts, "; ")
}

func leadTemperature(l domain.LeadExtract) string {
	switch {
	case l.Need != "" && l.Budget != "" && l.Timeline != "":
		return "panas"
	case l.Need != "":
		return "hangat"
	}
	return "dingin"
}

func temperatureRank(t string) int {
	return map[string]int{"dingin": 1, "hangat": 2, "panas": 3}[t]
}

// FlagSLABreaches marks conversations waiting on the team past the SLA
// (cs-sla-watcher) and notifies the assignee.
func (c *Chat) FlagSLABreaches(ctx context.Context) (int, error) {
	convs, err := c.Store.MarkSLABreaches(ctx, c.now().Add(-domain.SLAResponseMinutes*time.Minute))
	if err != nil {
		return 0, err
	}
	s, err := c.settings(ctx)
	if err != nil {
		return 0, err
	}
	for _, conv := range convs {
		to := append([]string(nil), s.HandoffUserIDs...)
		if conv.AssignedUserID != nil {
			to = append(to, *conv.AssignedUserID)
		}
		c.notify(ctx, Notification{UserIDs: dedupe(to), Type: NotifChatHandoff, Title: "SLA terlewati: " + firstNonEmpty(deref(conv.DisplayName), deref(conv.Phone), "Pengunjung"), Body: "Belum dibalas lebih dari 30 menit.", Href: "/erp/sales/inbox?conversation=" + conv.ID})
	}
	return len(convs), nil
}

// VerifyGatewayToken compares tokens in constant time (used by the WA
// provider).
func VerifyGatewayToken(expected, given string) bool {
	return expected != "" && subtle.ConstantTimeCompare([]byte(expected), []byte(given)) == 1
}
