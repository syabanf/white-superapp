package app_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/render"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

func kpi(t *testing.T, ks []domain.DashKpi, key string) domain.DashKpi {
	t.Helper()
	for _, k := range ks {
		if k.Key == key {
			return k
		}
	}
	t.Fatalf("kpi %s missing", key)
	return domain.DashKpi{}
}

func TestDashboardOverSeededHistory(t *testing.T) {
	pool := pg.TestPool(t, sales.Migrations(), "migrations", sales.Schema)
	pg.TruncateSchema(t, pool, sales.Schema)
	ctx := context.Background()
	if err := seed.Reference(ctx, pool); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for range 2 { // the second run must change nothing
		if err := seed.History(ctx, pool, []string{salesID, managerID}, now); err != nil {
			t.Fatal(err)
		}
	}
	deps := &app.Deps{Store: postgres.NewStore(pool), Clock: clock.Real{}, Events: &recorder{}, Notifier: &recorder{}, Directory: directory{}, Clients: &clients{}, AI: mockAI{}, WhatsApp: &whatsapp{}, Files: storage.NewLocal(t.TempDir()), Renderer: render.New()}
	svc := sales.Services(deps)
	apps := seed.Apps{Store: deps.Store, Leads: svc.Leads, Pipeline: svc.Pipeline, Quotes: svc.Quotes, Billing: svc.Billing, Contracts: svc.Contracts, Chat: svc.Chat, Decks: svc.Decks, Onboarding: svc.Onboarding}
	for range 2 {
		if err := seed.Showcase(ctx, pool, apps, []string{salesID, managerID}); err != nil {
			t.Fatal(err)
		}
	}
	for table, min := range map[string]int{"contacts": 50, "custom_fields": 6, "deal_members": 50, "approval_requests": 3, "contracts": 6, "pitch_decks": 4, "onboardings": 3, "onboarding_assets": 4, "conversation_notes": 1, "conversation_insights": 4} {
		var n int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM sales."+table).Scan(&n)
		if n < min {
			t.Fatalf("%s: %d rows, want >= %d", table, n, min)
		}
	}
	var statuses int
	_ = pool.QueryRow(ctx, `SELECT count(DISTINCT status) FROM sales.contracts`).Scan(&statuses)
	if statuses != 5 {
		t.Fatalf("contract statuses: %d", statuses)
	}
	var leads, wonDeals int
	var wonValue float64
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM sales.leads`).Scan(&leads)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, domain.Jakarta).AddDate(0, -11, 0)
	_ = pool.QueryRow(ctx, `SELECT count(*), coalesce(sum(coalesce(value_final, value_estimate)), 0)::float8 FROM sales.deals WHERE stage_id = 'st-menang' AND closed_at >= $1`, from).Scan(&wonDeals, &wonValue)
	if leads < 150 || wonDeals < 10 {
		t.Fatalf("thin history: %d leads, %d won", leads, wonDeals)
	}

	d := app.NewDashboard(&app.Deps{Store: postgres.NewStore(pool), Clock: clock.Real{}, Directory: directory{}})
	last12 := app.DashQuery{From: from.Format("2006-01-02"), To: now.In(domain.Jakarta).Format("2006-01-02"), Compare: "year"}

	sum := must(d.Summary(ctx, last12))
	eq := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > 1 {
			t.Fatalf("%s: got %.2f, want %.2f", name, got, want)
		}
	}
	eq("won value", kpi(t, sum.Kpis, "won_value").Value, wonValue)
	eq("won deals", kpi(t, sum.Kpis, "won_deals").Value, float64(wonDeals))
	if len(sum.Period.Months) != 12 || len(sum.Trend[0].Values) != 12 || len(kpi(t, sum.Kpis, "pipeline_value").Spark) != 12 {
		t.Fatalf("trend window: %v", sum.Period.Months)
	}
	sparkTotal := 0.0
	for _, v := range sum.Trend[0].Values {
		sparkTotal += v
	}
	eq("monthly won adds up", sparkTotal, wonValue)
	if kpi(t, sum.Kpis, "pipeline_value").Value <= 0 || len(sum.Stages) != 4 {
		t.Fatalf("open pipeline: %+v", sum.Stages)
	}

	pipe := must(d.Pipeline(ctx, last12))
	if pipe.PipelineID != "pl-agency" || len(pipe.Funnel) != 5 || pipe.Funnel[0].Count < pipe.Funnel[4].Count || pipe.Funnel[4].Count == 0 {
		t.Fatalf("funnel: %+v", pipe.Funnel)
	}
	if len(pipe.StageDays) == 0 || len(pipe.Aging) == 0 {
		t.Fatal("stage durations or aging empty")
	}

	sl := must(d.Sales(ctx, last12))
	if len(sl.ServiceLines) == 0 || len(sl.Industries) == 0 || len(sl.LostReasons) == 0 || kpi(t, sl.Kpis, "quotes_sent").Value == 0 {
		t.Fatalf("sales breakdowns: %+v", sl.Kpis)
	}

	team := must(d.Team(ctx, last12))
	if len(team.Members) != 2 || team.Members[0].Name == team.Members[0].UserID || team.Members[0].TargetValue <= 0 {
		t.Fatalf("team: %+v", team.Members)
	}
	eq("team won", kpi(t, team.Kpis, "won_value").Value, wonValue)

	ld := must(d.Leads(ctx, last12))
	if kpi(t, ld.Kpis, "leads").Value == 0 || len(ld.Scores) != 5 || kpi(t, ld.Kpis, "conversations").Value == 0 {
		t.Fatalf("leads: %+v", ld.Kpis)
	}

	bill := must(d.Billing(ctx, last12))
	if kpi(t, bill.Kpis, "invoiced").Value == 0 || kpi(t, bill.Kpis, "collected").Value == 0 || len(bill.Aging) != 5 || len(bill.Terms) == 0 {
		t.Fatalf("billing: %+v", bill.Kpis)
	}

	// Filters narrow every aggregate.
	mine := last12
	mine.Filter.OwnerUserID = salesID
	if got := kpi(t, must(d.Summary(ctx, mine)).Kpis, "won_value").Value; got >= wonValue || got <= 0 {
		t.Fatalf("owner filter: %.0f of %.0f", got, wonValue)
	}
	seo := last12
	seo.Filter.ServiceLine = domain.LineSEO
	for _, s := range must(d.Sales(ctx, seo)).ServiceLines {
		if s.Key == domain.LineSEO && s.Value <= 0 {
			t.Fatal("seo filter lost its own line")
		}
	}

	_, err := d.Summary(ctx, app.DashQuery{From: "2026-02-30"})
	if !apperr.Is(err, apperr.Validation) {
		t.Fatalf("bad date: %v", err)
	}
}
