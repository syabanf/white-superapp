package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Daily is daily.ts (without the audit step, which analytics owns now) plus
// the retention and client-purge handlers of the worker.
type Daily struct {
	store    domain.Store
	props    PropertySource
	tracker  *Tracker
	notifier Notifier
}

func NewDaily(store domain.Store, props PropertySource, tracker *Tracker, notifier Notifier) *Daily {
	return &Daily{store: store, props: props, tracker: tracker, notifier: notifier}
}

type DailyResult struct {
	Properties int32
	Ranks      int32
	Backlinks  int32
	Domains    int32
}

func posText(p *int32) string {
	if p == nil {
		return "–"
	}
	return strconv.Itoa(int(*p))
}

// Run processes every property of the client: one SEO_SUITE_DAILY job per
// property around ranks, backlinks and domains; RANK_DROP and BACKLINK_LOST
// notifications to the client audience; then the 30-day research purge.
func (d *Daily) Run(ctx context.Context, clientID, slug string, now time.Time) (DailyResult, error) {
	date := dates.Today(now)
	var out DailyResult
	props, err := d.props.ListProperties(ctx, clientID)
	if err != nil {
		return out, err
	}
	for _, p := range props {
		out.Properties++
		job := domain.SyncJob{ID: ids.New(), ClientID: p.ClientID, Kind: domain.JobKindDaily, Status: domain.JobRunning, StartedAt: now, Message: ptr(p.SiteURL)}
		if err := d.store.InsertJob(ctx, job); err != nil {
			return out, err
		}
		var notes, errs []string
		href := func(page string) string { return fmt.Sprintf("/clients/%s/seo/%s", slug, page) }

		updated, drops, err := d.tracker.TrackRanks(ctx, p, date, nil)
		if err != nil {
			errs = append(errs, "ranks: "+errText(err))
		} else {
			out.Ranks += updated
			notes = append(notes, countMsg(updated, "peringkat"))
			if len(drops) > 0 {
				sample := make([]string, 0, 3)
				for _, dr := range drops[:min(3, len(drops))] {
					sample = append(sample, fmt.Sprintf("%s (%s→%s)", dr.Keyword, posText(dr.Prev), posText(dr.Cur)))
				}
				_ = d.notifier.Notify(ctx, Notification{
					ClientID: p.ClientID, Type: "RANK_DROP", Title: msgRankDropTitle,
					Body: fmt.Sprintf("%d kata kunci turun tajam sejak kemarin: %s", len(drops), strings.Join(sample, ", ")),
					Href: href("rank"),
				})
			}
		}

		added, lost, err := d.tracker.RefreshBacklinks(ctx, p, date)
		if err != nil {
			errs = append(errs, "backlinks: "+errText(err))
		} else {
			out.Backlinks++
			notes = append(notes, fmt.Sprintf("backlink +%d/−%d", added, lost))
			if lost > domain.BacklinkLostThreshold {
				_ = d.notifier.Notify(ctx, Notification{
					ClientID: p.ClientID, Type: "BACKLINK_LOST", Title: msgBacklinkLostTitle,
					Body: fmt.Sprintf("%d backlink hilang dalam sehari terakhir.", lost),
					Href: href("backlinks"),
				})
			}
		}

		n, err := d.tracker.RefreshDomains(ctx, p, date, nil)
		if err != nil {
			errs = append(errs, "domains: "+errText(err))
		} else {
			out.Domains += n
			notes = append(notes, countMsg(n, "domain"))
		}

		status := domain.JobSuccess
		if len(errs) > 0 && len(notes) == 0 {
			status = domain.JobFailed
		}
		var message, errField *string
		if len(notes) > 0 {
			message = ptr(strings.Join(notes, " · "))
		}
		if len(errs) > 0 {
			errField = ptr(truncate(strings.Join(errs, " | "), 900))
		}
		if err := d.store.FinishJob(ctx, job.ID, status, time.Now().UTC(), message, errField); err != nil {
			return out, err
		}
	}
	if _, err := d.store.DeleteResearchBefore(ctx, dates.AddDays(date, -domain.ResearchPurgeDays)); err != nil {
		return out, err
	}
	return out, nil
}

type RetentionResult struct {
	RankSnapshots     int64
	BacklinkSnapshots int64
	DomainSnapshots   int64
	SyncJobs          int64
}

// Retention drops snapshots older than 760 days and sync jobs older than 90.
func (d *Daily) Retention(ctx context.Context, now time.Time) (RetentionResult, error) {
	var out RetentionResult
	var err error
	snapshotCutoff := now.Add(-domain.RetentionSnapshotDays * 24 * time.Hour)
	if out.RankSnapshots, err = d.store.DeleteRankSnapshotsBefore(ctx, snapshotCutoff); err != nil {
		return out, err
	}
	if out.BacklinkSnapshots, err = d.store.DeleteBacklinkSnapshotsBefore(ctx, snapshotCutoff); err != nil {
		return out, err
	}
	if out.DomainSnapshots, err = d.store.DeleteDomainSnapshotsBefore(ctx, snapshotCutoff); err != nil {
		return out, err
	}
	if out.SyncJobs, err = d.store.DeleteJobsBefore(ctx, now.Add(-domain.RetentionSyncJobDays*24*time.Hour)); err != nil {
		return out, err
	}
	return out, nil
}

// PurgeClient handles white.client.deleted.
func (d *Daily) PurgeClient(ctx context.Context, clientID string) error {
	return d.store.PurgeClient(ctx, clientID)
}
