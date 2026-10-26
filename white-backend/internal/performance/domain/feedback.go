package domain

import (
	"fmt"
	"sort"
	"strings"
)

// 360 feedback. NüHabit stored summaries the client computed; here the
// service derives them from the responses so strengths and weaknesses feed
// the improvement automation.

const (
	FeedbackDraft  = "draft"
	FeedbackActive = "active"
	FeedbackClosed = "closed"

	AssignmentPending    = "pending"
	AssignmentInProgress = "in_progress"
	AssignmentSubmitted  = "submitted"
	AssignmentApproved   = "approved"
	AssignmentRejected   = "rejected"

	RelSelf        = "self"
	RelManager     = "manager"
	RelPeer        = "peer"
	RelSubordinate = "subordinate"
)

// FeedbackRelationships are the reviewer relationships an assignment takes.
var FeedbackRelationships = []string{RelSelf, RelManager, RelPeer, RelSubordinate}

// AssignmentOpen reports work still owed by the reviewer.
func AssignmentOpen(status string) bool {
	return status == AssignmentPending || status == AssignmentInProgress || status == AssignmentRejected
}

// FeedbackRating is one answer as the summary reads it.
type FeedbackRating struct {
	CategoryCode string
	CategoryName string
	CriteriaName string
	Relationship string
	Rating       int
}

// FeedbackCategoryWeight is a category's code, name and weight.
type FeedbackCategoryWeight struct {
	Code   string
	Name   string
	Weight float64
}

// FeedbackSummaryScores is what GenerateSummary derives for one employee.
type FeedbackSummaryScores struct {
	// Category scores 0-100 keyed by category code; absent without answers.
	Categories map[string]float64
	Overall    *float64
	Strengths  []string
	Weaknesses []string
	Responses  int
}

const (
	strengthThreshold = 80.0
	weaknessThreshold = 70.0
	maxListed         = 3
)

// SummarizeFeedback averages the non-self ratings per category (1-5 scaled
// to 0-100), weights the categories into the overall 360 score and lists
// strengths (categories at 80 or above) and weaknesses (below 70, or the
// weakest category under 80 when none is below 70) with their lowest or
// highest criterion. Self ratings only count when nobody else answered.
func SummarizeFeedback(categories []FeedbackCategoryWeight, ratings []FeedbackRating) FeedbackSummaryScores {
	others := ratings[:0:0]
	for _, r := range ratings {
		if r.Relationship != RelSelf {
			others = append(others, r)
		}
	}
	if len(others) == 0 {
		others = ratings
	}
	out := FeedbackSummaryScores{Categories: map[string]float64{}, Strengths: []string{}, Weaknesses: []string{}, Responses: len(others)}
	type agg struct {
		sum, n float64
	}
	perCat := map[string]*agg{}
	perCrit := map[string]map[string]*agg{}
	for _, r := range others {
		a := perCat[r.CategoryCode]
		if a == nil {
			a = &agg{}
			perCat[r.CategoryCode] = a
			perCrit[r.CategoryCode] = map[string]*agg{}
		}
		a.sum += float64(r.Rating)
		a.n++
		c := perCrit[r.CategoryCode][r.CriteriaName]
		if c == nil {
			c = &agg{}
			perCrit[r.CategoryCode][r.CriteriaName] = c
		}
		c.sum += float64(r.Rating)
		c.n++
	}
	type scored struct {
		FeedbackCategoryWeight
		score float64
	}
	var list []scored
	weighted, weights := 0.0, 0.0
	for _, c := range categories {
		a := perCat[c.Code]
		if a == nil || a.n == 0 {
			continue
		}
		s := Round2(a.sum / a.n / 5 * 100)
		out.Categories[c.Code] = s
		list = append(list, scored{c, s})
		w := c.Weight
		if w <= 0 {
			w = 1
		}
		weighted += s * w
		weights += w
	}
	if weights == 0 {
		return out
	}
	overall := Round2(weighted / weights)
	out.Overall = &overall

	criterion := func(code string, best bool) string {
		name, avg := "", 0.0
		keys := make([]string, 0, len(perCrit[code]))
		for k := range perCrit[code] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			a := perCrit[code][k]
			v := a.sum / a.n
			if name == "" || (best && v > avg) || (!best && v < avg) {
				name, avg = k, v
			}
		}
		return fmt.Sprintf("%s (rata-rata %s dari 5)", name, FormatDecimal(avg, 1))
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
	for _, c := range list {
		if c.score >= strengthThreshold && len(out.Strengths) < maxListed {
			out.Strengths = append(out.Strengths, c.Name+": "+criterion(c.Code, true))
		}
	}
	for i := len(list) - 1; i >= 0; i-- {
		c := list[i]
		if c.score < weaknessThreshold && len(out.Weaknesses) < maxListed {
			out.Weaknesses = append(out.Weaknesses, c.Name+": "+criterion(c.Code, false))
		}
	}
	if len(out.Weaknesses) == 0 && len(list) > 0 && list[len(list)-1].score < strengthThreshold {
		c := list[len(list)-1]
		out.Weaknesses = append(out.Weaknesses, c.Name+": "+criterion(c.Code, false))
	}
	return out
}

// FormatDecimal is id-ID number formatting with at most digits decimals:
// "1.234,5".
func FormatDecimal(v float64, digits int) string {
	s := fmt.Sprintf("%.*f", digits, v)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, frac, _ := strings.Cut(s, ".")
	frac = strings.TrimRight(frac, "0")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	out := b.String()
	if frac != "" {
		out += "," + frac
	}
	if neg && strings.Trim(out, "0,.") != "" {
		out = "-" + out
	}
	return out
}
