package app

import (
	"context"
	"slices"
	"sort"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Dashboard serves the performance dashboard: one call per tab, all with
// the same window, comparison and people filters. hr readers see the
// company, leads their team, everyone else only themselves.
type Dashboard struct{ Deps }

func NewDashboard(d Deps) *Dashboard { return &Dashboard{d} }

// DashboardQuery is the filter bar of the page.
type DashboardQuery struct {
	From, To, Compare, DepartmentID, EmployeeID, Scope string
}

// DashWindow is the resolved window every response carries.
type DashWindow struct {
	domain.DashWindow
	Scope string
	Today string
}

// DashMetric is a tile: value, comparison value and its trend.
type DashMetric struct {
	Value, Previous *float64
	Spark           []float64
}

// DashCount is a labelled count with an optional comparison value.
type DashCount struct {
	Key      string
	Value    float64
	Previous *float64
}

// dashScope is a resolved query: window, scope and the people it covers.
type dashScope struct {
	DashWindow
	ppl *people
	// people in scope after the department and employee filters
	set []domain.Employee
	// rows filter (department by the row's department column)
	rows domain.DashFilter
	// employee-only tables (no department column)
	persons domain.DashFilter
}

func (s *dashScope) upTo() string { return domain.MinDay(s.To, s.Today) }

// in returns f limited to [from, to].
func in(f domain.DashFilter, from, to string) domain.DashFilter {
	f.From, f.To = from, to
	return f
}

// active are the active employees of the scope.
func (s *dashScope) active() []domain.Employee {
	var out []domain.Employee
	for _, e := range s.set {
		if e.Active {
			out = append(out, e)
		}
	}
	return out
}

func (s *dashScope) deptOf(e domain.Employee) string {
	if e.DepartmentID == nil {
		return ""
	}
	return *e.DepartmentID
}

// departments lists the departments the scope touches, by name.
func (s *dashScope) departments() []domain.Department {
	seen := map[string]bool{}
	var out []domain.Department
	add := func(id string) {
		if d, ok := s.ppl.depts[id]; ok && !seen[id] {
			seen[id] = true
			out = append(out, d)
		}
	}
	if s.Scope == domain.DashScopeAll && s.rows.All {
		for id := range s.ppl.depts {
			if s.rows.DepartmentID == "" || s.rows.DepartmentID == id {
				add(id)
			}
		}
	}
	for _, e := range s.set {
		add(s.deptOf(e))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *dashScope) deptName(id string) string { return s.ppl.depts[id].Name }

func (d *Dashboard) resolve(ctx context.Context, q DashboardQuery) (*dashScope, []string, error) {
	a, err := d.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	var scopes []string
	if a.Employee != nil {
		scopes = append(scopes, domain.DashScopeMe)
	}
	if a.IsLead() {
		scopes = append(scopes, domain.DashScopeTeam)
	}
	if a.HRRead {
		scopes = append(scopes, domain.DashScopeAll)
	}
	if len(scopes) == 0 {
		return nil, nil, errNoEmployee
	}
	scope := q.Scope
	if !slices.Contains(scopes, scope) {
		scope = scopes[len(scopes)-1]
	}
	today := d.today()
	w, ok := domain.ResolveWindow(q.From, q.To, q.Compare, today)
	if !ok {
		return nil, nil, apperr.Field("from", "Rentang tanggal tidak valid")
	}
	ppl, err := d.people(ctx)
	if err != nil {
		return nil, nil, err
	}
	s := &dashScope{DashWindow: DashWindow{DashWindow: w, Scope: scope, Today: today}, ppl: ppl}
	for _, e := range ppl.byID {
		switch {
		case scope == domain.DashScopeAll,
			scope == domain.DashScopeTeam && (a.Leads(e) || a.IsEmployee(e.ID)),
			scope == domain.DashScopeMe && a.IsEmployee(e.ID):
			if q.DepartmentID == "" || e.InDepartment(q.DepartmentID) {
				s.set = append(s.set, e)
			}
		}
	}
	if q.EmployeeID != "" {
		i := slices.IndexFunc(s.set, func(e domain.Employee) bool { return e.ID == q.EmployeeID })
		if i < 0 {
			return nil, nil, apperr.New(apperr.Unauthorized, "Karyawan ini di luar cakupan dashboard Anda")
		}
		s.set = s.set[i : i+1]
	}
	sort.Slice(s.set, func(i, j int) bool { return s.set[i].Name < s.set[j].Name })
	ids := make([]string, len(s.set))
	for i, e := range s.set {
		ids[i] = e.ID
	}
	all := scope == domain.DashScopeAll && q.EmployeeID == ""
	s.rows = domain.DashFilter{All: all, EmployeeIDs: ids, DepartmentID: q.DepartmentID}
	s.persons = domain.DashFilter{All: all && q.DepartmentID == "", EmployeeIDs: ids}
	return s, scopes, nil
}

// ─── shared math ─────────────────────────────────────────────

func ptr(v float64) *float64 { return &v }

func avgOf(sum float64, n int) *float64 {
	if n == 0 {
		return nil
	}
	return ptr(domain.Round2(sum / float64(n)))
}

func metric(value, previous *float64, spark []float64) DashMetric {
	if spark == nil {
		spark = []float64{}
	}
	return DashMetric{Value: value, Previous: previous, Spark: spark}
}

func count(v int) *float64 { return ptr(float64(v)) }

func orZero(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// compareOn reports a window with a comparison range.
func (s *dashScope) compareOn() bool { return s.CompareFrom != "" }

// prev passes a comparison value through, nil without comparison.
func (s *dashScope) prev(v *float64) *float64 {
	if !s.compareOn() {
		return nil
	}
	return v
}

// ─── context ─────────────────────────────────────────────────

// DashContext is the filter bar: resolved window and the options.
type DashContext struct {
	Window      DashWindow
	Scopes      []string
	Departments []domain.Department
	People      []domain.Employee
}

func (d *Dashboard) Context(ctx context.Context, q DashboardQuery) (*DashContext, error) {
	// Options cover the whole scope, so the department and employee
	// filters do not narrow their own lists.
	s, scopes, err := d.resolve(ctx, DashboardQuery{From: q.From, To: q.To, Compare: q.Compare, Scope: q.Scope})
	if err != nil {
		return nil, err
	}
	out := &DashContext{Window: s.DashWindow, Scopes: scopes, Departments: s.departments()}
	for _, e := range s.active() {
		if q.DepartmentID == "" || e.InDepartment(q.DepartmentID) {
			out.People = append(out.People, e)
		}
	}
	return out, nil
}

// ─── tasks math ──────────────────────────────────────────────

type taskAgg struct {
	due, approved, finished, onTime, overdue, waiting, rejected, pending, inProgress int
	reviewed                                                                         int
	turnaround                                                                       float64
}

func (t *taskAgg) add(r domain.TaskDay, today string) {
	t.due += r.Due()
	t.approved += r.Approved
	t.finished += r.Approved + r.Done
	t.onTime += r.OnTime
	t.waiting += r.Done
	t.rejected += r.Rejected
	t.pending += r.Pending
	t.inProgress += r.InProgress
	t.reviewed += r.Reviewed
	t.turnaround += r.TurnaroundHours
	if r.Day < today {
		t.overdue += r.Open()
	}
}

func (t taskAgg) completion() *float64 { return domain.Ratio(t.approved, t.due) }
func (t taskAgg) onTimeRate() *float64 { return domain.Ratio(t.onTime, t.finished) }
func (t taskAgg) turnaroundHours() *float64 {
	return avgOf(t.turnaround, t.reviewed)
}

func sumTasks(rows []domain.TaskDay, today string) taskAgg {
	var t taskAgg
	for _, r := range rows {
		t.add(r, today)
	}
	return t
}

func tasksByBucket(rows []domain.TaskDay, granularity, today string) map[string]*taskAgg {
	out := map[string]*taskAgg{}
	for _, r := range rows {
		b := domain.BucketOf(r.Day, granularity)
		if out[b] == nil {
			out[b] = &taskAgg{}
		}
		out[b].add(r, today)
	}
	return out
}

func (d *Dashboard) taskRows(ctx context.Context, s *dashScope, from, to string) ([]domain.TaskDay, error) {
	to = domain.MinDay(to, s.Today)
	if from == "" || to < from {
		return nil, nil
	}
	return d.Store.DashTaskDays(ctx, in(s.rows, from, to))
}

// ─── logbook math ────────────────────────────────────────────

type logAgg struct {
	expected, sent, reviewed, rejected, draft, submitted, recaps, recapExpected int
	score                                                                       float64
}

func (l logAgg) rate() *float64     { return domain.Ratio(l.sent, l.expected) }
func (l logAgg) avgScore() *float64 { return avgOf(l.score, l.sent) }
func (l logAgg) coverage() *float64 { return domain.Ratio(l.recaps, l.recapExpected) }

// logbookTo is the last day a logbook is owed: yesterday at the latest.
func (s *dashScope) logbookTo(to string) string {
	return domain.MinDay(to, domain.AddDays(s.Today, -1))
}

// logbookMembers are the active employees whose department keeps a daily
// logbook, by department.
func (d *Dashboard) logbookMembers(ctx context.Context, s *dashScope) (map[string]int, error) {
	depts, err := d.Store.DailyTemplateDepartments(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, e := range s.active() {
		if id := s.deptOf(e); id != "" && slices.Contains(depts, id) {
			out[id]++
		}
	}
	return out, nil
}

func total(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// logbookWindow sums entries and expectations over [from, to].
func (d *Dashboard) logbookWindow(ctx context.Context, s *dashScope, members map[string]int, from, to string) (logAgg, []domain.LogbookDay, error) {
	var l logAgg
	to = s.logbookTo(to)
	if from == "" || to < from {
		return l, nil, nil
	}
	rows, err := d.Store.DashLogbookDays(ctx, in(s.rows, from, to))
	if err != nil {
		return l, nil, err
	}
	l.expected = total(members) * domain.Workdays(from, to)
	for _, r := range rows {
		l.sent += r.Sent()
		l.reviewed += r.Reviewed
		l.rejected += r.Rejected
		l.draft += r.Draft
		l.submitted += r.Submitted
		l.score += r.ScoreSum
	}
	return l, rows, nil
}

// ─── scorecards math ─────────────────────────────────────────

func avgScore(cards []domain.DashScorecard) *float64 {
	sum, n := 0.0, 0
	for _, c := range cards {
		if c.Score != nil {
			sum += *c.Score
			n++
		}
	}
	return avgOf(sum, n)
}

// perEmployee averages each employee's scored months.
func perEmployee(cards []domain.DashScorecard) map[string]*float64 {
	sums, ns := map[string]float64{}, map[string]int{}
	for _, c := range cards {
		if c.Score != nil {
			sums[c.EmployeeID] += *c.Score
			ns[c.EmployeeID]++
		}
	}
	out := map[string]*float64{}
	for id, n := range ns {
		out[id] = avgOf(sums[id], n)
	}
	return out
}

func (d *Dashboard) cards(ctx context.Context, s *dashScope, from, to string) ([]domain.DashScorecard, error) {
	if from == "" {
		return nil, nil
	}
	return d.Store.DashScorecards(ctx, in(s.rows, from, to))
}

// monthly is the 12-month KPI history up to the window end: overall and
// per department averages.
type monthly struct {
	months []string
	all    []float64
	dept   []DeptMonthScore
}

// DeptMonthScore is a department's average score in a month.
type DeptMonthScore struct {
	DepartmentID, DepartmentName, Period string
	Score                                *float64
	Scorecards                           int
}

func (d *Dashboard) monthly(ctx context.Context, s *dashScope) (*monthly, error) {
	periods := domain.MonthsUpTo(s.To, 12)
	cards, err := d.cards(ctx, s, periods[0].Span().Start, s.To)
	if err != nil {
		return nil, err
	}
	byMonth := map[string][]domain.DashScorecard{}
	byDept := map[string]map[string][]domain.DashScorecard{}
	for _, c := range cards {
		k := c.Period.Key()
		byMonth[k] = append(byMonth[k], c)
		if c.DepartmentID != nil {
			if byDept[*c.DepartmentID] == nil {
				byDept[*c.DepartmentID] = map[string][]domain.DashScorecard{}
			}
			byDept[*c.DepartmentID][k] = append(byDept[*c.DepartmentID][k], c)
		}
	}
	m := &monthly{}
	for _, p := range periods {
		k := p.Key()
		m.months = append(m.months, k)
		m.all = append(m.all, orZero(avgScore(byMonth[k])))
	}
	for _, dep := range s.departments() {
		for _, k := range m.months {
			list := byDept[dep.ID][k]
			m.dept = append(m.dept, DeptMonthScore{DepartmentID: dep.ID, DepartmentName: dep.Name, Period: k, Score: avgScore(list), Scorecards: len(list)})
		}
	}
	return m, nil
}

// ─── improvements math ───────────────────────────────────────

func (s *dashScope) improvementDept(m domain.DashImprovement) string {
	if m.DepartmentID != nil {
		return *m.DepartmentID
	}
	if m.EmployeeID != nil {
		if e, ok := s.ppl.employee(*m.EmployeeID); ok {
			return s.deptOf(e)
		}
	}
	return ""
}

func createdIn(m domain.DashImprovement, from, to string) bool {
	d := domain.DayKey(m.CreatedAt, domain.Jakarta)
	return from != "" && d >= from && d <= to
}

func completedIn(m domain.DashImprovement, from, to string) bool {
	if m.Status != domain.ImpDone || m.CompletedAt == nil || from == "" {
		return false
	}
	d := domain.DayKey(*m.CompletedAt, domain.Jakarta)
	return d >= from && d <= to
}

func overdueAt(m domain.DashImprovement, day string) bool {
	return m.OpenAt(day) && m.DueDate != nil && *m.DueDate < day
}

func countOpen(items []domain.DashImprovement, day string) int {
	n := 0
	for _, m := range items {
		if m.OpenAt(day) {
			n++
		}
	}
	return n
}

// aiAcceptance is accepted / (accepted + declined) of AI suggestions made
// in the window; suggestions still pending do not count.
func aiAcceptance(items []domain.DashImprovement, from, to string) *float64 {
	acc, dec := 0, 0
	for _, m := range items {
		if m.Source != domain.SourceAI || !createdIn(m, from, to) {
			continue
		}
		switch m.Status {
		case domain.ImpSuggested:
		case domain.ImpDeclined:
			dec++
		default:
			acc++
		}
	}
	return domain.Ratio(acc, acc+dec)
}

// bucketEnd is the last day of bucket b clipped to the window end.
func bucketEnd(b, granularity, to string) string {
	var end string
	switch granularity {
	case domain.BucketWeek:
		end = domain.AddDays(b, 6)
	case domain.BucketMonth:
		end = domain.PeriodOf(b).Span().End
	default:
		end = b
	}
	return domain.MinDay(end, to)
}
