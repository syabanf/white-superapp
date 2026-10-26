package clients

import (
	"context"
	"sync"
	"time"

	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/providers/ai"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

const credentialsTTL = 60 * time.Second

// Integrations fetches the OPENROUTER credentials, cached for 60 s and
// busted by white.integration.updated.
type Integrations struct {
	client integrationsv1.SettingsServiceClient
	clk    clock.Clock

	mu        sync.Mutex
	creds     ai.Credentials
	fetchedAt time.Time
}

func NewIntegrations(conn grpc.ClientConnInterface, clk clock.Clock) *Integrations {
	return &Integrations{client: integrationsv1.NewSettingsServiceClient(conn), clk: clk}
}

// Credentials implements ai.CredentialsFunc.
func (c *Integrations) Credentials(ctx context.Context) (ai.Credentials, error) {
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
		return ai.Credentials{}, grpcx.FromStatus(err)
	}
	var creds ai.Credentials
	if res.GetConfigured() {
		creds = ai.Credentials{APIKey: res.GetSecret(), Model: res.GetPublicId()}
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
