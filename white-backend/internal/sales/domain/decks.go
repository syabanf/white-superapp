package domain

import (
	"encoding/json"
	"strings"
)

// Pitch deck automation: a fixed slide structure (cover, about Mixist,
// challenges, strategy per service line, timeline, team, pricing, case
// studies, next steps) whose free text comes from OpenRouter or the mock.

// Slide is one slide of a deck.
type Slide struct {
	Kind     string   `json:"kind"`
	Title    string   `json:"title"`
	Subtitle string   `json:"subtitle,omitempty"`
	Bullets  []string `json:"bullets,omitempty"`
}

// DeckOptions are the generation switches.
type DeckOptions struct {
	Tone               string `json:"tone,omitempty"` // profesional | santai | premium
	IncludePricing     bool   `json:"include_pricing"`
	IncludeCaseStudies bool   `json:"include_case_studies"`
	ExtraNotes         string `json:"extra_notes,omitempty"`
}

// CaseStudy is one portfolio entry shown in decks.
type CaseStudy struct {
	ID           string
	Title        string
	ClientName   string
	Industry     string
	ServiceLines []string
	Challenge    string
	Solution     string
	Results      string
	SortOrder    int
	IsActive     bool
}

// DeckInput is what a deck is generated from.
type DeckInput struct {
	ClientName   string
	Industry     string
	PicName      string
	Notes        string
	ServiceLines []string
	Quotation    *Quotation
	CaseStudies  []CaseStudy
	TeamNames    []string
	Options      DeckOptions
}

// DeckCopy is the free text the model writes.
type DeckCopy struct {
	Tagline    string              `json:"tagline"`
	Challenges []string            `json:"challenges"`
	Strategy   map[string][]string `json:"strategy"`
	Timeline   []string            `json:"timeline"`
	NextSteps  []string            `json:"next_steps"`
}

// DeckSystemPrompt asks for DeckCopy as JSON.
func DeckSystemPrompt(tone string) string {
	if tone == "" {
		tone = "profesional"
	}
	return strings.Join([]string{
		"Kamu adalah strategist senior di Mixist, digital marketing agency di Indonesia, yang menyusun isi pitch deck untuk calon klien.",
		"Nada penulisan: " + tone + ". Bahasa Indonesia, kalimat pendek, siap tampil di slide (maksimal 14 kata per butir).",
		"Gunakan hanya informasi dari data yang diberikan; jangan mengarang angka, harga, atau nama klien lain.",
		"Jawab HANYA dengan satu objek JSON tanpa pagar kode:",
		`{"tagline": string, "challenges": string[3-5], "strategy": {"<service_line>": string[3-4]}, "timeline": string[4-6], "next_steps": string[3-4]}`,
		"Kunci strategy harus persis sama dengan service line yang diberikan.",
	}, "\n")
}

// DeckUserMessage serialises the input for the model.
func DeckUserMessage(in DeckInput) string {
	lines := make([]string, len(in.ServiceLines))
	for i, l := range in.ServiceLines {
		lines[i] = l + " (" + ServiceLineLabel(l) + ")"
	}
	parts := []string{
		"Calon klien: " + in.ClientName,
		"Industri: " + orDash(in.Industry),
		"Service line: " + strings.Join(lines, ", "),
		"Catatan sales:\n" + orDash(in.Notes),
	}
	if in.Options.ExtraNotes != "" {
		parts = append(parts, "Arahan tambahan: "+in.Options.ExtraNotes)
	}
	return strings.Join(parts, "\n")
}

// ParseDeckCopy reads the model's JSON; ok is false when unusable.
func ParseDeckCopy(content string) (DeckCopy, bool) {
	raw := StripCodeFence(content)
	if i, j := strings.Index(raw, "{"), strings.LastIndex(raw, "}"); i >= 0 && j > i {
		raw = raw[i : j+1]
	}
	var c DeckCopy
	if err := json.Unmarshal([]byte(raw), &c); err != nil || len(c.Challenges) == 0 {
		return DeckCopy{}, false
	}
	return c, true
}

var lineStrategy = map[string][]string{
	LineSocialMedia: {"Audit akun dan kompetitor, lalu tetapkan pilar konten", "Kalender konten bulanan dengan format Reels dan carousel", "Community management dan laporan performa bulanan"},
	LineSEO:         {"Audit teknis dan riset kata kunci prioritas", "Optimasi on-page dan konten artikel bulanan", "Pemantauan peringkat dan trafik organik setiap bulan"},
	LineAds:         {"Struktur kampanye per tujuan: awareness, traffic, konversi", "Uji materi iklan dan audiens secara berkala", "Optimasi biaya per hasil dan laporan mingguan"},
	LineContent:     {"Konsep kreatif dan storyboard sesuai brand", "Produksi foto dan video untuk kebutuhan kanal digital", "Revisi terstruktur dan bank aset yang rapi"},
	LineWebsite:     {"Arsitektur informasi dan wireframe berbasis tujuan bisnis", "Desain responsif dan pengembangan yang cepat", "Integrasi analytics, SEO dasar, dan pelatihan admin"},
	LineBranding:    {"Riset brand, positioning, dan persona audiens", "Identitas visual: logo, warna, tipografi", "Panduan brand untuk konsistensi semua kanal"},
}

