package grpcapi

import (
	"context"
	"fmt"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/protobuf/types/known/emptypb"
)

// selfServer serves /v1/hr/me: every call is pinned to the caller's own
// employee record (self.service), whatever ids the body carries.
type selfServer struct {
	hrv1.UnimplementedSelfServiceServiceServer
	s *Services
}

// me resolves the caller and requires a linked employee.
func (h *selfServer) me(ctx context.Context) (app.Actor, string, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return a, "", err
	}
	id, err := app.RequireSelf(a)
	return a, id, err
}

func (h *selfServer) GetMyProfile(ctx context.Context, _ *emptypb.Empty) (*hrv1.EmployeeProfile, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	e, err := h.s.Self.Profile(ctx, a)
	if err != nil {
		return nil, err
	}
	out := profilePB(*e)
	out.UserEmail = a.Email
	return out, nil
}

func (h *selfServer) GetMyHome(ctx context.Context, _ *emptypb.Empty) (*hrv1.MyHome, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	home, err := h.s.Self.Home(ctx, a)
	if err != nil {
		return nil, err
	}
	e := home.Employee
	out := &hrv1.MyHome{Employee: refPB(e.ID, e.FullName, e.EmployeeNo, e.DepartmentName), Today: home.Today, Week: teamDaysPB(home.Week),
		LeaveBalance: balancePB(home.Balance), UnreadAnnouncements: int32(home.UnreadAnnouncements),
		PendingApprovals: int32(home.PendingApprovals), HolidayName: home.HolidayName}
	if t := home.TodayShift; t != nil {
		out.TodayShift = &hrv1.TodayShift{Name: t.Name, StartTime: domain.HHMM(str(t.StartTime)), EndTime: domain.HHMM(str(t.EndTime)), LateToleranceMinutes: int32(t.LateToleranceMinutes)}
	}
	if home.TodayAttendance != nil {
		out.TodayAttendance = attendancePB(*home.TodayAttendance)
	}
	return out, nil
}

func (h *selfServer) MyClockIn(ctx context.Context, req *hrv1.ClockInRequest) (*hrv1.AttendanceRecord, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	in := clockIn(req)
	in.EmployeeID, in.Date = id, ""
	r, err := h.s.Attendance.ClockIn(ctx, a, in)
	if err != nil {
		return nil, err
	}
	return attendancePB(*r), nil
}

func (h *selfServer) MyClockOut(ctx context.Context, req *hrv1.ClockOutRequest) (*hrv1.AttendanceRecord, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	att, err := h.s.Deps.Store.GetAttendance(ctx, req.AttendanceId)
	if err != nil {
		return nil, err
	}
	if att == nil || att.EmployeeID != id {
		return nil, notFoundAttendance
	}
	r, err := h.s.Attendance.ClockOut(ctx, a, clockOut(req))
	if err != nil {
		return nil, err
	}
	return attendancePB(*r), nil
}

func (h *selfServer) ListMyAttendance(ctx context.Context, req *hrv1.ListMyRequest) (*hrv1.ListAttendanceResponse, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	from, to, err := app.MonthOrRecent(req.Month, h.s.Deps.Clock.Now())
	if err != nil {
		return nil, err
	}
	rows, total, err := h.s.Attendance.List(ctx, a, domain.AttendanceFilter{EmployeeIDs: []string{id}, From: from, To: to, Status: req.Status})
	if err != nil {
		return nil, err
	}
	return attendanceList(rows, total), nil
}

func (h *selfServer) ListMyLeaves(ctx context.Context, req *hrv1.ListMyRequest) (*hrv1.ListLeavesResponse, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	q := app.LeaveQuery{EmployeeID: id, LeaveFilter: domain.LeaveFilter{Status: req.Status}}
	if req.Year != 0 {
		q.From, q.To = fmt.Sprintf("%d-01-01", req.Year), fmt.Sprintf("%d-12-31", req.Year)
	}
	rows, total, err := h.s.Leaves.List(ctx, selfOnly(a), q)
	if err != nil {
		return nil, err
	}
	return h.s.leaveList(ctx, a, rows, total), nil
}

func (h *selfServer) CreateMyLeave(ctx context.Context, req *hrv1.CreateLeaveRequest) (*hrv1.Leave, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	req.EmployeeId = id
	return h.s.createLeave(ctx, a, req)
}

func (h *selfServer) CancelMyLeave(ctx context.Context, req *hrv1.LeaveRequest) (*hrv1.Leave, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	l, err := h.s.Deps.Store.GetLeave(ctx, req.LeaveId)
	if err != nil {
		return nil, err
	}
	if l == nil || l.EmployeeID != id {
		return nil, notFoundLeave
	}
	return h.s.cancelLeave(ctx, a, req.LeaveId)
}

func (h *selfServer) GetMyLeaveBalance(ctx context.Context, req *hrv1.ListMyRequest) (*hrv1.LeaveBalance, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	b, err := h.s.Leaves.Balance(ctx, a, id, int(req.Year))
	if err != nil {
		return nil, err
	}
	return balancePB(b), nil
}

