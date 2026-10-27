package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The lead chatbot: prompt construction, answer parsing, lead field
// extraction, hand-off rules, the price guard and the deterministic mock
// used when no OpenRouter key is configured.

// ChatbotSettings is the chatbot_settings row.
type ChatbotSettings struct {
	Enabled            bool
	BotName            string
	Persona            string
	ServicesOffered    string
	PricingHints       string
	BusinessHoursStart int
	BusinessHoursEnd   int
	HandoffRules       string
	Greeting           string
	MinConfidence      float64
	HandoffUserIDs     []string
	UpdatedBy          *string
	UpdatedAt          time.Time
}

// ChatTurn is one message of the history the bot sees.
type ChatTurn struct {
	Sender string // customer | bot | agent | system
	Body   string
}

// LeadExtract holds lead fields the bot recognised.
type LeadExtract struct {
	Name     string `json:"name,omitempty"`
	Company  string `json:"company,omitempty"`
	Need     string `json:"need,omitempty"`
	Budget   string `json:"budget,omitempty"`
	Timeline string `json:"timeline,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Email    string `json:"email,omitempty"`
}

// Empty reports whether nothing was extracted.
func (l LeadExtract) Empty() bool { return l == LeadExtract{} }

// Merge keeps existing values and fills the blanks from other.
func (l LeadExtract) Merge(other LeadExtract) LeadExtract {
	pick := func(a, b string) string {
		if JSTrim(a) != "" {
			return a
		}
		return JSTrim(b)
	}
	return LeadExtract{
		Name: pick(l.Name, other.Name), Company: pick(l.Company, other.Company), Need: pick(l.Need, other.Need),
		Budget: pick(l.Budget, other.Budget), Timeline: pick(l.Timeline, other.Timeline),
		Phone: pick(l.Phone, other.Phone), Email: pick(l.Email, other.Email),
	}
}

// BotReply is the parsed answer of the model (or the mock).
type BotReply struct {
	Reply         string      `json:"reply"`
	Confidence    float64     `json:"confidence"`
	Handoff       bool        `json:"handoff"`
	HandoffReason string      `json:"handoff_reason,omitempty"`
	Lead          LeadExtract `json:"lead"`
}

// Hand-off reasons.
const (
	HandoffAskedHuman    = "minta_manusia"
	HandoffComplaint     = "komplain"
	HandoffLowConfidence = "keyakinan_rendah"
	HandoffPriceGuard    = "harga_di_luar_daftar"
)

// IsBusinessHour reports whether at falls inside [start, end) in WIB;
// start == end means always open, start > end spans midnight.
func IsBusinessHour(at time.Time, start, end int) bool {
	h := at.In(Jakarta).Hour()
	switch {
	case start == end:
		return true
	case start < end:
		return h >= start && h < end
	}
	return h >= start || h < end
}

// BotSystemPrompt builds the system prompt from the settings row.
func BotSystemPrompt(s ChatbotSettings, now time.Time) string {
	hours := fmt.Sprintf("%02d.00-%02d.00 WIB", s.BusinessHoursStart, s.BusinessHoursEnd)
	open := "sedang dalam jam kerja"
	if !IsBusinessHour(now, s.BusinessHoursStart, s.BusinessHoursEnd) {
		open = "di luar jam kerja; tim manusia membalas pada jam kerja berikutnya"
	}
	return strings.Join([]string{
		"Kamu adalah " + s.BotName + ", asisten chat untuk Mixist, digital marketing agency di Indonesia.",
		"Persona: " + s.Persona,
		"Layanan yang ditawarkan:\n" + s.ServicesOffered,
		"Panduan harga (HANYA boleh menyebut angka yang ada di sini):\n" + s.PricingHints,
		"Jam kerja tim: " + hours + ". Saat ini " + open + ".",
		"Aturan hand-off ke manusia:\n" + s.HandoffRules,
		"Aturan wajib:",
		"- Balas dalam Bahasa Indonesia yang ramah, singkat (maksimal 3 kalimat), tanpa markdown.",
		"- Jangan pernah mengarang harga, diskon, atau janji di luar panduan harga. Bila ditanya harga yang tidak ada di panduan, katakan tim sales akan mengirim penawaran.",
		"- Gali kebutuhan calon klien: nama, perusahaan, kebutuhan, budget, timeline, nomor WhatsApp atau email.",
		"- Bila pengguna minta bicara dengan manusia, menyampaikan komplain, atau kamu tidak yakin, set handoff true.",
		"Jawab HANYA dengan satu objek JSON tanpa pagar kode:",
		`{"reply": string, "confidence": number 0-1, "handoff": boolean, "handoff_reason": "minta_manusia"|"komplain"|"keyakinan_rendah"|"", "lead": {"name": string, "company": string, "need": string, "budget": string, "timeline": string, "phone": string, "email": string}}`,
		"Isi field lead hanya dengan data yang disebut pengguna; kosongkan yang tidak diketahui.",
	}, "\n\n")
}

// BotHistory renders the last turns for the user message.
func BotHistory(turns []ChatTurn, max int) string {
	if len(turns) > max {
		turns = turns[len(turns)-max:]
	}
	var b strings.Builder
	for _, t := range turns {
		label := "Tim"
		switch t.Sender {
		case "customer":
			label = "Pengguna"
		case "bot":
			label = "Bot"
		case "system":
			continue
		}
		b.WriteString(label + ": " + strings.ReplaceAll(t.Body, "\n", " ") + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

var (
	fenceOpen  = regexp.MustCompile("^```[a-zA-Z]*")
	fenceClose = regexp.MustCompile("```$")
)

