// Package grpcapi exposes the seosuite use cases over gRPC. Handlers only
// translate between proto and app values; rules live in app and domain.
package grpcapi

import (
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	seosuitev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/seosuite/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/providers/seodata"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/app"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func dayPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := dates.Format(*t)
	return &s
}

func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func device(d domain.Device) seosuitev1.DeviceType {
	if d == domain.DeviceDesktop {
		return seosuitev1.DeviceType_DESKTOP
	}
	return seosuitev1.DeviceType_MOBILE
}

func deviceFrom(d seosuitev1.DeviceType) domain.Device {
	switch d {
	case seosuitev1.DeviceType_DESKTOP:
		return domain.DeviceDesktop
	case seosuitev1.DeviceType_MOBILE:
		return domain.DeviceMobile
	}
	return ""
}

func intent(i *domain.Intent) seosuitev1.KeywordIntent {
	if i == nil {
		return seosuitev1.KeywordIntent_KEYWORD_INTENT_UNSPECIFIED
	}
	if v, ok := seosuitev1.KeywordIntent_value[string(*i)]; ok {
		return seosuitev1.KeywordIntent(v)
	}
	return seosuitev1.KeywordIntent_KEYWORD_INTENT_UNSPECIFIED
}

func intentFrom(i seosuitev1.KeywordIntent) *domain.Intent {
	if i == seosuitev1.KeywordIntent_KEYWORD_INTENT_UNSPECIFIED {
		return nil
	}
	v := domain.Intent(i.String())
	return &v
}

func researchMode(m domain.ResearchMode) seosuitev1.ResearchMode {
	switch m {
	case domain.ModeQuestions:
		return seosuitev1.ResearchMode_QUESTIONS
	case domain.ModeRelated:
		return seosuitev1.ResearchMode_RELATED
	}
	return seosuitev1.ResearchMode_IDEAS
}

func researchModeFrom(m seosuitev1.ResearchMode) domain.ResearchMode {
	switch m {
	case seosuitev1.ResearchMode_QUESTIONS:
		return domain.ModeQuestions
	case seosuitev1.ResearchMode_RELATED:
		return domain.ModeRelated
	}
	return domain.ModeIdeas
}

func changeKind(k domain.RankChangeKind) seosuitev1.RankChangeKind {
	switch k {
	case domain.ChangeUp:
		return seosuitev1.RankChangeKind_UP
	case domain.ChangeDown:
		return seosuitev1.RankChangeKind_DOWN
	case domain.ChangeFlat:
		return seosuitev1.RankChangeKind_FLAT
	case domain.ChangeNew:
		return seosuitev1.RankChangeKind_NEW
	case domain.ChangeLost:
		return seosuitev1.RankChangeKind_LOST
	}
	return seosuitev1.RankChangeKind_NONE
}

func rankChange(c domain.RankChange) *seosuitev1.RankChange {
	return &seosuitev1.RankChange{Kind: changeKind(c.Kind), Delta: c.Delta}
}

func jobStatus(s domain.JobStatus) commonv1.JobStatus {
	if v, ok := commonv1.JobStatus_value[string(s)]; ok {
		return commonv1.JobStatus(v)
	}
	return commonv1.JobStatus_JOB_STATUS_UNSPECIFIED
}

func providerMode(m seodata.Mode) *seosuitev1.ProviderMode {
	return &seosuitev1.ProviderMode{Mode: string(m)}
}

func keywordIdea(i domain.KeywordIdea) *seosuitev1.KeywordIdea {
	in := i.Intent
	return &seosuitev1.KeywordIdea{Keyword: i.Keyword, Volume: i.Volume, Difficulty: i.Difficulty, Cpc: i.CPC, Competition: i.Competition, Intent: intent(&in), Trend: i.Trend, SerpFeatures: i.SerpFeatures}
}

func rankKpis(k domain.RankKpis) *seosuitev1.RankKpis {
	return &seosuitev1.RankKpis{Visibility: k.Visibility, AvgPosition: k.AvgPosition, Top3: k.Top3, Top10: k.Top10}
}

