package seed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

type person struct {
	name, domicile, opening, status, source string
	experience, education, availability     string
	salary                                  int64 // million rupiah tenths: 65 = 6.5 juta
	daysAgo                                 int
}

var people = []person{
	// Social Media Specialist
	{"Anisa Rahmawati", "Jakarta Selatan", "smm", "applied", "portal", "Admin media sosial UMKM kuliner (1 tahun)", "S1 Ilmu Komunikasi, Universitas Pancasila", "immediate", 55, 2},
	{"Bagas Pratama", "Depok", "smm", "applied", "instagram", "Content creator freelance TikTok", "D3 Broadcasting, Politeknik Negeri Jakarta", "1_week", 50, 3},
	{"Citra Lestari", "Tangerang Selatan", "smm", "screening", "jobstreet", "Social media officer brand kosmetik (2 tahun)", "S1 Manajemen Pemasaran, Universitas Bina Nusantara", "2_weeks", 65, 9},
	{"Dimas Arya Wibowo", "Bekasi", "smm", "screening", "portal", "Community manager startup edukasi", "S1 Ilmu Komunikasi, Universitas Mercu Buana", "1_month", 60, 11},
	{"Elok Wulandari", "Jakarta Timur", "smm", "psikotes", "referral", "Social media specialist agency kecil (2 tahun), Instagram dan Canva", "S1 Desain Komunikasi Visual, Universitas Trisakti", "2_weeks", 70, 15},
	{"Fajar Setiawan", "Jakarta Barat", "smm", "interview", "portal", "Social media lead brand fashion lokal (3 tahun)", "S1 Ilmu Komunikasi, Universitas Indonesia", "1_month", 75, 19},
	{"Ayu Kusumawati", "Jakarta Selatan", "smm", "interview", "instagram", "Copywriter dan content planner Instagram, TikTok (2 tahun)", "S1 Sastra Inggris, Universitas Padjadjaran", "2_weeks", 68, 21},
	{"Hendra Wijaya", "Tangerang", "smm", "offer", "jobstreet", "Social media specialist e-commerce (3 tahun), laporan insight bulanan", "S1 Manajemen, Universitas Tarumanagara", "1_month", 72, 26},
	{"Intan Maharani", "Bogor", "smm", "talent_pool", "portal", "Content writer media online (1 tahun)", "S1 Jurnalistik, Universitas Islam Negeri Jakarta", "immediate", 55, 34},
	{"Joko Susilo", "Bekasi", "smm", "rejected", "walk_in", "Staf administrasi toko", "SMK Multimedia", "immediate", 45, 30},
	// SEO Analyst
	{"Kurnia Sari", "Jakarta Pusat", "seo", "applied", "portal", "Content writer SEO freelance, WordPress", "S1 Sastra Indonesia, Universitas Negeri Jakarta", "1_week", 60, 1},
	{"Lukman Prakoso", "Depok", "seo", "screening", "jobstreet", "SEO specialist toko online (1 tahun), Google Analytics dan Search Console", "S1 Sistem Informasi, Universitas Gunadarma", "2_weeks", 70, 8},
	{"Melati Anggraini", "Jakarta Selatan", "seo", "psikotes", "referral", "SEO analyst portal berita (2 tahun), Ahrefs dan Search Console", "S1 Teknik Informatika, Universitas Bina Nusantara", "1_month", 80, 13},
	{"Nanda Prasetyo", "Tangerang Selatan", "seo", "interview", "portal", "SEO specialist agency (3 tahun): technical SEO, Ahrefs, Semrush, Google Analytics", "S1 Ilmu Komputer, Universitas Indonesia", "1_month", 85, 18},
	{"Okta Ramadhan", "Bekasi", "seo", "interview", "jobstreet", "Digital marketing generalist dengan fokus SEO dan WordPress", "S1 Manajemen Informatika, Universitas Esa Unggul", "2_weeks", 75, 20},
	{"Putri Ayuningtyas", "Jakarta Timur", "seo", "hired", "referral", "SEO analyst marketplace (2 tahun), Search Console dan Excel", "S1 Statistika, Institut Pertanian Bogor", "1_month", 78, 52},
	{"Rizky Firmansyah", "Bogor", "seo", "talent_pool", "portal", "Web content specialist (1 tahun)", "S1 Teknik Informatika, Universitas Pakuan", "immediate", 65, 37},
	{"Salsabila Putri", "Depok", "seo", "rejected", "instagram", "Fresh graduate, magang digital marketing", "S1 Ilmu Komunikasi, Universitas Gunadarma", "immediate", 50, 28},
	// Graphic Designer
	{"Teguh Wicaksono", "Jakarta Selatan", "gd", "applied", "portal", "Desainer grafis freelance, Figma dan Illustrator", "S1 Desain Komunikasi Visual, Institut Kesenian Jakarta", "1_week", 60, 2},
	{"Umi Kalsum", "Bekasi", "gd", "applied", "jobfair", "Desainer percetakan (2 tahun), Photoshop dan Illustrator", "D3 Desain Grafis, Politeknik Negeri Media Kreatif", "immediate", 50, 4},
	{"Vina Melati", "Tangerang", "gd", "screening", "instagram", "Visual designer brand kopi (1 tahun), Canva dan Figma", "S1 Desain Komunikasi Visual, Universitas Multimedia Nusantara", "2_weeks", 58, 10},
	{"Wahyu Kurniawan", "Jakarta Barat", "gd", "psikotes", "portal", "Graphic designer agency (2 tahun), Photoshop, Illustrator, After Effects", "S1 Desain Komunikasi Visual, Universitas Trisakti", "1_month", 68, 14},
	{"Yoga Aditya", "Depok", "gd", "interview", "referral", "Senior graphic designer in-house startup (4 tahun), Figma dan motion CapCut", "S1 Desain Komunikasi Visual, Institut Teknologi Bandung", "1_month", 75, 17},
	{"Zahra Nabila", "Jakarta Selatan", "gd", "offer", "portal", "Graphic designer kampanye digital (3 tahun), Figma, Photoshop, Premiere", "S1 Desain Komunikasi Visual, Universitas Bina Nusantara", "1_month", 72, 24},
	{"Adi Gunawan", "Bogor", "gd", "talent_pool", "jobstreet", "Ilustrator freelance", "S1 Seni Rupa, Universitas Negeri Jakarta", "immediate", 55, 40},
	{"Bella Safitri", "Bekasi", "gd", "rejected", "walk_in", "Operator desain banner", "SMK Desain Grafis", "immediate", 45, 27},
	// Performance Marketing Specialist
	{"Cahyo Purnomo", "Jakarta Timur", "pm", "applied", "portal", "Digital ads specialist (2 tahun), Meta Ads dan Google Ads", "S1 Manajemen Pemasaran, Universitas Trisakti", "1_month", 95, 1},
	{"Desi Natalia", "Tangerang Selatan", "pm", "screening", "jobstreet", "Performance marketer e-commerce fashion (3 tahun), Meta Ads, TikTok Ads", "S1 Manajemen, Universitas Pelita Harapan", "1_month", 110, 7},
	{"Eko Prasetya", "Jakarta Selatan", "pm", "psikotes", "headhunter", "Paid media specialist agency (3 tahun), Google Ads dan Google Analytics", "S1 Teknik Industri, Universitas Indonesia", "1_month", 105, 12},
	{"Farhan Maulana", "Jakarta Barat", "pm", "interview", "referral", "Performance marketing lead fintech (4 tahun), Meta Ads, Google Ads, Excel", "S1 Ekonomi, Universitas Gadjah Mada", "1_month", 120, 16},
	{"Gilang Ramadhan", "Depok", "pm", "hired", "portal", "Performance marketing specialist (3 tahun), Meta Ads, Google Ads, TikTok Ads, Excel", "S1 Manajemen Pemasaran, Universitas Padjadjaran", "1_month", 100, 45},
	{"Hana Pertiwi", "Bekasi", "pm", "talent_pool", "jobstreet", "Digital marketing executive (2 tahun)", "S1 Manajemen, Universitas Esa Unggul", "2_weeks", 85, 38},
	{"Irfan Hakim", "Bogor", "pm", "archived", "other", "Marketing officer properti", "S1 Ekonomi, Universitas Pakuan", "immediate", 80, 60},
	// Account Executive
	{"Jessica Tanoto", "Jakarta Utara", "ae", "applied", "portal", "Sales executive SaaS (2 tahun), CRM dan presentasi", "S1 Manajemen Bisnis, Universitas Prasetiya Mulya", "2_weeks", 80, 3},
	{"Kevin Halim", "Jakarta Barat", "ae", "applied", "referral", "Business development media digital (1 tahun)", "S1 Ilmu Komunikasi, Universitas Pelita Harapan", "1_week", 70, 5},
	{"Larasati Dewi", "Jakarta Selatan", "ae", "screening", "jobstreet", "Account executive agency kreatif (2 tahun), prospecting dan negosiasi", "S1 Hubungan Internasional, Universitas Indonesia", "1_month", 85, 9},
	{"Mario Siregar", "Bekasi", "ae", "screening", "jobfair", "Sales B2B percetakan (3 tahun)", "S1 Manajemen, Universitas Krisnadwipa", "2_weeks", 75, 12},
	{"Nur Aini", "Depok", "ae", "psikotes", "portal", "Account manager agency digital (2 tahun), CRM", "S1 Manajemen Pemasaran, Universitas Gunadarma", "1_month", 78, 14},
	{"Oscar Pangaribuan", "Jakarta Timur", "ae", "offer", "headhunter", "Senior account executive digital agency (4 tahun), presentasi dan negosiasi", "S1 Manajemen, Universitas Katolik Atma Jaya", "1_month", 90, 23},
	{"Prita Anjani", "Tangerang", "ae", "rejected", "instagram", "Customer service e-commerce", "D3 Administrasi Bisnis", "immediate", 55, 25},
}

