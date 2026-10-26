package app_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/crawler"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/metagraph"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/pagespeed"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/app"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/sync"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

const (
	clientA = "client-a"
	clientB = "client-b"
	manager = "u-manager"
	viewer  = "u-viewer"
)

type fakeAccess struct{}

func (fakeAccess) Check(_ context.Context, userID, clientID string) (*domain.ClientAccess, error) {
	if clientID != clientA && clientID != clientB {
		return nil, apperr.New(apperr.NotFound, "Klien tidak ditemukan.")
	}
	switch userID {
	case manager:
		return &domain.ClientAccess{ClientID: clientID, Slug: clientID, Currency: "IDR", CanManage: true}, nil
	case viewer:
		return &domain.ClientAccess{ClientID: clientID, Slug: clientID, Currency: "IDR"}, nil
	}
	return nil, apperr.New(apperr.Unauthorized, "")
}

type fakeInteg struct{ metaConn string }

func (fakeInteg) Credentials(context.Context, string) (domain.Credentials, error) {
	return domain.Credentials{}, nil
}
func (fakeInteg) AccessToken(context.Context, string) (string, error) { return "", nil }
func (f fakeInteg) FindClientConnection(_ context.Context, _, provider string) (string, error) {
	if provider == domain.IntegrationMeta {
		return f.metaConn, nil
	}
	return "", nil
}

type fixture struct {
	store   *postgres.Store
	deps    app.Deps
	sources *app.Sources
	clk     clock.Fixed
	w       app.Window
}

func day(s string) time.Time { d, _ := dates.Parse(s); return d }
func ip(n int) *int          { return &n }
func sp(s string) *string    { return &s }

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := pg.TestPool(t, analytics.Migrations(), "migrations", analytics.Schema)
	pg.TruncateSchema(t, pool, analytics.Schema)
	store := postgres.NewStore(pool)
	clk := clock.Fixed{T: time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)}
	integ := fakeInteg{}
	syncer := sync.New(store, integ, clk, slog.Default(), "")
	syncer.Graph = metagraph.Mock{Now: clk.Now}
	d := app.Deps{Store: store, Access: fakeAccess{}, Integ: integ, Syncer: syncer, Clock: clk}
	cur := dates.Range{From: day("2026-09-01"), To: day("2026-09-07")}
	return &fixture{store: store, deps: d, sources: app.NewSources(d), clk: clk, w: app.Window{Current: cur, Previous: dates.Previous(cur)}}
}

func mustCode(t *testing.T, err error, code apperr.Code) *apperr.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got nil", code)
	}
	e := apperr.As(err)
	if e.Code != code {
		t.Fatalf("expected %s, got %s (%s)", code, e.Code, e.Message)
	}
	return e
}

