// Package seed inserts idempotent demo data for WHITE, an Indonesian
// digital agency: department KPI indicators and weights, a year of
// department tasks, logbook entries and daily recaps for the hr employees,
// thirteen months of KPI scorecards, four review cycles, two 360 cycles,
// monthly recaps and a year of improvement items. Employees
// and departments are read from hr over gRPC; without hr only the
// indicator catalog is seeded.
package seed

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/performance"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/logging"
	"github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

const seedUser = "seed-performance"

// Run connects, migrates and seeds.
func Run(ctx context.Context) error {
	_ = config.LoadDotEnv(".env")
	logger := logging.New("performance-seed")
	pool, err := postgres.Connect(ctx, config.Env("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool, performance.Migrations(), "migrations", performance.Schema, logger); err != nil {
		return err
	}
	w, err := performance.Wire(pkgapp.Deps{Logger: logger, Pool: pool})
	if err != nil {
		return err
	}
	s := &seeder{pool: pool, log: logger, now: time.Now().UTC(), w: w}
	return s.run(ctx)
}

type seeder struct {
	pool  *pgxpool.Pool
	log   *slog.Logger
	now   time.Time
	w     *performance.Wiring
	depts []domain.Department
	staff []domain.Employee
	// ids maps indicator codes to ids (an admin may have created one first).
	ids map[string]string
}

// pick is a stable pseudo-random number in [0, n) for a key.
func pick(key string, n int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(n))
}

func (s *seeder) exec(ctx context.Context, sql string, args ...any) error {
	_, err := s.pool.Exec(ctx, sql, args...)
	return err
}

func (s *seeder) run(ctx context.Context) error {
	if err := s.indicators(ctx); err != nil {
		return err
	}
	hrCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	depts, err := s.w.HR.Departments(hrCtx)
	if err == nil {
		s.staff, err = s.w.HR.Employees(hrCtx, false)
	}
	if err != nil {
		fmt.Fprintf(os.Stdout, "hr tidak bisa dihubungi (%v): hanya katalog indikator yang di-seed; tugas, logbook, KPI, review dan perbaikan dilewati. Jalankan hr lalu ulangi `seed`.\n", err)
		return nil
	}
	s.depts = depts
	if len(s.staff) == 0 {
		fmt.Fprintln(os.Stdout, "hr belum punya karyawan aktif: seed hr dulu, lalu ulangi `seed`.")
		return nil
	}
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"bobot KPI departemen", s.weights},
		{"tugas departemen", s.tasks},
		{"logbook", s.logbook},
		{"rekap harian", s.dailyRecaps},
		{"scorecard KPI", s.kpi},
		{"siklus review", s.review},
		{"feedback 360", s.feedback},
		{"item perbaikan", s.improvements},
		{"riwayat perbaikan", s.improvementHistory},
		{"rekap bulanan", s.recaps},
		{"target KPI", s.targets},
		{"komentar dan lampiran tugas", s.taskExtras},
		{"status feedback 360", s.feedbackStates},
		{"retro", s.retros},
		{"rekap terbaru", s.recentRecaps},
		{"umur perbaikan otomatis", s.ageAutoItems},
	}
	for _, step := range steps {
		if err := step.fn(ctx); err != nil {
			return fmt.Errorf("seed %s: %w", step.name, err)
		}
		s.log.Info("seeded", "step", step.name)
	}
	fmt.Fprintf(os.Stdout, "seed performance selesai: %d departemen, %d karyawan\n", len(s.depts), len(s.staff))
	return nil
}

// admin is a context acting as an ADMIN for the use cases.
func admin(ctx context.Context) context.Context {
	return authn.WithPrincipal(ctx, authn.Principal{UserID: seedUser, Role: authn.RoleAdmin, Name: "Seeder"})
}

// ─── catalog ─────────────────────────────────────────────────

type indicatorSeed struct {
	code, name, unit, direction string
	target                      float64
	// sample monthly actual range for the demo (min, spread)
	base, spread float64
}

// profile is what a WHITE department gets: its indicators, tasks and
// logbook checklist.
type profile struct {
	key        string
	match      []string
	indicators []indicatorSeed
	tasks      []taskSeed
	checklist  []string
}

