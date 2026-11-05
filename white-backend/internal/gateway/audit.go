package gateway

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
)

// Public paths carry tokens or are called by outside systems, and /internal is
// the frontend minting tokens rather than a person changing data, so they stay out
// of the log.
var auditSkipPrefixes = []string{"/api/public", "/v1/public", "/api/wa/", "/healthz", "/internal/"}

func auditable(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}
	for _, p := range auditSkipPrefixes {
		if strings.HasPrefix(r.URL.Path, p) {
			return false
		}
	}
	return true
}

// withAudit reports every change request: who, what path, and the result. It
// never reads the body or the query string, which may hold secrets. Reporting
// runs after the response and a failure is only logged, so an audit outage
// cannot block work.
func withAudit(d pkgapp.Deps, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !auditable(r) || d.Events == nil {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		rec := auditRecord(r, sw.status, start, d.Verifier)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := d.Events.Publish(ctx, events.AuditSubject, rec); err != nil {
				d.Logger.Warn("audit publish failed", "path", rec.Path, "err", err)
			}
		}()
	})
}

func auditRecord(r *http.Request, status int, at time.Time, v *authn.Verifier) events.AuditRecorded {
	rec := events.AuditRecorded{At: at, Method: r.Method, Path: r.URL.Path, Status: status, IP: clientIP(r), UserAgent: truncate(r.UserAgent(), 200)}
	if v == nil {
		return rec
	}
	token, ok := authn.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		if c, err := r.Cookie("white_access"); err == nil {
			token = c.Value
		}
	}
	if p, err := v.Verify(token); err == nil {
		rec.UserID, rec.UserName, rec.Role = p.UserID, p.Name, p.Role
	}
	return rec
}

func clientIP(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		first, _, _ := strings.Cut(f, ",")
		return strings.TrimSpace(first)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
