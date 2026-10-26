package app_test

import (
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/docs"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
)

func TestHrDashboardOverSeed(t *testing.T) {
	e := setup(t)
	e.d.Events = events.Noop{}
	users := []app.UserInfo{{ID: "user-admin", Email: "admin@white.id"}, {ID: "user-tim", Email: "tim@white.id"}}
	e.must(seed.New(e.d, docs.Renderer{CompanyName: "WHITE"}).Run(e.ctx, users))
	dash := app.NewDashboard(e.d, e.leaves)

	_, err := dash.HR(e.ctx, e.actor("user-tim", authn.PermSelfService), app.DashboardQuery{})
	wantCode(t, err, apperr.Unauthorized)
	_, err = dash.HR(e.ctx, e.admin, app.DashboardQuery{From: "2026-10-10", To: "2026-10-01"})
	wantCode(t, err, apperr.Validation)

	// September 2026 against August, the whole company.
	d, err := dash.HR(e.ctx, e.admin, app.DashboardQuery{From: "2026-09-01", To: "2026-09-30"})
	e.must(err)
	if d.CompareFrom != "2026-08-02" || d.CompareTo != "2026-08-31" || d.Bucket != "day" || len(d.Buckets) != 30 {
		t.Fatalf("range %s..%s bucket %s (%d)", d.CompareFrom, d.CompareTo, d.Bucket, len(d.Buckets))
	}
	if d.Current.Headcount != 29 || d.Current.AttendanceRecords == 0 || d.Current.AttendanceRate <= 80 {
		t.Fatalf("september kpis %+v", d.Current)
	}
	if !d.PayrollVisible || d.Current.PayrollGross <= 0 || d.Current.PayrollEmployees != 29 {
		t.Fatalf("september payroll %+v", d.Current)
	}
	if len(d.Months) != 12 || d.Months[0].Date != "2025-10-01" || d.Months[11].Date != "2026-09-01" {
		t.Fatalf("months %v", d.Months)
	}
	leavers := 0
	for _, m := range d.Months {
		leavers += m.Leavers
		if m.PayrollGross <= 0 {
			t.Fatalf("every history month is paid: %+v", m)
		}
	}
	if leavers != 5 {
		t.Fatalf("five people left during the year, got %d", leavers)
	}
	sum := 0
	for _, dep := range d.Departments {
		sum += dep.Headcount
	}
	if sum != d.Current.Headcount || len(d.DeptMonths) != 12*len(d.Departments) {
		t.Fatalf("departments add up to %d of %d", sum, d.Current.Headcount)
	}
	if len(d.ClockInOffsets) != 19 || len(d.LateByWeekday) == 0 || len(d.SalaryByLevel) == 0 || len(d.Loans) != 2 {
		t.Fatalf("breakdowns offsets=%d weekday=%d levels=%d loans=%d", len(d.ClockInOffsets), len(d.LateByWeekday), len(d.SalaryByLevel), len(d.Loans))
	}
	if len(d.PendingAging) != 4 || len(d.Pending) == 0 {
		t.Fatalf("pending %v", d.PendingAging)
	}

	// A twelve-month range buckets by month; year comparison shifts dates.
	y, err := dash.HR(e.ctx, e.admin, app.DashboardQuery{From: "2025-10-01", To: "2026-09-30", Compare: "year"})
	e.must(err)
	if y.Bucket != "month" || len(y.Buckets) != 12 || y.CompareFrom != "2024-10-01" || y.Current.Leavers != 5 || y.Current.Joiners == 0 {
		t.Fatalf("year view %s %d %s %+v", y.Bucket, len(y.Buckets), y.CompareFrom, y.Current)
	}

	// Filtering by department narrows every number; hr.read alone hides pay.
	var creative string
	for _, dep := range d.Departments {
		if dep.Name == "Creative" {
			creative = dep.DepartmentID
		}
	}
	reader := app.Actor{Principal: authn.Principal{UserID: "user-reader", Role: authn.RoleMember, Perms: []string{authn.PermHRRead}}}
	c, err := dash.HR(e.ctx, reader, app.DashboardQuery{From: "2026-09-01", To: "2026-09-30", Filter: domain.DashFilter{DepartmentID: creative}})
	e.must(err)
	if c.Current.Headcount == 0 || c.Current.Headcount >= d.Current.Headcount || len(c.Departments) != 1 {
		t.Fatalf("creative filter %+v", c.Current)
	}
	if c.PayrollVisible || c.Current.PayrollGross != 0 || c.SalaryByLevel != nil || c.Loans != nil {
		t.Fatal("payroll stays hidden without hr.payroll")
	}
}

func TestMyDashboardOverSeed(t *testing.T) {
	e := setup(t)
	e.d.Events = events.Noop{}
	users := []app.UserInfo{{ID: "user-admin", Email: "admin@white.id"}, {ID: "user-tim", Email: "tim@white.id"}}
	e.must(seed.New(e.d, docs.Renderer{CompanyName: "WHITE"}).Run(e.ctx, users))
	dash := app.NewDashboard(e.d, e.leaves)

	_, err := dash.Me(e.ctx, e.actor("user-nobody", authn.PermSelfService), "")
	wantCode(t, err, apperr.NotFound)

	// Gita (tim@white.id) is a content strategist reporting to Fajar.
	gita := e.actor("user-tim", authn.PermSelfService)
	m, err := dash.Me(e.ctx, gita, "2026-09")
	e.must(err)
	if m.Month != "2026-09" || len(m.Days) != 30 || len(m.Trend) != 12 || m.Trend[11].Month != "2026-09-01" {
		t.Fatalf("month %s days %d trend %d", m.Month, len(m.Days), len(m.Trend))
	}
	if m.Stats.ScheduledDays == 0 || m.Stats.PresentDays == 0 || m.Stats.AvgClockIn == "" || len(m.Weeks) < 4 {
		t.Fatalf("stats %+v weeks %d", m.Stats, len(m.Weeks))
	}
	if m.LatestPayslip == nil || m.LatestPayslip.PeriodMonth != 9 || m.Trend[11].NetPay <= 0 {
		t.Fatal("the september payslip is the latest")
	}
	if m.IsManager || len(m.Team) != 0 {
		t.Fatal("gita has no reports")
	}
	// The CEO sees his direct reports.
	ceo, err := dash.Me(e.ctx, e.actor("user-admin", authn.PermSelfService), "")
	e.must(err)
	if !ceo.IsManager || len(ceo.Team) == 0 {
		t.Fatal("the CEO manages the COO and department heads")
	}
}
