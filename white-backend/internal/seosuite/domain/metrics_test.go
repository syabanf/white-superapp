package domain

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

func p(v int32) *int32 { return &v }

func close(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestVisibilityAndPositions(t *testing.T) {
	if !close(Visibility([]*int32{p(1), p(1)}), ExpectedCtr(1)) {
		t.Fatal("visibility all #1")
	}
	if !close(Visibility([]*int32{p(1), nil}), ExpectedCtr(1)/2) {
		t.Fatal("unranked counts 0")
	}
	if Visibility(nil) != 0 || Visibility([]*int32{p(150)}) != 0 {
		t.Fatal("empty or >100")
	}
	if avg := AvgPosition([]*int32{p(2), p(4), nil}); avg == nil || *avg != 3 {
		t.Fatalf("avg %v", avg)
	}
	if AvgPosition([]*int32{nil}) != nil {
		t.Fatal("avg of unranked must be nil")
	}
	positions := []*int32{p(1), p(3), p(4), p(11), nil}
	if CountTop(positions, 3) != 2 || CountTop(positions, 10) != 3 {
		t.Fatal("countTop")
	}
}

func TestRankBuckets(t *testing.T) {
	cases := map[int32]string{1: "1-3", 3: "1-3", 4: "4-10", 10: "4-10", 11: "11-20", 21: "21-50", 51: "51-100", 100: "51-100", 101: "unranked"}
	for pos, want := range cases {
		if got := RankBucket(p(pos)); got != want {
			t.Fatalf("bucket(%d)=%s want %s", pos, got, want)
		}
	}
	if RankBucket(nil) != "unranked" {
		t.Fatal("nil bucket")
	}
	d := RankDistribution([]*int32{p(1), p(2), p(5), p(15), p(30), nil})
	var buckets []string
	var counts []int32
	for _, b := range d {
		buckets = append(buckets, b.Bucket)
		counts = append(counts, b.Count)
	}
	if !reflect.DeepEqual(buckets, RankBuckets) || !reflect.DeepEqual(counts, []int32{2, 1, 1, 1, 0, 1}) {
		t.Fatalf("distribution %v %v", buckets, counts)
	}
	if PositionToScore(p(1)) != 100 || PositionToScore(p(100)) != 1 || PositionToScore(nil) != 0 {
		t.Fatal("positionToScore")
	}
}

func TestClassifyRankChange(t *testing.T) {
	if c := ClassifyRankChange(p(8), p(3)); c.Kind != ChangeUp || *c.Delta != 5 {
		t.Fatalf("%+v", c)
	}
	if c := ClassifyRankChange(p(3), p(8)); c.Kind != ChangeDown || *c.Delta != -5 {
		t.Fatalf("%+v", c)
	}
	if c := ClassifyRankChange(p(5), p(5)); c.Kind != ChangeFlat || *c.Delta != 0 {
		t.Fatalf("%+v", c)
	}
	if ClassifyRankChange(nil, p(5)).Kind != ChangeNew || ClassifyRankChange(p(5), nil).Kind != ChangeLost || ClassifyRankChange(nil, nil).Kind != ChangeNone {
		t.Fatal("new/lost/none")
	}
	if ClassifyRankChange(p(120), p(5)).Kind != ChangeNew {
		t.Fatal(">100 counts as unranked")
	}
	alerts := []struct {
		prev, cur *int32
		want      bool
	}{
		{p(3), p(9), true}, {p(3), p(8), false}, {p(9), p(11), true}, {p(12), p(15), false}, {p(40), nil, true}, {nil, p(40), false}, {p(8), p(2), false},
	}
	for _, a := range alerts {
		if IsRankDropAlert(a.prev, a.cur) != a.want {
			t.Fatalf("alert(%v,%v) want %v", a.prev, a.cur, a.want)
		}
	}
}

func TestShareOfVoice(t *testing.T) {
	lists := [][]TopResult{
		{{Domain: "a.id", Position: 1}, {Domain: "b.id", Position: 2}},
		{{Domain: "a.id", Position: 1}, {Domain: "c.id", Position: 10}},
	}
	sov := ShareOfVoice(lists, 1, "Lainnya")
	if len(sov) != 2 || sov[0].Domain != "a.id" || !close(sov[0].Weight, 2*ExpectedCtr(1)) || sov[1].Domain != "Lainnya" {
		t.Fatalf("%+v", sov)
	}
	var total float64
	for _, r := range sov {
		total += r.Share
	}
	if !close(total, 100) {
		t.Fatalf("shares sum %v", total)
	}
	if len(ShareOfVoice(nil, 8, "Lainnya")) != 0 {
		t.Fatal("empty")
	}
}

func TestDifficultyBand(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	cases := map[float64]DifficultyBand{0: BandEasy, 29: BandEasy, 30: BandMedium, 59: BandMedium, 60: BandHard, 80: BandVeryHard}
	for kd, want := range cases {
		if got := DifficultyBandOf(f(kd)); got == nil || *got != want {
			t.Fatalf("band(%v)=%v", kd, got)
		}
	}
	if DifficultyBandOf(nil) != nil {
		t.Fatal("nil band")
	}
}

func TestCacheFreshness(t *testing.T) {
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	at := func(s string) time.Time { v, _ := time.Parse(time.RFC3339, s); return v }
	if !IsCacheFresh(at("2026-08-30T00:00:00Z"), now, 7) {
		t.Fatal("3 days old is fresh")
	}
	if IsCacheFresh(at("2026-08-26T09:00:00Z"), now, 7) {
		t.Fatal("7+ days old is stale")
	}
	if !IsCacheFresh(at("2026-09-01T00:00:00Z"), now, 7) {
		t.Fatal("yesterday is fresh")
	}
	if IsCacheFresh(at("2026-09-03T00:00:00Z"), now, 7) {
		t.Fatal("future forces refetch")
	}
}

func TestKeywordGap(t *testing.T) {
	own := []DomainKeyword{{Keyword: "kopi susu", Position: 3, Volume: 1000}, {Keyword: "kopi tubruk", Position: 12, Volume: 400}, {Keyword: "cold brew", Position: 5, Volume: 800}}
	comp := []DomainKeyword{{Keyword: "Kopi Susu", Position: 8, Volume: 1000}, {Keyword: "kopi tubruk", Position: 2, Volume: 400}, {Keyword: "kopi arabika", Position: 4, Volume: 2000}, {Keyword: "kopi robusta", Position: 9, Volume: 300}}
	gap := KeywordGapOf(own, comp)
	names := func(rows []GapRow) []string {
		out := []string{}
		for _, r := range rows {
			out = append(out, r.Keyword)
		}
		return out
	}
	if !reflect.DeepEqual(names(gap.Missing), []string{"kopi arabika", "kopi robusta"}) {
		t.Fatalf("missing %v", names(gap.Missing))
	}
	if !reflect.DeepEqual(names(gap.Weak), []string{"kopi tubruk"}) || !reflect.DeepEqual(names(gap.Strong), []string{"Kopi Susu"}) {
		t.Fatalf("weak %v strong %v", names(gap.Weak), names(gap.Strong))
	}
	if len(gap.Shared) != 2 || gap.Missing[0].OwnPosition != nil {
		t.Fatal("shared/ownPosition")
	}
}

func TestBacklinks(t *testing.T) {
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	if !IsToxic(p(60)) || IsToxic(p(59)) || IsToxic(nil) {
		t.Fatal("toxic")
	}
	if !IsNewBacklink(dates.UTC(2026, 8, 20), false, now, 30) || IsNewBacklink(dates.UTC(2026, 1, 20), false, now, 30) || IsNewBacklink(dates.UTC(2026, 8, 25), true, now, 30) {
		t.Fatal("new view")
	}
	if DofollowShare(75, 25) != 75 || DofollowShare(0, 0) != 0 {
		t.Fatal("dofollow share")
	}
}

func TestNormalizeDomain(t *testing.T) {
	cases := map[string]string{
		"https://www.KopiKenangan.com/menu?x=1": "kopikenangan.com",
		"  janjijiwa.com  ":                     "janjijiwa.com",
		"http://user@shop.tokopedia.com:8080/":  "shop.tokopedia.com",
		"sc-domain:adiharjo.co.id":              "adiharjo.co.id",
		"":                                      "",
		"kopi":                                  "",
		"not a domain":                          "",
		"-bad-.com":                             "",
	}
	for in, want := range cases {
		if got := NormalizeDomain(in); got != want {
			t.Fatalf("normalizeDomain(%q)=%q want %q", in, got, want)
		}
	}
	if SiteHost("https://kopinusantara.id/") != "kopinusantara.id" || SiteHost("sc-domain:adiharjo.co.id") != "adiharjo.co.id" {
		t.Fatal("siteHost")
	}
}

func TestTextHelpers(t *testing.T) {
	if got := ParseKeywordLines(" Kopi Susu \nkopi susu\n\nkedai kopi, cold  brew", 100); !reflect.DeepEqual(got, []string{"kopi susu", "kedai kopi", "cold brew"}) {
		t.Fatalf("%v", got)
	}
	if got := ParseKeywordLines("a\nb\nc", 2); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("%v", got)
	}
	if got := ParseTags("Brand, produk ,, brand", 10); !reflect.DeepEqual(got, []string{"brand", "produk"}) {
		t.Fatalf("%v", got)
	}
	if got := TopCounts([]string{"a", "b", "a", ""}, 2); !reflect.DeepEqual(got, []LabelCount{{"a", 2}, {"b", 1}}) {
		t.Fatalf("%v", got)
	}
	if got := TopCounts([]string{""}, 1); !reflect.DeepEqual(got, []LabelCount{{"(kosong)", 1}}) {
		t.Fatalf("%v", got)
	}
	got := PickSeriesDomains("me.id", map[string]int32{"me.id": 10, "x.id": 50, "y.id": 70, "z.id": 60, "w.id": 5}, 3)
	if !reflect.DeepEqual(got, []string{"me.id", "y.id", "z.id", "x.id"}) {
		t.Fatalf("%v", got)
	}
	if NormalizeMode("questions") != ModeQuestions || NormalizeMode("bogus") != ModeIdeas {
		t.Fatal("normalizeMode")
	}
}

