package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

// pastOpening is a past (or still live) opening; dept is the key of the
// live opening whose department it shares.
type pastOpening struct {
	key, title, dept, employment string
	openedDaysAgo, openDays      int
	headcount, applicants        int
	salaryMin, salaryMax         int64 // juta rupiah tenths: 55 = 5,5 juta
	unfilled, live               bool
}

var pastOpenings = []pastOpening{
	// the year before, so "vs tahun lalu" has something to compare with
	{key: "smm0", title: "Social Media Specialist", dept: "smm", openedDaysAgo: 760, openDays: 45, headcount: 1, applicants: 18, salaryMin: 50, salaryMax: 65},
	{key: "ae0", title: "Account Executive", dept: "ae", openedDaysAgo: 730, openDays: 55, headcount: 1, applicants: 16, salaryMin: 55, salaryMax: 85},
	{key: "gd0", title: "Graphic Designer", dept: "gd", openedDaysAgo: 700, openDays: 40, headcount: 1, applicants: 17, salaryMin: 45, salaryMax: 65},
	{key: "seo0", title: "SEO Specialist", dept: "seo", openedDaysAgo: 665, openDays: 50, headcount: 1, applicants: 12, salaryMin: 55, salaryMax: 80},
	{key: "pm0", title: "Meta Ads Specialist", dept: "pm", openedDaysAgo: 630, openDays: 45, headcount: 1, applicants: 14, salaryMin: 70, salaryMax: 100},
	{key: "cw0", title: "Content Writer", dept: "gd", openedDaysAgo: 590, openDays: 40, headcount: 1, applicants: 16, salaryMin: 40, salaryMax: 60},
	{key: "sdr0", title: "Sales Development Representative", dept: "ae", openedDaysAgo: 555, openDays: 50, headcount: 2, applicants: 19, salaryMin: 45, salaryMax: 65},
	{key: "ve0", title: "Video Editor", dept: "gd", openedDaysAgo: 520, openDays: 40, headcount: 1, applicants: 15, salaryMin: 45, salaryMax: 70},
	{key: "cm0", title: "Community Manager", dept: "smm", openedDaysAgo: 490, openDays: 45, headcount: 1, applicants: 15, salaryMin: 50, salaryMax: 70},
	{key: "cw1", title: "Content Writer", dept: "gd", openedDaysAgo: 455, openDays: 45, headcount: 1, applicants: 20, salaryMin: 45, salaryMax: 65},
	{key: "smm1", title: "Social Media Specialist", dept: "smm", openedDaysAgo: 420, openDays: 40, headcount: 2, applicants: 26, salaryMin: 50, salaryMax: 70},
	{key: "seo1", title: "SEO Specialist", dept: "seo", openedDaysAgo: 390, openDays: 50, headcount: 1, applicants: 16, salaryMin: 60, salaryMax: 85},
	{key: "ae1", title: "Account Executive", dept: "ae", openedDaysAgo: 365, openDays: 60, headcount: 2, applicants: 24, salaryMin: 60, salaryMax: 90},
	{key: "ve1", title: "Video Editor", dept: "gd", openedDaysAgo: 335, openDays: 35, headcount: 1, applicants: 19, salaryMin: 50, salaryMax: 75},
	{key: "pm1", title: "Performance Marketing Specialist", dept: "pm", openedDaysAgo: 305, openDays: 55, headcount: 1, applicants: 18, salaryMin: 80, salaryMax: 120},
	{key: "cm1", title: "Community Manager", dept: "smm", openedDaysAgo: 275, openDays: 40, headcount: 1, applicants: 21, salaryMin: 55, salaryMax: 75},
	{key: "gd1", title: "Graphic Designer", dept: "gd", openedDaysAgo: 245, openDays: 45, headcount: 1, applicants: 23, salaryMin: 50, salaryMax: 70},
	{key: "sdr1", title: "Sales Development Representative", dept: "ae", openedDaysAgo: 215, openDays: 50, headcount: 2, applicants: 25, salaryMin: 50, salaryMax: 70},
	{key: "wa1", title: "Web Analyst", dept: "seo", openedDaysAgo: 185, openDays: 40, headcount: 1, applicants: 15, salaryMin: 70, salaryMax: 100},
	{key: "ga1", title: "Google Ads Specialist", dept: "pm", openedDaysAgo: 155, openDays: 45, headcount: 1, applicants: 17, salaryMin: 75, salaryMax: 110},
	{key: "kol1", title: "KOL Specialist", dept: "smm", openedDaysAgo: 125, openDays: 40, headcount: 1, applicants: 22, salaryMin: 60, salaryMax: 80},
	{key: "md1", title: "Motion Designer", dept: "gd", openedDaysAgo: 95, openDays: 50, headcount: 1, applicants: 18, salaryMin: 60, salaryMax: 90},
	{key: "kam1", title: "Key Account Manager", dept: "ae", openedDaysAgo: 80, openDays: 45, headcount: 1, applicants: 14, salaryMin: 90, salaryMax: 130, unfilled: true},
	{key: "cp1", title: "Copywriter", dept: "gd", openedDaysAgo: 50, openDays: 60, headcount: 1, applicants: 20, salaryMin: 50, salaryMax: 70, live: true},
}

