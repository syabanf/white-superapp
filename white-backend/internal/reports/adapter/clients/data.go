package clients

import (
	"context"
	"strings"
	"time"

	analyticsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/analytics/v1"
	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	publishingv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/publishing/v1"
	seosuitev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/seosuite/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
	"google.golang.org/grpc"
)

func delta(d *commonv1.Delta) metrics.Delta {
	out := metrics.Delta{Current: d.GetCurrent(), Previous: d.GetPrevious(), Abs: d.GetAbs(), Pct: d.Pct, Direction: metrics.Flat}
	switch d.GetDirection() {
	case commonv1.Delta_UP:
		out.Direction = metrics.Up
	case commonv1.Delta_DOWN:
		out.Direction = metrics.Down
	}
	return out
}

func day(s string) time.Time {
	t, _ := dates.Parse(s)
	return t
}

func platform(p commonv1.SocialPlatform) string {
	if p == commonv1.SocialPlatform_SOCIAL_PLATFORM_UNSPECIFIED {
		return ""
	}
	return p.String()
}

func moduleKeys(modules []string) []commonv1.ModuleKey {
	out := make([]commonv1.ModuleKey, 0, len(modules))
	for _, m := range modules {
		if v, ok := commonv1.ModuleKey_value[m]; ok && v != 0 {
			out = append(out, commonv1.ModuleKey(v))
		}
	}
	return out
}

// Analytics reads ReportDataService and the client's SEO properties.
type Analytics struct {
	reports analyticsv1.ReportDataServiceClient
	sources analyticsv1.SourceServiceClient
}

func NewAnalytics(conn grpc.ClientConnInterface) *Analytics {
	return &Analytics{reports: analyticsv1.NewReportDataServiceClient(conn), sources: analyticsv1.NewSourceServiceClient(conn)}
}

func (c *Analytics) ReportData(ctx context.Context, clientID string, current, previous dates.Range, modules []string, topN int) (*domain.ReportData, error) {
	res, err := c.reports.GetReportData(ctx, &analyticsv1.GetReportDataRequest{
		ClientId: clientID, Current: current.Proto(), Previous: previous.Proto(), Modules: moduleKeys(modules), TopN: int32(topN),
	})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := &domain.ReportData{}
	if s := res.Social; s != nil {
		ov := s.GetOverview()
		data := &domain.SocialData{Overview: domain.SocialOverview{
			HasData: ov.GetHasData(), Followers: ov.GetFollowers(), FollowersDelta: delta(ov.GetFollowersDelta()),
			Posts: ov.GetPosts(), PostsDelta: delta(ov.GetPostsDelta()), EngagementRate: ov.GetEngagementRate(),
			EngagementRateDelta: delta(ov.GetEngagementRateDelta()), Reach: ov.GetReach(), ReachDelta: delta(ov.GetReachDelta()),
		}}
		for _, p := range s.GetDailyFollowers() {
			data.DailyFollowers = append(data.DailyFollowers, domain.FollowersPoint{Date: day(p.GetDate()), Followers: p.GetFollowers()})
		}
		for _, p := range s.GetTopPosts() {
			data.TopPosts = append(data.TopPosts, domain.TopPost{
				ID: p.GetId(), Platform: platform(p.GetPlatform()), Username: p.GetUsername(), Caption: p.GetCaption(),
				MediaType: p.GetMediaType(), ProductType: p.GetProductType(), PublishedAt: p.GetPublishedAt().AsTime(),
				Likes: p.GetLikes(), Comments: p.GetComments(), Engagements: p.GetEngagements(),
			})
		}
		out.Social = data
	}
	if s := res.Seo; s != nil {
		ov := s.GetOverview()
		data := &domain.SeoData{
			HasData: ov.GetHasData(), Clicks: ov.GetClicks(), ClicksDelta: delta(ov.GetClicksDelta()),
			Impressions: ov.GetImpressions(), ImpressionsDelta: delta(ov.GetImpressionsDelta()),
			Ctr: ov.GetCtr(), CtrDelta: delta(ov.GetCtrDelta()), Position: ov.GetPosition(), PositionDelta: delta(ov.GetPositionDelta()),
			Health: ov.Health,
		}
		for _, p := range ov.GetDaily() {
			data.Daily = append(data.Daily, domain.SeoDailyPoint{Date: day(p.GetDate()), Clicks: p.Clicks, Impressions: p.Impressions})
		}
		for _, q := range s.GetTopQueries() {
			data.TopQueries = append(data.TopQueries, domain.TopQuery{Query: q.GetQuery(), Clicks: q.GetClicks(), Impressions: q.GetImpressions(), Ctr: q.GetCtr(), Position: q.GetPosition()})
		}
		out.Seo = data
	}
	if a := res.Ads; a != nil {
		ov := a.GetOverview()
		k := ov.GetKpis()
		data := &domain.AdsData{
			HasData:    ov.GetHasData(),
			Kpis:       domain.AdKpis{Spend: k.GetSpend(), Results: k.GetResults(), Cpr: k.GetCpr(), Ctr: k.GetCtr(), Cpm: k.GetCpm(), Roas: k.Roas},
			SpendDelta: delta(ov.GetSpendDelta()), ResultsDelta: delta(ov.GetResultsDelta()), CprDelta: delta(ov.GetCprDelta()),
		}
		for _, p := range ov.GetDaily() {
			data.Daily = append(data.Daily, domain.AdsDailyPoint{Date: day(p.GetDate()), Spend: p.GetSpend(), Results: p.GetResults()})
		}
		for _, c := range a.GetCampaigns() {
			data.Campaigns = append(data.Campaigns, domain.Campaign{
				ID: c.GetId(), Name: c.GetName(), Objective: c.GetObjective(), Spend: c.GetSpend(), Results: c.GetResults(),
				ResultType: c.ResultType, Cpr: c.GetCpr(), IsConversion: c.GetIsConversion(),
			})
		}
		out.Ads = data
	}
	return out, nil
}

