package app

import (
	"context"
	"regexp"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Insights covers targets, the monthly forecast and the funnel report
// (salesfunnel insights.go).
type Insights struct{ *Deps }

func NewInsights(d *Deps) *Insights { return &Insights{d} }

var monthRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// Month validates "YYYY-MM"; empty is the current WIB month.
func (i *Insights) Month(month string) (string, error) {
	if month == "" {
		return i.now().In(domain.Jakarta).Format("2006-01"), nil
	}
	if !monthRe.MatchString(month) {
		return "", apperr.Field("month", "Bulan harus berformat YYYY-MM")
	}
	return month, nil
}

func (i *Insights) ListTargets(ctx context.Context, month string) (string, []domain.Target, error) {
	m, err := i.Month(month)
	if err != nil {
		return "", nil, err
	}
	t, err := i.Store.ListTargets(ctx, m)
	return m, t, err
}

// SaveTargets replaces the month's targets (user nil = company target).
func (i *Insights) SaveTargets(ctx context.Context, actor Actor, month string, targets []domain.Target) (string, []domain.Target, error) {
	m, err := i.Month(month)
	if err != nil {
		return "", nil, err
	}
	seen := map[string]bool{}
	for _, t := range targets {
		key := deref(t.UserID)
		if seen[key] {
			return "", nil, invalid("Target duplikat untuk pengguna yang sama")
		}
		seen[key] = true
		if t.TargetValue < 0 || (t.TargetDeals != nil && *t.TargetDeals < 0) {
			return "", nil, invalid("Target tidak boleh negatif")
		}
	}
	if err := i.Store.WithTx(ctx, func(st domain.Store) error { return st.ReplaceTargets(ctx, m, targets, actor.UserID) }); err != nil {
		return "", nil, err
	}
	t, err := i.Store.ListTargets(ctx, m)
	return m, t, err
}

// Forecast is the monthly forecast per salesperson plus the total row.
type Forecast struct {
	Month string
	Rows  []*domain.ForecastRow
	Total *domain.ForecastRow
}

func (i *Insights) Forecast(ctx context.Context, month string) (*Forecast, error) {
	m, err := i.Month(month)
	if err != nil {
		return nil, err
	}
	fromS, toS := domain.MonthRange(m)
	from, _ := time.Parse("2006-01-02", fromS)
	to, _ := time.Parse("2006-01-02", toS)
	deals, err := i.Store.ForecastDeals(ctx, from, to)
	if err != nil {
		return nil, err
	}
	targets, err := i.Store.ListTargets(ctx, m)
	if err != nil {
		return nil, err
	}
	var users []domain.ForecastUser
	if i.Directory != nil {
		list, err := i.Directory.ListUsers(ctx)
		if err != nil {
			i.warn("list users failed", "err", err)
		}
		owners := map[string]bool{}
		for _, d := range deals {
			if d.OwnerUserID != nil {
				owners[*d.OwnerUserID] = true
			}
		}
		for _, t := range targets {
			if t.UserID != nil {
				owners[*t.UserID] = true
			}
		}
		for _, u := range list {
			if owners[u.ID] {
				users = append(users, domain.ForecastUser{ID: u.ID, Name: u.Name})
			}
		}
	}
	company, _ := domain.SplitTargets(targets)
	rows := domain.AggregateForecast(deals, targets, users)
	return &Forecast{Month: m, Rows: rows, Total: domain.SumForecast(rows, company)}, nil
}

// Funnel returns the report for [from, to] (inclusive days, WIB).
func (i *Insights) Funnel(ctx context.Context, fromS, toS string) (*domain.FunnelReport, time.Time, time.Time, error) {
	now := i.now().In(domain.Jakarta)
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, domain.Jakarta)
	to := from.AddDate(0, 1, -1)
	if fromS != "" {
		t, err := time.ParseInLocation("2006-01-02", fromS, domain.Jakarta)
		if err != nil {
			return nil, from, to, apperr.Field("from", "Tanggal tidak valid")
		}
		from = t
	}
	if toS != "" {
		t, err := time.ParseInLocation("2006-01-02", toS, domain.Jakarta)
		if err != nil {
			return nil, from, to, apperr.Field("to", "Tanggal tidak valid")
		}
		to = t
	}
	if to.Before(from) {
		return nil, from, to, apperr.Field("to", "Tanggal akhir harus setelah tanggal awal")
	}
	leads, _, err := i.Store.ListLeads(ctx, domain.LeadFilter{Limit: 1000})
	if err != nil {
		return nil, from, to, err
	}
	deals, _, err := i.Store.ListDeals(ctx, domain.DealFilter{Limit: 1000})
	if err != nil {
		return nil, from, to, err
	}
	pls, err := i.Store.ListPipelines(ctx)
	if err != nil {
		return nil, from, to, err
	}
	var stages []domain.Stage
	for _, p := range pls {
		stages = append(stages, p.Stages...)
	}
	reasons, err := i.Store.ListLostReasons(ctx)
	if err != nil {
		return nil, from, to, err
	}
	r := domain.BuildFunnelReport(leads, deals, stages, reasons, from.UTC(), to.AddDate(0, 0, 1).UTC())
	return &r, from, to, nil
}
