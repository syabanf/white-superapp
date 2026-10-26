package grpcapi

import (
	"context"
	"strconv"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Services struct {
	Deps          app.Deps
	People        *app.People
	Attendance    *app.Attendance
	Schedules     *app.Schedules
	Leaves        *app.Leaves
	Overtime      *app.Overtime
	Contracts     *app.Contracts
	Announcements *app.Announcements
	Lifecycle     *app.Lifecycle
	Payroll       *app.Payroll
	Loans         *app.Loans
	Self          *app.SelfService
	Reports       *app.Reports
	Dashboard     *app.Dashboard
}

func Register(s *grpc.Server, svc *Services) {
	hrv1.RegisterEmployeeServiceServer(s, &employeeServer{s: svc})
	hrv1.RegisterPeopleServiceServer(s, &peopleServer{s: svc})
	hrv1.RegisterAttendanceServiceServer(s, &attendanceServer{s: svc})
	hrv1.RegisterScheduleServiceServer(s, &scheduleServer{s: svc})
	hrv1.RegisterLeaveServiceServer(s, &leaveServer{s: svc})
	hrv1.RegisterOvertimeServiceServer(s, &overtimeServer{s: svc})
	hrv1.RegisterContractServiceServer(s, &contractServer{s: svc})
	hrv1.RegisterAnnouncementServiceServer(s, &announcementServer{s: svc})
	hrv1.RegisterLifecycleServiceServer(s, &lifecycleServer{s: svc})
	hrv1.RegisterPayrollServiceServer(s, &payrollServer{s: svc})
	hrv1.RegisterSelfServiceServiceServer(s, &selfServer{s: svc})
	hrv1.RegisterHrReportServiceServer(s, &reportServer{s: svc})
}

var empty = &emptypb.Empty{}

// actor resolves the signed-in caller and their employee record.
func (s *Services) actor(ctx context.Context) (app.Actor, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return app.Actor{}, err
	}
	return s.Deps.ActorFor(ctx, p)
}

// directory trusts internal service-to-service calls (no x-white-edge
// marker, with or without a forwarded user token); calls through the
// gateway need hr.read.
func directory(ctx context.Context) error {
	if !grpcx.FromEdge(ctx) {
		return nil
	}
	_, err := authn.RequirePerm(ctx, authn.PermHRRead)
	return err
}

func file(name, contentType string, data []byte) *hrv1.FileBytes {
	return &hrv1.FileBytes{FileName: name, ContentType: contentType, Data: data}
}

func pageOf(page, size int32) (limit, offset int) {
	limit = int(size)
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if page < 1 {
		page = 1
	}
	return limit, (int(page) - 1) * limit
}

// ─── EmployeeService ─────────────────────────────────────────

type employeeServer struct {
	hrv1.UnimplementedEmployeeServiceServer
	s *Services
}

