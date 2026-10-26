// Package recruitment wires the recruitment service: job openings,
// candidates and their pipeline, talent pool, psikotes, AI interview,
// scheduled interviews with scorecards, offers and the public candidate
// portals.
//
// Env: HR_ADDR (localhost:9008), IDENTITY_ADDR (localhost:9001),
// INTEGRATIONS_ADDR (localhost:9002), RECRUITMENT_UPLOAD_DIR (CVs and
// drawings, default .uploads/recruitment), RECRUITMENT_NOTIFY_USER_IDS
// (comma-separated identity users told about applications, responses and
// reminders, on top of every recruitment.write holder from identity), RECRUITMENT_COMPANY_NAME (default WHITE), APP_BASE_URL
// (OpenRouter HTTP-Referer). GRPC_ADDR is :9009 in docker.
package recruitment

import (
	"context"
	"embed"
	"io/fs"
	"strings"
	stdsync "sync"
	_ "time/tzdata" // WIB must resolve inside slim containers

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/pdf"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/app"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "recruitment"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "recruitment",
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
	Deps         app.Deps
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

func splitIDs(s string) []string {
	var out []string
	for _, id := range strings.Split(s, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return out
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
	identity := clients.NewIdentity(conns["IDENTITY_ADDR"], clk)
	publisher := d.Events
	if publisher == nil {
		publisher = events.Noop{Logger: d.Logger}
	}
	deps := app.Deps{
		Store:     postgres.NewStore(d.Pool),
		Directory: hr,
		Notifier:  identity,
		Audience:  identity,
		AI: openrouter.NewSource(integ.Credentials, openrouter.Options{Referer: config.Env("APP_BASE_URL", "http://localhost:3000"),
			Logger: d.Logger}),
		Files:       storage.NewLocal(config.Env("RECRUITMENT_UPLOAD_DIR", ".uploads/recruitment")),
		PDF:         pdf.Renderer{},
		Events:      publisher,
		Clock:       clk,
		Logger:      d.Logger,
		Recruiters:  splitIDs(config.Env("RECRUITMENT_NOTIFY_USER_IDS", "")),
		CompanyName: config.Env("RECRUITMENT_COMPANY_NAME", "WHITE"),
	}
	return &Wiring{
		Services: grpcapi.Services{Jobs: app.NewJobOpenings(deps), Candidates: app.NewCandidates(deps), Psikotes: app.NewPsikotes(deps),
			Interviews: app.NewInterviews(deps), Offers: app.NewOffers(deps), Portal: app.NewPortal(deps)},
		Automations:  app.NewAutomations(deps),
		Deps:         deps,
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

// worker runs the daily reminders off white.erp.tick.daily and busts the
// credential cache on white.integration.updated.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	w, err := wire(d)
	if err != nil {
		return err
	}
	subjects := []string{events.ERPTickDaily, events.IntegrationUpdated}
	return d.Bus.Subscribe(ctx, "recruitment-events", subjects, func(ctx context.Context, msg events.Message) error {
		if msg.Subject == events.IntegrationUpdated {
			w.Integrations.Invalidate()
			return nil
		}
		var p events.TickPayload
		if err := msg.Decode(&p); err != nil {
			return err
		}
		w.HR.Invalidate()
		return w.Automations.TickDaily(ctx, p.RunAt)
	})
}