type taskSeed struct {
	title, recurrence, priority, kpi string
	day                              int
	subtasks                         []string
}

var profiles = []profile{
	{key: "social", match: []string{"social", "sosmed", "media sosial"},
		indicators: []indicatorSeed{
			{"posting_ontime", "Posting tepat jadwal", "%", domain.HigherBetter, 0.95, 0.82, 0.18},
			{"engagement_growth", "Pertumbuhan engagement", "%", domain.HigherBetter, 0.05, 0.01, 0.07},
		},
		tasks: []taskSeed{
			{"Posting konten harian klien", "daily", "high", "posting_ontime", 0, []string{"Cek caption dan hashtag", "Jadwalkan di Meta Business Suite", "Balas komentar 1 jam pertama"}},
			{"Laporan insight mingguan", "weekly", "normal", "engagement_growth", 5, nil},
			{"Rencana konten bulan depan", "monthly", "high", "", 25, []string{"Riset tren", "Draft kalender", "Kirim ke klien"}},
		},
		checklist: []string{"Cek DM dan komentar klien", "Posting sesuai kalender", "Catat insight harian", "Update tracker konten"}},
	{key: "seo", match: []string{"seo", "search"},
		indicators: []indicatorSeed{{"keyword_top10", "Jumlah keyword top 10", "keyword", domain.HigherBetter, 25, 15, 15}},
		tasks: []taskSeed{
			{"Cek ranking keyword prioritas", "weekly", "high", "keyword_top10", 1, nil},
			{"Tulis artikel blog SEO", "weekly", "normal", "", 3, []string{"Riset keyword", "Tulis draft", "Optimasi on-page"}},
			{"Audit teknis situs klien", "monthly", "normal", "", 10, nil},
		},
		checklist: []string{"Cek Search Console", "Pantau ranking harian", "Update laporan backlink", "Catat temuan teknis"}},
	{key: "ads", match: []string{"ads", "iklan", "advertis", "performance"},
		indicators: []indicatorSeed{{"roas", "ROAS kampanye", "x", domain.HigherBetter, 4, 2.6, 2.4}},
		tasks: []taskSeed{
			{"Optimasi budget kampanye", "daily", "urgent", "roas", 0, nil},
			{"Uji materi iklan A/B", "weekly", "normal", "", 2, []string{"Siapkan dua materi", "Set audiens", "Bandingkan CTR"}},
			{"Laporan performa iklan mingguan", "weekly", "high", "", 5, nil},
		},
		checklist: []string{"Cek spend vs budget", "Pantau CPA dan ROAS", "Matikan iklan berperforma buruk", "Catat insight audiens"}},
	{key: "creative", match: []string{"creative", "kreatif", "design", "desain"},
		indicators: []indicatorSeed{{"design_revision_turnaround", "Waktu revisi desain", "jam", domain.LowerBetter, 24, 14, 22}},
		tasks: []taskSeed{
			{"Desain feed klien", "daily", "high", "design_revision_turnaround", 0, []string{"Moodboard", "Desain utama", "Ekspor format"}},
			{"Revisi desain sesuai feedback", "weekly", "high", "", 3, nil},
			{"Arsip aset ke library", "weekly", "low", "", 5, nil},
		},
		checklist: []string{"Cek brief masuk", "Kerjakan antrean desain", "Kirim preview ke AM", "Arsipkan file final"}},
	{key: "sales", match: []string{"sales", "penjualan", "business"},
		indicators: []indicatorSeed{{"deals_won", "Deal dimenangkan", "deal", domain.HigherBetter, 4, 1, 5}},
		tasks: []taskSeed{
			{"Follow up prospek", "daily", "high", "deals_won", 0, nil},
			{"Kirim proposal ke prospek", "weekly", "high", "", 2, []string{"Susun scope", "Hitung harga", "Kirim proposal"}},
			{"Update pipeline CRM", "weekly", "normal", "", 5, nil},
		},
		checklist: []string{"Follow up lead baru", "Update status deal", "Jadwalkan meeting", "Catat kebutuhan klien"}},
	{key: "finance", match: []string{"finance", "keuangan", "accounting"},
		indicators: []indicatorSeed{{"closing_ontime", "Tutup buku tepat waktu", "ya/tidak", domain.Boolean, 1, 0, 1.4}},
		tasks: []taskSeed{
			{"Rekonsiliasi kas harian", "daily", "normal", "", 0, nil},
			{"Penagihan invoice jatuh tempo", "weekly", "high", "", 1, []string{"Tarik daftar AR", "Kirim reminder", "Catat janji bayar"}},
			{"Tutup buku bulanan", "monthly", "urgent", "closing_ontime", 5, nil},
		},
		checklist: []string{"Input transaksi kas", "Cek mutasi bank", "Verifikasi reimburse", "Arsip bukti transaksi"}},
	{key: "hr", match: []string{"hr", "human", "sdm", "people"},
		indicators: []indicatorSeed{{"time_to_hire", "Waktu rekrutmen", "hari", domain.LowerBetter, 30, 18, 25}},
		tasks: []taskSeed{
			{"Screening kandidat", "daily", "normal", "time_to_hire", 0, nil},
			{"Rekap absensi mingguan", "weekly", "normal", "", 5, nil},
			{"Onboarding karyawan baru", "weekly", "high", "", 1, []string{"Siapkan akun", "Sesi orientasi", "Checklist dokumen"}},
		},
		checklist: []string{"Cek absensi pagi", "Proses pengajuan cuti", "Update data karyawan", "Jawab pertanyaan karyawan"}},
}

