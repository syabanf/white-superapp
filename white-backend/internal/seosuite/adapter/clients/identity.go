// Package clients wraps the gRPC clients of the services seosuite depends
// on: identity (access checks, notifications), integrations (provider
// credentials) and analytics (SEO properties). Caches live here.
package clients

import (
	"context"
	"sync"
	"time"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

const accessTTL = 30 * time.Second

type accessEntry struct {
	access app.Access
	at     time.Time
}

// Identity answers CheckClientAccess (cached 30 s per user and client) and
// fans notifications out to a client's audience.
type Identity struct {
	access identityv1.AccessServiceClient
	notif  identityv1.NotificationServiceClient
	clk    clock.Clock

	mu    sync.Mutex
	cache map[string]accessEntry
}

func NewIdentity(conn grpc.ClientConnInterface, clk clock.Clock) *Identity {
	return &Identity{access: identityv1.NewAccessServiceClient(conn), notif: identityv1.NewNotificationServiceClient(conn), clk: clk, cache: map[string]accessEntry{}}
}

func (c *Identity) Check(ctx context.Context, userID, clientID string) (*app.Access, error) {
	key := userID + "|" + clientID
	now := c.clk.Now()
	c.mu.Lock()
	if e, ok := c.cache[key]; ok && now.Sub(e.at) < accessTTL {
		c.mu.Unlock()
		acc := e.access
		return &acc, nil
	}
	c.mu.Unlock()
	res, err := c.access.CheckClientAccess(ctx, &identityv1.CheckClientAccessRequest{UserId: userID, Client: &identityv1.CheckClientAccessRequest_ClientId{ClientId: clientID}})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	role := "VIEWER"
	switch res.GetRole() {
	case identityv1.ClientAccess_ADMIN_OVERRIDE:
		role = "ADMIN"
	case identityv1.ClientAccess_MANAGER:
		role = "MANAGER"
	}
	acc := app.Access{Role: role, CanManage: res.GetCanManage(), ClientSlug: res.GetClient().GetSlug()}
	c.mu.Lock()
	c.cache[key] = accessEntry{access: acc, at: now}
	c.mu.Unlock()
	return &acc, nil
}

func (c *Identity) Notify(ctx context.Context, n app.Notification) error {
	_, err := c.notif.Notify(ctx, &identityv1.NotifyRequest{
		ClientId: n.ClientID, Audience: identityv1.NotifyRequest_CLIENT_ALL,
		Type: n.Type, Title: n.Title, Body: n.Body, Href: n.Href,
	})
	if err != nil {
		return grpcx.FromStatus(err)
	}
	return nil
}
