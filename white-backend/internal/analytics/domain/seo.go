package domain

import (
	"math"
	"net/url"
	"sort"

	"github.com/syabanf/white-superapp/white-backend/pkg/metrics"
)

// SearchRow is anything with clicks, impressions and a position.
type SearchRow struct {
	Clicks      int
	Impressions int
	Position    float64
}

func CTR(clicks, impressions int) float64 {
	return metrics.SafeDiv(float64(clicks), float64(impressions)) * 100
}

type SearchAgg struct {
	Clicks      int
	Impressions int
	CTR         float64
	Position    float64
}

// AggregateSearch sums clicks and impressions, recomputes CTR and weights
// position by impressions.
func AggregateSearch(rows []SearchRow) SearchAgg {
	var a SearchAgg
	vals := make([]float64, 0, len(rows))
	weights := make([]float64, 0, len(rows))
	for _, r := range rows {
		a.Clicks += r.Clicks
		a.Impressions += r.Impressions
		vals = append(vals, r.Position)
		weights = append(weights, float64(r.Impressions))
	}
	a.CTR = CTR(a.Clicks, a.Impressions)
	a.Position = metrics.WeightedMean(vals, weights)
	return a
}

func DailyRows(rows []SeoDailyMetric) []SearchRow {
	out := make([]SearchRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, SearchRow{Clicks: r.Clicks, Impressions: r.Impressions, Position: r.Position})
	}
	return out
}

var PositionBuckets = []string{"1-3", "4-10", "11-20", "21+"}

func PositionBucket(p float64) string {
	switch {
	case p <= 3:
		return "1-3"
	case p <= 10:
		return "4-10"
	case p <= 20:
		return "11-20"
	}
	return "21+"
}

type BucketRow struct {
	Bucket      string
	Queries     int
	Clicks      int
	Impressions int
}

func PositionDistribution(rows []QueryAgg) []BucketRow {
	acc := map[string]*BucketRow{}
	out := make([]BucketRow, 0, len(PositionBuckets))
	for _, b := range PositionBuckets {
		acc[b] = &BucketRow{Bucket: b}
	}
	for _, r := range rows {
		b := acc[PositionBucket(r.Position)]
		b.Queries++
		b.Clicks += r.Clicks
		b.Impressions += r.Impressions
	}
	for _, b := range PositionBuckets {
		out = append(out, *acc[b])
	}
	return out
}

var expectedCTRByPosition = map[int]float64{1: 28, 2: 15, 3: 11, 4: 8, 5: 7, 6: 5, 7: 4, 8: 3, 9: 3, 10: 2.5}

// ExpectedCTR is the benchmark organic CTR curve by rounded position.
func ExpectedCTR(position float64) float64 {
	p := int(math.Max(1, math.Round(position)))
	if p <= 10 {
		return expectedCTRByPosition[p]
	}
	if p <= 20 {
		return 1.5
	}
	return 1
}

type OpportunityRule struct {
	MinImpressions int
	StrikingMin    float64
	StrikingMax    float64
	LowCtrFraction float64
	DecliningDrop  float64
	RisingGrowth   float64
}

var DefaultOpportunityRule = OpportunityRule{MinImpressions: 100, StrikingMin: 4, StrikingMax: 15, LowCtrFraction: 0.6, DecliningDrop: 0.3, RisingGrowth: 0.3}

type Opportunity struct {
	Kind            string
	Key             string
	Clicks          int
	Impressions     int
	Position        float64
	CTR             float64
	ExpectedCTR     *float64
	PrevClicks      *int
	PotentialClicks int
}

func sortByPotential(xs []Opportunity) []Opportunity {
	sort.SliceStable(xs, func(i, j int) bool { return xs[i].PotentialClicks > xs[j].PotentialClicks })
	return xs
}

func StrikingDistance(rows []QueryAgg, rule OpportunityRule) []Opportunity {
	out := []Opportunity{}
	for _, r := range rows {
		if r.Impressions < rule.MinImpressions || r.Position < rule.StrikingMin || r.Position > rule.StrikingMax {
			continue
		}
		target := ExpectedCTR(3)
		out = append(out, Opportunity{
			Kind: "striking_distance", Key: r.Key, Clicks: r.Clicks, Impressions: r.Impressions, Position: r.Position,
			CTR: CTR(r.Clicks, r.Impressions), ExpectedCTR: &target,
			PotentialClicks: int(math.Max(0, math.Round(float64(r.Impressions)*target/100-float64(r.Clicks)))),
		})
	}
	return sortByPotential(out)
}

func LowCTR(rows []QueryAgg, rule OpportunityRule) []Opportunity {
	out := []Opportunity{}
	for _, r := range rows {
		if r.Impressions < rule.MinImpressions || r.Position > 10 {
			continue
		}
		c := CTR(r.Clicks, r.Impressions)
		exp := ExpectedCTR(r.Position)
		if c >= exp*rule.LowCtrFraction {
			continue
		}
		e := exp
		out = append(out, Opportunity{
			Kind: "low_ctr", Key: r.Key, Clicks: r.Clicks, Impressions: r.Impressions, Position: r.Position, CTR: c, ExpectedCTR: &e,
			PotentialClicks: int(math.Max(0, math.Round(float64(r.Impressions)*exp/100-float64(r.Clicks)))),
		})
	}
	return sortByPotential(out)
}

