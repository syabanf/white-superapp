// Package domain holds the publishing entities, the pure rules ported from
// features/publishing/lib.ts and the ports the use cases need. Nothing here
// imports gRPC, Postgres or NATS.
package domain

import (
	"context"
	"time"
)

type PostStatus string

const (
	StatusDraft      PostStatus = "DRAFT"
	StatusInReview   PostStatus = "IN_REVIEW"
	StatusApproved   PostStatus = "APPROVED"
	StatusScheduled  PostStatus = "SCHEDULED"
	StatusPublishing PostStatus = "PUBLISHING"
	StatusPublished  PostStatus = "PUBLISHED"
	StatusFailed     PostStatus = "FAILED"
	StatusRejected   PostStatus = "REJECTED"
)

// StatusOrder is the presentation order; counts are reported for every entry.
var StatusOrder = []PostStatus{StatusDraft, StatusInReview, StatusApproved, StatusScheduled, StatusPublishing, StatusPublished, StatusFailed, StatusRejected}

type MediaKind string

const (
	MediaImage MediaKind = "IMAGE"
	MediaVideo MediaKind = "VIDEO"
)

type Platform string

const (
	PlatformInstagram Platform = "INSTAGRAM"
	PlatformFacebook  Platform = "FACEBOOK"
	PlatformTikTok    Platform = "TIKTOK"
)

var Platforms = []Platform{PlatformInstagram, PlatformFacebook, PlatformTikTok}

// ToPlatform coerces unknown values to INSTAGRAM, as isPlatform's fallback did.
func ToPlatform(v string) Platform {
	for _, p := range Platforms {
		if string(p) == v {
			return p
		}
	}
	return PlatformInstagram
}

// PlatformLabel is the prefix used in validation messages.
func PlatformLabel(p Platform) string {
	switch p {
	case PlatformFacebook:
		return "Facebook"
	case PlatformTikTok:
		return "TikTok"
	}
	return "Instagram"
}

type JobStatus string

const (
	JobPending JobStatus = "PENDING"
	JobRunning JobStatus = "RUNNING"
	JobSuccess JobStatus = "SUCCESS"
	JobFailed  JobStatus = "FAILED"
)

// PostRole is what the actor may do inside the publishing module.
type PostRole string

const (
	RoleAdmin   PostRole = "ADMIN"
	RoleManager PostRole = "MANAGER"
	RoleViewer  PostRole = "VIEWER"
)

// Activity types written to post_activities.
const (
	ActCreated       = "CREATED"
	ActEdited        = "EDITED"
	ActSubmitted     = "SUBMITTED"
	ActApproved      = "APPROVED"
	ActRejected      = "REJECTED"
	ActScheduled     = "SCHEDULED"
	ActUnscheduled   = "UNSCHEDULED"
	ActRescheduled   = "RESCHEDULED"
	ActBackToDraft   = "BACK_TO_DRAFT"
	ActPublished     = "PUBLISHED"
	ActPublishFailed = "PUBLISH_FAILED"
	ActPublishRetry  = "PUBLISH_RETRY"
)

const SyncKindPublish = "PUBLISH"

type MediaAsset struct {
	ID           string
	ClientID     string
	Kind         MediaKind
	URL          string
	ThumbnailURL *string
	Filename     string
	MimeType     string
	SizeBytes    int32
	Width        *int32
	Height       *int32
	DurationSec  *float64
	AltText      *string
	Tags         []string
	UploadedByID *string
	CreatedAt    time.Time
	UsageCount   int32
}