// tokenFor is a stable 64-hex portal token per seeded record.
func tokenFor(id string) string {
	sum := sha256.Sum256([]byte("white-recruitment-seed:" + id))
	return hex.EncodeToString(sum[:])
}

func candidateID(i int) string { return fmt.Sprintf("seed_cand_%02d", i+1) }

// pick is a stable pseudo-random number in [0, n) for a key.
func pick(key string, n int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(n))
}

func emailOf(name string) string {
	parts := strings.Fields(strings.ToLower(name))
	return strings.Join(parts, ".") + "@gmail.com"
}

func phoneOf(name string) string {
	return fmt.Sprintf("0812-%04d-%04d", 1000+pick(name+"a", 9000), 1000+pick(name+"b", 9000))
}

// pathTo is the stage history that leads to a status.
func pathTo(status string) []string {
	funnel := []string{"applied", "screening", "psikotes", "interview", "offer", "hired"}
	for i, s := range funnel {
		if s == status {
			return funnel[:i+1]
		}
	}
	switch status {
	case "talent_pool":
		return []string{"applied", "screening", "psikotes", "interview", "talent_pool"}
	case "rejected":
		return []string{"applied", "screening", "rejected"}
	case "archived":
		return []string{"applied", "screening", "psikotes", "interview", "talent_pool", "archived"}
	}
	return []string{status}
}

