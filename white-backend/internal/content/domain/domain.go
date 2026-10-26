// Package domain holds the content service entities (editorial plans,
// content items with their caption revisions and review trail, design
// tasks, the brand asset library and monthly content reports), the pure
// rules and the ports the use cases need. Nothing here imports gRPC,
// Postgres or NATS.
package domain

import (
	"context"
	"time"
)

type PlanStatus string

const (
	PlanDraft    PlanStatus = "DRAFT"
	PlanInReview PlanStatus = "IN_REVIEW"
	PlanApproved PlanStatus = "APPROVED"
)

type Platform string

const (
	PlatformInstagram Platform = "INSTAGRAM"
	PlatformFacebook  Platform = "FACEBOOK"
	PlatformTikTok    Platform = "TIKTOK"
)

var Platforms = []Platform{PlatformInstagram, PlatformFacebook, PlatformTikTok}

type Format string

const (
	FormatFeed     Format = "FEED"
	FormatCarousel Format = "CAROUSEL"
	FormatReel     Format = "REEL"
	FormatStory    Format = "STORY"
)

var Formats = []Format{FormatFeed, FormatCarousel, FormatReel, FormatStory}

type ItemStatus string

const (
	ItemIdea        ItemStatus = "IDEA"
	ItemCopywriting ItemStatus = "COPYWRITING"
	ItemDesign      ItemStatus = "DESIGN"
	ItemReview      ItemStatus = "REVIEW"
	ItemRevision    ItemStatus = "REVISION"
	ItemApproved    ItemStatus = "APPROVED"
	ItemScheduled   ItemStatus = "SCHEDULED"
	ItemPublished   ItemStatus = "PUBLISHED"
	ItemFailed      ItemStatus = "FAILED"
)

var ItemStatuses = []ItemStatus{ItemIdea, ItemCopywriting, ItemDesign, ItemReview, ItemRevision, ItemApproved, ItemScheduled, ItemPublished, ItemFailed}

type DesignStatus string

const (
	DesignTodo             DesignStatus = "TODO"
	DesignInProgress       DesignStatus = "IN_PROGRESS"
	DesignInReview         DesignStatus = "IN_REVIEW"
	DesignChangesRequested DesignStatus = "CHANGES_REQUESTED"
	DesignApproved         DesignStatus = "APPROVED"
)

type AssetKind string

const (
	AssetLogo           AssetKind = "LOGO"
	AssetBrandGuideline AssetKind = "BRAND_GUIDELINE"
	AssetFont           AssetKind = "FONT"
	AssetPhoto          AssetKind = "PHOTO"
	AssetTemplate       AssetKind = "TEMPLATE"
	AssetOther          AssetKind = "OTHER"
)

// Review decisions on items (comment kinds) and designs (feedback).
const (
	DecisionComment          = "COMMENT"
	DecisionChangesRequested = "CHANGES_REQUESTED"
	DecisionApproved         = "APPROVED"
)

// Role is what the caller may do with one marketing client (identity
// client access): VIEWER reads and comments, MANAGER and ADMIN write.
type Role string

const (
	RoleAdmin   Role = "ADMIN"
	RoleManager Role = "MANAGER"
	RoleViewer  Role = "VIEWER"
)

// Activity types written to item_activities.
const (
	ActCreated     = "CREATED"
	ActEdited      = "EDITED"
	ActMoved       = "MOVED"
	ActRevised     = "CAPTION_REVISED"
	ActScheduled   = "SCHEDULED"
	ActUnscheduled = "UNSCHEDULED"
	ActPublished   = "PUBLISHED"
	ActFailed      = "PUBLISH_FAILED"
	ActDesignDone  = "DESIGN_APPROVED"
)

// Deadline kinds for reminders.
const (
	DeadlineCopy   = "COPY"
	DeadlineDesign = "DESIGN"
	DeadlineReview = "REVIEW"
)

type Frequency struct {
	Platform Platform `json:"platform"`
	PerWeek  int32    `json:"perWeek"`
}

type Brief struct {
	Goals     string
	Pillars   []string
	Frequency []Frequency
	Tone      string
	Notes     string
}

type Plan struct {
	ID          string
	ClientID    string
	Month       Month
	Title       string
	Status      PlanStatus
	Brief       Brief
	AIModel     *string
	GeneratedAt *time.Time
	SubmittedBy *string
	SubmittedAt *time.Time
	ApprovedBy  *string
	ApprovedAt  *time.Time
	ReviewNote  *string
	CreatedBy   *string
	UpdatedBy   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Filled by list queries.
	ItemCount      int32
	PublishedCount int32
}

type PlanComment struct {
	ID        string
	PlanID    string
	UserID    *string
	Body      string
	CreatedAt time.Time
}

