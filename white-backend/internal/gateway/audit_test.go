package gateway

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	pkgapp "github.com/syabanf/white-superapp/white-backend/pkg/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
)

type recordingBus struct {
	mu   sync.Mutex
	sent []events.AuditRecorded
	done chan struct{}
}

func (b *recordingBus) Publish(_ context.Context, _ string, payload any) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, payload.(events.AuditRecorded))
	b.done <- struct{}{}
	return nil
}

func TestWithAudit(t *testing.T) {
	bus := &recordingBus{done: make(chan struct{}, 4)}
	h := withAudit(pkgapp.Deps{Events: bus, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))

	for _, r := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/v1/finance/accounts", nil),
		httptest.NewRequest(http.MethodPost, "/api/public/sales/contracts/secret-token/sign", nil),
		httptest.NewRequest(http.MethodPost, "/api/wa/inbound", nil),
	} {
		h.ServeHTTP(httptest.NewRecorder(), r)
	}
	req := httptest.NewRequest(http.MethodPatch, "/v1/sales/invoices/1?token=rahasia", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	h.ServeHTTP(httptest.NewRecorder(), req)

	select {
	case <-bus.done:
	case <-time.After(2 * time.Second):
		t.Fatal("no audit event")
	}
	bus.mu.Lock()
	defer bus.mu.Unlock()
	if len(bus.sent) != 1 {
		t.Fatalf("reads and public paths must be skipped: %+v", bus.sent)
	}
	got := bus.sent[0]
	if got.Path != "/v1/sales/invoices/1" || got.Method != http.MethodPatch || got.Status != http.StatusForbidden || got.IP != "203.0.113.9" {
		t.Fatalf("%+v", got)
	}
}
