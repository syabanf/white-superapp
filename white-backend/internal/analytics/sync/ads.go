package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/providers/metamarketing"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

type AdsResult struct {
	Mode         string
	Campaigns    int
	AdSets       int
	Ads          int
	Days         int
	Demographics int
	Err          string
}

type AdsClientResult struct {
	Accounts int
	Real     int
	Results  []AdsResult
}

// DailyInsightAgg is one aggregate per (date, campaign, ad set, ad).
type DailyInsightAgg struct {
	Date                                   string
	CampaignExt, AdSetExt, AdExt           string
	Spend                                  float64
	Impressions, Reach, Clicks, LinkClicks int
	Results                                int
	ResultType                             *string
	PurchaseValue                          *float64
}

// AggregateDailyInsights groups ad-level rows; rows without a date or ad are skipped.
func AggregateDailyInsights(rows []metamarketing.InsightRow) []DailyInsightAgg {
	by := map[string]*DailyInsightAgg{}
	var order []string
	for _, r := range rows {
		if !isoDayRe.MatchString(r.Date) || r.AdID == "" {
			continue
		}
		key := r.Date + "|" + r.CampaignID + "|" + r.AdSetID + "|" + r.AdID
		a := by[key]
		if a == nil {
			a = &DailyInsightAgg{Date: r.Date, CampaignExt: r.CampaignID, AdSetExt: r.AdSetID, AdExt: r.AdID}
			by[key] = a
			order = append(order, key)
		}
		a.Spend += r.Spend
		a.Impressions += jsRound(r.Impressions)
		a.Reach += jsRound(r.Reach)
		a.Clicks += jsRound(r.Clicks)
		a.LinkClicks += jsRound(r.LinkClicks)
		a.Results += jsRound(r.Results)
		if r.ResultType != nil {
			a.ResultType = r.ResultType
		}
		if r.PurchaseValue != nil {
			v := *r.PurchaseValue
			if a.PurchaseValue != nil {
				v += *a.PurchaseValue
			}
			a.PurchaseValue = &v
		}
	}
	out := make([]DailyInsightAgg, 0, len(order))
	for _, k := range order {
		out = append(out, *by[k])
	}
	return out
}

type DemographicAgg struct {
	CampaignExt, Age, Gender    string
	Spend                       float64
	Impressions, Reach, Results int
}

// AggregateDemographics sums campaign-level age/gender rows over the window;
// empty age or gender becomes "unknown", gender is lowercased.
func AggregateDemographics(rows []metamarketing.InsightRow) []DemographicAgg {
	by := map[string]*DemographicAgg{}
	var order []string
	for _, r := range rows {
		if r.CampaignID == "" {
			continue
		}
		age, gender := "unknown", "unknown"
		if r.Age != nil && strings.TrimSpace(*r.Age) != "" {
			age = strings.TrimSpace(*r.Age)
		}
		if r.Gender != nil && strings.TrimSpace(*r.Gender) != "" {
			gender = strings.TrimSpace(*r.Gender)
		}
		gender = strings.ToLower(gender)
		key := r.CampaignID + "|" + age + "|" + gender
		a := by[key]
		if a == nil {
			a = &DemographicAgg{CampaignExt: r.CampaignID, Age: age, Gender: gender}
			by[key] = a
			order = append(order, key)
		}
		a.Spend += r.Spend
		a.Impressions += jsRound(r.Impressions)
		a.Reach += jsRound(r.Reach)
		a.Results += jsRound(r.Results)
	}
	out := make([]DemographicAgg, 0, len(order))
	for _, k := range order {
		out = append(out, *by[k])
	}
	return out
}

func parseStart(s *string) *time.Time {
	if s == nil {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-0700"} {
		if t, err := time.Parse(layout, *s); err == nil {
			u := t.UTC()
			return &u
		}
	}
	return nil
}

