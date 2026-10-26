// Package postgres implements the hr ports on top of sqlc queries.
package postgres

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

type Store struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: sqlc.New(pool)}
}

func (s *Store) WithTx(ctx context.Context, fn func(tx domain.Store) error) error {
	return pg.Tx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&Store{pool: s.pool, q: s.q.WithTx(tx)})
	})
}

// wrap turns a unique violation into *domain.ErrUnique.
func wrap(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ErrUnique{Constraint: pgErr.ConstraintName}
	}
	return err
}

// one maps pgx.ErrNoRows to a nil result.
func one[T any](v T, err error) (*T, error) {
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ─── conversions ─────────────────────────────────────────────

func day(t time.Time) string { return t.Format("2006-01-02") }

func dayPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := day(*t)
	return &s
}

// parseDay reads "YYYY-MM-DD"; a malformed value is the zero time (the app
// validates dates before they reach the store).
func parseDay(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func parseDayPtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t := parseDay(*s)
	return &t
}

func num(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func numPtr(s *string) *float64 {
	if s == nil {
		return nil
	}
	f := num(*s)
	return &f
}

// instantPtr is nil for the zero time, so the column default applies.
func instantPtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func money(f float64) string { return domain.Money(f) }

func moneyPtr(f *float64) *string {
	if f == nil {
		return nil
	}
	s := money(*f)
	return &s
}

func i32(n int) int32 { return int32(n) }

func intPtr(n *int32) *int {
	if n == nil {
		return nil
	}
	v := int(*n)
	return &v
}

func i32Ptr(n *int) *int32 {
	if n == nil {
		return nil
	}
	v := int32(*n)
	return &v
}

// ─── Departments, positions, statuses ────────────────────────

func toDepartment(d sqlc.HrDepartment) domain.Department {
	return domain.Department{ID: d.ID, Code: d.Code, Name: d.Name, ParentID: d.ParentID, HeadEmployeeID: d.HeadEmployeeID, IsActive: d.IsActive}
}

func (s *Store) ListDepartments(ctx context.Context, includeInactive bool) ([]domain.Department, error) {
	rows, err := s.q.ListDepartments(ctx, includeInactive)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Department, len(rows))
	for i, r := range rows {
		out[i] = toDepartment(r)
	}
	return out, nil
}

