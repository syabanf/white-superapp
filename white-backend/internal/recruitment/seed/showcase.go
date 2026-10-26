package seed

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"slices"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/jackc/pgx/v5"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

// showcase fills what the other steps leave empty so every recruitment
// page has something to show: a draft opening, the Matematika bank, notes,
// CV files, AI analyses, psikotes sessions in every state (with drawings to
// review and proctoring flags), and demo portal links that stay valid.
func (s *seeder) showcase(ctx context.Context) error {
	steps := []func(context.Context) error{s.draftOpening, s.mathBank, s.notes, s.cvs, s.analyses, s.psikotesStates, s.livePipeline, s.portalLinks}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) draftOpening(ctx context.Context) error {
	dept := "Management"
	return s.exec(ctx, `INSERT INTO recruitment.job_openings (id, title, slug, department, location, employment_type, work_mode, headcount,
		description, requirements, benefits, salary_min, salary_max, status, created_by, updated_by)
		VALUES ('seed_job_draft', 'Data Analyst', 'data-analyst', $1, $2, 'Full-time', 'Hybrid', 1, $3, $4, $5, $6, $7, 'draft', $8, $8)
		ON CONFLICT (id) DO NOTHING`, dept, domain.DefaultLocation,
		"Menyusun dashboard performa klien dari data iklan, media sosial, dan website, lalu merangkum insight bulanan.",
		"Menguasai SQL, Spreadsheet, dan Looker Studio. Terbiasa membaca data GA4 dan Meta Ads.",
		"BPJS, laptop kerja, budget pelatihan.", domain.Rupiah(8_000_000), domain.Rupiah(11_000_000), seedUser)
}

var mathQuestions = []struct {
	body    string
	options [4]string
	correct string
}{
	{"2x + 6 = 20, maka x = ...", [4]string{"5", "6", "7", "8"}, "c"},
	{"Luas persegi panjang 12 m x 8 m adalah ...", [4]string{"86 m²", "92 m²", "96 m²", "104 m²"}, "c"},
	{"Harga naik 20% dari Rp 50.000 menjadi ...", [4]string{"Rp 55.000", "Rp 60.000", "Rp 62.500", "Rp 65.000"}, "b"},
	{"Rata-rata 4, 8, 15, 16, 23, 42 adalah ...", [4]string{"16", "18", "19", "21"}, "b"},
	{"Jika 3 desainer menyelesaikan 12 aset dalam 2 hari, 6 desainer menyelesaikan 12 aset dalam ...", [4]string{"1 hari", "2 hari", "3 hari", "4 hari"}, "a"},
	{"0,25 x 480 = ...", [4]string{"100", "110", "120", "130"}, "c"},
	{"Perbandingan 3 : 5 dengan total 64, bagian terbesar adalah ...", [4]string{"24", "36", "40", "44"}, "c"},
	{"√144 + 2³ = ...", [4]string{"18", "20", "22", "24"}, "b"},
	{"Diskon 15% dari Rp 2.000.000 adalah ...", [4]string{"Rp 250.000", "Rp 300.000", "Rp 350.000", "Rp 150.000"}, "b"},
	{"Deret 2, 5, 11, 23, ... angka berikutnya adalah", [4]string{"35", "41", "46", "47"}, "d"},
}

func (s *seeder) mathBank(ctx context.Context) error {
	keys := []string{"a", "b", "c", "d"}
	for i, q := range mathQuestions {
		opts := make([]domain.McqOption, 4)
		for k := range opts {
			opts[k] = domain.McqOption{Key: keys[k], Text: q.options[k]}
		}
		if err := s.store.InsertQuestion(ctx, domain.Question{ID: fmt.Sprintf("seed_q_mt_%02d", i+1), InstrumentID: "psi_matematika",
			Body: q.body, Options: opts, CorrectKey: q.correct, SortOrder: i + 1, IsActive: true}, seedUser); err != nil {
			return err
		}
	}
	return nil
}

var notePool = []string{
	"Portofolio rapi dan relevan dengan klien F&B kami.",
	"Komunikasi lancar saat dihubungi, responsif di WhatsApp.",
	"Ekspektasi gaji sedikit di atas rentang, perlu dibahas saat offer.",
	"Pernah menangani akun brand nasional, cek referensi kerja.",
	"Bisa mulai lebih cepat bila diperlukan.",
	"Perlu dicek kemampuan membaca data dan membuat laporan.",
}

// notes gives every seeded candidate internal notes.
func (s *seeder) notes(ctx context.Context) error {
	for i, p := range people {
		id := candidateID(i)
		created := s.now.AddDate(0, 0, -p.daysAgo)
		count := 1
		if p.status != "applied" {
			count = 2
		}
		for k := range count {
			at := created.Add(time.Duration(k*30+6) * time.Hour)
			text := notePool[pick(fmt.Sprintf("%s:n%d", id, k), len(notePool))]
			if err := s.exec(ctx, `INSERT INTO recruitment.candidate_notes (id, candidate_id, content, created_by, created_by_name, created_at)
				VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`, fmt.Sprintf("%s_note_%d", id, k+1), id, text, seedUser, hrName, at); err != nil {
				return err
			}
			if err := s.exec(ctx, `INSERT INTO recruitment.candidate_activities (id, candidate_id, activity_type, description, created_by,
				created_by_name, created_at) VALUES ($1, $2, 'note_added', 'Catatan internal ditambahkan', $3, $4, $5) ON CONFLICT (id) DO NOTHING`,
				fmt.Sprintf("%s_note_act_%d", id, k+1), id, seedUser, hrName, at); err != nil {
				return err
			}
		}
	}
	// one note a day after applying for every past applicant
	return s.exec(ctx, `INSERT INTO recruitment.candidate_notes (id, candidate_id, content, created_by, created_by_name, created_at)
		SELECT c.id || '_note_1', c.id, ($1::text[])[1 + mod(abs(hashtext(c.id)), cardinality($1::text[]))], $2, $3, c.created_at + interval '1 day'
		FROM recruitment.candidates c WHERE c.id LIKE 'seed\_hist\_%' ON CONFLICT (id) DO NOTHING`, notePool, seedUser, hrName)
}

// showcaseIDs are the seeded candidates, whose detail pages get CVs and analyses.
func (s *seeder) showcaseIDs(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM recruitment.candidates
		WHERE id LIKE 'seed\_cand\_%' OR id LIKE 'seed\_hist\_%' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// cvs stores a one-page PDF CV for candidates without one.
func (s *seeder) cvs(ctx context.Context) error {
	if s.files == nil {
		return nil
	}
	ids, err := s.showcaseIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		c, err := s.store.GetCandidate(ctx, id)
		if err != nil {
			return err
		}
		if c == nil || c.CVKey != nil {
			continue
		}
		data, err := cvPDF(*c)
		if err != nil {
			return err
		}
		key := "cv/" + id + "/seed.pdf"
		if err := s.files.Put(ctx, key, data); err != nil {
			return err
		}
		name := "cv-" + domain.Slugify(c.FullName) + ".pdf"
		if err := s.exec(ctx, `UPDATE recruitment.candidates SET cv_key = $2, cv_file_name = $3, cv_content_type = $4 WHERE id = $1 AND cv_key IS NULL`,
			id, key, name, domain.MimePDF); err != nil {
			return err
		}
	}
	return nil
}

func cvPDF(c domain.Candidate) ([]byte, error) {
	f := fpdf.New("P", "mm", "A4", "")
	tr := f.UnicodeTranslatorFromDescriptor("")
	f.SetMargins(20, 20, 20)
	f.AddPage()
	f.SetFont("Helvetica", "B", 20)
	f.Cell(0, 10, tr(c.FullName))
	f.Ln(9)
	f.SetFont("Helvetica", "", 10)
	f.Cell(0, 6, tr(c.Email+"  |  "+c.Phone+"  |  "+c.Domicile))
	f.Ln(12)
	section := func(title, body string) {
		f.SetFont("Helvetica", "B", 12)
		f.Cell(0, 7, tr(title))
		f.Ln(8)
		f.SetFont("Helvetica", "", 10)
		f.MultiCell(0, 5.5, tr(body), "", "L", false)
		f.Ln(4)
	}
	section("Pengalaman", c.LastExperience)
	section("Pendidikan", c.LastEducation)
	section("Keahlian", "Perencanaan konten, Canva, Spreadsheet, Google Analytics, presentasi ke klien, kerja tim lintas divisi.")
	var buf bytes.Buffer
	err := f.Output(&buf)
	return buf.Bytes(), err
}

