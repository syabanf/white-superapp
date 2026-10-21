package app_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/providers/seodata"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/app"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

const (
	clientID   = "client-1"
	slug       = "kopi"
	propertyID = "prop-1"
	manager    = "user-manager"
	viewer     = "user-viewer"
)

type fakeAccess struct{}

func (fakeAccess) Check(_ context.Context, userID, _ string) (*app.Access, error) {
	switch userID {
	case manager:
		return &app.Access{Role: "MANAGER", CanManage: true, ClientSlug: slug}, nil
	case viewer:
		return &app.Access{Role: "VIEWER", CanManage: false, ClientSlug: slug}, nil
	}
	return nil, apperr.New(apperr.Unauthorized, "")
}

type fakeProps struct{ props []app.Property }

func (f fakeProps) ListProperties(_ context.Context, cid string) ([]app.Property, error) {
	var out []app.Property
	for _, p := range f.props {
		if p.ClientID == cid {
			out = append(out, p)
		}
	}
	return out, nil
}

type fakeNotifier struct{ sent []app.Notification }

func (f *fakeNotifier) Notify(_ context.Context, n app.Notification) error {
	f.sent = append(f.sent, n)
	return nil
}

// scripted is the mock with SERP positions and backlinks under test control.
type scripted struct {
	*seodata.Mock
	positions map[string]*int32
	links     []domain.BacklinkItem
	summary   *domain.BacklinkSummary
}

func (s *scripted) SerpRanks(ctx context.Context, items []domain.SerpRankRequest, loc int32, lang, target string) ([]domain.RankResult, error) {
	out, err := s.Mock.SerpRanks(ctx, items, loc, lang, target)
	for i := range out {
		if pos, ok := s.positions[out[i].Keyword]; ok {
			out[i].Position = pos
		}
	}
	return out, err
}

func (s *scripted) Backlinks(ctx context.Context, dom string, limit int, mode domain.BacklinkMode) ([]domain.BacklinkItem, error) {
	if s.links != nil {
		return s.links, nil
	}
	return s.Mock.Backlinks(ctx, dom, limit, mode)
}

func (s *scripted) BacklinkSummary(ctx context.Context, dom string) (domain.BacklinkSummary, error) {
	if s.summary != nil {
		return *s.summary, nil
	}
	return s.Mock.BacklinkSummary(ctx, dom)
}

type fixture struct {
	store       *postgres.Store
	clk         *clock.Fixed
	provider    *scripted
	notifier    *fakeNotifier
	keywords    *app.Keywords
	research    *app.Research
	backlinks   *app.Backlinks
	competitors *app.Competitors
	reports     *app.Reports
	daily       *app.Daily
	property    app.Property
}

func setup(t *testing.T) *fixture {
	t.Helper()
	pool := pg.TestPool(t, seosuite.Migrations(), "migrations", seosuite.Schema)
	pg.TruncateSchema(t, pool, seosuite.Schema)
	store := postgres.NewStore(pool)
	clk := &clock.Fixed{T: time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)}
	provider := &scripted{Mock: seodata.NewMock(clk), positions: map[string]*int32{}}
	seo := seodata.Static(provider, seodata.ModeMock)
	property := app.Property{ID: propertyID, ClientID: clientID, SiteURL: "https://kopinusantara.id/", LocationCode: 2360, LanguageCode: "id", CreatedAt: clk.T}
	props := fakeProps{props: []app.Property{property, {ID: "prop-2", ClientID: clientID, SiteURL: "https://other.id/", LocationCode: 2360, LanguageCode: "id", CreatedAt: clk.T.Add(time.Hour)}}}
	guard := app.NewGuard(fakeAccess{}, props)
	jobs := app.NewJobs(store, clk)
	tracker := app.NewTracker(store, seo)
	views := app.NewViews(store)
	notifier := &fakeNotifier{}
	return &fixture{
		store: store, clk: clk, provider: provider, notifier: notifier, property: property,
		keywords:    app.NewKeywords(store, guard, seo, tracker, jobs, views, clk),
		research:    app.NewResearch(store, guard, seo, clk),
		backlinks:   app.NewBacklinks(guard, seo, tracker, jobs, views, clk),
		competitors: app.NewCompetitors(store, guard, seo, tracker, jobs, clk),
		reports:     app.NewReports(views),
		daily:       app.NewDaily(store, props, tracker, notifier),
	}
}

