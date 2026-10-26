package app_test

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/reports"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/adapter/pdf"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/adapter/providers/openrouter"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/app"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── fakes ───────────────────────────────────────────────────

var client = app.ClientInfo{ID: "client-1", Name: "Kopi Nusantara", Slug: "kopi-nusantara", Currency: "IDR", ShareModules: []string{"SOCIAL", "SEO"}}

type fakeIdentity struct {
	roles map[string]string // userID -> ADMIN | MANAGER | VIEWER
	users map[string]string // userID -> name
	share map[string]app.ShareContext
}

func (f *fakeIdentity) Check(_ context.Context, userID, clientID string) (*app.Access, error) {
	role, ok := f.roles[userID]
	if !ok || clientID != client.ID {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return &app.Access{Role: role, CanManage: role != "VIEWER", Client: client}, nil
}

func (f *fakeIdentity) UserName(_ context.Context, userID string) (string, bool, error) {
	name, ok := f.users[userID]
	return name, ok, nil
}

func (f *fakeIdentity) ShareContext(_ context.Context, slug, token string) (*app.ShareContext, error) {
	sc, ok := f.share[slug+"|"+token]
	if !ok {
		return &app.ShareContext{Status: app.ShareNotFound}, nil
	}
	return &sc, nil
}

type fakeAnalytics struct {
	calls []string
}

func (f *fakeAnalytics) ReportData(_ context.Context, _ string, current, _ dates.Range, modules []string, topN int) (*domain.ReportData, error) {
	f.calls = append(f.calls, strings.Join(modules, ",")+"/"+dates.Format(current.From))
	out := &domain.ReportData{}
	for _, m := range modules {
		switch m {
		case "SOCIAL":
			out.Social = &domain.SocialData{
				Overview: domain.SocialOverview{HasData: true, Followers: 1200, FollowersDelta: metrics.ComputeDelta(20, 10)},
				DailyFollowers: []domain.FollowersPoint{
					{Date: dates.AddDays(current.From, -1), Followers: 1180},
					{Date: current.From, Followers: 1190},
					{Date: current.To, Followers: 1200},
				},
				TopPosts: []domain.TopPost{{ID: "p1", Platform: "INSTAGRAM", Caption: "Halo", MediaType: "IMAGE", Engagements: 12}},
			}
		case "SEO":
			out.Seo = &domain.SeoData{HasData: true, Clicks: 300, TopQueries: []domain.TopQuery{{Query: "kopi", Clicks: 30}}}
		case "ADS":
			out.Ads = &domain.AdsData{HasData: false}
		}
	}
	return out, nil
}

func (f *fakeAnalytics) SuitePropertyID(context.Context, string) (string, error) {
	return "prop-1", nil
}

type fakePublishing struct{}

func (fakePublishing) Summary(context.Context, string, dates.Range) (*domain.PublishingSummary, error) {
	return &domain.PublishingSummary{Posts: 1, Mode: "recent", PerPlatform: []domain.PlatformCount{{Platform: "INSTAGRAM", Count: 1}},
		Top: []domain.PublishedTarget{{Title: "Promo", Platform: "INSTAGRAM", PublishedAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}}}, nil
}

type fakeSeoSuite struct{}

func (fakeSeoSuite) Rank(context.Context, string, dates.Range, dates.Range) (*domain.RankData, error) {
	return &domain.RankData{}, nil
}

func (fakeSeoSuite) Backlinks(context.Context, string, dates.Range) (*domain.BacklinkData, error) {
	return &domain.BacklinkData{HasSnapshot: true, Backlinks: 10}, nil
}

type failingInsights struct{}

func (failingInsights) Generate(context.Context, domain.InsightInput) (string, string, error) {
	return "", "", apperr.FromProvider("openrouter", apperr.RateLimit, apperr.Message(apperr.RateLimit), errors.New("429"))
}

// ─── fixture ─────────────────────────────────────────────────

type fixture struct {
	reports   *app.Reports
	store     *postgres.Store
	analytics *fakeAnalytics
	identity  *fakeIdentity
	deps      app.Deps
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := pg.TestPool(t, reports.Migrations(), "migrations", reports.Schema)
	pg.TruncateSchema(t, pool, reports.Schema)
	store := postgres.NewStore(pool)
	ident := &fakeIdentity{
		roles: map[string]string{"admin": "ADMIN", "manager": "MANAGER", "viewer": "VIEWER", "ghost": "VIEWER"},
		users: map[string]string{"admin": "Admin", "manager": "Manajer", "viewer": "Pemirsa"},
		share: map[string]app.ShareContext{
			"kopi-nusantara|":    {Status: app.ShareOK, Client: client},
			"tutup|":             {Status: app.ShareDisabled, Client: client},
			"berpin|":            {Status: app.ShareLocked, Client: client},
			"berpin|token-benar": {Status: app.ShareOK, Client: client},
			"tanpa-modul|":       {Status: app.ShareOK, Client: app.ClientInfo{ID: client.ID, Name: client.Name, Slug: "tanpa-modul"}},
		},
	}
	an := &fakeAnalytics{}
	deps := app.Deps{
		Store: store, Identity: ident, Analytics: an, Publishing: fakePublishing{}, SeoSuite: fakeSeoSuite{},
		Insights: openrouter.NewSource(func(context.Context) (openrouter.Credentials, error) { return openrouter.Credentials{}, nil }, openrouter.Options{}),
		Render:   pdf.Render,
		Clock:    clock.Fixed{T: time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)},
	}
	return &fixture{reports: app.New(deps), store: store, analytics: an, identity: ident, deps: deps}
}

