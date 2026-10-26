package grpcapi_test

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/identity"
	grpcapi "github.com/syabanf/white-superapp/white-backend/internal/identity/adapter/grpc"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/app"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Boots the real gRPC server (interceptors included) on bufconn against the
// test database and drives it through generated clients.
func TestIntegrationGRPC(t *testing.T) {
	pool := pg.TestPool(t, identity.Migrations(), "migrations", identity.Schema)
	pg.TruncateSchema(t, pool, identity.Schema)
	store := postgres.NewStore(pool)
	priv, _ := authn.KeyPair("", "test-secret")
	pub, _ := authn.PublicKey("", "", "test-secret")
	clk := clock.Real{}
	users := app.NewUsers(store, events.Noop{})
	ctx := context.Background()
	if _, err := users.Create(ctx, app.UserInput{Name: "Admin", Email: "admin@white.id", Password: "admin12345", Role: domain.RoleAdmin, IsActive: true}); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	srv := grpcx.NewServer(grpcx.ServerOptions{Logger: logger, Verifier: authn.NewVerifier(pub)})
	grpcapi.Register(srv, grpcapi.Services{
		InternalSecret: "s3cret",
		Auth:           app.NewAuth(store, authn.NewIssuer(priv, time.Minute), clk),
		Users:          users,
		Clients:        app.NewClients(store, events.Noop{}),
		Notifications:  app.NewNotifications(store),
		Share:          app.NewShare(store, "test-secret", clk),
		Workspace:      app.NewWorkspace(store, clk),
	})
	lis := bufconn.Listen(1 << 20)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	auth := identityv1.NewAuthServiceClient(conn)
	clients := identityv1.NewClientServiceClient(conn)

	// Unauthenticated calls are rejected with the shared error shape.
	_, err = clients.ListClients(ctx, &emptypb.Empty{})
	if e := grpcx.FromStatus(err); e == nil || e.Code != apperr.Unauthenticated {
		t.Fatalf("expected UNAUTHENTICATED, got %v", err)
	}

	_, err = auth.Login(ctx, &identityv1.LoginRequest{Email: "admin@white.id", Password: "nope"})
	if e := grpcx.FromStatus(err); e == nil || e.Code != apperr.Unauthenticated || e.Message != "Email atau kata sandi salah" {
		t.Fatalf("got %v", err)
	}
	tok, err := auth.Login(ctx, &identityv1.LoginRequest{Email: "admin@white.id", Password: "admin12345"})
	if err != nil {
		t.Fatal(err)
	}
	authed := grpcx.WithBearer(ctx, tok.AccessToken)

	me, err := auth.Me(authed, &emptypb.Empty{})
	if err != nil || me.Role != identityv1.UserRole_ADMIN {
		t.Fatalf("%v %v", me, err)
	}

	// Validation errors carry field violations back through the status details.
	_, err = clients.CreateClient(authed, &identityv1.CreateClientRequest{Profile: &identityv1.ClientProfile{Name: "", Slug: "x", Currency: "IDR", Timezone: "Asia/Jakarta"}})
	e := grpcx.FromStatus(err)
	if e == nil || e.Code != apperr.Validation || len(e.FieldErrors["name"]) == 0 || len(e.FieldErrors["slug"]) == 0 {
		t.Fatalf("%+v", e)
	}
	created, err := clients.CreateClient(authed, &identityv1.CreateClientRequest{Profile: &identityv1.ClientProfile{Name: "Kopi", Slug: "kopi", Currency: "IDR", Timezone: "Asia/Jakarta"}})
	if err != nil {
		t.Fatal(err)
	}
	list, err := clients.ListClients(authed, &emptypb.Empty{})
	if err != nil || len(list.Clients) != 1 || list.Clients[0].Client.Id != created.Id || list.Clients[0].MyRole != identityv1.ClientAccess_ADMIN_OVERRIDE {
		t.Fatalf("%v %v", list, err)
	}

	// Service tokens need the shared secret.
	if _, err := auth.IssueServiceToken(ctx, &identityv1.IssueServiceTokenRequest{UserId: me.Id}); grpcx.FromStatus(err).Code != apperr.Unauthorized {
		t.Fatalf("no secret: %v", err)
	}
	svcCtx := metadata.AppendToOutgoingContext(ctx, "x-internal-secret", "s3cret")
	st, err := auth.IssueServiceToken(svcCtx, &identityv1.IssueServiceTokenRequest{UserId: me.Id})
	if err != nil || st.AccessToken == "" || st.RefreshToken != "" {
		t.Fatalf("%v %v", st, err)
	}
	if _, err := clients.ListClients(grpcx.WithBearer(ctx, st.AccessToken), &emptypb.Empty{}); err != nil {
		t.Fatal("service token rejected:", err)
	}

	// Refresh rotates; the old token dies.
	pair, err := auth.Refresh(ctx, &identityv1.RefreshRequest{RefreshToken: tok.RefreshToken})
	if err != nil || pair.RefreshToken == tok.RefreshToken {
		t.Fatal(err)
	}
	if _, err := auth.Refresh(ctx, &identityv1.RefreshRequest{RefreshToken: tok.RefreshToken}); err == nil {
		t.Fatal("old refresh token accepted")
	}
	if _, err := auth.Logout(ctx, &identityv1.LogoutRequest{RefreshToken: pair.RefreshToken}); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Refresh(ctx, &identityv1.RefreshRequest{RefreshToken: pair.RefreshToken}); err == nil {
		t.Fatal("logged-out refresh token accepted")
	}
}
