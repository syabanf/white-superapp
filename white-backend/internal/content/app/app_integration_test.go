package app_test

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/pdf"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/providers/ai"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

const (
	clientID = "client_kopi"
	adminID  = "user_admin"
	copyID   = "user_copy"   // manager, copywriter
	leadID   = "user_lead"   // manager, reviewer
	designID = "user_design" // manager, designer
	viewerID = "user_viewer"
)

var pngBytes = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0x0d, 'I', 'H', 'D', 'R'}

// ─── fakes ───────────────────────────────────────────────────

type fakeIdentity struct {
	mu    sync.Mutex
	roles map[string]domain.Role
	sent  []app.NotifyInput
	asked []string
}

func (f *fakeIdentity) CheckAccess(_ context.Context, userID, cid string) (*app.Access, error) {
	role, ok := f.roles[userID]
	if !ok || cid != clientID {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return &app.Access{UserID: userID, Role: role, ClientID: cid, Slug: "kopi", Name: "Kopi Senja"}, nil
}

func (f *fakeIdentity) ClientInfo(context.Context, string) (*app.ClientInfo, error) {
	return &app.ClientInfo{ID: clientID, Slug: "kopi", Name: "Kopi Senja"}, nil
}

func (f *fakeIdentity) UserName(_ context.Context, id string) string { return "Nama " + id }

func (f *fakeIdentity) Notify(_ context.Context, in app.NotifyInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, in)
	return nil
}

type userKey struct{}

func (f *fakeIdentity) AsUser(ctx context.Context, userID string) (context.Context, error) {
	f.mu.Lock()
	f.asked = append(f.asked, userID)
	f.mu.Unlock()
	return context.WithValue(ctx, userKey{}, userID), nil
}

func (f *fakeIdentity) types() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.sent))
	for _, n := range f.sent {
		out = append(out, n.Type)
	}
	return out
}

type fakeAnalytics struct {
	accounts []app.SocialAccount
	posts    []app.SocialPost
}

func (f *fakeAnalytics) OwnAccounts(context.Context, string) ([]app.SocialAccount, error) {
	return f.accounts, nil
}

func (f *fakeAnalytics) Posts(_ context.Context, _ string, externalIDs []string, _, _ string) ([]app.SocialPost, error) {
	if len(externalIDs) == 0 {
		return f.posts, nil
	}
	var out []app.SocialPost
	for _, p := range f.posts {
		for _, id := range externalIDs {
			if p.ExternalID == id {
				out = append(out, p)
			}
		}
	}
	return out, nil
}

type fakePublishing struct {
	mu        sync.Mutex
	uploads   int
	saved     []app.PostInput
	author    string
	approved  []string
	scheduled map[string]time.Time
	state     *app.PostState
}

func (f *fakePublishing) UploadMedia(_ context.Context, _ string, files []app.MediaFile) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.uploads += len(files)
	return []string{"media_1"}, nil
}

func (f *fakePublishing) SavePost(ctx context.Context, in app.PostInput) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, in)
	f.author, _ = ctx.Value(userKey{}).(string)
	return "post_1", nil
}

func (f *fakePublishing) ApprovePost(_ context.Context, _, postID string) error {
	f.approved = append(f.approved, postID)
	return nil
}

func (f *fakePublishing) SchedulePost(_ context.Context, _, postID string, at time.Time) error {
	if f.scheduled == nil {
		f.scheduled = map[string]time.Time{}
	}
	f.scheduled[postID] = at
	return nil
}

func (f *fakePublishing) GetPost(context.Context, string, string) (*app.PostState, error) {
	if f.state == nil {
		return nil, apperr.New(apperr.Network, "publishing down")
	}
	return f.state, nil
}

// ─── fixture ─────────────────────────────────────────────────

