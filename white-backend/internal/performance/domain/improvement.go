package domain

import (
	"slices"
	"strings"
)

// "Things to improve" (new for WHITE): improvement items per employee or
// department, filed by people or by automations, and monthly retros.

const (
	ScopeEmployee   = "employee"
	ScopeDepartment = "department"

	ImpSuggested  = "suggested"
	ImpOpen       = "open"
	ImpInProgress = "in_progress"
	ImpDone       = "done"
	ImpDropped    = "dropped"
	ImpDeclined   = "declined"

	SourceReview        = "review"
	SourceFeedback      = "feedback"
	SourceTaskReject    = "task_reject"
	SourceLogbookReject = "logbook_reject"
	SourceRetro         = "retro"
	SourceManualItem    = "manual"
	SourceAI            = "ai"

	RetroDraft  = "draft"
	RetroClosed = "closed"
)

// ImprovementSources lists the accepted sources.
var ImprovementSources = []string{SourceReview, SourceFeedback, SourceTaskReject, SourceLogbookReject, SourceRetro, SourceManualItem, SourceAI}

var improvementTransitions = map[string][]string{
	ImpSuggested:  {ImpOpen, ImpDeclined},
	ImpOpen:       {ImpInProgress, ImpDone, ImpDropped},
	ImpInProgress: {ImpOpen, ImpDone, ImpDropped},
	ImpDone:       {ImpOpen},
	ImpDropped:    {ImpOpen},
	ImpDeclined:   {},
}

// CanTransitionImprovement reports an allowed status change.
func CanTransitionImprovement(from, to string) bool {
	return from == to || slices.Contains(improvementTransitions[from], to)
}

// ImprovementActive reports an item someone still has to act on.
func ImprovementActive(status string) bool { return status == ImpOpen || status == ImpInProgress }

// Title limits of an improvement.
const (
	ImprovementTitleMin = 3
	ImprovementTitleMax = 200
)

// ImprovementTitle derives a title from free text: its first line, cut at
// the limit on a word boundary.
func ImprovementTitle(prefix, text string) string {
	line := strings.TrimSpace(text)
	if i := strings.IndexAny(line, "\n."); i > 0 {
		line = line[:i]
	}
	t := strings.TrimSpace(prefix + line)
	r := []rune(t)
	if len(r) <= ImprovementTitleMax {
		return t
	}
	cut := string(r[:ImprovementTitleMax-3])
	if i := strings.LastIndex(cut, " "); i > ImprovementTitleMax/2 {
		cut = cut[:i]
	}
	return cut + "..."
}

// RetroAction is one action of a retro; on close it becomes an
// improvement item (ImprovementID set).
type RetroAction struct {
	Description     string  `json:"description"`
	OwnerEmployeeID *string `json:"ownerEmployeeId,omitempty"`
	DueDate         *string `json:"dueDate,omitempty"`
	ImprovementID   *string `json:"improvementId,omitempty"`
}

// CleanList trims, drops empty lines and caps the count.
func CleanList(items []string, limit int) []string {
	out := []string{}
	for _, s := range items {
		if t := strings.TrimSpace(s); t != "" && len(out) < limit {
			out = append(out, t)
		}
	}
	return out
}
