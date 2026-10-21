// Package googlesearch talks to Google Search Console (searchAnalytics
// query) and the GA4 Data API (runReport). The real adapter needs an OAuth
// access token; the mock is deterministic.
package googlesearch

import "context"

const (
	DimDate    = "date"
	DimQuery   = "query"
	DimPage    = "page"
	DimCountry = "country"
	DimDevice  = "device"
)

type GscQuery struct {
	// GSC property, "https://x/" or "sc-domain:x".
	SiteURL    string
	StartDate  string
	EndDate    string
	Dimensions []string
	RowLimit   int
	StartRow   int
}

type GscRow struct {
	Keys        []string
	Clicks      float64
	Impressions float64
	// Percent 0 to 100 (Google returns a 0 to 1 fraction).
	CTR      float64
	Position float64
}

type Ga4Request struct {
	// "properties/123" or the bare id.
	PropertyID string
	StartDate  string
	EndDate    string
	Metrics    []string
	Dimensions []string
	Limit      int
}

type Ga4Row struct {
	Dimensions []string
	Metrics    []float64
}

type Provider interface {
	QuerySearchAnalytics(ctx context.Context, token string, req GscQuery) ([]GscRow, error)
	RunGa4Report(ctx context.Context, token string, req Ga4Request) ([]Ga4Row, error)
}

// Pick returns the real adapter when the Google app is configured.
func Pick(configured bool) Provider {
	if configured {
		return NewReal()
	}
	return Mock{}
}