const hrName = "Tim HR WHITE"

func (s *seeder) candidates(ctx context.Context) error {
	for i, p := range people {
		id := candidateID(i)
		existing, err := s.store.GetCandidate(ctx, id)
		if err != nil {
			return err
		}
		created := s.now.AddDate(0, 0, -p.daysAgo).Add(-time.Duration(pick(p.name, 8)) * time.Hour)
		if existing == nil {
			if err := s.insertCandidate(ctx, id, p, created); err != nil {
				return fmt.Errorf("candidate %s: %w", p.name, err)
			}
		}
		var hiredAt *time.Time
		if p.status == "hired" {
			hiredAt = ptr(created.Add(time.Duration(p.daysAgo-5) * 24 * time.Hour))
		}
		// Re-applied on every run so the demo dates stay consistent.
		if err := s.exec(ctx, `UPDATE recruitment.candidates SET created_at = $2, updated_at = $3, hired_at = $4::timestamptz,
			hired_event_at = $4::timestamptz WHERE id = $1`, id, created, created.Add(time.Duration(p.daysAgo)*12*time.Hour), hiredAt); err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) insertCandidate(ctx context.Context, id string, p person, created time.Time) error {
	salary := p.salary * 100_000
	c := domain.Candidate{ID: id, FullName: p.name, Email: emailOf(p.name), Phone: phoneOf(p.name), Domicile: p.domicile, Source: p.source,
		JobOpeningID: ptr(openingID(p.opening)), Status: p.status, LastExperience: p.experience, LastEducation: p.education,
		Availability: p.availability, ExpectedSalary: &salary, CreatedBy: by()}
	path := pathTo(p.status)
	return s.store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertCandidate(ctx, c); err != nil {
			return err
		}
		first := domain.Activity{ID: id + "_act_00", CandidateID: id, Type: "applied", CreatedAt: created,
			Description: "Lamaran masuk melalui portal karir untuk " + openingTitle(p.opening)}
		if p.source != "portal" {
			first.Type, first.Description, first.CreatedBy, first.CreatedByName = "created", "Kandidat ditambahkan manual (sumber: "+domain.SourceLabel(p.source)+")", by(), hrName
		}
		if err := tx.InsertActivity(ctx, first); err != nil {
			return err
		}
		step := time.Duration(p.daysAgo) * 24 * time.Hour / time.Duration(len(path)+1)
		for k := 1; k < len(path); k++ {
			if err := tx.InsertActivity(ctx, domain.Activity{ID: fmt.Sprintf("%s_act_%02d", id, k), CandidateID: id, Type: "status_change",
				Description: domain.StageChangeDescription(path[k-1], path[k]), CreatedBy: by(), CreatedByName: hrName,
				CreatedAt: created.Add(step * time.Duration(k))}); err != nil {
				return err
			}
		}
		if len(path) > 1 {
			rec := "lolos"
			if p.status == "rejected" {
				rec = "tidak_lolos"
			}
			if err := tx.UpsertScreening(ctx, domain.Screening{CandidateID: id, Contacted: true, Interested: ptr(true),
				AvailabilityNote: availabilityNote(p.availability), ConfirmedSalary: &salary, WillingShift: ptr(p.opening != "gd"),
				WillingPlacement: ptr(true), Notes: "Dihubungi via WhatsApp, kandidat responsif.", Recommendation: rec,
				UpdatedBy: by(), UpdatedByName: hrName}); err != nil {
				return err
			}
		}
		return nil
	})
}

