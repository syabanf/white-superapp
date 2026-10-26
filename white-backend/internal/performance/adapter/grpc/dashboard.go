package grpcapi

import (
	"context"

	pb "github.com/syabanf/white-superapp/white-backend/gen/go/white/performance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/app"
)

func dashQuery(r *pb.DashboardRequest) app.DashboardQuery {
	return app.DashboardQuery{From: r.From, To: r.To, Compare: r.Compare, DepartmentID: r.DepartmentId, EmployeeID: r.EmployeeId, Scope: r.Scope}
}

func windowPB(w app.DashWindow) *pb.DashboardWindow {
	return &pb.DashboardWindow{From: w.From, To: w.To, CompareFrom: w.CompareFrom, CompareTo: w.CompareTo, Compare: w.Compare,
		Granularity: w.Granularity, Scope: w.Scope, Today: w.Today}
}

func metricPB(m app.DashMetric) *pb.DashboardMetric {
	return &pb.DashboardMetric{Value: m.Value, Previous: m.Previous, Spark: m.Spark}
}

func countsPB(list []app.DashCount) []*pb.DashboardCount {
	out := make([]*pb.DashboardCount, len(list))
	for i, c := range list {
		out[i] = &pb.DashboardCount{Key: c.Key, Value: c.Value, Previous: c.Previous}
	}
	return out
}

func deptMonthsPB(list []app.DeptMonthScore) []*pb.DepartmentMonthScore {
	out := make([]*pb.DepartmentMonthScore, len(list))
	for i, m := range list {
		out[i] = &pb.DepartmentMonthScore{DepartmentId: m.DepartmentID, DepartmentName: m.DepartmentName, Period: m.Period, Score: m.Score,
			Scorecards: int32(m.Scorecards)}
	}
	return out
}

func (s *kpiServer) GetDashboardContext(ctx context.Context, req *pb.DashboardRequest) (*pb.DashboardContext, error) {
	c, err := s.svc.Dashboard.Context(ctx, dashQuery(req))
	if err != nil {
		return nil, err
	}
	out := &pb.DashboardContext{Window: windowPB(c.Window), Scopes: c.Scopes, Departments: []*pb.Department{}, People: []*pb.Person{}}
	for _, d := range c.Departments {
		out.Departments = append(out.Departments, departmentPB(d))
	}
	for _, e := range c.People {
		out.People = append(out.People, personPB(e))
	}
	return out, nil
}

func (s *kpiServer) GetDashboardSummary(ctx context.Context, req *pb.DashboardRequest) (*pb.DashboardSummary, error) {
	r, err := s.svc.Dashboard.Summary(ctx, dashQuery(req))
	if err != nil {
		return nil, err
	}
	out := &pb.DashboardSummary{Window: windowPB(r.Window), KpiScore: metricPB(r.KPIScore), TaskCompletion: metricPB(r.TaskCompletion),
		OnTimeRate: metricPB(r.OnTimeRate), OverdueTasks: metricPB(r.OverdueTasks), LogbookRate: metricPB(r.LogbookRate),
		OpenImprovements: metricPB(r.OpenImprovements), Buckets: r.Buckets, Months: r.Months, DepartmentTrend: deptMonthsPB(r.DepartmentTrend),
		Departments: make([]*pb.DashboardDepartmentRow, len(r.Departments))}
	for i, d := range r.Departments {
		out.Departments[i] = &pb.DashboardDepartmentRow{DepartmentId: d.DepartmentID, DepartmentName: d.DepartmentName, Members: int32(d.Members),
			KpiScore: d.KPIScore, KpiScorePrevious: d.KPIScorePrevious, TaskCompletion: d.TaskCompletion, OnTimeRate: d.OnTimeRate,
			OverdueTasks: int32(d.OverdueTasks), LogbookRate: d.LogbookRate, OpenImprovements: int32(d.OpenImprovements)}
	}
	return out, nil
}

func (s *kpiServer) GetDashboardKpi(ctx context.Context, req *pb.DashboardRequest) (*pb.DashboardKpi, error) {
	r, err := s.svc.Dashboard.KPI(ctx, dashQuery(req))
	if err != nil {
		return nil, err
	}
	out := &pb.DashboardKpi{Window: windowPB(r.Window), AvgScore: metricPB(r.AvgScore), FinalRate: metricPB(r.FinalRate),
		Scorecards: metricPB(r.Scorecards), BelowTarget: metricPB(r.BelowTarget), Histogram: countsPB(r.Histogram), Months: r.Months,
		DepartmentMonths: deptMonthsPB(r.DepartmentMonths), Funnel: countsPB(r.Funnel),
		Indicators: make([]*pb.IndicatorAttainment, len(r.Indicators)), People: make([]*pb.PersonScore, len(r.People))}
	for i, ind := range r.Indicators {
		out.Indicators[i] = &pb.IndicatorAttainment{IndicatorId: ind.IndicatorID, Code: ind.Code, Name: ind.Name, Unit: ind.Unit,
			Direction: ind.Direction, Actual: ind.Actual, Target: ind.Target, Attainment: ind.Attainment,
			PreviousAttainment: ind.PreviousAttainment, Samples: int32(ind.Samples)}
	}
	for i, p := range r.People {
		out.People[i] = &pb.PersonScore{EmployeeId: p.EmployeeID, Name: p.Name, DepartmentId: p.DepartmentID, DepartmentName: p.DepartmentName,
			Score: p.Score, Previous: p.Previous, Months: int32(p.Months), FinalMonths: int32(p.FinalMonths)}
	}
	return out, nil
}

