package domain

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

// Department tasks (NüHabit payroll/domain/depttasks.go and
// depttask_views.go): recurring task lists per department whose
// occurrences are generated per day, the occurrence state machine,
// effective progress, view ranges, the monitor and the reminder texts.

// Occurrence statuses: pending -> in_progress -> done (waiting review) ->
// approved | rejected. Rejected goes back to work.
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
	StatusApproved   = "approved"
	StatusRejected   = "rejected"
)

// Occurrence actions.
const (
	ActStart        = "start"
	ActProgress     = "progress"
	ActCheckSubtask = "check_subtask"
	ActComment      = "comment"
	ActDone         = "done"
	ActApprove      = "approve"
	ActReject       = "reject"
	ActReopen       = "reopen"
)

// Update kinds of the occurrence history.
const (
	UpdateProgress   = "progress"
	UpdateComment    = "comment"
	UpdateStatus     = "status"
	UpdateReview     = "review"
	UpdateAttachment = "attachment"
)

var TaskPriorities = []string{"low", "normal", "high", "urgent"}

func ValidPriority(s string) bool { return slices.Contains(TaskPriorities, s) }

// PriorityRank orders urgent first.
func PriorityRank(p string) int {
	switch p {
	case "urgent":
		return 0
	case "high":
		return 1
	case "normal":
		return 2
	}
	return 3
}

// OccurrenceOpen reports whether work is still owed.
func OccurrenceOpen(status string) bool {
	return status == StatusPending || status == StatusInProgress || status == StatusRejected
}

// OccurrenceOverdue is isOverdue: due before today and still open.
func OccurrenceOverdue(date, status, today string) bool {
	return date < today && OccurrenceOpen(status)
}

// EffectiveProgress is effectiveProgress: done or approved 100, pending 0,
// otherwise the weighted sub-task share when the task has sub-tasks, else
// the percent the assignee entered, capped at 99.
func EffectiveProgress(status string, manualPercent, subtaskPercent *float64) int {
	switch status {
	case StatusDone, StatusApproved:
		return 100
	case StatusPending:
		return 0
	}
	raw := 0.0
	if subtaskPercent != nil {
		raw = *subtaskPercent
	} else if manualPercent != nil {
		raw = *manualPercent
	}
	return max(0, min(99, int(Round(raw))))
}

// TaskSchedule is the recurrence part of a task.
type TaskSchedule struct {
	Recurrence string
	WeeklyDay  *int
	MonthlyDay *int
	DueDate    *string
	// CreatedDate is the day the task was filed in its zone: a recurring
	// task is never scheduled before it.
	CreatedDate *string
}

// OccurrenceDates is occurrenceDatesFor: the task's dates in [start, end].
func OccurrenceDates(t TaskSchedule, start, end string) []string {
	out := []string{}
	if t.Recurrence != "once" && t.CreatedDate != nil && *t.CreatedDate > start {
		start = *t.CreatedDate
	}
	s, ok1 := ParseDay(start)
	e, ok2 := ParseDay(end)
	if !ok1 || !ok2 || s.After(e) {
		return out
	}
	if t.Recurrence == "once" {
		if t.DueDate != nil && *t.DueDate != "" && *t.DueDate >= start && *t.DueDate <= end {
			out = append(out, *t.DueDate)
		}
		return out
	}
	for d := s; !d.After(e); d = d.AddDate(0, 0, 1) {
		dow := int(d.Weekday())
		if dow == 0 {
			dow = 7
		}
		switch {
		case t.Recurrence == "daily",
			t.Recurrence == "weekly" && t.WeeklyDay != nil && *t.WeeklyDay == dow,
			t.Recurrence == "monthly" && t.MonthlyDay != nil && *t.MonthlyDay == d.Day():
			out = append(out, d.Format(IsoDate))
		}
	}
	return out
}

// Subtask is a weighted sub-task title.
type Subtask struct {
	Title  string
	Weight float64
}

// SplitSubtaskWeights shares 100% evenly; the rounding rest goes to the last.
func SplitSubtaskWeights(titles []string) []Subtask {
	out := []Subtask{}
	if len(titles) == 0 {
		return out
	}
	n := float64(len(titles))
	even := math.Floor(100/n*100) / 100
	for i, t := range titles {
		w := even
		if i == len(titles)-1 {
			w = Round2(100 - even*(n-1))
		}
		out = append(out, Subtask{t, w})
	}
	return out
}

// TaskBody is a create payload.
type TaskBody struct {
	Title, Description, Recurrence, DueDate string
	WeeklyDay, MonthlyDay                   int
	Subtasks                                []string
}

