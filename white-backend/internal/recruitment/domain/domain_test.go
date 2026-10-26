package domain

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

func ptr[T any](v T) *T { return &v }

func message(err error) string {
	if err == nil {
		return ""
	}
	return apperr.As(err).Message
}

// ─── ported from NüHabit domain_test.go ──────────────────────

func TestPortalToken(t *testing.T) {
	if !IsPortalToken(strings.Repeat("a", 48)) || IsPortalToken(strings.Repeat("a", 47)) || IsPortalToken(strings.Repeat("a", 47)+"/") {
		t.Fatal("token is hex of 48 to 128 characters")
	}
}

func TestIsLinkExpired(t *testing.T) {
	day := 24 * time.Hour
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	at := func(s string) *time.Time { v, _ := time.Parse(time.RFC3339, s); return &v }
	cases := []struct {
		expires, issued *time.Time
		want            bool
	}{
		{at("2026-10-03T23:59:59Z"), nil, true},
		{at("2026-10-05T00:00:00Z"), nil, false},
		{nil, at("2026-10-01T00:00:00Z"), true},
		{nil, at("2026-10-03T12:00:00Z"), false},
		{nil, nil, true},
	}
	for i, c := range cases {
		if got := IsLinkExpired(c.expires, c.issued, day, now); got != c.want {
			t.Errorf("case %d: got %v", i, got)
		}
	}
	if !OfferPortal.ShouldExpire("sent", at("2026-10-03T00:00:00Z"), nil, now) || OfferPortal.ShouldExpire("accepted", at("2026-10-03T00:00:00Z"), nil, now) {
		t.Fatal("only open links expire")
	}
}

func TestOfferResponseDescription(t *testing.T) {
	if got := OfferResponseDescription("decline", 1, nil); got != "Kandidat menolak offer v1 via portal" {
		t.Fatal(got)
	}
	long := OfferResponseDescription("negotiate", 3, ptr(strings.Repeat("x", 400)))
	if long != `Kandidat mengajukan negosiasi offer v3 via portal — "`+strings.Repeat("x", 300)+`"` {
		t.Fatal(long)
	}
	if got := ManualOfferResponseDescription("accepted", 2, "Siap"); got != `Kandidat menerima offer v2 (dicatat manual) — "Siap"` {
		t.Fatal(got)
	}
}

func TestLikePatternEscapesWildcards(t *testing.T) {
	if got := LikePattern(`50%_a`); got != `%50\%\_a%` {
		t.Fatal(got)
	}
}

func TestStatusLabels(t *testing.T) {
	if l, ok := StatusLabel("archived"); !ok || l != "Diarsipkan" {
		t.Fatal("archived is the talent pool archive")
	}
	if _, ok := StatusLabel("constructor"); ok {
		t.Fatal("only own keys are statuses")
	}
	if RecommendationSuffix(ptr("tidak_lolos")) != " (rekomendasi: Tidak Lolos)" || RecommendationSuffix(nil) != "" {
		t.Fatal("recommendation suffix")
	}
	if StageChangeDescription("applied", "screening") != "Tahap diubah: Applied → Screening" {
		t.Fatal(StageChangeDescription("applied", "screening"))
	}
}

func TestScoreMcq(t *testing.T) {
	q := func(id, c string) McqQuestion { return McqQuestion{ID: id, Correct: ptr(c)} }
	score, d := ScoreMcq([]McqQuestion{q("q1", "a"), q("q2", "c")}, map[string]string{"q1": "a", "q2": "c"})
	if score != 100 || d.Correct != 2 || d.Total != 2 {
		t.Fatal("all correct is 100")
	}
	if score, _ := ScoreMcq([]McqQuestion{q("q1", "a"), q("q2", "b"), q("q3", "c")}, map[string]string{"q1": "a", "q2": "b", "q3": "d"}); score != 67 {
		t.Fatalf("2 of 3 rounds to 67, got %d", score)
	}
	score, d = ScoreMcq([]McqQuestion{q("q1", "a"), q("q2", "b")}, map[string]string{"q1": "a"})
	want := []McqItem{{ID: "q1", Given: ptr("a"), CorrectKey: "a", IsCorrect: true}, {ID: "q2", CorrectKey: "b"}}
	if score != 50 || !reflect.DeepEqual(d.PerQuestion, want) {
		t.Fatalf("unanswered is wrong: %d %+v", score, d.PerQuestion)
	}
	if score, d := ScoreMcq(nil, nil); score != 0 || d.Total != 0 {
		t.Fatal("empty is 0")
	}
	if score, _ := ScoreMcq([]McqQuestion{q("q1", "a")}, map[string]string{"q1": "z"}); score != 0 {
		t.Fatal("unknown key is wrong")
	}
	if score, d := ScoreMcq([]McqQuestion{q("q1", "a"), {ID: "q2"}}, map[string]string{"q1": "a"}); d.Total != 1 || score != 100 {
		t.Fatal("questions without a key are skipped")
	}
}

