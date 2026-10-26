package domain

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Task, review, logbook and feedback rules. The department task and review
// cases are ported from NüHabit (hris_test.go, depttask_views_test.go).

func TestDeptTasks(t *testing.T) {
	ptrI := func(i int) *int { return &i }
	join := func(s []string) string { return strings.Join(s, ",") }
	if got := OccurrenceDates(TaskSchedule{Recurrence: "daily"}, "2026-09-01", "2026-09-05"); len(got) != 5 {
		t.Fatal(got)
	}
	if got := OccurrenceDates(TaskSchedule{Recurrence: "weekly", WeeklyDay: ptrI(1)}, "2026-09-01", "2026-09-30"); join(got) != "2026-09-07,2026-09-14,2026-09-21,2026-09-28" {
		t.Fatal(got)
	}
	if got := OccurrenceDates(TaskSchedule{Recurrence: "monthly", MonthlyDay: ptrI(15)}, "2026-09-01", "2026-10-31"); join(got) != "2026-09-15,2026-10-15" {
		t.Fatal(got)
	}
	due := "2026-09-10"
	once := TaskSchedule{Recurrence: "once", DueDate: &due}
	if join(OccurrenceDates(once, "2026-09-01", "2026-09-30")) != due || len(OccurrenceDates(once, "2026-10-01", "2026-10-31")) != 0 {
		t.Fatal("once")
	}
	if len(OccurrenceDates(TaskSchedule{Recurrence: "daily"}, "2026-09-05", "2026-09-01")) != 0 {
		t.Fatal("reversed")
	}
	filed := "2026-09-04"
	if got := OccurrenceDates(TaskSchedule{Recurrence: "daily", CreatedDate: &filed}, "2026-09-01", "2026-09-05"); join(got) != "2026-09-04,2026-09-05" {
		t.Fatal(got)
	}
	early := "2026-09-02"
	if got := OccurrenceDates(TaskSchedule{Recurrence: "once", DueDate: &early, CreatedDate: &filed}, "2026-09-01", "2026-09-30"); join(got) != early {
		t.Fatal(got)
	}
	w := SplitSubtaskWeights([]string{"a", "b", "c"})
	if w[0].Weight != 33.33 || w[2].Weight != 33.34 || len(SplitSubtaskWeights(nil)) != 0 {
		t.Fatal(w)
	}
	for body, msg := range map[*TaskBody]string{
		{Title: "ab"}: "Judul task minimal 3 karakter",
		{Title: "Cek stok", Recurrence: "tahunan"}:                 "Jenis pengulangan tidak dikenal",
		{Title: "Cek stok"}:                                        "Task sekali jalan membutuhkan tanggal jatuh tempo",
		{Title: "Cek stok", Recurrence: "weekly", WeeklyDay: 8}:    "Task mingguan membutuhkan hari (Senin–Minggu)",
		{Title: "Cek stok", Recurrence: "monthly", MonthlyDay: 31}: "Task bulanan membutuhkan tanggal 1–28",
	} {
		if _, got := NormalizeTask(*body); got != msg {
			t.Errorf("%v = %q, want %q", body.Title, got, msg)
		}
	}
	task, msg := NormalizeTask(TaskBody{Title: " Bersih gudang ", Recurrence: "weekly", WeeklyDay: 3, MonthlyDay: 5, DueDate: "2026-10-10",
		Subtasks: []string{"Sapu", " ", "Pel"}})
	if msg != "" || task.Title != "Bersih gudang" || *task.Schedule.WeeklyDay != 3 || task.Schedule.MonthlyDay != nil ||
		task.Schedule.DueDate != nil || len(task.Subtasks) != 2 || task.Subtasks[1].Weight != 50 {
		t.Fatalf("%q %+v", msg, task)
	}
}

func TestEffectiveProgress(t *testing.T) {
	cases := []struct {
		status          string
		manual, subtask *float64
		want            int
	}{
		{StatusApproved, nil, nil, 100},
		{StatusDone, fp(10), nil, 100},
		{StatusPending, fp(50), fp(50), 0},
		{StatusInProgress, fp(40), nil, 40},
		{StatusInProgress, fp(40), fp(66.7), 67},
		{StatusRejected, nil, fp(100), 99},
		{StatusInProgress, nil, nil, 0},
	}
	for _, c := range cases {
		if got := EffectiveProgress(c.status, c.manual, c.subtask); got != c.want {
			t.Errorf("%s: %d, want %d", c.status, got, c.want)
		}
	}
	today := "2026-10-22"
	if !OccurrenceOverdue("2026-10-21", StatusRejected, today) || OccurrenceOverdue("2026-10-21", StatusDone, today) || OccurrenceOverdue(today, StatusPending, today) {
		t.Fatal("overdue")
	}
}

