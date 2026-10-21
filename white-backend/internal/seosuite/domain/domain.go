// Package domain holds the SEO suite entities, the repository ports and the
// pure rules (metric formulas, rank aggregation, text parsing). Nothing here
// imports gRPC, Postgres, NATS or other services.
package domain

import (
	"context"
	"time"
)

type Device string

const (
	DeviceDesktop Device = "DESKTOP"
	DeviceMobile  Device = "MOBILE"
)

type Intent string

const (
	IntentInformational Intent = "INFORMATIONAL"
	IntentNavigational  Intent = "NAVIGATIONAL"
	IntentCommercial    Intent = "COMMERCIAL"
	IntentTransactional Intent = "TRANSACTIONAL"
)

type ResearchMode string

const (
	ModeIdeas     ResearchMode = "ideas"
	ModeQuestions ResearchMode = "questions"
	ModeRelated   ResearchMode = "related"
)

// NormalizeMode mirrors lib.ts normalizeMode: anything unknown is "ideas".
func NormalizeMode(v string) ResearchMode {
	switch ResearchMode(v) {
	case ModeQuestions, ModeRelated:
		return ResearchMode(v)
	}
	return ModeIdeas
}

type JobStatus string

const (
	JobPending JobStatus = "PENDING"
	JobRunning JobStatus = "RUNNING"
	JobSuccess JobStatus = "SUCCESS"
	JobFailed  JobStatus = "FAILED"
)

// SyncJob kinds this service writes.
const (
	JobKindRanks     = "SEO_SUITE_RANKS"
	JobKindBacklinks = "SEO_SUITE_BACKLINKS"
	JobKindDomains   = "SEO_SUITE_DOMAINS"
	JobKindDaily     = "SEO_SUITE_DAILY"
)

var JobKinds = []string{JobKindRanks, JobKindBacklinks, JobKindDomains, JobKindDaily}

// Limits and timings carried over from the Next.js suite.
const (
	RefreshCooldown       = 10 * time.Minute
	MaxCompetitorDomains  = 5
	MaxKeywordsPerAdd     = 100
	MaxTags               = 10
	ResearchMaxAgeDays    = 7
	ResearchPurgeDays     = 30
	ToxicSpamScore        = 60
	BacklinkListLimit     = 300
	BacklinkLostThreshold = 5
	SerpBatch             = 20
	RetentionSnapshotDays = 760
	RetentionSyncJobDays  = 90
)

