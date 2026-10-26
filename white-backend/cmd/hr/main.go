// Command hr runs the hr service: serve | migrate | worker (pkg/app) and
// seed, which inserts the WHITE demo data (idempotent).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/docs"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/seed"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"github.com/syabanf/white-superapp/white-backend/pkg/logging"
	"github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := runSeed(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	pkgapp.Main(hr.Spec())
}

// runSeed migrates, then seeds. Identity users are read with a short-lived
// admin token minted from JWT_PRIVATE_KEY / AUTH_SECRET so employees can be
// linked by email; without identity the seed still runs, unlinked. Events
// are not published (the paid demo run stays out of finance).
func runSeed(ctx context.Context) error {
	_ = config.LoadDotEnv(".env")
	logger := logging.New("hr-seed")
	pool, err := postgres.Connect(ctx, config.Env("DATABASE_URL", ""))
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool, hr.Migrations(), "migrations", hr.Schema, logger); err != nil {
		return err
	}
	svc, identity, closeFn, err := hr.Wire(pkgapp.Deps{Logger: logger, Pool: pool, Events: events.Noop{Logger: logger}})
	if err != nil {
		return err
	}
	defer closeFn()
	users := identityUsers(ctx, identity, logger)
	renderer := docs.Renderer{CompanyName: config.Env("HR_COMPANY_NAME", "WHITE")}
	if err := seed.New(svc.Deps, renderer).Run(ctx, users); err != nil {
		return err
	}
	logger.Info("seed done", "linkableUsers", len(users))
	return nil
}

func identityUsers(ctx context.Context, identity interface {
	UserOptions(context.Context) ([]app.UserInfo, error)
}, logger *slog.Logger) []app.UserInfo {
	key, err := authn.KeyPair(config.Env("JWT_PRIVATE_KEY", ""), config.Env("AUTH_SECRET", ""))
	if err != nil {
		logger.Warn("tanpa kunci JWT: karyawan tidak ditautkan ke akun", "err", err)
		return nil
	}
	tok, _, err := authn.NewIssuer(key, 5*time.Minute).Issue(authn.Principal{UserID: "hr-seed", Role: authn.RoleAdmin, Name: "HR Seeder"}, time.Now())
	if err != nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(grpcx.WithBearer(ctx, tok), 5*time.Second)
	defer cancel()
	users, err := identity.UserOptions(cctx)
	if err != nil {
		logger.Warn("identity tidak terjangkau: karyawan tidak ditautkan ke akun", "err", err)
		return nil
	}
	return users
}
