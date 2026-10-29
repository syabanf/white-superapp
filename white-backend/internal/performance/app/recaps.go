package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Recaps builds the daily work recap (17:30 WIB on workdays) and the
// monthly recap (on the 1st) per employee and per department, with an AI
// summary or narrative and a PDF export.
type Recaps struct {
	Deps
	improvements *Improvements
}

func NewRecaps(d Deps, imp *Improvements) *Recaps { return &Recaps{d, imp} }

const (
	msgRecapGone      = "Rekap tidak ditemukan"
	maxDailyNotes     = 4000
	monthlySignals    = 10
	accomplishmentTop = 5
)

// dayWindow is the WIB day of date as a UTC half-open interval.
func dayWindow(date string) (time.Time, time.Time) {
	d, _ := domain.ParseDay(date)
	from := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, domain.Jakarta)
	return from.UTC(), from.AddDate(0, 0, 1).UTC()
}

// buildDaily collects what the employee did on date: occurrences finished
// or updated (with sub-tasks and notes) and logbook entries of the day.
func (r *Recaps) buildDaily(ctx context.Context, employeeID, date string) (domain.DailyRecapItems, error) {
	items := domain.DailyRecapItems{Tasks: []domain.RecapTask{}, Logbook: []domain.RecapLogbook{}}
	from, to := dayWindow(date)
	occs, err := r.Store.RecapOccurrences(ctx, employeeID, from, to)
	if err != nil {
		return items, err
	}
	notes, err := r.Store.RecapNotes(ctx, employeeID, from, to)
	if err != nil {
		return items, err
	}
	byOcc := map[string][]string{}
	for _, n := range notes {
		byOcc[n.OccurrenceID] = append(byOcc[n.OccurrenceID], n.Note)
	}
	for _, o := range occs {
		var manual, sub *float64
		if o.ProgressPercent != nil {
			p := float64(*o.ProgressPercent)
			manual = &p
		}
		if o.SubtaskTotal != nil && *o.SubtaskTotal > 0 {
			v := *o.SubtaskDone / *o.SubtaskTotal * 100
			sub = &v
		}
		t := domain.RecapTask{OccurrenceID: o.OccurrenceID, Title: o.Title, Status: o.Status, Priority: o.Priority, Completed: o.CompletedInDay,
			Progress: domain.EffectiveProgress(o.Status, manual, sub), Subtasks: []domain.RecapSubtask{}, Notes: byOcc[o.OccurrenceID]}
		if t.Notes == nil {
			t.Notes = []string{}
		}
		if o.SubtaskTotal != nil {
			states, err := r.Store.ListSubtaskStates(ctx, o.OccurrenceID, o.TaskID)
			if err != nil {
				return items, err
			}
			for _, s := range states {
				t.Subtasks = append(t.Subtasks, domain.RecapSubtask{Title: s.Title, Checked: s.IsChecked})
			}
		}
		items.Tasks = append(items.Tasks, t)
	}
	entries, _, err := r.Store.ListEntries(ctx, domain.EntryFilter{EmployeeIDs: []string{employeeID}, Date: date, Limit: 50})
	if err != nil {
		return items, err
	}
	for _, e := range entries {
		items.Logbook = append(items.Logbook, domain.RecapLogbook{EntryID: e.ID, Title: e.Title, Status: e.Status, Completion: e.Completion, KPIScore: e.KPIScore})
	}
	return items, nil
}

// generateDaily rebuilds and stores the recap with a fresh summary.
func (r *Recaps) generateDaily(ctx context.Context, e domain.Employee, date string) (*domain.DailyRecap, error) {
	items, err := r.buildDaily(ctx, e.ID, date)
	if err != nil {
		return nil, err
	}
	cur, err := r.Store.GetDailyRecap(ctx, e.ID, date)
	if err != nil {
		return nil, err
	}
	notes := ""
	if cur != nil {
		notes = cur.Notes
	}
	summary, model, err := r.AI.DailySummary(ctx, domain.DailyRecapInput{EmployeeName: e.Name, Date: date, Items: items, Notes: notes})
	if err != nil {
		return nil, err
	}
	rec := domain.DailyRecap{EmployeeID: e.ID, DepartmentID: e.DepartmentID, Date: date, Items: items, Summary: summary, SummaryModel: model,
		GeneratedAt: timePtr(r.now())}
	if err := r.Store.UpsertDailyRecap(ctx, rec); err != nil {
		return nil, err
	}
	return r.Store.GetDailyRecap(ctx, e.ID, date)
}

