package grpcx

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ErrorBody is the JSON the gateway writes for failed calls. It keeps the
// shape the Next.js UI already understands from server actions.
type ErrorBody struct {
	OK                bool                `json:"ok"`
	Error             string              `json:"error"`
	Code              apperr.Code         `json:"code"`
	FieldErrors       map[string][]string `json:"fieldErrors,omitempty"`
	Provider          string              `json:"provider,omitempty"`
	RequiresReconnect bool                `json:"requiresReconnect,omitempty"`
}

var codeToHTTP = map[apperr.Code]int{
	apperr.Validation:      http.StatusBadRequest,
	apperr.Unauthenticated: http.StatusUnauthorized,
	apperr.Unauthorized:    http.StatusForbidden,
	apperr.NotFound:        http.StatusNotFound,
	apperr.Conflict:        http.StatusConflict,
	apperr.RateLimit:       http.StatusTooManyRequests,
	apperr.TokenExpired:    http.StatusFailedDependency,
	apperr.Permission:      http.StatusForbidden,
	apperr.Network:         http.StatusBadGateway,
	apperr.Unknown:         http.StatusInternalServerError,
}

// HTTPStatus maps an application code to the response status.
func HTTPStatus(code apperr.Code) int {
	if s, ok := codeToHTTP[code]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// GatewayErrorHandler is the runtime.ErrorHandlerFunc for grpc-gateway.
func GatewayErrorHandler(ctx context.Context, mux *runtime.ServeMux, m runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	e := FromStatus(err)
	if st, ok := status.FromError(err); ok && st.Code() == codes.Unimplemented {
		e = apperr.New(apperr.NotFound, "Endpoint tidak ditemukan.")
	}
	WriteError(w, e)
}

func WriteError(w http.ResponseWriter, e *apperr.Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(HTTPStatus(e.Code))
	_ = json.NewEncoder(w).Encode(ErrorBody{
		Error: e.Message, Code: e.Code, FieldErrors: e.FieldErrors,
		Provider: e.Provider, RequiresReconnect: e.RequiresReconnect,
	})
}

// IncomingHeaderMatcher forwards the headers services care about.
func IncomingHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "authorization", "x-forwarded-for", "x-real-ip", "cookie", "user-agent", "x-gateway-token":
		return strings.ToLower(key), true
	}
	return runtime.DefaultHeaderMatcher(key)
}

// RemoteIP mirrors the share rate limiter: first x-forwarded-for, then
// x-real-ip, then "local".
func RemoteIP(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if v := md.Get("x-forwarded-for"); len(v) > 0 && v[0] != "" {
		return strings.TrimSpace(strings.Split(v[0], ",")[0])
	}
	if v := md.Get("x-real-ip"); len(v) > 0 && v[0] != "" {
		return v[0]
	}
	return "local"
}

// EdgeHeader marks calls that entered through the gateway. The gateway sets
// it on every forwarded request; service-to-service calls never carry it, so
// an RPC that trusts internal callers can still reject anonymous edge calls.
const EdgeHeader = "x-white-edge"

func FromEdge(ctx context.Context) bool {
	md, _ := metadata.FromIncomingContext(ctx)
	return len(md.Get(EdgeHeader)) > 0
}

// EdgeMetadata is the runtime.WithMetadata hook the gateway installs.
func EdgeMetadata(context.Context, *http.Request) metadata.MD {
	return metadata.Pairs(EdgeHeader, "1")
}
