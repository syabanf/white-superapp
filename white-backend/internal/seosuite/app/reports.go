package app

import (
	"context"
	"sort"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

// Reports serves the raw numbers behind buildRankSection and
// buildBacklinkSection (extras.ts). Formatting stays with the reports service.
type Reports struct {
	views *Views
}

func NewReports(views *Views) *Reports { return &Reports{views: views} }

type RankSectionRow struct {
	Keyword  string
	Position *int32
	Change   domain.RankChange
	Volume   *int32
}

type RankSection struct {
	HasKeywords bool
	LatestDate  *time.Time
	Kpis        domain.RankKpis
	Deltas      domain.RankKpiDeltas
	Rows        []RankSectionRow
}

// RankSection: KPIs plus the top 10 rows ordered by position (unranked
// last), then volume desc, then keyword.
func (r *Reports) RankSection(ctx context.Context, propertyID string, current, previous dates.Range) (*RankSection, error) {
	data, err := r.views.RankData(ctx, propertyID, current, previous)
	if err != nil {
		return nil, err
	}
	rows := make([]RankSectionRow, 0, len(data.Rows))
	for _, row := range data.Rows {
		rows = append(rows, RankSectionRow{Keyword: row.Keyword, Position: row.Position, Change: row.Change, Volume: row.Volume})
	}
	pos := func(p *int32) int32 {
		if p == nil {
			return 1<<31 - 1
		}
		return *p
	}
	vol := func(v *int32) int32 {
		if v == nil {
			return 0
		}
		return *v
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if pos(rows[i].Position) != pos(rows[j].Position) {
			return pos(rows[i].Position) < pos(rows[j].Position)
		}
		if vol(rows[i].Volume) != vol(rows[j].Volume) {
			return vol(rows[i].Volume) > vol(rows[j].Volume)
		}
		return rows[i].Keyword < rows[j].Keyword
	})
	if len(rows) > 10 {
		rows = rows[:10]
	}
	return &RankSection{HasKeywords: data.HasKeywords, LatestDate: data.LatestDate, Kpis: data.Kpis, Deltas: data.Deltas, Rows: rows}, nil
}

type BacklinkSection struct {
	Latest        *domain.BacklinkSnapshot
	DofollowShare float64
	New           int32
	Lost          int32
}

// BacklinkSection: the latest snapshot, dofollow share and the sums of the
// weekly new/lost figures inside the range.
func (r *Reports) BacklinkSection(ctx context.Context, propertyID string, rng dates.Range) (*BacklinkSection, error) {
	data, err := r.views.BacklinkData(ctx, propertyID, rng)
	if err != nil {
		return nil, err
	}
	out := &BacklinkSection{Latest: data.Latest, DofollowShare: data.DofollowShare}
	for _, w := range data.Weekly {
		out.New += w.New
		out.Lost += w.Lost
	}
	return out, nil
}
