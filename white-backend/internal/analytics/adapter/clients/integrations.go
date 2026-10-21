package clients

import (
	"context"
	"sync"
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

const credentialsTTL = 60 * time.Second

type credEntry struct {
	creds   domain.Credentials
	expires time.Time
}

// Integrations fetches provider credentials (cached 60 s, busted by
// integration.updated), access tokens and connections.
type Integrations struct {
	settings    integrationsv1.SettingsServiceClient
	connections integrationsv1.ConnectionServiceClient
	now         func() time.Time
	mu          sync.Mutex
	cache       map[string]credEntry
}

func NewIntegrations(conn *grpc.ClientConn) *Integrations {
	return &Integrations{settings: integrationsv1.NewSettingsServiceClient(conn), connections: integrationsv1.NewConnectionServiceClient(conn), now: time.Now, cache: map[string]credEntry{}}
}

var integrationKeys = map[string]integrationsv1.IntegrationKey{
	domain.IntegrationMeta:   integrationsv1.IntegrationKey_META,
	domain.IntegrationGoogle: integrationsv1.IntegrationKey_GOOGLE,
	domain.IntegrationApify:  integrationsv1.IntegrationKey_APIFY,
}

func (c *Integrations) Credentials(ctx context.Context, key string) (domain.Credentials, error) {
	c.mu.Lock()
	if e, ok := c.cache[key]; ok && c.now().Before(e.expires) {
		c.mu.Unlock()
		return e.creds, nil
	}
	c.mu.Unlock()
	res, err := c.settings.GetProviderCredentials(ctx, &integrationsv1.GetProviderCredentialsRequest{Key: integrationKeys[key]})
	if err != nil {
		return domain.Credentials{}, grpcx.FromStatus(err)
	}
	creds := domain.Credentials{Configured: res.GetConfigured(), PublicID: res.GetPublicId(), Secret: res.GetSecret()}
	c.mu.Lock()
	c.cache[key] = credEntry{creds: creds, expires: c.now().Add(credentialsTTL)}
	c.mu.Unlock()
	return creds, nil
}

// Bust drops the credential cache (integration.updated).
func (c *Integrations) Bust() {
	c.mu.Lock()
	c.cache = map[string]credEntry{}
	c.mu.Unlock()
}

func (c *Integrations) AccessToken(ctx context.Context, connectionID string) (string, error) {
	res, err := c.connections.GetAccessToken(ctx, &integrationsv1.GetAccessTokenRequest{ConnectionId: connectionID})
	if err != nil {
		e := grpcx.FromStatus(err)
		if e.Code == apperr.NotFound {
			return "", nil
		}
		return "", e
	}
	return res.GetToken(), nil
}

func (c *Integrations) FindClientConnection(ctx context.Context, clientID, provider string) (string, error) {
	p := commonv1.ConnectionProvider_META
	if provider == domain.IntegrationGoogle {
		p = commonv1.ConnectionProvider_GOOGLE
	}
	res, err := c.connections.FindClientConnection(ctx, &integrationsv1.FindClientConnectionRequest{ClientId: clientID, Provider: p})
	if err != nil {
		return "", grpcx.FromStatus(err)
	}
	if res.GetConnection() == nil {
		return "", nil
	}
	return res.GetConnection().GetId(), nil
}
