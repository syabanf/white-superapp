// Package reports wires the reports service: saved reports, AI strategic
// summaries (OpenRouter or the template mock), report sections, branded PDF
// rendering and the public share dashboard data.
//
// Env: IDENTITY_ADDR, INTEGRATIONS_ADDR, ANALYTICS_ADDR, PUBLISHING_ADDR,
// SEOSUITE_ADDR, APP_BASE_URL (OpenRouter HTTP-Referer). GRPC_ADDR is :9006
// in docker.
package reports

import (
	"context"
	"embed"
	"io/fs"
	stdsync "sync"
	_ "time/tzdata" // "generated on" is printed in Asia/Jakarta

	"github.com/syabanf/white-superapp/white-backend/internal/reports/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/reports/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/adapter/pdf"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/app"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "reports"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "reports",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

type wiring struct {
	reports      *app.Reports
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
		conns := map[string]*grpc.ClientConn{}
		for env, def := range map[string]string{
			"IDENTITY_ADDR":     "localhost:9001",
			"INTEGRATIONS_ADDR": "localhost:9002",
			"ANALYTICS_ADDR":    "localhost:9003",
			"PUBLISHING_ADDR":   "localhost:9004",
			"SEOSUITE_ADDR":     "localhost:9005",
		} {
			conn, err := grpcx.Dial(config.Env(env, def))
			if err != nil {
				wiredErr = err
				return
			}
			conns[env] = conn
		}
		clk := clock.Real{}
		integ := clients.NewIntegrations(conns["INTEGRATIONS_ADDR"], clk)
		shared = &wiring{
			integrations: integ,
			reports: app.New(app.Deps{
				Store:      postgres.NewStore(d.Pool),
				Identity:   clients.NewIdentity(conns["IDENTITY_ADDR"], clk),
				Analytics:  clients.NewAnalytics(conns["ANALYTICS_ADDR"]),
				Publishing: clients.NewPublishing(conns["PUBLISHING_ADDR"]),
				SeoSuite:   clients.NewSeoSuite(conns["SEOSUITE_ADDR"]),
				Insights:   openrouter.NewSource(integ.Credentials, openrouter.Options{Referer: config.Env("APP_BASE_URL", "http://localhost:3000")}),
				Render:     pdf.Render,
				Clock:      clk,
			}),
		}
	})
	return shared, wiredErr
}

func register(_ context.Context, d pkgapp.Deps, s *grpc.Server) error {
	w, err := wire(d)
	if err != nil {
		return err
	}
	grpcapi.Register(s, w.reports)
	return nil
}

// worker purges deleted clients, forgets deleted users and busts the
// OpenRouter credential cache.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	w, err := wire(d)
	if err != nil {
		return err
	}
	subjects := []string{events.ClientDeleted, events.UserDeleted, events.IntegrationUpdated}
	return d.Bus.Subscribe(ctx, "reports-events", subjects, func(ctx context.Context, msg events.Message) error {
		switch msg.Subject {
		case events.ClientDeleted:
			var p events.ClientDeletedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			reports, insights, err := w.reports.PurgeClient(ctx, p.ClientID)
			if err != nil {
				return err
			}
			d.Logger.Info("client purged", "client", p.Slug, "reports", reports, "insights", insights)
		case events.UserDeleted:
			var p events.UserDeletedPayload
			if err := msg.Decode(&p); err != nil {
				return err
			}
			n, err := w.reports.ForgetUser(ctx, p.UserID)
			if err != nil {
				return err
			}
			d.Logger.Info("user forgotten", "user", p.UserID, "reports", n)
		case events.IntegrationUpdated:
			w.integrations.Invalidate()
		}
		return nil
	})
}
