package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/sync"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

const (
	msgSyncNoAccount  = "Tidak ada akun untuk disinkronkan."
	msgAccountsUpdate = "akun diperbarui"
	msgPostsUpdate    = "post diperbarui"
	msgViaApify       = "via Apify"
)

type PostRow struct {
	domain.SocialPost
	CaptionShort string
	Type         string
	Engagements  int
	ER           float64
	Username     *string
}

func toPostRow(p domain.SocialPost, followers int, username *string) PostRow {
	return PostRow{SocialPost: p, CaptionShort: domain.Truncate(p.Caption, 90), Type: domain.ContentType(p.MediaType, p.ProductType), Engagements: domain.PostEngagements(p), ER: metrics.Round(domain.PostEngagementRate(p, followers), 2), Username: username}
}

type SocialKpis struct {
	Followers        int
	FollowersDelta   metrics.Delta
	FollowersSpark   []float64
	GrowthAbs        int
	GrowthPct        *float64
	ER               float64
	ERDelta          metrics.Delta
	Reach            int
	ReachDelta       metrics.Delta
	Impressions      int
	ImpressionsDelta metrics.Delta
	Posts            int
	PostsDelta       metrics.Delta
	PostsPerWeek     float64
}

type GrowthPoint struct {
	Date                     string
	Followers, FollowersPrev *int
}

type SocialDashboard struct {
	IsDemo       bool
	Account      *domain.SocialAccount
	Kpis         *SocialKpis
	GrowthSeries []GrowthPoint
	ReachSeries  []domain.DailySocialPoint
	ErByType     []domain.ErByType
	Heatmap      []domain.HeatCell
	TopPosts     []PostRow
	Posts        []PostRow
}

type CompetitorRow struct {
	AccountID    string
	Username     string
	DisplayName  string
	AvatarURL    *string
	IsOwn        bool
	Followers    int
	GrowthAbs    int
	GrowthPct    *float64
	GrowthDelta  metrics.Delta
	AvgER        float64
	PostsPerWeek float64
	MediaCount   int
}

type ChartPoint struct {
	Date   string
	Values []*float64
}

type CompetitorDashboard struct {
	HasOwn   bool
	Rows     []CompetitorRow
	Series   [][2]string
	Chart    []ChartPoint
	TopPosts []PostRow
}

type Social struct{ Deps }

func NewSocial(d Deps) *Social { return &Social{d} }

func sumReach(snaps []domain.SocialSnapshot, r dates.Range, impressions bool) int {
	sum := 0
	for _, s := range snaps {
		if !inRange(s.Date, r) {
			continue
		}
		v := s.Reach
		if impressions {
			v = s.Impressions
		}
		if v != nil {
			sum += *v
		}
	}
	return sum
}