func TestScorePapi(t *testing.T) {
	p := func(id, a, b string) PapiQuestion { return PapiQuestion{ID: id, ScaleA: a, ScaleB: b} }
	r := ScorePapi([]PapiQuestion{p("q1", "A", "G"), p("q2", "A", "L"), p("q3", "P", "A")}, map[string]string{"q1": "a", "q2": "a", "q3": "b"})
	if r.Scales["A"] != 3 || r.Scales["G"] != 0 || r.Answered != 3 || r.Total != 3 || len(r.Scales) != 20 {
		t.Fatalf("%+v", r)
	}
	r = ScorePapi([]PapiQuestion{p("q1", "A", "G"), p("q2", "A", "L")}, map[string]string{"q1": "a", "q2": "a"})
	if !reflect.DeepEqual(r.Dominant, []PapiDominant{{Code: "A", Label: "Dorongan Berprestasi", Count: 2}}) {
		t.Fatalf("%+v", r.Dominant)
	}
	r = ScorePapi([]PapiQuestion{p("q1", "L", "G"), p("q2", "A", "P")}, map[string]string{"q1": "a", "q2": "a"})
	if len(r.Dominant) != 2 || r.Dominant[0].Code != "A" || r.Dominant[1].Code != "L" {
		t.Fatalf("ties follow the PAPI scale order: %+v", r.Dominant)
	}
	r = ScorePapi([]PapiQuestion{p("q1", "A", "G"), p("q2", "L", "P")}, map[string]string{"q1": "x"})
	if r.Answered != 0 || len(r.Dominant) != 0 {
		t.Fatal("invalid choices are ignored")
	}
	r = ScorePapi([]PapiQuestion{p("q1", "A", "G"), {ID: "q2"}}, map[string]string{"q1": "a", "q2": "a"})
	if r.Total != 1 || r.Scales["A"] != 1 {
		t.Fatal("pairs without options are skipped")
	}
}

func TestSanitizedQuestion(t *testing.T) {
	mcq := Question{ID: "q1", Body: "2+2?", Options: []McqOption{{Key: "a", Text: "4"}}, CorrectKey: "a"}.Sanitized()
	if mcq.CorrectKey != "" || !reflect.DeepEqual(mcq.Options, []McqOption{{Key: "a", Text: "4"}}) {
		t.Fatalf("%+v", mcq)
	}
	papi := Question{ID: "p1", Pair: &PapiPair{A: PapiStatement{"Saya rajin", "N"}, B: PapiStatement{"Saya tenang", "E"}}}
	clean := papi.Sanitized()
	if clean.Pair.A != (PapiStatement{Text: "Saya rajin"}) || clean.Pair.B.Scale != "" || papi.Pair.A.Scale != "N" {
		t.Fatal("scales stay on the server and the original is untouched")
	}
}

func TestPickDrawnAnswers(t *testing.T) {
	if got := PickDrawnAnswers([]string{"q1", "q2"}, map[string]string{"q1": "a", "q9": "b"}); !reflect.DeepEqual(got, map[string]string{"q1": "a"}) {
		t.Fatal(got)
	}
	if got := PickDrawnAnswers(nil, map[string]string{"q1": "a"}); len(got) != 0 {
		t.Fatal(got)
	}
	if got, ok := CleanAnswers(map[string]string{"q1": " a "}); !ok || got["q1"] != "a" {
		t.Fatal(got)
	}
	if _, ok := CleanAnswers(map[string]string{"q1": "abcde"}); ok {
		t.Fatal("choices are 1 to 4 characters")
	}
}

