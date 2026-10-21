package clients

import (
	"context"
	"sync"
	"time"

	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/providers/seodata"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

const credentialsTTL = 60 * time.Second

// Integrations fetches the DATAFORSEO and APIFY credentials, cached for
// 60 s and busted by white.integration.updated.
type Integrations struct {
	client integrationsv1.SettingsServiceClient
	clk    clock.Clock

	mu        sync.Mutex
	creds     seodata.Credentials
	fetchedAt time.Time
}

func NewIntegrations(conn grpc.ClientConnInterface, clk clock.Clock) *Integrations {
	return &Integrations{client: integrationsv1.NewSettingsServiceClient(conn), clk: clk}
}

// Credentials implements seodata.CredentialsFunc.
func (c *Integrations) Credentials(ctx context.Context) (seodata.Credentials, error) {
	now := c.clk.Now()
	c.mu.Lock()
	if !c.fetchedAt.IsZero() && now.Sub(c.fetchedAt) < credentialsTTL {
		creds := c.creds
		c.mu.Unlock()
		return creds, nil
	}
	c.mu.Unlock()
	var creds seodata.Credentials
	dfs, err := c.client.GetProviderCredentials(ctx, &integrationsv1.GetProviderCredentialsRequest{Key: integrationsv1.IntegrationKey_DATAFORSEO})
	if err != nil {
		return creds, grpcx.FromStatus(err)
	}
	if dfs.GetConfigured() {
		creds.DataForSEOLogin, creds.DataForSEOPassword = dfs.GetPublicId(), dfs.GetSecret()
	}
	apify, err := c.client.GetProviderCredentials(ctx, &integrationsv1.GetProviderCredentialsRequest{Key: integrationsv1.IntegrationKey_APIFY})
	if err != nil {
		return creds, grpcx.FromStatus(err)
	}
	if apify.GetConfigured() {
		creds.ApifyToken, creds.ApifyActors = apify.GetSecret(), apify.GetPublicId()
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
