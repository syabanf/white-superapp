package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/adapter/providers/publisher"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/app"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

const (
	clientID = "client_kopi"
	adminID  = "user_admin"
	mgrID    = "user_manager"
	mgr2ID   = "user_manager2"
	viewerID = "user_viewer"
	igAcc    = "acc_ig"
	fbAcc    = "acc_fb"
)

var (
	pngBytes  = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R'}
	jpegBytes = []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0x10, 'J', 'F', 'I', 'F', 0}
)

// ─── fakes ───────────────────────────────────────────────────

type fakeIdentity struct {
	mu    sync.Mutex
	roles map[string]domain.PostRole
	names map[string]string
	sent  []app.NotifyInput
}

func (f *fakeIdentity) CheckAccess(_ context.Context, userID, cid string) (*app.Access, error) {
	role, ok := f.roles[userID]
	if !ok || cid != clientID {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return &app.Access{UserID: userID, Role: role, ClientID: cid, Slug: "kopi", Timezone: "Asia/Jakarta"}, nil
}

func (f *fakeIdentity) ClientInfo(context.Context, string) (*app.ClientInfo, error) {
	return &app.ClientInfo{ID: clientID, Slug: "kopi", Timezone: "Asia/Jakarta"}, nil
}

func (f *fakeIdentity) UserName(_ context.Context, id string) string { return f.names[id] }

func (f *fakeIdentity) Notify(_ context.Context, in app.NotifyInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, in)
	return nil
}

func (f *fakeIdentity) last(t *testing.T) app.NotifyInput {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) == 0 {
		t.Fatal("no notification sent")
	}
	return f.sent[len(f.sent)-1]
}

type fakeIntegrations struct{ configured bool }

func (f fakeIntegrations) MetaConfigured(context.Context) bool                 { return f.configured }
func (f fakeIntegrations) AccessToken(context.Context, string) (string, error) { return "", nil }

type fakeAnalytics struct{ accounts []domain.SocialAccount }

func (f fakeAnalytics) OwnAccounts(context.Context, string) ([]domain.SocialAccount, error) {
	return f.accounts, nil
}

func (f fakeAnalytics) SyncedPosts(context.Context, string, []string, int32) ([]domain.SyncedPost, error) {
	return nil, nil
}

// fakeEvents records published outcome events.
type fakeEvents struct {
	mu   sync.Mutex
	sent []events.PublishPostPayload
	subj []string
}

func (f *fakeEvents) Publish(_ context.Context, subject string, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subj = append(f.subj, subject)
	f.sent = append(f.sent, payload.(events.PublishPostPayload))
	return nil
}

func (f *fakeEvents) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

func (f *fakeEvents) last(t *testing.T) (string, events.PublishPostPayload) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) == 0 {
		t.Fatal("no event published")
	}
	return f.subj[len(f.subj)-1], f.sent[len(f.sent)-1]
}

// ─── fixture ─────────────────────────────────────────────────

type fixture struct {
	store       *postgres.Store
	media       *app.Media
	posts       *app.Posts
	queries     *app.Queries
	publishing  *app.Publishing
	maintenance *app.Maintenance
	reports     *app.Reports
	identity    *fakeIdentity
	events      *fakeEvents
	clock       *clock.Fixed
	uploads     string
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := pg.TestPool(t, publishing.Migrations(), "migrations", publishing.Schema)
	pg.TruncateSchema(t, pool, publishing.Schema)
	store := postgres.NewStore(pool)
	clk := &clock.Fixed{T: time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)}
	id := &fakeIdentity{
		roles: map[string]domain.PostRole{adminID: domain.RoleAdmin, mgrID: domain.RoleManager, mgr2ID: domain.RoleManager, viewerID: domain.RoleViewer},
		names: map[string]string{adminID: "Admin", mgrID: "Mira", mgr2ID: "Maya", viewerID: "Vio"},
	}
	uploads := t.TempDir()
	ev := &fakeEvents{}
	deps := app.Deps{
		Events:       ev,
		Store:        store,
		Storage:      storage.NewLocal(uploads, "https://app.white.id"),
		Identity:     id,
		Integrations: fakeIntegrations{},
		Analytics: fakeAnalytics{accounts: []domain.SocialAccount{
			{ID: igAcc, Platform: domain.PlatformInstagram, ExternalID: "178", Username: "kopi.id", DisplayName: "Kopi"},
			{ID: fbAcc, Platform: domain.PlatformFacebook, ExternalID: "page1", Username: "kopipage", DisplayName: "Kopi Page"},
		}},
		Real:  publisher.NewReal(),
		Mock:  publisher.Mock{},
		Clock: clk,
	}
	pub := app.NewPublishing(deps)
	return &fixture{
		store: store, media: app.NewMedia(deps), posts: app.NewPosts(deps, pub), queries: app.NewQueries(deps),
		publishing: pub, maintenance: app.NewMaintenance(deps), reports: app.NewReports(deps), identity: id, events: ev, clock: clk, uploads: uploads,
	}
}