// StripCodeFence removes a ```json fence around a model answer.
func StripCodeFence(v string) string {
	t := JSTrim(v)
	if !strings.HasPrefix(t, "```") {
		return t
	}
	t = JSTrim(fenceOpen.ReplaceAllString(t, ""))
	return JSTrim(fenceClose.ReplaceAllString(t, ""))
}

// ParseBotReply reads the model's JSON. Plain text becomes the reply with
// a middling confidence so the guard rails still apply.
func ParseBotReply(content string) BotReply {
	raw := StripCodeFence(content)
	if i, j := strings.Index(raw, "{"), strings.LastIndex(raw, "}"); i >= 0 && j > i {
		var r BotReply
		if err := json.Unmarshal([]byte(raw[i:j+1]), &r); err == nil && JSTrim(r.Reply) != "" {
			r.Reply = JSTrim(r.Reply)
			r.Confidence = min(1, max(0, r.Confidence))
			return r
		}
	}
	return BotReply{Reply: SliceUTF16(raw, 1000), Confidence: 0.5}
}

var (
	humanRe     = regexp.MustCompile(`(?i)\b(manusia|admin|cs|customer service|orang asli|bicara dengan (tim|sales|orang)|hubungi saya|telepon saya|minta ditelepon|operator|agen)\b`)
	complaintRe = regexp.MustCompile(`(?i)\b(komplain|keluhan|kecewa|marah|tidak puas|penipuan|nipu|refund|uang kembali|lapor|buruk sekali|parah)\b`)
)

// DetectHandoff applies the fixed hand-off rules to a customer message.
func DetectHandoff(msg string) (bool, string) {
	switch {
	case complaintRe.MatchString(msg):
		return true, HandoffComplaint
	case humanRe.MatchString(msg):
		return true, HandoffAskedHuman
	}
	return false, ""
}

var (
	chatEmailRe    = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	chatPhoneRe    = regexp.MustCompile(`(?:\+?62|0)8[0-9\s\-]{7,14}[0-9]`)
	chatNameRe     = regexp.MustCompile(`\b(?i:nama saya|saya bernama|perkenalkan,? saya|panggil saya)\s+([A-Za-z][a-z.']*(?:\s+[A-Z][a-z.']*){0,2})`)
	chatCompanyRe  = regexp.MustCompile(`\b(?i:dari|perusahaan|brand|toko|usaha|bisnis)\s+(?:(?i:brand|perusahaan|toko|usaha)\s+)?((?:PT|CV|UD)\.?\s+[A-Za-z0-9&.\- ]{2,40}|[A-Z][A-Za-z0-9&.\-]+(?:\s+[A-Z][A-Za-z0-9&.\-]+){0,3})`)
	chatBudgetRe   = regexp.MustCompile(`(?i)\b(?:budget|bujet|anggaran|dana)\D{0,15}((?:rp\.?\s*)?[0-9][0-9.,]*\s*(?:jt|juta|rb|ribu|k|m|miliar)?)`)
	chatTimelineRe = regexp.MustCompile(`(?i)\b(secepatnya|segera|asap|minggu (?:ini|depan)|bulan (?:ini|depan)|awal bulan|akhir bulan|tahun depan|q[1-4]|kuartal [1-4]|(?:januari|februari|maret|april|mei|juni|juli|agustus|september|oktober|november|desember)(?:\s+\d{4})?)\b`)
)