// GenerateDaily writes an employee's recap of a day from that day's tasks
// and logbook (HR writers only; the seed backfills history with it).
func (r *Recaps) GenerateDaily(ctx context.Context, employeeID, date string) error {
	a, err := r.actor(ctx)
	if err != nil {
		return err
	}
	if err := requireHRWrite(a); err != nil {
		return err
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return err
	}
	e, ok := ppl.employee(employeeID)
	if !ok || !validDay(date) {
		return apperr.New(apperr.NotFound, "Karyawan atau tanggal tidak ditemukan")
	}
	// Like the 17:30 job: a day with nothing recorded gets no recap.
	items, err := r.buildDaily(ctx, e.ID, date)
	if err != nil || items.Empty() {
		return err
	}
	_, err = r.generateDaily(ctx, e, date)
	return err
}

// DailyView is a daily recap with names.
type DailyView struct {
	domain.DailyRecap
	EmployeeName   string
	DepartmentName string
	Saved          bool
}

func (r *Recaps) myEmployee(ctx context.Context) (*Actor, domain.Employee, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, domain.Employee{}, err
	}
	if a.Employee == nil {
		return nil, domain.Employee{}, errNoEmployee
	}
	return a, *a.Employee, nil
}

func (r *Recaps) recapDate(date string) (string, error) {
	if date == "" {
		return r.today(), nil
	}
	if !validDay(date) || date > r.today() {
		return "", apperr.Field("date", "Tanggal rekap tidak valid")
	}
	return date, nil
}

// MyDaily returns the stored recap, or a live preview of the day without
// a summary when none was generated yet.
func (r *Recaps) MyDaily(ctx context.Context, date string) (*DailyView, error) {
	_, e, err := r.myEmployee(ctx)
	if err != nil {
		return nil, err
	}
	if date, err = r.recapDate(date); err != nil {
		return nil, err
	}
	cur, err := r.Store.GetDailyRecap(ctx, e.ID, date)
	if err != nil {
		return nil, err
	}
	v := &DailyView{EmployeeName: e.Name, DepartmentName: e.DepartmentName}
	if cur != nil && cur.GeneratedAt != nil {
		v.DailyRecap, v.Saved = *cur, true
		return v, nil
	}
	items, err := r.buildDaily(ctx, e.ID, date)
	if err != nil {
		return nil, err
	}
	v.DailyRecap = domain.DailyRecap{EmployeeID: e.ID, DepartmentID: e.DepartmentID, Date: date, Items: items}
	if cur != nil {
		v.Notes, v.ID, v.Saved = cur.Notes, cur.ID, true
	}
	return v, nil
}

// UpdateMyNotes stores the free-text "catatan hari ini".
func (r *Recaps) UpdateMyNotes(ctx context.Context, date, notes string) (*DailyView, error) {
	_, e, err := r.myEmployee(ctx)
	if err != nil {
		return nil, err
	}
	if date, err = r.recapDate(date); err != nil {
		return nil, err
	}
	notes = strings.TrimSpace(notes)
	if len([]rune(notes)) > maxDailyNotes {
		return nil, apperr.Field("notes", "Catatan maksimal 4000 karakter")
	}
	if err := r.Store.SetDailyNotes(ctx, domain.DailyRecap{EmployeeID: e.ID, DepartmentID: e.DepartmentID, Date: date, Notes: notes}); err != nil {
		return nil, err
	}
	return r.MyDaily(ctx, date)
}

func (r *Recaps) RegenerateMyDaily(ctx context.Context, date string) (*DailyView, error) {
	_, e, err := r.myEmployee(ctx)
	if err != nil {
		return nil, err
	}
	if date, err = r.recapDate(date); err != nil {
		return nil, err
	}
	rec, err := r.generateDaily(ctx, e, date)
	if err != nil {
		return nil, err
	}
	return &DailyView{DailyRecap: *rec, EmployeeName: e.Name, DepartmentName: e.DepartmentName, Saved: true}, nil
}

// teamOf lists the employees the caller sees (HR: everyone), optionally
// one department.
func teamOf(a *Actor, ppl *people, departmentID string) []domain.Employee {
	var out []domain.Employee
	for _, e := range ppl.active() {
		if departmentID != "" && !e.InDepartment(departmentID) {
			continue
		}
		if a.HRRead || (a.Leads(e) && !a.IsEmployee(e.ID)) {
			out = append(out, e)
		}
	}
	return out
}