func openingTitle(key string) string {
	for _, o := range openings {
		if o.key == key {
			return o.title
		}
	}
	return key
}

func availabilityNote(a string) string {
	return map[string]string{"immediate": "Bisa mulai segera", "1_week": "Bisa mulai 1 minggu lagi", "2_weeks": "Perlu 2 minggu untuk resign",
		"1_month": "One month notice di kantor sekarang"}[a]
}

func indexOf(name string) int {
	for i, p := range people {
		if p.name == name {
			return i
		}
	}
	panic("seed: unknown candidate " + name)
}

func cid(name string) string { return candidateID(indexOf(name)) }

// ─── psikotes ────────────────────────────────────────────────

// psikotes gives Nanda a finished session (Aritmatika 8/10 and PAPI) with
// a "lolos" verdict, and Maya a fresh invitation to open in the portal.
func (s *seeder) psikotes(ctx context.Context) error {
	done, err := s.store.ListPsikotesSessions(ctx, cid("Nanda Prasetyo"))
	if err != nil {
		return err
	}
	if len(done) == 0 {
		if err := s.scoredSession(ctx); err != nil {
			return err
		}
	}
	pending, err := s.store.ListPsikotesSessions(ctx, cid("Melati Anggraini"))
	if err != nil || len(pending) > 0 {
		return err
	}
	invited := s.now.AddDate(0, 0, -1)
	return s.store.InsertPsikotesSession(ctx, domain.PsikotesSession{ID: "seed_psi_maya", CandidateID: cid("Melati Anggraini"),
		Token: tokenFor("seed_psi_maya"), Status: "sent", InvitedAt: &invited, ExpiresAt: ptr(invited.AddDate(0, 0, 3)), CreatedBy: by(),
		CreatedByName: hrName, Tests: []domain.SessionTest{{ID: "seed_pst_maya_ar", InstrumentID: "psi_aritmatika", SortOrder: 0},
			{ID: "seed_pst_maya_papi", InstrumentID: "psi_papi_kostick", SortOrder: 1}}})
}