func code(t *testing.T, err error, want apperr.Code) *apperr.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got nil", want)
	}
	e := apperr.As(err)
	if e.Code != want {
		t.Fatalf("expected %s, got %s (%s)", want, e.Code, e.Message)
	}
	return e
}

// must is used as must(call())(t): it fails the test on a non-nil error.
func must[T any](v T, err error) func(*testing.T) T {
	return func(t *testing.T) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func (f *fixture) uploadPNG(t *testing.T) domain.MediaAsset {
	t.Helper()
	assets := must(f.media.Upload(context.Background(), mgrID, clientID, []app.UploadFile{{Filename: "a.png", Content: pngBytes}}))(t)
	return assets[0]
}

func (f *fixture) post(t *testing.T, id string) domain.ContentPost {
	t.Helper()
	p := must(f.store.GetPost(context.Background(), id))(t)
	if p == nil {
		t.Fatalf("post %s missing", id)
	}
	return *p
}

func (f *fixture) activities(t *testing.T, id string) []domain.PostActivity {
	t.Helper()
	return must(f.store.ListActivities(context.Background(), id))(t)
}

// ─── tests ───────────────────────────────────────────────────

func TestIntegrationHappyPathToPublished(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	asset := f.uploadPNG(t)

	saved := must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{
		Title: "Promo kopi", Body: "Halo #promo", Targets: []app.TargetInput{{SocialAccountID: igAcc}, {SocialAccountID: fbAcc}},
		MediaIDs: []string{asset.ID}, MediaAlt: map[string]string{asset.ID: " Secangkir kopi "},
		LinkURL: stringp("https://kopi.id/promo"), UTM: domain.UTM{"source": "instagram", "medium": "social"}, Labels: []string{" promo ", "promo", "sep"},
	}))(t)
	if saved.Status != domain.StatusDraft {
		t.Fatal(saved.Status)
	}
	p := f.post(t, saved.ID)
	if len(p.Labels) != 2 || p.UTM["source"] != "instagram" || *p.CreatedByID != mgrID {
		t.Fatalf("%+v", p)
	}
	if alt := must(f.store.GetMedia(ctx, asset.ID))(t); alt.AltText == nil || *alt.AltText != "Secangkir kopi" || alt.UsageCount != 1 {
		t.Fatalf("%+v", alt)
	}
	targets := must(f.store.ListTargets(ctx, saved.ID))(t)
	if len(targets) != 2 || targets[0].Platform != domain.PlatformFacebook || targets[1].Platform != domain.PlatformInstagram {
		t.Fatalf("%+v", targets)
	}

	// Unknown account and foreign media are refused.
	code(t, errOf(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{Targets: []app.TargetInput{{SocialAccountID: "acc_competitor"}}})), apperr.Validation)
	code(t, errOf(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{MediaIDs: []string{"media_other"}})), apperr.Validation)
	code(t, errOf(f.posts.Save(ctx, viewerID, clientID, app.SaveInput{Body: "x"})), apperr.Unauthorized)

	// Submit: approvers notified, actor excluded.
	submitted := must(f.posts.SubmitForReview(ctx, mgrID, clientID, saved.ID))(t)
	if submitted.Status != domain.StatusInReview {
		t.Fatal(submitted.Status)
	}
	n := f.identity.last(t)
	if n.Type != app.NotifPostReview || n.Audience != app.AudienceClientApprovers || n.ExceptUserID != mgrID || n.Title != "Post menunggu review: Promo kopi" || n.Href != "/clients/kopi/publish/posts/"+saved.ID {
		t.Fatalf("%+v", n)
	}
	// Manager author cannot approve own post; admin can.
	e := code(t, errOf(f.posts.Approve(ctx, mgrID, clientID, saved.ID)), apperr.Unauthorized)
	if e.Message != "Penyetuju harus orang lain (kecuali Admin)." {
		t.Fatal(e.Message)
	}
	code(t, errOf(f.posts.Approve(ctx, viewerID, clientID, saved.ID)), apperr.Unauthorized)
	approved := must(f.posts.Approve(ctx, adminID, clientID, saved.ID))(t)
	if approved.Status != domain.StatusApproved {
		t.Fatal(approved.Status)
	}
	p = f.post(t, saved.ID)
	if p.ApprovedByID == nil || *p.ApprovedByID != adminID || p.ApprovedAt == nil {
		t.Fatalf("%+v", p)
	}
	n = f.identity.last(t)
	if n.Type != app.NotifPostApproved || n.UserIDs[0] != mgrID {
		t.Fatalf("%+v", n)
	}

	// Schedule: lead time enforced.
	soon := f.clock.T.Add(2 * time.Minute)
	e = code(t, errOf(f.posts.Schedule(ctx, mgrID, clientID, saved.ID, soon)), apperr.Validation)
	if !strings.Contains(e.Message, "Jadwal minimal 5 menit dari sekarang.") {
		t.Fatal(e.Message)
	}
	at := f.clock.T.Add(time.Hour)
	if r := must(f.posts.Schedule(ctx, mgrID, clientID, saved.ID, at))(t); r.Status != domain.StatusScheduled {
		t.Fatal(r.Status)
	}
	// Editing a scheduled post is refused; detail exposes names and targets.
	e = code(t, errOf(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{ID: saved.ID, Body: "edit"})), apperr.Validation)
	if e.Message != "Post dengan status ini tidak bisa diubah. Batalkan jadwal atau duplikat post." {
		t.Fatal(e.Message)
	}
	detail := must(f.queries.GetPost(ctx, viewerID, clientID, saved.ID))(t)
	if *detail.CreatedBy != "Mira" || *detail.ApprovedBy != "Admin" || len(detail.Targets) != 2 || detail.Targets[1].Username != "kopi.id" || len(detail.Media) != 1 {
		t.Fatalf("%+v", detail)
	}

	// Not due yet: nothing happens.
	run := must(f.publishing.PublishDuePosts(ctx, f.clock.T, 20))(t)
	if run.Attempted != 0 {
		t.Fatalf("%+v", run)
	}
	// Due: published through the mock.
	f.clock.T = at.Add(time.Minute)
	run = must(f.publishing.PublishDuePosts(ctx, f.clock.T, 20))(t)
	if run.Attempted != 1 || run.Published != 1 || run.Failed != 0 {
		t.Fatalf("%+v", run)
	}
	p = f.post(t, saved.ID)
	if p.Status != domain.StatusPublished || p.PublishedAt == nil || !p.PublishedAt.Equal(f.clock.T) {
		t.Fatalf("%+v", p)
	}
	for _, tg := range must(f.store.ListTargets(ctx, saved.ID))(t) {
		if tg.Status != domain.StatusPublished || tg.Attempts != 1 || tg.ExternalID == nil || tg.Permalink == nil || !strings.HasPrefix(*tg.ExternalID, "mock_") {
			t.Fatalf("%+v", tg)
		}
	}
	if subj, ev := f.events.last(t); f.events.count() != 1 || subj != events.PublishingPostPublished || ev.PostID != saved.ID || ev.ClientID != clientID ||
		ev.Status != "PUBLISHED" || !strings.HasPrefix(ev.ExternalID, "mock_") || ev.Permalink == "" || ev.Error != "" || !ev.At.Equal(f.clock.T) {
		t.Fatalf("event %s %+v", subj, ev)
	}
	acts := f.activities(t, saved.ID)
	if acts[0].Type != domain.ActPublished || acts[0].Meta["mock"] != true || acts[0].Meta["published"] != float64(2) || acts[0].UserID != nil {
		t.Fatalf("%+v", acts[0])
	}
	jobs := must(f.store.ListSyncJobs(ctx, clientID, 10))(t)
	if len(jobs) != 1 || jobs[0].Kind != "PUBLISH" || jobs[0].Status != domain.JobSuccess || *jobs[0].Message != "1 terbit · 0 gagal · 0 diulang (mock)" {
		t.Fatalf("%+v", jobs)
	}
	n = f.identity.last(t)
	if n.Type != app.NotifPostPublished || n.Audience != app.AudienceClientAll || n.Title != "Post terbit: Promo kopi" {
		t.Fatalf("%+v", n)
	}

	// Read models.
	list := must(f.queries.ListPosts(ctx, viewerID, clientID, domain.TabPublished))(t)
	if len(list.Rows) != 1 || list.Counts[domain.StatusPublished] != 1 || *list.Rows[0].Author != "Mira" || len(list.Rows[0].Platforms) != 2 || list.Rows[0].ThumbnailURL == nil {
		t.Fatalf("%+v", list)
	}
	if empty := must(f.queries.ListPosts(ctx, viewerID, clientID, domain.TabDraft))(t); len(empty.Rows) != 0 {
		t.Fatal("draft tab should be empty")
	}
	cal := must(f.queries.Calendar(ctx, viewerID, clientID, "2026-10-21", "2026-10-21"))(t)
	if len(cal.Posts) != 1 || cal.From.Format(time.RFC3339) != "2026-10-20T17:00:00Z" {
		t.Fatalf("%+v", cal)
	}
	stats := must(f.queries.Stats(ctx, viewerID, clientID))(t)
	if stats.Published30 != 1 || stats.Scheduled7d != 0 {
		t.Fatalf("%+v", stats)
	}
	sum := must(f.reports.PublishingSummary(ctx, clientID, dates.Range{From: dates.UTC(2026, 10, 1), To: dates.UTC(2026, 10, 31)}))(t)
	if sum.Posts != 1 || sum.Mode != "recent" || len(sum.PerPlatform) != 2 || len(sum.Top) != 2 {
		t.Fatalf("%+v", sum)
	}
	best := must(f.queries.BestTimes(ctx, viewerID, clientID, ""))(t)
	if len(best) != 3 || best[0].FromHistory || best[0].Day != 0 || best[0].Hour != 11 || domain.ToZoned(best[0].At, "Asia/Jakarta").Time != "11:00" {
		t.Fatalf("%+v", best)
	}
	// Published posts cannot be deleted by viewers, but can by managers.
	code(t, f.posts.Delete(ctx, viewerID, clientID, saved.ID), apperr.Unauthorized)
	if err := f.posts.Delete(ctx, mgrID, clientID, saved.ID); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationFailureAndRetry(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	saved := must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{Title: "Gagal", Body: "teks [fail]", Targets: []app.TargetInput{{SocialAccountID: fbAcc}}}))(t)
	must(f.posts.SubmitForReview(ctx, mgrID, clientID, saved.ID))(t)
	must(f.posts.Approve(ctx, mgr2ID, clientID, saved.ID))(t)

	// Manual publish fails at once (no automatic retry).
	res := must(f.posts.PublishNow(ctx, mgrID, clientID, saved.ID))(t)
	if res.Status != domain.StatusFailed || res.Failed != 1 || len(res.Errors) != 1 || res.Errors[0].Target != "facebook @kopipage" || !strings.Contains(res.Errors[0].Error, "Terjadi kesalahan tak terduga. (Publikasi gagal") {
		t.Fatalf("%+v", res)
	}
	n := f.identity.last(t)
	if n.Type != app.NotifPostFailed || n.Body != res.Errors[0].Error {
		t.Fatalf("%+v", n)
	}
	if subj, ev := f.events.last(t); subj != events.PublishingPostFailed || ev.PostID != saved.ID || ev.Status != "FAILED" || ev.Error != res.Errors[0].Error || ev.ExternalID != "" {
		t.Fatalf("event %s %+v", subj, ev)
	}
	jobs := must(f.store.ListSyncJobs(ctx, clientID, 10))(t)
	if len(jobs) != 1 || jobs[0].Status != domain.JobFailed || *jobs[0].Message != "Gagal: 0 terbit, 1 gagal (mock)" || jobs[0].Error == nil {
		t.Fatalf("%+v", jobs[0])
	}
	acts := f.activities(t, saved.ID)
	if acts[0].Type != domain.ActPublishFailed || acts[0].Meta["manual"] != true || *acts[0].UserID != mgrID {
		t.Fatalf("%+v", acts[0])
	}

	// Scheduled retries: SCHEDULED until attempts reach 3, then FAILED.
	at := f.clock.T.Add(10 * time.Minute)
	must(f.posts.Schedule(ctx, mgrID, clientID, saved.ID, at))(t)
	f.clock.T = at
	run := must(f.publishing.PublishDuePosts(ctx, at, 20))(t)
	if run.Attempted != 1 || run.Failed != 0 || f.post(t, saved.ID).Status != domain.StatusScheduled {
		t.Fatalf("%+v %s", run, f.post(t, saved.ID).Status)
	}
	if f.events.count() != 1 {
		t.Fatal("a retry-scheduled post must not emit an outcome event")
	}
	if acts := f.activities(t, saved.ID); acts[0].Type != domain.ActPublishRetry {
		t.Fatal(acts[0].Type)
	}
	run = must(f.publishing.PublishDuePosts(ctx, at.Add(5*time.Minute), 20))(t)
	if run.Failed != 1 || f.post(t, saved.ID).Status != domain.StatusFailed {
		t.Fatalf("%+v", run)
	}
	tg := must(f.store.ListTargets(ctx, saved.ID))(t)[0]
	if tg.Attempts != 3 || tg.Status != domain.StatusFailed {
		t.Fatalf("%+v", tg)
	}
	jobs = must(f.store.ListSyncJobs(ctx, clientID, 10))(t)
	if *jobs[0].Message != "0 terbit · 1 gagal · 0 diulang (mock)" || jobs[0].Status != domain.JobFailed {
		t.Fatalf("%+v", jobs[0])
	}
	// Retry is only for FAILED posts; a draft cannot be retried.
	draft := must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{Body: "draft"}))(t)
	code(t, errOf(f.posts.Retry(ctx, mgrID, clientID, draft.ID)), apperr.Validation)
	if res := must(f.posts.Retry(ctx, mgrID, clientID, saved.ID))(t); res.Status != domain.StatusFailed {
		t.Fatal(res.Status)
	}
	// Back to draft clears the approval and allows editing again.
	must(f.posts.BackToDraft(ctx, mgrID, clientID, saved.ID))(t)
	p := f.post(t, saved.ID)
	if p.Status != domain.StatusDraft || p.ApprovedByID != nil {
		t.Fatalf("%+v", p)
	}
}

