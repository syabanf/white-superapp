package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
)

// Audit stores what the gateway reports and lets admins read it. The log is
// append-only: no use case updates or deletes a row.
type Audit struct{ store domain.Store }

func NewAudit(store domain.Store) *Audit { return &Audit{store: store} }

// Record is idempotent per event id, so a redelivered event adds nothing.
func (a *Audit) Record(ctx context.Context, id string, p events.AuditRecorded) error {
	return a.store.InsertAuditLog(ctx, domain.AuditLog{ID: id, At: p.At, UserID: p.UserID, UserName: p.UserName, Role: p.Role,
		Method: p.Method, Path: p.Path, Status: p.Status, IP: p.IP, UserAgent: p.UserAgent})
}

// AuditFilter narrows the log. To is exclusive.
type AuditFilter struct {
	Query      string
	FailedOnly bool
	From, To   *time.Time
	Page, Size int
}

func (a *Audit) List(ctx context.Context, f AuditFilter) ([]domain.AuditLog, int, error) {
	if f.Size <= 0 || f.Size > 100 {
		f.Size = 50
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	return a.store.ListAuditLogs(ctx, f.Query, f.FailedOnly, f.From, f.To, f.Size, (f.Page-1)*f.Size)
}