// MockDeckCopy writes the deck text without a model. Same input, same
// output.
func MockDeckCopy(in DeckInput) DeckCopy {
	c := DeckCopy{Tagline: "Strategi digital terukur untuk pertumbuhan " + in.ClientName, Strategy: map[string][]string{}}
	for _, sentence := range strings.FieldsFunc(in.Notes, func(r rune) bool { return r == '.' || r == '\n' || r == ';' }) {
		if s := JSTrim(sentence); len(s) > 8 && len(c.Challenges) < 4 {
			c.Challenges = append(c.Challenges, SliceUTF16(s, 110))
		}
	}
	if len(c.Challenges) == 0 {
		c.Challenges = []string{"Brand belum dikenal luas oleh target audiens", "Kanal digital belum menghasilkan leads yang konsisten", "Belum ada pengukuran performa yang rapi"}
	}
	for _, l := range in.ServiceLines {
		c.Strategy[l] = lineStrategy[l]
	}
	c.Timeline = []string{"Minggu 1: kickoff, pengumpulan akses dan aset", "Minggu 2: audit, riset, dan strategi disetujui", "Minggu 3-4: produksi dan peluncuran awal", "Bulan 2: optimasi berdasarkan data", "Bulan 3: evaluasi KPI dan rencana kuartal berikutnya"}
	c.NextSteps = []string{"Konfirmasi ruang lingkup dan paket", "Tanda tangan kontrak dan pembayaran termin pertama", "Jadwalkan kickoff meeting"}
	return c
}

// BuildSlides assembles the fixed structure from the input and the copy.
func BuildSlides(in DeckInput, c DeckCopy) []Slide {
	slides := []Slide{
		{Kind: "cover", Title: "Proposal Kerja Sama " + in.ClientName, Subtitle: c.Tagline},
		{Kind: "about", Title: "Tentang Mixist", Bullets: []string{
			"Digital marketing agency di Jakarta dengan tim 25 orang",
			"Tim Creative, Social Media, SEO, Ads, dan Sales dalam satu atap",
			"Berbasis data: laporan performa transparan setiap bulan",
			"Mendampingi brand dari UMKM sampai korporasi",
		}},
		{Kind: "challenges", Title: "Tantangan " + in.ClientName, Bullets: c.Challenges},
	}
	for _, l := range in.ServiceLines {
		bullets := c.Strategy[l]
		if len(bullets) == 0 {
			bullets = lineStrategy[l]
		}
		slides = append(slides, Slide{Kind: "strategy", Title: "Strategi " + ServiceLineLabel(l), Bullets: bullets})
	}
	slides = append(slides, Slide{Kind: "timeline", Title: "Timeline", Bullets: c.Timeline})
	team := append([]string{}, in.TeamNames...)
	for _, l := range in.ServiceLines {
		team = append(team, teamRole(l))
	}
	slides = append(slides, Slide{Kind: "team", Title: "Tim yang Menangani", Bullets: team})
	if in.Options.IncludePricing {
		slides = append(slides, pricingSlide(in.Quotation))
	}
	if in.Options.IncludeCaseStudies {
		for _, cs := range in.CaseStudies {
			slides = append(slides, Slide{Kind: "case_study", Title: "Studi Kasus: " + cs.Title, Subtitle: cs.ClientName + " · " + cs.Industry, Bullets: []string{"Tantangan: " + cs.Challenge, "Solusi: " + cs.Solution, "Hasil: " + cs.Results}})
		}
	}
	slides = append(slides, Slide{Kind: "next_steps", Title: "Langkah Selanjutnya", Bullets: c.NextSteps})
	return slides
}

func teamRole(line string) string {
	switch line {
	case LineSocialMedia:
		return "Social Media Strategist dan Content Creator"
	case LineSEO:
		return "SEO Specialist dan Content Writer"
	case LineAds:
		return "Performance Marketing Specialist"
	case LineContent:
		return "Creative Director, Fotografer, dan Videografer"
	case LineWebsite:
		return "UI/UX Designer dan Web Developer"
	case LineBranding:
		return "Brand Strategist dan Graphic Designer"
	}
	return ServiceLineLabel(line)
}

func pricingSlide(q *Quotation) Slide {
	s := Slide{Kind: "pricing", Title: "Ringkasan Investasi"}
	if q == nil {
		s.Bullets = []string{"Penawaran harga dikirim terpisah setelah ruang lingkup disepakati"}
		return s
	}
	s.Subtitle = "Berdasarkan penawaran " + q.QuoteNumber
	for _, it := range q.Items {
		s.Bullets = append(s.Bullets, it.Description+": "+it.LineTotal.Rupiah())
	}
	if q.DiscountNominal > 0 {
		s.Bullets = append(s.Bullets, "Diskon: -"+q.DiscountNominal.Rupiah())
	}
	if q.UsePpn {
		s.Bullets = append(s.Bullets, "PPN "+FormatNumber(q.PpnPercent, 2)+"%: "+q.PpnNominal.Rupiah())
	}
	s.Bullets = append(s.Bullets, "Total: "+q.Total.Rupiah())
	return s
}
