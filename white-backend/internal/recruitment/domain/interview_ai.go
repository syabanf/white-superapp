package domain

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// The candidate's AI interview (interview-runner.ts, interview-ai.ts): one
// basic screening question per turn over five topics, then an indicative
// summary for HR. This port takes text answers only.

const (
	MaxQuestionsDefault = 8
	MaxQuestionsLimit   = 15
	MaxAnswerChars      = 4000
	MaxQuestionChars    = 600
)

// AiSession is an AI interview invitation.
type AiSession struct {
	ID            string
	CandidateID   string
	CandidateName string
	PositionTitle string
	Token         string
	Status        string
	WebcamConsent *bool
	MaxQuestions  int
	Summary       *InterviewSummary
	SummaryModel  string
	InvitedAt     *time.Time
	ExpiresAt     *time.Time
	StartedAt     *time.Time
	CompletedAt   *time.Time
	CreatedBy     *string
	CreatedByName string
	Turns         []Turn
	Proctor       ProctorTally
}

// Turn is one question and its answer.
type Turn struct {
	ID         string
	SessionID  string
	TurnNo     int
	Topic      *string
	Question   string
	Answer     *string
	AnswerMode *string
	AskedAt    time.Time
	AnsweredAt *time.Time
}

// Transcript is what the interviewer AI sees.
type Transcript struct {
	CandidateName string
	PositionTitle string
	MaxQuestions  int
	Turns         []Turn
}

// ProctorTally counts a session's proctoring flags.
type ProctorTally struct {
	Flags int
}

// ClampMaxQuestions is sessionMaxQuestions: below 3 falls back to 8, above
// 15 is capped.
func ClampMaxQuestions(n int) int {
	if n < 3 {
		return MaxQuestionsDefault
	}
	return min(MaxQuestionsLimit, n)
}

// ValidateAiInvite is parseInterviewInvite: 1..30 days (7) and 3..15
// questions (8).
func ValidateAiInvite(expiresDays, maxQuestions *int) error {
	v := validate.New()
	validateExpiresDays(v, expiresDays, 7)
	if *maxQuestions == 0 {
		*maxQuestions = MaxQuestionsDefault
	}
	if *maxQuestions < 3 || *maxQuestions > MaxQuestionsLimit {
		v.Add("max_questions", "Jumlah pertanyaan 3 sampai 15")
	}
	return v.Err()
}

// AiInviteDescription is the activity of an AI interview invitation.
func AiInviteDescription(maxQuestions, expiresDays int) string {
	return fmt.Sprintf("Undangan interview AI dibuat (maks %d pertanyaan; berlaku %d hari)", maxQuestions, expiresDays)
}

// AiCompletedDescription is the activity when the interview ends.
func AiCompletedDescription(turns int, summarized bool) string {
	s := "Interview AI selesai (" + strconv.Itoa(turns) + " pertanyaan)"
	if summarized {
		s += " — kesimpulan AI tersedia"
	}
	return s
}

// InterviewTopics are the mandatory topics, in order.
var InterviewTopics = []string{"pembuka", "keahlian", "motivasi", "ketersediaan", "gaji"}

// FallbackQuestions are asked when the AI fails, in topic order.
var FallbackQuestions = map[string]string{
	"pembuka":      "Perkenalkan diri Anda secara singkat, lalu ceritakan pengalaman kerja atau kegiatan yang paling relevan dengan posisi ini.",
	"keahlian":     "Keahlian apa yang paling Anda kuasai dan bagaimana Anda biasa menggunakannya dalam pekerjaan sehari-hari?",
	"motivasi":     "Apa yang membuat Anda tertarik melamar posisi ini?",
	"ketersediaan": "Kapan Anda bisa mulai bekerja, dan apakah ada kendala lokasi atau jam kerja yang perlu kami ketahui?",
	"gaji":         "Berapa ekspektasi gaji Anda untuk posisi ini?",
}