func (s *Social) Dashboard(ctx context.Context, userID, clientID string, platform domain.Platform, w Window) (*SocialDashboard, error) {
	if _, err := s.view(ctx, userID, clientID); err != nil {
		return nil, err
	}
	if !domain.IsPlatform(platform) {
		platform = domain.Instagram
	}
	accounts, err := s.Store.ListSocialAccounts(ctx, clientID, false, &platform)
	if err != nil {
		return nil, err
	}
	empty := &SocialDashboard{IsDemo: true, GrowthSeries: []GrowthPoint{}, ReachSeries: []domain.DailySocialPoint{}, ErByType: []domain.ErByType{}, Heatmap: []domain.HeatCell{}, TopPosts: []PostRow{}, Posts: []PostRow{}}
	if len(accounts) == 0 {
		return empty, nil
	}
	acc := accounts[0]
	cur, prev := w.Current, w.Previous
	snaps, err := s.Store.ListSnapshots(ctx, []string{acc.ID}, dates.AddDays(prev.From, -1), cur.To)
	if err != nil {
		return nil, err
	}
	posts, err := s.Store.ListPosts(ctx, []string{acc.ID}, cur.From, endExclusive(cur.To))
	if err != nil {
		return nil, err
	}
	prevPosts, err := s.Store.ListPosts(ctx, []string{acc.ID}, prev.From, endExclusive(prev.To))
	if err != nil {
		return nil, err
	}
	now := domain.FollowersAt(snaps, cur.To)
	start := domain.FollowersAt(snaps, dates.AddDays(cur.From, -1))
	prevStart := domain.FollowersAt(snaps, dates.AddDays(prev.From, -1))
	growth := domain.FollowerGrowth(start, now)

	curSeries := domain.DailyFollowerSeries(cur, snaps)
	prevSeries := domain.DailyFollowerSeries(prev, snaps)
	growthSeries := make([]GrowthPoint, 0, len(curSeries))
	spark := []float64{}
	for i, p := range curSeries {
		g := GrowthPoint{Date: p.Date, Followers: p.Followers}
		if i < len(prevSeries) {
			g.FollowersPrev = prevSeries[i].Followers
		}
		growthSeries = append(growthSeries, g)
		if p.Followers != nil {
			spark = append(spark, float64(*p.Followers))
		}
	}
	if len(spark) > 14 {
		spark = spark[len(spark)-14:]
	}
	reach, prevReach := sumReach(snaps, cur, false), sumReach(snaps, prev, false)
	impr, prevImpr := sumReach(snaps, cur, true), sumReach(snaps, prev, true)
	er := domain.AvgEngagementRate(posts, now)
	base := start
	if base == 0 {
		base = now
	}
	prevEr := domain.AvgEngagementRate(prevPosts, base)

	out := &SocialDashboard{
		IsDemo: acc.ConnectionID == nil, Account: &acc,
		Kpis: &SocialKpis{
			Followers: now, FollowersDelta: metrics.ComputeDelta(float64(growth.Abs), float64(start-prevStart)), FollowersSpark: spark,
			GrowthAbs: growth.Abs, GrowthPct: growth.Pct,
			ER: er, ERDelta: metrics.ComputeDelta(er, prevEr),
			Reach: reach, ReachDelta: metrics.ComputeDelta(float64(reach), float64(prevReach)),
			Impressions: impr, ImpressionsDelta: metrics.ComputeDelta(float64(impr), float64(prevImpr)),
			Posts: len(posts), PostsDelta: metrics.ComputeDelta(float64(len(posts)), float64(len(prevPosts))),
			PostsPerWeek: domain.PostsPerWeek(len(posts), cur.Days()),
		},
		GrowthSeries: growthSeries, ReachSeries: curSeries,
		ErByType: domain.ErByContentType(posts, now), Heatmap: domain.PostingHeatmap(posts),
	}
	for _, p := range domain.TopPosts(posts, 6) {
		out.TopPosts = append(out.TopPosts, toPostRow(p, now, nil))
	}
	for _, p := range posts {
		out.Posts = append(out.Posts, toPostRow(p, now, nil))
	}
	return out, nil
}

