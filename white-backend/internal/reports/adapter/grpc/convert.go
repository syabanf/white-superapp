package grpcapi

import (
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	reportsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/reports/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/app"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func modules(in []commonv1.ModuleKey) []string {
	out := make([]string, 0, len(in))
	for _, m := range in {
		if m != commonv1.ModuleKey_MODULE_KEY_UNSPECIFIED {
			out = append(out, m.String())
		}
	}
	return out
}

func moduleKeys(in []string) []commonv1.ModuleKey {
	out := make([]commonv1.ModuleKey, 0, len(in))
	for _, m := range in {
		if v, ok := commonv1.ModuleKey_value[m]; ok && v != 0 {
			out = append(out, commonv1.ModuleKey(v))
		}
	}
	return out
}

func insightModule(m reportsv1.InsightModule) string {
	if m == reportsv1.InsightModule_INSIGHT_MODULE_UNSPECIFIED {
		return ""
	}
	return m.String()
}

func ts(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }

func reportPB(r app.ReportRow) *reportsv1.Report {
	return &reportsv1.Report{
		Id: r.ID, ClientId: r.ClientID, Title: r.Title, DateFrom: dates.Format(r.DateFrom), DateTo: dates.Format(r.DateTo),
		Modules: moduleKeys(r.Modules), Language: r.Language, Compare: r.Compare,
		HasAiSummary: r.AiSummary != nil && *r.AiSummary != "", CreatedByName: r.CreatedByName, CreatedAt: ts(r.CreatedAt),
	}
}

func insightPB(in *domain.Insight) *reportsv1.Insight {
	return &reportsv1.Insight{
		Id: in.ID, Module: reportsv1.InsightModule(reportsv1.InsightModule_value[in.Module]),
		DateFrom: dates.Format(in.DateFrom), DateTo: dates.Format(in.DateTo), Language: in.Language,
		Model: in.Model, Content: in.Content, CreatedAt: ts(in.CreatedAt),
	}
}

func tablePB(t *domain.Table) *reportsv1.ReportTable {
	if t == nil {
		return nil
	}
	out := &reportsv1.ReportTable{Title: t.Title}
	for _, c := range t.Columns {
		out.Columns = append(out.Columns, &reportsv1.TableColumn{Label: c.Label, Width: int32(c.Width), Align: c.Align})
	}
	for _, r := range t.Rows {
		out.Rows = append(out.Rows, &reportsv1.TableRow{Cells: r})
	}
	return out
}

func sectionList(in []domain.Section) []*reportsv1.ReportSection {
	out := make([]*reportsv1.ReportSection, 0, len(in))
	for _, s := range in {
		sec := &reportsv1.ReportSection{Key: s.Key, Title: s.Title, Table: tablePB(s.Table)}
		for _, k := range s.Kpis {
			kpi := &reportsv1.SectionKpi{Label: k.Label, Value: k.Value, LowerIsBetter: k.LowerIsBetter}
			if k.Delta != nil {
				kpi.Delta = k.Delta.Proto()
			}
			sec.Kpis = append(sec.Kpis, kpi)
		}
		out = append(out, sec)
	}
	return out
}

func sectionsPB(s domain.Sections) *reportsv1.ReportSections {
	return &reportsv1.ReportSections{Social: sectionList(s.Social), Seo: sectionList(s.Seo)}
}

func shareStatusPB(s app.ShareStatus) reportsv1.ShareStatus {
	switch s {
	case app.ShareOK:
		return reportsv1.ShareStatus_OK
	case app.ShareDisabled:
		return reportsv1.ShareStatus_DISABLED
	case app.ShareLocked:
		return reportsv1.ShareStatus_LOCKED
	}
	return reportsv1.ShareStatus_NOT_FOUND
}

func platformPB(p string) commonv1.SocialPlatform {
	return commonv1.SocialPlatform(commonv1.SocialPlatform_value[p])
}

func shareDashboardPB(d *app.ShareDashboard) *reportsv1.ShareDashboard {
	out := &reportsv1.ShareDashboard{Status: shareStatusPB(d.Status)}
	if d.Status == app.ShareNotFound {
		return out
	}
	c := d.Client
	out.Client = &reportsv1.ShareClient{Id: c.ID, Name: c.Name, Slug: c.Slug, Currency: c.Currency, Modules: moduleKeys(c.ShareModules)}
	if d.Status != app.ShareOK {
		return out
	}
	out.Range = d.Range.Proto()
	if s := d.View.Social; s != nil {
		ov := s.Overview
		social := &reportsv1.ShareSocial{
			Followers: ov.Followers, FollowersDelta: ov.FollowersDelta.Proto(),
			EngagementRate: ov.EngagementRate, EngagementRateDelta: ov.EngagementRateDelta.Proto(),
			Reach: ov.Reach, ReachDelta: ov.ReachDelta.Proto(), Posts: ov.Posts, PostsDelta: ov.PostsDelta.Proto(),
		}
		for _, p := range s.DailyFollowers {
			social.Daily = append(social.Daily, &reportsv1.ShareFollowersPoint{Date: dates.Format(p.Date), Followers: p.Followers})
		}
		for _, p := range s.TopPosts {
			social.TopPosts = append(social.TopPosts, &reportsv1.ShareTopPost{
				Id: p.ID, Platform: platformPB(p.Platform), Username: p.Username, Caption: p.Caption, Type: p.ContentType(),
				Likes: p.Likes, Comments: p.Comments, Engagements: p.Engagements, PublishedAt: ts(p.PublishedAt),
			})
		}
		out.Social = social
	}
	if s := d.View.Seo; s != nil {
		seo := &reportsv1.ShareSeo{
			Clicks: s.Clicks, ClicksDelta: s.ClicksDelta.Proto(), Impressions: s.Impressions, ImpressionsDelta: s.ImpressionsDelta.Proto(),
			Ctr: s.Ctr, CtrDelta: s.CtrDelta.Proto(), Position: s.Position, PositionDelta: s.PositionDelta.Proto(),
		}
		for _, p := range s.Daily {
			seo.Daily = append(seo.Daily, &reportsv1.ShareSeoDailyPoint{Date: dates.Format(p.Date), Clicks: p.Clicks, Impressions: p.Impressions})
		}
		for _, q := range s.TopQueries {
			seo.TopQueries = append(seo.TopQueries, &reportsv1.ShareTopQuery{Query: q.Query, Clicks: q.Clicks, Impressions: q.Impressions, Ctr: q.Ctr, Position: q.Position})
		}
		out.Seo = seo
	}
	if a := d.View.Ads; a != nil {
		ads := &reportsv1.ShareAds{
			Currency: c.Currency, Spend: a.Kpis.Spend, SpendDelta: a.SpendDelta.Proto(), Results: a.Kpis.Results,
			ResultsDelta: a.ResultsDelta.Proto(), Cpr: a.Kpis.Cpr, CprDelta: a.CprDelta.Proto(), Ctr: a.Kpis.Ctr,
		}
		for _, p := range a.Daily {
			ads.Daily = append(ads.Daily, &reportsv1.ShareAdsDailyPoint{Date: dates.Format(p.Date), Spend: p.Spend, Results: p.Results})
		}
		for _, cp := range a.Campaigns {
			ads.Campaigns = append(ads.Campaigns, &reportsv1.ShareCampaign{
				Id: cp.ID, Name: cp.Name, Objective: cp.Objective, Spend: cp.Spend, Results: cp.Results,
				ResultType: cp.ResultType, Cpr: cp.Cpr, IsConversion: cp.IsConversion,
			})
		}
		out.Ads = ads
	}
	return out
}
