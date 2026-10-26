package app

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adscsv"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
	"golang.org/x/text/unicode/norm"
)

const (
	MaxCsvBytes         = 10 * 1024 * 1024
	msgNoFile           = "Pilih file CSV terlebih dahulu."
	msgFileTooLarge     = "Ukuran file melebihi 10MB"
	msgNotCsv           = "File harus berformat .csv"
	msgCsvNoRows        = "Tidak ada baris data yang bisa dibaca dari file."
	msgAdsSyncNoAccount = "Belum ada akun iklan untuk disinkronkan."
	fallbackAdSet       = "(Impor CSV)"
)

type AdsDeltas struct {
	Spend, Results, CPR, Impressions, Reach, LinkClicks, CTR, CPC, CPM, Frequency metrics.Delta
	ROAS                                                                          *metrics.Delta
}

type AdsDashboard struct {
	HasData, IsDemo    bool
	Account            *domain.AdAccount
	CampaignOptions    []domain.AdCampaign
	Kpis, Prev         domain.AdKpis
	Deltas             AdsDeltas
	DominantResultType *string
	SpendSpark         []float64
	ResultsSpark       []float64
	Daily              []AdsPoint
	CampaignShare      []domain.CampaignShare
	Breakdown          []domain.CampaignNode
	Demographics       domain.Demographics
	TopAds             []domain.TopAd
}

type ImportSummary struct {
	Campaigns, AdSets, Ads, Days, Demographics int
	Warnings                                   []string
}

type Ads struct{ Deps }

func NewAds(d Deps) *Ads { return &Ads{d} }

func (a *Ads) Dashboard(ctx context.Context, userID, clientID string, w Window, campaignID string) (*AdsDashboard, error) {
	if _, err := a.view(ctx, userID, clientID); err != nil {
		return nil, err
	}
	accounts, err := a.Store.ListAdAccounts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	empty := domain.AggregateAds(nil)
	if len(accounts) == 0 {
		z := zeroDelta
		return &AdsDashboard{IsDemo: true, CampaignOptions: []domain.AdCampaign{}, Kpis: empty, Prev: empty, Deltas: AdsDeltas{z, z, z, z, z, z, z, z, z, z, nil}, SpendSpark: []float64{}, ResultsSpark: []float64{}, Daily: []AdsPoint{}, CampaignShare: []domain.CampaignShare{}, Breakdown: []domain.CampaignNode{}, Demographics: domain.Demographics{ByAge: []domain.DemographicRow{}, ByGender: []domain.DemographicRow{}, Matrix: []domain.DemographicRow{}}, TopAds: []domain.TopAd{}}, nil
	}
	idList := make([]string, 0, len(accounts))
	conns := make([]*string, 0, len(accounts))
	for _, acc := range accounts {
		idList = append(idList, acc.ID)
		conns = append(conns, acc.ConnectionID)
	}
	campaigns, err := a.Store.ListCampaigns(ctx, idList)
	if err != nil {
		return nil, err
	}
	var filter *string
	for _, c := range campaigns {
		if campaignID != "" && c.ID == campaignID {
			filter = &campaignID
		}
	}
	rows, err := a.Store.ListInsights(ctx, idList, w.Current.From, w.Current.To, filter)
	if err != nil {
		return nil, err
	}
	prevRows, err := a.Store.ListInsights(ctx, idList, w.Previous.From, w.Previous.To, filter)
	if err != nil {
		return nil, err
	}
	demo, err := a.Store.ListDemographics(ctx, idList, w.Current.From, w.Current.To, filter)
	if err != nil {
		return nil, err
	}
	adSets, err := a.Store.ListAdSets(ctx, idList, filter)
	if err != nil {
		return nil, err
	}
	ads, err := a.Store.ListAds(ctx, idList, filter)
	if err != nil {
		return nil, err
	}
	kpis, prev := domain.KpisWithConversion(rows), domain.KpisWithConversion(prevRows)
	daily, prevDaily := adsDaily(w.Current, rows), adsDaily(w.Previous, prevRows)
	grouped := domain.GroupInsights(rows)
	scoped := campaigns
	if filter != nil {
		scoped = nil
		for _, c := range campaigns {
			if c.ID == *filter {
				scoped = append(scoped, c)
			}
		}
	}
	breakdown := domain.BuildBreakdown(scoped, adSets, ads, grouped)
	d := func(c, p float64) metrics.Delta { return metrics.ComputeDelta(c, p) }
	deltas := AdsDeltas{
		Spend: d(kpis.Spend, prev.Spend), Results: d(float64(kpis.Results), float64(prev.Results)), CPR: d(kpis.CPR, prev.CPR),
		Impressions: d(float64(kpis.Impressions), float64(prev.Impressions)), Reach: d(float64(kpis.Reach), float64(prev.Reach)),
		LinkClicks: d(float64(kpis.LinkClicks), float64(prev.LinkClicks)), CTR: d(kpis.CTR, prev.CTR), CPC: d(kpis.CPC, prev.CPC),
		CPM: d(kpis.CPM, prev.CPM), Frequency: d(kpis.Frequency, prev.Frequency),
	}
	if kpis.ROAS != nil && prev.ROAS != nil {
		deltas.ROAS = ptr(d(*kpis.ROAS, *prev.ROAS))
	}
	first := accounts[0]
	return &AdsDashboard{
		HasData: len(rows) > 0 || len(prevRows) > 0, IsDemo: domain.IsDemo(conns), Account: &first,
		CampaignOptions: campaigns, Kpis: kpis, Prev: prev, Deltas: deltas,
		DominantResultType: domain.DominantResultType(rows),
		SpendSpark:         domain.Sparkline(daily, 0, 14), ResultsSpark: domain.Sparkline(daily, 1, 14),
		Daily: adsPoints(daily, prevDaily), CampaignShare: domain.BuildCampaignShare(breakdown), Breakdown: breakdown,
		Demographics: domain.BuildDemographics(demo), TopAds: domain.BuildTopAds(campaigns, adSets, ads, grouped),
	}, nil
}