func TestSlugifyAndNormalize(t *testing.T) {
	if Slugify("  Head Barista (Jakarta)! ") != "head-barista-jakarta" || Slugify("  Kitchen & Bar (Shift Malam) ") != "kitchen-bar-shift-malam" {
		t.Fatal("slugify")
	}
	now := time.Date(2026, 10, 4, 3, 0, 0, 0, time.UTC)
	p, err := NormalizeJobOpening(JobOpeningInput{Title: " Barista Senior ", Status: "published", Headcount: 3}, nil, now)
	if err != nil || p.Slug != "barista-senior" || p.Location != "Jakarta, ID" || p.Department != "Operations" || p.Headcount != 3 ||
		p.PublishedAt == nil || !p.PublishedAt.Equal(now) {
		t.Fatalf("%v %+v", err, p)
	}
	first := now.Add(-time.Hour)
	p, _ = NormalizeJobOpening(JobOpeningInput{Title: "Kasir", Status: "published"}, &JobOpening{PublishedAt: &first}, now)
	if !p.PublishedAt.Equal(first) {
		t.Fatal("update keeps the first publication time")
	}
	p, _ = NormalizeJobOpening(JobOpeningInput{Title: "Kasir"}, nil, now)
	if p.Status != "draft" || p.PublishedAt != nil || p.Headcount != 1 {
		t.Fatalf("%+v", p)
	}
}

func TestValidateJobOpening(t *testing.T) {
	now := time.Now()
	if _, err := NormalizeJobOpening(JobOpeningInput{}, nil, now); message(err) != "Judul lowongan wajib diisi" {
		t.Fatal(err)
	}
	if _, err := NormalizeJobOpening(JobOpeningInput{Title: "Kasir", Status: "arsip"}, nil, now); message(err) != "Status lowongan tidak valid" {
		t.Fatal(err)
	}
	if _, err := NormalizeJobOpening(JobOpeningInput{Title: "Kasir", SalaryMin: "9000000", SalaryMax: "5000000"}, nil, now); message(err) !=
		"Gaji minimum tidak boleh melebihi gaji maksimum" {
		t.Fatal(err)
	}
	closing := "2026-10-05"
	o, _ := NormalizeJobOpening(JobOpeningInput{Title: "Kasir", Status: "published", ClosingDate: &closing}, nil, now)
	if !o.IsOpenForApplications(time.Date(2026, 10, 5, 16, 0, 0, 0, time.UTC)) || o.IsOpenForApplications(time.Date(2026, 10, 5, 17, 30, 0, 0, time.UTC)) {
		t.Fatal("the closing date is a WIB day")
	}
}

// ─── candidates and uploads ──────────────────────────────────

func TestNormalizeCandidate(t *testing.T) {
	f, err := NormalizeCandidate(CandidateFields{FullName: " Rani ", Email: "rani@mail.com", Phone: "0812-3456-7890", Domicile: "Depok"}, true)
	if err != nil || f.Source != "walk_in" || f.Status != "applied" || f.FullName != "Rani" {
		t.Fatal(err, f)
	}
	_, err = NormalizeCandidate(CandidateFields{FullName: "Rani", Email: ".rani@mail.com", Phone: "12", Domicile: ""}, true)
	fields := apperr.As(err).FieldErrors
	if fields["email"][0] != "Email tidak valid" || fields["phone"][0] != "Nomor telepon tidak valid" || fields["domicile"][0] != "Domisili wajib diisi" {
		t.Fatalf("%+v", fields)
	}
	patched := CandidatePatch{Notes: ptr("x"), ClearExpectedSalary: true, JobOpeningID: ptr("")}.Apply(CandidateFields{ExpectedSalary: ptr(int64(1)), JobOpeningID: ptr("j")})
	if patched.Notes != "x" || patched.ExpectedSalary != nil || patched.JobOpeningID != nil {
		t.Fatalf("%+v", patched)
	}
	if _, err := NormalizeApplication(Application{FullName: "Rani", Email: "rani@mail.com", Phone: "08123456789"}); message(err) != MsgFieldsRequired {
		t.Fatal(err)
	}
	if _, err := NormalizeApplication(Application{FullName: "Rani", Email: "rani@mail", Phone: "08123456789", Domicile: "Depok"}); message(err) != MsgEmailFormat {
		t.Fatal(err)
	}
	if f, err := NormalizeApplication(Application{FullName: "Rani", Email: "rani@mail.com", Phone: "08123456789", Domicile: "Depok"}); err != nil || f.Source != "portal" {
		t.Fatal(err, f)
	}
}