func at(s string, loc *time.Location) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, loc)
	if err != nil {
		panic(err)
	}
	return t
}

func TestViewRangeAndReminders(t *testing.T) {
	wit := Zone("Asia/Jayapura")
	now := at("2026-10-28 14:00", Jakarta) // Wednesday
	check := func(view string, loc *time.Location, start, end string) {
		t.Helper()
		if r := ViewRange(view, now, loc); r.Start != start || r.End != end {
			t.Errorf("%s: [%s, %s], want [%s, %s]", view, r.Start, r.End, start, end)
		}
	}
	check("today", Jakarta, "2026-10-28", "2026-10-28")
	check("week", Jakarta, "2026-10-26", "2026-11-01")
	check("month", Jakarta, "2026-10-01", "2026-10-31")
	check("bogus", Jakarta, "2026-10-28", "2026-10-28")
	now = at("2026-10-31 23:30", Jakarta)
	check("today", wit, "2026-11-01", "2026-11-01")
	check("month", wit, "2026-11-01", "2026-11-30")
	if ReminderDue(at("2026-10-27 06:59", Jakarta), DigestHour, 0, nil) || !ReminderDue(at("2026-10-27 07:00", Jakarta), DigestHour, 0, nil) {
		t.Fatal("digest hour")
	}
	if ReminderDue(at("2026-10-27 17:29", Jakarta), DailyRecapHour, DailyRecapMinute, Jakarta) ||
		!ReminderDue(at("2026-10-27 17:30", Jakarta), DailyRecapHour, DailyRecapMinute, Jakarta) ||
		!ReminderDue(at("2026-10-27 18:05", Jakarta), DailyRecapHour, DailyRecapMinute, Jakarta) {
		t.Fatal("recap time")
	}
	if p, ok := ParsePeriod("2028-02"); !ok || p.Span().End != "2028-02-29" || p.Previous().Key() != "2028-01" {
		t.Fatal("period")
	}
	if (Period{2026, 1}).Previous().Key() != "2025-12" || (Period{2026, 10}).Label() != "Oktober 2026" {
		t.Fatal("previous/label")
	}
}

func TestMonitorSummaries(t *testing.T) {
	today := "2026-10-22"
	pras, rara := "a1", "a2"
	row := func(dept, name, date, status string, progress int, assignee *string, assigneeName string) MonitorRow {
		r := MonitorRow{TaskID: "t1", Title: "Cek alat", DepartmentID: dept, DepartmentName: name, Date: date, Status: status, Progress: progress, AssigneeID: assignee}
		if assignee != nil {
			r.AssigneeName = &assigneeName
		}
		return r
	}
	rows := []MonitorRow{
		row("d1", "Creative", "2026-10-20", StatusApproved, 100, &pras, "Pras"),
		row("d1", "Creative", "2026-10-21", StatusInProgress, 50, &pras, "Pras"),
		row("d1", "Creative", today, StatusDone, 100, &pras, "Pras"),
		row("d1", "Creative", "2026-10-25", StatusPending, 0, &pras, "Pras"),
		row("d2", "SEO", "2026-10-18", StatusPending, 0, nil, ""),
		row("d2", "SEO", "2026-10-19", StatusApproved, 100, &rara, "Rara"),
	}
	s := Summarize(rows[:4], today)
	if s.Scheduled != 4 || s.Due != 3 || s.Approved != 1 || s.WaitingReview != 1 || s.InProgress != 1 || s.Overdue != 1 ||
		*s.Completion < 0.33 || *s.Completion > 0.34 || *s.AvgProgress != 83 {
		t.Fatalf("summary %+v", s)
	}
	g := GroupByDepartment(rows, today)
	if len(g) != 2 || g[0].DepartmentName != "Creative" || g[1].Assignees[0].AssigneeName != "Departemen (tanpa PJ)" {
		t.Fatalf("departments %+v", g)
	}
	if o := OverdueList(rows, today, 50); len(o) != 2 || o[0].Date != "2026-10-18" {
		t.Fatalf("overdue %+v", o)
	}
	if w := WaitingReview(rows, 50); len(w) != 1 || w[0].Date != today {
		t.Fatalf("waiting %+v", w)
	}
}

func TestComposeTaskNotice(t *testing.T) {
	title, body := ComposeTaskNotice([]string{"A", "B", "C", "D", "E", "F", "G"}, 2, false)
	if title != "Tugas hari ini: 7 (2 terlambat)" || !strings.Contains(body, "• A (terlambat)") || !strings.Contains(body, "... dan 2 tugas lagi") ||
		strings.Contains(body, "C (terlambat)") {
		t.Fatalf("%q %q", title, body)
	}
	title, body = ComposeTaskNotice([]string{"A"}, 1, true)
	if title != "1 tugas terlambat" || !strings.HasPrefix(body, "Masih terbuka dan lewat jatuh tempo:") || strings.Contains(body, "(terlambat)") {
		t.Fatalf("%q %q", title, body)
	}
}

