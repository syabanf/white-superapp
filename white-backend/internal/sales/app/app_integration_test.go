package app_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/render"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/salesevents"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── fakes ───────────────────────────────────────────────────

type recorder struct {
	mu     sync.Mutex
	notifs []app.Notification
	events map[string][]any
}

func (r *recorder) Notify(_ context.Context, n app.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifs = append(r.notifs, n)
	return nil
}

func (r *recorder) Publish(_ context.Context, subject string, payload any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.events == nil {
		r.events = map[string][]any{}
	}
	r.events[subject] = append(r.events[subject], payload)
	return nil
}

func (r *recorder) notified(typ string) []app.Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []app.Notification
	for _, n := range r.notifs {
		if n.Type == typ {
			out = append(out, n)
		}
	}
	return out
}

type directory struct{}

func (directory) ListUsers(context.Context) ([]app.User, error) {
	return []app.User{{ID: salesID, Name: "Sari Sales"}, {ID: managerID, Name: "Maya Manajer"}, {ID: adminID, Name: "Admin"}}, nil
}

func (directory) Approvers(context.Context) ([]app.User, error) {
	return []app.User{{ID: managerID, Name: "Maya Manajer"}, {ID: adminID, Name: "Admin", Admin: true}}, nil
}

type clients struct{ created []app.ClientInput }

func (c *clients) CreateClient(_ context.Context, in app.ClientInput) (string, error) {
	c.created = append(c.created, in)
	return "client-" + in.Slug, nil
}

type mockAI struct{}

func (mockAI) Complete(context.Context, app.AIRequest) (app.AIResult, error) {
	return app.AIResult{Model: app.MockModel, Mock: true}, nil
}

type whatsapp struct {
	enabled bool
	sent    []string
}

func (w *whatsapp) Enabled() bool                 { return w.enabled }
func (w *whatsapp) VerifyToken(token string) bool { return app.VerifyGatewayToken("gw-secret", token) }
func (w *whatsapp) Send(_ context.Context, phone, text string) (string, error) {
	w.sent = append(w.sent, phone+": "+text)
	return "wamid-1", nil
}

const (
	salesID   = "user-sales"
	managerID = "user-manager"
	adminID   = "user-admin"
)

var (
	salesActor   = app.Actor{UserID: salesID}
	managerActor = app.Actor{UserID: managerID, Approver: true}
	adminActor   = app.Actor{UserID: adminID, Admin: true, Approver: true}
)

type env struct {
	svc   grpcapi.Services
	deps  *app.Deps
	rec   *recorder
	wa    *whatsapp
	cl    *clients
	jobs  *app.Jobs
	store domain.Store
}

func setup(t *testing.T) *env {
	t.Helper()
	pool := pg.TestPool(t, sales.Migrations(), "migrations", sales.Schema)
	pg.TruncateSchema(t, pool, sales.Schema)
	ctx := context.Background()
	if err := seed.Reference(ctx, pool); err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	wa := &whatsapp{}
	cl := &clients{}
	deps := &app.Deps{Store: postgres.NewStore(pool), Clock: clock.Real{}, Events: rec, Notifier: rec, Directory: directory{}, Clients: cl, AI: mockAI{}, WhatsApp: wa, Files: storage.NewLocal(t.TempDir()), Renderer: render.New()}
	svc := sales.Services(deps)
	return &env{svc: svc, deps: deps, rec: rec, wa: wa, cl: cl, store: deps.Store, jobs: app.NewJobs(deps, svc.Automation, svc.Pipeline, svc.Chat, svc.Onboarding)}
}

// must panics on error; the test fails with the stack trace.
func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func code(t *testing.T, err error, want apperr.Code) {
	t.Helper()
	if !apperr.Is(err, want) {
		t.Fatalf("want %s, got %v", want, err)
	}
}

func stage(t *testing.T, e *env, code string) string {
	pls := must(e.store.ListPipelines(context.Background()))
	for _, p := range pls {
		for _, s := range p.Stages {
			if s.Code == code {
				return s.ID
			}
		}
	}
	t.Fatalf("stage %s", code)
	return ""
}