type Item struct {
	ID               string
	PlanID           string
	ClientID         string
	PlannedDate      time.Time
	PublishAt        *time.Time
	Platform         Platform
	Format           Format
	Pillar           string
	Title            string
	Hook             string
	Caption          string
	Hashtags         []string
	VisualBrief      string
	CopywriterID     *string
	DesignerID       *string
	CopyDue          *time.Time
	DesignDue        *time.Time
	ReviewDue        *time.Time
	Status           ItemStatus
	RevisionRound    int32
	CaptionVersion   int32
	CopyDoneAt       *time.Time
	DesignDoneAt     *time.Time
	ApprovedAt       *time.Time
	ApprovedBy       *string
	ScheduledBy      *string
	PublishingPostID *string
	PublishedAt      *time.Time
	Permalink        *string
	ExternalIDs      []string
	PublishError     *string
	CreatedBy        *string
	UpdatedBy        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	// Filled by reads: an approved design task with files exists.
	HasFinalDesign bool
}

type CaptionRevision struct {
	ID        string
	ItemID    string
	Version   int32
	Caption   string
	Hook      string
	Hashtags  []string
	Note      string
	CreatedBy *string
	CreatedAt time.Time
}

type ItemComment struct {
	ID        string
	ItemID    string
	UserID    *string
	Body      string
	Kind      string
	Round     int32
	CreatedAt time.Time
}

type ItemActivity struct {
	ID         string
	ItemID     string
	UserID     *string
	Type       string
	FromStatus *string
	ToStatus   *string
	Note       *string
	CreatedAt  time.Time
}

