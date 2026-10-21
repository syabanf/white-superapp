package domain

import (
	"sort"

	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

type AdKpis struct {
	Spend       float64
	Impressions int
	Reach       int
	Clicks      int
	LinkClicks  int
	Results     int
	CPR         float64
	CTR         float64
	CPC         float64
	CPM         float64
	Frequency   float64
	// ROAS is nil unless some row carries a purchase value.
	ROAS *float64
}

// AggregateAds sums every field (reach across days is an approximation).
func AggregateAds(rows []AdDailyInsight) AdKpis {
	var k AdKpis
	var purchase float64
	hasPurchase := false
	for _, r := range rows {
		k.Spend += r.Spend
		k.Impressions += r.Impressions
		k.Reach += r.Reach
		k.Clicks += r.Clicks
		k.LinkClicks += r.LinkClicks
		k.Results += r.Results
		if r.PurchaseValue != nil {
			hasPurchase = true
			purchase += *r.PurchaseValue
		}
	}
	k.CPR = metrics.SafeDiv(k.Spend, float64(k.Results))
	k.CTR = metrics.SafeDiv(float64(k.LinkClicks), float64(k.Impressions)) * 100
	k.CPC = metrics.SafeDiv(k.Spend, float64(k.LinkClicks))
	k.CPM = metrics.SafeDiv(k.Spend, float64(k.Impressions)) * 1000
	k.Frequency = metrics.SafeDiv(float64(k.Impressions), float64(k.Reach))
	if hasPurchase && k.Spend != 0 {
		roas := purchase / k.Spend
		k.ROAS = &roas
	}
	return k
}

// NonConversionResultTypes are awareness results that never count as "Hasil".
var NonConversionResultTypes = map[string]bool{"reach": true, "impressions": true, "video_view": true, "thruplay": true}

func IsConversionType(resultType *string) bool {
	if resultType == nil {
		return true
	}
	return !NonConversionResultTypes[*resultType]
}

func IsConversion(r AdDailyInsight) bool { return IsConversionType(r.ResultType) }

func ConversionRows(rows []AdDailyInsight) []AdDailyInsight {
	out := make([]AdDailyInsight, 0, len(rows))
	for _, r := range rows {
		if IsConversion(r) {
			out = append(out, r)
		}
	}
	return out
}

// KpisWithConversion aggregates every row but takes results, CPR and ROAS
// from conversion rows only.
func KpisWithConversion(rows []AdDailyInsight) AdKpis {
	all := AggregateAds(rows)
	conv := AggregateAds(ConversionRows(rows))
	all.Results, all.CPR, all.ROAS = conv.Results, conv.CPR, conv.ROAS
	return all
}

// DailySource zeroes results of non-conversion rows when any conversion row
// exists, so the daily results chart matches the KPI.
func DailySource(rows []AdDailyInsight) []AdDailyInsight {
	if len(ConversionRows(rows)) == 0 {
		return rows
	}
	out := make([]AdDailyInsight, 0, len(rows))
	for _, r := range rows {
		if !IsConversion(r) {
			r.Results = 0
		}
		out = append(out, r)
	}
	return out
}

// DominantResultType is the type with the largest result sum among
// conversion rows, falling back to all rows; nil when none. Ties keep the
// first type reached in row order.
func DominantResultType(rows []AdDailyInsight) *string {
	tally := func(rs []AdDailyInsight) *string {
		sums := map[string]int{}
		order := []string{}
		for _, r := range rs {
			if r.ResultType == nil {
				continue
			}
			if _, ok := sums[*r.ResultType]; !ok {
				order = append(order, *r.ResultType)
			}
			sums[*r.ResultType] += r.Results
		}
		var best *string
		max := -1
		for _, k := range order {
			if sums[k] > max {
				max = sums[k]
				kk := k
				best = &kk
			}
		}
		return best
	}
	if b := tally(ConversionRows(rows)); b != nil {
		return b
	}
	return tally(rows)
}

type NodeKpis struct {
	Spend       float64
	Results     int
	ResultType  *string
	CPR         float64
	Impressions int
	Reach       int
	LinkClicks  int
	CTR         float64
	CPC         float64
	Frequency   float64
}

func nodeKpis(rows []AdDailyInsight) NodeKpis {
	k := AggregateAds(rows)
	n := NodeKpis{Spend: k.Spend, Results: k.Results, CPR: k.CPR, Impressions: k.Impressions, Reach: k.Reach, LinkClicks: k.LinkClicks, CTR: k.CTR, CPC: k.CPC, Frequency: k.Frequency}
	for _, r := range rows {
		if r.ResultType != nil {
			n.ResultType = r.ResultType
			break
		}
	}
	return n
}

type AdNode struct {
	ID           string
	Name         string
	Status       string
	ThumbnailURL *string
	Kpis         NodeKpis
}

type AdSetNode struct {
	ID     string
	Name   string
	Status string
	Kpis   NodeKpis
	Ads    []AdNode
}

type CampaignNode struct {
	ID        string
	Name      string
	Status    string
	Objective string
	Kpis      NodeKpis
	AdSets    []AdSetNode
}

type GroupedInsights struct {
	ByCampaign map[string][]AdDailyInsight
	ByAdSet    map[string][]AdDailyInsight
	ByAd       map[string][]AdDailyInsight
	adOrder    []string
}

func GroupInsights(rows []AdDailyInsight) GroupedInsights {
	g := GroupedInsights{ByCampaign: map[string][]AdDailyInsight{}, ByAdSet: map[string][]AdDailyInsight{}, ByAd: map[string][]AdDailyInsight{}}
	for _, r := range rows {
		if r.CampaignID != nil {
			g.ByCampaign[*r.CampaignID] = append(g.ByCampaign[*r.CampaignID], r)
		}
		if r.AdSetID != nil {
			g.ByAdSet[*r.AdSetID] = append(g.ByAdSet[*r.AdSetID], r)
		}
		if r.AdID != nil {
			if _, ok := g.ByAd[*r.AdID]; !ok {
				g.adOrder = append(g.adOrder, *r.AdID)
			}
			g.ByAd[*r.AdID] = append(g.ByAd[*r.AdID], r)
		}
	}
	return g
}

// BuildBreakdown is the campaign -> ad set -> ad tree, every level sorted by spend desc.
func BuildBreakdown(campaigns []AdCampaign, adSets []AdSet, ads []Ad, g GroupedInsights) []CampaignNode {
	adsBySet := map[string][]Ad{}
	for _, a := range ads {
		adsBySet[a.AdSetID] = append(adsBySet[a.AdSetID], a)
	}
	setsByCampaign := map[string][]AdSet{}
	for _, s := range adSets {
		setsByCampaign[s.CampaignID] = append(setsByCampaign[s.CampaignID], s)
	}
	out := make([]CampaignNode, 0, len(campaigns))
	for _, c := range campaigns {
		var sets []AdSetNode
		for _, s := range setsByCampaign[c.ID] {
			var adNodes []AdNode
			for _, a := range adsBySet[s.ID] {
				adNodes = append(adNodes, AdNode{ID: a.ID, Name: a.Name, Status: a.Status, ThumbnailURL: a.ThumbnailURL, Kpis: nodeKpis(g.ByAd[a.ID])})
			}
			sort.SliceStable(adNodes, func(i, j int) bool { return adNodes[i].Kpis.Spend > adNodes[j].Kpis.Spend })
			sets = append(sets, AdSetNode{ID: s.ID, Name: s.Name, Status: s.Status, Kpis: nodeKpis(g.ByAdSet[s.ID]), Ads: adNodes})
		}
		sort.SliceStable(sets, func(i, j int) bool { return sets[i].Kpis.Spend > sets[j].Kpis.Spend })
		out = append(out, CampaignNode{ID: c.ID, Name: c.Name, Status: c.Status, Objective: c.Objective, Kpis: nodeKpis(g.ByCampaign[c.ID]), AdSets: sets})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Kpis.Spend > out[j].Kpis.Spend })
	return out
}

