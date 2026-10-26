package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
)

// MyDay is one calendar day of the personal dashboard.
type MyDay struct {
	Date, Status, ClockIn, ClockOut, Note string
	WorkHours, OvertimeHours              float64
	LateMinutes                           int
}

// MyStats sums one month of the caller's attendance.
type MyStats struct {
	ScheduledDays, PresentDays, LateDays, AbsentDays, LeaveDays int
	OnTimeRate, AttendanceRate, WorkHours, OvertimeHours        float64
	LateMinutes                                                 int
	AvgClockIn                                                  string
}

type MyMonth struct {
	Month                                      string
	OnTimeRate, TeamOnTimeRate                 float64
	WorkHours, OvertimeHours, NetPay, GrossPay float64
	PresentDays, LateDays                      int
	LeaveDays                                  float64
}

type MyWeek struct {
	WeekStart         string
	WorkHours, Target float64
	Days              int
}

type MyPending struct {
	Kind, ID, Type, Date, EndDate, Amount, Status string
	CreatedAt                                     time.Time
}

type MyTeamMember struct {
	EmployeeID, FullName, PositionName, Status, ClockIn, Note string
}

type MyDashboard struct {
	Employee      domain.Employee
	Month, Today  string
	Days          []MyDay
	Stats, Prev   MyStats
	Trend         []MyMonth
	Weeks         []MyWeek
	Balance       *Balance
	LeaveHistory  []domain.Leave
	LatestPayslip *domain.Payslip
	Pending       []MyPending
	IsManager     bool
	Team          []MyTeamMember
}

// myRecords is the caller's history the month views are cut from.
type myRecords struct {
	schedule []domain.ScheduleRow
	off      map[string]string // non-deducting holiday date -> name
	att      map[string]domain.Attendance
	leave    map[string]string // date -> leave type
	overtime map[string]float64
	today    string
}

// day classifies one date for the calendar.
func (r *myRecords) day(date string) MyDay {
	d := MyDay{Date: date, OvertimeHours: r.overtime[date]}
	a, hasAtt := r.att[date]
	switch {
	case r.leave[date] != "":
		d.Status, d.Note = "leave", r.leave[date]
	case hasAtt:
		d.Status = a.Status
		if a.Status == "present" && a.IsLate {
			d.Status = "late"
		}
		if a.Status == "half_day" {
			d.Status = "present"
		}
		d.ClockIn, d.ClockOut = clockWIB(a.ClockIn), clockWIB(a.ClockOut)
		d.WorkHours, d.LateMinutes = a.WorkHours, a.LateMinutes
	case r.off[date] != "":
		d.Status, d.Note = "holiday", r.off[date]
	case ShiftOn(r.schedule, date) == nil:
		d.Status = "off"
	case date >= r.today:
		d.Status = "upcoming"
	default:
		d.Status = "missing"
	}
	return d
}

func clockWIB(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Add(hris.WIBOffset).Format("15:04")
}

// stats sums [from, to] (days after today are left out).
func (r *myRecords) stats(from, to string) MyStats {
	var s MyStats
	clockSum, clocked := 0, 0
	for _, date := range hris.EachDateISO(from, min(to, r.today)) {
		d := r.day(date)
		s.OvertimeHours += d.OvertimeHours
		switch d.Status {
		case "off", "holiday", "upcoming":
			continue
		case "leave":
			s.LeaveDays++
			s.ScheduledDays++
			continue
		}
		s.ScheduledDays++
		switch d.Status {
		case "absent", "missing":
			if date < r.today {
				s.AbsentDays++
			} else {
				s.ScheduledDays--
			}
			continue
		}
		s.PresentDays++
		if d.Status == "late" {
			s.LateDays++
		}
		s.WorkHours += d.WorkHours
		s.LateMinutes += d.LateMinutes
		if a := r.att[date]; a.ClockIn != nil {
			in := a.ClockIn.UTC().Add(hris.WIBOffset)
			clockSum += in.Hour()*60 + in.Minute()
			clocked++
		}
	}
	s.OnTimeRate = pct(float64(s.PresentDays-s.LateDays), float64(s.PresentDays))
	s.AttendanceRate = pct(float64(s.PresentDays), float64(s.ScheduledDays-s.LeaveDays))
	s.WorkHours, s.OvertimeHours = round2(s.WorkHours), round2(s.OvertimeHours)
	if clocked > 0 {
		avg := clockSum / clocked
		s.AvgClockIn = fmt.Sprintf("%02d:%02d", avg/60, avg%60)
	}
	return s
}

