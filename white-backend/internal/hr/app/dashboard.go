package app

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// Dashboard builds the HR analytics page (hr.read, payroll for hr.payroll)
// and the employee's personal dashboard (self.service).
type Dashboard struct {
	d      Deps
	leaves *Leaves
}

func NewDashboard(d Deps, leaves *Leaves) *Dashboard { return &Dashboard{d: d, leaves: leaves} }

// DashboardQuery is the HR dashboard request.
type DashboardQuery struct {
	From, To, Compare string
	Filter            domain.DashFilter
}

// DashKpis are the headline numbers of one period.
type DashKpis struct {
	Headcount, Joiners, Leavers int
	TurnoverRate                float64
	AttendanceRecords           int
	AttendanceRate              float64
	LateCount                   int
	LateRate, AvgLateMinutes    float64
	AbsentCount                 int
	AvgWorkHours, LeaveDays     float64
	OvertimeHours, OvertimeCost float64
	PayrollGross, PayrollNet    float64
	PayrollEmployerCost         float64
	PayrollEmployees            int
}

// DashBucket is one day, week or month of a series.
type DashBucket struct {
	Date                                   string
	Headcount, Joiners, Leavers            int
	Records, Present, Late, Absent         int
	AttendanceRate, LateRate, AvgWork      float64
	LeaveDays, OvertimeHours, OvertimeCost float64
	TurnoverRate                           float64
	PayrollGross, PayrollNet, PayrollPph21 float64
	BpjsEmployee, BpjsEmployer             float64
	PayrollEmployees                       int
}

type DashDeptMonth struct {
	Month, DepartmentID         string
	Headcount, Records, Late    int
	AttendanceRate, LeaveDays   float64
	OvertimeHours, OvertimeCost float64
	PayrollGross                float64
}

type DashDept struct {
	DepartmentID, Name                string
	Headcount, Joiners, Leavers       int
	Records                           int
	AttendanceRate                    float64
	Late                              int
	LateRate                          float64
	LateMinutes, Absent               int
	AvgWorkHours, LeaveDays           float64
	OvertimeHours, OvertimeCost       float64
	PayrollGross, PayrollEmployerCost float64
	LeaveQuota, LeaveUsed             int
}

type DashSlice struct {
	Key, Label string
	Value      float64
	Count      int
}

type DashCell struct {
	Row          string
	Col          int
	Value, Total int
}

type DashTypeMonth struct {
	Month, Type string
	Days        float64
}

type DashEmployeeStat struct {
	EmployeeID, FullName, DepartmentName, PositionName string
	Records, Late, LateMinutes, Absent                 int
	AttendanceRate, OvertimeHours, OvertimeCost        float64
}

type DashPendingItem struct {
	domain.DashPending
	AgeDays int
}

type DashExpiry struct {
	Kind, EmployeeID, FullName, DepartmentName string
	ContractID, ContractNumber, ContractType   string
	EndDate                                    string
	DaysLeft                                   int
}

// HrDashboard is the whole HR dashboard payload.
type HrDashboard struct {
	From, To, CompareFrom, CompareTo, Bucket string
	PayrollVisible                           bool
	Current, Previous                        DashKpis
	Buckets, PreviousBuckets, Months         []DashBucket
	DeptMonths                               []DashDeptMonth
	Departments                              []DashDept
	LateByWeekday, LateByDay                 []DashCell
	AbsenceReasons, ClockInOffsets           []DashSlice
	TopLate, TopOvertime                     []DashEmployeeStat
	LeaveByType                              []DashSlice
	LeaveTypeMonths                          []DashTypeMonth
	PendingAging                             []DashSlice
	Pending                                  []DashPendingItem
	ByStatus, ByPosition, ByLevel            []DashSlice
	ByContractType, ByGender                 []DashSlice
	AgeBands, TenureBands                    []DashSlice
	Expiring                                 []DashExpiry
	SalaryByLevel                            []domain.DashLevel
	Loans                                    []domain.DashLoan
	LoansOutstanding                         float64
}

const maxDashboardDays = 3 * 366

