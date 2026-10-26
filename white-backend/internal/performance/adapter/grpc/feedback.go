package grpcapi

import (
	"context"
	"fmt"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/performance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

type feedbackServer struct {
	pb.UnimplementedFeedbackServiceServer
	svc Services
}

func categoriesPB(cats []domain.FeedbackCategory) []*pb.FeedbackCategory {
	out := make([]*pb.FeedbackCategory, len(cats))
	for i, c := range cats {
		cat := &pb.FeedbackCategory{Id: c.ID, Code: c.Code, Name: c.Name, Description: c.Description, Weight: c.Weight,
			DisplayOrder: int32(c.DisplayOrder), Criteria: make([]*pb.FeedbackCriterion, len(c.Criteria))}
		for j, cr := range c.Criteria {
			cat.Criteria[j] = &pb.FeedbackCriterion{Id: cr.ID, Name: cr.Name, Description: cr.Description, DisplayOrder: int32(cr.DisplayOrder)}
		}
		out[i] = cat
	}
	return out
}

func feedbackCyclePB(c domain.FeedbackCycle) *pb.FeedbackCycle {
	return &pb.FeedbackCycle{Id: c.ID, Name: c.Name, PeriodLabel: c.PeriodLabel, StartDate: c.StartDate, EndDate: c.EndDate, Status: c.Status,
		KpiWeight: c.KPIWeight, FeedbackWeight: c.FeedbackWeight, IsAnonymous: c.IsAnonymous, AllowSelfAssessment: c.AllowSelfAssessment,
		RequireManagerReview: c.RequireManagerReview, AssignmentsCount: int32(c.AssignmentsCount), SubmittedCount: int32(c.SubmittedCount),
		SummariesCount: int32(c.SummariesCount), CreatedAt: ts(c.CreatedAt)}
}

func assignmentPB(v app.AssignmentView) *pb.Assignment {
	a := v.FeedbackAssignment
	return &pb.Assignment{Id: a.ID, CycleId: a.CycleID, CycleName: a.CycleName, CycleStatus: a.CycleStatus, EmployeeId: a.EmployeeID,
		EmployeeName: v.EmployeeName, ReviewerId: a.ReviewerID, ReviewerName: v.ReviewerName, RelationshipType: a.Relationship,
		Status: a.Status, DueDate: a.DueDate, SubmittedAt: tsPtr(a.SubmittedAt), ReminderCount: int32(a.ReminderCount),
		LastReminderAt: tsPtr(a.LastReminderAt), ManagerComments: a.ManagerComments, RejectionReason: a.RejectionReason}
}

func assignmentsPB(list []app.AssignmentView) *pb.ListAssignmentsResponse {
	out := &pb.ListAssignmentsResponse{Assignments: make([]*pb.Assignment, len(list))}
	for i, v := range list {
		out.Assignments[i] = assignmentPB(v)
	}
	return out
}

func summaryPB(v app.SummaryView) *pb.FeedbackSummary {
	s := v.FeedbackSummary
	out := &pb.FeedbackSummary{Id: s.ID, CycleId: s.CycleID, EmployeeId: s.EmployeeID, EmployeeName: v.EmployeeName, DepartmentName: v.DepartmentName,
		Scores: map[string]float64{}, Overall_360Score: s.Overall360, KpiScore: s.KPIScore, FinalScore: s.FinalScore, FinalGrade: s.FinalGrade,
		Strengths: s.Strengths, Weaknesses: s.Weaknesses, ManagerComments: s.ManagerComments, ResponsesCount: int32(s.ResponsesCount),
		IsLocked: s.IsLocked, GeneratedAt: ts(s.GeneratedAt)}
	for code, score := range s.Scores {
		if score != nil {
			out.Scores[code] = *score
		}
	}
	return out
}

func (s *feedbackServer) ListFeedbackCategories(ctx context.Context, _ *emptypb.Empty) (*pb.ListFeedbackCategoriesResponse, error) {
	cats, err := s.svc.Feedback.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.ListFeedbackCategoriesResponse{Categories: categoriesPB(cats)}, nil
}

func (s *feedbackServer) ListFeedbackCycles(ctx context.Context, req *pb.ListFeedbackCyclesRequest) (*pb.ListFeedbackCyclesResponse, error) {
	list, err := s.svc.Feedback.ListCycles(ctx, req.Status)
	if err != nil {
		return nil, err
	}
	out := &pb.ListFeedbackCyclesResponse{Cycles: make([]*pb.FeedbackCycle, len(list))}
	for i, c := range list {
		out.Cycles[i] = feedbackCyclePB(c)
	}
	return out, nil
}

func (s *feedbackServer) GetFeedbackCycle(ctx context.Context, req *pb.IdRequest) (*pb.FeedbackCycle, error) {
	c, err := s.svc.Feedback.GetCycle(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return feedbackCyclePB(*c), nil
}

func (s *feedbackServer) SaveFeedbackCycle(ctx context.Context, req *pb.FeedbackCycle) (*pb.FeedbackCycle, error) {
	c, err := s.svc.Feedback.SaveCycle(ctx, domain.FeedbackCycle{ID: req.Id, Name: req.Name, PeriodLabel: req.PeriodLabel, StartDate: req.StartDate,
		EndDate: req.EndDate, Status: req.Status, KPIWeight: req.KpiWeight, FeedbackWeight: req.FeedbackWeight, IsAnonymous: req.IsAnonymous,
		AllowSelfAssessment: req.AllowSelfAssessment, RequireManagerReview: req.RequireManagerReview})
	if err != nil {
		return nil, err
	}
	return feedbackCyclePB(*c), nil
}

func (s *feedbackServer) DeleteFeedbackCycle(ctx context.Context, req *pb.IdRequest) (*pb.MessageResponse, error) {
	if err := s.svc.Feedback.DeleteCycle(ctx, req.Id); err != nil {
		return nil, err
	}
	return msg("Siklus feedback dihapus"), nil
}

func (s *feedbackServer) CreateAssignments(ctx context.Context, req *pb.CreateAssignmentsRequest) (*pb.CountResponse, error) {
	items := make([]app.AssignmentInput, len(req.Items))
	for i, it := range req.Items {
		items[i] = app.AssignmentInput{EmployeeID: it.EmployeeId, ReviewerID: it.ReviewerId, Relationship: it.RelationshipType, DueDate: it.DueDate}
	}
	n, err := s.svc.Feedback.CreateAssignments(ctx, req.CycleId, items)
	if err != nil {
		return nil, err
	}
	return &pb.CountResponse{Count: int32(n), Message: fmt.Sprintf("%d penugasan feedback dibuat", n)}, nil
}

func (s *feedbackServer) AutoAssign(ctx context.Context, req *pb.AutoAssignRequest) (*pb.CountResponse, error) {
	n, err := s.svc.Feedback.AutoAssign(ctx, req.CycleId, int(req.Peers))
	if err != nil {
		return nil, err
	}
	return &pb.CountResponse{Count: int32(n), Message: fmt.Sprintf("%d penugasan feedback dibuat", n)}, nil
}

func (s *feedbackServer) ListAssignments(ctx context.Context, req *pb.ListAssignmentsRequest) (*pb.ListAssignmentsResponse, error) {
	list, err := s.svc.Feedback.ListAssignments(ctx, req.CycleId, req.EmployeeId, req.ReviewerId, req.Status)
	if err != nil {
		return nil, err
	}
	return assignmentsPB(list), nil
}

func (s *feedbackServer) ListMyAssignments(ctx context.Context, _ *emptypb.Empty) (*pb.ListAssignmentsResponse, error) {
	list, err := s.svc.Feedback.MyAssignments(ctx)
	if err != nil {
		return nil, err
	}
	return assignmentsPB(list), nil
}

func (s *feedbackServer) GetAssignment(ctx context.Context, req *pb.IdRequest) (*pb.AssignmentDetail, error) {
	d, err := s.svc.Feedback.GetAssignment(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &pb.AssignmentDetail{Assignment: assignmentPB(d.Assignment), Categories: categoriesPB(d.Categories), CanAnswer: d.CanAnswer,
		CanDecide: d.CanDecide, Responses: make([]*pb.FeedbackResponse, len(d.Responses))}
	for i, r := range d.Responses {
		out.Responses[i] = &pb.FeedbackResponse{Id: r.ID, CriteriaId: r.CriteriaID, Rating: int32(r.Rating), Comments: r.Comments}
	}
	return out, nil
}

func (s *feedbackServer) SubmitResponses(ctx context.Context, req *pb.SubmitResponsesRequest) (*pb.Assignment, error) {
	answers := make([]app.ResponseInput, len(req.Answers))
	for i, a := range req.Answers {
		answers[i] = app.ResponseInput{CriteriaID: a.CriteriaId, Rating: int(a.Rating), Comments: a.Comments}
	}
	asg, err := s.svc.Feedback.SubmitResponses(ctx, req.Id, answers, req.Submit)
	if err != nil {
		return nil, err
	}
	return assignmentPB(app.AssignmentView{FeedbackAssignment: *asg}), nil
}

func (s *feedbackServer) DecideAssignment(ctx context.Context, req *pb.DecideAssignmentRequest) (*pb.DecideAssignmentResponse, error) {
	asg, text, err := s.svc.Feedback.Decide(ctx, req.Id, req.Approve, req.ManagerComments, req.RejectionReason)
	if err != nil {
		return nil, err
	}
	return &pb.DecideAssignmentResponse{Assignment: assignmentPB(app.AssignmentView{FeedbackAssignment: *asg}), Message: text}, nil
}

func (s *feedbackServer) DeleteAssignment(ctx context.Context, req *pb.IdRequest) (*pb.MessageResponse, error) {
	if err := s.svc.Feedback.DeleteAssignment(ctx, req.Id); err != nil {
		return nil, err
	}
	return msg("Penugasan feedback dihapus"), nil
}

func (s *feedbackServer) GenerateSummaries(ctx context.Context, req *pb.GenerateSummariesRequest) (*pb.GenerateSummariesResponse, error) {
	n, filed, err := s.svc.Feedback.GenerateSummaries(ctx, req.CycleId)
	if err != nil {
		return nil, err
	}
	return &pb.GenerateSummariesResponse{Generated: int32(n), ImprovementsFiled: int32(filed),
		Message: fmt.Sprintf("%d ringkasan feedback dibuat, %d item perbaikan ditambahkan", n, filed)}, nil
}

func (s *feedbackServer) ListSummaries(ctx context.Context, req *pb.ListSummariesRequest) (*pb.ListSummariesResponse, error) {
	list, err := s.svc.Feedback.ListSummaries(ctx, req.CycleId, req.EmployeeId)
	if err != nil {
		return nil, err
	}
	out := &pb.ListSummariesResponse{Summaries: make([]*pb.FeedbackSummary, len(list))}
	for i, v := range list {
		out.Summaries[i] = summaryPB(v)
	}
	return out, nil
}

func (s *feedbackServer) UpdateSummary(ctx context.Context, req *pb.UpdateSummaryRequest) (*pb.FeedbackSummary, error) {
	sum, err := s.svc.Feedback.UpdateSummary(ctx, req.Id, req.Strengths, req.Weaknesses, req.ManagerComments, req.IsLocked)
	if err != nil {
		return nil, err
	}
	return summaryPB(app.SummaryView{FeedbackSummary: *sum}), nil
}
