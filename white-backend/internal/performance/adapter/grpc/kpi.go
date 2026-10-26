package grpcapi

import (
	"context"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/performance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"google.golang.org/protobuf/types/known/emptypb"
)

type kpiServer struct {
	pb.UnimplementedKpiServiceServer
	svc Services
}

func indicatorPB(i domain.Indicator) *pb.Indicator {
	return &pb.Indicator{Id: i.ID, Code: i.Code, Name: i.Name, Description: i.Description, Unit: i.Unit, Direction: i.Direction,
		SourceKind: i.SourceKind, DefaultTarget: i.DefaultTarget, IsActive: i.IsActive, UpdatedAt: ts(i.UpdatedAt)}
}

func indicatorsPB(list []domain.Indicator) []*pb.Indicator {
	out := make([]*pb.Indicator, len(list))
	for i, ind := range list {
		out[i] = indicatorPB(ind)
	}
	return out
}

func scorecardPB(v app.ScorecardView) *pb.Scorecard {
	sc := scorecardBasePB(v.Scorecard)
	sc.EmployeeName, sc.DepartmentName = v.EmployeeName, v.DepartmentName
	return sc
}

func scorecardBasePB(c domain.Scorecard) *pb.Scorecard {
	out := &pb.Scorecard{Id: c.ID, EmployeeId: c.EmployeeID, DepartmentId: c.DepartmentID, PeriodYear: int32(c.Period.Year),
		PeriodMonth: int32(c.Period.Month), Score: c.Score, RawScore: c.RawScore, UsedWeight: c.UsedWeight, Status: c.Status,
		ReviewedBy: c.ReviewedBy, ReviewedAt: tsPtr(c.ReviewedAt), UpdatedAt: ts(c.UpdatedAt), Breakdown: []*pb.BreakdownRow{}}
	for _, b := range c.Breakdown {
		out.Breakdown = append(out.Breakdown, &pb.BreakdownRow{Code: b.Code, Weight: b.Weight, EffectiveWeight: b.EffectiveWeight,
			Attainment: b.Attainment, Contribution: b.Contribution})
	}
	return out
}

func scorecardsPB(list []domain.Scorecard) []*pb.Scorecard {
	out := make([]*pb.Scorecard, len(list))
	for i, c := range list {
		out[i] = scorecardBasePB(c)
	}
	return out
}

func (s *kpiServer) ListIndicators(ctx context.Context, req *pb.ListIndicatorsRequest) (*pb.ListIndicatorsResponse, error) {
	list, err := s.svc.KPI.ListIndicators(ctx, req.IncludeInactive)
	if err != nil {
		return nil, err
	}
	return &pb.ListIndicatorsResponse{Indicators: indicatorsPB(list)}, nil
}

func (s *kpiServer) SaveIndicator(ctx context.Context, req *pb.Indicator) (*pb.Indicator, error) {
	ind, err := s.svc.KPI.SaveIndicator(ctx, app.IndicatorInput{ID: req.Id, Code: req.Code, Name: req.Name, Description: req.Description,
		Unit: req.Unit, Direction: req.Direction, SourceKind: req.SourceKind, DefaultTarget: req.DefaultTarget, IsActive: req.IsActive})
	if err != nil {
		return nil, err
	}
	return indicatorPB(*ind), nil
}

func (s *kpiServer) GetKpiConfig(ctx context.Context, _ *emptypb.Empty) (*pb.KpiConfig, error) {
	cfg, err := s.svc.KPI.Config(ctx)
	if err != nil {
		return nil, err
	}
	out := &pb.KpiConfig{Indicators: indicatorsPB(cfg.Indicators), Departments: []*pb.Department{}, Weights: []*pb.DepartmentWeight{}}
	for _, d := range cfg.Departments {
		out.Departments = append(out.Departments, departmentPB(d))
	}
	for _, w := range cfg.Weights {
		var by string
		if w.UpdatedBy != nil {
			by = *w.UpdatedBy
		}
		out.Weights = append(out.Weights, &pb.DepartmentWeight{DepartmentId: w.DepartmentID, IndicatorId: w.IndicatorID,
			IndicatorCode: w.IndicatorCode, Weight: w.Weight, UpdatedBy: by, UpdatedAt: ts(w.UpdatedAt)})
	}
	return out, nil
}

