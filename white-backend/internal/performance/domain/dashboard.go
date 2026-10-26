package domain

import (
	"context"
	"time"
)

// Dashboard windows, buckets and the aggregate rows the store returns.

// Comparison modes of a dashboard window.
const (
	ComparePrevious = "previous"
	CompareYear     = "year"
	CompareNone     = "none"
)

// Trend bucket sizes.
const (
	BucketDay   = "day"
	BucketWeek  = "week"
	BucketMonth = "month"
)

// Dashboard scopes.
const (
	DashScopeMe   = "me"
	DashScopeTeam = "team"
	DashScopeAll  = "all"
)

// MaxDashboardDays caps a window (three years).
const MaxDashboardDays = 3 * 366

// DashWindow is a resolved dashboard range with its comparison range.
type DashWindow struct {
	From, To               string
	CompareFrom, CompareTo string // empty without comparison
	Compare                string
	Granularity            string
}

// ResolveWindow fills defaults (this month up to today) and derives the
// comparison range: the same number of days ending the day before, or the
// same days a year earlier. ok is false for unparseable or reversed days.
func ResolveWindow(from, to, compare, today string) (DashWindow, bool) {
	t, _ := ParseDay(today)
	if from == "" {
		from = t.Format("2006-01") + "-01"
	}
	if to == "" {
		to = today
	}
	f, ok1 := ParseDay(from)
	e, ok2 := ParseDay(to)
	if !ok1 || !ok2 || e.Before(f) || len(from) != 10 || len(to) != 10 {
		return DashWindow{}, false
	}
	days := int(e.Sub(f).Hours()/24) + 1
	if days > MaxDashboardDays {
		f = e.AddDate(0, 0, -MaxDashboardDays+1)
		days = MaxDashboardDays
	}
	w := DashWindow{From: f.Format(IsoDate), To: e.Format(IsoDate), Compare: compare, Granularity: GranularityFor(days)}
	switch compare {
	case CompareNone:
	case CompareYear:
		w.CompareFrom, w.CompareTo = f.AddDate(-1, 0, 0).Format(IsoDate), e.AddDate(-1, 0, 0).Format(IsoDate)
	default:
		w.Compare = ComparePrevious
		w.CompareFrom, w.CompareTo = f.AddDate(0, 0, -days).Format(IsoDate), f.AddDate(0, 0, -1).Format(IsoDate)
	}
	return w, true
}

// GranularityFor picks daily buckets up to 45 days, weekly up to 26 weeks,
// monthly beyond.
func GranularityFor(days int) string {
	switch {
	case days <= 45:
		return BucketDay
	case days <= 182:
		return BucketWeek
	}
	return BucketMonth
}

// BucketOf is the first day of the bucket holding day (weeks start Monday).
func BucketOf(day, granularity string) string {
	d, ok := ParseDay(day)
	if !ok {
		return day
	}
	switch granularity {
	case BucketWeek:
		offset := (int(d.Weekday()) + 6) % 7
		return d.AddDate(0, 0, -offset).Format(IsoDate)
	case BucketMonth:
		return d.Format("2006-01") + "-01"
	}
	return day
}

// Buckets lists the bucket keys covering [from, to] in order.
func Buckets(from, to, granularity string) []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range EachDate(from, to) {
		b := BucketOf(d, granularity)
		if !seen[b] {
			seen[b] = true
			out = append(out, b)
		}
	}
	return out
}

// MonthsUpTo lists the n months ending with the month of day, oldest
// first, as periods.
func MonthsUpTo(day string, n int) []Period {
	d, ok := ParseDay(day)
	if !ok {
		return nil
	}
	p := Period{Year: d.Year(), Month: int(d.Month())}
	out := make([]Period, n)
	for i := n - 1; i >= 0; i-- {
		out[i] = p
		p = p.Previous()
	}
	return out
}

// Workdays counts Monday to Friday days in [from, to].
func Workdays(from, to string) int {
	n := 0
	for _, d := range EachDate(from, to) {
		t, _ := ParseDay(d)
		if IsWorkday(t) {
			n++
		}
	}
	return n
}

// MinDay and MaxDay compare YYYY-MM-DD strings.
func MinDay(a, b string) string {
	if a < b {
		return a
	}
	return b
}

func MaxDay(a, b string) string {
	if a > b {
		return a
	}
	return b
}

// AddDays shifts a YYYY-MM-DD day.
func AddDays(day string, n int) string {
	d, ok := ParseDay(day)
	if !ok {
		return day
	}
	return d.AddDate(0, 0, n).Format(IsoDate)
}

// DaysBetween is whole days from a to b.
func DaysBetween(a, b string) int {
	x, ok1 := ParseDay(a)
	y, ok2 := ParseDay(b)
	if !ok1 || !ok2 {
		return 0
	}
	return int(y.Sub(x).Hours() / 24)
}

