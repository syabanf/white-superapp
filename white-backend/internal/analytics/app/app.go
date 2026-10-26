// Package app holds the analytics use cases: sources, overview, social,
// SEO, ads and the report data other services compose.
package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/sync"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

const (
	msgNoAccessClient = "Anda tidak punya akses ke klien ini."
	msgNoProperty     = "Properti SEO tidak ditemukan."
	msgTooLong        = "Terlalu panjang"
)

// Deps is what every use case struct shares.
type Deps struct {
	Store  domain.Store
	Access domain.AccessChecker
	Integ  domain.Integrations
	Syncer *sync.Syncer
	Clock  clock.Clock
}

func (d Deps) now() time.Time { return d.Clock.Now() }

// view resolves any membership. A missing client stays NOT_FOUND so read
// pages can tell it apart; actions use act/manage.
func (d Deps) view(ctx context.Context, userID, clientID string) (*domain.ClientAccess, error) {
	if clientID == "" {
		return nil, apperr.New(apperr.Validation, "")
	}
	return d.Access.Check(ctx, userID, clientID)
}

// act is the old clientAccess(): any member, and an unknown client is
// UNAUTHORIZED like a missing membership.
func (d Deps) act(ctx context.Context, userID, clientID string) (*domain.ClientAccess, error) {
	a, err := d.view(ctx, userID, clientID)
	if err != nil {
		if apperr.Is(err, apperr.NotFound) {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
		return nil, err
	}
	return a, nil
}

// manage is the ADMIN-or-MANAGER gate (manageContext / requireClientManage).
func (d Deps) manage(ctx context.Context, userID, clientID string) (*domain.ClientAccess, error) {
	a, err := d.act(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	if !a.CanManage {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return a, nil
}

// seoGuard is the SEO actions' guard: any member, then the property must
// belong to the client.
func (d Deps) seoGuard(ctx context.Context, userID, clientID, propertyID string) (*domain.SeoProperty, error) {
	if _, err := d.view(ctx, userID, clientID); err != nil {
		if apperr.Is(err, apperr.Unauthorized) {
			return nil, apperr.New(apperr.Unauthorized, msgNoAccessClient)
		}
		if !apperr.Is(err, apperr.NotFound) {
			return nil, err
		}
	}
	p, err := d.Store.GetSeoProperty(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.ClientID != clientID {
		return nil, apperr.New(apperr.NotFound, msgNoProperty)
	}
	return p, nil
}

// Window is a resolved current range plus its comparison period.
type Window struct {
	Current  dates.Range
	Previous dates.Range
}

func WindowOf(r dates.Resolved) Window { return Window{Current: r.Current, Previous: r.Previous} }

func endExclusive(t time.Time) time.Time { return dates.AddDays(t, 1) }

func ptr[T any](v T) *T { return &v }
