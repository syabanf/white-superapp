package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

type OverviewAccount struct {
	Platform  domain.Platform
	Username  string
	Followers int
}

type OverviewSocial struct {
	HasSource, HasData, IsDemo bool
	Quality                    domain.DataQuality
	Followers                  int
	FollowersDelta             metrics.Delta
	FollowersSpark             []float64
	Posts                      int
	PostsDelta                 metrics.Delta
	EngagementRate             float64
	EngagementRateDelta        metrics.Delta
	Reach                      int
	ReachDelta                 metrics.Delta
	Accounts                   []OverviewAccount
}

type SeoPoint struct {
	Date        string
	Clicks      *float64
	Impressions *float64
}

type OverviewSeo struct {
	HasSource, HasData, IsDemo bool
	Quality                    domain.DataQuality
	Clicks                     int
	ClicksDelta                metrics.Delta
	ClicksSpark                []float64
	Impressions                int
	ImpressionsDelta           metrics.Delta
	CTR                        float64
	CTRDelta                   metrics.Delta
	Position                   float64
	PositionDelta              metrics.Delta
	Health                     *int
	Daily                      []SeoPoint
}

type AdsPoint struct {
	Date        string
	Spend       float64
	Results     int
	SpendPrev   *float64
	ResultsPrev *float64
}

type OverviewAds struct {
	HasSource, HasData, IsDemo bool
	Quality                    domain.DataQuality
	Kpis, Prev                 domain.AdKpis
	SpendDelta                 metrics.Delta
	ResultsDelta               metrics.Delta
	CPRDelta                   metrics.Delta
	SpendSpark                 []float64
	Daily                      []AdsPoint
}

type Overview struct {
	Social OverviewSocial
	Seo    OverviewSeo
	Ads    OverviewAds
}

type Overviews struct{ Deps }

func NewOverviews(d Deps) *Overviews { return &Overviews{d} }

var zeroDelta = metrics.ComputeDelta(0, 0)

func (o *Overviews) Get(ctx context.Context, userID, clientID string, w Window) (*Overview, error) {
	if _, err := o.view(ctx, userID, clientID); err != nil {
		return nil, err
	}
	return o.compute(ctx, clientID, w)
}

func (o *Overviews) compute(ctx context.Context, clientID string, w Window) (*Overview, error) {
	social, err := o.Social(ctx, clientID, w)
	if err != nil {
		return nil, err
	}
	seo, err := o.Seo(ctx, clientID, w)
	if err != nil {
		return nil, err
	}
	ads, err := o.Ads(ctx, clientID, w)
	if err != nil {
		return nil, err
	}
	return &Overview{Social: *social, Seo: *seo, Ads: *ads}, nil
}

func inRange(t time.Time, r dates.Range) bool { return !t.Before(r.From) && !t.After(r.To) }

