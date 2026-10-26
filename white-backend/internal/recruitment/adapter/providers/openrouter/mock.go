package openrouter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

// The mock reads the transcript and profile with fixed rules: same input,
// same result.

// skillKeywords are the agency skills the mock recognizes, in report order.
var skillKeywords = []string{
	"Instagram", "TikTok", "YouTube", "LinkedIn", "Facebook", "Copywriting", "Content planning", "SEO", "Google Analytics",
	"Search Console", "Ahrefs", "Semrush", "Google Ads", "Meta Ads", "TikTok Ads", "Canva", "Figma", "Photoshop",
	"Illustrator", "Premiere", "CapCut", "After Effects", "Excel", "Spreadsheet", "WordPress", "HTML", "CRM", "Negosiasi",
	"Presentasi", "Prospecting",
}

var salaryPattern = regexp.MustCompile(`(?i)(rp\.?\s*)?\d[\d.,]*\s*(juta|jt|ribu|rb|k)?(\s*(-|sampai|hingga)\s*(rp\.?\s*)?\d[\d.,]*\s*(juta|jt|ribu|rb|k)?)?`)

func skillsIn(text string) []string {
	lower := strings.ToLower(text)
	var out []string
	for _, k := range skillKeywords {
		if strings.Contains(lower, strings.ToLower(k)) && len(out) < 8 {
			out = append(out, k)
		}
	}
	return out
}

func answerOf(t domain.Turn) string {
	if t.Answer == nil {
		return ""
	}
	return strings.TrimSpace(*t.Answer)
}

