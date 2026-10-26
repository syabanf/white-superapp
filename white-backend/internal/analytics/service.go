// Package analytics wires the analytics service: social, SEO (GSC/GA4,
// audits, crawls), Meta Ads, the overview, their sync jobs and the report
// data other services compose.
//
// Env: IDENTITY_ADDR, INTEGRATIONS_ADDR, PAGESPEED_API_KEY (optional).
package analytics

import (
	"context"
	"embed"
	"io/fs"
	stdsync "sync"
	_ "time/tzdata" // the posting heatmap needs Asia/Jakarta in slim containers

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/app"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/sync"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "analytics"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "analytics",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

type wiring struct {
	deps         app.Deps
	integrations *clients.Integrations
}

var (
	once     stdsync.Once
	shared   *wiring
	wiredErr error
)

// wire builds the dependencies once per process so `serve` (gRPC + worker)
// shares one credential cache, which integration.updated then busts.
func wire(d pkgapp.Deps) (*wiring, error) {
	once.Do(func() {
		identityConn, err := grpcx.Dial(config.Env("IDENTITY_ADDR", "localhost:9001"))
		if err != nil {
			wiredErr = err
			return
		}
		integrationsConn, err := grpcx.Dial(config.Env("INTEGRATIONS_ADDR", "localhost:9002"))
		if err != nil {
			wiredErr = err
			return
		}
		store := postgres.NewStore(d.Pool)
		integ := clients.NewIntegrations(integrationsConn)
		clk := clock.Real{}
		shared = &wiring{
			integrations: integ,
			deps: app.Deps{
				Store:  store,
				Access: clients.NewIdentity(identityConn),
				Integ:  integ,
				Syncer: sync.New(store, integ, clk, d.Logger, config.Env("PAGESPEED_API_KEY", "")),
				Clock:  clk,
			},
		}
	})
	return shared, wiredErr
}

func register(_ context.Context, d pkgapp.Deps, s *grpc.Server) error {
	w, err := wire(d)
	if err != nil {
		return err
	}
	grpcapi.Register(s, grpcapi.Services{
		Sources:   app.NewSources(w.deps),
		Overviews: app.NewOverviews(w.deps),
		Social:    app.NewSocial(w.deps),
		Seo:       app.NewSeo(w.deps),
		Ads:       app.NewAds(w.deps),
		Reports:   app.NewReports(w.deps),
		Clock:     w.deps.Clock,
	})
	return nil
}

// worker runs the daily sync (then cadence audits), creates the SEO
// property of new clients, purges deleted clients, prunes old sync jobs and
// busts the credential cache.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	w, err := wire(d)
	if err != nil {
		return err
	}
	sources := app.NewSources(w.deps)
	syncer := w.deps.Syncer
	subjects := []string{events.SyncDailyAnalytics, events.ClientCreated, events.ClientDeleted, events.RetentionTick, events.IntegrationUpdated}
	return d.Bus.Subscribe(ctx, "analytics-events", subjects, func(ctx context.Context, msg events.Message) error {
		switch msg.Subject {
		case events.SyncDailyAnalytics:
			var p events.SyncDailyPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			summary := syncer.RunDailySync(ctx, p.ClientID)
			audits, err := syncer.RunScheduledAudits(ctx, p.ClientID)
			if err != nil {
				return err
			}
			d.Logger.Info("daily sync", "client", p.Slug, "blocks", len(summary), "audits", audits)
		case events.ClientCreated:
			var p events.ClientCreatedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			return sources.OnClientCreated(ctx, p.ClientID, p.WebsiteURL, p.Modules)
		case events.ClientDeleted:
			var p events.ClientDeletedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			return sources.PurgeClient(ctx, p.ClientID)
		case events.RetentionTick:
			n, err := sources.RunRetention(ctx)
			if err != nil {
				return err
			}
			d.Logger.Info("retention", "syncJobs", n)
		case events.IntegrationUpdated:
			w.integrations.Bust()
		}
		return nil
	})
}
