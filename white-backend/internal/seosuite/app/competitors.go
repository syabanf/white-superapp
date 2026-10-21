package app

import (
	"context"
	"errors"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/providers/seodata"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Competitors is the competitor-domain half of actions-domains.ts plus
// getCompetitorData.
type Competitors struct {
	store   domain.Store
	guard   *Guard
	seo     seodata.Source
	tracker *Tracker
	jobs    *Jobs
	clk     clock.Clock
}

func NewCompetitors(store domain.Store, guard *Guard, seo seodata.Source, tracker *Tracker, jobs *Jobs, clk clock.Clock) *Competitors {
	return &Competitors{store: store, guard: guard, seo: seo, tracker: tracker, jobs: jobs, clk: clk}
}

// Add normalises the domain, refuses the own host, duplicates and a sixth
// domain, then fetches the first benchmark row (errors swallowed). ADMIN/MANAGER.
func (c *Competitors) Add(ctx context.Context, userID, clientID, propertyID, input string) (*domain.CompetitorDomain, error) {
	if input == "" || len(input) > 253 {
		return nil, apperr.New(apperr.Validation, "")
	}
	dom := domain.NormalizeDomain(input)
	if dom == "" {
		return nil, apperr.New(apperr.Validation, msgDomainInvalid)
	}
	p, err := c.guard.Property(ctx, userID, clientID, propertyID, true)
	if err != nil {
		return nil, err
	}
	if dom == p.Host() {
		return nil, apperr.New(apperr.Validation, msgDomainInvalid)
	}
	existing, err := c.store.FindCompetitorDomain(ctx, p.ID, dom)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperr.New(apperr.Validation, msgDomainExists)
	}
	count, err := c.store.CountCompetitorDomains(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if count >= domain.MaxCompetitorDomains {
		return nil, apperr.New(apperr.Validation, msgDomainMax)
	}
	row := domain.CompetitorDomain{ID: ids.New(), ClientID: p.ClientID, PropertyID: p.ID, Domain: dom, CreatedAt: c.clk.Now()}
	if err := c.store.InsertCompetitorDomain(ctx, row); err != nil {
		var u *domain.ErrUnique
		if errors.As(err, &u) {
			return nil, apperr.New(apperr.Validation, msgDomainExists)
		}
		return nil, err
	}
	_, _ = c.tracker.RefreshDomains(ctx, *p, dates.Today(c.clk.Now()), []string{dom})
	return &row, nil
}

// Remove deletes the domain and its snapshots. ADMIN/MANAGER.
func (c *Competitors) Remove(ctx context.Context, userID, clientID, id string) error {
	if id == "" {
		return apperr.New(apperr.Validation, "")
	}
	row, err := c.store.GetCompetitorDomain(ctx, id)
	if err != nil {
		return err
	}
	if row == nil {
		return apperr.New(apperr.NotFound, "")
	}
	if _, err := c.guard.Property(ctx, userID, clientID, row.PropertyID, true); err != nil {
		return err
	}
	return c.store.DeleteCompetitorDomain(ctx, *row)
}

// Refresh: VIEWER allowed, SEO_SUITE_DOMAINS cooldown, message "{n} domain".
func (c *Competitors) Refresh(ctx context.Context, userID, clientID, propertyID string) (int32, error) {
	p, err := c.guard.Property(ctx, userID, clientID, propertyID, false)
	if err != nil {
		return 0, err
	}
	job, err := c.jobs.Start(ctx, clientID, domain.JobKindDomains, p.SiteURL)
	if err != nil {
		return 0, err
	}
	n, runErr := c.tracker.RefreshDomains(ctx, *p, dates.Today(c.clk.Now()), nil)
	if err := c.jobs.Finish(ctx, job, runErr, countMsg(n, "domain")); err != nil {
		return 0, err
	}
	if runErr != nil {
		return 0, runErr
	}
	return n, nil
}

type BenchmarkRow struct {
	Domain string
	IsOwn  bool
	Latest *domain.DomainSnapshot
}

type TrafficSeries struct {
	Key   string
	Label string
}

type TrafficPoint struct {
	Date time.Time
	// Values is series key -> organic traffic; a missing key is a gap.
	Values map[string]int32
}

type CompetitorData struct {
	OwnDomain   string
	Competitors []domain.CompetitorDomain
	Benchmark   []BenchmarkRow
	Series      []TrafficSeries
	Traffic     []TrafficPoint
	Suggestions []domain.CompetitorSuggestion
	Range       dates.Resolved
	Mode        seodata.Mode
}

// Data: latest snapshot per domain, daily organic traffic for the own
// domain plus the top 3 competitors (keys d0..d3) and up to 6 provider
// suggestions not yet tracked. Provider failures leave suggestions empty.
func (c *Competitors) Data(ctx context.Context, userID, clientID, propertyID string, rng dates.Resolved) (*CompetitorData, error) {
	p, err := c.guard.Property(ctx, userID, clientID, propertyID, false)
	if err != nil {
		return nil, err
	}
	competitors, err := c.store.ListCompetitorDomains(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	own := p.Host()
	domains := []string{own}
	for _, cd := range competitors {
		domains = append(domains, cd.Domain)
	}
	to := dates.AddDays(rng.Current.To, 1)
	snaps, err := c.store.ListDomainSnapshots(ctx, p.ID, domains, rng.Current.From, to)
	if err != nil {
		return nil, err
	}
	latestRows, err := c.store.ListLatestDomainSnapshots(ctx, p.ID, domains)
	if err != nil {
		return nil, err
	}
	latestBy := map[string]domain.DomainSnapshot{}
	for _, r := range latestRows {
		latestBy[r.Domain] = r
	}
	out := &CompetitorData{OwnDomain: own, Competitors: competitors, Range: rng, Suggestions: []domain.CompetitorSuggestion{}}
	latestTraffic := map[string]int32{}
	for _, d := range domains {
		row := BenchmarkRow{Domain: d, IsOwn: d == own}
		if r, ok := latestBy[d]; ok {
			rr := r
			row.Latest = &rr
			latestTraffic[d] = r.OrganicTraffic
		} else {
			latestTraffic[d] = 0
		}
		out.Benchmark = append(out.Benchmark, row)
	}
	plotted := domain.PickSeriesDomains(own, latestTraffic, 3)
	keyOf := map[string]string{}
	for i, d := range plotted {
		key := "d" + string(rune('0'+i))
		keyOf[d] = key
		out.Series = append(out.Series, TrafficSeries{Key: key, Label: d})
	}
	byDate := map[string]map[string]int32{}
	for _, s := range snaps {
		key, ok := keyOf[s.Domain]
		if !ok {
			continue
		}
		day := dates.Format(s.Date)
		if byDate[day] == nil {
			byDate[day] = map[string]int32{}
		}
		byDate[day][key] = s.OrganicTraffic
	}
	dates.Range{From: rng.Current.From, To: to}.Each(func(day time.Time) {
		values := byDate[dates.Format(day)]
		if values == nil {
			values = map[string]int32{}
		}
		out.Traffic = append(out.Traffic, TrafficPoint{Date: day, Values: values})
	})
	provider, mode, err := c.seo.Provider(ctx)
	if err != nil {
		return nil, err
	}
	out.Mode = mode
	known := map[string]bool{}
	for _, d := range domains {
		known[d] = true
	}
	if suggestions, err := provider.Competitors(ctx, own, p.LocationCode, p.LanguageCode); err == nil {
		for _, s := range suggestions {
			if known[s.Domain] || len(out.Suggestions) >= 6 {
				continue
			}
			out.Suggestions = append(out.Suggestions, s)
		}
	}
	return out, nil
}
