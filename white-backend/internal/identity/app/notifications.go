package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

type Audience int

const (
	AudienceNone Audience = iota
	AudienceClientAll
	AudienceClientApprovers
)

type NotifyInput struct {
	UserIDs      []string
	ClientID     string
	Audience     Audience
	ExceptUserID string
	Type         string
	Title        string
	Body         string
	Href         string
}

type Notifications struct {
	store domain.Store
}

func NewNotifications(store domain.Store) *Notifications {
	return &Notifications{store: store}
}

func (s *Notifications) Feed(ctx context.Context, userID string, limit int32) ([]domain.Notification, int32, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.store.ListNotifications(ctx, userID, limit)
}

// MarkRead is a silent no-op for ids that belong to someone else.
func (s *Notifications) MarkRead(ctx context.Context, userID, id string) error {
	if id == "" {
		return apperr.New(apperr.Validation, "")
	}
	return s.store.MarkRead(ctx, id, userID)
}

func (s *Notifications) MarkAllRead(ctx context.Context, userID string) (int32, error) {
	return s.store.MarkAllRead(ctx, userID)
}

// Notify resolves the audience, dedupes, drops the excluded user and inserts
// one row per recipient. Returns how many rows were written.
func (s *Notifications) Notify(ctx context.Context, in NotifyInput) (int32, error) {
	if in.Type == "" || in.Title == "" {
		return 0, apperr.New(apperr.Validation, "type dan title wajib diisi")
	}
	recipients := append([]string{}, in.UserIDs...)
	if in.Audience != AudienceNone {
		if in.ClientID == "" {
			return 0, apperr.New(apperr.Validation, "client_id wajib untuk audience")
		}
		idsFromClient, err := s.store.ListAudience(ctx, in.ClientID, in.Audience == AudienceClientApprovers)
		if err != nil {
			return 0, err
		}
		recipients = append(recipients, idsFromClient...)
	}
	seen := map[string]bool{}
	rows := make([]domain.Notification, 0, len(recipients))
	var clientID *string
	if in.ClientID != "" {
		clientID = &in.ClientID
	}
	var href *string
	if in.Href != "" {
		href = &in.Href
	}
	for _, id := range recipients {
		if id == "" || id == in.ExceptUserID || seen[id] {
			continue
		}
		seen[id] = true
		rows = append(rows, domain.Notification{ID: ids.New(), UserID: id, ClientID: clientID, Type: in.Type, Title: in.Title, Body: in.Body, Href: href})
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return s.store.InsertNotifications(ctx, rows)
}

// Retention mirrors sync/policy.ts for the rows identity owns.
const (
	RetentionReadNotifications = 30 * 24 * time.Hour
	RetentionAnyNotifications  = 90 * 24 * time.Hour
)

func (s *Notifications) RunRetention(ctx context.Context, now time.Time) (int64, error) {
	read, err := s.store.DeleteReadBefore(ctx, now.Add(-RetentionReadNotifications))
	if err != nil {
		return 0, err
	}
	old, err := s.store.DeleteCreatedBefore(ctx, now.Add(-RetentionAnyNotifications))
	if err != nil {
		return read, err
	}
	expired, err := s.store.DeleteExpiredSessions(ctx, now)
	if err != nil {
		return read + old, err
	}
	return read + old + expired, nil
}
