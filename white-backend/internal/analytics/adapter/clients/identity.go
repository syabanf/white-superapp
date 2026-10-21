// Package clients wraps the gRPC clients of the services analytics depends
// on (identity for access checks, integrations for credentials and tokens)
// and keeps the short caches the architecture prescribes.
package clients

import (
	"context"
	"sync"
	"time"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

const accessTTL = 30 * time.Second

type accessEntry struct {
	access  domain.ClientAccess
	expires time.Time
}

// Identity answers CheckClientAccess with a 30 s cache per (user, client).
type Identity struct {
	client identityv1.AccessServiceClient
	now    func() time.Time
	mu     sync.Mutex
	cache  map[string]accessEntry
}

func NewIdentity(conn *grpc.ClientConn) *Identity {
	return &Identity{client: identityv1.NewAccessServiceClient(conn), now: time.Now, cache: map[string]accessEntry{}}
}

func (c *Identity) Check(ctx context.Context, userID, clientID string) (*domain.ClientAccess, error) {
	key := userID + "|" + clientID
	c.mu.Lock()
	if e, ok := c.cache[key]; ok && c.now().Before(e.expires) {
		c.mu.Unlock()
		a := e.access
		return &a, nil
	}
	c.mu.Unlock()
	res, err := c.client.CheckClientAccess(ctx, &identityv1.CheckClientAccessRequest{UserId: userID, Client: &identityv1.CheckClientAccessRequest_ClientId{ClientId: clientID}})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	a := domain.ClientAccess{ClientID: res.GetClient().GetId(), Slug: res.GetClient().GetSlug(), Currency: res.GetClient().GetCurrency(), CanManage: res.GetCanManage()}
	c.mu.Lock()
	c.cache[key] = accessEntry{access: a, expires: c.now().Add(accessTTL)}
	c.mu.Unlock()
	return &a, nil
}
