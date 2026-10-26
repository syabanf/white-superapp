// Package domain holds the reports entities, the shapes of the upstream
// numbers the service composes (analytics, publishing, seosuite) and the
// pure builders that turn them into report sections, PDF data, AI insight
// payloads and share dashboard blocks.
package domain

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

// Module keys of a report (ModuleKey) and of an insight (adds OVERVIEW).
const (
	ModuleOverview = "OVERVIEW"
	ModuleSocial   = "SOCIAL"
	ModuleSeo      = "SEO"
	ModuleAds      = "ADS"
)

// ReportModules is the fixed order of module pages in the PDF.
var ReportModules = []string{ModuleSocial, ModuleSeo, ModuleAds}

// IsInsightModule accepts OVERVIEW, SOCIAL, SEO and ADS.
func IsInsightModule(m string) bool {
	return m == ModuleOverview || slices.Contains(ReportModules, m)
}

// Lang maps anything but "en" to "id".
func Lang(s string) string {
	if s == "en" {
		return "en"
	}
	return "id"
}

type Report struct {
	ID          string
	ClientID    string
	Title       string
	DateFrom    time.Time
	DateTo      time.Time
	Compare     bool
	Modules     []string
	Language    string
	AiSummary   *string
	CreatedByID *string
	CreatedAt   time.Time
}

type Insight struct {
	ID        string
	ClientID  string
	Module    string
	DateFrom  time.Time
	DateTo    time.Time
	Language  string
	Model     string
	Content   string
	CreatedAt time.Time
}

// Store is the reports schema. Missing rows come back as nil, nil.
type Store interface {
	ListReports(ctx context.Context, clientID string) ([]Report, error)
	GetReport(ctx context.Context, id string) (*Report, error)
	InsertReport(ctx context.Context, r Report) (*Report, error)
	// DeleteReport reports whether a row of that client was removed.
	DeleteReport(ctx context.Context, clientID, id string) (bool, error)
	InsertInsight(ctx context.Context, in Insight) (*Insight, error)
	LatestInsight(ctx context.Context, clientID, module string, from, to time.Time) (*Insight, error)
	// PurgeClient removes every report and insight of a client.
	PurgeClient(ctx context.Context, clientID string) (reports, insights int64, err error)
	// ClearCreator nulls created_by_id of a deleted user's reports.
	ClearCreator(ctx context.Context, userID string) (int64, error)
}

// ─── Upstream numbers (already computed by their owning services) ────

type SocialOverview struct {
	HasData             bool
	Followers           int64
	FollowersDelta      metrics.Delta
	Posts               int64
	PostsDelta          metrics.Delta
	EngagementRate      float64
	EngagementRateDelta metrics.Delta
	Reach               int64
	ReachDelta          metrics.Delta
}

type FollowersPoint struct {
	Date      time.Time
	Followers int64
}

type TopPost struct {
	ID          string
	Platform    string // INSTAGRAM | FACEBOOK | TIKTOK
	Username    string
	Caption     string
	MediaType   string
	ProductType string
	PublishedAt time.Time
	Likes       int64
	Comments    int64
	Engagements int64
}

// ContentType is contentType(p) from lib/metrics/social.ts.
func (p TopPost) ContentType() string {
	pt, mt := strings.ToUpper(p.ProductType), strings.ToUpper(p.MediaType)
	switch {
	case pt == "REELS":
		return "Reels"
	case pt == "STORY":
		return "Story"
	case mt == "VIDEO":
		return "Video"
	case mt == "CAROUSEL_ALBUM":
		return "Carousel"
	}
	return "Foto"
}

type SocialData struct {
	Overview SocialOverview
	// From the day before the range through its last day.
	DailyFollowers []FollowersPoint
	// Sorted by engagements, newest analytics order.
	TopPosts []TopPost
}

type SeoDailyPoint struct {
	Date        time.Time
	Clicks      *int64
	Impressions *int64
}

type TopQuery struct {
	Query       string
	Clicks      int64
	Impressions int64
	Ctr         float64
	Position    float64
}

type SeoData struct {
	HasData          bool
	Clicks           int64
	ClicksDelta      metrics.Delta
	Impressions      int64
	ImpressionsDelta metrics.Delta
	Ctr              float64
	CtrDelta         metrics.Delta
	Position         float64
	PositionDelta    metrics.Delta
	Health           *int32
	Daily            []SeoDailyPoint
	TopQueries       []TopQuery
}

type AdKpis struct {
	Spend   float64
	Results int64
	Cpr     float64
	Ctr     float64
	Cpm     float64
	Roas    *float64
}

type AdsDailyPoint struct {
	Date    time.Time
	Spend   float64
	Results int64
}

type Campaign struct {
	ID           string
	Name         string
	Objective    string
	Spend        float64
	Results      int64
	ResultType   *string
	Cpr          float64
	IsConversion bool
}

type AdsData struct {
	HasData      bool
	Kpis         AdKpis
	SpendDelta   metrics.Delta
	ResultsDelta metrics.Delta
	CprDelta     metrics.Delta
	Daily        []AdsDailyPoint
	Campaigns    []Campaign
}

// ReportData is analytics ReportDataService.GetReportData. A block is nil
// when its module was not requested.
type ReportData struct {
	Social *SocialData
	Seo    *SeoData
	Ads    *AdsData
}

type PlatformCount struct {
	Platform string
	Count    int
}

type PublishedTarget struct {
	Title       string
	Body        string
	Platform    string
	PublishedAt time.Time
	Engagements *int64
}

// PublishingSummary is summarizePublishing, computed by publishing.
type PublishingSummary struct {
	Posts       int
	PerPlatform []PlatformCount
	Failed      int
	// "engagement" | "recent"
	Mode string
	Top  []PublishedTarget
}

type RankChange struct {
	// up | down | flat | new | lost | none
	Kind  string
	Delta *int
}

type RankRow struct {
	Keyword  string
	Position *int
	Change   RankChange
	Volume   *int
}

type RankKpis struct {
	Visibility  float64
	AvgPosition *float64
	Top3        int
	Top10       int
}

type RankDeltas struct {
	Visibility  metrics.Delta
	AvgPosition metrics.Delta
	Top3        metrics.Delta
	Top10       metrics.Delta
}

// RankData carries the inputs of buildRankSection (seosuite).
type RankData struct {
	HasKeywords bool
	LatestDate  *string
	Kpis        RankKpis
	Deltas      RankDeltas
	Rows        []RankRow
}

// BacklinkData carries the inputs of buildBacklinkSection (seosuite).
type BacklinkData struct {
	HasSnapshot      bool
	Backlinks        int
	ReferringDomains int
	DomainRank       *int
	DofollowShare    float64
	NewBacklinks     int
	LostBacklinks    int
}

// PlatformName is the display name; anything unknown stays as given.
func PlatformName(p string) string {
	switch p {
	case "INSTAGRAM":
		return "Instagram"
	case "FACEBOOK":
		return "Facebook"
	case "TIKTOK":
		return "TikTok"
	}
	return p
}
