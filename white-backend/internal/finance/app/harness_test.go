package app_test

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── fakes ──────────────────────────────────────────────────

type fakeHR struct{ byUser, byID map[string]*app.Employee }

func (f *fakeHR) EmployeeByUser(_ context.Context, userID string) (*app.Employee, error) {
	return f.byUser[userID], nil
}

func (f *fakeHR) Employee(_ context.Context, id string) (*app.Employee, error) {
	return f.byID[id], nil
}

type fakeIdentity struct {
	mu   sync.Mutex
	sent []app.NotifyInput
}

func (f *fakeIdentity) Notify(_ context.Context, in app.NotifyInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, in)
	return nil
}

func (f *fakeIdentity) UserName(context.Context, string) string { return "Tester" }

func (f *fakeIdentity) PermissionHolders(context.Context, string) ([]string, error) {
	return []string{"finance-user"}, nil
}

func (f *fakeIdentity) last() app.NotifyInput {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) == 0 {
		return app.NotifyInput{}
	}
	return f.sent[len(f.sent)-1]
}

type fakeStorage struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (f *fakeStorage) Put(_ context.Context, key string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[key] = data
	return nil
}

func (f *fakeStorage) Get(_ context.Context, key string) ([]byte, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.files[key], "etag", nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.files, key)
	return nil
}

type fakeInsights struct{ last app.InsightInput }

func (f *fakeInsights) Generate(_ context.Context, in app.InsightInput) (string, string, error) {
	f.last = in
	return "## Ringkasan eksekutif\nok", "mock", nil
}

type published struct {
	subject string
	payload any
}

type fakeEvents struct {
	mu   sync.Mutex
	sent []published
}

func (f *fakeEvents) Publish(_ context.Context, subject string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, published{subject, payload})
	return nil
}

// ─── harness ────────────────────────────────────────────────

type harness struct {
	t        *testing.T
	ctx      context.Context
	deps     app.Deps
	hr       *fakeHR
	identity *fakeIdentity
	storage  *fakeStorage
	events   *fakeEvents
	insights *fakeInsights
	acc      map[string]domain.Account
}

// today is pinned so period and age rules are deterministic.
const today = "2026-10-26"

// newHarness truncates the finance schema, loads the reference data and
// creates FY2026 with October OPEN.
func newHarness(t *testing.T) *harness {
	t.Helper()
	pool := pg.TestPool(t, finance.Migrations(), "migrations", finance.Schema)
	pg.TruncateSchema(t, pool, finance.Schema)
	h := &harness{
		t: t, ctx: context.Background(),
		hr:       &fakeHR{byUser: map[string]*app.Employee{}, byID: map[string]*app.Employee{}},
		identity: &fakeIdentity{}, storage: &fakeStorage{files: map[string][]byte{}}, events: &fakeEvents{}, insights: &fakeInsights{},
	}
	h.deps = app.Deps{
		Store: postgres.NewStore(pool), Identity: h.identity, HR: h.hr, Storage: h.storage, Insights: h.insights, Events: h.events,
		ReadSheet: func([]byte) ([][]string, error) { return nil, nil }, Clock: clock.At(today + "T03:00:00Z"),
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})),
	}
	if err := seed.Reference(h.ctx, h.deps); err != nil {
		t.Fatal(err)
	}
	h.acc = h.must2(h.deps.Store.AccountsByCode(h.ctx))
	year := h.mustYear(app.NewFiscal(h.deps).CreateYear(h.ctx, "u-fin", app.YearInput{Code: "FY2026", Name: "Tahun Buku 2026", StartDate: "2026-01-01", EndDate: "2026-12-31", IsActive: true}))
	if _, err := app.NewFiscal(h.deps).OpenPeriod(h.ctx, year.Periods[9].ID, true); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *harness) mustYear(y *domain.FiscalYear, err error) *domain.FiscalYear {
	h.t.Helper()
	if err != nil {
		h.t.Fatal(err)
	}
	return y
}

func (h *harness) must2(m map[string]domain.Account, err error) map[string]domain.Account {
	h.t.Helper()
	if err != nil {
		h.t.Fatal(err)
	}
	return m
}

func (h *harness) id(code string) string {
	h.t.Helper()
	a, ok := h.acc[code]
	if !ok {
		h.t.Fatalf("account %s missing", code)
	}
	return a.ID
}

func (h *harness) ok(err error) {
	h.t.Helper()
	if err != nil {
		h.t.Fatal(err)
	}
}

// wantErr asserts a user-facing error with the exact message.
func (h *harness) wantErr(err error, msg string) {
	h.t.Helper()
	if err == nil {
		h.t.Fatalf("want error %q, got nil", msg)
	}
	if got := apperr.As(err).Message; got != msg {
		h.t.Fatalf("want error %q, got %q", msg, got)
	}
}

func line(account, side string, rupiah int64) domain.Line {
	return domain.Line{AccountID: account, Side: side, Amount: domain.Rupiah(rupiah)}
}

func rp(n int64) domain.Money { return domain.Rupiah(n) }

func principal(userID string, perms ...string) authn.Principal {
	return authn.Principal{UserID: userID, Role: authn.RoleMember, Perms: perms}
}

func ptime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
