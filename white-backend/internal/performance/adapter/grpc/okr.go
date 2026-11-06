package grpcapi

import (
	"context"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/performance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type okrServer struct {
	pb.UnimplementedOkrServiceServer
	svc Services
}

func keyResultPB(k domain.KeyResult, confidence int) *pb.KeyResult {
	return &pb.KeyResult{Id: k.ID, ObjectiveId: k.ObjectiveID, Title: k.Title, Metric: k.Metric, StartValue: k.Start, TargetValue: k.Target,
		CurrentValue: k.Current, Unit: k.Unit, SortOrder: int32(k.SortOrder), Progress: k.Progress(), Confidence: int32(confidence)}
}

func objectivePB(o domain.Objective) *pb.Objective {
	return &pb.Objective{Id: o.ID, Period: o.Period, Level: o.Level, DepartmentId: str(o.DepartmentID), EmployeeId: str(o.EmployeeID),
		ParentId: str(o.ParentID), Title: o.Title, Description: o.Description, Status: o.Status}
}

func (s *okrServer) GetOkrBoard(ctx context.Context, req *pb.GetOkrBoardRequest) (*pb.OkrBoard, error) {
	views, err := s.svc.OKR.Board(ctx, req.Period)
	if err != nil {
		return nil, err
	}
	out := &pb.OkrBoard{CanCompany: views.CanCompany}
	for _, d := range views.Departments {
		out.Departments = append(out.Departments, &pb.OkrOwner{Id: d.ID, Name: d.Name})
	}
	for _, e := range views.Employees {
		out.Employees = append(out.Employees, &pb.OkrOwner{Id: e.ID, Name: e.Name})
	}
	for _, v := range views.Objectives {
		o := objectivePB(v.Objective)
		o.OwnerName, o.Progress, o.CanEdit = v.OwnerName, v.Progress, v.CanEdit
		for _, k := range v.KeyResults {
			o.KeyResults = append(o.KeyResults, keyResultPB(k.KeyResult, k.Confidence))
		}
		out.Objectives = append(out.Objectives, o)
	}
	return out, nil
}

func (s *okrServer) SaveObjective(ctx context.Context, req *pb.SaveObjectiveRequest) (*pb.Objective, error) {
	o, err := s.svc.OKR.SaveObjective(ctx, app.ObjectiveInput{ID: req.Id, Period: req.Period, Level: req.Level, DepartmentID: req.DepartmentId,
		EmployeeID: req.EmployeeId, ParentID: req.ParentId, Title: req.Title, Description: req.Description, Status: req.Status})
	if err != nil {
		return nil, err
	}
	return objectivePB(*o), nil
}

func (s *okrServer) DeleteObjective(ctx context.Context, req *pb.IdRequest) (*pb.MessageResponse, error) {
	if err := s.svc.OKR.DeleteObjective(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.MessageResponse{Message: "Objective dihapus"}, nil
}

func (s *okrServer) SaveKeyResult(ctx context.Context, req *pb.SaveKeyResultRequest) (*pb.KeyResult, error) {
	k, err := s.svc.OKR.SaveKeyResult(ctx, app.KeyResultInput{ID: req.Id, ObjectiveID: req.ObjectiveId, Title: req.Title, Metric: req.Metric, Unit: req.Unit,
		Start: req.StartValue, Target: req.TargetValue, SortOrder: int(req.SortOrder)})
	if err != nil {
		return nil, err
	}
	return keyResultPB(*k, 0), nil
}

func (s *okrServer) DeleteKeyResult(ctx context.Context, req *pb.IdRequest) (*pb.MessageResponse, error) {
	if err := s.svc.OKR.DeleteKeyResult(ctx, req.Id); err != nil {
		return nil, err
	}
	return &pb.MessageResponse{Message: "Key result dihapus"}, nil
}

func (s *okrServer) CheckInKeyResult(ctx context.Context, req *pb.CheckInKeyResultRequest) (*pb.KeyResult, error) {
	k, err := s.svc.OKR.CheckIn(ctx, req.Id, req.Value, int(req.Confidence), req.Note)
	if err != nil {
		return nil, err
	}
	return keyResultPB(*k, int(req.Confidence)), nil
}

func (s *okrServer) ListCheckins(ctx context.Context, req *pb.IdRequest) (*pb.ListCheckinsResponse, error) {
	rows, err := s.svc.OKR.Checkins(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &pb.ListCheckinsResponse{}
	for _, c := range rows {
		out.Checkins = append(out.Checkins, &pb.OkrCheckin{Id: c.ID, Value: c.Value, Confidence: int32(c.Confidence), Note: c.Note, CreatedAt: timestamppb.New(c.CreatedAt)})
	}
	return out, nil
}

func str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