var generic = profile{key: "general",
	tasks: []taskSeed{
		{"Update laporan mingguan tim", "weekly", "normal", "", 5, nil},
		{"Rapat koordinasi harian", "daily", "low", "", 0, nil},
	},
	checklist: []string{"Cek email dan pesan", "Kerjakan prioritas hari ini", "Update progres ke atasan"}}

func profileFor(d domain.Department) profile {
	name := strings.ToLower(d.Name + " " + d.Code)
	for _, p := range profiles {
		for _, m := range p.match {
			if strings.Contains(name, m) {
				return p
			}
		}
	}
	return generic
}

func (s *seeder) indicators(ctx context.Context) error {
	for _, p := range profiles {
		for _, ind := range p.indicators {
			if err := s.exec(ctx, `INSERT INTO performance.kpi_indicators (id, code, name, unit, direction, source_kind, default_target, created_by, updated_by)
				VALUES ($1, $2, $3, $4, $5, 'manual', $6, $7, $7) ON CONFLICT (code) DO NOTHING`,
				"ind_"+ind.code, ind.code, ind.name, ind.unit, ind.direction, ind.target, seedUser); err != nil {
				return err
			}
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT code, id FROM performance.kpi_indicators`)
	if err != nil {
		return err
	}
	defer rows.Close()
	s.ids = map[string]string{}
	for rows.Next() {
		var code, id string
		if err := rows.Scan(&code, &id); err != nil {
			return err
		}
		s.ids[code] = id
	}
	return rows.Err()
}

func (s *seeder) members(deptID string) []domain.Employee {
	var out []domain.Employee
	for _, e := range s.staff {
		if e.InDepartment(deptID) {
			out = append(out, e)
		}
	}
	return out
}

func (s *seeder) weights(ctx context.Context) error {
	for _, d := range s.depts {
		p := profileFor(d)
		items := map[string]float64{domain.CodeTaskCompletion: 20, domain.CodeAttOntime: 10, domain.CodeLogbookCompliance: 10, domain.CodeSupervisorRubric: 20}
		if len(p.indicators) == 0 {
			items[domain.CodeTaskCompletion], items[domain.CodeAttOntime], items[domain.CodeLogbookCompliance] = 40, 20, 20
		}
		for _, ind := range p.indicators {
			items[ind.code] = 40 / float64(len(p.indicators))
		}
		for code, w := range items {
			id, ok := s.ids[code]
			if !ok {
				return fmt.Errorf("indikator %s tidak ada, jalankan migrate", code)
			}
			if err := s.exec(ctx, `INSERT INTO performance.kpi_department_indicators (department_id, indicator_id, weight, updated_by)
				VALUES ($1, $2, $3, 'Seeder') ON CONFLICT DO NOTHING`, d.ID, id, w); err != nil {
				return err
			}
		}
	}
	return nil
}