func rankDeltas(d domain.RankKpiDeltas) *seosuitev1.RankKpiDeltas {
	return &seosuitev1.RankKpiDeltas{Visibility: d.Visibility.Proto(), AvgPosition: d.AvgPosition.Proto(), Top3: d.Top3.Proto(), Top10: d.Top10.Proto()}
}

func rankData(r *app.RankDataResult) *seosuitev1.RankData {
	out := &seosuitev1.RankData{
		HasKeywords: r.HasKeywords,
		LatestDate:  dayPtr(r.LatestDate),
		Kpis:        rankKpis(r.Kpis),
		Deltas:      rankDeltas(r.Deltas),
		Strip:       &seosuitev1.RankStrip{Tracked: r.Strip.Tracked, Ranked: r.Strip.Ranked, Unranked: r.Strip.Unranked, Improved: r.Strip.Improved, Declined: r.Strip.Declined},
		Tags:        r.Tags,
		Range:       r.Range.Proto(),
		Provider:    providerMode(r.Mode),
	}
	for _, p := range r.Daily {
		out.Daily = append(out.Daily, &seosuitev1.RankDailyPoint{Date: dates.Format(p.Date), Visibility: p.Visibility, VisibilityPrev: p.VisibilityPrev, AvgPosition: p.AvgPosition, AvgPositionPrev: p.AvgPositionPrev})
	}
	for _, b := range r.Distribution {
		out.Distribution = append(out.Distribution, &seosuitev1.RankBucketCount{Bucket: b.Bucket, Count: b.Count})
	}
	for _, s := range r.Sov {
		out.Sov = append(out.Sov, &seosuitev1.ShareOfVoiceRow{Domain: s.Domain, Weight: s.Weight, Share: s.Share})
	}
	for _, row := range r.Rows {
		out.Rows = append(out.Rows, &seosuitev1.RankKeywordRow{
			Id: row.ID, Keyword: row.Keyword, Device: device(row.Device), Tags: row.Tags,
			Volume: row.Volume, Difficulty: row.Difficulty, Cpc: row.CPC, Intent: intent(row.Intent),
			Position: row.Position, Url: row.URL, SerpFeatures: row.SerpFeatures,
			StartPosition: row.StartPosition, Best: row.Best, Change: rankChange(row.Change), Spark: row.Spark,
			CreatedAt: timestamppb.New(row.CreatedAt),
		})
	}
	return out
}

func syncJob(j domain.SyncJob) *seosuitev1.SyncJob {
	return &seosuitev1.SyncJob{Id: j.ID, ClientId: j.ClientID, Kind: j.Kind, Status: jobStatus(j.Status), StartedAt: timestamppb.New(j.StartedAt), FinishedAt: tsPtr(j.FinishedAt), Message: j.Message, Error: j.Error}
}

func gapRows(rows []domain.GapRow) []*seosuitev1.GapRow {
	out := make([]*seosuitev1.GapRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, &seosuitev1.GapRow{Keyword: r.Keyword, Volume: r.Volume, OwnPosition: r.OwnPosition, CompetitorPosition: r.CompetitorPosition})
	}
	return out
}

func backlinkSnapshot(s *domain.BacklinkSnapshot) *seosuitev1.BacklinkSnapshot {
	if s == nil {
		return nil
	}
	return &seosuitev1.BacklinkSnapshot{Date: dates.Format(s.Date), Backlinks: s.Backlinks, ReferringDomains: s.ReferringDomains, Dofollow: s.Dofollow, Nofollow: s.Nofollow, NewBacklinks: s.NewBacklinks, LostBacklinks: s.LostBacklinks, DomainRank: s.DomainRank, ToxicShare: s.ToxicShare}
}

func labelCounts(rows []domain.LabelCount) []*seosuitev1.LabelCount {
	out := make([]*seosuitev1.LabelCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, &seosuitev1.LabelCount{Label: r.Label, Count: r.Count})
	}
	return out
}