func newDeal(t *testing.T, e *env) (*domain.Lead, *domain.Deal) {
	ctx := context.Background()
	lead := must(e.svc.Leads.CreateLead(ctx, salesActor, app.LeadInput{OrgName: "PT Maju Jaya", PicName: "Andi", PicPhone: "0812 3456 7890", PicEmail: "andi@maju.id", Temperature: "panas", Source: "referral", Notes: "Butuh SEO untuk 3 kota."}))
	acc := must(e.svc.Leads.CreateAccount(ctx, salesActor, app.AccountInput{Name: "PT Maju Jaya", Address: "Jl. Sudirman 1", City: "Jakarta", Npwp: "01.234.567.8-901.000"}))
	deal := must(e.svc.Pipeline.CreateDeal(ctx, salesActor, app.DealInput{LeadID: lead.ID, AccountID: acc.ID, Title: "SEO PT Maju", ServiceLines: []string{domain.LineSEO, domain.LineSocialMedia}, ValueEstimate: "50000000", Notes: "Fokus kata kunci properti. Laporan bulanan."}))
	return lead, deal
}

// ─── tests ───────────────────────────────────────────────────

func TestLeadsWorkflowAndScoring(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	_, err := e.svc.Leads.CreateLead(ctx, salesActor, app.LeadInput{OrgName: "X", PicName: "Y"})
	code(t, err, apperr.Validation)
	lead, _ := newDeal(t, e)
	got := must(e.store.GetLead(ctx, lead.ID))
	if got.OwnerUserID == nil || *got.OwnerUserID != salesID || *got.PicPhone != "6281234567890" {
		t.Fatalf("lead %+v", got)
	}
	// panas 30 + referral 20 + email 5 (seeded rules).
	if got.Score != 55 {
		t.Fatalf("score %d %+v", got.Score, got.ScoreBreakdown)
	}
	tasks := must(e.store.ListActivities(ctx, domain.ActivityFilter{LeadID: lead.ID}))
	if len(tasks) != 1 || !strings.Contains(tasks[0].Title, "Andi") || *tasks[0].OwnerUserID != salesID {
		t.Fatalf("workflow task %+v", tasks)
	}
	// Finishing a meeting task adds the event points (15).
	mt := must(e.svc.Pipeline.CreateActivity(ctx, salesActor, app.ActivityInput{LeadID: lead.ID, ActivityType: "meeting", Title: "Discovery call"}))
	must(e.svc.Pipeline.UpdateActivity(ctx, salesActor, app.ActivityInput{ID: mt.ID, ActivityType: "meeting", Title: "Discovery call", Status: "done"}))
	if got := must(e.store.GetLead(ctx, lead.ID)); got.Score != 70 {
		t.Fatalf("score after meeting %d", got.Score)
	}
	res := must(e.svc.Leads.ImportLeads(ctx, salesActor, "leads.csv", []byte("nama_instansi,nama_pic,wa_pic,email\nPT Baru,Budi,081299990000,budi@baru.id\nPT Maju Jaya,Andi,081234567890,\nCV X,Cici,12,\n")))
	if res.Imported != 1 || res.Skipped != 2 {
		t.Fatalf("import %+v", res)
	}
}

