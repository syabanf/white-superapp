// Package seed inserts idempotent demo recruitment data for WHITE, an
// Indonesian digital agency: five live job openings with about forty
// candidates across the pipeline and talent pool, two years of past
// openings and applicants (history.go), a psikotes question bank with one
// scored session, AI interview transcripts (mock summaries), scheduled and
// scored interviews, and offers including one accepted hire. Departments
// and interviewers come from hr over gRPC when it is reachable.
package seed

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/app"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/logging"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

const seedUser = "seed-recruitment"

// Run connects, migrates and seeds.
func Run(ctx context.Context) error {
	_ = config.LoadDotEnv(".env")
	logger := logging.New("recruitment-seed")
	pool, err := pg.Connect(ctx, config.Env("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pg.Migrate(ctx, pool, recruitment.Migrations(), "migrations", recruitment.Schema, logger); err != nil {
		return err
	}
	w, err := recruitment.Wire(pkgapp.Deps{Logger: logger, Pool: pool})
	if err != nil {
		return err
	}
	files := storage.NewLocal(config.Env("RECRUITMENT_UPLOAD_DIR", ".uploads/recruitment"))
	if err := Seed(ctx, pool, w.HR, files, time.Now().UTC(), logger); err != nil {
		return err
	}
	for portal, path := range PortalURLs() {
		logger.Info("demo portal link", "portal", portal, "path", path)
	}
	return nil
}

type seeder struct {
	pool  *pgxpool.Pool
	store domain.Store
	files app.FileStore // CVs and drawings; nil skips them
	log   *slog.Logger
	now   time.Time
	depts map[string]domain.Department // by opening key
	staff map[string]domain.Employee   // interviewer by opening key
	hr    *domain.Employee             // HR interviewer
}

// Seed inserts the demo data; running it again changes nothing except
// refreshing the expiry of the demo portal links.
func Seed(ctx context.Context, pool *pgxpool.Pool, dir app.Directory, files app.FileStore, now time.Time, logger *slog.Logger) error {
	s := &seeder{pool: pool, store: postgres.NewStore(pool), files: files, log: logger, now: now,
		depts: map[string]domain.Department{}, staff: map[string]domain.Employee{}}
	s.loadDirectory(ctx, dir)
	steps := []func(context.Context) error{s.openings, s.questionBank, s.candidates, s.psikotes, s.aiInterviews, s.interviews, s.offers, s.history, s.showcase}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return err
		}
	}
	logger.Info("recruitment seed done", "openings", len(openings), "candidates", len(people), "pastOpenings", len(pastOpenings))
	return nil
}

// loadDirectory maps openings to hr departments and picks interviewers;
// without hr the openings keep their department names only.
func (s *seeder) loadDirectory(ctx context.Context, dir app.Directory) {
	if dir == nil {
		return
	}
	hrCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	depts, err := dir.Departments(hrCtx)
	if err != nil {
		s.log.Warn("hr unavailable, seeding without departments", "err", err)
		return
	}
	employees, err := dir.Employees(hrCtx)
	if err != nil {
		s.log.Warn("hr employees unavailable", "err", err)
	}
	for _, o := range openings {
		for _, d := range depts {
			if strings.EqualFold(d.Name, o.department) || strings.Contains(strings.ToLower(d.Name), strings.ToLower(o.deptHint)) {
				s.depts[o.key] = d
				break
			}
		}
	}
	for _, e := range employees {
		if !e.Active {
			continue
		}
		dept := strings.ToLower(e.DepartmentName)
		if s.hr == nil && (dept == "hr" || strings.Contains(dept, "human") || strings.Contains(dept, "sdm")) {
			s.hr = &e
		}
		for key, d := range s.depts {
			if cur, ok := s.staff[key]; (!ok || (cur.UserID == nil && e.UserID != nil)) && e.DepartmentID != nil && *e.DepartmentID == d.ID {
				s.staff[key] = e
			}
		}
	}
}

func (s *seeder) exec(ctx context.Context, sql string, args ...any) error {
	_, err := s.pool.Exec(ctx, sql, args...)
	return err
}

func ptr[T any](v T) *T { return &v }

func by() *string { return ptr(seedUser) }

// ─── openings ────────────────────────────────────────────────

type opening struct {
	key, title, department, deptHint string
	workMode, employment             string
	headcount                        int
	salaryMin, salaryMax             int64
	description, requirements        string
	benefits                         string
	status                           string
	closingInDays                    int
}

