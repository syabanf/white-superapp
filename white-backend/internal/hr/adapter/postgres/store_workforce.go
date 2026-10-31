package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── Shifts and schedules ────────────────────────────────────

func toShift(r sqlc.ListShiftsRow) domain.Shift {
	return domain.Shift{ID: r.ID, Name: r.Name, StartTime: r.StartTime, EndTime: r.EndTime, BreakMinutes: int(r.BreakMinutes),
		LateToleranceMinutes: int(r.LateToleranceMinutes), IsOvernight: r.IsOvernight, IsActive: r.IsActive, SortOrder: int(r.SortOrder)}
}

func (s *Store) ListShifts(ctx context.Context, includeInactive bool) ([]domain.Shift, error) {
	rows, err := s.q.ListShifts(ctx, includeInactive)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Shift, len(rows))
	for i, r := range rows {
		out[i] = toShift(r)
	}
	return out, nil
}

func (s *Store) GetShift(ctx context.Context, id string) (*domain.Shift, error) {
	r, err := one(s.q.GetShift(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	sh := toShift(sqlc.ListShiftsRow(*r))
	return &sh, nil
}

func (s *Store) InsertShift(ctx context.Context, sh domain.Shift, actor *string) error {
	return s.q.InsertShift(ctx, sqlc.InsertShiftParams{ID: sh.ID, Name: sh.Name, StartTime: sh.StartTime, EndTime: sh.EndTime,
		BreakMinutes: i32(sh.BreakMinutes), LateToleranceMinutes: i32(sh.LateToleranceMinutes), IsOvernight: sh.IsOvernight,
		IsActive: sh.IsActive, SortOrder: i32(sh.SortOrder), Actor: actor})
}

func (s *Store) UpdateShift(ctx context.Context, sh domain.Shift, actor *string) (bool, error) {
	n, err := s.q.UpdateShift(ctx, sqlc.UpdateShiftParams{ID: sh.ID, Name: sh.Name, StartTime: sh.StartTime, EndTime: sh.EndTime,
		BreakMinutes: i32(sh.BreakMinutes), LateToleranceMinutes: i32(sh.LateToleranceMinutes), IsOvernight: sh.IsOvernight,
		IsActive: sh.IsActive, SortOrder: i32(sh.SortOrder), Actor: actor})
	return n > 0, err
}

// DeleteShift deletes an unused shift and deactivates a used one; found
// is false when the shift does not exist.
func (s *Store) DeleteShift(ctx context.Context, id string) (deactivated, found bool, err error) {
	used, err := s.q.ShiftInUse(ctx, id)
	if err != nil {
		return false, false, err
	}
	if used {
		if _, err := s.q.GetShift(ctx, id); err != nil {
			if pg.IsNoRows(err) {
				return false, false, nil
			}
			return false, false, err
		}
		return true, true, s.q.DeactivateShift(ctx, id)
	}
	n, err := s.q.DeleteShift(ctx, id)
	return false, n > 0, err
}

func (s *Store) ListScheduleRows(ctx context.Context, employeeIDs []string) ([]domain.ScheduleRow, error) {
	rows, err := s.q.ListScheduleRows(ctx, nonNil(employeeIDs))
	if err != nil {
		return nil, err
	}
	out := make([]domain.ScheduleRow, len(rows))
	for i, r := range rows {
		row := domain.ScheduleRow{ID: r.ID, EmployeeID: r.EmployeeID, DayOfWeek: int(r.DayOfWeek), ShiftID: r.ShiftID,
			EffectiveFrom: day(r.EffectiveFrom), EffectiveTo: dayPtr(r.EffectiveTo)}
		if r.ShiftID != nil {
			row.Shift = &domain.Shift{ID: *r.ShiftID, Name: r.ShiftName, StartTime: r.StartTime, EndTime: r.EndTime,
				BreakMinutes: int(r.BreakMinutes), LateToleranceMinutes: int(r.LateToleranceMinutes), IsOvernight: r.IsOvernight}
		}
		out[i] = row
	}
	return out, nil
}

// ReplaceSchedule ends the running pattern the day before from, drops any
// pattern starting on or after from, and inserts rows.
func (s *Store) ReplaceSchedule(ctx context.Context, employeeID, from, closeOn string, rows []domain.ScheduleRow, actor *string) error {
	f := parseDay(from)
	if err := s.q.DeleteScheduleFrom(ctx, sqlc.DeleteScheduleFromParams{EmployeeID: employeeID, EffectiveFrom: f}); err != nil {
		return err
	}
	closeDay := parseDay(closeOn)
	if err := s.q.CloseSchedule(ctx, sqlc.CloseScheduleParams{EffectiveTo: &closeDay, EmployeeID: employeeID, EffectiveFrom: f}); err != nil {
		return err
	}
	for _, r := range rows {
		if err := s.q.InsertScheduleRow(ctx, sqlc.InsertScheduleRowParams{ID: r.ID, EmployeeID: employeeID, DayOfWeek: i32(r.DayOfWeek),
			ShiftID: r.ShiftID, EffectiveFrom: f, CreatedBy: actor}); err != nil {
			return err
		}
	}
	return nil
}

// ─── Holidays ────────────────────────────────────────────────

func toHoliday(h sqlc.HrPublicHoliday) domain.Holiday {
	return domain.Holiday{ID: h.ID, Date: day(h.HolidayDate), Name: h.Name, Type: h.Type, DeductsLeave: h.DeductsLeave, Status: h.Status, Note: h.Note}
}

func (s *Store) ListHolidays(ctx context.Context, from, to string, onlyActive bool) ([]domain.Holiday, error) {
	rows, err := s.q.ListHolidays(ctx, sqlc.ListHolidaysParams{DateFrom: parseDay(from), DateTo: parseDay(to), OnlyActive: onlyActive})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Holiday, len(rows))
	for i, r := range rows {
		out[i] = toHoliday(r)
	}
	return out, nil
}

func (s *Store) InsertHoliday(ctx context.Context, h domain.Holiday, actor *string) error {
	_, err := s.q.InsertHoliday(ctx, sqlc.InsertHolidayParams{ID: h.ID, HolidayDate: parseDay(h.Date), Name: h.Name, Type: h.Type,
		DeductsLeave: h.DeductsLeave, Status: h.Status, Note: h.Note, CreatedBy: actor})
	return err
}

func (s *Store) UpdateHoliday(ctx context.Context, h domain.Holiday, actor *string) (bool, error) {
	_, err := s.q.UpdateHoliday(ctx, sqlc.UpdateHolidayParams{ID: h.ID, HolidayDate: parseDay(h.Date), Name: h.Name, Type: h.Type,
		DeductsLeave: h.DeductsLeave, Status: h.Status, Note: h.Note, UpdatedBy: actor})
	if pg.IsNoRows(err) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) DeleteHoliday(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteHoliday(ctx, id)
	return n > 0, err
}

// ─── Attendance ──────────────────────────────────────────────

func decodeLocation(raw []byte) *domain.Location {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var l domain.Location
	if json.Unmarshal(raw, &l) != nil {
		return nil
	}
	return &l
}

func encodeLocation(l *domain.Location) []byte {
	if l == nil {
		return nil
	}
	raw, _ := json.Marshal(l)
	return raw
}

func toAttendance(a sqlc.HrAttendance, name, no, dept, pos, shift, validator string) domain.Attendance {
	return domain.Attendance{
		ID: a.ID, EmployeeID: a.EmployeeID, EmployeeName: name, EmployeeNo: no, DepartmentName: dept, PositionName: pos,
		Date: day(a.Date), ClockIn: a.ClockIn, ClockOut: a.ClockOut, ClockInLocation: decodeLocation(a.ClockInLocation),
		ClockOutLocation: decodeLocation(a.ClockOutLocation), ClockInPhotoKey: a.ClockInPhotoKey, ClockOutPhotoKey: a.ClockOutPhotoKey,
		WorkHours: num(a.WorkHours), BreakMinutes: int(a.BreakMinutes), Status: string(a.Status), IsLate: a.IsLate,
		LateMinutes: int(a.LateMinutes), OvertimeHours: num(a.OvertimeHours), ShiftID: a.ShiftID, ShiftName: shift,
		ScheduledStart: a.ScheduledStart, ScheduledEnd: a.ScheduledEnd, ValidatedBy: a.ValidatedBy, ValidatedByName: validator,
		ValidatedAt: a.ValidatedAt, ValidationNotes: a.ValidationNotes, Notes: a.Notes, CreatedAt: a.CreatedAt,
	}
}

func optDay(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := parseDay(s)
	return &t
}

func (s *Store) ListAttendance(ctx context.Context, f domain.AttendanceFilter) ([]domain.Attendance, int, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 5000
	}
	rows, err := s.q.ListAttendance(ctx, sqlc.ListAttendanceParams{EmployeeIds: nonNil(f.EmployeeIDs), DateFrom: optDay(f.From),
		DateTo: optDay(f.To), Status: f.Status, OnlyLate: f.OnlyLate, OnlyUnvalidated: f.OnlyUnvalidated, DepartmentID: f.DepartmentID,
		RowLimit: i32(limit), RowOffset: i32(f.Offset)})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Attendance, len(rows))
	total := 0
	for i, r := range rows {
		out[i] = toAttendance(r.HrAttendance, r.FullName, r.EmployeeNo, r.DepartmentName, r.PositionName, r.ShiftName, r.ValidatedByName)
		total = int(r.Total)
	}
	return out, total, nil
}

