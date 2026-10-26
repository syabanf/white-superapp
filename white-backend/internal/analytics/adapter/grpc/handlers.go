package grpcapi

import (
	"context"

	analyticsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/analytics/v1"
	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/app"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Services struct {
	Sources   *app.Sources
	Overviews *app.Overviews
	Social    *app.Social
	Seo       *app.Seo
	Ads       *app.Ads
	Reports   *app.Reports
	Clock     clock.Clock
}

func Register(s *grpc.Server, svc Services) {
	analyticsv1.RegisterSourceServiceServer(s, &sourceServer{svc: svc})
	analyticsv1.RegisterOverviewServiceServer(s, &overviewServer{svc: svc})
	analyticsv1.RegisterSocialServiceServer(s, &socialServer{svc: svc})
	analyticsv1.RegisterSeoServiceServer(s, &seoServer{svc: svc})
	analyticsv1.RegisterAdsServiceServer(s, &adsServer{svc: svc})
	analyticsv1.RegisterReportDataServiceServer(s, &reportServer{svc: svc})
}

func (svc Services) window(q *commonv1.RangeQuery) (dates.Resolved, app.Window) {
	r := dates.Resolve(q, svc.Clock.Now())
	return r, app.WindowOf(r)
}

// listCaller resolves the user of a list RPC that other services also call
// internally: a principal is checked; an internal call without one is
// trusted; a gateway call without one is UNAUTHENTICATED.
func listCaller(ctx context.Context) (string, error) {
	if p, ok := authn.FromContext(ctx); ok {
		return p.UserID, nil
	}
	if grpcx.FromEdge(ctx) {
		return "", apperr.New(apperr.Unauthenticated, "")
	}
	return "", nil
}

// ─── SourceService ───────────────────────────────────────────

type sourceServer struct {
	analyticsv1.UnimplementedSourceServiceServer
	svc Services
}

func (s *sourceServer) ListSocialAccounts(ctx context.Context, req *analyticsv1.ListSocialAccountsRequest) (*analyticsv1.ListSocialAccountsResponse, error) {
	uid, err := listCaller(ctx)
	if err != nil {
		return nil, err
	}
	var platform *domain.Platform
	if p := platformFrom(req.Platform); p != "" {
		platform = &p
	}
	rows, err := s.svc.Sources.ListSocialAccounts(ctx, uid, req.ClientId, req.IncludeCompetitors, platform)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.ListSocialAccountsResponse{}
	for _, r := range rows {
		out.Accounts = append(out.Accounts, socialAccountPB(r))
	}
	return out, nil
}

