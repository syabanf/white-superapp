package clients

import (
	"context"
	"time"

	analyticsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/analytics/v1"
	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
)

type Analytics struct {
	sources  analyticsv1.SourceServiceClient
	accounts *ttlCache[[]app.SocialAccount]
}

func NewAnalytics(conn *grpc.ClientConn) *Analytics {
	return &Analytics{sources: analyticsv1.NewSourceServiceClient(conn), accounts: newCache[[]app.SocialAccount](15 * time.Second)}
}

// PlatformFrom maps the shared enum; "" for unspecified.
func PlatformFrom(p commonv1.SocialPlatform) domain.Platform {
	switch p {
	case commonv1.SocialPlatform_INSTAGRAM:
		return domain.PlatformInstagram
	case commonv1.SocialPlatform_FACEBOOK:
		return domain.PlatformFacebook
	case commonv1.SocialPlatform_TIKTOK:
		return domain.PlatformTikTok
	}
	return ""
}

// OwnAccounts lists the client's own social accounts (competitors excluded).
func (c *Analytics) OwnAccounts(ctx context.Context, clientID string) ([]app.SocialAccount, error) {
	if v, ok := c.accounts.get(clientID); ok {
		return v, nil
	}
	res, err := c.sources.ListSocialAccounts(ctx, &analyticsv1.ListSocialAccountsRequest{ClientId: clientID})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := make([]app.SocialAccount, 0, len(res.Accounts))
	for _, a := range res.Accounts {
		if !a.IsCompetitor {
			out = append(out, app.SocialAccount{ID: a.Id, Platform: PlatformFrom(a.Platform), Username: a.Username})
		}
	}
	c.accounts.put(clientID, out)
	return out, nil
}

// Posts reads synced own-account posts (internal RPC).
func (c *Analytics) Posts(ctx context.Context, clientID string, externalIDs []string, fromDay, toDay string) ([]app.SocialPost, error) {
	res, err := c.sources.ListSocialPosts(ctx, &analyticsv1.ListSocialPostsRequest{ClientId: clientID, ExternalIds: externalIDs, FromDay: fromDay, ToDay: toDay, Limit: 1000})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := make([]app.SocialPost, 0, len(res.Posts))
	for _, p := range res.Posts {
		out = append(out, app.SocialPost{
			ExternalID: p.ExternalId, Caption: p.Caption, MediaType: p.MediaType, PublishedAt: p.PublishedAt.AsTime(),
			Metrics: domain.PostMetrics{Likes: p.Likes, Comments: p.Comments, Shares: p.Shares, Saves: p.Saves, Reach: p.Reach, Views: p.Views},
		})
	}
	return out, nil
}
