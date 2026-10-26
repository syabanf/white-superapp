package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
)

type DailySummary struct {
	Kind    string
	Status  domain.JobStatus
	Mode    string
	Message string
}

// RunDailySync is the per-client daily job: social (Meta, then Apify for
// competitors and unconnected accounts, else a mock job), SEO and ads over
// the last 30 days. Failures are captured per block.
func (s *Syncer) RunDailySync(ctx context.Context, clientID string) []DailySummary {
	var out []DailySummary
	apifyOn := s.ApifyConfigured(ctx)

	updated, posts, real := 0, 0, false
	var failed []string
	var fatal error
	for _, platform := range domain.Platforms {
		if platform != domain.TikTok {
			m, err := s.SyncSocialMeta(ctx, clientID, platform, 30)
			if err != nil {
				fatal = err
				break
			}
			updated += m.Updated
			posts += m.Posts
			for _, f := range m.Failed {
				failed = append(failed, fmt.Sprintf("%s @%s: %s", strings.ToLower(string(platform)), f.Username, f.Error))
			}
			real = real || m.Updated > 0
		}
		if apifyOn {
			r, err := s.SyncSocialPublic(ctx, clientID, platform, true, 0)
			if err != nil {
				fatal = err
				break
			}
			updated += r.Updated
			posts += r.Posts
			for _, f := range r.Failed {
				failed = append(failed, fmt.Sprintf("%s @%s: %s", strings.ToLower(string(platform)), f.Username, f.Error))
			}
			real = real || r.Updated > 0
		}
	}
	switch {
	case fatal != nil:
		out = append(out, DailySummary{Kind: domain.KindSocialSnapshot, Status: domain.JobFailed, Mode: "real", Message: errText(fatal)})
	case real || len(failed) > 0:
		status := domain.JobSuccess
		if updated == 0 && len(failed) > 0 {
			status = domain.JobFailed
		}
		msg := fmt.Sprintf("%d akun, %d post", updated, posts)
		var errMsg *string
		if len(failed) > 0 {
			errMsg = truncate(strings.Join(failed, "; "), 1000)
		}
		if err := s.job(ctx, clientID, domain.KindSocialSnapshot, status, strp(msg), errMsg); err != nil {
			s.Logger.Warn("sync job", "err", err)
		}
		out = append(out, DailySummary{Kind: domain.KindSocialSnapshot, Status: status, Mode: "real", Message: msg})
	default:
		if err := s.job(ctx, clientID, domain.KindSocialSnapshot, domain.JobSuccess, strp("mock"), nil); err != nil {
			s.Logger.Warn("sync job", "err", err)
		}
		out = append(out, DailySummary{Kind: domain.KindSocialSnapshot, Status: domain.JobSuccess, Mode: "demo", Message: "mock"})
	}

	if r, err := s.SyncSeoForClient(ctx, clientID, 30); err != nil {
		out = append(out, DailySummary{Kind: domain.KindSeoGSC, Status: domain.JobFailed, Mode: "real", Message: errText(err)})
	} else {
		out = append(out, summarize(domain.KindSeoGSC, r.Real, r.Properties, "properti", seoErr(r.Results)))
	}
	if r, err := s.SyncAdsForClient(ctx, clientID, 30); err != nil {
		out = append(out, DailySummary{Kind: domain.KindAdsInsights, Status: domain.JobFailed, Mode: "real", Message: errText(err)})
	} else {
		out = append(out, summarize(domain.KindAdsInsights, r.Real, r.Accounts, "akun", adsErr(r.Results)))
	}
	return out
}

func seoErr(rs []SeoResult) string {
	for _, r := range rs {
		if r.Err != "" {
			return r.Err
		}
	}
	return ""
}

func adsErr(rs []AdsResult) string {
	for _, r := range rs {
		if r.Err != "" {
			return r.Err
		}
	}
	return ""
}

func summarize(kind string, real, total int, unit, firstErr string) DailySummary {
	d := DailySummary{Kind: kind, Status: domain.JobSuccess, Mode: "demo", Message: "mock"}
	if real > 0 {
		d.Mode = "real"
		d.Message = fmt.Sprintf("%d/%d %s", real, total, unit)
	}
	if firstErr != "" {
		d.Status = domain.JobFailed
		d.Message = firstErr
	}
	return d
}

// RunScheduledAudits re-runs the audit of every property whose cadence is due.
func (s *Syncer) RunScheduledAudits(ctx context.Context, clientID string) (int, error) {
	props, err := s.Store.ListSeoProperties(ctx, clientID)
	if err != nil {
		return 0, err
	}
	ran := 0
	for _, p := range props {
		last, err := s.Store.LatestAuditAny(ctx, p.ID)
		if err != nil {
			return ran, err
		}
		var lastAt *time.Time
		if last != nil {
			lastAt = &last.RunAt
		}
		if !ShouldAudit(p.AuditCadence, lastAt, s.now()) {
			continue
		}
		res, err := s.RunAuditCore(ctx, p, "")
		if err != nil {
			s.Logger.Warn("scheduled audit", "property", p.ID, "err", err)
			continue
		}
		if !res.AllFailed {
			ran++
		}
	}
	return ran, nil
}