func (s *kpiServer) GetDashboardTasks(ctx context.Context, req *pb.DashboardRequest) (*pb.DashboardTasks, error) {
	r, err := s.svc.Dashboard.Tasks(ctx, dashQuery(req))
	if err != nil {
		return nil, err
	}
	out := &pb.DashboardTasks{Window: windowPB(r.Window), Due: metricPB(r.Due), Completion: metricPB(r.Completion),
		OnTimeRate: metricPB(r.OnTimeRate), Overdue: metricPB(r.Overdue), WaitingReview: metricPB(r.WaitingReview),
		TurnaroundHours: metricPB(r.TurnaroundHours), Aging: countsPB(r.Aging), Turnaround: countsPB(r.Turnaround),
		Trend: make([]*pb.TaskBucket, len(r.Trend)), Departments: make([]*pb.TaskDepartmentStatus, len(r.Departments)),
		People: make([]*pb.TaskPerson, len(r.People))}
	for i, b := range r.Trend {
		out.Trend[i] = &pb.TaskBucket{Bucket: b.Bucket, Due: int32(b.Due), Approved: int32(b.Approved), Finished: int32(b.Finished),
			OnTime: int32(b.OnTime), Overdue: int32(b.Overdue), Completion: b.Completion, OnTimeRate: b.OnTimeRate,
			PreviousCompletion: b.PreviousCompletion, PreviousOnTimeRate: b.PreviousOnTimeRate, TurnaroundHours: b.TurnaroundHours}
	}
	for i, d := range r.Departments {
		out.Departments[i] = &pb.TaskDepartmentStatus{DepartmentId: d.DepartmentID, DepartmentName: d.DepartmentName, Due: int32(d.Due),
			Pending: int32(d.Pending), InProgress: int32(d.InProgress), WaitingReview: int32(d.WaitingReview), Approved: int32(d.Approved),
			Rejected: int32(d.Rejected), Overdue: int32(d.Overdue), OnTimeRate: d.OnTimeRate}
	}
	for i, p := range r.People {
		out.People[i] = &pb.TaskPerson{EmployeeId: p.EmployeeID, Name: p.Name, DepartmentName: p.DepartmentName, Due: int32(p.Due),
			Approved: int32(p.Approved), OnTime: int32(p.OnTime), Overdue: int32(p.Overdue), WaitingReview: int32(p.WaitingReview),
			Rejected: int32(p.Rejected), Completion: p.Completion}
	}
	return out, nil
}

func (s *kpiServer) GetDashboardLogbook(ctx context.Context, req *pb.DashboardRequest) (*pb.DashboardLogbook, error) {
	r, err := s.svc.Dashboard.Logbook(ctx, dashQuery(req))
	if err != nil {
		return nil, err
	}
	out := &pb.DashboardLogbook{Window: windowPB(r.Window), SubmissionRate: metricPB(r.SubmissionRate), AvgScore: metricPB(r.AvgScore),
		RejectionRate: metricPB(r.RejectionRate), RecapCoverage: metricPB(r.RecapCoverage), Status: countsPB(r.Status),
		Trend: make([]*pb.LogbookBucket, len(r.Trend)), Departments: make([]*pb.LogbookDepartmentBucket, len(r.Departments)),
		Heatmap: make([]*pb.LogbookHeatCell, len(r.Heatmap)), People: make([]*pb.LogbookPerson, len(r.People))}
	for i, b := range r.Trend {
		out.Trend[i] = &pb.LogbookBucket{Bucket: b.Bucket, Expected: int32(b.Expected), Submitted: int32(b.Submitted), Reviewed: int32(b.Reviewed),
			Rejected: int32(b.Rejected), AvgScore: b.AvgScore, Recaps: int32(b.Recaps), PreviousRate: b.PreviousRate}
	}
	for i, d := range r.Departments {
		out.Departments[i] = &pb.LogbookDepartmentBucket{Bucket: d.Bucket, DepartmentId: d.DepartmentID, DepartmentName: d.DepartmentName,
			Expected: int32(d.Expected), Submitted: int32(d.Submitted), AvgScore: d.AvgScore}
	}
	for i, c := range r.Heatmap {
		out.Heatmap[i] = &pb.LogbookHeatCell{Week: c.Week, Weekday: int32(c.Weekday), Submitted: int32(c.Submitted), Expected: int32(c.Expected)}
	}
	for i, p := range r.People {
		out.People[i] = &pb.LogbookPerson{EmployeeId: p.EmployeeID, Name: p.Name, DepartmentName: p.DepartmentName, Expected: int32(p.Expected),
			Submitted: int32(p.Submitted), Reviewed: int32(p.Reviewed), Rejected: int32(p.Rejected), AvgScore: p.AvgScore, Recaps: int32(p.Recaps)}
	}
	return out, nil
}

