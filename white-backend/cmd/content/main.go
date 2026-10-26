// Command content runs the content service.
//
//	content serve    # migrate (unless AUTO_MIGRATE=false), gRPC :9011 + worker
//	content migrate  # apply migrations and exit
//	content worker   # ERP ticks and lifecycle events only
//	content seed     # demo data for up to three marketing clients
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/providers/ai"
	"github.com/syabanf/white-superapp/white-backend/internal/content/seed"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"github.com/syabanf/white-superapp/white-backend/pkg/logging"
	"github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

func main() {
	_ = config.LoadDotEnv(".env")
	if os.Getenv("GRPC_ADDR") == "" {
		_ = os.Setenv("GRPC_ADDR", ":9011")
	}
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := runSeed(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	pkgapp.Main(content.Spec())
}

// runSeed migrates, reads the marketing clients from identity and the
// users to assign with a short-lived admin token minted from
// JWT_PRIVATE_KEY / AUTH_SECRET, then seeds. The report summary uses the
// mock AI so the seed runs without OpenRouter.
func runSeed(ctx context.Context) error {
	logger := logging.New("content-seed")
	pool, err := postgres.Connect(ctx, config.Env("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool, content.Migrations(), "migrations", content.Schema, logger); err != nil {
		return err
	}
	w, err := content.Wire(pkgapp.Deps{Logger: logger, Pool: pool})
	if err != nil {
		return err
	}
	defer w.Close()

	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	list, err := w.Identity.Clients(cctx)
	if err != nil {
		return fmt.Errorf("identity tidak terjangkau, klien marketing tidak bisa dibaca: %w", err)
	}
	if len(list) == 0 {
		logger.Warn("belum ada klien marketing di identity: tidak ada data konten yang dibuat")
		return nil
	}
	clients := make([]seed.Client, 0, len(list))
	for _, c := range list {
		clients = append(clients, seed.Client{ID: c.ID, Name: c.Name})
	}

	var users []string
	key, err := authn.KeyPair(config.Env("JWT_PRIVATE_KEY", ""), config.Env("AUTH_SECRET", ""))
	if err == nil {
		tok, _, err := authn.NewIssuer(key, 5*time.Minute).Issue(authn.Principal{UserID: "content-seed", Role: authn.RoleAdmin, Name: "Content Seeder"}, time.Now())
		if err == nil {
			opts, err := w.Identity.UserOptions(grpcx.WithBearer(cctx, tok))
			if err != nil {
				logger.Warn("daftar pengguna tidak terbaca: konten tanpa penanggung jawab", "err", err)
			}
			for _, u := range opts {
				users = append(users, u.Id)
			}
		}
	}

	s := &seed.Seeder{
		Store: w.Deps.Store, Files: w.Deps.Files, Now: time.Now().UTC(),
		AI: ai.NewSource(func(context.Context) (ai.Credentials, error) { return ai.Credentials{}, nil }, ai.Options{}),
	}
	res, err := s.Run(ctx, clients, users)
	if err != nil {
		return err
	}
	logger.Info("seed done", "clients", res.Clients, "plans", res.Plans, "items", res.Items, "assets", res.Assets, "reports", res.Reports, "historyMonths", res.HistoryMonths, "skipped", res.Skipped, "users", len(users))
	return nil
}
