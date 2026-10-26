package app_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/pdf"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/app"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── fakes ───────────────────────────────────────────────────

func sp(s string) *string { return &s }

type fakeDirectory struct{}

var interviewer = domain.Employee{ID: "emp_gita", UserID: sp("u_gita"), Name: "Gita Permata", DepartmentID: sp("dept_social"),
	DepartmentName: "Social Media", Active: true}

func (fakeDirectory) Departments(context.Context) ([]domain.Department, error) {
	return []domain.Department{{ID: "dept_social", Code: "SOC", Name: "Social Media", Active: true}}, nil
}

func (fakeDirectory) Employee(_ context.Context, id string) (*domain.Employee, error) {
	if id == interviewer.ID {
		e := interviewer
		return &e, nil
	}
	return nil, nil
}

func (fakeDirectory) Employees(context.Context) ([]domain.Employee, error) {
	return []domain.Employee{interviewer}, nil
}

type fakeNotifier struct {
	mu   sync.Mutex
	sent []app.Notice
}

func (f *fakeNotifier) Notify(_ context.Context, n app.Notice) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, n)
	return nil
}

func (f *fakeNotifier) count(userID, typ string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.sent {
		for _, u := range s.UserIDs {
			if u == userID && (typ == "" || s.Type == typ) {
				n++
			}
		}
	}
	return n
}

type published struct {
	subject string
	payload any
}

type fakeEvents struct {
	mu   sync.Mutex
	fail bool
	sent []published
}

func (f *fakeEvents) Publish(_ context.Context, subject string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return errors.New("nats down")
	}
	f.sent = append(f.sent, published{subject, payload})
	return nil
}

type testClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *testClock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// fakeAudience answers identity's permission holders.
type fakeAudience map[string][]string

func (f fakeAudience) PermissionHolders(_ context.Context, perm string) ([]string, error) {
	return f[perm], nil
}

// ─── fixture ─────────────────────────────────────────────────

type fixture struct {
	store    *postgres.Store
	clk      *testClock
	notifier *fakeNotifier
	events   *fakeEvents
	jobs     *app.JobOpenings
	cands    *app.Candidates
	psi      *app.Psikotes
	ivs      *app.Interviews
	offers   *app.Offers
	portal   *app.Portal
	auto     *app.Automations
}