// QuestionPrompt is the interviewer system prompt.
const QuestionPrompt = `Kamu adalah AI interviewer HR yang ramah dan profesional, berbahasa Indonesia.
Tugasmu menanyakan hal-hal BASIC screening ke kandidat, satu pertanyaan per giliran, mencakup topik wajib berikut secara berurutan (boleh follow-up singkat bila jawaban menarik/kurang jelas):
1. pembuka — perkenalan & pengalaman relevan
2. keahlian — keahlian teknis utama & contoh penggunaannya
3. motivasi — alasan melamar & pemahaman posisi
4. ketersediaan — kapan bisa mulai, kendala lokasi/jam kerja
5. gaji — ekspektasi gaji

Balas HANYA dengan JSON valid (tanpa markdown) berbentuk:
{"action": "ask"|"finish", "topic": string|null, "question": string|null}

Ketentuan:
- "topic" salah satu dari: pembuka, keahlian, motivasi, ketersediaan, gaji.
- Pertanyaan singkat (1-2 kalimat), sopan, mudah dipahami, TANPA menyebut skor/penilaian.
- Maksimal satu follow-up per topik; setelah semua topik tertanya (terutama gaji sudah ditanya), balas {"action":"finish","topic":null,"question":null}.
- Jangan menanyakan SARA, status pernikahan, agama, atau hal di luar konteks pekerjaan.
- Jangan mengulang pertanyaan yang sudah ditanyakan.`

// SummaryPrompt is the summary system prompt.
const SummaryPrompt = `Kamu adalah asisten HR yang menyimpulkan hasil interview screening basic secara INDIKATIF.
Balas HANYA dengan JSON valid (tanpa markdown) berbentuk:
{
  "ringkasan": string,
  "relevansi": {"skor": number, "kesimpulan": "relevan"|"cukup_relevan"|"kurang_relevan", "alasan": string},
  "keahlian": [string],
  "ekspektasi_gaji": {"disebutkan": boolean, "nilai": string|null, "catatan": string},
  "red_flags": [string],
  "perhatikan_saat_interview_lanjutan": [string],
  "keterbatasan": string
}
Ketentuan:
- Dasarkan HANYA pada transkrip — jangan mengarang informasi yang tidak disebutkan kandidat.
- "ringkasan": 2-4 kalimat bahasa Indonesia, nada netral-profesional.
- "relevansi.skor": 0-100 kecocokan jawaban kandidat dengan posisi (pakai kata "cenderung/mengindikasikan" di alasan, hindari klaim pasti).
- "keahlian": daftar keahlian yang DISEBUT kandidat (maks 8).
- "ekspektasi_gaji.nilai": angka/rentang persis seperti disebut kandidat, null bila tidak disebut.
- "red_flags": hal yang perlu diwaspadai (jawaban kosong/berputar, ketidaksesuaian, dsb) — boleh kosong.
- "perhatikan_saat_interview_lanjutan": 2-5 hal konkret utk digali interviewer manusia.
- "keterbatasan": 1-2 kalimat menegaskan ini kesimpulan indikatif dari interview basic oleh AI, bukan keputusan final — keputusan tetap di tangan HRD.`

// DefaultLimitations is the keterbatasan fallback.
const DefaultLimitations = "Kesimpulan indikatif dari interview basic oleh AI — keputusan tetap di tangan HRD."

// NextQ is a question to ask; an empty question finishes the interview.
type NextQ struct {
	Question string
	Topic    *string
}

// Done reports the end of the interview.
func (q NextQ) Done() bool { return q.Question == "" }

func ask(topic, question string) NextQ { return NextQ{Question: question, Topic: &topic} }

func askedTopics(turns []Turn) []string {
	var topics []string
	for _, t := range turns {
		if t.Topic != nil && *t.Topic != "" && !slices.Contains(topics, *t.Topic) {
			topics = append(topics, *t.Topic)
		}
	}
	return topics
}

// PlanNextQuestion decides what generateNextInterviewQuestion can without
// the AI: the end after maxQ turns, the static greeting first, and gaji
// forced into the last slot. needAI asks the caller for an AI step, then
// ResolveNextQuestion.
func PlanNextQuestion(name, position string, maxQ int, turns []Turn) (q NextQ, needAI bool) {
	asked := len(turns)
	if asked >= maxQ {
		return NextQ{}, false
	}
	if asked == 0 {
		at := ""
		if position != "" {
			at = " posisi " + position
		}
		return ask("pembuka", "Halo "+name+"! Terima kasih sudah meluangkan waktu untuk interview"+at+". "+FallbackQuestions["pembuka"]), false
	}
	if !slices.Contains(askedTopics(turns), "gaji") && asked >= maxQ-1 {
		return ask("gaji", FallbackQuestions["gaji"]), false
	}
	return NextQ{}, true
}

