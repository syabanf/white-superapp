package metagraph

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/meta"
)

const (
	providerName = "meta"
	mediaFields  = "id,caption,like_count,comments_count,media_type,media_product_type,permalink,media_url,thumbnail_url,timestamp"
)

type Real struct {
	Client *meta.Client
}

func NewReal() *Real {
	return &Real{Client: meta.NewClient(providerName, 30*time.Second)}
}

type rawIgAccount struct {
	ID                string  `json:"id"`
	Username          *string `json:"username"`
	Name              *string `json:"name"`
	Biography         *string `json:"biography"`
	ProfilePictureURL *string `json:"profile_picture_url"`
	FollowersCount    *int    `json:"followers_count"`
	FollowsCount      *int    `json:"follows_count"`
	MediaCount        *int    `json:"media_count"`
}

type rawMedia struct {
	ID               string  `json:"id"`
	Caption          *string `json:"caption"`
	LikeCount        *int    `json:"like_count"`
	CommentsCount    *int    `json:"comments_count"`
	MediaType        *string `json:"media_type"`
	MediaProductType *string `json:"media_product_type"`
	Permalink        *string `json:"permalink"`
	MediaURL         *string `json:"media_url"`
	ThumbnailURL     *string `json:"thumbnail_url"`
	Timestamp        *string `json:"timestamp"`
}

type rawInsights struct {
	Data []struct {
		Name   string `json:"name"`
		Values []struct {
			Value   any    `json:"value"`
			EndTime string `json:"end_time"`
		} `json:"values"`
	} `json:"data"`
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}

func parseTime(s *string) time.Time {
	if s == nil {
		return time.Unix(0, 0).UTC()
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-0700"} {
		if t, err := time.Parse(layout, *s); err == nil {
			return t.UTC()
		}
	}
	return time.Unix(0, 0).UTC()
}

func toMediaType(v *string) string {
	if v != nil && (*v == "VIDEO" || *v == "CAROUSEL_ALBUM") {
		return *v
	}
	return "IMAGE"
}

func toProductType(v *string) *string {
	if v != nil && (*v == "REELS" || *v == "STORY" || *v == "FEED") {
		return v
	}
	return nil
}

func mapMedia(m rawMedia) IgMedia {
	thumb := m.ThumbnailURL
	if thumb == nil {
		thumb = m.MediaURL
	}
	return IgMedia{ID: m.ID, Caption: deref(m.Caption), MediaType: toMediaType(m.MediaType), MediaProductType: toProductType(m.MediaProductType), Permalink: m.Permalink, MediaURL: m.MediaURL, ThumbnailURL: thumb, Timestamp: parseTime(m.Timestamp), LikeCount: derefInt(m.LikeCount), CommentsCount: derefInt(m.CommentsCount)}
}

// parseInsights maps metric -> date -> value; end_time marks the end of the
// 24 h window and non-number values become 0.
func parseInsights(res rawInsights) (map[string]map[string]int, []string) {
	out := map[string]map[string]int{}
	seen := map[string]bool{}
	for _, series := range res.Data {
		byDate := map[string]int{}
		for _, v := range series.Values {
			if len(v.EndTime) < 10 {
				continue
			}
			date := v.EndTime[:10]
			n := 0
			if f, ok := v.Value.(float64); ok {
				n = int(f)
			}
			byDate[date] = n
			seen[date] = true
		}
		out[series.Name] = byDate
	}
	days := make([]string, 0, len(seen))
	for d := range seen {
		days = append(days, d)
	}
	sort.Strings(days)
	return out, days
}

func (r *Real) GetInstagramAccount(ctx context.Context, token, igUserID string) (*IgAccount, error) {
	var raw rawIgAccount
	if err := r.Client.Get(ctx, token, "/"+igUserID, url.Values{"fields": {"id,username,name,biography,profile_picture_url,followers_count,follows_count,media_count"}}, &raw); err != nil {
		return nil, err
	}
	name := deref(raw.Name)
	if name == "" {
		name = deref(raw.Username)
	}
	return &IgAccount{ID: raw.ID, Username: deref(raw.Username), Name: name, Biography: raw.Biography, ProfilePictureURL: raw.ProfilePictureURL, FollowersCount: derefInt(raw.FollowersCount), FollowsCount: derefInt(raw.FollowsCount), MediaCount: derefInt(raw.MediaCount)}, nil
}

func (r *Real) GetInstagramInsights(ctx context.Context, token, igUserID string, since, until time.Time) ([]IgDailyInsight, error) {
	var raw rawInsights
	params := url.Values{"metric": {"reach,impressions,profile_views,follower_count"}, "period": {"day"}, "since": {dates.Format(since)}, "until": {dates.Format(until)}}
	if err := r.Client.Get(ctx, token, "/"+igUserID+"/insights", params, &raw); err != nil {
		return nil, err
	}
	m, days := parseInsights(raw)
	out := make([]IgDailyInsight, 0, len(days))
	for _, d := range days {
		out = append(out, IgDailyInsight{Date: d, Reach: m["reach"][d], Impressions: m["impressions"][d], ProfileViews: m["profile_views"][d], FollowerCount: m["follower_count"][d]})
	}
	return out, nil
}