func code(t *testing.T, err error, want apperr.Code, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s, got nil", want)
	}
	e := apperr.As(err)
	if e.Code != want || (msg != "" && e.Message != msg) {
		t.Fatalf("expected %s %q, got %s %q", want, msg, e.Code, e.Message)
	}
}

func p(v int32) *int32 { return &v }

func (f *fixture) rng() dates.Resolved { return dates.Resolve(nil, f.clk.Now()) }

func TestIntegrationKeywordsAndRanks(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	today := dates.Today(f.clk.Now())
	yesterday := dates.AddDays(today, -1)
	f.provider.positions["kopi susu gula aren"] = p(1)
	f.provider.positions["kedai kopi jakarta"] = nil
	f.provider.positions["cold brew"] = p(3)

	if _, err := f.keywords.Add(ctx, viewer, clientID, propertyID, "kopi susu gula aren", domain.DeviceMobile, ""); err == nil {
		t.Fatal("viewer must not add keywords")
	} else {
		code(t, err, apperr.Unauthorized, "Hanya peran ADMIN/MANAGER yang dapat mengubah.")
	}
	if _, err := f.keywords.Add(ctx, manager, clientID, "missing", "x", domain.DeviceMobile, ""); err == nil {
		t.Fatal("unknown property")
	} else {
		code(t, err, apperr.NotFound, "Properti SEO tidak ditemukan.")
	}
	if _, err := f.keywords.Add(ctx, manager, clientID, propertyID, " , \n", domain.DeviceMobile, ""); err == nil {
		t.Fatal("empty text")
	} else {
		code(t, err, apperr.Validation, "Masukkan minimal satu kata kunci (maks. 100 per tambah).")
	}

	added, err := f.keywords.Add(ctx, manager, clientID, propertyID, "Kopi Susu Gula Aren\nkedai kopi jakarta, cold  brew\nkopi susu gula aren", "", "Brand, produk")
	if err != nil {
		t.Fatal(err)
	}
	if added != 3 {
		t.Fatalf("added %d", added)
	}
	again, err := f.keywords.Add(ctx, manager, clientID, propertyID, "cold brew", domain.DeviceMobile, "")
	if err != nil || again != 0 {
		t.Fatalf("duplicate add: %d %v", again, err)
	}
	keywords, _ := f.store.ListKeywords(ctx, propertyID)
	if len(keywords) != 3 || keywords[0].Keyword != "kopi susu gula aren" || keywords[0].Volume == nil || len(keywords[0].Tags) != 2 {
		t.Fatalf("%+v", keywords)
	}
	byKw := map[string]domain.TrackedKeyword{}
	for _, k := range keywords {
		byKw[k.Keyword] = k
	}
	// Yesterday's snapshots give the change kinds something to compare with.
	if err := f.store.UpsertRankSnapshots(ctx, []domain.RankSnapshot{
		{ID: ids.New(), ClientID: clientID, KeywordID: byKw["kopi susu gula aren"].ID, Date: yesterday, Position: p(20)},
		{ID: ids.New(), ClientID: clientID, KeywordID: byKw["kedai kopi jakarta"].ID, Date: yesterday, Position: p(5)},
	}); err != nil {
		t.Fatal(err)
	}

	updated, err := f.keywords.RefreshRanks(ctx, viewer, clientID, propertyID)
	if err != nil || updated != 3 {
		t.Fatalf("refresh: %d %v", updated, err)
	}
	_, err = f.keywords.RefreshRanks(ctx, viewer, clientID, propertyID)
	code(t, err, apperr.RateLimit, "Pembaruan terakhir kurang dari 10 menit lalu. Coba lagi beberapa menit.")
	jobs, err := f.keywords.ListJobs(ctx, viewer, clientID, nil, 10)
	if err != nil || len(jobs) != 1 || jobs[0].Kind != domain.JobKindRanks || jobs[0].Status != domain.JobSuccess || *jobs[0].Message != "3 kata kunci" {
		t.Fatalf("%+v %v", jobs, err)
	}

	data, err := f.keywords.RankData(ctx, viewer, clientID, propertyID, f.rng())
	if err != nil {
		t.Fatal(err)
	}
	if !data.HasKeywords || data.LatestDate == nil || !data.LatestDate.Equal(today) || data.Mode != seodata.ModeMock {
		t.Fatalf("%+v", data.RankData)
	}
	if data.Kpis.Visibility != 13 || data.Kpis.AvgPosition == nil || *data.Kpis.AvgPosition != 2 || data.Kpis.Top3 != 2 || data.Kpis.Top10 != 2 {
		t.Fatalf("kpis %+v", data.Kpis)
	}
	if data.Strip != (app.RankStrip{Tracked: 3, Ranked: 2, Unranked: 1, Improved: 1, Declined: 1}) {
		t.Fatalf("strip %+v", data.Strip)
	}
	changes := map[string]domain.RankChange{}
	for _, r := range data.Rows {
		changes[r.Keyword] = r.Change
	}
	if c := changes["kopi susu gula aren"]; c.Kind != domain.ChangeUp || *c.Delta != 19 {
		t.Fatalf("%+v", c)
	}
	if changes["kedai kopi jakarta"].Kind != domain.ChangeLost || changes["cold brew"].Kind != domain.ChangeFlat {
		t.Fatalf("%+v", changes)
	}
	if len(data.Daily) != 29 || data.Distribution[0].Count != 2 || data.Distribution[5].Count != 1 || len(data.Sov) == 0 {
		t.Fatalf("daily %d dist %+v sov %d", len(data.Daily), data.Distribution, len(data.Sov))
	}
	if strings.Join(data.Tags, ",") != "brand,produk" {
		t.Fatalf("tags %v", data.Tags)
	}
	set, _ := f.keywords.TrackedSet(ctx, viewer, clientID, propertyID)
	if len(set) != 3 {
		t.Fatal(set)
	}

	sec, err := f.reports.RankSection(ctx, propertyID, data.Range.Current, data.Range.Previous)
	if err != nil || len(sec.Rows) != 3 || sec.Rows[0].Keyword != "kopi susu gula aren" || sec.Rows[2].Position != nil {
		t.Fatalf("%+v %v", sec, err)
	}

	tags, err := f.keywords.UpdateTags(ctx, manager, clientID, byKw["cold brew"].ID, "Promo,, promo")
	if err != nil || strings.Join(tags, ",") != "promo" {
		t.Fatal(tags, err)
	}
	if err := f.keywords.Delete(ctx, viewer, clientID, byKw["cold brew"].ID); err == nil {
		t.Fatal("viewer must not delete")
	}
	if err := f.keywords.Delete(ctx, manager, clientID, byKw["cold brew"].ID); err != nil {
		t.Fatal(err)
	}
	if err := f.keywords.Delete(ctx, manager, clientID, byKw["cold brew"].ID); !apperr.Is(err, apperr.NotFound) {
		t.Fatal(err)
	}
	tracked, _ := f.keywords.Track(ctx, manager, clientID, propertyID, []app.IdeaInput{{Keyword: "Cold Brew", Volume: p(90), Difficulty: ptrF(44.6)}}, domain.DeviceDesktop, []string{"riset"})
	if tracked != 1 {
		t.Fatalf("tracked %d", tracked)
	}
	keywords, _ = f.store.ListKeywords(ctx, propertyID)
	if last := keywords[len(keywords)-1]; last.Device != domain.DeviceDesktop || *last.Difficulty != 45 || *last.Volume != 90 {
		t.Fatalf("%+v", last)
	}
}