func TestQuotationContractOnboardingFlow(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	_, deal := newDeal(t, e)
	must(e.svc.Pipeline.MoveDealStage(ctx, salesActor, deal.ID, stage(t, e, "proposal"), "", ""))
	if len(e.rec.notified(app.NotifWorkflow)) != 1 {
		t.Fatal("proposal workflow notifies the owner")
	}
	draft := must(e.svc.Quotes.GenerateDraft(ctx, salesActor, deal.ID, app.DraftOptions{Tier: "standard", RetainerMonths: 6, UseAI: true}))
	q := draft.Quotation
	if !strings.HasPrefix(q.QuoteNumber, "QT-") || len(q.Items) != 2 || q.DiscountPercent != 5 || q.ApprovalStatus != domain.ApprovalNone || deref(q.GeneratedBy) != "ai" || draft.Model != app.MockModel {
		t.Fatalf("draft %+v", q)
	}
	if !strings.Contains(q.Items[0].Scope, "Fokus kata kunci properti") {
		t.Fatalf("mock scope %q", q.Items[0].Scope)
	}
	if _, data, err := e.svc.Quotes.RenderPDF(ctx, q.ID); err != nil || !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatal("pdf", err)
	}
	must(e.svc.Quotes.SetStatus(ctx, salesActor, q.ID, domain.QuoteSent))
	_, err := e.svc.Quotes.UpdateQuotation(ctx, salesActor, app.QuotationInput{ID: q.ID})
	code(t, err, apperr.Conflict)
	must(e.svc.Quotes.SetStatus(ctx, salesActor, q.ID, domain.QuoteAccepted))
	_, err = e.svc.Pipeline.MoveDealStage(ctx, salesActor, deal.ID, stage(t, e, "menang"), "", "")
	code(t, err, apperr.Validation)
	won := must(e.svc.Pipeline.MoveDealStage(ctx, salesActor, deal.ID, stage(t, e, "menang"), q.Total.String(), ""))
	if !won.IsWon || won.ForecastCategory != "closed_won" || len(e.rec.events[events.SalesDealWon]) != 1 {
		t.Fatalf("won %+v", won)
	}
	ob := must(e.store.GetOnboardingByDeal(ctx, deal.ID))
	if ob == nil || len(ob.Items) != 8 || ob.Progress != 0 {
		t.Fatalf("onboarding %+v", ob)
	}

	// Invoices per term, finance issues the first, a payment follows.
	invs := must(e.svc.Billing.SubmitFromTerms(ctx, salesActor, q.ID))
	if len(invs) != 6 || invs[0].Status != app.InvoiceSubmitted {
		t.Fatalf("invoices %d", len(invs))
	}
	again := must(e.svc.Billing.SubmitFromTerms(ctx, salesActor, q.ID))
	if len(again) != 6 {
		t.Fatal("terms are billed once")
	}
	first := invs[len(invs)-1]
	issued := must(e.svc.Billing.IssueInvoice(ctx, adminActor, first.ID))
	if issued.Status != app.InvoiceSent {
		t.Fatal("issued")
	}
	p := e.rec.events[events.SalesInvoiceIssued][0].(salesevents.InvoiceIssued)
	sub, tax, total := domain.MustMoney(p.Subtotal), domain.MustMoney(p.TaxAmount), domain.MustMoney(p.TotalAmount)
	if sub+tax != total || total != first.Amount || tax == 0 || p.CustomerName != "PT Maju Jaya" || p.InvoiceID != first.ID {
		t.Fatalf("invoice event %+v", p)
	}
	raw, _ := json.Marshal(p)
	if !strings.Contains(string(raw), `"totalAmount"`) || !strings.Contains(string(raw), `"customerName"`) {
		t.Fatalf("finance field names: %s", raw)
	}
	must(e.svc.Billing.RecordPayment(ctx, salesActor, app.PaymentInput{DealID: deal.ID, InvoiceID: first.ID, Amount: first.Amount.String(), Method: "transfer"}))
	pay := e.rec.events[events.SalesPaymentRecorded][0].(salesevents.PaymentRecorded)
	if pay.InvoiceID != first.ID || pay.Amount != first.Amount.String() {
		t.Fatalf("payment event %+v", pay)
	}
	if got := must(e.svc.Billing.GetInvoice(ctx, first.ID)); got.InvoiceNumber != first.InvoiceNumber || got.PaidTotal != first.Amount {
		t.Fatalf("get invoice %+v", got)
	}
	_, err = e.svc.Billing.GetInvoice(ctx, "missing")
	code(t, err, apperr.NotFound)
	if _, data, err := e.svc.Billing.RenderInvoicePDF(ctx, first.ID); err != nil || !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatal("invoice pdf", err)
	}
	_, err = e.svc.Billing.CancelInvoice(ctx, adminActor, first.ID, "salah")
	code(t, err, apperr.Conflict)
	detail := must(e.svc.Pipeline.GetDeal(ctx, deal.ID))
	if detail.PaymentStatus != "sebagian" || detail.TermProgress[0].Status != "lunas" {
		t.Fatalf("detail %+v", detail.TermProgress)
	}

	// Contract: generate, send, sign publicly.
	ct := must(e.svc.Contracts.Generate(ctx, salesActor, app.GenerateInput{QuotationID: q.ID, TemplateID: "ct-retainer"}))
	if !strings.HasPrefix(ct.ContractNumber, "KTR-") || strings.Contains(ct.Body, "{{") || !strings.Contains(ct.Body, "PT Maju Jaya") || domain.FormatDay(ct.EndDate) == "" {
		t.Fatalf("contract %s\n%s", ct.ContractNumber, ct.Body)
	}
	_, err = e.svc.Contracts.PublicGet(ctx, ct.SignToken, "1.1.1.1")
	code(t, err, apperr.NotFound)
	must(e.svc.Contracts.Send(ctx, salesActor, ct.ID))
	_, err = e.svc.Contracts.Sign(ctx, app.SignInput{Token: ct.SignToken, SignerName: "Andi", SignaturePNG: "data:image/png;base64,AAAA", Agree: true, IP: "1.1.1.1"})
	code(t, err, apperr.Validation)
	signed := must(e.svc.Contracts.Sign(ctx, app.SignInput{Token: ct.SignToken, SignerName: "Andi Wijaya", SignaturePNG: signatureURL(t), Agree: true, IP: "1.1.1.1", UserAgent: "test"}))
	if signed.Status != domain.ContractSigned || deref(signed.SignerIP) != "1.1.1.1" || len(e.rec.notified(app.NotifContractSigned)) != 1 {
		t.Fatalf("signed %+v", signed.Status)
	}
	if _, data, err := e.svc.Contracts.RenderPDF(ctx, "", ct.SignToken); err != nil || !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Fatal("signed pdf", err)
	}

	// Onboarding: contract and first invoice steps are done, portal works.
	ob = must(e.store.GetOnboardingByDeal(ctx, deal.ID))
	done := map[string]bool{}
	for _, it := range ob.Items {
		done[it.Key] = it.Status == domain.ItemDone
	}
	if !done[domain.KeySignContract] || !done[domain.KeyFirstInvoice] || ob.Progress != 25 {
		t.Fatalf("onboarding items %+v progress %d", done, ob.Progress)
	}
	_, err = e.svc.Onboarding.SubmitBrief(ctx, ob.PortalToken, "2.2.2.2", map[string]string{"brand_name": "Maju"})
	code(t, err, apperr.Validation)
	must(e.svc.Onboarding.SubmitBrief(ctx, ob.PortalToken, "2.2.2.2", map[string]string{"brand_name": "Maju", "brand_description": "Properti", "goals": "Leads", "target_audience": "Keluarga muda", "website": "https://maju.id"}))
	_, err = e.svc.Onboarding.UploadAsset(ctx, ob.PortalToken, "2.2.2.2", "virus.exe", []byte("MZ\x90"))
	code(t, err, apperr.Validation)
	ob = must(e.svc.Onboarding.UploadAsset(ctx, ob.PortalToken, "2.2.2.2", "logo.png", pngBytes(t)))
	if len(ob.Assets) != 1 || ob.Progress != 37 {
		t.Fatalf("assets %d progress %d", len(ob.Assets), ob.Progress)
	}
	if a, data, err := e.svc.Onboarding.Asset(ctx, ob.Assets[0].ID); err != nil || a.ContentType != "image/png" || len(data) == 0 {
		t.Fatal("asset", err)
	}
	ob = must(e.svc.Onboarding.CreateMarketingClient(ctx, adminActor, ob.ID, app.ClientInput{}))
	if deref(ob.MarketingClientID) != "client-pt-maju-jaya" || e.cl.created[0].WebsiteURL != "https://maju.id" {
		t.Fatalf("marketing client %+v", e.cl.created)
	}
	if _, err := e.svc.Onboarding.CreateMarketingClient(ctx, adminActor, ob.ID, app.ClientInput{}); !apperr.Is(err, apperr.Conflict) {
		t.Fatal("client created once")
	}
}

