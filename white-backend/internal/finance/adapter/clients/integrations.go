package clients

import (
	"context"
	"time"

	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

// Integrations fetches the OPENROUTER credentials, cached for 60 s and
// busted by white.integration.updated.
type Integrations struct {
	client integrationsv1.SettingsServiceClient
	creds  *ttlCache[openrouter.Credentials]
}

func NewIntegrations(conn grpc.ClientConnInterface) *Integrations {
	return &Integrations{client: integrationsv1.NewSettingsServiceClient(conn), creds: newCache[openrouter.Credentials](60 * time.Second)}
}

// Credentials implements openrouter.CredentialsFunc.
func (c *Integrations) Credentials(ctx context.Context) (openrouter.Credentials, error) {
	if v, ok := c.creds.get("openrouter"); ok {
		return v, nil
	}
	res, err := c.client.GetProviderCredentials(ctx, &integrationsv1.GetProviderCredentialsRequest{Key: integrationsv1.IntegrationKey_OPENROUTER})
	if err != nil {
		return openrouter.Credentials{}, grpcx.FromStatus(err)
	}
	var creds openrouter.Credentials
	if res.GetConfigured() {
		creds = openrouter.Credentials{APIKey: res.GetSecret(), Model: res.GetPublicId()}
	}
	c.creds.put("openrouter", creds)
	return creds, nil
}

func (c *Integrations) Invalidate() { c.creds.clear() }
