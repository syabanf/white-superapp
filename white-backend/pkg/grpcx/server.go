// Package grpcx wires gRPC servers and clients the same way in every
// service: recovery, request logging, principal extraction and the shared
// error mapping.
package grpcx

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type ServerOptions struct {
	Logger   *slog.Logger
	Verifier *authn.Verifier
	// MaxRecvBytes raises the default 4 MB limit (CSV import, media upload).
	MaxRecvBytes int
}

// NewServer builds a gRPC server with health and reflection registered.
func NewServer(opts ServerOptions) *grpc.Server {
	if opts.MaxRecvBytes == 0 {
		opts.MaxRecvBytes = 32 << 20
	}
	s := grpc.NewServer(
		grpc.MaxRecvMsgSize(opts.MaxRecvBytes),
		grpc.ChainUnaryInterceptor(
			recoverUnary(opts.Logger),
			authUnary(opts.Verifier),
			logUnary(opts.Logger),
			errorUnary,
		),
	)
	healthpb.RegisterHealthServer(s, health.NewServer())
	reflection.Register(s)
	return s
}

// Serve listens until ctx is done, then drains in-flight calls.
func Serve(ctx context.Context, s *grpc.Server, addr string, logger *slog.Logger) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- s.Serve(lis) }()
	logger.Info("grpc listening", "addr", addr)
	select {
	case <-ctx.Done():
		done := make(chan struct{})
		go func() { s.GracefulStop(); close(done) }()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			s.Stop()
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func recoverUnary(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "Terjadi kesalahan tak terduga.")
			}
		}()
		return handler(ctx, req)
	}
}

// authUnary reads a bearer token from metadata when present. Rejection is
// the use case's call (authn.Require), so public RPCs keep working.
func authUnary(v *authn.Verifier) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if v == nil {
			return handler(ctx, req)
		}
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			for _, h := range md.Get(authn.MetadataHeader) {
				if tok, ok := authn.BearerToken(h); ok {
					if p, err := v.Verify(tok); err == nil {
						ctx = authn.WithPrincipal(ctx, p)
					}
					break
				}
			}
		}
		return handler(ctx, req)
	}
}

func logUnary(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		attrs := []any{"method", info.FullMethod, "ms", time.Since(start).Milliseconds()}
		if p, ok := authn.FromContext(ctx); ok {
			attrs = append(attrs, "user", p.UserID)
		}
		if err != nil {
			st, _ := status.FromError(err)
			attrs = append(attrs, "code", st.Code().String(), "err", st.Message())
			if st.Code() == codes.Internal || st.Code() == codes.Unknown {
				logger.Error("rpc", attrs...)
			} else {
				logger.Info("rpc", attrs...)
			}
			return resp, err
		}
		logger.Debug("rpc", attrs...)
		return resp, nil
	}
}

func errorUnary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	resp, err := handler(ctx, req)
	return resp, ToStatus(err)
}

// Dial opens a plaintext client connection (services talk over a private
// network; TLS terminates at the gateway). Outgoing calls forward the
// caller's authorization metadata so downstream services see the same user.
func Dial(addr string) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(propagateAuth),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(32<<20)),
	)
}

func propagateAuth(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(authn.MetadataHeader); len(vals) > 0 {
			if _, has := metadata.FromOutgoingContext(ctx); !has {
				ctx = metadata.AppendToOutgoingContext(ctx, authn.MetadataHeader, vals[0])
			}
		}
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

// WithBearer attaches a token to an outgoing context (used by the scheduler
// and by tests that call services directly).
func WithBearer(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, authn.MetadataHeader, "Bearer "+token)
}