// TeamDaily lists the team's recaps of a day; members without one show up
// unsaved so the page can show who recorded nothing.
func (r *Recaps) TeamDaily(ctx context.Context, date, departmentID string) ([]DailyView, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, err
	}
	if date, err = r.recapDate(date); err != nil {
		return nil, err
	}
	if !a.HRRead && !a.IsLead() {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return nil, err
	}
	team := teamOf(a, ppl, departmentID)
	if len(team) == 0 {
		return []DailyView{}, nil
	}
	ids := make([]string, len(team))
	for i, e := range team {
		ids[i] = e.ID
	}
	rows, err := r.Store.ListDailyRecaps(ctx, domain.DailyRecapFilter{Date: date, EmployeeIDs: ids, Limit: 500})
	if err != nil {
		return nil, err
	}
	byEmp := map[string]domain.DailyRecap{}
	for _, rec := range rows {
		byEmp[rec.EmployeeID] = rec
	}
	out := make([]DailyView, len(team))
	for i, e := range team {
		rec, ok := byEmp[e.ID]
		if !ok {
			rec = domain.DailyRecap{EmployeeID: e.ID, DepartmentID: e.DepartmentID, Date: date,
				Items: domain.DailyRecapItems{Tasks: []domain.RecapTask{}, Logbook: []domain.RecapLogbook{}}}
		}
		out[i] = DailyView{DailyRecap: rec, EmployeeName: e.Name, DepartmentName: ppl.deptName(e.DepartmentID), Saved: ok}
	}
	return out, nil
}

// History lists an employee's recaps (own, or a team member's).
func (r *Recaps) History(ctx context.Context, employeeID, from, to string, limit int) ([]DailyView, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, err
	}
	if employeeID == "" || employeeID == "me" {
		employeeID = a.EmployeeID()
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return nil, err
	}
	e, ok := ppl.employee(employeeID)
	if !ok {
		return nil, apperr.New(apperr.NotFound, msgEmployeeGone)
	}
	if !a.SeesEmployee(e) {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	if limit <= 0 || limit > 90 {
		limit = 30
	}
	rows, err := r.Store.ListDailyRecaps(ctx, domain.DailyRecapFilter{EmployeeIDs: []string{employeeID}, From: from, To: to, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]DailyView, len(rows))
	for i, rec := range rows {
		out[i] = DailyView{DailyRecap: rec, EmployeeName: e.Name, DepartmentName: ppl.deptName(e.DepartmentID), Saved: true}
	}
	return out, nil
}

// RunDaily is the 17:30 WIB automation: once per workday it stores every
// active employee's recap (those with activity or notes) and sends each
// lead a team digest.
func (r *Recaps) RunDaily(ctx context.Context, now time.Time) (int, error) {
	local := now.In(domain.Jakarta)
	if !domain.IsWorkday(local) || !domain.ReminderDue(now, domain.DailyRecapHour, domain.DailyRecapMinute, domain.Jakarta) {
		return 0, nil
	}
	today := domain.DayKey(now, domain.Jakarta)
	claimed, err := r.Store.ClaimSend(ctx, "daily_recap", "all", today)
	if err != nil || !claimed {
		return 0, err
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return 0, err
	}
	type tally struct{ team, active, completed int }
	stats := map[string]*tally{}
	for _, e := range ppl.active() {
		for _, lead := range ppl.leadsOf(e) {
			if stats[lead] == nil {
				stats[lead] = &tally{}
			}
			stats[lead].team++
		}
	}
	generated := 0
	for _, e := range ppl.active() {
		items, err := r.buildDaily(ctx, e.ID, today)
		if err != nil {
			return generated, err
		}
		cur, err := r.Store.GetDailyRecap(ctx, e.ID, today)
		if err != nil {
			return generated, err
		}
		if items.Empty() && (cur == nil || cur.Notes == "") {
			continue
		}
		if _, err := r.generateDaily(ctx, e, today); err != nil {
			r.log().Warn("daily recap failed", "employee", e.ID, "err", err)
			continue
		}
		generated++
		for _, lead := range ppl.leadsOf(e) {
			stats[lead].active++
			stats[lead].completed += items.CompletedCount()
		}
	}
	for lead, t := range stats {
		r.notify(ctx, Notice{UserIDs: ppl.userIDs(lead), Type: NotifDailyRecap, Href: hrefRecaps,
			Title: "Rekap harian tim " + today,
			Body:  fmt.Sprintf("%d dari %d anggota tim mencatat aktivitas, %d tugas selesai hari ini.", t.active, t.team, t.completed)})
	}
	return generated, nil
}

