package grpcapi

import (
	"context"
	"strings"
	"time"

	contentv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/content/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
)

func (s *reportServer) GetDashboard(ctx context.Context, req *contentv1.DashboardRequest) (*contentv1.ContentDashboard, error) {
	uid, err := caller(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Dashboard.Get(ctx, uid, req.ClientId, app.DashboardQuery{
		From: strings.TrimSpace(req.From), To: strings.TrimSpace(req.To), Compare: req.Compare,
		Filter: domain.DashboardFilter{
			Platform: domain.Platform(strings.TrimSpace(req.Platform)), Format: domain.Format(strings.TrimSpace(req.Format)),
			Pillar: strings.TrimSpace(req.Pillar), PlanID: strings.TrimSpace(req.PlanId), AssigneeID: strings.TrimSpace(req.AssigneeId),
		},
	})
	if err != nil {
		return nil, err
	}
	return &contentv1.ContentDashboard{
		From: day(res.From), To: day(res.To), PrevFrom: day(res.PrevFrom), PrevTo: day(res.PrevTo), Compare: res.Compare,
		Current: dashWindowPB(res.Current), Previous: dashWindowPB(res.Previous), Pillars: res.Pillars,
	}, nil
}

func day(t time.Time) string { return t.Format(time.DateOnly) }

func countsPB(cs []domain.DashCount) []*contentv1.DashboardCount {
	out := make([]*contentv1.DashboardCount, 0, len(cs))
	for _, c := range cs {
		out = append(out, &contentv1.DashboardCount{Key: c.Key, Count: c.Count})
	}
	return out
}

func dashBreakdownsPB(bs []domain.DashBreakdown) []*contentv1.DashboardBreakdown {
	out := make([]*contentv1.DashboardBreakdown, 0, len(bs))
	for _, b := range bs {
		out = append(out, &contentv1.DashboardBreakdown{Key: b.Key, Planned: b.Planned, Published: b.Published, Engagements: b.Engagements, Reach: b.Reach, Measured: b.Measured})
	}
	return out
}

func dashWindowPB(d domain.Dashboard) *contentv1.DashboardWindow {
	k := d.Kpis
	w := &contentv1.DashboardWindow{
		Granularity: d.Granularity,
		Kpis: &contentv1.DashboardKpis{
			Planned: k.Planned, Published: k.Published, Failed: k.Failed, InReview: k.InReview, Overdue: k.Overdue,
			DeadlinesMet: k.DeadlinesMet, DeadlinesTotal: k.DeadlinesTotal, OnTimeRate: k.OnTimeRate, RevisionsTotal: k.RevisionsTotal,
			RevisionAvg: k.RevisionAvg, Engagements: k.Engagements, Reach: k.Reach, EngagementRate: k.EngagementRate, MeasuredPosts: k.MeasuredPosts,
			CycleDaysAvg: k.CycleDaysAvg, DesignTasks: k.DesignTasks, DesignApproved: k.DesignApproved, DesignTurnaroundDays: k.DesignTurnaroundDays,
			FirstPassRate: k.FirstPassRate, AssetUploads: k.AssetUploads,
		},
		Funnel: countsPB(d.Funnel), Statuses: countsPB(d.Statuses), OverdueByStage: countsPB(d.OverdueByStage),
		RevisionRounds: countsPB(d.RevisionRounds), DesignVersions: countsPB(d.DesignVersions), AssetUploadsByKind: countsPB(d.AssetUploadsKind),
		ByPlatform: dashBreakdownsPB(d.ByPlatform), ByFormat: dashBreakdownsPB(d.ByFormat), ByPillar: dashBreakdownsPB(d.ByPillar),
	}
	for _, b := range d.Buckets {
		w.Buckets = append(w.Buckets, &contentv1.DashboardBucket{
			Start: day(b.Start), Planned: b.Planned, Published: b.Published, Engagements: b.Engagements, Reach: b.Reach,
			DeadlinesMet: b.DeadlinesMet, DeadlinesTotal: b.DeadlinesTotal, OnTimeRate: b.OnTimeRate, RevisionAvg: b.RevisionAvg,
			StageCopy: b.StageCopy, StageDesign: b.StageDesign, StageReview: b.StageReview, StageReady: b.StageReady,
			DesignTurnaroundDays: b.DesignTurnaroundDays, AssetUploads: b.AssetUploads,
		})
	}
	for _, c := range d.StageCycles {
		w.StageCycles = append(w.StageCycles, &contentv1.DashboardStageCycle{Stage: c.Stage, AvgDays: c.AvgDays, MedianDays: c.MedianDays, Count: c.Count})
	}
	for _, p := range d.People {
		w.People = append(w.People, &contentv1.DashboardPerson{
			UserId: p.UserID, Role: p.Role, Assigned: p.Assigned, Published: p.Published, Overdue: p.Overdue,
			OverdueCopy: p.OverdueCopy, OverdueDesign: p.OverdueDesign, OverdueReview: p.OverdueReview, RevisionAvg: p.RevisionAvg,
			OnTimeRate: p.OnTimeRate, DesignTasks: p.DesignTasks, DesignApproved: p.DesignApproved, TurnaroundDays: p.TurnaroundDays, FirstPassRate: p.FirstPassRate,
		})
	}
	for _, dd := range d.Days {
		w.Days = append(w.Days, &contentv1.DashboardDay{Date: day(dd.Date), Planned: dd.Planned, Published: dd.Published})
	}
	for _, c := range d.PostingHeat {
		w.PostingHeat = append(w.PostingHeat, &contentv1.DashboardHeatCell{Weekday: c.Weekday, Hour: c.Hour, Posts: c.Posts, Engagements: c.Engagements})
	}
	for _, p := range d.Posts {
		m := p.Metrics
		w.Posts = append(w.Posts, &contentv1.DashboardPost{
			ItemId: p.ItemID, Title: p.Title, Platform: platformPB[p.Platform], Format: formatPB[p.Format], Pillar: p.Pillar,
			PlannedDate: day(p.PlannedDate), PublishedAt: tsPtr(p.PublishedAt), Permalink: p.Permalink, ThumbnailFileId: p.ThumbnailFileID,
			HasMetrics: p.HasMetrics, Likes: m.Likes, Comments: m.Comments, Shares: m.Shares, Saves: m.Saves, Reach: m.Reach, Views: m.Views,
			Engagements: m.Engagements(),
		})
	}
	for _, a := range d.TopAssets {
		w.TopAssets = append(w.TopAssets, &contentv1.DashboardAsset{AssetId: a.AssetID, Name: a.Name, Kind: assetKindPB(a.Kind), Uploads: a.Uploads, CurrentVersion: a.CurrentVersion, LastUploadedAt: ts(a.LastUploadedAt)})
	}
	return w
}