// resolve picks the period, the comparison period and the bucket size.
func (q DashboardQuery) resolve(today string) (from, to, cFrom, cTo, bucket string, err error) {
	from, to = strings.TrimSpace(q.From), strings.TrimSpace(q.To)
	if from == "" && to == "" {
		from, to = today[:8]+"01", today
	}
	if _, ok := hris.ParseDate(from); !ok {
		return "", "", "", "", "", invalid("from", msgInvalidDate)
	}
	if _, ok := hris.ParseDate(to); !ok {
		return "", "", "", "", "", invalid("to", msgInvalidDate)
	}
	if from > to {
		return "", "", "", "", "", invalid("to", "Tanggal akhir harus setelah tanggal awal.")
	}
	n := len(hris.EachDateISO(from, to))
	if n > maxDashboardDays {
		return "", "", "", "", "", invalid("from", "Rentang maksimal 3 tahun.")
	}
	switch q.Compare {
	case "year":
		cFrom, cTo = shiftYear(from, -1), shiftYear(to, -1)
	default:
		cTo = hris.AddDaysISO(from, -1)
		cFrom = hris.AddDaysISO(cTo, -(n - 1))
	}
	switch {
	case n <= 45:
		bucket = "day"
	case n <= 180:
		bucket = "week"
	default:
		bucket = "month"
	}
	return from, to, cFrom, cTo, bucket, nil
}

func shiftYear(date string, years int) string {
	t, _ := hris.ParseDate(date)
	return t.AddDate(years, 0, 0).Format(hris.DateLayout)
}

// bucketOf is the first day of the bucket holding date.
func bucketOf(date, bucket string) string {
	switch bucket {
	case "week":
		return hris.AddDaysISO(date, 1-hris.IsoDayOfWeek(date))
	case "month":
		return date[:8] + "01"
	}
	return date
}

// bucketStarts lists the buckets covering [from, to].
func bucketStarts(from, to, bucket string) []string {
	var out []string
	for d := bucketOf(from, bucket); d <= to; d = nextBucket(d, bucket) {
		out = append(out, d)
	}
	return out
}

func nextBucket(start, bucket string) string {
	switch bucket {
	case "week":
		return hris.AddDaysISO(start, 7)
	case "month":
		t, _ := hris.ParseDate(start)
		return t.AddDate(0, 1, 0).Format(hris.DateLayout)
	}
	return hris.AddDaysISO(start, 1)
}

// bucketEnd is the last day of the bucket starting at start, capped at to.
func bucketEnd(start, bucket, to string) string {
	return min(hris.AddDaysISO(nextBucket(start, bucket), -1), to)
}

func pct(part, whole float64) float64 {
	if whole == 0 {
		return 0
	}
	return round2(part / whole * 100)
}

func round2(v float64) float64 {
	if v < 0 {
		return -float64(int64(-v*100+0.5)) / 100
	}
	return float64(int64(v*100+0.5)) / 100
}

// dashData is everything fetched once and sliced per period.
type dashData struct {
	emps     []domain.DashEmployee
	days     []domain.DashAttendanceDay
	leaveDay []leaveDay
	overtime []domain.DashOvertime
	payroll  []domain.DashPayroll
}

// leaveDay is one working day of approved leave.
type leaveDay struct{ Date, DepartmentID, Type, EmployeeID string }

// onBooks is whether e was employed on date.
func onBooks(e domain.DashEmployee, date string) bool {
	if e.JoinDate > date {
		return false
	}
	if e.EndDate != nil {
		return *e.EndDate >= date
	}
	return e.IsActive
}

func (dd *dashData) headcount(date, dept string) int {
	n := 0
	for _, e := range dd.emps {
		if (dept == "*" || e.DepartmentID == dept) && onBooks(e, date) {
			n++
		}
	}
	return n
}

func (dd *dashData) moves(from, to, dept string) (joiners, leavers int) {
	for _, e := range dd.emps {
		if dept != "*" && e.DepartmentID != dept {
			continue
		}
		if e.JoinDate >= from && e.JoinDate <= to {
			joiners++
		}
		if e.EndDate != nil && *e.EndDate >= from && *e.EndDate <= to {
			leavers++
		}
	}
	return joiners, leavers
}