var openings = []opening{
	{key: "smm", title: "Social Media Specialist", department: "Social Media", deptHint: "social", workMode: "Hybrid", employment: "Full-time",
		headcount: 2, salaryMin: 5_500_000, salaryMax: 7_500_000, status: "published", closingInDays: 30,
		description:  "Mengelola akun media sosial klien (Instagram, TikTok, LinkedIn): kalender konten, copywriting, publikasi, dan laporan performa bulanan.",
		requirements: "Pengalaman minimal 1 tahun mengelola akun brand. Menguasai Instagram, TikTok, Canva, dan copywriting. Terbiasa membaca insight dan membuat laporan.",
		benefits:     "BPJS Kesehatan dan Ketenagakerjaan, laptop kerja, WFH 2 hari per minggu, budget pelatihan."},
	{key: "seo", title: "SEO Analyst", department: "SEO", deptHint: "seo", workMode: "Hybrid", employment: "Full-time",
		headcount: 1, salaryMin: 6_000_000, salaryMax: 9_000_000, status: "published", closingInDays: 21,
		description:  "Melakukan riset kata kunci, audit teknis, optimasi on-page, dan laporan peringkat untuk klien e-commerce dan B2B.",
		requirements: "Menguasai Google Analytics, Search Console, Ahrefs atau Semrush. Paham technical SEO dasar dan WordPress. Mampu menyusun laporan di Spreadsheet.",
		benefits:     "BPJS, laptop kerja, langganan tools SEO, bonus kinerja kuartalan."},
	{key: "gd", title: "Graphic Designer", department: "Creative", deptHint: "creative", workMode: "On-site", employment: "Full-time",
		headcount: 1, salaryMin: 5_000_000, salaryMax: 7_500_000, status: "published", closingInDays: 14,
		description:  "Membuat aset visual kampanye: feed dan story media sosial, banner iklan, presentasi pitching, dan materi cetak.",
		requirements: "Portofolio wajib. Menguasai Figma, Photoshop, Illustrator. Nilai plus bila bisa motion sederhana dengan After Effects atau CapCut.",
		benefits:     "BPJS, perangkat desain, makan siang Jumat, ruang kreatif di kantor Kemang."},
	{key: "pm", title: "Performance Marketing Specialist", department: "Ads", deptHint: "ads", workMode: "Hybrid", employment: "Full-time",
		headcount: 1, salaryMin: 8_000_000, salaryMax: 12_000_000, status: "published", closingInDays: 30,
		description:  "Merencanakan dan mengoptimasi kampanye Meta Ads, Google Ads, dan TikTok Ads dengan target ROAS dan CPA klien.",
		requirements: "Pengalaman minimal 2 tahun mengelola budget iklan. Menguasai Meta Ads, Google Ads, Google Analytics, dan Excel. Terbiasa A/B testing.",
		benefits:     "BPJS, bonus berbasis performa klien, laptop kerja, sertifikasi Google dan Meta dibiayai."},
	{key: "ae", title: "Account Executive", department: "Sales", deptHint: "sales", workMode: "On-site", employment: "Full-time",
		headcount: 2, salaryMin: 6_000_000, salaryMax: 9_000_000, status: "published", closingInDays: 45,
		description:  "Mencari dan menutup klien baru untuk layanan digital marketing, menyiapkan proposal, dan menjaga hubungan dengan klien yang ada.",
		requirements: "Pengalaman sales B2B atau agency minimal 1 tahun. Kuat dalam prospecting, presentasi, dan negosiasi. Terbiasa memakai CRM.",
		benefits:     "BPJS, komisi penjualan, tunjangan transportasi, laptop kerja."},
}

func openingID(key string) string { return "seed_job_" + key }

func (s *seeder) openings(ctx context.Context) error {
	for _, o := range openings {
		id := openingID(o.key)
		existing, err := s.store.GetJobOpening(ctx, id)
		if err != nil {
			return err
		}
		// published before the oldest live applicant (60 days ago)
		published := s.now.AddDate(0, 0, -70)
		if existing != nil {
			// databases seeded before the dashboard history had them published 20 days ago
			if err := s.exec(ctx, `UPDATE recruitment.job_openings SET published_at = $2 WHERE id = $1 AND published_at > $2`, id, published); err != nil {
				return err
			}
			continue
		}
		min, max := domain.Rupiah(o.salaryMin), domain.Rupiah(o.salaryMax)
		closing := domain.DayKey(s.now.AddDate(0, 0, o.closingInDays))
		j := domain.JobOpening{ID: id, Title: o.title, Slug: domain.Slugify(o.title), Department: o.department, Location: domain.DefaultLocation,
			EmploymentType: o.employment, WorkMode: o.workMode, Headcount: o.headcount, Description: o.description,
			Requirements: o.requirements, Benefits: o.benefits, SalaryMin: &min, SalaryMax: &max, Status: o.status,
			PublishedAt: &published, ClosingDate: &closing, CreatedBy: by()}
		if d, ok := s.depts[o.key]; ok {
			j.DepartmentID, j.Department = &d.ID, d.Name
		}
		if e, ok := s.staff[o.key]; ok {
			j.HiringManagerUserID = e.UserID
		}
		if err := s.store.InsertJobOpening(ctx, j); err != nil {
			return fmt.Errorf("opening %s: %w", o.key, err)
		}
	}
	return nil
}

