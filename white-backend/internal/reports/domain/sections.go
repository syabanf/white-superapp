package domain

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

// Report extras (features/reports/extras.ts): rank tracker, backlinks and
// publishing performance, shared by the PDF and the builder preview.

type Column struct {
	Label string
	Width int
	Align string // "left" | "right"
}

type Table struct {
	Title   string
	Columns []Column
	Rows    [][]string
}

type SectionKpi struct {
	Label         string
	Value         string
	Delta         *metrics.Delta
	LowerIsBetter bool
}

type Section struct {
	Key   string // RANK | BACKLINKS | PUBLISHING
	Title string
	Kpis  []SectionKpi
	Table *Table
}

// Sections are the extra blocks per parent module.
type Sections struct {
	Social []Section
	Seo    []Section
}

// SectionInputs is what getReportSections fetched; nil means absent.
type SectionInputs struct {
	Publishing *PublishingSummary
	Rank       *RankData
	Backlinks  *BacklinkData
}

// BuildSections keeps only the blocks that have data.
func BuildSections(in SectionInputs, compare bool, L Labels) Sections {
	out := Sections{Social: []Section{}, Seo: []Section{}}
	if s := BuildPublishingSection(in.Publishing, L); s != nil {
		out.Social = append(out.Social, *s)
	}
	if s := BuildRankSection(in.Rank, compare, L); s != nil {
		out.Seo = append(out.Seo, *s)
	}
	if s := BuildBacklinkSection(in.Backlinks, L); s != nil {
		out.Seo = append(out.Seo, *s)
	}
	return out
}

