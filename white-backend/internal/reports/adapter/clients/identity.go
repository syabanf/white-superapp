// Package clients wraps the gRPC clients of identity, integrations,
// analytics, publishing and seosuite behind the app ports. Access checks
// are cached 30 s and OpenRouter credentials 60 s.
package clients

import (
	"context"
	"sync"
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

const accessTTL = 30 * time.Second

type accessEntry struct {
	access app.Access
	at     time.Time
}

type Identity struct {
	access identityv1.AccessServiceClient
	share  identityv1.ShareServiceClient
	clk    clock.Clock

	mu    sync.Mutex
	cache map[string]accessEntry
}

func NewIdentity(conn grpc.ClientConnInterface, clk clock.Clock) *Identity {
	return &Identity{
		access: identityv1.NewAccessServiceClient(conn),
		share:  identityv1.NewShareServiceClient(conn),
		clk:    clk,
		cache:  map[string]accessEntry{},
	}
}

func clientInfo(c *identityv1.Client) app.ClientInfo {
	modules := make([]string, 0, len(c.GetShareModules()))
	for _, m := range c.GetShareModules() {
		if m != commonv1.ModuleKey_MODULE_KEY_UNSPECIFIED {
			modules = append(modules, m.String())
		}
	}
	currency := c.GetCurrency()
	if currency == "" {
		currency = "IDR"
	}
	return app.ClientInfo{ID: c.GetId(), Name: c.GetName(), Slug: c.GetSlug(), Currency: currency, ShareModules: modules}
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
	acc := app.Access{Role: role, CanManage: res.GetCanManage(), Client: clientInfo(res.GetClient())}
	c.mu.Lock()
	c.cache[key] = accessEntry{access: acc, at: now}
	c.mu.Unlock()
	return &acc, nil
}

func (c *Identity) UserName(ctx context.Context, userID string) (string, bool, error) {
	if userID == "" {
		return "", false, nil
	}
	u, err := c.access.GetUser(ctx, &identityv1.GetUserRequest{UserId: userID})
	if err != nil {
		e := grpcx.FromStatus(err)
		if e.Code == apperr.NotFound {
			return "", false, nil
		}
		return "", false, e
	}
	return u.GetName(), true, nil
}

func (c *Identity) ShareContext(ctx context.Context, slug, unlockToken string) (*app.ShareContext, error) {
	res, err := c.share.GetShareContext(ctx, &identityv1.GetShareContextRequest{Slug: slug, UnlockToken: unlockToken})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := &app.ShareContext{Status: app.ShareNotFound}
	switch res.GetStatus() {
	case identityv1.ShareContext_OK:
		out.Status = app.ShareOK
	case identityv1.ShareContext_LOCKED:
		out.Status = app.ShareLocked
	case identityv1.ShareContext_DISABLED:
		out.Status = app.ShareDisabled
	}
	if out.Status != app.ShareNotFound && res.GetClient() != nil {
		out.Client = clientInfo(res.GetClient())
	}
	return out, nil
}
