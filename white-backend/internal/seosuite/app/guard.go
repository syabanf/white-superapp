package app

import (
	"context"
	"sort"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Guard is guardProperty from access.ts: client membership through identity
// (MANAGER or ADMIN for writes) and the property lookup through analytics.
type Guard struct {
	access AccessChecker
	props  PropertySource
}

func NewGuard(access AccessChecker, props PropertySource) *Guard {
	return &Guard{access: access, props: props}
}

// Client checks membership; manage additionally requires ADMIN or MANAGER.
func (g *Guard) Client(ctx context.Context, userID, clientID string, manage bool) (*Access, error) {
	acc, err := g.access.Check(ctx, userID, clientID)
	if err != nil {
		if apperr.Is(err, apperr.Unauthorized) {
			return nil, apperr.New(apperr.Unauthorized, msgUnauthorizedClient)
		}
		return nil, err
	}
	if manage && !acc.CanManage {
		return nil, apperr.New(apperr.Unauthorized, msgViewer)
	}
	return acc, nil
}

// Property is Client plus "the property must belong to the client".
func (g *Guard) Property(ctx context.Context, userID, clientID, propertyID string, manage bool) (*Property, error) {
	if _, err := g.Client(ctx, userID, clientID, manage); err != nil {
		return nil, err
	}
	return g.FindProperty(ctx, clientID, propertyID)
}

func (g *Guard) FindProperty(ctx context.Context, clientID, propertyID string) (*Property, error) {
	props, err := g.props.ListProperties(ctx, clientID)
	if err != nil {
		return nil, err
	}
	for _, p := range props {
		if p.ID == propertyID {
			return &p, nil
		}
	}
	return nil, apperr.New(apperr.NotFound, msgNoProperty)
}

// SuiteProperty is getSuiteProperty: the client's oldest property.
func (g *Guard) SuiteProperty(ctx context.Context, clientID string) (*Property, error) {
	props, err := g.props.ListProperties(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if len(props) == 0 {
		return nil, apperr.New(apperr.NotFound, msgNoProperty)
	}
	sort.SliceStable(props, func(i, j int) bool { return props[i].CreatedAt.Before(props[j].CreatedAt) })
	return &props[0], nil
}