func TestUploads(t *testing.T) {
	docx := append([]byte("PK\x03\x04"), make([]byte, 20)...)
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"cv.pdf", []byte("%PDF-1.4"), MimePDF},
		{"cv.doc", []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, MimeDOC},
		{"cv.docx", docx, MimeDOCX},
		{"cv.zip", docx, ""},
		{"foto.jpg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, MimeJPEG},
		{"x.webp", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), MimeWEBP},
		{"x.txt", []byte("hello"), ""},
	}
	for _, c := range cases {
		if got := SniffFile(c.data, c.name); got != c.want {
			t.Errorf("%s: %q", c.name, got)
		}
	}
	if _, err := CheckPortalCV("cv.jpg", []byte{0xFF, 0xD8, 0xFF}); message(err) != "CV harus berupa file PDF atau DOC" {
		t.Fatal(err)
	}
	if _, err := CheckCV("cv.pdf", make([]byte, MaxCVBytes+1)); message(err) != "Ukuran file maksimal 10MB" {
		t.Fatal(err)
	}
	if _, err := CheckCV("cv.pdf", []byte("<script>")); message(err) != "Isi file tidak sesuai dengan tipenya" {
		t.Fatal(err)
	}
	if SafeFileName(`..\..\evil".pdf`, "cv.pdf") != "evil.pdf" || SafeFileName("", "cv.pdf") != "cv.pdf" {
		t.Fatal(SafeFileName(`..\..\evil".pdf`, "cv.pdf"))
	}
}

func TestMoney(t *testing.T) {
	m, err := ParseMoney(" 8500000.505 ")
	if err != nil || m.String() != "8500000.51" || m.FormatIDR() != "Rp 8.500.000" {
		t.Fatal(m, err)
	}
	if _, err := ParseMoney("8,5 juta"); err == nil {
		t.Fatal("not a number")
	}
	if Thousands(-1234567) != "-1.234.567" {
		t.Fatal(Thousands(-1234567))
	}
}

// ─── AI interview ────────────────────────────────────────────

func turns(topics ...string) []Turn {
	out := make([]Turn, len(topics))
	for i, tp := range topics {
		out[i] = Turn{TurnNo: i + 1, Topic: ptr(tp), Question: "Q" + tp, Answer: ptr("jawaban " + tp)}
	}
	return out
}

func TestPlanNextQuestion(t *testing.T) {
	q, need := PlanNextQuestion("Rani", "SEO Analyst", 5, nil)
	if need || *q.Topic != "pembuka" ||
		q.Question != "Halo Rani! Terima kasih sudah meluangkan waktu untuk interview posisi SEO Analyst. "+FallbackQuestions["pembuka"] {
		t.Fatal(q.Question)
	}
	if q, _ := PlanNextQuestion("Rani", "", 5, nil); !strings.HasPrefix(q.Question, "Halo Rani! Terima kasih sudah meluangkan waktu untuk interview. ") {
		t.Fatal(q.Question)
	}
	if q, need := PlanNextQuestion("Rani", "", 5, turns("pembuka", "keahlian", "motivasi", "ketersediaan")); need || *q.Topic != "gaji" {
		t.Fatal("gaji takes the last slot")
	}
	if q, need := PlanNextQuestion("Rani", "", 5, turns("pembuka", "keahlian", "motivasi", "gaji", "ketersediaan")); need || !q.Done() {
		t.Fatal("max questions ends the interview")
	}
	if _, need := PlanNextQuestion("Rani", "", 5, turns("pembuka")); !need {
		t.Fatal("the middle turns ask the AI")
	}
	if ClampMaxQuestions(2) != 8 || ClampMaxQuestions(40) != 15 || ClampMaxQuestions(5) != 5 {
		t.Fatal("clamp")
	}
}

func TestResolveNextQuestion(t *testing.T) {
	asked := turns("pembuka")
	if q := ResolveNextQuestion(asked, nil); *q.Topic != "keahlian" || q.Question != FallbackQuestions["keahlian"] {
		t.Fatal("no AI falls back to the first unasked topic")
	}
	if q := ResolveNextQuestion(asked, &AIStep{Action: "finish"}); *q.Topic != "gaji" {
		t.Fatal("finishing before gaji still asks gaji")
	}
	if q := ResolveNextQuestion(turns("pembuka", "gaji"), &AIStep{Action: "finish"}); !q.Done() {
		t.Fatal("finish after gaji ends")
	}
	q := ResolveNextQuestion(asked, &AIStep{Action: "ask", Topic: ptr("lain"), Question: ptr("  " + strings.Repeat("x", 700))})
	if q.Topic != nil || UTF16Len(q.Question) != MaxQuestionChars {
		t.Fatalf("unknown topics are dropped and questions capped: %+v", q)
	}
	if q := ResolveNextQuestion(turns(InterviewTopics...), nil); !q.Done() {
		t.Fatal("every topic asked ends")
	}
}

