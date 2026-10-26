// Package grpcapi exposes the analytics use cases over gRPC. Handlers only
// translate between proto and Go values; rules live in app.
package grpcapi

import (
	"time"

	analyticsv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/analytics/v1"
	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/app"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ts(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }

func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func i64(n int) int64 { return int64(n) }

func i64p(v *float64) *int64 {
	if v == nil {
		return nil
	}
	n := int64(*v)
	return &n
}

func i64pi(v *int) *int64 {
	if v == nil {
		return nil
	}
	n := int64(*v)
	return &n
}

func i32p(v *int) *int32 {
	if v == nil {
		return nil
	}
	n := int32(*v)
	return &n
}

func delta(d metrics.Delta) *commonv1.Delta { return d.Proto() }

func floats(xs []float64) []float64 {
	if xs == nil {
		return []float64{}
	}
	return xs
}

func platformPB(p domain.Platform) commonv1.SocialPlatform {
	switch p {
	case domain.Instagram:
		return commonv1.SocialPlatform_INSTAGRAM
	case domain.Facebook:
		return commonv1.SocialPlatform_FACEBOOK
	case domain.TikTok:
		return commonv1.SocialPlatform_TIKTOK
	}
	return commonv1.SocialPlatform_SOCIAL_PLATFORM_UNSPECIFIED
}

func platformFrom(p commonv1.SocialPlatform) domain.Platform {
	switch p {
	case commonv1.SocialPlatform_INSTAGRAM:
		return domain.Instagram
	case commonv1.SocialPlatform_FACEBOOK:
		return domain.Facebook
	case commonv1.SocialPlatform_TIKTOK:
		return domain.TikTok
	}
	return ""
}

func cadencePB(c domain.AuditCadence) analyticsv1.AuditCadence {
	switch c {
	case domain.CadenceNone:
		return analyticsv1.AuditCadence_NONE
	case domain.CadenceWeekly:
		return analyticsv1.AuditCadence_WEEKLY
	case domain.CadenceDaily:
		return analyticsv1.AuditCadence_DAILY
	}
	return analyticsv1.AuditCadence_AUDIT_CADENCE_UNSPECIFIED
}

func cadenceFrom(c analyticsv1.AuditCadence) domain.AuditCadence {
	switch c {
	case analyticsv1.AuditCadence_NONE:
		return domain.CadenceNone
	case analyticsv1.AuditCadence_WEEKLY:
		return domain.CadenceWeekly
	case analyticsv1.AuditCadence_DAILY:
		return domain.CadenceDaily
	}
	return ""
}

func jobStatusPB(s domain.JobStatus) commonv1.JobStatus {
	switch s {
	case domain.JobPending:
		return commonv1.JobStatus_PENDING
	case domain.JobRunning:
		return commonv1.JobStatus_RUNNING
	case domain.JobSuccess:
		return commonv1.JobStatus_SUCCESS
	case domain.JobFailed:
		return commonv1.JobStatus_FAILED
	}
	return commonv1.JobStatus_JOB_STATUS_UNSPECIFIED
}

func statePB(s domain.DataState) commonv1.DataState {
	switch s {
	case domain.StateHealthy:
		return commonv1.DataState_HEALTHY
	case domain.StateStale:
		return commonv1.DataState_STALE
	case domain.StateFailed:
		return commonv1.DataState_FAILED_SYNC
	case domain.StateDemo:
		return commonv1.DataState_DEMO
	case domain.StateEmpty:
		return commonv1.DataState_EMPTY
	case domain.StateNotConnected:
		return commonv1.DataState_NOT_CONNECTED
	}
	return commonv1.DataState_DATA_STATE_UNSPECIFIED
}

func qualityPB(q domain.DataQuality) *analyticsv1.DataQuality {
	return &analyticsv1.DataQuality{State: statePB(q.State), LastSyncedAt: tsPtr(q.LastSyncedAt), DataFrom: tsPtr(q.DataFrom), DataTo: tsPtr(q.DataTo)}
}

func socialAccountPB(a app.AccountWithFollowers) *analyticsv1.SocialAccount {
	return &analyticsv1.SocialAccount{Id: a.ID, ClientId: a.ClientID, Platform: platformPB(a.Platform), ExternalId: a.ExternalID, Username: a.Username, DisplayName: a.DisplayName, AvatarUrl: a.AvatarURL, Biography: a.Biography, IsCompetitor: a.IsCompetitor, ConnectionId: a.ConnectionID, Followers: int32(a.Followers), CreatedAt: ts(a.CreatedAt)}
}

func seoPropertyPB(p domain.SeoProperty) *analyticsv1.SeoProperty {
	return &analyticsv1.SeoProperty{Id: p.ID, ClientId: p.ClientID, SiteUrl: p.SiteURL, Ga4PropertyId: p.Ga4PropertyID, ConnectionId: p.ConnectionID, LocationCode: int32(p.LocationCode), LanguageCode: p.LanguageCode, AuditCadence: cadencePB(p.AuditCadence), CreatedAt: ts(p.CreatedAt)}
}

func adAccountPB(a domain.AdAccount) *analyticsv1.AdAccount {
	return &analyticsv1.AdAccount{Id: a.ID, ClientId: a.ClientID, ExternalId: a.ExternalID, Name: a.Name, Currency: a.Currency, ConnectionId: a.ConnectionID, LastSyncedAt: tsPtr(a.LastSyncedAt), CreatedAt: ts(a.CreatedAt)}
}

func socialPostPB(p domain.SocialPost) *analyticsv1.SocialPost {
	return &analyticsv1.SocialPost{Id: p.ID, SocialAccountId: p.AccountID, ExternalId: p.ExternalID, Caption: p.Caption, MediaType: p.MediaType, ProductType: p.ProductType, Permalink: p.Permalink, PublishedAt: ts(p.PublishedAt), Likes: i64(p.Likes), Comments: i64(p.Comments), Shares: i64(p.Shares), Saves: i64(p.Saves), Views: i64(p.Views), Reach: i64(p.Reach), Impressions: i64(p.Impressions)}
}

func adKpisPB(k domain.AdKpis) *analyticsv1.AdKpis {
	return &analyticsv1.AdKpis{Spend: k.Spend, Impressions: i64(k.Impressions), Reach: i64(k.Reach), Clicks: i64(k.Clicks), LinkClicks: i64(k.LinkClicks), Results: i64(k.Results), Cpr: k.CPR, Ctr: k.CTR, Cpc: k.CPC, Cpm: k.CPM, Frequency: k.Frequency, Roas: k.ROAS}
}

func overviewSocialPB(o app.OverviewSocial) *analyticsv1.OverviewSocial {
	out := &analyticsv1.OverviewSocial{HasSource: o.HasSource, HasData: o.HasData, IsDemo: o.IsDemo, Quality: qualityPB(o.Quality), Followers: i64(o.Followers), FollowersDelta: delta(o.FollowersDelta), FollowersSpark: floats(o.FollowersSpark), Posts: i64(o.Posts), PostsDelta: delta(o.PostsDelta), EngagementRate: o.EngagementRate, EngagementRateDelta: delta(o.EngagementRateDelta), Reach: i64(o.Reach), ReachDelta: delta(o.ReachDelta)}
	for _, a := range o.Accounts {
		out.Accounts = append(out.Accounts, &analyticsv1.OverviewSocialAccount{Platform: platformPB(a.Platform), Username: a.Username, Followers: i64(a.Followers)})
	}
	return out
}

func overviewSeoPB(o app.OverviewSeo) *analyticsv1.OverviewSeo {
	out := &analyticsv1.OverviewSeo{HasSource: o.HasSource, HasData: o.HasData, IsDemo: o.IsDemo, Quality: qualityPB(o.Quality), Clicks: i64(o.Clicks), ClicksDelta: delta(o.ClicksDelta), ClicksSpark: floats(o.ClicksSpark), Impressions: i64(o.Impressions), ImpressionsDelta: delta(o.ImpressionsDelta), Ctr: o.CTR, CtrDelta: delta(o.CTRDelta), Position: o.Position, PositionDelta: delta(o.PositionDelta), Health: i32p(o.Health)}
	for _, d := range o.Daily {
		out.Daily = append(out.Daily, &analyticsv1.SeoDailyPoint{Date: d.Date, Clicks: i64p(d.Clicks), Impressions: i64p(d.Impressions)})
	}
	return out
}

func adsPointsPB(points []app.AdsPoint) []*analyticsv1.AdsDailyPoint {
	out := make([]*analyticsv1.AdsDailyPoint, 0, len(points))
	for _, p := range points {
		out = append(out, &analyticsv1.AdsDailyPoint{Date: p.Date, Spend: p.Spend, Results: i64(p.Results), SpendPrev: p.SpendPrev, ResultsPrev: i64p(p.ResultsPrev)})
	}
	return out
}

func overviewAdsPB(o app.OverviewAds) *analyticsv1.OverviewAds {
	return &analyticsv1.OverviewAds{HasSource: o.HasSource, HasData: o.HasData, IsDemo: o.IsDemo, Quality: qualityPB(o.Quality), Kpis: adKpisPB(o.Kpis), Prev: adKpisPB(o.Prev), SpendDelta: delta(o.SpendDelta), ResultsDelta: delta(o.ResultsDelta), CprDelta: delta(o.CPRDelta), SpendSpark: floats(o.SpendSpark), Daily: adsPointsPB(o.Daily)}
}

func syncJobPB(j domain.SyncJob) *analyticsv1.SyncJob {
	return &analyticsv1.SyncJob{Id: j.ID, ClientId: j.ClientID, Kind: j.Kind, Status: jobStatusPB(j.Status), StartedAt: ts(j.StartedAt), FinishedAt: tsPtr(j.FinishedAt), Message: j.Message, Error: j.Error}
}

func postRowPB(p app.PostRow) *analyticsv1.SocialPostRow {
	return &analyticsv1.SocialPostRow{Id: p.ID, Caption: p.Caption, CaptionShort: p.CaptionShort, ThumbnailUrl: p.ThumbnailURL, Permalink: p.Permalink, Type: p.Type, PublishedAt: ts(p.PublishedAt), Likes: i64(p.Likes), Comments: i64(p.Comments), Shares: i64(p.Shares), Saves: i64(p.Saves), Views: i64(p.Views), Reach: i64(p.Reach), Engagements: i64(p.Engagements), Er: p.ER, Username: p.Username}
}

func searchAggQuery(q domain.QueryAgg) *analyticsv1.QueryAgg {
	out := &analyticsv1.QueryAgg{Key: q.Key, Clicks: i64(q.Clicks), Impressions: i64(q.Impressions), Ctr: q.CTR, Position: q.Position, PrevClicks: i64pi(q.PrevClicks), PrevPosition: q.PrevPosition, PrevImpressions: i64pi(q.PrevImpressions)}
	return out
}

func dimAggPB(rows []domain.DimAgg) []*analyticsv1.DimAgg {
	out := make([]*analyticsv1.DimAgg, 0, len(rows))
	for _, r := range rows {
		out = append(out, &analyticsv1.DimAgg{Key: r.Key, Clicks: i64(r.Clicks), Impressions: i64(r.Impressions), Ctr: r.CTR, Position: r.Position})
	}
	return out
}

func opportunitiesPB(xs []domain.Opportunity) []*analyticsv1.Opportunity {
	out := make([]*analyticsv1.Opportunity, 0, len(xs))
	for _, o := range xs {
		out = append(out, &analyticsv1.Opportunity{Kind: o.Kind, Key: o.Key, Clicks: i64(o.Clicks), Impressions: i64(o.Impressions), Position: o.Position, Ctr: o.CTR, ExpectedCtr: o.ExpectedCTR, PrevClicks: i64pi(o.PrevClicks), PotentialClicks: i64(o.PotentialClicks)})
	}
	return out
}

func auditRowPB(a *domain.SeoAudit) *analyticsv1.AuditRow {
	if a == nil {
		return nil
	}
	s := analyticsv1.AuditStrategy_MOBILE
	if a.Strategy == domain.Desktop {
		s = analyticsv1.AuditStrategy_DESKTOP
	}
	return &analyticsv1.AuditRow{Id: a.ID, Url: a.URL, Strategy: s, RunAt: ts(a.RunAt), Performance: int32(a.Performance), Seo: int32(a.Seo), Accessibility: int32(a.Accessibility), BestPractices: int32(a.BestPractices), LcpMs: a.LcpMs, InpMs: a.InpMs, Cls: a.Cls, FcpMs: a.FcpMs, TtfbMs: a.TtfbMs, TbtMs: a.TbtMs, SpeedIndexMs: a.SpeedIndexMs}
}

func severityPB(s domain.IssueSeverity) analyticsv1.IssueSeverity {
	switch s {
	case domain.SeverityError:
		return analyticsv1.IssueSeverity_ERROR
	case domain.SeverityWarning:
		return analyticsv1.IssueSeverity_WARNING
	case domain.SeverityNotice:
		return analyticsv1.IssueSeverity_NOTICE
	}
	return analyticsv1.IssueSeverity_ISSUE_SEVERITY_UNSPECIFIED
}

func nodeKpisPB(k domain.NodeKpis) *analyticsv1.AdsNodeKpis {
	return &analyticsv1.AdsNodeKpis{Spend: k.Spend, Results: i64(k.Results), ResultType: k.ResultType, Cpr: k.CPR, Impressions: i64(k.Impressions), Reach: i64(k.Reach), LinkClicks: i64(k.LinkClicks), Ctr: k.CTR, Cpc: k.CPC, Frequency: k.Frequency}
}

func demoRowsPB(rows []domain.DemographicRow) []*analyticsv1.AdsDemographicRow {
	out := make([]*analyticsv1.AdsDemographicRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, &analyticsv1.AdsDemographicRow{Age: r.Age, Gender: r.Gender, Spend: r.Spend, Results: i64(r.Results)})
	}
	return out
}
