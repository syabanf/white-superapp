package domain

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
)

// Conversation analysis ported from crm/inbox/domain/insights.go and the
// SLA bookkeeping of the CS inbox.

const (
	MaxTranscriptMessages = 300
	MaxTranscriptChars    = 6000
	// SLAResponseMinutes is the first-response target before a
	// conversation that waits on the team is flagged as breached.
	SLAResponseMinutes = 30
)

// Conversation statuses.
var ConversationStatuses = []string{"open", "in_progress", "waiting_customer", "resolved"}

// TranscriptMessage is one message from the business's point of view.
type TranscriptMessage struct {
	Direction string // in | out
	Body      string
}

// Fingerprint mirrors fingerprintTranscript: FNV-1a over the UTF-16 units
// of "dir:body" lines, prefixed with the message count.
func Fingerprint(messages []TranscriptMessage) string {
	lines := make([]string, len(messages))
	for i, m := range messages {
		lines[i] = m.Direction + ":" + m.Body
	}
	hash := uint32(0x811c9dc5)
	for _, u := range utf16.Encode([]rune(strings.Join(lines, "\n"))) {
		hash ^= uint32(u)
		hash *= 0x01000193
	}
	return strconv.Itoa(len(messages)) + "-" + strconv.FormatUint(uint64(hash), 16)
}

// NeedsAnalysis analyses when nothing is stored or the fingerprint moved.
func NeedsAnalysis(storedFingerprint *string, fingerprint string) bool {
	return storedFingerprint == nil || *storedFingerprint == "" || *storedFingerprint != fingerprint
}

const truncationMarker = "\n… (bagian tengah percakapan dipotong) …\n"

func utf16Len(s string) int { return len(utf16.Encode([]rune(s))) }

func utf16Slice(s string, from, to int) string {
	u := utf16.Encode([]rune(s))
	return string(utf16.Decode(u[from:to]))
}

// TranscriptText mirrors transcriptToText: labelled lines; a long
// transcript keeps its start and end around a marker.
func TranscriptText(messages []TranscriptMessage, maxChars int) string {
	var lines []string
	for _, m := range messages {
		body := JSTrim(m.Body)
		if body == "" {
			continue
		}
		label := "Agen"
		if m.Direction == "in" {
			label = "Pelanggan"
		}
		lines = append(lines, label+": "+body)
	}
	full := strings.Join(lines, "\n")
	n := utf16Len(full)
	if n <= maxChars {
		return full
	}
	half := (maxChars - utf16Len(truncationMarker)) / 2
	return utf16Slice(full, 0, half) + truncationMarker + utf16Slice(full, n-half, n)
}

// InsightSystemPrompt is the analysis prompt (systemPrompt in insights.go).
var InsightSystemPrompt = strings.Join([]string{
	"Kamu menganalisa percakapan layanan pelanggan berbahasa Indonesia.",
	"Jawab HANYA dengan satu objek JSON tanpa penjelasan tambahan dan tanpa pagar kode.",
	`Bentuk JSON: {"summary": string, "topic": string, "sentiment": "positif"|"netral"|"negatif", "is_complaint": boolean, "keywords": string[]}.`,
	"summary: 1-2 kalimat ringkas berisi inti permintaan pelanggan dan hasil akhirnya.",
	`topic: label singkat maksimal 4 kata, huruf kecil, mis. "keluhan pengiriman" atau "tanya harga".`,
	"sentiment: perasaan pelanggan, bukan perasaan agen.",
	"is_complaint: true hanya bila pelanggan menyatakan ketidakpuasan atau masalah nyata.",
	"keywords: 3-8 kata/frasa kunci spesifik dari isi percakapan (produk, masalah, lokasi, layanan).",
	"Jangan memasukkan nama orang, nomor telepon, atau alamat ke keywords maupun topic.",
	`Bila percakapan terlalu pendek atau tidak bermakna, tetap jawab JSON dengan topic "tidak jelas".`,
}, " ")

// Insight is a parsed analysis.
type Insight struct {
	ConversationID string
	Summary        string
	Topic          string
	Sentiment      string
	IsComplaint    bool
	Keywords       []string
	Fingerprint    string
	Model          string
	AnalyzedAt     time.Time
}

func collapseSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

func clampText(v any, max int) string {
	s, _ := v.(string)
	s = JSTrim(collapseSpace(s))
	if utf16Len(s) > max {
		return JSTrim(utf16Slice(s, 0, max))
	}
	return s
}

