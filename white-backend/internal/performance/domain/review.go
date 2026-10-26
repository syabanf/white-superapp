package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Quarterly performance reviews (NüHabit payroll/domain/performance.go):
// work result 60% (KPI average), behaviour 30%, project contribution 10%;
// an empty component's weight is redistributed.

const (
	ReviewDraft = "draft"
	ReviewFinal = "final"

	CycleOpen   = "open"
	CycleClosed = "closed"
)

// Review actions.
const (
	ReviewSelfAssessment = "self_assessment"
	ReviewRateItem       = "rate_item"
	ReviewReviewerNotes  = "reviewer_notes"
	ReviewManagerNotes   = "manager_notes"
	ReviewSign           = "sign"
	ReviewFinalize       = "finalize"
)

// QuarterMonths is quarterMonths.
func QuarterMonths(q int) []int {
	s := (q-1)*3 + 1
	return []int{s, s + 1, s + 2}
}

// QuarterRange is quarterRange: first and last ISO date of the quarter.
func QuarterRange(year, q int) (start, end string) {
	m := QuarterMonths(q)
	last := time.Date(year, time.Month(m[2])+1, 0, 0, 0, 0, 0, time.UTC)
	return fmt.Sprintf("%d-%02d-01", year, m[0]), fmt.Sprintf("%d-%02d-%d", year, m[2], last.Day())
}

// QuarterLabel is quarterLabel: "Q3 2026".
func QuarterLabel(year, q int) string { return fmt.Sprintf("Q%d %d", q, year) }

// QuarterOf is the quarter of a month.
func QuarterOf(month int) int { return (month-1)/3 + 1 }

// CombineReviewScores is combineReviewScores (nil when nothing is scored).
func CombineReviewScores(work, behavior, project *float64) *float64 {
	type part struct{ v, w float64 }
	var parts []part
	for _, p := range []struct {
		v *float64
		w float64
	}{{work, 60}, {behavior, 30}, {project, 10}} {
		if p.v != nil {
			parts = append(parts, part{*p.v, p.w})
		}
	}
	total := 0.0
	for _, p := range parts {
		total += p.w
	}
	if total == 0 {
		return nil
	}
	score := 0.0
	for _, p := range parts {
		score += p.v * (p.w / total)
	}
	r := Round2(score)
	return &r
}

// BehaviorItem is a rated behavioural item (score 1-5).
type BehaviorItem struct {
	Score  *int
	Weight float64
}

// BehaviorScore is behaviorScore: the weighted 1-5 average scaled to 0-100.
func BehaviorScore(items []BehaviorItem) *float64 {
	var scored []BehaviorItem
	total := 0.0
	for _, it := range items {
		if it.Score != nil && it.Weight > 0 {
			scored = append(scored, it)
			total += it.Weight
		}
	}
	if len(scored) == 0 || total <= 0 {
		return nil
	}
	avg := 0.0
	for _, it := range scored {
		avg += float64(*it.Score) / 5 * 100 * (it.Weight / total)
	}
	r := Round2(avg)
	return &r
}

// ProjectScore is parseProjectScore: nil for no score.
func ProjectScore(n *float64) (*float64, string) {
	if n == nil {
		return nil, ""
	}
	if !Finite(*n) || *n < 0 || *n > 100 {
		return nil, "Nilai kontribusi harus 0–100"
	}
	return n, ""
}

// BehaviorRating is parseBehaviorScore: an integer 1-5.
func BehaviorRating(n float64) (int, string) {
	if !Finite(n) || n != math.Trunc(n) || n < 1 || n > 5 {
		return 0, "Skor harus 1–5"
	}
	return int(n), ""
}

// Category is a performance_categories band.
type Category struct {
	Name     string
	MinScore float64
	MaxScore float64
}

// CategoryFor is the band holding score (highest min first), "" for none.
func CategoryFor(categories []Category, score *float64) string {
	if score == nil {
		return ""
	}
	sorted := append([]Category(nil), categories...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].MinScore > sorted[j].MinScore })
	for _, c := range sorted {
		if *score >= c.MinScore && *score <= c.MaxScore {
			return c.Name
		}
	}
	return ""
}

// FinalGrade maps a 360 final score to A-E.
func FinalGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	}
	return "E"
}

// FinalScore is computeFinalScore: KPI and 360 scores weighted by the
// cycle, 70/30 when the cycle's weight is null.
func FinalScore(kpi, overall360 float64, kpiWeight, feedbackWeight *float64) (float64, string) {
	kw, fw := 70.0, 30.0
	if kpiWeight != nil {
		kw = *kpiWeight
	}
	if feedbackWeight != nil {
		fw = *feedbackWeight
	}
	s := kpi*(kw/100) + overall360*(fw/100)
	return s, FinalGrade(s)
}
