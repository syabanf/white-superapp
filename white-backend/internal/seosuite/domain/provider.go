package domain

// Value types every SEO data provider (DataForSEO, Apify SERP, mock) returns.
// Money is IDR, dates are YYYY-MM-DD strings so results can be cached as JSON.

type KeywordIdea struct {
	Keyword      string   `json:"keyword"`
	Volume       int32    `json:"volume"`
	Difficulty   int32    `json:"difficulty"`
	CPC          float64  `json:"cpc"`
	Competition  float64  `json:"competition"`
	Intent       Intent   `json:"intent"`
	Trend        []int32  `json:"trend"`
	SerpFeatures []string `json:"serpFeatures"`
}

type SerpRankRequest struct {
	Keyword string
	Device  Device
}

type RankResult struct {
	Keyword      string
	Device       Device
	Position     *int32
	URL          *string
	SerpFeatures []string
	Top10        []TopResult
}

type BacklinkSummary struct {
	Backlinks        int32
	ReferringDomains int32
	Dofollow         int32
	Nofollow         int32
	DomainRank       int32
	NewLast30        int32
	LostLast30       int32
	// ToxicShare is 0..1.
	ToxicShare float64
}

type BacklinkItem struct {
	SourceURL    string
	SourceDomain string
	TargetURL    string
	Anchor       string
	Dofollow     bool
	DomainRank   int32
	SpamScore    int32
	FirstSeen    string
	LastSeen     string
	IsLost       bool
}

type DomainOverview struct {
	Domain           string
	OrganicKeywords  int32
	OrganicTraffic   int32
	OrganicCost      float64
	Top3             int32
	Top10            int32
	Top100           int32
	Backlinks        int32
	ReferringDomains int32
	DomainRank       int32
}

type DomainKeyword struct {
	Keyword  string
	Position int32
	Volume   int32
	URL      string
}

type CompetitorSuggestion struct {
	Domain         string
	Intersections  int32
	OrganicTraffic int32
}

type BacklinkMode string

const (
	BacklinksAll  BacklinkMode = "all"
	BacklinksNew  BacklinkMode = "new"
	BacklinksLost BacklinkMode = "lost"
)
