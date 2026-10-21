package googlesearch

import (
	"context"
	"math"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"github.com/syabanf/white-superapp/white-backend/pkg/providers/mockrand"
)

// Mock returns the same rows for the same inputs so demo dashboards stay
// stable. It uses the xmur3 string hash of the TS mock, not FNV-1a.
type Mock struct{}

// xmur3 hashes a string to a 32-bit seed the way the TS mock did.
func xmur3(s string) uint32 {
	h := uint32(1779033703) ^ uint32(len(s))
	for i := 0; i < len(s); i++ {
		h = (h ^ uint32(s[i])) * 3432918353
		h = (h << 13) | (h >> 19)
	}
	h = (h ^ (h >> 16)) * 2246822507
	h = (h ^ (h >> 13)) * 3266489909
	return h ^ (h >> 16)
}

func jsRound(x float64) float64 { return math.Floor(x + 0.5) }

var (
	mockQueries = []string{
		"kopi susu gula aren", "kedai kopi jakarta", "biji kopi arabika", "cold brew coffee", "kopi kekinian",
		"harga kopi susu", "kopi terdekat", "resep es kopi susu", "kopi single origin", "kopi toraja",
		"coffee shop instagramable", "kopi arabika gayo", "franchise kopi", "kopi literan", "promo kopi hari ini",
		"kopi drip bag", "supplier biji kopi", "kopi tanpa gula", "manfaat kopi hitam", "cara seduh v60",
	}
	mockPages     = []string{"/", "/menu", "/lokasi", "/promo", "/blog/resep-kopi", "/produk/biji-kopi", "/tentang", "/kontak", "/blog/arabika-vs-robusta", "/franchise"}
	mockCountries = []string{"idn", "sgp", "mys", "aus", "usa"}
	mockDevices   = []string{"MOBILE", "DESKTOP", "TABLET"}
	ga4Channels   = []string{"Organic Search", "Direct", "Paid Search", "Organic Social"}
)

func eachDate(start, end string) []string {
	from, err1 := dates.Parse(start)
	to, err2 := dates.Parse(end)
	if err1 != nil || err2 != nil {
		return nil
	}
	out := []string{}
	for d, guard := from, 0; !d.After(to) && guard < 500; d, guard = dates.AddDays(d, 1), guard+1 {
		out = append(out, dates.Format(d))
	}
	return out
}

func keysFor(dim string, req GscQuery) []string {
	switch dim {
	case DimDate:
		return eachDate(req.StartDate, req.EndDate)
	case DimQuery:
		return mockQueries
	case DimPage:
		base := strings.TrimSuffix(req.SiteURL, "/")
		base = strings.Replace(base, "sc-domain:", "https://", 1)
		out := make([]string, 0, len(mockPages))
		for _, p := range mockPages {
			out = append(out, base+p)
		}
		return out
	case DimCountry:
		return mockCountries
	case DimDevice:
		return mockDevices
	}
	return nil
}

// crossProduct is the capped cartesian product of per-dimension key lists.
func crossProduct(lists [][]string, limit int) [][]string {
	out := [][]string{{}}
	for _, list := range lists {
		next := [][]string{}
	outer:
		for _, combo := range out {
			for _, k := range list {
				c := append(append([]string{}, combo...), k)
				next = append(next, c)
				if len(next) >= limit {
					break outer
				}
			}
		}
		out = next
	}
	return out
}

func (Mock) QuerySearchAnalytics(_ context.Context, _ string, req GscQuery) ([]GscRow, error) {
	limit := req.RowLimit
	if limit == 0 {
		limit = 1000
	}
	if limit > 5000 {
		limit = 5000
	}
	lists := make([][]string, 0, len(req.Dimensions))
	for _, d := range req.Dimensions {
		lists = append(lists, keysFor(d, req))
	}
	if len(lists) == 0 {
		lists = [][]string{{"(all)"}}
	}
	combos := crossProduct(lists, limit)
	rows := make([]GscRow, 0, len(combos))
	for _, keys := range combos {
		rng := mockrand.FromSeed(xmur3("gsc|" + req.SiteURL + "|" + strings.Join(keys, "|")))
		base := 20 + math.Floor(rng.Float64()*400)
		impressions := base * (5 + math.Floor(rng.Float64()*30))
		position := 1 + rng.Float64()*30
		ctrPct := math.Max(0.2, 30/position-rng.Float64()*2)
		clicks := math.Min(impressions, jsRound(impressions*ctrPct/100))
		ctr := 0.0
		if impressions > 0 {
			ctr = clicks / impressions * 100
		}
		rows = append(rows, GscRow{Keys: keys, Clicks: clicks, Impressions: impressions, CTR: ctr, Position: position})
	}
	return rows, nil
}

func (Mock) RunGa4Report(_ context.Context, _ string, req Ga4Request) ([]Ga4Row, error) {
	wantsDate, wantsChannel := false, false
	for _, d := range req.Dimensions {
		if d == "date" {
			wantsDate = true
		}
		if d == "sessionDefaultChannelGroup" {
			wantsChannel = true
		}
	}
	days := []string{"(all)"}
	if wantsDate {
		days = eachDate(req.StartDate, req.EndDate)
	}
	channels := []string{"(all)"}
	if wantsChannel {
		channels = ga4Channels
	}
	rows := []Ga4Row{}
	for _, date := range days {
		for _, channel := range channels {
			rng := mockrand.FromSeed(xmur3("ga4|" + req.PropertyID + "|" + date + "|" + channel))
			share := 1.0
			switch channel {
			case "Organic Search":
				share = 0.45
			case "Direct":
				share = 0.3
			case "Paid Search":
				share = 0.15
			case "Organic Social":
				share = 0.1
			}
			sessions := jsRound((800 + rng.Float64()*900) * share)
			values := map[string]float64{
				"sessions":               sessions,
				"engagedSessions":        jsRound(sessions * (0.5 + rng.Float64()*0.2)),
				"totalUsers":             jsRound(sessions * (0.75 + rng.Float64()*0.15)),
				"conversions":            jsRound(sessions * (0.015 + rng.Float64()*0.02)),
				"userEngagementDuration": jsRound(sessions * (40 + rng.Float64()*60)),
			}
			dims := make([]string, 0, len(req.Dimensions))
			for _, d := range req.Dimensions {
				switch d {
				case "date":
					dims = append(dims, strings.ReplaceAll(date, "-", ""))
				case "sessionDefaultChannelGroup":
					dims = append(dims, channel)
				default:
					dims = append(dims, "(other)")
				}
			}
			ms := make([]float64, 0, len(req.Metrics))
			for _, m := range req.Metrics {
				ms = append(ms, values[m])
			}
			rows = append(rows, Ga4Row{Dimensions: dims, Metrics: ms})
		}
	}
	if req.Limit > 0 && len(rows) > req.Limit {
		rows = rows[:req.Limit]
	}
	return rows, nil
}