// period sums [from, to] for one department ("*" = all) into a bucket.
func (dd *dashData) period(from, to, dept string) DashBucket {
	b := DashBucket{Date: from, Headcount: dd.headcount(to, dept)}
	b.Joiners, b.Leavers = dd.moves(from, to, dept)
	avgHead := float64(dd.headcount(from, dept)+b.Headcount) / 2
	b.TurnoverRate = pct(float64(b.Leavers), avgHead)
	var work float64
	for _, a := range dd.days {
		if a.Date < from || a.Date > to || (dept != "*" && a.DepartmentID != dept) {
			continue
		}
		b.Records += a.Records
		b.Present += a.Present
		b.Late += a.Late
		b.Absent += a.Absent
		work += a.WorkHours
	}
	b.AttendanceRate = pct(float64(b.Present), float64(b.Records))
	b.LateRate = pct(float64(b.Late), float64(b.Present))
	if b.Present > 0 {
		b.AvgWork = round2(work / float64(b.Present))
	}
	for _, l := range dd.leaveDay {
		if l.Date >= from && l.Date <= to && (dept == "*" || l.DepartmentID == dept) {
			b.LeaveDays++
		}
	}
	for _, o := range dd.overtime {
		if o.Date >= from && o.Date <= to && (dept == "*" || o.DepartmentID == dept) {
			b.OvertimeHours += o.Hours
			b.OvertimeCost += o.Cost
		}
	}
	b.OvertimeHours, b.OvertimeCost = round2(b.OvertimeHours), round2(b.OvertimeCost)
	for _, p := range dd.payroll {
		if p.Month >= from[:8]+"01" && p.Month <= to && (dept == "*" || p.DepartmentID == dept) {
			b.PayrollGross += p.Gross
			b.PayrollNet += p.Net
			b.PayrollPph21 += p.Pph21
			b.BpjsEmployee += p.BpjsEmployee
			b.BpjsEmployer += p.BpjsEmployer
			b.PayrollEmployees += p.Employees
		}
	}
	return b
}

func (dd *dashData) kpis(from, to string) DashKpis {
	b := dd.period(from, to, "*")
	k := DashKpis{Headcount: b.Headcount, Joiners: b.Joiners, Leavers: b.Leavers, TurnoverRate: b.TurnoverRate,
		AttendanceRecords: b.Records, AttendanceRate: b.AttendanceRate, LateCount: b.Late, LateRate: b.LateRate,
		AbsentCount: b.Absent, AvgWorkHours: b.AvgWork, LeaveDays: b.LeaveDays, OvertimeHours: b.OvertimeHours,
		OvertimeCost: b.OvertimeCost, PayrollGross: b.PayrollGross, PayrollNet: b.PayrollNet,
		PayrollEmployerCost: b.PayrollGross + b.BpjsEmployer, PayrollEmployees: b.PayrollEmployees}
	lateMinutes := 0
	for _, a := range dd.days {
		if a.Date >= from && a.Date <= to {
			lateMinutes += a.LateMinutes
		}
	}
	if b.Late > 0 {
		k.AvgLateMinutes = round2(float64(lateMinutes) / float64(b.Late))
	}
	return k
}

func (dd *dashData) series(from, to, bucket string) []DashBucket {
	starts := bucketStarts(from, to, bucket)
	out := make([]DashBucket, len(starts))
	for i, s := range starts {
		out[i] = dd.period(max(s, from), bucketEnd(s, bucket, to), "*")
		out[i].Date = s
	}
	return out
}