func (s *sourceServer) ListSeoProperties(ctx context.Context, req *analyticsv1.ListSeoPropertiesRequest) (*analyticsv1.ListSeoPropertiesResponse, error) {
	uid, err := listCaller(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.svc.Sources.ListSeoProperties(ctx, uid, req.ClientId)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.ListSeoPropertiesResponse{}
	for _, r := range rows {
		out.Properties = append(out.Properties, seoPropertyPB(r))
	}
	return out, nil
}

func (s *sourceServer) ListAdAccounts(ctx context.Context, req *analyticsv1.ListAdAccountsRequest) (*analyticsv1.ListAdAccountsResponse, error) {
	uid, err := listCaller(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.svc.Sources.ListAdAccounts(ctx, uid, req.ClientId)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.ListAdAccountsResponse{}
	for _, r := range rows {
		out.Accounts = append(out.Accounts, adAccountPB(r))
	}
	return out, nil
}

func (s *sourceServer) AddSocialAccount(ctx context.Context, req *analyticsv1.AddSocialAccountRequest) (*analyticsv1.SocialAccount, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Sources.AddSocialAccount(ctx, p.UserID, req.ClientId, platformFrom(req.Platform), req.Username, req.DisplayName)
	if err != nil {
		return nil, err
	}
	return socialAccountPB(*a), nil
}

func (s *sourceServer) RemoveSocialAccount(ctx context.Context, req *analyticsv1.RemoveSocialAccountRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Sources.RemoveSocialAccount(ctx, p.UserID, req.ClientId, req.AccountId)
}

func (s *sourceServer) AddCompetitor(ctx context.Context, req *analyticsv1.AddCompetitorRequest) (*analyticsv1.AddCompetitorResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	id, username, err := s.svc.Sources.AddCompetitor(ctx, p.UserID, req.ClientId, req.Username)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.AddCompetitorResponse{AccountId: id, Username: username}, nil
}

func (s *sourceServer) RemoveCompetitor(ctx context.Context, req *analyticsv1.RemoveCompetitorRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Sources.RemoveCompetitor(ctx, p.UserID, req.ClientId, req.AccountId)
}

func (s *sourceServer) AddSeoProperty(ctx context.Context, req *analyticsv1.AddSeoPropertyRequest) (*analyticsv1.SeoProperty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	prop, err := s.svc.Sources.AddSeoProperty(ctx, p.UserID, req.ClientId, req.SiteUrl, req.Ga4PropertyId)
	if err != nil {
		return nil, err
	}
	return seoPropertyPB(*prop), nil
}

func (s *sourceServer) RemoveSeoProperty(ctx context.Context, req *analyticsv1.RemoveSeoPropertyRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Sources.RemoveSeoProperty(ctx, p.UserID, req.ClientId, req.PropertyId)
}

func (s *sourceServer) SetAuditCadence(ctx context.Context, req *analyticsv1.SetAuditCadenceRequest) (*analyticsv1.SetAuditCadenceResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Sources.SetAuditCadence(ctx, p.UserID, req.ClientId, req.PropertyId, cadenceFrom(req.Cadence))
	if err != nil {
		return nil, err
	}
	return &analyticsv1.SetAuditCadenceResponse{Cadence: cadencePB(c)}, nil
}

func (s *sourceServer) AddAdAccount(ctx context.Context, req *analyticsv1.AddAdAccountRequest) (*analyticsv1.AdAccount, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Sources.AddAdAccount(ctx, p.UserID, req.ClientId, req.ExternalId, req.Name)
	if err != nil {
		return nil, err
	}
	return adAccountPB(*a), nil
}

func (s *sourceServer) RemoveAdAccount(ctx context.Context, req *analyticsv1.RemoveAdAccountRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Sources.RemoveAdAccount(ctx, p.UserID, req.ClientId, req.AccountId)
}

func (s *sourceServer) GetClientSources(ctx context.Context, req *analyticsv1.GetClientSourcesRequest) (*analyticsv1.GetClientSourcesResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	cs, err := s.svc.Sources.GetClientSources(ctx, p.UserID, req.ClientId)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.GetClientSourcesResponse{}
	for _, a := range cs.Social {
		out.SocialAccounts = append(out.SocialAccounts, socialAccountPB(a))
	}
	for _, pr := range cs.Seo {
		out.SeoProperties = append(out.SeoProperties, seoPropertyPB(pr))
	}
	for _, a := range cs.Ads {
		out.AdAccounts = append(out.AdAccounts, adAccountPB(a))
	}
	return out, nil
}

func (s *sourceServer) GetSeoProperty(ctx context.Context, req *analyticsv1.GetSeoPropertyRequest) (*analyticsv1.SeoProperty, error) {
	p, err := s.svc.Sources.GetSeoProperty(ctx, req.PropertyId)
	if err != nil {
		return nil, err
	}
	return seoPropertyPB(*p), nil
}

func (s *sourceServer) ListSocialPosts(ctx context.Context, req *analyticsv1.ListSocialPostsRequest) (*analyticsv1.ListSocialPostsResponse, error) {
	rows, err := s.svc.Sources.ListSocialPosts(ctx, domain.PostFilter{ClientID: req.ClientId, AccountIDs: req.SocialAccountIds, ExternalIDs: req.ExternalIds, Limit: int(req.Limit)}, req.FromDay, req.ToDay)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.ListSocialPostsResponse{}
	for _, r := range rows {
		out.Posts = append(out.Posts, socialPostPB(r))
	}
	return out, nil
}

// ─── OverviewService ─────────────────────────────────────────

type overviewServer struct {
	analyticsv1.UnimplementedOverviewServiceServer
	svc Services
}

func (s *overviewServer) GetOverview(ctx context.Context, req *analyticsv1.GetOverviewRequest) (*analyticsv1.Overview, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, w := s.svc.window(req.Range)
	o, err := s.svc.Overviews.Get(ctx, p.UserID, req.ClientId, w)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.Overview{Range: r.Proto(), Social: overviewSocialPB(o.Social), Seo: overviewSeoPB(o.Seo), Ads: overviewAdsPB(o.Ads)}, nil
}

func (s *overviewServer) GetPortfolioTotals(ctx context.Context, req *analyticsv1.GetPortfolioTotalsRequest) (*analyticsv1.PortfolioTotals, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, w := s.svc.window(req.Range)
	t, err := s.svc.Overviews.Portfolio(ctx, p.UserID, req.ClientIds, w)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.PortfolioTotals{Range: r.Proto(), Followers: i64(t.Followers), FollowersDelta: delta(t.FollowersDelta), Clicks: i64(t.Clicks), ClicksDelta: delta(t.ClicksDelta), Spend: t.Spend, SpendDelta: delta(t.SpendDelta)}, nil
}

func (s *overviewServer) ListSyncJobs(ctx context.Context, req *analyticsv1.ListSyncJobsRequest) (*analyticsv1.ListSyncJobsResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	jobs, err := s.svc.Overviews.ListSyncJobs(ctx, p.UserID, req.ClientId, int(req.Limit))
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.ListSyncJobsResponse{}
	for _, j := range jobs {
		out.Jobs = append(out.Jobs, syncJobPB(j))
	}
	return out, nil
}

func (s *overviewServer) GetLatestSyncByClients(ctx context.Context, req *analyticsv1.GetLatestSyncByClientsRequest) (*analyticsv1.GetLatestSyncByClientsResponse, error) {
	m, err := s.svc.Overviews.LatestSyncByClients(ctx, req.ClientIds)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.GetLatestSyncByClientsResponse{Latest: map[string]*timestamppb.Timestamp{}}
	for k, v := range m {
		out.Latest[k] = ts(v)
	}
	return out, nil
}

// ─── SocialService ───────────────────────────────────────────

type socialServer struct {
	analyticsv1.UnimplementedSocialServiceServer
	svc Services
}

func (s *socialServer) GetSocialDashboard(ctx context.Context, req *analyticsv1.GetSocialDashboardRequest) (*analyticsv1.SocialDashboard, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, w := s.svc.window(req.Range)
	d, err := s.svc.Social.Dashboard(ctx, p.UserID, req.ClientId, platformFrom(req.Platform), w)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.SocialDashboard{Range: r.Proto(), IsDemo: d.IsDemo}
	if d.Account != nil {
		out.Account = &analyticsv1.SocialDashboardAccount{Id: d.Account.ID, Username: d.Account.Username, DisplayName: d.Account.DisplayName, AvatarUrl: d.Account.AvatarURL}
	}
	if k := d.Kpis; k != nil {
		out.Kpis = &analyticsv1.SocialKpis{Followers: i64(k.Followers), FollowersDelta: delta(k.FollowersDelta), FollowersSpark: floats(k.FollowersSpark), GrowthAbs: i64(k.GrowthAbs), GrowthPct: k.GrowthPct, Er: k.ER, ErDelta: delta(k.ERDelta), Reach: i64(k.Reach), ReachDelta: delta(k.ReachDelta), Impressions: i64(k.Impressions), ImpressionsDelta: delta(k.ImpressionsDelta), Posts: i64(k.Posts), PostsDelta: delta(k.PostsDelta), PostsPerWeek: k.PostsPerWeek}
	}
	for _, g := range d.GrowthSeries {
		out.GrowthSeries = append(out.GrowthSeries, &analyticsv1.GrowthPoint{Date: g.Date, Followers: i64pi(g.Followers), FollowersPrev: i64pi(g.FollowersPrev)})
	}
	for _, rp := range d.ReachSeries {
		out.ReachSeries = append(out.ReachSeries, &analyticsv1.ReachPoint{Date: rp.Date, Reach: i64(rp.Reach), Impressions: i64(rp.Impressions)})
	}
	for _, e := range d.ErByType {
		out.ErByType = append(out.ErByType, &analyticsv1.ErByType{Type: e.Type, AvgEr: e.AvgEr, Count: int32(e.Count)})
	}
	for _, c := range d.Heatmap {
		out.Heatmap = append(out.Heatmap, &analyticsv1.HeatCell{Day: int32(c.Day), Hour: int32(c.Hour), Posts: int32(c.Posts), AvgEngagement: c.AvgEngagement})
	}
	for _, pr := range d.TopPosts {
		out.TopPosts = append(out.TopPosts, postRowPB(pr))
	}
	for _, pr := range d.Posts {
		out.Posts = append(out.Posts, postRowPB(pr))
	}
	return out, nil
}

func (s *socialServer) GetCompetitorDashboard(ctx context.Context, req *analyticsv1.GetCompetitorDashboardRequest) (*analyticsv1.CompetitorDashboard, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, w := s.svc.window(req.Range)
	d, err := s.svc.Social.Competitors(ctx, p.UserID, req.ClientId, w)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.CompetitorDashboard{Range: r.Proto(), HasOwn: d.HasOwn}
	for _, row := range d.Rows {
		out.Rows = append(out.Rows, &analyticsv1.CompetitorRow{AccountId: row.AccountID, Username: row.Username, DisplayName: row.DisplayName, AvatarUrl: row.AvatarURL, IsOwn: row.IsOwn, Followers: i64(row.Followers), GrowthAbs: i64(row.GrowthAbs), GrowthPct: row.GrowthPct, GrowthDelta: delta(row.GrowthDelta), AvgEr: row.AvgER, PostsPerWeek: row.PostsPerWeek, MediaCount: i64(row.MediaCount)})
	}
	for _, sr := range d.Series {
		out.Series = append(out.Series, &analyticsv1.CompetitorSeries{Key: sr[0], Label: sr[1]})
	}
	for _, pt := range d.Chart {
		cp := &analyticsv1.CompetitorChartPoint{Date: pt.Date}
		for i, v := range pt.Values {
			cp.Values = append(cp.Values, &analyticsv1.CompetitorChartValue{Key: d.Series[i][0], Value: v})
		}
		out.Chart = append(out.Chart, cp)
	}
	for _, pr := range d.TopPosts {
		out.TopPosts = append(out.TopPosts, postRowPB(pr))
	}
	return out, nil
}

func (s *socialServer) SyncSocial(ctx context.Context, req *analyticsv1.SyncSocialRequest) (*analyticsv1.SyncSocialResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	n, err := s.svc.Social.Sync(ctx, p.UserID, req.ClientId, platformFrom(req.Platform))
	if err != nil {
		return nil, err
	}
	return &analyticsv1.SyncSocialResponse{Updated: int32(n)}, nil
}

// ─── SeoService ──────────────────────────────────────────────

type seoServer struct {
	analyticsv1.UnimplementedSeoServiceServer
	svc Services
}

func (s *seoServer) GetSeoOverview(ctx context.Context, req *analyticsv1.SeoPropertyRangeRequest) (*analyticsv1.SeoOverview, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, w := s.svc.window(req.Range)
	o, err := s.svc.Seo.Overview(ctx, p.UserID, req.ClientId, req.PropertyId, w)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.SeoOverview{
		Range: r.Proto(), HasData: o.HasData,
		Kpis:        &analyticsv1.SeoKpis{Clicks: i64(o.Kpis.Clicks), Impressions: i64(o.Kpis.Impressions), Ctr: o.Kpis.CTR, Position: o.Kpis.Position},
		Deltas:      &analyticsv1.SeoDeltas{Clicks: delta(o.Clicks), Impressions: delta(o.Impressions), Ctr: delta(o.CTR), Position: delta(o.Position)},
		ClicksSpark: floats(o.ClicksSpark), ImpressionsSpark: floats(o.ImpressionsSpark),
		Ga4:    &analyticsv1.Ga4Summary{HasData: o.Ga4.HasData, OrganicSessions: i64(o.Ga4.OrganicSessions), Sessions: i64(o.Ga4.Sessions), Conversions: i64(o.Ga4.Convs), OrganicDelta: delta(o.Ga4.OrganicDelta), ConversionsDelta: delta(o.Ga4.ConversionsDelta), OrganicSpark: floats(o.Ga4.OrganicSpark)},
		Health: i32p(o.Health),
	}
	for _, d := range o.Daily {
		out.Daily = append(out.Daily, &analyticsv1.SearchDailyPoint{Date: d.Date, Clicks: i64p(d.Clicks), Impressions: i64p(d.Impressions), Position: d.Position, ClicksPrev: i64p(d.ClicksPrev), ImpressionsPrev: i64p(d.ImpressionsPrev), PositionPrev: d.PositionPrev})
	}
	for _, g := range o.Ga4Daily {
		out.Ga4Daily = append(out.Ga4Daily, &analyticsv1.Ga4DailyPoint{Date: g.Date, Organik: i64(g.Organik), Lainnya: i64(g.Lainnya)})
	}
	return out, nil
}

func (s *seoServer) GetQueryStats(ctx context.Context, req *analyticsv1.SeoPropertyRangeRequest) (*analyticsv1.QueryStats, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, w := s.svc.window(req.Range)
	q, err := s.svc.Seo.Queries(ctx, p.UserID, req.ClientId, req.PropertyId, w)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.QueryStats{Range: r.Proto(), Opportunities: &analyticsv1.KeywordOpportunities{Striking: opportunitiesPB(q.Striking), LowCtr: opportunitiesPB(q.LowCTR), Declining: opportunitiesPB(q.Declining), Rising: opportunitiesPB(q.Rising)}}
	for _, row := range q.Rows {
		out.Rows = append(out.Rows, searchAggQuery(row))
	}
	for _, b := range q.Distribution {
		out.Distribution = append(out.Distribution, &analyticsv1.PositionBucketRow{Bucket: b.Bucket, Queries: int32(b.Queries), Clicks: i64(b.Clicks), Impressions: i64(b.Impressions)})
	}
	return out, nil
}

func (s *seoServer) GetPageStats(ctx context.Context, req *analyticsv1.SeoPropertyRangeRequest) (*analyticsv1.PageStats, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, w := s.svc.window(req.Range)
	rows, err := s.svc.Seo.Pages(ctx, p.UserID, req.ClientId, req.PropertyId, w)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.PageStats{Range: r.Proto()}
	for _, row := range rows {
		pb := searchAggQuery(row)
		pb.Path = domain.URLPath(row.Key)
		out.Rows = append(out.Rows, pb)
	}
	return out, nil
}

func (s *seoServer) GetDimBreakdown(ctx context.Context, req *analyticsv1.SeoPropertyRangeRequest) (*analyticsv1.DimBreakdown, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, w := s.svc.window(req.Range)
	devices, countries, err := s.svc.Seo.Breakdown(ctx, p.UserID, req.ClientId, req.PropertyId, w)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.DimBreakdown{Range: r.Proto(), Devices: dimAggPB(devices), Countries: dimAggPB(countries)}, nil
}

func (s *seoServer) GetAuditData(ctx context.Context, req *analyticsv1.SeoPropertyRequest) (*analyticsv1.AuditData, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Seo.Audit(ctx, p.UserID, req.ClientId, req.PropertyId)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.AuditData{LatestMobile: auditRowPB(a.LatestMobile), LatestDesktop: auditRowPB(a.LatestDesktop), LastRunAt: tsPtr(a.LastRunAt), SeverityCounts: &analyticsv1.SeverityCounts{Errors: int32(a.Counts.Errors), Warnings: int32(a.Counts.Warnings), Notices: int32(a.Counts.Notices)}}
	for _, h := range a.History {
		out.History = append(out.History, &analyticsv1.ScoreHistoryPoint{Date: h.Date, Performance: int32(h.Scores.Performance), Seo: int32(h.Scores.Seo), Accessibility: int32(h.Scores.Accessibility), BestPractices: int32(h.Scores.BestPractices)})
	}
	if c := a.Crawl; c != nil {
		out.Crawl = &analyticsv1.CrawlSummary{Id: c.ID, Status: jobStatusPB(c.Status), StartedAt: ts(c.StartedAt), FinishedAt: tsPtr(c.FinishedAt), PagesCrawled: int32(c.PagesCrawled), MaxPages: int32(c.MaxPages), Error: c.Error}
	}
	for _, pg := range a.Pages {
		row := &analyticsv1.CrawlPageRow{Id: pg.ID, Url: pg.URL, Path: domain.URLPath(pg.URL), StatusCode: int32(pg.StatusCode), H1Count: int32(pg.H1Count), WordCount: int32(pg.WordCount), ImagesMissingAlt: int32(pg.ImagesMissingAlt), Indexable: pg.Indexable, LoadMs: i32p(pg.LoadMs)}
		if pg.Title != nil {
			row.TitleLength = int32(len([]rune(*pg.Title)))
		}
		if pg.MetaDescription != nil {
			row.MetaLength = int32(len([]rune(*pg.MetaDescription)))
		}
		out.Pages = append(out.Pages, row)
	}
	for _, i := range a.Issues {
		out.Issues = append(out.Issues, &analyticsv1.IssueRow{Id: i.ID, Severity: severityPB(i.Severity), Code: i.Code, Message: i.Message, Url: i.URL})
	}
	for _, c := range a.CodeCounts {
		out.CodeCounts = append(out.CodeCounts, &analyticsv1.CodeCount{Code: c.Code, Count: int32(c.Count)})
	}
	return out, nil
}

func (s *seoServer) GetIssueTrend(ctx context.Context, req *analyticsv1.SeoPropertyRequest) (*analyticsv1.IssueTrend, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	pts, err := s.svc.Seo.IssueTrend(ctx, p.UserID, req.ClientId, req.PropertyId)
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.IssueTrend{}
	for _, t := range pts {
		out.Points = append(out.Points, &analyticsv1.IssueTrendPoint{Date: t.Date, Errors: int32(t.Counts.Errors), Warnings: int32(t.Counts.Warnings), Notices: int32(t.Counts.Notices), Pages: int32(t.Pages)})
	}
	return out, nil
}

func (s *seoServer) RunAudit(ctx context.Context, req *analyticsv1.RunAuditRequest) (*analyticsv1.RunAuditResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Seo.RunAudit(ctx, p.UserID, req.ClientId, req.PropertyId, req.Url)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.RunAuditResponse{Message: r.Message, Audits: int32(r.Audits), PagesCrawled: int32(r.PagesCrawled), Issues: int32(r.Issues)}, nil
}

func (s *seoServer) SyncSeo(ctx context.Context, req *analyticsv1.SyncSeoRequest) (*analyticsv1.SyncSeoResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Seo.Sync(ctx, p.UserID, req.ClientId)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.SyncSeoResponse{Mode: r.Mode, Daily: int32(r.Daily), Ga4: int32(r.Ga4)}, nil
}

// ─── AdsService ──────────────────────────────────────────────

type adsServer struct {
	analyticsv1.UnimplementedAdsServiceServer
	svc Services
}

func (s *adsServer) GetAdsDashboard(ctx context.Context, req *analyticsv1.GetAdsDashboardRequest) (*analyticsv1.AdsDashboard, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, w := s.svc.window(req.Range)
	d, err := s.svc.Ads.Dashboard(ctx, p.UserID, req.ClientId, w, req.CampaignId)
	if err != nil {
		return nil, err
	}
	dl := d.Deltas
	out := &analyticsv1.AdsDashboard{
		Range: r.Proto(), HasData: d.HasData, IsDemo: d.IsDemo, Kpis: adKpisPB(d.Kpis), Prev: adKpisPB(d.Prev),
		Deltas:             &analyticsv1.AdsDeltas{Spend: delta(dl.Spend), Results: delta(dl.Results), Cpr: delta(dl.CPR), Impressions: delta(dl.Impressions), Reach: delta(dl.Reach), LinkClicks: delta(dl.LinkClicks), Ctr: delta(dl.CTR), Cpc: delta(dl.CPC), Cpm: delta(dl.CPM), Frequency: delta(dl.Frequency)},
		DominantResultType: d.DominantResultType, SpendSpark: floats(d.SpendSpark), ResultsSpark: floats(d.ResultsSpark), Daily: adsPointsPB(d.Daily),
		Demographics: &analyticsv1.AdsDemographics{HasData: d.Demographics.HasData, ByAge: demoRowsPB(d.Demographics.ByAge), ByGender: demoRowsPB(d.Demographics.ByGender), Matrix: demoRowsPB(d.Demographics.Matrix)},
	}
	if dl.ROAS != nil {
		out.Deltas.Roas = delta(*dl.ROAS)
	}
	if d.Account != nil {
		out.Account = adAccountPB(*d.Account)
	}
	for _, c := range d.CampaignOptions {
		out.CampaignOptions = append(out.CampaignOptions, &analyticsv1.CampaignOption{Id: c.ID, Name: c.Name})
	}
	for _, c := range d.CampaignShare {
		out.CampaignShare = append(out.CampaignShare, &analyticsv1.AdsCampaignShare{Id: c.ID, Name: c.Name, Spend: c.Spend, Results: i64(c.Results), ResultType: c.ResultType})
	}
	for _, c := range d.Breakdown {
		cn := &analyticsv1.AdsCampaignNode{Id: c.ID, Name: c.Name, Status: c.Status, Objective: c.Objective, Kpis: nodeKpisPB(c.Kpis)}
		for _, st := range c.AdSets {
			sn := &analyticsv1.AdsAdSetNode{Id: st.ID, Name: st.Name, Status: st.Status, Kpis: nodeKpisPB(st.Kpis)}
			for _, ad := range st.Ads {
				sn.Ads = append(sn.Ads, &analyticsv1.AdsAdNode{Id: ad.ID, Name: ad.Name, Status: ad.Status, ThumbnailUrl: ad.ThumbnailURL, Kpis: nodeKpisPB(ad.Kpis)})
			}
			cn.AdSets = append(cn.AdSets, sn)
		}
		out.Breakdown = append(out.Breakdown, cn)
	}
	for _, t := range d.TopAds {
		out.TopAds = append(out.TopAds, &analyticsv1.AdsTopAd{Id: t.ID, Name: t.Name, CampaignName: t.CampaignName, Status: t.Status, ThumbnailUrl: t.ThumbnailURL, Spend: t.Spend, Results: i64(t.Results), ResultType: t.ResultType, Cpr: t.CPR, Ctr: t.CTR})
	}
	return out, nil
}

func (s *adsServer) ImportAdsCsv(ctx context.Context, req *analyticsv1.ImportAdsCsvRequest) (*analyticsv1.ImportSummary, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	sum, err := s.svc.Ads.ImportCsv(ctx, p.UserID, req.ClientId, req.Filename, req.ContentType, req.Content)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.ImportSummary{Campaigns: int32(sum.Campaigns), AdSets: int32(sum.AdSets), Ads: int32(sum.Ads), Days: int32(sum.Days), Demographics: int32(sum.Demographics), Warnings: sum.Warnings}, nil
}

func (s *adsServer) SyncAds(ctx context.Context, req *analyticsv1.SyncAdsRequest) (*analyticsv1.SyncAdsResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	at, err := s.svc.Ads.Sync(ctx, p.UserID, req.ClientId)
	if err != nil {
		return nil, err
	}
	return &analyticsv1.SyncAdsResponse{LastSyncedAt: ts(at)}, nil
}

// ─── ReportDataService (internal) ────────────────────────────

type reportServer struct {
	analyticsv1.UnimplementedReportDataServiceServer
	svc Services
}

func (s *reportServer) GetReportData(ctx context.Context, req *analyticsv1.GetReportDataRequest) (*analyticsv1.ReportData, error) {
	cur, err := dates.FromProto(req.Current)
	if err != nil {
		return nil, apperr.Field("current", err.Error())
	}
	prev := dates.Previous(cur)
	if req.Previous != nil {
		if prev, err = dates.FromProto(req.Previous); err != nil {
			return nil, apperr.Field("previous", err.Error())
		}
	}
	modules := make([]string, 0, len(req.Modules))
	for _, m := range req.Modules {
		if m != commonv1.ModuleKey_MODULE_KEY_UNSPECIFIED {
			modules = append(modules, m.String())
		}
	}
	d, err := s.svc.Reports.Data(ctx, req.ClientId, app.Window{Current: cur, Previous: prev}, modules, int(req.TopN))
	if err != nil {
		return nil, err
	}
	out := &analyticsv1.ReportData{Current: cur.Proto(), Previous: prev.Proto()}
	if d.Social != nil {
		rs := &analyticsv1.ReportSocial{Overview: overviewSocialPB(d.Social.Overview)}
		for _, f := range d.Social.DailyFollowers {
			rs.DailyFollowers = append(rs.DailyFollowers, &analyticsv1.DailyFollowersPoint{Date: f.Date, Followers: i64(f.Followers)})
		}
		for _, pw := range d.Social.TopPosts {
			rs.TopPosts = append(rs.TopPosts, &analyticsv1.TopPostRow{Id: pw.ID, Platform: platformPB(pw.Platform), Username: pw.Username, Caption: pw.Caption, MediaType: pw.MediaType, ProductType: pw.ProductType, Permalink: pw.Permalink, PublishedAt: ts(pw.PublishedAt), Likes: i64(pw.Likes), Comments: i64(pw.Comments), Shares: i64(pw.Shares), Saves: i64(pw.Saves), Engagements: i64(domain.PostEngagements(pw.SocialPost)), Reach: i64(pw.Reach)})
		}
		out.Social = rs
	}
	if d.Seo != nil {
		rs := &analyticsv1.ReportSeo{Overview: overviewSeoPB(d.Seo.Overview)}
		for _, q := range d.Seo.TopQueries {
			rs.TopQueries = append(rs.TopQueries, &analyticsv1.TopQueryRow{Query: q.Query, Clicks: i64(q.Clicks), Impressions: i64(q.Impressions), Ctr: q.CTR, Position: q.Position})
		}
		out.Seo = rs
	}
	if d.Ads != nil {
		ra := &analyticsv1.ReportAds{Overview: overviewAdsPB(d.Ads.Overview)}
		for _, c := range d.Ads.Campaigns {
			ra.Campaigns = append(ra.Campaigns, &analyticsv1.CampaignRow{Id: c.ID, Name: c.Name, Objective: c.Objective, Status: c.Status, Spend: c.Spend, Impressions: i64(c.Impressions), Results: i64(c.Results), ResultType: c.ResultType, Cpr: c.CPR, IsConversion: c.IsConversion})
		}
		out.Ads = ra
	}
	return out, nil
}
