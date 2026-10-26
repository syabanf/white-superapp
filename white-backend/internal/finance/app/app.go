// Package app holds the finance use cases: chart of accounts and mappings,
// fiscal calendar, journals and automatic posting, cash and bank,
// payables and receivables, reports, reimbursement claims, financial
// analysis and maintenance.
package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
)

// Jakarta is the calendar every finance date follows.
var Jakarta = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}()

// Employee is what hr's EmployeeService tells finance about a person.
type Employee struct {
	ID             string
	UserID         string
	Name           string
	DepartmentName string
	ManagerID      string
}

// HR resolves employees (hr EmployeeService).
type HR interface {
	// EmployeeByUser returns nil when the user has no employee record.
	EmployeeByUser(ctx context.Context, userID string) (*Employee, error)
	Employee(ctx context.Context, id string) (*Employee, error)
}

// NotifyInput is one in-app notification (identity NotificationService).
type NotifyInput struct {
	UserIDs []string
	Type    string
	Title   string
	Body    string
	Href    string
}

// Identity is the identity service seen from finance.
type Identity interface {
	Notify(ctx context.Context, in NotifyInput) error
	UserName(ctx context.Context, userID string) string
	// PermissionHolders lists active users holding perm (admins included).
	PermissionHolders(ctx context.Context, perm string) ([]string, error)
}

// Storage keeps receipt files.
type Storage interface {
	Put(ctx context.Context, key string, data []byte) error
	// Get returns nil when the object is missing.
	Get(ctx context.Context, key string) (data []byte, etag string, err error)
	Delete(ctx context.Context, key string) error
}

// InsightInput is what the CFO memo prompt receives.
type InsightInput struct {
	From, To string
	Language string
	Data     map[string]any
}

// InsightGenerator writes the AI memo; Model is stored with it.
type InsightGenerator interface {
	Generate(ctx context.Context, in InsightInput) (content, model string, err error)
}

// SheetReader returns the rows of the "coa" sheet (or the first one).
type SheetReader func(data []byte) ([][]string, error)

type Deps struct {
	Store     domain.Store
	Identity  Identity
	HR        HR
	Storage   Storage
	Insights  InsightGenerator
	Events    events.Publisher
	ReadSheet SheetReader
	Clock     clock.Clock
	Logger    *slog.Logger
}

func (d Deps) now() time.Time { return d.Clock.Now() }

// today is the Jakarta calendar date.
func (d Deps) today() string { return d.Clock.Now().In(Jakarta).Format(time.DateOnly) }

// notify never fails the use case: notifications are best effort.
func (d Deps) notify(ctx context.Context, in NotifyInput) {
	if d.Identity == nil || len(domain.UniqueIDs(in.UserIDs)) == 0 {
		return
	}
	in.UserIDs = domain.UniqueIDs(in.UserIDs)
	if err := d.Identity.Notify(ctx, in); err != nil && d.Logger != nil {
		d.Logger.Warn("notify failed", "type", in.Type, "err", err)
	}
}

// financeApprovers are the users notified about the finance approval step;
// a lookup failure only skips the notice.
func (d Deps) financeApprovers(ctx context.Context) []string {
	if d.Identity == nil {
		return nil
	}
	ids, err := d.Identity.PermissionHolders(ctx, authn.PermFinanceApprove)
	if err != nil && d.Logger != nil {
		d.Logger.Warn("approver lookup failed", "err", err)
	}
	return ids
}

// ─── shared input helpers ───────────────────────────────────

func trim(s string) string { return strings.TrimSpace(s) }

// parseDay validates a YYYY-MM-DD field.
func parseDay(field, s string) (string, error) {
	s = trim(s)
	if _, err := time.Parse(time.DateOnly, s); err != nil {
		return "", apperr.Field(field, "Tanggal tidak valid (format YYYY-MM-DD).")
	}
	return s, nil
}

// optionalDay is "" when empty, else a validated day.
func optionalDay(field, s string) (string, error) {
	if trim(s) == "" {
		return "", nil
	}
	return parseDay(field, s)
}

// Page bounds a list request: 1-based page, size 1..100 (default 25).
func Page(page, size int) (limit, offset int) {
	if size <= 0 {
		size = 25
	}
	if size > 100 {
		size = 100
	}
	if page < 1 {
		page = 1
	}
	return size, (page - 1) * size
}

// monthRange is the first and last day of a YYYY-MM month ("" = current).
func (d Deps) monthRange(month string) (string, string, error) {
	if month == "" {
		month = d.today()[:7]
	}
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", apperr.Field("month", "Bulan tidak valid (format YYYY-MM).")
	}
	return t.Format(time.DateOnly), t.AddDate(0, 1, -1).Format(time.DateOnly), nil
}