// HR is the HR dashboard for a period.
func (x *Dashboard) HR(ctx context.Context, a Actor, q DashboardQuery) (*HrDashboard, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, err
	}
	today := x.d.today()
	from, to, cFrom, cTo, bucket, err := q.resolve(today)
	if err != nil {
		return nil, err
	}
	f := q.Filter
	monthsFrom := min(firstOfMonth(from), addMonths(firstOfMonth(to), -11))
	lo := min(monthsFrom, cFrom)
	out := &HrDashboard{From: from, To: to, CompareFrom: cFrom, CompareTo: cTo, Bucket: bucket, PayrollVisible: a.Can(authn.PermPayroll)}

	dd := &dashData{}
	if dd.emps, err = x.d.Store.DashEmployees(ctx, f); err != nil {
		return nil, err
	}
	if dd.days, err = x.d.Store.DashAttendanceDaily(ctx, lo, to, f); err != nil {
		return nil, err
	}
	if dd.overtime, err = x.d.Store.DashOvertime(ctx, lo, to, f); err != nil {
		return nil, err
	}
	if dd.leaveDay, err = x.leaveDays(ctx, lo, to, f); err != nil {
		return nil, err
	}
	if out.PayrollVisible {
		if dd.payroll, err = x.d.Store.DashPayroll(ctx, firstOfMonth(lo), to, f); err != nil {
			return nil, err
		}
	}

	out.Current, out.Previous = dd.kpis(from, to), dd.kpis(cFrom, cTo)
	out.Buckets = dd.series(from, to, bucket)
	out.PreviousBuckets = dd.series(cFrom, cTo, bucket)
	out.Months = dd.series(monthsFrom, to, "month")

	depts, err := x.d.Store.ListDepartments(ctx, true)
	if err != nil {
		return nil, err
	}
	deptIDs := dashDeptIDs(dd, depts)
	for _, m := range out.Months {
		end := bucketEnd(m.Date, "month", to)
		for _, id := range deptIDs {
			p := dd.period(m.Date, end, id)
			out.DeptMonths = append(out.DeptMonths, DashDeptMonth{Month: m.Date, DepartmentID: id, Headcount: p.Headcount, Records: p.Records,
				Late: p.Late, AttendanceRate: p.AttendanceRate, LeaveDays: p.LeaveDays, OvertimeHours: p.OvertimeHours,
				OvertimeCost: p.OvertimeCost, PayrollGross: p.PayrollGross})
		}
	}
	balances, err := x.d.Store.DashLeaveBalances(ctx, yearOf(to), f)
	if err != nil {
		return nil, err
	}
	names := map[string]string{"": "Tanpa departemen"}
	for _, d := range depts {
		names[d.ID] = d.Name
	}
	for _, id := range deptIDs {
		p := dd.period(from, to, id)
		dept := DashDept{DepartmentID: id, Name: names[id], Headcount: p.Headcount, Joiners: p.Joiners, Leavers: p.Leavers,
			Records: p.Records, AttendanceRate: p.AttendanceRate, Late: p.Late, LateRate: p.LateRate, Absent: p.Absent,
			AvgWorkHours: p.AvgWork, LeaveDays: p.LeaveDays, OvertimeHours: p.OvertimeHours, OvertimeCost: p.OvertimeCost,
			PayrollGross: p.PayrollGross, PayrollEmployerCost: p.PayrollGross + p.BpjsEmployer}
		for _, a := range dd.days {
			if a.Date >= from && a.Date <= to && a.DepartmentID == id {
				dept.LateMinutes += a.LateMinutes
			}
		}
		for _, b := range balances {
			if b.DepartmentID == id {
				dept.LeaveQuota, dept.LeaveUsed = b.Quota, b.Used
			}
		}
		out.Departments = append(out.Departments, dept)
	}

	out.LateByWeekday, out.LateByDay = lateCells(dd.days, from, to)
	if out.ClockInOffsets, err = x.offsets(ctx, from, to, f); err != nil {
		return nil, err
	}
	out.LeaveByType, out.AbsenceReasons = leaveSlices(dd, from, to)
	for _, l := range dd.leaveDay {
		if l.Date >= monthsFrom && l.Date <= to {
			addTypeMonth(&out.LeaveTypeMonths, l.Date[:8]+"01", l.Type)
		}
	}
	if out.TopLate, err = x.topLate(ctx, from, to, f); err != nil {
		return nil, err
	}
	out.TopOvertime = topOvertime(dd.overtime, from, to)
	if err := x.pending(ctx, out, f, today); err != nil {
		return nil, err
	}
	workforceSlices(out, dd.emps, to)
	if out.Expiring, err = x.expiring(ctx, dd.emps, f, today); err != nil {
		return nil, err
	}
	if out.PayrollVisible {
		if err := x.payrollExtras(ctx, out, dd, f, to); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func firstOfMonth(date string) string { return date[:8] + "01" }

func addMonths(date string, n int) string {
	t, _ := hris.ParseDate(date)
	return t.AddDate(0, n, 0).Format(hris.DateLayout)
}

// dashDeptIDs are the departments with anyone on the books or any record,
// ordered by name with "no department" last.
func dashDeptIDs(dd *dashData, depts []domain.Department) []string {
	seen := map[string]bool{}
	for _, e := range dd.emps {
		seen[e.DepartmentID] = true
	}
	var out []string
	for _, d := range depts {
		if seen[d.ID] {
			out = append(out, d.ID)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return deptName(depts, out[i]) < deptName(depts, out[j]) })
	if seen[""] {
		out = append(out, "")
	}
	return out
}

func deptName(depts []domain.Department, id string) string {
	for _, d := range depts {
		if d.ID == id {
			return d.Name
		}
	}
	return ""
}

// leaveDays expands approved leave into working days (weekdays that are
// not a non-deducting holiday) inside [from, to].
func (x *Dashboard) leaveDays(ctx context.Context, from, to string, f domain.DashFilter) ([]leaveDay, error) {
	rows, err := x.d.Store.DashLeaves(ctx, from, to, f)
	if err != nil {
		return nil, err
	}
	holidays, err := x.d.Store.ListHolidays(ctx, from, to, true)
	if err != nil {
		return nil, err
	}
	off := map[string]bool{}
	for _, h := range holidays {
		if !h.DeductsLeave {
			off[h.Date] = true
		}
	}
	var out []leaveDay
	for _, l := range rows {
		for _, d := range hris.EachDateISO(max(l.StartDate, from), min(l.EndDate, to)) {
			if !hris.IsWeekend(d) && !off[d] {
				out = append(out, leaveDay{Date: d, DepartmentID: l.DepartmentID, Type: l.LeaveType, EmployeeID: l.EmployeeID})
			}
		}
	}
	return out, nil
}

// lateCells counts late arrivals by department x weekday and by day of
// month over [from, to]; total is the attendance records of the cell.
func lateCells(days []domain.DashAttendanceDay, from, to string) (byWeekday, byDay []DashCell) {
	type cellKey struct {
		dept string
		dow  int
	}
	wk := map[cellKey]*DashCell{}
	dom := map[int]*DashCell{}
	for _, a := range days {
		if a.Date < from || a.Date > to {
			continue
		}
		dow := hris.IsoDayOfWeek(a.Date)
		key := cellKey{a.DepartmentID, dow}
		c, ok := wk[key]
		if !ok {
			c = &DashCell{Row: a.DepartmentID, Col: dow}
			wk[key] = c
		}
		c.Value += a.Late
		c.Total += a.Records
		t, _ := hris.ParseDate(a.Date)
		dc, ok := dom[t.Day()]
		if !ok {
			dc = &DashCell{Col: t.Day()}
			dom[t.Day()] = dc
		}
		dc.Value += a.Late
		dc.Total += a.Records
	}
	for _, c := range wk {
		byWeekday = append(byWeekday, *c)
	}
	sort.Slice(byWeekday, func(i, j int) bool {
		if byWeekday[i].Row != byWeekday[j].Row {
			return byWeekday[i].Row < byWeekday[j].Row
		}
		return byWeekday[i].Col < byWeekday[j].Col
	})
	for d := 1; d <= 31; d++ {
		if c, ok := dom[d]; ok {
			byDay = append(byDay, *c)
		}
	}
	return byWeekday, byDay
}

// offsets bins clock-ins by minutes from the shift start, folding the
// tails into the first (-30) and last (60) bins.
func (x *Dashboard) offsets(ctx context.Context, from, to string, f domain.DashFilter) ([]DashSlice, error) {
	rows, err := x.d.Store.DashClockInOffsets(ctx, from, to, f)
	if err != nil {
		return nil, err
	}
	counts := map[int]int{}
	for _, r := range rows {
		counts[min(max(r.Bin, -30), 60)] += r.Records
	}
	var out []DashSlice
	for bin := -30; bin <= 60; bin += 5 {
		out = append(out, DashSlice{Key: strconv.Itoa(bin), Value: float64(counts[bin]), Count: counts[bin]})
	}
	return out, nil
}

// leaveSlices are leave days by type, and absence reasons (absent records
// plus leave days by type) over [from, to].
func leaveSlices(dd *dashData, from, to string) (byType, reasons []DashSlice) {
	days := map[string]float64{}
	for _, l := range dd.leaveDay {
		if l.Date >= from && l.Date <= to {
			days[l.Type]++
		}
	}
	for t, v := range days {
		byType = append(byType, DashSlice{Key: t, Value: v, Count: int(v)})
	}
	sort.Slice(byType, func(i, j int) bool { return byType[i].Value > byType[j].Value })
	absent := 0
	for _, a := range dd.days {
		if a.Date >= from && a.Date <= to {
			absent += a.Absent
		}
	}
	reasons = append([]DashSlice{{Key: "absent", Value: float64(absent), Count: absent}}, byType...)
	sort.SliceStable(reasons, func(i, j int) bool { return reasons[i].Value > reasons[j].Value })
	return byType, reasons
}

func addTypeMonth(rows *[]DashTypeMonth, month, typ string) {
	for i := range *rows {
		if (*rows)[i].Month == month && (*rows)[i].Type == typ {
			(*rows)[i].Days++
			return
		}
	}
	*rows = append(*rows, DashTypeMonth{Month: month, Type: typ, Days: 1})
}

func (x *Dashboard) topLate(ctx context.Context, from, to string, f domain.DashFilter) ([]DashEmployeeStat, error) {
	rows, err := x.d.Store.DashAttendanceByEmployee(ctx, from, to, f)
	if err != nil {
		return nil, err
	}
	out := make([]DashEmployeeStat, 0, len(rows))
	for _, r := range rows {
		if r.Late == 0 {
			continue
		}
		out = append(out, DashEmployeeStat{EmployeeID: r.EmployeeID, FullName: r.FullName, DepartmentName: r.DepartmentName,
			PositionName: r.PositionName, Records: r.Records, Late: r.Late, LateMinutes: r.LateMinutes, Absent: r.Absent,
			AttendanceRate: pct(float64(r.Records-r.Absent), float64(r.Records))})
	}
	return out, nil
}

func topOvertime(rows []domain.DashOvertime, from, to string) []DashEmployeeStat {
	by := map[string]*DashEmployeeStat{}
	for _, o := range rows {
		if o.Date < from || o.Date > to {
			continue
		}
		s, ok := by[o.EmployeeID]
		if !ok {
			s = &DashEmployeeStat{EmployeeID: o.EmployeeID, FullName: o.FullName, DepartmentName: o.DepartmentName, PositionName: o.PositionName}
			by[o.EmployeeID] = s
		}
		s.Records++
		s.OvertimeHours += o.Hours
		s.OvertimeCost += o.Cost
	}
	out := make([]DashEmployeeStat, 0, len(by))
	for _, s := range by {
		s.OvertimeHours, s.OvertimeCost = round2(s.OvertimeHours), round2(s.OvertimeCost)
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OvertimeHours != out[j].OvertimeHours {
			return out[i].OvertimeHours > out[j].OvertimeHours
		}
		return out[i].FullName < out[j].FullName
	})
	return out[:min(len(out), 10)]
}