func ptrF(v float64) *float64 { return &v }

func TestIntegrationBacklinks(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	link := func(src, anchor string, lost bool) domain.BacklinkItem {
		return domain.BacklinkItem{SourceURL: "https://" + src + "/p", SourceDomain: src, TargetURL: "https://kopinusantara.id/", Anchor: anchor, Dofollow: true, DomainRank: 40, SpamScore: 5, FirstSeen: "2026-09-01", LastSeen: "2026-10-20", IsLost: lost}
	}
	f.provider.summary = &domain.BacklinkSummary{Backlinks: 100, ReferringDomains: 20, Dofollow: 75, Nofollow: 25, DomainRank: 30, NewLast30: 60, LostLast30: 15, ToxicShare: 0.1}
	f.provider.links = []domain.BacklinkItem{link("a.com", "kopi", false), link("b.com", "kopi", false), link("c.com", "web", true)}

	// First run has no baseline: new = round(60/30), lost = round(15/30).
	added, lost, err := f.backlinks.Refresh(ctx, viewer, clientID, propertyID)
	if err != nil || added != 2 || lost != 1 {
		t.Fatalf("first run +%d/-%d %v", added, lost, err)
	}
	_, _, err = f.backlinks.Refresh(ctx, viewer, clientID, propertyID)
	code(t, err, apperr.RateLimit, "")

	// Second run a day later: one new link, one live link turned lost, one
	// link missing from the provider list (left untouched).
	f.clk.T = f.clk.T.Add(24 * time.Hour)
	f.provider.links = []domain.BacklinkItem{link("a.com", "kopi", true), link("d.com", "baru", false)}
	added, lost, err = f.backlinks.Refresh(ctx, viewer, clientID, propertyID)
	if err != nil || added != 1 || lost != 1 {
		t.Fatalf("second run +%d/-%d %v", added, lost, err)
	}
	jobs, _ := f.store.ListJobs(ctx, clientID, []string{domain.JobKindBacklinks}, 5)
	if len(jobs) != 2 || *jobs[0].Message != "+1 / −1" {
		t.Fatalf("%+v", jobs)
	}

	data, err := f.backlinks.Data(ctx, viewer, clientID, propertyID, f.rng())
	if err != nil {
		t.Fatal(err)
	}
	if data.Latest == nil || data.Latest.NewBacklinks != 1 || data.Latest.LostBacklinks != 1 || data.Latest.Backlinks != 100 {
		t.Fatalf("%+v", data.Latest)
	}
	if data.DofollowShare != 75 || data.Deltas != nil || len(data.Rows) != 4 || len(data.Daily) != 29 {
		t.Fatalf("share %v deltas %v rows %d daily %d", data.DofollowShare, data.Deltas, len(data.Rows), len(data.Daily))
	}
	if data.Rows[0].IsLost || !data.Rows[3].IsLost {
		t.Fatal("live links first")
	}
	var weeklyNew, weeklyLost int32
	for _, w := range data.Weekly {
		weeklyNew += w.New
		weeklyLost += w.Lost
	}
	if weeklyNew != 3 || weeklyLost != 2 || len(data.TopAnchors) == 0 || data.TopAnchors[0].Label != "kopi" {
		t.Fatalf("weekly %d/%d anchors %+v", weeklyNew, weeklyLost, data.TopAnchors)
	}
	sec, err := f.reports.BacklinkSection(ctx, propertyID, data.Range.Current)
	if err != nil || sec.Latest == nil || sec.New != 3 || sec.Lost != 2 || sec.DofollowShare != 75 {
		t.Fatalf("%+v %v", sec, err)
	}
}

