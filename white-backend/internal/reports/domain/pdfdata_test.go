package domain

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

func day(d int) time.Time { return time.Date(2026, 8, d, 0, 0, 0, 0, time.UTC) }

func i64(v int64) *int64 { return &v }

func strp(s string) *string { return &s }

// fixture mirrors what analytics GetReportData returns for 3 August days.
func fixture() ReportData {
	roas := 3.4
	health := int32(82)
	return ReportData{
		Social: &SocialData{
			Overview: SocialOverview{
				HasData: true, Followers: 125_400, FollowersDelta: metrics.ComputeDelta(2600, 2000),
				Posts: 24, PostsDelta: metrics.ComputeDelta(24, 20), EngagementRate: 4.32, EngagementRateDelta: metrics.ComputeDelta(4.32, 4.5),
				Reach: 890_000, ReachDelta: metrics.ComputeDelta(890_000, 0),
			},
			DailyFollowers: []FollowersPoint{{day(9), 1000}, {day(10), 1010}, {day(11), 1005}, {day(12), 1030}},
			TopPosts: []TopPost{
				{ID: "p1", Platform: "INSTAGRAM", Caption: strings.Repeat("Kopi Gayo ", 12), MediaType: "VIDEO", ProductType: "REELS", Likes: 3210, Comments: 148, Engagements: 3590},
				{ID: "p2", Platform: "TIKTOK", Caption: "Video", MediaType: "VIDEO", Likes: 10, Comments: 1, Engagements: 11},
			},
		},
		Seo: &SeoData{
			HasData: true, Clicks: 8412, ClicksDelta: metrics.ComputeDelta(8412, 7490), Impressions: 402_000,
			ImpressionsDelta: metrics.ComputeDelta(402_000, 0), Ctr: 2.09, CtrDelta: metrics.ComputeDelta(2.09, 2), Position: 9.4,
			PositionDelta: metrics.ComputeDelta(9.4, 10), Health: &health,
			Daily: []SeoDailyPoint{{day(10), i64(300), i64(9000)}, {day(11), nil, nil}, {day(12), i64(420), i64(9900)}},
			TopQueries: []TopQuery{
				{"kopi gayo asli", 812, 22_400, 3.63, 2.4},
				{"beli kopi arabica online", 402, 31_000, 1.3, 6.8},
			},
		},
		Ads: &AdsData{
			HasData:    true,
			Kpis:       AdKpis{Spend: 45_200_000, Results: 1204, Cpr: 37_542, Ctr: 1.84, Cpm: 21_400, Roas: &roas},
			SpendDelta: metrics.ComputeDelta(45_200_000, 41_000_000), ResultsDelta: metrics.ComputeDelta(1204, 1000),
			CprDelta: metrics.ComputeDelta(37_542, 41_000),
			Daily:    []AdsDailyPoint{{day(10), 1_500_000, 40}, {day(11), 1_700_000, 44}},
			Campaigns: []Campaign{
				{ID: "c1", Name: "Promo Kemerdekaan", Objective: "OUTCOME_SALES", Spend: 15_100_000, Results: 480, ResultType: strp("purchase"), Cpr: 31_458, IsConversion: true},
				{ID: "c2", Name: "Brand Awareness", Objective: "OUTCOME_AWARENESS", Spend: 5_000_000, Results: 90_000, ResultType: strp("reach"), IsConversion: false},
			},
		},
	}
}

var window = dates.Range{From: day(10), To: day(12)}

func args(modules ...string) PdfArgs {
	return PdfArgs{
		ClientName: "Kopi Nusantara", Currency: "IDR", Title: "Laporan Agustus", Current: window, Previous: dates.Previous(window),
		Compare: true, Modules: modules, Language: "id", Now: time.Date(2026, 8, 19, 7, 5, 0, 0, time.UTC),
	}
}

func TestBuildPdfDataLabelsAndOrder(t *testing.T) {
	d := BuildPdfData(args(ModuleAds, ModuleSocial, ModuleSeo), fixture(), Sections{})
	var keys []string
	for _, m := range d.Modules {
		keys = append(keys, m.Key)
	}
	if !reflect.DeepEqual(keys, []string{"SOCIAL", "SEO", "ADS"}) {
		t.Fatalf("module order %v", keys)
	}
	if d.PeriodLabel != "10 Agu – 12 Agu 2026" || d.ComparedLabel != "vs periode sebelumnya: 7 Agu – 9 Agu 2026" || d.GeneratedLabel != "19 Agu 2026, 14.05" {
		t.Fatalf("labels %q | %q | %q", d.PeriodLabel, d.ComparedLabel, d.GeneratedLabel)
	}
	off := args(ModuleSocial)
	off.Compare = false
	d = BuildPdfData(off, fixture(), Sections{})
	if d.ComparedLabel != "" {
		t.Fatal("no compared label without compare")
	}
	for _, k := range d.Modules[0].Kpis {
		if k.Delta != "" {
			t.Fatalf("compare off still shows %q", k.Delta)
		}
	}
}