func (h *selfServer) ListMyOvertime(ctx context.Context, req *hrv1.ListMyRequest) (*hrv1.ListOvertimeResponse, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Overtime.List(ctx, selfOnly(a), app.OvertimeQuery{Status: req.Status, Month: req.Month})
	if err != nil {
		return nil, err
	}
	return overtimeList(rows), nil
}

func (h *selfServer) CreateMyOvertime(ctx context.Context, req *hrv1.CreateOvertimeRequest) (*hrv1.Overtime, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	req.EmployeeId = id
	return h.s.createOvertime(ctx, a, req)
}

func (h *selfServer) DecideMyOvertime(ctx context.Context, req *hrv1.DecideOvertimeRequest) (*hrv1.Overtime, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	o, err := h.s.Deps.Store.GetOvertime(ctx, req.OvertimeId)
	if err != nil {
		return nil, err
	}
	if o == nil || o.EmployeeID != id {
		return nil, notFoundOvertime
	}
	return h.s.decideOvertime(ctx, a, req)
}

func (h *selfServer) ListMyLoans(ctx context.Context, req *hrv1.ListMyRequest) (*hrv1.ListLoansResponse, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Loans.List(ctx, a, id, req.Status)
	if err != nil {
		return nil, err
	}
	return loanList(rows), nil
}

func (h *selfServer) CreateMyLoan(ctx context.Context, req *hrv1.CreateLoanRequest) (*hrv1.Loan, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	req.EmployeeId, req.InterestRate = "", ""
	return h.s.createLoan(ctx, selfOnly(a), req)
}

// CancelMyLoan scopes to the caller's own loans.
func (h *selfServer) CancelMyLoan(ctx context.Context, req *hrv1.LoanRequest) (*hrv1.Loan, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	return h.s.cancelLoan(ctx, selfOnly(a), req.LoanId)
}

func (h *selfServer) ListMyPayslips(ctx context.Context, req *hrv1.ListMyRequest) (*hrv1.ListPayslipsResponse, error) {
	a, id, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Payroll.Payslips(ctx, a, domain.PayslipFilter{EmployeeID: id, Year: int(req.Year)})
	if err != nil {
		return nil, err
	}
	return payslipList(rows), nil
}

func (h *selfServer) GetMyPayslipPdf(ctx context.Context, req *hrv1.PayslipRequest) (*hrv1.FileBytes, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	return h.s.payslipPDF(ctx, selfOnly(a), req.PayslipId)
}

func (h *selfServer) GetMyTeamSchedule(ctx context.Context, req *hrv1.GetTeamScheduleRequest) (*hrv1.TeamSchedule, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	monday, rows, err := h.s.Schedules.TeamSchedule(ctx, selfOnly(a), "", req.WeekOf)
	if err != nil {
		return nil, err
	}
	return teamSchedulePB(monday, rows), nil
}

func (h *selfServer) ListMyAnnouncements(ctx context.Context, _ *emptypb.Empty) (*hrv1.ListAnnouncementsResponse, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	rows, unread, err := h.s.Announcements.Feed(ctx, a)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListAnnouncementsResponse{Unread: int32(unread)}
	for _, r := range rows {
		out.Announcements = append(out.Announcements, announcementPB(r))
	}
	return out, nil
}

func (h *selfServer) GetMyAnnouncement(ctx context.Context, req *hrv1.AnnouncementRequest) (*hrv1.Announcement, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	an, err := h.s.Announcements.Open(ctx, a, req.AnnouncementId)
	if err != nil {
		return nil, err
	}
	return announcementPB(*an), nil
}

func (h *selfServer) ListMyApprovals(ctx context.Context, _ *emptypb.Empty) (*hrv1.MyApprovals, error) {
	a, _, err := h.me(ctx)
	if err != nil {
		return nil, err
	}
	ap, err := h.s.Self.Approvals(ctx, a)
	if err != nil {
		return nil, err
	}
	out := &hrv1.MyApprovals{}
	for _, l := range ap.Leaves {
		out.Leaves = append(out.Leaves, h.s.leavePB(ctx, a, l, false))
	}
	for _, o := range ap.Overtime {
		out.Overtime = append(out.Overtime, overtimePB(o))
	}
	return out, nil
}

// selfOnly drops every permission but self.service so the shared use
// cases scope to the caller's own rows.
func selfOnly(a app.Actor) app.Actor {
	a.Role, a.Perms = authn.RoleMember, []string{authn.PermSelfService}
	return a
}

var (
	notFoundAttendance = apperr.New(apperr.NotFound, "Data absensi tidak ditemukan.")
	notFoundLeave      = apperr.New(apperr.NotFound, "Pengajuan cuti tidak ditemukan.")
	notFoundOvertime   = apperr.New(apperr.NotFound, "Pengajuan lembur tidak ditemukan.")
)
