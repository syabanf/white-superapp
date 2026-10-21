// Package publishing wires the publishing service: media library, composer
// and approvals, scheduled publishing to Instagram and Facebook, calendar
// and report data.
//
// Env: UPLOAD_DIR (default ./.uploads), APP_BASE_URL (public media URLs),
// IDENTITY_ADDR, INTEGRATIONS_ADDR, ANALYTICS_ADDR.
package publishing

import (
	"context"
	"embed"
	"io/fs"
	_ "time/tzdata" // client timezones must resolve inside slim containers

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/publishing/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/adapter/providers/publisher"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/app"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "publishing"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "publishing",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

type wiring struct {
	deps         app.Deps
	integrations *clients.Integrations
	closers      []func() error
}

func (w *wiring) close() {
	for _, c := range w.closers {
		_ = c()
	}
}

func wire(d pkgapp.Deps) (*wiring, error) {
	w := &wiring{}
	dial := func(env, fallback string) (*grpc.ClientConn, error) {
		conn, err := grpcx.Dial(config.Env(env, fallback))
		if err != nil {
			w.close()
			return nil, err
		}
		w.closers = append(w.closers, conn.Close)
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
	w.integrations = clients.NewIntegrations(integrationsConn, d.Logger)
	w.deps = app.Deps{
		Store:        postgres.NewStore(d.Pool),
		Storage:      storage.NewLocal(config.Env("UPLOAD_DIR", ".uploads"), config.Env("APP_BASE_URL", "")),
		Identity:     clients.NewIdentity(identityConn),
		Integrations: w.integrations,
		Analytics:    clients.NewAnalytics(analyticsConn),
		Real:         publisher.NewReal(),
		Mock:         publisher.Mock{},
		Clock:        clock.Real{},
		Logger:       d.Logger,
	}
	return w, nil
}

func register(ctx context.Context, d pkgapp.Deps, s *grpc.Server) error {
	w, err := wire(d)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		w.close()
	}()
	publishing := app.NewPublishing(w.deps)
	grpcapi.Register(s, grpcapi.Services{
		Media:   app.NewMedia(w.deps),
		Posts:   app.NewPosts(w.deps, publishing),
		Queries: app.NewQueries(w.deps),
		Reports: app.NewReports(w.deps),
	})
	return nil
}

// worker publishes due posts on every publish tick, purges deleted clients
// and users, prunes old sync jobs and busts credential caches.
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
	publishing := app.NewPublishing(w.deps)
	maintenance := app.NewMaintenance(w.deps)
	subjects := []string{events.PublishTick, events.ClientDeleted, events.UserDeleted, events.RetentionTick, events.IntegrationUpdated}
	return d.Bus.Subscribe(ctx, "publishing-events", subjects, func(ctx context.Context, msg events.Message) error {
		switch msg.Subject {
		case events.PublishTick:
			var p events.TickPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			run, err := publishing.PublishDuePosts(ctx, p.RunAt, 20)
			if err != nil {
				return err
			}
			d.Logger.Info("publish tick", "attempted", run.Attempted, "published", run.Published, "failed", run.Failed)
		case events.ClientDeleted:
			var p events.ClientDeletedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			return maintenance.PurgeClient(ctx, p.ClientID)
		case events.UserDeleted:
			var p events.UserDeletedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			return maintenance.ForgetUser(ctx, p.UserID)
		case events.RetentionTick:
			var p events.TickPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			n, err := maintenance.RunRetention(ctx, p.RunAt)
			if err != nil {
				return err
			}
			d.Logger.Info("retention", "deleted", n)
		case events.IntegrationUpdated:
			w.integrations.Bust()
		}
		return nil
	})
}
