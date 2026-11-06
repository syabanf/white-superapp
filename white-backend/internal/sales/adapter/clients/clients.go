// Package clients wraps the identity and integrations gRPC clients behind
// the sales app ports. OpenRouter credentials are cached for 60 s.
package clients

import (
	"context"
	"strconv"
	"sync"
	"time"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
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
	share         identityv1.ShareServiceClient
	serviceToken  string
}

func NewIdentity(conn grpc.ClientConnInterface, serviceToken string) *Identity {
	return &Identity{
		notifications: identityv1.NewNotificationServiceClient(conn),
		users:         identityv1.NewUserServiceClient(conn),
		access:        identityv1.NewAccessServiceClient(conn),
		clients:       identityv1.NewClientServiceClient(conn),
		share:         identityv1.NewShareServiceClient(conn),
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
	return c.Holders(ctx, authn.PermSalesApprove)
}

// UnlockedClient implements app.ShareGate. Only an enabled page with a PIN
// the visitor entered qualifies.
func (c *Identity) UnlockedClient(ctx context.Context, slug, unlockToken string) (*app.ShareClient, error) {
	res, err := c.share.GetShareContext(ctx, &identityv1.GetShareContextRequest{Slug: slug, UnlockToken: unlockToken})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	cl := res.GetClient()
	if res.GetStatus() != identityv1.ShareContext_OK || cl == nil || !cl.GetSharePinSet() {
		return nil, nil
	}
	return &app.ShareClient{ID: cl.GetId(), Name: cl.GetName()}, nil
}

// Holders lists active users holding a permission plus admins.
func (c *Identity) Holders(ctx context.Context, permission string) ([]app.User, error) {
	res, err := c.access.ListPermissionHolders(ctx, &identityv1.ListPermissionHoldersRequest{Permission: permission})
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

// Taxes reads finance tax codes for quotations, with the caller's token.
type Taxes struct {
	client financev1.TaxSetupServiceClient
}

func NewTaxes(conn grpc.ClientConnInterface) *Taxes {
	return &Taxes{client: financev1.NewTaxSetupServiceClient(conn)}
}

func (c *Taxes) list(ctx context.Context) ([]*financev1.TaxCode, error) {
	res, err := c.client.ListTaxCodes(ctx, &financev1.ListTaxMasterRequest{ActiveOnly: true})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	return res.GetTaxes(), nil
}

func saleVAT(t *financev1.TaxCode) *app.TaxInfo {
	rate, err := strconv.ParseFloat(t.GetRate(), 64)
	if err != nil || t.GetBaseDenominator() <= 0 {
		return nil
	}
	return &app.TaxInfo{Code: t.GetCode(), Name: t.GetName(), EffectivePercent: rate * float64(t.GetBaseNumerator()) / float64(t.GetBaseDenominator())}
}

// VATByCode returns an active VAT code usable on sales, or nil.
func (c *Taxes) VATByCode(ctx context.Context, code string) (*app.TaxInfo, error) {
	rows, err := c.list(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range rows {
		if t.GetCode() == code && t.GetKind() == "VAT" && t.GetScope() != "PURCHASE" {
			return saleVAT(t), nil
		}
	}
	return nil, nil
}

// DefaultSaleVAT returns the code marked as the sales default, or nil.
func (c *Taxes) DefaultSaleVAT(ctx context.Context) (*app.TaxInfo, error) {
	rows, err := c.list(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range rows {
		if t.GetIsDefaultSale() && t.GetKind() == "VAT" {
			return saleVAT(t), nil
		}
	}
	return nil, nil
}

// Company reads Finance's company profile for letterheads and the portal,
// cached for a minute. When Finance cannot answer it serves the last good
// profile, or the deployed defaults.
type Company struct {
	client   financev1.CompanyServiceClient
	fallback app.Company
	clk      clock.Clock
	log      func(msg string, args ...any)

	mu        sync.Mutex
	cached    *app.Company
	fetchedAt time.Time
}

func NewCompany(conn grpc.ClientConnInterface, fallback app.Company, clk clock.Clock, log func(msg string, args ...any)) *Company {
	return &Company{client: financev1.NewCompanyServiceClient(conn), fallback: fallback, clk: clk, log: log}
}

const companyTTL = 60 * time.Second

func (c *Company) Company(ctx context.Context) app.Company {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cached != nil && c.clk.Now().Sub(c.fetchedAt) < companyTTL {
		return *c.cached
	}
	res, err := c.client.GetCompanyProfile(ctx, &emptypb.Empty{})
	if err != nil {
		if c.log != nil {
			c.log("company profile unavailable", "err", err)
		}
		if c.cached != nil {
			return *c.cached
		}
		return c.fallback
	}
	co := app.Company{LegalName: res.LegalName, Address: res.Address, Email: res.Email, Phone: res.Phone, NPWP: res.Npwp, BankLine: res.BankLine}
	c.cached, c.fetchedAt = &co, c.clk.Now()
	return co
}
