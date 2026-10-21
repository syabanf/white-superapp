package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Spec runs the scheduler as a worker-only binary (no database, no gRPC).
func Spec() pkgapp.Spec {
	return pkgapp.Spec{Name: "scheduler", NoDatabase: true, Worker: run}
}

// Schedules (UTC). Override with DAILY_CRON, PUBLISH_CRON, RETENTION_CRON.
const (
	defaultDailyCron     = "0 2 * * *"
	defaultPublishCron   = "*/5 * * * *"
	defaultRetentionCron = "30 1 * * *"
)

type Scheduler struct {
	logger  *slog.Logger
	bus     events.Publisher
	clients identityv1.AccessServiceClient
	plan    PlanOptions

	mu      sync.Mutex
	lastRun map[string]time.Time
}

func run(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		return fmt.Errorf("scheduler membutuhkan NATS_URL")
	}
	conn, err := grpcx.Dial(config.Env("IDENTITY_ADDR", "localhost:9001"))
	if err != nil {
		return err
	}
	defer conn.Close()
	s := &Scheduler{
		logger:  d.Logger,
		bus:     d.Bus,
		clients: identityv1.NewAccessServiceClient(conn),
		plan: PlanOptions{
			Concurrency: config.EnvInt("DAILY_CONCURRENCY", DefaultPlan.Concurrency),
			BudgetMs:    config.EnvInt("DAILY_BUDGET_MS", DefaultPlan.BudgetMs),
			PerClientMs: config.EnvInt("DAILY_PER_CLIENT_MS", DefaultPlan.PerClientMs),
		},
		lastRun: map[string]time.Time{},
	}

	c := cron.New(cron.WithLocation(time.UTC))
	add := func(name, spec string, fn func(context.Context) error) error {
		_, err := c.AddFunc(spec, func() {
			start := time.Now()
			if err := fn(ctx); err != nil {
				s.logger.Error("tick failed", "job", name, "err", err)
				return
			}
			s.logger.Info("tick", "job", name, "ms", time.Since(start).Milliseconds())
		})
		return err
	}
	if err := add("daily", config.Env("DAILY_CRON", defaultDailyCron), s.Daily); err != nil {
		return err
	}
	if err := add("publish", config.Env("PUBLISH_CRON", defaultPublishCron), s.Publish); err != nil {
		return err
	}
	if err := add("retention", config.Env("RETENTION_CRON", defaultRetentionCron), s.Retention); err != nil {
		return err
	}
	c.Start()
	s.logger.Info("scheduler started", "daily", config.Env("DAILY_CRON", defaultDailyCron), "publish", config.Env("PUBLISH_CRON", defaultPublishCron))
	<-ctx.Done()
	<-c.Stop().Done()
	return nil
}

// Daily fans out one event per client to analytics and seosuite. Clients
// the plan defers keep their old lastRun and sort first next time.
func (s *Scheduler) Daily(ctx context.Context) error {
	res, err := s.clients.ListAllClients(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("list clients: %w", err)
	}
	now := time.Now().UTC()
	runs := make([]ClientRun, 0, len(res.Clients))
	s.mu.Lock()
	for _, c := range res.Clients {
		r := ClientRun{ID: c.Id, Slug: c.Slug}
		if t, ok := s.lastRun[c.Id]; ok {
			tt := t
			r.LastRunAt = &tt
		}
		runs = append(runs, r)
	}
	s.mu.Unlock()
	plan := PlanDailyRun(runs, s.plan)
	for _, c := range plan.Run {
		payload := events.SyncDailyPayload{ClientID: c.ID, Slug: c.Slug, RunAt: now}
		for _, subject := range []string{events.SyncDailyAnalytics, events.SyncDailySeoSuite} {
			if err := s.bus.Publish(ctx, subject, payload); err != nil {
				return err
			}
		}
		s.mu.Lock()
		s.lastRun[c.ID] = now
		s.mu.Unlock()
	}
	if len(plan.Deferred) > 0 {
		slugs := make([]string, 0, len(plan.Deferred))
		for _, c := range plan.Deferred {
			slugs = append(slugs, c.Slug)
		}
		s.logger.Warn("daily sync deferred", "clients", slugs)
	}
	return nil
}

func (s *Scheduler) Publish(ctx context.Context) error {
	return s.bus.Publish(ctx, events.PublishTick, events.TickPayload{RunAt: time.Now().UTC()})
}

func (s *Scheduler) Retention(ctx context.Context) error {
	return s.bus.Publish(ctx, events.RetentionTick, events.TickPayload{RunAt: time.Now().UTC()})
}
