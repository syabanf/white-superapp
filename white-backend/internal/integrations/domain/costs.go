package domain

import "math"

// Monthly estimates are fixed at 30 days.
const estimateDays = 30

// DataForSEO unit prices in USD (public price list; verify before budgeting).
type SeoCostPrices struct {
	SerpPerKeyword                float64
	BacklinkSummaryPerDay         float64
	DomainOverviewPerDomainPerDay float64
	ResearchPerRun                float64
}

var DefaultSeoPrices = SeoCostPrices{
	SerpPerKeyword:                0.002,
	BacklinkSummaryPerDay:         0.02,
	DomainOverviewPerDomainPerDay: 0.01,
	ResearchPerRun:                0.05,
}

// Apify pay-per-result prices in USD.
var ApifyPrices = struct {
	SerpPerKeyword     float64
	InstagramPerResult float64
	TiktokPerResult    float64
	FacebookPerResult  float64
}{
	SerpPerKeyword:     0.0035,
	InstagramPerResult: 0.0023,
	TiktokPerResult:    0.004,
	FacebookPerResult:  0.003,
}

type CostEstimate struct {
	// LinesUSD is one amount per line item (rank, backlinks, domains, research / serp, social).
	LinesUSD map[string]float64
	TotalUSD float64
	TotalIDR float64
}

type SeoCostInput struct {
	TrackedKeywords      int
	CompetitorDomains    int
	ResearchRunsPerMonth int
	UsdIdr               float64
}

// EstimateSeoCost prices the daily rank job, daily backlink and domain
// snapshots (own domain + competitors) and ad-hoc research runs.
func EstimateSeoCost(in SeoCostInput) CostEstimate {
	p := DefaultSeoPrices
	kw := clampZero(in.TrackedKeywords)
	domains := 1 + clampZero(in.CompetitorDomains)
	rank := kw * estimateDays * p.SerpPerKeyword
	backlinks := estimateDays * p.BacklinkSummaryPerDay
	dom := domains * estimateDays * p.DomainOverviewPerDomainPerDay
	research := clampZero(in.ResearchRunsPerMonth) * p.ResearchPerRun
	total := rank + backlinks + dom + research
	return CostEstimate{
		LinesUSD: map[string]float64{"rank": rank, "backlinks": backlinks, "domains": dom, "research": research},
		TotalUSD: total,
		TotalIDR: total * math.Max(0, in.UsdIdr),
	}
}

type ApifyCostInput struct {
	TrackedKeywords int
	SocialAccounts  int
	PostsPerAccount int
	UsdIdr          float64
}

// EstimateApifyCost prices a daily SERP per keyword plus a daily profile and
// posts refresh per account. Every account is priced at the Instagram rate.
func EstimateApifyCost(in ApifyCostInput) CostEstimate {
	serp := clampZero(in.TrackedKeywords) * estimateDays * ApifyPrices.SerpPerKeyword
	perAccountDay := (1 + clampZero(in.PostsPerAccount)) * ApifyPrices.InstagramPerResult
	social := clampZero(in.SocialAccounts) * estimateDays * perAccountDay
	total := serp + social
	return CostEstimate{
		LinesUSD: map[string]float64{"serp": serp, "social": social},
		TotalUSD: total,
		TotalIDR: total * math.Max(0, in.UsdIdr),
	}
}

func clampZero(n int) float64 {
	if n < 0 {
		return 0
	}
	return float64(n)
}