func (s *seeder) scoredSession(ctx context.Context) error {
	candidate := cid("Nanda Prasetyo")
	invited := s.now.AddDate(0, 0, -6)
	session := domain.PsikotesSession{ID: "seed_psi_nanda", CandidateID: candidate, Token: tokenFor("seed_psi_nanda"), Status: "sent",
		InvitedAt: &invited, ExpiresAt: ptr(invited.AddDate(0, 0, 3)), CreatedBy: by(), CreatedByName: hrName,
		Tests: []domain.SessionTest{{ID: "seed_pst_nanda_ar", InstrumentID: "psi_aritmatika", SortOrder: 0},
			{ID: "seed_pst_nanda_papi", InstrumentID: "psi_papi_kostick", SortOrder: 1}}}
	if err := s.store.InsertPsikotesSession(ctx, session); err != nil {
		return err
	}
	if _, err := s.store.StartPsikotesSession(ctx, session.ID, true); err != nil {
		return err
	}
	// Aritmatika: questions 3 and 7 wrong.
	var arIDs []string
	answers := map[string]string{}
	var mcq []domain.McqQuestion
	for i, q := range arithmetic {
		id := fmt.Sprintf("seed_q_ar_%02d", i+1)
		arIDs = append(arIDs, id)
		given := q.correct
		if i == 2 || i == 6 {
			given = "a"
		}
		answers[id] = given
		mcq = append(mcq, domain.McqQuestion{ID: id, Correct: ptr(q.correct)})
	}
	n, detail := domain.ScoreMcq(mcq, answers)
	score := float64(n)
	if err := s.finishTest(ctx, "seed_pst_nanda_ar", arIDs, answers, &score, domain.McqScoreDetail(detail)); err != nil {
		return err
	}
	// PAPI: a, b alternating with a bias to achievement and detail.
	var papiIDs []string
	papiAnswers := map[string]string{}
	var pq []domain.PapiQuestion
	for i, p := range papiPairs {
		id := fmt.Sprintf("seed_q_papi_%02d", i+1)
		papiIDs = append(papiIDs, id)
		choice := "a"
		if i%3 == 2 {
			choice = "b"
		}
		papiAnswers[id] = choice
		pq = append(pq, domain.PapiQuestion{ID: id, ScaleA: p.sa, ScaleB: p.sb})
	}
	if err := s.finishTest(ctx, "seed_pst_nanda_papi", papiIDs, papiAnswers, nil, domain.PapiScoreDetail(domain.ScorePapi(pq, papiAnswers))); err != nil {
		return err
	}
	if _, err := s.store.CompletePsikotesSession(ctx, session.ID); err != nil {
		return err
	}
	if err := s.store.UpsertPsikotesSummary(ctx, domain.PsikotesSummary{CandidateID: candidate, Recommendation: "lolos",
		Notes:     "Aritmatika 80/100, profil PAPI menonjol pada dorongan berprestasi dan ketelitian. Lanjut interview.",
		UpdatedBy: by(), UpdatedByName: hrName}); err != nil {
		return err
	}
	return s.exec(ctx, `UPDATE recruitment.psikotes_sessions SET started_at = $2::timestamptz, completed_at = $2::timestamptz + interval '55 minutes' WHERE id = $1`,
		session.ID, invited.Add(20*time.Hour))
}

func (s *seeder) finishTest(ctx context.Context, id string, questionIDs []string, answers map[string]string, score *float64, detail domain.ScoreDetail) error {
	if _, err := s.store.BeginTest(ctx, id, questionIDs); err != nil {
		return err
	}
	_, err := s.store.ScoreTest(ctx, id, answers, score, detail)
	return err
}