func (s *Store) GetAttendance(ctx context.Context, id string) (*domain.Attendance, error) {
	r, err := one(s.q.GetAttendance(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	a := toAttendance(r.HrAttendance, r.FullName, r.EmployeeNo, r.DepartmentName, r.PositionName, r.ShiftName, r.ValidatedByName)
	return &a, nil
}

func (s *Store) AttendanceIDOn(ctx context.Context, employeeID, date string) (string, error) {
	id, err := s.q.AttendanceIDOn(ctx, sqlc.AttendanceIDOnParams{EmployeeID: employeeID, Date: parseDay(date)})
	if pg.IsNoRows(err) {
		return "", nil
	}
	return id, err
}

func (s *Store) InsertAttendance(ctx context.Context, a domain.Attendance, actor *string) error {
	return wrap(s.q.InsertAttendance(ctx, sqlc.InsertAttendanceParams{ID: a.ID, EmployeeID: a.EmployeeID, Date: parseDay(a.Date),
		ClockIn: a.ClockIn, ClockInLocation: encodeLocation(a.ClockInLocation), ClockInPhotoKey: a.ClockInPhotoKey,
		Status: sqlc.HrAttendanceStatus(a.Status), IsLate: a.IsLate, LateMinutes: i32(a.LateMinutes), ShiftID: a.ShiftID,
		ScheduledStart: a.ScheduledStart, ScheduledEnd: a.ScheduledEnd, BreakMinutes: i32(a.BreakMinutes), Notes: a.Notes, CreatedBy: actor,
		ClockOut: a.ClockOut, WorkHours: money(a.WorkHours), OvertimeHours: money(a.OvertimeHours)}))
}

// UpdateAttendance rewrites the times, status and derived columns of a row.
func (s *Store) UpdateAttendance(ctx context.Context, a domain.Attendance) (bool, error) {
	n, err := s.q.UpdateAttendance(ctx, sqlc.UpdateAttendanceParams{ID: a.ID, ClockIn: a.ClockIn, ClockOut: a.ClockOut,
		Status: sqlc.HrAttendanceStatus(a.Status), IsLate: a.IsLate, LateMinutes: i32(a.LateMinutes), WorkHours: money(a.WorkHours),
		OvertimeHours: money(a.OvertimeHours), ShiftID: a.ShiftID, ScheduledStart: a.ScheduledStart, ScheduledEnd: a.ScheduledEnd,
		BreakMinutes: i32(a.BreakMinutes), Notes: a.Notes})
	return n > 0, err
}

func (s *Store) DeleteAttendance(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteAttendance(ctx, id)
	return n > 0, err
}

// InsertAttendanceRecord stores a complete row (seed and back-fill); an
// existing row for the employee and date is kept.
func (s *Store) InsertAttendanceRecord(ctx context.Context, a domain.Attendance, actor *string) error {
	return s.q.InsertAttendanceRow(ctx, sqlc.InsertAttendanceRowParams{ID: a.ID, EmployeeID: a.EmployeeID, Date: parseDay(a.Date),
		ClockIn: a.ClockIn, ClockOut: a.ClockOut, Status: sqlc.HrAttendanceStatus(a.Status), IsLate: a.IsLate,
		LateMinutes: i32(a.LateMinutes), WorkHours: money(a.WorkHours), OvertimeHours: money(a.OvertimeHours), ShiftID: a.ShiftID,
		ScheduledStart: a.ScheduledStart, ScheduledEnd: a.ScheduledEnd, ValidatedBy: a.ValidatedBy, ValidatedAt: a.ValidatedAt, CreatedBy: actor})
}

func (s *Store) ClockOut(ctx context.Context, a domain.Attendance) (bool, error) {
	n, err := s.q.ClockOut(ctx, sqlc.ClockOutParams{ID: a.ID, ClockOut: a.ClockOut, ClockOutLocation: encodeLocation(a.ClockOutLocation),
		ClockOutPhotoKey: a.ClockOutPhotoKey, WorkHours: money(a.WorkHours), Notes: a.Notes, OvertimeHours: money(a.OvertimeHours)})
	return n > 0, err
}

func (s *Store) ValidateAttendance(ctx context.Context, id, status string, validatedBy *string, notes string) (bool, error) {
	n, err := s.q.ValidateAttendance(ctx, sqlc.ValidateAttendanceParams{ID: id, Status: sqlc.HrAttendanceStatus(status), ValidatedBy: validatedBy, ValidationNotes: notes})
	return n > 0, err
}

func (s *Store) AttendanceTotals(ctx context.Context, employeeIDs []string, from, to string) (map[string]domain.AttendanceTotals, error) {
	rows, err := s.q.AttendanceSummary(ctx, sqlc.AttendanceSummaryParams{EmployeeIds: nonNil(employeeIDs), DateFrom: parseDay(from), DateTo: parseDay(to)})
	if err != nil {
		return nil, err
	}
	out := map[string]domain.AttendanceTotals{}
	for _, r := range rows {
		out[r.EmployeeID] = domain.AttendanceTotals{PresentDays: int(r.PresentDays), LateDays: int(r.LateDays), AbsentDays: int(r.AbsentDays), LateMinutes: int(r.LateMinutes)}
	}
	ot, err := s.q.ApprovedOvertimeHours(ctx, sqlc.ApprovedOvertimeHoursParams{EmployeeIds: nonNil(employeeIDs), DateFrom: parseDay(from), DateTo: parseDay(to)})
	if err != nil {
		return nil, err
	}
	for _, r := range ot {
		t := out[r.EmployeeID]
		t.OvertimeHours = r.Hours
		out[r.EmployeeID] = t
	}
	return out, nil
}

// ─── Leave ───────────────────────────────────────────────────

func toBalance(b sqlc.HrLeaveBalance) domain.LeaveBalance {
	return domain.LeaveBalance{ID: b.ID, EmployeeID: b.EmployeeID, Year: int(b.Year), AnnualTotal: int(b.AnnualLeaveTotal),
		AnnualUsed: int(b.AnnualLeaveUsed), SickUsed: int(b.SickLeaveUsed), UnpaidUsed: int(b.UnpaidLeaveUsed), OtherUsed: int(b.OtherLeaveUsed)}
}

func (s *Store) GetLeaveBalance(ctx context.Context, employeeID string, year int) (*domain.LeaveBalance, error) {
	r, err := one(s.q.GetLeaveBalance(ctx, sqlc.GetLeaveBalanceParams{EmployeeID: employeeID, Year: i32(year)}))
	if r == nil || err != nil {
		return nil, err
	}
	b := toBalance(*r)
	return &b, nil
}

// EnsureLeaveBalance inserts the year's row unless it exists and returns it.
func (s *Store) EnsureLeaveBalance(ctx context.Context, b domain.LeaveBalance, actor *string) (*domain.LeaveBalance, error) {
	r, err := s.q.InsertLeaveBalance(ctx, sqlc.InsertLeaveBalanceParams{ID: b.ID, EmployeeID: b.EmployeeID, Year: i32(b.Year),
		AnnualLeaveTotal: i32(b.AnnualTotal), AnnualLeaveUsed: i32(b.AnnualUsed), UpdatedBy: actor})
	if err != nil {
		return nil, err
	}
	out := toBalance(r)
	return &out, nil
}

func (s *Store) SetLeaveBalance(ctx context.Context, employeeID string, year, total, used int, actor *string) (*domain.LeaveBalance, error) {
	r, err := s.q.SetLeaveBalance(ctx, sqlc.SetLeaveBalanceParams{EmployeeID: employeeID, Year: i32(year), AnnualLeaveTotal: i32(total), AnnualLeaveUsed: i32(used), UpdatedBy: actor})
	if err != nil {
		return nil, err
	}
	out := toBalance(r)
	return &out, nil
}

// AddLeaveUsage moves the used counters of a year (negative refunds).
func (s *Store) AddLeaveUsage(ctx context.Context, employeeID string, year int, u domain.LeaveUsage) error {
	return s.q.AddLeaveUsage(ctx, sqlc.AddLeaveUsageParams{EmployeeID: employeeID, Year: i32(year), Annual: i32(u.Annual), Sick: i32(u.Sick), Unpaid: i32(u.Unpaid), Other: i32(u.Other)})
}

func (s *Store) PendingAnnualDays(ctx context.Context, employeeID string, year int) (int, error) {
	n, err := s.q.PendingLeaveDays(ctx, sqlc.PendingLeaveDaysParams{EmployeeID: employeeID, Year: i32(year)})
	return int(n), err
}

func toLeave(l sqlc.HrLeafe, name, no string, mgr *string, dept, mgrApprover, approver string) domain.Leave {
	return domain.Leave{ID: l.ID, EmployeeID: l.EmployeeID, EmployeeName: name, EmployeeNo: no, DepartmentName: dept,
		EmployeeManagerID: mgr, LeaveType: l.LeaveType, StartDate: day(l.StartDate), EndDate: day(l.EndDate),
		TotalDays: int(l.TotalDays), Reason: l.Reason, Status: l.Status, ManagerApprovedBy: l.ManagerApprovedBy,
		ManagerApprovedByName: mgrApprover, ManagerApprovedAt: l.ManagerApprovedAt, ApprovedBy: l.ApprovedBy,
		ApprovedByName: approver, ApprovedAt: l.ApprovedAt, RejectedBy: l.RejectedBy, RejectionReason: l.RejectionReason,
		CreatedBy: l.CreatedBy, CreatedAt: l.CreatedAt}
}

func (s *Store) ListLeaves(ctx context.Context, f domain.LeaveFilter) ([]domain.Leave, int, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 1000
	}
	statuses := f.Statuses
	if f.Status != "" {
		statuses = []string{f.Status}
	}
	rows, err := s.q.ListLeaves(ctx, sqlc.ListLeavesParams{EmployeeIds: nonNil(f.EmployeeIDs), Statuses: nonNil(statuses),
		LeaveType: f.LeaveType, DateFrom: optDay(f.From), DateTo: optDay(f.To), DepartmentID: f.DepartmentID, RowLimit: i32(limit), RowOffset: i32(f.Offset)})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Leave, len(rows))
	total := 0
	for i, r := range rows {
		out[i] = toLeave(r.HrLeafe, r.FullName, r.EmployeeNo, r.EmployeeManagerID, r.DepartmentName, r.ManagerApprovedByName, r.ApprovedByName)
		total = int(r.Total)
	}
	return out, total, nil
}