// ─── Monthly ─────────────────────────────────────────────────

// monthlyFor computes metrics, accomplishments and the context signals
// for an employee (subject employee id) or a department.
func (r *Recaps) monthlyFor(ctx context.Context, ppl *people, scope, subjectID string, p domain.Period) (*domain.MonthlyRecap, domain.MonthlyRecapInput, error) {
	in := domain.MonthlyRecapInput{Scope: scope, Period: p}
	rec := &domain.MonthlyRecap{Scope: scope, SubjectID: subjectID, Period: p}
	var members []domain.Employee
	switch scope {
	case domain.ScopeEmployee:
		e, ok := ppl.employee(subjectID)
		if !ok {
			return nil, in, apperr.New(apperr.NotFound, msgEmployeeGone)
		}
		members, rec.DepartmentID = []domain.Employee{e}, e.DepartmentID
		in.SubjectName, in.DepartmentName = e.Name, ppl.deptName(e.DepartmentID)
	case domain.ScopeDepartment:
		d, ok := ppl.depts[subjectID]
		if !ok {
			return nil, in, apperr.New(apperr.NotFound, msgDeptNotFound)
		}
		members, rec.DepartmentID = ppl.members(subjectID), &d.ID
		in.SubjectName, in.DepartmentName = d.Name, d.Name
		rec.Metrics.Members = len(members)
	default:
		return nil, in, apperr.Field("scope", "Cakupan harus karyawan atau departemen")
	}
	ids := make([]string, len(members))
	for i, m := range members {
		ids[i] = m.ID
	}
	span := p.Span()
	upTo := span.End
	if today := r.today(); today < upTo {
		upTo = today
	}
	m := &rec.Metrics
	var all []domain.MonthTask
	if len(ids) > 0 {
		tasks, err := r.Store.MonthTasks(ctx, ids, span)
		if err != nil {
			return nil, in, err
		}
		for _, id := range ids {
			all = append(all, tasks[id]...)
		}
		m.TasksPlanned, m.TasksDone, m.TasksOnTime, m.TasksOverdue = domain.MonthlyTaskStats(all, upTo)
		m.CompletionRate, m.OnTimeRate = domain.Ratio(m.TasksDone, m.TasksPlanned), domain.Ratio(m.TasksOnTime, m.TasksDone)
		cards, err := r.Store.ListScorecards(ctx, p, ids)
		if err != nil {
			return nil, in, err
		}
		sum, n := 0.0, 0
		for _, c := range cards {
			if c.Score != nil {
				sum += *c.Score
				n++
			}
		}
		if n > 0 {
			v := domain.Round2(sum / float64(n))
			m.KPIScore = &v
		}
		stats, err := r.Store.LogbookStats(ctx, span)
		if err != nil {
			return nil, in, err
		}
		checked, total := 0, 0
		for _, id := range ids {
			s := stats[id]
			m.LogbookEntries += s.Entries
			checked, total = checked+s.Checked, total+s.Total
		}
		m.LogbookCompliance = domain.Ratio(checked, total)
		if att, err := r.Directory.Attendance(ctx, ids, p); err != nil {
			r.log().Warn("monthly recap: attendance unavailable", "period", p.Key(), "err", err)
		} else if len(att) > 0 {
			s := &domain.AttendanceSummary{}
			for _, a := range att {
				s.WorkingDays += a.WorkingDays
				s.PresentDays += a.PresentDays
				s.LateDays += a.LateDays
				s.AbsentDays += a.AbsentDays
				s.LeaveDays += a.LeaveDays
				s.OvertimeHrs += a.OvertimeHours
			}
			m.Attendance = s
		}
	}
	rec.Accomplishments = domain.TopAccomplishments(all, accomplishmentTop)
	f := domain.ImprovementFilter{Statuses: []string{domain.ImpOpen, domain.ImpInProgress}}
	if scope == domain.ScopeEmployee {
		f.EmployeeIDs = ids
	} else {
		f.DepartmentIDs = []string{subjectID}
	}
	open, err := r.Store.ListImprovements(ctx, f)
	if err != nil {
		return nil, in, err
	}
	m.ImprovementsOpen = len(open)
	for _, imp := range open {
		if len(in.Signals) < monthlySignals/2 {
			in.Signals = append(in.Signals, "Perbaikan berjalan: "+imp.Title)
		}
	}
	if len(ids) > 0 {
		daily, err := r.Store.ListDailyRecaps(ctx, domain.DailyRecapFilter{EmployeeIDs: ids, From: span.Start, To: span.End, Limit: 200})
		if err != nil {
			return nil, in, err
		}
		for _, d := range daily {
			if d.Notes != "" && len(in.Signals) < monthlySignals {
				in.Signals = append(in.Signals, "Catatan "+d.Date+": "+d.Notes)
			}
		}
	}
	in.Metrics, in.Accomplishments = rec.Metrics, rec.Accomplishments
	return rec, in, nil
}