type ContentPost struct {
	ID           string
	ClientID     string
	Title        string
	Body         string
	Status       PostStatus
	ScheduledAt  *time.Time
	PublishedAt  *time.Time
	LinkURL      *string
	FirstComment *string
	Labels       []string
	UTM          UTM
	ReviewNote   *string
	CreatedByID  *string
	ApprovedByID *string
	ApprovedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Instant is the time a post is (or was) meant to go live: publishedAt,
// else scheduledAt, else createdAt.
func (p ContentPost) Instant() time.Time {
	switch {
	case p.PublishedAt != nil:
		return *p.PublishedAt
	case p.ScheduledAt != nil:
		return *p.ScheduledAt
	}
	return p.CreatedAt
}

type PostTarget struct {
	ID              string
	PostID          string
	SocialAccountID string
	Platform        Platform
	BodyOverride    *string
	Status          PostStatus
	ExternalID      *string
	Permalink       *string
	Error           *string
	Attempts        int32
	PublishedAt     *time.Time
}

type PostMedia struct {
	PostID  string
	AssetID string
	Order   int32
}

type PostComment struct {
	ID        string
	PostID    string
	UserID    *string
	Body      string
	CreatedAt time.Time
}

type PostActivity struct {
	ID     string
	PostID string
	UserID *string
	Type   string
	// Meta is a JSON object or nil.
	Meta      map[string]any
	CreatedAt time.Time
}

type SyncJob struct {
	ID         string
	ClientID   *string
	Kind       string
	Status     JobStatus
	StartedAt  time.Time
	FinishedAt *time.Time
	Message    *string
	Error      *string
}

// SocialAccount is the slice of an analytics social account publishing needs.
type SocialAccount struct {
	ID           string
	Platform     Platform
	ExternalID   string
	Username     string
	DisplayName  string
	AvatarURL    *string
	ConnectionID *string
}

// PostSummaryRow is a list/calendar row before presentation.
type PostSummaryRow struct {
	Post          ContentPost
	Platforms     []Platform
	FailedTargets int32
	ThumbnailURL  *string
}

// StatusPatch is what a transition writes besides the status. Each Set*
// flag applies the matching value (nil clears the column).
type StatusPatch struct {
	SetSchedule  bool
	ScheduledAt  *time.Time
	SetReview    bool
	ReviewNote   *string
	SetApproval  bool
	ApprovedByID *string
	ApprovedAt   *time.Time
	SetPublished bool
	PublishedAt  *time.Time
}

// TargetResult is written after one publish attempt on a target.
type TargetResult struct {
	Status      PostStatus
	ExternalID  *string
	Permalink   *string
	Error       *string
	Attempts    int32
	PublishedAt *time.Time
}

type PublishingStats struct {
	Scheduled7d int32
	InReview    int32
	Published30 int32
	Failed      int32
	Drafts      int32
}

// PublishedTarget feeds the report summary.
type PublishedTarget struct {
	PostID          string
	Title           string
	Body            string
	Platform        Platform
	SocialAccountID string
	ExternalID      *string
	PublishedAt     time.Time
	Engagements     *int64
}

// ─── Ports ────────────────────────────────────────────────────

type MediaRepo interface {
	ListMedia(ctx context.Context, clientID string) ([]MediaAsset, error)
	GetMedia(ctx context.Context, id string) (*MediaAsset, error)
	// ListMediaOwned returns the subset of ids that belong to the client.
	ListMediaOwned(ctx context.Context, clientID string, ids []string) ([]MediaAsset, error)
	InsertMedia(ctx context.Context, m MediaAsset) (*MediaAsset, error)
	UpdateMedia(ctx context.Context, id string, altText *string, tags []string) (*MediaAsset, error)
	UpdateMediaAlt(ctx context.Context, id string, altText *string) error
	DeleteMedia(ctx context.Context, id string) (bool, error)
	ListMediaURLsByClient(ctx context.Context, clientID string) ([]string, error)
	DeleteMediaByClient(ctx context.Context, clientID string) (int64, error)
}

type PostRepo interface {
	GetPost(ctx context.Context, id string) (*ContentPost, error)
	GetPostInClient(ctx context.Context, clientID, id string) (*ContentPost, error)
	InsertPost(ctx context.Context, p ContentPost) (*ContentPost, error)
	// UpdatePostContent writes the composer fields plus status and approval reset.
	UpdatePostContent(ctx context.Context, p ContentPost, clearApproval bool) (*ContentPost, error)
	UpdatePostStatus(ctx context.Context, id string, status PostStatus, patch StatusPatch) (*ContentPost, error)
	UpdatePostScheduledAt(ctx context.Context, id string, at *time.Time) error
	// ClaimPost flips the post to PUBLISHING when its status is one of `from`.
	ClaimPost(ctx context.Context, id string, from []PostStatus) (bool, error)
	ClaimDuePost(ctx context.Context, id string, now time.Time) (bool, error)
	ListDuePosts(ctx context.Context, now time.Time, limit int32) ([]ContentPost, error)
	ListPostsPublishing(ctx context.Context, ids []string) ([]ContentPost, error)
	DeletePost(ctx context.Context, id string) (bool, error)
	ListPosts(ctx context.Context, clientID string) ([]PostSummaryRow, error)
	CountPostsByStatus(ctx context.Context, clientID string) (map[PostStatus]int32, error)
	ListCalendarPosts(ctx context.Context, clientID string, from, to time.Time) ([]PostSummaryRow, error)
	GetPublishingStats(ctx context.Context, clientID string, now time.Time) (*PublishingStats, error)
	DeletePostsByClient(ctx context.Context, clientID string) (int64, error)
	// NullUserRefs clears created_by/approved_by/uploaded_by/user_id for a deleted user.
	NullUserRefs(ctx context.Context, userID string) error

	ListTargets(ctx context.Context, postID string) ([]PostTarget, error)
	UpsertTarget(ctx context.Context, t PostTarget) error
	DeleteTargetsNotIn(ctx context.Context, postID string, keepAccountIDs []string) error
	SetTargetsStatusExceptPublished(ctx context.Context, postID string, status PostStatus) error
	SetTargetsStatus(ctx context.Context, ids []string, status PostStatus) error
	UpdateTargetResult(ctx context.Context, id string, r TargetResult) error
	ListPublishedTargets(ctx context.Context, clientID string, from, to time.Time) ([]PublishedTarget, error)
	CountFailedTargets(ctx context.Context, clientID string, from, to time.Time) (int32, error)

	ReplacePostMedia(ctx context.Context, postID string, rows []PostMedia) error
	ListPostMedia(ctx context.Context, postID string) ([]MediaAsset, error)
	ListPostMediaRows(ctx context.Context, postID string) ([]PostMedia, error)

	InsertComment(ctx context.Context, c PostComment) (*PostComment, error)
	ListComments(ctx context.Context, postID string) ([]PostComment, error)
	InsertActivity(ctx context.Context, a PostActivity) error
	ListActivities(ctx context.Context, postID string) ([]PostActivity, error)
}

type SyncJobRepo interface {
	InsertSyncJob(ctx context.Context, j SyncJob) error
	ListSyncJobs(ctx context.Context, clientID string, limit int32) ([]SyncJob, error)
	DeleteSyncJobsBefore(ctx context.Context, cutoff time.Time) (int64, error)
	DeleteSyncJobsByClient(ctx context.Context, clientID string) (int64, error)
}

// Store bundles every repository and can run a function inside one
// transaction, handing it a Store bound to that transaction.
type Store interface {
	MediaRepo
	PostRepo
	SyncJobRepo
	WithTx(ctx context.Context, fn func(tx Store) error) error
}

// StoredObject is what the storage adapter returns after a write.
type StoredObject struct {
	Key         string
	URL         string
	SizeBytes   int64
	ContentType string
}

// Object is a stored file read back for serving.
type Object struct {
	Body        []byte
	ContentType string
	ETag        string
}

// Storage is the media object store (local disk today).
type Storage interface {
	Put(ctx context.Context, clientID, contentType string, data []byte) (*StoredObject, error)
	// Get returns nil for a missing or invalid key.
	Get(ctx context.Context, key string) (*Object, error)
	Delete(ctx context.Context, key string) error
	DeleteClient(ctx context.Context, clientID string) error
}

// Publisher sends one post to one account (real Meta Graph or the mock).
type Publisher interface {
	Publish(ctx context.Context, in PublishInput) (*PublishResult, error)
}

type PublishMedia struct {
	URL  string
	Kind MediaKind
}

type PublishInput struct {
	Platform Platform
	// Decrypted page or IG user token. Ignored by the mock.
	AccessToken string
	// IG user id, Facebook page id or TikTok open id.
	AccountExternalID string
	// Final caption: per-target override plus UTM link when relevant.
	Text         string
	Media        []PublishMedia
	LinkURL      string
	FirstComment string
}

type PublishResult struct {
	ExternalID string
	Permalink  *string
}

// ErrUnique is returned by repositories on a unique-constraint violation.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }
