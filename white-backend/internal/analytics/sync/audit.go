package sync

import (
	"context"
	"fmt"
	"strings"
	stdsync "sync"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/crawler"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/pagespeed"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// AuditCooldown is the minimum gap between two manual audits of a property.
const AuditCooldown = 10 * time.Minute

// AuditTarget is the override URL, https://x/ for sc-domain properties, or the site URL.
func AuditTarget(siteURL, override string) string {
	if override != "" {
		return override
	}
	if strings.HasPrefix(siteURL, "sc-domain:") {
		return "https://" + strings.TrimPrefix(siteURL, "sc-domain:") + "/"
	}
	return siteURL
}

// ShouldAudit is the scheduled cadence rule: NONE never, DAILY after 20 h,
// WEEKLY after 6 days, and always when there is no previous audit.
func ShouldAudit(c domain.AuditCadence, last *time.Time, now time.Time) bool {
	switch c {
	case domain.CadenceDaily:
		return last == nil || now.Sub(*last) > 20*time.Hour
	case domain.CadenceWeekly:
		return last == nil || now.Sub(*last) > 6*24*time.Hour
	}
	return false
}

type AuditResult struct {
	Target       string
	AllFailed    bool
	Parts        []string
	ErrorParts   []string
	Audits       int
	PagesCrawled int
	Issues       int
}

// RunAuditCore runs PageSpeed mobile + desktop and the crawler concurrently
// and persists audits, crawl pages, issues and the SEO_AUDIT job.
func (s *Syncer) RunAuditCore(ctx context.Context, p domain.SeoProperty, override string) (*AuditResult, error) {
	target := AuditTarget(p.SiteURL, override)
	started := s.now()
	job, err := s.Store.InsertSyncJob(ctx, domain.SyncJob{ClientID: p.ClientID, Kind: domain.KindSeoAudit, Status: domain.JobRunning, StartedAt: started, Message: strp(target)})
	if err != nil {
		return nil, err
	}
	crawlID := ids.New()
	if err := s.Store.InsertCrawl(ctx, domain.SeoCrawl{ID: crawlID, PropertyID: p.ID, StartedAt: started, Status: domain.JobRunning, MaxPages: 30}); err != nil {
		return nil, err
	}

	var (
		wg       stdsync.WaitGroup
		psi      [2]*pagespeed.Result
		psiErr   [2]error
		crawlRes *crawler.Result
		crawlErr error
	)
	for i, strategy := range []string{"mobile", "desktop"} {
		wg.Add(1)
		go func(i int, strategy string) {
			defer wg.Done()
			psi[i], psiErr[i] = s.pagespeed(ctx, target, strategy)
		}(i, strategy)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		crawlRes, crawlErr = s.crawl(ctx, target, crawler.Options{MaxPages: 30, TimeBudget: 45 * time.Second})
	}()
	wg.Wait()

	res := &AuditResult{Target: target}
	for i := range psi {
		if psiErr[i] != nil {
			res.ErrorParts = append(res.ErrorParts, errText(psiErr[i]))
			continue
		}
		r := psi[i]
		strategy := domain.Mobile
		if r.Strategy == "desktop" {
			strategy = domain.Desktop
		}
		a := domain.SeoAudit{PropertyID: p.ID, URL: target, Strategy: strategy, RunAt: s.now(), Performance: r.Scores.Performance, Seo: r.Scores.Seo, Accessibility: r.Scores.Accessibility, BestPractices: r.Scores.BestPractices, LcpMs: r.Metrics.LcpMs, InpMs: r.Metrics.InpMs, Cls: r.Metrics.Cls, FcpMs: r.Metrics.FcpMs, TtfbMs: r.Metrics.TtfbMs, TbtMs: r.Metrics.TbtMs, SpeedIndexMs: r.Metrics.SpeedIndexMs, Raw: r.Raw}
		if err := s.Store.InsertAudit(ctx, a); err != nil {
			return nil, err
		}
		res.Audits++
	}

	if crawlErr == nil {
		res.PagesCrawled = crawlRes.PagesCrawled
		res.Issues = len(crawlRes.Issues)
		pages := make([]domain.SeoCrawlPage, 0, len(crawlRes.Pages))
		for _, pg := range crawlRes.Pages {
			pages = append(pages, domain.SeoCrawlPage{CrawlID: crawlID, URL: pg.URL, StatusCode: pg.StatusCode, Title: pg.Title, MetaDescription: pg.MetaDescription, H1Count: pg.H1Count, WordCount: pg.WordCount, Canonical: pg.Canonical, Indexable: pg.Indexable, ImagesTotal: pg.ImagesTotal, ImagesMissingAlt: pg.ImagesMissingAlt, InternalLinks: pg.InternalLinks, ExternalLinks: pg.ExternalLinks, LoadMs: pg.LoadMs, HasJsonLd: pg.HasJsonLd, HasHreflang: pg.HasHreflang})
		}
		if err := s.Store.InsertCrawlPages(ctx, pages); err != nil {
			return nil, err
		}
		issues := make([]domain.SeoIssue, 0, len(crawlRes.Issues))
		for _, is := range crawlRes.Issues {
			issues = append(issues, domain.SeoIssue{CrawlID: crawlID, Severity: domain.IssueSeverity(is.Severity), Code: is.Code, Message: is.Message, URL: is.URL})
		}
		if len(issues) > 0 {
			if err := s.Store.InsertIssues(ctx, issues); err != nil {
				return nil, err
			}
		}
		if err := s.Store.UpdateCrawl(ctx, crawlID, domain.JobSuccess, s.now(), crawlRes.PagesCrawled, nil); err != nil {
			return nil, err
		}
	} else {
		msg := errText(crawlErr)
		res.ErrorParts = append(res.ErrorParts, msg)
		if err := s.Store.UpdateCrawl(ctx, crawlID, domain.JobFailed, s.now(), 0, &msg); err != nil {
			return nil, err
		}
	}

	res.AllFailed = res.Audits == 0 && crawlErr != nil
	if res.Audits > 0 {
		res.Parts = append(res.Parts, fmt.Sprintf("PageSpeed %d/2 strategi", res.Audits))
	}
	if crawlErr == nil {
		res.Parts = append(res.Parts, fmt.Sprintf("Crawl %d halaman, %d isu", res.PagesCrawled, res.Issues))
	}
	status := domain.JobSuccess
	if res.AllFailed {
		status = domain.JobFailed
	}
	var message, errMsg *string
	if len(res.Parts) > 0 {
		message = strp(strings.Join(res.Parts, " · "))
	}
	if len(res.ErrorParts) > 0 {
		errMsg = truncate(strings.Join(res.ErrorParts, " | "), 900)
	}
	if err := s.Store.UpdateSyncJob(ctx, job.ID, status, s.now(), message, errMsg); err != nil {
		return nil, err
	}
	return res, nil
}
