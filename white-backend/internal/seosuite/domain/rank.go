package domain

import (
	"sort"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

// Rank tracker aggregation over plain snapshot rows, ported from
// rank-aggregate.ts. Windows are inclusive UTC days.

type RankKpis struct {
	Visibility  float64
	AvgPosition *float64
	Top3        int32
	Top10       int32
}

type RankKpiDeltas struct {
	Visibility  metrics.Delta
	AvgPosition metrics.Delta
	Top3        metrics.Delta
	Top10       metrics.Delta
}

type RankDailyPoint struct {
	Date            time.Time
	Visibility      *float64
	VisibilityPrev  *float64
	AvgPosition     *float64
	AvgPositionPrev *float64
}

type KeywordRank struct {
	Position      *int32
	URL           *string
	SerpFeatures  []string
	StartPosition *int32
	Best          *int32
	Change        RankChange
	Spark         []int32
}

// SnapshotIndex is keyword id -> date key -> snapshot.
type SnapshotIndex map[string]map[string]RankSnapshot

func IndexSnapshots(rows []RankSnapshot) SnapshotIndex {
	by := SnapshotIndex{}
	for _, r := range rows {
		m := by[r.KeywordID]
		if m == nil {
			m = map[string]RankSnapshot{}
			by[r.KeywordID] = m
		}
		m[dates.Format(r.Date)] = r
	}
	return by
}

// LatestSnapshotDate is the most recent day on or before `date` with any snapshot.
func LatestSnapshotDate(rows []RankSnapshot, date time.Time) *time.Time {
	var latest *time.Time
	for _, r := range rows {
		d := dates.Day(r.Date)
		if d.After(date) {
			continue
		}
		if latest == nil || d.After(*latest) {
			dd := d
			latest = &dd
		}
	}
	return latest
}

// LatestOnOrBefore picks, per keyword, the newest snapshot on or before date.
func LatestOnOrBefore(by SnapshotIndex, date time.Time) map[string]RankSnapshot {
	key := dates.Format(date)
	out := map[string]RankSnapshot{}
	for kid, m := range by {
		var best *RankSnapshot
		bestKey := ""
		for d, r := range m {
			if d <= key && (best == nil || d > bestKey) {
				rr := r
				best, bestKey = &rr, d
			}
		}
		if best != nil {
			out[kid] = *best
		}
	}
	return out
}

func positionsOn(by SnapshotIndex, date time.Time) []*int32 {
	key := dates.Format(date)
	var positions []*int32
	for _, m := range by {
		if r, ok := m[key]; ok {
			positions = append(positions, r.Position)
		}
	}
	return positions
}

func KpisOn(by SnapshotIndex, date *time.Time) RankKpis {
	if date == nil {
		return RankKpis{}
	}
	positions := positionsOn(by, *date)
	return RankKpis{Visibility: Visibility(positions), AvgPosition: AvgPosition(positions), Top3: CountTop(positions, 3), Top10: CountTop(positions, 10)}
}

func KpiDeltas(cur, prev RankKpis) RankKpiDeltas {
	zero := func(v *float64) float64 {
		if v == nil {
			return 0
		}
		return *v
	}
	return RankKpiDeltas{
		Visibility:  metrics.ComputeDelta(cur.Visibility, prev.Visibility),
		AvgPosition: metrics.ComputeDelta(zero(cur.AvgPosition), zero(prev.AvgPosition)),
		Top3:        metrics.ComputeDelta(float64(cur.Top3), float64(prev.Top3)),
		Top10:       metrics.ComputeDelta(float64(cur.Top10), float64(prev.Top10)),
	}
}

type dailyPoint struct {
	date        time.Time
	visibility  *float64
	avgPosition *float64
}

func dailySeries(by SnapshotIndex, window dates.Range) []dailyPoint {
	var out []dailyPoint
	window.Each(func(day time.Time) {
		positions := positionsOn(by, day)
		p := dailyPoint{date: day}
		if len(positions) > 0 {
			v := Visibility(positions)
			p.visibility = &v
			p.avgPosition = AvgPosition(positions)
		}
		out = append(out, p)
	})
	return out
}

// DailyRankSeries aligns the current window with the previous one by index.
func DailyRankSeries(by SnapshotIndex, current, previous dates.Range) []RankDailyPoint {
	cur := dailySeries(by, current)
	prev := dailySeries(by, previous)
	out := make([]RankDailyPoint, 0, len(cur))
	for i, p := range cur {
		pt := RankDailyPoint{Date: p.date, Visibility: p.visibility, AvgPosition: p.avgPosition}
		if i < len(prev) {
			pt.VisibilityPrev = prev[i].visibility
			pt.AvgPositionPrev = prev[i].avgPosition
		}
		out = append(out, pt)
	}
	return out
}

// KeywordRankOf: current (<= to), start (first in [from, to]), best in the
// window, change between start and current, 30-day sparkline.
func KeywordRankOf(m map[string]RankSnapshot, from, to, sparkFrom time.Time) KeywordRank {
	out := KeywordRank{SerpFeatures: []string{}, Spark: []int32{}, Change: RankChange{Kind: ChangeNone}}
	if len(m) == 0 {
		return out
	}
	keys := make([]string, 0, len(m))
	for d := range m {
		keys = append(keys, d)
	}
	sort.Strings(keys)
	fromKey, toKey, sparkKey := dates.Format(from), dates.Format(to), dates.Format(sparkFrom)
	var curRow, startRow *RankSnapshot
	for i := len(keys) - 1; i >= 0; i-- {
		if keys[i] <= toKey {
			r := m[keys[i]]
			curRow = &r
			break
		}
	}
	for _, d := range keys {
		if d >= fromKey && d <= toKey {
			r := m[d]
			startRow = &r
			break
		}
	}
	for _, d := range keys {
		if d < fromKey || d > toKey {
			continue
		}
		if p := m[d].Position; p != nil && (out.Best == nil || *p < *out.Best) {
			v := *p
			out.Best = &v
		}
	}
	for _, d := range keys {
		if d >= sparkKey && d <= toKey {
			out.Spark = append(out.Spark, PositionToScore(m[d].Position))
		}
	}
	var startPos, curPos *int32
	if curRow != nil {
		out.Position = curRow.Position
		out.URL = curRow.URL
		if curRow.SerpFeatures != nil {
			out.SerpFeatures = curRow.SerpFeatures
		}
		curPos = curRow.Position
	}
	if startRow != nil {
		out.StartPosition = startRow.Position
		startPos = startRow.Position
	}
	out.Change = ClassifyRankChange(startPos, curPos)
	return out
}

func DistributionOn(by SnapshotIndex, date *time.Time) []BucketCount {
	var positions []*int32
	if date != nil {
		for _, r := range LatestOnOrBefore(by, *date) {
			positions = append(positions, r.Position)
		}
	}
	return RankDistribution(positions)
}

func ShareOfVoiceOn(by SnapshotIndex, date *time.Time, otherLabel string) []ShareOfVoiceRow {
	if date == nil {
		return []ShareOfVoiceRow{}
	}
	latest := LatestOnOrBefore(by, *date)
	ids := make([]string, 0, len(latest))
	for kid := range latest {
		ids = append(ids, kid)
	}
	sort.Strings(ids)
	lists := make([][]TopResult, 0, len(ids))
	for _, kid := range ids {
		lists = append(lists, latest[kid].TopResults)
	}
	return ShareOfVoice(lists, 8, otherLabel)
}