func TestPerformanceReview(t *testing.T) {
	if !reflect.DeepEqual(QuarterMonths(3), []int{7, 8, 9}) {
		t.Fatal("months")
	}
	if s, e := QuarterRange(2026, 2); s != "2026-04-01" || e != "2026-06-30" {
		t.Fatal(s, e)
	}
	if _, e := QuarterRange(2028, 1); e != "2028-03-31" || QuarterLabel(2026, 3) != "Q3 2026" || QuarterOf(10) != 4 {
		t.Fatal(e)
	}
	if *CombineReviewScores(fp(80), fp(90), fp(100)) != 85 || *CombineReviewScores(fp(90), fp(60), nil) != 80 ||
		*CombineReviewScores(fp(77.5), nil, nil) != 77.5 || CombineReviewScores(nil, nil, nil) != nil {
		t.Fatal("combine")
	}
	i := func(v int) *int { return &v }
	if *BehaviorScore([]BehaviorItem{{i(5), 20}, {i(4), 20}, {i(3), 20}, {i(4), 20}, {i(4), 20}}) != 80 ||
		*BehaviorScore([]BehaviorItem{{i(5), 20}, {nil, 20}}) != 100 || BehaviorScore(nil) != nil {
		t.Fatal("behavior")
	}
	if _, msg := ProjectScore(fp(101)); msg == "" {
		t.Fatal("project range")
	}
	if n, msg := BehaviorRating(4); n != 4 || msg != "" {
		t.Fatal("rating")
	}
	if _, msg := BehaviorRating(4.5); msg != "Skor harus 1–5" {
		t.Fatal("fraction")
	}
	for score, grade := range map[float64]string{95: "A", 90: "A", 85: "B", 70: "C", 60: "D", 59.9: "E"} {
		if FinalGrade(score) != grade {
			t.Errorf("%v", score)
		}
	}
	if s, g := FinalScore(80, 100, fp(50), fp(50)); s != 90 || g != "A" {
		t.Fatal(s, g)
	}
	if s, g := FinalScore(80, 60, nil, nil); math.Abs(s-74) > 1e-9 || g != "C" {
		t.Fatal(s, g)
	}
	cats := []Category{{"Istimewa", 90, 100}, {"Baik", 75, 89.99}, {"Cukup", 60, 74.99}, {"Perlu Pembinaan", 0, 59.99}}
	if CategoryFor(cats, fp(91)) != "Istimewa" || CategoryFor(cats, fp(75)) != "Baik" || CategoryFor(cats, nil) != "" || CategoryFor(cats, fp(74.995)) != "" {
		t.Fatal("category")
	}
}

func TestLogbook(t *testing.T) {
	c, k := LogbookScore([]LogbookItemState{{true, 2}, {false, 1}, {true, 1}})
	if c != 66.67 || k != 75 {
		t.Fatal(c, k)
	}
	if c, k := LogbookScore(nil); c != 0 || k != 0 {
		t.Fatal("empty")
	}
	sum := SummarizeLogbookEntries([]LogbookSummaryEntry{
		{"d1", LogbookSubmitted, 50, 40}, {"d2", LogbookReviewed, 100, 100}, {"d1", LogbookRejected, 100, 90},
	})
	if len(sum) != 2 || sum[0].DepartmentID != "d1" || sum[0].TotalEntries != 2 || sum[0].AvgCompletion != 75 || sum[0].RejectedEntries != 1 {
		t.Fatalf("%+v", sum)
	}
	if TemplateItemWeight(nil) != 1 || TemplateItemWeight(fp(-2)) != 0 || TemplateItemWeight(fp(0)) != 0 {
		t.Fatal("weight")
	}
	if n, ok := NormalizeLogbookNote(""); !ok || n != nil {
		t.Fatal("empty note")
	}
	if _, ok := NormalizeLogbookNote(strings.Repeat("a", LogbookNoteMaxLength+1)); ok {
		t.Fatal("long note")
	}
	if LogbookTitle("Harian", "2026-10-01", " ") != "Harian - 2026-10-01" || LogbookTitle("Harian", "2026-10-01", "Custom") != "Custom" {
		t.Fatal("title")
	}
}

