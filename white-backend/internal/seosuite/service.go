// Package seosuite wires the SEO suite service: keyword research, rank
// tracking, backlinks and competitor domains on top of DataForSEO, Apify or
// the deterministic mock.
package seosuite

import (
	"context"
	"embed"
	"io/fs"
	"strconv"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/providers/seodata"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/app"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "seosuite"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "seosuite",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

// wiring holds everything both the gRPC server and the worker need.
type wiring struct {
	conns        []*grpc.ClientConn
	integrations *clients.Integrations
	keywords     *app.Keywords
	research     *app.Research
	backlinks    *app.Backlinks
	competitors  *app.Competitors
	reports      *app.Reports
	daily        *app.Daily
}

func (w *wiring) close() {
	for _, c := range w.conns {
		_ = c.Close()
	}
}

// Env: IDENTITY_ADDR, INTEGRATIONS_ADDR, ANALYTICS_ADDR, DATAFORSEO_USD_IDR.
func wire(d pkgapp.Deps) (*wiring, error) {
	w := &wiring{}
	dial := func(env, def string) (*grpc.ClientConn, error) {
		conn, err := grpcx.Dial(config.Env(env, def))
		if err != nil {
			w.close()
			return nil, err
		}
		w.conns = append(w.conns, conn)
		return conn, nil
	}
	identityConn, err := dial("IDENTITY_ADDR", "localhost:9001")
	if err != nil {
		return nil, err
	}
	integrationsConn, err := dial("INTEGRATIONS_ADDR", "localhost:9002")
	if err != nil {
		return nil, err
	}
	analyticsConn, err := dial("ANALYTICS_ADDR", "localhost:9003")
	if err != nil {
		return nil, err
	}
	clk := clock.Real{}
	identity := clients.NewIdentity(identityConn, clk)
	w.integrations = clients.NewIntegrations(integrationsConn, clk)
	analytics := clients.NewAnalytics(analyticsConn)

	rate, err := strconv.ParseFloat(config.Env("DATAFORSEO_USD_IDR", "16000"), 64)
	if err != nil || rate <= 0 {
		d.Logger.Warn("DATAFORSEO_USD_IDR tidak valid, memakai 16000")
		rate = 16000
	}
	seo := seodata.NewSource(w.integrations.Credentials, seodata.NewPicker(seodata.Options{USDToIDR: rate, Clock: clk}))

	store := postgres.NewStore(d.Pool)
	guard := app.NewGuard(identity, analytics)
	jobs := app.NewJobs(store, clk)
	tracker := app.NewTracker(store, seo)
	views := app.NewViews(store)
	w.keywords = app.NewKeywords(store, guard, seo, tracker, jobs, views, clk)
	w.research = app.NewResearch(store, guard, seo, clk)
	w.backlinks = app.NewBacklinks(guard, seo, tracker, jobs, views, clk)
	w.competitors = app.NewCompetitors(store, guard, seo, tracker, jobs, clk)
	w.reports = app.NewReports(views)
	w.daily = app.NewDaily(store, analytics, tracker, identity)
	return w, nil
}

func register(ctx context.Context, d pkgapp.Deps, s *grpc.Server) error {
	w, err := wire(d)
	if err != nil {
		return err
	}
	context.AfterFunc(ctx, w.close)
	grpcapi.Register(s, grpcapi.Services{
		Keywords:    w.keywords,
		Research:    w.research,
		Backlinks:   w.backlinks,
		Competitors: w.competitors,
		Reports:     w.reports,
	})
	return nil
}

// worker consumes the daily SEO suite tick, the retention tick, client
// deletions and integration updates (credential cache bust).
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	w, err := wire(d)
	if err != nil {
		return err
	}
	defer w.close()

	subs := []struct {
		durable  string
		subjects []string
		handle   events.Handler
	}{
		{"seosuite-daily", []string{events.SyncDailySeoSuite}, func(ctx context.Context, msg events.Message) error {
			var p events.SyncDailyPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			res, err := w.daily.Run(ctx, p.ClientID, p.Slug, p.RunAt)
			if err != nil {
				return err
			}
			d.Logger.Info("seosuite daily", "client", p.Slug, "properties", res.Properties, "ranks", res.Ranks, "backlinks", res.Backlinks, "domains", res.Domains)
			return nil
		}},
		{"seosuite-retention", []string{events.RetentionTick}, func(ctx context.Context, msg events.Message) error {
			var p events.TickPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			res, err := w.daily.Retention(ctx, p.RunAt)
			if err != nil {
				return err
			}
			d.Logger.Info("retention", "rankSnapshots", res.RankSnapshots, "backlinkSnapshots", res.BacklinkSnapshots, "domainSnapshots", res.DomainSnapshots, "syncJobs", res.SyncJobs)
			return nil
		}},
		{"seosuite-client-deleted", []string{events.ClientDeleted}, func(ctx context.Context, msg events.Message) error {
			var p events.ClientDeletedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			return w.daily.PurgeClient(ctx, p.ClientID)
		}},
		{"seosuite-integrations", []string{events.IntegrationUpdated}, func(context.Context, events.Message) error {
			w.integrations.Invalidate()
			return nil
		}},
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, len(subs))
	for _, s := range subs {
		go func() { errCh <- d.Bus.Subscribe(runCtx, s.durable, s.subjects, s.handle) }()
	}
	for range subs {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}
