// Package clients wraps the identity and integrations gRPC clients behind
// the sales app ports. OpenRouter credentials are cached for 60 s.
package clients

import (
	"context"
	"sync"
	"time"

	identityv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/identity/v1"
	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Identity implements app.Notifier, app.Directory and app.MarketingClients.
// ServiceToken (SALES_SERVICE_TOKEN, an ADMIN access token) is used for
// CreateClient when set; otherwise the caller's token is forwarded.
type Identity struct {
	notifications identityv1.NotificationServiceClient
	users         identityv1.UserServiceClient
	access        identityv1.AccessServiceClient
	clients       identityv1.ClientServiceClient
	serviceToken  string
}

func NewIdentity(conn grpc.ClientConnInterface, serviceToken string) *Identity {
	return &Identity{
		notifications: identityv1.NewNotificationServiceClient(conn),
		users:         identityv1.NewUserServiceClient(conn),
		access:        identityv1.NewAccessServiceClient(conn),
		clients:       identityv1.NewClientServiceClient(conn),
		serviceToken:  serviceToken,
	}
}

func (c *Identity) admin(ctx context.Context) context.Context {
	if c.serviceToken == "" {
		return ctx
	}
	return grpcx.WithBearer(ctx, c.serviceToken)
}

func (c *Identity) Notify(ctx context.Context, n app.Notification) error {
	_, err := c.notifications.Notify(ctx, &identityv1.NotifyRequest{UserIds: n.UserIDs, Type: n.Type, Title: n.Title, Body: n.Body, Href: n.Href})
	if err != nil {
		return grpcx.FromStatus(err)
	}
	return nil
}

func (c *Identity) ListUsers(ctx context.Context) ([]app.User, error) {
	res, err := c.users.ListUserOptions(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := make([]app.User, 0, len(res.GetUsers()))
	for _, u := range res.GetUsers() {
		out = append(out, app.User{ID: u.GetId(), Name: u.GetName(), Email: u.GetEmail()})
	}
	return out, nil
}

// Approvers lists active users holding sales.approve (or sales.*) plus
// admins, through identity's internal AccessService.
func (c *Identity) Approvers(ctx context.Context) ([]app.User, error) {
	res, err := c.access.ListPermissionHolders(ctx, &identityv1.ListPermissionHoldersRequest{Permission: authn.PermSalesApprove})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := make([]app.User, 0, len(res.GetUsers()))
	for _, u := range res.GetUsers() {
		out = append(out, app.User{ID: u.GetId(), Name: u.GetName(), Email: u.GetEmail(), Admin: u.GetAdmin()})
	}
	return out, nil
}

func (c *Identity) CreateClient(ctx context.Context, in app.ClientInput) (string, error) {
	res, err := c.clients.CreateClient(c.admin(ctx), &identityv1.CreateClientRequest{Profile: &identityv1.ClientProfile{Name: in.Name, Slug: in.Slug, WebsiteUrl: in.WebsiteURL, Industry: in.Industry, Currency: "IDR", Timezone: "Asia/Jakarta"}})
	if err != nil {
		return "", grpcx.FromStatus(err)
	}
	return res.GetId(), nil
}

const credentialsTTL = 60 * time.Second

// Integrations fetches the OPENROUTER credentials, cached for 60 s and
// busted by white.integration.updated.
type Integrations struct {
	client integrationsv1.SettingsServiceClient
	clk    clock.Clock

	mu        sync.Mutex
	creds     openrouter.Credentials
	fetchedAt time.Time
}

func NewIntegrations(conn grpc.ClientConnInterface, clk clock.Clock) *Integrations {
	return &Integrations{client: integrationsv1.NewSettingsServiceClient(conn), clk: clk}
}

// Credentials implements openrouter.CredentialsFunc.
func (c *Integrations) Credentials(ctx context.Context) (openrouter.Credentials, error) {
	now := c.clk.Now()
	c.mu.Lock()
	if !c.fetchedAt.IsZero() && now.Sub(c.fetchedAt) < credentialsTTL {
		creds := c.creds
		c.mu.Unlock()
		return creds, nil
	}
	c.mu.Unlock()
	res, err := c.client.GetProviderCredentials(ctx, &integrationsv1.GetProviderCredentialsRequest{Key: integrationsv1.IntegrationKey_OPENROUTER})
	if err != nil {
		return openrouter.Credentials{}, grpcx.FromStatus(err)
	}
	var creds openrouter.Credentials
	if res.GetConfigured() {
		creds = openrouter.Credentials{APIKey: res.GetSecret(), Model: res.GetPublicId()}
	}
	c.mu.Lock()
	c.creds, c.fetchedAt = creds, now
	c.mu.Unlock()
	return creds, nil
}

// Invalidate drops the cache so the next call sees new keys.
func (c *Integrations) Invalidate() {
	c.mu.Lock()
	c.fetchedAt = time.Time{}
	c.mu.Unlock()
}
