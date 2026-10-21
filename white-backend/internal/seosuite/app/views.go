package app

import (
	"context"
	"sort"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

// Views are the read models shared by the public RPCs and the report RPCs
// (queries.ts getRankData / getBacklinkData without the auth or mode bits).
type Views struct {
	store domain.Store
}

func NewViews(store domain.Store) *Views { return &Views{store: store} }

type RankKeywordRow struct {
	domain.TrackedKeyword
	domain.KeywordRank
}

type RankStrip struct {
	Tracked, Ranked, Unranked, Improved, Declined int32
}

type RankData struct {
	HasKeywords  bool
	LatestDate   *time.Time
	Kpis         domain.RankKpis
	Deltas       domain.RankKpiDeltas
	Strip        RankStrip
	Daily        []domain.RankDailyPoint
	Distribution []domain.BucketCount
	Sov          []domain.ShareOfVoiceRow
	Rows         []RankKeywordRow
	Tags         []string
}

// RankData: snapshots are taken during the day, so both windows run one day
// past their last complete day. KPIs sit on the latest snapshot day in each
// window; the per-row figures and the 30-day spark end on curEnd.
func (v *Views) RankData(ctx context.Context, propertyID string, current, previous dates.Range) (*RankData, error) {
	curEnd := dates.AddDays(current.To, 1)
	prevEnd := dates.AddDays(previous.To, 1)
	sparkFrom := dates.AddDays(curEnd, -29)
	loadFrom := previous.From
	if sparkFrom.Before(loadFrom) {
		loadFrom = sparkFrom
	}
	keywords, err := v.store.ListKeywords(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	var snaps []domain.RankSnapshot
	if len(keywords) > 0 {
		ids := make([]string, 0, len(keywords))
		for _, k := range keywords {
			ids = append(ids, k.ID)
		}
		if snaps, err = v.store.ListRankSnapshots(ctx, ids, loadFrom, curEnd); err != nil {
			return nil, err
		}
	}
	by := domain.IndexSnapshots(snaps)
	latest := domain.LatestSnapshotDate(snaps, curEnd)
	prevDate := domain.LatestSnapshotDate(snaps, prevEnd)
	kpis := domain.KpisOn(by, latest)

	out := &RankData{
		HasKeywords:  len(keywords) > 0,
		LatestDate:   latest,
		Kpis:         kpis,
		Deltas:       domain.KpiDeltas(kpis, domain.KpisOn(by, prevDate)),
		Daily:        domain.DailyRankSeries(by, dates.Range{From: current.From, To: curEnd}, dates.Range{From: previous.From, To: prevEnd}),
		Distribution: domain.DistributionOn(by, latest),
		Sov:          domain.ShareOfVoiceOn(by, latest, msgOthers),
		Rows:         make([]RankKeywordRow, 0, len(keywords)),
		Tags:         []string{},
	}
	tagSet := map[string]bool{}
	out.Strip.Tracked = int32(len(keywords))
	for _, k := range keywords {
		row := RankKeywordRow{TrackedKeyword: k, KeywordRank: domain.KeywordRankOf(by[k.ID], current.From, curEnd, sparkFrom)}
		out.Rows = append(out.Rows, row)
		if row.Position != nil {
			out.Strip.Ranked++
		} else {
			out.Strip.Unranked++
		}
		switch row.Change.Kind {
		case domain.ChangeUp, domain.ChangeNew:
			out.Strip.Improved++
		case domain.ChangeDown, domain.ChangeLost:
			out.Strip.Declined++
		}
		for _, t := range k.Tags {
			if !tagSet[t] {
				tagSet[t] = true
				out.Tags = append(out.Tags, t)
			}
		}
	}
	sort.Strings(out.Tags)
	return out, nil
}

type BacklinkDeltas struct {
	Backlinks        metrics.Delta
	ReferringDomains metrics.Delta
	DomainRank       metrics.Delta
	DofollowShare    metrics.Delta
}

type BacklinkDailyPoint struct {
	Date             time.Time
	ReferringDomains *int32
}

type BacklinkWeeklyPoint struct {
	// Date is the Monday week key, YYYY-MM-DD.
	Date string
	New  int32
	Lost int32
}

type BacklinkData struct {
	Latest        *domain.BacklinkSnapshot
	Deltas        *BacklinkDeltas
	DofollowShare float64
	Daily         []BacklinkDailyPoint
	Weekly        []BacklinkWeeklyPoint
	Rows          []domain.Backlink
	TopAnchors    []domain.LabelCount
	TopDomains    []domain.LabelCount
}

func zeroIfNil(p *int32) float64 {
	if p == nil {
		return 0
	}
	return float64(*p)
}

// BacklinkData: latest snapshot, the series inside [from, to + 1], deltas
// against the newest snapshot at least 30 days older, weekly new/lost sums
// by Monday, and the live-link anchor and domain tops.
func (v *Views) BacklinkData(ctx context.Context, propertyID string, rng dates.Range) (*BacklinkData, error) {
	latest, err := v.store.GetLatestBacklinkSnapshot(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	seriesTo := dates.AddDays(rng.To, 1)
	series, err := v.store.ListBacklinkSnapshots(ctx, propertyID, rng.From, seriesTo)
	if err != nil {
		return nil, err
	}
	links, err := v.store.ListBacklinks(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	out := &BacklinkData{Latest: latest, Rows: links, Daily: []BacklinkDailyPoint{}, Weekly: []BacklinkWeeklyPoint{}}
	byDay := map[string]int32{}
	weeks := map[string]*BacklinkWeeklyPoint{}
	for _, s := range series {
		byDay[dates.Format(s.Date)] = s.ReferringDomains
		wk := dates.MondayWeekKey(s.Date)
		w := weeks[wk]
		if w == nil {
			w = &BacklinkWeeklyPoint{Date: wk}
			weeks[wk] = w
		}
		w.New += s.NewBacklinks
		w.Lost += s.LostBacklinks
	}
	dates.Range{From: rng.From, To: seriesTo}.Each(func(day time.Time) {
		p := BacklinkDailyPoint{Date: day}
		if v, ok := byDay[dates.Format(day)]; ok {
			p.ReferringDomains = ptr(v)
		}
		out.Daily = append(out.Daily, p)
	})
	for _, w := range weeks {
		out.Weekly = append(out.Weekly, *w)
	}
	sort.Slice(out.Weekly, func(i, j int) bool { return out.Weekly[i].Date < out.Weekly[j].Date })

	if latest != nil {
		out.DofollowShare = domain.DofollowShare(latest.Dofollow, latest.Nofollow)
		compareAt, err := v.store.GetBacklinkSnapshotOnOrBefore(ctx, propertyID, dates.AddDays(latest.Date, -30))
		if err != nil {
			return nil, err
		}
		if compareAt != nil {
			out.Deltas = &BacklinkDeltas{
				Backlinks:        metrics.ComputeDelta(float64(latest.Backlinks), float64(compareAt.Backlinks)),
				ReferringDomains: metrics.ComputeDelta(float64(latest.ReferringDomains), float64(compareAt.ReferringDomains)),
				DomainRank:       metrics.ComputeDelta(zeroIfNil(latest.DomainRank), zeroIfNil(compareAt.DomainRank)),
				DofollowShare:    metrics.ComputeDelta(out.DofollowShare, domain.DofollowShare(compareAt.Dofollow, compareAt.Nofollow)),
			}
		}
	}
	anchors := []string{}
	domains := []string{}
	for _, l := range links {
		if l.IsLost {
			continue
		}
		anchors = append(anchors, l.Anchor)
		domains = append(domains, l.SourceDomain)
	}
	out.TopAnchors = domain.TopCounts(anchors, 8)
	out.TopDomains = domain.TopCounts(domains, 8)
	return out, nil
}