func (r *Real) GetInstagramMedia(ctx context.Context, token, igUserID string, limit int) ([]IgMedia, error) {
	if limit <= 0 {
		limit = 50
	}
	var raw struct {
		Data []rawMedia `json:"data"`
	}
	if err := r.Client.Get(ctx, token, "/"+igUserID+"/media", url.Values{"fields": {mediaFields}, "limit": {strconv.Itoa(limit)}}, &raw); err != nil {
		return nil, err
	}
	out := make([]IgMedia, 0, len(raw.Data))
	for _, m := range raw.Data {
		out = append(out, mapMedia(m))
	}
	return out, nil
}

func (r *Real) DiscoverBusiness(ctx context.Context, token, igUserID, username string) (*IgBusinessDiscovery, error) {
	clean := strings.TrimLeft(strings.TrimSpace(username), "@")
	var raw struct {
		BusinessDiscovery *struct {
			rawIgAccount
			Media *struct {
				Data []rawMedia `json:"data"`
			} `json:"media"`
		} `json:"business_discovery"`
	}
	fields := fmt.Sprintf("business_discovery.username(%s){followers_count,media_count,username,name,biography,profile_picture_url,media.limit(25){%s}}", clean, mediaFields)
	if err := r.Client.Get(ctx, token, "/"+igUserID, url.Values{"fields": {fields}}, &raw); err != nil {
		return nil, err
	}
	bd := raw.BusinessDiscovery
	if bd == nil {
		return nil, apperr.FromProvider(providerName, apperr.NotFound, fmt.Sprintf("Akun @%s tidak ditemukan lewat Business Discovery", clean), nil)
	}
	u := deref(bd.Username)
	if u == "" {
		u = clean
	}
	name := deref(bd.Name)
	if name == "" {
		name = u
	}
	out := &IgBusinessDiscovery{Username: u, Name: name, Biography: bd.Biography, ProfilePictureURL: bd.ProfilePictureURL, FollowersCount: derefInt(bd.FollowersCount), MediaCount: derefInt(bd.MediaCount)}
	if bd.Media != nil {
		for _, m := range bd.Media.Data {
			out.Media = append(out.Media, mapMedia(m))
		}
	}
	return out, nil
}

func (r *Real) GetFacebookPage(ctx context.Context, token, pageID string) (*FbPage, error) {
	var raw struct {
		ID             string  `json:"id"`
		Name           *string `json:"name"`
		Username       *string `json:"username"`
		FanCount       *int    `json:"fan_count"`
		FollowersCount *int    `json:"followers_count"`
	}
	if err := r.Client.Get(ctx, token, "/"+pageID, url.Values{"fields": {"id,name,username,fan_count,followers_count"}}, &raw); err != nil {
		return nil, err
	}
	followers := raw.FollowersCount
	if followers == nil {
		followers = raw.FanCount
	}
	return &FbPage{ID: raw.ID, Name: deref(raw.Name), Username: raw.Username, FanCount: derefInt(raw.FanCount), FollowersCount: derefInt(followers)}, nil
}

func (r *Real) GetPageInsights(ctx context.Context, token, pageID string, since, until time.Time) ([]FbPageDailyInsight, error) {
	var raw rawInsights
	params := url.Values{"metric": {"page_impressions_unique,page_post_engagements"}, "period": {"day"}, "since": {dates.Format(since)}, "until": {dates.Format(until)}}
	if err := r.Client.Get(ctx, token, "/"+pageID+"/insights", params, &raw); err != nil {
		return nil, err
	}
	m, days := parseInsights(raw)
	out := make([]FbPageDailyInsight, 0, len(days))
	for _, d := range days {
		out = append(out, FbPageDailyInsight{Date: d, ImpressionsUnique: m["page_impressions_unique"][d], PostEngagements: m["page_post_engagements"][d]})
	}
	return out, nil
}

func (r *Real) GetPagePosts(ctx context.Context, token, pageID string, limit int) ([]FbPost, error) {
	if limit <= 0 {
		limit = 25
	}
	var raw struct {
		Data []struct {
			ID           string  `json:"id"`
			Message      *string `json:"message"`
			CreatedTime  *string `json:"created_time"`
			PermalinkURL *string `json:"permalink_url"`
			Shares       *struct {
				Count int `json:"count"`
			} `json:"shares"`
			Reactions *struct {
				Summary struct {
					TotalCount int `json:"total_count"`
				} `json:"summary"`
			} `json:"reactions"`
			Comments *struct {
				Summary struct {
					TotalCount int `json:"total_count"`
				} `json:"summary"`
			} `json:"comments"`
		} `json:"data"`
	}
	params := url.Values{"fields": {"id,message,created_time,permalink_url,shares,reactions.summary(true),comments.summary(true)"}, "limit": {strconv.Itoa(limit)}}
	if err := r.Client.Get(ctx, token, "/"+pageID+"/posts", params, &raw); err != nil {
		return nil, err
	}
	out := make([]FbPost, 0, len(raw.Data))
	for _, p := range raw.Data {
		post := FbPost{ID: p.ID, Message: deref(p.Message), CreatedTime: parseTime(p.CreatedTime), PermalinkURL: p.PermalinkURL}
		if p.Shares != nil {
			post.Shares = p.Shares.Count
		}
		if p.Reactions != nil {
			post.Reactions = p.Reactions.Summary.TotalCount
		}
		if p.Comments != nil {
			post.Comments = p.Comments.Summary.TotalCount
		}
		out = append(out, post)
	}
	return out, nil
}