var csvNameRe = regexp.MustCompile(`(?i)\.csv$`)

// ImportCsv imports a Meta Ads Manager export (MANAGER/ADMIN).
func (a *Ads) ImportCsv(ctx context.Context, userID, clientID, filename, contentType string, content []byte) (*ImportSummary, error) {
	if clientID == "" {
		return nil, apperr.New(apperr.Validation, "")
	}
	if _, err := a.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, apperr.New(apperr.Validation, msgNoFile)
	}
	if len(content) > MaxCsvBytes {
		return nil, apperr.New(apperr.Validation, msgFileTooLarge)
	}
	if !csvNameRe.MatchString(filename) && !strings.Contains(contentType, "csv") && contentType != "text/plain" {
		return nil, apperr.New(apperr.Validation, msgNotCsv)
	}
	parsed := adscsv.Parse(string(content))
	if len(parsed.Rows) == 0 {
		msg := msgCsvNoRows
		if len(parsed.Warnings) > 0 {
			msg = parsed.Warnings[0]
		}
		return nil, apperr.New(apperr.Validation, msg)
	}
	sum, err := a.persist(ctx, clientID, parsed.Rows, parsed.Warnings)
	now := a.now()
	if err != nil {
		_, _ = a.Store.InsertSyncJob(ctx, domain.SyncJob{ClientID: clientID, Kind: domain.KindAdsCsvImport, Status: domain.JobFailed, StartedAt: now, FinishedAt: &now, Error: ptr(err.Error())})
		return nil, apperr.Wrap(apperr.Unknown, "", err)
	}
	msg := fmt.Sprintf("%d kampanye, %d set iklan, %d iklan, %d hari data, %d baris demografi.", sum.Campaigns, sum.AdSets, sum.Ads, sum.Days, sum.Demographics)
	if _, err := a.Store.InsertSyncJob(ctx, domain.SyncJob{ClientID: clientID, Kind: domain.KindAdsCsvImport, Status: domain.JobSuccess, StartedAt: now, FinishedAt: &now, Message: &msg}); err != nil {
		return nil, err
	}
	return sum, nil
}

