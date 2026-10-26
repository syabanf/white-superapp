package domain

import (
	"sort"
	"strconv"
	"time"
)

// Daily and monthly work recaps (new for WHITE): what an employee did,
// built from task occurrences, their sub-tasks and history, logbook entries
// and an optional free-text note, summarised by OpenRouter or the mock.

// Daily recap time (WIB): each workday at 17:30.
const (
	DailyRecapHour   = 17
	DailyRecapMinute = 30
)

// RecapTask is one task occurrence touched that day.
type RecapTask struct {
	OccurrenceID string         `json:"occurrenceId"`
	Title        string         `json:"title"`
	Status       string         `json:"status"`
	Progress     int            `json:"progress"`
	Completed    bool           `json:"completed"`
	Priority     string         `json:"priority"`
	Subtasks     []RecapSubtask `json:"subtasks"`
	Notes        []string       `json:"notes"`
}

// RecapSubtask is a sub-task tick of that occurrence.
type RecapSubtask struct {
	Title   string `json:"title"`
	Checked bool   `json:"checked"`
}

// RecapLogbook is a logbook entry of that day.
type RecapLogbook struct {
	EntryID    string  `json:"entryId"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Completion float64 `json:"completion"`
	KPIScore   float64 `json:"kpiScore"`
}

// DailyRecapItems is daily_recaps.items.
type DailyRecapItems struct {
	Tasks   []RecapTask    `json:"tasks"`
	Logbook []RecapLogbook `json:"logbook"`
}

// CompletedCount counts the occurrences finished that day.
func (d DailyRecapItems) CompletedCount() int {
	n := 0
	for _, t := range d.Tasks {
		if t.Completed {
			n++
		}
	}
	return n
}

// Empty reports a day with nothing recorded.
func (d DailyRecapItems) Empty() bool { return len(d.Tasks) == 0 && len(d.Logbook) == 0 }

// DailyRecapInput is what the summariser receives.
type DailyRecapInput struct {
	EmployeeName string
	Date         string
	Items        DailyRecapItems
	Notes        string
}

// AttendanceSummary is the month's attendance in a monthly recap.
type AttendanceSummary struct {
	WorkingDays int     `json:"workingDays"`
	PresentDays int     `json:"presentDays"`
	LateDays    int     `json:"lateDays"`
	AbsentDays  int     `json:"absentDays"`
	LeaveDays   int     `json:"leaveDays"`
	OvertimeHrs float64 `json:"overtimeHours"`
}

// MonthlyMetrics is monthly_recaps.metrics.
type MonthlyMetrics struct {
	TasksPlanned      int                `json:"tasksPlanned"`
	TasksDone         int                `json:"tasksDone"`
	TasksOnTime       int                `json:"tasksOnTime"`
	TasksOverdue      int                `json:"tasksOverdue"`
	CompletionRate    *float64           `json:"completionRate"`
	OnTimeRate        *float64           `json:"onTimeRate"`
	KPIScore          *float64           `json:"kpiScore"`
	LogbookEntries    int                `json:"logbookEntries"`
	LogbookCompliance *float64           `json:"logbookCompliance"`
	Attendance        *AttendanceSummary `json:"attendance"`
	Members           int                `json:"members,omitempty"`
	ImprovementsOpen  int                `json:"improvementsOpen"`
}

// MonthTask is one occurrence of the month as the monthly recap reads it.
type MonthTask struct {
	Title          string
	Date           string
	Status         string
	Priority       string
	DoneAt         *time.Time
	KPIIndicator   string
	KPIContributed float64
}

// MonthlyTaskStats counts planned, done (approved), on-time (finished on or
// before its day, WIB) and overdue occurrences, up to today.
func MonthlyTaskStats(tasks []MonthTask, today string) (planned, done, onTime, overdue int) {
	for _, t := range tasks {
		if t.Date > today {
			continue
		}
		planned++
		if t.Status == StatusApproved {
			done++
			if t.DoneAt != nil && DayKey(*t.DoneAt, Jakarta) <= t.Date {
				onTime++
			}
		}
		if OccurrenceOverdue(t.Date, t.Status, today) {
			overdue++
		}
	}
	return
}

// Ratio is num/den rounded to four decimals, nil without a denominator.
func Ratio(num, den int) *float64 {
	if den <= 0 {
		return nil
	}
	v := Round(float64(num)/float64(den)*10000) / 10000
	return &v
}

// TopAccomplishments lists up to n approved tasks: KPI-linked and urgent
// first, distinct titles (a daily task counts once, with its count).
func TopAccomplishments(tasks []MonthTask, n int) []string {
	type agg struct {
		title    string
		count    int
		rank     int
		kpi      bool
		contrib  float64
		lastDate string
	}
	byTitle := map[string]*agg{}
	var order []string
	for _, t := range tasks {
		if t.Status != StatusApproved {
			continue
		}
		a := byTitle[t.Title]
		if a == nil {
			a = &agg{title: t.Title, rank: PriorityRank(t.Priority)}
			byTitle[t.Title] = a
			order = append(order, t.Title)
		}
		a.count++
		a.kpi = a.kpi || t.KPIIndicator != ""
		a.contrib += t.KPIContributed
		a.rank = min(a.rank, PriorityRank(t.Priority))
		if t.Date > a.lastDate {
			a.lastDate = t.Date
		}
	}
	list := make([]*agg, 0, len(order))
	for _, k := range order {
		list = append(list, byTitle[k])
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.kpi != b.kpi {
			return a.kpi
		}
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		return a.count > b.count
	})
	out := []string{}
	for _, a := range list {
		if len(out) == n {
			break
		}
		line := a.title
		if a.count > 1 {
			line += " (" + strconv.Itoa(a.count) + "x)"
		}
		out = append(out, line)
	}
	return out
}

// MonthlyRecapInput is what the narrative writer and the improvement
// suggester receive.
type MonthlyRecapInput struct {
	Scope           string
	SubjectName     string
	DepartmentName  string
	Period          Period
	Metrics         MonthlyMetrics
	Accomplishments []string
	// Notes the reader wrote in daily recaps and review notes of rejected
	// work, newest first; context for the suggestions.
	Signals []string
}

// ImprovementSuggestion is one AI-proposed improvement.
type ImprovementSuggestion struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// MaxSuggestions caps the AI pass over a monthly recap.
const MaxSuggestions = 5
