package postgres

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
)

// ─── assets ──────────────────────────────────────────────────

func toAsset(a sqlc.HrAsset, name, no string) domain.Asset {
	return domain.Asset{ID: a.ID, AssetCode: a.AssetCode, Name: a.Name, Category: a.Category, SerialNo: a.SerialNo, Status: a.Status, Notes: a.Notes,
		PurchaseDate: dayPtr(a.PurchaseDate), PurchaseValue: num(a.PurchaseValue), EmployeeID: a.EmployeeID, CreatedBy: a.CreatedBy, EmployeeName: name, EmployeeNo: no}
}

func (s *Store) ListAssets(ctx context.Context, f domain.AssetFilter) ([]domain.Asset, error) {
	rows, err := s.q.ListAssets(ctx, sqlc.ListAssetsParams{EmployeeID: f.EmployeeID, Status: f.Status, Search: f.Search})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Asset, len(rows))
	for i, r := range rows {
		out[i] = toAsset(r.HrAsset, r.EmployeeName, r.EmployeeNo)
	}
	return out, nil
}

func (s *Store) GetAsset(ctx context.Context, id string) (*domain.Asset, error) {
	r, err := one(s.q.GetAsset(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	a := toAsset(r.HrAsset, r.EmployeeName, r.EmployeeNo)
	return &a, nil
}

func (s *Store) SaveAsset(ctx context.Context, a domain.Asset) error {
	return wrap(s.q.UpsertAsset(ctx, sqlc.UpsertAssetParams{ID: a.ID, AssetCode: a.AssetCode, Name: a.Name, Category: a.Category, SerialNo: a.SerialNo,
		PurchaseDate: parseDayPtr(a.PurchaseDate), PurchaseValue: money(a.PurchaseValue), Status: a.Status, EmployeeID: a.EmployeeID, Notes: a.Notes, CreatedBy: a.CreatedBy}))
}

func (s *Store) InsertAssetAssignment(ctx context.Context, x domain.AssetAssignment) error {
	return s.q.InsertAssetAssignment(ctx, sqlc.InsertAssetAssignmentParams{ID: x.ID, AssetID: x.AssetID, EmployeeID: x.EmployeeID,
		AssignedOn: parseDay(x.AssignedOn), ConditionOut: x.ConditionOut, CreatedBy: x.CreatedBy})
}

func (s *Store) CloseAssetAssignment(ctx context.Context, assetID, returnedOn, condition string) (bool, error) {
	on := parseDay(returnedOn)
	n, err := s.q.CloseAssetAssignment(ctx, sqlc.CloseAssetAssignmentParams{AssetID: assetID, ReturnedOn: &on, ConditionIn: condition})
	return n > 0, err
}

func (s *Store) ListAssetAssignments(ctx context.Context, assetID string) ([]domain.AssetAssignment, error) {
	rows, err := s.q.ListAssetAssignments(ctx, assetID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AssetAssignment, len(rows))
	for i, r := range rows {
		x := r.HrAssetAssignment
		out[i] = domain.AssetAssignment{ID: x.ID, AssetID: x.AssetID, EmployeeID: x.EmployeeID, AssignedOn: day(x.AssignedOn), ReturnedOn: dayPtr(x.ReturnedOn),
			ConditionOut: x.ConditionOut, ConditionIn: x.ConditionIn, CreatedBy: x.CreatedBy, EmployeeName: r.FullName, EmployeeNo: r.EmployeeNo}
	}
	return out, nil
}

// ─── training ────────────────────────────────────────────────

func toTraining(t sqlc.HrTraining, count int) domain.Training {
	return domain.Training{ID: t.ID, Title: t.Title, Provider: t.Provider, StartDate: day(t.StartDate), EndDate: day(t.EndDate), Location: t.Location,
		Status: t.Status, Notes: t.Notes, Cost: num(t.Cost), CreatedBy: t.CreatedBy, ParticipantCount: count}
}

func (s *Store) ListTrainings(ctx context.Context, f domain.TrainingFilter) ([]domain.Training, error) {
	rows, err := s.q.ListTrainings(ctx, sqlc.ListTrainingsParams{Status: f.Status, Year: i32(f.Year), EmployeeID: f.EmployeeID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Training, len(rows))
	for i, r := range rows {
		out[i] = toTraining(r.HrTraining, int(r.ParticipantCount))
	}
	return out, nil
}

func (s *Store) GetTraining(ctx context.Context, id string) (*domain.Training, error) {
	r, err := one(s.q.GetTraining(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	t := toTraining(*r, 0)
	return &t, nil
}

func (s *Store) SaveTraining(ctx context.Context, t domain.Training) error {
	return s.q.UpsertTraining(ctx, sqlc.UpsertTrainingParams{ID: t.ID, Title: t.Title, Provider: t.Provider, StartDate: parseDay(t.StartDate), EndDate: parseDay(t.EndDate),
		Location: t.Location, Cost: money(t.Cost), Status: t.Status, Notes: t.Notes, CreatedBy: t.CreatedBy})
}

func (s *Store) DeleteTraining(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteTraining(ctx, id)
	return n > 0, err
}

func (s *Store) ListTrainingParticipants(ctx context.Context, trainingID string) ([]domain.TrainingParticipant, error) {
	rows, err := s.q.ListTrainingParticipants(ctx, trainingID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TrainingParticipant, len(rows))
	for i, r := range rows {
		p := r.HrTrainingParticipant
		out[i] = domain.TrainingParticipant{TrainingID: p.TrainingID, EmployeeID: p.EmployeeID, Status: p.Status, Certificate: p.Certificate, Score: numPtr(p.Score),
			EmployeeName: r.FullName, EmployeeNo: r.EmployeeNo, DepartmentName: r.DepartmentName}
	}
	return out, nil
}

func (s *Store) SetTrainingParticipants(ctx context.Context, trainingID string, employeeIDs []string) error {
	if err := s.q.DeleteTrainingParticipantsExcept(ctx, sqlc.DeleteTrainingParticipantsExceptParams{TrainingID: trainingID, Keep: employeeIDs}); err != nil {
		return err
	}
	for _, id := range employeeIDs {
		if err := s.q.AddTrainingParticipant(ctx, sqlc.AddTrainingParticipantParams{TrainingID: trainingID, EmployeeID: id}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) UpdateTrainingParticipant(ctx context.Context, p domain.TrainingParticipant) (bool, error) {
	n, err := s.q.UpdateTrainingParticipant(ctx, sqlc.UpdateTrainingParticipantParams{TrainingID: p.TrainingID, EmployeeID: p.EmployeeID, Status: p.Status,
		Score: moneyPtr(p.Score), Certificate: p.Certificate})
	return n > 0, err
}

// ─── timesheet ───────────────────────────────────────────────

func toTimesheet(t sqlc.HrTimesheetEntry, name, no string) domain.TimesheetEntry {
	return domain.TimesheetEntry{ID: t.ID, EmployeeID: t.EmployeeID, WorkDate: day(t.WorkDate), ProjectID: t.ProjectID, ProjectName: t.ProjectName,
		Description: t.Description, Status: t.Status, RejectReason: t.RejectReason, Hours: num(t.Hours), DecidedBy: t.DecidedBy, EmployeeName: name, EmployeeNo: no}
}

func (s *Store) ListTimesheet(ctx context.Context, f domain.TimesheetFilter) ([]domain.TimesheetEntry, error) {
	rows, err := s.q.ListTimesheet(ctx, sqlc.ListTimesheetParams{EmployeeID: f.EmployeeID, Status: f.Status, FromDate: parseDay(f.From), ToDate: parseDay(f.To)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.TimesheetEntry, len(rows))
	for i, r := range rows {
		out[i] = toTimesheet(r.HrTimesheetEntry, r.FullName, r.EmployeeNo)
	}
	return out, nil
}

func (s *Store) GetTimesheetEntry(ctx context.Context, id string) (*domain.TimesheetEntry, error) {
	r, err := one(s.q.GetTimesheetEntry(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	e := toTimesheet(r.HrTimesheetEntry, r.FullName, r.EmployeeNo)
	return &e, nil
}

func (s *Store) SumTimesheetDay(ctx context.Context, employeeID, d, exceptID string) (float64, error) {
	v, err := s.q.SumTimesheetDay(ctx, sqlc.SumTimesheetDayParams{EmployeeID: employeeID, WorkDate: parseDay(d), ID: exceptID})
	return num(v), err
}

func (s *Store) SaveTimesheetEntry(ctx context.Context, e domain.TimesheetEntry) error {
	return s.q.UpsertTimesheetEntry(ctx, sqlc.UpsertTimesheetEntryParams{ID: e.ID, EmployeeID: e.EmployeeID, WorkDate: parseDay(e.WorkDate), ProjectID: e.ProjectID,
		ProjectName: e.ProjectName, Hours: money(e.Hours), Description: e.Description})
}

func (s *Store) DeleteTimesheetEntry(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteTimesheetEntry(ctx, id)
	return n > 0, err
}

func (s *Store) SubmitTimesheet(ctx context.Context, employeeID, from, to string) (int, error) {
	n, err := s.q.SubmitTimesheet(ctx, sqlc.SubmitTimesheetParams{EmployeeID: employeeID, FromDate: parseDay(from), ToDate: parseDay(to)})
	return int(n), err
}

func (s *Store) DecideTimesheetEntry(ctx context.Context, id, status, reason string, by *string) (bool, error) {
	n, err := s.q.DecideTimesheetEntry(ctx, sqlc.DecideTimesheetEntryParams{ID: id, Status: status, RejectReason: reason, DecidedBy: by})
	return n > 0, err
}

func (s *Store) ApprovedProjectHours(ctx context.Context, from, to string) ([]domain.ProjectHours, error) {
	rows, err := s.q.TimesheetHours(ctx, sqlc.TimesheetHoursParams{FromDate: parseDay(from), ToDate: parseDay(to)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.ProjectHours, len(rows))
	for i, r := range rows {
		out[i] = domain.ProjectHours{EmployeeID: r.EmployeeID, ProjectID: r.ProjectID, ProjectName: r.ProjectName, Hours: num(r.Hours)}
	}
	return out, nil
}
