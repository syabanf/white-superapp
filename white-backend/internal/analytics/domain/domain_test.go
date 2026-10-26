package domain

import (
	"math"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

func sp(s string) *string    { return &s }
func fp(f float64) *float64  { return &f }
func ip(n int) *int          { return &n }
func day(s string) time.Time { d, _ := dates.Parse(s); return d }

func near(t *testing.T, got, want float64, label string) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s: got %v want %v", label, got, want)
	}
}

func TestAggregateAdsFormulas(t *testing.T) {
	k := AggregateAds([]AdDailyInsight{
		{Spend: 100, Impressions: 1000, Reach: 500, Clicks: 30, LinkClicks: 20, Results: 4, PurchaseValue: fp(300)},
		{Spend: 50, Impressions: 1000, Reach: 500, Clicks: 10, LinkClicks: 5, Results: 1},
	})
	near(t, k.CPR, 30, "cpr")
	near(t, k.CTR, 25.0/2000*100, "ctr uses link clicks")
	near(t, k.CPC, 6, "cpc")
	near(t, k.CPM, 75, "cpm")
	near(t, k.Frequency, 2, "frequency")
	if k.ROAS == nil || *k.ROAS != 2 {
		t.Fatalf("roas %v", k.ROAS)
	}
	if AggregateAds([]AdDailyInsight{{Spend: 10}}).ROAS != nil {
		t.Fatal("roas must be nil without purchase values")
	}
}

func TestKpisWithConversionAndDaily(t *testing.T) {
	rows := []AdDailyInsight{
		{Date: day("2026-09-01"), Spend: 100, Results: 10, ResultType: sp("link_click")},
		{Date: day("2026-09-01"), Spend: 100, Results: 5000, ResultType: sp("reach")},
		{Date: day("2026-09-02"), Spend: 50, Results: 5, ResultType: nil},
	}
	k := KpisWithConversion(rows)
	if k.Spend != 250 || k.Results != 15 {
		t.Fatalf("spend %v results %d", k.Spend, k.Results)
	}
	near(t, k.CPR, 150.0/15, "cpr from conversion rows")
	ds := DailySource(rows)
	if ds[1].Results != 0 || ds[0].Results != 10 {
		t.Fatalf("non-conversion results not zeroed: %+v", ds)
	}
	if DailySource(rows[1:2])[0].Results != 5000 {
		t.Fatal("awareness-only rows keep their results")
	}
	if got := *DominantResultType(rows); got != "link_click" {
		t.Fatalf("dominant %s", got)
	}
}

func TestCampaignRowsAndTopAds(t *testing.T) {
	c1, c2, a1, a2 := "c1", "c2", "a1", "a2"
	rows := []AdDailyInsight{
		{CampaignID: &c1, AdID: &a1, Spend: 100, Results: 10, ResultType: sp("lead")},
		{CampaignID: &c2, AdID: &a2, Spend: 300, Results: 9000, ResultType: sp("reach")},
	}
	cr := CampaignRows(rows, []AdCampaign{{ID: "c1", Name: "Leads"}})
	if cr[0].ID != "c2" || cr[0].IsConversion || cr[0].CPR != 0 || cr[0].Name != "—" {
		t.Fatalf("%+v", cr[0])
	}
	if cr[1].CPR != 10 || !cr[1].IsConversion {
		t.Fatalf("%+v", cr[1])
	}
	top := BuildTopAds(nil, nil, nil, GroupInsights(rows))
	if len(top) != 1 || top[0].ID != "a1" || top[0].CampaignName != "–" {
		t.Fatalf("%+v", top)
	}
}

func TestAggregateSearchAndHealth(t *testing.T) {
	a := AggregateSearch([]SearchRow{{Clicks: 10, Impressions: 100, Position: 2}, {Clicks: 0, Impressions: 300, Position: 10}})
	near(t, a.CTR, 2.5, "ctr recomputed")
	near(t, a.Position, 8, "impression-weighted position")
	if HealthScore(nil, 0, 0) != nil {
		t.Fatal("no audit -> nil")
	}
	h := HealthScore(&AuditScores{Performance: 80, Seo: 90, Accessibility: 100, BestPractices: 100}, 15, 30)
	// base 24+36+15+15 = 90; penalty 20 + 10.
	if *h != 60 {
		t.Fatalf("health %d", *h)
	}
	if ExpectedCTR(0.4) != 28 || ExpectedCTR(15) != 1.5 || ExpectedCTR(40) != 1 {
		t.Fatal("expected ctr curve")
	}
}

