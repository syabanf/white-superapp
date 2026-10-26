package app

import (
	"context"
	"sort"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

// ─── Ringkasan ───────────────────────────────────────────────

// DashDepartmentRow is a department line of the summary table.
type DashDepartmentRow struct {
	DepartmentID, DepartmentName string
	Members                      int
	KPIScore, KPIScorePrevious   *float64
	TaskCompletion, OnTimeRate   *float64
	OverdueTasks                 int
	LogbookRate                  *float64
	OpenImprovements             int
}

type DashSummary struct {
	Window                                             DashWindow
	KPIScore, TaskCompletion, OnTimeRate, OverdueTasks DashMetric
	LogbookRate, OpenImprovements                      DashMetric
	Buckets, Months                                    []string
	DepartmentTrend                                    []DeptMonthScore
	Departments                                        []DashDepartmentRow
}

func (d *Dashboard) Summary(ctx context.Context, q DashboardQuery) (*DashSummary, error) {
	s, _, err := d.resolve(ctx, q)
	if err != nil {
		return nil, err
	}
	out := &DashSummary{Window: s.DashWindow, Buckets: domain.Buckets(s.From, s.upTo(), s.Granularity)}

	// Tasks
	cur, err := d.taskRows(ctx, s, s.From, s.To)
	if err != nil {
		return nil, err
	}
	prev, err := d.taskRows(ctx, s, s.CompareFrom, s.CompareTo)
	if err != nil {
		return nil, err
	}
	tc, tp := sumTasks(cur, s.Today), sumTasks(prev, s.Today)
	byBucket := tasksByBucket(cur, s.Granularity, s.Today)
	var sparkDone, sparkOnTime, sparkOverdue []float64
	for _, b := range out.Buckets {
		t := byBucket[b]
		if t == nil {
			t = &taskAgg{}
		}
		sparkDone = append(sparkDone, orZero(t.completion()))
		sparkOnTime = append(sparkOnTime, orZero(t.onTimeRate()))
		sparkOverdue = append(sparkOverdue, float64(t.overdue))
	}
	out.TaskCompletion = metric(tc.completion(), s.prev(tp.completion()), sparkDone)
	out.OnTimeRate = metric(tc.onTimeRate(), s.prev(tp.onTimeRate()), sparkOnTime)
	out.OverdueTasks = metric(count(tc.overdue), s.prev(count(tp.overdue)), sparkOverdue)

	// KPI
	m, err := d.monthly(ctx, s)
	if err != nil {
		return nil, err
	}
	cards, err := d.cards(ctx, s, s.From, s.To)
	if err != nil {
		return nil, err
	}
	prevCards, err := d.cards(ctx, s, s.CompareFrom, s.CompareTo)
	if err != nil {
		return nil, err
	}
	out.KPIScore = metric(avgScore(cards), s.prev(avgScore(prevCards)), m.all)
	out.Months, out.DepartmentTrend = m.months, m.dept

	// Logbook
	members, err := d.logbookMembers(ctx, s)
	if err != nil {
		return nil, err
	}
	lc, lrows, err := d.logbookWindow(ctx, s, members, s.From, s.To)
	if err != nil {
		return nil, err
	}
	lp, _, err := d.logbookWindow(ctx, s, members, s.CompareFrom, s.CompareTo)
	if err != nil {
		return nil, err
	}
	sentBy := map[string]int{}
	for _, r := range lrows {
		sentBy[domain.BucketOf(r.Day, s.Granularity)] += r.Sent()
	}
	var sparkLog []float64
	lastLog := s.logbookTo(s.To)
	for _, b := range out.Buckets {
		exp := total(members) * domain.Workdays(domain.MaxDay(b, s.From), bucketEnd(b, s.Granularity, lastLog))
		sparkLog = append(sparkLog, orZero(domain.Ratio(sentBy[b], exp)))
	}
	out.LogbookRate = metric(lc.rate(), s.prev(lp.rate()), sparkLog)

	// Improvements
	items, err := d.Store.DashImprovements(ctx, in(s.rows, s.From, s.To))
	if err != nil {
		return nil, err
	}
	var sparkImp []float64
	for _, b := range out.Buckets {
		sparkImp = append(sparkImp, float64(countOpen(items, bucketEnd(b, s.Granularity, s.To))))
	}
	var prevOpen *float64
	if s.compareOn() {
		prevOpen = count(countOpen(items, s.CompareTo))
	}
	out.OpenImprovements = metric(count(countOpen(items, s.To)), prevOpen, sparkImp)

	// Departments
	rows := map[string]*DashDepartmentRow{}
	order := []string{}
	row := func(id string) *DashDepartmentRow {
		if rows[id] == nil {
			rows[id] = &DashDepartmentRow{DepartmentID: id, DepartmentName: s.deptName(id)}
			order = append(order, id)
		}
		return rows[id]
	}
	for _, dep := range s.departments() {
		row(dep.ID)
	}
	for _, e := range s.active() {
		if id := s.deptOf(e); id != "" {
			row(id).Members++
		}
	}
	deptTasks := map[string]*taskAgg{}
	for _, r := range cur {
		if deptTasks[r.DepartmentID] == nil {
			deptTasks[r.DepartmentID] = &taskAgg{}
		}
		deptTasks[r.DepartmentID].add(r, s.Today)
	}
	for id, t := range deptTasks {
		dr := row(id)
		dr.TaskCompletion, dr.OnTimeRate, dr.OverdueTasks = t.completion(), t.onTimeRate(), t.overdue
	}
	byDeptCards := func(list []domain.DashScorecard) map[string][]domain.DashScorecard {
		out := map[string][]domain.DashScorecard{}
		for _, c := range list {
			if c.DepartmentID != nil {
				out[*c.DepartmentID] = append(out[*c.DepartmentID], c)
			}
		}
		return out
	}
	pc := byDeptCards(prevCards)
	for id, list := range byDeptCards(cards) {
		dr := row(id)
		dr.KPIScore, dr.KPIScorePrevious = avgScore(list), avgScore(pc[id])
	}
	sentDept := map[string]int{}
	for _, r := range lrows {
		sentDept[r.DepartmentID] += r.Sent()
	}
	days := 0
	if lastLog >= s.From {
		days = domain.Workdays(s.From, lastLog)
	}
	for id, n := range members {
		row(id).LogbookRate = domain.Ratio(sentDept[id], n*days)
	}
	for _, it := range items {
		if it.OpenAt(s.To) {
			if id := s.improvementDept(it); id != "" {
				row(id).OpenImprovements++
			}
		}
	}
	for _, id := range order {
		if id != "" && rows[id].DepartmentName != "" {
			out.Departments = append(out.Departments, *rows[id])
		}
	}
	return out, nil
}

// ─── KPI ─────────────────────────────────────────────────────

type DashIndicator struct {
	IndicatorID, Code, Name, Unit, Direction string
	Actual, Target, Attainment               *float64
	PreviousAttainment                       *float64
	Samples                                  int
}

type DashPersonScore struct {
	EmployeeID, Name, DepartmentName string
	DepartmentID                     *string
	Score, Previous                  *float64
	Months, FinalMonths              int
}

type DashKPI struct {
	Window                                       DashWindow
	AvgScore, FinalRate, Scorecards, BelowTarget DashMetric
	Histogram                                    []DashCount
	Months                                       []string
	DepartmentMonths                             []DeptMonthScore
	Indicators                                   []DashIndicator
	People                                       []DashPersonScore
	Funnel                                       []DashCount
}

func finalRate(cards []domain.DashScorecard) *float64 {
	n := 0
	for _, c := range cards {
		if c.Status == "final" {
			n++
		}
	}
	return domain.Ratio(n, len(cards))
}

func belowTarget(scores map[string]*float64) int {
	n := 0
	for _, v := range scores {
		if v != nil && *v < domain.BelowTargetScore {
			n++
		}
	}
	return n
}

func (d *Dashboard) KPI(ctx context.Context, q DashboardQuery) (*DashKPI, error) {
	s, _, err := d.resolve(ctx, q)
	if err != nil {
		return nil, err
	}
	cards, err := d.cards(ctx, s, s.From, s.To)
	if err != nil {
		return nil, err
	}
	prevCards, err := d.cards(ctx, s, s.CompareFrom, s.CompareTo)
	if err != nil {
		return nil, err
	}
	m, err := d.monthly(ctx, s)
	if err != nil {
		return nil, err
	}
	cur, prev := perEmployee(cards), perEmployee(prevCards)
	out := &DashKPI{Window: s.DashWindow, Months: m.months, DepartmentMonths: m.dept}
	var prevAvg, prevFinal, prevCount, prevBelow *float64
	if s.compareOn() {
		prevAvg, prevFinal, prevCount, prevBelow = avgScore(prevCards), finalRate(prevCards), count(len(prevCards)), count(belowTarget(prev))
	}
	out.AvgScore = metric(avgScore(cards), prevAvg, m.all)
	out.FinalRate = metric(finalRate(cards), prevFinal, nil)
	out.Scorecards = metric(count(len(cards)), prevCount, nil)
	out.BelowTarget = metric(count(belowTarget(cur)), prevBelow, nil)

	hist := make([]int, len(domain.ScoreBins))
	histPrev := make([]int, len(domain.ScoreBins))
	for _, v := range cur {
		if v != nil {
			hist[domain.ScoreBinIndex(*v)]++
		}
	}
	for _, v := range prev {
		if v != nil {
			histPrev[domain.ScoreBinIndex(*v)]++
		}
	}
	for i, b := range domain.ScoreBins {
		c := DashCount{Key: b.Key, Value: float64(hist[i])}
		if s.compareOn() {
			c.Previous = count(histPrev[i])
		}
		out.Histogram = append(out.Histogram, c)
	}

	inds, err := d.Store.DashIndicatorAverages(ctx, in(s.persons, s.From, s.To))
	if err != nil {
		return nil, err
	}
	prevAtt := map[string]*float64{}
	if s.compareOn() {
		pinds, err := d.Store.DashIndicatorAverages(ctx, in(s.persons, s.CompareFrom, s.CompareTo))
		if err != nil {
			return nil, err
		}
		for _, p := range pinds {
			prevAtt[p.IndicatorID] = p.Attainment
		}
	}
	for _, i := range inds {
		out.Indicators = append(out.Indicators, DashIndicator{IndicatorID: i.IndicatorID, Code: i.Code, Name: i.Name, Unit: i.Unit,
			Direction: i.Direction, Actual: i.Actual, Target: i.Target, Attainment: i.Attainment, PreviousAttainment: prevAtt[i.IndicatorID],
			Samples: i.Samples})
	}

	months, finals := map[string]int{}, map[string]int{}
	for _, c := range cards {
		months[c.EmployeeID]++
		if c.Status == "final" {
			finals[c.EmployeeID]++
		}
	}
	withCard, allFinal := 0, 0
	active := s.active()
	for _, e := range active {
		if months[e.ID] > 0 {
			withCard++
			if finals[e.ID] == months[e.ID] {
				allFinal++
			}
		}
	}
	out.Funnel = []DashCount{{Key: "members", Value: float64(len(active))}, {Key: "scorecard", Value: float64(withCard)},
		{Key: "final", Value: float64(allFinal)}}
	for id, n := range months {
		e, _ := s.ppl.employee(id)
		out.People = append(out.People, DashPersonScore{EmployeeID: id, Name: e.Name, DepartmentID: e.DepartmentID,
			DepartmentName: s.deptName(s.deptOf(e)), Score: cur[id], Previous: prev[id], Months: n, FinalMonths: finals[id]})
	}
	sort.Slice(out.People, func(i, j int) bool { return orZero(out.People[i].Score) > orZero(out.People[j].Score) })
	return out, nil
}

// ─── Tugas ───────────────────────────────────────────────────

type DashTaskBucket struct {
	Bucket                                   string
	Due, Approved, Finished, OnTime, Overdue int
	Completion, OnTimeRate                   *float64
	PreviousCompletion, PreviousOnTimeRate   *float64
	TurnaroundHours                          *float64
}

type DashTaskDepartment struct {
	DepartmentID, DepartmentName                                         string
	Due, Pending, InProgress, WaitingReview, Approved, Rejected, Overdue int
	OnTimeRate                                                           *float64
}

type DashTaskPerson struct {
	EmployeeID, Name, DepartmentName                        string
	Due, Approved, OnTime, Overdue, WaitingReview, Rejected int
	Completion                                              *float64
}

type DashTasks struct {
	Window                                              DashWindow
	Due, Completion, OnTimeRate, Overdue, WaitingReview DashMetric
	TurnaroundHours                                     DashMetric
	Trend                                               []DashTaskBucket
	Departments                                         []DashTaskDepartment
	Aging, Turnaround                                   []DashCount
	People                                              []DashTaskPerson
}

func turnaroundBins(rows []domain.TaskDay) []int {
	out := make([]int, 4)
	for _, r := range rows {
		out[0] += r.TurnLt4h
		out[1] += r.Turn4to24h
		out[2] += r.Turn1to3d
		out[3] += r.TurnOver3d
	}
	return out
}

func (d *Dashboard) Tasks(ctx context.Context, q DashboardQuery) (*DashTasks, error) {
	s, _, err := d.resolve(ctx, q)
	if err != nil {
		return nil, err
	}
	cur, err := d.taskRows(ctx, s, s.From, s.To)
	if err != nil {
		return nil, err
	}
	prev, err := d.taskRows(ctx, s, s.CompareFrom, s.CompareTo)
	if err != nil {
		return nil, err
	}
	tc, tp := sumTasks(cur, s.Today), sumTasks(prev, s.Today)
	out := &DashTasks{Window: s.DashWindow}

	buckets := domain.Buckets(s.From, s.upTo(), s.Granularity)
	var prevBuckets []string
	if s.compareOn() {
		prevBuckets = domain.Buckets(s.CompareFrom, domain.MinDay(s.CompareTo, s.Today), s.Granularity)
	}
	bc, bp := tasksByBucket(cur, s.Granularity, s.Today), tasksByBucket(prev, s.Granularity, s.Today)
	spark := map[string][]float64{}
	for i, b := range buckets {
		t := bc[b]
		if t == nil {
			t = &taskAgg{}
		}
		row := DashTaskBucket{Bucket: b, Due: t.due, Approved: t.approved, Finished: t.finished, OnTime: t.onTime, Overdue: t.overdue,
			Completion: t.completion(), OnTimeRate: t.onTimeRate(), TurnaroundHours: t.turnaroundHours()}
		if i < len(prevBuckets) {
			if p := bp[prevBuckets[i]]; p != nil {
				row.PreviousCompletion, row.PreviousOnTimeRate = p.completion(), p.onTimeRate()
			}
		}
		out.Trend = append(out.Trend, row)
		spark["due"] = append(spark["due"], float64(t.due))
		spark["completion"] = append(spark["completion"], orZero(t.completion()))
		spark["ontime"] = append(spark["ontime"], orZero(t.onTimeRate()))
		spark["overdue"] = append(spark["overdue"], float64(t.overdue))
		spark["waiting"] = append(spark["waiting"], float64(t.waiting))
		spark["turnaround"] = append(spark["turnaround"], orZero(t.turnaroundHours()))
	}
	out.Due = metric(count(tc.due), s.prev(count(tp.due)), spark["due"])
	out.Completion = metric(tc.completion(), s.prev(tp.completion()), spark["completion"])
	out.OnTimeRate = metric(tc.onTimeRate(), s.prev(tp.onTimeRate()), spark["ontime"])
	out.Overdue = metric(count(tc.overdue), s.prev(count(tp.overdue)), spark["overdue"])
	out.WaitingReview = metric(count(tc.waiting), s.prev(count(tp.waiting)), spark["waiting"])
	out.TurnaroundHours = metric(tc.turnaroundHours(), s.prev(tp.turnaroundHours()), spark["turnaround"])

	depts := map[string]*taskAgg{}
	people := map[string]*taskAgg{}
	aging := make([]int, len(domain.OverdueAgeBins))
	for _, r := range cur {
		if depts[r.DepartmentID] == nil {
			depts[r.DepartmentID] = &taskAgg{}
		}
		depts[r.DepartmentID].add(r, s.Today)
		if r.EmployeeID != "" {
			if people[r.EmployeeID] == nil {
				people[r.EmployeeID] = &taskAgg{}
			}
			people[r.EmployeeID].add(r, s.Today)
		}
		if r.Day < s.Today && r.Open() > 0 {
			aging[domain.OverdueAgeIndex(domain.DaysBetween(r.Day, s.Today))] += r.Open()
		}
	}
	for id, t := range depts {
		out.Departments = append(out.Departments, DashTaskDepartment{DepartmentID: id, DepartmentName: s.deptName(id), Due: t.due,
			Pending: t.pending, InProgress: t.inProgress, WaitingReview: t.waiting, Approved: t.approved, Rejected: t.rejected,
			Overdue: t.overdue, OnTimeRate: t.onTimeRate()})
	}
	sort.Slice(out.Departments, func(i, j int) bool { return out.Departments[i].DepartmentName < out.Departments[j].DepartmentName })
	for i, b := range domain.OverdueAgeBins {
		out.Aging = append(out.Aging, DashCount{Key: b.Key, Value: float64(aging[i])})
	}
	tcur, tprev := turnaroundBins(cur), turnaroundBins(prev)
	for i, k := range []string{"lt4h", "4-24h", "1-3d", "gt3d"} {
		c := DashCount{Key: k, Value: float64(tcur[i])}
		if s.compareOn() {
			c.Previous = count(tprev[i])
		}
		out.Turnaround = append(out.Turnaround, c)
	}
	for id, t := range people {
		e, _ := s.ppl.employee(id)
		out.People = append(out.People, DashTaskPerson{EmployeeID: id, Name: e.Name, DepartmentName: s.deptName(s.deptOf(e)), Due: t.due,
			Approved: t.approved, OnTime: t.onTime, Overdue: t.overdue, WaitingReview: t.waiting, Rejected: t.rejected, Completion: t.completion()})
	}
	sort.Slice(out.People, func(i, j int) bool { return out.People[i].Due > out.People[j].Due })
	return out, nil
}

// ─── Logbook & rekap ─────────────────────────────────────────

type DashLogbookBucket struct {
	Bucket                                  string
	Expected, Submitted, Reviewed, Rejected int
	AvgScore                                *float64
	Recaps                                  int
	PreviousRate                            *float64
}

type DashLogbookDepartment struct {
	Bucket, DepartmentID, DepartmentName string
	Expected, Submitted                  int
	AvgScore                             *float64
}

type DashHeatCell struct {
	Week                         string
	Weekday, Submitted, Expected int
}

type DashLogbookPerson struct {
	EmployeeID, Name, DepartmentName                string
	Expected, Submitted, Reviewed, Rejected, Recaps int
	AvgScore                                        *float64
}

type DashLogbook struct {
	Window                                                 DashWindow
	SubmissionRate, AvgScore, RejectionRate, RecapCoverage DashMetric
	Trend                                                  []DashLogbookBucket
	Departments                                            []DashLogbookDepartment
	Heatmap                                                []DashHeatCell
	Status                                                 []DashCount
	People                                                 []DashLogbookPerson
}

func (d *Dashboard) recaps(ctx context.Context, s *dashScope, from, to string) ([]domain.RecapDay, error) {
	to = s.logbookTo(to)
	if from == "" || to < from {
		return nil, nil
	}
	return d.Store.DashRecapDays(ctx, in(s.rows, from, to))
}

func (d *Dashboard) Logbook(ctx context.Context, q DashboardQuery) (*DashLogbook, error) {
	s, _, err := d.resolve(ctx, q)
	if err != nil {
		return nil, err
	}
	members, err := d.logbookMembers(ctx, s)
	if err != nil {
		return nil, err
	}
	lc, rows, err := d.logbookWindow(ctx, s, members, s.From, s.To)
	if err != nil {
		return nil, err
	}
	lp, prows, err := d.logbookWindow(ctx, s, members, s.CompareFrom, s.CompareTo)
	if err != nil {
		return nil, err
	}
	rc, err := d.recaps(ctx, s, s.From, s.To)
	if err != nil {
		return nil, err
	}
	rp, err := d.recaps(ctx, s, s.CompareFrom, s.CompareTo)
	if err != nil {
		return nil, err
	}
	staff := len(s.active())
	last := s.logbookTo(s.To)
	workdays := func(from, to string) int {
		if to < from {
			return 0
		}
		return domain.Workdays(from, to)
	}
	lc.recapExpected = staff * workdays(s.From, last)
	for _, r := range rc {
		lc.recaps += r.Recaps
	}
	if s.compareOn() {
		lp.recapExpected = staff * workdays(s.CompareFrom, s.logbookTo(s.CompareTo))
		for _, r := range rp {
			lp.recaps += r.Recaps
		}
	}
	out := &DashLogbook{Window: s.DashWindow}

	buckets := []string{}
	if last >= s.From {
		buckets = domain.Buckets(s.From, last, s.Granularity)
	}
	type agg struct {
		sent, reviewed, rejected, recaps int
		score                            float64
	}
	byBucket, byDept := map[string]*agg{}, map[string]*agg{}
	get := func(m map[string]*agg, k string) *agg {
		if m[k] == nil {
			m[k] = &agg{}
		}
		return m[k]
	}
	cells := map[string]int{}
	for _, r := range rows {
		b := get(byBucket, domain.BucketOf(r.Day, s.Granularity))
		b.sent += r.Sent()
		b.reviewed += r.Reviewed
		b.rejected += r.Rejected
		b.score += r.ScoreSum
		dk := domain.BucketOf(r.Day, s.Granularity) + "|" + r.DepartmentID
		get(byDept, dk).sent += r.Sent()
		get(byDept, dk).score += r.ScoreSum
		cells[r.Day] += r.Sent()
	}
	for _, r := range rc {
		get(byBucket, domain.BucketOf(r.Day, s.Granularity)).recaps += r.Recaps
	}
	prevSent := map[string]int{}
	var prevBuckets []string
	if s.compareOn() {
		if pl := s.logbookTo(s.CompareTo); pl >= s.CompareFrom {
			prevBuckets = domain.Buckets(s.CompareFrom, pl, s.Granularity)
		}
		for _, r := range prows {
			prevSent[domain.BucketOf(r.Day, s.Granularity)] += r.Sent()
		}
	}
	all := total(members)
	spark := map[string][]float64{}
	for i, b := range buckets {
		days := workdays(domain.MaxDay(b, s.From), bucketEnd(b, s.Granularity, last))
		a := get(byBucket, b)
		if days == 0 && a.sent == 0 {
			continue // weekend day with nothing owed
		}
		row := DashLogbookBucket{Bucket: b, Expected: all * days, Submitted: a.sent, Reviewed: a.reviewed, Rejected: a.rejected,
			AvgScore: avgOf(a.score, a.sent), Recaps: a.recaps}
		if i < len(prevBuckets) {
			pb := prevBuckets[i]
			pdays := workdays(domain.MaxDay(pb, s.CompareFrom), bucketEnd(pb, s.Granularity, s.logbookTo(s.CompareTo)))
			row.PreviousRate = domain.Ratio(prevSent[pb], all*pdays)
		}
		out.Trend = append(out.Trend, row)
		spark["rate"] = append(spark["rate"], orZero(domain.Ratio(a.sent, row.Expected)))
		spark["score"] = append(spark["score"], orZero(row.AvgScore))
		spark["rejected"] = append(spark["rejected"], orZero(domain.Ratio(a.rejected, a.sent)))
		spark["recaps"] = append(spark["recaps"], orZero(domain.Ratio(a.recaps, staff*days)))
		for _, dep := range s.departments() {
			n, ok := members[dep.ID]
			if !ok {
				continue
			}
			da := get(byDept, b+"|"+dep.ID)
			out.Departments = append(out.Departments, DashLogbookDepartment{Bucket: b, DepartmentID: dep.ID, DepartmentName: dep.Name,
				Expected: n * days, Submitted: da.sent, AvgScore: avgOf(da.score, da.sent)})
		}
	}
	out.SubmissionRate = metric(lc.rate(), s.prev(lp.rate()), spark["rate"])
	out.AvgScore = metric(lc.avgScore(), s.prev(lp.avgScore()), spark["score"])
	out.RejectionRate = metric(domain.Ratio(lc.rejected, lc.sent), s.prev(domain.Ratio(lp.rejected, lp.sent)), spark["rejected"])
	out.RecapCoverage = metric(lc.coverage(), s.prev(lp.coverage()), spark["recaps"])

	if last >= s.From {
		for _, day := range domain.EachDate(s.From, last) {
			t, _ := domain.ParseDay(day)
			wd := (int(t.Weekday())+6)%7 + 1
			exp := 0
			if domain.IsWorkday(t) {
				exp = all
			}
			if exp == 0 && cells[day] == 0 {
				continue
			}
			out.Heatmap = append(out.Heatmap, DashHeatCell{Week: domain.BucketOf(day, domain.BucketWeek), Weekday: wd, Submitted: cells[day], Expected: exp})
		}
	}
	statusCounts := []struct {
		key       string
		cur, prev int
	}{{"draft", lc.draft, lp.draft}, {"submitted", lc.submitted, lp.submitted}, {"reviewed", lc.reviewed, lp.reviewed},
		{"rejected", lc.rejected, lp.rejected}}
	for _, sc := range statusCounts {
		out.Status = append(out.Status, DashCount{Key: sc.key, Value: float64(sc.cur), Previous: s.prev(count(sc.prev))})
	}

	per := map[string]*DashLogbookPerson{}
	scores := map[string]float64{}
	person := func(id string) *DashLogbookPerson {
		if per[id] == nil {
			e, _ := s.ppl.employee(id)
			per[id] = &DashLogbookPerson{EmployeeID: id, Name: e.Name, DepartmentName: s.deptName(s.deptOf(e))}
		}
		return per[id]
	}
	days := workdays(s.From, last)
	for _, e := range s.active() {
		if _, ok := members[s.deptOf(e)]; ok {
			person(e.ID).Expected = days
		}
	}
	for _, r := range rows {
		p := person(r.EmployeeID)
		p.Submitted += r.Sent()
		p.Reviewed += r.Reviewed
		p.Rejected += r.Rejected
		scores[r.EmployeeID] += r.ScoreSum
	}
	for _, r := range rc {
		if p := per[r.EmployeeID]; p != nil {
			p.Recaps += r.Recaps
		}
	}
	for id, p := range per {
		p.AvgScore = avgOf(scores[id], p.Submitted)
		out.People = append(out.People, *p)
	}
	sort.Slice(out.People, func(i, j int) bool {
		ri := orZero(domain.Ratio(out.People[i].Submitted, out.People[i].Expected))
		rj := orZero(domain.Ratio(out.People[j].Submitted, out.People[j].Expected))
		if ri != rj {
			return ri < rj
		}
		return out.People[i].Name < out.People[j].Name
	})
	return out, nil
}

// ─── Review & 360 ────────────────────────────────────────────

type DashCycleProgress struct {
	CycleID, Name, Status, DueDate                string
	Total, SelfAssessed, Rated, Signed, Finalized int
	AvgScore                                      *float64
}

type DashRelationship struct {
	Relationship     string
	Total, Submitted int
}

type DashCategoryScore struct {
	Code, Name      string
	Score, Previous *float64
}

type DashReviewDepartment struct {
	DepartmentID, DepartmentName string
	Reviews, Finalized           int
	AvgScore, Avg360             *float64
}

type DashReviews struct {
	Window                                                        DashWindow
	Cycle                                                         DashCycleProgress
	Cycles                                                        []DashCycleProgress
	Ratings                                                       []DashCount
	ReviewScore                                                   DashMetric
	FeedbackCycleID, FeedbackCycleName, PreviousFeedbackCycleName string
	FeedbackProgress                                              []DashRelationship
	FeedbackCategories                                            []DashCategoryScore
	FeedbackScore, FeedbackCompletion                             DashMetric
	Departments                                                   []DashReviewDepartment
}

func (s *dashScope) covers(employeeID string) bool {
	if s.persons.All {
		return true
	}
	for _, id := range s.persons.EmployeeIDs {
		if id == employeeID {
			return true
		}
	}
	return false
}

// pickCycle is the latest cycle overlapping [from, to], else the latest
// one starting before to; -1 when none. Cycles are newest first.
func pickCycle(n int, span func(i int) (string, string), from, to string) int {
	for i := 0; i < n; i++ {
		start, end := span(i)
		if start <= to && end >= from {
			return i
		}
	}
	for i := 0; i < n; i++ {
		if start, _ := span(i); start <= to {
			return i
		}
	}
	return -1
}

func (d *Dashboard) cycleReviews(ctx context.Context, s *dashScope, c domain.ReviewCycle) ([]domain.Review, DashCycleProgress, error) {
	all, err := d.Store.ListReviewsByCycle(ctx, c.ID)
	if err != nil {
		return nil, DashCycleProgress{}, err
	}
	p := DashCycleProgress{CycleID: c.ID, Name: c.Name, Status: c.Status, DueDate: c.DueDate}
	var list []domain.Review
	sum, n := 0.0, 0
	for _, r := range all {
		if !s.covers(r.EmployeeID) {
			continue
		}
		list = append(list, r)
		p.Total++
		if r.SelfAssessment != nil && strings.TrimSpace(*r.SelfAssessment) != "" {
			p.SelfAssessed++
		}
		if r.RatedItems > 0 {
			p.Rated++
		}
		if r.ReviewerSignDate != nil {
			p.Signed++
		}
		if r.Status == domain.ReviewFinal {
			p.Finalized++
		}
		if r.Grand != nil {
			sum += *r.Grand
			n++
		}
	}
	p.AvgScore = avgOf(sum, n)
	return list, p, nil
}

func (d *Dashboard) Reviews(ctx context.Context, q DashboardQuery) (*DashReviews, error) {
	s, _, err := d.resolve(ctx, q)
	if err != nil {
		return nil, err
	}
	out := &DashReviews{Window: s.DashWindow}
	cycles, err := d.Store.ListReviewCycles(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(cycles, func(i, j int) bool {
		return cycles[i].Year*10+cycles[i].Quarter > cycles[j].Year*10+cycles[j].Quarter
	})
	chosen := pickCycle(len(cycles), func(i int) (string, string) { return cycles[i].StartDate, cycles[i].EndDate }, s.From, s.To)
	reviewsByDept := map[string][]domain.Review{}
	if chosen >= 0 {
		var spark []float64
		var prevScore *float64
		var prevReviews []domain.Review
		// The chosen cycle and up to three before it, oldest first.
		last := min(len(cycles), chosen+4)
		for i := last - 1; i >= chosen; i-- {
			list, p, err := d.cycleReviews(ctx, s, cycles[i])
			if err != nil {
				return nil, err
			}
			out.Cycles = append(out.Cycles, p)
			spark = append(spark, orZero(p.AvgScore))
			switch i {
			case chosen:
				out.Cycle = p
				for _, r := range list {
					if r.DepartmentID != nil {
						reviewsByDept[*r.DepartmentID] = append(reviewsByDept[*r.DepartmentID], r)
					}
				}
				categories, err := d.Store.ListPerformanceCategories(ctx)
				if err != nil {
					return nil, err
				}
				sort.Slice(categories, func(a, b int) bool { return categories[a].MinScore > categories[b].MinScore })
				for _, c := range categories {
					cur, prev := 0, 0
					for _, r := range list {
						if r.Category != nil && *r.Category == c.Name {
							cur++
						}
					}
					for _, r := range prevReviews {
						if r.Category != nil && *r.Category == c.Name {
							prev++
						}
					}
					dc := DashCount{Key: c.Name, Value: float64(cur)}
					if chosen+1 < len(cycles) {
						dc.Previous = count(prev)
					}
					out.Ratings = append(out.Ratings, dc)
				}
			case chosen + 1:
				prevScore, prevReviews = p.AvgScore, list
			}
		}
		out.ReviewScore = metric(out.Cycle.AvgScore, prevScore, spark)
	} else {
		out.ReviewScore = metric(nil, nil, nil)
	}

	// 360 feedback
	fcs, err := d.Store.ListFeedbackCycles(ctx, "")
	if err != nil {
		return nil, err
	}
	span := func(c domain.FeedbackCycle) (string, string) {
		created := domain.DayKey(c.CreatedAt, domain.Jakarta)
		start, end := created, created
		if c.StartDate != nil {
			start = *c.StartDate
		}
		if c.EndDate != nil {
			end = *c.EndDate
		}
		return start, end
	}
	sort.Slice(fcs, func(i, j int) bool {
		a, _ := span(fcs[i])
		b, _ := span(fcs[j])
		return a > b
	})
	fi := pickCycle(len(fcs), func(i int) (string, string) { return span(fcs[i]) }, s.From, s.To)
	scoreByDept := map[string][]float64{}
	out.FeedbackScore, out.FeedbackCompletion = metric(nil, nil, nil), metric(nil, nil, nil)
	if fi >= 0 {
		cats, err := d.Store.ListFeedbackCategories(ctx)
		if err != nil {
			return nil, err
		}
		type cycleStats struct {
			overall    *float64
			completion *float64
			categories map[string]*float64
			byRel      map[string]*DashRelationship
			summaries  []domain.FeedbackSummary
		}
		stats := func(c domain.FeedbackCycle) (*cycleStats, error) {
			st := &cycleStats{categories: map[string]*float64{}, byRel: map[string]*DashRelationship{}}
			counts, err := d.Store.DashFeedbackAssignments(ctx, c.ID, s.persons)
			if err != nil {
				return nil, err
			}
			done, all := 0, 0
			for _, a := range counts {
				r := st.byRel[a.Relationship]
				if r == nil {
					r = &DashRelationship{Relationship: a.Relationship}
					st.byRel[a.Relationship] = r
				}
				r.Total += a.Total
				all += a.Total
				if a.Status == domain.AssignmentSubmitted || a.Status == domain.AssignmentApproved {
					r.Submitted += a.Total
					done += a.Total
				}
			}
			st.completion = domain.Ratio(done, all)
			sums, err := d.Store.ListFeedbackSummaries(ctx, c.ID, "")
			if err != nil {
				return nil, err
			}
			osum, on := 0.0, 0
			csum, cn := map[string]float64{}, map[string]int{}
			for _, sm := range sums {
				if !s.covers(sm.EmployeeID) {
					continue
				}
				st.summaries = append(st.summaries, sm)
				if sm.Overall360 != nil {
					osum += *sm.Overall360
					on++
				}
				for code, v := range sm.Scores {
					if v != nil {
						csum[code] += *v
						cn[code]++
					}
				}
			}
			st.overall = avgOf(osum, on)
			for code, n := range cn {
				st.categories[code] = avgOf(csum[code], n)
			}
			return st, nil
		}
		cur, err := stats(fcs[fi])
		if err != nil {
			return nil, err
		}
		prev := &cycleStats{categories: map[string]*float64{}}
		if fi+1 < len(fcs) {
			if prev, err = stats(fcs[fi+1]); err != nil {
				return nil, err
			}
			out.PreviousFeedbackCycleName = fcs[fi+1].Name
		}
		out.FeedbackCycleID, out.FeedbackCycleName = fcs[fi].ID, fcs[fi].Name
		for _, rel := range []string{domain.RelSelf, domain.RelManager, domain.RelPeer, domain.RelSubordinate} {
			if r := cur.byRel[rel]; r != nil {
				out.FeedbackProgress = append(out.FeedbackProgress, *r)
			}
		}
		for _, c := range cats {
			out.FeedbackCategories = append(out.FeedbackCategories, DashCategoryScore{Code: c.Code, Name: c.Name, Score: cur.categories[c.Code],
				Previous: prev.categories[c.Code]})
		}
		out.FeedbackScore = metric(cur.overall, prev.overall, nil)
		out.FeedbackCompletion = metric(cur.completion, prev.completion, nil)
		for _, sm := range cur.summaries {
			if e, ok := s.ppl.employee(sm.EmployeeID); ok && sm.Overall360 != nil {
				scoreByDept[s.deptOf(e)] = append(scoreByDept[s.deptOf(e)], *sm.Overall360)
			}
		}
	}

	for _, dep := range s.departments() {
		list := reviewsByDept[dep.ID]
		row := DashReviewDepartment{DepartmentID: dep.ID, DepartmentName: dep.Name, Reviews: len(list)}
		sum, n := 0.0, 0
		for _, r := range list {
			if r.Status == domain.ReviewFinal {
				row.Finalized++
			}
			if r.Grand != nil {
				sum += *r.Grand
				n++
			}
		}
		row.AvgScore = avgOf(sum, n)
		fsum := 0.0
		for _, v := range scoreByDept[dep.ID] {
			fsum += v
		}
		row.Avg360 = avgOf(fsum, len(scoreByDept[dep.ID]))
		if row.Reviews > 0 || row.Avg360 != nil {
			out.Departments = append(out.Departments, row)
		}
	}
	return out, nil
}

// ─── Perbaikan ───────────────────────────────────────────────

type DashImprovementBucket struct {
	Bucket                                                          string
	Manual, AI, Retro, Automatic, Completed, AIAccepted, AIDeclined int
}

type DashOverdueImprovement struct {
	ID, Title, Scope, Source, Status, OwnerName, DepartmentName, DueDate string
	DaysOverdue                                                          int
}

type DashImprovementDepartment struct {
	DepartmentID, DepartmentName               string
	Open, InProgress, Done, Overdue, Suggested int
}

type DashImprovements struct {
	Window                                      DashWindow
	Open, Created, Completed, Overdue, AIAccept DashMetric
	Trend                                       []DashImprovementBucket
	Status, Sources                             []DashCount
	OverdueItems                                []DashOverdueImprovement
	Departments                                 []DashImprovementDepartment
}

func (d *Dashboard) Improvements(ctx context.Context, q DashboardQuery) (*DashImprovements, error) {
	s, _, err := d.resolve(ctx, q)
	if err != nil {
		return nil, err
	}
	items, err := d.Store.DashImprovements(ctx, in(s.rows, s.From, s.To))
	if err != nil {
		return nil, err
	}
	out := &DashImprovements{Window: s.DashWindow}
	buckets := domain.Buckets(s.From, s.To, s.Granularity)
	byBucket := map[string]*DashImprovementBucket{}
	for _, b := range buckets {
		byBucket[b] = &DashImprovementBucket{Bucket: b}
	}
	created, prevCreated, completed, prevCompleted := 0, 0, 0, 0
	status, prevStatus := map[string]int{}, map[string]int{}
	sources, prevSources := map[string]int{}, map[string]int{}
	depts := map[string]*DashImprovementDepartment{}
	dept := func(id string) *DashImprovementDepartment {
		if depts[id] == nil {
			depts[id] = &DashImprovementDepartment{DepartmentID: id, DepartmentName: s.deptName(id)}
		}
		return depts[id]
	}
	for _, m := range items {
		if createdIn(m, s.From, s.To) {
			created++
			status[m.Status]++
			sources[m.Source]++
			b := byBucket[domain.BucketOf(domain.DayKey(m.CreatedAt, domain.Jakarta), s.Granularity)]
			if b != nil {
				switch m.Source {
				case domain.SourceManualItem:
					b.Manual++
				case domain.SourceAI:
					b.AI++
					switch m.Status {
					case domain.ImpSuggested:
					case domain.ImpDeclined:
						b.AIDeclined++
					default:
						b.AIAccepted++
					}
				case domain.SourceRetro:
					b.Retro++
				default:
					b.Automatic++
				}
			}
		}
		if createdIn(m, s.CompareFrom, s.CompareTo) {
			prevCreated++
			prevStatus[m.Status]++
			prevSources[m.Source]++
		}
		if completedIn(m, s.From, s.To) {
			completed++
			if b := byBucket[domain.BucketOf(domain.DayKey(*m.CompletedAt, domain.Jakarta), s.Granularity)]; b != nil {
				b.Completed++
			}
			dept(s.improvementDept(m)).Done++
		}
		if completedIn(m, s.CompareFrom, s.CompareTo) {
			prevCompleted++
		}
		id := s.improvementDept(m)
		switch {
		case m.Status == domain.ImpSuggested:
			dept(id).Suggested++
		case m.OpenAt(s.To) && m.Status == domain.ImpInProgress:
			dept(id).InProgress++
		case m.OpenAt(s.To):
			dept(id).Open++
		}
		if overdueAt(m, s.To) {
			dept(id).Overdue++
		}
		if domain.ImprovementActive(m.Status) && m.DueDate != nil && *m.DueDate < s.Today {
			owner := ""
			if m.OwnerEmployeeID != nil {
				owner = s.ppl.name(m.OwnerEmployeeID)
			} else if m.EmployeeID != nil {
				owner = s.ppl.name(m.EmployeeID)
			}
			out.OverdueItems = append(out.OverdueItems, DashOverdueImprovement{ID: m.ID, Title: m.Title, Scope: m.Scope, Source: m.Source,
				Status: m.Status, OwnerName: owner, DepartmentName: s.deptName(id), DueDate: *m.DueDate,
				DaysOverdue: domain.DaysBetween(*m.DueDate, s.Today)})
		}
	}
	sort.Slice(out.OverdueItems, func(i, j int) bool { return out.OverdueItems[i].DaysOverdue > out.OverdueItems[j].DaysOverdue })
	if len(out.OverdueItems) > 10 {
		out.OverdueItems = out.OverdueItems[:10]
	}
	var sparkOpen, sparkCreated, sparkDone, sparkOverdue []float64
	for _, b := range buckets {
		end := bucketEnd(b, s.Granularity, s.To)
		bb := byBucket[b]
		out.Trend = append(out.Trend, *bb)
		sparkOpen = append(sparkOpen, float64(countOpen(items, end)))
		sparkCreated = append(sparkCreated, float64(bb.Manual+bb.AI+bb.Retro+bb.Automatic))
		sparkDone = append(sparkDone, float64(bb.Completed))
		od := 0
		for _, m := range items {
			if overdueAt(m, end) {
				od++
			}
		}
		sparkOverdue = append(sparkOverdue, float64(od))
	}
	curOverdue, prevOverdue := 0, 0
	for _, m := range items {
		if overdueAt(m, s.To) {
			curOverdue++
		}
		if overdueAt(m, s.CompareTo) {
			prevOverdue++
		}
	}
	out.Open = metric(count(countOpen(items, s.To)), s.prev(count(countOpen(items, s.CompareTo))), sparkOpen)
	out.Created = metric(count(created), s.prev(count(prevCreated)), sparkCreated)
	out.Completed = metric(count(completed), s.prev(count(prevCompleted)), sparkDone)
	out.Overdue = metric(count(curOverdue), s.prev(count(prevOverdue)), sparkOverdue)
	out.AIAccept = metric(aiAcceptance(items, s.From, s.To), s.prev(aiAcceptance(items, s.CompareFrom, s.CompareTo)), nil)
	for _, st := range []string{domain.ImpSuggested, domain.ImpOpen, domain.ImpInProgress, domain.ImpDone, domain.ImpDropped, domain.ImpDeclined} {
		out.Status = append(out.Status, DashCount{Key: st, Value: float64(status[st]), Previous: s.prev(count(prevStatus[st]))})
	}
	for _, src := range domain.ImprovementSources {
		out.Sources = append(out.Sources, DashCount{Key: src, Value: float64(sources[src]), Previous: s.prev(count(prevSources[src]))})
	}
	for _, dep := range s.departments() {
		if r := depts[dep.ID]; r != nil {
			out.Departments = append(out.Departments, *r)
		}
	}
	return out, nil
}