type fixture struct {
	store      *postgres.Store
	plans      *app.Plans
	items      *app.Items
	designs    *app.Designs
	assets     *app.Assets
	reports    *app.Reports
	dashboard  *app.Dashboard
	scheduling *app.Scheduling
	jobs       *app.Jobs
	identity   *fakeIdentity
	analytics  *fakeAnalytics
	publishing *fakePublishing
	clock      *clock.Fixed
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := pg.TestPool(t, content.Migrations(), "migrations", content.Schema)
	pg.TruncateSchema(t, pool, content.Schema)
	clk := &clock.Fixed{T: time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC)}
	f := &fixture{
		store:      postgres.NewStore(pool),
		identity:   &fakeIdentity{roles: map[string]domain.Role{adminID: domain.RoleAdmin, copyID: domain.RoleManager, leadID: domain.RoleManager, designID: domain.RoleManager, viewerID: domain.RoleViewer}},
		analytics:  &fakeAnalytics{accounts: []app.SocialAccount{{ID: "acc_ig", Platform: domain.PlatformInstagram, Username: "kopisenja"}}},
		publishing: &fakePublishing{},
		clock:      clk,
	}
	deps := app.Deps{
		Store: f.store, Files: storage.NewLocal(t.TempDir()), Identity: f.identity, Analytics: f.analytics, Publishing: f.publishing,
		AI:       ai.NewSource(func(context.Context) (ai.Credentials, error) { return ai.Credentials{}, nil }, ai.Options{}),
		Renderer: pdf.Renderer{}, Clock: clk,
	}
	f.plans, f.items, f.designs, f.assets = app.NewPlans(deps), app.NewItems(deps), app.NewDesigns(deps), app.NewAssets(deps)
	f.reports, f.scheduling, f.jobs = app.NewReports(deps), app.NewScheduling(deps), app.NewJobs(deps)
	f.dashboard = app.NewDashboard(deps)
	return f
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