func (h *employeeServer) ListEmployees(ctx context.Context, req *hrv1.ListEmployeesRequest) (*hrv1.ListEmployeesResponse, error) {
	if err := directory(ctx); err != nil {
		return nil, err
	}
	// Without paging every match comes back (other services read the
	// whole directory); page_token is the page number.
	limit, offset := 1000, 0
	if req.PageSize > 0 || req.PageToken != "" {
		page := 1
		if req.PageToken != "" {
			n, err := strconv.Atoi(req.PageToken)
			if err != nil || n < 1 {
				return nil, apperr.Field("pageToken", "Token halaman tidak valid.")
			}
			page = n
		}
		limit, offset = pageOf(int32(page), req.PageSize)
	}
	rows, total, err := h.s.People.List(ctx, domain.EmployeeFilter{DepartmentID: req.DepartmentId, ManagerID: req.ManagerId,
		IDs: req.EmployeeIds, IncludeInactive: req.IncludeInactive, Search: req.Search, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListEmployeesResponse{Total: int32(total)}
	for _, e := range rows {
		out.Employees = append(out.Employees, employeePB(e))
	}
	if offset+len(rows) < total {
		out.NextPageToken = strconv.Itoa(offset/limit + 2)
	}
	return out, nil
}

func (h *employeeServer) GetEmployee(ctx context.Context, req *hrv1.GetEmployeeRequest) (*hrv1.Employee, error) {
	if err := directory(ctx); err != nil {
		return nil, err
	}
	e, err := h.s.People.Get(ctx, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	return employeePB(*e), nil
}

func (h *employeeServer) ListDepartments(ctx context.Context, req *hrv1.ListDepartmentsRequest) (*hrv1.ListDepartmentsResponse, error) {
	if err := directory(ctx); err != nil {
		return nil, err
	}
	rows, err := h.s.People.Departments(ctx, req.IncludeInactive)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListDepartmentsResponse{}
	for _, d := range rows {
		out.Departments = append(out.Departments, departmentPB(d))
	}
	return out, nil
}

// GetEmployeeByUser is internal: it trusts its parameter.
func (h *employeeServer) GetEmployeeByUser(ctx context.Context, req *hrv1.GetEmployeeByUserRequest) (*hrv1.Employee, error) {
	e, err := h.s.People.ByUser(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return employeePB(*e), nil
}

// GetAttendanceSummary is internal (KPI auto-collect).
func (h *employeeServer) GetAttendanceSummary(ctx context.Context, req *hrv1.GetAttendanceSummaryRequest) (*hrv1.GetAttendanceSummaryResponse, error) {
	rows, err := h.s.People.AttendanceSummary(ctx, req.EmployeeIds, req.Month)
	if err != nil {
		return nil, err
	}
	out := &hrv1.GetAttendanceSummaryResponse{}
	for _, r := range rows {
		out.Summaries = append(out.Summaries, &hrv1.AttendanceSummary{EmployeeId: r.EmployeeID, WorkingDays: int32(r.WorkingDays),
			PresentDays: int32(r.PresentDays), LateDays: int32(r.LateDays), AbsentDays: int32(r.AbsentDays), LeaveDays: int32(r.LeaveDays),
			LateMinutes: int32(r.LateMinutes), OvertimeHours: r.OvertimeHours})
	}
	return out, nil
}

// ─── PeopleService ───────────────────────────────────────────

type peopleServer struct {
	hrv1.UnimplementedPeopleServiceServer
	s *Services
}

func (h *peopleServer) profile(ctx context.Context, e *domain.Employee) (*hrv1.EmployeeProfile, error) {
	out := profilePB(*e)
	if e.UserID != nil {
		if u, err := h.s.Deps.Identity.GetUser(ctx, *e.UserID); err == nil && u != nil {
			out.UserEmail = u.Email
		}
	}
	return out, nil
}

func (h *peopleServer) GetEmployeeProfile(ctx context.Context, req *hrv1.GetEmployeeRequest) (*hrv1.EmployeeProfile, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if req.EmployeeId != a.EmployeeID || !a.Can(authn.PermSelfService) {
		if !a.Can(authn.PermHRRead) {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
	}
	e, err := h.s.People.Get(ctx, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	return h.profile(ctx, e)
}

func (h *peopleServer) CreateEmployee(ctx context.Context, req *hrv1.EmployeeInput) (*hrv1.EmployeeProfile, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	e, err := h.s.People.Create(ctx, a, employeeInput(req))
	if err != nil {
		return nil, err
	}
	return h.profile(ctx, e)
}

func (h *peopleServer) UpdateEmployee(ctx context.Context, req *hrv1.UpdateEmployeeRequest) (*hrv1.EmployeeProfile, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	e, err := h.s.People.Update(ctx, a, req.EmployeeId, employeeInput(req.Input))
	if err != nil {
		return nil, err
	}
	return h.profile(ctx, e)
}

func (h *peopleServer) DeleteEmployee(ctx context.Context, req *hrv1.GetEmployeeRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.People.Delete(ctx, a, req.EmployeeId)
}

func (h *peopleServer) LinkEmployeeUser(ctx context.Context, req *hrv1.LinkEmployeeUserRequest) (*hrv1.EmployeeProfile, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	e, err := h.s.People.LinkUser(ctx, a, req.EmployeeId, req.UserId)
	if err != nil {
		return nil, err
	}
	return h.profile(ctx, e)
}

func (h *peopleServer) ListEmploymentHistory(ctx context.Context, req *hrv1.GetEmployeeRequest) (*hrv1.ListEmploymentHistoryResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.People.History(ctx, a, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListEmploymentHistoryResponse{}
	for _, r := range rows {
		out.Entries = append(out.Entries, &hrv1.EmploymentHistoryEntry{Id: r.ID, ChangeType: r.ChangeType, Notes: r.Notes,
			PrevEmploymentStatus: str(r.PrevStatus), NewEmploymentStatus: str(r.NewStatus), PrevDepartment: r.PrevDepartment,
			NewDepartment: r.NewDepartment, PrevPosition: r.PrevPosition, NewPosition: r.NewPosition, CreatedAt: ts(r.CreatedAt)})
	}
	return out, nil
}

func (h *peopleServer) ListEmployeeDocuments(ctx context.Context, req *hrv1.GetEmployeeRequest) (*hrv1.ListEmployeeDocumentsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.People.Documents(ctx, a, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListEmployeeDocumentsResponse{}
	for _, d := range rows {
		out.Documents = append(out.Documents, documentPB(d))
	}
	return out, nil
}

func (h *peopleServer) UploadEmployeeDocument(ctx context.Context, req *hrv1.UploadEmployeeDocumentRequest) (*hrv1.EmployeeDocument, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	d, err := h.s.People.UploadDocument(ctx, a, req.EmployeeId, req.DocType, req.Name, req.Data)
	if err != nil {
		return nil, err
	}
	return documentPB(*d), nil
}

func (h *peopleServer) DownloadEmployeeDocument(ctx context.Context, req *hrv1.DocumentRequest) (*hrv1.FileBytes, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	d, data, err := h.s.People.DownloadDocument(ctx, a, req.DocumentId)
	if err != nil {
		return nil, err
	}
	return file(d.Name, d.ContentType, data), nil
}

func (h *peopleServer) DeleteEmployeeDocument(ctx context.Context, req *hrv1.DocumentRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.People.DeleteDocument(ctx, a, req.DocumentId)
}

func (h *peopleServer) GetOrgChart(ctx context.Context, req *hrv1.GetOrgChartRequest) (*hrv1.OrgChart, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	roots, total, err := h.s.People.OrgChart(ctx, a, req.RootEmployeeId)
	if err != nil {
		return nil, err
	}
	out := &hrv1.OrgChart{Total: int32(total)}
	for _, r := range roots {
		out.Roots = append(out.Roots, orgNodePB(r))
	}
	return out, nil
}

func departmentInput(in *hrv1.DepartmentInput) app.DepartmentInput {
	if in == nil {
		return app.DepartmentInput{}
	}
	return app.DepartmentInput{Code: in.Code, Name: in.Name, ParentID: in.ParentId, HeadEmployeeID: in.HeadEmployeeId, IsActive: in.IsActive}
}

func (h *peopleServer) CreateDepartment(ctx context.Context, req *hrv1.DepartmentInput) (*hrv1.Department, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	d, err := h.s.People.SaveDepartment(ctx, a, "", departmentInput(req))
	if err != nil {
		return nil, err
	}
	return departmentPB(*d), nil
}

func (h *peopleServer) UpdateDepartment(ctx context.Context, req *hrv1.UpdateDepartmentRequest) (*hrv1.Department, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	d, err := h.s.People.SaveDepartment(ctx, a, req.DepartmentId, departmentInput(req.Input))
	if err != nil {
		return nil, err
	}
	return departmentPB(*d), nil
}

func (h *peopleServer) DeleteDepartment(ctx context.Context, req *hrv1.DepartmentRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.People.DeleteDepartment(ctx, a, req.DepartmentId)
}

func (h *peopleServer) ListPositions(ctx context.Context, req *hrv1.ListPositionsRequest) (*hrv1.ListPositionsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.People.Positions(ctx, a, req.DepartmentId, req.IncludeInactive)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListPositionsResponse{}
	for _, p := range rows {
		out.Positions = append(out.Positions, positionPB(p))
	}
	return out, nil
}

func positionInput(in *hrv1.PositionInput) app.PositionInput {
	if in == nil {
		return app.PositionInput{}
	}
	return app.PositionInput{Title: in.Title, DepartmentID: in.DepartmentId, Level: in.Level, SalaryMin: in.SalaryMin, SalaryMax: in.SalaryMax, IsActive: in.IsActive}
}

func (h *peopleServer) CreatePosition(ctx context.Context, req *hrv1.PositionInput) (*hrv1.Position, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	p, err := h.s.People.SavePosition(ctx, a, "", positionInput(req))
	if err != nil {
		return nil, err
	}
	return positionPB(*p), nil
}

func (h *peopleServer) UpdatePosition(ctx context.Context, req *hrv1.UpdatePositionRequest) (*hrv1.Position, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	p, err := h.s.People.SavePosition(ctx, a, req.PositionId, positionInput(req.Input))
	if err != nil {
		return nil, err
	}
	return positionPB(*p), nil
}

func (h *peopleServer) DeletePosition(ctx context.Context, req *hrv1.PositionRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.People.DeletePosition(ctx, a, req.PositionId)
}

func (h *peopleServer) ListEmploymentStatuses(ctx context.Context, _ *emptypb.Empty) (*hrv1.ListEmploymentStatusesResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.People.EmploymentStatuses(ctx, a)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListEmploymentStatusesResponse{}
	for _, r := range rows {
		out.Statuses = append(out.Statuses, &hrv1.EmploymentStatusOption{Code: r.Code, Name: r.Name, IsActive: r.IsActive, SortOrder: int32(r.SortOrder)})
	}
	return out, nil
}

func (h *peopleServer) UpsertEmploymentStatus(ctx context.Context, req *hrv1.EmploymentStatusOption) (*hrv1.EmploymentStatusOption, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := h.s.People.UpsertEmploymentStatus(ctx, a, domain.EmploymentStatusOption{Code: req.Code, Name: req.Name, IsActive: req.IsActive, SortOrder: int(req.SortOrder)})
	if err != nil {
		return nil, err
	}
	return &hrv1.EmploymentStatusOption{Code: o.Code, Name: o.Name, IsActive: o.IsActive, SortOrder: int32(o.SortOrder)}, nil
}

func (h *peopleServer) DeleteEmploymentStatus(ctx context.Context, req *hrv1.EmploymentStatusRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.People.DeleteEmploymentStatus(ctx, a, req.Code)
}