// Me is the caller's dashboard for month ("YYYY-MM", "" = this month).
func (x *Dashboard) Me(ctx context.Context, a Actor, month string) (*MyDashboard, error) {
	id, err := self(a)
	if err != nil {
		return nil, err
	}
	today := x.d.today()
	if strings.TrimSpace(month) == "" {
		month = today[:7]
	}
	mStart, mEnd, ok := monthBounds(strings.TrimSpace(month))
	if !ok {
		return nil, invalid("month", "Bulan tidak valid (YYYY-MM).")
	}
	emp, err := (&People{d: x.d}).Get(ctx, id)
	if err != nil {
		return nil, err
	}
	trendFrom := addMonths(mStart, -11)
	prevStart := addMonths(mStart, -1)
	r, err := x.myRecords(ctx, id, trendFrom, mEnd, today)
	if err != nil {
		return nil, err
	}
	out := &MyDashboard{Employee: *emp, Month: mStart[:7], Today: today}
	for _, d := range hris.EachDateISO(mStart, mEnd) {
		out.Days = append(out.Days, r.day(d))
	}
	out.Stats = r.stats(mStart, mEnd)
	out.Prev = r.stats(prevStart, hris.AddDaysISO(mStart, -1))
	out.Weeks = r.weeks(mStart, mEnd)
	if out.Trend, err = x.myTrend(ctx, r, *emp, trendFrom, mEnd); err != nil {
		return nil, err
	}
	if out.Balance, err = x.leaves.Balance(ctx, a, id, yearOf(mStart)); err != nil {
		return nil, err
	}
	if out.LeaveHistory, _, err = x.d.Store.ListLeaves(ctx, domain.LeaveFilter{EmployeeIDs: []string{id}, Limit: 8}); err != nil {
		return nil, err
	}
	slips, err := x.d.Store.ListPayslips(ctx, domain.PayslipFilter{EmployeeID: id, OnlyPaid: true})
	if err != nil {
		return nil, err
	}
	for i, p := range slips {
		if fmt.Sprintf("%04d-%02d", p.PeriodYear, p.PeriodMonth) <= mStart[:7] &&
			(out.LatestPayslip == nil || p.PeriodYear*100+p.PeriodMonth > out.LatestPayslip.PeriodYear*100+out.LatestPayslip.PeriodMonth) {
			out.LatestPayslip = &slips[i]
		}
	}
	if out.Pending, err = x.myPending(ctx, id); err != nil {
		return nil, err
	}
	if out.Team, err = x.teamToday(ctx, id, today); err != nil {
		return nil, err
	}
	out.IsManager = len(out.Team) > 0
	return out, nil
}

