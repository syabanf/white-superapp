// Package postgres implements the performance ports on top of sqlc queries.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

type Store struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: sqlc.New(pool)}
}

var _ domain.Store = (*Store)(nil)

func (s *Store) WithTx(ctx context.Context, fn func(tx domain.Store) error) error {
	return pg.Tx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&Store{pool: s.pool, q: s.q.WithTx(tx)})
	})
}

// wrap turns unique violations into *domain.ErrUnique.
func wrap(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ErrUnique{Constraint: pgErr.ConstraintName}
	}
	return err
}

// one maps pgx.ErrNoRows to a nil result.
func one[R any, T any](r R, err error, conv func(R) T) (*T, error) {
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := conv(r)
	return &v, nil
}

func each[R any, T any](rows []R, err error, conv func(R) T) ([]T, error) {
	if err != nil {
		return nil, err
	}
	out := make([]T, len(rows))
	for i, r := range rows {
		out[i] = conv(r)
	}
	return out, nil
}

// ─── conversions ─────────────────────────────────────────────

func day(t time.Time) string { return t.Format(domain.IsoDate) }

func dayPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := day(*t)
	return &s
}

func toDay(s string) time.Time {
	t, _ := domain.ParseDay(s)
	return t
}

func toDayPtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, ok := domain.ParseDay(*s)
	if !ok {
		return nil
	}
	return &t
}

func i32p(n *int) *int32 {
	if n == nil {
		return nil
	}
	v := int32(*n)
	return &v
}

func intp(n *int32) *int {
	if n == nil {
		return nil
	}
	v := int(*n)
	return &v
}

// filterIDs keeps nil as "no filter" (SQL NULL).
func filterIDs(ids []string) []string {
	if ids == nil {
		return nil
	}
	return append([]string{}, ids...)
}

func jsonOr(v any, empty string) []byte {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return []byte(empty)
	}
	return b
}

// ─── KPI ─────────────────────────────────────────────────────