var (
	slugDropRe  = regexp.MustCompile(`[^a-z0-9\s-]`)
	slugSpaceRe = regexp.MustCompile(`\s+`)
	slugDashRe  = regexp.MustCompile(`-+`)
)

// Slugify mirrors lib/format.ts slugify.
func Slugify(name string) string {
	s := norm.NFKD.String(strings.ToLower(name))
	s = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, s)
	s = strings.TrimSpace(slugDropRe.ReplaceAllString(s, ""))
	s = slugSpaceRe.ReplaceAllString(s, "-")
	return slugDashRe.ReplaceAllString(s, "-")
}

func objectiveFor(resultType *string) string {
	if resultType == nil {
		return "OUTCOME_TRAFFIC"
	}
	switch *resultType {
	case "purchase":
		return "OUTCOME_SALES"
	case "lead":
		return "OUTCOME_LEADS"
	case "messaging_conversation_started":
		return "OUTCOME_ENGAGEMENT"
	case "reach", "impressions", "video_view", "thruplay":
		return "OUTCOME_AWARENESS"
	}
	return "OUTCOME_TRAFFIC"
}

func or(s *string, d string) string {
	if s != nil {
		return *s
	}
	return d
}

// persist writes parsed rows: account (or a new "Impor CSV" one), campaign,
// ad set and ad upserts keyed by Meta id or name slug, idempotent daily rows
// and demographics over the file's date span.
func (a *Ads) persist(ctx context.Context, clientID string, rows []adscsv.Row, warnings []string) (*ImportSummary, error) {
	warnings = append([]string{}, warnings...)
	accounts, err := a.Store.ListAdAccounts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	var account domain.AdAccount
	if len(accounts) > 0 {
		account = accounts[0]
	} else {
		created, err := a.Store.InsertAdAccount(ctx, domain.AdAccount{ID: ids.New(), ClientID: clientID, ExternalID: "csv_" + clientID, Name: "Impor CSV", Currency: "IDR"})
		if err != nil {
			return nil, err
		}
		account = *created
	}

	campaignKey := func(r adscsv.Row) string { return or(r.CampaignExternalID, Slugify(r.CampaignName)) }
	type campInfo struct {
		name       string
		resultType *string
	}
	campInfos := map[string]*campInfo{}
	var campOrder []string
	for _, r := range rows {
		k := campaignKey(r)
		if c, ok := campInfos[k]; !ok {
			campInfos[k] = &campInfo{name: r.CampaignName, resultType: r.ResultType}
			campOrder = append(campOrder, k)
		} else if r.ResultType != nil && c.resultType == nil {
			c.resultType = r.ResultType
		}
	}
	campaignIDs := map[string]string{}
	for _, k := range campOrder {
		c := campInfos[k]
		id, err := a.Store.UpsertCampaign(ctx, domain.AdCampaign{AccountID: account.ID, ExternalID: k, Name: c.name, Objective: objectiveFor(c.resultType), Status: "ACTIVE"}, false)
		if err != nil {
			return nil, err
		}
		campaignIDs[k] = id
	}

	adSetKey := func(r adscsv.Row) string {
		if r.AdSetName != nil {
			return campaignKey(r) + "|" + or(r.AdSetExternalID, Slugify(*r.AdSetName))
		}
		if r.AdName != nil {
			return campaignKey(r) + "|" + Slugify(r.CampaignName+" "+fallbackAdSet)
		}
		return ""
	}
	type setInfo struct{ campaignKey, externalID, name string }
	setInfos := map[string]setInfo{}
	var setOrder []string
	for _, r := range rows {
		k := adSetKey(r)
		if k == "" {
			continue
		}
		if _, ok := setInfos[k]; ok {
			continue
		}
		setInfos[k] = setInfo{campaignKey: campaignKey(r), externalID: strings.SplitN(k, "|", 2)[1], name: or(r.AdSetName, fallbackAdSet)}
		setOrder = append(setOrder, k)
	}
	adSetIDs := map[string]string{}
	for _, k := range setOrder {
		si := setInfos[k]
		id, err := a.Store.UpsertAdSet(ctx, domain.AdSet{CampaignID: campaignIDs[si.campaignKey], ExternalID: si.externalID, Name: si.name, Status: "ACTIVE"}, false)
		if err != nil {
			return nil, err
		}
		adSetIDs[k] = id
	}

	adKey := func(r adscsv.Row) string {
		sk := adSetKey(r)
		if sk == "" || r.AdName == nil {
			return ""
		}
		return sk + "|" + or(r.AdExternalID, Slugify(*r.AdName))
	}
	type adInfo struct{ setKey, externalID, name string }
	adInfos := map[string]adInfo{}
	var adOrder []string
	for _, r := range rows {
		k := adKey(r)
		if k == "" {
			continue
		}
		if _, ok := adInfos[k]; ok {
			continue
		}
		adInfos[k] = adInfo{setKey: adSetKey(r), externalID: strings.SplitN(k, "|", 3)[2], name: *r.AdName}
		adOrder = append(adOrder, k)
	}
	adIDs := map[string]string{}
	for _, k := range adOrder {
		ai := adInfos[k]
		id, err := a.Store.UpsertAd(ctx, domain.Ad{AdSetID: adSetIDs[ai.setKey], ExternalID: ai.externalID, Name: ai.name, Status: "ACTIVE"}, false)
		if err != nil {
			return nil, err
		}
		adIDs[k] = id
	}

	daily := map[string]*domain.AdDailyInsight{}
	var dailyOrder []string
	campaignLevel := 0
	days := map[string]bool{}
	for _, r := range rows {
		if r.Kind != "daily" || r.Date == nil {
			continue
		}
		date, err := dates.Parse(*r.Date)
		if err != nil {
			continue
		}
		cID := idPtr(campaignIDs, campaignKey(r))
		sk, ak := adSetKey(r), adKey(r)
		if ak == "" {
			campaignLevel++
		}
		sID, aID := idPtr(adSetIDs, sk), idPtr(adIDs, ak)
		key := fmt.Sprintf("%s|%s|%s|%s", *r.Date, or(cID, ""), or(sID, ""), or(aID, ""))
		acc := daily[key]
		if acc == nil {
			acc = &domain.AdDailyInsight{AccountID: account.ID, Date: date, CampaignID: cID, AdSetID: sID, AdID: aID}
			daily[key] = acc
			dailyOrder = append(dailyOrder, key)
		}
		days[*r.Date] = true
		acc.Spend += r.Spend
		acc.Impressions += r.Impressions
		acc.Reach += r.Reach
		acc.Clicks += r.Clicks
		acc.LinkClicks += r.LinkClicks
		acc.Results += r.Results
		if r.ResultType != nil {
			acc.ResultType = r.ResultType
		}
		if r.PurchaseValue != nil {
			v := *r.PurchaseValue
			if acc.PurchaseValue != nil {
				v += *acc.PurchaseValue
			}
			acc.PurchaseValue = &v
		}
	}
	if campaignLevel > 0 {
		warnings = append(warnings, fmt.Sprintf("%d baris tanpa kolom iklan diimpor pada level kampanye/set — ekspor level iklan disarankan agar semua rincian terisi.", campaignLevel))
	}
	insights := make([]domain.AdDailyInsight, 0, len(dailyOrder))
	for _, k := range dailyOrder {
		insights = append(insights, *daily[k])
	}
	for i := 0; i < len(insights); i += 500 {
		end := min(i+500, len(insights))
		part := insights[i:end]
		if err := a.Store.WithTx(ctx, func(tx domain.Store) error { return tx.UpsertInsights(ctx, part) }); err != nil {
			return nil, err
		}
	}

	demographics := 0
	var demoRows []adscsv.Row
	var allDates []string
	for _, r := range rows {
		if r.Kind == "demographic" {
			demoRows = append(demoRows, r)
		}
		if r.Date != nil {
			allDates = append(allDates, *r.Date)
		}
	}
	if len(demoRows) > 0 {
		sort.Strings(allDates)
		from, to := dates.Today(a.now()), dates.Today(a.now())
		if len(allDates) > 0 {
			from, _ = dates.Parse(allDates[0])
			to, _ = dates.Parse(allDates[len(allDates)-1])
		}
		agg := map[string]*domain.AdDemographic{}
		var order []string
		for _, r := range demoRows {
			cID := idPtr(campaignIDs, campaignKey(r))
			age, gender := or(r.Age, "unknown"), or(r.Gender, "unknown")
			k := or(cID, "") + "|" + age + "|" + gender
			d := agg[k]
			if d == nil {
				d = &domain.AdDemographic{AccountID: account.ID, DateFrom: from, DateTo: to, CampaignID: cID, Age: age, Gender: gender}
				agg[k] = d
				order = append(order, k)
			}
			d.Spend += r.Spend
			d.Impressions += r.Impressions
			d.Reach += r.Reach
			d.Results += r.Results
		}
		demographics = len(order)
		list := make([]domain.AdDemographic, 0, len(order))
		for _, k := range order {
			list = append(list, *agg[k])
		}
		if err := a.Store.UpsertDemographics(ctx, list); err != nil {
			return nil, err
		}
	}
	if err := a.Store.TouchAdAccounts(ctx, []string{account.ID}, a.now()); err != nil {
		return nil, err
	}
	if len(warnings) > adscsv.MaxWarnings {
		warnings = warnings[:adscsv.MaxWarnings]
	}
	return &ImportSummary{Campaigns: len(campOrder), AdSets: len(setOrder), Ads: len(adOrder), Days: len(days), Demographics: demographics, Warnings: warnings}, nil
}

func idPtr(m map[string]string, k string) *string {
	if k == "" {
		return nil
	}
	if v, ok := m[k]; ok {
		return &v
	}
	return nil
}

// Sync is MANAGER/ADMIN: the Marketing API pull when a connection exists,
// otherwise (or when nothing real ran) the demo touch.
func (a *Ads) Sync(ctx context.Context, userID, clientID string) (time.Time, error) {
	if clientID == "" {
		return time.Time{}, apperr.New(apperr.Validation, "")
	}
	if _, err := a.manage(ctx, userID, clientID); err != nil {
		return time.Time{}, err
	}
	accounts, err := a.Store.ListAdAccounts(ctx, clientID)
	if err != nil {
		return time.Time{}, err
	}
	if len(accounts) == 0 {
		return time.Time{}, apperr.New(apperr.Validation, msgAdsSyncNoAccount)
	}
	now := a.now()
	hasConn := false
	idList := make([]string, 0, len(accounts))
	for _, acc := range accounts {
		idList = append(idList, acc.ID)
		if acc.ConnectionID != nil {
			hasConn = true
		}
	}
	if hasConn {
		r, err := a.Syncer.SyncAdsForClient(ctx, clientID, 90)
		if err != nil {
			return time.Time{}, err
		}
		if r.Real == 0 {
			for _, x := range r.Results {
				if x.Err != "" {
					return time.Time{}, apperr.New(apperr.Unknown, x.Err)
				}
			}
		} else {
			return now, nil
		}
	}
	fail := func(err error) (time.Time, error) {
		_, _ = a.Store.InsertSyncJob(ctx, domain.SyncJob{ClientID: clientID, Kind: domain.KindAdsInsights, Status: domain.JobFailed, StartedAt: now, FinishedAt: &now, Error: ptr(err.Error())})
		return time.Time{}, apperr.Wrap(apperr.Unknown, "", err)
	}
	count, err := a.Store.CountCampaigns(ctx, idList)
	if err != nil {
		return fail(err)
	}
	if err := a.Store.TouchAdAccounts(ctx, idList, now); err != nil {
		return fail(err)
	}
	msg := fmt.Sprintf("Sinkronisasi demo: %d kampanye diperbarui.", count)
	if _, err := a.Store.InsertSyncJob(ctx, domain.SyncJob{ClientID: clientID, Kind: domain.KindAdsInsights, Status: domain.JobSuccess, StartedAt: now, FinishedAt: &now, Message: &msg}); err != nil {
		return fail(err)
	}
	return now, nil
}