func (x *Dashboard) myRecords(ctx context.Context, id, from, to, today string) (*myRecords, error) {
	r := &myRecords{off: map[string]string{}, att: map[string]domain.Attendance{}, leave: map[string]string{}, overtime: map[string]float64{}, today: today}
	var err error
	if r.schedule, err = x.d.Store.ListScheduleRows(ctx, []string{id}); err != nil {
		return nil, err
	}
	holidays, err := x.d.Store.ListHolidays(ctx, from, to, true)
	if err != nil {
		return nil, err
	}
	for _, h := range holidays {
		if !h.DeductsLeave {
			r.off[h.Date] = h.Name
		}
	}
	atts, _, err := x.d.Store.ListAttendance(ctx, domain.AttendanceFilter{EmployeeIDs: []string{id}, From: from, To: to})
	if err != nil {
		return nil, err
	}
	for _, at := range atts {
		r.att[at.Date] = at
	}
	leaves, _, err := x.d.Store.ListLeaves(ctx, domain.LeaveFilter{EmployeeIDs: []string{id}, Status: hris.LeaveApproved, From: from, To: to})
	if err != nil {
		return nil, err
	}
	for _, l := range leaves {
		for _, d := range hris.EachDateISO(max(l.StartDate, from), min(l.EndDate, to)) {
			if !hris.IsWeekend(d) && r.off[d] == "" {
				r.leave[d] = l.LeaveType
			}
		}
	}
	ots, err := x.d.Store.OvertimeForPeriod(ctx, id, from, to)
	if err != nil {
		return nil, err
	}
	for _, o := range ots {
		r.overtime[o.Date] += o.Hours
	}
	return r, nil
}

// weeks are the Monday-based weeks touching the month, counting only the
// month's days; target is the scheduled shift hours of those days.
func (r *myRecords) weeks(mStart, mEnd string) []MyWeek {
	var out []MyWeek
	for w := weekStart(mStart); w <= mEnd; w = hris.AddDaysISO(w, 7) {
		wk := MyWeek{WeekStart: w}
		for i := 0; i < 7; i++ {
			d := hris.AddDaysISO(w, i)
			if d < mStart || d > mEnd {
				continue
			}
			if sh := ShiftOn(r.schedule, d); sh != nil && r.off[d] == "" && r.leave[d] == "" {
				wk.Target += shiftHours(*sh)
			}
			if a, ok := r.att[d]; ok && a.Status != "absent" {
				wk.WorkHours += a.WorkHours
				wk.Days++
			}
		}
		wk.WorkHours, wk.Target = round2(wk.WorkHours), round2(wk.Target)
		out = append(out, wk)
	}
	return out
}

// shiftHours is the shift length minus the break.
func shiftHours(sh domain.Shift) float64 {
	start, err1 := time.Parse("15:04", domain.HHMM(sh.StartTime))
	end, err2 := time.Parse("15:04", domain.HHMM(sh.EndTime))
	if err1 != nil || err2 != nil {
		return 0
	}
	d := end.Sub(start)
	if d <= 0 {
		d += 24 * time.Hour
	}
	return d.Hours() - float64(sh.BreakMinutes)/60
}

// myTrend is 12 months of the caller's on-time rate against the
// department's, hours, overtime, leave and pay.
func (x *Dashboard) myTrend(ctx context.Context, r *myRecords, emp domain.Employee, from, to string) ([]MyMonth, error) {
	team := map[string][2]int{} // month -> present, late
	if emp.DepartmentID != nil {
		rows, err := x.d.Store.DashAttendanceDaily(ctx, from, to, domain.DashFilter{DepartmentID: *emp.DepartmentID})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			m := team[firstOfMonth(row.Date)]
			team[firstOfMonth(row.Date)] = [2]int{m[0] + row.Present, m[1] + row.Late}
		}
	}
	slips, err := x.d.Store.ListPayslips(ctx, domain.PayslipFilter{EmployeeID: emp.ID, OnlyPaid: true})
	if err != nil {
		return nil, err
	}
	pay := map[string]domain.Payslip{}
	for _, p := range slips {
		pay[fmt.Sprintf("%04d-%02d-01", p.PeriodYear, p.PeriodMonth)] = p
	}
	var out []MyMonth
	for m := from; m <= to; m = addMonths(m, 1) {
		end := hris.AddDaysISO(addMonths(m, 1), -1)
		s := r.stats(m, end)
		t := team[m]
		p := pay[m]
		out = append(out, MyMonth{Month: m, OnTimeRate: s.OnTimeRate, TeamOnTimeRate: pct(float64(t[0]-t[1]), float64(t[0])),
			WorkHours: s.WorkHours, OvertimeHours: s.OvertimeHours, NetPay: p.NetSalary, GrossPay: p.GrossSalary,
			PresentDays: s.PresentDays, LateDays: s.LateDays, LeaveDays: float64(s.LeaveDays)})
	}
	return out, nil
}

