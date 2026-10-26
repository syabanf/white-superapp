// Command finance runs the finance service.
//
//	finance serve            # migrate (unless AUTO_MIGRATE=false), gRPC :9007 + worker
//	finance migrate          # apply migrations and exit
//	finance worker           # event consumer only
//	finance seed [reference] # reference data (chart, mappings, policies), plus demo data unless "reference"
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/syabanf/white-superapp/white-backend/internal/finance"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/xlsx"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/seed"
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
		_ = os.Setenv("GRPC_ADDR", ":9007")
	}
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := runSeed(context.Background(), os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	pkgapp.Main(finance.Spec())
}

// runSeed migrates, then loads reference data and (unless "reference" is
// passed) demo transactions. Without NATS_URL the reimbursement event is
// only logged; without an OpenRouter key the AI memo is the mock.
func runSeed(ctx context.Context, args []string) error {
	logger := logging.New("finance-seed")
	pool, err := pg.Connect(ctx, config.Env("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pg.Migrate(ctx, pool, finance.Migrations(), "migrations", finance.Schema, logger); err != nil {
		return err
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
	deps := app.Deps{
		Store:     postgres.NewStore(pool),
		Storage:   storage.NewLocal(config.Env("FINANCE_UPLOAD_DIR", "")),
		Insights:  openrouter.NewSource(func(context.Context) (openrouter.Credentials, error) { return openrouter.Credentials{}, nil }, openrouter.Options{}),
		Events:    publisher,
		ReadSheet: xlsx.ReadCoaSheet,
		Clock:     clock.Real{},
		Logger:    logger,
	}
	if err := seed.Reference(ctx, deps); err != nil {
		return fmt.Errorf("reference: %w", err)
	}
	logger.Info("reference data ready")
	if len(args) > 0 && args[0] == "reference" {
		return nil
	}
	if err := seed.Demo(ctx, deps); err != nil {
		return fmt.Errorf("demo: %w", err)
	}
	logger.Info("demo data ready")
	return nil
}