// analyses stores the mock AI profile analysis for showcase candidates.
func (s *seeder) analyses(ctx context.Context) error {
	ids, err := s.showcaseIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		existing, err := s.store.GetAnalysis(ctx, id)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		c, err := s.store.GetCandidate(ctx, id)
		if err != nil || c == nil {
			return err
		}
		var job *domain.JobOpening
		if c.JobOpeningID != nil {
			if job, err = s.store.GetJobOpening(ctx, *c.JobOpeningID); err != nil {
				return err
			}
		}
		jobContext := domain.JobContext(job)
		a := openrouter.MockAnalysis(domain.AnalysisInput{CandidateName: c.FullName, Profile: domain.CandidateProfile(*c), JobContext: jobContext})
		a.CandidateID, a.JobContext, a.Model = id, jobContext, openrouter.MockModel
		if err := s.store.UpsertAnalysis(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

// psikotesStates covers the session states the existing seed misses: Elok
// mid-test, Wahyu with drawings (one waiting for review, one reviewed),
// Eko finished with Matematika and Nur with an expired link.
func (s *seeder) psikotesStates(ctx context.Context) error {
	day := func(n int) time.Time { return s.now.AddDate(0, 0, -n) }

	// Elok: started yesterday, Aritmatika half answered, PAPI not opened
	elok := "seed_psi_elok"
	if err := s.session(ctx, elok, cid("Elok Wulandari"), "in_progress", day(2), ptr(day(1)), nil); err != nil {
		return err
	}
	half := map[string]string{}
	var arIDs []string
	for i, q := range arithmetic {
		qid := fmt.Sprintf("seed_q_ar_%02d", i+1)
		arIDs = append(arIDs, qid)
		if i < 5 {
			half[qid] = q.correct
		}
	}
	if err := s.test(ctx, elok+"_ar", elok, "psi_aritmatika", "in_progress", 0, arIDs, half, nil, nil, ptr(day(1)), nil); err != nil {
		return err
	}
	if err := s.test(ctx, elok+"_papi", elok, "psi_papi_kostick", "pending", 1, nil, nil, nil, nil, nil, nil); err != nil {
		return err
	}
	if err := s.proctor(ctx, "psikotes", elok, []string{"fullscreen_exit", "tab_blur"}, day(1)); err != nil {
		return err
	}

	// Wahyu: finished with two drawings, Baum waiting for review, DAP reviewed
	wahyu := "seed_psi_wahyu"
	done := day(3)
	if err := s.session(ctx, wahyu, cid("Wahyu Kurniawan"), "completed", day(5), ptr(done.Add(-90*time.Minute)), &done); err != nil {
		return err
	}
	for i, inst := range []string{"psi_baum", "psi_dap"} {
		id := fmt.Sprintf("%s_%d", wahyu, i)
		status := []string{"perlu_review", "reviewed"}[i]
		if err := s.test(ctx, id, wahyu, inst, status, i, nil, nil, nil, nil, ptr(done.Add(-80*time.Minute)), &done); err != nil {
			return err
		}
		if err := s.drawing(ctx, id, wahyu, i); err != nil {
			return err
		}
	}
	if err := s.exec(ctx, `UPDATE recruitment.psikotes_session_tests SET review_notes = $2, reviewed_by = $3, reviewed_by_name = $4,
		reviewed_at = $5 WHERE id = $1 AND reviewed_at IS NULL`, wahyu+"_1",
		"Proporsi figur seimbang, detail wajah lengkap. Kesan percaya diri dan terbuka.", seedUser, hrName, done.Add(20*time.Hour)); err != nil {
		return err
	}
	if err := s.proctor(ctx, "psikotes", wahyu, []string{"tab_blur", "tab_blur", "face_not_detected"}, done.Add(-time.Hour)); err != nil {
		return err
	}

	// Nur: the invitation lapsed before she opened it
	if err := s.session(ctx, "seed_psi_nur", cid("Nur Aini"), "expired", day(9), nil, nil); err != nil {
		return err
	}
	if err := s.test(ctx, "seed_psi_nur_ar", "seed_psi_nur", "psi_aritmatika", "pending", 0, nil, nil, nil, nil, nil, nil); err != nil {
		return err
	}

	// Eko finished every test and waits for the psikotes verdict
	return s.finishedSession(ctx, "Eko Prasetya")
}

func (s *seeder) finishedSession(ctx context.Context, name string) error {
	candidate := cid(name)
	id := "seed_psi_" + candidate
	p := people[indexOf(name)]
	done := s.now.AddDate(0, 0, -max(1, p.daysAgo/2))
	started := done.Add(-70 * time.Minute)
	if err := s.session(ctx, id, candidate, "completed", done.AddDate(0, 0, -1), &started, &done); err != nil {
		return err
	}
	right := 6 + pick(candidate+":psi", 5)
	for k, bank := range []struct {
		inst, prefix string
		keys         []string
	}{{"psi_aritmatika", "seed_q_ar_%02d", correctKeys(len(arithmetic), func(i int) string { return arithmetic[i].correct })},
		{"psi_matematika", "seed_q_mt_%02d", correctKeys(len(mathQuestions), func(i int) string { return mathQuestions[i].correct })}} {
		var ids []string
		answers := map[string]string{}
		var mcq []domain.McqQuestion
		for i, key := range bank.keys {
			qid := fmt.Sprintf(bank.prefix, i+1)
			ids = append(ids, qid)
			answers[qid] = key
			if i >= right-k {
				answers[qid] = map[string]string{"a": "b", "b": "c", "c": "d", "d": "a"}[key]
			}
			mcq = append(mcq, domain.McqQuestion{ID: qid, Correct: ptr(key)})
		}
		score, detail := domain.ScoreMcq(mcq, answers)
		sc := float64(score)
		if err := s.test(ctx, fmt.Sprintf("%s_%d", id, k), id, bank.inst, "selesai", k, ids, answers, &sc, domain.McqScoreDetail(detail),
			&started, &done); err != nil {
			return err
		}
	}
	var ids []string
	answers := map[string]string{}
	var pq []domain.PapiQuestion
	for i, pp := range papiPairs {
		qid := fmt.Sprintf("seed_q_papi_%02d", i+1)
		ids = append(ids, qid)
		answers[qid] = []string{"a", "b"}[pick(candidate+qid, 2)]
		pq = append(pq, domain.PapiQuestion{ID: qid, ScaleA: pp.sa, ScaleB: pp.sb})
	}
	if err := s.test(ctx, id+"_papi", id, "psi_papi_kostick", "selesai", 2, ids, answers, nil, domain.PapiScoreDetail(domain.ScorePapi(pq, answers)),
		&started, &done); err != nil {
		return err
	}
	return s.exec(ctx, `INSERT INTO recruitment.candidate_psikotes_summary (candidate_id, recommendation, notes, updated_by, updated_by_name)
		VALUES ($1, 'lolos', $2, $3, $4) ON CONFLICT (candidate_id) DO NOTHING`, candidate,
		fmt.Sprintf("Aritmatika %d/100, profil PAPI seimbang. Lanjut interview.", right*10), seedUser, hrName)
}

// livePipeline gives live candidates past a stage what that stage leaves
// behind: a finished psikotes, an AI interview with transcript and scored
// interviews, for those that have none yet.
func (s *seeder) livePipeline(ctx context.Context) error {
	for i, p := range people {
		path := pathTo(p.status)
		id := candidateID(i)
		var sessions, interviews, aiSessions int
		if err := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM recruitment.psikotes_sessions WHERE candidate_id = $1),
			(SELECT count(*) FROM recruitment.interviews WHERE candidate_id = $1),
			(SELECT count(*) FROM recruitment.interview_ai_sessions WHERE candidate_id = $1)`, id).Scan(&sessions, &interviews, &aiSessions); err != nil {
			return err
		}
		if sessions == 0 && slices.Contains(path, "interview") {
			if err := s.finishedSession(ctx, p.name); err != nil {
				return fmt.Errorf("psikotes %s: %w", p.name, err)
			}
		}
		if (interviews > 0 && aiSessions > 0) || !slices.Contains(path, "interview") {
			continue
		}
		created := s.now.AddDate(0, 0, -p.daysAgo).Add(-time.Duration(pick(p.name, 8)) * time.Hour)
		step := time.Duration(p.daysAgo) * 24 * time.Hour / time.Duration(len(path)+1)
		entered := map[string]time.Time{}
		for k, st := range path {
			entered[st] = created.Add(step * time.Duration(k))
		}
		var live opening
		for _, o := range openings {
			if o.key == p.opening {
				live = o
			}
		}
		o := pastOpening{key: live.key, title: live.title, dept: live.key, salaryMin: live.salaryMin / 100_000, salaryMax: live.salaryMax / 100_000}
		a := applicant{id: id, key: id, name: p.name, source: p.source, quality: 700 + pick(id, 300), created: created, live: true}
		b := &pgx.Batch{}
		if interviews == 0 {
			s.queueInterviews(b, o, a, entered["interview"], entered, i)
		} else {
			s.queueAiInterview(b, o, a, entered["interview"])
		}
		if err := s.pool.SendBatch(ctx, b).Close(); err != nil {
			return fmt.Errorf("interviews %s: %w", p.name, err)
		}
	}
	return nil
}

func correctKeys(n int, key func(int) string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = key(i)
	}
	return out
}

func (s *seeder) session(ctx context.Context, id, candidate, status string, invited time.Time, started, completed *time.Time) error {
	return s.exec(ctx, `INSERT INTO recruitment.psikotes_sessions (id, candidate_id, token, status, webcam_consent, invited_at, expires_at,
		started_at, completed_at, created_by, created_by_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $6, $6) ON CONFLICT (id) DO NOTHING`,
		id, candidate, tokenFor(id), status, started != nil, invited, invited.AddDate(0, 0, 3), started, completed, seedUser, hrName)
}

func (s *seeder) test(ctx context.Context, id, session, instrument, status string, order int, questionIDs []string, answers map[string]string,
	score *float64, detail any, started, completed *time.Time) error {
	if questionIDs == nil {
		questionIDs = []string{}
	}
	if answers == nil {
		answers = map[string]string{}
	}
	var rawDetail []byte
	if detail != nil {
		rawDetail = mustJSON(detail)
	}
	return s.exec(ctx, `INSERT INTO recruitment.psikotes_session_tests (id, session_id, instrument_id, status, question_ids, answers, score,
		score_detail, sort_order, started_at, completed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) ON CONFLICT (id) DO NOTHING`,
		id, session, instrument, status, mustJSON(questionIDs), mustJSON(answers), score, rawDetail, order, started, completed)
}

func (s *seeder) proctor(ctx context.Context, kind, session string, events []string, at time.Time) error {
	for i, e := range events {
		if err := s.exec(ctx, `INSERT INTO recruitment.proctor_events (id, session_type, session_id, event_type, meta, created_at)
			VALUES ($1, $2, $3, $4, '{}', $5) ON CONFLICT (id) DO NOTHING`, fmt.Sprintf("%s_pe_%d", session, i+1), kind, session, e,
			at.Add(time.Duration(i*7)*time.Minute)); err != nil {
			return err
		}
	}
	return nil
}

// drawing stores a small PNG as the uploaded photo of a drawing test.
func (s *seeder) drawing(ctx context.Context, testID, session string, variant int) error {
	if s.files == nil {
		return nil
	}
	key := "psikotes/" + session + "/" + testID + "-seed.png"
	if err := s.files.Put(ctx, key, sketch(variant)); err != nil {
		return err
	}
	return s.exec(ctx, `UPDATE recruitment.psikotes_session_tests SET attachment_key = $2, attachment_content_type = 'image/png'
		WHERE id = $1 AND attachment_key IS NULL`, testID, key)
}

// sketch draws a pencil tree (0) or stick figure (1) on paper.
func sketch(variant int) []byte {
	const w, h = 360, 480
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 248
	}
	ink := color.Gray{Y: 60}
	line := func(x0, y0, x1, y1 int) {
		steps := max(abs(x1-x0), abs(y1-y0))
		for i := 0; i <= steps; i++ {
			x, y := x0+(x1-x0)*i/max(steps, 1), y0+(y1-y0)*i/max(steps, 1)
			img.SetGray(x, y, ink)
			img.SetGray(x+1, y, ink)
		}
	}
	circle := func(cx, cy, r int) {
		for a := 0; a < 360; a++ {
			rad := float64(a) * math.Pi / 180
			x := cx + int(float64(r)*math.Cos(rad))
			y := cy + int(float64(r)*math.Sin(rad))
			img.SetGray(x, y, ink)
		}
	}
	if variant == 0 {
		line(170, 420, 170, 260)
		line(190, 420, 190, 260)
		line(150, 420, 210, 420)
		for i, r := range []int{80, 60, 45} {
			circle(180, 210-i*20, r)
		}
		line(180, 300, 130, 250)
		line(180, 290, 230, 245)
	} else {
		circle(180, 110, 40)
		line(180, 150, 180, 300)
		line(180, 190, 120, 250)
		line(180, 190, 240, 250)
		line(180, 300, 130, 410)
		line(180, 300, 230, 410)
		circle(165, 105, 4)
		circle(195, 105, 4)
		line(168, 128, 192, 128)
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// portalLinks keeps one working link per candidate portal: Melati's
// psikotes, Yoga's AI interview and Oscar's offer. Each run moves their
// expiry forward while the candidate has not acted on them.
func (s *seeder) portalLinks(ctx context.Context) error {
	if err := s.exec(ctx, `UPDATE recruitment.psikotes_sessions SET status = 'sent', expires_at = $2
		WHERE id = $1 AND started_at IS NULL`, "seed_psi_maya", s.now.AddDate(0, 0, 3)); err != nil {
		return err
	}
	yoga := cid("Yoga Aditya")
	ai := "seed_ai_portal_" + yoga
	invited := s.now.Add(-2 * time.Hour)
	if err := s.exec(ctx, `INSERT INTO recruitment.interview_ai_sessions (id, candidate_id, token, status, max_questions, invited_at, expires_at,
		created_by, created_by_name, created_at, updated_at) VALUES ($1, $2, $3, 'sent', 6, $4, $5, $6, $7, $4, $4) ON CONFLICT (id) DO NOTHING`,
		ai, yoga, tokenFor(ai), invited, invited.AddDate(0, 0, 7), seedUser, hrName); err != nil {
		return err
	}
	if err := s.exec(ctx, `UPDATE recruitment.interview_ai_sessions SET status = 'sent', expires_at = $2 WHERE id = $1 AND started_at IS NULL`,
		ai, s.now.AddDate(0, 0, 7)); err != nil {
		return err
	}
	return s.exec(ctx, `UPDATE recruitment.candidate_offers SET status = 'sent', expires_at = $2
		WHERE candidate_id = $1 AND responded_at IS NULL AND status IN ('sent', 'expired')`, cid("Oscar Pangaribuan"), s.now.AddDate(0, 0, 7))
}

// PortalURLs are the demo portal paths kept valid by the seed.
func PortalURLs() map[string]string {
	return map[string]string{
		"psikotes":  "/psikotes/" + tokenFor("seed_psi_maya"),
		"wawancara": "/wawancara/" + tokenFor("seed_ai_portal_"+cid("Yoga Aditya")),
		"offer":     "/offer/" + tokenFor("seed_offer_"+cid("Oscar Pangaribuan")),
	}
}
