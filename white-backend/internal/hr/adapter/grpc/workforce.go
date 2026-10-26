package grpcapi

import (
	"context"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

const xlsxType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// ─── AttendanceService ───────────────────────────────────────

type attendanceServer struct {
	hrv1.UnimplementedAttendanceServiceServer
	s *Services
}

func clockIn(req *hrv1.ClockInRequest) app.ClockInput {
	return app.ClockInput{EmployeeID: req.EmployeeId, Date: req.Date, Photo: req.Photo, Location: locationFrom(req.Location), Notes: req.Notes}
}

func clockOut(req *hrv1.ClockOutRequest) app.ClockInput {
	return app.ClockInput{AttendanceID: req.AttendanceId, Photo: req.Photo, Location: locationFrom(req.Location), Notes: req.Notes}
}

func (h *attendanceServer) ClockIn(ctx context.Context, req *hrv1.ClockInRequest) (*hrv1.AttendanceRecord, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.s.Attendance.ClockIn(ctx, a, clockIn(req))
	if err != nil {
		return nil, err
	}
	return attendancePB(*r), nil
}

func (h *attendanceServer) ClockOut(ctx context.Context, req *hrv1.ClockOutRequest) (*hrv1.AttendanceRecord, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.s.Attendance.ClockOut(ctx, a, clockOut(req))
	if err != nil {
		return nil, err
	}
	return attendancePB(*r), nil
}

func (h *attendanceServer) ListAttendance(ctx context.Context, req *hrv1.ListAttendanceRequest) (*hrv1.ListAttendanceResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	limit, offset := pageOf(req.Page, req.PageSize)
	f := domain.AttendanceFilter{From: req.StartDate, To: req.EndDate, Status: req.Status, OnlyLate: req.OnlyLate,
		OnlyUnvalidated: req.OnlyUnvalidated, Limit: limit, Offset: offset}
	if req.EmployeeId != "" {
		f.EmployeeIDs = []string{req.EmployeeId}
	}
	rows, total, err := h.s.Attendance.List(ctx, a, f)
	if err != nil {
		return nil, err
	}
	return attendanceList(rows, total), nil
}

func (h *attendanceServer) GetDailyRoster(ctx context.Context, req *hrv1.GetDailyRosterRequest) (*hrv1.DailyRoster, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.s.Attendance.Roster(ctx, a, req.Date, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	out := &hrv1.DailyRoster{Date: r.Date, HolidayName: r.HolidayName, Summary: map[string]int32{}}
	for k, v := range r.Summary {
		out.Summary[k] = int32(v)
	}
	for _, e := range r.Entries {
		entry := &hrv1.RosterEntry{Employee: refPB(e.Employee.ID, e.Employee.FullName, e.Employee.EmployeeNo, e.Employee.DepartmentName),
			Status: e.Status, IsOverdue: e.IsOverdue, LeaveType: e.LeaveType}
		if e.Shift != nil {
			entry.ShiftId, entry.ShiftName = e.Shift.ID, e.Shift.Name
			entry.ShiftStart, entry.ShiftEnd = domain.HHMM(e.Shift.StartTime), domain.HHMM(e.Shift.EndTime)
		}
		if at := e.Attendance; at != nil {
			entry.AttendanceId, entry.ClockIn, entry.ClockOut, entry.LateMinutes = at.ID, tsPtr(at.ClockIn), tsPtr(at.ClockOut), int32(at.LateMinutes)
		}
		out.Entries = append(out.Entries, entry)
	}
	return out, nil
}

func (h *attendanceServer) ValidateAttendance(ctx context.Context, req *hrv1.ValidateAttendanceRequest) (*hrv1.AttendanceRecord, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.s.Attendance.Validate(ctx, a, req.AttendanceId, req.Status, req.Notes)
	if err != nil {
		return nil, err
	}
	return attendancePB(*r), nil
}

func (h *attendanceServer) GetAttendancePhoto(ctx context.Context, req *hrv1.GetAttendancePhotoRequest) (*hrv1.FileBytes, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	data, ct, err := h.s.Attendance.Photo(ctx, a, req.AttendanceId, req.Kind)
	if err != nil {
		return nil, err
	}
	return file("selfie-"+req.AttendanceId, ct, data), nil
}

func (h *attendanceServer) ExportAttendance(ctx context.Context, req *hrv1.ExportAttendanceRequest) (*hrv1.FileBytes, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	data, name, err := h.s.Attendance.ExportXLSX(ctx, a, app.ExportFilter{EmployeeID: req.EmployeeId, StartDate: req.StartDate, EndDate: req.EndDate, Status: req.Status})
	if err != nil {
		return nil, err
	}
	return file(name, xlsxType, data), nil
}

// ─── ScheduleService ─────────────────────────────────────────

type scheduleServer struct {
	hrv1.UnimplementedScheduleServiceServer
	s *Services
}

func shiftInput(id string, in *hrv1.ShiftInput) domain.Shift {
	if in == nil {
		return domain.Shift{ID: id}
	}
	return domain.Shift{ID: id, Name: in.Name, StartTime: in.StartTime, EndTime: in.EndTime, BreakMinutes: int(in.BreakMinutes),
		LateToleranceMinutes: int(in.LateToleranceMinutes), IsOvernight: in.IsOvernight, IsActive: in.IsActive, SortOrder: int(in.SortOrder)}
}

func (h *scheduleServer) ListShifts(ctx context.Context, req *hrv1.ListShiftsRequest) (*hrv1.ListShiftsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Schedules.Shifts(ctx, a, req.IncludeInactive)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListShiftsResponse{}
	for _, r := range rows {
		out.Shifts = append(out.Shifts, shiftPB(r))
	}
	return out, nil
}

func (h *scheduleServer) CreateShift(ctx context.Context, req *hrv1.ShiftInput) (*hrv1.Shift, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	sh, err := h.s.Schedules.SaveShift(ctx, a, shiftInput("", req))
	if err != nil {
		return nil, err
	}
	return shiftPB(*sh), nil
}

func (h *scheduleServer) UpdateShift(ctx context.Context, req *hrv1.UpdateShiftRequest) (*hrv1.Shift, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	sh, err := h.s.Schedules.SaveShift(ctx, a, shiftInput(req.ShiftId, req.Input))
	if err != nil {
		return nil, err
	}
	return shiftPB(*sh), nil
}

func (h *scheduleServer) DeleteShift(ctx context.Context, req *hrv1.ShiftRequest) (*hrv1.ActionResult, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	msg, err := h.s.Schedules.DeleteShift(ctx, a, req.ShiftId)
	if err != nil {
		return nil, err
	}
	return &hrv1.ActionResult{Message: msg}, nil
}

func schedulePB(employeeID string, current, history []domain.ScheduleRow) *hrv1.EmployeeSchedule {
	out := &hrv1.EmployeeSchedule{EmployeeId: employeeID}
	for _, r := range current {
		out.Current = append(out.Current, scheduleDayPB(r))
	}
	for _, r := range history {
		out.History = append(out.History, scheduleDayPB(r))
	}
	return out
}

func (h *scheduleServer) GetEmployeeSchedule(ctx context.Context, req *hrv1.GetEmployeeRequest) (*hrv1.EmployeeSchedule, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	current, history, err := h.s.Schedules.EmployeeSchedule(ctx, a, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	return schedulePB(req.EmployeeId, current, history), nil
}

func (h *scheduleServer) SetEmployeeSchedule(ctx context.Context, req *hrv1.SetEmployeeScheduleRequest) (*hrv1.EmployeeSchedule, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	days := make([]domain.ScheduleRow, len(req.Days))
	for i, d := range req.Days {
		shift := d.ShiftId
		days[i] = domain.ScheduleRow{DayOfWeek: int(d.DayOfWeek), ShiftID: &shift}
	}
	if err := h.s.Schedules.SetSchedule(ctx, a, req.EmployeeId, req.EffectiveFrom, days); err != nil {
		return nil, err
	}
	current, history, err := h.s.Schedules.EmployeeSchedule(ctx, a, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	return schedulePB(req.EmployeeId, current, history), nil
}

func (h *scheduleServer) GetTeamSchedule(ctx context.Context, req *hrv1.GetTeamScheduleRequest) (*hrv1.TeamSchedule, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	monday, rows, err := h.s.Schedules.TeamSchedule(ctx, a, req.DepartmentId, req.WeekOf)
	if err != nil {
		return nil, err
	}
	return teamSchedulePB(monday, rows), nil
}

func (h *scheduleServer) ListHolidays(ctx context.Context, req *hrv1.ListHolidaysRequest) (*hrv1.ListHolidaysResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Schedules.Holidays(ctx, a, int(req.Year))
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListHolidaysResponse{}
	for _, r := range rows {
		out.Holidays = append(out.Holidays, holidayPB(r))
	}
	return out, nil
}

func holidayInput(in *hrv1.HolidayInput) app.HolidayInput {
	if in == nil {
		return app.HolidayInput{}
	}
	return app.HolidayInput{Date: in.HolidayDate, Name: in.Name, Type: in.Type, Status: in.Status, Note: in.Note, DeductsLeave: in.DeductsLeave}
}

func (h *scheduleServer) CreateHoliday(ctx context.Context, req *hrv1.HolidayInput) (*hrv1.Holiday, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	hd, err := h.s.Schedules.SaveHoliday(ctx, a, "", holidayInput(req))
	if err != nil {
		return nil, err
	}
	return holidayPB(*hd), nil
}

func (h *scheduleServer) UpdateHoliday(ctx context.Context, req *hrv1.UpdateHolidayRequest) (*hrv1.Holiday, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	hd, err := h.s.Schedules.SaveHoliday(ctx, a, req.HolidayId, holidayInput(req.Input))
	if err != nil {
		return nil, err
	}
	return holidayPB(*hd), nil
}

func (h *scheduleServer) DeleteHoliday(ctx context.Context, req *hrv1.HolidayRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.Schedules.DeleteHoliday(ctx, a, req.HolidayId)
}

// ─── LeaveService ────────────────────────────────────────────

type leaveServer struct {
	hrv1.UnimplementedLeaveServiceServer
	s *Services
}

func (s *Services) leavePB(ctx context.Context, a app.Actor, l domain.Leave, withExcluded bool) *hrv1.Leave {
	canDecide, canCancel := s.Leaves.Flags(a, l)
	out := &hrv1.Leave{Id: l.ID, Employee: refPB(l.EmployeeID, l.EmployeeName, l.EmployeeNo, l.DepartmentName), LeaveType: l.LeaveType,
		StartDate: l.StartDate, EndDate: l.EndDate, TotalDays: int32(l.TotalDays), Reason: l.Reason, Status: l.Status,
		ManagerApprovedByName: l.ManagerApprovedByName, ManagerApprovedAt: tsPtr(l.ManagerApprovedAt), ApprovedByName: l.ApprovedByName,
		ApprovedAt: tsPtr(l.ApprovedAt), RejectionReason: l.RejectionReason, CreatedAt: ts(l.CreatedAt), CanDecide: canDecide, CanCancel: canCancel}
	if withExcluded {
		out.ExcludedHolidays = s.Leaves.Excluded(ctx, l)
	}
	return out
}

func (s *Services) leaveList(ctx context.Context, a app.Actor, rows []domain.Leave, total int) *hrv1.ListLeavesResponse {
	out := &hrv1.ListLeavesResponse{Total: int32(total)}
	for _, r := range rows {
		out.Leaves = append(out.Leaves, s.leavePB(ctx, a, r, false))
	}
	return out
}

func leaveQuery(req *hrv1.ListLeavesRequest) app.LeaveQuery {
	limit, offset := pageOf(req.Page, req.PageSize)
	return app.LeaveQuery{EmployeeID: req.EmployeeId, Scope: req.Scope, LeaveFilter: domain.LeaveFilter{Status: req.Status,
		LeaveType: req.LeaveType, From: req.StartDate, To: req.EndDate, Limit: limit, Offset: offset}}
}

func (h *leaveServer) ListLeaveTypes(ctx context.Context, _ *emptypb.Empty) (*hrv1.ListLeaveTypesResponse, error) {
	if _, err := h.s.actor(ctx); err != nil {
		return nil, err
	}
	out := &hrv1.ListLeaveTypesResponse{}
	for _, t := range app.LeaveTypes {
		out.Types = append(out.Types, &hrv1.LeaveType{Code: t.Code, Name: t.Name, UsesAnnualQuota: t.Annual})
	}
	return out, nil
}

func (h *leaveServer) ListLeaves(ctx context.Context, req *hrv1.ListLeavesRequest) (*hrv1.ListLeavesResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, total, err := h.s.Leaves.List(ctx, a, leaveQuery(req))
	if err != nil {
		return nil, err
	}
	return h.s.leaveList(ctx, a, rows, total), nil
}

func (h *leaveServer) GetLeave(ctx context.Context, req *hrv1.LeaveRequest) (*hrv1.Leave, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	l, err := h.s.Leaves.Get(ctx, a, req.LeaveId)
	if err != nil {
		return nil, err
	}
	return h.s.leavePB(ctx, a, *l, true), nil
}

func (s *Services) createLeave(ctx context.Context, a app.Actor, req *hrv1.CreateLeaveRequest) (*hrv1.Leave, error) {
	l, err := s.Leaves.Create(ctx, a, app.LeaveInput{EmployeeID: req.EmployeeId, LeaveType: req.LeaveType, StartDate: req.StartDate, EndDate: req.EndDate, Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	return s.leavePB(ctx, a, *l, true), nil
}

func (h *leaveServer) CreateLeave(ctx context.Context, req *hrv1.CreateLeaveRequest) (*hrv1.Leave, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return h.s.createLeave(ctx, a, req)
}

func (h *leaveServer) DecideLeave(ctx context.Context, req *hrv1.DecideLeaveRequest) (*hrv1.Leave, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	l, err := h.s.Leaves.Decide(ctx, a, req.LeaveId, req.Approve, req.RejectionReason)
	if err != nil {
		return nil, err
	}
	return h.s.leavePB(ctx, a, *l, false), nil
}

func (s *Services) cancelLeave(ctx context.Context, a app.Actor, id string) (*hrv1.Leave, error) {
	l, err := s.Leaves.Cancel(ctx, a, id)
	if err != nil {
		return nil, err
	}
	return s.leavePB(ctx, a, *l, false), nil
}

func (h *leaveServer) CancelLeave(ctx context.Context, req *hrv1.LeaveRequest) (*hrv1.Leave, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return h.s.cancelLeave(ctx, a, req.LeaveId)
}

func (h *leaveServer) GetLeaveBalance(ctx context.Context, req *hrv1.GetLeaveBalanceRequest) (*hrv1.LeaveBalance, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	b, err := h.s.Leaves.Balance(ctx, a, req.EmployeeId, int(req.Year))
	if err != nil {
		return nil, err
	}
	return balancePB(b), nil
}

func (h *leaveServer) UpdateLeaveBalance(ctx context.Context, req *hrv1.UpdateLeaveBalanceRequest) (*hrv1.LeaveBalance, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	var used *int
	if req.AnnualLeaveUsed != nil {
		v := int(*req.AnnualLeaveUsed)
		used = &v
	}
	b, err := h.s.Leaves.UpdateBalance(ctx, a, req.EmployeeId, int(req.Year), int(req.AnnualLeaveTotal), used)
	if err != nil {
		return nil, err
	}
	return balancePB(b), nil
}

func (h *leaveServer) ExportLeaves(ctx context.Context, req *hrv1.ListLeavesRequest) (*hrv1.FileBytes, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	data, err := h.s.Leaves.ExportCSV(ctx, a, leaveQuery(req))
	if err != nil {
		return nil, err
	}
	return file("data-cuti-"+domain.DateOf(h.s.Deps.Clock.Now())+".csv", "text/csv; charset=utf-8", data), nil
}

// ─── OvertimeService ─────────────────────────────────────────

type overtimeServer struct {
	hrv1.UnimplementedOvertimeServiceServer
	s *Services
}

func (h *overtimeServer) ListOvertime(ctx context.Context, req *hrv1.ListOvertimeRequest) (*hrv1.ListOvertimeResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Overtime.List(ctx, a, app.OvertimeQuery{EmployeeID: req.EmployeeId, Status: req.Status, Month: req.Month, Scope: req.Scope})
	if err != nil {
		return nil, err
	}
	return overtimeList(rows), nil
}

func (s *Services) createOvertime(ctx context.Context, a app.Actor, req *hrv1.CreateOvertimeRequest) (*hrv1.Overtime, error) {
	o, err := s.Overtime.Create(ctx, a, app.OvertimeInput{EmployeeID: req.EmployeeId, Date: req.Date, StartTime: req.StartTime, EndTime: req.EndTime, Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	return overtimePB(*o), nil
}

func (h *overtimeServer) CreateOvertime(ctx context.Context, req *hrv1.CreateOvertimeRequest) (*hrv1.Overtime, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return h.s.createOvertime(ctx, a, req)
}

func (s *Services) decideOvertime(ctx context.Context, a app.Actor, req *hrv1.DecideOvertimeRequest) (*hrv1.Overtime, error) {
	o, err := s.Overtime.Decide(ctx, a, req.OvertimeId, req.Action, req.RejectionReason)
	if err != nil {
		return nil, err
	}
	return overtimePB(*o), nil
}

func (h *overtimeServer) DecideOvertime(ctx context.Context, req *hrv1.DecideOvertimeRequest) (*hrv1.Overtime, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return h.s.decideOvertime(ctx, a, req)
}
