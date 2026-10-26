package domain

import (
	"encoding/json"
	"time"
)

// Task vocabularies of lib/sales-funnel/tasks.ts.
var (
	TaskSubjectTypes = []string{"lead", "deal", "account", "contact"}
	TaskPriorities   = []string{"low", "normal", "high", "urgent"}
	TaskStatuses     = []string{"open", "in_progress", "done", "cancelled"}
	RecurrenceFreqs  = []string{"daily", "weekly", "monthly"}
)

// Recurrence is recurrenceSchema's output.
type Recurrence struct {
	Freq     string  `json:"freq"`
	Interval int     `json:"interval"`
	Until    *string `json:"until"`
}

// ParseRecurrence is recurrenceSchema.safeParse on a stored jsonb value:
// strict object, freq enum, integer interval 1..52 (default 1), until
// YYYY-MM-DD or null.
func ParseRecurrence(raw []byte) (Recurrence, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return Recurrence{}, false
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return Recurrence{}, false
	}
	for k := range obj {
		if k != "freq" && k != "interval" && k != "until" {
			return Recurrence{}, false
		}
	}
	freq, ok := obj["freq"].(string)
	if !ok || !Contains(RecurrenceFreqs, freq) {
		return Recurrence{}, false
	}
	r := Recurrence{Freq: freq, Interval: 1}
	if v, present := obj["interval"]; present {
		n, ok := v.(float64)
		if !ok || n != float64(int(n)) || n < 1 || n > 52 {
			return Recurrence{}, false
		}
		r.Interval = int(n)
	}
	if v, present := obj["until"]; present && v != nil {
		s, ok := v.(string)
		if !ok || !IsValidCalendarDate(s) {
			return Recurrence{}, false
		}
		r.Until = &s
	}
	return r, true
}

// ResolveTaskSubject is resolveTaskSubject: deal_id, then lead_id, then the
// explicit subject.
func ResolveTaskSubject(subjectType, subjectID, dealID, leadID string) (string, string, bool) {
	switch {
	case dealID != "":
		return "deal", dealID, true
	case leadID != "":
		return "lead", leadID, true
	case subjectType != "" && subjectID != "":
		return subjectType, subjectID, true
	}
	return "", "", false
}

// IsTaskOpen is isTaskOpen.
func IsTaskOpen(status string) bool { return status == "open" || status == "in_progress" }

func addMonthsClamped(t time.Time, months int) time.Time {
	day := t.Day()
	first := time.Date(t.Year(), t.Month(), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC).AddDate(0, months, 0)
	last := time.Date(first.Year(), first.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return first.AddDate(0, 0, min(day, last)-1)
}

// NextOccurrence is nextOccurrence: nil once past until (inclusive, end of
// that UTC day); monthly clamps to the end of the month.
func NextOccurrence(from time.Time, r Recurrence) *time.Time {
	from = from.UTC()
	var next time.Time
	switch r.Freq {
	case "daily":
		next = from.Add(time.Duration(r.Interval) * 24 * time.Hour)
	case "weekly":
		next = from.Add(time.Duration(r.Interval) * 7 * 24 * time.Hour)
	default:
		next = addMonthsClamped(from, r.Interval)
	}
	if r.Until != nil {
		u, err := time.Parse("2006-01-02", *r.Until)
		if err == nil && next.After(u.Add(24*time.Hour-time.Millisecond)) {
			return nil
		}
	}
	return &next
}

// SpawnNextTask is spawnNextTask: the next due date, with the reminder
// shifted by the same offset. ok is false when the series is over.
func SpawnNextTask(due time.Time, reminder *time.Time, r Recurrence) (nextDue time.Time, nextReminder *time.Time, ok bool) {
	next := NextOccurrence(due, r)
	if next == nil {
		return time.Time{}, nil, false
	}
	if reminder != nil {
		rem := next.Add(-due.Sub(*reminder))
		nextReminder = &rem
	}
	return *next, nextReminder, true
}