// pending lists open requests with their age and the aging buckets.
func (x *Dashboard) pending(ctx context.Context, out *HrDashboard, f domain.DashFilter, today string) error {
	rows, err := x.d.Store.DashPending(ctx, f)
	if err != nil {
		return err
	}
	bands := []DashSlice{{Key: "0-2"}, {Key: "3-7"}, {Key: "8-14"}, {Key: "15+"}}
	for _, r := range rows {
		age := len(hris.EachDateISO(hris.TodayWIB(r.CreatedAt), today)) - 1
		age = max(age, 0)
		out.Pending = append(out.Pending, DashPendingItem{DashPending: r, AgeDays: age})
		i := 3
		switch {
		case age <= 2:
			i = 0
		case age <= 7:
			i = 1
		case age <= 14:
			i = 2
		}
		bands[i].Count++
		bands[i].Value++
	}
	out.PendingAging = bands
	sort.SliceStable(out.Pending, func(i, j int) bool { return out.Pending[i].AgeDays > out.Pending[j].AgeDays })
	return nil
}

// workforceSlices groups the people on the books at `to`.
func workforceSlices(out *HrDashboard, emps []domain.DashEmployee, to string) {
	status, position, level, contract, gender, age, tenure := counter{}, counter{}, counter{}, counter{}, counter{}, counter{}, counter{}
	t, _ := hris.ParseDate(to)
	for _, e := range emps {
		if !onBooks(e, to) {
			continue
		}
		status.add(e.EmploymentStatus, e.StatusName)
		position.add(e.PositionID, e.PositionName)
		level.add(e.Level, e.Level)
		contract.add(e.ContractType, e.ContractType)
		gender.add(e.Gender, e.Gender)
		if e.BirthDate != nil {
			b, _ := hris.ParseDate(*e.BirthDate)
			age.add(ageBand(yearsBetween(b, t)), "")
		}
		j, _ := hris.ParseDate(e.JoinDate)
		tenure.add(tenureBand(yearsBetween(j, t)), "")
	}
	out.ByStatus, out.ByPosition, out.ByLevel = status.slices(), position.slices(), level.slices()
	out.ByContractType, out.ByGender = contract.slices(), gender.slices()
	out.AgeBands = age.ordered([]string{"<25", "25-29", "30-34", "35-39", "40+"})
	out.TenureBands = tenure.ordered([]string{"<1", "1-2", "2-3", "3-5", "5+"})
}

