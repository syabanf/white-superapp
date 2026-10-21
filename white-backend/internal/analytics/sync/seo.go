package sync

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/googlesearch"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

type SeoResult struct {
	// real | demo
	Mode       string
	Daily      int
	Dimensions int
	Ga4        int
	Err        string
}

type SeoClientResult struct {
	Properties int
	Real       int
	Results    []SeoResult
}

var (
	isoDayRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	ga4DayRe = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)

	Ga4Metrics    = []string{"sessions", "engagedSessions", "totalUsers", "conversions", "userEngagementDuration"}
	Ga4Dimensions = []string{"date", "sessionDefaultChannelGroup"}
	gscDims       = []string{googlesearch.DimQuery, googlesearch.DimPage, googlesearch.DimCountry, googlesearch.DimDevice}
	dimensionFor  = map[string]domain.SeoDimension{googlesearch.DimQuery: domain.DimQuery, googlesearch.DimPage: domain.DimPage, googlesearch.DimCountry: domain.DimCountry, googlesearch.DimDevice: domain.DimDevice}
)

func round2(n float64) float64 { return math.Round(n*100) / 100 }

func jsRound(n float64) int { return int(math.Floor(n + 0.5)) }

// MapGscDaily maps rows queried with dimensions ["date"].
func MapGscDaily(propertyID string, rows []googlesearch.GscRow) []domain.SeoDailyMetric {
	out := []domain.SeoDailyMetric{}
	for _, r := range rows {
		if len(r.Keys) == 0 || !isoDayRe.MatchString(r.Keys[0]) {
			continue
		}
		d, err := dates.Parse(r.Keys[0])
		if err != nil {
			continue
		}
		out = append(out, domain.SeoDailyMetric{PropertyID: propertyID, Date: d, Clicks: jsRound(r.Clicks), Impressions: jsRound(r.Impressions), CTR: round2(r.CTR), Position: round2(r.Position)})
	}
	return out
}

// MapGscDimension maps rows queried with ["date", dim]; query and country
// keys are lowercased, device keys uppercased.
func MapGscDimension(propertyID string, rows []googlesearch.GscRow, dim string) []domain.SeoDimensionMetric {
	out := []domain.SeoDimensionMetric{}
	for _, r := range rows {
		if len(r.Keys) < 2 {
			continue
		}
		date := r.Keys[0]
		key := strings.TrimSpace(r.Keys[1])
		if !isoDayRe.MatchString(date) || key == "" {
			continue
		}
		switch dim {
		case googlesearch.DimCountry, googlesearch.DimQuery:
			key = strings.ToLower(key)
		case googlesearch.DimDevice:
			key = strings.ToUpper(key)
		}
		d, err := dates.Parse(date)
		if err != nil {
			continue
		}
		out = append(out, domain.SeoDimensionMetric{PropertyID: propertyID, Date: d, Dimension: dimensionFor[dim], Key: key, Clicks: jsRound(r.Clicks), Impressions: jsRound(r.Impressions), CTR: round2(r.CTR), Position: round2(r.Position)})
	}
	return out
}

// Ga4Date turns "20260901" into "2026-09-01"; ISO input passes through.
func Ga4Date(raw string) string {
	if isoDayRe.MatchString(raw) {
		return raw
	}
	if m := ga4DayRe.FindStringSubmatch(raw); m != nil {
		return m[1] + "-" + m[2] + "-" + m[3]
	}
	return ""
}

// MapGa4Daily aggregates [date, channel] rows per date; organic sessions
// come from the "Organic Search" channel, avg engagement = duration / sessions.
func MapGa4Daily(propertyID string, rows []googlesearch.Ga4Row) []domain.Ga4DailyMetric {
	type acc struct {
		domain.Ga4DailyMetric
		duration float64
		sessions float64
		organic  float64
		users    float64
		engaged  float64
		conv     float64
	}
	by := map[string]*acc{}
	for _, r := range rows {
		if len(r.Dimensions) == 0 {
			continue
		}
		date := Ga4Date(r.Dimensions[0])
		if date == "" {
			continue
		}
		channel := ""
		if len(r.Dimensions) > 1 {
			channel = strings.ToLower(r.Dimensions[1])
		}
		m := make([]float64, 5)
		copy(m, r.Metrics)
		a := by[date]
		if a == nil {
			d, err := dates.Parse(date)
			if err != nil {
				continue
			}
			a = &acc{Ga4DailyMetric: domain.Ga4DailyMetric{PropertyID: propertyID, Date: d}}
			by[date] = a
		}
		a.sessions += m[0]
		a.engaged += m[1]
		a.users += m[2]
		a.conv += m[3]
		a.duration += m[4]
		if channel == "organic search" {
			a.organic += m[0]
		}
	}
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]domain.Ga4DailyMetric, 0, len(keys))
	for _, k := range keys {
		a := by[k]
		row := a.Ga4DailyMetric
		row.Sessions, row.OrganicSessions, row.Users, row.EngagedSessions, row.Conversions = jsRound(a.sessions), jsRound(a.organic), jsRound(a.users), jsRound(a.engaged), jsRound(a.conv)
		if a.sessions > 0 {
			row.AvgEngagementSec = round2(a.duration / a.sessions)
		}
		out = append(out, row)
	}
	return out
}