func TestIntegrationCompetitors(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if _, err := f.competitors.Add(ctx, manager, clientID, propertyID, "kopi"); err == nil {
		t.Fatal("invalid domain")
	} else {
		code(t, err, apperr.Validation, "Format domain tidak valid.")
	}
	if _, err := f.competitors.Add(ctx, manager, clientID, propertyID, "https://kopinusantara.id/"); err == nil {
		t.Fatal("own domain")
	} else {
		code(t, err, apperr.Validation, "Format domain tidak valid.")
	}
	if _, err := f.competitors.Add(ctx, viewer, clientID, propertyID, "kopikenangan.com"); err == nil {
		t.Fatal("viewer must not add")
	}
	first, err := f.competitors.Add(ctx, manager, clientID, propertyID, "https://www.KopiKenangan.com/menu")
	if err != nil || first.Domain != "kopikenangan.com" {
		t.Fatal(first, err)
	}
	_, err = f.competitors.Add(ctx, manager, clientID, propertyID, "kopikenangan.com")
	code(t, err, apperr.Validation, "Domain sudah ada.")
	for _, d := range []string{"janjijiwa.com", "fore.coffee", "tomoro.id", "kopisenja.id"} {
		if _, err := f.competitors.Add(ctx, manager, clientID, propertyID, d); err != nil {
			t.Fatal(d, err)
		}
	}
	_, err = f.competitors.Add(ctx, manager, clientID, propertyID, "ngopidulu.com")
	code(t, err, apperr.Validation, "Maksimal 5 kompetitor domain.")

	data, err := f.competitors.Data(ctx, viewer, clientID, propertyID, f.rng())
	if err != nil {
		t.Fatal(err)
	}
	if data.OwnDomain != "kopinusantara.id" || len(data.Competitors) != 5 || len(data.Benchmark) != 6 || data.Benchmark[0].Latest != nil || data.Benchmark[1].Latest == nil {
		t.Fatalf("%+v", data)
	}
	if len(data.Suggestions) != 1 || data.Suggestions[0].Domain != "ngopidulu.com" {
		t.Fatalf("suggestions %+v", data.Suggestions)
	}
	n, err := f.competitors.Refresh(ctx, viewer, clientID, propertyID)
	if err != nil || n != 6 {
		t.Fatal(n, err)
	}
	data, _ = f.competitors.Data(ctx, viewer, clientID, propertyID, f.rng())
	if data.Benchmark[0].Latest == nil || len(data.Series) != 4 || data.Series[0].Key != "d0" || data.Series[0].Label != "kopinusantara.id" || len(data.Traffic) != 29 {
		t.Fatalf("%+v", data)
	}
	if last := data.Traffic[len(data.Traffic)-1]; len(last.Values) != 4 || len(data.Traffic[0].Values) != 0 {
		t.Fatalf("%+v", last)
	}

	if err := f.competitors.Remove(ctx, manager, clientID, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.competitors.Remove(ctx, manager, clientID, first.ID); !apperr.Is(err, apperr.NotFound) {
		t.Fatal(err)
	}
	snaps, _ := f.store.ListLatestDomainSnapshots(ctx, propertyID, []string{"kopikenangan.com"})
	if len(snaps) != 0 {
		t.Fatal("snapshots of a removed domain must be gone")
	}
}

func TestIntegrationResearchCache(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if _, err := f.research.Get(ctx, viewer, clientID, "  ", domain.ModeIdeas); !apperr.Is(err, apperr.Validation) {
		t.Fatal(err)
	}
	if _, err := f.research.Get(ctx, viewer, "client-2", "kopi", domain.ModeIdeas); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("client without property: %v", err)
	}
	a, err := f.research.Get(ctx, viewer, clientID, " Kopi  Susu ", domain.ModeIdeas)
	if err != nil || a.FromCache || a.Provider != "mock" || a.Seed != "kopi susu" || len(a.Ideas) == 0 {
		t.Fatalf("%+v %v", a, err)
	}
	b, err := f.research.Get(ctx, viewer, clientID, "kopi susu", domain.ModeIdeas)
	if err != nil || !b.FromCache || len(b.Ideas) != len(a.Ideas) || !b.FetchedAt.Equal(a.FetchedAt) {
		t.Fatalf("%+v %v", b, err)
	}
	q, err := f.research.Get(ctx, viewer, clientID, "kopi susu", domain.ModeQuestions)
	if err != nil || q.FromCache {
		t.Fatal("other mode is a different key")
	}
	f.clk.T = f.clk.T.Add(8 * 24 * time.Hour)
	c, err := f.research.Get(ctx, viewer, clientID, "kopi susu", domain.ModeIdeas)
	if err != nil || c.FromCache {
		t.Fatal("stale cache must refetch")
	}
	row, _ := f.store.GetLatestResearch(ctx, clientID, "ideas:kopi susu", 2360, "id")
	if row == nil || !row.FetchedAt.Equal(f.clk.T) {
		t.Fatalf("%+v", row)
	}
	gap, err := f.research.KeywordGap(ctx, viewer, clientID, "kopikenangan.com")
	if err != nil || gap.OwnDomain != "kopinusantara.id" || len(gap.Shared) == 0 {
		t.Fatalf("%+v %v", gap, err)
	}
}

