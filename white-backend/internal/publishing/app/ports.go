// Package app holds the publishing use cases: media library, composer and
// approvals, the publish flow, read models, report data and maintenance.
package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
)

// User-facing strings (features/publishing/strings.ts).
const (
	msgNotFound           = "Post tidak ditemukan."
	msgTransition         = "Aksi tidak diizinkan untuk status post saat ini."
	msgAccountNotOwn      = "Salah satu akun bukan milik klien ini."
	msgMediaNotOwn        = "Salah satu media bukan milik klien ini."
	msgFixIssues          = "Perbaiki dulu isu berikut:"
	msgNoFailedTargets    = "Tidak ada target yang gagal."
	msgNothingToPublish   = "Tidak ada target untuk diterbitkan."
	msgCannotEditStatus   = "Post dengan status ini tidak bisa diubah. Batalkan jadwal atau duplikat post."
	msgSelfApprove        = "Penyetuju harus orang lain (kecuali Admin)."
	msgMediaInUse         = "Media masih dipakai oleh post. Lepaskan dari post terlebih dahulu."
	msgFileTooLarge       = "Ukuran file melebihi 25 MB."
	msgFileTypeNotAllowed = "Jenis file tidak didukung."
	msgInvalidHTTPSURL    = "URL harus diawali https://"
	msgReviewBody         = "Buka untuk menyetujui atau menolak."
	msgAccountMissing     = "Akun sosial tidak ditemukan."
)

func notifReview(title string) string    { return "Post menunggu review: " + title }
func notifApproved(title string) string  { return "Post disetujui: " + title }
func notifRejected(title string) string  { return "Post ditolak: " + title }
func notifPublished(title string) string { return "Post terbit: " + title }
func notifFailed(title string) string    { return "Post gagal terbit: " + title }

// Notification types.
const (
	NotifPostReview    = "POST_REVIEW"
	NotifPostApproved  = "POST_APPROVED"
	NotifPostRejected  = "POST_REJECTED"
	NotifPostPublished = "POST_PUBLISHED"
	NotifPostFailed    = "POST_FAILED"
)

// Access is what identity says the caller may do with one client
// (access.ts resolvePublishingAccess).
type Access struct {
	UserID   string
	Role     domain.PostRole
	ClientID string
	Slug     string
	Timezone string
}

func (a Access) canWrite() bool { return a.Role != domain.RoleViewer }

type ClientInfo struct {
	ID       string
	Slug     string
	Timezone string
}

type Audience int

const (
	AudienceNone Audience = iota
	AudienceClientAll
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

// IdentityClient is the slice of the identity service publishing uses.
type IdentityClient interface {
	// CheckAccess returns an UNAUTHORIZED or NOT_FOUND apperr when the user
	// may not see the client.
	CheckAccess(ctx context.Context, userID, clientID string) (*Access, error)
	// ClientInfo resolves slug and timezone without a user (cron paths); nil when unknown.
	ClientInfo(ctx context.Context, clientID string) (*ClientInfo, error)
	// UserName is "" when the user is unknown.
	UserName(ctx context.Context, userID string) string
	Notify(ctx context.Context, in NotifyInput) error
}

// IntegrationsClient answers whether the Meta app is configured and hands
// out decrypted page tokens.
type IntegrationsClient interface {
	MetaConfigured(ctx context.Context) bool
	AccessToken(ctx context.Context, connectionID string) (string, error)
}

// AnalyticsClient lists the client's own social accounts (never competitors).
type AnalyticsClient interface {
	OwnAccounts(ctx context.Context, clientID string) ([]domain.SocialAccount, error)
	// PostingHeatmap returns engagement cells for the client's synced posts,
	// or nil when analytics exposes no post lookup yet.
	PostingHeatmap(ctx context.Context, clientID, timeZone string) ([]domain.HeatCell, error)
}

// Deps is shared by every use-case struct.
type Deps struct {
	Store        domain.Store
	Storage      domain.Storage
	Identity     IdentityClient
	Integrations IntegrationsClient
	Analytics    AnalyticsClient
	Real         domain.Publisher
	Mock         domain.Publisher
	Clock        clock.Clock
	Logger       *slog.Logger
}

func (d Deps) now() time.Time {
	if d.Clock == nil {
		return time.Now().UTC()
	}
	return d.Clock.Now()
}

func (d Deps) log() *slog.Logger {
	if d.Logger == nil {
		return slog.Default()
	}
	return d.Logger
}

// access resolves the caller's role; any identity failure is UNAUTHORIZED,
// matching the old "no access -> UNAUTHORIZED" branch of the actions.
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

func postHref(slug, postID string) string { return "/clients/" + slug + "/publish/posts/" + postID }

func strPtr(s string) *string { return &s }

// trimToNil trims and maps "" to nil (SQL NULL).
func trimToNil(s *string) *string {
	if s == nil {
		return nil
	}
	v := trim(*s)
	if v == "" {
		return nil
	}
	return &v
}

func dedupe(xs []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	return out
}

func accountsByID(accounts []domain.SocialAccount) map[string]domain.SocialAccount {
	m := make(map[string]domain.SocialAccount, len(accounts))
	for _, a := range accounts {
		m[a.ID] = a
	}
	return m
}

// draftInput builds the validation input from stored targets and media.
func draftInput(post domain.ContentPost, targets []domain.PostTarget, media []domain.MediaAsset) domain.DraftInput {
	in := domain.DraftInput{Body: post.Body, ScheduledAt: post.ScheduledAt, LinkURL: post.LinkURL, FirstComment: post.FirstComment}
	for _, t := range targets {
		in.Targets = append(in.Targets, domain.DraftTarget{Platform: t.Platform, BodyOverride: t.BodyOverride})
	}
	for _, m := range media {
		in.Media = append(in.Media, m.Kind)
	}
	return in
}
