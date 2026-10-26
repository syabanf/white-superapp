package seed_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// The seed is idempotent and leaves a balanced ledger with every automatic
// journal posted.
func TestSeedIsIdempotent(t *testing.T) {
	pool := pg.TestPool(t, finance.Migrations(), "migrations", finance.Schema)
	pg.TruncateSchema(t, pool, finance.Schema)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	deps := app.Deps{
		Store: postgres.NewStore(pool), Storage: storage.NewLocal(t.TempDir()), Events: events.Noop{},
		Insights: openrouter.NewSource(func(context.Context) (openrouter.Credentials, error) { return openrouter.Credentials{}, nil }, openrouter.Options{}),
		Clock:    clock.At("2026-10-26T03:00:00Z"), Logger: logger,
	}
	count := func() (int, int) {
		_, entries, err := deps.Store.ListJournalEntries(ctx, domain.JournalFilter{Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		_, claims, _, err := deps.Store.ListClaims(ctx, domain.ClaimFilter{Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		return entries, claims
	}
	for run := 0; run < 2; run++ {
		if err := seed.Reference(ctx, deps); err != nil {
			t.Fatal(err)
		}
		if err := seed.Demo(ctx, deps); err != nil {
			t.Fatal(err)
		}
	}
	entries, claims := count()
	if entries < 800 || claims < 200 {
		t.Fatalf("entries %d claims %d", entries, claims)
	}
	// Only the bonus run dated in a closed period stays DRAFT.
	logs, err := deps.Store.ListPostingLogs(ctx, domain.PostDraft, "", 50)
	if err != nil || len(logs) == 0 {
		t.Fatalf("draft postings %+v %v", logs, err)
	}
	for _, l := range logs {
		if !strings.HasPrefix(l.DocumentID, "seed-bonus-") {
			t.Fatalf("unexpected draft posting %+v", l)
		}
	}
	reports := app.NewReports(deps)
	tb, err := reports.TrialBalance(ctx, "")
	if err != nil || tb.TotalDebit != tb.TotalCredit || tb.TotalDebit == 0 {
		t.Fatalf("trial balance %v %v %v", tb.TotalDebit, tb.TotalCredit, err)
	}
	// Both years balance, and the prior year closes at the current year's
	// opening balance: January's cash flow reconciles across the boundary.
	for _, asOf := range []string{"2025-06-30", "2025-12-31", "2026-01-01", "2026-10-26"} {
		bs, err := reports.BalanceSheet(ctx, asOf)
		if err != nil || bs.TotalAssets != bs.TotalLiabilitiesAndEquity {
			t.Fatalf("balance sheet %s: assets %v, liabilities and equity %v, %v", asOf, bs.TotalAssets, bs.TotalLiabilitiesAndEquity, err)
		}
	}
	closing, err := reports.BalanceSheet(ctx, "2025-12-31")
	if err != nil {
		t.Fatal(err)
	}
	opening, err := reports.BalanceSheet(ctx, "2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	if closing.TotalAssets != opening.TotalAssets {
		t.Fatalf("prior year closes at %v, current year opens at %v", closing.TotalAssets, opening.TotalAssets)
	}
	cf, err := reports.CashFlow(ctx, "2026-01-01", "2026-01-31")
	if err != nil {
		t.Fatal(err)
	}
	if got := cf.Operating.Total + cf.Investing.Total + cf.Financing.Total + cf.NonCash.Total; got != cf.NetChange {
		t.Fatalf("january cash flow %v, cash change %v", got, cf.NetChange)
	}
}
