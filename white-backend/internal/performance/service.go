// Package performance wires the performance service: KPI scorecards,
// quarterly reviews, 360 feedback, department tasks, logbook, daily and
// monthly work recaps, deadline reminders and improvement items.
//
// Env: HR_ADDR (localhost:9008), IDENTITY_ADDR (localhost:9001),
// INTEGRATIONS_ADDR (localhost:9002), PERFORMANCE_UPLOAD_DIR (task
// attachments, default .uploads/performance), APP_BASE_URL (OpenRouter
// HTTP-Referer). GRPC_ADDR is :9012 in docker.
package performance

import (
	"context"
	"embed"
	"io/fs"
	stdsync "sync"
	_ "time/tzdata" // WIB must resolve inside slim containers

	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/pdf"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "performance"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "performance",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

// Wiring is every use case built on one set of clients.
type Wiring struct {
	Services     grpcapi.Services
	Automations  *app.Automations
	HR           *clients.HR
	Integrations *clients.Integrations
}

var (
	once     stdsync.Once
	shared   *Wiring
	wiredErr error
)

// wire builds the dependencies once per process so `serve` (gRPC + worker)
// shares the directory and credential caches.
func wire(d pkgapp.Deps) (*Wiring, error) {
	once.Do(func() { shared, wiredErr = Wire(d) })
	return shared, wiredErr
}

// Wire connects the clients and builds the use cases.
func Wire(d pkgapp.Deps) (*Wiring, error) {
	conns := map[string]*grpc.ClientConn{}
	for env, def := range map[string]string{
		"HR_ADDR":           "localhost:9008",
		"IDENTITY_ADDR":     "localhost:9001",
		"INTEGRATIONS_ADDR": "localhost:9002",
	} {
		conn, err := grpcx.Dial(config.Env(env, def))
		if err != nil {
			return nil, err
		}
		conns[env] = conn
	}
	clk := clock.Real{}
	hr := clients.NewHR(conns["HR_ADDR"], clk)
	integ := clients.NewIntegrations(conns["INTEGRATIONS_ADDR"], clk)
	deps := app.Deps{
		Store:     postgres.NewStore(d.Pool),
		Directory: hr,
		Notifier:  clients.NewIdentity(conns["IDENTITY_ADDR"]),
		AI: openrouter.NewSource(integ.Credentials, openrouter.Options{Referer: config.Env("APP_BASE_URL", "http://localhost:3000"),
			Logger: d.Logger}),
		Files:     storage.NewLocal(config.Env("PERFORMANCE_UPLOAD_DIR", ".uploads/performance")),
		RenderPDF: pdf.RenderMonthlyRecap,
		Clock:     clk,
		Logger:    d.Logger,
	}
	imp := app.NewImprovements(deps)
	kpi := app.NewKPI(deps)
	tasks := app.NewTasks(deps, imp)
	recaps := app.NewRecaps(deps, imp)
	return &Wiring{
		Services: grpcapi.Services{KPI: kpi, Reviews: app.NewReviews(deps, imp), Feedback: app.NewFeedback(deps, imp), Tasks: tasks,
			Logbook: app.NewLogbook(deps, imp), Recaps: recaps, Improvements: imp, Dashboard: app.NewDashboard(deps)},
		Automations:  app.NewAutomations(deps, kpi, tasks, recaps),
		HR:           hr,
		Integrations: integ,
	}, nil
}

func register(_ context.Context, d pkgapp.Deps, s *grpc.Server) error {
	w, err := wire(d)
	if err != nil {
		return err
	}
	grpcapi.Register(s, w.Services)
	return nil
}

// worker runs the scheduled automations off the ERP ticks and busts the
// credential cache on integration.updated.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	w, err := wire(d)
	if err != nil {
		return err
	}
	subjects := []string{events.ERPTick15m, events.ERPTickHourly, events.ERPTickDaily, events.IntegrationUpdated}
	return d.Bus.Subscribe(ctx, "performance-events", subjects, func(ctx context.Context, msg events.Message) error {
		if msg.Subject == events.IntegrationUpdated {
			w.Integrations.Invalidate()
			return nil
		}
		var p events.TickPayload
		if err := msg.Decode(&p); err != nil {
			return err
		}
		switch msg.Subject {
		case events.ERPTick15m:
			return w.Automations.Tick15m(ctx, p.RunAt)
		case events.ERPTickHourly:
			w.HR.Invalidate()
			return w.Automations.TickHourly(ctx, p.RunAt)
		case events.ERPTickDaily:
			return w.Automations.TickDaily(ctx, p.RunAt)
		}
		return nil
	})
}