func must[T any](v T, err error) func(*testing.T) T {
	return func(t *testing.T) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func errOf[T any](_ T, err error) error { return err }

var brief = domain.Brief{
	Goals: "Naikkan awareness menu baru", Pillars: []string{"Edukasi kopi", "Promo", "Behind the bar"}, Tone: "hangat, santai",
	Frequency: []domain.Frequency{{Platform: domain.PlatformInstagram, PerWeek: 3}, {Platform: domain.PlatformFacebook, PerWeek: 2}},
}

func (f *fixture) newPlan(t *testing.T, month string) *app.PlanDetail {
	t.Helper()
	return must(f.plans.Create(context.Background(), copyID, clientID, month, "", brief))(t)
}

func (f *fixture) newItem(t *testing.T, planID string, format domain.Format) *app.ItemDetail {
	t.Helper()
	in := app.ItemInput{PlannedDate: "2026-11-10", Platform: domain.PlatformInstagram, Format: format, Pillar: "Promo", Title: "Promo kopi susu", CopywriterID: copyID, DesignerID: designID, CopyDue: "2026-11-05", DesignDue: "2026-11-07", ReviewDue: "2026-11-09"}
	return must(f.items.Create(context.Background(), copyID, clientID, planID, in, app.CaptionInput{Hook: "Pagi butuh kopi?", Caption: "Kopi susu gula aren diskon 20%.", Hashtags: []string{"kopi", "#KopiSusu", " kopi "}}))(t)
}

func (f *fixture) move(t *testing.T, userID, itemID string, to domain.ItemStatus, note string) *app.ItemDetail {
	t.Helper()
	return must(f.items.Move(context.Background(), userID, clientID, itemID, to, note))(t)
}

// ─── tests ───────────────────────────────────────────────────

func TestIntegrationPlanLifecycle(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	code(t, errOf(f.plans.Create(ctx, copyID, clientID, "2026/11", "", brief)), apperr.Validation)
	code(t, errOf(f.plans.Create(ctx, viewerID, clientID, "2026-11", "", brief)), apperr.Unauthorized)
	plan := f.newPlan(t, "2026-11")
	if plan.Plan.Title != "EP November 2026" || plan.Plan.Status != domain.PlanDraft || len(plan.Plan.Brief.Frequency) != 2 {
		t.Fatalf("%+v", plan.Plan)
	}
	e := code(t, errOf(f.plans.Create(ctx, copyID, clientID, "2026-11", "", brief)), apperr.Validation)
	if e.FieldErrors["month"][0] != "Editorial plan untuk bulan ini sudah ada." {
		t.Fatalf("%+v", e.FieldErrors)
	}

	// Empty plans cannot be submitted.
	code(t, errOf(f.plans.Submit(ctx, copyID, clientID, plan.Plan.ID)), apperr.Validation)

	// Mock draft: 3/week IG and 2/week FB over 30 days = 13 + 9 items.
	gen := must(f.plans.Generate(ctx, copyID, clientID, plan.Plan.ID, false))(t)
	if len(gen.Items) != 22 || gen.Plan.AIModel == nil || *gen.Plan.AIModel != "mock" {
		t.Fatalf("items=%d model=%v", len(gen.Items), gen.Plan.AIModel)
	}
	first := gen.Items[0]
	if first.Status != domain.ItemIdea || first.CaptionVersion != 1 || first.CopyDue == nil || first.Pillar != "Edukasi kopi" || len(first.Hashtags) == 0 {
		t.Fatalf("%+v", first)
	}
	// Regenerating with replace swaps the IDEA items instead of doubling them.
	gen = must(f.plans.Generate(ctx, copyID, clientID, plan.Plan.ID, true))(t)
	if len(gen.Items) != 22 {
		t.Fatal(len(gen.Items))
	}

	must(f.plans.Submit(ctx, copyID, clientID, plan.Plan.ID))(t)
	code(t, errOf(f.plans.Update(ctx, copyID, clientID, plan.Plan.ID, nil, &brief)), apperr.Validation)
	e = code(t, errOf(f.plans.Approve(ctx, copyID, clientID, plan.Plan.ID, "")), apperr.Unauthorized)
	if e.Message != "Penyetuju harus orang lain (kecuali Admin)." {
		t.Fatal(e.Message)
	}
	code(t, errOf(f.plans.RequestChanges(ctx, leadID, clientID, plan.Plan.ID, " ")), apperr.Validation)
	back := must(f.plans.RequestChanges(ctx, leadID, clientID, plan.Plan.ID, "Tambah konten promo payday"))(t)
	if back.Plan.Status != domain.PlanDraft || len(back.Comments) != 1 {
		t.Fatalf("%+v", back.Plan)
	}
	must(f.plans.Submit(ctx, copyID, clientID, plan.Plan.ID))(t)
	approved := must(f.plans.Approve(ctx, leadID, clientID, plan.Plan.ID, "Oke, lanjut produksi"))(t)
	if approved.Plan.Status != domain.PlanApproved || *approved.Plan.ApprovedBy != leadID {
		t.Fatalf("%+v", approved.Plan)
	}
	types := strings.Join(f.identity.types(), ",")
	if !strings.Contains(types, app.NotifPlanReview) || !strings.Contains(types, app.NotifPlanChanges) || !strings.Contains(types, app.NotifPlanApproved) {
		t.Fatal(types)
	}

	// Viewers comment but cannot delete; approved plans cannot be deleted.
	must(f.plans.Comment(ctx, viewerID, clientID, plan.Plan.ID, "Mantap"))(t)
	code(t, f.plans.Delete(ctx, viewerID, clientID, plan.Plan.ID), apperr.Unauthorized)
	code(t, f.plans.Delete(ctx, copyID, clientID, plan.Plan.ID), apperr.Validation)
	list := must(f.plans.List(ctx, viewerID, clientID))(t)
	if len(list) != 1 || list[0].ItemCount != 22 {
		t.Fatalf("%+v", list)
	}
}

func TestIntegrationItemWorkflowDesignScheduleAndReport(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	plan := f.newPlan(t, "2026-11")

	code(t, errOf(f.items.Create(ctx, copyID, clientID, plan.Plan.ID, app.ItemInput{PlannedDate: "2026-12-01", Platform: domain.PlatformInstagram, Format: domain.FormatFeed}, app.CaptionInput{})), apperr.Validation)
	d := f.newItem(t, plan.Plan.ID, domain.FormatFeed)
	it := d.Item
	if it.CaptionVersion != 1 || len(d.Revisions) != 1 || strings.Join(it.Hashtags, " ") != "#kopi #KopiSusu" {
		t.Fatalf("%+v", it)
	}

	f.move(t, copyID, it.ID, domain.ItemCopywriting, "")
	code(t, errOf(f.items.Move(ctx, copyID, clientID, it.ID, domain.ItemApproved, "")), apperr.Unauthorized)
	f.move(t, copyID, it.ID, domain.ItemDesign, "")
	d = f.move(t, designID, it.ID, domain.ItemReview, "")
	if d.Item.CopyDoneAt == nil || d.Item.DesignDoneAt == nil {
		t.Fatalf("%+v", d.Item)
	}
	// The copywriter cannot review their own copy; a viewer cannot move.
	e := code(t, errOf(f.items.Move(ctx, copyID, clientID, it.ID, domain.ItemApproved, "")), apperr.Unauthorized)
	if e.Message != "Penyetuju harus orang lain (kecuali Admin)." {
		t.Fatal(e.Message)
	}
	code(t, errOf(f.items.Move(ctx, viewerID, clientID, it.ID, domain.ItemApproved, "")), apperr.Unauthorized)
	code(t, errOf(f.items.Move(ctx, leadID, clientID, it.ID, domain.ItemRevision, "")), apperr.Validation)
	d = f.move(t, leadID, it.ID, domain.ItemRevision, "CTA kurang jelas")
	if d.Item.RevisionRound != 1 || len(d.Comments) != 1 || d.Comments[0].Kind != domain.DecisionChangesRequested {
		t.Fatalf("%+v %+v", d.Item, d.Comments)
	}
	d = must(f.items.Revise(ctx, copyID, clientID, it.ID, app.CaptionInput{Caption: "Kopi susu gula aren diskon 20%, cuma minggu ini!", Hashtags: []string{"KopiSusu"}, Note: "CTA dipertegas"}))(t)
	if d.Item.CaptionVersion != 2 || len(d.Revisions) != 2 || d.Revisions[0].Version != 2 {
		t.Fatalf("%+v", d.Revisions)
	}
	f.move(t, copyID, it.ID, domain.ItemReview, "")
	d = f.move(t, leadID, it.ID, domain.ItemApproved, "Siap tayang")
	if d.Item.ApprovedAt == nil || *d.Item.ApprovedBy != leadID {
		t.Fatalf("%+v", d.Item)
	}

	// No final design yet.
	at := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)
	code(t, errOf(f.scheduling.Schedule(ctx, leadID, clientID, it.ID, &at)), apperr.Validation)

	task := must(f.designs.Create(ctx, leadID, clientID, it.ID, app.DesignInput{References: []string{"https://pinterest.com/x", " "}}))(t)
	if task.Task.SizePreset != "feed-portrait" || task.Task.Width != 1080 || task.Task.Height != 1350 || *task.Task.DesignerID != designID || task.Task.Brief != "" || len(task.Task.References) != 1 {
		t.Fatalf("%+v", task.Task)
	}
	code(t, errOf(f.designs.Review(ctx, leadID, clientID, task.Task.ID, domain.DecisionApproved, "", "")), apperr.Validation)
	code(t, errOf(f.designs.Upload(ctx, designID, clientID, task.Task.ID, []app.UploadFile{{Filename: "x.exe", Content: []byte("MZ")}}, "")), apperr.Validation)
	task = must(f.designs.Upload(ctx, designID, clientID, task.Task.ID, []app.UploadFile{{Filename: "feed.png", Content: pngBytes}}, "Versi pertama"))(t)
	if task.Task.CurrentVersion != 1 || task.Task.Status != domain.DesignInReview || len(task.Files) != 1 || len(task.Feedback) != 1 {
		t.Fatalf("%+v", task)
	}
	code(t, errOf(f.designs.Review(ctx, designID, clientID, task.Task.ID, domain.DecisionApproved, "", "")), apperr.Unauthorized)
	task = must(f.designs.Review(ctx, leadID, clientID, task.Task.ID, domain.DecisionChangesRequested, "Logo kekecilan", task.Files[0].ID))(t)
	if task.Task.Status != domain.DesignChangesRequested {
		t.Fatal(task.Task.Status)
	}
	task = must(f.designs.Upload(ctx, designID, clientID, task.Task.ID, []app.UploadFile{{Filename: "feed-v2.png", Content: pngBytes}}, ""))(t)
	task = must(f.designs.Review(ctx, leadID, clientID, task.Task.ID, domain.DecisionApproved, "", ""))(t)
	if task.Task.Status != domain.DesignApproved || task.Task.CurrentVersion != 2 || len(task.Files) != 2 {
		t.Fatalf("%+v", task.Task)
	}
	file := must(f.designs.Download(ctx, viewerID, clientID, task.Files[0].ID))(t)
	if !bytes.Equal(file.Data, pngBytes) || file.ContentType != "image/png" || file.Name != "feed-v2.png" {
		t.Fatalf("%+v", file.Name)
	}
	if got := must(f.items.Get(ctx, copyID, clientID, it.ID))(t); !got.Item.HasFinalDesign {
		t.Fatal("final design not detected")
	}

	// Scheduling: the copywriter authors the post, the lead approves and schedules it.
	past := f.clock.T.Add(-time.Hour)
	code(t, errOf(f.scheduling.Schedule(ctx, leadID, clientID, it.ID, &past)), apperr.Validation)
	sched := must(f.scheduling.Schedule(ctx, leadID, clientID, it.ID, &at))(t)
	if sched.Status != domain.ItemScheduled || *sched.PublishingPostID != "post_1" || *sched.ScheduledBy != leadID {
		t.Fatalf("%+v", sched)
	}
	saved := f.publishing.saved[0]
	if f.publishing.author != copyID || saved.Format != domain.FormatFeed || saved.Body != "Kopi susu gula aren diskon 20%, cuma minggu ini!\n\n#KopiSusu" || len(saved.AccountIDs) != 1 || f.publishing.uploads != 1 {
		t.Fatalf("author=%s %+v", f.publishing.author, saved)
	}
	if !f.publishing.scheduled["post_1"].Equal(at) || len(f.publishing.approved) != 1 {
		t.Fatalf("%+v", f.publishing)
	}
	code(t, errOf(f.items.Revise(ctx, copyID, clientID, it.ID, app.CaptionInput{Caption: "x"})), apperr.Validation)

	// The worker reconciles after the publish time.
	ext, link := "ig_123", "https://www.instagram.com/p/abc/"
	f.publishing.state = &app.PostState{Status: "PUBLISHED", PublishedAt: &at, Targets: []app.PostTarget{{ExternalID: &ext, Permalink: &link}}}
	run := must(f.scheduling.Reconcile(ctx, at.Add(time.Minute), 10))(t)
	if run.Checked != 1 || run.Changed != 1 {
		t.Fatalf("%+v", run)
	}
	got := must(f.items.Get(ctx, copyID, clientID, it.ID))(t).Item
	if got.Status != domain.ItemPublished || got.ExternalIDs[0] != ext || *got.Permalink != link {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(strings.Join(f.identity.types(), ","), app.NotifItemPublished) {
		t.Fatal(f.identity.types())
	}

	// A second item fails to publish; sync marks it failed.
	d2 := f.newItem(t, plan.Plan.ID, domain.FormatStory)
	if d2.Item.Status != domain.ItemIdea {
		t.Fatal(d2.Item.Status)
	}

	// Report: engagement comes from analytics by external id.
	f.analytics.posts = []app.SocialPost{{ExternalID: ext, Metrics: domain.PostMetrics{Likes: 120, Comments: 14, Shares: 6, Saves: 30, Reach: 4100}}}
	f.clock.T = time.Date(2026, 12, 2, 3, 0, 0, 0, time.UTC)
	rep := must(f.reports.Generate(ctx, leadID, clientID, "2026-11"))(t)
	if rep.Data.Planned != 2 || rep.Data.Published != 1 || !rep.Data.EngagementAvailable || rep.Data.TotalEngagements != 170 || rep.Data.Top[0].ItemID != it.ID || rep.AIModel != "mock" {
		t.Fatalf("%+v", rep.Data)
	}
	if rep.Data.RevisionsTotal != 1 || rep.Data.DeadlinesTotal == 0 || !strings.Contains(rep.Summary, "## Rekomendasi") {
		t.Fatalf("%+v\n%s", rep.Data, rep.Summary)
	}
	again := must(f.reports.Generate(ctx, leadID, clientID, "2026-11"))(t)
	if again.ID != rep.ID {
		t.Fatal("regenerating created a second report")
	}
	pdfFile := must(f.reports.PDF(ctx, viewerID, clientID, rep.ID))(t)
	if !bytes.HasPrefix(pdfFile.Data, []byte("%PDF-")) || pdfFile.ContentType != "application/pdf" {
		t.Fatal("pdf")
	}
	code(t, errOf(f.reports.Generate(ctx, viewerID, clientID, "2026-11")), apperr.Unauthorized)

	// Dashboard: viewers read it; engagement from analytics is kept as the
	// item's snapshot and used when analytics has nothing.
	q := app.DashboardQuery{From: "2026-11-01", To: "2026-11-30", Compare: domain.CompareYear}
	dash := must(f.dashboard.Get(ctx, viewerID, clientID, q))(t)
	if k := dash.Current.Kpis; k.Planned != 2 || k.Published != 1 || k.Engagements != 170 || k.Reach != 4100 || dash.PrevFrom.Format(time.DateOnly) != "2025-11-01" {
		t.Fatalf("%+v prev %s", k, dash.PrevFrom)
	}
	if len(dash.Current.Posts) != 1 || !dash.Current.Posts[0].HasMetrics || len(dash.Pillars) != 1 || dash.Pillars[0] != "Promo" {
		t.Fatalf("%+v %v", dash.Current.Posts, dash.Pillars)
	}
	f.analytics.posts = nil
	q.Filter.Format = domain.FormatFeed
	dash = must(f.dashboard.Get(ctx, viewerID, clientID, q))(t)
	if k := dash.Current.Kpis; k.Planned != 1 || k.Engagements != 170 {
		t.Fatalf("snapshot or filter ignored: %+v", k)
	}
	code(t, errOf(f.dashboard.Get(ctx, viewerID, clientID, app.DashboardQuery{From: "2026-11-30", To: "2026-11-01"})), apperr.Validation)
	code(t, errOf(f.dashboard.Get(ctx, "user_stranger", clientID, q)), apperr.Unauthorized)
}

func TestIntegrationScheduleFailsWithoutAccountAndSyncMarksFailed(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	plan := f.newPlan(t, "2026-11")
	d := f.newItem(t, plan.Plan.ID, domain.FormatStory)
	it := d.Item
	for _, step := range []struct {
		user string
		to   domain.ItemStatus
	}{{copyID, domain.ItemCopywriting}, {copyID, domain.ItemReview}, {adminID, domain.ItemApproved}} {
		f.move(t, step.user, it.ID, step.to, "")
	}
	task := must(f.designs.Create(ctx, leadID, clientID, it.ID, app.DesignInput{}))(t)
	if task.Task.SizePreset != "story" {
		t.Fatal(task.Task.SizePreset)
	}
	must(f.designs.Upload(ctx, designID, clientID, task.Task.ID, []app.UploadFile{{Filename: "s.png", Content: pngBytes}}, ""))(t)
	must(f.designs.Review(ctx, leadID, clientID, task.Task.ID, domain.DecisionApproved, "", ""))(t)

	at := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)
	f.analytics.accounts = []app.SocialAccount{{ID: "acc_fb", Platform: domain.PlatformFacebook}}
	e := code(t, errOf(f.scheduling.Schedule(ctx, leadID, clientID, it.ID, &at)), apperr.Validation)
	if e.Message != "Klien belum punya akun Instagram yang terhubung." {
		t.Fatal(e.Message)
	}
	f.analytics.accounts = []app.SocialAccount{{ID: "acc_ig", Platform: domain.PlatformInstagram}}
	must(f.scheduling.Schedule(ctx, leadID, clientID, it.ID, &at))(t)
	if saved := f.publishing.saved[0]; saved.Format != domain.FormatStory || saved.Body != "" {
		t.Fatalf("%+v", saved)
	}

	code(t, f.scheduling.Sync(ctx, viewerID, clientID, "missing"), apperr.NotFound)
	msg := "Token kedaluwarsa"
	f.publishing.state = &app.PostState{Status: "FAILED", Targets: []app.PostTarget{{Error: &msg}}}
	if err := f.scheduling.Sync(ctx, leadID, clientID, it.ID); err != nil {
		t.Fatal(err)
	}
	got := must(f.items.Get(ctx, leadID, clientID, it.ID))(t)
	if got.Item.Status != domain.ItemFailed || *got.Item.PublishError != msg {
		t.Fatalf("%+v", got.Item)
	}
	// Failed items can be scheduled again: a new publishing post is created.
	must(f.scheduling.Schedule(ctx, leadID, clientID, it.ID, &at))(t)
	if len(f.publishing.saved) != 2 || f.publishing.saved[1].PostID != "" {
		t.Fatalf("%+v", f.publishing.saved)
	}
}