var firstNames = []string{"Adi", "Bella", "Cahya", "Dewi", "Eka", "Fikri", "Gita", "Hana", "Ilham", "Joko", "Kirana", "Lutfi", "Maya",
	"Nadia", "Omar", "Putri", "Rizky", "Salsa", "Taufik", "Utari", "Vina", "Wahyu", "Yusuf", "Zulfa", "Arif", "Bunga", "Dimas", "Endah",
	"Feri", "Gilang", "Indah", "Krisna", "Laila", "Mega", "Novi", "Pandu", "Ratna", "Sigit", "Tasya", "Wulan"}

var lastNames = []string{"Saputra", "Lestari", "Nugroho", "Permata", "Hidayat", "Kurniawan", "Anggraini", "Santoso", "Wijayanti",
	"Pratama", "Siregar", "Rahayu", "Setiadi", "Halim", "Purnomo", "Maharani", "Gunawan", "Utami", "Hakim", "Febrianti", "Simanjuntak",
	"Wibisono", "Puspita", "Ramadhani", "Susanto", "Kusuma", "Firmansyah", "Andriani", "Tanjung", "Sulistyo"}

var domiciles = []string{"Jakarta Selatan", "Jakarta Barat", "Jakarta Timur", "Jakarta Utara", "Jakarta Pusat", "Depok", "Bekasi",
	"Tangerang", "Tangerang Selatan", "Bogor"}

// sourceWeights sums to 100.
var sourceWeights = []struct {
	source string
	weight int
}{{"portal", 30}, {"jobstreet", 20}, {"instagram", 14}, {"referral", 12}, {"jobfair", 7}, {"headhunter", 5},
	{"internal_referral", 5}, {"walk_in", 3}, {"other", 4}}

func sourceFor(key string) string {
	n := pick(key+":src", 100)
	for _, w := range sourceWeights {
		if n < w.weight {
			return w.source
		}
		n -= w.weight
	}
	return "portal"
}

// quality ranks applicants of an opening; referrals and headhunters lead.
func quality(key, source string) int {
	q := pick(key+":q", 1000)
	switch source {
	case "referral", "internal_referral", "headhunter":
		q += 220
	case "jobstreet", "walk_in":
		q -= 80
	}
	return q
}

func pastOpeningID(key string) string { return "seed_job_h_" + key }

type applicant struct {
	id, key, name, source string
	quality               int
	created               time.Time
	live                  bool // a live demo candidate: always gets an AI interview
}

// stayDays is the minimum and spread of days spent in a stage.
var stayDays = map[string][2]int{"applied": {1, 4}, "screening": {2, 5}, "psikotes": {2, 5}, "interview": {4, 9}, "offer": {2, 5}}

