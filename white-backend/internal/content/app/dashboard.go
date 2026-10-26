package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Dashboard computes the content overview analytics for one client.
type Dashboard struct{ Deps }

func NewDashboard(d Deps) *Dashboard { return &Dashboard{d} }

// DashboardQuery is the request as the handler parsed it. From/To are
// YYYY-MM-DD; empty means the last 28 complete days.
type DashboardQuery struct {
	From, To string
	Compare  string
	Filter   domain.DashboardFilter
}

type DashboardResult struct {
	From, To, PrevFrom, PrevTo time.Time
	Compare                    string
	Current, Previous          domain.Dashboard
	Pillars                    []string
}

// maxDashboardDays bounds one window (two years).
const maxDashboardDays = 731

// analyticsTimeout keeps the dashboard responsive when analytics is slow:
// the stored snapshot covers the gap.
const analyticsTimeout = 3 * time.Second

// Get checks client access (VIEWER may read) and computes both windows.
func (s *Dashboard) Get(ctx context.Context, userID, clientID string, q DashboardQuery) (*DashboardResult, error) {
	if _, err := s.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	to := domain.Day(s.now()).AddDate(0, 0, -1)
	from := to.AddDate(0, 0, -27)
	v := validate.New()
	if q.From != "" || q.To != "" {
		v.Required("from", q.From, msgDateInvalid).Required("to", q.To, msgDateInvalid)
		if d := parseDay(v, "from", q.From); d != nil {
			from = *d
		}
		if d := parseDay(v, "to", q.To); d != nil {
			to = *d
		}
	}
	if !v.HasErrors() && (to.Before(from) || to.Sub(from).Hours()/24 >= maxDashboardDays) {
		v.Add("to", msgRangeInvalid)
	}
	f := q.Filter
	if f.Platform != "" && !domain.IsPlatform(f.Platform) {
		v.Add("platform", msgPlatformInvalid)
	}
	if f.Format != "" && !domain.IsFormat(f.Format) {
		v.Add("format", msgFormatInvalid)
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	compare := domain.ComparePrevious
	if q.Compare == domain.CompareYear {
		compare = domain.CompareYear
	}
	prevFrom, prevTo := domain.CompareWindow(from, to, compare)

	res := &DashboardResult{From: from, To: to, PrevFrom: prevFrom, PrevTo: prevTo, Compare: compare}
	var err error
	if res.Current, err = s.window(ctx, clientID, f, from, to); err != nil {
		return nil, err
	}
	if res.Previous, err = s.window(ctx, clientID, f, prevFrom, prevTo); err != nil {
		return nil, err
	}
	if res.Pillars, err = s.Store.ClientPillars(ctx, clientID); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Dashboard) window(ctx context.Context, clientID string, f domain.DashboardFilter, from, to time.Time) (domain.Dashboard, error) {
	f.From, f.To = from, to
	items, err := s.Store.DashboardItems(ctx, clientID, f)
	if err != nil {
		return domain.Dashboard{}, err
	}
	tasks, err := s.Store.DashboardDesignTasks(ctx, clientID, f)
	if err != nil {
		return domain.Dashboard{}, err
	}
	loc := domain.AgencyLocation()
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	end := time.Date(to.Year(), to.Month(), to.Day()+1, 0, 0, 0, 0, loc)
	assets, err := s.Store.DashboardAssetVersions(ctx, clientID, start, end)
	if err != nil {
		return domain.Dashboard{}, err
	}
	published := make([]publishedRef, 0, len(items))
	for _, it := range items {
		if it.Status == domain.ItemPublished {
			published = append(published, publishedRef{ID: it.ID, ExternalIDs: it.ExternalIDs})
		}
	}
	metrics, err := s.Deps.metrics(ctx, clientID, published)
	if err != nil {
		return domain.Dashboard{}, err
	}
	return domain.ComputeDashboard(domain.DashboardInput{From: from, To: to, Items: items, Tasks: tasks, Assets: assets, Metrics: metrics}, s.now()), nil
}

type publishedRef struct {
	ID          string
	ExternalIDs []string
}

// metrics returns engagement per published item: analytics first (written
// back as the item's snapshot), the stored snapshot for the rest. An
// unreachable analytics service only means the snapshot is used.
func (d Deps) metrics(ctx context.Context, clientID string, items []publishedRef) (map[string]domain.PostMetrics, error) {
	ids := make([]string, 0, len(items))
	owner := map[string]string{}
	var externalIDs []string
	for _, it := range items {
		ids = append(ids, it.ID)
		for _, ext := range it.ExternalIDs {
			owner[ext] = it.ID
			externalIDs = append(externalIDs, ext)
		}
	}
	out, err := d.Store.ItemMetrics(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(externalIDs) == 0 || d.Analytics == nil {
		return out, nil
	}
	actx, cancel := context.WithTimeout(ctx, analyticsTimeout)
	defer cancel()
	posts, err := d.Analytics.Posts(actx, clientID, externalIDs, "", "")
	if err != nil {
		d.log().Info("engagement from analytics unavailable", "client", clientID, "err", err)
		return out, nil
	}
	live := map[string]domain.PostMetrics{}
	for _, p := range posts {
		if itemID, ok := owner[p.ExternalID]; ok {
			m := live[itemID]
			m.Add(p.Metrics)
			live[itemID] = m
		}
	}
	for itemID, m := range live {
		out[itemID] = m
		if err := d.Store.UpsertItemMetrics(ctx, itemID, m); err != nil {
			d.log().Warn("item metrics snapshot failed", "item", itemID, "err", err)
		}
	}
	return out, nil
}
