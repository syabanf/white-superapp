// Package scheduler replaces the Vercel cron routes. It publishes ticks on
// NATS: one daily sync event per client for analytics and seosuite (ordered
// so the longest-waiting clients go first), a publish tick every five
// minutes and a retention tick once a day. The consuming services do the
// work, so a slow sync never blocks the scheduler.
package scheduler

import (
	"sort"
	"time"
)

// ClientRun is a client with the moment its last daily sync started.
type ClientRun struct {
	ID        string
	Slug      string
	LastRunAt *time.Time
}

type PlanOptions struct {
	Concurrency int
	BudgetMs    int
	PerClientMs int
}

// DefaultPlan mirrors the old dispatcher: 4 workers, 270 s budget, 90 s per
// client, so at most 12 clients run per tick and the rest are deferred to
// the next day. With events the budget is advisory; it caps how much work a
// single tick enqueues.
var DefaultPlan = PlanOptions{Concurrency: 4, BudgetMs: 270_000, PerClientMs: 90_000}

type Plan struct {
	Run      []ClientRun
	Deferred []ClientRun
}

// PlanDailyRun ports sync/policy.ts planDailyRun: never-synced clients first,
// then the longest waiting; capacity = waves x concurrency, at least one wave.
func PlanDailyRun(clients []ClientRun, opts PlanOptions) Plan {
	sorted := append([]ClientRun(nil), clients...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return unix(sorted[i].LastRunAt) < unix(sorted[j].LastRunAt)
	})
	waves := opts.BudgetMs / max(1, opts.PerClientMs)
	if waves < 1 {
		waves = 1
	}
	capacity := waves * max(1, opts.Concurrency)
	if capacity >= len(sorted) {
		return Plan{Run: sorted}
	}
	return Plan{Run: sorted[:capacity], Deferred: sorted[capacity:]}
}

func unix(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}
