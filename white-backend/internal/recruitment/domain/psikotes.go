package domain

import (
	"math"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Psikotes scoring and question shaping (psikotes-scoring.ts,
// psikotes-runner.ts). Answer keys never leave the server; scores are
// computed here and snapshotted into psikotes_session_tests.score_detail.

// Instrument kinds.
const (
	KindMcq          = "mcq"
	KindForcedChoice = "forced_choice"
	KindDrawing      = "drawing"
)

// PapiScaleCodes are the 20 PAPI Kostick scales in declaration order.
var PapiScaleCodes = []string{"N", "G", "A", "L", "P", "I", "T", "V", "X", "S", "B", "O", "R", "D", "C", "Z", "E", "K", "F", "W"}

var papiScaleLabels = map[string]string{
	"N": "Kebutuhan Menyelesaikan Tugas",
	"G": "Peran Pekerja Keras",
	"A": "Dorongan Berprestasi",
	"L": "Peran Kepemimpinan",
	"P": "Kebutuhan Mengatur Orang Lain",
	"I": "Peran Membuat Keputusan",
	"T": "Peran Sibuk",
	"V": "Peran Penuh Semangat",
	"X": "Kebutuhan Diperhatikan",
	"S": "Peran Hubungan Sosial",
	"B": "Kebutuhan Diterima Kelompok",
	"O": "Kedekatan & Kasih Sayang",
	"R": "Peran Teoretis",
	"D": "Peran Bekerja dengan Detail",
	"C": "Peran Keteraturan",
	"Z": "Kebutuhan akan Perubahan",
	"E": "Pengendalian Emosi",
	"K": "Kebutuhan Agresi",
	"F": "Dukungan terhadap Atasan",
	"W": "Kebutuhan Aturan & Arahan",
}

// TerminalTestStatuses are the session test statuses that end a test.
var TerminalTestStatuses = []string{"selesai", "perlu_review", "reviewed"}

// IsTerminalTest reports whether a session test is finished.
func IsTerminalTest(status string) bool { return slices.Contains(TerminalTestStatuses, status) }

// IsReviewable reports whether a test sits in the manual review queue.
func IsReviewable(status string) bool { return status == "perlu_review" || status == "reviewed" }

var testStatusLabels = map[string]string{
	"pending": "Belum dikerjakan", "in_progress": "Sedang dikerjakan", "selesai": "Selesai",
	"perlu_review": "Perlu Review", "reviewed": "Sudah Direview",
}

// TestStatusLabel is the label of a session test status.
func TestStatusLabel(s string) string {
	if l, ok := testStatusLabels[s]; ok {
		return l
	}
	return s
}

// McqQuestion is a question with its answer key (nil when absent).
type McqQuestion struct {
	ID      string
	Correct *string
}

// McqItem is one per_question entry of an MCQ score.
type McqItem struct {
	ID         string  `json:"id"`
	Given      *string `json:"given"`
	CorrectKey string  `json:"correct_key"`
	IsCorrect  bool    `json:"is_correct"`
}

// McqDetail is the MCQ score_detail snapshot.
type McqDetail struct {
	Total       int       `json:"total"`
	Correct     int       `json:"correct"`
	PerQuestion []McqItem `json:"per_question"`
}

// ScoreMcq scores 0..100 (rounded half up) over the questions with a key.
func ScoreMcq(questions []McqQuestion, answers map[string]string) (int, McqDetail) {
	detail := McqDetail{PerQuestion: []McqItem{}}
	for _, q := range questions {
		if q.Correct == nil || *q.Correct == "" {
			continue
		}
		item := McqItem{ID: q.ID, CorrectKey: *q.Correct}
		if given, ok := answers[q.ID]; ok {
			item.Given = &given
			item.IsCorrect = given == *q.Correct
		}
		if item.IsCorrect {
			detail.Correct++
		}
		detail.PerQuestion = append(detail.PerQuestion, item)
	}
	detail.Total = len(detail.PerQuestion)
	if detail.Total == 0 {
		return 0, detail
	}
	return int(math.Floor(float64(detail.Correct)/float64(detail.Total)*100 + 0.5)), detail
}

// PapiQuestion is a forced-choice pair with the scale of each statement
// ("" when the pair is malformed).
type PapiQuestion struct {
	ID     string
	ScaleA string
	ScaleB string
}

// PapiDominant is one top scale.
type PapiDominant struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// PapiResult is the forced-choice score_detail snapshot.
type PapiResult struct {
	Scales   map[string]int `json:"scales"`
	Dominant []PapiDominant `json:"dominant"`
	Answered int            `json:"answered"`
	Total    int            `json:"total"`
}

// ScorePapi counts the chosen scale of every answered pair; the dominant
// scales are the ties at the highest count, in scale order.
func ScorePapi(questions []PapiQuestion, answers map[string]string) PapiResult {
	res := PapiResult{Scales: map[string]int{}, Dominant: []PapiDominant{}}
	for _, c := range PapiScaleCodes {
		res.Scales[c] = 0
	}
	for _, q := range questions {
		if q.ScaleA == "" || q.ScaleB == "" {
			continue
		}
		res.Total++
		scale := ""
		switch answers[q.ID] {
		case "a":
			scale = q.ScaleA
		case "b":
			scale = q.ScaleB
		default:
			continue
		}
		if _, ok := papiScaleLabels[scale]; !ok {
			continue
		}
		res.Scales[scale]++
		res.Answered++
	}
	top := 0
	for _, n := range res.Scales {
		top = max(top, n)
	}
	if top == 0 {
		return res
	}
	for _, c := range PapiScaleCodes {
		if res.Scales[c] == top {
			res.Dominant = append(res.Dominant, PapiDominant{Code: c, Label: papiScaleLabels[c], Count: top})
		}
	}
	return res
}

// ScoreDetail is what score_detail stores: MCQ or PAPI, never both.
type ScoreDetail struct {
	// MCQ
	Total       int       `json:"total,omitempty"`
	Correct     int       `json:"correct,omitempty"`
	PerQuestion []McqItem `json:"per_question,omitempty"`
	// PAPI
	Scales    map[string]int `json:"scales,omitempty"`
	Dominant  []PapiDominant `json:"dominant,omitempty"`
	Answered  int            `json:"answered,omitempty"`
	PapiTotal int            `json:"papi_total,omitempty"`
}

// McqScoreDetail wraps an MCQ snapshot.
func McqScoreDetail(d McqDetail) ScoreDetail {
	return ScoreDetail{Total: d.Total, Correct: d.Correct, PerQuestion: d.PerQuestion}
}

// PapiScoreDetail wraps a PAPI snapshot.
func PapiScoreDetail(r PapiResult) ScoreDetail {
	return ScoreDetail{Scales: r.Scales, Dominant: r.Dominant, Answered: r.Answered, PapiTotal: r.Total}
}

// ─── questions ───────────────────────────────────────────────

// McqOption is one MCQ choice.
type McqOption struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

// PapiStatement is one side of a forced-choice pair.
type PapiStatement struct {
	Text  string `json:"text"`
	Scale string `json:"scale"`
}

// PapiPair is the stored options of a forced-choice question.
type PapiPair struct {
	A PapiStatement `json:"a"`
	B PapiStatement `json:"b"`
}

// Question is a bank question. MCQ questions use Options and CorrectKey,
// forced-choice ones use Pair.
type Question struct {
	ID           string
	InstrumentID string
	Body         string
	Options      []McqOption
	Pair         *PapiPair
	CorrectKey   string
	SortOrder    int
	IsActive     bool
}

// Sanitized strips the answer key and PAPI scale codes for the candidate.
func (q Question) Sanitized() Question {
	q.CorrectKey = ""
	if q.Pair != nil {
		p := PapiPair{A: PapiStatement{Text: q.Pair.A.Text}, B: PapiStatement{Text: q.Pair.B.Text}}
		q.Pair = &p
	}
	return q
}

// PickDrawnAnswers keeps only the answers for questions drawn for the test.
func PickDrawnAnswers(questionIDs []string, answers map[string]string) map[string]string {
	out := map[string]string{}
	for qid, a := range answers {
		if slices.Contains(questionIDs, qid) {
			out[qid] = a
		}
	}
	return out
}

// MaxAnswers caps the answers of one save (sessionAnswersSchema).
const MaxAnswers = 500

// CleanAnswers is sessionAnswersSchema: at most 500 entries, each choice
// trimmed to 1..4 characters. ok is false on any failure.
func CleanAnswers(in map[string]string) (map[string]string, bool) {
	if len(in) > MaxAnswers {
		return nil, false
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		v = strings.TrimSpace(v)
		if k == "" || UTF16Len(v) < 1 || UTF16Len(v) > 4 {
			return nil, false
		}
		out[k] = v
	}
	return out, true
}

var mcqKeys = []string{"a", "b", "c", "d", "e", "f"}

// QuestionInput is a bank question as HR submits it.
type QuestionInput struct {
	Body       string
	Options    []McqOption
	StatementA PapiStatement
	StatementB PapiStatement
	CorrectKey string
	SortOrder  int
	IsActive   bool
}

func validateSortOrder(v *validate.Validator, n int) {
	if n < 0 {
		v.Add("sort_order", "Urutan tidak boleh negatif")
	} else if n > 10000 {
		v.Add("sort_order", "Urutan maksimal 10000")
	}
}

// ValidateMcqQuestion is parseMcqQuestion: body 1..2000, 2 to 6 options
// with unique keys a..f, and a key among the options.
func ValidateMcqQuestion(in *QuestionInput) error {
	v := validate.New()
	in.Body = strings.TrimSpace(in.Body)
	switch {
	case in.Body == "":
		v.Add("body", "Soal wajib diisi")
	case UTF16Len(in.Body) > 2000:
		v.Add("body", "Soal maksimal 2000 karakter")
	}
	keys := map[string]bool{}
	dup := false
	for i := range in.Options {
		o := &in.Options[i]
		o.Text = strings.TrimSpace(o.Text)
		if !slices.Contains(mcqKeys, o.Key) {
			v.Add("options", "Key opsi harus a sampai f")
		}
		if o.Text == "" {
			v.Add("options", "Teks opsi wajib diisi")
		} else if UTF16Len(o.Text) > 500 {
			v.Add("options", "Teks opsi maksimal 500 karakter")
		}
		dup = dup || keys[o.Key]
		keys[o.Key] = true
	}
	switch {
	case len(in.Options) < 2:
		v.Add("options", "Minimal 2 opsi")
	case len(in.Options) > 6:
		v.Add("options", "Maksimal 6 opsi")
	case dup && !v.HasErrors():
		v.Add("options", "Key opsi tidak boleh duplikat")
	}
	validateSortOrder(v, in.SortOrder)
	if !v.HasErrors() && !keys[in.CorrectKey] {
		v.Add("answer_key", "Kunci jawaban harus salah satu key opsi")
	}
	return v.Err()
}

// ValidatePapiQuestion is parsePapiQuestion: two statements with distinct
// PAPI scales, an optional note of 500 characters.
func ValidatePapiQuestion(in *QuestionInput) error {
	v := validate.New()
	in.Body = strings.TrimSpace(in.Body)
	if UTF16Len(in.Body) > 500 {
		v.Add("body", "Keterangan maksimal 500 karakter")
	}
	for _, s := range []struct {
		field string
		st    *PapiStatement
	}{{"options.a", &in.StatementA}, {"options.b", &in.StatementB}} {
		s.st.Text = strings.TrimSpace(s.st.Text)
		if s.st.Text == "" {
			v.Add(s.field, "Pernyataan wajib diisi")
		} else if UTF16Len(s.st.Text) > 500 {
			v.Add(s.field, "Pernyataan maksimal 500 karakter")
		}
		if _, ok := papiScaleLabels[s.st.Scale]; !ok {
			v.Add(s.field, "Skala PAPI tidak valid")
		}
	}
	validateSortOrder(v, in.SortOrder)
	if !v.HasErrors() && in.StatementA.Scale == in.StatementB.Scale {
		v.Add("options.b", "Skala pernyataan A dan B tidak boleh sama")
	}
	return v.Err()
}

// ValidateReview is parseReview: the review conclusion, 1..2000 characters.
func ValidateReview(notes string) (string, error) {
	notes = strings.TrimSpace(notes)
	switch {
	case notes == "":
		return "", apperr.Field("review_notes", "Kesimpulan review wajib diisi")
	case UTF16Len(notes) > 2000:
		return "", apperr.Field("review_notes", "Kesimpulan maksimal 2000 karakter")
	}
	return notes, nil
}

// ─── instruments ─────────────────────────────────────────────

// Instrument is a psikotes instrument and its runner config.
type Instrument struct {
	ID              string
	Code            string
	Name            string
	Kind            string
	DurationSeconds int
	QuestionCount   *int
	Shuffle         bool
	Instructions    string
	IsActive        bool
	SortOrder       int
	ActiveQuestions int
}

// InstrumentPatch is parseInstrumentUpdate: only the set fields change.
type InstrumentPatch struct {
	Name               *string
	IsActive           *bool
	DurationSeconds    *int
	QuestionCount      *int
	ClearQuestionCount bool
	Shuffle            *bool
	Instructions       *string
}

// Validate applies the schema messages; an empty patch is refused.
func (p *InstrumentPatch) Validate() error {
	v := validate.New()
	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		p.Name = &n
		if n == "" {
			v.Add("name", "Nama wajib diisi")
		} else if UTF16Len(n) > 100 {
			v.Add("name", "Nama maksimal 100 karakter")
		}
	}
	if d := p.DurationSeconds; d != nil {
		if *d < 30 {
			v.Add("config.duration_seconds", "Durasi minimal 30 detik")
		} else if *d > 14400 {
			v.Add("config.duration_seconds", "Durasi maksimal 4 jam")
		}
	}
	if c := p.QuestionCount; c != nil {
		if *c < 1 {
			v.Add("config.question_count", "Jumlah soal minimal 1")
		} else if *c > 200 {
			v.Add("config.question_count", "Jumlah soal maksimal 200")
		}
	}
	if s := p.Instructions; s != nil {
		t := strings.TrimSpace(*s)
		p.Instructions = &t
		if UTF16Len(t) > 2000 {
			v.Add("config.instructions", "Instruksi maksimal 2000 karakter")
		}
	}
	if v.HasErrors() {
		return v.Err()
	}
	if p.Name == nil && p.IsActive == nil && p.DurationSeconds == nil && p.QuestionCount == nil && !p.ClearQuestionCount &&
		p.Shuffle == nil && p.Instructions == nil {
		return apperr.New(apperr.Validation, "Tidak ada perubahan yang dikirim")
	}
	return nil
}

// ─── invitations ─────────────────────────────────────────────

// ValidateInvite is parsePsikotesInvite: 1 to 10 distinct instruments and
// a link of 1 to 30 days (3 by default).
func ValidateInvite(instrumentIDs []string, expiresDays *int) error {
	v := validate.New()
	switch {
	case len(instrumentIDs) < 1:
		v.Add("instrument_ids", "Pilih minimal satu instrumen")
	case len(instrumentIDs) > 10:
		v.Add("instrument_ids", "Maksimal 10 instrumen")
	default:
		seen := map[string]bool{}
		for _, id := range instrumentIDs {
			if id == "" {
				v.Add("instrument_ids", "ID instrumen tidak valid")
				break
			}
			if seen[id] {
				v.Add("instrument_ids", "Instrumen duplikat")
				break
			}
			seen[id] = true
		}
	}
	validateExpiresDays(v, expiresDays, 3)
	return v.Err()
}

// validateExpiresDays defaults 0 to def and checks 1..30.
func validateExpiresDays(v *validate.Validator, days *int, def int) {
	if *days == 0 {
		*days = def
	}
	if *days < 1 {
		v.Add("expires_days", "Minimal 1 hari")
	} else if *days > 30 {
		v.Add("expires_days", "Maksimal 30 hari")
	}
}

// ─── portal runner ───────────────────────────────────────────

// AnswerGrace is the slack after the deadline for the last autosave.
const AnswerGrace = 30 * time.Second

// TestDeadline is started_at plus the instrument duration (nil before start).
func TestDeadline(startedAt *time.Time, durationSeconds int) *time.Time {
	if startedAt == nil {
		return nil
	}
	end := startedAt.Add(time.Duration(durationSeconds) * time.Second)
	return &end
}

// PastAnswerGrace reports a test whose deadline plus grace has passed.
func PastAnswerGrace(startedAt *time.Time, durationSeconds int, now time.Time) bool {
	d := TestDeadline(startedAt, durationSeconds)
	return d != nil && now.UnixMilli() > d.Add(AnswerGrace).UnixMilli()
}

// DrawLimit is how many questions a test draws: question_count (at least
// 1) for MCQ, every active question otherwise (0).
func DrawLimit(kind string, questionCount *int) int {
	if kind != KindMcq || questionCount == nil || *questionCount == 0 {
		return 0
	}
	return max(1, *questionCount)
}

// Drawing upload limits.
const MaxDrawingBytes = 8 * 1024 * 1024

// Proctor event types per portal (snapshots are not stored by this port).
var (
	PsikotesProctorEvents  = []string{"tab_blur", "fullscreen_exit", "paste", "disconnect"}
	InterviewProctorEvents = []string{"tab_blur", "fullscreen_exit", "paste", "disconnect", "face_not_detected", "multiple_faces", "camera_off"}
)

// ValidateProctorMeta is the meta record rule: at most 20 entries, keys of
// 50 and values of 200 characters.
func ValidateProctorMeta(meta map[string]string) error {
	if len(meta) > 20 {
		return apperr.Field("meta", "Meta terlalu besar")
	}
	for k, val := range meta {
		if UTF16Len(k) > 50 || UTF16Len(val) > 200 {
			return apperr.Field("meta", "Meta tidak valid")
		}
	}
	return nil
}
