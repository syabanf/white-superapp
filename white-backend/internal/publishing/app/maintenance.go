package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

// Maintenance handles lifecycle events and retention.
type Maintenance struct{ Deps }

func NewMaintenance(d Deps) *Maintenance { return &Maintenance{d} }

// Retention keeps PUBLISH sync jobs for 90 days (sync/policy.ts).
const RetentionSyncJobs = 90 * 24 * time.Hour

// PurgeClient removes every row and stored file of a deleted client.
func (m *Maintenance) PurgeClient(ctx context.Context, clientID string) error {
	if _, err := m.Store.DeletePostsByClient(ctx, clientID); err != nil {
		return err
	}
	if _, err := m.Store.DeleteMediaByClient(ctx, clientID); err != nil {
		return err
	}
	if _, err := m.Store.DeleteSyncJobsByClient(ctx, clientID); err != nil {
		return err
	}
	if err := m.Storage.DeleteClient(ctx, clientID); err != nil {
		m.log().Warn("client files not removed", "client", clientID, "err", err)
	}
	return nil
}

// ForgetUser clears author, approver, uploader and commenter references.
func (m *Maintenance) ForgetUser(ctx context.Context, userID string) error {
	return m.Store.NullUserRefs(ctx, userID)
}

func (m *Maintenance) RunRetention(ctx context.Context, now time.Time) (int64, error) {
	return m.Store.DeleteSyncJobsBefore(ctx, now.Add(-RetentionSyncJobs))
}

// Reports serves the internal report-data RPCs.
type Reports struct{ Deps }

func NewReports(d Deps) *Reports { return &Reports{d} }

// PublishingSummary counts published targets in [from, to+1d) and the
// failed targets planned in that window. Engagement stats need synced
// posts from analytics, which exposes no post lookup yet, so the mode is
// always "recent".
func (r *Reports) PublishingSummary(ctx context.Context, clientID string, rng dates.Range) (*domain.PublishingSummary, error) {
	from, to := rng.From, dates.AddDays(rng.To, 1)
	published, err := r.Store.ListPublishedTargets(ctx, clientID, from, to)
	if err != nil {
		return nil, err
	}
	failed, err := r.Store.CountFailedTargets(ctx, clientID, from, to)
	if err != nil {
		return nil, err
	}
	s := domain.SummarizePublishing(published, failed, 5)
	return &s, nil
}
