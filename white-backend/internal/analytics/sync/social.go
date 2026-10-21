package sync

import (
	"context"
	"math"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/apify"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/metagraph"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

type Failure struct {
	Username string
	Error    string
}

type SocialResult struct {
	Updated int
	Posts   int
	// Skipped counts own accounts without a usable Meta token.
	Skipped int
	Failed  []Failure
}

func postFromPublic(accountID string, p apify.Post) domain.SocialPost {
	return domain.SocialPost{AccountID: accountID, ExternalID: p.ExternalID, Caption: p.Caption, MediaType: p.MediaType, ProductType: p.ProductType, Permalink: p.Permalink, MediaURL: p.MediaURL, ThumbnailURL: p.ThumbnailURL, PublishedAt: p.PublishedAt, Likes: p.Likes, Comments: p.Comments, Shares: p.Shares, Saves: p.Saves, Views: p.Views}
}

// SyncSocialPublic refreshes every account of the platform (own and, unless
// includeCompetitors is false, competitors) from public data: today's
// snapshot (followers, following, media count), the profile and the latest
// posts. postLimit 0 means 20 for competitors and 30 for own accounts.
func (s *Syncer) SyncSocialPublic(ctx context.Context, clientID string, platform domain.Platform, includeCompetitors bool, postLimit int) (SocialResult, error) {
	accounts, err := s.Store.ListSocialAccounts(ctx, clientID, includeCompetitors, &platform)
	if err != nil {
		return SocialResult{}, err
	}
	provider := s.PublicSocial(ctx)
	today := dates.Today(s.now())
	res := SocialResult{Failed: []Failure{}}
	for _, acc := range accounts {
		limit := postLimit
		if limit == 0 {
			limit = 30
			if acc.IsCompetitor {
				limit = 20
			}
		}
		profile, err := provider.Profile(ctx, string(platform), acc.Username)
		if err != nil {
			res.Failed = append(res.Failed, Failure{acc.Username, errText(err)})
			continue
		}
		posts, err := provider.Posts(ctx, string(platform), acc.Username, limit)
		if err != nil {
			res.Failed = append(res.Failed, Failure{acc.Username, errText(err)})
			continue
		}
		err = s.Store.WithTx(ctx, func(tx domain.Store) error {
			if err := tx.UpdateSocialAccountProfile(ctx, acc.ID, domain.ProfilePatch{DisplayName: profile.DisplayName, SetBiography: true, Biography: profile.Biography, AvatarURL: profile.AvatarURL}); err != nil {
				return err
			}
			snap := domain.SocialSnapshot{AccountID: acc.ID, Date: today, Followers: profile.Followers, Following: profile.Following, MediaCount: profile.MediaCount}
			if err := tx.UpsertSnapshot(ctx, snap, domain.SnapshotPatch{Followers: intp(profile.Followers), Following: profile.Following, MediaCount: intp(profile.MediaCount)}); err != nil {
				return err
			}
			for _, p := range posts {
				row := postFromPublic(acc.ID, p)
				patch := domain.PostPatch{Caption: strp(p.Caption), Likes: intp(p.Likes), Comments: intp(p.Comments), Shares: intp(p.Shares), Saves: intp(p.Saves), Views: intp(p.Views), ThumbnailURL: p.ThumbnailURL}
				if err := tx.UpsertPost(ctx, row, patch); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			res.Failed = append(res.Failed, Failure{acc.Username, errText(err)})
			continue
		}
		res.Updated++
		res.Posts += len(posts)
	}
	return res, nil
}

// SyncSocialMeta pulls the client's own Instagram or Facebook accounts that
// carry a Meta connection: profile, daily insights for the last `days` and
// media with per-post numbers. Accounts without a token are skipped.
func (s *Syncer) SyncSocialMeta(ctx context.Context, clientID string, platform domain.Platform, days int) (SocialResult, error) {
	accounts, err := s.Store.ListSocialAccounts(ctx, clientID, false, &platform)
	if err != nil {
		return SocialResult{}, err
	}
	if days <= 0 {
		days = 30
	}
	provider := s.graph(ctx)
	until := dates.LatestCompleteDay(s.now())
	since := dates.AddDays(until, -(days - 1))
	today := dates.Today(s.now())
	res := SocialResult{Failed: []Failure{}}
	for _, acc := range accounts {
		token, err := s.token(ctx, acc.ConnectionID)
		if err != nil {
			res.Failed = append(res.Failed, Failure{acc.Username, errText(err)})
			continue
		}
		if token == "" {
			res.Skipped++
			continue
		}
		var posts int
		if platform == domain.Instagram {
			posts, err = s.syncInstagram(ctx, provider, token, acc, since, until, today)
		} else {
			posts, err = s.syncFacebook(ctx, provider, token, acc, since, until, today)
		}
		if err != nil {
			res.Failed = append(res.Failed, Failure{acc.Username, errText(err)})
			continue
		}
		res.Posts += posts
		res.Updated++
	}
	return res, nil
}

func (s *Syncer) syncInstagram(ctx context.Context, p metagraph.Provider, token string, acc domain.SocialAccount, since, until, today time.Time) (int, error) {
	account, err := p.GetInstagramAccount(ctx, token, acc.ExternalID)
	if err != nil {
		return 0, err
	}
	insights, err := p.GetInstagramInsights(ctx, token, acc.ExternalID, since, until)
	if err != nil {
		return 0, err
	}
	media, err := p.GetInstagramMedia(ctx, token, acc.ExternalID, 50)
	if err != nil {
		return 0, err
	}
	if err := s.Store.UpdateSocialAccountProfile(ctx, acc.ID, domain.ProfilePatch{Username: strp(account.Username), DisplayName: account.Name, SetBiography: true, Biography: account.Biography, AvatarURL: account.ProfilePictureURL}); err != nil {
		return 0, err
	}
	for _, d := range insights {
		date, err := dates.Parse(d.Date)
		if err != nil {
			continue
		}
		create := domain.SocialSnapshot{AccountID: acc.ID, Date: date, Followers: account.FollowersCount, Following: intp(account.FollowsCount), MediaCount: account.MediaCount, Reach: intp(d.Reach), Impressions: intp(d.Impressions), ProfileViews: intp(d.ProfileViews)}
		if err := s.Store.UpsertSnapshot(ctx, create, domain.SnapshotPatch{Reach: intp(d.Reach), Impressions: intp(d.Impressions), ProfileViews: intp(d.ProfileViews)}); err != nil {
			return 0, err
		}
	}
	todayRow := domain.SocialSnapshot{AccountID: acc.ID, Date: today, Followers: account.FollowersCount, Following: intp(account.FollowsCount), MediaCount: account.MediaCount}
	if err := s.Store.UpsertSnapshot(ctx, todayRow, domain.SnapshotPatch{Followers: intp(account.FollowersCount), Following: intp(account.FollowsCount), MediaCount: intp(account.MediaCount)}); err != nil {
		return 0, err
	}
	for _, m := range media {
		shares, saves, views, reach, impressions := zero(m.Shares), zero(m.Saved), zero(m.Views), zero(m.Reach), zero(m.Impressions)
		row := domain.SocialPost{AccountID: acc.ID, ExternalID: m.ID, Caption: m.Caption, MediaType: m.MediaType, ProductType: m.MediaProductType, Permalink: m.Permalink, MediaURL: m.MediaURL, ThumbnailURL: m.ThumbnailURL, PublishedAt: m.Timestamp, Likes: m.LikeCount, Comments: m.CommentsCount, Shares: shares, Saves: saves, Views: views, Reach: reach, Impressions: impressions}
		patch := domain.PostPatch{Caption: strp(m.Caption), MediaType: strp(m.MediaType), ProductType: m.MediaProductType, Permalink: m.Permalink, MediaURL: m.MediaURL, ThumbnailURL: m.ThumbnailURL, Likes: intp(m.LikeCount), Comments: intp(m.CommentsCount), Shares: intp(shares), Saves: intp(saves), Views: intp(views), Reach: intp(reach), Impressions: intp(impressions)}
		if err := s.Store.UpsertPost(ctx, row, patch); err != nil {
			return 0, err
		}
	}
	return len(media), nil
}

func (s *Syncer) syncFacebook(ctx context.Context, p metagraph.Provider, token string, acc domain.SocialAccount, since, until, today time.Time) (int, error) {
	page, err := p.GetFacebookPage(ctx, token, acc.ExternalID)
	if err != nil {
		return 0, err
	}
	insights, err := p.GetPageInsights(ctx, token, acc.ExternalID, since, until)
	if err != nil {
		return 0, err
	}
	posts, err := p.GetPagePosts(ctx, token, acc.ExternalID, 50)
	if err != nil {
		return 0, err
	}
	if err := s.Store.UpdateSocialAccountProfile(ctx, acc.ID, domain.ProfilePatch{Username: page.Username, DisplayName: page.Name}); err != nil {
		return 0, err
	}
	for _, d := range insights {
		date, err := dates.Parse(d.Date)
		if err != nil {
			continue
		}
		create := domain.SocialSnapshot{AccountID: acc.ID, Date: date, Followers: page.FollowersCount, Reach: intp(d.ImpressionsUnique)}
		if err := s.Store.UpsertSnapshot(ctx, create, domain.SnapshotPatch{Reach: intp(d.ImpressionsUnique)}); err != nil {
			return 0, err
		}
	}
	if err := s.Store.UpsertSnapshot(ctx, domain.SocialSnapshot{AccountID: acc.ID, Date: today, Followers: page.FollowersCount}, domain.SnapshotPatch{Followers: intp(page.FollowersCount)}); err != nil {
		return 0, err
	}
	for _, post := range posts {
		row := domain.SocialPost{AccountID: acc.ID, ExternalID: post.ID, Caption: post.Message, MediaType: "IMAGE", Permalink: post.PermalinkURL, PublishedAt: post.CreatedTime, Likes: post.Reactions, Comments: post.Comments, Shares: post.Shares}
		patch := domain.PostPatch{Caption: strp(post.Message), Permalink: post.PermalinkURL, Likes: intp(post.Reactions), Comments: intp(post.Comments), Shares: intp(post.Shares)}
		if err := s.Store.UpsertPost(ctx, row, patch); err != nil {
			return 0, err
		}
	}
	return len(posts), nil
}

func zero(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}

// SyncSocialDemo extrapolates today's snapshot of each account from its
// last 8 snapshots (mean gain plus deterministic jitter). Accounts without
// any snapshot are left alone. Returns how many accounts were updated.
func (s *Syncer) SyncSocialDemo(ctx context.Context, accounts []domain.SocialAccount) (int, error) {
	today := dates.Today(s.now())
	updated := 0
	for _, acc := range accounts {
		recent, err := s.Store.ListRecentSnapshots(ctx, acc.ID, 8)
		if err != nil {
			return updated, err
		}
		if len(recent) == 0 {
			continue
		}
		last := recent[0]
		var avgGain float64
		if len(recent) > 1 {
			for i := 0; i < len(recent)-1; i++ {
				avgGain += float64(recent[i].Followers - recent[i+1].Followers)
			}
			avgGain /= float64(len(recent) - 1)
		} else {
			avgGain = math.Max(1, float64(last.Followers)*0.001)
		}
		rand := mockrand.New("sync:" + acc.ID + ":" + dates.Format(today))
		followers := int(math.Max(0, math.Floor(float64(last.Followers)+avgGain*(0.7+rand.Float64()*0.6)+0.5)))
		scale := func(v *int) *int {
			if v == nil {
				return nil
			}
			n := int(math.Max(0, math.Floor(float64(*v)*(0.85+rand.Float64()*0.3)+0.5)))
			return &n
		}
		mediaCount := last.MediaCount
		if rand.Float64() < 0.3 {
			mediaCount++
		}
		row := domain.SocialSnapshot{AccountID: acc.ID, Date: today, Followers: followers, Following: last.Following, MediaCount: mediaCount, Reach: scale(last.Reach), Impressions: scale(last.Impressions), ProfileViews: scale(last.ProfileViews), WebsiteClicks: scale(last.WebsiteClicks)}
		patch := domain.SnapshotPatch{Followers: &row.Followers, Following: row.Following, MediaCount: &row.MediaCount, Reach: row.Reach, Impressions: row.Impressions, ProfileViews: row.ProfileViews, WebsiteClicks: row.WebsiteClicks}
		if err := s.Store.UpsertSnapshot(ctx, row, patch); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}
