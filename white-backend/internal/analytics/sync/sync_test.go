package sync

import (
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/googlesearch"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/metamarketing"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
)

func TestSeoMappers(t *testing.T) {
	daily := MapGscDaily("p", []googlesearch.GscRow{{Keys: []string{"2026-09-01"}, Clicks: 2.6, Impressions: 10.4, CTR: 3.14159, Position: 4.567}, {Keys: []string{"bad"}}})
	if len(daily) != 1 || daily[0].Clicks != 3 || daily[0].Impressions != 10 || daily[0].CTR != 3.14 || daily[0].Position != 4.57 {
		t.Fatalf("%+v", daily)
	}
	dims := MapGscDimension("p", []googlesearch.GscRow{{Keys: []string{"2026-09-01", " Kopi SUSU "}}, {Keys: []string{"2026-09-01", "mobile"}}, {Keys: []string{"2026-09-01", ""}}}, googlesearch.DimQuery)
	if len(dims) != 2 || dims[0].Key != "kopi susu" || dims[0].Dimension != domain.DimQuery {
		t.Fatalf("%+v", dims)
	}
	if MapGscDimension("p", []googlesearch.GscRow{{Keys: []string{"2026-09-01", "mobile"}}}, googlesearch.DimDevice)[0].Key != "MOBILE" {
		t.Fatal("device upper")
	}
	ga := MapGa4Daily("p", []googlesearch.Ga4Row{
		{Dimensions: []string{"20260902", "Direct"}, Metrics: []float64{10, 5, 8, 1, 100}},
		{Dimensions: []string{"20260901", "Organic Search"}, Metrics: []float64{30, 20, 25, 2, 900}},
		{Dimensions: []string{"20260901", "Direct"}, Metrics: []float64{10, 5, 8, 0, 300}},
	})
	if len(ga) != 2 || ga[0].Sessions != 40 || ga[0].OrganicSessions != 30 || ga[0].AvgEngagementSec != 30 {
		t.Fatalf("%+v", ga)
	}
	if Ga4Date("2026-09-01") != "2026-09-01" || Ga4Date("x") != "" {
		t.Fatal("ga4 date")
	}
}

func TestAdsMappers(t *testing.T) {
	rt := "purchase"
	pv := 10.0
	rows := []metamarketing.InsightRow{
		{Date: "2026-09-01", CampaignID: "c", AdSetID: "s", AdID: "a", Spend: 1, Results: 1, ResultType: &rt, PurchaseValue: &pv},
		{Date: "2026-09-01", CampaignID: "c", AdSetID: "s", AdID: "a", Spend: 2, Results: 2, PurchaseValue: &pv},
		{Date: "2026-09-01", CampaignID: "c", AdSetID: "s", AdID: ""},
	}
	agg := AggregateDailyInsights(rows)
	if len(agg) != 1 || agg[0].Spend != 3 || *agg[0].PurchaseValue != 20 || *agg[0].ResultType != "purchase" {
		t.Fatalf("%+v", agg)
	}
	g := "Female"
	demo := AggregateDemographics([]metamarketing.InsightRow{{CampaignID: "c", Gender: &g, Spend: 1}, {CampaignID: "c", Gender: &g, Spend: 2}, {CampaignID: ""}})
	if len(demo) != 1 || demo[0].Gender != "female" || demo[0].Age != "unknown" || demo[0].Spend != 3 {
		t.Fatalf("%+v", demo)
	}
}

func TestAuditRules(t *testing.T) {
	now := time.Date(2026, 10, 21, 2, 0, 0, 0, time.UTC)
	at := func(h time.Duration) *time.Time { t := now.Add(-h); return &t }
	cases := []struct {
		c    domain.AuditCadence
		last *time.Time
		want bool
	}{
		{domain.CadenceNone, nil, false},
		{domain.CadenceDaily, nil, true},
		{domain.CadenceDaily, at(19 * time.Hour), false},
		{domain.CadenceDaily, at(21 * time.Hour), true},
		{domain.CadenceWeekly, at(5 * 24 * time.Hour), false},
		{domain.CadenceWeekly, at(7 * 24 * time.Hour), true},
	}
	for _, c := range cases {
		if ShouldAudit(c.c, c.last, now) != c.want {
			t.Errorf("%s %v", c.c, c.last)
		}
	}
	if AuditTarget("sc-domain:kopi.id", "") != "https://kopi.id/" || AuditTarget("https://a/", "https://b/") != "https://b/" {
		t.Fatal("audit target")
	}
}
