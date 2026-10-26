package domain

import (
	"reflect"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

// Ported from tests/reports-extras.test.ts.

var L = LabelsFor("id")

func ip(v int) *int { return &v }

func kw(keyword string, position, volume *int) RankRow {
	return RankRow{Keyword: keyword, Position: position, Volume: volume, Change: RankChange{Kind: "flat", Delta: ip(0)}}
}

func TestTopRankRows(t *testing.T) {
	rows := []RankRow{kw("c", nil, ip(9000)), kw("a", ip(4), ip(100)), kw("b", ip(4), ip(500)), kw("d", ip(1), ip(10))}
	var got []string
	for _, r := range TopRankRows(rows, 10) {
		got = append(got, r.Keyword)
	}
	if !reflect.DeepEqual(got, []string{"d", "b", "a", "c"}) {
		t.Fatalf("order = %v", got)
	}

	many := make([]RankRow, 14)
	for i := range many {
		many[i] = kw("k"+string(rune('a'+i)), ip(14-i), ip(10))
	}
	top := TopRankRows(many, 10)
	if len(top) != 10 || top[0].Keyword != "kn" || many[0].Keyword != "ka" {
		t.Fatalf("cap/mutation: len=%d first=%s input0=%s", len(top), top[0].Keyword, many[0].Keyword)
	}
}

func TestRankChangeText(t *testing.T) {
	cases := []struct {
		c    RankChange
		want string
	}{
		{RankChange{Kind: "up", Delta: ip(3)}, "Naik 3"},
		{RankChange{Kind: "down", Delta: ip(-2)}, "Turun 2"},
		{RankChange{Kind: "flat", Delta: ip(0)}, "Tetap"},
		{RankChange{Kind: "new"}, "Baru masuk"},
		{RankChange{Kind: "lost"}, "Keluar 100 besar"},
		{RankChange{Kind: "none"}, "-"},
	}
	for _, c := range cases {
		if got := RankChangeText(c.c, L); got != c.want {
			t.Errorf("%+v = %q, want %q", c.c, got, c.want)
		}
	}
}

func TestBuildRankSection(t *testing.T) {
	latest := "2026-09-01"
	avg := 8.0
	base := RankData{
		HasKeywords: true, LatestDate: &latest,
		Kpis: RankKpis{Visibility: 12, AvgPosition: &avg, Top3: 2, Top10: 5},
		Deltas: RankDeltas{
			Visibility: metrics.ComputeDelta(12, 10), AvgPosition: metrics.ComputeDelta(8, 10),
			Top3: metrics.ComputeDelta(2, 1), Top10: metrics.ComputeDelta(5, 5),
		},
		Rows: []RankRow{
			{Keyword: "kopi gayo", Position: ip(3), Volume: ip(1300), Change: RankChange{Kind: "up", Delta: ip(2)}},
			{Keyword: "kopi arabika", Volume: nil, Change: RankChange{Kind: "lost"}},
		},
	}
	noKeywords := base
	noKeywords.HasKeywords = false
	noSnapshot := base
	noSnapshot.LatestDate = nil
	if BuildRankSection(&noKeywords, true, L) != nil || BuildRankSection(&noSnapshot, true, L) != nil || BuildRankSection(nil, true, L) != nil {
		t.Fatal("section must be absent without keywords or snapshot")
	}

	on := BuildRankSection(&base, true, L)
	var abs []float64
	for _, k := range on.Kpis {
		abs = append(abs, k.Delta.Abs)
	}
	if !reflect.DeepEqual(abs, []float64{2, -2, 1, 0}) || !on.Kpis[1].LowerIsBetter {
		t.Fatalf("deltas %v lowerIsBetter %v", abs, on.Kpis[1].LowerIsBetter)
	}
	if on.Kpis[0].Value != "12,0%" || on.Kpis[1].Value != "8,0" {
		t.Fatalf("values %q %q", on.Kpis[0].Value, on.Kpis[1].Value)
	}
	want := [][]string{{"kopi gayo", "3", "Naik 2", "1,3 rb"}, {"kopi arabika", "Di luar 100", "Keluar 100 besar", "-"}}
	if !reflect.DeepEqual(on.Table.Rows, want) {
		t.Fatalf("rows %q", on.Table.Rows)
	}

	off := BuildRankSection(&base, false, L)
	for _, k := range off.Kpis {
		if k.Delta != nil {
			t.Fatal("compare off must drop deltas")
		}
	}
	base.Kpis.AvgPosition = nil
	if BuildRankSection(&base, true, L).Kpis[1].Value != "-" {
		t.Fatal("missing average position prints -")
	}
}

func TestBuildBacklinkSection(t *testing.T) {
	if BuildBacklinkSection(&BacklinkData{}, L) != nil {
		t.Fatal("absent without a snapshot")
	}
	s := BuildBacklinkSection(&BacklinkData{HasSnapshot: true, Backlinks: 1200, ReferringDomains: 300, DofollowShare: 75, NewBacklinks: 12, LostBacklinks: 3}, L)
	value := func(label string) string {
		for _, k := range s.Kpis {
			if k.Label == label {
				return k.Value
			}
		}
		return "?"
	}
	checks := [][2]string{
		{value(L.BacklinksTotal), "1,2 rb"},
		{value(L.BacklinksNew), "12"},
		{value(L.BacklinksLost), "3"},
		{value(L.DomainRank), "-"},
		{value(L.DofollowShare), "75,0%"},
	}
	for _, c := range checks {
		if c[0] != c[1] {
			t.Errorf("got %q, want %q", c[0], c[1])
		}
	}
	if s.Table != nil {
		t.Fatal("backlinks have no table")
	}
}

func TestBuildPublishingSection(t *testing.T) {
	if BuildPublishingSection(&PublishingSummary{Failed: 3}, L) != nil {
		t.Fatal("no section when nothing was published")
	}
	e55 := int64(55)
	recent := &PublishingSummary{
		Posts: 3, Failed: 2, Mode: "recent",
		PerPlatform: []PlatformCount{{"INSTAGRAM", 3}, {"FACEBOOK", 1}},
		Top:         []PublishedTarget{{Title: "", Body: "Promo\n\n  kemerdekaan", Platform: "INSTAGRAM", PublishedAt: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)}},
	}
	s := BuildPublishingSection(recent, L)
	var labels []string
	for _, k := range s.Kpis {
		labels = append(labels, k.Label+"="+k.Value)
	}
	if !reflect.DeepEqual(labels, []string{"Postingan terbit=3", "Terbit di Instagram=3", "Terbit di Facebook=1", "Gagal terbit=2"}) {
		t.Fatalf("kpis %v", labels)
	}
	if len(s.Table.Columns) != 3 || s.Table.Title != L.PublishedRecent || s.Table.Rows[0][1] != "Promo kemerdekaan" {
		t.Fatalf("recent table %+v", s.Table)
	}

	engaged := &PublishingSummary{Posts: 2, Mode: "engagement", Top: []PublishedTarget{{Title: "Post p2", Platform: "INSTAGRAM", PublishedAt: time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC), Engagements: &e55}}}
	s = BuildPublishingSection(engaged, L)
	if s.Table.Title != L.PublishedTop || !reflect.DeepEqual(s.Table.Rows[0], []string{"Instagram", "Post p2", "9 Agu", "55"}) {
		t.Fatalf("engagement table %+v", s.Table)
	}
}