// serviceKeywords map customer words to service lines.
var serviceKeywords = []struct {
	line string
	re   *regexp.Regexp
}{
	{LineSocialMedia, regexp.MustCompile(`(?i)\b(sosmed|sosial media|social media|instagram|tiktok|feed|followers?|smm)\b`)},
	{LineSEO, regexp.MustCompile(`(?i)\b(seo|google search|ranking|halaman satu|page one|organik)\b`)},
	{LineAds, regexp.MustCompile(`(?i)\b(iklan|ads|meta ads|google ads|tiktok ads|campaign|kampanye|beriklan)\b`)},
	{LineContent, regexp.MustCompile(`(?i)\b(konten|content|video|foto|fotografi|reels|copywriting)\b`)},
	{LineWebsite, regexp.MustCompile(`(?i)\b(website|web|situs|landing page|toko online)\b`)},
	{LineBranding, regexp.MustCompile(`(?i)\b(branding|logo|brand identity|identitas brand|rebranding)\b`)},
}

// DetectServiceLines lists the service lines a message mentions.
func DetectServiceLines(msg string) []string {
	var out []string
	for _, k := range serviceKeywords {
		if k.re.MatchString(msg) {
			out = append(out, k.line)
		}
	}
	return out
}

// ExtractLeadFields pulls contact and qualification data out of free text.
func ExtractLeadFields(msg string) LeadExtract {
	var l LeadExtract
	if m := chatEmailRe.FindString(msg); m != "" {
		l.Email = strings.ToLower(m)
	}
	if m := chatPhoneRe.FindString(msg); m != "" {
		if p := NormalizePhone(m); IsValidNormalizedPhone(p) {
			l.Phone = p
		}
	}
	if m := chatNameRe.FindStringSubmatch(msg); m != nil {
		l.Name = JSTrim(m[1])
	}
	if m := chatCompanyRe.FindStringSubmatch(msg); m != nil {
		l.Company = JSTrim(strings.TrimRight(m[1], ".,"))
	}
	if m := chatBudgetRe.FindStringSubmatch(msg); m != nil {
		l.Budget = JSTrim(m[1])
	}
	if m := chatTimelineRe.FindStringSubmatch(msg); m != nil {
		l.Timeline = strings.ToLower(m[1])
	}
	if lines := DetectServiceLines(msg); len(lines) > 0 {
		labels := make([]string, len(lines))
		for i, line := range lines {
			labels[i] = ServiceLineLabel(line)
		}
		l.Need = strings.Join(labels, ", ")
	}
	return l
}

var (
	rupiahRe = regexp.MustCompile(`(?i)rp\.?\s*([0-9][0-9.,]*)\s*(jt|juta|rb|ribu|k)?`)
	numberRe = regexp.MustCompile(`[0-9][0-9.,]*\s*(?:jt|juta|rb|ribu|k)?`)
	unitRe   = regexp.MustCompile(`(?i)(jt|juta|rb|ribu|k)$`)
)

// amountValue normalises "7,5 jt", "7.500.000" and "7500k" to rupiah.
func amountValue(digits, unit string) int64 {
	digits = strings.TrimSpace(digits)
	mult := 1.0
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "jt", "juta":
		mult = 1_000_000
	case "rb", "ribu", "k":
		mult = 1_000
	}
	if mult > 1 {
		// "7,5" and "7.5" are decimals when followed by a unit.
		f, err := strconv.ParseFloat(strings.ReplaceAll(digits, ",", "."), 64)
		if err != nil {
			return -1
		}
		return int64(f * mult)
	}
	clean := strings.NewReplacer(".", "", ",", "").Replace(digits)
	n, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		return -1
	}
	return n
}

// AllowedAmounts lists every rupiah amount mentioned in the pricing hints.
func AllowedAmounts(hints string) map[int64]bool {
	out := map[int64]bool{}
	for _, m := range numberRe.FindAllString(hints, -1) {
		m = strings.TrimSpace(m)
		unit := unitRe.FindString(m)
		digits := strings.TrimSpace(strings.TrimSuffix(m, unit))
		if v := amountValue(digits, unit); v > 0 {
			out[v] = true
		}
	}
	return out
}

