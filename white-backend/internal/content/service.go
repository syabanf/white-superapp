// Package content wires the content service (gRPC :9011, schema content):
// editorial plans with AI drafts, the content item workflow, design tasks,
// the brand asset library, scheduling through publishing (feed, reel and
// story), monthly content reports and deadline reminders.
//
// Env: IDENTITY_ADDR (localhost:9001), INTEGRATIONS_ADDR (localhost:9002),
// ANALYTICS_ADDR (localhost:9003), PUBLISHING_ADDR (localhost:9004),
// CONTENT_UPLOAD_DIR (./.uploads/content), INTERNAL_API_SECRET (identity
// service tokens: posting as the copywriter and checking publish results
// from the worker), APP_BASE_URL (OpenRouter referer).
package content

import (
	"context"
	"embed"
	"io/fs"

	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/content/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/pdf"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/providers/ai"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "content"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "content",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

// Services builds every use case over shared deps.
func Services(d app.Deps) grpcapi.Services {
	return grpcapi.Services{
		Plans:      app.NewPlans(d),
		Items:      app.NewItems(d),
		Scheduling: app.NewScheduling(d),
		Designs:    app.NewDesigns(d),
		Assets:     app.NewAssets(d),
		Reports:    app.NewReports(d),
		Dashboard:  app.NewDashboard(d),
	}
}

// Wiring holds the deps and the clients the worker and seed reach into.
type Wiring struct {
	Deps         app.Deps
	Identity     *clients.Identity
	integrations *clients.Integrations
	conns        []*grpc.ClientConn
}

func (w *Wiring) Close() {
	for _, c := range w.conns {
		_ = c.Close()
	}
}

func Wire(d pkgapp.Deps) (*Wiring, error) {
	w := &Wiring{}
	dial := func(env, fallback string) (*grpc.ClientConn, error) {
		conn, err := grpcx.Dial(config.Env(env, fallback))
		if err != nil {
			w.Close()
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
	publishingConn, err := dial("PUBLISHING_ADDR", "localhost:9004")
	if err != nil {
		return nil, err
	}
	clk := clock.Real{}
	w.Identity = clients.NewIdentity(identityConn, config.Env("INTERNAL_API_SECRET", ""))
	w.integrations = clients.NewIntegrations(integrationsConn, clk)
	w.Deps = app.Deps{
		Store:      postgres.NewStore(d.Pool),
		Files:      storage.NewLocal(config.Env("CONTENT_UPLOAD_DIR", "")),
		Identity:   w.Identity,
		Analytics:  clients.NewAnalytics(analyticsConn),
		Publishing: clients.NewPublishing(publishingConn),
		AI:         ai.NewSource(w.integrations.Credentials, ai.Options{Referer: config.Env("APP_BASE_URL", "")}),
		Renderer:   pdf.Renderer{},
		Clock:      clk,
		Logger:     d.Logger,
	}
	return w, nil
}

func register(ctx context.Context, d pkgapp.Deps, s *grpc.Server) error {
	w, err := Wire(d)
	if err != nil {
		return err
	}
	go func() { <-ctx.Done(); w.Close() }()
	grpcapi.Register(s, Services(w.Deps))
	return nil
}

// worker applies publishing outcome events, re-checks publish results (the
// fallback for missed events) and sends deadline reminders on the ERP
// ticks, purges deleted clients and users and busts the credentials cache.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	w, err := Wire(d)
	if err != nil {
		return err
	}
	defer w.Close()
	scheduling := app.NewScheduling(w.Deps)
	jobs := app.NewJobs(w.Deps)
	subjects := []string{events.ERPTick15m, events.ERPTickDaily, events.ClientDeleted, events.UserDeleted, events.IntegrationUpdated,
		events.PublishingPostPublished, events.PublishingPostFailed}
	return d.Bus.Subscribe(ctx, "content-events", subjects, func(ctx context.Context, msg events.Message) error {
		switch msg.Subject {
		case events.ERPTick15m, events.ERPTickDaily:
			var p events.TickPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			if msg.Subject == events.ERPTick15m {
				if run, err := scheduling.Reconcile(ctx, p.RunAt, 50); err != nil {
					d.Logger.Warn("publish reconcile failed", "err", err)
				} else if run.Checked > 0 {
					d.Logger.Info("publish reconcile", "checked", run.Checked, "changed", run.Changed)
				}
			}
			if n, err := jobs.SendReminders(ctx, p.RunAt); err != nil {
				d.Logger.Warn("deadline reminders failed", "err", err)
			} else if n > 0 {
				d.Logger.Info("deadline reminders", "sent", n)
			}
			// Ticks are not retried: the next tick picks the work up.
			return nil
		case events.ClientDeleted:
			var p events.ClientDeletedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			return jobs.PurgeClient(ctx, p.ClientID)
		case events.UserDeleted:
			var p events.UserDeletedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			return jobs.ForgetUser(ctx, p.UserID)
		case events.IntegrationUpdated:
			w.integrations.Invalidate()
		case events.PublishingPostPublished, events.PublishingPostFailed:
			var p events.PublishPostPayload
			if err := msg.Decode(&p); err != nil {
				d.Logger.Warn("malformed publish outcome", "subject", msg.Subject, "err", err)
				return nil
			}
			return scheduling.OnPostOutcome(ctx, p)
		}
		return nil
	})
}