// ScoreBins are the histogram bins of a 0-100 score.
var ScoreBins = []struct {
	Key      string
	Min, Max float64
}{{"0-49", 0, 50}, {"50-59", 50, 60}, {"60-69", 60, 70}, {"70-79", 70, 80}, {"80-89", 80, 90}, {"90-100", 90, 1000}}

// ScoreBinIndex places a score in ScoreBins.
func ScoreBinIndex(score float64) int {
	for i, b := range ScoreBins {
		if score < b.Max {
			return i
		}
	}
	return len(ScoreBins) - 1
}

// OverdueAgeBins are the overdue aging buckets in days.
var OverdueAgeBins = []struct {
	Key      string
	Min, Max int
}{{"1-3", 1, 3}, {"4-7", 4, 7}, {"8-14", 8, 14}, {"15-30", 15, 30}, {"31+", 31, 1 << 30}}

// OverdueAgeIndex places an age (days late) in OverdueAgeBins.
func OverdueAgeIndex(days int) int {
	for i, b := range OverdueAgeBins {
		if days <= b.Max {
			return i
		}
	}
	return len(OverdueAgeBins) - 1
}

// BelowTargetScore marks a scorecard that needs attention.
const BelowTargetScore = 70

// ─── store rows ──────────────────────────────────────────────

// DashFilter selects the people of a dashboard query: everyone, or the
// listed employees; DepartmentID narrows by the row's department.
type DashFilter struct {
	All          bool
	EmployeeIDs  []string
	DepartmentID string
	From, To     string
}

// TaskDay is the occurrences of one day, department and assignee.
type TaskDay struct {
	Day, DepartmentID, EmployeeID                 string
	Pending, InProgress, Done, Approved, Rejected int
	OnTime, Reviewed                              int
	TurnaroundHours                               float64
	TurnLt4h, Turn4to24h, Turn1to3d, TurnOver3d   int
}

// Due counts every occurrence of the row.
func (t TaskDay) Due() int { return t.Pending + t.InProgress + t.Done + t.Approved + t.Rejected }

// Open counts the occurrences still owed.
func (t TaskDay) Open() int { return t.Pending + t.InProgress + t.Rejected }

// LogbookDay is the entries of one day and employee.
type LogbookDay struct {
	Day, DepartmentID, EmployeeID        string
	Draft, Submitted, Reviewed, Rejected int
	ScoreSum                             float64
}

// Sent counts entries that were submitted (reviewed and rejected ones too).
func (l LogbookDay) Sent() int { return l.Submitted + l.Reviewed + l.Rejected }

// RecapDay counts an employee's daily recaps of a day.
type RecapDay struct {
	Day, EmployeeID string
	Recaps          int
}

// DashScorecard is one scorecard as the dashboard reads it.
type DashScorecard struct {
	EmployeeID   string
	DepartmentID *string
	Period       Period
	Score        *float64
	Status       string
}

// IndicatorAverage is an indicator's average over the window's snapshots.
type IndicatorAverage struct {
	IndicatorID, Code, Name, Unit, Direction string
	Actual, Target, Attainment               *float64
	Samples                                  int
}

// DashImprovement is one improvement item as the dashboard reads it.
type DashImprovement struct {
	ID, Scope, Source, Status, Title          string
	EmployeeID, DepartmentID, OwnerEmployeeID *string
	DueDate                                   *string
	CreatedAt, UpdatedAt                      time.Time
	CompletedAt                               *time.Time
}

// OpenAt reports an item that was open or in progress at the end of day
// (WIB): created by then, not yet completed, and not dropped or declined
// before then (their last update stands in for that moment).
func (m DashImprovement) OpenAt(day string) bool {
	if DayKey(m.CreatedAt, Jakarta) > day {
		return false
	}
	switch m.Status {
	case ImpSuggested:
		return false
	case ImpDone:
		return m.CompletedAt != nil && DayKey(*m.CompletedAt, Jakarta) > day
	case ImpDropped, ImpDeclined:
		return DayKey(m.UpdatedAt, Jakarta) > day
	}
	return true
}

// FeedbackAssignmentCount counts a cycle's assignments by relationship
// and status.
type FeedbackAssignmentCount struct {
	Relationship, Status string
	Total                int
}

// DashboardStore reads the dashboard aggregates.
type DashboardStore interface {
	DashTaskDays(ctx context.Context, f DashFilter) ([]TaskDay, error)
	DashLogbookDays(ctx context.Context, f DashFilter) ([]LogbookDay, error)
	DashRecapDays(ctx context.Context, f DashFilter) ([]RecapDay, error)
	DashScorecards(ctx context.Context, f DashFilter) ([]DashScorecard, error)
	DashIndicatorAverages(ctx context.Context, f DashFilter) ([]IndicatorAverage, error)
	DashImprovements(ctx context.Context, f DashFilter) ([]DashImprovement, error)
	DashFeedbackAssignments(ctx context.Context, cycleID string, f DashFilter) ([]FeedbackAssignmentCount, error)
}