type counter struct {
	keys   []string
	labels map[string]string
	n      map[string]int
}

func (c *counter) add(key, label string) {
	if c.n == nil {
		c.n, c.labels = map[string]int{}, map[string]string{}
	}
	if _, ok := c.n[key]; !ok {
		c.keys = append(c.keys, key)
		c.labels[key] = label
	}
	c.n[key]++
}

// slices lists the keys by count, largest first.
func (c *counter) slices() []DashSlice {
	out := make([]DashSlice, 0, len(c.keys))
	for _, k := range c.keys {
		out = append(out, DashSlice{Key: k, Label: c.labels[k], Value: float64(c.n[k]), Count: c.n[k]})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

func (c *counter) ordered(keys []string) []DashSlice {
	out := make([]DashSlice, len(keys))
	for i, k := range keys {
		out[i] = DashSlice{Key: k, Value: float64(c.n[k]), Count: c.n[k]}
	}
	return out
}

func yearsBetween(a, b time.Time) int {
	y := b.Year() - a.Year()
	if b.YearDay() < a.YearDay() {
		y--
	}
	return y
}

func ageBand(y int) string {
	switch {
	case y < 25:
		return "<25"
	case y < 30:
		return "25-29"
	case y < 35:
		return "30-34"
	case y < 40:
		return "35-39"
	}
	return "40+"
}

func tenureBand(y int) string {
	switch {
	case y < 1:
		return "<1"
	case y < 2:
		return "1-2"
	case y < 3:
		return "2-3"
	case y < 5:
		return "3-5"
	}
	return "5+"
}

// expiring are contracts and probations ending within 90 days of today.
func (x *Dashboard) expiring(ctx context.Context, emps []domain.DashEmployee, f domain.DashFilter, today string) ([]DashExpiry, error) {
	until := hris.AddDaysISO(today, 90)
	rows, err := x.d.Store.DashContractsEnding(ctx, today, until, f)
	if err != nil {
		return nil, err
	}
	daysLeft := func(end string) int { return len(hris.EachDateISO(today, end)) - 1 }
	var out []DashExpiry
	for _, c := range rows {
		out = append(out, DashExpiry{Kind: "contract", EmployeeID: c.EmployeeID, FullName: c.FullName, DepartmentName: c.DepartmentName,
			ContractID: c.ID, ContractNumber: c.ContractNumber, ContractType: c.ContractType, EndDate: c.EndDate, DaysLeft: daysLeft(c.EndDate)})
	}
	for _, e := range emps {
		if e.IsActive && e.EmploymentStatus == "probation" && e.ProbationEndDate != nil && *e.ProbationEndDate >= today && *e.ProbationEndDate <= until {
			out = append(out, DashExpiry{Kind: "probation", EmployeeID: e.ID, FullName: e.FullName, DepartmentName: e.DepartmentName,
				EndDate: *e.ProbationEndDate, DaysLeft: daysLeft(*e.ProbationEndDate)})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].EndDate < out[j].EndDate })
	return out, nil
}

// payrollExtras adds pay by level for the latest paid month up to `to`
// and the loans still being repaid.
func (x *Dashboard) payrollExtras(ctx context.Context, out *HrDashboard, dd *dashData, f domain.DashFilter, to string) error {
	latest := ""
	for _, p := range dd.payroll {
		if p.Month <= to && p.Month > latest {
			latest = p.Month
		}
	}
	if latest != "" {
		levels, err := x.d.Store.DashPayrollByLevel(ctx, latest, latest, f)
		if err != nil {
			return err
		}
		out.SalaryByLevel = levels
	}
	loans, err := x.d.Store.DashLoans(ctx, f)
	if err != nil {
		return err
	}
	out.Loans = loans
	for _, l := range loans {
		out.LoansOutstanding += l.Remaining
	}
	return nil
}