func TestOpportunities(t *testing.T) {
	prev := 100
	rows := []QueryAgg{
		{DimAgg: DimAgg{Key: "strike", Clicks: 5, Impressions: 1000, Position: 7}},
		{DimAgg: DimAgg{Key: "lowctr", Clicks: 10, Impressions: 1000, Position: 1}},
		{DimAgg: DimAgg{Key: "down", Clicks: 50, Impressions: 500, Position: 20}, PrevClicks: &prev},
		{DimAgg: DimAgg{Key: "up", Clicks: 200, Impressions: 500, Position: 20}, PrevClicks: &prev},
		{DimAgg: DimAgg{Key: "few", Clicks: 0, Impressions: 50, Position: 5}},
	}
	r := DefaultOpportunityRule
	if s := StrikingDistance(rows, r); len(s) != 1 || s[0].Key != "strike" || s[0].PotentialClicks != 105 {
		t.Fatalf("striking %+v", s)
	}
	if l := LowCTR(rows, r); len(l) != 2 || l[0].Key != "lowctr" || l[0].PotentialClicks != 270 || l[1].PotentialClicks != 35 {
		t.Fatalf("low ctr %+v", l)
	}
	if d := Declining(rows, r); len(d) != 1 || d[0].PotentialClicks != 50 {
		t.Fatalf("declining %+v", d)
	}
	if u := Rising(rows, r); len(u) != 1 || u[0].PotentialClicks != 100 {
		t.Fatalf("rising %+v", u)
	}
}

func TestSocialRules(t *testing.T) {
	posts := []SocialPost{{Likes: 90, Comments: 10, PublishedAt: time.Date(2026, 9, 7, 3, 0, 0, 0, time.UTC)}, {Likes: 0, Comments: 0, Shares: 0, Saves: 50}}
	near(t, AvgEngagementRate(posts, 1000), 7.5, "avg er")
	if AvgEngagementRate(posts, 0) != 0 || AvgEngagementRate(nil, 10) != 0 {
		t.Fatal("er zero cases")
	}
	// Monday 2026-09-07 03:00 UTC is 10:00 WIB.
	hm := PostingHeatmap(posts[:1])
	if len(hm) != 168 || hm[10].Posts != 1 || hm[10].AvgEngagement != 100 {
		t.Fatalf("heatmap cell %+v", hm[10])
	}
	if g := FollowerGrowth(0, 10); g.Pct != nil || g.Abs != 10 {
		t.Fatal("growth from zero has nil pct")
	}
	if ContentType("VIDEO", sp("REELS")) != "Reels" || ContentType("CAROUSEL_ALBUM", nil) != "Carousel" || ContentType("IMAGE", nil) != "Foto" {
		t.Fatal("content type")
	}
	if NormalizeIgUsername(" @Kopi.Nusantara ") != "kopi.nusantara" || NormalizeIgUsername(".bad") != "" || NormalizeIgUsername("a") != "" {
		t.Fatal("ig username")
	}
	if Truncate("abcdefghij", 5) != "abcd…" || Truncate("abc", 5) != "abc" {
		t.Fatal("truncate")
	}
	idx := IndexTo100([]*int{nil, ip(0), ip(200), ip(300)})
	if idx[0] != nil || idx[1] != nil || *idx[2] != 100 || *idx[3] != 150 {
		t.Fatal("index to 100")
	}
}

func TestDailyFollowerSeries(t *testing.T) {
	snaps := []SocialSnapshot{{Date: day("2026-08-30"), Followers: 100}, {Date: day("2026-09-02"), Followers: 120, Reach: ip(5)}}
	s := DailyFollowerSeries(dates.Range{From: day("2026-09-01"), To: day("2026-09-03")}, snaps)
	if *s[0].Followers != 100 || *s[1].Followers != 120 || *s[2].Followers != 120 || s[1].Reach != 5 || s[0].Reach != 0 {
		t.Fatalf("%+v", s)
	}
	none := DailyFollowerSeries(dates.Range{From: day("2026-08-01"), To: day("2026-08-01")}, snaps)
	if none[0].Followers != nil {
		t.Fatal("no prior snapshot -> nil followers")
	}
	if FollowersAtAcrossAccounts([]SocialSnapshot{{AccountID: "a", Date: day("2026-09-01"), Followers: 5}, {AccountID: "b", Date: day("2026-09-05"), Followers: 7}}, day("2026-09-02")) != 5 {
		t.Fatal("account without snapshot yet contributes nothing")
	}
}