// generateMonthly stores the recap with its narrative; with suggest the AI
// proposals become suggested improvement items.
func (r *Recaps) generateMonthly(ctx context.Context, ppl *people, scope, subjectID string, p domain.Period, suggest bool) (*domain.MonthlyRecap, int, error) {
	rec, in, err := r.monthlyFor(ctx, ppl, scope, subjectID, p)
	if err != nil {
		return nil, 0, err
	}
	if rec.Narrative, rec.NarrativeModel, err = r.AI.MonthlyNarrative(ctx, in); err != nil {
		return nil, 0, err
	}
	if rec.ID, err = r.Store.UpsertMonthlyRecap(ctx, *rec); err != nil {
		return nil, 0, err
	}
	filed := 0
	if suggest {
		suggestions, _, err := r.AI.SuggestImprovements(ctx, in)
		if err != nil {
			r.log().Warn("improvement suggestions failed", "recap", rec.ID, "err", err)
		} else {
			filed = r.improvements.fromSuggestions(ctx, *rec, suggestions)
		}
	}
	stored, err := r.Store.GetMonthlyRecap(ctx, rec.ID)
	return stored, filed, err
}

// MonthlyView is a monthly recap with names.
type MonthlyView struct {
	domain.MonthlyRecap
	SubjectName    string
	DepartmentName string
}

func (r *Recaps) monthlyView(ppl *people, rec domain.MonthlyRecap) MonthlyView {
	v := MonthlyView{MonthlyRecap: rec, DepartmentName: ppl.deptName(rec.DepartmentID)}
	if rec.Scope == domain.ScopeEmployee {
		v.SubjectName = ppl.name(&rec.SubjectID)
	} else {
		v.SubjectName = ppl.depts[rec.SubjectID].Name
	}
	return v
}

func (r *Recaps) canSeeMonthly(a *Actor, ppl *people, rec domain.MonthlyRecap) bool {
	if rec.Scope == domain.ScopeEmployee {
		e, ok := ppl.employee(rec.SubjectID)
		return (ok && a.SeesEmployee(e)) || a.HRRead
	}
	return a.SeesDepartment(rec.SubjectID) || a.DepartmentID() == rec.SubjectID
}

// MonthlyQuery filters the monthly list.
type MonthlyQuery struct {
	Period       *domain.Period
	Scope        string
	EmployeeID   string
	DepartmentID string
}

func (r *Recaps) ListMonthly(ctx context.Context, q MonthlyQuery) ([]MonthlyView, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, err
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return nil, err
	}
	f := domain.MonthlyRecapFilter{Period: q.Period, Scope: q.Scope, DepartmentID: q.DepartmentID}
	if q.EmployeeID != "" {
		f.Scope, f.SubjectIDs = domain.ScopeEmployee, []string{q.EmployeeID}
	}
	rows, err := r.Store.ListMonthlyRecaps(ctx, f)
	if err != nil {
		return nil, err
	}
	out := []MonthlyView{}
	for _, rec := range rows {
		if r.canSeeMonthly(a, ppl, rec) {
			out = append(out, r.monthlyView(ppl, rec))
		}
	}
	return out, nil
}

func (r *Recaps) GetMonthly(ctx context.Context, id string) (*MonthlyView, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, err
	}
	rec, err := r.Store.GetMonthlyRecap(ctx, id)
	if err != nil {
		return nil, err
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return nil, err
	}
	if rec == nil || !r.canSeeMonthly(a, ppl, *rec) {
		return nil, apperr.New(apperr.NotFound, msgRecapGone)
	}
	v := r.monthlyView(ppl, *rec)
	return &v, nil
}

