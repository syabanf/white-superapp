// Package domain holds the analytics entities, the repository ports and the
// pure metric rules ported from lib/metrics. Nothing here imports gRPC,
// Postgres or NATS.
package domain

import (
	"context"
	"time"
)

type Platform string

const (
	Instagram Platform = "INSTAGRAM"
	Facebook  Platform = "FACEBOOK"
	TikTok    Platform = "TIKTOK"
)

var Platforms = []Platform{Instagram, Facebook, TikTok}

func IsPlatform(p Platform) bool {
	return p == Instagram || p == Facebook || p == TikTok
}

type SeoDimension string

const (
	DimQuery   SeoDimension = "QUERY"
	DimPage    SeoDimension = "PAGE"
	DimCountry SeoDimension = "COUNTRY"
	DimDevice  SeoDimension = "DEVICE"
)

type AuditStrategy string

const (
	Mobile  AuditStrategy = "MOBILE"
	Desktop AuditStrategy = "DESKTOP"
)

type IssueSeverity string

const (
	SeverityError   IssueSeverity = "ERROR"
	SeverityWarning IssueSeverity = "WARNING"
	SeverityNotice  IssueSeverity = "NOTICE"
)

type JobStatus string

const (
	JobPending JobStatus = "PENDING"
	JobRunning JobStatus = "RUNNING"
	JobSuccess JobStatus = "SUCCESS"
	JobFailed  JobStatus = "FAILED"
)

type AuditCadence string

const (
	CadenceNone   AuditCadence = "NONE"
	CadenceWeekly AuditCadence = "WEEKLY"
	CadenceDaily  AuditCadence = "DAILY"
)

func IsCadence(c AuditCadence) bool {
	return c == CadenceNone || c == CadenceWeekly || c == CadenceDaily
}

// Sync job kinds written by this service.
const (
	KindSocialSnapshot = "SOCIAL_SNAPSHOT"
	KindSeoGSC         = "SEO_GSC"
	KindSeoGA4         = "SEO_GA4"
	KindSeoAudit       = "SEO_AUDIT"
	KindAdsInsights    = "ADS_INSIGHTS"
	KindAdsCsvImport   = "ADS_CSV_IMPORT"
)

// ─── Social ──────────────────────────────────────────────────

