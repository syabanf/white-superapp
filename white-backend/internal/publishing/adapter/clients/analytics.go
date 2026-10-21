package clients

import (
	"context"
	"time"

	analyticsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/analytics/v1"
	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

type Analytics struct {
	sources  analyticsv1.SourceServiceClient
	accounts *ttlCache[[]domain.SocialAccount]
}

func NewAnalytics(conn *grpc.ClientConn) *Analytics {
	return &Analytics{sources: analyticsv1.NewSourceServiceClient(conn), accounts: newCache[[]domain.SocialAccount](15 * time.Second)}
}

// PlatformFrom maps the shared enum to the publishing platform.
func PlatformFrom(p commonv1.SocialPlatform) domain.Platform {
	switch p {
	case commonv1.SocialPlatform_FACEBOOK:
		return domain.PlatformFacebook
	case commonv1.SocialPlatform_TIKTOK:
		return domain.PlatformTikTok
	}
	return domain.PlatformInstagram
}

// OwnAccounts lists the client's own social accounts (competitors excluded).
func (c *Analytics) OwnAccounts(ctx context.Context, clientID string) ([]domain.SocialAccount, error) {
	if v, ok := c.accounts.get(clientID); ok {
		return v, nil
	}
	res, err := c.sources.ListSocialAccounts(ctx, &analyticsv1.ListSocialAccountsRequest{ClientId: clientID, IncludeCompetitors: false})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := make([]domain.SocialAccount, 0, len(res.Accounts))
	for _, a := range res.Accounts {
		if a.IsCompetitor {
			continue
		}
		out = append(out, domain.SocialAccount{ID: a.Id, Platform: PlatformFrom(a.Platform), ExternalID: a.ExternalId, Username: a.Username, DisplayName: a.DisplayName, AvatarURL: a.AvatarUrl, ConnectionID: a.ConnectionId})
	}
	c.accounts.put(clientID, out)
	return out, nil
}

// PostingHeatmap has no source yet: analytics exposes no synced-post RPC,
// so best times fall back to the defaults.
func (c *Analytics) PostingHeatmap(context.Context, string, string) ([]domain.HeatCell, error) {
	return nil, nil
}