func wantCode(t *testing.T, err error, code apperr.Code) *apperr.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got nil", code)
	}
	e := apperr.As(err)
	if e.Code != code {
		t.Fatalf("expected %s, got %s (%v)", code, e.Code, err)
	}
	return e
}

var ctx = context.Background()

func input(includeAI bool) app.CreateInput {
	return app.CreateInput{Title: "  Laporan Oktober  ", Modules: []string{"SOCIAL", "SEO"}, Compare: true, IncludeAI: includeAI, From: "2026-10-01", To: "2026-10-14"}
}

// ─── tests ───────────────────────────────────────────────────

func TestCreateReportWithoutAI(t *testing.T) {
	f := setup(t)
	row, err := f.reports.Create(ctx, "viewer", client.ID, input(false))
	if err != nil {
		t.Fatal(err)
	}
	if row.Title != "Laporan Oktober" || row.Language != "id" || row.AiSummary != nil || *row.CreatedByID != "viewer" || *row.CreatedByName != "Pemirsa" {
		t.Fatalf("created %+v", row)
	}
	if len(f.analytics.calls) != 0 {
		t.Fatal("no insight means no analytics call")
	}
	// A session whose user row is gone still creates, without a creator.
	ghost, err := f.reports.Create(ctx, "ghost", client.ID, input(false))
	if err != nil {
		t.Fatal(err)
	}
	if ghost.CreatedByID != nil {
		t.Fatal("created_by_id must be null for a missing user")
	}
	list, err := f.reports.List(ctx, "viewer", client.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list %+v", list)
	}
	for _, r := range list {
		if (r.ID == row.ID) != (r.CreatedByName != nil) {
			t.Fatalf("creator names %+v", r)
		}
	}
	wantCode(t, func() error { _, err := f.reports.List(ctx, "stranger", client.ID); return err }(), apperr.Unauthorized)
}

