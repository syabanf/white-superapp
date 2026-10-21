package scheduler

import (
	"testing"
	"time"
)

func at(s string) *time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return &t
}

// Same cases as tests/cron.test.ts in the Next.js app.
func TestPlanDailyRun(t *testing.T) {
	clients := []ClientRun{
		{ID: "a", LastRunAt: at("2026-09-18")},
		{ID: "b"},
		{ID: "c", LastRunAt: at("2026-09-10")},
		{ID: "d", LastRunAt: at("2026-09-17")},
		{ID: "e", LastRunAt: at("2026-09-16")},
	}
	p := PlanDailyRun(clients, PlanOptions{Concurrency: 2, BudgetMs: 200, PerClientMs: 100})
	got := []string{}
	for _, c := range p.Run {
		got = append(got, c.ID)
	}
	if len(got) != 4 || got[0] != "b" || got[1] != "c" || got[2] != "e" || got[3] != "d" {
		t.Fatalf("run = %v", got)
	}
	if len(p.Deferred) != 1 || p.Deferred[0].ID != "a" {
		t.Fatalf("deferred = %v", p.Deferred)
	}
	if one := PlanDailyRun([]ClientRun{{ID: "a"}}, PlanOptions{Concurrency: 1, BudgetMs: 10, PerClientMs: 100}); len(one.Run) != 1 {
		t.Fatal("always at least one wave")
	}
}
