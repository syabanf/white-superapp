// Package metamarketing reads campaigns, ad sets, ads and insights from the
// Meta Marketing API. Real needs app credentials and a token; otherwise the
// deterministic blueprint mock answers.
package metamarketing

import "context"

type Campaign struct {
	ID          string
	Name        string
	Objective   string
	Status      string
	DailyBudget *float64
	StartTime   *string
}

type AdSet struct {
	ID          string
	CampaignID  string
	Name        string
	Status      string
	DailyBudget *float64
}

type Ad struct {
	ID           string
	AdSetID      string
	CampaignID   string
	Name         string
	Status       string
	ThumbnailURL *string
}

type InsightRow struct {
	CampaignID string
	AdSetID    string
	AdID       string
	// date_start, YYYY-MM-DD.
	Date          string
	Spend         float64
	Impressions   float64
	Reach         float64
	Clicks        float64
	LinkClicks    float64
	Frequency     float64
	Results       float64
	ResultType    *string
	PurchaseValue *float64
	Age           *string
	Gender        *string
}

type InsightsQuery struct {
	Since string
	Until string
	// ad | adset | campaign (default ad)
	Level         string
	TimeIncrement int
	Breakdowns    []string
}

type Adapter interface {
	ListCampaigns(ctx context.Context, actID string) ([]Campaign, error)
	ListAdSets(ctx context.Context, actID string) ([]AdSet, error)
	ListAds(ctx context.Context, actID string) ([]Ad, error)
	GetInsights(ctx context.Context, actID string, q InsightsQuery) ([]InsightRow, error)
}

// Pick returns the real adapter only with app credentials and a token.
func Pick(configured bool, token string) Adapter {
	if configured && token != "" {
		return NewReal(token)
	}
	return Mock{}
}
