// Package clients wraps the gRPC clients of other services. The identity
// access check is cached for 30 s per (user, client).
package clients

import (
	"context"
	"sync"
	"time"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

const accessTTL = 30 * time.Second

type cached struct {
	access  domain.ClientAccess
	expires time.Time
}

type Access struct {
	c   identityv1.AccessServiceClient
	clk clock.Clock
	mu  sync.Mutex
	// entries is keyed by userID + "|" + clientID + "|" + slug.
	entries map[string]cached
}

func NewAccess(conn grpc.ClientConnInterface, clk clock.Clock) *Access {
	return &Access{c: identityv1.NewAccessServiceClient(conn), clk: clk, entries: map[string]cached{}}
}

// Check resolves the user's role on a client given by id or slug. Only
// successful answers are cached; denials are re-asked every time.
func (a *Access) Check(ctx context.Context, userID, clientID, slug string) (*domain.ClientAccess, error) {
	key := userID + "|" + clientID + "|" + slug
	now := a.clk.Now()
	a.mu.Lock()
	if e, ok := a.entries[key]; ok && now.Before(e.expires) {
		a.mu.Unlock()
		acc := e.access
		return &acc, nil
	}
	a.mu.Unlock()

	req := &identityv1.CheckClientAccessRequest{UserId: userID}
	if clientID != "" {
		req.Client = &identityv1.CheckClientAccessRequest_ClientId{ClientId: clientID}
	} else {
		req.Client = &identityv1.CheckClientAccessRequest_Slug{Slug: slug}
	}
	res, err := a.c.CheckClientAccess(ctx, req)
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	acc := domain.ClientAccess{ClientID: res.GetClient().GetId(), Slug: res.GetClient().GetSlug(), CanManage: res.GetCanManage()}
	a.mu.Lock()
	a.entries[key] = cached{access: acc, expires: now.Add(accessTTL)}
	a.mu.Unlock()
	return &acc, nil
}
