package app_test

import (
	"context"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/performance"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

func near(t *testing.T, what string, got *float64, want float64) {
	t.Helper()
	if got == nil || *got < want-0.01 || *got > want+0.01 {
		if got == nil {
			t.Fatalf("%s: want %.2f, got nil", what, want)
		}
		t.Fatalf("%s: want %.2f, got %.2f", what, want, *got)
	}
}

// dashboardFixture: Social Media has a daily task for Rani (approved on
// time on 5 Oct, still pending from 6 Oct, approved on 10 Sep for the
// comparison window), October scorecards for Rani, Dimas and Bayu, and
// three improvement items.
func dashboardFixture(t *testing.T) *fixture {
	f := setup(t)
	pool := pg.TestPool(t, performance.Migrations(), "migrations", performance.Schema)
	ctx := context.Background()
	for _, sql := range []string{
		`INSERT INTO performance.department_tasks (id, department_id, assignee_employee_id, title, recurrence) VALUES
			('t1', 'dept_social', 'emp_rani', 'Posting harian', 'daily')`,
		`INSERT INTO performance.department_task_occurrences (id, task_id, occurrence_date, status, done_at, reviewed_at) VALUES
			('o1', 't1', '2026-10-05', 'approved', '2026-10-05 08:00+00', '2026-10-05 11:00+00'),
			('o2', 't1', '2026-10-06', 'pending', NULL, NULL),
			('o3', 't1', '2026-09-10', 'approved', '2026-09-12 08:00+00', '2026-09-12 09:00+00')`,
		`INSERT INTO performance.kpi_scorecards (id, employee_id, department_id, period_year, period_month, score, status) VALUES
			('s1', 'emp_rani', 'dept_social', 2026, 10, 82, 'final'),
			('s2', 'emp_dimas', 'dept_social', 2026, 10, 60, 'draft'),
			('s3', 'emp_bayu', 'dept_seo', 2026, 10, 90, 'draft')`,
		`INSERT INTO performance.improvements (id, scope, employee_id, department_id, source, title, status, due_date, created_at) VALUES
			('i1', 'employee', 'emp_rani', 'dept_social', 'ai', 'Kirim update lebih awal', 'declined', NULL, '2026-10-02 03:00+00'),
			('i2', 'employee', 'emp_rani', 'dept_social', 'ai', 'Pecah tugas besar', 'open', NULL, '2026-10-03 03:00+00'),
			('i3', 'department', NULL, 'dept_seo', 'manual', 'Rapikan SOP', 'open', '2026-10-10', '2026-10-01 03:00+00')`,
		`UPDATE performance.improvements SET updated_at = created_at`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func TestDashboardScopesAndNumbers(t *testing.T) {
	f := dashboardFixture(t)
	d := app.NewDashboard(app.Deps{Store: f.store, Directory: fakeDirectory{}, Clock: f.clk})
	oct := app.DashboardQuery{From: "2026-10-01", To: "2026-10-21"}

	sum, err := d.Summary(hrAdmin, oct)
	must(t, err)
	if sum.Window.Scope != "all" || sum.Window.CompareFrom != "2026-09-10" || sum.Window.CompareTo != "2026-09-30" || sum.Window.Granularity != "day" {
		t.Fatalf("window %+v", sum.Window)
	}
	near(t, "task completion", sum.TaskCompletion.Value, 0.5)
	near(t, "previous completion", sum.TaskCompletion.Previous, 1)
	near(t, "overdue", sum.OverdueTasks.Value, 1)
	near(t, "kpi", sum.KPIScore.Value, 77.33)
	near(t, "open improvements", sum.OpenImprovements.Value, 2)
	if len(sum.Months) != 12 || sum.Months[11] != "2026-10" {
		t.Fatalf("months %v", sum.Months)
	}

	// Self-service only: always "me", whatever the request says.
	mine, err := d.KPI(rani, app.DashboardQuery{From: oct.From, To: oct.To, Scope: "all"})
	must(t, err)
	if mine.Window.Scope != "me" {
		t.Fatal(mine.Window.Scope)
	}
	near(t, "own score", mine.AvgScore.Value, 82)
	_, err = d.KPI(rani, app.DashboardQuery{EmployeeID: "emp_dimas"})
	code(t, err, apperr.Unauthorized)

	// A department head sees the team.
	team, err := d.KPI(hana, oct)
	must(t, err)
	if team.Window.Scope != "team" || len(team.People) != 2 {
		t.Fatalf("%s %+v", team.Window.Scope, team.People)
	}
	near(t, "team score", team.AvgScore.Value, 71)
	if team.Funnel[0].Value != 3 || team.Funnel[1].Value != 2 || team.Funnel[2].Value != 1 {
		t.Fatalf("funnel %+v", team.Funnel)
	}

	tasks, err := d.Tasks(hrAdmin, app.DashboardQuery{From: oct.From, To: oct.To, DepartmentID: "dept_social"})
	must(t, err)
	if tasks.Aging[3].Key != "15-30" || tasks.Aging[3].Value != 1 || tasks.Turnaround[0].Value != 1 {
		t.Fatalf("aging %+v turnaround %+v", tasks.Aging, tasks.Turnaround)
	}
	near(t, "turnaround", tasks.TurnaroundHours.Value, 3)

	imp, err := d.Improvements(hrAdmin, oct)
	must(t, err)
	near(t, "ai acceptance", imp.AIAccept.Value, 0.5)
	near(t, "overdue items", imp.Overdue.Value, 1)
	if len(imp.OverdueItems) != 1 || imp.OverdueItems[0].DaysOverdue != 11 {
		t.Fatalf("%+v", imp.OverdueItems)
	}

	ctxOpts, err := d.Context(hana, app.DashboardQuery{})
	must(t, err)
	if len(ctxOpts.Scopes) != 2 || len(ctxOpts.People) != 3 || len(ctxOpts.Departments) != 1 {
		t.Fatalf("%+v", ctxOpts)
	}
	_, err = d.Summary(hrAdmin, app.DashboardQuery{From: "2026-10-21", To: "2026-10-01"})
	code(t, err, apperr.Validation)
}
