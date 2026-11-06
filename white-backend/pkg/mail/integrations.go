package mail

import (
	"context"
	"sync"
	"time"

	integrationsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/integrations/v1"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

const credentialsTTL = time.Minute

// FromIntegrations reads the RESEND credentials the admin saved (env
// RESEND_API_KEY and MAIL_FROM when none), cached for a minute.
func FromIntegrations(conn grpc.ClientConnInterface) Credentials {
	client := integrationsv1.NewSettingsServiceClient(conn)
	var (
		mu        sync.Mutex
		key, from string
		fetchedAt time.Time
	)
	return func(ctx context.Context) (string, string, error) {
		mu.Lock()
		defer mu.Unlock()
		if !fetchedAt.IsZero() && time.Since(fetchedAt) < credentialsTTL {
			return key, from, nil
		}
		res, err := client.GetProviderCredentials(ctx, &integrationsv1.GetProviderCredentialsRequest{Key: integrationsv1.IntegrationKey_RESEND})
		if err != nil {
			return "", "", grpcx.FromStatus(err)
		}
		key, from, fetchedAt = res.GetSecret(), res.GetPublicId(), time.Now()
		return key, from, nil
	}
}