// Social mirrors getOverviewSocial.
func (o *Overviews) Social(ctx context.Context, clientID string, w Window) (*OverviewSocial, error) {
	accounts, err := o.Store.ListSocialAccounts(ctx, clientID, false, nil)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return &OverviewSocial{IsDemo: true, Quality: domain.Quality(false, true, nil, nil, o.now()), FollowersDelta: zeroDelta, PostsDelta: zeroDelta, EngagementRateDelta: zeroDelta, ReachDelta: zeroDelta, FollowersSpark: []float64{}, Accounts: []OverviewAccount{}}, nil
	}
	idList := make([]string, 0, len(accounts))
	conns := make([]*string, 0, len(accounts))
	for _, a := range accounts {
		idList = append(idList, a.ID)
		conns = append(conns, a.ConnectionID)
	}
	cur, prev := w.Current, w.Previous
	snaps, err := o.Store.ListSnapshots(ctx, idList, dates.AddDays(prev.From, -1), cur.To)
	if err != nil {
		return nil, err
	}
	posts, err := o.Store.ListPosts(ctx, idList, cur.From, endExclusive(cur.To))
	if err != nil {
		return nil, err
	}
	prevPosts, err := o.Store.ListPosts(ctx, idList, prev.From, endExclusive(prev.To))
	if err != nil {
		return nil, err
	}
	job, err := o.Store.LatestSyncJob(ctx, clientID, []string{domain.KindSocialSnapshot})
	if err != nil {
		return nil, err
	}

	now := domain.FollowersAtAcrossAccounts(snaps, cur.To)
	start := domain.FollowersAtAcrossAccounts(snaps, dates.AddDays(cur.From, -1))
	prevStart := domain.FollowersAtAcrossAccounts(snaps, dates.AddDays(prev.From, -1))

	// Spark: running total per snapshot date from the range start, last 14.
	perAcc := map[string]int{}
	var order []string
	totals := map[string]float64{}
	for _, s := range snaps {
		perAcc[s.AccountID] = s.Followers
		if s.Date.Before(cur.From) {
			continue
		}
		sum := 0
		for _, v := range perAcc {
			sum += v
		}
		k := dates.Format(s.Date)
		if _, ok := totals[k]; !ok {
			order = append(order, k)
		}
		totals[k] = float64(sum)
	}
	spark := make([]float64, 0, len(order))
	for _, k := range order {
		spark = append(spark, totals[k])
	}
	if len(spark) > 14 {
		spark = spark[len(spark)-14:]
	}

	reach, prevReach := 0, 0
	var points []time.Time
	followersByAcc := map[string]int{}
	for _, s := range snaps {
		followersByAcc[s.AccountID] = s.Followers
		if inRange(s.Date, cur) {
			points = append(points, s.Date)
			if s.Reach != nil {
				reach += *s.Reach
			}
		}
		if inRange(s.Date, prev) && s.Reach != nil {
			prevReach += *s.Reach
		}
	}
	erFor := func(rows []domain.SocialPost) float64 {
		if len(rows) == 0 {
			return 0
		}
		acc := 0.0
		for _, a := range accounts {
			var mine []domain.SocialPost
			for _, p := range rows {
				if p.AccountID == a.ID {
					mine = append(mine, p)
				}
			}
			if len(mine) == 0 {
				continue
			}
			acc += domain.AvgEngagementRate(mine, followersByAcc[a.ID]) * float64(len(mine))
		}
		return acc / float64(len(rows))
	}
	er, prevEr := erFor(posts), erFor(prevPosts)
	for _, p := range posts {
		points = append(points, p.PublishedAt)
	}
	isDemo := domain.IsDemo(conns)
	out := &OverviewSocial{
		HasSource: true, HasData: len(points) > 0, IsDemo: isDemo,
		Quality:        domain.Quality(true, isDemo, points, job, o.now()),
		Followers:      now,
		FollowersDelta: metrics.ComputeDelta(float64(now-start), float64(start-prevStart)),
		FollowersSpark: spark,
		Posts:          len(posts), PostsDelta: metrics.ComputeDelta(float64(len(posts)), float64(len(prevPosts))),
		EngagementRate: er, EngagementRateDelta: metrics.ComputeDelta(er, prevEr),
		Reach: reach, ReachDelta: metrics.ComputeDelta(float64(reach), float64(prevReach)),
	}
	for _, a := range accounts {
		out.Accounts = append(out.Accounts, OverviewAccount{Platform: a.Platform, Username: a.Username, Followers: followersByAcc[a.ID]})
	}
	return out, nil
}

// healthFor is the health score of the latest MOBILE audit with penalties
// from the latest successful crawl.
func healthFor(ctx context.Context, store domain.Store, propertyIDs []string) (*int, error) {
	audit, err := store.LatestAudit(ctx, propertyIDs, domain.Mobile)
	if err != nil || audit == nil {
		return nil, err
	}
	success := domain.JobSuccess
	crawl, err := store.LatestCrawl(ctx, propertyIDs, &success)
	if err != nil {
		return nil, err
	}
	var counts domain.SeverityCounts
	if crawl != nil {
		m, err := store.CountIssues(ctx, []string{crawl.ID})
		if err != nil {
			return nil, err
		}
		counts = m[crawl.ID]
	}
	scores := audit.Scores()
	return domain.HealthScore(&scores, counts.Errors, counts.Warnings), nil
}

func seoDailyFields() []func(domain.SeoDailyMetric) float64 {
	return []func(domain.SeoDailyMetric) float64{
		func(r domain.SeoDailyMetric) float64 { return float64(r.Clicks) },
		func(r domain.SeoDailyMetric) float64 { return float64(r.Impressions) },
		func(r domain.SeoDailyMetric) float64 { return r.Position },
	}
}

func seoDay(r domain.SeoDailyMetric) time.Time { return r.Date }