func TestFillDailyKeepsNull(t *testing.T) {
	type row struct {
		d time.Time
		v float64
	}
	r := dates.Range{From: day("2026-09-01"), To: day("2026-09-03")}
	rows := []row{{day("2026-09-01"), 2}, {day("2026-09-01"), 3}, {day("2026-09-03"), 4}}
	fields := []func(row) float64{func(x row) float64 { return x.v }}
	nulls := FillDaily(r, rows, func(x row) time.Time { return x.d }, fields, false)
	if *nulls[0].Get(0) != 5 || nulls[1].Get(0) != nil || *nulls[2].Get(0) != 4 {
		t.Fatalf("null fill %+v", nulls)
	}
	zeros := FillDaily(r, rows, func(x row) time.Time { return x.d }, fields, true)
	if *zeros[1].Get(0) != 0 {
		t.Fatal("zero fill")
	}
	spark := Sparkline(nulls, 0, 2)
	if len(spark) != 2 || spark[0] != 0 || spark[1] != 4 {
		t.Fatalf("sparkline turns nil into 0: %v", spark)
	}
	aligned := AlignPrevious(nulls, nulls[:2], 0)
	if *aligned[0] != 5 || aligned[1] != nil || aligned[2] != nil {
		t.Fatalf("align previous %v", aligned)
	}
}

func TestQualityStates(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	pts := []time.Time{day("2026-10-20"), day("2026-10-01")}
	fresh := &SyncJob{Status: JobSuccess, StartedAt: now.Add(-47 * time.Hour)}
	old := &SyncJob{Status: JobSuccess, StartedAt: now.Add(-49 * time.Hour)}
	failed := &SyncJob{Status: JobFailed, StartedAt: now}
	cases := []struct {
		name      string
		hasSource bool
		demo      bool
		points    []time.Time
		job       *SyncJob
		want      DataState
	}{
		{"no source", false, true, nil, failed, StateNotConnected},
		{"failed beats demo", true, true, pts, failed, StateFailed},
		{"demo", true, true, pts, fresh, StateDemo},
		{"empty", true, false, nil, fresh, StateEmpty},
		{"stale after 48h", true, false, pts, old, StateStale},
		{"healthy at 47h", true, false, pts, fresh, StateHealthy},
		{"no job -> healthy", true, false, pts, nil, StateHealthy},
	}
	for _, c := range cases {
		q := Quality(c.hasSource, c.demo, c.points, c.job, now)
		if q.State != c.want {
			t.Errorf("%s: got %s want %s", c.name, q.State, c.want)
		}
	}
	q := Quality(true, false, pts, fresh, now)
	if !q.DataFrom.Equal(day("2026-10-01")) || !q.DataTo.Equal(day("2026-10-20")) {
		t.Fatal("data range")
	}
	if !IsDemo(nil) || IsDemo([]*string{nil, sp("c")}) {
		t.Fatal("is demo")
	}
	_ = metrics.ComputeDelta
}

func TestDemographicsOrder(t *testing.T) {
	d := BuildDemographics([]AdDemographic{{Age: "65+", Gender: "male", Spend: 1}, {Age: "18-24", Gender: "female", Spend: 2}, {Age: "weird", Gender: "unknown", Spend: 3}})
	if d.ByAge[0].Age != "18-24" || d.ByAge[2].Age != "weird" || d.ByGender[0].Gender != "female" {
		t.Fatalf("%+v", d)
	}
}

func TestURLPath(t *testing.T) {
	if URLPath("https://x.id/menu?a=1") != "/menu?a=1" || URLPath("https://x.id") != "/" || URLPath("not a url") != "not a url" {
		t.Fatal("url path")
	}
}