// referenceData re-applies the Up part of 00002 after the truncate.
func referenceData(t *testing.T) string {
	t.Helper()
	raw, err := fs.ReadFile(recruitment.Migrations(), "00002_reference_data.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _ := strings.Cut(string(raw), "-- +goose Down")
	return up
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := pg.TestPool(t, recruitment.Migrations(), "migrations", recruitment.Schema)
	pg.TruncateSchema(t, pool, recruitment.Schema)
	if _, err := pool.Exec(context.Background(), referenceData(t)); err != nil {
		t.Fatal(err)
	}
	f := &fixture{store: postgres.NewStore(pool), clk: &testClock{t: time.Now().UTC()}, notifier: &fakeNotifier{}, events: &fakeEvents{}}
	deps := app.Deps{Store: f.store, Directory: fakeDirectory{}, Notifier: f.notifier, AI: openrouter.NewSource(nil, openrouter.Options{}),
		Files: storage.NewLocal(t.TempDir()), PDF: pdf.Renderer{}, Events: f.events, Clock: f.clk, Recruiters: []string{"u_hr"},
		Audience: fakeAudience{authn.PermRecruitWrite: {"u_recruiter", "u_hr"}}}
	f.jobs, f.cands, f.psi = app.NewJobOpenings(deps), app.NewCandidates(deps), app.NewPsikotes(deps)
	f.ivs, f.offers, f.portal, f.auto = app.NewInterviews(deps), app.NewOffers(deps), app.NewPortal(deps), app.NewAutomations(deps)
	return f
}

func as(perms ...string) context.Context {
	return authn.WithPrincipal(context.Background(), authn.Principal{UserID: "u_hr", Name: "Nadia (HR)", Role: authn.RoleMember, Perms: perms})
}

var (
	hrCtx     = as(authn.PermRecruitWrite)
	readerCtx = as(authn.PermRecruitRead)
	anon      = context.Background()
)

func wantCode(t *testing.T, err error, code apperr.Code, msg string) {
	t.Helper()
	e := apperr.As(err)
	if err == nil || e.Code != code || (msg != "" && e.Message != msg) {
		t.Fatalf("want %s %q, got %v", code, msg, err)
	}
}

func (f *fixture) opening(t *testing.T, title string) *domain.JobOpening {
	t.Helper()
	o, err := f.jobs.Create(hrCtx, domain.JobOpeningInput{Title: title, DepartmentID: sp("dept_social"), Status: "published",
		HiringManagerUserID: sp("u_lead"), SalaryMin: "6000000", SalaryMax: "8000000",
		Requirements: "Menguasai Instagram, TikTok dan Canva"})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func (f *fixture) candidate(t *testing.T, name string, openingID *string) *domain.Candidate {
	t.Helper()
	salary := int64(7_000_000)
	c, err := f.cands.Create(hrCtx, domain.CandidateFields{FullName: name, Email: strings.ToLower(strings.ReplaceAll(name, " ", ".")) + "@mail.com",
		Phone: "0812-3456-7890", Domicile: "Jakarta", JobOpeningID: openingID, LastExperience: "Social media specialist, Instagram dan Canva",
		ExpectedSalary: &salary})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

var pdfBytes = []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n")

// ─── job openings ────────────────────────────────────────────

func TestJobOpenings(t *testing.T) {
	f := setup(t)
	if _, err := f.jobs.List(anon, "", ""); !apperr.Is(err, apperr.Unauthenticated) {
		t.Fatal("login required", err)
	}
	if _, err := f.jobs.Create(readerCtx, domain.JobOpeningInput{Title: "X"}); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatal("readers cannot write", err)
	}
	_, err := f.jobs.Create(hrCtx, domain.JobOpeningInput{})
	wantCode(t, err, apperr.Validation, "Judul lowongan wajib diisi")
	o := f.opening(t, "Social Media Specialist")
	if o.Slug != "social-media-specialist" || o.Department != "Social Media" || o.Location != "Jakarta, ID" || o.PublishedAt == nil ||
		o.SalaryMin.String() != "6000000.00" {
		t.Fatalf("%+v", o)
	}
	_, err = f.jobs.Create(hrCtx, domain.JobOpeningInput{Title: "Social Media Specialist"})
	wantCode(t, err, apperr.Validation, "Slug lowongan sudah dipakai")

	first := *o.PublishedAt
	f.clk.add(time.Hour)
	u, err := f.jobs.Update(hrCtx, o.ID, domain.JobOpeningInput{Title: "Social Media Specialist", Status: "closed", Headcount: 3,
		DepartmentID: sp("dept_social")})
	if err != nil || u.Status != "closed" || !u.PublishedAt.Equal(first) || u.Headcount != 3 || u.Department != "Social Media" {
		t.Fatalf("%v %+v", err, u)
	}
	list, err := f.jobs.List(readerCtx, "closed", "social")
	if err != nil || len(list) != 1 {
		t.Fatal(err, list)
	}
	if err := f.jobs.Delete(hrCtx, o.ID); err != nil {
		t.Fatal(err)
	}
	_, err = f.jobs.Get(readerCtx, o.ID)
	wantCode(t, err, apperr.NotFound, app.MsgJobNotFound)
}

// ─── candidates ──────────────────────────────────────────────

func TestCandidatesPipeline(t *testing.T) {
	f := setup(t)
	o := f.opening(t, "Social Media Specialist")
	_, err := f.cands.Create(hrCtx, domain.CandidateFields{FullName: "A", Email: "x", Phone: "1", Domicile: "Jakarta"})
	wantCode(t, err, apperr.Validation, "Nama minimal 2 karakter")
	if e := apperr.As(err); len(e.FieldErrors["email"]) != 1 || e.FieldErrors["phone"][0] != "Nomor telepon tidak valid" {
		t.Fatalf("%+v", e.FieldErrors)
	}
	_, err = f.cands.Create(hrCtx, domain.CandidateFields{FullName: "Budi", Email: "b@mail.com", Phone: "081234567890", Domicile: "Depok", Status: "hired"})
	wantCode(t, err, apperr.Validation, "Status awal kandidat harus applied atau screening")

	c := f.candidate(t, "Rani Putri", &o.ID)
	if c.Status != "applied" || c.Source != "walk_in" || c.JobOpeningTitle != "Social Media Specialist" {
		t.Fatalf("%+v", c)
	}
	status, prev, message, err := f.cands.MoveStage(hrCtx, c.ID, "interview")
	if err != nil || status != "interview" || *prev != "applied" || message != "Kandidat dipindahkan ke Interview" {
		t.Fatal(status, prev, message, err)
	}
	if _, _, message, _ := f.cands.MoveStage(hrCtx, c.ID, "interview"); message != "Status tidak berubah" {
		t.Fatal(message)
	}
	_, _, _, err = f.cands.MoveStage(hrCtx, c.ID, "promoted")
	wantCode(t, err, apperr.Validation, "Status tidak valid")
	acts, _ := f.cands.Activities(readerCtx, c.ID)
	if len(acts) != 1 || acts[0].Description != "Tahap diubah: Applied → Interview" || acts[0].CreatedByName != "Nadia (HR)" {
		t.Fatalf("%+v", acts)
	}

	if _, err := f.cands.AddNote(hrCtx, c.ID, "  "); err == nil {
		t.Fatal("empty note")
	}
	if _, err := f.cands.AddNote(hrCtx, c.ID, "Portofolio kuat"); err != nil {
		t.Fatal(err)
	}
	rec := "lolos"
	scr, err := f.cands.SaveScreening(hrCtx, domain.Screening{CandidateID: c.ID, Contacted: true, Recommendation: rec})
	if err != nil || scr.UpdatedByName != "Nadia (HR)" {
		t.Fatal(err, scr)
	}
	acts, _ = f.cands.Activities(readerCtx, c.ID)
	if acts[0].Description != "Hasil screening disimpan (rekomendasi: Lolos)" || acts[1].Description != "Catatan internal ditambahkan" {
		t.Fatalf("%+v", acts)
	}

	other := f.candidate(t, "Bayu Saputra", nil)
	page, err := f.cands.List(readerCtx, app.ListQuery{Status: "applied"})
	if err != nil || page.Total != 1 || page.Candidates[0].ID != other.ID {
		t.Fatal(err, page)
	}
	page, _ = f.cands.List(readerCtx, app.ListQuery{Search: "rani"})
	if page.Total != 1 {
		t.Fatal(page)
	}
	if _, err := f.cands.Update(hrCtx, other.ID, domain.CandidatePatch{Source: sp("referral")}); err != nil {
		t.Fatal(err)
	}
	page, _ = f.cands.List(readerCtx, app.ListQuery{Source: "referral"})
	if page.Total != 1 || page.Candidates[0].ID != other.ID {
		t.Fatal(page)
	}
	if page, _ = f.cands.List(readerCtx, app.ListQuery{Source: "jobstreet"}); page.Total != 0 {
		t.Fatal(page)
	}
	cols, err := f.cands.Pipeline(readerCtx, o.ID, "")
	if err != nil || len(cols) != 8 || cols[3].Status != "interview" || cols[3].Count != 1 || cols[0].Count != 0 {
		t.Fatalf("%v %+v", err, cols)
	}
	upd, err := f.cands.Update(hrCtx, c.ID, domain.CandidatePatch{Domicile: sp("Bekasi"), ClearExpectedSalary: true})
	if err != nil || upd.Domicile != "Bekasi" || upd.ExpectedSalary != nil || upd.FullName != "Rani Putri" {
		t.Fatal(err, upd)
	}

	// CV: magic bytes decide, the old file is replaced
	_, err = f.cands.UploadCV(hrCtx, c.ID, "cv.pdf", []byte("<html>"))
	wantCode(t, err, apperr.Validation, "Isi file tidak sesuai dengan tipenya")
	_, err = f.cands.UploadCV(hrCtx, c.ID, "cv.exe", pdfBytes)
	wantCode(t, err, apperr.Validation, "Tipe file tidak valid. Allowed: pdf, doc, docx, jpg, jpeg, png")
	withCV, err := f.cands.UploadCV(hrCtx, c.ID, "CV Rani.pdf", pdfBytes)
	if err != nil || withCV.CVFileName != "CV Rani.pdf" {
		t.Fatal(err, withCV)
	}
	file, err := f.cands.DownloadCV(readerCtx, c.ID)
	if err != nil || file.ContentType != domain.MimePDF || string(file.Data) != string(pdfBytes) {
		t.Fatal(err, file)
	}

	analysis, err := f.cands.Analyze(hrCtx, c.ID)
	if err != nil || analysis.Model != openrouter.MockModel || analysis.JobContext == "" || analysis.MatchScore < 50 {
		t.Fatal(err, analysis)
	}

	_, err = f.cands.PipelineReport(readerCtx, c.ID)
	wantCode(t, err, apperr.Conflict, "Laporan pipeline tersedia mulai tahap Offer")
	sum, err := f.cands.Summary(readerCtx)
	if err != nil || sum.OpenJobs != 1 || sum.ActiveCandidates != 2 || len(sum.ByStatus) != 9 {
		t.Fatal(err, sum)
	}
	if err := f.cands.Delete(hrCtx, c.ID); err != nil {
		t.Fatal(err)
	}
	_, err = f.cands.Get(readerCtx, c.ID)
	wantCode(t, err, apperr.NotFound, app.MsgCandidateNotFound)
}

// ─── psikotes ────────────────────────────────────────────────

func TestPsikotesFlow(t *testing.T) {
	f := setup(t)
	c := f.candidate(t, "Dinda Pratiwi", nil)
	// bank: 3 MCQ with keys, 2 PAPI pairs
	for i, correct := range []string{"a", "b", "c"} {
		_, err := f.psi.CreateQuestion(hrCtx, "psi_aritmatika", domain.QuestionInput{Body: "Soal " + string(rune('1'+i)), CorrectKey: correct,
			IsActive: true, Options: []domain.McqOption{{Key: "a", Text: "1"}, {Key: "b", Text: "2"}, {Key: "c", Text: "3"}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := f.psi.CreateQuestion(hrCtx, "psi_aritmatika", domain.QuestionInput{Body: "Q", CorrectKey: "z", IsActive: true,
		Options: []domain.McqOption{{Key: "a", Text: "1"}, {Key: "b", Text: "2"}}})
	wantCode(t, err, apperr.Validation, "Kunci jawaban harus salah satu key opsi")
	_, err = f.psi.CreateQuestion(hrCtx, "psi_baum", domain.QuestionInput{Body: "x"})
	wantCode(t, err, apperr.Validation, "Instrumen tes gambar tidak memiliki bank soal — atur instruksi di config")
	_, err = f.psi.CreateQuestion(hrCtx, "psi_papi_kostick", domain.QuestionInput{IsActive: true,
		StatementA: domain.PapiStatement{Text: "A", Scale: "N"}, StatementB: domain.PapiStatement{Text: "B", Scale: "N"}})
	wantCode(t, err, apperr.Validation, "Skala pernyataan A dan B tidak boleh sama")
	for _, pair := range [][2]string{{"A", "G"}, {"A", "L"}} {
		if _, err := f.psi.CreateQuestion(hrCtx, "psi_papi_kostick", domain.QuestionInput{IsActive: true,
			StatementA: domain.PapiStatement{Text: "Saya " + pair[0], Scale: pair[0]}, StatementB: domain.PapiStatement{Text: "Saya " + pair[1], Scale: pair[1]}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.psi.UpdateInstrument(hrCtx, "psi_aritmatika", domain.InstrumentPatch{}); err == nil {
		t.Fatal("empty patch")
	}

	_, err = f.psi.Invite(hrCtx, c.ID, nil, 0)
	wantCode(t, err, apperr.Validation, "Pilih minimal satu instrumen")
	session, err := f.psi.Invite(hrCtx, c.ID, []string{"psi_aritmatika", "psi_papi_kostick", "psi_baum"}, 0)
	if err != nil || len(session.Tests) != 3 || session.Status != "sent" || !domain.IsPortalToken(session.Token) {
		t.Fatal(err, session)
	}
	_, sessions, _ := f.psi.Panel(readerCtx, c.ID)
	if sessions[0].Token != "" {
		t.Fatal("readers do not see tokens")
	}
	tok := session.Token

	// portal
	_, err = f.portal.PsikotesPortal(anon, "nope")
	wantCode(t, err, apperr.NotFound, "Link tes tidak berlaku")
	byCode := map[string]string{}
	for _, t := range session.Tests {
		byCode[t.InstrumentCode] = t.ID
	}
	ar, papi, baum := byCode["aritmatika"], byCode["papi_kostick"], byCode["baum"]
	_, err = f.portal.StartTest(anon, tok, ar)
	wantCode(t, err, apperr.Conflict, "Sesi belum dimulai atau sudah berakhir")
	if _, err := f.portal.StartPsikotes(anon, tok, true); err != nil {
		t.Fatal(err)
	}
	run, err := f.portal.StartTest(anon, tok, ar)
	if err != nil || len(run.Questions) != 3 {
		t.Fatal(err, run)
	}
	for _, q := range run.Questions {
		if q.CorrectKey != "" {
			t.Fatal("answer keys never leave the server")
		}
	}
	keys := map[string]string{}
	bank, _ := f.psi.Questions(readerCtx, "psi_aritmatika")
	for _, q := range bank {
		keys[q.ID] = q.CorrectKey
	}
	answers := map[string]string{}
	for i, q := range run.Questions {
		answers[q.ID] = keys[q.ID]
		if i == 2 {
			answers[q.ID] = "a" // one wrong (keys are a, b, c; "a" is only right for the first)
			if keys[q.ID] == "a" {
				answers[q.ID] = "b"
			}
		}
	}
	answers["not-drawn"] = "a"
	n, err := f.portal.SaveAnswers(anon, tok, ar, answers)
	if err != nil || n != 3 {
		t.Fatal(err, n)
	}
	status, message, err := f.portal.FinishTest(anon, tok, ar, nil)
	if err != nil || status != "selesai" || message != "Tes selesai" {
		t.Fatal(status, message, err)
	}
	if status, message, _ := f.portal.FinishTest(anon, tok, ar, nil); status != "selesai" || message != "Tes sudah selesai" {
		t.Fatal(status, message)
	}
	prun, err := f.portal.StartTest(anon, tok, papi)
	if err != nil || len(prun.Questions) != 2 || prun.Questions[0].Pair.A.Scale != "" {
		t.Fatal(err, prun)
	}
	papiAnswers := map[string]string{prun.Questions[0].ID: "a", prun.Questions[1].ID: "a"}
	if _, _, err := f.portal.FinishTest(anon, tok, papi, papiAnswers); err != nil {
		t.Fatal(err)
	}
	if _, err := f.portal.StartTest(anon, tok, baum); err != nil {
		t.Fatal(err)
	}
	_, _, err = f.portal.FinishTest(anon, tok, baum, nil)
	wantCode(t, err, apperr.Validation, "Unggah hasil gambar terlebih dahulu")
	err = f.portal.UploadDrawing(anon, tok, baum, "pohon.png", []byte("not an image"))
	wantCode(t, err, apperr.Validation, "Isi file bukan gambar JPG/PNG/WebP yang valid")
	png := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0}
	if err := f.portal.UploadDrawing(anon, tok, baum, "pohon.png", png); err != nil {
		t.Fatal(err)
	}
	if status, _, err := f.portal.FinishTest(anon, tok, baum, nil); err != nil || status != "perlu_review" {
		t.Fatal(status, err)
	}
	if _, err := f.portal.RecordProctorEvent(anon, "psikotes", tok, "tab_blur", map[string]string{"at": "1"}); err != nil {
		t.Fatal(err)
	}
	_, err = f.portal.RecordProctorEvent(anon, "psikotes", tok, "webcam_snapshot", nil)
	wantCode(t, err, apperr.Validation, "Jenis event tidak valid")
	status, message, err = f.portal.FinishPsikotes(anon, tok)
	if err != nil || status != "completed" || message != "Seluruh tes selesai — terima kasih" {
		t.Fatal(status, message, err)
	}
	if f.notifier.count("u_hr", app.NotifPsikotesDone) != 1 {
		t.Fatal("recruiters hear about finished psikotes")
	}

	// HR side
	_, sessions, _ = f.psi.Panel(hrCtx, c.ID)
	tests := map[string]domain.SessionTest{}
	for _, t := range sessions[0].Tests {
		tests[t.InstrumentCode] = t
	}
	if a := tests["aritmatika"]; *a.Score != 67 || a.ScoreDetail.Correct != 2 || sessions[0].Proctor.Flags != 1 {
		t.Fatalf("%+v %+v", a, sessions[0].Proctor)
	}
	if d := tests["papi_kostick"].ScoreDetail; d.Scales["A"] != 2 || len(d.Dominant) != 1 || d.Dominant[0].Label != "Dorongan Berprestasi" {
		t.Fatalf("%+v", d)
	}
	kind, items, err := f.psi.Answers(readerCtx, ar)
	if err != nil || kind != "mcq" || len(items) != 3 || items[2].Question.ID != run.Questions[2].ID || *items[2].IsCorrect {
		t.Fatal(kind, items, err)
	}
	if _, err := f.psi.Attachment(readerCtx, baum); err != nil {
		t.Fatal(err)
	}
	_, err = f.psi.Review(hrCtx, ar, "ok")
	wantCode(t, err, apperr.Conflict, "Tes ini tidak dalam antrian review manual")
	reviewed, err := f.psi.Review(hrCtx, baum, "Batang kokoh, akar terlihat")
	if err != nil || reviewed.Status != "reviewed" || reviewed.ReviewedByName != "Nadia (HR)" {
		t.Fatal(err, reviewed)
	}
	if _, err := f.psi.SaveSummary(hrCtx, domain.PsikotesSummary{CandidateID: c.ID, Recommendation: "lolos"}); err != nil {
		t.Fatal(err)
	}
	acts, _ := f.cands.Activities(readerCtx, c.ID)
	if acts[0].Description != "Rekomendasi psikotes disimpan (rekomendasi: Lolos)" ||
		!strings.HasPrefix(acts[len(acts)-1].Description, "Undangan psikotes online dibuat (PAPI Kostick, Baum (Tree Test), Aritmatika; berlaku 3 hari)") {
		t.Fatalf("%+v", acts)
	}
	events, err := f.psi.ProctorEvents(readerCtx, "psikotes", session.ID)
	if err != nil || len(events) != 1 || events[0].Meta["at"] != "1" {
		t.Fatal(err, events)
	}
}

func TestPortalLinkExpiryAndRateLimit(t *testing.T) {
	f := setup(t)
	c := f.candidate(t, "Eko Wibowo", nil)
	s, err := f.psi.Invite(hrCtx, c.ID, []string{"psi_baum"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	f.clk.add(25 * time.Hour)
	p, err := f.portal.PsikotesPortal(anon, s.Token)
	if err != nil || p.Status != "expired" {
		t.Fatal(err, p)
	}
	_, err = f.portal.StartPsikotes(anon, s.Token, true)
	wantCode(t, err, apperr.Conflict, "Link tes sudah kedaluwarsa")
	for i := 0; i < domain.DefaultRateLimit-1; i++ {
		if _, err := f.portal.PsikotesPortal(anon, s.Token); err != nil {
			t.Fatal(i, err)
		}
	}
	_, err = f.portal.PsikotesPortal(anon, s.Token)
	wantCode(t, err, apperr.RateLimit, domain.TooManyRequests)
}

// ─── AI interview ────────────────────────────────────────────

func TestAiInterview(t *testing.T) {
	f := setup(t)
	o := f.opening(t, "SEO Analyst")
	c := f.candidate(t, "Nanda Prasetyo", &o.ID)
	_, err := f.ivs.InviteAi(hrCtx, c.ID, 0, 20)
	wantCode(t, err, apperr.Validation, "Jumlah pertanyaan 3 sampai 15")
	s, err := f.ivs.InviteAi(hrCtx, c.ID, 0, 3)
	if err != nil || s.MaxQuestions != 3 {
		t.Fatal(err, s)
	}
	_, err = f.portal.StartInterview(anon, s.Token, false)
	wantCode(t, err, apperr.Validation, "Interview mewajibkan kamera aktif — izinkan kamera untuk memulai")
	turn, err := f.portal.StartInterview(anon, s.Token, true)
	if err != nil || turn.TurnNo != 1 || !strings.HasPrefix(turn.Question, "Halo Nanda Prasetyo! Terima kasih sudah meluangkan waktu untuk interview posisi SEO Analyst.") {
		t.Fatal(err, turn)
	}
	again, _ := f.portal.StartInterview(anon, s.Token, true)
	if again.ID != turn.ID {
		t.Fatal("a running session returns its active question")
	}
	_, _, err = f.portal.AnswerInterview(anon, s.Token, turn.ID, "  ")
	wantCode(t, err, apperr.Validation, "Jawaban kosong")
	answers := []string{"Saya tiga tahun di agency mengelola SEO dengan Ahrefs dan Search Console.", "Riset kata kunci dan audit teknis.", "Rp 8 juta"}
	topics := []string{"pembuka", "keahlian", "gaji"}
	for i, a := range answers {
		if *turn.Topic != topics[i] {
			t.Fatalf("turn %d topic %s", i+1, *turn.Topic)
		}
		next, done, err := f.portal.AnswerInterview(anon, s.Token, turn.ID, a)
		if err != nil {
			t.Fatal(err)
		}
		if i == len(answers)-1 {
			if !done {
				t.Fatal("the last slot ends the interview")
			}
			break
		}
		turn = next
	}
	_, _, err = f.portal.AnswerInterview(anon, s.Token, turn.ID, "lagi")
	wantCode(t, err, apperr.Conflict, "Sesi tidak sedang berjalan")
	portal, _, err := f.portal.InterviewPortal(anon, s.Token)
	if err != nil || portal.Status != "completed" || portal.Summary != nil || len(portal.Turns) != 3 {
		t.Fatal(err, portal)
	}
	panel, err := f.ivs.AiPanel(hrCtx, c.ID)
	if err != nil || panel[0].Summary == nil || panel[0].SummaryModel != openrouter.MockModel || *panel[0].Summary.Salary.Value != "Rp 8 juta" {
		t.Fatal(err, panel)
	}
	acts, _ := f.cands.Activities(readerCtx, c.ID)
	if acts[0].Description != "Interview AI selesai (3 pertanyaan) — kesimpulan AI tersedia" {
		t.Fatal(acts[0].Description)
	}
	if f.notifier.count("u_lead", app.NotifAiInterviewDone) != 1 {
		t.Fatal("the hiring manager hears about the interview")
	}
	ref, _, err := f.offers.Panel(readerCtx, c.ID)
	if err != nil || ref.InterviewExpectation != "Rp 8 juta" || ref.SalaryMin.String() != "6000000.00" {
		t.Fatal(err, ref)
	}
}

// ─── scheduled interviews ────────────────────────────────────

func TestScheduledInterviews(t *testing.T) {
	f := setup(t)
	c := f.candidate(t, "Yoga Aditya", nil)
	when := f.clk.Now().Add(26 * time.Hour)
	_, err := f.ivs.Schedule(hrCtx, domain.InterviewInput{CandidateID: c.ID, InterviewerEmployeeID: sp(interviewer.ID),
		InterviewDate: f.clk.Now().Add(-48 * time.Hour), Type: "hrd"})
	wantCode(t, err, apperr.Validation, "Jadwal interview tidak boleh di masa lalu")
	_, err = f.ivs.Schedule(hrCtx, domain.InterviewInput{CandidateID: c.ID, InterviewerName: "Andre", InterviewDate: when, Type: "hrd", Mode: "online"})
	wantCode(t, err, apperr.Validation, "Link meeting wajib diisi untuk interview online")
	iv, err := f.ivs.Schedule(hrCtx, domain.InterviewInput{CandidateID: c.ID, InterviewerEmployeeID: sp(interviewer.ID), InterviewDate: when,
		Type: "hiring_manager", Mode: "online", MeetingLink: "https://meet.google.com/abc"})
	if err != nil || iv.InterviewerName != "Gita Permata" || *iv.InterviewerUserID != "u_gita" || iv.DurationMinutes != 60 {
		t.Fatal(err, iv)
	}
	if f.notifier.count("u_gita", app.NotifInterviewSchedule) != 1 {
		t.Fatal("the interviewer is notified")
	}

	// reminder for tomorrow, once per day
	// The interview's calendar day in WIB: taking it from `when` in UTC
	// picks the wrong day between 00:00 and 07:00 WIB.
	wib := when.In(domain.Jakarta)
	day := time.Date(wib.Year(), wib.Month(), wib.Day(), 0, 0, 0, 0, domain.Jakarta)
	runAt := day.Add(-24 * time.Hour).Add(7 * time.Hour)
	for i := 0; i < 2; i++ {
		if err := f.auto.TickDaily(context.Background(), runAt); err != nil {
			t.Fatal(err)
		}
	}
	if n := f.notifier.count("u_gita", app.NotifInterviewReminder); n != 1 {
		t.Fatal("one reminder per day", n)
	}

	moved, err := f.ivs.Update(hrCtx, iv.ID, domain.InterviewInput{InterviewerEmployeeID: sp(interviewer.ID), InterviewDate: when.Add(time.Hour),
		Type: "hiring_manager", Mode: "offline", MeetingLink: "https://meet.google.com/abc", Location: "Kemang"})
	if err != nil || moved.MeetingLink != "" || moved.Location != "Kemang" {
		t.Fatal(err, moved)
	}
	_, err = f.ivs.SubmitScorecard(hrCtx, iv.ID, []domain.ScoreItem{{Criterion: "Komunikasi", Score: 6}}, "proceed", "")
	wantCode(t, err, apperr.Validation, "Nilai kriteria harus 1 sampai 5")
	scored, err := f.ivs.SubmitScorecard(hrCtx, iv.ID, []domain.ScoreItem{{Criterion: "Komunikasi", Score: 4}, {Criterion: "Keahlian teknis", Score: 5}},
		"proceed", "Bagus")
	if err != nil || scored.Status != "completed" || *scored.OverallScore != 4.5 || scored.Recommendation != "proceed" {
		t.Fatal(err, scored)
	}
	_, err = f.ivs.Cancel(hrCtx, iv.ID, "")
	wantCode(t, err, apperr.Conflict, "Interview yang sudah selesai atau dibatalkan tidak bisa diubah")
	acts, _ := f.cands.Activities(readerCtx, c.ID)
	if acts[0].Description != "Scorecard interview Hiring Manager disimpan (nilai 4.50; rekomendasi: Lanjut)" ||
		!strings.HasPrefix(acts[1].Description, "Interview Hiring Manager dijadwalkan ulang ke ") {
		t.Fatalf("%+v", acts)
	}
	list, err := f.ivs.List(readerCtx, app.ListFilter{CandidateID: c.ID, Status: "completed"})
	if err != nil || len(list) != 1 {
		t.Fatal(err, list)
	}
}

// ─── offers and hire ─────────────────────────────────────────

func TestOffersAndHire(t *testing.T) {
	f := setup(t)
	o := f.opening(t, "Performance Marketing Specialist")
	c := f.candidate(t, "Gilang Ramadhan", &o.ID)
	_, err := f.offers.Create(hrCtx, c.ID, domain.OfferInput{BaseSalary: "abc"})
	wantCode(t, err, apperr.Validation, "Gaji pokok tidak valid")
	v1, err := f.offers.Create(hrCtx, c.ID, domain.OfferInput{BaseSalary: "9500000", Benefits: []string{"BPJS"}})
	if err != nil || v1.Version != 1 || v1.PositionTitle != "Performance Marketing Specialist" {
		t.Fatal(err, v1)
	}
	start := f.clk.Now().AddDate(0, 0, 14).Format(domain.IsoDate)
	v2, err := f.offers.Create(hrCtx, c.ID, domain.OfferInput{BaseSalary: "10000000", StartDate: &start, ExpiresDays: 3})
	if err != nil || v2.Version != 2 {
		t.Fatal(err, v2)
	}
	_, err = f.portal.RespondOffer(anon, v1.Token, "accept", "", "1.2.3.4")
	wantCode(t, err, apperr.Conflict, domain.MsgOfferClosed)
	_, err = f.portal.RespondOffer(anon, v2.Token, "negotiate", " ", "1.2.3.4")
	wantCode(t, err, apperr.Validation, domain.MsgNegotiateNoteNeed)
	neg, err := f.portal.RespondOffer(anon, v2.Token, "negotiate", "Bisa 11 juta?", "1.2.3.4")
	if err != nil || neg.Status != "negotiating" || neg.Notes != "" {
		t.Fatal(err, neg)
	}
	view, _ := f.portal.OfferPortal(anon, v2.Token)
	if view.Notes != "" || view.BaseSalary.String() != "10000000.00" {
		t.Fatal("HR notes stay out of the portal")
	}

	// reminder for an offer expiring within a day
	f.clk.add(50 * time.Hour)
	if err := f.auto.TickDaily(context.Background(), f.clk.Now()); err != nil {
		t.Fatal(err)
	}
	if f.notifier.count("u_hr", app.NotifOfferExpiring) != 1 {
		t.Fatal("offer expiring reminder")
	}

	accepted, err := f.portal.RespondOffer(anon, v2.Token, "accept", "Siap bergabung", "1.2.3.4")
	if err != nil || accepted.Status != "accepted" {
		t.Fatal(err, accepted)
	}
	hired, _ := f.cands.Get(readerCtx, c.ID)
	if hired.Status != "hired" || hired.HiredAt == nil || hired.HiredEventAt == nil {
		t.Fatalf("%+v", hired)
	}
	if len(f.events.sent) != 1 || f.events.sent[0].subject != events.RecruitmentCandidateHired {
		t.Fatalf("%+v", f.events.sent)
	}
	p := f.events.sent[0].payload.(domain.CandidateHiredPayload)
	if p.FullName != "Gilang Ramadhan" || p.JobTitle != "Performance Marketing Specialist" || *p.DepartmentID != "dept_social" ||
		*p.StartDate != start || *p.BaseSalary != "10000000.00" || *p.OfferID != v2.ID {
		t.Fatalf("%+v", p)
	}
	if f.notifier.count("u_lead", app.NotifHired) != 1 || f.notifier.count("u_hr", app.NotifOfferResponse) != 2 {
		t.Fatal("hire and response notices")
	}
	acts, _ := f.cands.Activities(readerCtx, c.ID)
	if acts[0].Description != "Tahap diubah: Applied → Hired" || acts[0].CreatedByName != "Sistem" ||
		acts[1].Description != `Kandidat menerima offer v2 via portal — "Siap bergabung"` {
		t.Fatalf("%+v", acts[:2])
	}
	_, err = f.offers.RecordResponse(hrCtx, v2.ID, "declined", "")
	wantCode(t, err, apperr.Conflict, domain.MsgOfferFinal)
	letter, err := f.portal.OfferLetter(anon, v2.Token)
	if err != nil || letter.ContentType != domain.MimePDF || !strings.HasPrefix(string(letter.Data), "%PDF-") {
		t.Fatal(err)
	}
	report, err := f.cands.PipelineReport(readerCtx, c.ID)
	if err != nil || report.Name != "laporan-pipeline-gilang-ramadhan.pdf" {
		t.Fatal(err, report)
	}
	// moving back and forth does not publish twice
	_, _, _, _ = f.cands.MoveStage(hrCtx, c.ID, "offer")
	_, _, _, _ = f.cands.MoveStage(hrCtx, c.ID, "hired")
	if len(f.events.sent) != 1 {
		t.Fatal("one hire event per candidate")
	}
}

func TestManualHireRetriesEvent(t *testing.T) {
	f := setup(t)
	c := f.candidate(t, "Putri Ayu", nil)
	o, err := f.offers.Create(hrCtx, c.ID, domain.OfferInput{BaseSalary: "7000000"})
	if err != nil {
		t.Fatal(err)
	}
	f.events.fail = true
	if _, err := f.offers.RecordResponse(hrCtx, o.ID, "accepted", "Via telepon"); err != nil {
		t.Fatal(err)
	}
	cand, _ := f.cands.Get(readerCtx, c.ID)
	if cand.Status != "hired" || cand.HiredEventAt != nil {
		t.Fatalf("%+v", cand)
	}
	f.events.fail = false
	if err := f.auto.TickDaily(context.Background(), f.clk.Now()); err != nil {
		t.Fatal(err)
	}
	if len(f.events.sent) != 1 {
		t.Fatal("the daily tick retries the hire event")
	}
	acts, _ := f.cands.Activities(readerCtx, c.ID)
	if acts[1].Description != `Kandidat menerima offer v1 (dicatat manual) — "Via telepon"` || acts[0].CreatedByName != "Nadia (HR)" {
		t.Fatalf("%+v", acts[:2])
	}
}

// ─── career page ─────────────────────────────────────────────

func TestApply(t *testing.T) {
	f := setup(t)
	o := f.opening(t, "Graphic Designer")
	jobs, err := f.portal.PublicJobs(anon, "", "")
	if err != nil || len(jobs) != 1 {
		t.Fatal(err, jobs)
	}
	app1 := domain.Application{FullName: "Taufik Hidayat", Email: "taufik@mail.com", Phone: "081298765432", Domicile: "Bekasi"}
	_, err = f.portal.Apply(anon, o.Slug, domain.Application{FullName: "X"}, "", nil, "10.0.0.1")
	wantCode(t, err, apperr.Validation, domain.MsgFieldsRequired)
	bad := app1
	bad.Email = "taufik@mail"
	_, err = f.portal.Apply(anon, o.Slug, bad, "", nil, "10.0.0.1")
	wantCode(t, err, apperr.Validation, domain.MsgEmailFormat)
	_, err = f.portal.Apply(anon, o.Slug, app1, "cv.pdf", []byte("MZ\x90\x00"), "10.0.0.1")
	wantCode(t, err, apperr.Validation, "Isi file tidak sesuai dengan tipenya")
	_, err = f.portal.Apply(anon, o.Slug, app1, "cv.png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, "10.0.0.1")
	wantCode(t, err, apperr.Validation, "CV harus berupa file PDF atau DOC")
	id, err := f.portal.Apply(anon, o.Slug, app1, "Portofolio.pdf", pdfBytes, "10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	c, _ := f.cands.Get(readerCtx, id)
	if c.Status != "applied" || c.Source != "portal" || c.CVKey == nil || *c.JobOpeningID != o.ID {
		t.Fatalf("%+v", c)
	}
	if f.notifier.count("u_lead", app.NotifApplication) != 1 || f.notifier.count("u_hr", app.NotifApplication) != 1 || f.notifier.count("u_recruiter", app.NotifApplication) != 1 {
		t.Fatal("recruiters and the hiring manager hear about the application")
	}
	// the sixth attempt from the same network within 10 minutes is refused
	_, err = f.portal.Apply(anon, o.Slug, app1, "", nil, "10.0.0.1")
	wantCode(t, err, apperr.RateLimit, domain.MsgApplyTooMany)
	_, err = f.portal.Apply(anon, o.Slug, app1, "", nil, "10.0.0.2")
	wantCode(t, err, apperr.Conflict, domain.MsgAlreadyApplied)

	closing := f.clk.Now().AddDate(0, 0, -2).Format(domain.IsoDate)
	if _, err := f.jobs.Update(hrCtx, o.ID, domain.JobOpeningInput{Title: o.Title, Status: "published", ClosingDate: &closing}); err != nil {
		t.Fatal(err)
	}
	_, err = f.portal.Apply(anon, o.Slug, app1, "", nil, "10.0.0.3")
	wantCode(t, err, apperr.Conflict, domain.MsgJobClosed)
	if jobs, _ := f.portal.PublicJobs(anon, "", ""); len(jobs) != 0 {
		t.Fatal("closed openings leave the career page")
	}
}