func TestIntegrationAssets(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	folder := must(f.assets.CreateFolder(ctx, leadID, clientID, "Logo", ""))(t)
	code(t, errOf(f.assets.CreateFolder(ctx, leadID, clientID, "logo", "")), apperr.Validation)
	code(t, errOf(f.assets.CreateFolder(ctx, viewerID, clientID, "Foto", "")), apperr.Unauthorized)
	sub := must(f.assets.CreateFolder(ctx, leadID, clientID, "Varian", folder.ID))(t)

	code(t, errOf(f.assets.Create(ctx, leadID, clientID, app.AssetInput{Name: "Logo", Kind: domain.AssetLogo}, app.UploadFile{Filename: "a.txt", Content: []byte("hello")})), apperr.Validation)
	a := must(f.assets.Create(ctx, leadID, clientID, app.AssetInput{Name: "Logo utama", Kind: domain.AssetLogo, Tags: []string{"Primary", "primary", "warna"}, FolderID: folder.ID}, app.UploadFile{Filename: "logo.png", Content: pngBytes}))(t)
	if a.Asset.CurrentVersion != 1 || len(a.Asset.Tags) != 2 || a.Asset.Current == nil {
		t.Fatalf("%+v", a.Asset)
	}
	pdfBytes := []byte("%PDF-1.4 brand guideline")
	a = must(f.assets.UploadVersion(ctx, leadID, clientID, a.Asset.ID, app.UploadFile{Filename: "logo-v2.pdf", Content: pdfBytes}, "Revisi warna"))(t)
	if a.Asset.CurrentVersion != 2 || len(a.Versions) != 2 || a.Asset.Current.MimeType != "application/pdf" {
		t.Fatalf("%+v", a)
	}
	cur := must(f.assets.Download(ctx, viewerID, clientID, a.Asset.ID, 0))(t)
	old := must(f.assets.Download(ctx, viewerID, clientID, a.Asset.ID, 1))(t)
	if !bytes.Equal(cur.Data, pdfBytes) || !bytes.Equal(old.Data, pngBytes) {
		t.Fatal("version bytes")
	}
	code(t, errOf(f.assets.Download(ctx, viewerID, clientID, a.Asset.ID, 9)), apperr.NotFound)

	for _, q := range []domain.AssetFilter{{Query: "UTAMA"}, {Tag: "Warna"}, {Kind: domain.AssetLogo}, {FolderID: folder.ID}, {Query: "primary"}} {
		if got := must(f.assets.Search(ctx, viewerID, clientID, q))(t); len(got) != 1 || got[0].Current.Version != 2 {
			t.Fatalf("%+v -> %+v", q, got)
		}
	}
	if got := must(f.assets.Search(ctx, viewerID, clientID, domain.AssetFilter{Query: "tidak ada"}))(t); len(got) != 0 {
		t.Fatal(got)
	}
	code(t, f.assets.DeleteFolder(ctx, leadID, clientID, folder.ID), apperr.Validation)
	if err := f.assets.DeleteFolder(ctx, leadID, clientID, sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.assets.Delete(ctx, leadID, clientID, a.Asset.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.assets.DeleteFolder(ctx, leadID, clientID, folder.ID); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationRemindersAndPurge(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	plan := f.newPlan(t, "2026-11")
	// Copy due tomorrow (agency day 2026-10-27), design overdue, review far away.
	mk := func(status domain.ItemStatus, copyDue, designDue string) string {
		d := must(f.items.Create(ctx, copyID, clientID, plan.Plan.ID, app.ItemInput{PlannedDate: "2026-11-02", Platform: domain.PlatformInstagram, Format: domain.FormatFeed, Title: string(status), CopywriterID: copyID, DesignerID: designID, CopyDue: copyDue, DesignDue: designDue, ReviewDue: "2026-11-01"}, app.CaptionInput{}))(t)
		return d.Item.ID
	}
	mk(domain.ItemIdea, "2026-10-27", "")
	designItem := mk(domain.ItemIdea, "2026-10-20", "2026-10-25")
	mk(domain.ItemIdea, "2026-11-01", "")
	f.move(t, copyID, designItem, domain.ItemCopywriting, "")
	f.move(t, copyID, designItem, domain.ItemDesign, "")

	f.identity.sent = nil
	n := must(f.jobs.SendReminders(ctx, f.clock.T))(t)
	if n != 2 {
		t.Fatalf("sent %d: %+v", n, f.identity.sent)
	}
	byUser := map[string]string{}
	for _, s := range f.identity.sent {
		byUser[s.UserIDs[0]] = s.Title
	}
	if byUser[copyID] != "Deadline copywriting besok: IDEA" || !strings.HasPrefix(byUser[designID], "Deadline desain terlewat") {
		t.Fatalf("%+v", byUser)
	}
	// Deduplicated within the day, sent again the next day.
	if n := must(f.jobs.SendReminders(ctx, f.clock.T.Add(time.Hour)))(t); n != 0 {
		t.Fatal(n)
	}
	if n := must(f.jobs.SendReminders(ctx, f.clock.T.Add(24*time.Hour)))(t); n == 0 {
		t.Fatal("no reminder the next day")
	}

	if err := f.jobs.ForgetUser(ctx, designID); err != nil {
		t.Fatal(err)
	}
	if got := must(f.items.Get(ctx, copyID, clientID, designItem))(t); got.Item.DesignerID != nil {
		t.Fatal("designer kept")
	}
	if err := f.jobs.PurgeClient(ctx, clientID); err != nil {
		t.Fatal(err)
	}
	if got := must(f.plans.List(ctx, copyID, clientID))(t); len(got) != 0 {
		t.Fatal(got)
	}
}

func TestIntegrationPublishOutcomeEvents(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	plan := f.newPlan(t, "2026-11")
	scheduled := func(postID string) domain.Item {
		it := f.newItem(t, plan.Plan.ID, domain.FormatFeed).Item
		lead := leadID
		it.Status, it.PublishingPostID, it.ScheduledBy = domain.ItemScheduled, &postID, &lead
		if err := f.store.UpdateItem(ctx, it); err != nil {
			t.Fatal(err)
		}
		return it
	}
	at := f.clock.T

	// Unknown posts are ignored.
	if err := f.scheduling.OnPostOutcome(ctx, events.PublishPostPayload{PostID: "post_x", ClientID: clientID, Status: "PUBLISHED", At: at}); err != nil {
		t.Fatal(err)
	}

	// Publishing unreachable: the payload alone marks the item failed.
	failed := scheduled("post_fail")
	must(0, f.scheduling.OnPostOutcome(ctx, events.PublishPostPayload{PostID: "post_fail", ClientID: clientID, Status: "FAILED", Error: "Token kedaluwarsa", At: at}))(t)
	got := must(f.items.Get(ctx, copyID, clientID, failed.ID))(t).Item
	if got.Status != domain.ItemFailed || got.PublishError == nil || *got.PublishError != "Token kedaluwarsa" {
		t.Fatalf("%+v", got)
	}

	// Published: the full post state (every target) wins over the payload.
	ok := scheduled("post_ok")
	ig, fb, link := "ig_1", "fb_1", "https://www.instagram.com/p/x/"
	f.publishing.state = &app.PostState{Status: "PUBLISHED", PublishedAt: &at, Targets: []app.PostTarget{{ExternalID: &ig, Permalink: &link}, {ExternalID: &fb}}}
	must(0, f.scheduling.OnPostOutcome(ctx, events.PublishPostPayload{PostID: "post_ok", ClientID: clientID, Status: "PUBLISHED", ExternalID: ig, At: at}))(t)
	got = must(f.items.Get(ctx, copyID, clientID, ok.ID))(t).Item
	if got.Status != domain.ItemPublished || len(got.ExternalIDs) != 2 || *got.Permalink != link {
		t.Fatalf("%+v", got)
	}

	// A redelivered event is a no-op.
	must(0, f.scheduling.OnPostOutcome(ctx, events.PublishPostPayload{PostID: "post_ok", ClientID: clientID, Status: "PUBLISHED", At: at}))(t)
	if n := strings.Count(strings.Join(f.identity.types(), ","), app.NotifItemPublished); n != 1 {
		t.Fatalf("published notifications %d", n)
	}
}
