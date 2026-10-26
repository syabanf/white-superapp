package app

import (
	"context"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/sync"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

const (
	msgAuditThrottled = "Audit terakhir dijalankan kurang dari 10 menit lalu. Coba lagi beberapa menit."
	msgAuditDone      = "Audit selesai"
	msgAuditPartial   = "Audit selesai sebagian"
	msgAuditAllFailed = "Audit gagal dijalankan"
)

type SearchPoint struct {
	Date                                      string
	Clicks, Impressions, Position             *float64
	ClicksPrev, ImpressionsPrev, PositionPrev *float64
}

type Ga4Point struct {
	Date             string
	Organik, Lainnya int
}

type Ga4Summary struct {
	HasData                          bool
	OrganicSessions, Sessions, Convs int
	OrganicDelta, ConversionsDelta   metrics.Delta
	OrganicSpark                     []float64
}

type SeoOverview struct {
	HasData          bool
	Kpis             domain.SearchAgg
	Clicks           metrics.Delta
	Impressions      metrics.Delta
	CTR              metrics.Delta
	Position         metrics.Delta
	ClicksSpark      []float64
	ImpressionsSpark []float64
	Daily            []SearchPoint
	Ga4              Ga4Summary
	Ga4Daily         []Ga4Point
	Health           *int
}

type QueryStats struct {
	Rows         []domain.QueryAgg
	Distribution []domain.BucketRow
	Striking     []domain.Opportunity
	LowCTR       []domain.Opportunity
	Declining    []domain.Opportunity
	Rising       []domain.Opportunity
}

type AuditData struct {
	LatestMobile, LatestDesktop *domain.SeoAudit
	LastRunAt                   *time.Time
	History                     []ScorePoint
	Crawl                       *domain.SeoCrawl
	Pages                       []domain.SeoCrawlPage
	Issues                      []domain.SeoIssue
	Counts                      domain.SeverityCounts
	CodeCounts                  []CodeCount
}

type ScorePoint struct {
	Date   string
	Scores domain.AuditScores
}

type CodeCount struct {
	Code  string
	Count int
}

type TrendPoint struct {
	Date   string
	Counts domain.SeverityCounts
	Pages  int
}

type RunAuditResult struct {
	Message                      string
	Audits, PagesCrawled, Issues int
}

type Seo struct{ Deps }

func NewSeo(d Deps) *Seo { return &Seo{d} }

// property checks access (any member) and that the property is the client's.
func (s *Seo) property(ctx context.Context, userID, clientID, propertyID string) (*domain.SeoProperty, error) {
	if _, err := s.view(ctx, userID, clientID); err != nil {
		return nil, err
	}
	p, err := s.Store.GetSeoProperty(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.ClientID != clientID {
		return nil, apperr.New(apperr.NotFound, msgNoProperty)
	}
	return p, nil
}

func (s *Seo) Overview(ctx context.Context, userID, clientID, propertyID string, w Window) (*SeoOverview, error) {
	p, err := s.property(ctx, userID, clientID, propertyID)
	if err != nil {
		return nil, err
	}
	ids := []string{p.ID}
	rows, err := s.Store.ListSeoDaily(ctx, ids, w.Current.From, w.Current.To)
	if err != nil {
		return nil, err
	}
	prevRows, err := s.Store.ListSeoDaily(ctx, ids, w.Previous.From, w.Previous.To)
	if err != nil {
		return nil, err
	}
	ga4, err := s.Store.ListGa4Daily(ctx, p.ID, w.Current.From, w.Current.To)
	if err != nil {
		return nil, err
	}
	ga4Prev, err := s.Store.ListGa4Daily(ctx, p.ID, w.Previous.From, w.Previous.To)
	if err != nil {
		return nil, err
	}
	health, err := healthFor(ctx, s.Store, ids)
	if err != nil {
		return nil, err
	}
	cur := domain.AggregateSearch(domain.DailyRows(rows))
	prev := domain.AggregateSearch(domain.DailyRows(prevRows))
	daily := domain.FillDaily(w.Current, rows, seoDay, seoDailyFields(), false)
	prevDaily := domain.FillDaily(w.Previous, prevRows, seoDay, seoDailyFields(), false)
	points := make([]SearchPoint, 0, len(daily))
	for i, d := range daily {
		pt := SearchPoint{Date: d.Date, Clicks: d.Get(0), Impressions: d.Get(1), Position: d.Get(2)}
		if i < len(prevDaily) {
			pt.ClicksPrev, pt.ImpressionsPrev, pt.PositionPrev = prevDaily[i].Get(0), prevDaily[i].Get(1), prevDaily[i].Get(2)
		}
		points = append(points, pt)
	}
	var organic, sessions, convs, prevOrganic, prevConvs int
	for _, r := range ga4 {
		organic += r.OrganicSessions
		sessions += r.Sessions
		convs += r.Conversions
	}
	for _, r := range ga4Prev {
		prevOrganic += r.OrganicSessions
		prevConvs += r.Conversions
	}
	ga4Fill := domain.FillDaily(w.Current, ga4, func(r domain.Ga4DailyMetric) time.Time { return r.Date }, []func(domain.Ga4DailyMetric) float64{
		func(r domain.Ga4DailyMetric) float64 { return float64(r.OrganicSessions) },
		func(r domain.Ga4DailyMetric) float64 { return float64(max(0, r.Sessions-r.OrganicSessions)) },
	}, true)
	ga4Points := make([]Ga4Point, 0, len(ga4Fill))
	for _, d := range ga4Fill {
		ga4Points = append(ga4Points, Ga4Point{Date: d.Date, Organik: int(*d.Get(0)), Lainnya: int(*d.Get(1))})
	}
	return &SeoOverview{
		HasData: len(rows) > 0 || len(prevRows) > 0, Kpis: cur,
		Clicks: metrics.ComputeDelta(float64(cur.Clicks), float64(prev.Clicks)), Impressions: metrics.ComputeDelta(float64(cur.Impressions), float64(prev.Impressions)),
		CTR: metrics.ComputeDelta(cur.CTR, prev.CTR), Position: metrics.ComputeDelta(cur.Position, prev.Position),
		ClicksSpark: domain.Sparkline(daily, 0, 14), ImpressionsSpark: domain.Sparkline(daily, 1, 14),
		Daily:    points,
		Ga4:      Ga4Summary{HasData: len(ga4) > 0, OrganicSessions: organic, Sessions: sessions, Convs: convs, OrganicDelta: metrics.ComputeDelta(float64(organic), float64(prevOrganic)), ConversionsDelta: metrics.ComputeDelta(float64(convs), float64(prevConvs)), OrganicSpark: domain.Sparkline(ga4Fill, 0, 14)},
		Ga4Daily: ga4Points, Health: health,
	}, nil
}

func (s *Seo) dimStats(ctx context.Context, propertyID string, dim domain.SeoDimension, w Window) ([]domain.QueryAgg, error) {
	cur, err := s.Store.ListSeoDimension(ctx, []string{propertyID}, dim, w.Current.From, w.Current.To)
	if err != nil {
		return nil, err
	}
	prev, err := s.Store.ListSeoDimension(ctx, []string{propertyID}, dim, w.Previous.From, w.Previous.To)
	if err != nil {
		return nil, err
	}
	return domain.MergeCurrentPrevious(domain.AggregateDimRows(cur), domain.AggregateDimRows(prev)), nil
}

func (s *Seo) Queries(ctx context.Context, userID, clientID, propertyID string, w Window) (*QueryStats, error) {
	p, err := s.property(ctx, userID, clientID, propertyID)
	if err != nil {
		return nil, err
	}
	rows, err := s.dimStats(ctx, p.ID, domain.DimQuery, w)
	if err != nil {
		return nil, err
	}
	rule := domain.DefaultOpportunityRule
	return &QueryStats{Rows: rows, Distribution: domain.PositionDistribution(rows), Striking: domain.StrikingDistance(rows, rule), LowCTR: domain.LowCTR(rows, rule), Declining: domain.Declining(rows, rule), Rising: domain.Rising(rows, rule)}, nil
}

func (s *Seo) Pages(ctx context.Context, userID, clientID, propertyID string, w Window) ([]domain.QueryAgg, error) {
	p, err := s.property(ctx, userID, clientID, propertyID)
	if err != nil {
		return nil, err
	}
	return s.dimStats(ctx, p.ID, domain.DimPage, w)
}

func (s *Seo) Breakdown(ctx context.Context, userID, clientID, propertyID string, w Window) (devices, countries []domain.DimAgg, err error) {
	p, err := s.property(ctx, userID, clientID, propertyID)
	if err != nil {
		return nil, nil, err
	}
	d, err := s.Store.ListSeoDimension(ctx, []string{p.ID}, domain.DimDevice, w.Current.From, w.Current.To)
	if err != nil {
		return nil, nil, err
	}
	c, err := s.Store.ListSeoDimension(ctx, []string{p.ID}, domain.DimCountry, w.Current.From, w.Current.To)
	if err != nil {
		return nil, nil, err
	}
	return domain.AggregateDimRows(d), domain.AggregateDimRows(c), nil
}

func (s *Seo) Audit(ctx context.Context, userID, clientID, propertyID string) (*AuditData, error) {
	p, err := s.property(ctx, userID, clientID, propertyID)
	if err != nil {
		return nil, err
	}
	ids := []string{p.ID}
	out := &AuditData{History: []ScorePoint{}, Pages: []domain.SeoCrawlPage{}, Issues: []domain.SeoIssue{}, CodeCounts: []CodeCount{}}
	if out.LatestMobile, err = s.Store.LatestAudit(ctx, ids, domain.Mobile); err != nil {
		return nil, err
	}
	if out.LatestDesktop, err = s.Store.LatestAudit(ctx, ids, domain.Desktop); err != nil {
		return nil, err
	}
	history, err := s.Store.ListMobileAuditHistory(ctx, p.ID, 24)
	if err != nil {
		return nil, err
	}
	if out.Crawl, err = s.Store.LatestCrawl(ctx, ids, nil); err != nil {
		return nil, err
	}
	if out.Crawl != nil {
		if out.Pages, err = s.Store.ListCrawlPages(ctx, out.Crawl.ID); err != nil {
			return nil, err
		}
		if out.Issues, err = s.Store.ListIssues(ctx, out.Crawl.ID); err != nil {
			return nil, err
		}
	}
	idx := map[string]int{}
	for _, h := range history {
		d := dates.Format(h.RunAt)
		pt := ScorePoint{Date: d, Scores: h.Scores()}
		if i, ok := idx[d]; ok {
			out.History[i] = pt
		} else {
			idx[d] = len(out.History)
			out.History = append(out.History, pt)
		}
	}
	codeIdx := map[string]int{}
	for _, i := range out.Issues {
		switch i.Severity {
		case domain.SeverityError:
			out.Counts.Errors++
		case domain.SeverityWarning:
			out.Counts.Warnings++
		case domain.SeverityNotice:
			out.Counts.Notices++
		}
		if j, ok := codeIdx[i.Code]; ok {
			out.CodeCounts[j].Count++
		} else {
			codeIdx[i.Code] = len(out.CodeCounts)
			out.CodeCounts = append(out.CodeCounts, CodeCount{Code: i.Code, Count: 1})
		}
	}
	sortBy(out.CodeCounts, func(a, b CodeCount) bool { return a.Count > b.Count })
	for _, a := range []*domain.SeoAudit{out.LatestMobile, out.LatestDesktop} {
		if a != nil && (out.LastRunAt == nil || a.RunAt.After(*out.LastRunAt)) {
			t := a.RunAt
			out.LastRunAt = &t
		}
	}
	return out, nil
}

func (s *Seo) IssueTrend(ctx context.Context, userID, clientID, propertyID string) ([]TrendPoint, error) {
	p, err := s.property(ctx, userID, clientID, propertyID)
	if err != nil {
		return nil, err
	}
	crawls, err := s.Store.ListRecentSuccessCrawls(ctx, p.ID, 30)
	if err != nil {
		return nil, err
	}
	idList := make([]string, 0, len(crawls))
	for _, c := range crawls {
		idList = append(idList, c.ID)
	}
	counts, err := s.Store.CountIssues(ctx, idList)
	if err != nil {
		return nil, err
	}
	out := []TrendPoint{}
	idx := map[string]int{}
	for i := len(crawls) - 1; i >= 0; i-- {
		c := crawls[i]
		d := dates.Format(c.StartedAt)
		pt := TrendPoint{Date: d, Counts: counts[c.ID], Pages: c.PagesCrawled}
		if j, ok := idx[d]; ok {
			out[j] = pt
		} else {
			idx[d] = len(out)
			out = append(out, pt)
		}
	}
	return out, nil
}

// RunAudit is allowed for any member, with a 10 minute cooldown.
func (s *Seo) RunAudit(ctx context.Context, userID, clientID, propertyID, override string) (*RunAuditResult, error) {
	override = strings.TrimSpace(override)
	if clientID == "" || propertyID == "" || (override != "" && !validate.IsHTTPURL(override)) {
		return nil, apperr.New(apperr.Validation, "")
	}
	p, err := s.seoGuard(ctx, userID, clientID, propertyID)
	if err != nil {
		return nil, err
	}
	last, err := s.Store.LatestAuditAny(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if last != nil && s.now().Sub(last.RunAt) < sync.AuditCooldown {
		return nil, apperr.New(apperr.RateLimit, msgAuditThrottled)
	}
	res, err := s.Syncer.RunAuditCore(ctx, *p, override)
	if err != nil {
		return nil, err
	}
	if res.AllFailed {
		first := apperr.Message(apperr.Unknown)
		if len(res.ErrorParts) > 0 {
			first = res.ErrorParts[0]
		}
		return nil, apperr.New(apperr.Unknown, msgAuditAllFailed+": "+first)
	}
	head := msgAuditDone
	if len(res.ErrorParts) > 0 {
		head = msgAuditPartial
	}
	msg := head + ": " + strings.Join(res.Parts, " · ")
	if len(res.ErrorParts) > 0 {
		msg += " — " + res.ErrorParts[0]
	}
	return &RunAuditResult{Message: msg, Audits: res.Audits, PagesCrawled: res.PagesCrawled, Issues: res.Issues}, nil
}

type SyncSeoResult struct {
	Mode       string
	Daily, Ga4 int
}

// Sync pulls 90 days of GSC/GA4 for every property. Any member.
func (s *Seo) Sync(ctx context.Context, userID, clientID string) (*SyncSeoResult, error) {
	if _, err := s.view(ctx, userID, clientID); err != nil {
		if apperr.Is(err, apperr.Unauthorized) {
			return nil, apperr.New(apperr.Unauthorized, msgNoAccessClient)
		}
		return nil, err
	}
	r, err := s.Syncer.SyncSeoForClient(ctx, clientID, 90)
	if err != nil {
		return nil, err
	}
	if len(r.Results) == 0 {
		return nil, apperr.New(apperr.NotFound, msgNoProperty)
	}
	if r.Results[0].Err != "" {
		return nil, apperr.New(apperr.Unknown, r.Results[0].Err)
	}
	out := &SyncSeoResult{Mode: r.Results[0].Mode}
	for _, x := range r.Results {
		out.Daily += x.Daily
		out.Ga4 += x.Ga4
	}
	return out, nil
}