// stay is how long a candidate spends in a stage before moving on.
func stay(key, stage string) time.Duration {
	d := stayDays[stage]
	hours := (d[0]+pick(key+":"+stage, d[1]))*24 + pick(key+":"+stage+":h", 20)
	return time.Duration(hours) * time.Hour
}

// history adds two years of hiring behind the live demo: past openings
// (closed, most of them filled), their applicants with stage histories,
// psikotes sessions, AI interviews, scored interviews and offers, so the
// dashboard trends have something to show. Rows have stable ids and are
// inserted with ON CONFLICT DO NOTHING, so a second run changes nothing.
// Hires carry hired_event_at, so no hire event goes out for them.
func (s *seeder) history(ctx context.Context) error {
	for _, o := range pastOpenings {
		if err := s.pastOpening(ctx, o); err != nil {
			return fmt.Errorf("past opening %s: %w", o.key, err)
		}
	}
	return nil
}

func (s *seeder) pastOpening(ctx context.Context, o pastOpening) error {
	id := pastOpeningID(o.key)
	opened := s.now.AddDate(0, 0, -o.openedDaysAgo)
	closing := opened.AddDate(0, 0, o.openDays)
	status, updated := "closed", closing
	if o.live {
		status, updated = "published", opened
	}
	var deptID, manager *string
	dept := map[string]string{"smm": "Social Media", "seo": "SEO", "gd": "Creative", "pm": "Ads", "ae": "Sales"}[o.dept]
	if d, ok := s.depts[o.dept]; ok {
		deptID, dept = &d.ID, d.Name
	}
	if e, ok := s.staff[o.dept]; ok {
		manager = e.UserID
	}
	slug := fmt.Sprintf("%s-%s", domain.Slugify(o.title), opened.In(domain.Jakarta).Format("200601"))
	if err := s.exec(ctx, `INSERT INTO recruitment.job_openings (id, title, slug, department_id, department, location, employment_type,
		work_mode, headcount, description, requirements, benefits, salary_min, salary_max, hiring_manager_user_id, status, published_at,
		closing_date, created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'Full-time', 'Hybrid', $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $17, $18, $19)
		ON CONFLICT (id) DO NOTHING`,
		id, o.title, slug, deptID, dept, domain.DefaultLocation, o.headcount,
		"Posisi "+o.title+" untuk tim "+dept+" Mixist, mendukung akun klien agency.",
		"Pengalaman relevan minimal 1 tahun, portofolio atau studi kasus, terbiasa bekerja dengan target klien.",
		"BPJS Kesehatan dan Ketenagakerjaan, laptop kerja, budget pelatihan.",
		domain.Rupiah(o.salaryMin*100_000), domain.Rupiah(o.salaryMax*100_000), manager, status, opened,
		domain.DayKey(closing), seedUser, opened.Add(-48*time.Hour), updated); err != nil {
		return err
	}

	apps := make([]applicant, o.applicants)
	window := float64(o.openDays) * 0.85
	for k := range apps {
		key := fmt.Sprintf("%s_%02d", o.key, k+1)
		src := sourceFor(key)
		offset := time.Duration(window*float64(pick(key+":d", 1000))/1000*24) * time.Hour
		apps[k] = applicant{id: "seed_hist_" + key, key: key, source: src, quality: quality(key, src), created: opened.Add(offset),
			name: firstNames[pick(key+":f", len(firstNames))] + " " + lastNames[pick(key+":l", len(lastNames))]}
	}
	// best first: the stage an applicant reaches follows the rank
	ranked := slices.Clone(apps)
	slices.SortStableFunc(ranked, func(a, b applicant) int { return b.quality - a.quality })
	n := len(ranked)
	hires := o.headcount
	if o.unfilled || o.live {
		hires = 0
	}
	for rank, a := range ranked {
		reach := 0 // furthest funnel index
		switch {
		case rank < hires:
			reach = 5
		case rank < hires+pick(o.key+":extra", 2):
			reach = 4
		case rank < n*30/100:
			reach = 3
		case rank < n*50/100:
			reach = 2
		case rank < n*75/100:
			reach = 1
		}
		if err := s.pastApplicant(ctx, o, id, a, reach, rank); err != nil {
			return fmt.Errorf("%s: %w", a.id, err)
		}
	}
	return nil
}