func TestDiscountApprovalChain(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	_, deal := newDeal(t, e)
	q := must(e.svc.Quotes.CreateQuotation(ctx, salesActor, app.QuotationInput{DealID: deal.ID, UsePpn: true, PpnPercent: 11, DiscountPercent: 25,
		Items: []app.QuotationItemInput{{ItemType: "katalog", CatalogID: "cat-seo-retainer", Qty: 3, UnitPrice: "7500000"}},
		Terms: []app.QuotationTermInput{{Label: "DP", Percent: 50}, {Label: "Pelunasan", Percent: 50}}}))
	if q.ApprovalStatus != domain.ApprovalPending || q.Total.String() != "18731250.00" {
		t.Fatalf("quote %s %s", q.ApprovalStatus, q.Total)
	}
	if n := e.rec.notified(app.NotifApproval); len(n) != 1 || len(n[0].UserIDs) != 2 {
		t.Fatalf("approvers notified %+v", n)
	}
	_, err := e.svc.Quotes.SetStatus(ctx, salesActor, q.ID, domain.QuoteSent)
	code(t, err, apperr.Conflict)
	inbox := must(e.svc.Quotes.ListApprovals(ctx, managerActor, "", ""))
	if len(inbox) != 1 || !inbox[0].CanDecide || len(inbox[0].Steps) != 2 {
		t.Fatalf("inbox %+v", inbox)
	}
	_, err = e.svc.Quotes.Decide(ctx, salesActor, *q.ApprovalRequestID, "approve", "")
	code(t, err, apperr.Unauthorized)
	res := must(e.svc.Quotes.Decide(ctx, managerActor, *q.ApprovalRequestID, "approve", ""))
	if res.Status != "pending" || res.NextLevel != 2 {
		t.Fatalf("level 1 %+v", res)
	}
	_, err = e.svc.Quotes.Decide(ctx, managerActor, *q.ApprovalRequestID, "approve", "")
	code(t, err, apperr.Unauthorized)
	res = must(e.svc.Quotes.Decide(ctx, adminActor, *q.ApprovalRequestID, "approve", "OK"))
	if res.Status != "approved" {
		t.Fatalf("level 2 %+v", res)
	}
	must(e.svc.Quotes.SetStatus(ctx, salesActor, q.ID, domain.QuoteSent))
	rev := must(e.svc.Quotes.Revise(ctx, salesActor, q.ID))
	old := must(e.store.GetQuotation(ctx, q.ID))
	if rev.Version != 2 || rev.QuoteNumber != q.QuoteNumber+"-R1" || old.Status != domain.QuoteSuperseded || rev.ApprovalStatus != domain.ApprovalPending || len(rev.Terms) != 2 {
		t.Fatalf("revise %+v / %s", rev, old.Status)
	}
}

