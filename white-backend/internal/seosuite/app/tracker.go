package app

import (
	"context"
	"math"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/providers/seodata"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Tracker is tracker.ts: the auth-free write paths shared by the refresh
// actions and the daily job. Provider errors propagate to the caller.
type Tracker struct {
	store domain.Store
	seo   seodata.Source
}

func NewTracker(store domain.Store, seo seodata.Source) *Tracker {
	return &Tracker{store: store, seo: seo}
}

type RankDrop struct {
	Keyword string
	Prev    *int32
	Cur     *int32
}

// TrackRanks fetches today's SERP positions (optionally for a subset of
// keywords) in batches of 20 and upserts rank_snapshots. A drop is reported
// only when yesterday had a snapshot and IsRankDropAlert fires.
func (t *Tracker) TrackRanks(ctx context.Context, p Property, date time.Time, keywordIDs []string) (int32, []RankDrop, error) {
	var keywords []domain.TrackedKeyword
	var err error
	if keywordIDs != nil {
		keywords, err = t.store.ListKeywordsByIDs(ctx, p.ID, keywordIDs)
	} else {
		keywords, err = t.store.ListKeywords(ctx, p.ID)
	}
	if err != nil {
		return 0, nil, err
	}
	if len(keywords) == 0 {
		return 0, []RankDrop{}, nil
	}
	provider, _, err := t.seo.Provider(ctx)
	if err != nil {
		return 0, nil, err
	}
	allIDs := make([]string, 0, len(keywords))
	for _, k := range keywords {
		allIDs = append(allIDs, k.ID)
	}
	prevBy, err := t.store.ListRankPositionsOn(ctx, allIDs, dates.AddDays(date, -1))
	if err != nil {
		return 0, nil, err
	}
	target := p.Host()
	var updated int32
	drops := []RankDrop{}
	for i := 0; i < len(keywords); i += domain.SerpBatch {
		batch := keywords[i:min(i+domain.SerpBatch, len(keywords))]
		reqs := make([]domain.SerpRankRequest, 0, len(batch))
		for _, k := range batch {
			reqs = append(reqs, domain.SerpRankRequest{Keyword: k.Keyword, Device: k.Device})
		}
		results, err := provider.SerpRanks(ctx, reqs, p.LocationCode, p.LanguageCode, target)
		if err != nil {
			return updated, drops, err
		}
		rows := make([]domain.RankSnapshot, 0, len(batch))
		for j, k := range batch {
			if j >= len(results) {
				break
			}
			r := results[j]
			updated++
			if prev, had := prevBy[k.ID]; had && domain.IsRankDropAlert(prev, r.Position) {
				drops = append(drops, RankDrop{Keyword: k.Keyword, Prev: prev, Cur: r.Position})
			}
			rows = append(rows, domain.RankSnapshot{ID: ids.New(), ClientID: p.ClientID, KeywordID: k.ID, Date: date, Position: r.Position, URL: r.URL, SerpFeatures: r.SerpFeatures, TopResults: r.Top10})
		}
		if err := t.store.UpsertRankSnapshots(ctx, rows); err != nil {
			return updated, drops, err
		}
	}
	return updated, drops, nil
}

// RefreshBacklinks upserts the provider's link list (chunks of 25) and
// today's summary snapshot. added = keys never seen before; lost = keys that
// were live and are now flagged lost. Links missing from the provider list
// are left as they are. A first run spreads the 30-day figures over one day.
func (t *Tracker) RefreshBacklinks(ctx context.Context, p Property, date time.Time) (added, lost int32, err error) {
	provider, _, err := t.seo.Provider(ctx)
	if err != nil {
		return 0, 0, err
	}
	host := p.Host()
	summary, err := provider.BacklinkSummary(ctx, host)
	if err != nil {
		return 0, 0, err
	}
	list, err := provider.Backlinks(ctx, host, domain.BacklinkListLimit, domain.BacklinksAll)
	if err != nil {
		return 0, 0, err
	}
	existing, err := t.store.ListBacklinkKeys(ctx, p.ID)
	if err != nil {
		return 0, 0, err
	}
	wasLost := make(map[string]bool, len(existing))
	for _, e := range existing {
		wasLost[e.SourceURL+"|"+e.TargetURL] = e.IsLost
	}
	parseDay := func(s string) time.Time {
		if d, err := dates.Parse(s); err == nil {
			return d
		}
		return date
	}
	for i := 0; i < len(list); i += 25 {
		chunk := list[i:min(i+25, len(list))]
		rows := make([]domain.Backlink, 0, len(chunk))
		for _, b := range chunk {
			key := b.SourceURL + "|" + b.TargetURL
			if was, seen := wasLost[key]; !seen {
				added++
			} else if !was && b.IsLost {
				lost++
			}
			rows = append(rows, domain.Backlink{
				ID: ids.New(), ClientID: p.ClientID, PropertyID: p.ID,
				SourceURL: b.SourceURL, SourceDomain: b.SourceDomain, TargetURL: b.TargetURL, Anchor: b.Anchor,
				Dofollow: b.Dofollow, DomainRank: ptr(b.DomainRank), SpamScore: ptr(b.SpamScore),
				FirstSeen: parseDay(b.FirstSeen), LastSeen: parseDay(b.LastSeen), IsLost: b.IsLost,
			})
		}
		if err := t.store.UpsertBacklinks(ctx, rows); err != nil {
			return 0, 0, err
		}
	}
	if len(existing) == 0 {
		added = int32(math.Round(float64(summary.NewLast30) / 30))
		lost = int32(math.Round(float64(summary.LostLast30) / 30))
	}
	snap := domain.BacklinkSnapshot{
		ID: ids.New(), ClientID: p.ClientID, PropertyID: p.ID, Date: date,
		Backlinks: summary.Backlinks, ReferringDomains: summary.ReferringDomains, Dofollow: summary.Dofollow, Nofollow: summary.Nofollow,
		NewBacklinks: added, LostBacklinks: lost, DomainRank: ptr(summary.DomainRank), ToxicShare: ptr(summary.ToxicShare),
	}
	if err := t.store.UpsertBacklinkSnapshot(ctx, snap); err != nil {
		return 0, 0, err
	}
	return added, lost, nil
}

// RefreshDomains writes a domain_snapshots row for the own host and every
// competitor (or only the given domains). Returns how many were written.
func (t *Tracker) RefreshDomains(ctx context.Context, p Property, date time.Time, only []string) (int32, error) {
	domains := only
	if domains == nil {
		competitors, err := t.store.ListCompetitorDomains(ctx, p.ID)
		if err != nil {
			return 0, err
		}
		domains = []string{p.Host()}
		for _, c := range competitors {
			domains = append(domains, c.Domain)
		}
	}
	provider, _, err := t.seo.Provider(ctx)
	if err != nil {
		return 0, err
	}
	var n int32
	for _, d := range domains {
		o, err := provider.DomainOverview(ctx, d, p.LocationCode, p.LanguageCode)
		if err != nil {
			return n, err
		}
		snap := domain.DomainSnapshot{
			ID: ids.New(), ClientID: p.ClientID, PropertyID: p.ID, Domain: d, Date: date,
			OrganicKeywords: o.OrganicKeywords, OrganicTraffic: o.OrganicTraffic, OrganicCost: ptr(o.OrganicCost),
			Top3: o.Top3, Top10: o.Top10, Top100: o.Top100,
			Backlinks: ptr(o.Backlinks), ReferringDomains: ptr(o.ReferringDomains), DomainRank: ptr(o.DomainRank),
		}
		if err := t.store.UpsertDomainSnapshot(ctx, snap); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
