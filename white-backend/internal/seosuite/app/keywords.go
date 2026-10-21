package app

import (
	"context"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/providers/seodata"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Keywords is actions.ts: tracked keywords and the rank tracker.
type Keywords struct {
	store   domain.Store
	guard   *Guard
	seo     seodata.Source
	tracker *Tracker
	jobs    *Jobs
	views   *Views
	clk     clock.Clock
}

func NewKeywords(store domain.Store, guard *Guard, seo seodata.Source, tracker *Tracker, jobs *Jobs, views *Views, clk clock.Clock) *Keywords {
	return &Keywords{store: store, guard: guard, seo: seo, tracker: tracker, jobs: jobs, views: views, clk: clk}
}

// IdeaInput is one research idea the UI asks to track.
type IdeaInput struct {
	Keyword    string
	Volume     *int32
	Difficulty *float64
	CPC        *float64
	Intent     *domain.Intent
}

func validIdea(i IdeaInput) bool {
	n := utf8.RuneCountInString(i.Keyword)
	if n < 1 || n > 120 {
		return false
	}
	if i.Volume != nil && *i.Volume < 0 {
		return false
	}
	if i.Difficulty != nil && (*i.Difficulty < 0 || *i.Difficulty > 100) {
		return false
	}
	if i.CPC != nil && *i.CPC < 0 {
		return false
	}
	if i.Intent != nil {
		switch *i.Intent {
		case domain.IntentInformational, domain.IntentNavigational, domain.IntentCommercial, domain.IntentTransactional:
		default:
			return false
		}
	}
	return true
}

func normalizeDevice(d domain.Device) (domain.Device, bool) {
	switch d {
	case "":
		return domain.DeviceMobile, true
	case domain.DeviceDesktop, domain.DeviceMobile:
		return d, true
	}
	return "", false
}

// Track is "Lacak" from the research table: ideas with known metrics. ADMIN/MANAGER.
func (k *Keywords) Track(ctx context.Context, userID, clientID, propertyID string, ideas []IdeaInput, device domain.Device, tags []string) (int32, error) {
	dev, ok := normalizeDevice(device)
	if !ok || len(ideas) == 0 || len(ideas) > domain.MaxKeywordsPerAdd || len(tags) > domain.MaxTags {
		return 0, apperr.New(apperr.Validation, "")
	}
	for _, i := range ideas {
		if !validIdea(i) {
			return 0, apperr.New(apperr.Validation, "")
		}
	}
	for _, t := range tags {
		if n := utf8.RuneCountInString(t); n < 1 || n > 30 {
			return 0, apperr.New(apperr.Validation, "")
		}
	}
	p, err := k.guard.Property(ctx, userID, clientID, propertyID, true)
	if err != nil {
		return 0, err
	}
	return k.insertTracked(ctx, *p, ideas, dev, tags)
}

// Add is the bulk textarea: parse, enrich with provider metrics (failures
// ignored), insert. ADMIN/MANAGER.
func (k *Keywords) Add(ctx context.Context, userID, clientID, propertyID, text string, device domain.Device, tagsText string) (int32, error) {
	dev, ok := normalizeDevice(device)
	if !ok || strings.TrimSpace(text) == "" || utf8.RuneCountInString(text) > 20000 || utf8.RuneCountInString(tagsText) > 500 {
		return 0, apperr.New(apperr.Validation, "")
	}
	keywords := domain.ParseKeywordLines(text, domain.MaxKeywordsPerAdd)
	if len(keywords) == 0 {
		return 0, apperr.New(apperr.Validation, msgKeywordsInvalid)
	}
	p, err := k.guard.Property(ctx, userID, clientID, propertyID, true)
	if err != nil {
		return 0, err
	}
	ideas := make([]IdeaInput, 0, len(keywords))
	for _, kw := range keywords {
		ideas = append(ideas, IdeaInput{Keyword: kw})
	}
	if provider, _, err := k.seo.Provider(ctx); err == nil {
		if metricsList, err := provider.KeywordMetrics(ctx, keywords, p.LocationCode, p.LanguageCode); err == nil {
			by := map[string]domain.KeywordIdea{}
			for _, m := range metricsList {
				by[strings.ToLower(m.Keyword)] = m
			}
			for i, kw := range keywords {
				if m, ok := by[kw]; ok {
					intent := m.Intent
					ideas[i] = IdeaInput{Keyword: kw, Volume: ptr(m.Volume), Difficulty: ptr(float64(m.Difficulty)), CPC: ptr(m.CPC), Intent: &intent}
				}
			}
		}
	}
	return k.insertTracked(ctx, *p, ideas, dev, domain.ParseTags(tagsText, domain.MaxTags))
}

// insertTracked skips duplicates, then fetches first positions for the new
// rows right away (errors swallowed: the daily job fills them in).
func (k *Keywords) insertTracked(ctx context.Context, p Property, ideas []IdeaInput, device domain.Device, tags []string) (int32, error) {
	before, err := k.store.ListKeywordIDs(ctx, p.ID)
	if err != nil {
		return 0, err
	}
	rows := make([]domain.TrackedKeyword, 0, len(ideas))
	for _, i := range ideas {
		row := domain.TrackedKeyword{
			ID: ids.New(), ClientID: p.ClientID, PropertyID: p.ID, Keyword: strings.ToLower(strings.TrimSpace(i.Keyword)),
			LocationCode: p.LocationCode, LanguageCode: p.LanguageCode, Device: device, Tags: tags,
			Volume: i.Volume, CPC: i.CPC, Intent: i.Intent,
		}
		if i.Difficulty != nil {
			row.Difficulty = ptr(int32(math.Round(*i.Difficulty)))
		}
		rows = append(rows, row)
	}
	if err := k.store.InsertKeywords(ctx, rows); err != nil {
		return 0, err
	}
	after, err := k.store.ListKeywordIDs(ctx, p.ID)
	if err != nil {
		return 0, err
	}
	seen := make(map[string]bool, len(before))
	for _, id := range before {
		seen[id] = true
	}
	newIDs := []string{}
	for _, id := range after {
		if !seen[id] {
			newIDs = append(newIDs, id)
		}
	}
	if len(newIDs) > 0 {
		_, _, _ = k.tracker.TrackRanks(ctx, p, dates.Today(k.clk.Now()), newIDs)
	}
	return int32(len(newIDs)), nil
}

func (k *Keywords) ownedKeyword(ctx context.Context, userID, clientID, keywordID string) (*domain.TrackedKeyword, error) {
	if keywordID == "" {
		return nil, apperr.New(apperr.Validation, "")
	}
	kw, err := k.store.GetKeyword(ctx, keywordID)
	if err != nil {
		return nil, err
	}
	if kw == nil {
		return nil, apperr.New(apperr.NotFound, "")
	}
	if _, err := k.guard.Property(ctx, userID, clientID, kw.PropertyID, true); err != nil {
		return nil, err
	}
	return kw, nil
}

// Delete removes a keyword and its snapshots. ADMIN/MANAGER.
func (k *Keywords) Delete(ctx context.Context, userID, clientID, keywordID string) error {
	kw, err := k.ownedKeyword(ctx, userID, clientID, keywordID)
	if err != nil {
		return err
	}
	_, err = k.store.DeleteKeyword(ctx, kw.ID)
	return err
}

func (k *Keywords) UpdateTags(ctx context.Context, userID, clientID, keywordID, tagsText string) ([]string, error) {
	if utf8.RuneCountInString(tagsText) > 500 {
		return nil, apperr.New(apperr.Validation, "")
	}
	kw, err := k.ownedKeyword(ctx, userID, clientID, keywordID)
	if err != nil {
		return nil, err
	}
	tags := domain.ParseTags(tagsText, domain.MaxTags)
	if err := k.store.UpdateKeywordTags(ctx, kw.ID, tags); err != nil {
		return nil, err
	}
	return tags, nil
}

// RefreshRanks re-checks every tracked keyword. VIEWER allowed; one
// SEO_SUITE_RANKS job per 10 minutes per client.
func (k *Keywords) RefreshRanks(ctx context.Context, userID, clientID, propertyID string) (int32, error) {
	p, err := k.guard.Property(ctx, userID, clientID, propertyID, false)
	if err != nil {
		return 0, err
	}
	job, err := k.jobs.Start(ctx, clientID, domain.JobKindRanks, p.SiteURL)
	if err != nil {
		return 0, err
	}
	updated, _, runErr := k.tracker.TrackRanks(ctx, *p, dates.Today(k.clk.Now()), nil)
	if err := k.jobs.Finish(ctx, job, runErr, countMsg(updated, "kata kunci")); err != nil {
		return 0, err
	}
	if runErr != nil {
		return 0, runErr
	}
	return updated, nil
}

// RankDataResult adds the resolved range and provider mode to the view.
type RankDataResult struct {
	*RankData
	Range dates.Resolved
	Mode  seodata.Mode
}

func (k *Keywords) RankData(ctx context.Context, userID, clientID, propertyID string, rng dates.Resolved) (*RankDataResult, error) {
	p, err := k.guard.Property(ctx, userID, clientID, propertyID, false)
	if err != nil {
		return nil, err
	}
	data, err := k.views.RankData(ctx, p.ID, rng.Current, rng.Previous)
	if err != nil {
		return nil, err
	}
	_, mode, err := k.seo.Provider(ctx)
	if err != nil {
		return nil, err
	}
	return &RankDataResult{RankData: data, Range: rng, Mode: mode}, nil
}

// TrackedSet lists the property's keywords, lowercased.
func (k *Keywords) TrackedSet(ctx context.Context, userID, clientID, propertyID string) ([]string, error) {
	p, err := k.guard.Property(ctx, userID, clientID, propertyID, false)
	if err != nil {
		return nil, err
	}
	rows, err := k.store.ListKeywords(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, strings.ToLower(r.Keyword))
	}
	return out, nil
}

func (k *Keywords) ListJobs(ctx context.Context, userID, clientID string, kinds []string, limit int32) ([]domain.SyncJob, error) {
	if _, err := k.guard.Client(ctx, userID, clientID, false); err != nil {
		return nil, err
	}
	return k.jobs.List(ctx, clientID, kinds, limit)
}