func TestIntegrationRejectDuplicateCommentReschedule(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	asset := f.uploadPNG(t)
	saved := must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{Title: "Review", Body: "isi", Targets: []app.TargetInput{{SocialAccountID: igAcc, BodyOverride: stringp(" khusus ig ")}}, MediaIDs: []string{asset.ID}}))(t)
	must(f.posts.SubmitForReview(ctx, mgrID, clientID, saved.ID))(t)

	code(t, errOf(f.posts.Reject(ctx, adminID, clientID, saved.ID, "   ")), apperr.Validation)
	code(t, errOf(f.posts.Reject(ctx, mgrID, clientID, saved.ID, "revisi")), apperr.Unauthorized)
	rejected := must(f.posts.Reject(ctx, adminID, clientID, saved.ID, " Ganti visual "))(t)
	if rejected.Status != domain.StatusRejected {
		t.Fatal(rejected.Status)
	}
	p := f.post(t, saved.ID)
	if p.ReviewNote == nil || *p.ReviewNote != "Ganti visual" {
		t.Fatalf("%+v", p)
	}
	acts := f.activities(t, saved.ID)
	if acts[0].Type != domain.ActRejected || acts[0].Meta["note"] != "Ganti visual" {
		t.Fatalf("%+v", acts[0])
	}
	n := f.identity.last(t)
	if n.Type != app.NotifPostRejected || n.Body != "Ganti visual" || n.UserIDs[0] != mgrID || n.Title != "Post ditolak: Review" {
		t.Fatalf("%+v", n)
	}
	// Editing a rejected post goes back to DRAFT and clears the review note? (note stays; status resets)
	edited := must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{ID: saved.ID, Title: "Review v2", Body: "isi baru", Targets: []app.TargetInput{{SocialAccountID: fbAcc}}}))(t)
	if edited.Status != domain.StatusDraft {
		t.Fatal(edited.Status)
	}
	targets := must(f.store.ListTargets(ctx, saved.ID))(t)
	if len(targets) != 1 || targets[0].SocialAccountID != fbAcc || targets[0].Status != domain.StatusDraft {
		t.Fatalf("%+v", targets)
	}
	if m := must(f.store.ListPostMedia(ctx, saved.ID))(t); len(m) != 0 {
		t.Fatal("media should have been cleared")
	}
	if acts := f.activities(t, saved.ID); acts[0].Type != domain.ActEdited || acts[0].Meta["from"] != "REJECTED" {
		t.Fatalf("%+v", acts[0])
	}

	// Comments: viewers may comment; empty bodies are refused.
	code(t, errOf(f.posts.AddComment(ctx, viewerID, clientID, saved.ID, " ")), apperr.Validation)
	c := must(f.posts.AddComment(ctx, viewerID, clientID, saved.ID, " Catatan tim "))(t)
	if c.Body != "Catatan tim" || *c.UserID != viewerID {
		t.Fatalf("%+v", c)
	}
	detail := must(f.queries.GetPost(ctx, viewerID, clientID, saved.ID))(t)
	if len(detail.Comments) != 1 || *detail.Comments[0].UserName != "Vio" {
		t.Fatalf("%+v", detail.Comments)
	}

	// Reschedule a draft (planned date), then duplicate.
	planned := f.clock.T.Add(48 * time.Hour)
	must(f.posts.Reschedule(ctx, mgrID, clientID, saved.ID, planned))(t)
	if p := f.post(t, saved.ID); p.ScheduledAt == nil || !p.ScheduledAt.Equal(planned) || p.Status != domain.StatusDraft {
		t.Fatalf("%+v", p)
	}
	code(t, errOf(f.posts.Duplicate(ctx, viewerID, clientID, saved.ID)), apperr.Unauthorized)
	dup := must(f.posts.Duplicate(ctx, mgr2ID, clientID, saved.ID))(t)
	dp := f.post(t, dup.ID)
	if dp.Title != "Review v2 (salinan)" || dp.Body != "isi baru" || dp.ScheduledAt != nil || dp.Status != domain.StatusDraft || *dp.CreatedByID != mgr2ID {
		t.Fatalf("%+v", dp)
	}
	if tg := must(f.store.ListTargets(ctx, dup.ID))(t); len(tg) != 1 || tg[0].SocialAccountID != fbAcc {
		t.Fatalf("%+v", tg)
	}
	if acts := f.activities(t, dup.ID); acts[0].Meta["duplicatedFrom"] != saved.ID {
		t.Fatalf("%+v", acts[0])
	}
	// Unschedule only from SCHEDULED.
	code(t, errOf(f.posts.Unschedule(ctx, mgrID, clientID, saved.ID)), apperr.Unauthorized)
}

