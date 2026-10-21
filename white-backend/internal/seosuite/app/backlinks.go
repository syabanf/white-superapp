package app

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/providers/seodata"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

// Backlinks is the backlink half of actions-domains.ts plus getBacklinkData.
type Backlinks struct {
	guard   *Guard
	seo     seodata.Source
	tracker *Tracker
	jobs    *Jobs
	views   *Views
	clk     clock.Clock
}

func NewBacklinks(guard *Guard, seo seodata.Source, tracker *Tracker, jobs *Jobs, views *Views, clk clock.Clock) *Backlinks {
	return &Backlinks{guard: guard, seo: seo, tracker: tracker, jobs: jobs, views: views, clk: clk}
}

// Refresh: VIEWER allowed, SEO_SUITE_BACKLINKS cooldown, message "+a / −l".
func (b *Backlinks) Refresh(ctx context.Context, userID, clientID, propertyID string) (added, lost int32, err error) {
	p, err := b.guard.Property(ctx, userID, clientID, propertyID, false)
	if err != nil {
		return 0, 0, err
	}
	job, err := b.jobs.Start(ctx, clientID, domain.JobKindBacklinks, p.SiteURL)
	if err != nil {
		return 0, 0, err
	}
	added, lost, runErr := b.tracker.RefreshBacklinks(ctx, *p, dates.Today(b.clk.Now()))
	if err := b.jobs.Finish(ctx, job, runErr, fmt.Sprintf("+%d / −%d", added, lost)); err != nil {
		return 0, 0, err
	}
	if runErr != nil {
		return 0, 0, runErr
	}
	return added, lost, nil
}

type BacklinkDataResult struct {
	*BacklinkData
	Today time.Time
	Range dates.Resolved
	Mode  seodata.Mode
}

func (b *Backlinks) Data(ctx context.Context, userID, clientID, propertyID string, rng dates.Resolved) (*BacklinkDataResult, error) {
	p, err := b.guard.Property(ctx, userID, clientID, propertyID, false)
	if err != nil {
		return nil, err
	}
	data, err := b.views.BacklinkData(ctx, p.ID, rng.Current)
	if err != nil {
		return nil, err
	}
	_, mode, err := b.seo.Provider(ctx)
	if err != nil {
		return nil, err
	}
	return &BacklinkDataResult{BacklinkData: data, Today: dates.Today(b.clk.Now()), Range: rng, Mode: mode}, nil
}
