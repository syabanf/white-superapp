// Package gateway is the HTTP edge: grpc-gateway REST for every service's
// public RPCs, plus the handful of browser flows that need cookies or raw
// bytes (OAuth redirects, share PIN cookie, media files, PDF downloads).
package gateway

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	analyticsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/analytics/v1"
	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed openapi/white.swagger.json
var openapiFS embed.FS

func Spec() pkgapp.Spec {
	return pkgapp.Spec{Name: "gateway", NoDatabase: true, Worker: run}
}

type Upstreams struct {
	Identity     *grpc.ClientConn
	Integrations *grpc.ClientConn
	Analytics    *grpc.ClientConn
	Publishing   *grpc.ClientConn
	SeoSuite     *grpc.ClientConn
	Reports      *grpc.ClientConn
}

func dialAll() (*Upstreams, error) {
	dial := func(env, def string) (*grpc.ClientConn, error) {
		return grpcx.Dial(config.Env(env, def))
	}
	var u Upstreams
	var err error
	if u.Identity, err = dial("IDENTITY_ADDR", "localhost:9001"); err != nil {
		return nil, err
	}
	if u.Integrations, err = dial("INTEGRATIONS_ADDR", "localhost:9002"); err != nil {
		return nil, err
	}
	if u.Analytics, err = dial("ANALYTICS_ADDR", "localhost:9003"); err != nil {
		return nil, err
	}
	if u.Publishing, err = dial("PUBLISHING_ADDR", "localhost:9004"); err != nil {
		return nil, err
	}
	if u.SeoSuite, err = dial("SEOSUITE_ADDR", "localhost:9005"); err != nil {
		return nil, err
	}
	if u.Reports, err = dial("REPORTS_ADDR", "localhost:9006"); err != nil {
		return nil, err
	}
	return &u, nil
}

// registerGateway wires every public RPC. Services added later register here.
func registerGateway(ctx context.Context, mux *runtime.ServeMux, u *Upstreams) error {
	regs := []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error{
		identityv1.RegisterAuthServiceHandler,
		identityv1.RegisterUserServiceHandler,
		identityv1.RegisterClientServiceHandler,
		identityv1.RegisterNotificationServiceHandler,
		identityv1.RegisterShareServiceHandler,
		identityv1.RegisterWorkspaceServiceHandler,
	}
	for _, r := range regs {
		if err := r(ctx, mux, u.Identity); err != nil {
			return err
		}
	}
	for _, r := range []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error{
		integrationsv1.RegisterConnectionServiceHandler,
		integrationsv1.RegisterSettingsServiceHandler,
	} {
		if err := r(ctx, mux, u.Integrations); err != nil {
			return err
		}
	}
	if err := analyticsv1.RegisterSourceServiceHandler(ctx, mux, u.Analytics); err != nil {
		return err
	}
	return registerMore(ctx, mux, u)
}

func run(ctx context.Context, d pkgapp.Deps) error {
	up, err := dialAll()
	if err != nil {
		return err
	}
	mux := runtime.NewServeMux(
		runtime.WithErrorHandler(grpcx.GatewayErrorHandler),
		runtime.WithIncomingHeaderMatcher(grpcx.IncomingHeaderMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions:   protojson.MarshalOptions{UseProtoNames: false, EmitUnpopulated: true},
			UnmarshalOptions: protojson.UnmarshalOptions{DiscardUnknown: true},
		}),
	)
	if err := registerGateway(ctx, mux, up); err != nil {
		return err
	}

	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	root.Handle("GET /openapi.json", http.FileServerFS(openapiFS))
	browser := NewBrowserRoutes(up, d.Logger)
	browser.Mount(root)
	root.Handle("/", mux)

	handler := withCORS(withRequestLog(d.Logger, root), strings.Split(config.Env("CORS_ORIGINS", "http://localhost:3000"), ","))
	srv := &http.Server{
		Addr:              config.Env("HTTP_ADDR", ":8080"),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	d.Logger.Info("gateway listening", "addr", srv.Addr)
	select {
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func withCORS(next http.Handler, origins []string) http.Handler {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[strings.TrimSpace(o)] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowed["*"] || allowed[origin]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, If-None-Match")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func withRequestLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		logger.Debug("http", "method", r.Method, "path", r.URL.Path, "status", sw.status, "ms", time.Since(start).Milliseconds())
	})
}

func cookieSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func appBaseURL(r *http.Request) string {
	if v := config.Env("APP_BASE_URL", ""); v != "" {
		return strings.TrimRight(v, "/")
	}
	scheme := "http"
	if cookieSecure(r) {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