type CampaignShare struct {
	ID         string
	Name       string
	Spend      float64
	Results    int
	ResultType *string
}

func BuildCampaignShare(breakdown []CampaignNode) []CampaignShare {
	out := []CampaignShare{}
	for _, c := range breakdown {
		if c.Kpis.Spend > 0 {
			out = append(out, CampaignShare{ID: c.ID, Name: c.Name, Spend: c.Kpis.Spend, Results: c.Kpis.Results, ResultType: c.Kpis.ResultType})
		}
	}
	return out
}

type TopAd struct {
	ID           string
	Name         string
	CampaignName string
	Status       string
	ThumbnailURL *string
	Spend        float64
	Results      int
	ResultType   *string
	CPR          float64
	CTR          float64
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 0 {
		return (s[mid-1] + s[mid]) / 2
	}
	return s[mid]
}

// BuildTopAds keeps conversion ads with results >= median (or >= 5), sorted
// by CPR ascending, top 6.
func BuildTopAds(campaigns []AdCampaign, adSets []AdSet, ads []Ad, g GroupedInsights) []TopAd {
	adMeta := map[string]Ad{}
	for _, a := range ads {
		adMeta[a.ID] = a
	}
	setToCampaign := map[string]string{}
	for _, s := range adSets {
		setToCampaign[s.ID] = s.CampaignID
	}
	campaignName := map[string]string{}
	for _, c := range campaigns {
		campaignName[c.ID] = c.Name
	}
	type perAd struct {
		id         string
		k          AdKpis
		resultType *string
	}
	var list []perAd
	for _, adID := range g.adOrder {
		rows := ConversionRows(g.ByAd[adID])
		if len(rows) == 0 {
			continue
		}
		k := AggregateAds(rows)
		if k.Results <= 0 {
			continue
		}
		list = append(list, perAd{id: adID, k: k, resultType: rows[0].ResultType})
	}
	results := make([]float64, 0, len(list))
	for _, p := range list {
		results = append(results, float64(p.k.Results))
	}
	med := median(results)
	kept := list[:0]
	for _, p := range list {
		if float64(p.k.Results) >= med || p.k.Results >= 5 {
			kept = append(kept, p)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].k.CPR < kept[j].k.CPR })
	if len(kept) > 6 {
		kept = kept[:6]
	}
	out := make([]TopAd, 0, len(kept))
	for _, p := range kept {
		t := TopAd{ID: p.id, Name: p.id, CampaignName: "–", Status: "ACTIVE", Spend: p.k.Spend, Results: p.k.Results, ResultType: p.resultType, CPR: p.k.CPR, CTR: p.k.CTR}
		if meta, ok := adMeta[p.id]; ok {
			t.Name, t.Status, t.ThumbnailURL = meta.Name, meta.Status, meta.ThumbnailURL
			if cid, ok := setToCampaign[meta.AdSetID]; ok && campaignName[cid] != "" {
				t.CampaignName = campaignName[cid]
			}
		}
		out = append(out, t)
	}
	return out
}

var (
	ageOrder    = []string{"13-17", "18-24", "25-34", "35-44", "45-54", "55-64", "65+", "unknown"}
	genderOrder = []string{"female", "male", "unknown"}
)

func orderIndex(order []string, v string) int {
	for i, o := range order {
		if o == v {
			return i
		}
	}
	return len(order)
}

type DemographicRow struct {
	Age     string
	Gender  string
	Spend   float64
	Results int
}

type Demographics struct {
	HasData  bool
	ByAge    []DemographicRow
	ByGender []DemographicRow
	Matrix   []DemographicRow
}

// BuildDemographics sums spend and results by age, by gender and by age|gender.
func BuildDemographics(rows []AdDemographic) Demographics {
	type sum struct {
		spend   float64
		results int
	}
	agg := func(key func(AdDemographic) [2]string) []DemographicRow {
		m := map[[2]string]*sum{}
		order := [][2]string{}
		for _, r := range rows {
			k := key(r)
			s := m[k]
			if s == nil {
				s = &sum{}
				m[k] = s
				order = append(order, k)
			}
			s.spend += r.Spend
			s.results += r.Results
		}
		out := make([]DemographicRow, 0, len(order))
		for _, k := range order {
			out = append(out, DemographicRow{Age: k[0], Gender: k[1], Spend: m[k].spend, Results: m[k].results})
		}
		return out
	}
	byAge := agg(func(r AdDemographic) [2]string { return [2]string{r.Age, ""} })
	sort.SliceStable(byAge, func(i, j int) bool {
		a, b := orderIndex(ageOrder, byAge[i].Age), orderIndex(ageOrder, byAge[j].Age)
		if a != b {
			return a < b
		}
		return byAge[i].Age < byAge[j].Age
	})
	byGender := agg(func(r AdDemographic) [2]string { return [2]string{"", r.Gender} })
	sort.SliceStable(byGender, func(i, j int) bool {
		return orderIndex(genderOrder, byGender[i].Gender) < orderIndex(genderOrder, byGender[j].Gender)
	})
	matrix := agg(func(r AdDemographic) [2]string { return [2]string{r.Age, r.Gender} })
	sort.SliceStable(matrix, func(i, j int) bool {
		a, b := orderIndex(ageOrder, matrix[i].Age), orderIndex(ageOrder, matrix[j].Age)
		if a != b {
			return a < b
		}
		return orderIndex(genderOrder, matrix[i].Gender) < orderIndex(genderOrder, matrix[j].Gender)
	})
	return Demographics{HasData: len(rows) > 0, ByAge: byAge, ByGender: byGender, Matrix: matrix}
}