func TestBuildPdfDataSocial(t *testing.T) {
	m := BuildPdfData(args(ModuleSocial), fixture(), Sections{}).Modules[0]
	if m.Kpis[0].Value != "125,4 rb" || m.Kpis[0].Delta != "+30,0%" || *m.Kpis[0].DeltaGood != true {
		t.Fatalf("followers kpi %+v", m.Kpis[0])
	}
	if m.Kpis[1].Delta != "−4,0%" || *m.Kpis[1].DeltaGood != false {
		t.Fatalf("engagement kpi %+v", m.Kpis[1])
	}
	if m.Kpis[2].Delta != "" {
		t.Fatal("a delta without pct stays hidden")
	}
	var values []float64
	for _, p := range m.Chart.Points {
		values = append(values, p.Value)
	}
	if !reflect.DeepEqual(values, []float64{10, -5, 25}) || m.Chart.MaxLabel != "+25/hari" || m.Chart.Points[0].Date != "10 Agu" {
		t.Fatalf("growth chart %+v", m.Chart)
	}
	row := m.Tables[0].Rows[0]
	if row[0] != "Instagram" || row[2] != "Reels" || len([]rune(row[1])) != 64 || row[5] != "3.590" {
		t.Fatalf("top post row %q", row)
	}
	if m.Tables[0].Rows[1][0] != "TikTok" || m.Tables[0].Rows[1][2] != "Video" {
		t.Fatalf("second row %q", m.Tables[0].Rows[1])
	}
}

func TestBuildPdfDataSeoSkipsUnreportedDays(t *testing.T) {
	m := BuildPdfData(args(ModuleSeo), fixture(), Sections{}).Modules[0]
	if len(m.Chart.Points) != 2 || m.Chart.Points[1].Date != "12 Agu" || m.Chart.MaxLabel != "420" {
		t.Fatalf("seo chart %+v", m.Chart)
	}
	pos := m.Kpis[3]
	if pos.Label != "Posisi rata-rata" || pos.Value != "9,4" || *pos.DeltaGood != true {
		t.Fatalf("position is lower-is-better: %+v", pos)
	}
	if m.Kpis[4].Value != "82/100" {
		t.Fatalf("health %+v", m.Kpis[4])
	}
	if !reflect.DeepEqual(m.Tables[0].Rows[1], []string{"beli kopi arabica online", "402", "31 rb", "1,30%", "6,8"}) {
		t.Fatalf("query row %q", m.Tables[0].Rows[1])
	}

	one := fixture()
	one.Seo.Daily = []SeoDailyPoint{{day(10), i64(5), nil}, {day(11), nil, nil}}
	if BuildPdfData(args(ModuleSeo), one, Sections{}).Modules[0].Chart != nil {
		t.Fatal("one reported day draws no chart")
	}
}

func TestBuildPdfDataAds(t *testing.T) {
	m := BuildPdfData(args(ModuleAds), fixture(), Sections{}).Modules[0]
	if m.Kpis[0].Value != "Rp 45,2 jt" || m.Kpis[2].Value != "Rp 37.542" || *m.Kpis[2].DeltaGood != true {
		t.Fatalf("ads kpis %+v", m.Kpis)
	}
	if m.Kpis[5].Value != "3,40x" {
		t.Fatalf("roas %+v", m.Kpis[5])
	}
	rows := m.Tables[0].Rows
	if !reflect.DeepEqual(rows[0], []string{"Promo Kemerdekaan", "Sales", "Rp 15,1 jt", "480", "Rp 31.458"}) {
		t.Fatalf("conversion row %q", rows[0])
	}
	if rows[1][1] != "Awareness" || rows[1][4] != "—" {
		t.Fatalf("non-conversion row %q", rows[1])
	}

	empty := fixture()
	empty.Ads.HasData = false
	if got := BuildPdfData(args(ModuleAds), empty, Sections{Seo: []Section{{Title: "x"}}}).Modules; len(got) != 0 {
		t.Fatalf("ADS without data has no page: %+v", got)
	}
}