// ─── AI interviews ───────────────────────────────────────────

var transcripts = map[string][]struct{ topic, question, answer string }{
	"Ayu Kusumawati": {
		{"pembuka", "Halo Ayu Kusumawati! Terima kasih sudah meluangkan waktu untuk interview posisi Social Media Specialist. " + domain.FallbackQuestions["pembuka"],
			"Saya Ayu, dua tahun terakhir menjadi copywriter dan content planner untuk tiga brand F&B. Saya memegang kalender konten Instagram dan TikTok dari ide sampai laporan."},
		{"keahlian", domain.FallbackQuestions["keahlian"],
			"Paling kuat di copywriting dan content planning. Saya pakai Canva untuk mockup, CapCut untuk video pendek, dan Spreadsheet untuk laporan insight mingguan."},
		{"motivasi", domain.FallbackQuestions["motivasi"],
			"Saya ingin pegang lebih banyak klien dengan industri berbeda. Agency seperti WHITE memberi ruang belajar strategi, bukan hanya eksekusi."},
		{"ketersediaan", domain.FallbackQuestions["ketersediaan"], "Saya perlu dua minggu untuk serah terima. Lokasi Kemang terjangkau dari rumah."},
		{"gaji", domain.FallbackQuestions["gaji"], "Harapan saya di kisaran Rp 6,5 juta sampai 7 juta per bulan."},
	},
	"Nanda Prasetyo": {
		{"pembuka", "Halo Nanda Prasetyo! Terima kasih sudah meluangkan waktu untuk interview posisi SEO Analyst. " + domain.FallbackQuestions["pembuka"],
			"Saya Nanda, tiga tahun di agency sebagai SEO specialist. Saya menangani audit teknis, riset kata kunci, dan laporan bulanan untuk klien e-commerce."},
		{"keahlian", domain.FallbackQuestions["keahlian"],
			"Technical SEO dan riset kata kunci. Sehari-hari pakai Ahrefs, Semrush, Search Console, dan Google Analytics, lalu merangkum di Spreadsheet."},
		{"motivasi", domain.FallbackQuestions["motivasi"], "Ingin pindah ke tim yang lebih fokus pada data dan eksperimen konten."},
		{"ketersediaan", domain.FallbackQuestions["ketersediaan"], "Satu bulan notice."},
		{"gaji", domain.FallbackQuestions["gaji"], "Sekitar 8,5 juta."},
	},
}

func (s *seeder) aiInterviews(ctx context.Context) error {
	for name, turns := range transcripts {
		candidate := cid(name)
		existing, err := s.store.ListAiSessions(ctx, candidate)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			continue
		}
		invited := s.now.AddDate(0, 0, -4)
		session := domain.AiSession{ID: "seed_ai_" + candidate, CandidateID: candidate, Token: tokenFor("seed_ai_" + candidate),
			Status: "sent", MaxQuestions: 5, InvitedAt: &invited, ExpiresAt: ptr(invited.AddDate(0, 0, 7)), CreatedBy: by(), CreatedByName: hrName}
		if err := s.store.InsertAiSession(ctx, session); err != nil {
			return err
		}
		if _, err := s.store.StartAiSession(ctx, session.ID); err != nil {
			return err
		}
		var list []domain.Turn
		asked := invited.Add(26 * time.Hour)
		for i, t := range turns {
			answered := asked.Add(90 * time.Second)
			turn := domain.Turn{ID: fmt.Sprintf("%s_t%d", session.ID, i+1), SessionID: session.ID, TurnNo: i + 1, Topic: ptr(t.topic),
				Question: t.question, Answer: ptr(t.answer), AnswerMode: ptr("text"), AskedAt: asked, AnsweredAt: &answered}
			if _, err := s.store.InsertTurn(ctx, turn); err != nil {
				return err
			}
			list = append(list, turn)
			asked = answered.Add(10 * time.Second)
		}
		summary := openrouter.MockSummary(domain.Transcript{CandidateName: name, PositionTitle: openingTitle(people[indexOf(name)].opening),
			MaxQuestions: 5, Turns: list})
		if err := s.store.CompleteAiSession(ctx, session.ID, &summary, ptr(openrouter.MockModel)); err != nil {
			return err
		}
		if err := s.store.InsertActivity(ctx, domain.Activity{ID: session.ID + "_done", CandidateID: candidate, Type: "interview_ai_completed",
			Description: domain.AiCompletedDescription(len(list), true), CreatedAt: asked}); err != nil {
			return err
		}
		if err := s.exec(ctx, `UPDATE recruitment.interview_ai_sessions SET started_at = $2, completed_at = $3 WHERE id = $1`,
			session.ID, invited.Add(26*time.Hour), asked); err != nil {
			return err
		}
	}
	return nil
}