type step struct {
	status string
	at     time.Time
}

func (s *seeder) pastApplicant(ctx context.Context, o pastOpening, jobID string, a applicant, reach, rank int) error {
	// walk the funnel; a live opening stops at today
	path := []step{{"applied", a.created}}
	at := a.created
	for i := 1; i <= reach; i++ {
		at = at.Add(stay(a.key, domain.FunnelStatuses[i-1]))
		if at.After(s.now) {
			break
		}
		path = append(path, step{domain.FunnelStatuses[i], at})
	}
	last := path[len(path)-1]
	finalStatus := last.status
	offerOutcome := ""
	if last.status == "offer" && !o.live {
		offerOutcome = []string{"declined", "expired", "declined"}[pick(a.key+":offer", 3)]
	}
	// past openings close every application; a live one only drops the weakest
	if last.status != "hired" && (!o.live || reach <= 1) {
		exit := "rejected"
		if last.status == "interview" || last.status == "offer" {
			exit = []string{"talent_pool", "rejected", "talent_pool"}[pick(a.key+":exit", 3)]
		}
		exitAt := last.at.Add(stay(a.key, last.status))
		path = append(path, step{exit, exitAt})
		if exit == "talent_pool" && pick(a.key+":arch", 4) == 0 {
			path = append(path, step{"archived", exitAt.AddDate(0, 2, 0)})
		}
		finalStatus = path[len(path)-1].status
		if path[len(path)-1].at.After(s.now) {
			path, finalStatus = path[:len(path)-1], path[len(path)-2].status
		}
	}

	var hiredAt *time.Time
	if finalStatus == "hired" {
		hiredAt = &path[len(path)-1].at
	}
	salary := (o.salaryMin + int64(pick(a.key+":sal", int(o.salaryMax-o.salaryMin+10)))) * 100_000
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `INSERT INTO recruitment.candidates (id, full_name, email, phone, domicile, source, job_opening_id, status,
		last_experience, last_education, availability, expected_salary, hired_at, hired_event_at, created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13, $14, $14, $15, $16)
		ON CONFLICT (id) DO NOTHING`,
		a.id, a.name, strings.ReplaceAll(strings.ToLower(a.name), " ", ".")+fmt.Sprintf("%d@gmail.com", 10+pick(a.key+":em", 90)), phoneOf(a.name+a.key), domiciles[pick(a.key+":dom", len(domiciles))], a.source, jobID,
		finalStatus, o.title+" di agency atau brand ("+fmt.Sprint(1+pick(a.key+":xp", 4))+" tahun)", "S1, universitas di Jabodetabek",
		[]string{"immediate", "1_week", "2_weeks", "1_month"}[pick(a.key+":av", 4)], salary, hiredAt, seedUser, a.created,
		path[len(path)-1].at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil // seeded before
	}
	first := "Lamaran masuk melalui portal karir untuk " + o.title
	firstType := "applied"
	if a.source != "portal" {
		first, firstType = "Kandidat ditambahkan manual (sumber: "+domain.SourceLabel(a.source)+")", "created"
	}
	b := &pgx.Batch{}
	activity := func(suffix, typ, desc string, at time.Time) {
		b.Queue(`INSERT INTO recruitment.candidate_activities (id, candidate_id, activity_type, description, created_by, created_by_name, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO NOTHING`, a.id+"_"+suffix, a.id, typ, desc, seedUser, hrName, at)
	}
	activity("act_00", firstType, first, a.created)
	for k := 1; k < len(path); k++ {
		activity(fmt.Sprintf("act_%02d", k), "status_change", domain.StageChangeDescription(path[k-1].status, path[k].status), path[k].at)
	}
	entered := map[string]time.Time{}
	for _, p := range path {
		entered[p.status] = p.at
	}
	if _, ok := entered["screening"]; ok {
		rec := "lolos"
		if _, passed := entered["psikotes"]; !passed {
			rec = []string{"tidak_lolos", "hold"}[pick(a.key+":scr", 2)]
		}
		b.Queue(`INSERT INTO recruitment.candidate_screenings (candidate_id, contacted, interested, availability_note, confirmed_salary,
			willing_shift, willing_placement, notes, recommendation, updated_by, updated_by_name, created_at, updated_at)
			VALUES ($1, true, true, $2, $3, true, true, $4, $5, $6, $7, $8, $8) ON CONFLICT (candidate_id) DO NOTHING`,
			a.id, availabilityNote([]string{"immediate", "1_week", "2_weeks", "1_month"}[pick(a.key+":av", 4)]), salary,
			"Dihubungi via WhatsApp untuk konfirmasi minat dan ekspektasi gaji.", rec, seedUser, hrName, entered["screening"])
	}
	if t, ok := entered["psikotes"]; ok {
		s.queuePsikotes(b, a, t, entered)
	}
	if t, ok := entered["interview"]; ok {
		s.queueInterviews(b, o, a, t, entered, rank)
	}
	if t, ok := entered["offer"]; ok {
		s.queueOffer(b, o, a, t, finalStatus, offerOutcome, entered)
	}
	if err := tx.SendBatch(ctx, b).Close(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// next is when the candidate left a stage (or now when it did not).
func (s *seeder) next(entered map[string]time.Time, stage string) time.Time {
	order := []string{"applied", "screening", "psikotes", "interview", "offer", "hired", "talent_pool", "rejected", "archived"}
	start := entered[stage]
	best := s.now
	for _, st := range order {
		if t, ok := entered[st]; ok && t.After(start) && t.Before(best) {
			best = t
		}
	}
	return best
}

func (s *seeder) queuePsikotes(b *pgx.Batch, a applicant, invited time.Time, entered map[string]time.Time) {
	id := "seed_psi_" + a.id
	left := s.next(entered, "psikotes")
	_, passed := entered["interview"]
	done := passed || pick(a.key+":psi", 5) > 0
	completed := invited.Add(time.Duration(10+pick(a.key+":psih", 30)) * time.Hour)
	if completed.After(left) {
		completed = left.Add(-time.Hour)
	}
	status, started, finished := "expired", (*time.Time)(nil), (*time.Time)(nil)
	if done && completed.Before(s.now) {
		status, started, finished = "completed", ptr(completed.Add(-55*time.Minute)), &completed
	} else if invited.AddDate(0, 0, 3).After(s.now) {
		status = "sent"
	}
	b.Queue(`INSERT INTO recruitment.psikotes_sessions (id, candidate_id, token, status, webcam_consent, invited_at, expires_at, started_at,
		completed_at, created_by, created_by_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $6, $6) ON CONFLICT (id) DO NOTHING`,
		id, a.id, tokenFor(id), status, started != nil, invited, invited.AddDate(0, 0, 3), started, finished, seedUser, hrName)
	if finished == nil {
		return
	}
	// a stronger applicant answers more questions right: 20..100
	correct := max(2, min(10, 3+a.quality*6/1100+pick(a.key+":ar", 2)))
	var arIDs []string
	arAnswers := map[string]string{}
	var mcq []domain.McqQuestion
	for i, q := range arithmetic {
		qid := fmt.Sprintf("seed_q_ar_%02d", i+1)
		arIDs = append(arIDs, qid)
		given := q.correct
		if i >= correct {
			given = map[string]string{"a": "b", "b": "c", "c": "d", "d": "a"}[q.correct]
		}
		arAnswers[qid] = given
		mcq = append(mcq, domain.McqQuestion{ID: qid, Correct: ptr(q.correct)})
	}
	score, arDetail := domain.ScoreMcq(mcq, arAnswers)
	b.Queue(`INSERT INTO recruitment.psikotes_session_tests (id, session_id, instrument_id, status, question_ids, answers, score, score_detail,
		sort_order, started_at, completed_at) VALUES ($1, $2, 'psi_aritmatika', 'selesai', $3, $4, $5, $6, 0, $7, $8) ON CONFLICT (id) DO NOTHING`,
		id+"_ar", id, mustJSON(arIDs), mustJSON(arAnswers), score, mustJSON(domain.McqScoreDetail(arDetail)), started, finished)
	var papiIDs []string
	papiAnswers := map[string]string{}
	var pq []domain.PapiQuestion
	for i, pp := range papiPairs {
		qid := fmt.Sprintf("seed_q_papi_%02d", i+1)
		papiIDs = append(papiIDs, qid)
		papiAnswers[qid] = []string{"a", "b"}[pick(a.key+qid, 2)]
		pq = append(pq, domain.PapiQuestion{ID: qid, ScaleA: pp.sa, ScaleB: pp.sb})
	}
	b.Queue(`INSERT INTO recruitment.psikotes_session_tests (id, session_id, instrument_id, status, question_ids, answers, score_detail,
		sort_order, started_at, completed_at) VALUES ($1, $2, 'psi_papi_kostick', 'selesai', $3, $4, $5, 1, $6, $7) ON CONFLICT (id) DO NOTHING`,
		id+"_papi", id, mustJSON(papiIDs), mustJSON(papiAnswers), mustJSON(domain.PapiScoreDetail(domain.ScorePapi(pq, papiAnswers))),
		started, finished)
	rec := "tidak_lolos"
	if _, ok := entered["interview"]; ok {
		rec = "lolos"
	}
	b.Queue(`INSERT INTO recruitment.candidate_psikotes_summary (candidate_id, recommendation, notes, updated_by, updated_by_name)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT (candidate_id) DO NOTHING`, a.id, rec,
		fmt.Sprintf("Aritmatika %d/100.", score), seedUser, hrName)
}

// queueAiInterview invites the applicant to the AI interview when the
// stage starts; a finished one carries its transcript.
func (s *seeder) queueAiInterview(b *pgx.Batch, o pastOpening, a applicant, start time.Time) {
	id := "seed_ai_" + a.id
	status, done := "completed", start.Add(time.Duration(6+pick(a.key+":aih", 30))*time.Hour)
	if (!a.live && pick(a.key+":aix", 6) == 0) || done.After(s.now) {
		status = "expired"
		if start.AddDate(0, 0, 7).After(s.now) {
			status = "sent"
		}
	}
	var started, completed *time.Time
	if status == "completed" {
		started, completed = ptr(done.Add(-20*time.Minute)), &done
	}
	b.Queue(`INSERT INTO recruitment.interview_ai_sessions (id, candidate_id, token, status, webcam_consent, max_questions, invited_at,
		expires_at, started_at, completed_at, created_by, created_by_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 5, $6, $7, $8, $9, $10, $11, $6, $6) ON CONFLICT (id) DO NOTHING`,
		id, a.id, tokenFor(id), status, started != nil, start, start.AddDate(0, 0, 7), started, completed, seedUser, hrName)
	if completed != nil {
		s.queueTranscript(b, o, a, id, *started)
	}
}

func (s *seeder) queueInterviews(b *pgx.Batch, o pastOpening, a applicant, start time.Time, entered map[string]time.Time, rank int) {
	left := s.next(entered, "interview")
	_, offered := entered["offer"]
	// AI interview for most applicants at this stage
	if a.live || pick(a.key+":ai", 10) < 7 {
		s.queueAiInterview(b, o, a, start)
	}
	types := []string{"hrd"}
	if offered || pick(a.key+":hm", 2) == 0 {
		types = append(types, "hiring_manager")
	}
	for i, typ := range types {
		when := start.Add(time.Duration(1+i*3+pick(a.key+":ivd"+typ, 2)) * 24 * time.Hour)
		when = time.Date(when.Year(), when.Month(), when.Day(), 9+pick(a.key+":ivh"+typ, 7), 0, 0, 0, domain.Jakarta)
		if !when.Before(left) {
			when = left.Add(-3 * time.Hour)
		}
		id := fmt.Sprintf("seed_iv_%s_%d", a.id, i+1)
		empID, userID, name := s.interviewer(o.dept, typ)
		status := "completed"
		if pick(a.key+":ivc"+typ, 12) == 0 {
			status = "cancelled"
		}
		if when.After(s.now) {
			status = "scheduled"
		}
		var scorecard []byte
		var overall *float64
		rec := ""
		if status == "completed" {
			base := 2 + min(3, a.quality*3/1000)
			items := make([]domain.ScoreItem, len(domain.DefaultScorecardCriteria))
			for k, c := range domain.DefaultScorecardCriteria {
				items[k] = domain.ScoreItem{Criterion: c, Score: max(1, min(5, base+pick(fmt.Sprintf("%s:sc%d%s", a.key, k, typ), 3)-1))}
			}
			rec = "pool"
			switch {
			case offered:
				rec = "proceed"
			case rank%3 == 0:
				rec = "reject"
			}
			clean, avg, err := domain.ValidateScorecard(items, rec, "")
			if err != nil {
				continue
			}
			scorecard, _ = json.Marshal(clean)
			overall = &avg
		}
		if scorecard == nil {
			scorecard = []byte("[]")
		}
		mode, link, location := "offline", (*string)(nil), ptr("Kantor Mixist, Kemang, Jakarta Selatan")
		if pick(a.key+":mode"+typ, 2) == 0 {
			mode, link, location = "online", ptr(fmt.Sprintf("https://meet.google.com/whi-%03d-hst", pick(id, 1000))), nil
		}
		var cancel *string
		if status == "cancelled" {
			cancel = ptr("Kandidat meminta jadwal ulang")
		}
		b.Queue(`INSERT INTO recruitment.interviews (id, candidate_id, interviewer_employee_id, interviewer_user_id, interviewer_name,
			interview_date, duration_minutes, type, mode, meeting_link, location, status, scorecard, overall_score, recommendation, cancel_reason,
			created_by, updated_by, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, 60, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $16, $17, $6) ON CONFLICT (id) DO NOTHING`,
			id, a.id, empID, userID, name, when, typ, mode, link, location, status, scorecard, overall, nilIfBlank(rec), cancel, seedUser,
			start)
	}
}

// aiTopics are the screening topics of the AI interview, in order.
var aiTopics = []string{"pembuka", "keahlian", "motivasi", "ketersediaan", "gaji"}

// queueTranscript writes a five-turn AI interview and its mock summary.
func (s *seeder) queueTranscript(b *pgx.Batch, o pastOpening, a applicant, sessionID string, started time.Time) {
	answers := map[string]string{
		"pembuka": fmt.Sprintf("Saya %s, %d tahun terakhir bekerja sebagai %s di agency dan brand lokal.", a.name, 1+pick(a.key+":xp", 4),
			strings.ToLower(o.title)),
		"keahlian":     "Saya paling kuat di perencanaan, eksekusi, dan membaca data hasil kampanye. Sehari-hari memakai Spreadsheet, Canva, dan tools analitik.",
		"motivasi":     "Saya ingin menangani lebih banyak klien dari industri berbeda dan belajar strategi di agency.",
		"ketersediaan": []string{"Bisa mulai segera.", "Perlu dua minggu untuk serah terima.", "Satu bulan notice di kantor sekarang."}[pick(a.key+":aiav", 3)],
		"gaji":         fmt.Sprintf("Harapan saya sekitar Rp %d juta per bulan.", (o.salaryMin+o.salaryMax)/20),
	}
	var turns []domain.Turn
	asked := started.Add(time.Minute)
	for i, topic := range aiTopics {
		question := domain.FallbackQuestions[topic]
		if i == 0 {
			question = "Halo " + a.name + "! Terima kasih sudah meluangkan waktu untuk interview posisi " + o.title + ". " + question
		}
		answered := asked.Add(time.Duration(60+pick(a.key+topic, 90)) * time.Second)
		t := domain.Turn{TurnNo: i + 1, Topic: ptr(topic), Question: question, Answer: ptr(answers[topic]), AnswerMode: ptr("text"),
			AskedAt: asked, AnsweredAt: &answered}
		turns = append(turns, t)
		b.Queue(`INSERT INTO recruitment.interview_ai_turns (id, session_id, turn_no, topic, question, answer_transcript, answer_mode, asked_at,
			answered_at) VALUES ($1, $2, $3, $4, $5, $6, 'text', $7, $8) ON CONFLICT (id) DO NOTHING`,
			fmt.Sprintf("%s_t%d", sessionID, i+1), sessionID, i+1, topic, question, answers[topic], asked, answered)
		asked = answered.Add(10 * time.Second)
	}
	summary := openrouter.MockSummary(domain.Transcript{CandidateName: a.name, PositionTitle: o.title, MaxQuestions: 5, Turns: turns})
	b.Queue(`UPDATE recruitment.interview_ai_sessions SET ai_summary = $2, summary_model = $3, summarized_at = $4 WHERE id = $1 AND ai_summary IS NULL`,
		sessionID, mustJSON(summary), openrouter.MockModel, asked)
}

func (s *seeder) queueOffer(b *pgx.Batch, o pastOpening, a applicant, sent time.Time, final, outcome string, entered map[string]time.Time) {
	id := "seed_offer_" + a.id
	span := o.salaryMax - o.salaryMin
	salary := (o.salaryMin + span/3 + int64(pick(a.key+":os", int(span*2/3+1)))) * 100_000
	if a.source == "headhunter" && pick(a.key+":hh", 2) == 0 {
		salary = o.salaryMax*100_000 + 500_000 // above the band to close a sought-after hire
	}
	status, responded := "sent", (*time.Time)(nil)
	switch {
	case final == "hired":
		status, responded = "accepted", ptr(entered["hired"])
	case outcome == "declined":
		status, responded = "declined", ptr(sent.Add(time.Duration(30+pick(a.key+":odh", 90))*time.Hour))
	case outcome == "expired":
		status = "expired"
	case pick(a.key+":neg", 2) == 0:
		status, responded = "negotiating", ptr(sent.Add(26*time.Hour))
	}
	if responded != nil && responded.After(s.now) {
		status, responded = "sent", nil
	}
	note := map[string]string{"accepted": "Siap bergabung, terima kasih.", "declined": "Terima kasih, saya memilih tawaran lain.",
		"negotiating": "Apakah gaji pokok bisa dinaikkan sedikit?"}[status]
	var source *string
	if responded != nil {
		source = ptr("portal")
	}
	b.Queue(`INSERT INTO recruitment.candidate_offers (id, candidate_id, version, token, status, position_title, base_salary, benefits,
		start_date, response_note, responded_at, response_source, sent_at, expires_at, created_by, created_by_name, created_at, updated_at)
		VALUES ($1, $2, 1, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $12, COALESCE($10::timestamptz, $12::timestamptz)) ON CONFLICT (id) DO NOTHING`,
		id, a.id, tokenFor(id), status, o.title, domain.Rupiah(salary), mustJSON(offerBenefits), domain.DayKey(sent.AddDate(0, 0, 21)),
		nilIfBlank(note), responded, source, sent, sent.AddDate(0, 0, 7), seedUser, hrName)
	b.Queue(`INSERT INTO recruitment.candidate_activities (id, candidate_id, activity_type, description, created_by, created_by_name, created_at)
		VALUES ($1, $2, 'offer_sent', $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`, id+"_sent", a.id, domain.OfferCreatedDescription(1, 7),
		seedUser, hrName, sent)
}

func nilIfBlank(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