type SocialAccount struct {
	ID           string
	ClientID     string
	Platform     Platform
	ExternalID   string
	Username     string
	DisplayName  string
	AvatarURL    *string
	Biography    *string
	IsCompetitor bool
	ConnectionID *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SocialSnapshot struct {
	ID            string
	AccountID     string
	Date          time.Time
	Followers     int
	Following     *int
	MediaCount    int
	Reach         *int
	Impressions   *int
	ProfileViews  *int
	WebsiteClicks *int
}

// SnapshotPatch is applied when the (account, date) row already exists; nil
// fields keep the stored value.
type SnapshotPatch struct {
	Followers     *int
	Following     *int
	MediaCount    *int
	Reach         *int
	Impressions   *int
	ProfileViews  *int
	WebsiteClicks *int
}

type SocialPost struct {
	ID           string
	AccountID    string
	ExternalID   string
	Caption      string
	MediaType    string
	ProductType  *string
	Permalink    *string
	MediaURL     *string
	ThumbnailURL *string
	PublishedAt  time.Time
	Likes        int
	Comments     int
	Shares       int
	Saves        int
	Views        int
	Reach        int
	Impressions  int
	SyncedAt     time.Time
}

// PostPatch is applied when the (account, external id) row exists; nil keeps.
type PostPatch struct {
	Caption      *string
	MediaType    *string
	ProductType  *string
	Permalink    *string
	MediaURL     *string
	ThumbnailURL *string
	Likes        *int
	Comments     *int
	Shares       *int
	Saves        *int
	Views        *int
	Reach        *int
	Impressions  *int
}

// PostWithAccount is a post joined with its account for report tables.
type PostWithAccount struct {
	SocialPost
	Platform Platform
	Username string
}

// PostFilter selects own-account posts for other services.
type PostFilter struct {
	ClientID    string
	AccountIDs  []string
	ExternalIDs []string
	// From and ToExclusive bound published_at when set.
	From        *time.Time
	ToExclusive *time.Time
	Limit       int
}

// ProfilePatch updates the public profile of an account after a sync.
type ProfilePatch struct {
	Username    *string
	DisplayName string
	// SetBiography false leaves the stored biography untouched.
	SetBiography bool
	Biography    *string
	AvatarURL    *string
}

// ─── SEO ─────────────────────────────────────────────────────

type SeoProperty struct {
	ID            string
	ClientID      string
	SiteURL       string
	Ga4PropertyID *string
	ConnectionID  *string
	LocationCode  int
	LanguageCode  string
	AuditCadence  AuditCadence
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SeoDailyMetric struct {
	PropertyID  string
	Date        time.Time
	Clicks      int
	Impressions int
	CTR         float64
	Position    float64
}

type SeoDimensionMetric struct {
	PropertyID  string
	Date        time.Time
	Dimension   SeoDimension
	Key         string
	Clicks      int
	Impressions int
	CTR         float64
	Position    float64
}

type Ga4DailyMetric struct {
	PropertyID       string
	Date             time.Time
	Sessions         int
	OrganicSessions  int
	Users            int
	EngagedSessions  int
	Conversions      int
	AvgEngagementSec float64
}

type SeoAudit struct {
	ID            string
	PropertyID    string
	URL           string
	Strategy      AuditStrategy
	RunAt         time.Time
	Performance   int
	Seo           int
	Accessibility int
	BestPractices int
	LcpMs         *float64
	InpMs         *float64
	Cls           *float64
	FcpMs         *float64
	TtfbMs        *float64
	TbtMs         *float64
	SpeedIndexMs  *float64
	Raw           []byte
}

func (a SeoAudit) Scores() AuditScores {
	return AuditScores{Performance: a.Performance, Seo: a.Seo, Accessibility: a.Accessibility, BestPractices: a.BestPractices}
}

type SeoCrawl struct {
	ID           string
	PropertyID   string
	StartedAt    time.Time
	FinishedAt   *time.Time
	Status       JobStatus
	PagesCrawled int
	MaxPages     int
	Error        *string
}

type SeoCrawlPage struct {
	ID               string
	CrawlID          string
	URL              string
	StatusCode       int
	Title            *string
	MetaDescription  *string
	H1Count          int
	WordCount        int
	Canonical        *string
	Indexable        bool
	ImagesTotal      int
	ImagesMissingAlt int
	InternalLinks    int
	ExternalLinks    int
	LoadMs           *int
	HasJsonLd        bool
	HasHreflang      bool
}

type SeoIssue struct {
	ID       string
	CrawlID  string
	Severity IssueSeverity
	Code     string
	Message  string
	URL      string
}

// SeverityCounts of one crawl.
type SeverityCounts struct {
	Errors   int
	Warnings int
	Notices  int
}

// ─── Ads ─────────────────────────────────────────────────────

type AdAccount struct {
	ID           string
	ClientID     string
	ExternalID   string
	Name         string
	Currency     string
	ConnectionID *string
	LastSyncedAt *time.Time
	CreatedAt    time.Time
}

type AdCampaign struct {
	ID          string
	AccountID   string
	ExternalID  string
	Name        string
	Objective   string
	Status      string
	DailyBudget *float64
	StartTime   *time.Time
}

type AdSet struct {
	ID          string
	CampaignID  string
	ExternalID  string
	Name        string
	Status      string
	DailyBudget *float64
}

type Ad struct {
	ID           string
	AdSetID      string
	ExternalID   string
	Name         string
	Status       string
	ThumbnailURL *string
}

type AdDailyInsight struct {
	ID            string
	AccountID     string
	Date          time.Time
	CampaignID    *string
	AdSetID       *string
	AdID          *string
	Spend         float64
	Impressions   int
	Reach         int
	Clicks        int
	LinkClicks    int
	Results       int
	ResultType    *string
	PurchaseValue *float64
}

type AdDemographic struct {
	ID          string
	AccountID   string
	DateFrom    time.Time
	DateTo      time.Time
	CampaignID  *string
	Age         string
	Gender      string
	Spend       float64
	Impressions int
	Reach       int
	Results     int
}

// ─── Sync jobs ───────────────────────────────────────────────

type SyncJob struct {
	ID         string
	ClientID   string
	Kind       string
	Status     JobStatus
	StartedAt  time.Time
	FinishedAt *time.Time
	Message    *string
	Error      *string
}

// ─── Repository ports ────────────────────────────────────────

type SocialRepo interface {
	ListSocialAccounts(ctx context.Context, clientID string, includeCompetitors bool, platform *Platform) ([]SocialAccount, error)
	GetSocialAccount(ctx context.Context, id string) (*SocialAccount, error)
	FindCompetitor(ctx context.Context, clientID, username string) (*SocialAccount, error)
	InsertSocialAccount(ctx context.Context, a SocialAccount) (*SocialAccount, error)
	UpdateSocialAccountProfile(ctx context.Context, id string, p ProfilePatch) error
	DeleteSocialAccount(ctx context.Context, id string) (bool, error)

	LatestFollowers(ctx context.Context, accountIDs []string) (map[string]int, error)
	ListSnapshots(ctx context.Context, accountIDs []string, from, to time.Time) ([]SocialSnapshot, error)
	ListRecentSnapshots(ctx context.Context, accountID string, limit int) ([]SocialSnapshot, error)
	UpsertSnapshot(ctx context.Context, create SocialSnapshot, patch SnapshotPatch) error
	InsertSnapshotsIgnore(ctx context.Context, rows []SocialSnapshot) error

	// ListPosts returns posts published in [from, toExclusive), newest first.
	ListPosts(ctx context.Context, accountIDs []string, from, toExclusive time.Time) ([]SocialPost, error)
	ListOwnPosts(ctx context.Context, clientID string, from, toExclusive time.Time) ([]PostWithAccount, error)
	ListOwnPostsFiltered(ctx context.Context, f PostFilter) ([]SocialPost, error)
	UpsertPost(ctx context.Context, create SocialPost, patch PostPatch) error
	InsertPostsIgnore(ctx context.Context, rows []SocialPost) error
}

type SeoRepo interface {
	ListSeoProperties(ctx context.Context, clientID string) ([]SeoProperty, error)
	GetSeoProperty(ctx context.Context, id string) (*SeoProperty, error)
	InsertSeoProperty(ctx context.Context, p SeoProperty) (*SeoProperty, error)
	SetAuditCadence(ctx context.Context, id string, c AuditCadence) error
	DeleteSeoProperty(ctx context.Context, id string) (bool, error)

	ListSeoDaily(ctx context.Context, propertyIDs []string, from, to time.Time) ([]SeoDailyMetric, error)
	UpsertSeoDaily(ctx context.Context, rows []SeoDailyMetric) error
	SumSeoClicks(ctx context.Context, clientIDs []string, from, to time.Time) (int, error)
	ListSeoDimension(ctx context.Context, propertyIDs []string, dim SeoDimension, from, to time.Time) ([]SeoDimensionMetric, error)
	UpsertSeoDimension(ctx context.Context, rows []SeoDimensionMetric) error
	ListGa4Daily(ctx context.Context, propertyID string, from, to time.Time) ([]Ga4DailyMetric, error)
	UpsertGa4Daily(ctx context.Context, rows []Ga4DailyMetric) error

	LatestAudit(ctx context.Context, propertyIDs []string, strategy AuditStrategy) (*SeoAudit, error)
	LatestAuditAny(ctx context.Context, propertyID string) (*SeoAudit, error)
	ListMobileAuditHistory(ctx context.Context, propertyID string, limit int) ([]SeoAudit, error)
	InsertAudit(ctx context.Context, a SeoAudit) error

	// LatestCrawl with status nil returns the latest crawl of any status.
	LatestCrawl(ctx context.Context, propertyIDs []string, status *JobStatus) (*SeoCrawl, error)
	ListRecentSuccessCrawls(ctx context.Context, propertyID string, limit int) ([]SeoCrawl, error)
	InsertCrawl(ctx context.Context, c SeoCrawl) error
	UpdateCrawl(ctx context.Context, id string, status JobStatus, finishedAt time.Time, pagesCrawled int, errMsg *string) error
	InsertCrawlPages(ctx context.Context, rows []SeoCrawlPage) error
	InsertIssues(ctx context.Context, rows []SeoIssue) error
	ListCrawlPages(ctx context.Context, crawlID string) ([]SeoCrawlPage, error)
	ListIssues(ctx context.Context, crawlID string) ([]SeoIssue, error)
	CountIssues(ctx context.Context, crawlIDs []string) (map[string]SeverityCounts, error)
}

type AdsRepo interface {
	ListAdAccounts(ctx context.Context, clientID string) ([]AdAccount, error)
	GetAdAccount(ctx context.Context, id string) (*AdAccount, error)
	InsertAdAccount(ctx context.Context, a AdAccount) (*AdAccount, error)
	DeleteAdAccount(ctx context.Context, id string) (bool, error)
	TouchAdAccounts(ctx context.Context, ids []string, at time.Time) error

	ListCampaigns(ctx context.Context, accountIDs []string) ([]AdCampaign, error)
	CountCampaigns(ctx context.Context, accountIDs []string) (int, error)
	// UpsertCampaign returns the row id. fullUpdate also refreshes objective,
	// status, budget and start time on an existing row.
	UpsertCampaign(ctx context.Context, c AdCampaign, fullUpdate bool) (string, error)
	ListAdSets(ctx context.Context, accountIDs []string, campaignID *string) ([]AdSet, error)
	UpsertAdSet(ctx context.Context, s AdSet, fullUpdate bool) (string, error)
	ListAds(ctx context.Context, accountIDs []string, campaignID *string) ([]Ad, error)
	UpsertAd(ctx context.Context, a Ad, fullUpdate bool) (string, error)

	// ListInsights returns ad-level rows (ad_id set) in [from, to], date asc.
	ListInsights(ctx context.Context, accountIDs []string, from, to time.Time, campaignID *string) ([]AdDailyInsight, error)
	UpsertInsights(ctx context.Context, rows []AdDailyInsight) error
	SumAdSpend(ctx context.Context, clientIDs []string, from, to time.Time) (float64, error)
	ListDemographics(ctx context.Context, accountIDs []string, from, to time.Time, campaignID *string) ([]AdDemographic, error)
	UpsertDemographics(ctx context.Context, rows []AdDemographic) error
}

type JobRepo interface {
	InsertSyncJob(ctx context.Context, j SyncJob) (*SyncJob, error)
	UpdateSyncJob(ctx context.Context, id string, status JobStatus, finishedAt time.Time, message, errMsg *string) error
	LatestSyncJob(ctx context.Context, clientID string, kinds []string) (*SyncJob, error)
	ListSyncJobs(ctx context.Context, clientID string, limit int) ([]SyncJob, error)
	LatestSyncByClients(ctx context.Context, clientIDs []string) (map[string]time.Time, error)
	DeleteSyncJobsBefore(ctx context.Context, cutoff time.Time) (int64, error)
	// PurgeClient removes every row of a client across all tables.
	PurgeClient(ctx context.Context, clientID string) error
}

// Store bundles every repository and can run a function inside one
// transaction, handing it a Store bound to that transaction.
type Store interface {
	SocialRepo
	SeoRepo
	AdsRepo
	JobRepo
	WithTx(ctx context.Context, fn func(tx Store) error) error
}

// ErrUnique is returned by repositories on a unique-constraint violation so
// use cases can translate it into the right field message.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }

// ─── Other service ports ─────────────────────────────────────

// ClientAccess is what identity says a user may do with a client.
type ClientAccess struct {
	ClientID  string
	Slug      string
	Currency  string
	CanManage bool
}

// AccessChecker resolves client access through identity. A missing client
// is NOT_FOUND, a missing membership UNAUTHORIZED (as *apperr.Error).
type AccessChecker interface {
	Check(ctx context.Context, userID, clientID string) (*ClientAccess, error)
}

// Credentials of one integration as the integrations service reports them.
type Credentials struct {
	Configured bool
	PublicID   string
	Secret     string
}

const (
	IntegrationMeta   = "META"
	IntegrationGoogle = "GOOGLE"
	IntegrationApify  = "APIFY"
)

// Integrations is the slice of the integrations service this service uses.
type Integrations interface {
	Credentials(ctx context.Context, key string) (Credentials, error)
	// AccessToken returns "" with a nil error when the connection or the
	// provider app is missing; an expired token that cannot be refreshed is
	// an error.
	AccessToken(ctx context.Context, connectionID string) (string, error)
	// FindClientConnection returns the id of the first connection of the
	// provider for the client, or "".
	FindClientConnection(ctx context.Context, clientID, provider string) (string, error)
}