func TestCreateReportWithAI(t *testing.T) {
	f := setup(t)
	row, err := f.reports.Create(ctx, "manager", client.ID, input(true))
	if err != nil {
		t.Fatal(err)
	}
	if row.AiSummary == nil || !strings.HasPrefix(*row.AiSummary, "## Ringkasan eksekutif") {
		t.Fatalf("ai summary %v", row.AiSummary)
	}
	if f.analytics.calls[0] != "SOCIAL,SEO/2026-10-01" {
		t.Fatalf("payload modules %v", f.analytics.calls)
	}
	in, err := f.reports.LatestInsight(ctx, "viewer", client.ID, "OVERVIEW", "2026-10-01", "2026-10-14")
	if err != nil || in == nil || in.Model != "mock" || in.Content != *row.AiSummary {
		t.Fatalf("cached insight %+v %v", in, err)
	}

	// Provider failure aborts the creation.
	f.deps.Insights = failingInsights{}
	failing := app.New(f.deps)
	wantCode(t, func() error { _, err := failing.Create(ctx, "manager", client.ID, input(true)); return err }(), apperr.RateLimit)
	list, _ := f.reports.List(ctx, "manager", client.ID)
	if len(list) != 1 {
		t.Fatalf("a failed insight must not create a report, got %d", len(list))
	}
}

func TestCreateReportValidation(t *testing.T) {
	f := setup(t)
	bad := input(false)
	bad.Title, bad.Modules = " ", nil
	e := wantCode(t, func() error { _, err := f.reports.Create(ctx, "viewer", client.ID, bad); return err }(), apperr.Validation)
	if e.Message != "Judul laporan wajib diisi." || e.FieldErrors["modules"][0] != "Pilih minimal satu modul." {
		t.Fatalf("validation %+v", e)
	}
	bad = input(false)
	bad.To = "2026-09-01"
	e = wantCode(t, func() error { _, err := f.reports.Create(ctx, "viewer", client.ID, bad); return err }(), apperr.Validation)
	if e.Message != "Periode laporan tidak valid." {
		t.Fatalf("range %+v", e)
	}
	e = wantCode(t, func() error {
		_, err := f.reports.GenerateInsight(ctx, "viewer", client.ID, "", "2026-10-01", "2026-10-14", "id")
		return err
	}(), apperr.Validation)
	if e.Message != apperr.Message(apperr.Validation) {
		t.Fatalf("module %+v", e)
	}
}

func TestDeleteReportPermissions(t *testing.T) {
	f := setup(t)
	row, err := f.reports.Create(ctx, "viewer", client.ID, input(false))
	if err != nil {
		t.Fatal(err)
	}
	wantCode(t, f.reports.Delete(ctx, "viewer", client.ID, row.ID), apperr.Unauthorized)
	wantCode(t, f.reports.Delete(ctx, "manager", client.ID, "missing"), apperr.NotFound)
	if err := f.reports.Delete(ctx, "manager", client.ID, row.ID); err != nil {
		t.Fatal(err)
	}
	wantCode(t, f.reports.Delete(ctx, "admin", client.ID, row.ID), apperr.NotFound)
}

