package grpcapi

import (
	"context"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ─── ContractService ─────────────────────────────────────────

type contractServer struct {
	hrv1.UnimplementedContractServiceServer
	s *Services
}

func (h *contractServer) today() string { return hris.TodayWIB(h.s.Deps.Clock.Now()) }

func contractInput(in *hrv1.ContractInput) app.ContractInput {
	if in == nil {
		return app.ContractInput{}
	}
	return app.ContractInput{EmployeeID: in.EmployeeId, ContractType: in.ContractType, StartDate: in.StartDate, EndDate: in.EndDate,
		ProbationEndDate: in.ProbationEndDate, PositionTitle: in.PositionTitle, DepartmentName: in.DepartmentName,
		WorkLocation: in.WorkLocation, BaseSalary: in.BaseSalary, Notes: in.Notes}
}

func (h *contractServer) ListContracts(ctx context.Context, req *hrv1.ListContractsRequest) (*hrv1.ListContractsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Contracts.List(ctx, a, domain.ContractFilter{EmployeeID: req.EmployeeId, Status: req.Status, ContractType: req.ContractType, Search: req.Search})
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListContractsResponse{}
	today := h.today()
	for _, r := range rows {
		out.Contracts = append(out.Contracts, contractPB(r, today))
	}
	return out, nil
}

func (h *contractServer) ListExpiringContracts(ctx context.Context, req *hrv1.ListExpiringContractsRequest) (*hrv1.ExpiringContracts, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	ex, err := h.s.Contracts.Expiring(ctx, a, int(req.Days))
	if err != nil {
		return nil, err
	}
	today := h.today()
	out := &hrv1.ExpiringContracts{Days: int32(ex.Days)}
	for _, c := range ex.Contracts {
		out.Contracts = append(out.Contracts, contractPB(c, today))
	}
	for _, e := range ex.Probations {
		p := &hrv1.ProbationEnding{Employee: refPB(e.ID, e.FullName, e.EmployeeNo, e.DepartmentName), ProbationEndDate: str(e.ProbationEndDate)}
		if c := contractPB(domain.Contract{EndDate: e.ProbationEndDate}, today); c.DaysLeft != nil {
			p.DaysLeft = *c.DaysLeft
		}
		out.Probations = append(out.Probations, p)
	}
	for _, e := range ex.WithoutContract {
		out.WithoutContract = append(out.WithoutContract, refPB(e.ID, e.FullName, e.EmployeeNo, e.DepartmentName))
	}
	return out, nil
}

func (h *contractServer) GetContract(ctx context.Context, req *hrv1.ContractRequest) (*hrv1.Contract, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.s.Contracts.Get(ctx, a, req.ContractId)
	if err != nil {
		return nil, err
	}
	return contractPB(*c, h.today()), nil
}

func (h *contractServer) CreateContract(ctx context.Context, req *hrv1.ContractInput) (*hrv1.Contract, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.s.Contracts.Create(ctx, a, contractInput(req))
	if err != nil {
		return nil, err
	}
	return contractPB(*c, h.today()), nil
}

func (h *contractServer) UpdateContract(ctx context.Context, req *hrv1.UpdateContractRequest) (*hrv1.Contract, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	c, err := h.s.Contracts.UpdateDraft(ctx, a, req.ContractId, contractInput(req.Input))
	if err != nil {
		return nil, err
	}
	return contractPB(*c, h.today()), nil
}

func (h *contractServer) ContractAction(ctx context.Context, req *hrv1.ContractActionRequest) (*hrv1.ContractActionResult, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	res, err := h.s.Contracts.Action(ctx, a, req.ContractId, app.ContractActionInput{Action: req.Action, EndDate: req.EndDate, Reason: req.Reason, SignedAt: req.SignedAt})
	if err != nil {
		return nil, err
	}
	out := &hrv1.ContractActionResult{Message: res.Message, CompensationAmount: moneyPtr(res.Compensation)}
	if res.Contract != nil {
		out.Contract = contractPB(*res.Contract, h.today())
	}
	return out, nil
}

func (h *contractServer) DeleteContract(ctx context.Context, req *hrv1.ContractRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.Contracts.Delete(ctx, a, req.ContractId)
}

// ─── AnnouncementService ─────────────────────────────────────

type announcementServer struct {
	hrv1.UnimplementedAnnouncementServiceServer
	s *Services
}

func (h *announcementServer) ListAnnouncements(ctx context.Context, req *hrv1.ListAnnouncementsRequest) (*hrv1.ListAnnouncementsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Announcements.List(ctx, a, req.Status)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListAnnouncementsResponse{}
	for _, r := range rows {
		out.Announcements = append(out.Announcements, announcementPB(r))
	}
	return out, nil
}

func (h *announcementServer) GetAnnouncement(ctx context.Context, req *hrv1.AnnouncementRequest) (*hrv1.Announcement, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	an, err := h.s.Announcements.Get(ctx, a, req.AnnouncementId)
	if err != nil {
		return nil, err
	}
	return announcementPB(*an), nil
}

func (h *announcementServer) CreateAnnouncement(ctx context.Context, req *hrv1.AnnouncementInput) (*hrv1.Announcement, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	an, err := h.s.Announcements.Save(ctx, a, announcementInput("", req))
	if err != nil {
		return nil, err
	}
	return announcementPB(*an), nil
}

func (h *announcementServer) UpdateAnnouncement(ctx context.Context, req *hrv1.UpdateAnnouncementRequest) (*hrv1.Announcement, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	an, err := h.s.Announcements.Save(ctx, a, announcementInput(req.AnnouncementId, req.Input))
	if err != nil {
		return nil, err
	}
	return announcementPB(*an), nil
}

func (h *announcementServer) DeleteAnnouncement(ctx context.Context, req *hrv1.AnnouncementRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.Announcements.Delete(ctx, a, req.AnnouncementId)
}

// ─── LifecycleService ────────────────────────────────────────

type lifecycleServer struct {
	hrv1.UnimplementedLifecycleServiceServer
	s *Services
}

func (h *lifecycleServer) ListOnboardingTasks(ctx context.Context, req *hrv1.GetEmployeeRequest) (*hrv1.OnboardingChecklist, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Lifecycle.Tasks(ctx, a, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	return checklistPB(req.EmployeeId, rows), nil
}

func (h *lifecycleServer) ApplyOnboardingTemplate(ctx context.Context, req *hrv1.GetEmployeeRequest) (*hrv1.OnboardingChecklist, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Lifecycle.ApplyTemplate(ctx, a, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	return checklistPB(req.EmployeeId, rows), nil
}

func (h *lifecycleServer) AddOnboardingTask(ctx context.Context, req *hrv1.OnboardingTaskInput) (*hrv1.OnboardingTask, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	var due *string
	if req.DueDate != "" {
		due = &req.DueDate
	}
	t, err := h.s.Lifecycle.AddTask(ctx, a, domain.OnboardingTask{EmployeeID: req.EmployeeId, TaskName: req.TaskName, Category: req.Category,
		Priority: int(req.Priority), DueDate: due})
	if err != nil {
		return nil, err
	}
	return taskPB(*t), nil
}

func (h *lifecycleServer) UpdateOnboardingTask(ctx context.Context, req *hrv1.UpdateOnboardingTaskRequest) (*hrv1.OnboardingTask, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.s.Lifecycle.UpdateTask(ctx, a, req.TaskId, app.TaskUpdate{IsCompleted: req.IsCompleted, Notes: req.Notes, TaskName: req.TaskName, DueDate: req.DueDate})
	if err != nil {
		return nil, err
	}
	return taskPB(*t), nil
}

func (h *lifecycleServer) DeleteOnboardingTask(ctx context.Context, req *hrv1.OnboardingTaskRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.Lifecycle.DeleteTask(ctx, a, req.TaskId)
}

func (h *lifecycleServer) GetOffboarding(ctx context.Context, req *hrv1.GetEmployeeRequest) (*hrv1.Offboarding, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := h.s.Lifecycle.Offboarding(ctx, a, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	return offboardingPB(*o), nil
}

func (h *lifecycleServer) StartOffboarding(ctx context.Context, req *hrv1.StartOffboardingRequest) (*hrv1.Offboarding, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := h.s.Lifecycle.StartOffboarding(ctx, a, domain.Offboarding{EmployeeID: req.EmployeeId, ResignationType: req.ResignationType,
		ResignationDate: req.ResignationDate, LastWorkingDay: req.LastWorkingDay, Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	return offboardingPB(*o), nil
}

func (h *lifecycleServer) UpdateOffboarding(ctx context.Context, req *hrv1.UpdateOffboardingRequest) (*hrv1.Offboarding, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	o, err := h.s.Lifecycle.UpdateOffboarding(ctx, a, req.EmployeeId, app.OffboardingUpdate{Status: req.Status, ClearanceType: req.ClearanceType,
		ExitInterviewNotes: req.ExitInterviewNotes, Notes: req.Notes, Cleared: req.Cleared})
	if err != nil {
		return nil, err
	}
	return offboardingPB(*o), nil
}

// ─── HrReportService ─────────────────────────────────────────

type reportServer struct {
	hrv1.UnimplementedHrReportServiceServer
	s *Services
}

func (h *reportServer) GetHrisReport(ctx context.Context, req *hrv1.HrisReportRequest) (*hrv1.HrisReport, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.s.Reports.Monthly(ctx, a, int(req.Month), int(req.Year))
	if err != nil {
		return nil, err
	}
	return reportPB(r), nil
}

func (h *reportServer) ExportHrisReport(ctx context.Context, req *hrv1.HrisReportRequest) (*hrv1.FileBytes, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	data, name, err := h.s.Reports.MonthlyCSV(ctx, a, int(req.Month), int(req.Year))
	if err != nil {
		return nil, err
	}
	return file(name, "text/csv; charset=utf-8", data), nil
}