func TestBuildPdfDataSectionsOnlyPages(t *testing.T) {
	data := fixture()
	data.Social.Overview.HasData = false
	data.Seo.HasData = false
	pub := BuildPublishingSection(&PublishingSummary{Posts: 1, Mode: "recent", Top: []PublishedTarget{{Title: "a", Platform: "FACEBOOK", PublishedAt: day(11)}}}, L)
	d := BuildPdfData(args(ModuleSocial, ModuleSeo), data, Sections{Social: []Section{*pub}})
	if len(d.Modules) != 1 || d.Modules[0].Key != "SOCIAL" || len(d.Modules[0].Kpis) != 0 || len(d.Modules[0].Sections) != 1 {
		t.Fatalf("sections-only page %+v", d.Modules)
	}
}

func TestBuildInsightPayload(t *testing.T) {
	data := fixture()
	data.Seo.TopQueries = make([]TopQuery, 12)
	for i := range data.Seo.TopQueries {
		data.Seo.TopQueries[i] = TopQuery{Query: "q", Position: 5}
	}
	for i := 0; i < 5; i++ {
		data.Ads.Campaigns = append(data.Ads.Campaigns, Campaign{Name: "extra"})
	}
	p := BuildInsightPayload(ModuleOverview, data, "")
	if len(p.Seo.TopQueries) != 5 || len(p.Ads.Campaigns) != 6 || p.Ads.Currency != "IDR" {
		t.Fatalf("overview limits: queries=%d campaigns=%d currency=%q", len(p.Seo.TopQueries), len(p.Ads.Campaigns), p.Ads.Currency)
	}
	if len([]rune(p.Social.TopPosts[0].Caption)) != 80 || p.Social.TopPosts[0].Type != "Reels" {
		t.Fatalf("top post %+v", p.Social.TopPosts[0])
	}
	if got := BuildInsightPayload(ModuleSeo, data, "IDR"); len(got.Seo.TopQueries) != 10 {
		t.Fatalf("SEO insight carries 10 queries, got %d", len(got.Seo.TopQueries))
	}
	data.Social.Overview.HasData = false
	raw, err := json.Marshal(BuildInsightPayload(ModuleOverview, data, "IDR"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if strings.Contains(s, `"social"`) || !strings.Contains(s, `"impressionsDeltaPct":null`) || !strings.HasPrefix(s, `{"seo":{"clicks":8412,"clicksDeltaPct":12.309`) {
		t.Fatalf("payload json %s", s)
	}
}

func TestInsightModules(t *testing.T) {
	cases := []struct {
		module    string
		selection []string
		want      []string
	}{
		{ModuleOverview, nil, []string{"SOCIAL", "SEO", "ADS"}},
		{ModuleOverview, []string{"ADS", "SOCIAL"}, []string{"SOCIAL", "ADS"}},
		{ModuleSeo, []string{"ADS"}, []string{"SEO"}},
	}
	for _, c := range cases {
		if got := InsightModules(c.module, c.selection); !reflect.DeepEqual(got, c.want) {
			t.Errorf("InsightModules(%s, %v) = %v", c.module, c.selection, got)
		}
	}
	r := ToRangeInfo(window, dates.Previous(window))
	if r != (InsightRange{From: "2026-08-10", To: "2026-08-12", Days: 3, PrevFrom: "2026-08-07", PrevTo: "2026-08-09"}) {
		t.Fatalf("range info %+v", r)
	}
}

func TestShareView(t *testing.T) {
	data := fixture()
	data.Ads.HasData = false
	v := ShareView(data, day(10))
	if v.Ads != nil {
		t.Fatal("ads without data stays absent")
	}
	if len(v.Social.DailyFollowers) != 3 || !v.Social.DailyFollowers[0].Date.Equal(day(10)) {
		t.Fatalf("daily followers cover the range only: %+v", v.Social.DailyFollowers)
	}
	if n := len([]rune(v.Social.TopPosts[0].Caption)); n != 90 {
		t.Fatalf("caption truncated to 90, got %d", n)
	}
	if v.Seo.Daily[1].Clicks != nil {
		t.Fatal("unreported GSC day stays null")
	}
	if data.Social.TopPosts[0].Caption == v.Social.TopPosts[0].Caption {
		t.Fatal("input must not be mutated")
	}
}

func TestSharePresetAndFilename(t *testing.T) {
	if SharePreset("90d") != "90d" || SharePreset("12m") != "28d" || SharePreset("") != "28d" {
		t.Fatal("share presets")
	}
	if got := PdfFilename("kopi-nusantara", day(10), day(12)); got != "Laporan_kopi-nusantara_2026-08-10_2026-08-12.pdf" {
		t.Fatalf("filename %q", got)
	}
	if ShareTitle("Kopi") != "Laporan Performa: Kopi" {
		t.Fatal("share title")
	}
}
