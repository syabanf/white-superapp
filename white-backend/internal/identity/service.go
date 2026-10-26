// Package identity wires the identity service: users, auth, clients,
// memberships, notifications, public share access and workspace flags.
package identity

import (
	"context"
	"embed"
	"io/fs"
	"time"

	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/identity/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/app"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "identity"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "identity",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

func register(_ context.Context, d pkgapp.Deps, s *grpc.Server) error {
	priv, err := authn.KeyPair(config.Env("JWT_PRIVATE_KEY", ""), config.Env("AUTH_SECRET", ""))
	if err != nil {
		return err
	}
	if config.Env("JWT_PRIVATE_KEY", "") == "" {
		d.Logger.Warn("JWT_PRIVATE_KEY kosong: kunci diturunkan dari AUTH_SECRET (hanya untuk pengembangan)")
	}
	store := postgres.NewStore(d.Pool)
	clk := clock.Real{}
	issuer := authn.NewIssuer(priv, config.EnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute))
	grpcapi.Register(s, grpcapi.Services{
		InternalSecret: config.Env("INTERNAL_API_SECRET", ""),
		Auth:           app.NewAuth(store, issuer, clk),
		Users:          app.NewUsers(store, d.Events),
		Clients:        app.NewClients(store, d.Events),
		Notifications:  app.NewNotifications(store),
		Share:          app.NewShare(store, config.Env("AUTH_SECRET", ""), clk),
		Workspace:      app.NewWorkspace(store, clk),
	})
	return nil
}

// worker prunes read notifications and expired sessions on the retention tick.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	notif := app.NewNotifications(postgres.NewStore(d.Pool))
	return d.Bus.Subscribe(ctx, "identity-retention", []string{events.RetentionTick}, func(ctx context.Context, msg events.Message) error {
		var p events.TickPayload
		if err := msg.Decode(&p); err != nil {
			return err
		}
		n, err := notif.RunRetention(ctx, p.RunAt)
		if err != nil {
			return err
		}
		d.Logger.Info("retention", "deleted", n)
		return nil
	})
}
