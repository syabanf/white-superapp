// Package sales wires the sales service (gRPC :9010, schema sales): the CRM
// pipeline ported from NüHabit plus quotation drafts, contracts with
// e-signature, pitch decks, client onboarding and the lead chatbot.
//
// Env: IDENTITY_ADDR (localhost:9001), INTEGRATIONS_ADDR (localhost:9002),
// SALES_UPLOAD_DIR (./.uploads/sales), SALES_SERVICE_TOKEN (optional ADMIN
// access token for client creation from background
// work), WA_GATEWAY_URL + WA_GATEWAY_TOKEN (WhatsApp, off when unset),
// APP_BASE_URL (OpenRouter referer).
package sales

import (
	"context"
	"embed"
	"io/fs"
	_ "time/tzdata" // Asia/Jakarta must resolve inside slim containers

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/providers/wagateway"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/render"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/seed"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "sales"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "sales",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

// Services builds every use case over shared deps.
func Services(d *app.Deps) grpcapi.Services {
	auto := app.NewAutomation(d)
	onboarding := app.NewOnboarding(d)
	return grpcapi.Services{
		Leads:      app.NewLeads(d, auto),
		Pipeline:   app.NewPipeline(d, auto, onboarding),
		Quotes:     app.NewQuotes(d, auto),
		Billing:    app.NewBilling(d, onboarding),
		Insights:   app.NewInsights(d),
		Dashboard:  app.NewDashboard(d),
		Automation: auto,
		Forms:      app.NewForms(d, auto),
		Contracts:  app.NewContracts(d, onboarding),
		Decks:      app.NewDecks(d),
		Onboarding: onboarding,
		Chat:       app.NewChat(d, auto),
	}
}

type wiring struct {
	deps         *app.Deps
	integrations *clients.Integrations
	conns        []*grpc.ClientConn
}

func (w *wiring) close() {
	for _, c := range w.conns {
		_ = c.Close()
	}
}

func wire(pool *pgxpool.Pool, d pkgapp.Deps) (*wiring, error) {
	w := &wiring{}
	dial := func(env, fallback string) (*grpc.ClientConn, error) {
		conn, err := grpcx.Dial(config.Env(env, fallback))
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
	clk := clock.Real{}
	identity := clients.NewIdentity(identityConn, config.Env("SALES_SERVICE_TOKEN", ""))
	w.integrations = clients.NewIntegrations(integrationsConn, clk)
	w.deps = &app.Deps{
		Store:     postgres.NewStore(pool),
		Clock:     clk,
		Events:    d.Events,
		Notifier:  identity,
		Directory: identity,
		Clients:   identity,
		AI:        openrouter.NewSource(w.integrations.Credentials, openrouter.Options{Referer: config.Env("APP_BASE_URL", "")}),
		WhatsApp:  wagateway.New(config.Env("WA_GATEWAY_URL", ""), config.Env("WA_GATEWAY_TOKEN", ""), nil),
		Files:     storage.NewLocal(config.Env("SALES_UPLOAD_DIR", "")),
		Renderer:  render.New(),
		Logger:    d.Logger,
	}
	return w, nil
}

func register(ctx context.Context, d pkgapp.Deps, s *grpc.Server) error {
	w, err := wire(d.Pool, d)
	if err != nil {
		return err
	}
	go func() { <-ctx.Done(); w.close() }()
	if err := seed.Reference(ctx, d.Pool); err != nil {
		return err
	}
	grpcapi.Register(s, Services(w.deps))
	return nil
}

// worker runs the ERP ticks and busts the credentials cache.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	w, err := wire(d.Pool, d)
	if err != nil {
		return err
	}
	defer w.close()
	svc := Services(w.deps)
	jobs := app.NewJobs(w.deps, svc.Automation, svc.Pipeline, svc.Chat, svc.Onboarding)
	errCh := make(chan error, 2)
	go func() {
		errCh <- d.Bus.Subscribe(ctx, "sales-erp-ticks", []string{events.ERPTick15m, events.ERPTickDaily}, func(ctx context.Context, msg events.Message) error {
			var err error
			switch msg.Subject {
			case events.ERPTick15m:
				err = jobs.Every15m(ctx)
			case events.ERPTickDaily:
				err = jobs.Daily(ctx)
			}
			if err != nil {
				d.Logger.Warn("sales tick failed", "subject", msg.Subject, "err", err)
			}
			// Ticks are not retried: the next tick picks the work up.
			return nil
		})
	}()
	go func() {
		errCh <- d.Bus.Subscribe(ctx, "sales-integration-updated", []string{events.IntegrationUpdated}, func(context.Context, events.Message) error {
			w.integrations.Invalidate()
			return nil
		})
	}()
	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}