func (s *Store) GetDepartment(ctx context.Context, id string) (*domain.Department, error) {
	r, err := one(s.q.GetDepartment(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	d := toDepartment(*r)
	return &d, nil
}

func (s *Store) InsertDepartment(ctx context.Context, d domain.Department, actor *string) error {
	_, err := s.q.InsertDepartment(ctx, sqlc.InsertDepartmentParams{ID: d.ID, Code: d.Code, Name: d.Name, ParentID: d.ParentID, HeadEmployeeID: d.HeadEmployeeID, IsActive: d.IsActive, CreatedBy: actor})
	return wrap(err)
}

func (s *Store) UpdateDepartment(ctx context.Context, d domain.Department, actor *string) (bool, error) {
	_, err := s.q.UpdateDepartment(ctx, sqlc.UpdateDepartmentParams{ID: d.ID, Code: d.Code, Name: d.Name, ParentID: d.ParentID, HeadEmployeeID: d.HeadEmployeeID, IsActive: d.IsActive, UpdatedBy: actor})
	if pg.IsNoRows(err) {
		return false, nil
	}
	return err == nil, wrap(err)
}

func (s *Store) DeleteDepartment(ctx context.Context, id string, actor *string) (bool, error) {
	n, err := s.q.SoftDeleteDepartment(ctx, sqlc.SoftDeleteDepartmentParams{ID: id, UpdatedBy: actor})
	return n > 0, err
}

func (s *Store) CountDepartmentEmployees(ctx context.Context, id string) (int, error) {
	n, err := s.q.CountDepartmentEmployees(ctx, &id)
	return int(n), err
}

func toPosition(p sqlc.HrPosition, dept string) domain.Position {
	out := domain.Position{ID: p.ID, Title: p.Title, DepartmentID: p.DepartmentID, DepartmentName: dept, Level: p.Level, IsActive: p.IsActive}
	if p.SalaryMin != nil {
		out.SalaryMin = num(*p.SalaryMin)
	}
	if p.SalaryMax != nil {
		out.SalaryMax = num(*p.SalaryMax)
	}
	return out
}

func (s *Store) ListPositions(ctx context.Context, departmentID string, includeInactive bool) ([]domain.Position, error) {
	rows, err := s.q.ListPositions(ctx, sqlc.ListPositionsParams{IncludeInactive: includeInactive, DepartmentID: departmentID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Position, len(rows))
	for i, r := range rows {
		out[i] = toPosition(r.HrPosition, r.DepartmentName)
	}
	return out, nil
}

func (s *Store) GetPosition(ctx context.Context, id string) (*domain.Position, error) {
	r, err := one(s.q.GetPosition(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	p := toPosition(r.HrPosition, r.DepartmentName)
	return &p, nil
}

func salaryBound(v float64) *string {
	if v <= 0 {
		return nil
	}
	return moneyPtr(&v)
}

func (s *Store) InsertPosition(ctx context.Context, p domain.Position, actor *string) error {
	return wrap(s.q.InsertPosition(ctx, sqlc.InsertPositionParams{ID: p.ID, Title: p.Title, DepartmentID: p.DepartmentID, Level: p.Level,
		SalaryMin: salaryBound(p.SalaryMin), SalaryMax: salaryBound(p.SalaryMax), IsActive: p.IsActive, CreatedBy: actor}))
}

func (s *Store) UpdatePosition(ctx context.Context, p domain.Position, actor *string) (bool, error) {
	n, err := s.q.UpdatePosition(ctx, sqlc.UpdatePositionParams{ID: p.ID, Title: p.Title, DepartmentID: p.DepartmentID, Level: p.Level,
		SalaryMin: salaryBound(p.SalaryMin), SalaryMax: salaryBound(p.SalaryMax), IsActive: p.IsActive, UpdatedBy: actor})
	return n > 0, wrap(err)
}

func (s *Store) DeletePosition(ctx context.Context, id string, actor *string) (bool, error) {
	n, err := s.q.SoftDeletePosition(ctx, sqlc.SoftDeletePositionParams{ID: id, UpdatedBy: actor})
	return n > 0, err
}

func (s *Store) ListEmploymentStatuses(ctx context.Context) ([]domain.EmploymentStatusOption, error) {
	rows, err := s.q.ListEmploymentStatuses(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.EmploymentStatusOption, len(rows))
	for i, r := range rows {
		out[i] = domain.EmploymentStatusOption{Code: r.Code, Name: r.Name, IsActive: r.IsActive, SortOrder: int(r.SortOrder)}
	}
	return out, nil
}

func (s *Store) UpsertEmploymentStatus(ctx context.Context, o domain.EmploymentStatusOption) error {
	_, err := s.q.UpsertEmploymentStatus(ctx, sqlc.UpsertEmploymentStatusParams{Code: o.Code, Name: o.Name, IsActive: o.IsActive, SortOrder: i32(o.SortOrder)})
	return err
}

func (s *Store) DeleteEmploymentStatus(ctx context.Context, code string) (bool, error) {
	n, err := s.q.DeleteEmploymentStatus(ctx, code)
	return n > 0, err
}

// ─── Employees ───────────────────────────────────────────────

func toEmployee(e sqlc.HrEmployee, dept, pos, mgr, status string, mgrUser *string) domain.Employee {
	return domain.Employee{ManagerUserID: mgrUser,
		ID: e.ID, UserID: e.UserID, EmployeeNo: e.EmployeeNo, FullName: e.FullName, Email: e.Email, Phone: e.Phone,
		NIK: e.Nik, NPWP: e.Npwp, BpjsTkNo: e.BpjsTkNo, BpjsKesNo: e.BpjsKesNo, BirthPlace: e.BirthPlace,
		BirthDate: dayPtr(e.BirthDate), Gender: e.Gender, MaritalStatus: e.MaritalStatus, Religion: e.Religion,
		Address: e.Address, DepartmentID: e.DepartmentID, DepartmentName: dept, PositionID: e.PositionID,
		PositionName: pos, ManagerID: e.ManagerID, ManagerName: mgr, EmploymentStatus: e.EmploymentStatus,
		EmploymentStatusName: status, IsActive: e.IsActive, JoinDate: day(e.JoinDate), EndDate: dayPtr(e.EndDate),
		ProbationEndDate: dayPtr(e.ProbationEndDate), BankName: e.BankName, BankAccountNo: e.BankAccountNo,
		BankAccountName: e.BankAccountName, EmergencyName: e.EmergencyContactName, EmergencyPhone: e.EmergencyContactPhone,
		EmergencyRelation: e.EmergencyContactRelation, CreatedBy: e.CreatedBy, UpdatedBy: e.UpdatedBy,
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	}
}

func (s *Store) ListEmployees(ctx context.Context, f domain.EmployeeFilter) ([]domain.Employee, int, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 1000
	}
	rows, err := s.q.ListEmployees(ctx, sqlc.ListEmployeesParams{IncludeInactive: f.IncludeInactive, DepartmentID: f.DepartmentID,
		ManagerID: f.ManagerID, Ids: nonNil(f.IDs), Search: f.Search, PositionID: f.PositionID, Level: f.Level, Gender: f.Gender,
		EmploymentStatus: f.EmploymentStatus, RowLimit: i32(limit), RowOffset: i32(f.Offset)})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Employee, len(rows))
	total := 0
	for i, r := range rows {
		out[i] = toEmployee(r.HrEmployee, r.DepartmentName, r.PositionName, r.ManagerName, r.EmploymentStatusName, r.ManagerUserID)
		total = int(r.Total)
	}
	return out, total, nil
}

func (s *Store) GetEmployee(ctx context.Context, id string) (*domain.Employee, error) {
	r, err := one(s.q.GetEmployee(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	e := toEmployee(r.HrEmployee, r.DepartmentName, r.PositionName, r.ManagerName, r.EmploymentStatusName, r.ManagerUserID)
	return &e, nil
}

func (s *Store) EmployeeIDByUser(ctx context.Context, userID string) (string, error) {
	id, err := s.q.GetEmployeeIDByUser(ctx, &userID)
	if pg.IsNoRows(err) {
		return "", nil
	}
	return id, err
}

func (s *Store) ListEmployeeNos(ctx context.Context, prefix string) ([]string, error) {
	return s.q.ListEmployeeNos(ctx, prefix)
}

func (s *Store) InsertEmployee(ctx context.Context, e domain.Employee) error {
	return wrap(s.q.InsertEmployee(ctx, sqlc.InsertEmployeeParams{
		ID: e.ID, UserID: e.UserID, EmployeeNo: e.EmployeeNo, FullName: e.FullName, Email: e.Email, Phone: e.Phone,
		Nik: e.NIK, Npwp: e.NPWP, BpjsTkNo: e.BpjsTkNo, BpjsKesNo: e.BpjsKesNo, BirthPlace: e.BirthPlace,
		BirthDate: parseDayPtr(e.BirthDate), Gender: e.Gender, MaritalStatus: e.MaritalStatus, Religion: e.Religion,
		Address: e.Address, DepartmentID: e.DepartmentID, PositionID: e.PositionID, ManagerID: e.ManagerID,
		EmploymentStatus: e.EmploymentStatus, IsActive: e.IsActive, JoinDate: parseDay(e.JoinDate),
		EndDate: parseDayPtr(e.EndDate), ProbationEndDate: parseDayPtr(e.ProbationEndDate), BankName: e.BankName,
		BankAccountNo: e.BankAccountNo, BankAccountName: e.BankAccountName, EmergencyContactName: e.EmergencyName,
		EmergencyContactPhone: e.EmergencyPhone, EmergencyContactRelation: e.EmergencyRelation, CreatedBy: e.CreatedBy,
	}))
}

func (s *Store) UpdateEmployee(ctx context.Context, e domain.Employee) (bool, error) {
	n, err := s.q.UpdateEmployee(ctx, sqlc.UpdateEmployeeParams{
		ID: e.ID, EmployeeNo: e.EmployeeNo, FullName: e.FullName, Email: e.Email, Phone: e.Phone,
		Nik: e.NIK, Npwp: e.NPWP, BpjsTkNo: e.BpjsTkNo, BpjsKesNo: e.BpjsKesNo, BirthPlace: e.BirthPlace,
		BirthDate: parseDayPtr(e.BirthDate), Gender: e.Gender, MaritalStatus: e.MaritalStatus, Religion: e.Religion,
		Address: e.Address, DepartmentID: e.DepartmentID, PositionID: e.PositionID, ManagerID: e.ManagerID,
		EmploymentStatus: e.EmploymentStatus, IsActive: e.IsActive, JoinDate: parseDay(e.JoinDate),
		EndDate: parseDayPtr(e.EndDate), ProbationEndDate: parseDayPtr(e.ProbationEndDate), BankName: e.BankName,
		BankAccountNo: e.BankAccountNo, BankAccountName: e.BankAccountName, EmergencyContactName: e.EmergencyName,
		EmergencyContactPhone: e.EmergencyPhone, EmergencyContactRelation: e.EmergencyRelation, UpdatedBy: e.UpdatedBy,
	})
	return n > 0, wrap(err)
}

func (s *Store) SetEmployeeStatus(ctx context.Context, id, status string, active bool, endDate *string, actor *string) error {
	return s.q.SetEmployeeStatus(ctx, sqlc.SetEmployeeStatusParams{ID: id, EmploymentStatus: status, IsActive: active, EndDate: parseDayPtr(endDate), UpdatedBy: actor})
}

func (s *Store) DeleteEmployee(ctx context.Context, id string, actor *string) (bool, error) {
	n, err := s.q.SoftDeleteEmployee(ctx, sqlc.SoftDeleteEmployeeParams{ID: id, UpdatedBy: actor})
	return n > 0, err
}

func (s *Store) SetEmployeeUser(ctx context.Context, id string, userID, actor *string) (bool, error) {
	n, err := s.q.SetEmployeeUser(ctx, sqlc.SetEmployeeUserParams{ID: id, UserID: userID, UpdatedBy: actor})
	return n > 0, wrap(err)
}

func (s *Store) UnlinkUser(ctx context.Context, userID string) (int, error) {
	n, err := s.q.UnlinkUser(ctx, &userID)
	return int(n), err
}

func (s *Store) LinkedUserIDsInDepartment(ctx context.Context, code string) ([]string, error) {
	return s.q.ListLinkedUserIDsInDepartment(ctx, code)
}

func (s *Store) ClaimRecruitmentHire(ctx context.Context, candidateID, employeeID string) (bool, error) {
	n, err := s.q.InsertRecruitmentHire(ctx, sqlc.InsertRecruitmentHireParams{CandidateID: candidateID, EmployeeID: employeeID})
	return n == 1, err
}

func (s *Store) InsertHistory(ctx context.Context, h domain.HistoryEntry) error {
	return s.q.InsertHistory(ctx, sqlc.InsertHistoryParams{ID: h.ID, EmployeeID: h.EmployeeID, ChangeType: h.ChangeType, Notes: h.Notes,
		PrevEmploymentStatus: h.PrevStatus, NewEmploymentStatus: h.NewStatus, PrevDepartmentID: h.PrevDepartmentID,
		NewDepartmentID: h.NewDepartmentID, PrevPositionID: h.PrevPositionID, NewPositionID: h.NewPositionID, CreatedBy: h.CreatedBy,
		CreatedAt: instantPtr(h.CreatedAt)})
}

func (s *Store) ListHistory(ctx context.Context, employeeID string) ([]domain.HistoryEntry, error) {
	rows, err := s.q.ListHistory(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.HistoryEntry, len(rows))
	for i, r := range rows {
		h := r.HrEmploymentHistory
		out[i] = domain.HistoryEntry{ID: h.ID, EmployeeID: h.EmployeeID, ChangeType: h.ChangeType, Notes: h.Notes,
			PrevStatus: h.PrevEmploymentStatus, NewStatus: h.NewEmploymentStatus, PrevDepartmentID: h.PrevDepartmentID,
			NewDepartmentID: h.NewDepartmentID, PrevPositionID: h.PrevPositionID, NewPositionID: h.NewPositionID,
			PrevDepartment: r.PrevDepartment, NewDepartment: r.NewDepartment, PrevPosition: r.PrevPosition,
			NewPosition: r.NewPosition, CreatedBy: h.CreatedBy, CreatedAt: h.CreatedAt}
	}
	return out, nil
}

func toDocument(d sqlc.HrEmployeeDocument) domain.Document {
	return domain.Document{ID: d.ID, EmployeeID: d.EmployeeID, DocType: d.DocType, Name: d.Name, StorageKey: d.StorageKey,
		ContentType: d.ContentType, SizeBytes: d.SizeBytes, CreatedBy: d.CreatedBy, CreatedAt: d.CreatedAt}
}

func (s *Store) InsertDocument(ctx context.Context, d domain.Document) error {
	return s.q.InsertDocument(ctx, sqlc.InsertDocumentParams{ID: d.ID, EmployeeID: d.EmployeeID, DocType: d.DocType, Name: d.Name,
		StorageKey: d.StorageKey, ContentType: d.ContentType, SizeBytes: d.SizeBytes, CreatedBy: d.CreatedBy})
}

func (s *Store) ListDocuments(ctx context.Context, employeeID string) ([]domain.Document, error) {
	rows, err := s.q.ListDocuments(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Document, len(rows))
	for i, r := range rows {
		out[i] = toDocument(r)
	}
	return out, nil
}

func (s *Store) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
	r, err := one(s.q.GetDocument(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	d := toDocument(*r)
	return &d, nil
}

func (s *Store) DeleteDocument(ctx context.Context, id string) (bool, error) {
	n, err := s.q.SoftDeleteDocument(ctx, id)
	return n > 0, err
}

// ClaimReminder records a reminder of kind for ref on day; false when it
// was already sent.
func (s *Store) ClaimReminder(ctx context.Context, kind, ref, d string) (bool, error) {
	n, err := s.q.ClaimReminder(ctx, sqlc.ClaimReminderParams{Kind: kind, Ref: ref, Day: parseDay(d)})
	return n > 0, err
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
