package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

// DashboardQuery is the analytics page request; empty days default to the
// last twelve months ending today (WIB).
type DashboardQuery struct {
	From, To string
	Compare  string
	Filter   domain.DashboardFilter
}

// Dashboard is the recruitment analytics of a period with its comparison.
func (c *Candidates) Dashboard(ctx context.Context, q DashboardQuery) (*domain.Dashboard, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	now := c.now()
	cur, err := dashboardPeriod(q.From, q.To, now)
	if err != nil {
		return nil, err
	}
	if q.Filter.Stage != "" && !domain.IsCandidateStatus(q.Filter.Stage) {
		return nil, invalid("Tahap tidak valid")
	}
	mode := domain.ComparePrevious
	if q.Compare == domain.CompareYear {
		mode = domain.CompareYear
	}
	prev := domain.ComparePeriod(cur, mode)

	out := &domain.Dashboard{Current: cur, Previous: prev, Compare: mode, Now: now}
	if out.Trend, err = c.Store.DashboardTrend(ctx, q.Filter, cur); err != nil {
		return nil, err
	}
	if out.PrevTrend, err = c.Store.DashboardTrend(ctx, q.Filter, prev); err != nil {
		return nil, err
	}
	if out.Detail, err = c.Store.DashboardDetail(ctx, q.Filter, cur, now); err != nil {
		return nil, err
	}
	out.Kpis, out.PrevKpis = domain.KpisFromTrend(out.Trend), domain.KpisFromTrend(out.PrevTrend)
	out.Funnel, out.Sources = domain.Funnel(out.Detail.Cohort), domain.Sources(out.Detail.Cohort)
	return out, nil
}

func dashboardPeriod(from, to string, now time.Time) (domain.Period, error) {
	today, _ := domain.ParseDay(domain.DayKey(now))
	start, end := today.AddDate(-1, 0, 1), today
	if from != "" || to != "" {
		var ok1, ok2 bool
		start, ok1 = domain.ParseDay(from)
		end, ok2 = domain.ParseDay(to)
		if !ok1 || !ok2 {
			return domain.Period{}, invalid("Rentang tanggal tidak valid")
		}
	}
	p := domain.NewPeriod(start, end)
	if end.Before(start) || p.Days() > domain.MaxDashboardDays {
		return domain.Period{}, invalid("Rentang tanggal tidak valid")
	}
	return p, nil
}