// TopRankRows orders by position (unranked last), then volume desc, then
// keyword, and keeps the first n without touching the input.
func TopRankRows(rows []RankRow, n int) []RankRow {
	pos := func(p *int) int {
		if p == nil {
			return math.MaxInt
		}
		return *p
	}
	vol := func(v *int) int {
		if v == nil {
			return 0
		}
		return *v
	}
	out := slices.Clone(rows)
	slices.SortStableFunc(out, func(a, b RankRow) int {
		return cmp.Or(
			cmp.Compare(pos(a.Position), pos(b.Position)),
			cmp.Compare(vol(b.Volume), vol(a.Volume)),
			strings.Compare(a.Keyword, b.Keyword),
		)
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// RankChangeText names the direction instead of printing a signed number.
func RankChangeText(c RankChange, L Labels) string {
	abs := func() string {
		if c.Delta == nil {
			return FormatNumber(0, 0)
		}
		return FormatNumber(math.Abs(float64(*c.Delta)), 0)
	}
	switch c.Kind {
	case "up":
		return L.RankUp + " " + abs()
	case "down":
		return L.RankDown + " " + abs()
	case "flat":
		return L.RankFlat
	case "new":
		return L.RankNew
	case "lost":
		return L.RankLost
	}
	return "-"
}

func BuildRankSection(r *RankData, compare bool, L Labels) *Section {
	if r == nil || !r.HasKeywords || r.LatestDate == nil {
		return nil
	}
	d := func(x metrics.Delta) *metrics.Delta {
		if !compare {
			return nil
		}
		return &x
	}
	avg := "-"
	if r.Kpis.AvgPosition != nil {
		avg = FormatNumber(*r.Kpis.AvgPosition, 1)
	}
	rows := [][]string{}
	for _, row := range TopRankRows(r.Rows, 10) {
		position := L.RankUnranked
		if row.Position != nil {
			position = FormatNumber(float64(*row.Position), 0)
		}
		volume := "-"
		if row.Volume != nil {
			volume = FormatCompact(float64(*row.Volume))
		}
		rows = append(rows, []string{Truncate(row.Keyword, 60), position, RankChangeText(row.Change, L), volume})
	}
	return &Section{
		Key:   "RANK",
		Title: L.RankTitle,
		Kpis: []SectionKpi{
			{Label: L.RankVisibility, Value: FormatPercent(r.Kpis.Visibility, 1), Delta: d(r.Deltas.Visibility)},
			{Label: L.RankAvgPosition, Value: avg, Delta: d(r.Deltas.AvgPosition), LowerIsBetter: true},
			{Label: L.RankTop3, Value: FormatNumber(float64(r.Kpis.Top3), 0), Delta: d(r.Deltas.Top3)},
			{Label: L.RankTop10, Value: FormatNumber(float64(r.Kpis.Top10), 0), Delta: d(r.Deltas.Top10)},
		},
		Table: &Table{
			Title: L.RankTable,
			Columns: []Column{
				{Label: L.ColKeyword, Width: 46, Align: "left"},
				{Label: L.ColPosition, Width: 14, Align: "right"},
				{Label: L.ColChange, Width: 22, Align: "right"},
				{Label: L.ColVolume, Width: 18, Align: "right"},
			},
			Rows: rows,
		},
	}
}

func BuildBacklinkSection(b *BacklinkData, L Labels) *Section {
	if b == nil || !b.HasSnapshot {
		return nil
	}
	rank := "-"
	if b.DomainRank != nil {
		rank = FormatNumber(float64(*b.DomainRank), 0)
	}
	return &Section{
		Key:   "BACKLINKS",
		Title: L.BacklinksTitle,
		Kpis: []SectionKpi{
			{Label: L.BacklinksTotal, Value: FormatCompact(float64(b.Backlinks))},
			{Label: L.ReferringDomains, Value: FormatCompact(float64(b.ReferringDomains))},
			{Label: L.DomainRank, Value: rank},
			{Label: L.DofollowShare, Value: FormatPercent(b.DofollowShare, 1)},
			{Label: L.BacklinksNew, Value: FormatNumber(float64(b.NewBacklinks), 0)},
			{Label: L.BacklinksLost, Value: FormatNumber(float64(b.LostBacklinks), 0)},
		},
	}
}

func BuildPublishingSection(s *PublishingSummary, L Labels) *Section {
	if s == nil || s.Posts == 0 {
		return nil
	}
	byEngagement := s.Mode == "engagement"
	kpis := []SectionKpi{{Label: L.PublishedPosts, Value: FormatNumber(float64(s.Posts), 0)}}
	for _, p := range s.PerPlatform {
		kpis = append(kpis, SectionKpi{Label: L.PublishedOn + " " + PlatformName(p.Platform), Value: FormatNumber(float64(p.Count), 0)})
	}
	kpis = append(kpis, SectionKpi{Label: L.PublishFailed, Value: FormatNumber(float64(s.Failed), 0)})

	postWidth, postMax := 66, 90
	if byEngagement {
		postWidth, postMax = 52, 70
	}
	cols := []Column{
		{Label: "Platform", Width: 16, Align: "left"},
		{Label: L.ColPost, Width: postWidth, Align: "left"},
		{Label: L.ColPublishedAt, Width: 18, Align: "right"},
	}
	title := L.PublishedRecent
	if byEngagement {
		cols = append(cols, Column{Label: L.ColEngagements, Width: 14, Align: "right"})
		title = L.PublishedTop
	}
	rows := [][]string{}
	for _, t := range s.Top {
		text := t.Title
		if text == "" {
			text = t.Body
		}
		row := []string{PlatformName(t.Platform), Truncate(collapseSpace(text), postMax), FormatDateShort(t.PublishedAt)}
		if byEngagement {
			var e int64
			if t.Engagements != nil {
				e = *t.Engagements
			}
			row = append(row, FormatNumber(float64(e), 0))
		}
		rows = append(rows, row)
	}
	return &Section{Key: "PUBLISHING", Title: L.PublishingTitle, Kpis: kpis, Table: &Table{Title: title, Columns: cols, Rows: rows}}
}

// collapseSpace is replace(/\s+/g, " "): runs become one space, no trim.
func collapseSpace(s string) string {
	var b strings.Builder
	inSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !inSpace {
				b.WriteByte(' ')
			}
			inSpace = true
			continue
		}
		inSpace = false
		b.WriteRune(r)
	}
	return b.String()
}