func Declining(rows []QueryAgg, rule OpportunityRule) []Opportunity {
	out := []Opportunity{}
	for _, r := range rows {
		prev := 0
		if r.PrevClicks != nil {
			prev = *r.PrevClicks
		}
		if prev <= 0 || r.Impressions < rule.MinImpressions {
			continue
		}
		if float64(prev-r.Clicks)/float64(prev) < rule.DecliningDrop {
			continue
		}
		p := prev
		out = append(out, Opportunity{
			Kind: "declining", Key: r.Key, Clicks: r.Clicks, Impressions: r.Impressions, Position: r.Position,
			CTR: CTR(r.Clicks, r.Impressions), PrevClicks: &p, PotentialClicks: int(math.Max(0, float64(prev-r.Clicks))),
		})
	}
	return sortByPotential(out)
}

func Rising(rows []QueryAgg, rule OpportunityRule) []Opportunity {
	out := []Opportunity{}
	for _, r := range rows {
		prev := 0
		if r.PrevClicks != nil {
			prev = *r.PrevClicks
		}
		if prev <= 0 || r.Impressions < rule.MinImpressions {
			continue
		}
		if float64(r.Clicks-prev)/float64(prev) < rule.RisingGrowth {
			continue
		}
		p := prev
		out = append(out, Opportunity{
			Kind: "rising", Key: r.Key, Clicks: r.Clicks, Impressions: r.Impressions, Position: r.Position,
			CTR: CTR(r.Clicks, r.Impressions), PrevClicks: &p, PotentialClicks: r.Clicks - prev,
		})
	}
	return sortByPotential(out)
}

type AuditScores struct {
	Performance   int
	Seo           int
	Accessibility int
	BestPractices int
}

// HealthScore weights Lighthouse categories (perf 0.3, seo 0.4, a11y 0.15,
// BP 0.15) and subtracts crawl penalties (errors x2 up to 20, warnings x0.5 up to 10).
func HealthScore(s *AuditScores, errors, warnings int) *int {
	if s == nil {
		return nil
	}
	base := float64(s.Performance)*0.3 + float64(s.Seo)*0.4 + float64(s.Accessibility)*0.15 + float64(s.BestPractices)*0.15
	penalty := metrics.Clamp(float64(errors)*2, 0, 20) + metrics.Clamp(float64(warnings)*0.5, 0, 10)
	v := int(math.Round(metrics.Clamp(base-penalty, 0, 100)))
	return &v
}

// ─── Dimension aggregation (features/seo/aggregate.ts) ───────

type DimAgg struct {
	Key         string
	Clicks      int
	Impressions int
	CTR         float64
	Position    float64
}

// AggregateDimRows groups per key: sums, impression-weighted position,
// recomputed CTR; sorted by clicks desc then impressions desc.
func AggregateDimRows(rows []SeoDimensionMetric) []DimAgg {
	type acc struct {
		clicks, impressions int
		vals, weights       []float64
	}
	accs := map[string]*acc{}
	order := []string{}
	for _, r := range rows {
		a := accs[r.Key]
		if a == nil {
			a = &acc{}
			accs[r.Key] = a
			order = append(order, r.Key)
		}
		a.clicks += r.Clicks
		a.impressions += r.Impressions
		a.vals = append(a.vals, r.Position)
		a.weights = append(a.weights, float64(r.Impressions))
	}
	out := make([]DimAgg, 0, len(order))
	for _, k := range order {
		a := accs[k]
		out = append(out, DimAgg{Key: k, Clicks: a.clicks, Impressions: a.impressions, CTR: CTR(a.clicks, a.impressions), Position: metrics.WeightedMean(a.vals, a.weights)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Clicks != out[j].Clicks {
			return out[i].Clicks > out[j].Clicks
		}
		return out[i].Impressions > out[j].Impressions
	})
	return out
}

type QueryAgg struct {
	DimAgg
	PrevClicks      *int
	PrevPosition    *float64
	PrevImpressions *int
}

// MergeCurrentPrevious attaches previous-period aggregates per key.
func MergeCurrentPrevious(current, previous []DimAgg) []QueryAgg {
	prev := map[string]DimAgg{}
	for _, p := range previous {
		prev[p.Key] = p
	}
	out := make([]QueryAgg, 0, len(current))
	for _, c := range current {
		q := QueryAgg{DimAgg: c}
		if p, ok := prev[c.Key]; ok {
			pc, pp, pi := p.Clicks, p.Position, p.Impressions
			q.PrevClicks, q.PrevPosition, q.PrevImpressions = &pc, &pp, &pi
		}
		out = append(out, q)
	}
	return out
}

// URLPath strips the origin: "/menu?x=1" from a full URL, "/" when empty,
// the raw value when it is not a URL.
func URLPath(full string) string {
	u, err := url.Parse(full)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return full
	}
	p := u.Path
	if u.RawQuery != "" {
		p += "?" + u.RawQuery
	}
	if p == "" {
		return "/"
	}
	return p
}