// RegenerateMonthly rebuilds one recap: the employee or a lead for an
// employee recap, the head or HR writers for a department recap.
func (r *Recaps) RegenerateMonthly(ctx context.Context, scope, subjectID string, p domain.Period) (*MonthlyView, int, error) {
	a, err := r.actor(ctx)
	if err != nil {
		return nil, 0, err
	}
	if !p.Valid() {
		return nil, 0, apperr.New(apperr.Validation, msgPeriodInvalid)
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return nil, 0, err
	}
	switch scope {
	case domain.ScopeEmployee:
		if subjectID == "" || subjectID == "me" {
			subjectID = a.EmployeeID()
		}
		e, ok := ppl.employee(subjectID)
		if !ok {
			return nil, 0, apperr.New(apperr.NotFound, msgEmployeeGone)
		}
		if !a.HRWrite && !a.IsEmployee(e.ID) && !a.Leads(e) {
			return nil, 0, apperr.New(apperr.Unauthorized, "")
		}
	case domain.ScopeDepartment:
		if !a.HRWrite && !a.HeadOf[subjectID] {
			return nil, 0, apperr.New(apperr.Unauthorized, "")
		}
	}
	rec, filed, err := r.generateMonthly(ctx, ppl, scope, subjectID, p, true)
	if err != nil {
		return nil, 0, err
	}
	v := r.monthlyView(ppl, *rec)
	return &v, filed, nil
}

// ExportMonthly renders the recap as PDF; returns the file name and bytes.
func (r *Recaps) ExportMonthly(ctx context.Context, id string) (string, []byte, error) {
	v, err := r.GetMonthly(ctx, id)
	if err != nil {
		return "", nil, err
	}
	data, err := r.RenderPDF(v.MonthlyRecap, v.SubjectName, v.DepartmentName)
	if err != nil {
		return "", nil, err
	}
	slug := strings.ToLower(strings.Join(strings.Fields(v.SubjectName), "-"))
	return fmt.Sprintf("rekap-%s-%s.pdf", slug, v.Period.Key()), data, nil
}

// RunMonthly is the 1st-of-month automation (daily tick): recaps of the
// previous month for every active employee and department, AI improvement
// suggestions, and a notice to each person.
func (r *Recaps) RunMonthly(ctx context.Context, now time.Time) (int, error) {
	local := now.In(domain.Jakarta)
	if local.Day() != 1 {
		return 0, nil
	}
	prev := domain.Period{Year: local.Year(), Month: int(local.Month())}.Previous()
	claimed, err := r.Store.ClaimSend(ctx, "monthly_recap", prev.Key(), domain.DayKey(now, domain.Jakarta))
	if err != nil || !claimed {
		return 0, err
	}
	ppl, err := r.people(ctx)
	if err != nil {
		return 0, err
	}
	made := 0
	for _, e := range ppl.active() {
		rec, _, err := r.generateMonthly(ctx, ppl, domain.ScopeEmployee, e.ID, prev, true)
		if err != nil {
			r.log().Warn("monthly recap failed", "employee", e.ID, "err", err)
			continue
		}
		made++
		r.notify(ctx, Notice{UserIDs: ppl.userIDs(e.ID), Type: NotifMonthlyRecap, Href: hrefRecaps + "?recap=" + rec.ID,
			Title: "Rekap kerja " + prev.Label() + " siap", Body: "Lihat ringkasan dan saran perbaikan bulan lalu."})
	}
	for id, d := range ppl.depts {
		if !d.Active {
			continue
		}
		rec, _, err := r.generateMonthly(ctx, ppl, domain.ScopeDepartment, id, prev, true)
		if err != nil {
			r.log().Warn("monthly department recap failed", "department", id, "err", err)
			continue
		}
		made++
		if d.HeadEmployeeID != nil {
			r.notify(ctx, Notice{UserIDs: ppl.userIDs(*d.HeadEmployeeID), Type: NotifMonthlyRecap, Href: hrefRecaps + "?recap=" + rec.ID,
				Title: "Rekap departemen " + d.Name + " " + prev.Label() + " siap", Body: "Saatnya retrospektif bulanan bersama tim."})
		}
	}
	return made, nil
}