func (s *Social) Competitors(ctx context.Context, userID, clientID string, w Window) (*CompetitorDashboard, error) {
	if _, err := s.view(ctx, userID, clientID); err != nil {
		return nil, err
	}
	ig := domain.Instagram
	accounts, err := s.Store.ListSocialAccounts(ctx, clientID, true, &ig)
	if err != nil {
		return nil, err
	}
	sortBy(accounts, func(a, b domain.SocialAccount) bool { return a.CreatedAt.Before(b.CreatedAt) })
	out := &CompetitorDashboard{Rows: []CompetitorRow{}, Series: [][2]string{}, Chart: []ChartPoint{}, TopPosts: []PostRow{}}
	if len(accounts) == 0 {
		return out, nil
	}
	idList := make([]string, 0, len(accounts))
	for _, a := range accounts {
		idList = append(idList, a.ID)
	}
	cur, prev := w.Current, w.Previous
	snaps, err := s.Store.ListSnapshots(ctx, idList, dates.AddDays(prev.From, -1), cur.To)
	if err != nil {
		return nil, err
	}
	posts, err := s.Store.ListPosts(ctx, idList, cur.From, endExclusive(cur.To))
	if err != nil {
		return nil, err
	}
	snapsBy := map[string][]domain.SocialSnapshot{}
	for _, sn := range snaps {
		snapsBy[sn.AccountID] = append(snapsBy[sn.AccountID], sn)
	}
	postsBy := map[string][]domain.SocialPost{}
	for _, p := range posts {
		postsBy[p.AccountID] = append(postsBy[p.AccountID], p)
	}
	days := cur.Days()
	build := func(a domain.SocialAccount) CompetitorRow {
		ss := snapsBy[a.ID]
		mine := postsBy[a.ID]
		followers := domain.FollowersAt(ss, cur.To)
		start := domain.FollowersAt(ss, dates.AddDays(cur.From, -1))
		prevStart := domain.FollowersAt(ss, dates.AddDays(prev.From, -1))
		g := domain.FollowerGrowth(start, followers)
		media := 0
		for _, sn := range ss {
			if !sn.Date.After(cur.To) {
				media = sn.MediaCount
			}
		}
		return CompetitorRow{AccountID: a.ID, Username: a.Username, DisplayName: a.DisplayName, AvatarURL: a.AvatarURL, IsOwn: !a.IsCompetitor, Followers: followers, GrowthAbs: g.Abs, GrowthPct: g.Pct, GrowthDelta: metrics.ComputeDelta(float64(g.Abs), float64(start-prevStart)), AvgER: domain.AvgEngagementRate(mine, followers), PostsPerWeek: domain.PostsPerWeek(len(mine), days), MediaCount: media}
	}
	var compRows []CompetitorRow
	usernames := map[string]string{}
	competitorIDs := map[string]bool{}
	for _, a := range accounts {
		usernames[a.ID] = a.Username
		if a.IsCompetitor {
			competitorIDs[a.ID] = true
			compRows = append(compRows, build(a))
		} else if !out.HasOwn {
			out.HasOwn = true
			out.Rows = append(out.Rows, build(a))
		}
	}
	sortBy(compRows, func(x, y CompetitorRow) bool { return x.Followers > y.Followers })
	out.Rows = append(out.Rows, compRows...)

	chartRows := out.Rows
	if len(chartRows) > 4 {
		chartRows = chartRows[:4]
	}
	var perAcc [][]*float64
	var axis []string
	for i, r := range chartRows {
		series := domain.DailyFollowerSeries(cur, snapsBy[r.AccountID])
		followers := make([]*int, 0, len(series))
		if i == 0 {
			for _, p := range series {
				axis = append(axis, p.Date)
			}
		}
		for _, p := range series {
			followers = append(followers, p.Followers)
		}
		idx := domain.IndexTo100(followers)
		for j, v := range idx {
			if v != nil {
				idx[j] = ptr(metrics.Round(*v, 1))
			}
		}
		perAcc = append(perAcc, idx)
		out.Series = append(out.Series, [2]string{fmt.Sprintf("a%d", i), "@" + r.Username})
	}
	for j, d := range axis {
		pt := ChartPoint{Date: d}
		for _, vals := range perAcc {
			if j < len(vals) {
				pt.Values = append(pt.Values, vals[j])
			} else {
				pt.Values = append(pt.Values, nil)
			}
		}
		out.Chart = append(out.Chart, pt)
	}

	followersBy := map[string]int{}
	for _, r := range out.Rows {
		followersBy[r.AccountID] = r.Followers
	}
	var compPosts []domain.SocialPost
	for _, p := range posts {
		if competitorIDs[p.AccountID] {
			compPosts = append(compPosts, p)
		}
	}
	for _, p := range domain.TopPosts(compPosts, 6) {
		u := usernames[p.AccountID]
		out.TopPosts = append(out.TopPosts, toPostRow(p, followersBy[p.AccountID], &u))
	}
	return out, nil
}

func failureText(failed []sync.Failure) *string {
	if len(failed) == 0 {
		return nil
	}
	parts := make([]string, 0, len(failed))
	for _, f := range failed {
		parts = append(parts, "@"+f.Username+": "+f.Error)
	}
	s := strings.Join(parts, "; ")
	if len(s) > 1000 {
		s = s[:1000]
	}
	return &s
}