// InterviewStepPrompt is the user turn of the interviewer prompt.
func InterviewStepPrompt(name, position string, maxQ int, turns []Turn) string {
	lines := []string{"KANDIDAT: " + name}
	if position != "" {
		lines = append(lines, "POSISI YANG DILAMAR: "+position)
	}
	askedList := strings.Join(askedTopics(turns), ", ")
	if askedList == "" {
		askedList = "(belum ada)"
	}
	lines = append(lines,
		"PERTANYAAN TERPAKAI: "+strconv.Itoa(len(turns))+"/"+strconv.Itoa(maxQ)+" (sisakan slot utk topik yang belum tertanya)",
		"TOPIK SUDAH DITANYA: "+askedList,
		"\nPERCAKAPAN SEJAUH INI:\n"+TranscriptContext(turns),
		"\nTentukan langkah berikutnya.")
	return strings.Join(lines, "\n")
}

// AIStep is the interviewer's JSON answer.
type AIStep struct {
	Action   string  `json:"action"`
	Topic    *string `json:"topic"`
	Question *string `json:"question"`
}

// ResolveNextQuestion applies an AI step (nil when the AI is unavailable or
// failed): "finish" before gaji still asks gaji, and a missing step falls
// back to the first topic not asked yet so the interview never stalls.
func ResolveNextQuestion(turns []Turn, step *AIStep) NextQ {
	topics := askedTopics(turns)
	gajiAsked := slices.Contains(topics, "gaji")
	if step != nil {
		switch {
		case step.Action == "finish" && !gajiAsked:
			return ask("gaji", FallbackQuestions["gaji"])
		case step.Action == "finish":
			return NextQ{}
		case step.Action == "ask" && step.Question != nil && strings.TrimSpace(*step.Question) != "":
			q := NextQ{Question: JSSlice(strings.TrimSpace(*step.Question), MaxQuestionChars)}
			if step.Topic != nil && slices.Contains(InterviewTopics, *step.Topic) {
				t := *step.Topic
				q.Topic = &t
			}
			return q
		}
	}
	for _, topic := range InterviewTopics {
		if !slices.Contains(topics, topic) {
			return ask(topic, FallbackQuestions[topic])
		}
	}
	return NextQ{}
}

// TranscriptContext renders the turns for the prompts (12000 characters).
func TranscriptContext(turns []Turn) string {
	if len(turns) == 0 {
		return "(belum ada tanya-jawab)"
	}
	blocks := make([]string, len(turns))
	for i, t := range turns {
		head := "#" + strconv.Itoa(t.TurnNo)
		if t.Topic != nil && *t.Topic != "" {
			head += " [" + *t.Topic + "]"
		}
		answer := ""
		if t.Answer != nil {
			answer = strings.TrimSpace(*t.Answer)
		}
		if answer == "" {
			answer = "(tidak menjawab)"
		}
		blocks[i] = head + "\nAI: " + t.Question + "\nKandidat: " + answer
	}
	return JSSlice(strings.Join(blocks, "\n\n"), 12_000)
}

// SummaryInput is the user turn of the summary prompt.
func SummaryInput(name, position string, turns []Turn) string {
	pos := "POSISI: (tidak diisi)"
	if position != "" {
		pos = "POSISI YANG DILAMAR: " + position
	}
	return "KANDIDAT: " + name + "\n" + pos + "\n\nTRANSKRIP INTERVIEW:\n" + TranscriptContext(turns)
}

// ValidateAnswer trims a text answer to 4000 characters; empty is refused.
func ValidateAnswer(text string) (string, bool) {
	text = JSSlice(strings.TrimSpace(text), MaxAnswerChars)
	return text, text != ""
}

// ─── summary ─────────────────────────────────────────────────

// Relevance is the relevansi block.
type Relevance struct {
	Score      int    `json:"skor"`
	Conclusion string `json:"kesimpulan"`
	Reason     string `json:"alasan"`
}

// SalaryExpectation is the ekspektasi_gaji block.
type SalaryExpectation struct {
	Mentioned bool    `json:"disebutkan"`
	Value     *string `json:"nilai"`
	Note      string  `json:"catatan"`
}