func TestIntegrationMediaLibrary(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	code(t, errOf(f.media.Upload(ctx, viewerID, clientID, []app.UploadFile{{Filename: "a.png", Content: pngBytes}})), apperr.Unauthorized)
	e := code(t, errOf(f.media.Upload(ctx, mgrID, clientID, []app.UploadFile{{Filename: "ok.png", Content: pngBytes}, {Filename: "evil.png", Content: []byte("<html>")}})), apperr.Validation)
	if e.Message != "Jenis file tidak didukung. (evil.png)" {
		t.Fatal(e.Message)
	}
	if rows := must(f.media.List(ctx, viewerID, clientID))(t); len(rows) != 0 {
		t.Fatal("nothing should be stored when one file fails")
	}
	assets := must(f.media.Upload(ctx, mgrID, clientID, []app.UploadFile{{Filename: "a.png", Content: pngBytes}, {Filename: "b.jpg", Content: jpegBytes}}))(t)
	if len(assets) != 2 || assets[0].MimeType != "image/png" || assets[1].MimeType != "image/jpeg" || assets[1].Kind != domain.MediaImage || !strings.HasPrefix(assets[0].URL, "https://app.white.id/api/media/"+clientID+"/") {
		t.Fatalf("%+v", assets)
	}
	key := domain.KeyFromURL(assets[0].URL)
	if obj := must(f.media.Object(ctx, key))(t); obj == nil || obj.ContentType != "image/png" || string(obj.Body) != string(pngBytes) {
		t.Fatal("stored object not served")
	}

	ext := must(f.media.AddExternal(ctx, mgrID, clientID, app.ExternalMediaInput{URL: "https://cdn.example.com/img/Kopi%20Promo.PNG", Kind: domain.MediaImage, Tags: []string{"promo", " promo "}}))(t)
	if ext.Filename != "Kopi Promo.PNG" || ext.MimeType != "image/png" || len(ext.Tags) != 1 || ext.SizeBytes != 0 {
		t.Fatalf("%+v", ext)
	}
	e = code(t, errOf(f.media.AddExternal(ctx, mgrID, clientID, app.ExternalMediaInput{URL: "http://cdn.example.com/a.jpg", Kind: domain.MediaImage})), apperr.Validation)
	if e.Message != "URL harus diawali https://" {
		t.Fatal(e.Message)
	}

	upd := must(f.media.Update(ctx, mgrID, clientID, ext.ID, stringp("  Alt baru "), nil))(t)
	if *upd.AltText != "Alt baru" || len(upd.Tags) != 1 {
		t.Fatalf("%+v", upd)
	}
	upd = must(f.media.Update(ctx, mgrID, clientID, ext.ID, stringp(""), []string{"x"}))(t)
	if upd.AltText != nil || upd.Tags[0] != "x" {
		t.Fatalf("%+v", upd)
	}
	code(t, errOf(f.media.Update(ctx, mgrID, clientID, "nope", nil, nil)), apperr.NotFound)

	// In use: refused until the post lets go.
	saved := must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{Body: "x", MediaIDs: []string{assets[0].ID}}))(t)
	e = code(t, f.media.Delete(ctx, mgrID, clientID, assets[0].ID), apperr.Validation)
	if e.Message != "Media masih dipakai oleh post. Lepaskan dari post terlebih dahulu." {
		t.Fatal(e.Message)
	}
	must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{ID: saved.ID, Body: "x"}))(t)
	if err := f.media.Delete(ctx, mgrID, clientID, assets[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.uploads, key)); !os.IsNotExist(err) {
		t.Fatal("file should be gone")
	}
	if err := f.media.Delete(ctx, mgrID, clientID, ext.ID); err != nil {
		t.Fatal("external media delete should skip storage")
	}
}

