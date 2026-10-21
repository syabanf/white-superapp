package clients

import (
	"context"

	analyticsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/analytics/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/app"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

// Analytics lists a client's SEO properties.
type Analytics struct {
	client analyticsv1.SourceServiceClient
}

func NewAnalytics(conn grpc.ClientConnInterface) *Analytics {
	return &Analytics{client: analyticsv1.NewSourceServiceClient(conn)}
}

func (c *Analytics) ListProperties(ctx context.Context, clientID string) ([]app.Property, error) {
	res, err := c.client.ListSeoProperties(ctx, &analyticsv1.ListSeoPropertiesRequest{ClientId: clientID})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := make([]app.Property, 0, len(res.GetProperties()))
	for _, p := range res.GetProperties() {
		out = append(out, app.Property{
			ID: p.GetId(), ClientID: p.GetClientId(), SiteURL: p.GetSiteUrl(),
			LocationCode: p.GetLocationCode(), LanguageCode: p.GetLanguageCode(),
			CreatedAt: p.GetCreatedAt().AsTime(),
		})
	}
	return out, nil
}
