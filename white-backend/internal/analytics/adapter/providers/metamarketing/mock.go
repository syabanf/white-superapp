package metamarketing

import (
	"context"
	"fmt"
	"math"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

// Mock serves three blueprint campaigns per ad account, seeded by actId.
type Mock struct{}

type blueprintSet struct {
	name string
	ads  []string
}

type blueprint struct {
	name        string
	objective   string
	status      string
	dailyBudget float64
	resultType  string
	cpr         float64
	ctr         float64
	adSets      []blueprintSet
}

var blueprints = []blueprint{
	{name: "[Prospecting] Traffic – Demo", objective: "OUTCOME_TRAFFIC", status: "ACTIVE", dailyBudget: 300000, resultType: "link_click", cpr: 1800, ctr: 1.8,
		adSets: []blueprintSet{{"Broad 18-34", []string{"Video 15s", "Carousel Produk"}}, {"Lookalike 1%", []string{"Reels UGC", "Static Promo"}}}},
	{name: "[Retargeting] Sales – Demo", objective: "OUTCOME_SALES", status: "ACTIVE", dailyBudget: 200000, resultType: "purchase", cpr: 42000, ctr: 2.4,
		adSets: []blueprintSet{{"Website Visitors 30D", []string{"Promo Bundling", "Testimoni"}}}},
	{name: "[Leads] Form – Demo", objective: "OUTCOME_LEADS", status: "PAUSED", dailyBudget: 150000, resultType: "lead", cpr: 55000, ctr: 1.1,
		adSets: []blueprintSet{{"Interest Bisnis 30-50", []string{"Lead Form 2026"}}}},
}

var (
	ages          = []string{"18-24", "25-34", "35-44", "45-54", "55-64", "65+"}
	genders       = []string{"female", "male", "unknown"}
	ageWeights    = []float64{0.12, 0.36, 0.28, 0.14, 0.07, 0.03}
	genderWeights = []float64{0.55, 0.43, 0.02}
)

func jsRound(x float64) float64 { return math.Floor(x + 0.5) }

func campaignID(actID string, i int) string {
	return fmt.Sprintf("%d", 120_000_000_000+int64(mockrand.Seed(actID)%900_000)*100+int64(i)*10)
}

type built struct {
	campaigns []Campaign
	adSets    []AdSet
	ads       []Ad
	bpByCamp  map[string]blueprint
}

func build(actID string) built {
	b := built{bpByCamp: map[string]blueprint{}}
	start := "2026-01-01T00:00:00+0700"
	for i, bp := range blueprints {
		cid := campaignID(actID, i)
		b.bpByCamp[cid] = bp
		budget := bp.dailyBudget
		b.campaigns = append(b.campaigns, Campaign{ID: cid, Name: bp.name, Objective: bp.objective, Status: bp.status, DailyBudget: &budget, StartTime: &start})
		for j, set := range bp.adSets {
			sid := fmt.Sprintf("%s%d", cid, j+1)
			setBudget := bp.dailyBudget / float64(len(bp.adSets))
			b.adSets = append(b.adSets, AdSet{ID: sid, CampaignID: cid, Name: set.name, Status: bp.status, DailyBudget: &setBudget})
			for m, adName := range set.ads {
				thumb := fmt.Sprintf("https://picsum.photos/seed/meta%d%d%d/400/400", i, j, m)
				b.ads = append(b.ads, Ad{ID: fmt.Sprintf("%s%d", sid, m+1), AdSetID: sid, CampaignID: cid, Name: adName, Status: bp.status, ThumbnailURL: &thumb})
			}
		}
	}
	return b
}

func (Mock) ListCampaigns(_ context.Context, actID string) ([]Campaign, error) {
	return build(actID).campaigns, nil
}

func (Mock) ListAdSets(_ context.Context, actID string) ([]AdSet, error) {
	return build(actID).adSets, nil
}

func (Mock) ListAds(_ context.Context, actID string) ([]Ad, error) { return build(actID).ads, nil }

func (Mock) GetInsights(_ context.Context, actID string, q InsightsQuery) ([]InsightRow, error) {
	b := build(actID)
	from, err1 := dates.Parse(q.Since)
	until, err2 := dates.Parse(q.Until)
	if err1 != nil || err2 != nil {
		return nil, nil
	}
	days := []string{}
	for d := from; !d.After(until) && len(days) < 400; d = dates.AddDays(d, 1) {
		days = append(days, dates.Format(d))
	}
	rows := []InsightRow{}
	withBreakdown := len(q.Breakdowns) > 0
	for _, day := range days {
		for ai, ad := range b.ads {
			bp, ok := b.bpByCamp[ad.CampaignID]
			if !ok {
				bp = blueprints[ai%len(blueprints)]
			}
			rnd := mockrand.New(actID + "|" + ad.ID + "|" + day)
			spend := jsRound((bp.dailyBudget / 2) * (0.7 + rnd.Float64()*0.6))
			impressions := jsRound(spend / (25000 + rnd.Float64()*12000) * 1000)
			reach := jsRound(impressions / (1.5 + rnd.Float64()))
			linkClicks := jsRound(impressions * (bp.ctr / 100) * (0.75 + rnd.Float64()*0.5))
			clicks := jsRound(linkClicks * 1.3)
			results := math.Max(0, jsRound(spend/(bp.cpr*(0.8+rnd.Float64()*0.4))))
			freq := 0.0
			if reach > 0 {
				freq = impressions / reach
			}
			rt := bp.resultType
			base := InsightRow{CampaignID: ad.CampaignID, AdSetID: ad.AdSetID, AdID: ad.ID, Date: day, Spend: spend, Impressions: impressions, Reach: reach, Clicks: clicks, LinkClicks: linkClicks, Frequency: freq, Results: results, ResultType: &rt}
			if bp.resultType == "purchase" {
				pv := jsRound(results * bp.cpr * 3)
				base.PurchaseValue = &pv
			}
			if !withBreakdown {
				rows = append(rows, base)
				continue
			}
			for x, age := range ages {
				for y, gender := range genders {
					w := ageWeights[x] * genderWeights[y]
					row := base
					a, g := age, gender
					row.Age, row.Gender = &a, &g
					row.Spend = jsRound(base.Spend * w)
					row.Impressions = jsRound(base.Impressions * w)
					row.Reach = jsRound(base.Reach * w)
					row.Clicks = jsRound(base.Clicks * w)
					row.LinkClicks = jsRound(base.LinkClicks * w)
					row.Results = jsRound(base.Results * w)
					if base.PurchaseValue != nil {
						pv := jsRound(*base.PurchaseValue * w)
						row.PurchaseValue = &pv
					}
					rows = append(rows, row)
				}
			}
		}
	}
	return rows, nil
}