func TestIntegrationMaintenance(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	asset := f.uploadPNG(t)
	saved := must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{Body: "x", MediaIDs: []string{asset.ID}, Targets: []app.TargetInput{{SocialAccountID: igAcc}}}))(t)
	must(f.posts.AddComment(ctx, mgrID, clientID, saved.ID, "hai"))(t)
	if err := f.store.InsertSyncJob(ctx, domain.SyncJob{ID: "job_old", ClientID: stringp(clientID), Kind: "PUBLISH", Status: domain.JobSuccess, StartedAt: f.clock.T.Add(-100 * 24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := f.store.InsertSyncJob(ctx, domain.SyncJob{ID: "job_new", ClientID: stringp(clientID), Kind: "PUBLISH", Status: domain.JobSuccess, StartedAt: f.clock.T}); err != nil {
		t.Fatal(err)
	}
	if n := must(f.maintenance.RunRetention(ctx, f.clock.T))(t); n != 1 {
		t.Fatal(n)
	}

	if err := f.maintenance.ForgetUser(ctx, mgrID); err != nil {
		t.Fatal(err)
	}
	if p := f.post(t, saved.ID); p.CreatedByID != nil {
		t.Fatal("author should be cleared")
	}
	if c := must(f.store.ListComments(ctx, saved.ID))(t); c[0].UserID != nil {
		t.Fatal("comment user should be cleared")
	}

	if err := f.maintenance.PurgeClient(ctx, clientID); err != nil {
		t.Fatal(err)
	}
	if p := must(f.store.GetPost(ctx, saved.ID))(t); p != nil {
		t.Fatal("post survived purge")
	}
	if rows := must(f.store.ListMedia(ctx, clientID))(t); len(rows) != 0 {
		t.Fatal("media survived purge")
	}
	if jobs := must(f.store.ListSyncJobs(ctx, clientID, 10))(t); len(jobs) != 0 {
		t.Fatal("sync jobs survived purge")
	}
	if _, err := os.Stat(filepath.Join(f.uploads, clientID)); !os.IsNotExist(err) {
		t.Fatal("client files survived purge")
	}
}

func errOf[T any](_ T, err error) error { return err }

func stringp(s string) *string { return &s }

func TestIntegrationStoryFormat(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	a, b := f.uploadPNG(t), f.uploadPNG(t)

	// Unset format stays FEED.
	feed := must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{Body: "feed", Targets: []app.TargetInput{{SocialAccountID: igAcc}}, MediaIDs: []string{a.ID}}))(t)
	if got := f.post(t, feed.ID).Format; got != domain.FormatFeed {
		t.Fatal(got)
	}

	story := must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{Format: domain.FormatStory, Targets: []app.TargetInput{{SocialAccountID: igAcc}}, MediaIDs: []string{a.ID, b.ID}}))(t)
	if got := f.post(t, story.ID).Format; got != domain.FormatStory {
		t.Fatal(got)
	}
	// Two media cannot be approved as a story.
	e := code(t, errOf(f.posts.Approve(ctx, adminID, clientID, story.ID)), apperr.Validation)
	if !strings.Contains(e.Message, domain.MsgStoryOneMedia) {
		t.Fatal(e.Message)
	}
	must(f.posts.Save(ctx, mgrID, clientID, app.SaveInput{ID: story.ID, Format: domain.FormatStory, Targets: []app.TargetInput{{SocialAccountID: igAcc}}, MediaIDs: []string{a.ID}}))(t)
	must(f.posts.Approve(ctx, adminID, clientID, story.ID))(t)
	out := must(f.posts.PublishNow(ctx, adminID, clientID, story.ID))(t)
	if out.Status != domain.StatusPublished {
		t.Fatalf("%+v", out)
	}
	dup := must(f.posts.Duplicate(ctx, mgrID, clientID, story.ID))(t)
	if got := f.post(t, dup.ID).Format; got != domain.FormatStory {
		t.Fatal(got)
	}
}