// InterviewSummary is the stored ai_summary (NüHabit JSON keys).
type InterviewSummary struct {
	Summary     string            `json:"ringkasan"`
	Relevance   Relevance         `json:"relevansi"`
	Skills      []string          `json:"keahlian"`
	Salary      SalaryExpectation `json:"ekspektasi_gaji"`
	RedFlags    []string          `json:"red_flags"`
	FollowUps   []string          `json:"perhatikan_saat_interview_lanjutan"`
	Limitations string            `json:"keterbatasan"`
}

var relevanceConclusions = []string{"relevan", "cukup_relevan", "kurang_relevan"}

var relevanceLabels = map[string]string{"relevan": "Relevan", "cukup_relevan": "Cukup Relevan", "kurang_relevan": "Kurang Relevan"}

// RelevanceLabel is the label of a conclusion.
func RelevanceLabel(s string) string {
	if l, ok := relevanceLabels[s]; ok {
		return l
	}
	return s
}

// NormalizeSummary is summarizeInterview's normalization of the decoded
// model answer: clamped score, a known conclusion, capped lists and texts.
func NormalizeSummary(raw map[string]any) InterviewSummary {
	rel, _ := raw["relevansi"].(map[string]any)
	gaji, _ := raw["ekspektasi_gaji"].(map[string]any)
	conclusion, _ := rel["kesimpulan"].(string)
	if !slices.Contains(relevanceConclusions, conclusion) {
		conclusion = "cukup_relevan"
	}
	out := InterviewSummary{
		Summary: JSSlice(strOr(raw["ringkasan"], ""), 2000),
		Relevance: Relevance{
			Score:      ClampScore(numberOf(rel["skor"])),
			Conclusion: conclusion,
			Reason:     JSSlice(strOr(rel["alasan"], ""), 2000),
		},
		Skills: stringList(raw["keahlian"], 8, 120),
		Salary: SalaryExpectation{
			Mentioned: truthy(gaji["disebutkan"]),
			Note:      JSSlice(strOr(gaji["catatan"], ""), 500),
		},
		RedFlags:    stringList(raw["red_flags"], 6, 300),
		FollowUps:   stringList(raw["perhatikan_saat_interview_lanjutan"], 5, 300),
		Limitations: JSSlice(strOr(raw["keterbatasan"], DefaultLimitations), 500),
	}
	if v := gaji["nilai"]; truthy(v) {
		s := JSSlice(jsString(v), 120)
		out.Salary.Value = &s
	}
	return out
}

// ClampScore rounds half up and clamps to 0..100 (NaN is 0).
func ClampScore(n float64) int {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		return 0
	}
	return int(math.Min(100, math.Max(0, math.Floor(n+0.5))))
}

func numberOf(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return math.NaN()
		}
		return f
	case nil:
		return 0
	}
	return math.NaN()
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case float64:
		return x != 0 && !math.IsNaN(x)
	case string:
		return x != ""
	}
	return true
}

func jsString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	case nil:
		return "null"
	}
	return fmt.Sprint(v)
}

func strOr(v any, fallback string) string {
	if v == nil {
		return fallback
	}
	return jsString(v)
}

// stringList is (Array.isArray(v) ? v : []).slice(0, n).map(s => String(s).slice(0, chars)).
func stringList(v any, n, chars int) []string {
	items, _ := v.([]any)
	out := make([]string, 0, min(n, len(items)))
	for _, it := range items[:min(n, len(items))] {
		out = append(out, JSSlice(jsString(it), chars))
	}
	return out
}

// NormalizeAnalysis reads the analysis JSON: score clamped to 0..100,
// capped texts and at most five strengths and gaps.
func NormalizeAnalysis(raw map[string]any) CandidateAnalysis {
	return CandidateAnalysis{
		Summary:     JSSlice(strOr(raw["ringkasan"], ""), 2000),
		MatchScore:  ClampScore(numberOf(raw["skor_kecocokan"])),
		MatchReason: JSSlice(strOr(raw["alasan_kecocokan"], ""), 1000),
		Strengths:   stringList(raw["kekuatan"], 5, 200),
		Gaps:        stringList(raw["kekurangan"], 5, 200),
	}
}
