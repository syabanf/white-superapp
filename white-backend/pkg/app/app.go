// Package app is the shared binary skeleton: read env, open Postgres, run
// this service's migrations, connect NATS, start the gRPC server and the
// background worker, stop cleanly on SIGTERM.
//
//	white-identity serve    # migrate (unless AUTO_MIGRATE=false) and serve
//	white-identity migrate  # apply migrations and exit
//	white-identity worker   # run only the background worker
package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"github.com/syabanf/white-superapp/white-backend/pkg/logging"
	"github.com/syabanf/white-superapp/white-backend/pkg/postgres"
	"google.golang.org/grpc"
)

// Deps is what a service's wiring receives.
type Deps struct {
	Logger   *slog.Logger
	Pool     *pgxpool.Pool
	Bus      *events.Bus // nil without NATS_URL
	Events   events.Publisher
	Verifier *authn.Verifier
}

type Spec struct {
	Name string
	// Schema is the Postgres schema this service owns.
	Schema string
	// Migrations is the embedded FS holding `migrations/*.sql`.
	Migrations fs.FS
	// Register adds gRPC services to the server. Nil for worker-only binaries.
	Register func(ctx context.Context, d Deps, s *grpc.Server) error
	// Worker runs background consumers until ctx ends. Optional.
	Worker func(ctx context.Context, d Deps) error
	// NoDatabase skips Postgres (gateway, scheduler).
	NoDatabase bool
	// NoVerifier skips JWT setup (identity builds its own issuer+verifier).
	GRPCAddrEnv string
}

// Main runs the service and exits the process on failure.
func Main(spec Spec) {
	if err := Run(context.Background(), spec, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Run(parent context.Context, spec Spec, args []string) error {
	_ = config.LoadDotEnv(".env")
	logger := logging.New(spec.Name)
	cmd := "serve"
	if len(args) > 0 {
		cmd = args[0]
	}
	ctx, stop := signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	d := Deps{Logger: logger, Events: events.Noop{Logger: logger}}

	if !spec.NoDatabase {
		pool, err := postgres.Connect(ctx, config.Env("DATABASE_URL", ""))
		if err != nil {
			return err
		}
		defer pool.Close()
		d.Pool = pool
		if cmd == "migrate" || config.EnvBool("AUTO_MIGRATE", true) {
			if err := postgres.Migrate(ctx, pool, spec.Migrations, "migrations", spec.Schema, logger); err != nil {
				return err
			}
		}
		if cmd == "migrate" {
			logger.Info("migrations up to date", "schema", spec.Schema)
			return nil
		}
	}

	if url := config.Env("NATS_URL", ""); url != "" {
		bus, err := events.Connect(ctx, url, logger)
		if err != nil {
			return fmt.Errorf("nats: %w", err)
		}
		defer bus.Close()
		d.Bus = bus
		d.Events = bus
	} else {
		logger.Warn("NATS_URL kosong: event hanya dicatat di log")
	}

	pub, err := authn.PublicKey(config.Env("JWT_PUBLIC_KEY", ""), config.Env("JWT_PRIVATE_KEY", ""), config.Env("AUTH_SECRET", ""))
	if err != nil {
		return err
	}
	d.Verifier = authn.NewVerifier(pub)

	errCh := make(chan error, 2)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	switch cmd {
	case "serve":
		if spec.Register == nil {
			return errors.New("service ini tidak punya server gRPC")
		}
		srv := grpcx.NewServer(grpcx.ServerOptions{Logger: logger, Verifier: d.Verifier})
		if err := spec.Register(runCtx, d, srv); err != nil {
			return err
		}
		addrEnv := spec.GRPCAddrEnv
		if addrEnv == "" {
			addrEnv = "GRPC_ADDR"
		}
		go func() { errCh <- grpcx.Serve(runCtx, srv, config.Env(addrEnv, ":9000"), logger) }()
		if spec.Worker != nil && config.EnvBool("RUN_WORKER", true) {
			go func() { errCh <- spec.Worker(runCtx, d) }()
		}
	case "worker":
		if spec.Worker == nil {
			return errors.New("service ini tidak punya worker")
		}
		go func() { errCh <- spec.Worker(runCtx, d) }()
	default:
		return fmt.Errorf("perintah tidak dikenal: %s (serve|migrate|worker)", cmd)
	}

	select {
	case <-runCtx.Done():
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	}
}