func cycleProgressPB(c app.DashCycleProgress) *pb.ReviewCycleProgress {
	return &pb.ReviewCycleProgress{CycleId: c.CycleID, Name: c.Name, Status: c.Status, DueDate: c.DueDate, Total: int32(c.Total),
		SelfAssessed: int32(c.SelfAssessed), Rated: int32(c.Rated), Signed: int32(c.Signed), Finalized: int32(c.Finalized), AvgScore: c.AvgScore}
}

func (s *kpiServer) GetDashboardReviews(ctx context.Context, req *pb.DashboardRequest) (*pb.DashboardReviews, error) {
	r, err := s.svc.Dashboard.Reviews(ctx, dashQuery(req))
	if err != nil {
		return nil, err
	}
	out := &pb.DashboardReviews{Window: windowPB(r.Window), Cycle: cycleProgressPB(r.Cycle), Ratings: countsPB(r.Ratings), ReviewScore: metricPB(r.ReviewScore),
		FeedbackCycleId: r.FeedbackCycleID, FeedbackCycleName: r.FeedbackCycleName, PreviousFeedbackCycleName: r.PreviousFeedbackCycleName,
		FeedbackScore: metricPB(r.FeedbackScore), FeedbackCompletion: metricPB(r.FeedbackCompletion),
		Cycles: make([]*pb.ReviewCycleProgress, len(r.Cycles)), FeedbackProgress: make([]*pb.FeedbackRelationshipProgress, len(r.FeedbackProgress)),
		FeedbackCategories: make([]*pb.FeedbackCategoryScore, len(r.FeedbackCategories)), Departments: make([]*pb.ReviewDepartmentRow, len(r.Departments))}
	for i, c := range r.Cycles {
		out.Cycles[i] = cycleProgressPB(c)
	}
	for i, p := range r.FeedbackProgress {
		out.FeedbackProgress[i] = &pb.FeedbackRelationshipProgress{Relationship: p.Relationship, Total: int32(p.Total), Submitted: int32(p.Submitted)}
	}
	for i, c := range r.FeedbackCategories {
		out.FeedbackCategories[i] = &pb.FeedbackCategoryScore{Code: c.Code, Name: c.Name, Score: c.Score, Previous: c.Previous}
	}
	for i, d := range r.Departments {
		out.Departments[i] = &pb.ReviewDepartmentRow{DepartmentId: d.DepartmentID, DepartmentName: d.DepartmentName, Reviews: int32(d.Reviews),
			Finalized: int32(d.Finalized), AvgScore: d.AvgScore, Avg_360: d.Avg360}
	}
	return out, nil
}

func (s *kpiServer) GetDashboardImprovements(ctx context.Context, req *pb.DashboardRequest) (*pb.DashboardImprovements, error) {
	r, err := s.svc.Dashboard.Improvements(ctx, dashQuery(req))
	if err != nil {
		return nil, err
	}
	out := &pb.DashboardImprovements{Window: windowPB(r.Window), Open: metricPB(r.Open), Created: metricPB(r.Created),
		Completed: metricPB(r.Completed), Overdue: metricPB(r.Overdue), AiAcceptance: metricPB(r.AIAccept), Status: countsPB(r.Status),
		Sources: countsPB(r.Sources), Trend: make([]*pb.ImprovementBucket, len(r.Trend)),
		OverdueItems: make([]*pb.OverdueImprovement, len(r.OverdueItems)), Departments: make([]*pb.ImprovementDepartmentRow, len(r.Departments))}
	for i, b := range r.Trend {
		out.Trend[i] = &pb.ImprovementBucket{Bucket: b.Bucket, Manual: int32(b.Manual), Ai: int32(b.AI), Retro: int32(b.Retro),
			Automatic: int32(b.Automatic), Completed: int32(b.Completed), AiAccepted: int32(b.AIAccepted), AiDeclined: int32(b.AIDeclined)}
	}
	for i, m := range r.OverdueItems {
		out.OverdueItems[i] = &pb.OverdueImprovement{Id: m.ID, Title: m.Title, Scope: m.Scope, Source: m.Source, Status: m.Status,
			OwnerName: m.OwnerName, DepartmentName: m.DepartmentName, DueDate: m.DueDate, DaysOverdue: int32(m.DaysOverdue)}
	}
	for i, d := range r.Departments {
		out.Departments[i] = &pb.ImprovementDepartmentRow{DepartmentId: d.DepartmentID, DepartmentName: d.DepartmentName, Open: int32(d.Open),
			InProgress: int32(d.InProgress), Done: int32(d.Done), Overdue: int32(d.Overdue), Suggested: int32(d.Suggested)}
	}
	return out, nil
}
