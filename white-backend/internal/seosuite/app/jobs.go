package app

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Jobs writes the SEO_SUITE_* sync_jobs rows and enforces the 10 minute
// cooldown per (client, kind) that keeps "Perbarui" from hammering providers.
type Jobs struct {
	store domain.Store
	clk   clock.Clock
}

func NewJobs(store domain.Store, clk clock.Clock) *Jobs {
	return &Jobs{store: store, clk: clk}
}

// Start refuses when any job of that kind, whatever its status, started less
// than RefreshCooldown ago. Otherwise it inserts a RUNNING row.
func (j *Jobs) Start(ctx context.Context, clientID, kind, message string) (*domain.SyncJob, error) {
	now := j.clk.Now()
	last, err := j.store.GetLatestJob(ctx, clientID, kind)
	if err != nil {
		return nil, err
	}
	if last != nil && now.Sub(last.StartedAt) < domain.RefreshCooldown {
		return nil, apperr.New(apperr.RateLimit, msgThrottled)
	}
	job := domain.SyncJob{ID: ids.New(), ClientID: clientID, Kind: kind, Status: domain.JobRunning, StartedAt: now}
	if message != "" {
		job.Message = &message
	}
	if err := j.store.InsertJob(ctx, job); err != nil {
		return nil, err
	}
	return &job, nil
}

// Finish marks SUCCESS with a message or FAILED with the error text (cut at
// 900 characters, the start message kept).
func (j *Jobs) Finish(ctx context.Context, job *domain.SyncJob, runErr error, message string) error {
	now := j.clk.Now()
	if runErr != nil {
		return j.store.FinishJob(ctx, job.ID, domain.JobFailed, now, job.Message, ptr(truncate(errText(runErr), 900)))
	}
	return j.store.FinishJob(ctx, job.ID, domain.JobSuccess, now, &message, nil)
}

// List returns the newest jobs of the given kinds (default: every suite kind).
func (j *Jobs) List(ctx context.Context, clientID string, kinds []string, limit int32) ([]domain.SyncJob, error) {
	if len(kinds) == 0 {
		kinds = domain.JobKinds
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return j.store.ListJobs(ctx, clientID, kinds, limit)
}