func TestSummarizeFeedback(t *testing.T) {
	cats := []FeedbackCategoryWeight{{"communication", "Komunikasi", 20}, {"collaboration", "Kolaborasi", 20}, {"leadership", "Kepemimpinan", 20}}
	r := func(code, name, crit, rel string, rating int) FeedbackRating {
		return FeedbackRating{CategoryCode: code, CategoryName: name, CriteriaName: crit, Relationship: rel, Rating: rating}
	}
	s := SummarizeFeedback(cats, []FeedbackRating{
		r("communication", "Komunikasi", "Jelas", RelPeer, 2), r("communication", "Komunikasi", "Mendengarkan", RelPeer, 3),
		r("collaboration", "Kolaborasi", "Membantu", RelManager, 5), r("collaboration", "Kolaborasi", "Berbagi", RelPeer, 4),
		r("communication", "Komunikasi", "Jelas", RelSelf, 5), // ignored: others answered
	})
	if s.Categories["communication"] != 50 || s.Categories["collaboration"] != 90 || *s.Overall != 70 || s.Responses != 4 {
		t.Fatalf("%+v", s)
	}
	if len(s.Strengths) != 1 || s.Strengths[0] != "Kolaborasi: Membantu (rata-rata 5 dari 5)" {
		t.Fatalf("strengths %v", s.Strengths)
	}
	if len(s.Weaknesses) != 1 || s.Weaknesses[0] != "Komunikasi: Jelas (rata-rata 2 dari 5)" {
		t.Fatalf("weaknesses %v", s.Weaknesses)
	}
	// Only self answers: they count; nothing below 70 but under 80 -> weakest.
	only := SummarizeFeedback(cats, []FeedbackRating{r("leadership", "Kepemimpinan", "Arahan", RelSelf, 4)})
	if *only.Overall != 80 || len(only.Weaknesses) != 0 || len(only.Strengths) != 1 {
		t.Fatalf("%+v", only)
	}
	if empty := SummarizeFeedback(cats, nil); empty.Overall != nil {
		t.Fatal("empty")
	}
	if FormatDecimal(1234.5, 1) != "1.234,5" || FormatDecimal(4, 1) != "4" || FormatDecimal(-0.04, 1) != "0" || FormatDecimal(83.333, 2) != "83,33" {
		t.Fatal(FormatDecimal(1234.5, 1), FormatDecimal(-0.04, 1))
	}
}

func TestMonthlyRecapRules(t *testing.T) {
	tm := func(s string) *time.Time { v := at(s, Jakarta); return &v }
	tasks := []MonthTask{
		{Title: "Posting", Date: "2026-10-01", Status: StatusApproved, Priority: "high", DoneAt: tm("2026-10-01 15:00"), KPIIndicator: "posting_ontime"},
		{Title: "Posting", Date: "2026-10-02", Status: StatusApproved, Priority: "high", DoneAt: tm("2026-10-03 09:00"), KPIIndicator: "posting_ontime"},
		{Title: "Laporan", Date: "2026-10-03", Status: StatusApproved, Priority: "urgent", DoneAt: tm("2026-10-03 23:59")},
		{Title: "Audit", Date: "2026-10-04", Status: StatusPending, Priority: "normal"},
		{Title: "Besok", Date: "2026-10-30", Status: StatusPending, Priority: "normal"},
	}
	planned, done, onTime, overdue := MonthlyTaskStats(tasks, "2026-10-20")
	if planned != 4 || done != 3 || onTime != 2 || overdue != 1 {
		t.Fatal(planned, done, onTime, overdue)
	}
	if *Ratio(2, 3) != 0.6667 || Ratio(1, 0) != nil {
		t.Fatal("ratio")
	}
	if got := TopAccomplishments(tasks, 2); !reflect.DeepEqual(got, []string{"Posting (2x)", "Laporan"}) {
		t.Fatal(got)
	}
}

func TestImprovementRules(t *testing.T) {
	if !CanTransitionImprovement(ImpSuggested, ImpOpen) || !CanTransitionImprovement(ImpOpen, ImpDone) || CanTransitionImprovement(ImpDeclined, ImpOpen) ||
		CanTransitionImprovement(ImpSuggested, ImpDone) || !CanTransitionImprovement(ImpDone, ImpOpen) {
		t.Fatal("transitions")
	}
	if got := ImprovementTitle("Perbaiki: ", "Posting konten harian.\nDetail lain"); got != "Perbaiki: Posting konten harian" {
		t.Fatal(got)
	}
	long := ImprovementTitle("", strings.Repeat("kata ", 80))
	if len([]rune(long)) > ImprovementTitleMax || !strings.HasSuffix(long, "...") {
		t.Fatal(long)
	}
	if got := CleanList([]string{" a ", "", "b", "c"}, 2); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatal(got)
	}
}