func TestRankAggregate(t *testing.T) {
	d1, d2, d3 := dates.UTC(2026, 9, 1), dates.UTC(2026, 9, 2), dates.UTC(2026, 9, 3)
	rows := []RankSnapshot{
		{KeywordID: "a", Date: d1, Position: p(8)},
		{KeywordID: "a", Date: d3, Position: p(3), TopResults: []TopResult{{Domain: "me.id", Position: 3}}},
		{KeywordID: "b", Date: d1, Position: p(5)},
		{KeywordID: "b", Date: d2, Position: nil},
	}
	by := IndexSnapshots(rows)
	latest := LatestSnapshotDate(rows, d3)
	if latest == nil || !latest.Equal(d3) {
		t.Fatalf("latest %v", latest)
	}
	k := KpisOn(by, latest)
	if k.Top3 != 1 || k.Top10 != 1 || k.AvgPosition == nil || *k.AvgPosition != 3 {
		t.Fatalf("%+v", k)
	}
	kr := KeywordRankOf(by["a"], d1, d3, d1)
	if *kr.Position != 3 || *kr.StartPosition != 8 || *kr.Best != 3 || kr.Change.Kind != ChangeUp || *kr.Change.Delta != 5 || !reflect.DeepEqual(kr.Spark, []int32{93, 98}) {
		t.Fatalf("%+v", kr)
	}
	if kb := KeywordRankOf(by["b"], d1, d3, d1); kb.Change.Kind != ChangeLost || kb.Position != nil {
		t.Fatalf("%+v", kb)
	}
	if dist := DistributionOn(by, latest); dist[0].Count != 1 || dist[5].Count != 1 {
		t.Fatalf("%+v", dist)
	}
	series := DailyRankSeries(by, dates.Range{From: d1, To: d3}, dates.Range{From: dates.UTC(2026, 8, 29), To: dates.UTC(2026, 8, 31)})
	if len(series) != 3 || series[0].Visibility == nil || series[1].VisibilityPrev != nil {
		t.Fatalf("%+v", series)
	}
	if sov := ShareOfVoiceOn(by, latest, "Lainnya"); len(sov) != 1 || sov[0].Domain != "me.id" {
		t.Fatalf("%+v", sov)
	}
}