func (s *kpiServer) SaveDepartmentConfig(ctx context.Context, req *pb.SaveDepartmentConfigRequest) (*pb.MessageResponse, error) {
	items := make([]app.WeightInput, len(req.Items))
	for i, it := range req.Items {
		items[i] = app.WeightInput{IndicatorID: it.IndicatorId, Weight: it.Weight}
	}
	text, err := s.svc.KPI.SaveDepartmentConfig(ctx, req.DepartmentId, items)
	if err != nil {
		return nil, err
	}
	return msg(text), nil
}

func targetPB(t domain.KPITarget) *pb.KpiTarget {
	return &pb.KpiTarget{Id: t.ID, IndicatorId: t.IndicatorID, PeriodYear: i32Ptr(t.PeriodYear), PeriodMonth: i32Ptr(t.PeriodMonth),
		DepartmentId: t.DepartmentID, EmployeeId: t.EmployeeID, Target: t.Target}
}

func (s *kpiServer) ListTargets(ctx context.Context, _ *emptypb.Empty) (*pb.ListTargetsResponse, error) {
	list, err := s.svc.KPI.ListTargets(ctx)
	if err != nil {
		return nil, err
	}
	out := &pb.ListTargetsResponse{Targets: make([]*pb.KpiTarget, len(list))}
	for i, t := range list {
		out.Targets[i] = targetPB(t)
	}
	return out, nil
}

func (s *kpiServer) CreateTarget(ctx context.Context, req *pb.KpiTarget) (*pb.KpiTarget, error) {
	t, err := s.svc.KPI.CreateTarget(ctx, domain.KPITarget{IndicatorID: req.IndicatorId, PeriodYear: intPtr(req.PeriodYear),
		PeriodMonth: intPtr(req.PeriodMonth), DepartmentID: req.DepartmentId, EmployeeID: req.EmployeeId, Target: req.Target})
	if err != nil {
		return nil, err
	}
	return targetPB(*t), nil
}

func (s *kpiServer) DeleteTarget(ctx context.Context, req *pb.IdRequest) (*pb.MessageResponse, error) {
	if err := s.svc.KPI.DeleteTarget(ctx, req.Id); err != nil {
		return nil, err
	}
	return msg("Target dihapus"), nil
}

func (s *kpiServer) RunSnapshot(ctx context.Context, req *pb.PeriodRequest) (*pb.SnapshotSummary, error) {
	sum, text, err := s.svc.KPI.RunSnapshot(ctx, period(req.PeriodYear, req.PeriodMonth))
	if err != nil {
		return nil, err
	}
	out := &pb.SnapshotSummary{PeriodYear: int32(sum.Period.Year), PeriodMonth: int32(sum.Period.Month), Employees: int32(sum.Employees),
		SnapshotsUpserted: int32(sum.SnapshotsUpserted), ScorecardsUpserted: int32(sum.ScorecardsUpserted),
		ScorecardsSkippedFinal: int32(sum.SkippedFinal), Collectors: map[string]int32{}, Message: text}
	for k, v := range sum.Collectors {
		out.Collectors[k] = int32(v)
	}
	return out, nil
}

func (s *kpiServer) ListScorecards(ctx context.Context, req *pb.ListScorecardsRequest) (*pb.ListScorecardsResponse, error) {
	list, err := s.svc.KPI.ListScorecards(ctx, app.ScorecardQuery{Period: period(req.PeriodYear, req.PeriodMonth), EmployeeID: req.EmployeeId,
		Team: req.Team, History: int(req.History)})
	if err != nil {
		return nil, err
	}
	inds, err := s.svc.KPI.ListIndicators(ctx, false)
	if err != nil {
		return nil, err
	}
	out := &pb.ListScorecardsResponse{Scorecards: make([]*pb.Scorecard, len(list)), Indicators: indicatorsPB(inds)}
	for i, v := range list {
		out.Scorecards[i] = scorecardPB(v)
	}
	return out, nil
}