func TestGenerateInsightAndReportPdf(t *testing.T) {
	f := setup(t)
	in, err := f.reports.GenerateInsight(ctx, "viewer", client.ID, "SEO", "2026-10-01", "2026-10-14", "fr")
	if err != nil {
		t.Fatal(err)
	}
	if in.Language != "id" || !strings.Contains(in.Content, "## Peluang kata kunci") {
		t.Fatalf("insight %+v", in)
	}
	row, err := f.reports.Create(ctx, "viewer", client.ID, input(true))
	if err != nil {
		t.Fatal(err)
	}
	name, out, err := f.reports.RenderReportPdf(ctx, "viewer", row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if name != "Laporan_kopi-nusantara_2026-10-01_2026-10-14.pdf" || !bytes.HasPrefix(out, []byte("%PDF")) {
		t.Fatalf("pdf %s %q", name, out[:8])
	}
	wantCode(t, func() error { _, _, err := f.reports.RenderReportPdf(ctx, "viewer", "missing"); return err }(), apperr.NotFound)
	wantCode(t, func() error { _, _, err := f.reports.RenderReportPdf(ctx, "stranger", row.ID); return err }(), apperr.Unauthorized)

	secs, err := f.reports.Sections(ctx, "viewer", client.ID, "2026-10-01", "2026-10-14", true, []string{"SOCIAL", "SEO"}, "id")
	if err != nil {
		t.Fatal(err)
	}
	if len(secs.Social) != 1 || secs.Social[0].Key != "PUBLISHING" || len(secs.Seo) != 1 || secs.Seo[0].Key != "BACKLINKS" {
		t.Fatalf("sections %+v", secs)
	}
}

func TestShareStatuses(t *testing.T) {
	f := setup(t)
	for slug, want := range map[string]app.ShareStatus{"nope": app.ShareNotFound, "tutup": app.ShareDisabled, "berpin": app.ShareLocked} {
		d, err := f.reports.ShareDashboard(ctx, slug, "", "28d")
		if err != nil {
			t.Fatal(err)
		}
		p, err := f.reports.SharePdf(ctx, slug, "", "28d")
		if err != nil {
			t.Fatal(err)
		}
		if d.Status != want || p.Status != want || p.Pdf != nil || d.View.Social != nil {
			t.Fatalf("%s: dashboard %s pdf %s", slug, d.Status, p.Status)
		}
	}
	if len(f.analytics.calls) != 0 {
		t.Fatal("blocked shares must not load data")
	}

	d, err := f.reports.ShareDashboard(ctx, "berpin", "token-benar", "7d")
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != app.ShareOK || d.Range.Preset != "7d" || dates.Format(d.Range.Current.To) != "2026-10-20" || dates.Format(d.Range.Current.From) != "2026-10-14" {
		t.Fatalf("unlocked dashboard %+v", d.Range)
	}
	if d.View.Social == nil || d.View.Seo == nil || d.View.Ads != nil || len(d.View.Social.DailyFollowers) != 2 {
		t.Fatalf("view %+v", d.View)
	}
	d, _ = f.reports.ShareDashboard(ctx, "kopi-nusantara", "", "12m")
	if d.Range.Preset != "28d" || d.Range.Current.Days() != 28 {
		t.Fatalf("preset fallback %+v", d.Range)
	}
	d, _ = f.reports.ShareDashboard(ctx, "tanpa-modul", "", "")
	if d.Status != app.ShareOK || d.View.Social != nil {
		t.Fatal("no shared modules, no blocks")
	}

	// The share PDF picks up the latest OVERVIEW insight of that exact range.
	cur := dates.ForPreset("28d", f.deps.Clock.Now())
	if _, err := f.store.InsertInsight(ctx, domain.Insight{ID: "i1", ClientID: client.ID, Module: "OVERVIEW", DateFrom: cur.From, DateTo: cur.To, Language: "id", Model: "mock", Content: "## Ringkasan eksekutif\nIsi", CreatedAt: f.deps.Clock.Now()}); err != nil {
		t.Fatal(err)
	}
	p, err := f.reports.SharePdf(ctx, "kopi-nusantara", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != app.ShareOK || p.Filename != "Laporan_kopi-nusantara_2026-09-23_2026-10-20.pdf" || !bytes.HasPrefix(p.Pdf, []byte("%PDF")) {
		t.Fatalf("share pdf %s %s", p.Status, p.Filename)
	}
	// cover + SOCIAL + SEO + summary + closing
	if n := len(regexp.MustCompile(`/Type /Page\b`).FindAll(p.Pdf, -1)); n != 5 {
		t.Fatalf("share pdf has %d pages, want 5", n)
	}
}

func TestPurgeClientAndForgetUser(t *testing.T) {
	f := setup(t)
	if _, err := f.reports.Create(ctx, "viewer", client.ID, input(true)); err != nil {
		t.Fatal(err)
	}
	n, err := f.reports.ForgetUser(ctx, "viewer")
	if err != nil || n != 1 {
		t.Fatalf("forget user %d %v", n, err)
	}
	list, _ := f.reports.List(ctx, "viewer", client.ID)
	if list[0].CreatedByID != nil || list[0].CreatedByName != nil {
		t.Fatal("creator must be cleared")
	}
	reportsN, insightsN, err := f.reports.PurgeClient(ctx, client.ID)
	if err != nil || reportsN != 1 || insightsN != 1 {
		t.Fatalf("purge %d %d %v", reportsN, insightsN, err)
	}
	list, _ = f.reports.List(ctx, "viewer", client.ID)
	if len(list) != 0 {
		t.Fatal("reports left after purge")
	}
}
