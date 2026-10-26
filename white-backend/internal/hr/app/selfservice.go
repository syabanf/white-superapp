package app

import (
	"context"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
)

// SelfService composes the ESS home (lib/hris/me-profile). The other /me
// endpoints call the area use cases with the caller as the employee.
type SelfService struct {
	d      Deps
	leaves *Leaves
	ann    *Announcements
}

func NewSelfService(d Deps, leaves *Leaves, ann *Announcements) *SelfService {
	return &SelfService{d: d, leaves: leaves, ann: ann}
}

type Home struct {
	Employee            domain.Employee
	Today               string
	TodayShift          *hris.TodayShift
	TodayAttendance     *domain.Attendance
	Week                []TeamDay
	Balance             *Balance
	UnreadAnnouncements int
	PendingApprovals    int
	HolidayName         string
}

func (s *SelfService) Profile(ctx context.Context, a Actor) (*domain.Employee, error) {
	id, err := self(a)
	if err != nil {
		return nil, err
	}
	return (&People{d: s.d}).Get(ctx, id)
}

func (s *SelfService) Home(ctx context.Context, a Actor) (*Home, error) {
	emp, err := s.Profile(ctx, a)
	if err != nil {
		return nil, err
	}
	today := s.d.today()
	h := &Home{Employee: *emp, Today: today}
	rows, err := s.d.Store.ListScheduleRows(ctx, []string{emp.ID})
	if err != nil {
		return nil, err
	}
	ess := make([]hris.ESSScheduleRow, len(rows))
	for i, r := range rows {
		ess[i] = hris.ESSScheduleRow{ScheduleRow: hris.ScheduleRow{DayOfWeek: r.DayOfWeek, ShiftID: r.ShiftID, EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo}}
		if r.Shift != nil {
			name, start, end, tol := r.Shift.Name, r.Shift.StartTime, r.Shift.EndTime, int64(r.Shift.LateToleranceMinutes)
			ess[i].ShiftName, ess[i].StartTime, ess[i].EndTime, ess[i].LateToleranceMinutes = &name, &start, &end, &tol
		}
	}
	h.TodayShift = hris.TodayShiftOf(ess, today)
	atts, _, err := s.d.Store.ListAttendance(ctx, domain.AttendanceFilter{EmployeeIDs: []string{emp.ID}, From: today, To: today})
	if err != nil {
		return nil, err
	}
	if len(atts) > 0 {
		h.TodayAttendance = &atts[0]
	}
	monday := weekStart(today)
	sunday := hris.AddDaysISO(monday, 6)
	leaves, _, err := s.d.Store.ListLeaves(ctx, domain.LeaveFilter{EmployeeIDs: []string{emp.ID}, Status: hris.LeaveApproved, From: monday, To: sunday})
	if err != nil {
		return nil, err
	}
	holidays, err := s.d.Store.ListHolidays(ctx, monday, sunday, true)
	if err != nil {
		return nil, err
	}
	h.Week = weekDays(rows, leaves, holidays, monday)
	for _, hd := range holidays {
		if hd.Date == today {
			h.HolidayName = hd.Name
		}
	}
	if h.Balance, err = s.leaves.Balance(ctx, a, emp.ID, 0); err != nil {
		return nil, err
	}
	if _, h.UnreadAnnouncements, err = s.ann.Feed(ctx, a); err != nil {
		return nil, err
	}
	approvals, err := s.Approvals(ctx, a)
	if err != nil {
		return nil, err
	}
	h.PendingApprovals = len(approvals.Leaves) + len(approvals.Overtime)
	return h, nil
}

type Approvals struct {
	Leaves   []domain.Leave
	Overtime []domain.Overtime
}

// Approvals are the direct reports' requests waiting for the caller.
func (s *SelfService) Approvals(ctx context.Context, a Actor) (*Approvals, error) {
	out := &Approvals{}
	if a.EmployeeID == "" {
		return out, nil
	}
	leaves, _, err := s.leaves.List(ctx, a, LeaveQuery{Scope: "approvals"})
	if err != nil {
		return nil, err
	}
	out.Leaves = leaves
	ots, err := (&Overtime{d: s.d}).List(ctx, a, OvertimeQuery{Scope: "approvals", Status: "pending"})
	if err != nil {
		return nil, err
	}
	out.Overtime = ots
	return out, nil
}

// MonthOrRecent turns a ListMyRequest month into a date range: the month
// when given, else the last 31 days.
func MonthOrRecent(month string, now time.Time) (string, string, error) {
	if month == "" {
		today := hris.TodayWIB(now)
		return hris.AddDaysISO(today, -31), today, nil
	}
	from, to, ok := monthBounds(strings.TrimSpace(month))
	if !ok {
		return "", "", invalid("month", "Bulan tidak valid (YYYY-MM).")
	}
	return from, to, nil
}
