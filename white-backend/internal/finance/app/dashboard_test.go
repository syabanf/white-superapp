package app_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// The dashboard tabs agree with the statements they summarize, on the demo
// books (FY2025 closed, FY2026 to 26 October).
func TestDashboardTabs(t *testing.T) {
	pool := pg.TestPool(t, finance.Migrations(), "migrations", finance.Schema)
	pg.TruncateSchema(t, pool, finance.Schema)
	ctx := context.Background()
	deps := app.Deps{
		Store: postgres.NewStore(pool), Storage: storage.NewLocal(t.TempDir()), Events: events.Noop{}, Insights: &fakeInsights{},
		Identity: &fakeIdentity{}, Clock: clock.At(today + "T03:00:00Z"),
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}
	if err := seed.Reference(ctx, deps); err != nil {
		t.Fatal(err)
	}
	if err := seed.Demo(ctx, deps); err != nil {
		t.Fatal(err)
	}
	reports := app.NewReports(deps)
	f := app.AnalyticsFilter{From: "2026-07-01", To: "2026-09-30", Compare: app.CompareYear}

	sum, err := reports.Summary(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	w := sum.Window
	if w.PrevFrom != "2025-07-01" || w.PrevTo != "2025-09-30" || len(w.Months) != 3 || w.PrevMonths[0] != "2025-07" {
		t.Fatalf("window %+v", w)
	}
	is, err := reports.IncomeStatement(ctx, f.From, f.To)
	if err != nil {
		t.Fatal(err)
	}
	prevIs, err := reports.IncomeStatement(ctx, w.PrevFrom, w.PrevTo)
	if err != nil {
		t.Fatal(err)
	}
	kpi := map[string]app.Kpi{}
	for _, k := range sum.Kpis {
		kpi[k.Key] = k
	}
	if kpi["revenue"].Value != is.TotalRevenue.String() || kpi["revenue"].Previous != prevIs.TotalRevenue.String() || prevIs.TotalRevenue == 0 {
		t.Fatalf("revenue kpi %+v, statements %v / %v", kpi["revenue"], is.TotalRevenue, prevIs.TotalRevenue)
	}
	if kpi["net_income"].Value != is.NetIncome.String() {
		t.Fatalf("net income kpi %+v, statement %v", kpi["net_income"], is.NetIncome)
	}
	var monthly domain.Money
	for _, v := range sum.Pnl[0].Values {
		monthly += v
	}
	if monthly != is.TotalRevenue {
		t.Fatalf("monthly revenue %v, statement %v", monthly, is.TotalRevenue)
	}

	prof, err := reports.Profitability(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	var running domain.Money
	for _, s := range prof.Waterfall {
		if s.Total {
			if s.Value != running && s.Key != "revenue" {
				t.Fatalf("waterfall %s = %v, running %v", s.Key, s.Value, running)
			}
			running = s.Value
			continue
		}
		running += s.Value
	}
	if running != is.NetIncome || len(prof.RevenueClients) == 0 || len(prof.ExpenseGroups) == 0 {
		t.Fatalf("waterfall ends at %v, net income %v, %d clients", running, is.NetIncome, len(prof.RevenueClients))
	}
	staff, err := reports.Profitability(ctx, app.AnalyticsFilter{From: f.From, To: f.To, AccountType: domain.TypeExpense})
	if err != nil {
		t.Fatal(err)
	}
	if len(staff.RevenueAccounts) != 0 || len(staff.ExpenseAccounts) == 0 {
		t.Fatalf("type filter kept %d revenue accounts", len(staff.RevenueAccounts))
	}

	cash, err := reports.CashAnalytics(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	var bridge domain.Money
	for _, s := range cash.Bridge[:len(cash.Bridge)-1] {
		bridge += s.Value
	}
	if closing := cash.Bridge[len(cash.Bridge)-1].Value; bridge != closing || closing == 0 {
		t.Fatalf("bridge sums to %v, closing %v", bridge, closing)
	}
	var accounts domain.Money
	for _, a := range cash.Accounts {
		accounts += a.Closing
		if a.Opening+a.Inflow-a.Outflow != a.Closing {
			t.Fatalf("cash account %s does not roll forward: %+v", a.Name, a)
		}
	}
	if accounts != cash.Bridge[len(cash.Bridge)-1].Value {
		t.Fatalf("accounts %v, closing %v", accounts, cash.Bridge[len(cash.Bridge)-1].Value)
	}

	arap, err := reports.ArAp(ctx, app.AnalyticsFilter{To: today})
	if err != nil {
		t.Fatal(err)
	}
	var aged domain.Money
	for _, a := range arap.Aging {
		if a.Kind == domain.KindAR {
			aged += a.Amount
		}
	}
	open, err := app.NewLedgers(deps).OpenInvoices(ctx, domain.KindAR)
	if err != nil {
		t.Fatal(err)
	}
	var outstanding domain.Money
	for _, inv := range open {
		outstanding += inv.Outstanding()
	}
	if aged != outstanding || len(arap.Overdue) == 0 {
		t.Fatalf("aging %v, open invoices %v, %d overdue", aged, outstanding, len(arap.Overdue))
	}

	spend, err := reports.Spend(ctx, app.AnalyticsFilter{From: "2026-01-01", To: today, Department: "Creative"})
	if err != nil {
		t.Fatal(err)
	}
	if len(spend.ByDepartment) != 1 || spend.ByDepartment[0].Label != "Creative" || len(spend.ClaimCategories) == 0 {
		t.Fatalf("department filter: %+v", spend.ByDepartment)
	}
	if spend.LeadTime[3].Decimal == "" || len(spend.Departments) < 5 {
		t.Fatalf("lead time %+v, departments %v", spend.LeadTime, spend.Departments)
	}

	months, ratios, err := app.NewAnalysis(deps).RatioTrend(ctx, today, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(months) != 12 || months[0] != "2025-11" || months[11] != "2026-10" {
		t.Fatalf("months %v", months)
	}
	for _, r := range ratios {
		if r.Key == "current_ratio" && (r.Benchmark == nil || !r.Points[0].OK) {
			t.Fatalf("current ratio trend %+v", r)
		}
	}
}