// Sentiment maps a model sentiment to positif/netral/negatif.
func Sentiment(v any) string {
	s, _ := v.(string)
	switch strings.ToLower(JSTrim(s)) {
	case "positif", "positive":
		return "positif"
	case "negatif", "negative":
		return "negatif"
	}
	return "netral"
}

// ParseInsight mirrors parseInsight: nil when the answer is not a JSON
// object.
func ParseInsight(raw string) *Insight {
	var src map[string]any
	if JSTrim(raw) == "" || json.Unmarshal([]byte(StripCodeFence(raw)), &src) != nil {
		return nil
	}
	topic := strings.ToLower(clampText(src["topic"], 60))
	if topic == "" {
		topic = "tidak jelas"
	}
	complaint, _ := src["is_complaint"].(bool)
	return &Insight{Summary: clampText(src["summary"], 500), Topic: topic, Sentiment: Sentiment(src["sentiment"]), IsComplaint: complaint, Keywords: NormalizeKeywordList(src["keywords"])}
}

var stopwordsID = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`yang dan di ke dari untuk dengan pada adalah itu
		ini atau juga sudah belum tidak bukan ada akan
		saya aku kami kita anda kamu dia mereka
		bisa boleh mau ingin harus saja aja kok sih dong
		ya iya oke ok gak ngga nggak tak nya
		pak bu bapak ibu mas mbak kak bang
		terima kasih tolong mohon selamat halo hai assalamualaikum
		pagi siang sore malam
		apa kapan dimana mana kenapa bagaimana gimana berapa
		lagi masih sangat sekali banget agak cuma hanya`) {
		stopwordsID[w] = true
	}
}

// NormalizeKeyword mirrors normalizeKeyword: lower case, punctuation to
// spaces, 3..40 units, three letters in a row, not only stopwords.
func NormalizeKeyword(raw any) string {
	s, ok := raw.(string)
	if !ok {
		return ""
	}
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	cleaned := JSTrim(collapseSpace(b.String()))
	if n := utf16Len(cleaned); n < 3 || n > 40 {
		return ""
	}
	run, hasWord := 0, false
	for _, r := range cleaned {
		if unicode.IsLetter(r) {
			if run++; run >= 3 {
				hasWord = true
				break
			}
		} else {
			run = 0
		}
	}
	if !hasWord {
		return ""
	}
	for _, w := range strings.Split(cleaned, " ") {
		if !stopwordsID[w] {
			return cleaned
		}
	}
	return ""
}

// NormalizeKeywordList keeps at most eight distinct cleaned keywords.
func NormalizeKeywordList(raw any) []string {
	items, _ := raw.([]any)
	out := []string{}
	for _, item := range items {
		if k := NormalizeKeyword(item); k != "" && !slices.Contains(out, k) {
			out = append(out, k)
		}
		if len(out) >= 8 {
			break
		}
	}
	return out
}

// MockInsight analyses a transcript without a model: complaint keywords,
// detected service lines and a counted summary. Same input, same output.
func MockInsight(messages []TranscriptMessage) Insight {
	var customer []string
	for _, m := range messages {
		if m.Direction == "in" {
			customer = append(customer, m.Body)
		}
	}
	text := strings.Join(customer, " ")
	complaint, _ := DetectHandoff(text)
	isComplaint := complaint && complaintRe.MatchString(text)
	lines := DetectServiceLines(text)
	keywords := []string{}
	for _, l := range lines {
		keywords = append(keywords, strings.ToLower(ServiceLineLabel(l)))
	}
	topic := "tanya layanan"
	switch {
	case isComplaint:
		topic = "keluhan layanan"
	case pricingAskRe.MatchString(text):
		topic = "tanya harga"
	case len(customer) == 0:
		topic = "tidak jelas"
	}
	sentiment := "netral"
	if isComplaint {
		sentiment = "negatif"
	} else if strings.Contains(strings.ToLower(text), "terima kasih") {
		sentiment = "positif"
	}
	summary := "Calon klien mengirim " + strconv.Itoa(len(customer)) + " pesan"
	if len(keywords) > 0 {
		summary += " tentang " + strings.Join(keywords, ", ")
	}
	return Insight{Summary: summary + ".", Topic: topic, Sentiment: sentiment, IsComplaint: isComplaint, Keywords: keywords}
}
