// Package integrations wires the integrations service: workspace provider
// credentials (setup wizard) and per-client OAuth connections, the only
// place that holds ENCRYPTION_KEY.
package integrations

import (
	"context"
	"embed"
	"errors"
	"io/fs"

	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/clients"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/providers/googleoauth"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/providers/metaoauth"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/providers/verify"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/app"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/crypto"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const Schema = "integrations"

func Migrations() fs.FS {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

func Spec() pkgapp.Spec {
	return pkgapp.Spec{
		Name:       "integrations",
		Schema:     Schema,
		Migrations: Migrations(),
		Register:   register,
		Worker:     worker,
	}
}

func register(_ context.Context, d pkgapp.Deps, s *grpc.Server) error {
	cipher, err := crypto.New(config.Env("ENCRYPTION_KEY", ""))
	if err != nil {
		return err
	}
	secret := config.Env("AUTH_SECRET", "")
	if secret == "" {
		return errors.New("AUTH_SECRET wajib diisi (penanda tangan state OAuth)")
	}
	idConn, err := grpcx.Dial(config.Env("IDENTITY_ADDR", "localhost:9001"))
	if err != nil {
		return err
	}
	store := postgres.NewStore(d.Pool)
	clk := clock.Real{}
	settings := app.NewSettings(store, cipher, verify.New(), d.Events, clk)
	conns := app.NewConnections(app.ConnectionsDeps{
		Store:       store,
		Cipher:      cipher,
		Creds:       settings,
		Access:      clients.NewAccess(idConn, clk),
		Google:      googleoauth.New(),
		Meta:        metaoauth.New(),
		Bus:         d.Events,
		Clock:       clk,
		StateSecret: secret,
		AppBaseURL:  config.Env("APP_BASE_URL", "http://localhost:3000"),
		Logger:      d.Logger,
	})
	grpcapi.Register(s, grpcapi.Services{Settings: settings, Connections: conns})
	return nil
}

// worker purges a deleted client's connections.
func worker(ctx context.Context, d pkgapp.Deps) error {
	if d.Bus == nil {
		<-ctx.Done()
		return nil
	}
	store := postgres.NewStore(d.Pool)
	return d.Bus.Subscribe(ctx, "integrations-client-deleted", []string{events.ClientDeleted}, func(ctx context.Context, msg events.Message) error {
		var p events.ClientDeletedPayload
		if err := msg.Decode(&p); err != nil {
			return err
		}
		n, err := store.DeleteClientConnections(ctx, p.ClientID)
		if err != nil {
			return err
		}
		d.Logger.Info("client purged", "client", p.ClientID, "connections", n)
		return nil
	})
}
