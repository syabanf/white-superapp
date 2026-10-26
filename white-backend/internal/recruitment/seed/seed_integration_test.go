package seed

import (
	"context"
	"io/fs"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

type fakeDirectory struct{}

func (fakeDirectory) Departments(context.Context) ([]domain.Department, error) {
	return []domain.Department{{ID: "d_soc", Name: "Social Media", Active: true}, {ID: "d_hr", Name: "HR", Active: true}}, nil
}

func (fakeDirectory) Employee(context.Context, string) (*domain.Employee, error) { return nil, nil }

func (fakeDirectory) Employees(context.Context) ([]domain.Employee, error) {
	return []domain.Employee{
		{ID: "e_rani", UserID: ptr("u_rani"), Name: "Rani", DepartmentID: ptr("d_soc"), DepartmentName: "Social Media", Active: true},
		{ID: "e_wulan", Name: "Wulan Sari", DepartmentID: ptr("d_hr"), DepartmentName: "HR", Active: true},
	}, nil
}

func TestSeedIsIdempotent(t *testing.T) {
	pool := pg.TestPool(t, recruitment.Migrations(), "migrations", recruitment.Schema)
	pg.TruncateSchema(t, pool, recruitment.Schema)
	raw, err := fs.ReadFile(recruitment.Migrations(), "00002_reference_data.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _ := strings.Cut(string(raw), "-- +goose Down")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatal(err)
	}
	counts := func() string {
		var out string
		err := pool.QueryRow(ctx, `SELECT concat_ws(',',
			(SELECT count(*) FROM recruitment.job_openings), (SELECT count(*) FROM recruitment.candidates),
			(SELECT count(*) FROM recruitment.candidate_activities), (SELECT count(*) FROM recruitment.psikotes_questions),
			(SELECT count(*) FROM recruitment.psikotes_sessions), (SELECT count(*) FROM recruitment.interview_ai_turns),
			(SELECT count(*) FROM recruitment.interviews), (SELECT count(*) FROM recruitment.candidate_offers),
			(SELECT count(*) FROM recruitment.candidates WHERE status = 'talent_pool'))`).Scan(&out)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	now := time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC)
	files := storage.NewLocal(t.TempDir())
	logger := slog.New(slog.DiscardHandler)
	if err := Seed(ctx, pool, fakeDirectory{}, files, now, logger); err != nil {
		t.Fatal(err)
	}
	first := counts()
	if err := Seed(ctx, pool, fakeDirectory{}, files, now, logger); err != nil {
		t.Fatal(err)
	}
	if again := counts(); again != first {
		t.Fatalf("seed must be idempotent: %s then %s", first, again)
	}
	if first != "30,480,1813,30,234,425,210,41,41" {
		t.Fatal(first)
	}
	// every page has data: statuses of each record kind, CVs, analyses, notes
	var coverage string
	if err := pool.QueryRow(ctx, `SELECT concat_ws('|',
		(SELECT string_agg(DISTINCT status, ',' ORDER BY status) FROM recruitment.job_openings),
		(SELECT string_agg(DISTINCT status, ',' ORDER BY status) FROM recruitment.psikotes_sessions),
		(SELECT string_agg(DISTINCT status, ',' ORDER BY status) FROM recruitment.psikotes_session_tests),
		(SELECT string_agg(DISTINCT status, ',' ORDER BY status) FROM recruitment.candidate_offers),
		(SELECT string_agg(DISTINCT status, ',' ORDER BY status) FROM recruitment.interviews),
		(SELECT count(*) >= 40 FROM recruitment.candidates WHERE cv_key IS NOT NULL),
		(SELECT count(*) >= 40 FROM recruitment.candidate_ai_analysis),
		(SELECT count(*) >= 40 FROM recruitment.candidate_notes))`).Scan(&coverage); err != nil {
		t.Fatal(err)
	}
	if coverage != "closed,draft,published|completed,expired,in_progress,sent|in_progress,pending,perlu_review,reviewed,selesai|"+
		"accepted,declined,expired,negotiating,sent|cancelled,completed,scheduled|t|t|t" {
		t.Fatal(coverage)
	}
	cv, err := files.Get(ctx, "cv/seed_cand_01/seed.pdf")
	if err != nil || !strings.HasPrefix(string(cv), "%PDF-") {
		t.Fatal("cv file", err)
	}
	// history spans a year and its hires never publish a hire event
	var months, unpublished int
	if err := pool.QueryRow(ctx, `SELECT count(DISTINCT date_trunc('month', created_at)),
		count(*) FILTER (WHERE status = 'hired' AND hired_event_at IS NULL) FROM recruitment.candidates`).Scan(&months, &unpublished); err != nil ||
		months < 12 || unpublished != 0 {
		t.Fatal(err, months, unpublished)
	}
	var dept, manager, interviewer string
	if err := pool.QueryRow(ctx, `SELECT department_id, hiring_manager_user_id FROM recruitment.job_openings WHERE id = 'seed_job_smm'`).
		Scan(&dept, &manager); err != nil || dept != "d_soc" || manager != "u_rani" {
		t.Fatal(err, dept, manager)
	}
	if err := pool.QueryRow(ctx, `SELECT interviewer_name FROM recruitment.interviews WHERE type = 'hrd' LIMIT 1`).Scan(&interviewer); err != nil ||
		interviewer != "Wulan Sari" {
		t.Fatal(err, interviewer)
	}
	var score float64
	if err := pool.QueryRow(ctx, `SELECT score FROM recruitment.psikotes_session_tests WHERE id = 'seed_pst_nanda_ar'`).Scan(&score); err != nil || score != 80 {
		t.Fatal(err, score)
	}
}
