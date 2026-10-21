// Package metagraph reads organic Instagram and Facebook data from the Meta
// Graph API. The mock is deterministic and shared with the competitor flow.
package metagraph

import (
	"context"
	"time"
)

type IgAccount struct {
	ID                string
	Username          string
	Name              string
	Biography         *string
	ProfilePictureURL *string
	FollowersCount    int
	FollowsCount      int
	MediaCount        int
}

type IgDailyInsight struct {
	Date          string
	Reach         int
	Impressions   int
	ProfileViews  int
	FollowerCount int
}

type IgMedia struct {
	ID               string
	Caption          string
	MediaType        string
	MediaProductType *string
	Permalink        *string
	MediaURL         *string
	ThumbnailURL     *string
	Timestamp        time.Time
	LikeCount        int
	CommentsCount    int
	// Per-media insights: only the mock fills them for own media.
	Saved       *int
	Shares      *int
	Reach       *int
	Impressions *int
	Views       *int
}

type IgBusinessDiscovery struct {
	Username          string
	Name              string
	Biography         *string
	ProfilePictureURL *string
	FollowersCount    int
	MediaCount        int
	Media             []IgMedia
}

type FbPage struct {
	ID             string
	Name           string
	Username       *string
	FanCount       int
	FollowersCount int
}

type FbPageDailyInsight struct {
	Date             string
	ImpressionsUnique int
	PostEngagements  int
}

type FbPost struct {
	ID           string
	Message      string
	CreatedTime  time.Time
	PermalinkURL *string
	Shares       int
	Reactions    int
	Comments     int
}

type Provider interface {
	GetInstagramAccount(ctx context.Context, token, igUserID string) (*IgAccount, error)
	GetInstagramInsights(ctx context.Context, token, igUserID string, since, until time.Time) ([]IgDailyInsight, error)
	GetInstagramMedia(ctx context.Context, token, igUserID string, limit int) ([]IgMedia, error)
	// DiscoverBusiness reads a competitor's public profile through an own IG account.
	DiscoverBusiness(ctx context.Context, token, igUserID, username string) (*IgBusinessDiscovery, error)
	GetFacebookPage(ctx context.Context, token, pageID string) (*FbPage, error)
	GetPageInsights(ctx context.Context, token, pageID string, since, until time.Time) ([]FbPageDailyInsight, error)
	GetPagePosts(ctx context.Context, token, pageID string, limit int) ([]FbPost, error)
}

// Pick returns the real adapter when the Meta app is configured.
func Pick(configured bool) Provider {
	if configured {
		return NewReal()
	}
	return Mock{Now: time.Now}
}