// ─── scheduled interviews ────────────────────────────────────

func (s *seeder) interviewer(opening, typ string) (*string, *string, string) {
	if typ == "hrd" {
		if s.hr != nil {
			return &s.hr.ID, s.hr.UserID, s.hr.Name
		}
		return nil, nil, hrName
	}
	if e, ok := s.staff[opening]; ok {
		return &e.ID, e.UserID, e.Name
	}
	return nil, nil, map[string]string{"smm": "Rani (Social Media Lead)", "seo": "Sari (SEO Lead)", "gd": "Bima (Creative Lead)",
		"pm": "Andre (Ads Lead)", "ae": "Kartika (Sales Manager)"}[opening]
}

type plannedInterview struct {
	name, typ, mode, status, rec string
	inDays, hour                 int
	scores                       []int
}

var plannedInterviews = []plannedInterview{
	{name: "Yoga Aditya", typ: "hiring_manager", mode: "offline", status: "scheduled", inDays: 1, hour: 10},
	{name: "Okta Ramadhan", typ: "hrd", mode: "online", status: "scheduled", inDays: 2, hour: 14},
	{name: "Ayu Kusumawati", typ: "hiring_manager", mode: "online", status: "scheduled", inDays: 3, hour: 11},
	{name: "Farhan Maulana", typ: "hrd", mode: "offline", status: "completed", rec: "proceed", inDays: -6, hour: 10, scores: []int{4, 4, 5, 4, 4}},
	{name: "Farhan Maulana", typ: "hiring_manager", mode: "offline", status: "completed", rec: "proceed", inDays: -3, hour: 15, scores: []int{4, 5, 5, 3, 4}},
	{name: "Hendra Wijaya", typ: "hiring_manager", mode: "online", status: "completed", rec: "proceed", inDays: -8, hour: 13, scores: []int{4, 4, 4, 4, 5}},
	{name: "Intan Maharani", typ: "hrd", mode: "online", status: "completed", rec: "pool", inDays: -20, hour: 9, scores: []int{3, 3, 2, 4, 3}},
	{name: "Fajar Setiawan", typ: "hrd", mode: "offline", status: "cancelled", inDays: -1, hour: 10},
}

func (s *seeder) interviews(ctx context.Context) error {
	for i, pl := range plannedInterviews {
		id := fmt.Sprintf("seed_iv_%02d", i+1)
		existing, err := s.store.GetInterview(ctx, id)
		if err != nil {
			return err
		}
		if existing != nil {
			continue
		}
		p := people[indexOf(pl.name)]
		day, _ := time.ParseInLocation(domain.IsoDate, domain.DayKey(s.now.AddDate(0, 0, pl.inDays)), domain.Jakarta)
		iv := domain.Interview{ID: id, CandidateID: cid(pl.name), InterviewDate: day.Add(time.Duration(pl.hour) * time.Hour), DurationMinutes: 60,
			Type: pl.typ, Mode: pl.mode, Status: pl.status, Recommendation: pl.rec, CreatedBy: by()}
		iv.InterviewerEmployeeID, iv.InterviewerUserID, iv.InterviewerName = s.interviewer(p.opening, pl.typ)
		if pl.mode == "online" {
			iv.MeetingLink = "https://meet.google.com/whi-" + fmt.Sprintf("%03d", pick(id, 1000)) + "-rec"
		} else {
			iv.Location = "Kantor WHITE, Kemang, Jakarta Selatan"
		}
		if pl.status == "cancelled" {
			iv.CancelReason = "Kandidat meminta jadwal ulang karena sakit"
		}
		if len(pl.scores) > 0 {
			items := make([]domain.ScoreItem, len(pl.scores))
			for k, sc := range pl.scores {
				items[k] = domain.ScoreItem{Criterion: domain.DefaultScorecardCriteria[k], Score: sc}
			}
			items[0].Note = "Jelas dan terstruktur saat menjelaskan studi kasus"
			clean, overall, err := domain.ValidateScorecard(items, pl.rec, "")
			if err != nil {
				return err
			}
			iv.Scorecard, iv.OverallScore = clean, &overall
			iv.Notes = "Contoh kampanye relevan dengan klien kami."
		}
		if err := s.store.InsertInterview(ctx, iv); err != nil {
			return err
		}
	}
	return nil
}

