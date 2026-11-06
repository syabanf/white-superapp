package postgres

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

func toObjective(o sqlc.PerformanceObjective) domain.Objective {
	return domain.Objective{ID: o.ID, Period: o.Period, Level: o.Level, Title: o.Title, Description: o.Description, Status: o.Status,
		DepartmentID: o.DepartmentID, EmployeeID: o.EmployeeID, ParentID: o.ParentID, CreatedBy: o.CreatedBy}
}

func toKeyResult(k sqlc.PerformanceKeyResult) domain.KeyResult {
	return domain.KeyResult{ID: k.ID, ObjectiveID: k.ObjectiveID, Title: k.Title, Metric: k.Metric, Unit: k.Unit, Start: k.StartValue,
		Target: k.TargetValue, Current: k.CurrentValue, SortOrder: int(k.SortOrder)}
}

func (s *Store) ListObjectives(ctx context.Context, period string) ([]domain.Objective, error) {
	rows, err := s.q.ListObjectives(ctx, period)
	return each(rows, err, toObjective)
}

func (s *Store) GetObjective(ctx context.Context, id string) (*domain.Objective, error) {
	r, err := s.q.GetObjective(ctx, id)
	return one(r, err, toObjective)
}

func (s *Store) SaveObjective(ctx context.Context, o domain.Objective) error {
	return s.q.UpsertObjective(ctx, sqlc.UpsertObjectiveParams{ID: o.ID, Period: o.Period, Level: o.Level, DepartmentID: o.DepartmentID, EmployeeID: o.EmployeeID,
		ParentID: o.ParentID, Title: o.Title, Description: o.Description, Status: o.Status, CreatedBy: o.CreatedBy})
}

func (s *Store) DeleteObjective(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteObjective(ctx, id)
	return n > 0, err
}

func (s *Store) ListKeyResults(ctx context.Context, period string) ([]domain.KeyResult, error) {
	rows, err := s.q.ListKeyResults(ctx, period)
	return each(rows, err, toKeyResult)
}

func (s *Store) GetKeyResult(ctx context.Context, id string) (*domain.KeyResult, error) {
	r, err := s.q.GetKeyResult(ctx, id)
	return one(r, err, toKeyResult)
}

func (s *Store) SaveKeyResult(ctx context.Context, k domain.KeyResult) error {
	return s.q.UpsertKeyResult(ctx, sqlc.UpsertKeyResultParams{ID: k.ID, ObjectiveID: k.ObjectiveID, Title: k.Title, Metric: k.Metric, StartValue: k.Start,
		TargetValue: k.Target, CurrentValue: k.Current, Unit: k.Unit, SortOrder: int32(k.SortOrder)})
}

func (s *Store) DeleteKeyResult(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteKeyResult(ctx, id)
	return n > 0, err
}

func (s *Store) SetKeyResultValue(ctx context.Context, id string, v float64) error {
	return s.q.SetKeyResultValue(ctx, sqlc.SetKeyResultValueParams{ID: id, CurrentValue: v})
}

func (s *Store) InsertCheckin(ctx context.Context, c domain.Checkin) error {
	return s.q.InsertCheckin(ctx, sqlc.InsertCheckinParams{ID: c.ID, KeyResultID: c.KeyResultID, Value: c.Value, Confidence: int32(c.Confidence), Note: c.Note, CreatedBy: c.CreatedBy})
}

func (s *Store) ListCheckins(ctx context.Context, keyResultID string) ([]domain.Checkin, error) {
	rows, err := s.q.ListCheckins(ctx, keyResultID)
	return each(rows, err, func(c sqlc.PerformanceOkrCheckin) domain.Checkin {
		return domain.Checkin{ID: c.ID, KeyResultID: c.KeyResultID, Note: c.Note, Value: c.Value, Confidence: int(c.Confidence), CreatedBy: c.CreatedBy, CreatedAt: c.CreatedAt}
	})
}

func (s *Store) LatestConfidence(ctx context.Context, period string) (map[string]int, error) {
	rows, err := s.q.LatestConfidence(ctx, period)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.KeyResultID] = int(r.Confidence)
	}
	return out, nil
}
