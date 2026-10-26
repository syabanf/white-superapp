package domain

import (
	"strings"
	"unicode/utf16"
)

// Department logbook (NüHabit hris/logbook.go, hris/domain/insights.go and
// the DB function update_hris_logbook_entry_score): checklist templates per
// department, dated entries per employee moving draft -> submitted ->
// reviewed | rejected.

const (
	LogbookDraft     = "draft"
	LogbookSubmitted = "submitted"
	LogbookReviewed  = "reviewed"
	LogbookRejected  = "rejected"
)

// LogbookFrequencies are the template cadences.
var LogbookFrequencies = []string{"daily", "weekly", "monthly"}

// LogbookNoteMaxLength caps notes and review notes (UTF-16 units).
const LogbookNoteMaxLength = 20000

// NormalizeLogbookNote trims nothing: "" is nil, a string within the cap
// is kept; a longer one fails.
func NormalizeLogbookNote(s string) (*string, bool) {
	if len(utf16.Encode([]rune(s))) > LogbookNoteMaxLength {
		return nil, false
	}
	if s == "" {
		return nil, true
	}
	return &s, true
}

// TemplateItemWeight: nil is 1, negatives become 0.
func TemplateItemWeight(w *float64) float64 {
	if w == nil || !Finite(*w) {
		return 1
	}
	return max(0, *w)
}

// LogbookItemState is one checklist item as the score reads it.
type LogbookItemState struct {
	Checked bool
	Weight  float64
}

// LogbookScore is update_hris_logbook_entry_score: completion is checked
// over total items, the KPI score checked weight over total weight, both
// 0-100 with two decimals.
func LogbookScore(items []LogbookItemState) (completion, kpiScore float64) {
	if len(items) == 0 {
		return 0, 0
	}
	checked, total, checkedW := 0, 0.0, 0.0
	for _, it := range items {
		total += it.Weight
		if it.Checked {
			checked++
			checkedW += it.Weight
		}
	}
	completion = Round2(float64(checked) / float64(len(items)) * 100)
	if total > 0 {
		kpiScore = Round2(checkedW / total * 100)
	}
	return completion, kpiScore
}

// LogbookSummaryEntry is one entry of the summary.
type LogbookSummaryEntry struct {
	DepartmentID string
	Status       string
	Completion   float64
	KPIScore     float64
}

// LogbookDepartmentSummary is one department row of the summary.
type LogbookDepartmentSummary struct {
	DepartmentID     string
	TotalEntries     int
	SubmittedEntries int
	ReviewedEntries  int
	RejectedEntries  int
	AvgCompletion    float64
	AvgKPIScore      float64
}

// SummarizeLogbookEntries groups by department in first-seen order.
func SummarizeLogbookEntries(entries []LogbookSummaryEntry) []LogbookDepartmentSummary {
	out := []LogbookDepartmentSummary{}
	index := map[string]int{}
	for _, e := range entries {
		i, ok := index[e.DepartmentID]
		if !ok {
			i = len(out)
			index[e.DepartmentID] = i
			out = append(out, LogbookDepartmentSummary{DepartmentID: e.DepartmentID})
		}
		row := &out[i]
		row.TotalEntries++
		switch e.Status {
		case LogbookSubmitted:
			row.SubmittedEntries++
		case LogbookReviewed:
			row.ReviewedEntries++
		case LogbookRejected:
			row.RejectedEntries++
		}
		row.AvgCompletion += e.Completion
		row.AvgKPIScore += e.KPIScore
	}
	for i := range out {
		n := float64(out[i].TotalEntries)
		out[i].AvgCompletion = Round2(out[i].AvgCompletion / n)
		out[i].AvgKPIScore = Round2(out[i].AvgKPIScore / n)
	}
	return out
}

// LogbookTitle is the default entry title: "<template> - <date>".
func LogbookTitle(templateName, date, custom string) string {
	if t := strings.TrimSpace(custom); t != "" {
		return t
	}
	return templateName + " - " + date
}