func (s *Social) job(ctx context.Context, clientID string, status domain.JobStatus, message, errMsg *string) error {
	now := s.now()
	_, err := s.Store.InsertSyncJob(ctx, domain.SyncJob{ClientID: clientID, Kind: domain.KindSocialSnapshot, Status: status, StartedAt: now, FinishedAt: &now, Message: message, Error: errMsg})
	return err
}

// Sync is the manual "Sinkronkan": Meta for connected accounts, then Apify,
// then demo extrapolation. Any member may call it.
func (s *Social) Sync(ctx context.Context, userID, clientID string, platform domain.Platform) (int, error) {
	if !domain.IsPlatform(platform) {
		return 0, apperr.New(apperr.Validation, "")
	}
	if _, err := s.act(ctx, userID, clientID); err != nil {
		return 0, err
	}
	accounts, err := s.Store.ListSocialAccounts(ctx, clientID, false, &platform)
	if err != nil {
		return 0, err
	}
	if len(accounts) == 0 {
		return 0, apperr.New(apperr.NotFound, msgSyncNoAccount)
	}
	updated, failure, err := s.sync(ctx, clientID, platform, accounts)
	if failure != nil {
		return 0, failure
	}
	if err != nil {
		var ae *apperr.Error
		code := apperr.Unknown
		if e := apperr.As(err); e.Provider != "" {
			ae = apperr.New(e.Code, "")
			code = e.Code
		} else {
			ae = apperr.New(apperr.Unknown, "")
		}
		_ = s.job(ctx, clientID, domain.JobFailed, nil, ptr(apperr.Message(code)))
		return 0, ae
	}
	return updated, nil
}

// sync returns a failure (already logged as a job) separately from an
// unexpected error, which the caller logs as a FAILED job.
func (s *Social) sync(ctx context.Context, clientID string, platform domain.Platform, accounts []domain.SocialAccount) (int, *apperr.Error, error) {
	lower := strings.ToLower(string(platform))
	hasConn := false
	for _, a := range accounts {
		if a.ConnectionID != nil {
			hasConn = true
		}
	}
	if platform != domain.TikTok && hasConn {
		r, err := s.Syncer.SyncSocialMeta(ctx, clientID, platform, 30)
		if err != nil {
			return 0, nil, err
		}
		if r.Updated > 0 || len(r.Failed) > 0 {
			status := domain.JobSuccess
			if r.Updated == 0 {
				status = domain.JobFailed
			}
			msg := fmt.Sprintf("%d %s, %d %s (%s · Meta)", r.Updated, msgAccountsUpdate, r.Posts, msgPostsUpdate, lower)
			if err := s.job(ctx, clientID, status, &msg, failureText(r.Failed)); err != nil {
				return 0, nil, err
			}
			if r.Updated == 0 && len(r.Failed) > 0 {
				return 0, apperr.New(apperr.Unknown, r.Failed[0].Error), nil
			}
			return r.Updated, nil, nil
		}
	}
	if s.Syncer.ApifyConfigured(ctx) {
		r, err := s.Syncer.SyncSocialPublic(ctx, clientID, platform, true, 0)
		if err != nil {
			return 0, nil, err
		}
		status := domain.JobSuccess
		if len(r.Failed) > 0 && r.Updated == 0 {
			status = domain.JobFailed
		}
		msg := fmt.Sprintf("%d %s, %d %s (%s · %s)", r.Updated, msgAccountsUpdate, r.Posts, msgPostsUpdate, lower, msgViaApify)
		if err := s.job(ctx, clientID, status, &msg, failureText(r.Failed)); err != nil {
			return 0, nil, err
		}
		if r.Updated == 0 && len(r.Failed) > 0 {
			return 0, apperr.New(apperr.Unknown, r.Failed[0].Error), nil
		}
		return r.Updated, nil, nil
	}
	updated, err := s.Syncer.SyncSocialDemo(ctx, accounts)
	if err != nil {
		return 0, nil, err
	}
	msg := fmt.Sprintf("%d %s (%s)", updated, msgAccountsUpdate, lower)
	return updated, nil, s.job(ctx, clientID, domain.JobSuccess, &msg, nil)
}