func (s *kpiServer) GetScorecard(ctx context.Context, req *pb.IdRequest) (*pb.ScorecardDetail, error) {
	d, err := s.svc.KPI.GetScorecard(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &pb.ScorecardDetail{Scorecard: scorecardPB(d.Scorecard), CanRate: d.CanRate, Snapshots: make([]*pb.Snapshot, len(d.Snapshots))}
	for i, sn := range d.Snapshots {
		out.Snapshots[i] = &pb.Snapshot{IndicatorId: sn.IndicatorID, IndicatorCode: sn.IndicatorCode, IndicatorName: sn.IndicatorName,
			SourceKind: sn.SourceKind, Actual: sn.Actual, Target: sn.Target, Attainment: sn.Attainment, SampleSize: sn.SampleSize,
			SourceDetail: string(sn.SourceDetail)}
	}
	return out, nil
}

func (s *kpiServer) SetScorecardStatus(ctx context.Context, req *pb.SetScorecardStatusRequest) (*pb.ScorecardResponse, error) {
	v, text, err := s.svc.KPI.SetScorecardStatus(ctx, req.Id, req.Action)
	if err != nil {
		return nil, err
	}
	return &pb.ScorecardResponse{Scorecard: scorecardPB(*v), Message: text}, nil
}

func (s *kpiServer) saveManual(ctx context.Context, req *pb.SaveManualRequest, indicator, done string) (*pb.SaveManualResponse, error) {
	score, status, err := s.svc.KPI.SaveManual(ctx, app.ManualInput{EmployeeID: req.EmployeeId, IndicatorID: indicator,
		Period: period(req.PeriodYear, req.PeriodMonth), Value: req.Value, Notes: req.Notes})
	if err != nil {
		return nil, err
	}
	return &pb.SaveManualResponse{Score: score, Status: status, Message: done}, nil
}

func (s *kpiServer) SaveRubric(ctx context.Context, req *pb.SaveManualRequest) (*pb.SaveManualResponse, error) {
	return s.saveManual(ctx, req, "", "Rubrik tersimpan & scorecard diperbarui")
}

func (s *kpiServer) RecordManualActual(ctx context.Context, req *pb.SaveManualRequest) (*pb.SaveManualResponse, error) {
	if req.IndicatorId == "" {
		return nil, apperr.Field("indicatorId", "Indikator wajib diisi")
	}
	return s.saveManual(ctx, req, req.IndicatorId, "Nilai aktual tersimpan & scorecard diperbarui")
}

func (s *kpiServer) GetRecommendations(ctx context.Context, req *pb.GetRecommendationsRequest) (*pb.GetRecommendationsResponse, error) {
	list, err := s.svc.KPI.Recommendations(ctx, req.EmployeeIds)
	if err != nil {
		return nil, err
	}
	out := &pb.GetRecommendationsResponse{Items: make([]*pb.GetRecommendationsResponse_Item, len(list))}
	for i, r := range list {
		out.Items[i] = &pb.GetRecommendationsResponse_Item{EmployeeId: r.EmployeeID, AvgScore: r.AvgScore, Periods: int32(r.Periods), HasFinal: r.HasFinal}
	}
	return out, nil
}

// ─── ReviewService ───────────────────────────────────────────

type reviewServer struct {
	pb.UnimplementedReviewServiceServer
	svc Services
}

func cyclePB(c domain.ReviewCycle) *pb.ReviewCycle {
	return &pb.ReviewCycle{Id: c.ID, Name: c.Name, PeriodYear: int32(c.Year), PeriodQuarter: int32(c.Quarter), StartDate: c.StartDate,
		EndDate: c.EndDate, DueDate: c.DueDate, Status: c.Status, CreatedByName: c.CreatedByName, TotalReviews: int32(c.TotalReviews),
		FinalReviews: int32(c.FinalReviews), RatedReviews: int32(c.RatedReviews)}
}

func reviewPB(v app.ReviewView) *pb.Review {
	r := v.Review
	return &pb.Review{Id: r.ID, CycleId: r.CycleID, CycleName: r.CycleName, CycleStatus: r.CycleStatus, CycleDueDate: r.CycleDueDate,
		EmployeeId: r.EmployeeID, EmployeeName: v.EmployeeName, DepartmentId: r.DepartmentID, DepartmentName: v.DepartmentName,
		PeriodLabel: r.PeriodLabel, StartDate: r.StartDate, EndDate: r.EndDate, WorkResultScore: r.Work, BehavioralScore: r.Behavior,
		ProjectScore: r.Project, GrandTotalScore: r.Grand, Category: r.Category, Status: r.Status, SelfAssessment: r.SelfAssessment,
		ReviewerId: r.ReviewerID, ReviewerName: r.ReviewerName, ReviewerNotes: r.ReviewerNotes, ManagerId: r.ManagerID,
		ManagerNotes: r.ManagerNotes, EmployeeSignDate: r.EmployeeSignDate, ReviewerSignDate: r.ReviewerSignDate,
		RatedItems: int32(r.RatedItems), TotalItems: int32(r.TotalItems), FinalizedAt: tsPtr(r.FinalizedAt)}
}

func (s *reviewServer) ListReviewCycles(ctx context.Context, _ *emptypb.Empty) (*pb.ListReviewCyclesResponse, error) {
	list, canManage, err := s.svc.Reviews.ListCycles(ctx)
	if err != nil {
		return nil, err
	}
	out := &pb.ListReviewCyclesResponse{Cycles: make([]*pb.ReviewCycle, len(list)), CanManage: canManage}
	for i, c := range list {
		out.Cycles[i] = cyclePB(c)
	}
	return out, nil
}

func (s *reviewServer) OpenReviewCycle(ctx context.Context, req *pb.OpenReviewCycleRequest) (*pb.ReviewCycleResponse, error) {
	c, text, err := s.svc.Reviews.OpenCycle(ctx, int(req.PeriodYear), int(req.PeriodQuarter), req.DueDate)
	if err != nil {
		return nil, err
	}
	return &pb.ReviewCycleResponse{Cycle: cyclePB(*c), Message: text}, nil
}

func (s *reviewServer) CloseReviewCycle(ctx context.Context, req *pb.IdRequest) (*pb.MessageResponse, error) {
	if err := s.svc.Reviews.CloseCycle(ctx, req.Id); err != nil {
		return nil, err
	}
	return msg("Siklus review ditutup"), nil
}

func (s *reviewServer) ListReviews(ctx context.Context, req *pb.ListReviewsRequest) (*pb.ListReviewsResponse, error) {
	list, canReview, err := s.svc.Reviews.ListReviews(ctx, req.CycleId)
	if err != nil {
		return nil, err
	}
	out := &pb.ListReviewsResponse{Reviews: make([]*pb.Review, len(list)), CanReview: canReview}
	for i, v := range list {
		out.Reviews[i] = reviewPB(v)
	}
	return out, nil
}

func (s *reviewServer) GetReview(ctx context.Context, req *pb.IdRequest) (*pb.ReviewDetail, error) {
	d, err := s.svc.Reviews.GetReview(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	out := &pb.ReviewDetail{Review: reviewPB(d.Review), KpiMonths: scorecardsPB(d.KPIMonths), CanRate: d.CanRate,
		CanFinalize: d.CanFinalize, IsOwner: d.IsOwner, Items: make([]*pb.ReviewItem, len(d.Items))}
	for i, it := range d.Items {
		out.Items[i] = &pb.ReviewItem{Id: it.ID, ValueName: it.ValueName, Competency: it.Competency, BehavioralStandard: it.BehavioralStandard,
			ScoreDescriptions: it.ScoreDescriptions[:], Weight: it.Weight, Score: i32Ptr(it.Score), WeightedScore: it.WeightedScore,
			Notes: it.Notes, ItemOrder: int32(it.ItemOrder)}
	}
	return out, nil
}

func (s *reviewServer) ReviewAction(ctx context.Context, req *pb.ReviewActionRequest) (*pb.MessageResponse, error) {
	text, err := s.svc.Reviews.Act(ctx, req.Id, app.ReviewAction{Action: req.Action, Text: req.Text, ItemID: req.ItemId, Score: req.Score,
		Notes: req.Notes, ProjectScore: req.ProjectScore})
	if err != nil {
		return nil, err
	}
	return msg(text), nil
}

func (s *reviewServer) RealtimeKpi(ctx context.Context, req *pb.RealtimeKpiRequest) (*pb.RealtimeKpiResponse, error) {
	year, quarter := int(req.Year), int(req.Quarter)
	if year == 0 || quarter == 0 {
		y, m := s.svc.KPI.CurrentPeriod()
		if year == 0 {
			year = y
		}
		if quarter == 0 {
			quarter = domain.QuarterOf(m)
		}
	}
	rows, err := s.svc.Reviews.Realtime(ctx, year, quarter)
	if err != nil {
		return nil, err
	}
	out := &pb.RealtimeKpiResponse{Year: int32(year), Quarter: int32(quarter), Employees: make([]*pb.RealtimeKpiResponse_Row, len(rows))}
	for _, m := range domain.QuarterMonths(quarter) {
		out.Months = append(out.Months, int32(m))
	}
	for i, r := range rows {
		out.Employees[i] = &pb.RealtimeKpiResponse_Row{EmployeeId: r.EmployeeID, EmployeeName: r.EmployeeName, DepartmentName: r.DepartmentName,
			AvgScore: r.AvgScore, Months: scorecardsPB(r.Months)}
	}
	return out, nil
}