// MockSummary builds the indicative summary from the transcript.
func MockSummary(t domain.Transcript) domain.InterviewSummary {
	var all []string
	answered, long, totalLen := 0, 0, 0
	var flags []string
	salary := domain.SalaryExpectation{Note: "Topik gaji belum tertanya."}
	for _, turn := range t.Turns {
		a := answerOf(turn)
		switch {
		case a == "":
			flags = append(flags, fmt.Sprintf("Pertanyaan #%d tidak dijawab", turn.TurnNo))
		case domain.UTF16Len(a) < 20:
			flags = append(flags, fmt.Sprintf("Jawaban #%d sangat singkat", turn.TurnNo))
		}
		if a != "" {
			answered++
			totalLen += domain.UTF16Len(a)
			all = append(all, a)
		}
		if domain.UTF16Len(a) >= 40 {
			long++
		}
		if turn.Topic != nil && *turn.Topic == "gaji" {
			if m := salaryPattern.FindString(a); strings.TrimSpace(m) != "" && strings.ContainsAny(m, "0123456789") {
				v := domain.JSSlice(strings.TrimSpace(m), 120)
				salary = domain.SalaryExpectation{Mentioned: true, Value: &v, Note: "Disebutkan kandidat pada pertanyaan gaji."}
			} else if a != "" {
				salary = domain.SalaryExpectation{Note: "Kandidat belum menyebut angka yang jelas."}
			}
		}
	}
	skills := skillsIn(strings.Join(all, " "))
	score := domain.ClampScore(float64(40 + 8*long + 3*len(skills) - 10*len(flags)))
	conclusion := "kurang_relevan"
	switch {
	case score >= 75:
		conclusion = "relevan"
	case score >= 50:
		conclusion = "cukup_relevan"
	}
	style := "singkat"
	if answered > 0 && totalLen/answered >= 80 {
		style = "rinci"
	}
	position := t.PositionTitle
	if position == "" {
		position = "yang dilamar"
	}
	summary := fmt.Sprintf("%s menjawab %d dari %d pertanyaan interview AI untuk posisi %s. Jawaban kandidat cenderung %s.",
		t.CandidateName, answered, len(t.Turns), position, style)
	if len(skills) > 0 {
		summary += " Keahlian yang disebut: " + strings.Join(skills, ", ") + "."
	}
	follow := []string{"Gali contoh hasil kerja nyata yang relevan dengan posisi.", "Konfirmasi kesiapan tanggal mulai dan jam kerja."}
	if !salary.Mentioned {
		follow = append(follow, "Tanyakan ekspektasi gaji dalam angka yang jelas.")
	}
	if len(skills) == 0 {
		follow = append(follow, "Minta kandidat menyebut tools yang dikuasai.")
	}
	return domain.InterviewSummary{
		Summary: summary,
		Relevance: domain.Relevance{Score: score, Conclusion: conclusion,
			Reason: fmt.Sprintf("Skor mengindikasikan %d jawaban yang cukup rinci dan %d keahlian relevan yang disebut kandidat.", long, len(skills))},
		Skills:      nonNil(skills),
		Salary:      salary,
		RedFlags:    nonNil(flags),
		FollowUps:   follow,
		Limitations: domain.DefaultLimitations,
	}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

var wordPattern = regexp.MustCompile(`[a-z0-9]{4,}`)

var stopWords = map[string]bool{"yang": true, "dengan": true, "untuk": true, "atau": true, "dari": true, "dalam": true, "pada": true,
	"posisi": true, "deskripsi": true, "persyaratan": true, "minimal": true, "tahun": true, "pengalaman": true, "mampu": true,
	"memiliki": true, "bisa": true, "akan": true, "kami": true, "anda": true, "serta": true, "lebih": true, "tim": true}

func keywords(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range wordPattern.FindAllString(strings.ToLower(text), -1) {
		if !stopWords[w] {
			out[w] = true
		}
	}
	return out
}

// MockAnalysis scores the profile by keyword overlap with the opening.
func MockAnalysis(in domain.AnalysisInput) domain.CandidateAnalysis {
	profile := keywords(in.Profile)
	var matched []string
	for _, k := range skillKeywords {
		lk := strings.ToLower(k)
		if strings.Contains(strings.ToLower(in.Profile), lk) && (in.JobContext == "" || strings.Contains(strings.ToLower(in.JobContext), lk)) {
			matched = append(matched, k)
		}
	}
	overlap := 0
	for w := range keywords(in.JobContext) {
		if profile[w] {
			overlap++
		}
	}
	hasEdu := strings.Contains(in.Profile, "Pendidikan terakhir:")
	hasExp := strings.Contains(in.Profile, "Pengalaman terakhir:")
	score := 35 + 4*min(overlap, 8) + 5*min(len(matched), 4)
	if hasEdu {
		score += 5
	}
	if hasExp {
		score += 10
	}
	var strengths, gaps []string
	if len(matched) > 0 {
		strengths = append(strengths, "Menyebut keahlian yang dibutuhkan: "+strings.Join(matched, ", "))
	}
	if hasExp {
		strengths = append(strengths, "Memiliki pengalaman kerja yang tercatat")
	} else {
		gaps = append(gaps, "Profil belum mencantumkan pengalaman kerja")
	}
	if !hasEdu {
		gaps = append(gaps, "Pendidikan terakhir belum diisi")
	}
	reason := fmt.Sprintf("Skor dihitung dari %d kata kunci lowongan yang muncul di profil dan %d keahlian yang cocok.", overlap, len(matched))
	if in.JobContext == "" {
		reason = "Deskripsi pekerjaan tidak tersedia, skor dinilai dari kelengkapan profil."
		gaps = append(gaps, "Kandidat belum terhubung ke lowongan")
	}
	if len(matched) == 0 && in.JobContext != "" {
		gaps = append(gaps, "Belum terlihat keahlian spesifik yang diminta lowongan")
	}
	return domain.CandidateAnalysis{
		Summary:     strings.ReplaceAll(strings.TrimSpace(in.Profile), "\n", "; ") + ".",
		MatchScore:  domain.ClampScore(float64(score)),
		MatchReason: reason,
		Strengths:   nonNil(strengths),
		Gaps:        nonNil(gaps),
	}
}