// PricesWithinHints reports whether every rupiah amount in reply appears
// in the pricing hints, so the bot never invents a price.
func PricesWithinHints(reply, hints string) bool {
	allowed := AllowedAmounts(hints)
	for _, m := range rupiahRe.FindAllStringSubmatch(reply, -1) {
		if v := amountValue(m[1], m[2]); v > 0 && !allowed[v] {
			return false
		}
	}
	return true
}

// PriceGuardReply replaces an answer that quoted an unknown price.
const PriceGuardReply = "Untuk angka pastinya, tim sales kami akan menyiapkan penawaran resmi sesuai kebutuhan Anda. Boleh saya catat nama dan nomor WhatsApp atau email Anda?"

// HandoffReply is what the bot says when it passes the chat on.
func HandoffReply(reason string, s ChatbotSettings, now time.Time) string {
	msg := "Baik, saya teruskan ke tim sales kami ya. "
	if reason == HandoffComplaint {
		msg = "Mohon maaf atas ketidaknyamanannya. Pesan Anda sudah kami teruskan ke tim kami untuk ditangani langsung. "
	}
	if IsBusinessHour(now, s.BusinessHoursStart, s.BusinessHoursEnd) {
		return msg + "Mereka akan segera membalas di chat ini."
	}
	return msg + fmt.Sprintf("Tim kami akan membalas pada jam kerja (%02d.00-%02d.00 WIB).", s.BusinessHoursStart, s.BusinessHoursEnd)
}

var pricingAskRe = regexp.MustCompile(`(?i)\b(harga|biaya|tarif|price|pricelist|berapa|paket|rate ?card)\b`)

// MockBotReply is the deterministic bot used without an OpenRouter key:
// keyword rules over the latest message and what is already known.
func MockBotReply(s ChatbotSettings, known LeadExtract, msg string, now time.Time) BotReply {
	extract := ExtractLeadFields(msg)
	merged := known.Merge(extract)
	if ok, reason := DetectHandoff(msg); ok {
		return BotReply{Reply: HandoffReply(reason, s, now), Confidence: 0.95, Handoff: true, HandoffReason: reason, Lead: extract}
	}
	lines := DetectServiceLines(msg)
	var parts []string
	confidence := 0.85
	switch {
	case pricingAskRe.MatchString(msg):
		parts = append(parts, "Sebagai gambaran, berikut kisaran harga kami: "+strings.ReplaceAll(JSTrim(s.PricingHints), "\n", "; ")+".")
		parts = append(parts, "Harga final menyesuaikan kebutuhan dan akan dikirim dalam penawaran resmi.")
	case len(lines) > 0:
		labels := make([]string, len(lines))
		for i, l := range lines {
			labels[i] = ServiceLineLabel(l)
		}
		parts = append(parts, "Tentu, Mixist bisa bantu untuk "+strings.Join(labels, " dan ")+".")
	case !extract.Empty():
		parts = append(parts, "Terima kasih, sudah saya catat.")
	case strings.Contains(msg, "?"):
		return BotReply{Reply: "Pertanyaan yang bagus. Supaya jawabannya tepat, saya hubungkan dengan tim sales kami ya.", Confidence: 0.4, Lead: extract}
	default:
		parts = append(parts, "Terima kasih sudah menghubungi Mixist.")
		confidence = 0.7
	}
	switch {
	case merged.Need == "":
		parts = append(parts, "Layanan apa yang sedang Anda butuhkan: social media, SEO, iklan digital, konten, website, atau branding?")
	case merged.Name == "" || merged.Company == "":
		parts = append(parts, "Boleh tahu nama Anda dan nama perusahaan atau brand-nya?")
	case merged.Phone == "" && merged.Email == "":
		parts = append(parts, "Boleh minta nomor WhatsApp atau email agar tim kami bisa mengirim penawaran?")
	case merged.Timeline == "" || merged.Budget == "":
		parts = append(parts, "Kira-kira kapan ingin mulai dan berapa anggaran yang disiapkan?")
	default:
		parts = append(parts, "Data Anda sudah lengkap. Tim sales kami akan menghubungi Anda untuk menyiapkan penawaran.")
	}
	return BotReply{Reply: strings.Join(parts, " "), Confidence: confidence, Lead: extract}
}