// ─── psikotes bank ───────────────────────────────────────────

var arithmetic = []struct {
	body    string
	options [4]string
	correct string
}{
	{"125 + 378 = ...", [4]string{"493", "503", "513", "403"}, "b"},
	{"15% dari 240 adalah ...", [4]string{"32", "34", "36", "38"}, "c"},
	{"Budget iklan Rp 4.500.000 dibagi rata ke 3 kampanye. Tiap kampanye mendapat ...", [4]string{"Rp 1.400.000", "Rp 1.500.000", "Rp 1.550.000", "Rp 1.600.000"}, "b"},
	{"48 x 12 = ...", [4]string{"556", "566", "576", "586"}, "c"},
	{"Jika CTR 2% dari 15.000 impresi, jumlah klik adalah ...", [4]string{"150", "200", "250", "300"}, "d"},
	{"1.000 - 487 = ...", [4]string{"513", "523", "503", "613"}, "a"},
	{"Deret 3, 6, 12, 24, ... angka berikutnya adalah", [4]string{"36", "42", "48", "52"}, "c"},
	{"Rata-rata dari 70, 80, dan 90 adalah ...", [4]string{"75", "80", "85", "90"}, "b"},
	{"3/4 dari 360 adalah ...", [4]string{"240", "260", "270", "280"}, "c"},
	{"Follower naik dari 8.000 menjadi 10.000. Kenaikannya ...", [4]string{"20%", "25%", "30%", "35%"}, "b"},
}

var papiPairs = []struct{ a, sa, b, sb string }{
	{"Saya bekerja keras untuk menyelesaikan setiap tugas", "G", "Saya suka memimpin kelompok", "L"},
	{"Saya ingin menjadi yang terbaik di bidang saya", "A", "Saya senang berada di tengah banyak orang", "S"},
	{"Saya menyelesaikan tugas sampai tuntas", "N", "Saya suka mengambil keputusan dengan cepat", "I"},
	{"Saya teliti terhadap detail pekerjaan", "D", "Saya suka mencoba cara kerja baru", "Z"},
	{"Saya suka mengatur pekerjaan orang lain", "P", "Saya mengikuti arahan atasan", "F"},
	{"Saya bekerja dengan tempo cepat", "T", "Saya suka mendapat perhatian", "X"},
	{"Saya menjaga barang dan file tetap rapi", "C", "Saya senang memikirkan teori", "R"},
	{"Saya tetap tenang di bawah tekanan", "E", "Saya suka aktivitas fisik", "V"},
	{"Saya ingin diterima oleh tim", "B", "Saya mempertahankan pendapat saya", "K"},
	{"Saya dekat dengan rekan kerja", "O", "Saya suka aturan yang jelas", "W"},
}

func (s *seeder) questionBank(ctx context.Context) error {
	keys := []string{"a", "b", "c", "d"}
	for i, q := range arithmetic {
		opts := make([]domain.McqOption, 4)
		for k := range opts {
			opts[k] = domain.McqOption{Key: keys[k], Text: q.options[k]}
		}
		if err := s.store.InsertQuestion(ctx, domain.Question{ID: fmt.Sprintf("seed_q_ar_%02d", i+1), InstrumentID: "psi_aritmatika",
			Body: q.body, Options: opts, CorrectKey: q.correct, SortOrder: i + 1, IsActive: true}, seedUser); err != nil {
			return err
		}
	}
	for i, p := range papiPairs {
		if err := s.store.InsertQuestion(ctx, domain.Question{ID: fmt.Sprintf("seed_q_papi_%02d", i+1), InstrumentID: "psi_papi_kostick",
			Pair:      &domain.PapiPair{A: domain.PapiStatement{Text: p.a, Scale: p.sa}, B: domain.PapiStatement{Text: p.b, Scale: p.sb}},
			SortOrder: i + 1, IsActive: true}, seedUser); err != nil {
			return err
		}
	}
	return nil
}