func TestTranscriptAndPrompts(t *testing.T) {
	ts := []Turn{{TurnNo: 1, Topic: ptr("pembuka"), Question: "Halo", Answer: ptr(" Saya Rani ")}, {TurnNo: 2, Question: "Lagi"}}
	want := "#1 [pembuka]\nAI: Halo\nKandidat: Saya Rani\n\n#2\nAI: Lagi\nKandidat: (tidak menjawab)"
	if got := TranscriptContext(ts); got != want {
		t.Fatalf("%q", got)
	}
	if TranscriptContext(nil) != "(belum ada tanya-jawab)" {
		t.Fatal("empty transcript")
	}
	p := InterviewStepPrompt("Rani", "SEO", 8, ts)
	if !strings.Contains(p, "PERTANYAAN TERPAKAI: 2/8") || !strings.Contains(p, "TOPIK SUDAH DITANYA: pembuka") {
		t.Fatal(p)
	}
	if !strings.HasPrefix(SummaryInput("Rani", "", nil), "KANDIDAT: Rani\nPOSISI: (tidak diisi)") {
		t.Fatal(SummaryInput("Rani", "", nil))
	}
	if AiCompletedDescription(5, true) != "Interview AI selesai (5 pertanyaan) — kesimpulan AI tersedia" {
		t.Fatal(AiCompletedDescription(5, true))
	}
}

func TestNormalizeSummary(t *testing.T) {
	s := NormalizeSummary(map[string]any{
		"ringkasan": "Ok", "relevansi": map[string]any{"skor": 87.5, "kesimpulan": "relevan"},
		"keahlian":        []any{"SEO", 3.0, "a", "b", "c", "d", "e", "f", "g"},
		"ekspektasi_gaji": map[string]any{"disebutkan": true, "nilai": 8000000.0},
		"red_flags":       "bukan list",
	})
	if s.Relevance.Score != 88 || s.Relevance.Conclusion != "relevan" || len(s.Skills) != 8 || s.Skills[1] != "3" ||
		*s.Salary.Value != "8000000" || len(s.RedFlags) != 0 || s.Limitations != DefaultLimitations {
		t.Fatalf("%+v", s)
	}
	if s := NormalizeSummary(map[string]any{"relevansi": map[string]any{"skor": "x", "kesimpulan": "?"}}); s.Relevance.Score != 0 ||
		s.Relevance.Conclusion != "cukup_relevan" || s.Salary.Value != nil {
		t.Fatalf("%+v", s)
	}
}

// ─── interviews, offers, psikotes bank ───────────────────────

func TestInterviewInput(t *testing.T) {
	now := time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC)
	in := InterviewInput{CandidateID: "c", InterviewerName: "Andre", InterviewDate: now.Add(time.Hour), Type: "hrd"}
	if err := in.Normalize(now, true); err != nil || in.DurationMinutes != 60 || in.Mode != "offline" {
		t.Fatal(err, in)
	}
	bad := InterviewInput{CandidateID: "c", InterviewerName: "Andre", InterviewDate: now.Add(time.Hour), Type: "hrd", Mode: "online", MeetingLink: "zoom"}
	if message(bad.Normalize(now, true)) != "Link meeting tidak valid" {
		t.Fatal(bad.Normalize(now, true))
	}
	if !strings.HasPrefix(ScheduledDescription("hrd", now, false), "Interview HRD dijadwalkan 26 Oktober 2026 pukul 10.00") {
		t.Fatal(ScheduledDescription("hrd", now, false))
	}
	items, avg, err := ValidateScorecard([]ScoreItem{{Criterion: " Komunikasi ", Score: 4}, {Criterion: "Teknis", Score: 3}}, "pool", "")
	if err != nil || avg != 3.5 || items[0].Criterion != "Komunikasi" {
		t.Fatal(err, avg, items)
	}
	if _, _, err := ValidateScorecard(nil, "", ""); message(err) != "Isi minimal satu kriteria penilaian" {
		t.Fatal(err)
	}
}