type TrackedKeyword struct {
	ID           string
	ClientID     string
	PropertyID   string
	Keyword      string
	LocationCode int32
	LanguageCode string
	Device       Device
	Tags         []string
	TargetURL    *string
	Volume       *int32
	Difficulty   *int32
	CPC          *float64
	Intent       *Intent
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TopResult is one entry of a top-10 list stored on a rank snapshot.
type TopResult struct {
	Domain   string `json:"domain"`
	Position int32  `json:"position"`
	URL      string `json:"url,omitempty"`
}

type RankSnapshot struct {
	ID           string
	ClientID     string
	KeywordID    string
	Date         time.Time
	Position     *int32
	URL          *string
	SerpFeatures []string
	TopResults   []TopResult
}

type KeywordResearch struct {
	ID           string
	ClientID     string
	Seed         string
	LocationCode int32
	LanguageCode string
	Provider     string
	Results      []KeywordIdea
	FetchedAt    time.Time
}

type Backlink struct {
	ID           string
	ClientID     string
	PropertyID   string
	SourceURL    string
	SourceDomain string
	TargetURL    string
	Anchor       string
	Dofollow     bool
	DomainRank   *int32
	SpamScore    *int32
	FirstSeen    time.Time
	LastSeen     time.Time
	IsLost       bool
}

// BacklinkKey is what refreshBacklinks needs from the existing rows.
type BacklinkKey struct {
	SourceURL string
	TargetURL string
	IsLost    bool
}

type BacklinkSnapshot struct {
	ID               string
	ClientID         string
	PropertyID       string
	Date             time.Time
	Backlinks        int32
	ReferringDomains int32
	Dofollow         int32
	Nofollow         int32
	NewBacklinks     int32
	LostBacklinks    int32
	DomainRank       *int32
	ToxicShare       *float64
}

type CompetitorDomain struct {
	ID         string
	ClientID   string
	PropertyID string
	Domain     string
	CreatedAt  time.Time
}

type DomainSnapshot struct {
	ID               string
	ClientID         string
	PropertyID       string
	Domain           string
	Date             time.Time
	OrganicKeywords  int32
	OrganicTraffic   int32
	OrganicCost      *float64
	Top3             int32
	Top10            int32
	Top100           int32
	Backlinks        *int32
	ReferringDomains *int32
	DomainRank       *int32
}

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

// ─── Ports ────────────────────────────────────────────────────

type KeywordRepo interface {
	ListKeywords(ctx context.Context, propertyID string) ([]TrackedKeyword, error)
	ListKeywordsByIDs(ctx context.Context, propertyID string, ids []string) ([]TrackedKeyword, error)
	ListKeywordIDs(ctx context.Context, propertyID string) ([]string, error)
	GetKeyword(ctx context.Context, id string) (*TrackedKeyword, error)
	// InsertKeywords skips rows that collide on (property, keyword, location, device).
	InsertKeywords(ctx context.Context, rows []TrackedKeyword) error
	UpdateKeywordTags(ctx context.Context, id string, tags []string) error
	DeleteKeyword(ctx context.Context, id string) (bool, error)
}

type RankRepo interface {
	ListRankSnapshots(ctx context.Context, keywordIDs []string, from, to time.Time) ([]RankSnapshot, error)
	// ListRankPositionsOn returns keyword id -> position for snapshots on one day.
	ListRankPositionsOn(ctx context.Context, keywordIDs []string, date time.Time) (map[string]*int32, error)
	UpsertRankSnapshots(ctx context.Context, rows []RankSnapshot) error
	DeleteRankSnapshotsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type ResearchRepo interface {
	GetLatestResearch(ctx context.Context, clientID, seed string, locationCode int32, languageCode string) (*KeywordResearch, error)
	InsertResearch(ctx context.Context, r KeywordResearch) error
	// DeleteOtherResearch removes older runs of the same key, keeping keepID.
	DeleteOtherResearch(ctx context.Context, clientID, seed string, locationCode int32, languageCode, keepID string) error
	DeleteResearchBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type BacklinkRepo interface {
	ListBacklinks(ctx context.Context, propertyID string) ([]Backlink, error)
	ListBacklinkKeys(ctx context.Context, propertyID string) ([]BacklinkKey, error)
	UpsertBacklinks(ctx context.Context, rows []Backlink) error
	UpsertBacklinkSnapshot(ctx context.Context, s BacklinkSnapshot) error
	GetLatestBacklinkSnapshot(ctx context.Context, propertyID string) (*BacklinkSnapshot, error)
	GetBacklinkSnapshotOnOrBefore(ctx context.Context, propertyID string, date time.Time) (*BacklinkSnapshot, error)
	ListBacklinkSnapshots(ctx context.Context, propertyID string, from, to time.Time) ([]BacklinkSnapshot, error)
	DeleteBacklinkSnapshotsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type DomainRepo interface {
	ListCompetitorDomains(ctx context.Context, propertyID string) ([]CompetitorDomain, error)
	GetCompetitorDomain(ctx context.Context, id string) (*CompetitorDomain, error)
	FindCompetitorDomain(ctx context.Context, propertyID, domain string) (*CompetitorDomain, error)
	CountCompetitorDomains(ctx context.Context, propertyID string) (int32, error)
	InsertCompetitorDomain(ctx context.Context, c CompetitorDomain) error
	// DeleteCompetitorDomain removes the row and its domain snapshots.
	DeleteCompetitorDomain(ctx context.Context, c CompetitorDomain) error
	UpsertDomainSnapshot(ctx context.Context, s DomainSnapshot) error
	ListDomainSnapshots(ctx context.Context, propertyID string, domains []string, from, to time.Time) ([]DomainSnapshot, error)
	// ListLatestDomainSnapshots returns the newest row per domain.
	ListLatestDomainSnapshots(ctx context.Context, propertyID string, domains []string) ([]DomainSnapshot, error)
	DeleteDomainSnapshotsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type JobRepo interface {
	GetLatestJob(ctx context.Context, clientID, kind string) (*SyncJob, error)
	InsertJob(ctx context.Context, j SyncJob) error
	FinishJob(ctx context.Context, id string, status JobStatus, finishedAt time.Time, message, errText *string) error
	ListJobs(ctx context.Context, clientID string, kinds []string, limit int32) ([]SyncJob, error)
	DeleteJobsBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type Store interface {
	KeywordRepo
	RankRepo
	ResearchRepo
	BacklinkRepo
	DomainRepo
	JobRepo
	// PurgeClient deletes every row of a client (white.client.deleted).
	PurgeClient(ctx context.Context, clientID string) error
	WithTx(ctx context.Context, fn func(tx Store) error) error
}

// ErrUnique is returned by repositories on a unique-constraint violation.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }
