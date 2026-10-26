package domain

import (
	"slices"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

// Insight payload (features/reports/insights.ts + lib/providers/openrouter
// types). JSON names match the old payload because the real provider sends
// it to the model verbatim.

type InsightRange struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Days     int    `json:"days"`
	PrevFrom string `json:"prevFrom"`
	PrevTo   string `json:"prevTo"`
}

type InsightTopPost struct {
	Platform    string `json:"platform"`
	Caption     string `json:"caption"`
	Type        string `json:"type"`
	Engagements int64  `json:"engagements"`
	Likes       int64  `json:"likes"`
	Comments    int64  `json:"comments"`
}

type InsightSocial struct {
	Followers              int64            `json:"followers"`
	FollowerGrowth         float64          `json:"followerGrowth"`
	FollowerGrowthPct      *float64         `json:"followerGrowthPct"`
	EngagementRate         float64          `json:"engagementRate"`
	EngagementRateDeltaPct *float64         `json:"engagementRateDeltaPct"`
	Posts                  int64            `json:"posts"`
	Reach                  int64            `json:"reach"`
	ReachDeltaPct          *float64         `json:"reachDeltaPct"`
	TopPosts               []InsightTopPost `json:"topPosts"`
}

type InsightTopQuery struct {
	Query       string  `json:"query"`
	Clicks      int64   `json:"clicks"`
	Impressions int64   `json:"impressions"`
	Ctr         float64 `json:"ctr"`
	Position    float64 `json:"position"`
}

type InsightSeo struct {
	Clicks              int64             `json:"clicks"`
	ClicksDeltaPct      *float64          `json:"clicksDeltaPct"`
	Impressions         int64             `json:"impressions"`
	ImpressionsDeltaPct *float64          `json:"impressionsDeltaPct"`
	Ctr                 float64           `json:"ctr"`
	Position            float64           `json:"position"`
	PositionDeltaAbs    float64           `json:"positionDeltaAbs"`
	Health              *int32            `json:"health"`
	TopQueries          []InsightTopQuery `json:"topQueries"`
}

type InsightCampaign struct {
	Name       string  `json:"name"`
	Objective  string  `json:"objective"`
	Spend      float64 `json:"spend"`
	Results    int64   `json:"results"`
	ResultType *string `json:"resultType"`
	Cpr        float64 `json:"cpr"`
}

type InsightAds struct {
	Currency        string            `json:"currency"`
	Spend           float64           `json:"spend"`
	SpendDeltaPct   *float64          `json:"spendDeltaPct"`
	Results         int64             `json:"results"`
	ResultsDeltaPct *float64          `json:"resultsDeltaPct"`
	Cpr             float64           `json:"cpr"`
	CprDeltaPct     *float64          `json:"cprDeltaPct"`
	Ctr             float64           `json:"ctr"`
	Cpm             float64           `json:"cpm"`
	Roas            *float64          `json:"roas"`
	Campaigns       []InsightCampaign `json:"campaigns"`
}

// InsightPayload leaves out the modules that were not selected or have no data.
type InsightPayload struct {
	Social *InsightSocial `json:"social,omitempty"`
	Seo    *InsightSeo    `json:"seo,omitempty"`
	Ads    *InsightAds    `json:"ads,omitempty"`
}

type InsightInput struct {
	Module     string
	Language   string
	ClientName string
	Range      InsightRange
	Data       InsightPayload
}

func ToRangeInfo(current, previous dates.Range) InsightRange {
	return InsightRange{
		From: dates.Format(current.From), To: dates.Format(current.To), Days: current.Days(),
		PrevFrom: dates.Format(previous.From), PrevTo: dates.Format(previous.To),
	}
}

// InsightModules lists the report modules an insight reads: its own module,
// or for OVERVIEW the selection (every module when empty).
func InsightModules(module string, selection []string) []string {
	if module != ModuleOverview {
		return []string{module}
	}
	if len(selection) == 0 {
		return slices.Clone(ReportModules)
	}
	var out []string
	for _, m := range ReportModules {
		if slices.Contains(selection, m) {
			out = append(out, m)
		}
	}
	return out
}

// InsightTopN is how many top queries the payload carries (posts stay at 5).
func InsightTopN(module string) int {
	if module == ModuleSeo {
		return 10
	}
	return 5
}

// BuildInsightPayload is buildInsightPayload over already fetched data.
// Each block is included only when its overview has data.
func BuildInsightPayload(module string, data ReportData, currency string) InsightPayload {
	var p InsightPayload
	if s := data.Social; s != nil && s.Overview.HasData {
		ov := s.Overview
		posts := []InsightTopPost{}
		for _, tp := range firstN(s.TopPosts, 5) {
			posts = append(posts, InsightTopPost{Platform: tp.Platform, Caption: Truncate(tp.Caption, 80), Type: tp.ContentType(), Engagements: tp.Engagements, Likes: tp.Likes, Comments: tp.Comments})
		}
		p.Social = &InsightSocial{
			Followers: ov.Followers, FollowerGrowth: ov.FollowersDelta.Abs, FollowerGrowthPct: ov.FollowersDelta.Pct,
			EngagementRate: ov.EngagementRate, EngagementRateDeltaPct: ov.EngagementRateDelta.Pct,
			Posts: ov.Posts, Reach: ov.Reach, ReachDeltaPct: ov.ReachDelta.Pct, TopPosts: posts,
		}
	}
	if s := data.Seo; s != nil && s.HasData {
		queries := []InsightTopQuery{}
		for _, q := range firstN(s.TopQueries, InsightTopN(module)) {
			queries = append(queries, InsightTopQuery(q))
		}
		p.Seo = &InsightSeo{
			Clicks: s.Clicks, ClicksDeltaPct: s.ClicksDelta.Pct, Impressions: s.Impressions, ImpressionsDeltaPct: s.ImpressionsDelta.Pct,
			Ctr: s.Ctr, Position: s.Position, PositionDeltaAbs: s.PositionDelta.Abs, Health: s.Health, TopQueries: queries,
		}
	}
	if a := data.Ads; a != nil && a.HasData {
		if currency == "" {
			currency = "IDR"
		}
		camps := []InsightCampaign{}
		for _, c := range firstN(a.Campaigns, 6) {
			camps = append(camps, InsightCampaign{Name: c.Name, Objective: c.Objective, Spend: c.Spend, Results: c.Results, ResultType: c.ResultType, Cpr: c.Cpr})
		}
		p.Ads = &InsightAds{
			Currency: currency, Spend: a.Kpis.Spend, SpendDeltaPct: a.SpendDelta.Pct, Results: a.Kpis.Results, ResultsDeltaPct: a.ResultsDelta.Pct,
			Cpr: a.Kpis.Cpr, CprDeltaPct: a.CprDelta.Pct, Ctr: a.Kpis.Ctr, Cpm: a.Kpis.Cpm, Roas: a.Kpis.Roas, Campaigns: camps,
		}
	}
	return p
}

func firstN[T any](xs []T, n int) []T {
	if len(xs) > n {
		return xs[:n]
	}
	return xs
}