// SyncAdAccount pulls hierarchy, ad-level daily insights and campaign
// demographics for the window. Without a token it is demo and writes nothing.
func (s *Syncer) SyncAdAccount(ctx context.Context, acc domain.AdAccount, days int) (AdsResult, error) {
	adapter := s.Marketing
	if adapter == nil {
		token, err := s.token(ctx, acc.ConnectionID)
		if err != nil {
			return AdsResult{}, err
		}
		if token == "" {
			return AdsResult{Mode: "demo"}, nil
		}
		adapter = s.marketing(ctx, token)
	} else if acc.ConnectionID == nil {
		return AdsResult{Mode: "demo"}, nil
	}
	if days <= 0 {
		days = 30
	}
	until := dates.LatestCompleteDay(s.now())
	since := dates.AddDays(until, -(days - 1))
	actID := acc.ExternalID

	campaigns, err := adapter.ListCampaigns(ctx, actID)
	if err != nil {
		return AdsResult{}, err
	}
	adSets, err := adapter.ListAdSets(ctx, actID)
	if err != nil {
		return AdsResult{}, err
	}
	ads, err := adapter.ListAds(ctx, actID)
	if err != nil {
		return AdsResult{}, err
	}
	campaignIDs := map[string]string{}
	for _, c := range campaigns {
		id, err := s.Store.UpsertCampaign(ctx, domain.AdCampaign{AccountID: acc.ID, ExternalID: c.ID, Name: c.Name, Objective: c.Objective, Status: c.Status, DailyBudget: c.DailyBudget, StartTime: parseStart(c.StartTime)}, true)
		if err != nil {
			return AdsResult{}, err
		}
		campaignIDs[c.ID] = id
	}
	adSetIDs := map[string]string{}
	for _, a := range adSets {
		cid, ok := campaignIDs[a.CampaignID]
		if !ok {
			continue
		}
		id, err := s.Store.UpsertAdSet(ctx, domain.AdSet{CampaignID: cid, ExternalID: a.ID, Name: a.Name, Status: a.Status, DailyBudget: a.DailyBudget}, true)
		if err != nil {
			return AdsResult{}, err
		}
		adSetIDs[a.ID] = id
	}
	adIDs := map[string]string{}
	for _, a := range ads {
		sid, ok := adSetIDs[a.AdSetID]
		if !ok {
			continue
		}
		id, err := s.Store.UpsertAd(ctx, domain.Ad{AdSetID: sid, ExternalID: a.ID, Name: a.Name, Status: a.Status, ThumbnailURL: a.ThumbnailURL}, true)
		if err != nil {
			return AdsResult{}, err
		}
		adIDs[a.ID] = id
	}

	insights, err := adapter.GetInsights(ctx, actID, metamarketing.InsightsQuery{Since: dates.Format(since), Until: dates.Format(until), Level: "ad", TimeIncrement: 1})
	if err != nil {
		return AdsResult{}, err
	}
	seenDays := map[string]bool{}
	var rows []domain.AdDailyInsight
	for _, d := range AggregateDailyInsights(insights) {
		adID, ok := adIDs[d.AdExt]
		if !ok {
			continue
		}
		date, err := dates.Parse(d.Date)
		if err != nil {
			continue
		}
		seenDays[d.Date] = true
		row := domain.AdDailyInsight{AccountID: acc.ID, Date: date, AdID: &adID, Spend: d.Spend, Impressions: d.Impressions, Reach: d.Reach, Clicks: d.Clicks, LinkClicks: d.LinkClicks, Results: d.Results, ResultType: d.ResultType, PurchaseValue: d.PurchaseValue}
		if id, ok := campaignIDs[d.CampaignExt]; ok {
			row.CampaignID = &id
		}
		if id, ok := adSetIDs[d.AdSetExt]; ok {
			row.AdSetID = &id
		}
		rows = append(rows, row)
	}
	for i := 0; i < len(rows); i += 100 {
		end := i + 100
		if end > len(rows) {
			end = len(rows)
		}
		part := rows[i:end]
		if err := s.Store.WithTx(ctx, func(tx domain.Store) error { return tx.UpsertInsights(ctx, part) }); err != nil {
			return AdsResult{}, err
		}
	}

	demoRows, err := adapter.GetInsights(ctx, actID, metamarketing.InsightsQuery{Since: dates.Format(since), Until: dates.Format(until), Level: "campaign", Breakdowns: []string{"age", "gender"}})
	if err != nil {
		return AdsResult{}, err
	}
	var demos []domain.AdDemographic
	for _, d := range AggregateDemographics(demoRows) {
		row := domain.AdDemographic{AccountID: acc.ID, DateFrom: since, DateTo: until, Age: d.Age, Gender: d.Gender, Spend: d.Spend, Impressions: d.Impressions, Reach: d.Reach, Results: d.Results}
		if id, ok := campaignIDs[d.CampaignExt]; ok {
			row.CampaignID = &id
		}
		demos = append(demos, row)
	}
	if err := s.Store.UpsertDemographics(ctx, demos); err != nil {
		return AdsResult{}, err
	}
	if err := s.Store.TouchAdAccounts(ctx, []string{acc.ID}, s.now()); err != nil {
		return AdsResult{}, err
	}
	return AdsResult{Mode: "real", Campaigns: len(campaigns), AdSets: len(adSets), Ads: len(ads), Days: len(seenDays), Demographics: len(demos)}, nil
}

// SyncAdsForClient syncs every ad account and writes one ADS_INSIGHTS job each.
func (s *Syncer) SyncAdsForClient(ctx context.Context, clientID string, days int) (AdsClientResult, error) {
	accounts, err := s.Store.ListAdAccounts(ctx, clientID)
	if err != nil {
		return AdsClientResult{}, err
	}
	out := AdsClientResult{Accounts: len(accounts)}
	for _, a := range accounts {
		r, err := s.SyncAdAccount(ctx, a, days)
		if err != nil {
			r = AdsResult{Mode: "real", Err: errText(err)}
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
			msg = fmt.Sprintf("%d kampanye, %d iklan, %d hari · %s", r.Campaigns, r.Ads, r.Days, a.ExternalID)
		}
		if err := s.job(ctx, clientID, domain.KindAdsInsights, status, strp(msg), errMsg); err != nil {
			return out, err
		}
	}
	return out, nil
}
