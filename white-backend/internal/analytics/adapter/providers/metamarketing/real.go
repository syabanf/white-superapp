package metamarketing

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/providers/meta"
)

const providerName = "meta-marketing"

type Real struct {
	Client *meta.Client
	Token  string
}

func NewReal(token string) *Real {
	return &Real{Client: meta.NewClient(providerName, 60*time.Second), Token: token}
}

func num(s *string) float64 {
	if s == nil {
		return 0
	}
	f, err := strconv.ParseFloat(*s, 64)
	if err != nil {
		return 0
	}
	return f
}

func minor(s *string) *float64 {
	if s == nil {
		return nil
	}
	// daily_budget is in minor units; IDR has no decimals so /100 is safe for most currencies.
	v := num(s) / 100
	return &v
}

func (r *Real) list(ctx context.Context, path string, fields string, maxPages int, out any) error {
	params := url.Values{"fields": {fields}, "limit": {"200"}}
	rows, err := meta.GetAll[json.RawMessage](ctx, r.Client, r.Token, path, params, maxPages)
	if err != nil {
		return err
	}
	buf, _ := json.Marshal(rows)
	return json.Unmarshal(buf, out)
}

func (r *Real) ListCampaigns(ctx context.Context, actID string) ([]Campaign, error) {
	var raw []struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Objective   *string `json:"objective"`
		Status      *string `json:"status"`
		DailyBudget *string `json:"daily_budget"`
		StartTime   *string `json:"start_time"`
	}
	if err := r.list(ctx, "/"+actID+"/campaigns", "id,name,objective,status,daily_budget,start_time", 25, &raw); err != nil {
		return nil, err
	}
	out := make([]Campaign, 0, len(raw))
	for _, c := range raw {
		out = append(out, Campaign{ID: c.ID, Name: c.Name, Objective: orDefault(c.Objective, "OUTCOME_TRAFFIC"), Status: orDefault(c.Status, "ACTIVE"), DailyBudget: minor(c.DailyBudget), StartTime: c.StartTime})
	}
	return out, nil
}

func (r *Real) ListAdSets(ctx context.Context, actID string) ([]AdSet, error) {
	var raw []struct {
		ID          string  `json:"id"`
		CampaignID  string  `json:"campaign_id"`
		Name        string  `json:"name"`
		Status      *string `json:"status"`
		DailyBudget *string `json:"daily_budget"`
	}
	if err := r.list(ctx, "/"+actID+"/adsets", "id,name,campaign_id,status,daily_budget", 25, &raw); err != nil {
		return nil, err
	}
	out := make([]AdSet, 0, len(raw))
	for _, s := range raw {
		out = append(out, AdSet{ID: s.ID, CampaignID: s.CampaignID, Name: s.Name, Status: orDefault(s.Status, "ACTIVE"), DailyBudget: minor(s.DailyBudget)})
	}
	return out, nil
}

func (r *Real) ListAds(ctx context.Context, actID string) ([]Ad, error) {
	var raw []struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Status     *string `json:"status"`
		AdSetID    string  `json:"adset_id"`
		CampaignID string  `json:"campaign_id"`
		Creative   *struct {
			ThumbnailURL *string `json:"thumbnail_url"`
		} `json:"creative"`
	}
	if err := r.list(ctx, "/"+actID+"/ads", "id,name,status,adset_id,campaign_id,creative{thumbnail_url}", 25, &raw); err != nil {
		return nil, err
	}
	out := make([]Ad, 0, len(raw))
	for _, a := range raw {
		ad := Ad{ID: a.ID, AdSetID: a.AdSetID, CampaignID: a.CampaignID, Name: a.Name, Status: orDefault(a.Status, "ACTIVE")}
		if a.Creative != nil {
			ad.ThumbnailURL = a.Creative.ThumbnailURL
		}
		out = append(out, ad)
	}
	return out, nil
}

type rawAction struct {
	ActionType *string `json:"action_type"`
	Value      *string `json:"value"`
}

type resultRule struct {
	typ   string
	match func(string) bool
}

// resultPriority picks the "result" from Meta's actions list.
var resultPriority = []resultRule{
	{"purchase", func(t string) bool { return strings.Contains(t, "purchase") }},
	{"lead", func(t string) bool { return t == "lead" || strings.Contains(t, ".lead") || strings.Contains(t, "lead_grouped") }},
	{"messaging_conversation_started", func(t string) bool { return strings.Contains(t, "messaging_conversation_started") }},
	{"link_click", func(t string) bool { return t == "link_click" }},
}

// PickResult sums the values of the first matching action family.
func PickResult(actions []rawAction) (float64, *string) {
	if len(actions) == 0 {
		return 0, nil
	}
	for _, rule := range resultPriority {
		sum := 0.0
		hit := false
		for _, a := range actions {
			if a.ActionType != nil && rule.match(*a.ActionType) {
				hit = true
				sum += num(a.Value)
			}
		}
		if hit {
			t := rule.typ
			return sum, &t
		}
	}
	return 0, nil
}

func pickPurchaseValue(values []rawAction) *float64 {
	sum := 0.0
	hit := false
	for _, a := range values {
		if a.ActionType != nil && strings.Contains(*a.ActionType, "purchase") {
			hit = true
			sum += num(a.Value)
		}
	}
	if !hit {
		return nil
	}
	return &sum
}

func (r *Real) GetInsights(ctx context.Context, actID string, q InsightsQuery) ([]InsightRow, error) {
	level := q.Level
	if level == "" {
		level = "ad"
	}
	inc := q.TimeIncrement
	if inc == 0 {
		inc = 1
	}
	tr, _ := json.Marshal(map[string]string{"since": q.Since, "until": q.Until})
	params := url.Values{
		"level":          {level},
		"time_increment": {strconv.Itoa(inc)},
		"time_range":     {string(tr)},
		"fields":         {"campaign_id,adset_id,ad_id,spend,impressions,reach,clicks,inline_link_clicks,actions,action_values,frequency,date_start"},
		"limit":          {"200"},
	}
	if len(q.Breakdowns) > 0 {
		params.Set("breakdowns", strings.Join(q.Breakdowns, ","))
	}
	type rawInsight struct {
		CampaignID       *string     `json:"campaign_id"`
		AdSetID          *string     `json:"adset_id"`
		AdID             *string     `json:"ad_id"`
		Spend            *string     `json:"spend"`
		Impressions      *string     `json:"impressions"`
		Reach            *string     `json:"reach"`
		Clicks           *string     `json:"clicks"`
		InlineLinkClicks *string     `json:"inline_link_clicks"`
		Frequency        *string     `json:"frequency"`
		Actions          []rawAction `json:"actions"`
		ActionValues     []rawAction `json:"action_values"`
		DateStart        *string     `json:"date_start"`
		Age              *string     `json:"age"`
		Gender           *string     `json:"gender"`
	}
	rows, err := meta.GetAll[rawInsight](ctx, r.Client, r.Token, "/"+actID+"/insights", params, 100)
	if err != nil {
		return nil, err
	}
	out := make([]InsightRow, 0, len(rows))
	for _, x := range rows {
		results, resultType := PickResult(x.Actions)
		out = append(out, InsightRow{
			CampaignID: orDefault(x.CampaignID, ""), AdSetID: orDefault(x.AdSetID, ""), AdID: orDefault(x.AdID, ""),
			Date: orDefault(x.DateStart, q.Since), Spend: num(x.Spend), Impressions: num(x.Impressions), Reach: num(x.Reach),
			Clicks: num(x.Clicks), LinkClicks: num(x.InlineLinkClicks), Frequency: num(x.Frequency),
			Results: results, ResultType: resultType, PurchaseValue: pickPurchaseValue(x.ActionValues), Age: x.Age, Gender: x.Gender,
		})
	}
	return out, nil
}

func orDefault(s *string, d string) string {
	if s == nil {
		return d
	}
	return *s
}