// NormalizedTask is a validated new task.
type NormalizedTask struct {
	Title       string
	Description *string
	Schedule    TaskSchedule
	Subtasks    []Subtask
}

var isoDateRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// MaxSubtasks caps the sub-tasks of one task.
const MaxSubtasks = 50

// NormalizeTask is normalizeDeptTask: the task or a message.
func NormalizeTask(b TaskBody) (*NormalizedTask, string) {
	title := strings.TrimSpace(b.Title)
	if len([]rune(title)) < 3 {
		return nil, "Judul task minimal 3 karakter"
	}
	recurrence := b.Recurrence
	if recurrence == "" {
		recurrence = "once"
	}
	if !slices.Contains([]string{"once", "daily", "weekly", "monthly"}, recurrence) {
		return nil, "Jenis pengulangan tidak dikenal"
	}
	if recurrence == "once" {
		if _, ok := ParseDay(b.DueDate); !ok || !isoDateRE.MatchString(b.DueDate) {
			return nil, "Task sekali jalan membutuhkan tanggal jatuh tempo"
		}
	}
	if recurrence == "weekly" && (b.WeeklyDay < 1 || b.WeeklyDay > 7) {
		return nil, "Task mingguan membutuhkan hari (Senin–Minggu)"
	}
	if recurrence == "monthly" && (b.MonthlyDay < 1 || b.MonthlyDay > 28) {
		return nil, "Task bulanan membutuhkan tanggal 1–28"
	}
	var titles []string
	for _, s := range b.Subtasks {
		if t := strings.TrimSpace(s); t != "" {
			titles = append(titles, t)
		}
	}
	if len(titles) > MaxSubtasks {
		return nil, fmt.Sprintf("Maksimal %d sub-task", MaxSubtasks)
	}
	n := &NormalizedTask{Title: title, Schedule: TaskSchedule{Recurrence: recurrence}, Subtasks: SplitSubtaskWeights(titles)}
	if d := strings.TrimSpace(b.Description); d != "" {
		n.Description = &d
	}
	switch recurrence {
	case "weekly":
		w := b.WeeklyDay
		n.Schedule.WeeklyDay = &w
	case "monthly":
		m := b.MonthlyDay
		n.Schedule.MonthlyDay = &m
	case "once":
		d := b.DueDate
		n.Schedule.DueDate = &d
	}
	return n, ""
}

// ViewRange is the day span of a list view at now in loc: today, this ISO
// week (Monday to Sunday) or this calendar month. An unknown view is today.
func ViewRange(view string, now time.Time, loc *time.Location) DaySpan {
	if loc == nil {
		loc = Jakarta
	}
	l := now.In(loc)
	today := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
	switch view {
	case "week":
		offset := (int(today.Weekday()) + 6) % 7
		start := today.AddDate(0, 0, -offset)
		return DaySpan{start.Format(IsoDate), start.AddDate(0, 0, 6).Format(IsoDate)}
	case "month":
		start := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		return DaySpan{start.Format(IsoDate), start.AddDate(0, 1, -1).Format(IsoDate)}
	default:
		return DaySpan{today.Format(IsoDate), today.Format(IsoDate)}
	}
}

// Reminder hours (WIB): morning digest and afternoon overdue notice.
const (
	DigestHour  = 7
	OverdueHour = 16
)

// MonitorRow is one occurrence as the monitor counts it.
type MonitorRow struct {
	OccurrenceID   string
	TaskID         string
	Title          string
	DepartmentID   string
	DepartmentName string
	AssigneeID     *string
	AssigneeName   *string
	Date           string
	Status         string
	Progress       int
	LastNote       *string
}

// MonitorSummary counts the rows due up to today: completion is approved
// over due (the task_completion KPI).
type MonitorSummary struct {
	Scheduled     int
	Due           int
	Approved      int
	WaitingReview int
	InProgress    int
	NotStarted    int
	Rejected      int
	Overdue       int
	Completion    *float64
	AvgProgress   *int
}

// Summarize is summarize.
func Summarize(rows []MonitorRow, today string) MonitorSummary {
	s := MonitorSummary{Scheduled: len(rows)}
	sum := 0
	for _, r := range rows {
		if OccurrenceOverdue(r.Date, r.Status, today) {
			s.Overdue++
		}
		if r.Date > today {
			continue
		}
		s.Due++
		sum += r.Progress
		switch r.Status {
		case StatusApproved:
			s.Approved++
		case StatusDone:
			s.WaitingReview++
		case StatusInProgress:
			s.InProgress++
		case StatusPending:
			s.NotStarted++
		case StatusRejected:
			s.Rejected++
		}
	}
	if s.Due > 0 {
		c := float64(s.Approved) / float64(s.Due)
		avg := int(Round(float64(sum) / float64(s.Due)))
		s.Completion, s.AvgProgress = &c, &avg
	}
	return s
}

// AssigneeSummary is a MonitorSummary per person responsible.
type AssigneeSummary struct {
	AssigneeID   *string
	AssigneeName string
	MonitorSummary
}

// DepartmentSummary is a MonitorSummary per department with its assignees.
type DepartmentSummary struct {
	DepartmentID   string
	DepartmentName string
	MonitorSummary
	Assignees []AssigneeSummary
}

// GroupByDepartment is groupByDepartment: a summary per department and
// per assignee, the most overdue first.
func GroupByDepartment(rows []MonitorRow, today string) []DepartmentSummary {
	byDept := map[string][]MonitorRow{}
	var order []string
	for _, r := range rows {
		if _, ok := byDept[r.DepartmentID]; !ok {
			order = append(order, r.DepartmentID)
		}
		byDept[r.DepartmentID] = append(byDept[r.DepartmentID], r)
	}
	out := []DepartmentSummary{}
	for _, id := range order {
		deptRows := byDept[id]
		byAssignee := map[string][]MonitorRow{}
		var keys []string
		for _, r := range deptRows {
			key := ""
			if r.AssigneeID != nil {
				key = *r.AssigneeID
			}
			if _, ok := byAssignee[key]; !ok {
				keys = append(keys, key)
			}
			byAssignee[key] = append(byAssignee[key], r)
		}
		assignees := []AssigneeSummary{}
		for _, key := range keys {
			items := byAssignee[key]
			a := AssigneeSummary{AssigneeName: "Departemen (tanpa PJ)", MonitorSummary: Summarize(items, today)}
			if key != "" {
				k := key
				a.AssigneeID = &k
				a.AssigneeName = "-"
				if items[0].AssigneeName != nil {
					a.AssigneeName = *items[0].AssigneeName
				}
			}
			assignees = append(assignees, a)
		}
		sort.SliceStable(assignees, func(i, j int) bool {
			if assignees[i].Overdue != assignees[j].Overdue {
				return assignees[i].Overdue > assignees[j].Overdue
			}
			return assignees[i].AssigneeName < assignees[j].AssigneeName
		})
		out = append(out, DepartmentSummary{DepartmentID: id, DepartmentName: deptRows[0].DepartmentName,
			MonitorSummary: Summarize(deptRows, today), Assignees: assignees})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Overdue != out[j].Overdue {
			return out[i].Overdue > out[j].Overdue
		}
		return out[i].DepartmentName < out[j].DepartmentName
	})
	return out
}

// OverdueList is overdueList: the overdue rows, oldest first, at most limit.
func OverdueList(rows []MonitorRow, today string, limit int) []MonitorRow {
	out := []MonitorRow{}
	for _, r := range rows {
		if OccurrenceOverdue(r.Date, r.Status, today) {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		return out[i].Title < out[j].Title
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// WaitingReview lists the rows waiting for a reviewer, oldest first.
func WaitingReview(rows []MonitorRow, limit int) []MonitorRow {
	out := []MonitorRow{}
	for _, r := range rows {
		if r.Status == StatusDone {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// DigestLines caps the lines listed in a reminder.
const DigestLines = 5

// ComposeTaskNotice writes the morning digest or the afternoon overdue
// text; titles arrive overdue first, so the first `overdue` are marked.
func ComposeTaskNotice(titles []string, overdue int, overdueOnly bool) (title, body string) {
	var lines []string
	for i, t := range titles {
		if i == DigestLines {
			lines = append(lines, fmt.Sprintf("... dan %d tugas lagi", len(titles)-DigestLines))
			break
		}
		mark := ""
		if i < overdue && !overdueOnly {
			mark = " (terlambat)"
		}
		lines = append(lines, "• "+t+mark)
	}
	if overdueOnly {
		return fmt.Sprintf("%d tugas terlambat", len(titles)), "Masih terbuka dan lewat jatuh tempo:\n" + strings.Join(lines, "\n")
	}
	title = fmt.Sprintf("Tugas hari ini: %d", len(titles))
	if overdue > 0 {
		title += fmt.Sprintf(" (%d terlambat)", overdue)
	}
	return title, strings.Join(lines, "\n")
}