type DesignTask struct {
	ID             string
	ItemID         string
	ClientID       string
	Title          string
	Brief          string
	References     []string
	SizePreset     string
	Width          int32
	Height         int32
	DesignerID     *string
	DueDate        *time.Time
	Status         DesignStatus
	CurrentVersion int32
	CreatedBy      *string
	UpdatedBy      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type DesignFile struct {
	ID         string
	TaskID     string
	Version    int32
	SortOrder  int32
	FileKey    string
	FileName   string
	MimeType   string
	SizeBytes  int64
	UploadedBy *string
	CreatedAt  time.Time
}

type DesignFeedback struct {
	ID        string
	TaskID    string
	FileID    *string
	Version   int32
	UserID    *string
	Decision  string
	Body      string
	CreatedAt time.Time
}

type AssetFolder struct {
	ID         string
	ClientID   string
	ParentID   *string
	Name       string
	CreatedBy  *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	AssetCount int32
}

type Asset struct {
	ID             string
	ClientID       string
	FolderID       *string
	Kind           AssetKind
	Name           string
	Description    string
	Tags           []string
	CurrentVersion int32
	CreatedBy      *string
	UpdatedBy      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	// The current version's file, filled by reads.
	Current *AssetVersion
}

type AssetVersion struct {
	ID         string
	AssetID    string
	Version    int32
	FileKey    string
	FileName   string
	MimeType   string
	SizeBytes  int64
	Note       string
	UploadedBy *string
	CreatedAt  time.Time
}

type AssetFilter struct {
	Query    string
	Kind     AssetKind
	FolderID string
	Tag      string
}

type ItemFilter struct {
	PlanID     string
	Status     ItemStatus
	AssigneeID string
	// Month zero means any month.
	Month Month
}

type Report struct {
	ID        string
	ClientID  string
	Month     Month
	Data      ReportData
	Summary   string
	AIModel   string
	CreatedBy *string
	UpdatedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Reminder is one deadline notification candidate.
type Reminder struct {
	Item ItemRef
	Kind string
	Due  time.Time
}

// ItemRef is the slice of an item reminders need.
type ItemRef struct {
	ID           string
	ClientID     string
	Title        string
	Status       ItemStatus
	CopywriterID *string
	DesignerID   *string
	CopyDue      *time.Time
	DesignDue    *time.Time
	ReviewDue    *time.Time
}

// ─── Ports ────────────────────────────────────────────────────

type PlanRepo interface {
	InsertPlan(ctx context.Context, p Plan) error
	GetPlan(ctx context.Context, clientID, id string) (*Plan, error)
	GetPlanByMonth(ctx context.Context, clientID string, month Month) (*Plan, error)
	ListPlans(ctx context.Context, clientID string) ([]Plan, error)
	UpdatePlan(ctx context.Context, p Plan) error
	SoftDeletePlan(ctx context.Context, id, userID string) error
	InsertPlanComment(ctx context.Context, c PlanComment) error
	ListPlanComments(ctx context.Context, planID string) ([]PlanComment, error)
}

type ItemRepo interface {
	InsertItem(ctx context.Context, it Item) error
	GetItem(ctx context.Context, clientID, id string) (*Item, error)
	ListItems(ctx context.Context, clientID string, f ItemFilter) ([]Item, error)
	UpdateItem(ctx context.Context, it Item) error
	SoftDeleteItem(ctx context.Context, id, userID string) error
	// SoftDeleteIdeaItems removes the plan's untouched IDEA items (draft regeneration).
	SoftDeleteIdeaItems(ctx context.Context, planID, userID string) error
	ListScheduledDue(ctx context.Context, before time.Time, limit int32) ([]Item, error)
	// GetItemByPublishingPost is nil when no live item owns the post.
	GetItemByPublishingPost(ctx context.Context, clientID, postID string) (*Item, error)
	ListOpenItemRefs(ctx context.Context) ([]ItemRef, error)
	InsertRevision(ctx context.Context, r CaptionRevision) error
	ListRevisions(ctx context.Context, itemID string) ([]CaptionRevision, error)
	InsertItemComment(ctx context.Context, c ItemComment) error
	ListItemComments(ctx context.Context, itemID string) ([]ItemComment, error)
	InsertActivity(ctx context.Context, a ItemActivity) error
	ListActivities(ctx context.Context, itemID string) ([]ItemActivity, error)
	// BackdateItem sets created_at and updated_at (seeded history).
	BackdateItem(ctx context.Context, itemID string, created, updated time.Time) error
	// MarkReminder returns false when the reminder was already sent that day.
	MarkReminder(ctx context.Context, itemID, kind string, day time.Time) (bool, error)
}

type DesignRepo interface {
	InsertDesignTask(ctx context.Context, t DesignTask) error
	GetDesignTask(ctx context.Context, clientID, id string) (*DesignTask, error)
	ListDesignTasks(ctx context.Context, clientID, itemID, designerID string, status DesignStatus) ([]DesignTask, error)
	UpdateDesignTask(ctx context.Context, t DesignTask) error
	InsertDesignFile(ctx context.Context, f DesignFile) error
	GetDesignFile(ctx context.Context, clientID, id string) (*DesignFile, error)
	ListDesignFiles(ctx context.Context, taskID string) ([]DesignFile, error)
	// FinalDesignFiles are the latest version's files of the item's newest approved task.
	FinalDesignFiles(ctx context.Context, itemID string) ([]DesignFile, error)
	InsertDesignFeedback(ctx context.Context, f DesignFeedback) error
	ListDesignFeedback(ctx context.Context, taskID string) ([]DesignFeedback, error)
}

type AssetRepo interface {
	InsertFolder(ctx context.Context, f AssetFolder) error
	GetFolder(ctx context.Context, clientID, id string) (*AssetFolder, error)
	ListFolders(ctx context.Context, clientID string) ([]AssetFolder, error)
	RenameFolder(ctx context.Context, id, name string) error
	// DeleteFolder refuses (false) while the folder holds assets or subfolders.
	DeleteFolder(ctx context.Context, id string) (bool, error)
	InsertAsset(ctx context.Context, a Asset) error
	GetAsset(ctx context.Context, clientID, id string) (*Asset, error)
	SearchAssets(ctx context.Context, clientID string, f AssetFilter) ([]Asset, error)
	UpdateAsset(ctx context.Context, a Asset) error
	SoftDeleteAsset(ctx context.Context, id, userID string) error
	InsertAssetVersion(ctx context.Context, v AssetVersion) error
	ListAssetVersions(ctx context.Context, assetID string) ([]AssetVersion, error)
}

type ReportRepo interface {
	UpsertReport(ctx context.Context, r Report) (*Report, error)
	GetReport(ctx context.Context, clientID, id string) (*Report, error)
	ListReports(ctx context.Context, clientID string) ([]Report, error)
}

// DashboardRepo reads the rows the content dashboard aggregates and keeps
// the last known engagement per item.
type DashboardRepo interface {
	DashboardItems(ctx context.Context, clientID string, f DashboardFilter) ([]DashItem, error)
	DashboardDesignTasks(ctx context.Context, clientID string, f DashboardFilter) ([]DashDesignTask, error)
	// DashboardAssetVersions lists uploads with from <= created_at < to.
	DashboardAssetVersions(ctx context.Context, clientID string, from, to time.Time) ([]DashAssetVersion, error)
	ClientPillars(ctx context.Context, clientID string) ([]string, error)
	ItemMetrics(ctx context.Context, itemIDs []string) (map[string]PostMetrics, error)
	UpsertItemMetrics(ctx context.Context, itemID string, m PostMetrics) error
}

type MaintenanceRepo interface {
	// DeleteClientData removes every row of the client (white.client.deleted).
	DeleteClientData(ctx context.Context, clientID string) error
	// ForgetUser clears assignments to a deleted user.
	ForgetUser(ctx context.Context, userID string) error
}

// Store bundles every repository and can run a function inside one
// transaction, handing it a Store bound to that transaction.
type Store interface {
	PlanRepo
	ItemRepo
	DesignRepo
	AssetRepo
	ReportRepo
	DashboardRepo
	MaintenanceRepo
	WithTx(ctx context.Context, fn func(tx Store) error) error
}

// StoredFile is what the file store returns after a write.
type StoredFile struct {
	Key       string
	SizeBytes int64
}

// Files is the object store for design files and brand assets.
type Files interface {
	Put(ctx context.Context, clientID, ext string, data []byte) (*StoredFile, error)
	// Get returns nil for a missing or invalid key.
	Get(ctx context.Context, key string) ([]byte, error)
	DeleteClient(ctx context.Context, clientID string) error
}

// ErrUnique is returned by repositories on a unique-constraint violation.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }
