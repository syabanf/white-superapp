package domain

import (
	"slices"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

// PDF data (features/reports/pdf/build-data.ts): everything the renderer
// needs, pre-formatted.

type PdfKpi struct {
	Label string
	Value string
	// Formatted delta like "+12,3%"; empty hides it.
	Delta string
	// true good, false bad, nil neutral.
	DeltaGood *bool
}

type ChartPoint struct {
	Date  string
	Value float64
}

type PdfChart struct {
	Title    string
	Points   []ChartPoint
	MaxLabel string
}

type PdfSection struct {
	Title string
	Kpis  []PdfKpi
	Table *Table
}

type PdfModule struct {
	Key      string
	Title    string
	Kpis     []PdfKpi
	Chart    *PdfChart
	Tables   []Table
	Sections []PdfSection
}

type PdfData struct {
	Lang           string
	Title          string
	ClientName     string
	PeriodLabel    string
	ComparedLabel  string // empty when compare is off
	GeneratedLabel string
	Modules        []PdfModule
	AiSummary      string
}

type PdfArgs struct {
	ClientName string
	Currency   string
	Title      string
	Current    dates.Range
	Previous   dates.Range
	Compare    bool
	Modules    []string
	Language   string
	AiSummary  string
	Now        time.Time
}

// Objective labels of the campaign table.
var objectiveLabel = map[string]string{
	"OUTCOME_TRAFFIC":       "Traffic",
	"OUTCOME_SALES":         "Sales",
	"OUTCOME_LEADS":         "Leads",
	"OUTCOME_ENGAGEMENT":    "Engagement",
	"OUTCOME_AWARENESS":     "Awareness",
	"OUTCOME_APP_PROMOTION": "App",
}

// kpi shows a delta only when there is one and its pct is known.
func kpi(label, value string, delta *metrics.Delta, lowerIsBetter bool) PdfKpi {
	if delta == nil || delta.Pct == nil {
		return PdfKpi{Label: label, Value: value}
	}
	return PdfKpi{Label: label, Value: value, Delta: FormatDeltaPercent(*delta.Pct), DeltaGood: metrics.IsGoodChange(*delta, lowerIsBetter)}
}

func toPdfSections(sections []Section) []PdfSection {
	out := make([]PdfSection, 0, len(sections))
	for _, s := range sections {
		kpis := make([]PdfKpi, 0, len(s.Kpis))
		for _, k := range s.Kpis {
			kpis = append(kpis, kpi(k.Label, k.Value, k.Delta, k.LowerIsBetter))
		}
		out = append(out, PdfSection{Title: s.Title, Kpis: kpis, Table: s.Table})
	}
	return out
}

// BuildPdfData shapes the module pages in the order SOCIAL, SEO, ADS. Demo
// data is not suppressed. A SOCIAL or SEO module without channel data still
// gets a page when one of its sections has data; ADS does not.
func BuildPdfData(a PdfArgs, data ReportData, sections Sections) PdfData {
	lang := Lang(a.Language)
	L := LabelsFor(lang)
	d := func(x metrics.Delta) *metrics.Delta {
		if !a.Compare {
			return nil
		}
		return &x
	}
	want := func(m string) bool { return slices.Contains(a.Modules, m) }
	var blocks []PdfModule
	sectionsOnly := func(key, title string, extra []Section) {
		if len(extra) > 0 {
			blocks = append(blocks, PdfModule{Key: key, Title: title, Sections: toPdfSections(extra)})
		}
	}

	if want(ModuleSocial) {
		if data.Social != nil && data.Social.Overview.HasData {
			blocks = append(blocks, socialModule(data.Social, d, L, toPdfSections(sections.Social)))
		} else {
			sectionsOnly(ModuleSocial, L.Social, sections.Social)
		}
	}
	if want(ModuleSeo) {
		if data.Seo != nil && data.Seo.HasData {
			blocks = append(blocks, seoModule(data.Seo, d, L, toPdfSections(sections.Seo)))
		} else {
			sectionsOnly(ModuleSeo, L.Seo, sections.Seo)
		}
	}
	if want(ModuleAds) && data.Ads != nil && data.Ads.HasData {
		blocks = append(blocks, adsModule(data.Ads, a.Currency, d, L))
	}

	out := PdfData{
		Lang:           lang,
		Title:          a.Title,
		ClientName:     a.ClientName,
		PeriodLabel:    FormatDateRange(a.Current.From, a.Current.To),
		GeneratedLabel: FormatDateTime(a.Now),
		Modules:        blocks,
		AiSummary:      a.AiSummary,
	}
	if a.Compare {
		out.ComparedLabel = L.VsPrevious + ": " + FormatDateRange(a.Previous.From, a.Previous.To)
	}
	return out
}

type deltaFn func(metrics.Delta) *metrics.Delta

func socialModule(s *SocialData, d deltaFn, L Labels, sections []PdfSection) PdfModule {
	ov := s.Overview
	// Running totals are near-constant, so the chart plots daily net growth.
	var growth []ChartPoint
	for i := 1; i < len(s.DailyFollowers); i++ {
		p := s.DailyFollowers[i]
		growth = append(growth, ChartPoint{Date: FormatDateShort(p.Date), Value: float64(p.Followers - s.DailyFollowers[i-1].Followers)})
	}
	var chart *PdfChart
	if len(growth) > 1 {
		chart = &PdfChart{Title: L.DailyFollowers, MaxLabel: FormatDeltaNumber(maxValue(growth)) + L.PerDay, Points: growth}
	}
	rows := [][]string{}
	for _, p := range firstN(s.TopPosts, 7) {
		platform := "TikTok"
		switch p.Platform {
		case "INSTAGRAM":
			platform = "Instagram"
		case "FACEBOOK":
			platform = "Facebook"
		}
		rows = append(rows, []string{platform, Truncate(p.Caption, 64), p.ContentType(),
			FormatNumber(float64(p.Likes), 0), FormatNumber(float64(p.Comments), 0), FormatNumber(float64(p.Engagements), 0)})
	}
	return PdfModule{
		Key:   ModuleSocial,
		Title: L.Social,
		Kpis: []PdfKpi{
			kpi("Followers", FormatCompact(float64(ov.Followers)), d(ov.FollowersDelta), false),
			kpi("Engagement rate", FormatPercent(ov.EngagementRate, 2), d(ov.EngagementRateDelta), false),
			kpi("Jangkauan", FormatCompact(float64(ov.Reach)), d(ov.ReachDelta), false),
			kpi("Postingan", FormatNumber(float64(ov.Posts), 0), d(ov.PostsDelta), false),
		},
		Chart: chart,
		Tables: []Table{{
			Title: L.TopPosts,
			Columns: []Column{
				{Label: "Platform", Width: 14, Align: "left"},
				{Label: "Caption", Width: 44, Align: "left"},
				{Label: "Tipe", Width: 12, Align: "left"},
				{Label: "Suka", Width: 10, Align: "right"},
				{Label: "Komentar", Width: 10, Align: "right"},
				{Label: "Interaksi", Width: 10, Align: "right"},
			},
			Rows: rows,
		}},
		Sections: sections,
	}
}

func seoModule(s *SeoData, d deltaFn, L Labels, sections []PdfSection) PdfModule {
	// Days GSC did not report are skipped: a zero bar would claim traffic fell.
	var reported []ChartPoint
	for _, p := range s.Daily {
		if p.Clicks != nil {
			reported = append(reported, ChartPoint{Date: FormatDateShort(p.Date), Value: float64(*p.Clicks)})
		}
	}
	var chart *PdfChart
	if len(reported) > 1 {
		chart = &PdfChart{Title: L.DailyClicks, MaxLabel: FormatCompact(maxValue(reported)), Points: reported}
	}
	kpis := []PdfKpi{
		kpi("Klik", FormatCompact(float64(s.Clicks)), d(s.ClicksDelta), false),
		kpi("Impresi", FormatCompact(float64(s.Impressions)), d(s.ImpressionsDelta), false),
		kpi("CTR", FormatPercent(s.Ctr, 2), d(s.CtrDelta), false),
		kpi("Posisi rata-rata", FormatNumber(s.Position, 1), d(s.PositionDelta), true),
	}
	if s.Health != nil {
		kpis = append(kpis, PdfKpi{Label: "Skor kesehatan", Value: FormatNumber(float64(*s.Health), 0) + "/100"})
	}
	rows := [][]string{}
	for _, q := range firstN(s.TopQueries, 8) {
		rows = append(rows, []string{q.Query, FormatNumber(float64(q.Clicks), 0), FormatCompact(float64(q.Impressions)), FormatPercent(q.Ctr, 2), FormatNumber(q.Position, 1)})
	}
	return PdfModule{
		Key:   ModuleSeo,
		Title: L.Seo,
		Kpis:  kpis,
		Chart: chart,
		Tables: []Table{{
			Title: L.TopQueries,
			Columns: []Column{
				{Label: "Kata kunci", Width: 44, Align: "left"},
				{Label: "Klik", Width: 14, Align: "right"},
				{Label: "Impresi", Width: 14, Align: "right"},
				{Label: "CTR", Width: 14, Align: "right"},
				{Label: "Posisi", Width: 14, Align: "right"},
			},
			Rows: rows,
		}},
		Sections: sections,
	}
}

func adsModule(a *AdsData, currency string, d deltaFn, L Labels) PdfModule {
	kpis := []PdfKpi{
		kpi("Belanja iklan", FormatCurrency(a.Kpis.Spend, currency, true), d(a.SpendDelta), false),
		kpi("Hasil", FormatNumber(float64(a.Kpis.Results), 0), d(a.ResultsDelta), false),
		kpi("Biaya / hasil", FormatCurrency(a.Kpis.Cpr, currency, false), d(a.CprDelta), true),
		{Label: "CTR", Value: FormatPercent(a.Kpis.Ctr, 2)},
		{Label: "CPM", Value: FormatCurrency(a.Kpis.Cpm, currency, false)},
	}
	if a.Kpis.Roas != nil {
		kpis = append(kpis, PdfKpi{Label: "ROAS", Value: FormatNumber(*a.Kpis.Roas, 2) + "x"})
	}
	var chart *PdfChart
	if len(a.Daily) > 1 {
		points := make([]ChartPoint, 0, len(a.Daily))
		for _, p := range a.Daily {
			points = append(points, ChartPoint{Date: FormatDateShort(p.Date), Value: p.Spend})
		}
		chart = &PdfChart{Title: L.DailySpend, MaxLabel: FormatCurrency(maxValue(points), currency, true), Points: points}
	}
	rows := [][]string{}
	for _, c := range a.Campaigns {
		objective, ok := objectiveLabel[c.Objective]
		if !ok {
			objective = c.Objective
		}
		cpr := "—" // copied from the old PDF: no cost per result for non-conversion rows
		if c.IsConversion {
			cpr = FormatCurrency(c.Cpr, currency, false)
		}
		rows = append(rows, []string{Truncate(c.Name, 42), objective, FormatCurrency(c.Spend, currency, true), FormatNumber(float64(c.Results), 0), cpr})
	}
	return PdfModule{
		Key:   ModuleAds,
		Title: L.Ads,
		Kpis:  kpis,
		Chart: chart,
		Tables: []Table{{
			Title: L.Campaigns,
			Columns: []Column{
				{Label: "Kampanye", Width: 38, Align: "left"},
				{Label: "Tujuan", Width: 14, Align: "left"},
				{Label: "Belanja", Width: 16, Align: "right"},
				{Label: "Hasil", Width: 12, Align: "right"},
				{Label: "Biaya/hasil", Width: 20, Align: "right"},
			},
			Rows: rows,
		}},
	}
}

func maxValue(points []ChartPoint) float64 {
	m := points[0].Value
	for _, p := range points[1:] {
		m = max(m, p.Value)
	}
	return m
}
