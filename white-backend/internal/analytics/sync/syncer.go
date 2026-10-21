// Package sync holds the auth-free write paths shared by the manual
// "Sinkronkan" actions and the daily worker: social (Meta, Apify, demo),
// SEO (GSC, GA4), ads (Marketing API) and the site audit.
package sync

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/apify"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/crawler"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/googlesearch"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/metagraph"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/metamarketing"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/pagespeed"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
)

// Syncer runs the provider pulls. Providers are picked per call from the
// integrations credentials unless an override is set (tests).
type Syncer struct {
	Store        domain.Store
	Integ        domain.Integrations
	Clock        clock.Clock
	Logger       *slog.Logger
	PagespeedKey string

	Google    googlesearch.Provider
	Graph     metagraph.Provider
	Marketing metamarketing.Adapter
	Public    apify.PublicSocial
	Pagespeed func(ctx context.Context, target, strategy string) (*pagespeed.Result, error)
	Crawl     func(ctx context.Context, startURL string, opt crawler.Options) (*crawler.Result, error)
}

func New(store domain.Store, integ domain.Integrations, clk clock.Clock, logger *slog.Logger, pagespeedKey string) *Syncer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Syncer{Store: store, Integ: integ, Clock: clk, Logger: logger, PagespeedKey: pagespeedKey}
}

func (s *Syncer) now() time.Time { return s.Clock.Now() }

// token resolves a connection's access token; "" means "no real
// credentials" (no connection, provider not configured, connection gone).
func (s *Syncer) token(ctx context.Context, connectionID *string) (string, error) {
	if connectionID == nil || *connectionID == "" {
		return "", nil
	}
	return s.Integ.AccessToken(ctx, *connectionID)
}

func (s *Syncer) configured(ctx context.Context, key string) bool {
	c, err := s.Integ.Credentials(ctx, key)
	return err == nil && c.Configured
}

func (s *Syncer) MetaConfigured(ctx context.Context) bool {
	return s.configured(ctx, domain.IntegrationMeta)
}

func (s *Syncer) GoogleConfigured(ctx context.Context) bool {
	return s.configured(ctx, domain.IntegrationGoogle)
}

func (s *Syncer) ApifyConfigured(ctx context.Context) bool {
	return s.configured(ctx, domain.IntegrationApify)
}

func (s *Syncer) graph(ctx context.Context) metagraph.Provider {
	if s.Graph != nil {
		return s.Graph
	}
	return metagraph.Pick(s.MetaConfigured(ctx))
}

func (s *Syncer) google(ctx context.Context) googlesearch.Provider {
	if s.Google != nil {
		return s.Google
	}
	return googlesearch.Pick(s.GoogleConfigured(ctx))
}

func (s *Syncer) marketing(ctx context.Context, token string) metamarketing.Adapter {
	if s.Marketing != nil {
		return s.Marketing
	}
	return metamarketing.Pick(s.MetaConfigured(ctx), token)
}

// PublicSocial is the Apify adapter (real when the token is set).
func (s *Syncer) PublicSocial(ctx context.Context) apify.PublicSocial {
	if s.Public != nil {
		return s.Public
	}
	c, _ := s.Integ.Credentials(ctx, domain.IntegrationApify)
	return apify.Pick(c.Secret, c.PublicID)
}

func (s *Syncer) pagespeed(ctx context.Context, target, strategy string) (*pagespeed.Result, error) {
	if s.Pagespeed != nil {
		return s.Pagespeed(ctx, target, strategy)
	}
	return pagespeed.New(s.PagespeedKey).Run(ctx, target, strategy)
}

func (s *Syncer) crawl(ctx context.Context, start string, opt crawler.Options) (*crawler.Result, error) {
	if s.Crawl != nil {
		return s.Crawl(ctx, start, opt)
	}
	return (&crawler.Crawler{}).Crawl(ctx, start, opt)
}

// errText is what the old code stored from `e.message`: the user-facing
// message of provider errors, the plain text of anything else.
func errText(err error) string {
	var e *apperr.Error
	if errors.As(err, &e) {
		return e.Message
	}
	return err.Error()
}

func strp(s string) *string { return &s }

func intp(n int) *int { return &n }

func truncate(s string, n int) *string {
	if len(s) > n {
		s = s[:n]
	}
	return &s
}

// job writes one finished sync_jobs row.
func (s *Syncer) job(ctx context.Context, clientID, kind string, status domain.JobStatus, message, errMsg *string) error {
	now := s.now()
	_, err := s.Store.InsertSyncJob(ctx, domain.SyncJob{ClientID: clientID, Kind: kind, Status: status, StartedAt: now, FinishedAt: &now, Message: message, Error: errMsg})
	return err
}