// ─── offers ──────────────────────────────────────────────────

type plannedOffer struct {
	name, status, note string
	salary             int64
	sentDaysAgo        int
	startInDays        int
}

var plannedOffers = []plannedOffer{
	{name: "Gilang Ramadhan", status: "accepted", note: "Siap bergabung, terima kasih.", salary: 10_500_000, sentDaysAgo: 9, startInDays: 7},
	{name: "Hendra Wijaya", status: "sent", salary: 7_000_000, sentDaysAgo: 6, startInDays: 21},
	{name: "Zahra Nabila", status: "negotiating", note: "Apakah gaji pokok bisa Rp 7.500.000? Saya juga butuh laptop kerja.", salary: 6_800_000, sentDaysAgo: 3, startInDays: 30},
	{name: "Oscar Pangaribuan", status: "sent", salary: 8_500_000, sentDaysAgo: 1, startInDays: 30},
}

var offerBenefits = []string{"BPJS Kesehatan dan Ketenagakerjaan", "Laptop kerja", "Cuti tahunan 12 hari", "Budget pelatihan Rp 3.000.000 per tahun"}

func (s *seeder) offers(ctx context.Context) error {
	for _, pl := range plannedOffers {
		candidate := cid(pl.name)
		existing, err := s.store.ListOffers(ctx, candidate)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			continue
		}
		sent := s.now.AddDate(0, 0, -pl.sentDaysAgo)
		expires := sent.AddDate(0, 0, 7)
		o := domain.Offer{ID: "seed_offer_" + candidate, CandidateID: candidate, Version: 1, Token: tokenFor("seed_offer_" + candidate),
			Status: pl.status, PositionTitle: openingTitle(people[indexOf(pl.name)].opening), BaseSalary: domain.Rupiah(pl.salary),
			Benefits: offerBenefits, StartDate: ptr(domain.DayKey(s.now.AddDate(0, 0, pl.startInDays))), SentAt: &sent, ExpiresAt: &expires,
			CreatedBy: by(), CreatedByName: hrName}
		if pl.status != "sent" {
			o.ResponseNote, o.RespondedAt, o.ResponseSource = pl.note, ptr(sent.Add(30*time.Hour)), "portal"
		}
		err = s.store.WithTx(ctx, func(tx domain.Store) error {
			if err := tx.InsertOffer(ctx, o); err != nil {
				return err
			}
			if err := tx.InsertActivity(ctx, domain.Activity{ID: o.ID + "_sent", CandidateID: candidate, Type: "offer_sent",
				Description: domain.OfferCreatedDescription(1, 7), CreatedBy: by(), CreatedByName: hrName, CreatedAt: sent}); err != nil {
				return err
			}
			if o.RespondedAt == nil {
				return nil
			}
			action := map[string]string{"accepted": "accept", "negotiating": "negotiate"}[pl.status]
			return tx.InsertActivity(ctx, domain.Activity{ID: o.ID + "_resp", CandidateID: candidate, Type: "offer_response",
				Description: domain.OfferResponseDescription(action, 1, &o.ResponseNote), CreatedAt: *o.RespondedAt})
		})
		if err != nil {
			return fmt.Errorf("offer %s: %w", pl.name, err)
		}
	}
	return nil
}
