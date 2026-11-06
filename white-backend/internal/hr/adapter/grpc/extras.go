package grpcapi

import (
	"context"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetProjectCosts is internal: salaries never leave HR, only per-project totals.
func (h *employeeServer) GetProjectCosts(ctx context.Context, req *hrv1.GetProjectCostsRequest) (*hrv1.GetProjectCostsResponse, error) {
	if grpcx.FromEdge(ctx) {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	rows, err := h.s.Timesheet.ProjectCosts(ctx, req.Month)
	if err != nil {
		return nil, err
	}
	out := &hrv1.GetProjectCostsResponse{}
	for _, r := range rows {
		out.Costs = append(out.Costs, &hrv1.ProjectCost{ProjectId: r.ProjectID, ProjectName: r.ProjectName, Hours: r.Hours, Cost: money(r.Cost)})
	}
	return out, nil
}

// ─── AssetService ────────────────────────────────────────────

type assetServer struct {
	hrv1.UnimplementedAssetServiceServer
	s *Services
}

func assetPB(a domain.Asset) *hrv1.Asset {
	out := &hrv1.Asset{Id: a.ID, AssetCode: a.AssetCode, Name: a.Name, Category: a.Category, SerialNo: a.SerialNo, PurchaseDate: str(a.PurchaseDate),
		PurchaseValue: money(a.PurchaseValue), Status: a.Status, Notes: a.Notes}
	if a.EmployeeID != nil {
		out.Holder = refPB(*a.EmployeeID, a.EmployeeName, a.EmployeeNo, "")
	}
	return out
}

func (h *assetServer) ListAssets(ctx context.Context, req *hrv1.ListAssetsRequest) (*hrv1.ListAssetsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Assets.List(ctx, a, domain.AssetFilter{EmployeeID: req.EmployeeId, Status: req.Status, Search: req.Search})
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListAssetsResponse{}
	for _, r := range rows {
		out.Assets = append(out.Assets, assetPB(r))
	}
	return out, nil
}

func (h *assetServer) GetAsset(ctx context.Context, req *hrv1.AssetRequest) (*hrv1.AssetDetail, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	asset, history, err := h.s.Assets.Get(ctx, a, req.Id)
	if err != nil {
		return nil, err
	}
	out := &hrv1.AssetDetail{Asset: assetPB(*asset)}
	for _, x := range history {
		out.History = append(out.History, &hrv1.AssetAssignment{Id: x.ID, Employee: refPB(x.EmployeeID, x.EmployeeName, x.EmployeeNo, ""), AssignedOn: x.AssignedOn,
			ReturnedOn: str(x.ReturnedOn), ConditionOut: x.ConditionOut, ConditionIn: x.ConditionIn})
	}
	return out, nil
}

func (h *assetServer) SaveAsset(ctx context.Context, req *hrv1.SaveAssetRequest) (*hrv1.Asset, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	asset, err := h.s.Assets.Save(ctx, a, app.AssetInput{ID: req.Id, AssetCode: req.AssetCode, Name: req.Name, Category: req.Category, SerialNo: req.SerialNo,
		PurchaseDate: req.PurchaseDate, PurchaseValue: req.PurchaseValue, Notes: req.Notes, Status: req.Status})
	if err != nil {
		return nil, err
	}
	return assetPB(*asset), nil
}

func (h *assetServer) AssignAsset(ctx context.Context, req *hrv1.AssignAssetRequest) (*hrv1.Asset, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	asset, err := h.s.Assets.Assign(ctx, a, req.Id, req.EmployeeId, req.AssignedOn, req.Condition)
	if err != nil {
		return nil, err
	}
	return assetPB(*asset), nil
}

func (h *assetServer) ReturnAsset(ctx context.Context, req *hrv1.ReturnAssetRequest) (*hrv1.Asset, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	asset, err := h.s.Assets.Return(ctx, a, req.Id, req.ReturnedOn, req.Condition)
	if err != nil {
		return nil, err
	}
	return assetPB(*asset), nil
}

// ─── TrainingService ─────────────────────────────────────────

type trainingServer struct {
	hrv1.UnimplementedTrainingServiceServer
	s *Services
}

func trainingPB(t domain.Training) *hrv1.Training {
	return &hrv1.Training{Id: t.ID, Title: t.Title, Provider: t.Provider, StartDate: t.StartDate, EndDate: t.EndDate, Location: t.Location,
		Cost: money(t.Cost), Status: t.Status, Notes: t.Notes, ParticipantCount: int32(t.ParticipantCount)}
}

func (h *trainingServer) detail(ctx context.Context, a app.Actor, id string) (*hrv1.TrainingDetail, error) {
	t, people, err := h.s.Trainings.Get(ctx, a, id)
	if err != nil {
		return nil, err
	}
	out := &hrv1.TrainingDetail{Training: trainingPB(*t)}
	for _, p := range people {
		score := ""
		if p.Score != nil {
			score = money(*p.Score)
		}
		out.Participants = append(out.Participants, &hrv1.TrainingParticipant{Employee: refPB(p.EmployeeID, p.EmployeeName, p.EmployeeNo, p.DepartmentName),
			Status: p.Status, Score: score, Certificate: p.Certificate})
	}
	return out, nil
}

func (h *trainingServer) ListTrainings(ctx context.Context, req *hrv1.ListTrainingsRequest) (*hrv1.ListTrainingsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Trainings.List(ctx, a, domain.TrainingFilter{Status: req.Status, Year: int(req.Year)})
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListTrainingsResponse{}
	for _, t := range rows {
		out.Trainings = append(out.Trainings, trainingPB(t))
	}
	return out, nil
}

func (h *trainingServer) GetTraining(ctx context.Context, req *hrv1.TrainingRequest) (*hrv1.TrainingDetail, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return h.detail(ctx, a, req.Id)
}

func (h *trainingServer) SaveTraining(ctx context.Context, req *hrv1.Training) (*hrv1.Training, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	t, err := h.s.Trainings.Save(ctx, a, app.TrainingInput{ID: req.Id, Title: req.Title, Provider: req.Provider, StartDate: req.StartDate, EndDate: req.EndDate,
		Location: req.Location, Cost: req.Cost, Status: req.Status, Notes: req.Notes})
	if err != nil {
		return nil, err
	}
	return trainingPB(*t), nil
}

func (h *trainingServer) DeleteTraining(ctx context.Context, req *hrv1.TrainingRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.Trainings.Delete(ctx, a, req.Id)
}

func (h *trainingServer) SetTrainingParticipants(ctx context.Context, req *hrv1.SetTrainingParticipantsRequest) (*hrv1.TrainingDetail, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := h.s.Trainings.SetParticipants(ctx, a, req.Id, req.EmployeeIds); err != nil {
		return nil, err
	}
	return h.detail(ctx, a, req.Id)
}

func (h *trainingServer) RecordTrainingResult(ctx context.Context, req *hrv1.RecordTrainingResultRequest) (*hrv1.TrainingDetail, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	err = h.s.Trainings.RecordResult(ctx, a, req.Id, app.ParticipantResult{EmployeeID: req.EmployeeId, Status: req.Status, Score: req.Score, Certificate: req.Certificate})
	if err != nil {
		return nil, err
	}
	return h.detail(ctx, a, req.Id)
}

// ─── TimesheetService ────────────────────────────────────────

type timesheetServer struct {
	hrv1.UnimplementedTimesheetServiceServer
	s *Services
}

func timesheetPB(e domain.TimesheetEntry) *hrv1.TimesheetEntry {
	return &hrv1.TimesheetEntry{Id: e.ID, Employee: refPB(e.EmployeeID, e.EmployeeName, e.EmployeeNo, ""), WorkDate: e.WorkDate, ProjectId: e.ProjectID,
		ProjectName: e.ProjectName, Hours: money(e.Hours), Description: e.Description, Status: e.Status, RejectReason: e.RejectReason}
}

func (h *timesheetServer) ListTimesheet(ctx context.Context, req *hrv1.ListTimesheetRequest) (*hrv1.ListTimesheetResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Timesheet.List(ctx, a, domain.TimesheetFilter{EmployeeID: req.EmployeeId, Status: req.Status, From: req.From, To: req.To})
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListTimesheetResponse{}
	for _, e := range rows {
		out.Entries = append(out.Entries, timesheetPB(e))
	}
	return out, nil
}

func (h *timesheetServer) SaveTimesheetEntry(ctx context.Context, req *hrv1.SaveTimesheetEntryRequest) (*hrv1.TimesheetEntry, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	e, err := h.s.Timesheet.Save(ctx, a, app.TimesheetInput{ID: req.Id, WorkDate: req.WorkDate, ProjectID: req.ProjectId, ProjectName: req.ProjectName,
		Hours: req.Hours, Description: req.Description})
	if err != nil {
		return nil, err
	}
	return timesheetPB(*e), nil
}

func (h *timesheetServer) DeleteTimesheetEntry(ctx context.Context, req *hrv1.TimesheetEntryRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.Timesheet.Delete(ctx, a, req.Id)
}

func (h *timesheetServer) SubmitTimesheet(ctx context.Context, req *hrv1.SubmitTimesheetRequest) (*hrv1.TimesheetCount, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	n, err := h.s.Timesheet.Submit(ctx, a, req.From, req.To)
	if err != nil {
		return nil, err
	}
	return &hrv1.TimesheetCount{Count: int32(n)}, nil
}

func (h *timesheetServer) DecideTimesheet(ctx context.Context, req *hrv1.DecideTimesheetRequest) (*hrv1.TimesheetCount, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	n, err := h.s.Timesheet.Decide(ctx, a, req.Ids, req.Approve, req.Reason)
	if err != nil {
		return nil, err
	}
	return &hrv1.TimesheetCount{Count: int32(n)}, nil
}