func TestOfferInput(t *testing.T) {
	in := OfferInput{BaseSalary: "8500000", Benefits: []string{" BPJS ", "Laptop"}}
	salary, err := in.Validate()
	if err != nil || salary != Rupiah(8_500_000) || in.ExpiresDays != DefaultOfferExpires || in.Benefits[0] != "BPJS" {
		t.Fatal(err, salary, in)
	}
	bad := OfferInput{BaseSalary: "1", StartDate: ptr("26-10-2026"), ExpiresDays: 31}
	_, err = bad.Validate()
	fields := apperr.As(err).FieldErrors
	if fields["start_date"][0] != "Format tanggal harus YYYY-MM-DD" || fields["expires_days"][0] != "Maksimal 30 hari" {
		t.Fatalf("%+v", fields)
	}
	if status, note, err := ValidateOfferResponse("accept", " "); err != nil || status != "accepted" || note != nil {
		t.Fatal(status, note, err)
	}
	if _, _, err := ValidateOfferResponse("negotiate", ""); message(err) != MsgNegotiateNoteNeed {
		t.Fatal(err)
	}
	if !IsOfferOpen("negotiating") || IsOfferOpen("accepted") || !IsOfferFinal("declined") {
		t.Fatal("offer states")
	}
}

func TestBankValidation(t *testing.T) {
	mcq := QuestionInput{Body: " 2+2? ", CorrectKey: "a", Options: []McqOption{{Key: "a", Text: "4"}, {Key: "a", Text: "5"}}}
	if message(ValidateMcqQuestion(&mcq)) != "Key opsi tidak boleh duplikat" {
		t.Fatal(ValidateMcqQuestion(&mcq))
	}
	one := QuestionInput{Body: "x", CorrectKey: "a", Options: []McqOption{{Key: "a", Text: "4"}}}
	if message(ValidateMcqQuestion(&one)) != "Minimal 2 opsi" {
		t.Fatal(ValidateMcqQuestion(&one))
	}
	papi := QuestionInput{StatementA: PapiStatement{Text: "A", Scale: "N"}, StatementB: PapiStatement{Text: "B", Scale: "Q"}}
	if message(ValidatePapiQuestion(&papi)) != "Skala PAPI tidak valid" {
		t.Fatal(ValidatePapiQuestion(&papi))
	}
	if message(ValidateInvite([]string{"a", "a"}, ptr(0))) != "Instrumen duplikat" {
		t.Fatal("duplicate instruments")
	}
	days := 0
	if err := ValidateInvite([]string{"a"}, &days); err != nil || days != 3 {
		t.Fatal(err, days)
	}
	if message((&InstrumentPatch{}).Validate()) != "Tidak ada perubahan yang dikirim" {
		t.Fatal("empty patch")
	}
	if message((&InstrumentPatch{DurationSeconds: ptr(20000)}).Validate()) != "Durasi maksimal 4 jam" {
		t.Fatal("duration")
	}
	started := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	if PastAnswerGrace(&started, 600, started.Add(630*time.Second)) || !PastAnswerGrace(&started, 600, started.Add(631*time.Second)) {
		t.Fatal("30 s grace after the deadline")
	}
	if DrawLimit(KindMcq, ptr(0)) != 0 || DrawLimit(KindMcq, ptr(10)) != 10 || DrawLimit(KindForcedChoice, ptr(10)) != 0 {
		t.Fatal("draw limit")
	}
}

func TestHiredPayload(t *testing.T) {
	c := Candidate{ID: "c1", FullName: "Gilang", Email: "g@mail.com", Phone: "0812", JobOpeningTitle: "PM", JobOpeningID: ptr("j1")}
	at := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	p := HiredPayload(c, &JobOpening{Title: "Performance Marketing", DepartmentID: ptr("d1")},
		&Offer{ID: "o1", PositionTitle: "Performance Marketing Specialist", BaseSalary: Rupiah(10_000_000), StartDate: ptr("2026-11-02")}, at)
	if p.JobTitle != "Performance Marketing Specialist" || *p.DepartmentID != "d1" || *p.BaseSalary != "10000000.00" || *p.StartDate != "2026-11-02" {
		t.Fatalf("%+v", p)
	}
	if p := HiredPayload(c, nil, nil, at); p.JobTitle != "PM" || p.OfferID != nil {
		t.Fatalf("%+v", p)
	}
}
