package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/providers/ai"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/content/seed"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

func TestSeedIsIdempotentAndCoversEveryState(t *testing.T) {
	pool := pg.TestPool(t, content.Migrations(), "migrations", content.Schema)
	pg.TruncateSchema(t, pool, content.Schema)
	ctx := context.Background()
	store := postgres.NewStore(pool)
	s := &seed.Seeder{
		Store: store, Files: storage.NewLocal(t.TempDir()), Now: time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC),
		AI: ai.NewSource(func(context.Context) (ai.Credentials, error) { return ai.Credentials{}, nil }, ai.Options{}),
	}
	clients := []seed.Client{{ID: "c1", Name: "Kopi Senja"}, {ID: "c2", Name: "Batik Nusa"}}
	res, err := s.Run(ctx, clients, []string{"u1", "u2", "u3"})
	if err != nil {
		t.Fatal(err)
	}
	// Per client: last and next month, 23 history months, the current month
	// and the plan in review two months ahead.
	if res.Clients != 2 || res.Plans != 54 || res.HistoryMonths != 48 || res.Reports != 48 || res.Assets != 22 {
		t.Fatalf("%+v", res)
	}
	items, err := store.ListItems(ctx, "c1", domain.ItemFilter{})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[domain.ItemStatus]bool{}
	for _, it := range items {
		seen[it.Status] = true
	}
	for _, st := range domain.ItemStatuses {
		if !seen[st] {
			t.Errorf("no item in %s", st)
		}
	}
	reports, _ := store.ListReports(ctx, "c1")
	if len(reports) != 24 || reports[0].Month.String() != "2026-09" || reports[23].Month.String() != "2024-10" || !reports[0].Data.EngagementAvailable || reports[0].Data.Published != 9 {
		t.Fatalf("%+v", reports)
	}
	plans, _ := store.ListPlans(ctx, "c1")
	if len(plans) != 27 || plans[0].Status != domain.PlanInReview || plans[1].Status != domain.PlanDraft || plans[2].Status != domain.PlanApproved {
		t.Fatalf("%+v", plans)
	}
	tasks, _ := store.ListDesignTasks(ctx, "c1", "", "", "")
	designSeen := map[domain.DesignStatus]bool{}
	for _, tk := range tasks {
		designSeen[tk.Status] = true
	}
	assets, _ := store.SearchAssets(ctx, "c1", domain.AssetFilter{})
	kinds := map[domain.AssetKind]bool{}
	for _, a := range assets {
		kinds[a.Kind] = true
	}
	if len(designSeen) != 5 || len(kinds) != 6 {
		t.Fatalf("design statuses %v asset kinds %v", designSeen, kinds)
	}
	// A year of history: published items carry backdated stamps and a snapshot.
	year, err := store.DashboardItems(ctx, "c1", domain.DashboardFilter{From: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	var published []string
	for _, it := range year {
		if it.Status == domain.ItemPublished {
			if it.PublishedAt == nil || it.CreatedAt.After(*it.PublishedAt) || it.ThumbnailFileID == "" {
				t.Fatalf("stamps %+v", it)
			}
			published = append(published, it.ID)
		}
	}
	metrics, _ := store.ItemMetrics(ctx, published)
	if len(year) < 140 || len(published) < 110 || len(metrics) != len(published) {
		t.Fatalf("items %d published %d metrics %d", len(year), len(published), len(metrics))
	}

	again, err := s.Run(ctx, clients, nil)
	if err != nil || again.Clients != 0 || len(again.Skipped) != 2 || again.HistoryMonths != 0 || again.Reports != 0 || again.Assets != 0 {
		t.Fatalf("%+v %v", again, err)
	}
}