// SyncSeoProperty pulls GSC daily and dimension rows plus GA4 for the last
// `days` complete days. Without a token it reports demo and writes nothing.
func (s *Syncer) SyncSeoProperty(ctx context.Context, p domain.SeoProperty, days int) (SeoResult, error) {
	token, err := s.token(ctx, p.ConnectionID)
	if err != nil {
		return SeoResult{}, err
	}
	if token == "" {
		return SeoResult{Mode: "demo"}, nil
	}
	if days <= 0 {
		days = 30
	}
	provider := s.google(ctx)
	end := dates.LatestCompleteDay(s.now())
	start := dates.AddDays(end, -(days - 1))
	startDate, endDate := dates.Format(start), dates.Format(end)
	res := SeoResult{Mode: "real"}

	rows, err := provider.QuerySearchAnalytics(ctx, token, googlesearch.GscQuery{SiteURL: p.SiteURL, StartDate: startDate, EndDate: endDate, Dimensions: []string{googlesearch.DimDate}})
	if err != nil {
		return res, err
	}
	daily := MapGscDaily(p.ID, rows)
	if err := s.Store.UpsertSeoDaily(ctx, daily); err != nil {
		return res, err
	}
	res.Daily = len(daily)

	for _, dim := range gscDims {
		rows, err := provider.QuerySearchAnalytics(ctx, token, googlesearch.GscQuery{SiteURL: p.SiteURL, StartDate: startDate, EndDate: endDate, Dimensions: []string{googlesearch.DimDate, dim}, RowLimit: 5000})
		if err != nil {
			return res, err
		}
		mapped := MapGscDimension(p.ID, rows, dim)
		for i := 0; i < len(mapped); i += 200 {
			end := i + 200
			if end > len(mapped) {
				end = len(mapped)
			}
			part := mapped[i:end]
			if err := s.Store.WithTx(ctx, func(tx domain.Store) error { return tx.UpsertSeoDimension(ctx, part) }); err != nil {
				return res, err
			}
			res.Dimensions += len(part)
		}
	}

	if p.Ga4PropertyID != nil && *p.Ga4PropertyID != "" {
		rows, err := provider.RunGa4Report(ctx, token, googlesearch.Ga4Request{PropertyID: googlesearch.Ga4PropertyName(*p.Ga4PropertyID), StartDate: startDate, EndDate: endDate, Metrics: Ga4Metrics, Dimensions: Ga4Dimensions, Limit: 10000})
		if err != nil {
			return res, err
		}
		ga4 := MapGa4Daily(p.ID, rows)
		if err := s.Store.UpsertGa4Daily(ctx, ga4); err != nil {
			return res, err
		}
		res.Ga4 = len(ga4)
	}
	return res, nil
}

// SyncSeoForClient syncs every property and writes SEO_GSC (and SEO_GA4
// when a GA4 property is set) jobs.
func (s *Syncer) SyncSeoForClient(ctx context.Context, clientID string, days int) (SeoClientResult, error) {
	props, err := s.Store.ListSeoProperties(ctx, clientID)
	if err != nil {
		return SeoClientResult{}, err
	}
	out := SeoClientResult{Properties: len(props)}
	for _, p := range props {
		r, err := s.SyncSeoProperty(ctx, p, days)
		if err != nil {
			r = SeoResult{Mode: "real", Err: errText(err)}
		}
		out.Results = append(out.Results, r)
		if r.Mode == "real" && r.Err == "" {
			out.Real++
		}
		status := domain.JobSuccess
		var errMsg *string
		if r.Err != "" {
			status = domain.JobFailed
			errMsg = strp(r.Err)
		}
		msg := "mock"
		if r.Mode != "demo" {
			msg = fmt.Sprintf("%d hari, %d baris dimensi · %s", r.Daily, r.Dimensions, p.SiteURL)
		}
		if err := s.job(ctx, clientID, domain.KindSeoGSC, status, strp(msg), errMsg); err != nil {
			return out, err
		}
		if p.Ga4PropertyID != nil && *p.Ga4PropertyID != "" {
			msg := "mock"
			if r.Mode != "demo" {
				msg = fmt.Sprintf("%d hari · %s", r.Ga4, *p.Ga4PropertyID)
			}
			if err := s.job(ctx, clientID, domain.KindSeoGA4, status, strp(msg), errMsg); err != nil {
				return out, err
			}
		}
	}
	return out, nil
}
