package postgres

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

func idsOrEmpty(ids []string) []string {
	if ids == nil {
		return []string{}
	}
	return ids
}

// countOr is a nullable average: nil when no row had a value.
func countOr(avg float64, n int32) *float64 {
	if n == 0 {
		return nil
	}
	return &avg
}

func (s *Store) DashTaskDays(ctx context.Context, f domain.DashFilter) ([]domain.TaskDay, error) {
	rows, err := s.q.DashTaskDays(ctx, sqlc.DashTaskDaysParams{FromDate: toDay(f.From), ToDate: toDay(f.To), AllEmployees: f.All,
		EmployeeIds: idsOrEmpty(f.EmployeeIDs), DepartmentID: f.DepartmentID})
	return each(rows, err, func(r sqlc.DashTaskDaysRow) domain.TaskDay {
		return domain.TaskDay{Day: day(r.Day), DepartmentID: r.DepartmentID, EmployeeID: r.EmployeeID, Pending: int(r.Pending),
			InProgress: int(r.InProgress), Done: int(r.Done), Approved: int(r.Approved), Rejected: int(r.Rejected), OnTime: int(r.OnTime),
			Reviewed: int(r.Reviewed), TurnaroundHours: r.TurnaroundHours, TurnLt4h: int(r.TurnLt4h), Turn4to24h: int(r.Turn424h),
			Turn1to3d: int(r.Turn13d), TurnOver3d: int(r.TurnGt3d)}
	})
}

func (s *Store) DashLogbookDays(ctx context.Context, f domain.DashFilter) ([]domain.LogbookDay, error) {
	rows, err := s.q.DashLogbookDays(ctx, sqlc.DashLogbookDaysParams{FromDate: toDay(f.From), ToDate: toDay(f.To), AllEmployees: f.All,
		EmployeeIds: idsOrEmpty(f.EmployeeIDs), DepartmentID: f.DepartmentID})
	return each(rows, err, func(r sqlc.DashLogbookDaysRow) domain.LogbookDay {
		return domain.LogbookDay{Day: day(r.Day), DepartmentID: r.DepartmentID, EmployeeID: r.EmployeeID, Draft: int(r.Draft),
			Submitted: int(r.Submitted), Reviewed: int(r.Reviewed), Rejected: int(r.Rejected), ScoreSum: r.ScoreSum}
	})
}

func (s *Store) DashRecapDays(ctx context.Context, f domain.DashFilter) ([]domain.RecapDay, error) {
	rows, err := s.q.DashRecapDays(ctx, sqlc.DashRecapDaysParams{FromDate: toDay(f.From), ToDate: toDay(f.To), AllEmployees: f.All,
		EmployeeIds: idsOrEmpty(f.EmployeeIDs), DepartmentID: f.DepartmentID})
	return each(rows, err, func(r sqlc.DashRecapDaysRow) domain.RecapDay {
		return domain.RecapDay{Day: day(r.Day), EmployeeID: r.EmployeeID, Recaps: int(r.Recaps)}
	})
}

func (s *Store) DashScorecards(ctx context.Context, f domain.DashFilter) ([]domain.DashScorecard, error) {
	rows, err := s.q.DashScorecards(ctx, sqlc.DashScorecardsParams{FromDate: toDay(f.From), ToDate: toDay(f.To), AllEmployees: f.All,
		EmployeeIds: idsOrEmpty(f.EmployeeIDs), DepartmentID: f.DepartmentID})
	return each(rows, err, func(r sqlc.DashScorecardsRow) domain.DashScorecard {
		return domain.DashScorecard{EmployeeID: r.EmployeeID, DepartmentID: r.DepartmentID,
			Period: domain.Period{Year: int(r.PeriodYear), Month: int(r.PeriodMonth)}, Score: r.Score, Status: r.Status}
	})
}

func (s *Store) DashIndicatorAverages(ctx context.Context, f domain.DashFilter) ([]domain.IndicatorAverage, error) {
	rows, err := s.q.DashIndicatorAttainment(ctx, sqlc.DashIndicatorAttainmentParams{FromDate: toDay(f.From), ToDate: toDay(f.To),
		AllEmployees: f.All, EmployeeIds: idsOrEmpty(f.EmployeeIDs)})
	return each(rows, err, func(r sqlc.DashIndicatorAttainmentRow) domain.IndicatorAverage {
		return domain.IndicatorAverage{IndicatorID: r.IndicatorID, Code: r.Code, Name: r.Name, Unit: r.Unit, Direction: r.Direction,
			Actual: countOr(r.Actual, r.Actuals), Target: countOr(r.Target, r.Targets), Attainment: countOr(r.Attainment, r.Samples),
			Samples: int(r.Samples)}
	})
}

func (s *Store) DashImprovements(ctx context.Context, f domain.DashFilter) ([]domain.DashImprovement, error) {
	rows, err := s.q.DashImprovements(ctx, sqlc.DashImprovementsParams{ToDate: toDay(f.To), AllEmployees: f.All,
		EmployeeIds: idsOrEmpty(f.EmployeeIDs), DepartmentID: f.DepartmentID})
	return each(rows, err, func(r sqlc.DashImprovementsRow) domain.DashImprovement {
		return domain.DashImprovement{ID: r.ID, Scope: r.Scope, Source: r.Source, Status: r.Status, Title: r.Title, EmployeeID: r.EmployeeID,
			DepartmentID: r.DepartmentID, OwnerEmployeeID: r.OwnerEmployeeID, DueDate: dayPtr(r.DueDate), CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt, CompletedAt: r.CompletedAt}
	})
}

func (s *Store) DashFeedbackAssignments(ctx context.Context, cycleID string, f domain.DashFilter) ([]domain.FeedbackAssignmentCount, error) {
	rows, err := s.q.DashFeedbackAssignments(ctx, sqlc.DashFeedbackAssignmentsParams{CycleID: cycleID, AllEmployees: f.All,
		EmployeeIds: idsOrEmpty(f.EmployeeIDs)})
	return each(rows, err, func(r sqlc.DashFeedbackAssignmentsRow) domain.FeedbackAssignmentCount {
		return domain.FeedbackAssignmentCount{Relationship: r.RelationshipType, Status: r.Status, Total: int(r.Total)}
	})
}