func TestIntegrationDailyRetentionPurge(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	today := dates.Today(f.clk.Now())
	f.provider.positions["kopi susu gula aren"] = p(1)
	f.provider.positions["kedai kopi jakarta"] = nil
	f.provider.summary = &domain.BacklinkSummary{Backlinks: 100, ReferringDomains: 20, Dofollow: 75, Nofollow: 25, DomainRank: 30, NewLast30: 60, LostLast30: 300}
	if _, err := f.keywords.Add(ctx, manager, clientID, propertyID, "kopi susu gula aren\nkedai kopi jakarta", domain.DeviceMobile, ""); err != nil {
		t.Fatal(err)
	}
	keywords, _ := f.store.ListKeywords(ctx, propertyID)
	if err := f.store.UpsertRankSnapshots(ctx, []domain.RankSnapshot{
		{ID: ids.New(), ClientID: clientID, KeywordID: keywords[0].ID, Date: dates.AddDays(today, -1), Position: p(20)},
		{ID: ids.New(), ClientID: clientID, KeywordID: keywords[1].ID, Date: dates.AddDays(today, -1), Position: p(5)},
	}); err != nil {
		t.Fatal(err)
	}

	res, err := f.daily.Run(ctx, clientID, slug, f.clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if res != (app.DailyResult{Properties: 2, Ranks: 2, Backlinks: 2, Domains: 2}) {
		t.Fatalf("%+v", res)
	}
	jobs, _ := f.store.ListJobs(ctx, clientID, []string{domain.JobKindDaily}, 5)
	if len(jobs) != 2 {
		t.Fatalf("%+v", jobs)
	}
	var withKeywords *domain.SyncJob
	for i := range jobs {
		if *jobs[i].Message == "2 peringkat · backlink +2/−10 · 1 domain" {
			withKeywords = &jobs[i]
		}
	}
	if withKeywords == nil || withKeywords.Status != domain.JobSuccess || withKeywords.Error != nil || withKeywords.FinishedAt == nil {
		t.Fatalf("%+v", jobs)
	}
	if len(f.notifier.sent) != 3 {
		t.Fatalf("notifications %+v", f.notifier.sent)
	}
	drop := f.notifier.sent[0]
	if drop.Type != "RANK_DROP" || drop.Title != "Peringkat turun" || drop.Body != "1 kata kunci turun tajam sejak kemarin: kedai kopi jakarta (5→–)" || drop.Href != "/clients/kopi/seo/rank" {
		t.Fatalf("%+v", drop)
	}
	lost := f.notifier.sent[1]
	if lost.Type != "BACKLINK_LOST" || lost.Title != "Backlink hilang" || lost.Body != "10 backlink hilang dalam sehari terakhir." || lost.Href != "/clients/kopi/seo/backlinks" {
		t.Fatalf("%+v", lost)
	}

	// Retention: rows older than 760 days (snapshots) and 90 days (jobs) go.
	old := dates.AddDays(today, -800)
	if err := f.store.UpsertRankSnapshots(ctx, []domain.RankSnapshot{{ID: ids.New(), ClientID: clientID, KeywordID: keywords[0].ID, Date: old, Position: p(9)}}); err != nil {
		t.Fatal(err)
	}
	if err := f.store.UpsertBacklinkSnapshot(ctx, domain.BacklinkSnapshot{ID: ids.New(), ClientID: clientID, PropertyID: propertyID, Date: old}); err != nil {
		t.Fatal(err)
	}
	if err := f.store.UpsertDomainSnapshot(ctx, domain.DomainSnapshot{ID: ids.New(), ClientID: clientID, PropertyID: propertyID, Domain: "kopinusantara.id", Date: old}); err != nil {
		t.Fatal(err)
	}
	if err := f.store.InsertJob(ctx, domain.SyncJob{ID: ids.New(), ClientID: clientID, Kind: domain.JobKindRanks, Status: domain.JobSuccess, StartedAt: f.clk.Now().Add(-100 * 24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := f.store.InsertResearch(ctx, domain.KeywordResearch{ID: ids.New(), ClientID: clientID, Seed: "ideas:x", LocationCode: 2360, LanguageCode: "id", Provider: "mock", Results: []domain.KeywordIdea{}, FetchedAt: f.clk.Now().Add(-40 * 24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	ret, err := f.daily.Retention(ctx, f.clk.Now())
	if err != nil || ret != (app.RetentionResult{RankSnapshots: 1, BacklinkSnapshots: 1, DomainSnapshots: 1, SyncJobs: 1}) {
		t.Fatalf("%+v %v", ret, err)
	}
	if _, err := f.daily.Run(ctx, clientID, slug, f.clk.Now()); err != nil {
		t.Fatal(err)
	}
	if row, _ := f.store.GetLatestResearch(ctx, clientID, "ideas:x", 2360, "id"); row != nil {
		t.Fatal("daily job must purge research older than 30 days")
	}

	if err := f.daily.PurgeClient(ctx, clientID); err != nil {
		t.Fatal(err)
	}
	if left, _ := f.store.ListKeywords(ctx, propertyID); len(left) != 0 {
		t.Fatal("keywords left after purge")
	}
	if job, _ := f.store.GetLatestJob(ctx, clientID, domain.JobKindDaily); job != nil {
		t.Fatal("jobs left after purge")
	}
	if snap, _ := f.store.GetLatestBacklinkSnapshot(ctx, propertyID); snap != nil {
		t.Fatal("snapshots left after purge")
	}
}