// seed creates one client with an IG account, an SEO property and an ad account.
func (f *fixture) seed(t *testing.T) (igID, propID, adAccID string) {
	ctx := context.Background()
	s := f.store
	ig, err := s.InsertSocialAccount(ctx, domain.SocialAccount{ID: ids.New(), ClientID: clientA, Platform: domain.Instagram, ExternalID: "instagram:kopi", Username: "kopi", DisplayName: "Kopi"})
	if err != nil {
		t.Fatal(err)
	}
	for _, sn := range []domain.SocialSnapshot{
		{AccountID: ig.ID, Date: day("2026-08-24"), Followers: 1000, Reach: ip(10)},
		{AccountID: ig.ID, Date: day("2026-08-31"), Followers: 1100, Reach: ip(20)},
		{AccountID: ig.ID, Date: day("2026-09-03"), Followers: 1200, Reach: ip(30), Impressions: ip(60)},
		{AccountID: ig.ID, Date: day("2026-09-07"), Followers: 1300, Reach: ip(40), Impressions: ip(80)},
	} {
		if err := s.UpsertSnapshot(ctx, sn, domain.SnapshotPatch{}); err != nil {
			t.Fatal(err)
		}
	}
	posts := []domain.SocialPost{
		{AccountID: ig.ID, ExternalID: "p1", Caption: "satu", MediaType: "VIDEO", ProductType: sp("REELS"), PublishedAt: time.Date(2026, 9, 2, 3, 0, 0, 0, time.UTC), Likes: 100, Comments: 30},
		{AccountID: ig.ID, ExternalID: "p2", Caption: "dua", MediaType: "IMAGE", PublishedAt: time.Date(2026, 9, 7, 15, 0, 0, 0, time.UTC), Likes: 40},
		{AccountID: ig.ID, ExternalID: "p0", Caption: "lama", MediaType: "IMAGE", PublishedAt: time.Date(2026, 8, 28, 3, 0, 0, 0, time.UTC), Likes: 10},
	}
	if err := s.InsertPostsIgnore(ctx, posts); err != nil {
		t.Fatal(err)
	}
	prop, err := s.InsertSeoProperty(ctx, domain.SeoProperty{ID: ids.New(), ClientID: clientA, SiteURL: "https://kopi.id/"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertSeoDaily(ctx, []domain.SeoDailyMetric{
		{PropertyID: prop.ID, Date: day("2026-09-01"), Clicks: 10, Impressions: 100, CTR: 10, Position: 5},
		{PropertyID: prop.ID, Date: day("2026-09-03"), Clicks: 0, Impressions: 300, CTR: 0, Position: 9},
		{PropertyID: prop.ID, Date: day("2026-08-30"), Clicks: 5, Impressions: 50, CTR: 10, Position: 4},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertSeoDimension(ctx, []domain.SeoDimensionMetric{
		{PropertyID: prop.ID, Date: day("2026-09-01"), Dimension: domain.DimQuery, Key: "kopi susu", Clicks: 8, Impressions: 80, Position: 3},
		{PropertyID: prop.ID, Date: day("2026-09-02"), Dimension: domain.DimQuery, Key: "kopi susu", Clicks: 2, Impressions: 20, Position: 5},
		{PropertyID: prop.ID, Date: day("2026-09-01"), Dimension: domain.DimQuery, Key: "kedai", Clicks: 1, Impressions: 300, Position: 7},
	}); err != nil {
		t.Fatal(err)
	}
	acc, err := s.InsertAdAccount(ctx, domain.AdAccount{ID: ids.New(), ClientID: clientA, ExternalID: "act_1", Name: "Utama", Currency: "IDR"})
	if err != nil {
		t.Fatal(err)
	}
	cid, _ := s.UpsertCampaign(ctx, domain.AdCampaign{AccountID: acc.ID, ExternalID: "c1", Name: "Traffic", Objective: "OUTCOME_TRAFFIC", Status: "ACTIVE"}, true)
	sid, _ := s.UpsertAdSet(ctx, domain.AdSet{CampaignID: cid, ExternalID: "s1", Name: "Broad", Status: "ACTIVE"}, true)
	aid, _ := s.UpsertAd(ctx, domain.Ad{AdSetID: sid, ExternalID: "a1", Name: "Video", Status: "ACTIVE"}, true)
	cid2, _ := s.UpsertCampaign(ctx, domain.AdCampaign{AccountID: acc.ID, ExternalID: "c2", Name: "Awareness", Objective: "OUTCOME_AWARENESS", Status: "ACTIVE"}, true)
	sid2, _ := s.UpsertAdSet(ctx, domain.AdSet{CampaignID: cid2, ExternalID: "s2", Name: "Reach", Status: "ACTIVE"}, true)
	aid2, _ := s.UpsertAd(ctx, domain.Ad{AdSetID: sid2, ExternalID: "a2", Name: "Banner", Status: "ACTIVE"}, true)
	if err := s.UpsertInsights(ctx, []domain.AdDailyInsight{
		{AccountID: acc.ID, Date: day("2026-09-01"), CampaignID: &cid, AdSetID: &sid, AdID: &aid, Spend: 100, Impressions: 1000, Reach: 500, LinkClicks: 20, Results: 10, ResultType: sp("link_click")},
		{AccountID: acc.ID, Date: day("2026-09-02"), CampaignID: &cid2, AdSetID: &sid2, AdID: &aid2, Spend: 50, Impressions: 5000, Reach: 5000, Results: 5000, ResultType: sp("reach")},
		{AccountID: acc.ID, Date: day("2026-09-02"), CampaignID: &cid, Spend: 999},
		{AccountID: acc.ID, Date: day("2026-08-30"), CampaignID: &cid, AdSetID: &sid, AdID: &aid, Spend: 80, Results: 4, ResultType: sp("link_click")},
	}); err != nil {
		t.Fatal(err)
	}
	return ig.ID, prop.ID, acc.ID
}

func TestIntegrationOverview(t *testing.T) {
	f := setup(t)
	f.seed(t)
	ctx := context.Background()
	o, err := app.NewOverviews(f.deps).Get(ctx, viewer, clientA, f.w)
	if err != nil {
		t.Fatal(err)
	}
	s := o.Social
	if !s.HasSource || !s.IsDemo || s.Quality.State != domain.StateDemo || s.Followers != 1300 {
		t.Fatalf("social %+v", s)
	}
	if s.FollowersDelta.Current != 200 || s.FollowersDelta.Previous != 100 || *s.FollowersDelta.Pct != 100 {
		t.Fatalf("followers delta compares growth: %+v", s.FollowersDelta)
	}
	if s.Posts != 2 || s.Reach != 70 || len(s.FollowersSpark) != 2 || s.FollowersSpark[1] != 1300 {
		t.Fatalf("social %+v", s)
	}
	seo := o.Seo
	if seo.Clicks != 10 || seo.Impressions != 400 || seo.Daily[1].Clicks != nil || *seo.Daily[2].Impressions != 300 {
		t.Fatalf("seo %+v", seo)
	}
	if seo.ClicksSpark[1] != 0 || seo.Health != nil {
		t.Fatal("spark turns null into 0; no audit -> no health")
	}
	ads := o.Ads
	if ads.Kpis.Spend != 150 || ads.Kpis.Results != 10 || ads.Kpis.CPR != 10 || ads.Prev.Spend != 80 {
		t.Fatalf("ads %+v", ads.Kpis)
	}
	if ads.Daily[1].Results != 0 || ads.Daily[1].Spend != 50 {
		t.Fatalf("daily zeroes non-conversion results: %+v", ads.Daily[1])
	}
	empty, err := app.NewOverviews(f.deps).Get(ctx, viewer, clientB, f.w)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Social.HasSource || empty.Social.Quality.State != domain.StateNotConnected || empty.Seo.Daily == nil {
		t.Fatal("empty client")
	}
	if _, err := app.NewOverviews(f.deps).Get(ctx, "stranger", clientA, f.w); err == nil {
		t.Fatal("stranger allowed")
	}
	tot, err := app.NewOverviews(f.deps).Portfolio(ctx, viewer, []string{clientA, clientB, "nope"}, f.w)
	if err != nil {
		t.Fatal(err)
	}
	if tot.Followers != 1300 || tot.Clicks != 10 || tot.Spend != 150 || tot.SpendDelta.Previous != 80 {
		t.Fatalf("portfolio %+v", tot)
	}
}

func TestIntegrationDashboards(t *testing.T) {
	f := setup(t)
	igID, propID, _ := f.seed(t)
	ctx := context.Background()
	d, err := app.NewSocial(f.deps).Dashboard(ctx, viewer, clientA, domain.Instagram, f.w)
	if err != nil {
		t.Fatal(err)
	}
	if d.Account.ID != igID || d.Kpis.Followers != 1300 || d.Kpis.GrowthAbs != 200 || d.Kpis.Impressions != 140 || len(d.GrowthSeries) != 7 {
		t.Fatalf("%+v", d.Kpis)
	}
	if *d.GrowthSeries[0].Followers != 1100 || *d.GrowthSeries[0].FollowersPrev != 1000 {
		t.Fatalf("growth series %+v", d.GrowthSeries[0])
	}
	if d.TopPosts[0].ExternalID != "p1" || d.TopPosts[0].Type != "Reels" || d.TopPosts[0].ER != 10 {
		t.Fatalf("top posts %+v", d.TopPosts[0])
	}
	so, err := app.NewSeo(f.deps).Overview(ctx, viewer, clientA, propID, f.w)
	if err != nil {
		t.Fatal(err)
	}
	if so.Daily[1].Clicks != nil || so.Daily[1].Position != nil || *so.Daily[0].Clicks != 10 {
		t.Fatalf("null days must stay null: %+v", so.Daily[1])
	}
	if so.Kpis.Clicks != 10 || so.Clicks.Previous != 5 || len(so.Ga4Daily) != 7 || so.Ga4Daily[0].Organik != 0 {
		t.Fatalf("%+v", so)
	}
	qs, err := app.NewSeo(f.deps).Queries(ctx, viewer, clientA, propID, f.w)
	if err != nil {
		t.Fatal(err)
	}
	if qs.Rows[0].Key != "kopi susu" || qs.Rows[0].Clicks != 10 || len(qs.Striking) != 1 {
		t.Fatalf("%+v", qs)
	}
	if _, err := app.NewSeo(f.deps).Overview(ctx, viewer, clientB, propID, f.w); apperr.As(err).Code != apperr.NotFound {
		t.Fatal("foreign property must be NOT_FOUND")
	}
	ads, err := app.NewAds(f.deps).Dashboard(ctx, viewer, clientA, f.w, "")
	if err != nil {
		t.Fatal(err)
	}
	if ads.Kpis.Results != 10 || *ads.DominantResultType != "link_click" || len(ads.Breakdown) != 2 || ads.Breakdown[0].Name != "Traffic" {
		t.Fatalf("%+v", ads.Kpis)
	}
	if ads.Daily[0].SpendPrev == nil || len(ads.TopAds) != 1 || ads.Deltas.ROAS != nil {
		t.Fatalf("%+v", ads.Daily[0])
	}
	camp := ads.Breakdown[0].ID
	filtered, err := app.NewAds(f.deps).Dashboard(ctx, viewer, clientA, f.w, camp)
	if err != nil || filtered.Kpis.Spend != 100 || len(filtered.Breakdown) != 1 {
		t.Fatalf("campaign filter %v %+v", err, filtered.Kpis)
	}
	ignored, _ := app.NewAds(f.deps).Dashboard(ctx, viewer, clientA, f.w, "stale-id")
	if ignored.Kpis.Spend != 150 {
		t.Fatal("unknown campaign filter is ignored")
	}
}

func TestIntegrationSources(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	src := f.sources
	if _, err := src.AddSocialAccount(ctx, viewer, clientA, domain.Instagram, "kopi", ""); apperr.As(err).Code != apperr.Unauthorized {
		t.Fatal("viewer may not add sources")
	}
	a, err := src.AddSocialAccount(ctx, manager, clientA, domain.Instagram, " @Kopi ", "")
	if err != nil {
		t.Fatal(err)
	}
	if a.ExternalID != "instagram:kopi" || a.DisplayName != "kopi" {
		t.Fatalf("%+v", a)
	}
	e := mustCode(t, func() error {
		_, err := src.AddSocialAccount(ctx, manager, clientA, domain.Instagram, "kopi", "")
		return err
	}(), apperr.Validation)
	if e.FieldErrors["username"][0] != "Sudah terdaftar untuk klien ini" {
		t.Fatalf("%+v", e.FieldErrors)
	}
	mustCode(t, func() error { _, err := src.AddSeoProperty(ctx, manager, clientA, "kopi", ""); return err }(), apperr.Validation)
	p, err := src.AddSeoProperty(ctx, manager, clientA, "sc-domain:kopi.id", " 123 ")
	if err != nil || *p.Ga4PropertyID != "123" {
		t.Fatal(err)
	}
	e = mustCode(t, func() error { _, err := src.AddSeoProperty(ctx, manager, clientA, "sc-domain:kopi.id", ""); return err }(), apperr.Validation)
	if len(e.FieldErrors["siteUrl"]) != 1 {
		t.Fatal("duplicate site url field")
	}
	ad, err := src.AddAdAccount(ctx, manager, clientA, "act_9", "Utama")
	if err != nil || ad.Currency != "IDR" {
		t.Fatal(err)
	}
	mustCode(t, func() error { _, err := src.AddAdAccount(ctx, manager, clientA, "act_9", "Lagi"); return err }(), apperr.Validation)
	if c, err := src.SetAuditCadence(ctx, viewer, clientA, p.ID, domain.CadenceWeekly); err != nil || c != domain.CadenceWeekly {
		t.Fatalf("viewer may set cadence: %v", err)
	}
	cs, err := src.GetClientSources(ctx, viewer, clientA)
	if err != nil || len(cs.Social) != 1 || len(cs.Seo) != 1 || len(cs.Ads) != 1 || cs.Seo[0].AuditCadence != domain.CadenceWeekly {
		t.Fatalf("%v %+v", err, cs)
	}
	mustCode(t, src.RemoveSocialAccount(ctx, manager, clientB, a.ID), apperr.NotFound)
	if err := src.RemoveSocialAccount(ctx, manager, clientA, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := src.RemoveSeoProperty(ctx, manager, clientA, p.ID); err != nil {
		t.Fatal(err)
	}
	if err := src.RemoveAdAccount(ctx, manager, clientA, ad.ID); err != nil {
		t.Fatal(err)
	}
	// Competitors: business discovery mock + 120 days of history.
	mustCode(t, func() error { _, _, err := src.AddCompetitor(ctx, manager, clientA, ".bad"); return err }(), apperr.Validation)
	id, username, err := src.AddCompetitor(ctx, manager, clientA, "@KopiKenangan")
	if err != nil || username != "kopikenangan" {
		t.Fatal(err)
	}
	snaps, _ := f.store.ListSnapshots(ctx, []string{id}, day("2026-01-01"), day("2026-12-31"))
	if len(snaps) != 120 || !snaps[119].Date.Equal(day("2026-10-20")) {
		t.Fatalf("history %d", len(snaps))
	}
	e = mustCode(t, func() error { _, _, err := src.AddCompetitor(ctx, manager, clientA, "kopikenangan"); return err }(), apperr.Validation)
	if e.Message != "Kompetitor dengan username ini sudah ada." {
		t.Fatal(e.Message)
	}
	comp, err := app.NewSocial(f.deps).Competitors(ctx, viewer, clientA, app.Window{Current: dates.Range{From: day("2026-10-01"), To: day("2026-10-20")}, Previous: dates.Range{From: day("2026-09-11"), To: day("2026-09-30")}})
	if err != nil || comp.HasOwn || len(comp.Rows) != 1 || len(comp.Chart) != 20 || comp.Series[0][1] != "@kopikenangan" || *comp.Chart[0].Values[0] != 100 {
		t.Fatalf("%v %+v", err, comp.Rows)
	}
	mustCode(t, src.RemoveCompetitor(ctx, viewer, clientA, id), apperr.Unauthorized)
	if err := src.RemoveCompetitor(ctx, manager, clientA, id); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationSyncSocialDemo(t *testing.T) {
	f := setup(t)
	igID, _, _ := f.seed(t)
	ctx := context.Background()
	social := app.NewSocial(f.deps)
	mustCode(t, func() error { _, err := social.Sync(ctx, viewer, clientA, domain.TikTok); return err }(), apperr.NotFound)
	n, err := social.Sync(ctx, viewer, clientA, domain.Instagram)
	if err != nil || n != 1 {
		t.Fatalf("%v %d", err, n)
	}
	snaps, _ := f.store.ListRecentSnapshots(ctx, igID, 1)
	if !snaps[0].Date.Equal(day("2026-10-21")) || snaps[0].Followers < 1300 {
		t.Fatalf("%+v", snaps[0])
	}
	jobs, _ := f.store.ListSyncJobs(ctx, clientA, 5)
	if len(jobs) != 1 || jobs[0].Kind != domain.KindSocialSnapshot || jobs[0].Status != domain.JobSuccess || *jobs[0].Message != "1 akun diperbarui (instagram)" {
		t.Fatalf("%+v", jobs)
	}
	// A second run the same day upserts the same (account, day) row.
	if _, err := social.Sync(ctx, viewer, clientA, domain.Instagram); err != nil {
		t.Fatal(err)
	}
	all, _ := f.store.ListSnapshots(ctx, []string{igID}, day("2026-10-21"), day("2026-10-21"))
	if len(all) != 1 {
		t.Fatalf("snapshots today %d", len(all))
	}
}

const csvFixture = "Campaign name,Ad set name,Ad name,Day,Amount spent (IDR),Impressions,Reach,Link clicks,Results,Result indicator,Age,Gender\n" +
	"Promo Kopi,Broad,Video A,2026-09-01,\"150,000\",\"12,000\",8000,240,24,actions:link_click,,\n" +
	"Promo Kopi,Broad,Video A,2026-09-02,100000,10000,7000,200,20,actions:link_click,,\n" +
	"Promo Kopi,,,2026-09-01,50000,4000,3000,80,8,actions:link_click,25-34,female\n"

func TestIntegrationImportAdsCsv(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	ads := app.NewAds(f.deps)
	mustCode(t, func() error {
		_, err := ads.ImportCsv(ctx, viewer, clientA, "a.csv", "text/csv", []byte(csvFixture))
		return err
	}(), apperr.Unauthorized)
	e := mustCode(t, func() error {
		_, err := ads.ImportCsv(ctx, manager, clientA, "a.xlsx", "application/vnd", []byte(csvFixture))
		return err
	}(), apperr.Validation)
	if e.Message != "File harus berformat .csv" {
		t.Fatal(e.Message)
	}
	sum, err := ads.ImportCsv(ctx, manager, clientA, "export.csv", "", []byte(csvFixture))
	if err != nil {
		t.Fatal(err)
	}
	if sum.Campaigns != 1 || sum.AdSets != 1 || sum.Ads != 1 || sum.Days != 2 || sum.Demographics != 1 {
		t.Fatalf("%+v", sum)
	}
	if _, err := ads.ImportCsv(ctx, manager, clientA, "export.csv", "", []byte(csvFixture)); err != nil {
		t.Fatal(err)
	}
	d, err := ads.Dashboard(ctx, viewer, clientA, app.Window{Current: dates.Range{From: day("2026-09-01"), To: day("2026-09-02")}, Previous: dates.Range{From: day("2026-08-30"), To: day("2026-08-31")}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if d.Kpis.Spend != 250000 || d.Kpis.Results != 44 || !d.Demographics.HasData || d.Account.Name != "Impor CSV" {
		t.Fatalf("re-import must not duplicate: %+v", d.Kpis)
	}
	if d.Breakdown[0].Objective != "OUTCOME_TRAFFIC" {
		t.Fatal(d.Breakdown[0].Objective)
	}
	jobs, _ := f.store.ListSyncJobs(ctx, clientA, 5)
	if jobs[0].Kind != domain.KindAdsCsvImport || *jobs[0].Message != "1 kampanye, 1 set iklan, 1 iklan, 2 hari data, 1 baris demografi." {
		t.Fatalf("%+v", *jobs[0].Message)
	}
	at, err := ads.Sync(ctx, manager, clientA)
	if err != nil || !at.Equal(f.clk.T) {
		t.Fatal(err)
	}
	jobs, _ = f.store.ListSyncJobs(ctx, clientA, 1)
	if *jobs[0].Message != "Sinkronisasi demo: 1 kampanye diperbarui." {
		t.Fatal(*jobs[0].Message)
	}
}

func TestIntegrationAuditAndReport(t *testing.T) {
	f := setup(t)
	_, propID, _ := f.seed(t)
	ctx := context.Background()
	f.deps.Syncer.Pagespeed = func(_ context.Context, target, strategy string) (*pagespeed.Result, error) {
		if strategy == "desktop" {
			return nil, apperr.FromProvider("pagespeed", apperr.RateLimit, "Kuota PageSpeed Insights tercapai. Coba lagi beberapa menit.", nil)
		}
		return &pagespeed.Result{Strategy: strategy, Scores: pagespeed.Scores{Performance: 80, Seo: 90, Accessibility: 100, BestPractices: 100}, Raw: []byte(`{}`)}, nil
	}
	f.deps.Syncer.Crawl = func(_ context.Context, start string, _ crawler.Options) (*crawler.Result, error) {
		return &crawler.Result{PagesCrawled: 1, Pages: []crawler.Page{{URL: start, StatusCode: 200}}, Issues: []crawler.Issue{{Code: "MISSING_H1", Severity: crawler.SeverityError, Message: "x", URL: start}}}, nil
	}
	seo := app.NewSeo(f.deps)
	res, err := seo.RunAudit(ctx, viewer, clientA, propID, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(res.Message, "Audit selesai sebagian: PageSpeed 1/2 strategi · Crawl 1 halaman, 1 isu — Kuota") {
		t.Fatal(res.Message)
	}
	mustCode(t, func() error { _, err := seo.RunAudit(ctx, viewer, clientA, propID, ""); return err }(), apperr.RateLimit)
	audit, err := seo.Audit(ctx, viewer, clientA, propID)
	if err != nil || audit.LatestMobile == nil || audit.Counts.Errors != 1 || len(audit.Pages) != 1 {
		t.Fatalf("%v %+v", err, audit)
	}
	o, _ := app.NewOverviews(f.deps).Seo(ctx, clientA, f.w)
	if o.Health == nil || *o.Health != 88 {
		t.Fatalf("health %v", o.Health)
	}
	jobs, _ := f.store.ListSyncJobs(ctx, clientA, 1)
	if jobs[0].Kind != domain.KindSeoAudit || jobs[0].Status != domain.JobSuccess {
		t.Fatalf("%+v", jobs[0])
	}

	rd, err := app.NewReports(f.deps).Data(ctx, clientA, f.w, nil, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(rd.Social.DailyFollowers) != 8 || rd.Social.DailyFollowers[0].Followers != 1100 || rd.Social.TopPosts[0].ExternalID != "p1" {
		t.Fatalf("%+v", rd.Social.DailyFollowers)
	}
	if rd.Seo.TopQueries[0].Query != "kopi susu" || rd.Seo.TopQueries[0].Clicks != 10 {
		t.Fatalf("%+v", rd.Seo.TopQueries)
	}
	if len(rd.Ads.Campaigns) != 2 || rd.Ads.Campaigns[0].Name != "Traffic" || rd.Ads.Campaigns[1].CPR != 0 || rd.Ads.Campaigns[1].IsConversion {
		t.Fatalf("%+v", rd.Ads.Campaigns)
	}
	only, _ := app.NewReports(f.deps).Data(ctx, clientA, f.w, []string{"SEO"}, 1)
	if only.Social != nil || only.Ads != nil || len(only.Seo.TopQueries) != 1 {
		t.Fatal("module filter")
	}
	posts, err := f.sources.ListSocialPosts(ctx, domain.PostFilter{ClientID: clientA, ExternalIDs: []string{"p1", "p2"}}, "2026-09-02", "2026-09-07")
	if err != nil || len(posts) != 2 || posts[0].ExternalID != "p2" {
		t.Fatalf("%v %+v", err, posts)
	}
}

func TestIntegrationLifecycle(t *testing.T) {
	f := setup(t)
	f.seed(t)
	ctx := context.Background()
	if err := f.sources.OnClientCreated(ctx, clientB, "https://b.id/", []string{"SOCIAL", "SEO"}); err != nil {
		t.Fatal(err)
	}
	if err := f.sources.OnClientCreated(ctx, clientB, "https://b.id/", []string{"SEO"}); err != nil {
		t.Fatal("duplicate must be ignored")
	}
	_ = f.sources.OnClientCreated(ctx, clientB, "https://c.id/", []string{"ADS"})
	props, _ := f.store.ListSeoProperties(ctx, clientB)
	if len(props) != 1 {
		t.Fatalf("%d", len(props))
	}
	now := f.clk.T
	old := now.AddDate(0, 0, -91)
	_, _ = f.store.InsertSyncJob(ctx, domain.SyncJob{ClientID: clientA, Kind: "SEO_GSC", Status: domain.JobSuccess, StartedAt: old})
	_, _ = f.store.InsertSyncJob(ctx, domain.SyncJob{ClientID: clientA, Kind: "SEO_GSC", Status: domain.JobSuccess, StartedAt: now})
	if n, err := f.sources.RunRetention(ctx); err != nil || n != 1 {
		t.Fatalf("retention %v %d", err, n)
	}
	if err := f.sources.PurgeClient(ctx, clientA); err != nil {
		t.Fatal(err)
	}
	accs, _ := f.store.ListSocialAccounts(ctx, clientA, true, nil)
	p, _ := f.store.ListSeoProperties(ctx, clientA)
	ads, _ := f.store.ListAdAccounts(ctx, clientA)
	jobs, _ := f.store.ListSyncJobs(ctx, clientA, 10)
	if len(accs)+len(p)+len(ads)+len(jobs) != 0 {
		t.Fatal("purge left rows")
	}
	if props, _ := f.store.ListSeoProperties(ctx, clientB); len(props) != 1 {
		t.Fatal("purge touched another client")
	}
}

// TestDevOverviewSanity reads the imported dev copy (set WHITE_DEV_DATABASE_URL).
func TestDevOverviewSanity(t *testing.T) {
	url := os.Getenv("WHITE_DEV_DATABASE_URL")
	if url == "" {
		t.Skip("WHITE_DEV_DATABASE_URL tidak diset")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var clientID string
	if err := pool.QueryRow(ctx, `SELECT id FROM identity.clients WHERE slug = 'kopi-nusantara'`).Scan(&clientID); err != nil {
		t.Fatal(err)
	}
	store := postgres.NewStore(pool)
	clk := clock.Fixed{T: time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)}
	d := app.Deps{Store: store, Access: devAccess{}, Integ: fakeInteg{}, Clock: clk}
	r := dates.ForPreset("28d", clk.Now())
	o, err := app.NewOverviews(d).Get(ctx, "admin", clientID, app.Window{Current: r, Previous: dates.Previous(r)})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("range %s..%s", dates.Format(r.From), dates.Format(r.To))
	t.Logf("social followers=%d delta=%+v posts=%d er=%.2f reach=%d state=%s", o.Social.Followers, o.Social.FollowersDelta, o.Social.Posts, o.Social.EngagementRate, o.Social.Reach, o.Social.Quality.State)
	t.Logf("seo clicks=%d impressions=%d ctr=%.2f position=%.2f health=%v state=%s", o.Seo.Clicks, o.Seo.Impressions, o.Seo.CTR, o.Seo.Position, o.Seo.Health, o.Seo.Quality.State)
	t.Logf("ads spend=%.0f results=%d cpr=%.0f state=%s", o.Ads.Kpis.Spend, o.Ads.Kpis.Results, o.Ads.Kpis.CPR, o.Ads.Quality.State)
	if o.Social.Followers == 0 || o.Seo.Clicks == 0 || o.Ads.Kpis.Spend == 0 {
		t.Fatal("expected non-zero followers, clicks and spend for kopi-nusantara")
	}
}

type devAccess struct{}

func (devAccess) Check(_ context.Context, _, clientID string) (*domain.ClientAccess, error) {
	return &domain.ClientAccess{ClientID: clientID, CanManage: true}, nil
}