func deltaProto(d metrics.Delta) *commonv1.Delta { return d.Proto() }

func backlinkData(r *app.BacklinkDataResult) *seosuitev1.BacklinkData {
	out := &seosuitev1.BacklinkData{
		Latest:        backlinkSnapshot(r.Latest),
		DofollowShare: r.DofollowShare,
		TopAnchors:    labelCounts(r.TopAnchors),
		TopDomains:    labelCounts(r.TopDomains),
		Today:         dates.Format(r.Today),
		Range:         r.Range.Proto(),
		Provider:      providerMode(r.Mode),
	}
	if r.Deltas != nil {
		out.Deltas = &seosuitev1.BacklinkDeltas{Backlinks: deltaProto(r.Deltas.Backlinks), ReferringDomains: deltaProto(r.Deltas.ReferringDomains), DomainRank: deltaProto(r.Deltas.DomainRank), DofollowShare: deltaProto(r.Deltas.DofollowShare)}
	}
	for _, p := range r.Daily {
		out.Daily = append(out.Daily, &seosuitev1.BacklinkDailyPoint{Date: dates.Format(p.Date), ReferringDomains: p.ReferringDomains})
	}
	for _, w := range r.Weekly {
		out.Weekly = append(out.Weekly, &seosuitev1.BacklinkWeeklyPoint{Date: w.Date, NewBacklinks: w.New, LostBacklinks: w.Lost})
	}
	for _, l := range r.Rows {
		out.Rows = append(out.Rows, &seosuitev1.Backlink{
			Id: l.ID, SourceUrl: l.SourceURL, SourceDomain: l.SourceDomain, TargetUrl: l.TargetURL, Anchor: l.Anchor,
			Dofollow: l.Dofollow, DomainRank: l.DomainRank, SpamScore: l.SpamScore,
			FirstSeen: dates.Format(l.FirstSeen), LastSeen: dates.Format(l.LastSeen), IsLost: l.IsLost,
			IsToxic: !l.IsLost && domain.IsToxic(l.SpamScore),
			IsNew:   domain.IsNewBacklink(l.FirstSeen, l.IsLost, r.Today, 30),
		})
	}
	return out
}

func competitorDomain(c domain.CompetitorDomain) *seosuitev1.CompetitorDomain {
	return &seosuitev1.CompetitorDomain{Id: c.ID, Domain: c.Domain, CreatedAt: timestamppb.New(c.CreatedAt)}
}

func competitorData(r *app.CompetitorData) *seosuitev1.CompetitorData {
	out := &seosuitev1.CompetitorData{OwnDomain: r.OwnDomain, Range: r.Range.Proto(), Provider: providerMode(r.Mode)}
	for _, c := range r.Competitors {
		out.Competitors = append(out.Competitors, competitorDomain(c))
	}
	for _, b := range r.Benchmark {
		row := &seosuitev1.BenchmarkRow{Domain: b.Domain, IsOwn: b.IsOwn}
		if s := b.Latest; s != nil {
			row.Date = dayPtr(&s.Date)
			row.OrganicKeywords = &s.OrganicKeywords
			row.OrganicTraffic = &s.OrganicTraffic
			row.Top3 = &s.Top3
			row.Top10 = &s.Top10
			row.Backlinks = s.Backlinks
			row.ReferringDomains = s.ReferringDomains
			row.DomainRank = s.DomainRank
		}
		out.Benchmark = append(out.Benchmark, row)
	}
	for _, s := range r.Series {
		out.Series = append(out.Series, &seosuitev1.TrafficSeries{Key: s.Key, Label: s.Label})
	}
	for _, p := range r.Traffic {
		out.Traffic = append(out.Traffic, &seosuitev1.TrafficPoint{Date: dates.Format(p.Date), Traffic: p.Values})
	}
	for _, s := range r.Suggestions {
		out.Suggestions = append(out.Suggestions, &seosuitev1.CompetitorSuggestion{Domain: s.Domain, Intersections: s.Intersections, OrganicTraffic: s.OrganicTraffic})
	}
	return out
}