func (s *Store) GetLeave(ctx context.Context, id string) (*domain.Leave, error) {
	r, err := one(s.q.GetLeave(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	l := toLeave(r.HrLeafe, r.FullName, r.EmployeeNo, r.EmployeeManagerID, r.DepartmentName, r.ManagerApprovedByName, r.ApprovedByName)
	return &l, nil
}

func (s *Store) InsertLeave(ctx context.Context, l domain.Leave) error {
	return s.q.InsertLeave(ctx, sqlc.InsertLeaveParams{ID: l.ID, EmployeeID: l.EmployeeID, LeaveType: l.LeaveType,
		StartDate: parseDay(l.StartDate), EndDate: parseDay(l.EndDate), TotalDays: i32(l.TotalDays), Reason: l.Reason,
		Status: l.Status, ManagerApprovedBy: l.ManagerApprovedBy, ManagerApprovedAt: l.ManagerApprovedAt,
		ApprovedBy: l.ApprovedBy, ApprovedAt: l.ApprovedAt, CreatedBy: l.CreatedBy, CreatedAt: instantPtr(l.CreatedAt)})
}

func (s *Store) LeaveOverlaps(ctx context.Context, employeeID, start, end string) (bool, error) {
	return s.q.OverlappingLeave(ctx, sqlc.OverlappingLeaveParams{EmployeeID: employeeID, StartDate: parseDay(start), EndDate: parseDay(end)})
}

// SetLeaveStatus moves a leave from one status to another; false when it
// was no longer in from (someone else decided first).
func (s *Store) SetLeaveStatus(ctx context.Context, u domain.LeaveStatusUpdate) (bool, error) {
	n, err := s.q.SetLeaveStatus(ctx, sqlc.SetLeaveStatusParams{ID: u.ID, FromStatus: u.From, Status: u.To,
		ManagerApprovedBy: u.ManagerApprovedBy, ApprovedBy: u.ApprovedBy, RejectedBy: u.RejectedBy,
		RejectionReason: u.RejectionReason, Actor: u.Actor})
	return n > 0, err
}

func (s *Store) ApprovedLeavesOn(ctx context.Context, date string) (map[string]string, error) {
	rows, err := s.q.ApprovedLeavesOn(ctx, parseDay(date))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, r := range rows {
		out[r.EmployeeID] = r.LeaveType
	}
	return out, nil
}

// ─── Overtime ────────────────────────────────────────────────

func toOvertime(o sqlc.HrOvertimeRequest, start, end, name, no string, mgr *string, dept, requester, decider string) domain.Overtime {
	return domain.Overtime{ID: o.ID, EmployeeID: o.EmployeeID, EmployeeName: name, EmployeeNo: no, DepartmentName: dept,
		EmployeeManagerID: mgr, Date: day(o.Date), StartTime: domain.HHMM(start), EndTime: domain.HHMM(end), Hours: num(o.Hours),
		Source: o.Source, Status: o.Status, Reason: o.Reason, RequestedBy: o.RequestedBy, RequestedByName: requester,
		DecidedBy: o.DecidedBy, DecidedByName: decider, DecidedAt: o.DecidedAt, RejectionReason: o.RejectionReason, CreatedAt: o.CreatedAt}
}

func (s *Store) ListOvertime(ctx context.Context, f domain.OvertimeFilter) ([]domain.Overtime, error) {
	rows, err := s.q.ListOvertime(ctx, sqlc.ListOvertimeParams{EmployeeIds: nonNil(f.EmployeeIDs), Source: f.Source, Status: f.Status,
		DateFrom: optDay(f.From), DateTo: optDay(f.To), DepartmentID: f.DepartmentID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Overtime, len(rows))
	for i, r := range rows {
		out[i] = toOvertime(r.HrOvertimeRequest, r.StartHhmm, r.EndHhmm, r.FullName, r.EmployeeNo, r.EmployeeManagerID, r.DepartmentName, r.RequestedByName, r.DecidedByName)
	}
	return out, nil
}

func (s *Store) GetOvertime(ctx context.Context, id string) (*domain.Overtime, error) {
	r, err := one(s.q.GetOvertime(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	o := toOvertime(r.HrOvertimeRequest, r.StartHhmm, r.EndHhmm, r.FullName, r.EmployeeNo, r.EmployeeManagerID, r.DepartmentName, r.RequestedByName, r.DecidedByName)
	return &o, nil
}

func (s *Store) InsertOvertime(ctx context.Context, o domain.Overtime, actor *string) error {
	return wrap(s.q.InsertOvertime(ctx, sqlc.InsertOvertimeParams{ID: o.ID, EmployeeID: o.EmployeeID, Date: parseDay(o.Date),
		StartTime: o.StartTime, EndTime: o.EndTime, Hours: money(o.Hours), Source: o.Source, Status: o.Status, Reason: o.Reason,
		RequestedBy: o.RequestedBy, DecidedBy: o.DecidedBy, DecidedAt: o.DecidedAt, CreatedBy: actor, CreatedAt: instantPtr(o.CreatedAt)}))
}

// UpdateOvertime changes a pending entry; false when it is no longer pending.
func (s *Store) UpdateOvertime(ctx context.Context, o domain.Overtime) (bool, error) {
	n, err := s.q.UpdateOvertime(ctx, sqlc.UpdateOvertimeParams{ID: o.ID, Date: parseDay(o.Date), StartTime: o.StartTime,
		EndTime: o.EndTime, Hours: money(o.Hours), Reason: o.Reason})
	return n > 0, wrap(err)
}

// DeleteOvertime removes an entry unless it is approved.
func (s *Store) DeleteOvertime(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteOvertime(ctx, id)
	return n > 0, err
}

func (s *Store) DecideOvertime(ctx context.Context, id, status string, decidedBy *string, rejection string) (bool, error) {
	n, err := s.q.DecideOvertime(ctx, sqlc.DecideOvertimeParams{ID: id, Status: status, DecidedBy: decidedBy, RejectionReason: rejection})
	return n > 0, err
}