type CampaignRow struct {
	ID           string
	Name         string
	Objective    string
	Status       string
	Spend        float64
	Impressions  int
	Results      int
	ResultType   *string
	CPR          float64
	IsConversion bool
}

// CampaignRows groups ad-level rows per campaign for reports: sums, first
// non-null result type, CPR only for conversion campaigns, spend desc.
func CampaignRows(rows []AdDailyInsight, campaigns []AdCampaign) []CampaignRow {
	type acc struct {
		spend                float64
		impressions, results int
		resultType           *string
	}
	byCampaign := map[string]*acc{}
	order := []string{}
	for _, r := range rows {
		if r.CampaignID == nil || r.AdID == nil {
			continue
		}
		a := byCampaign[*r.CampaignID]
		if a == nil {
			a = &acc{}
			byCampaign[*r.CampaignID] = a
			order = append(order, *r.CampaignID)
		}
		a.spend += r.Spend
		a.impressions += r.Impressions
		a.results += r.Results
		if a.resultType == nil {
			a.resultType = r.ResultType
		}
	}
	meta := map[string]AdCampaign{}
	for _, c := range campaigns {
		meta[c.ID] = c
	}
	out := make([]CampaignRow, 0, len(order))
	for _, id := range order {
		a := byCampaign[id]
		row := CampaignRow{ID: id, Name: "—", Objective: "", Status: "ACTIVE", Spend: a.spend, Impressions: a.impressions, Results: a.results, ResultType: a.resultType, IsConversion: IsConversionType(a.resultType)}
		if c, ok := meta[id]; ok {
			row.Name, row.Objective, row.Status = c.Name, c.Objective, c.Status
		}
		if row.IsConversion {
			row.CPR = metrics.SafeDiv(a.spend, float64(a.results))
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Spend > out[j].Spend })
	return out
}