func toIndicator(r sqlc.PerformanceKpiIndicator) domain.Indicator {
	return domain.Indicator{ID: r.ID, Code: r.Code, Name: r.Name, Description: r.Description, Unit: r.Unit, Direction: r.Direction,
		SourceKind: r.SourceKind, DefaultTarget: r.DefaultTarget, IsActive: r.IsActive, CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) ListIndicators(ctx context.Context, includeInactive bool) ([]domain.Indicator, error) {
	rows, err := s.q.ListIndicators(ctx, includeInactive)
	return each(rows, err, toIndicator)
}

func (s *Store) GetIndicator(ctx context.Context, id string) (*domain.Indicator, error) {
	r, err := s.q.GetIndicator(ctx, id)
	return one(r, err, toIndicator)
}

func (s *Store) GetIndicatorByCode(ctx context.Context, code string) (*domain.Indicator, error) {
	r, err := s.q.GetIndicatorByCode(ctx, code)
	return one(r, err, toIndicator)
}

func (s *Store) UpsertIndicator(ctx context.Context, in domain.Indicator) error {
	return wrap(s.q.UpsertIndicator(ctx, sqlc.UpsertIndicatorParams{ID: in.ID, Code: in.Code, Name: in.Name, Description: in.Description,
		Unit: in.Unit, Direction: in.Direction, SourceKind: in.SourceKind, DefaultTarget: in.DefaultTarget, IsActive: in.IsActive, CreatedBy: in.UpdatedBy}))
}

func (s *Store) ListDepartmentWeights(ctx context.Context) ([]domain.DepartmentWeight, error) {
	rows, err := s.q.ListDepartmentWeights(ctx)
	return each(rows, err, func(r sqlc.ListDepartmentWeightsRow) domain.DepartmentWeight {
		return domain.DepartmentWeight{DepartmentID: r.DepartmentID, IndicatorID: r.IndicatorID, IndicatorCode: r.Code, Weight: r.Weight,
			UpdatedBy: r.UpdatedBy, UpdatedAt: r.UpdatedAt}
	})
}

func (s *Store) DeleteDepartmentWeights(ctx context.Context, departmentID string) error {
	return s.q.DeleteDepartmentWeights(ctx, departmentID)
}

func (s *Store) InsertDepartmentWeight(ctx context.Context, w domain.DepartmentWeight) error {
	return wrap(s.q.InsertDepartmentWeight(ctx, sqlc.InsertDepartmentWeightParams{DepartmentID: w.DepartmentID, IndicatorID: w.IndicatorID,
		Weight: w.Weight, UpdatedBy: w.UpdatedBy}))
}

func toTarget(r sqlc.PerformanceKpiTarget) domain.KPITarget {
	return domain.KPITarget{ID: r.ID, IndicatorID: r.IndicatorID, PeriodYear: intp(r.PeriodYear), PeriodMonth: intp(r.PeriodMonth),
		DepartmentID: r.DepartmentID, EmployeeID: r.EmployeeID, Target: r.Target}
}

func (s *Store) ListTargets(ctx context.Context) ([]domain.KPITarget, error) {
	rows, err := s.q.ListTargets(ctx)
	return each(rows, err, toTarget)
}

func (s *Store) InsertTarget(ctx context.Context, t domain.KPITarget, by string) error {
	return wrap(s.q.InsertTarget(ctx, sqlc.InsertTargetParams{ID: t.ID, IndicatorID: t.IndicatorID, PeriodYear: i32p(t.PeriodYear),
		PeriodMonth: i32p(t.PeriodMonth), DepartmentID: t.DepartmentID, EmployeeID: t.EmployeeID, Target: t.Target, CreatedBy: &by}))
}

func (s *Store) DeleteTarget(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteTarget(ctx, id)
	return n > 0, err
}

func (s *Store) UpsertSnapshot(ctx context.Context, sn domain.Snapshot) error {
	detail := sn.SourceDetail
	if len(detail) == 0 {
		detail = []byte("{}")
	}
	return s.q.UpsertSnapshot(ctx, sqlc.UpsertSnapshotParams{ID: sn.ID, EmployeeID: sn.EmployeeID, IndicatorID: sn.IndicatorID,
		PeriodYear: int32(sn.Period.Year), PeriodMonth: int32(sn.Period.Month), Actual: sn.Actual, Target: sn.Target,
		Attainment: sn.Attainment, SampleSize: sn.SampleSize, SourceDetail: detail})
}

func toSnapshot(r sqlc.ListSnapshotsRow) domain.Snapshot {
	return domain.Snapshot{ID: r.ID, EmployeeID: r.EmployeeID, IndicatorID: r.IndicatorID, IndicatorCode: r.IndicatorCode,
		IndicatorName: r.IndicatorName, SourceKind: r.SourceKind, Period: domain.Period{Year: int(r.PeriodYear), Month: int(r.PeriodMonth)},
		Actual: r.Actual, Target: r.Target, Attainment: r.Attainment, SampleSize: r.SampleSize, SourceDetail: r.SourceDetail, UpdatedAt: r.UpdatedAt}
}

func (s *Store) ListSnapshots(ctx context.Context, employeeID string, p domain.Period) ([]domain.Snapshot, error) {
	rows, err := s.q.ListSnapshots(ctx, sqlc.ListSnapshotsParams{EmployeeID: employeeID, PeriodYear: int32(p.Year), PeriodMonth: int32(p.Month)})
	return each(rows, err, toSnapshot)
}

func (s *Store) ListManualSnapshots(ctx context.Context, p domain.Period) ([]domain.Snapshot, error) {
	rows, err := s.q.ListManualSnapshots(ctx, sqlc.ListManualSnapshotsParams{PeriodYear: int32(p.Year), PeriodMonth: int32(p.Month)})
	return each(rows, err, func(r sqlc.ListManualSnapshotsRow) domain.Snapshot { return toSnapshot(sqlc.ListSnapshotsRow(r)) })
}

func toScorecard(r sqlc.PerformanceKpiScorecard) domain.Scorecard {
	var breakdown []domain.BreakdownRow
	_ = json.Unmarshal(r.Breakdown, &breakdown)
	return domain.Scorecard{ID: r.ID, EmployeeID: r.EmployeeID, DepartmentID: r.DepartmentID,
		Period: domain.Period{Year: int(r.PeriodYear), Month: int(r.PeriodMonth)}, Score: r.Score, RawScore: r.RawScore,
		UsedWeight: r.UsedWeight, Breakdown: breakdown, Status: r.Status, ReviewedBy: r.ReviewedBy, ReviewedAt: r.ReviewedAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) UpsertDraftScorecard(ctx context.Context, sc domain.Scorecard) (bool, error) {
	n, err := s.q.UpsertDraftScorecard(ctx, sqlc.UpsertDraftScorecardParams{ID: sc.ID, EmployeeID: sc.EmployeeID, DepartmentID: sc.DepartmentID,
		PeriodYear: int32(sc.Period.Year), PeriodMonth: int32(sc.Period.Month), Score: sc.Score, RawScore: sc.RawScore,
		UsedWeight: sc.UsedWeight, Breakdown: jsonOr(sc.Breakdown, "[]")})
	return n > 0, err
}

func (s *Store) GetScorecard(ctx context.Context, id string) (*domain.Scorecard, error) {
	r, err := s.q.GetScorecard(ctx, id)
	return one(r, err, toScorecard)
}

func (s *Store) GetScorecardFor(ctx context.Context, employeeID string, p domain.Period) (*domain.Scorecard, error) {
	r, err := s.q.GetScorecardFor(ctx, sqlc.GetScorecardForParams{EmployeeID: employeeID, PeriodYear: int32(p.Year), PeriodMonth: int32(p.Month)})
	return one(r, err, toScorecard)
}

func (s *Store) ListScorecards(ctx context.Context, p domain.Period, employeeIDs []string) ([]domain.Scorecard, error) {
	rows, err := s.q.ListScorecards(ctx, sqlc.ListScorecardsParams{PeriodYear: int32(p.Year), PeriodMonth: int32(p.Month), EmployeeIds: filterIDs(employeeIDs)})
	return each(rows, err, toScorecard)
}

func (s *Store) ScorecardHistory(ctx context.Context, employeeID string, limit int) ([]domain.Scorecard, error) {
	rows, err := s.q.ScorecardHistory(ctx, sqlc.ScorecardHistoryParams{EmployeeID: employeeID, Limit: int32(limit)})
	return each(rows, err, toScorecard)
}

func (s *Store) QuarterScorecards(ctx context.Context, employeeIDs []string, year int, months []int) ([]domain.Scorecard, error) {
	ms := make([]int32, len(months))
	for i, m := range months {
		ms[i] = int32(m)
	}
	rows, err := s.q.QuarterScorecards(ctx, sqlc.QuarterScorecardsParams{EmployeeIds: append([]string{}, employeeIDs...), PeriodYear: int32(year), Months: ms})
	return each(rows, err, toScorecard)
}

func (s *Store) SetScorecardStatus(ctx context.Context, id, status string, reviewedBy *string, at *time.Time) error {
	return s.q.SetScorecardStatus(ctx, sqlc.SetScorecardStatusParams{ID: id, Status: status, ReviewedBy: reviewedBy, ReviewedAt: at})
}

func (s *Store) FinalScorecardEmployees(ctx context.Context, p domain.Period) ([]string, error) {
	return s.q.FinalScorecardEmployees(ctx, sqlc.FinalScorecardEmployeesParams{PeriodYear: int32(p.Year), PeriodMonth: int32(p.Month)})
}

func (s *Store) RecentScoreAverages(ctx context.Context, employeeIDs []string) ([]domain.ScoreAverage, error) {
	rows, err := s.q.RecentScoreAverages(ctx, append([]string{}, employeeIDs...))
	return each(rows, err, func(r sqlc.RecentScoreAveragesRow) domain.ScoreAverage {
		return domain.ScoreAverage{EmployeeID: r.EmployeeID, AvgScore: domain.Round2(r.AvgScore), Periods: int(r.Periods), HasFinal: r.HasFinal}
	})
}

func (s *Store) TaskCounts(ctx context.Context, span domain.DaySpan) ([]domain.TaskCount, error) {
	rows, err := s.q.TaskCounts(ctx, sqlc.TaskCountsParams{StartDate: toDay(span.Start), EndDate: toDay(span.End)})
	return each(rows, err, func(r sqlc.TaskCountsRow) domain.TaskCount {
		return domain.TaskCount{DepartmentID: r.DepartmentID, AssigneeEmployeeID: r.AssigneeEmployeeID, DoneBy: r.DoneBy, Status: r.Status, Count: int(r.Total)}
	})
}

func (s *Store) LogbookStats(ctx context.Context, span domain.DaySpan) (map[string]domain.LogbookStats, error) {
	rows, err := s.q.LogbookStats(ctx, sqlc.LogbookStatsParams{StartDate: toDay(span.Start), EndDate: toDay(span.End)})
	if err != nil {
		return nil, err
	}
	out := make(map[string]domain.LogbookStats, len(rows))
	for _, r := range rows {
		out[r.EmployeeID] = domain.LogbookStats{Entries: int(r.Entries), LogbookCount: domain.LogbookCount{Checked: int(r.Checked), Total: int(r.Total)}}
	}
	return out, nil
}

// ─── Reviews ─────────────────────────────────────────────────

func toCycle(r sqlc.PerformanceReviewCycle) domain.ReviewCycle {
	return domain.ReviewCycle{ID: r.ID, Name: r.Name, Year: int(r.PeriodYear), Quarter: int(r.PeriodQuarter), StartDate: day(r.StartDate),
		EndDate: day(r.EndDate), DueDate: day(r.DueDate), Status: r.Status, CreatedBy: r.CreatedBy, CreatedByName: r.CreatedByName, CreatedAt: r.CreatedAt}
}

func (s *Store) ListReviewCycles(ctx context.Context) ([]domain.ReviewCycle, error) {
	rows, err := s.q.ListReviewCycles(ctx)
	return each(rows, err, func(r sqlc.ListReviewCyclesRow) domain.ReviewCycle {
		c := toCycle(sqlc.PerformanceReviewCycle{ID: r.ID, Name: r.Name, PeriodYear: r.PeriodYear, PeriodQuarter: r.PeriodQuarter,
			StartDate: r.StartDate, EndDate: r.EndDate, DueDate: r.DueDate, Status: r.Status, CreatedBy: r.CreatedBy,
			CreatedByName: r.CreatedByName, CreatedAt: r.CreatedAt})
		c.TotalReviews, c.FinalReviews, c.RatedReviews = int(r.TotalReviews), int(r.FinalReviews), int(r.RatedReviews)
		return c
	})
}

func (s *Store) GetReviewCycle(ctx context.Context, id string) (*domain.ReviewCycle, error) {
	r, err := s.q.GetReviewCycle(ctx, id)
	return one(r, err, toCycle)
}

func (s *Store) ReviewCycleExists(ctx context.Context, year, quarter int) (bool, error) {
	return s.q.ReviewCycleExists(ctx, sqlc.ReviewCycleExistsParams{PeriodYear: int32(year), PeriodQuarter: int32(quarter)})
}

func (s *Store) InsertReviewCycle(ctx context.Context, c domain.ReviewCycle) error {
	return wrap(s.q.InsertReviewCycle(ctx, sqlc.InsertReviewCycleParams{ID: c.ID, Name: c.Name, PeriodYear: int32(c.Year),
		PeriodQuarter: int32(c.Quarter), StartDate: toDay(c.StartDate), EndDate: toDay(c.EndDate), DueDate: toDay(c.DueDate),
		Status: c.Status, CreatedBy: c.CreatedBy, CreatedByName: c.CreatedByName}))
}

func (s *Store) SetReviewCycleStatus(ctx context.Context, id, status string) error {
	return s.q.SetReviewCycleStatus(ctx, sqlc.SetReviewCycleStatusParams{ID: id, Status: status})
}

func (s *Store) ListBehavioralStandards(ctx context.Context) ([]domain.BehavioralStandard, error) {
	rows, err := s.q.ListBehavioralStandards(ctx)
	return each(rows, err, func(r sqlc.PerformanceBehavioralStandard) domain.BehavioralStandard {
		return domain.BehavioralStandard{ID: r.ID, ValueName: r.ValueName, CompetencyName: r.CompetencyName, StandardDescription: r.StandardDescription,
			ScoreDescriptions: [5]string{r.Score1Description, r.Score2Description, r.Score3Description, r.Score4Description, r.Score5Description},
			Weight:            r.Weight}
	})
}

func (s *Store) ListPerformanceCategories(ctx context.Context) ([]domain.Category, error) {
	rows, err := s.q.ListPerformanceCategories(ctx)
	return each(rows, err, func(r sqlc.PerformancePerformanceCategory) domain.Category {
		return domain.Category{Name: r.CategoryName, MinScore: r.MinScore, MaxScore: r.MaxScore}
	})
}

func (s *Store) InsertReview(ctx context.Context, r domain.Review) error {
	return wrap(s.q.InsertReview(ctx, sqlc.InsertReviewParams{ID: r.ID, CycleID: r.CycleID, EmployeeID: r.EmployeeID, DepartmentID: r.DepartmentID,
		PeriodLabel: r.PeriodLabel, StartDate: toDay(r.StartDate), EndDate: toDay(r.EndDate)}))
}

func (s *Store) InsertReviewItem(ctx context.Context, it domain.ReviewItem) error {
	d := it.ScoreDescriptions
	return s.q.InsertReviewItem(ctx, sqlc.InsertReviewItemParams{ID: it.ID, ReviewID: it.ReviewID, ValueName: it.ValueName, Competency: it.Competency,
		BehavioralStandard: it.BehavioralStandard, Score1Description: d[0], Score2Description: d[1], Score3Description: d[2],
		Score4Description: d[3], Score5Description: d[4], Weight: it.Weight, ItemOrder: int32(it.ItemOrder)})
}

func toReview(r sqlc.GetReviewRow) domain.Review {
	return domain.Review{ID: r.ID, CycleID: r.CycleID, CycleName: r.CycleName, CycleStatus: r.CycleStatus, CycleDueDate: day(r.CycleDueDate),
		CycleYear: int(r.CycleYear), CycleQuarter: int(r.CycleQuarter), EmployeeID: r.EmployeeID, DepartmentID: r.DepartmentID,
		PeriodLabel: r.PeriodLabel, StartDate: day(r.StartDate), EndDate: day(r.EndDate), Work: r.TotalWorkResultScore,
		Behavior: r.TotalBehavioralScore, Project: r.TotalProjectScore, Grand: r.GrandTotalScore, Category: r.Category, Status: r.Status,
		SelfAssessment: r.SelfAssessment, ReviewerID: r.ReviewerID, ReviewerName: r.ReviewerName, ReviewerNotes: r.ReviewerNotes,
		ManagerID: r.ManagerID, ManagerNotes: r.ManagerNotes, EmployeeSignDate: dayPtr(r.EmployeeSignDate), ReviewerSignDate: dayPtr(r.ReviewerSignDate),
		FinalizedBy: r.FinalizedBy, FinalizedAt: r.FinalizedAt, RatedItems: int(r.RatedItems), TotalItems: int(r.TotalItems),
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) ListReviewsByCycle(ctx context.Context, cycleID string) ([]domain.Review, error) {
	rows, err := s.q.ListReviewsByCycle(ctx, cycleID)
	return each(rows, err, func(r sqlc.ListReviewsByCycleRow) domain.Review { return toReview(sqlc.GetReviewRow(r)) })
}

func (s *Store) ListOpenCycleReviews(ctx context.Context) ([]domain.Review, error) {
	rows, err := s.q.ListOpenCycleReviews(ctx)
	return each(rows, err, func(r sqlc.ListOpenCycleReviewsRow) domain.Review { return toReview(sqlc.GetReviewRow(r)) })
}

func (s *Store) GetReview(ctx context.Context, id string) (*domain.Review, error) {
	r, err := s.q.GetReview(ctx, id)
	return one(r, err, toReview)
}

func (s *Store) UpdateReview(ctx context.Context, r domain.Review) error {
	return s.q.UpdateReview(ctx, sqlc.UpdateReviewParams{ID: r.ID, TotalWorkResultScore: r.Work, TotalBehavioralScore: r.Behavior,
		TotalProjectScore: r.Project, GrandTotalScore: r.Grand, Category: r.Category, Status: r.Status, SelfAssessment: r.SelfAssessment,
		ReviewerID: r.ReviewerID, ReviewerName: r.ReviewerName, ReviewerNotes: r.ReviewerNotes, ManagerID: r.ManagerID,
		ManagerNotes: r.ManagerNotes, EmployeeSignDate: toDayPtr(r.EmployeeSignDate), ReviewerSignDate: toDayPtr(r.ReviewerSignDate),
		FinalizedBy: r.FinalizedBy, FinalizedAt: r.FinalizedAt})
}

func (s *Store) ListReviewItems(ctx context.Context, reviewID string) ([]domain.ReviewItem, error) {
	rows, err := s.q.ListReviewItems(ctx, reviewID)
	return each(rows, err, func(r sqlc.PerformanceBehavioralReviewItem) domain.ReviewItem {
		return domain.ReviewItem{ID: r.ID, ReviewID: r.ReviewID, ValueName: r.ValueName, Competency: r.Competency, BehavioralStandard: r.BehavioralStandard,
			ScoreDescriptions: [5]string{r.Score1Description, r.Score2Description, r.Score3Description, r.Score4Description, r.Score5Description},
			Weight:            r.Weight, Score: intp(r.Score), WeightedScore: r.WeightedScore, Notes: r.Notes, ItemOrder: int(r.ItemOrder)}
	})
}

func (s *Store) UpdateReviewItem(ctx context.Context, it domain.ReviewItem) error {
	return s.q.UpdateReviewItem(ctx, sqlc.UpdateReviewItemParams{ID: it.ID, Score: i32p(it.Score), WeightedScore: it.WeightedScore, Notes: it.Notes})
}

// ─── Feedback ────────────────────────────────────────────────

func (s *Store) ListFeedbackCategories(ctx context.Context) ([]domain.FeedbackCategory, error) {
	cats, err := s.q.ListFeedbackCategories(ctx)
	if err != nil {
		return nil, err
	}
	crit, err := s.q.ListFeedbackCriteria(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.FeedbackCategory, len(cats))
	for i, c := range cats {
		out[i] = domain.FeedbackCategory{ID: c.ID, Code: c.Code, Name: c.Name, Description: c.Description, Weight: c.Weight,
			DisplayOrder: int(c.DisplayOrder), Criteria: []domain.FeedbackCriterion{}}
		for _, cr := range crit {
			if cr.CategoryID == c.ID {
				out[i].Criteria = append(out[i].Criteria, domain.FeedbackCriterion{ID: cr.ID, CategoryID: cr.CategoryID, Name: cr.Name,
					Description: cr.Description, DisplayOrder: int(cr.DisplayOrder)})
			}
		}
	}
	return out, nil
}

func toFeedbackCycle(r sqlc.GetFeedbackCycleRow) domain.FeedbackCycle {
	return domain.FeedbackCycle{ID: r.ID, Name: r.Name, PeriodLabel: r.PeriodLabel, StartDate: dayPtr(r.StartDate), EndDate: dayPtr(r.EndDate),
		Status: r.Status, KPIWeight: r.KpiWeight, FeedbackWeight: r.FeedbackWeight, IsAnonymous: r.IsAnonymous,
		AllowSelfAssessment: r.AllowSelfAssessment, RequireManagerReview: r.RequireManagerReview, CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, AssignmentsCount: int(r.AssignmentsCount), SubmittedCount: int(r.SubmittedCount),
		SummariesCount: int(r.SummariesCount)}
}

func (s *Store) ListFeedbackCycles(ctx context.Context, status string) ([]domain.FeedbackCycle, error) {
	rows, err := s.q.ListFeedbackCycles(ctx, status)
	return each(rows, err, func(r sqlc.ListFeedbackCyclesRow) domain.FeedbackCycle {
		return toFeedbackCycle(sqlc.GetFeedbackCycleRow(r))
	})
}

func (s *Store) GetFeedbackCycle(ctx context.Context, id string) (*domain.FeedbackCycle, error) {
	r, err := s.q.GetFeedbackCycle(ctx, id)
	return one(r, err, toFeedbackCycle)
}

func (s *Store) InsertFeedbackCycle(ctx context.Context, c domain.FeedbackCycle) error {
	return s.q.InsertFeedbackCycle(ctx, sqlc.InsertFeedbackCycleParams{ID: c.ID, Name: c.Name, PeriodLabel: c.PeriodLabel,
		StartDate: toDayPtr(c.StartDate), EndDate: toDayPtr(c.EndDate), Status: c.Status, KpiWeight: c.KPIWeight, FeedbackWeight: c.FeedbackWeight,
		IsAnonymous: c.IsAnonymous, AllowSelfAssessment: c.AllowSelfAssessment, RequireManagerReview: c.RequireManagerReview, CreatedBy: c.CreatedBy})
}

func (s *Store) UpdateFeedbackCycle(ctx context.Context, c domain.FeedbackCycle) error {
	return s.q.UpdateFeedbackCycle(ctx, sqlc.UpdateFeedbackCycleParams{ID: c.ID, Name: c.Name, PeriodLabel: c.PeriodLabel,
		StartDate: toDayPtr(c.StartDate), EndDate: toDayPtr(c.EndDate), Status: c.Status, KpiWeight: c.KPIWeight, FeedbackWeight: c.FeedbackWeight,
		IsAnonymous: c.IsAnonymous, AllowSelfAssessment: c.AllowSelfAssessment, RequireManagerReview: c.RequireManagerReview, UpdatedBy: c.UpdatedBy})
}

func (s *Store) DeleteFeedbackCycle(ctx context.Context, id string) error {
	return s.q.DeleteFeedbackCycle(ctx, id)
}

func (s *Store) InsertAssignment(ctx context.Context, a domain.FeedbackAssignment) (bool, error) {
	n, err := s.q.InsertAssignment(ctx, sqlc.InsertAssignmentParams{ID: a.ID, CycleID: a.CycleID, EmployeeID: a.EmployeeID, ReviewerID: a.ReviewerID,
		RelationshipType: a.Relationship, DueDate: toDayPtr(a.DueDate), CreatedBy: a.CreatedBy})
	return n > 0, err
}

func toAssignment(r sqlc.GetAssignmentRow) domain.FeedbackAssignment {
	return domain.FeedbackAssignment{ID: r.ID, CycleID: r.CycleID, CycleName: r.CycleName, CycleStatus: r.CycleStatus, EmployeeID: r.EmployeeID,
		ReviewerID: r.ReviewerID, Relationship: r.RelationshipType, Status: r.Status, DueDate: dayPtr(r.DueDate), StartedAt: r.StartedAt,
		SubmittedAt: r.SubmittedAt, ReminderCount: int(r.ReminderCount), LastReminderAt: r.LastReminderAt, ManagerComments: r.ManagerComments,
		RejectionReason: r.RejectionReason, ApprovedBy: r.ApprovedBy, ApprovedAt: r.ApprovedAt, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) ListAssignments(ctx context.Context, cycleID, employeeID, reviewerID, status string) ([]domain.FeedbackAssignment, error) {
	rows, err := s.q.ListAssignments(ctx, sqlc.ListAssignmentsParams{CycleID: cycleID, EmployeeID: employeeID, ReviewerID: reviewerID, Status: status})
	return each(rows, err, func(r sqlc.ListAssignmentsRow) domain.FeedbackAssignment {
		return toAssignment(sqlc.GetAssignmentRow(r))
	})
}

func (s *Store) ListRemindableAssignments(ctx context.Context) ([]domain.FeedbackAssignment, error) {
	rows, err := s.q.ListRemindableAssignments(ctx)
	return each(rows, err, func(r sqlc.ListRemindableAssignmentsRow) domain.FeedbackAssignment {
		return toAssignment(sqlc.GetAssignmentRow(r))
	})
}

func (s *Store) GetAssignment(ctx context.Context, id string) (*domain.FeedbackAssignment, error) {
	r, err := s.q.GetAssignment(ctx, id)
	return one(r, err, toAssignment)
}

func (s *Store) UpdateAssignment(ctx context.Context, a domain.FeedbackAssignment) error {
	return s.q.UpdateAssignment(ctx, sqlc.UpdateAssignmentParams{ID: a.ID, RelationshipType: a.Relationship, Status: a.Status,
		DueDate: toDayPtr(a.DueDate), StartedAt: a.StartedAt, SubmittedAt: a.SubmittedAt, ReminderCount: int32(a.ReminderCount),
		LastReminderAt: a.LastReminderAt, ManagerComments: a.ManagerComments, RejectionReason: a.RejectionReason,
		ApprovedBy: a.ApprovedBy, ApprovedAt: a.ApprovedAt})
}

func (s *Store) DeleteAssignment(ctx context.Context, id string) error {
	return s.q.DeleteAssignment(ctx, id)
}

func (s *Store) UpsertResponse(ctx context.Context, r domain.FeedbackResponse) error {
	return s.q.UpsertResponse(ctx, sqlc.UpsertResponseParams{ID: r.ID, AssignmentID: r.AssignmentID, CriteriaID: r.CriteriaID,
		Rating: int32(r.Rating), Comments: r.Comments})
}

func (s *Store) ListResponses(ctx context.Context, assignmentID string) ([]domain.FeedbackResponse, error) {
	rows, err := s.q.ListResponses(ctx, assignmentID)
	return each(rows, err, func(r sqlc.PerformanceFeedbackResponse) domain.FeedbackResponse {
		return domain.FeedbackResponse{ID: r.ID, AssignmentID: r.AssignmentID, CriteriaID: r.CriteriaID, Rating: int(r.Rating), Comments: r.Comments}
	})
}

func (s *Store) CycleRatings(ctx context.Context, cycleID string) ([]domain.CycleRating, error) {
	rows, err := s.q.CycleRatings(ctx, cycleID)
	return each(rows, err, func(r sqlc.CycleRatingsRow) domain.CycleRating {
		return domain.CycleRating{EmployeeID: r.EmployeeID, AssignmentStatus: r.AssignmentStatus, Comments: r.Comments,
			FeedbackRating: domain.FeedbackRating{CategoryCode: r.CategoryCode, CategoryName: r.CategoryName, CriteriaName: r.CriteriaName,
				Relationship: r.RelationshipType, Rating: int(r.Rating)}}
	})
}

func toFeedbackSummary(r sqlc.PerformanceFeedbackSummary) domain.FeedbackSummary {
	return domain.FeedbackSummary{ID: r.ID, CycleID: r.CycleID, EmployeeID: r.EmployeeID,
		Scores: map[string]*float64{"leadership": r.LeadershipScore, "communication": r.CommunicationScore, "collaboration": r.CollaborationScore,
			"accountability": r.AccountabilityScore, "problem_solving": r.ProblemSolvingScore},
		Overall360: r.Overall360Score, KPIScore: r.KpiScore, FinalScore: r.FinalScore, FinalGrade: r.FinalGrade, Strengths: r.Strengths,
		Weaknesses: r.Weaknesses, ManagerComments: r.ManagerComments, ResponsesCount: int(r.ResponsesCount), IsLocked: r.IsLocked,
		GeneratedAt: r.GeneratedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) UpsertFeedbackSummary(ctx context.Context, f domain.FeedbackSummary) error {
	return s.q.UpsertFeedbackSummary(ctx, sqlc.UpsertFeedbackSummaryParams{ID: f.ID, CycleID: f.CycleID, EmployeeID: f.EmployeeID,
		LeadershipScore: f.Scores["leadership"], CommunicationScore: f.Scores["communication"], CollaborationScore: f.Scores["collaboration"],
		AccountabilityScore: f.Scores["accountability"], ProblemSolvingScore: f.Scores["problem_solving"], Overall360Score: f.Overall360,
		KpiScore: f.KPIScore, FinalScore: f.FinalScore, FinalGrade: f.FinalGrade, Strengths: append([]string{}, f.Strengths...),
		Weaknesses: append([]string{}, f.Weaknesses...), ResponsesCount: int32(f.ResponsesCount)})
}

func (s *Store) ListFeedbackSummaries(ctx context.Context, cycleID, employeeID string) ([]domain.FeedbackSummary, error) {
	rows, err := s.q.ListFeedbackSummaries(ctx, sqlc.ListFeedbackSummariesParams{CycleID: cycleID, EmployeeID: employeeID})
	return each(rows, err, toFeedbackSummary)
}

func (s *Store) GetFeedbackSummary(ctx context.Context, id string) (*domain.FeedbackSummary, error) {
	r, err := s.q.GetFeedbackSummary(ctx, id)
	return one(r, err, toFeedbackSummary)
}

func (s *Store) UpdateFeedbackSummary(ctx context.Context, f domain.FeedbackSummary) error {
	return s.q.UpdateFeedbackSummary(ctx, sqlc.UpdateFeedbackSummaryParams{ID: f.ID, Strengths: append([]string{}, f.Strengths...),
		Weaknesses: append([]string{}, f.Weaknesses...), ManagerComments: f.ManagerComments, IsLocked: f.IsLocked})
}

// ─── Reminder log ────────────────────────────────────────────

func (s *Store) ClaimSend(ctx context.Context, kind, subjectID, dayKey string) (bool, error) {
	n, err := s.q.ClaimSend(ctx, sqlc.ClaimSendParams{Kind: kind, SubjectID: subjectID, Day: toDay(dayKey)})
	return n > 0, err
}

func newID() string { return ids.New() }