// Seo mirrors getOverviewSeo: days GSC did not report stay nil.
func (o *Overviews) Seo(ctx context.Context, clientID string, w Window) (*OverviewSeo, error) {
	props, err := o.Store.ListSeoProperties(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if len(props) == 0 {
		return &OverviewSeo{IsDemo: true, Quality: domain.Quality(false, true, nil, nil, o.now()), ClicksDelta: zeroDelta, ImpressionsDelta: zeroDelta, CTRDelta: zeroDelta, PositionDelta: zeroDelta, ClicksSpark: []float64{}, Daily: []SeoPoint{}}, nil
	}
	idList := make([]string, 0, len(props))
	conns := make([]*string, 0, len(props))
	for _, p := range props {
		idList = append(idList, p.ID)
		conns = append(conns, p.ConnectionID)
	}
	rows, err := o.Store.ListSeoDaily(ctx, idList, w.Current.From, w.Current.To)
	if err != nil {
		return nil, err
	}
	prevRows, err := o.Store.ListSeoDaily(ctx, idList, w.Previous.From, w.Previous.To)
	if err != nil {
		return nil, err
	}
	health, err := healthFor(ctx, o.Store, idList)
	if err != nil {
		return nil, err
	}
	job, err := o.Store.LatestSyncJob(ctx, clientID, []string{domain.KindSeoGSC, domain.KindSeoGA4})
	if err != nil {
		return nil, err
	}
	cur := domain.AggregateSearch(domain.DailyRows(rows))
	prev := domain.AggregateSearch(domain.DailyRows(prevRows))
	daily := domain.FillDaily(w.Current, rows, seoDay, seoDailyFields()[:2], false)
	points := make([]SeoPoint, 0, len(daily))
	for _, p := range daily {
		points = append(points, SeoPoint{Date: p.Date, Clicks: p.Get(0), Impressions: p.Get(1)})
	}
	dataDays := make([]time.Time, 0, len(rows))
	for _, r := range rows {
		dataDays = append(dataDays, r.Date)
	}
	isDemo := domain.IsDemo(conns)
	return &OverviewSeo{
		HasSource: true, HasData: len(rows) > 0 || len(prevRows) > 0, IsDemo: isDemo,
		Quality: domain.Quality(true, isDemo, dataDays, job, o.now()),
		Clicks:  cur.Clicks, ClicksDelta: metrics.ComputeDelta(float64(cur.Clicks), float64(prev.Clicks)),
		ClicksSpark: domain.Sparkline(daily, 0, 14),
		Impressions: cur.Impressions, ImpressionsDelta: metrics.ComputeDelta(float64(cur.Impressions), float64(prev.Impressions)),
		CTR: cur.CTR, CTRDelta: metrics.ComputeDelta(cur.CTR, prev.CTR),
		Position: cur.Position, PositionDelta: metrics.ComputeDelta(cur.Position, prev.Position),
		Health: health, Daily: points,
	}, nil
}

func adDay(r domain.AdDailyInsight) time.Time { return r.Date }

var adFields = []func(domain.AdDailyInsight) float64{
	func(r domain.AdDailyInsight) float64 { return r.Spend },
	func(r domain.AdDailyInsight) float64 { return float64(r.Results) },
}

// adsDaily fills spend/results per day (missing 0) with non-conversion
// results zeroed when conversion rows exist.
func adsDaily(r dates.Range, rows []domain.AdDailyInsight) []domain.DailyPoint {
	return domain.FillDaily(r, domain.DailySource(rows), adDay, adFields, true)
}

func adsPoints(cur, prev []domain.DailyPoint) []AdsPoint {
	out := make([]AdsPoint, 0, len(cur))
	var sp, rp []*float64
	if prev != nil {
		sp, rp = domain.AlignPrevious(cur, prev, 0), domain.AlignPrevious(cur, prev, 1)
	}
	for i, p := range cur {
		pt := AdsPoint{Date: p.Date, Spend: *p.Get(0), Results: int(*p.Get(1))}
		if prev != nil {
			pt.SpendPrev, pt.ResultsPrev = sp[i], rp[i]
		}
		out = append(out, pt)
	}
	return out
}

// Ads mirrors getOverviewAds (ad-level rows only, conversion-only results).
func (o *Overviews) Ads(ctx context.Context, clientID string, w Window) (*OverviewAds, error) {
	accounts, err := o.Store.ListAdAccounts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	empty := domain.AggregateAds(nil)
	if len(accounts) == 0 {
		return &OverviewAds{IsDemo: true, Quality: domain.Quality(false, true, nil, nil, o.now()), Kpis: empty, Prev: empty, SpendDelta: zeroDelta, ResultsDelta: zeroDelta, CPRDelta: zeroDelta, SpendSpark: []float64{}, Daily: []AdsPoint{}}, nil
	}
	idList := make([]string, 0, len(accounts))
	conns := make([]*string, 0, len(accounts))
	for _, a := range accounts {
		idList = append(idList, a.ID)
		conns = append(conns, a.ConnectionID)
	}
	rows, err := o.Store.ListInsights(ctx, idList, w.Current.From, w.Current.To, nil)
	if err != nil {
		return nil, err
	}
	prevRows, err := o.Store.ListInsights(ctx, idList, w.Previous.From, w.Previous.To, nil)
	if err != nil {
		return nil, err
	}
	job, err := o.Store.LatestSyncJob(ctx, clientID, []string{domain.KindAdsInsights, domain.KindAdsCsvImport})
	if err != nil {
		return nil, err
	}
	kpis, prev := domain.KpisWithConversion(rows), domain.KpisWithConversion(prevRows)
	daily := adsDaily(w.Current, rows)
	dataDays := make([]time.Time, 0, len(rows))
	for _, r := range rows {
		dataDays = append(dataDays, r.Date)
	}
	isDemo := domain.IsDemo(conns)
	return &OverviewAds{
		HasSource: true, HasData: len(rows) > 0 || len(prevRows) > 0, IsDemo: isDemo,
		Quality: domain.Quality(true, isDemo, dataDays, job, o.now()),
		Kpis:    kpis, Prev: prev,
		SpendDelta: metrics.ComputeDelta(kpis.Spend, prev.Spend), ResultsDelta: metrics.ComputeDelta(float64(kpis.Results), float64(prev.Results)), CPRDelta: metrics.ComputeDelta(kpis.CPR, prev.CPR),
		SpendSpark: domain.Sparkline(daily, 0, 14), Daily: adsPoints(daily, nil),
	}, nil
}

type PortfolioTotals struct {
	Followers      int
	FollowersDelta metrics.Delta
	Clicks         int
	ClicksDelta    metrics.Delta
	Spend          float64
	SpendDelta     metrics.Delta
}

// Portfolio sums followers, clicks and spend across clients the caller may
// see; ids without access are dropped.
func (o *Overviews) Portfolio(ctx context.Context, userID string, clientIDs []string, w Window) (*PortfolioTotals, error) {
	allowed := make([]string, 0, len(clientIDs))
	for _, id := range clientIDs {
		if _, err := o.view(ctx, userID, id); err == nil {
			allowed = append(allowed, id)
		}
	}
	if len(allowed) == 0 {
		return &PortfolioTotals{FollowersDelta: zeroDelta, ClicksDelta: zeroDelta, SpendDelta: zeroDelta}, nil
	}
	var accountIDs []string
	for _, id := range allowed {
		accs, err := o.Store.ListSocialAccounts(ctx, id, false, nil)
		if err != nil {
			return nil, err
		}
		for _, a := range accs {
			accountIDs = append(accountIDs, a.ID)
		}
	}
	snaps, err := o.Store.ListSnapshots(ctx, accountIDs, dates.AddDays(w.Previous.From, -1), w.Current.To)
	if err != nil {
		return nil, err
	}
	clicks, err := o.Store.SumSeoClicks(ctx, allowed, w.Current.From, w.Current.To)
	if err != nil {
		return nil, err
	}
	prevClicks, err := o.Store.SumSeoClicks(ctx, allowed, w.Previous.From, w.Previous.To)
	if err != nil {
		return nil, err
	}
	spend, err := o.Store.SumAdSpend(ctx, allowed, w.Current.From, w.Current.To)
	if err != nil {
		return nil, err
	}
	prevSpend, err := o.Store.SumAdSpend(ctx, allowed, w.Previous.From, w.Previous.To)
	if err != nil {
		return nil, err
	}
	now := domain.FollowersAtAcrossAccounts(snaps, w.Current.To)
	start := domain.FollowersAtAcrossAccounts(snaps, dates.AddDays(w.Current.From, -1))
	prevStart := domain.FollowersAtAcrossAccounts(snaps, dates.AddDays(w.Previous.From, -1))
	return &PortfolioTotals{
		Followers: now, FollowersDelta: metrics.ComputeDelta(float64(now-start), float64(start-prevStart)),
		Clicks: clicks, ClicksDelta: metrics.ComputeDelta(float64(clicks), float64(prevClicks)),
		Spend: spend, SpendDelta: metrics.ComputeDelta(spend, prevSpend),
	}, nil
}

func (o *Overviews) ListSyncJobs(ctx context.Context, userID, clientID string, limit int) ([]domain.SyncJob, error) {
	if _, err := o.view(ctx, userID, clientID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 6
	}
	return o.Store.ListSyncJobs(ctx, clientID, limit)
}

func (o *Overviews) LatestSyncByClients(ctx context.Context, clientIDs []string) (map[string]time.Time, error) {
	return o.Store.LatestSyncByClients(ctx, clientIDs)
}
