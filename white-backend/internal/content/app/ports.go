// Package app holds the content use cases: editorial plans, the item
// workflow, design tasks, the asset library, scheduling through the
// publishing service, monthly reports, deadline reminders and maintenance.
package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
)

// Access is what identity says the caller may do with one client.
type Access struct {
	UserID   string
	Role     domain.Role
	ClientID string
	Slug     string
	Name     string
}

func (a Access) canWrite() bool { return a.Role != domain.RoleViewer }

type ClientInfo struct {
	ID   string
	Slug string
	Name string
}

type Audience int

const (
	AudienceNone Audience = iota
	AudienceClientApprovers
)

type NotifyInput struct {
	UserIDs      []string
	ClientID     string
	Audience     Audience
	ExceptUserID string
	Type         string
	Title        string
	Body         string
	Href         string
}

// IdentityClient is the slice of the identity service content uses.
type IdentityClient interface {
	// CheckAccess returns an UNAUTHORIZED or NOT_FOUND apperr when the user
	// may not see the client.
	CheckAccess(ctx context.Context, userID, clientID string) (*Access, error)
	// ClientInfo works without a user (workers); nil when unknown.
	ClientInfo(ctx context.Context, clientID string) (*ClientInfo, error)
	// UserName is "" when the user is unknown.
	UserName(ctx context.Context, userID string) string
	Notify(ctx context.Context, in NotifyInput) error
	// AsUser returns ctx carrying a short-lived access token for userID
	// (identity IssueServiceToken), so calls to other services act as that
	// user. Fails when the internal secret is not configured.
	AsUser(ctx context.Context, userID string) (context.Context, error)
}

// SocialAccount is the slice of an analytics social account content needs.
type SocialAccount struct {
	ID       string
	Platform domain.Platform
	Username string
}

// SocialPost is a post analytics synced from the platform.
type SocialPost struct {
	ExternalID  string
	Caption     string
	MediaType   string
	PublishedAt time.Time
	Metrics     domain.PostMetrics
}

// AnalyticsClient reads the client's own accounts and synced posts.
type AnalyticsClient interface {
	OwnAccounts(ctx context.Context, clientID string) ([]SocialAccount, error)
	// Posts returns synced posts; with externalIDs set only those, otherwise
	// the window fromDay..toDay (YYYY-MM-DD, inclusive).
	Posts(ctx context.Context, clientID string, externalIDs []string, fromDay, toDay string) ([]SocialPost, error)
}

// PostInput is the publishing post content creates for an item.
type PostInput struct {
	ClientID string
	// PostID empty creates; set updates the existing post (retry).
	PostID     string
	Title      string
	Body       string
	Format     domain.Format
	AccountIDs []string
	MediaIDs   []string
}

type PostTarget struct {
	ExternalID *string
	Permalink  *string
	Error      *string
}

// PostState is what publishing says about a post now.
type PostState struct {
	// DRAFT, IN_REVIEW, APPROVED, SCHEDULED, PUBLISHING, PUBLISHED, FAILED, REJECTED.
	Status      string
	PublishedAt *time.Time
	Targets     []PostTarget
}

type MediaFile struct {
	Filename string
	Content  []byte
}

// PublishingClient drives the publishing service as the calling user.
type PublishingClient interface {
	UploadMedia(ctx context.Context, clientID string, files []MediaFile) ([]string, error)
	SavePost(ctx context.Context, in PostInput) (string, error)
	ApprovePost(ctx context.Context, clientID, postID string) error
	SchedulePost(ctx context.Context, clientID, postID string, at time.Time) error
	GetPost(ctx context.Context, clientID, postID string) (*PostState, error)
}

// TopPost is a recent well-performing post given to the plan generator.
type TopPost struct {
	Caption     string `json:"caption"`
	MediaType   string `json:"mediaType"`
	PublishedAt string `json:"publishedAt"`
	Engagements int64  `json:"engagements"`
}

// PlanPrompt is everything the generator sees.
type PlanPrompt struct {
	ClientName string
	Month      domain.Month
	Brief      domain.Brief
	TopPosts   []TopPost
}

// ItemDraft is one generated item proposal.
type ItemDraft struct {
	PlannedDate time.Time
	Platform    domain.Platform
	Format      domain.Format
	Pillar      string
	Title       string
	Hook        string
	Caption     string
	Hashtags    []string
	VisualBrief string
}

// AI generates plan drafts and report summaries (OpenRouter or the mock).
// Model is the id stored with the result.
type AI interface {
	DraftPlan(ctx context.Context, in PlanPrompt) (items []ItemDraft, model string, err error)
	SummarizeReport(ctx context.Context, data domain.ReportData) (markdown, model string, err error)
}

// ReportRenderer turns a report into a PDF.
type ReportRenderer interface {
	Render(r domain.Report) ([]byte, error)
}

// Deps is shared by every use-case struct.
type Deps struct {
	Store      domain.Store
	Files      domain.Files
	Identity   IdentityClient
	Analytics  AnalyticsClient
	Publishing PublishingClient
	AI         AI
	Renderer   ReportRenderer
	Clock      clock.Clock
	Logger     *slog.Logger
}

func (d Deps) now() time.Time {
	if d.Clock == nil {
		return time.Now().UTC()
	}
	return d.Clock.Now().UTC()
}

func (d Deps) log() *slog.Logger {
	if d.Logger == nil {
		return slog.Default()
	}
	return d.Logger
}

// access resolves the caller's role; any identity refusal is UNAUTHORIZED.
func (d Deps) access(ctx context.Context, userID, clientID string) (*Access, error) {
	a, err := d.Identity.CheckAccess(ctx, userID, clientID)
	if err != nil {
		if apperr.Is(err, apperr.NotFound) || apperr.Is(err, apperr.Unauthorized) {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
		return nil, err
	}
	return a, nil
}

// manage is access plus write permission (MANAGER or ADMIN).
func (d Deps) manage(ctx context.Context, userID, clientID string) (*Access, error) {
	a, err := d.access(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	if !a.canWrite() {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return a, nil
}

func (d Deps) notify(ctx context.Context, in NotifyInput) {
	if len(in.UserIDs) == 0 && in.Audience == AudienceNone {
		return
	}
	if err := d.Identity.Notify(ctx, in); err != nil {
		d.log().Warn("notify failed", "type", in.Type, "err", err)
	}
}

// userNames resolves display names once per id.
func (d Deps) userNames(ctx context.Context, ids ...*string) map[string]string {
	out := map[string]string{}
	for _, id := range ids {
		if id == nil || *id == "" {
			continue
		}
		if _, ok := out[*id]; !ok {
			out[*id] = d.Identity.UserName(ctx, *id)
		}
	}
	return out
}

func strPtr(s string) *string { return &s }

// optID trims and maps "" to nil.
func optID(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func trim(s string) string { return strings.TrimSpace(s) }

func planHref(slug, planID string) string { return "/clients/" + slug + "/content/plans/" + planID }
func itemHref(slug, itemID string) string { return "/clients/" + slug + "/content/items/" + itemID }

// ensureFound maps a nil repository result to NOT_FOUND with msg:
// ensureFound(store.GetX(...))(msg).
func ensureFound[T any](v *T, err error) func(msg string) (*T, error) {
	return func(msg string) (*T, error) {
		if err != nil {
			return nil, err
		}
		if v == nil {
			return nil, apperr.New(apperr.NotFound, msg)
		}
		return v, nil
	}
}