func TestPublicFormAndChatbot(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	started := float64(time.Now().Add(-time.Minute).UnixMilli())
	res := must(e.svc.Forms.Submit(ctx, "konsultasi-gratis", "9.9.9.9", map[string]any{"pic_name": "Rudi", "org_name": "Toko Rudi", "pic_phone": "081211112222", "layanan": "SEO", "utm_source": "google", "form_started_at": started}))
	if !res.OK {
		t.Fatalf("submit %+v", res)
	}
	leads, total := must2(e.store.ListLeads(ctx, domain.LeadFilter{Query: "Rudi"}))
	if total != 1 || leads[0].Source != "google" || deref(leads[0].PicPhone) != "6281211112222" {
		t.Fatalf("form lead %+v", leads)
	}
	res = must(e.svc.Forms.Submit(ctx, "konsultasi-gratis", "9.9.9.9", map[string]any{"pic_name": "Bot", "org_name": "X", "website_url": "spam"}))
	if !res.OK {
		t.Fatal("bots get the same answer")
	}
	res = must(e.svc.Forms.Submit(ctx, "konsultasi-gratis", "9.9.9.9", map[string]any{"org_name": "X"}))
	if res.OK || len(res.Errors) == 0 {
		t.Fatal("invalid submission")
	}

	chat := must(e.svc.Chat.StartPublic(ctx, "8.8.8.8", app.StartInput{Name: "Lia", PageURL: "https://mixist.id"}))
	if len(chat.Token) < 32 || len(chat.Messages) != 1 {
		t.Fatalf("start %+v", chat)
	}
	out := must(e.svc.Chat.SendPublic(ctx, "8.8.8.8", chat.Token, "Halo, berapa harga kelola Instagram?"))
	if len(out.Messages) != 2 || out.Messages[1].Sender != "bot" || !strings.Contains(out.Messages[1].Body, "Rp5.500.000") {
		t.Fatalf("bot reply %+v", out.Messages)
	}
	must(e.svc.Chat.SendPublic(ctx, "8.8.8.8", chat.Token, "Nama saya Lia dari brand Kedai Lia, email lia@kedai.id, budget 8 juta mulai bulan depan"))
	conv := must(e.store.GetConversationByToken(ctx, chat.Token))
	lead := must(e.store.GetLead(ctx, deref(conv.LeadID)))
	if lead.Source != "chatbot" || lead.OrgName != "Kedai Lia" || deref(lead.PicEmail) != "lia@kedai.id" || lead.Temperature != "panas" {
		t.Fatalf("chatbot lead %+v", lead)
	}
	settings := must(e.svc.Chat.GetSettings(ctx))
	settings.HandoffUserIDs = []string{managerID}
	must(e.svc.Chat.SaveSettings(ctx, adminActor, settings))
	out = must(e.svc.Chat.SendPublic(ctx, "8.8.8.8", chat.Token, "Saya mau komplain, kecewa sekali"))
	if !out.HumanTakeover || len(e.rec.notified(app.NotifChatHandoff)) != 1 {
		t.Fatalf("handoff %+v", out)
	}
	before := len(must(e.store.RecentMessages(ctx, conv.ID, 100)))
	must(e.svc.Chat.SendPublic(ctx, "8.8.8.8", chat.Token, "Halo?"))
	if after := len(must(e.store.RecentMessages(ctx, conv.ID, 100))); after != before+1 {
		t.Fatal("bot stays quiet after hand-off")
	}
	detail := must(e.svc.Chat.Reply(ctx, salesActor, conv.ID, "Mohon maaf Kak, saya bantu cek."))
	if detail.Conversation.FirstResponseSeconds == nil || detail.Conversation.AssignedUserID == nil {
		t.Fatal("first response")
	}
	in, status := must2(e.svc.Chat.Analyze(ctx, conv.ID))
	if status != "analyzed" || !in.IsComplaint {
		t.Fatalf("insight %+v %s", in, status)
	}
	if _, status = must2(e.svc.Chat.Analyze(ctx, conv.ID)); status != "cache" {
		t.Fatal("cached insight")
	}
}

func TestWhatsAppInbound(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	msg := []app.WAInbound{{RemoteJid: "6281333344444@s.whatsapp.net", MessageID: "M1", Text: "Halo, mau tanya SEO", PushName: "Dodi"}, {RemoteJid: "120363@g.us", Text: "grup"}}
	_, _, err := e.svc.Chat.ReceiveWhatsApp(ctx, "7.7.7.7", "gw-secret", msg)
	code(t, err, apperr.NotFound)
	e.wa.enabled = true
	_, _, err = e.svc.Chat.ReceiveWhatsApp(ctx, "7.7.7.7", "salah", msg)
	code(t, err, apperr.Unauthenticated)
	stored, skipped := must2(e.svc.Chat.ReceiveWhatsApp(ctx, "7.7.7.7", "gw-secret", msg))
	if stored != 1 || skipped != 1 || len(e.wa.sent) != 1 || !strings.HasPrefix(e.wa.sent[0], "6281333344444") {
		t.Fatalf("stored %d skipped %d sent %v", stored, skipped, e.wa.sent)
	}
	stored, skipped = must2(e.svc.Chat.ReceiveWhatsApp(ctx, "7.7.7.7", "gw-secret", msg[:1]))
	if stored != 0 || skipped != 1 {
		t.Fatal("duplicate message id is skipped")
	}
	conv := must(e.store.GetConversationByExternal(ctx, "whatsapp", "6281333344444"))
	if conv.LeadID == nil {
		t.Fatal("WhatsApp chats create a lead from the phone")
	}
}

func TestDecksForecastAndJobs(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	lead, deal := newDeal(t, e)
	d1 := must(e.svc.Decks.Generate(ctx, salesActor, deal.ID, "", domain.DeckOptions{IncludePricing: true, IncludeCaseStudies: true}))
	d2 := must(e.svc.Decks.Generate(ctx, salesActor, "", lead.ID, domain.DeckOptions{}))
	d3 := must(e.svc.Decks.Generate(ctx, salesActor, deal.ID, "", domain.DeckOptions{Tone: "premium"}))
	if d1.Version != 1 || d3.Version != 2 || d2.Version != 1 || !bytes.HasPrefix(d1.File, []byte("PK")) || d1.Model != app.MockModel {
		t.Fatalf("decks v%d v%d v%d", d1.Version, d2.Version, d3.Version)
	}
	kinds := map[string]int{}
	for _, s := range d1.Slides {
		kinds[s.Kind]++
	}
	if kinds["case_study"] != 3 || kinds["strategy"] != 2 || kinds["pricing"] != 1 {
		t.Fatalf("slides %v", kinds)
	}

	month := time.Now().In(domain.Jakarta).Format("2006-01")
	_, _, err := e.svc.Insights.SaveTargets(ctx, managerActor, month, []domain.Target{{UserID: ptr(salesID), TargetValue: domain.MustMoney("100000000")}})
	if err != nil {
		t.Fatal(err)
	}
	f := must(e.svc.Insights.Forecast(ctx, ""))
	if f.Total.PipelineValue != domain.MustMoney("50000000") || f.Total.TargetValue != domain.MustMoney("100000000") || f.Rows[0].UserName != "Sari Sales" {
		t.Fatalf("forecast %+v %+v", f.Total, f.Rows[0])
	}
	rep, _, _ := must3(e.svc.Insights.Funnel(ctx, "", ""))
	if rep.Leads != 1 || rep.DealsCreated != 1 {
		t.Fatalf("funnel %+v", rep)
	}

	soon := time.Now().Add(-time.Minute)
	must(e.svc.Pipeline.CreateActivity(ctx, salesActor, app.ActivityInput{DealID: deal.ID, ActivityType: "telepon", Title: "Follow up", ReminderAt: &soon, DueAt: ptr(time.Now().Add(time.Hour))}))
	if err := e.jobs.Every15m(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e.jobs.Every15m(ctx); err != nil {
		t.Fatal(err)
	}
	if n := len(e.rec.notified(app.NotifTaskReminder)); n != 1 {
		t.Fatalf("reminders sent %d", n)
	}
	q := must(e.svc.Quotes.CreateQuotation(ctx, salesActor, app.QuotationInput{DealID: deal.ID, ValidUntil: time.Now().In(domain.Jakarta).AddDate(0, 0, 2).Format("2006-01-02"), Items: []app.QuotationItemInput{{Description: "Audit SEO", Qty: 1, UnitPrice: "5000000"}}}))
	must(e.svc.Quotes.SetStatus(ctx, salesActor, q.ID, domain.QuoteSent))
	if err := e.jobs.Daily(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e.jobs.Daily(ctx); err != nil {
		t.Fatal(err)
	}
	if n := len(e.rec.notified(app.NotifQuoteExpiry)); n != 1 {
		t.Fatalf("expiry reminders %d", n)
	}
}

// ─── helpers ─────────────────────────────────────────────────

func must2[A, B any](a A, b B, err error) (A, B) {
	if err != nil {
		panic(err)
	}
	return a, b
}

func must3[A, B, C any](a A, b B, c C, err error) (A, B, C) {
	if err != nil {
		panic(err)
	}
	return a, b, c
}

func ptr[T any](v T) *T { return &v }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func pngBytes(t *testing.T) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func signatureURL(t *testing.T) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes(t))
}
