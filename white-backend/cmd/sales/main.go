// Command sales runs the sales service.
//
//	sales serve            # migrate (unless AUTO_MIGRATE=false), gRPC :9010 + worker
//	sales migrate          # apply migrations and exit
//	sales worker           # ERP tick consumer only
//	sales seed [reference] # reference data, plus Mixist demo data unless "reference"
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/render"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/seed"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/logging"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

func main() {
	_ = config.LoadDotEnv(".env")
	if os.Getenv("GRPC_ADDR") == "" {
		_ = os.Setenv("GRPC_ADDR", ":9010")
	}
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := runSeed(context.Background(), os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	pkgapp.Main(sales.Spec())
}

// runSeed migrates, loads reference data and (unless "reference") the
// demo data. It runs offline: no identity, the mock AI, events logged
// unless NATS_URL is set. SALES_SEED_OWNER_IDS (comma separated identity
// user ids) assigns the demo leads and deals to real users. The dashboard
// history is written directly, without events.
func runSeed(ctx context.Context, args []string) error {
	logger := logging.New("sales-seed")
	pool, err := pg.Connect(ctx, config.Env("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pg.Migrate(ctx, pool, sales.Migrations(), "migrations", sales.Schema, logger); err != nil {
		return err
	}
	if err := seed.Reference(ctx, pool); err != nil {
		return fmt.Errorf("reference: %w", err)
	}
	logger.Info("reference data ready")
	if len(args) > 0 && args[0] == "reference" {
		return nil
	}
	var publisher events.Publisher = events.Noop{Logger: logger}
	if url := config.Env("NATS_URL", ""); url != "" {
		bus, err := events.Connect(ctx, url, logger)
		if err != nil {
			return err
		}
		defer bus.Close()
		publisher = bus
	}
	deps := &app.Deps{
		Store:    postgres.NewStore(pool),
		Clock:    clock.Real{},
		Events:   publisher,
		AI:       openrouter.NewSource(func(context.Context) (openrouter.Credentials, error) { return openrouter.Credentials{}, nil }, openrouter.Options{}),
		Files:    storage.NewLocal(config.Env("SALES_UPLOAD_DIR", "")),
		Renderer: render.New(),
		Logger:   logger,
	}
	svc := sales.Services(deps)
	var owners []string
	for _, id := range strings.Split(config.Env("SALES_SEED_OWNER_IDS", ""), ",") {
		if id = strings.TrimSpace(id); id != "" {
			owners = append(owners, id)
		}
	}
	apps := seed.Apps{Store: deps.Store, Leads: svc.Leads, Pipeline: svc.Pipeline, Quotes: svc.Quotes, Billing: svc.Billing, Contracts: svc.Contracts, Chat: svc.Chat, Decks: svc.Decks, Onboarding: svc.Onboarding}
	if err := seed.Demo(ctx, apps, owners); err != nil {
		return fmt.Errorf("demo: %w", err)
	}
	if err := seed.History(ctx, pool, owners, time.Now()); err != nil {
		return fmt.Errorf("history: %w", err)
	}
	if err := seed.Showcase(ctx, pool, apps, owners); err != nil {
		return fmt.Errorf("showcase: %w", err)
	}
	logger.Info("demo data ready")
	return nil
}