func (x *Dashboard) myPending(ctx context.Context, id string) ([]MyPending, error) {
	var out []MyPending
	leaves, _, err := x.d.Store.ListLeaves(ctx, domain.LeaveFilter{EmployeeIDs: []string{id}, Statuses: []string{hris.LeavePending, hris.LeaveManagerApproved}})
	if err != nil {
		return nil, err
	}
	for _, l := range leaves {
		out = append(out, MyPending{Kind: "leave", ID: l.ID, Type: l.LeaveType, Date: l.StartDate, EndDate: l.EndDate,
			Amount: fmt.Sprint(l.TotalDays), Status: l.Status, CreatedAt: l.CreatedAt})
	}
	ots, err := x.d.Store.ListOvertime(ctx, domain.OvertimeFilter{EmployeeIDs: []string{id}, Status: "pending"})
	if err != nil {
		return nil, err
	}
	for _, o := range ots {
		out = append(out, MyPending{Kind: "overtime", ID: o.ID, Date: o.Date, Amount: fmt.Sprint(o.Hours), Status: o.Status, CreatedAt: o.CreatedAt})
	}
	loans, err := x.d.Store.ListLoans(ctx, id, "pending")
	if err != nil {
		return nil, err
	}
	for _, l := range loans {
		out = append(out, MyPending{Kind: "loan", ID: l.ID, Type: l.LoanType, Amount: domain.Money(l.PrincipalAmount), Status: l.Status, CreatedAt: l.CreatedAt})
	}
	return out, nil
}

// teamToday is where the caller's direct reports are today.
func (x *Dashboard) teamToday(ctx context.Context, managerID, today string) ([]MyTeamMember, error) {
	reports, _, err := x.d.Store.ListEmployees(ctx, domain.EmployeeFilter{ManagerID: managerID})
	if err != nil || len(reports) == 0 {
		return nil, err
	}
	ids := make([]string, len(reports))
	for i, e := range reports {
		ids[i] = e.ID
	}
	atts, _, err := x.d.Store.ListAttendance(ctx, domain.AttendanceFilter{EmployeeIDs: ids, From: today, To: today})
	if err != nil {
		return nil, err
	}
	byEmp := map[string]domain.Attendance{}
	for _, at := range atts {
		byEmp[at.EmployeeID] = at
	}
	onLeave, err := x.d.Store.ApprovedLeavesOn(ctx, today)
	if err != nil {
		return nil, err
	}
	rows, err := x.d.Store.ListScheduleRows(ctx, ids)
	if err != nil {
		return nil, err
	}
	holidays, err := x.d.Store.ListHolidays(ctx, today, today, true)
	if err != nil {
		return nil, err
	}
	holiday := false
	for _, h := range holidays {
		holiday = holiday || !h.DeductsLeave
	}
	out := make([]MyTeamMember, 0, len(reports))
	for _, e := range reports {
		m := MyTeamMember{EmployeeID: e.ID, FullName: e.FullName, PositionName: e.PositionName}
		var mine []domain.ScheduleRow
		for _, r := range rows {
			if r.EmployeeID == e.ID {
				mine = append(mine, r)
			}
		}
		a, hasAtt := byEmp[e.ID]
		switch {
		case onLeave[e.ID] != "":
			m.Status, m.Note = "leave", onLeave[e.ID]
		case hasAtt && a.Status == "absent":
			m.Status = "absent"
		case hasAtt:
			m.Status, m.ClockIn = "in", clockWIB(a.ClockIn)
			if a.IsLate {
				m.Status = "late"
			} else if a.Status == "remote" {
				m.Status = "remote"
			}
		case holiday:
			m.Status = "off"
		default:
			if sh := ShiftOn(mine, today); sh != nil {
				m.Status, m.Note = "not_in", domain.HHMM(sh.StartTime)
			} else {
				m.Status = "off"
			}
		}
		out = append(out, m)
	}
	return out, nil
}
