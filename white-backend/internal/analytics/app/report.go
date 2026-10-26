package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

type FollowersPoint struct {
	Date      string
	Followers int
}

type TopQuery struct {
	Query string
	domain.SearchAgg
}

type ReportSocial struct {
	Overview       OverviewSocial
	DailyFollowers []FollowersPoint
	TopPosts       []domain.PostWithAccount
}

type ReportSeo struct {
	Overview   OverviewSeo
	TopQueries []TopQuery
}

type ReportAds struct {
	Overview  OverviewAds
	Campaigns []domain.CampaignRow
}

type ReportData struct {
	Social *ReportSocial
	Seo    *ReportSeo
	Ads    *ReportAds
}

// Reports serves the internal report/share data (no user check: the
// gateway never exposes it).
type Reports struct {
	Deps
	overview *Overviews
}

func NewReports(d Deps) *Reports { return &Reports{Deps: d, overview: NewOverviews(d)} }

// Data computes the blocks for the requested modules (empty = all).
func (r *Reports) Data(ctx context.Context, clientID string, w Window, modules []string, topN int) (*ReportData, error) {
	if topN <= 0 {
		topN = 5
	}
	want := func(m string) bool { return len(modules) == 0 || contains(modules, m) }
	out := &ReportData{}
	if want("SOCIAL") {
		ov, err := r.overview.Social(ctx, clientID, w)
		if err != nil {
			return nil, err
		}
		daily, err := r.DailyFollowers(ctx, clientID, dates.Range{From: dates.AddDays(w.Current.From, -1), To: w.Current.To})
		if err != nil {
			return nil, err
		}
		top, err := r.TopPosts(ctx, clientID, w.Current, topN)
		if err != nil {
			return nil, err
		}
		out.Social = &ReportSocial{Overview: *ov, DailyFollowers: daily, TopPosts: top}
	}
	if want("SEO") {
		ov, err := r.overview.Seo(ctx, clientID, w)
		if err != nil {
			return nil, err
		}
		top, err := r.TopQueries(ctx, clientID, w.Current, topN)
		if err != nil {
			return nil, err
		}
		out.Seo = &ReportSeo{Overview: *ov, TopQueries: top}
	}
	if want("ADS") {
		ov, err := r.overview.Ads(ctx, clientID, w)
		if err != nil {
			return nil, err
		}
		camps, err := r.CampaignRows(ctx, clientID, w.Current)
		if err != nil {
			return nil, err
		}
		out.Ads = &ReportAds{Overview: *ov, Campaigns: camps}
	}
	return out, nil
}

// DailyFollowers is getSocialDailyFollowers: own accounts carried forward
// per account from 30 days before the range, one total per day of the range.
func (r *Reports) DailyFollowers(ctx context.Context, clientID string, rg dates.Range) ([]FollowersPoint, error) {
	accounts, err := r.Store.ListSocialAccounts(ctx, clientID, false, nil)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return []FollowersPoint{}, nil
	}
	idList := make([]string, 0, len(accounts))
	for _, a := range accounts {
		idList = append(idList, a.ID)
	}
	start := dates.AddDays(rg.From, -30)
	snaps, err := r.Store.ListSnapshots(ctx, idList, start, rg.To)
	if err != nil {
		return nil, err
	}
	byDay := map[string][]domain.SocialSnapshot{}
	for _, s := range snaps {
		k := dates.Format(s.Date)
		byDay[k] = append(byDay[k], s)
	}
	perAcc := map[string]int{}
	out := []FollowersPoint{}
	dates.Range{From: start, To: rg.To}.Each(func(d time.Time) {
		k := dates.Format(d)
		for _, s := range byDay[k] {
			perAcc[s.AccountID] = s.Followers
		}
		if d.Before(rg.From) {
			return
		}
		sum := 0
		for _, v := range perAcc {
			sum += v
		}
		out = append(out, FollowersPoint{Date: k, Followers: sum})
	})
	return out, nil
}

func (r *Reports) TopPosts(ctx context.Context, clientID string, rg dates.Range, n int) ([]domain.PostWithAccount, error) {
	posts, err := r.Store.ListOwnPosts(ctx, clientID, rg.From, endExclusive(rg.To))
	if err != nil {
		return nil, err
	}
	sortBy(posts, func(a, b domain.PostWithAccount) bool {
		return domain.PostEngagements(a.SocialPost) > domain.PostEngagements(b.SocialPost)
	})
	if len(posts) > n {
		posts = posts[:n]
	}
	return posts, nil
}

func (r *Reports) TopQueries(ctx context.Context, clientID string, rg dates.Range, n int) ([]TopQuery, error) {
	props, err := r.Store.ListSeoProperties(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if len(props) == 0 {
		return []TopQuery{}, nil
	}
	idList := make([]string, 0, len(props))
	for _, p := range props {
		idList = append(idList, p.ID)
	}
	rows, err := r.Store.ListSeoDimension(ctx, idList, domain.DimQuery, rg.From, rg.To)
	if err != nil {
		return nil, err
	}
	byKey := map[string][]domain.SearchRow{}
	var order []string
	for _, x := range rows {
		if _, ok := byKey[x.Key]; !ok {
			order = append(order, x.Key)
		}
		byKey[x.Key] = append(byKey[x.Key], domain.SearchRow{Clicks: x.Clicks, Impressions: x.Impressions, Position: x.Position})
	}
	out := make([]TopQuery, 0, len(order))
	for _, k := range order {
		out = append(out, TopQuery{Query: k, SearchAgg: domain.AggregateSearch(byKey[k])})
	}
	sortBy(out, func(a, b TopQuery) bool { return a.Clicks > b.Clicks })
	if len(out) > n {
		out = out[:n]
	}
	return out, nil
}

func (r *Reports) CampaignRows(ctx context.Context, clientID string, rg dates.Range) ([]domain.CampaignRow, error) {
	accounts, err := r.Store.ListAdAccounts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return []domain.CampaignRow{}, nil
	}
	idList := make([]string, 0, len(accounts))
	for _, a := range accounts {
		idList = append(idList, a.ID)
	}
	rows, err := r.Store.ListInsights(ctx, idList, rg.From, rg.To, nil)
	if err != nil {
		return nil, err
	}
	camps, err := r.Store.ListCampaigns(ctx, idList)
	if err != nil {
		return nil, err
	}
	return domain.CampaignRows(rows, camps), nil
}