// SuitePropertyID is getSuiteProperty: the first SEO property by created_at.
func (c *Analytics) SuitePropertyID(ctx context.Context, clientID string) (string, error) {
	res, err := c.sources.ListSeoProperties(ctx, &analyticsv1.ListSeoPropertiesRequest{ClientId: clientID})
	if err != nil {
		return "", grpcx.FromStatus(err)
	}
	var first *analyticsv1.SeoProperty
	for _, p := range res.GetProperties() {
		if first == nil || p.GetCreatedAt().AsTime().Before(first.GetCreatedAt().AsTime()) {
			first = p
		}
	}
	return first.GetId(), nil
}

// Publishing reads the publishing summary of a window.
type Publishing struct {
	client publishingv1.ReportDataServiceClient
}

func NewPublishing(conn grpc.ClientConnInterface) *Publishing {
	return &Publishing{client: publishingv1.NewReportDataServiceClient(conn)}
}

func (c *Publishing) Summary(ctx context.Context, clientID string, rg dates.Range) (*domain.PublishingSummary, error) {
	res, err := c.client.GetPublishingSummary(ctx, &publishingv1.GetPublishingSummaryRequest{ClientId: clientID, Range: rg.Proto()})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	out := &domain.PublishingSummary{Posts: int(res.GetPosts()), Failed: int(res.GetFailed()), Mode: res.GetMode()}
	for _, p := range res.GetPerPlatform() {
		out.PerPlatform = append(out.PerPlatform, domain.PlatformCount{Platform: platform(p.GetPlatform()), Count: int(p.GetCount())})
	}
	for _, t := range res.GetTop() {
		out.Top = append(out.Top, domain.PublishedTarget{
			Title: t.GetTitle(), Body: t.GetBody(), Platform: platform(t.GetPlatform()),
			PublishedAt: t.GetPublishedAt().AsTime(), Engagements: t.Engagements,
		})
	}
	return out, nil
}

// SeoSuite reads the rank tracker and backlink section inputs.
type SeoSuite struct {
	client seosuitev1.ReportDataServiceClient
}

func NewSeoSuite(conn grpc.ClientConnInterface) *SeoSuite {
	return &SeoSuite{client: seosuitev1.NewReportDataServiceClient(conn)}
}

func intPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}

func (c *SeoSuite) Rank(ctx context.Context, propertyID string, current, previous dates.Range) (*domain.RankData, error) {
	res, err := c.client.GetRankSection(ctx, &seosuitev1.GetRankSectionRequest{PropertyId: propertyID, Current: current.Proto(), Previous: previous.Proto()})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	k, d := res.GetKpis(), res.GetDeltas()
	out := &domain.RankData{
		HasKeywords: res.GetHasKeywords(), LatestDate: res.LatestDate,
		Kpis: domain.RankKpis{Visibility: k.GetVisibility(), AvgPosition: k.AvgPosition, Top3: int(k.GetTop3()), Top10: int(k.GetTop10())},
		Deltas: domain.RankDeltas{
			Visibility: delta(d.GetVisibility()), AvgPosition: delta(d.GetAvgPosition()), Top3: delta(d.GetTop3()), Top10: delta(d.GetTop10()),
		},
	}
	for _, r := range res.GetRows() {
		kind := "none"
		if ck := r.GetChange().GetKind(); ck != seosuitev1.RankChangeKind_RANK_CHANGE_KIND_UNSPECIFIED {
			kind = strings.ToLower(ck.String())
		}
		out.Rows = append(out.Rows, domain.RankRow{
			Keyword: r.GetKeyword(), Position: intPtr(r.Position), Volume: intPtr(r.Volume),
			Change: domain.RankChange{Kind: kind, Delta: intPtr(r.GetChange().Delta)},
		})
	}
	return out, nil
}

func (c *SeoSuite) Backlinks(ctx context.Context, propertyID string, rg dates.Range) (*domain.BacklinkData, error) {
	res, err := c.client.GetBacklinkSection(ctx, &seosuitev1.GetBacklinkSectionRequest{PropertyId: propertyID, Range: rg.Proto()})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	return &domain.BacklinkData{
		HasSnapshot: res.GetHasSnapshot(), Backlinks: int(res.GetBacklinks()), ReferringDomains: int(res.GetReferringDomains()),
		DomainRank: intPtr(res.DomainRank), DofollowShare: res.GetDofollowShare(),
		NewBacklinks: int(res.GetNewBacklinks()), LostBacklinks: int(res.GetLostBacklinks()),
	}, nil
}
