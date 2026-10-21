package clients

import (
	"context"
	"log/slog"
	"time"

	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

type Integrations struct {
	settings    integrationsv1.SettingsServiceClient
	connections integrationsv1.ConnectionServiceClient
	configured  *ttlCache[bool]
	tokens      *ttlCache[string]
	logger      *slog.Logger
}

func NewIntegrations(conn *grpc.ClientConn, logger *slog.Logger) *Integrations {
	return &Integrations{
		settings:    integrationsv1.NewSettingsServiceClient(conn),
		connections: integrationsv1.NewConnectionServiceClient(conn),
		configured:  newCache[bool](60 * time.Second),
		tokens:      newCache[string](60 * time.Second),
		logger:      logger,
	}
}

// MetaConfigured is false when the lookup fails, which routes publishing
// through the mock rather than a token error.
func (c *Integrations) MetaConfigured(ctx context.Context) bool {
	if v, ok := c.configured.get("META"); ok {
		return v
	}
	res, err := c.settings.GetProviderCredentials(ctx, &integrationsv1.GetProviderCredentialsRequest{Key: integrationsv1.IntegrationKey_META})
	if err != nil {
		c.logger.Warn("provider credentials lookup failed", "err", err)
		return false
	}
	c.configured.put("META", res.Configured)
	return res.Configured
}

func (c *Integrations) AccessToken(ctx context.Context, connectionID string) (string, error) {
	if tok, ok := c.tokens.get(connectionID); ok {
		return tok, nil
	}
	res, err := c.connections.GetAccessToken(ctx, &integrationsv1.GetAccessTokenRequest{ConnectionId: connectionID})
	if err != nil {
		return "", grpcx.FromStatus(err)
	}
	c.tokens.put(connectionID, res.Token)
	return res.Token, nil
}

// Bust drops cached credentials after an integration.updated event.
func (c *Integrations) Bust() {
	c.configured.clear()
	c.tokens.clear()
}
