// Package app holds the hr use cases: people and master data, workforce
// (attendance, schedules, leave, overtime, contracts, announcements,
// on/offboarding), compensation (salaries, payroll runs, payslips, loans),
// self service, reports and the event-driven maintenance jobs.
package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// User-facing strings shared by several areas.
const (
	msgEmployeeNotFound = "Karyawan tidak ditemukan."
	msgNotLinked        = "Akun ini tidak terhubung ke data karyawan."
	msgInvalidDate      = "Tanggal tidak valid (YYYY-MM-DD)."
	msgInvalidAmount    = "Nominal tidak valid."
	msgNotFound         = "Data tidak ditemukan."
)

// UserInfo is what identity says about a login.
type UserInfo struct {
	ID, Name, Email string
	IsActive        bool
}

// NotifyInput is one in-app notification to explicit recipients.
type NotifyInput struct {
	UserIDs           []string
	Type, Title, Body string
	Href              string
}

// Identity is the slice of the identity service hr uses.
type Identity interface {
	// GetUser returns nil when the user does not exist.
	GetUser(ctx context.Context, userID string) (*UserInfo, error)
	Notify(ctx context.Context, in NotifyInput) error
	// PermissionHolders lists the active users holding perm (admins included).
	PermissionHolders(ctx context.Context, perm string) ([]string, error)
}

// Files stores attendance selfies and employee documents.
type Files interface {
	// Put sniffs data, refuses types outside allowed or above maxBytes, and
	// stores it under prefix. It returns the key and the sniffed type.
	Put(ctx context.Context, prefix string, data []byte, allowed []string, maxBytes int) (key, contentType string, err error)
	Get(ctx context.Context, key string) ([]byte, string, error)
	Delete(ctx context.Context, key string) error
}

// Files errors the app maps to user messages.
var (
	ErrFileType  = errors.New("jenis file tidak didukung")
	ErrFileLarge = errors.New("file terlalu besar")
)

type Deps struct {
	Store    domain.Store
	Identity Identity
	Files    Files
	Events   events.Publisher
	Clock    clock.Clock
	Logger   *slog.Logger
}

func (d Deps) now() time.Time { return d.Clock.Now() }

// today is the WIB calendar date.
func (d Deps) today() string { return hris.TodayWIB(d.now()) }

// Actor is the caller: the principal plus the employee linked to it.
type Actor struct {
	authn.Principal
	EmployeeID string
}

// UserRef is the caller's user id for audit columns.
func (a Actor) UserRef() *string {
	if a.UserID == "" {
		return nil
	}
	id := a.UserID
	return &id
}

// EmployeeRef is the caller's employee id, nil when unlinked.
func (a Actor) EmployeeRef() *string {
	if a.EmployeeID == "" {
		return nil
	}
	id := a.EmployeeID
	return &id
}

// ActorFor resolves the employee linked to the principal.
func (d Deps) ActorFor(ctx context.Context, p authn.Principal) (Actor, error) {
	id, err := d.Store.EmployeeIDByUser(ctx, p.UserID)
	if err != nil {
		return Actor{}, err
	}
	return Actor{Principal: p, EmployeeID: id}, nil
}

// require returns UNAUTHORIZED unless the actor holds one of perms.
func require(a Actor, perms ...string) error {
	for _, p := range perms {
		if a.Can(p) {
			return nil
		}
	}
	return apperr.New(apperr.Unauthorized, "")
}

// self returns the caller's employee id; self.service is required.
func self(a Actor) (string, error) {
	if err := require(a, authn.PermSelfService); err != nil {
		return "", err
	}
	if a.EmployeeID == "" {
		return "", apperr.New(apperr.NotFound, msgNotLinked)
	}
	return a.EmployeeID, nil
}

// RequireSelf is self for the /me handlers.
func RequireSelf(a Actor) (string, error) { return self(a) }

// employeeOrSelf picks the target employee: anyone with one of perms, the
// caller's own record otherwise.
func employeeOrSelf(a Actor, requested string, perms ...string) (string, error) {
	if requested != "" && requested != a.EmployeeID {
		if err := require(a, perms...); err != nil {
			return "", err
		}
		return requested, nil
	}
	return self(a)
}

// forbidden is UNAUTHORIZED with the rule that refused the action.
func forbidden(msg string) error { return apperr.New(apperr.Unauthorized, msg) }

func notFound(msg string) error { return apperr.New(apperr.NotFound, msg) }

func conflict(msg string) error { return apperr.New(apperr.Conflict, msg) }

func invalid(field, msg string) error { return apperr.Field(field, msg) }

func newID() string { return ids.New() }

// optDate validates an optional date field.
func optDate(field, v string) (*string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	if _, ok := hris.ParseDate(v); !ok {
		return nil, invalid(field, msgInvalidDate)
	}
	return &v, nil
}

// reqDate validates a required date field.
func reqDate(field, v string) (string, error) {
	d, err := optDate(field, v)
	if err != nil {
		return "", err
	}
	if d == nil {
		return "", invalid(field, msgInvalidDate)
	}
	return *d, nil
}

// amount parses a non-negative decimal string ("" is 0).
func amount(field, v string) (float64, error) {
	f, ok := domain.ParseMoney(strings.TrimSpace(v))
	if !ok || f < 0 {
		return 0, invalid(field, msgInvalidAmount)
	}
	return f, nil
}

func strPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// uniqueErr maps a unique violation to a conflict with msg.
func uniqueErr(err error, msg string) error {
	var u *domain.ErrUnique
	if errors.As(err, &u) {
		return conflict(msg)
	}
	return err
}

// monthBounds parses "YYYY-MM" into the month's first and last day.
func monthBounds(month string) (string, string, bool) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", false
	}
	start, end := hris.PeriodBounds(int(t.Month()), t.Year())
	return start, end, true
}
