package postgres

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── Contracts ───────────────────────────────────────────────

func toContract(c sqlc.HrEmploymentContract, name, no, dept string) domain.Contract {
	return domain.Contract{ID: c.ID, ContractNumber: c.ContractNumber, EmployeeID: c.EmployeeID, EmployeeName: name, EmployeeNo: no,
		DepartmentNameNow: dept, ContractType: c.ContractType, Status: c.Status, StartDate: day(c.StartDate), EndDate: dayPtr(c.EndDate),
		ProbationEndDate: dayPtr(c.ProbationEndDate), ParentContractID: c.ParentContractID, Sequence: int(c.Sequence),
		PositionTitle: c.PositionTitle, DepartmentName: c.DepartmentName, WorkLocation: c.WorkLocation, BaseSalary: numPtr(c.BaseSalary),
		SignedAt: dayPtr(c.SignedAt), CompensationAmount: numPtr(c.CompensationAmount), TerminatedReason: c.TerminatedReason,
		Notes: c.Notes, CreatedBy: c.CreatedBy, CreatedAt: c.CreatedAt}
}

func (s *Store) ListContracts(ctx context.Context, f domain.ContractFilter) ([]domain.Contract, error) {
	rows, err := s.q.ListContracts(ctx, sqlc.ListContractsParams{EmployeeID: f.EmployeeID, Status: f.Status, ContractType: f.ContractType, Search: f.Search})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Contract, len(rows))
	for i, r := range rows {
		out[i] = toContract(r.HrEmploymentContract, r.FullName, r.EmployeeNo, r.EmployeeDepartment)
	}
	return out, nil
}

func (s *Store) GetContract(ctx context.Context, id string) (*domain.Contract, error) {
	r, err := one(s.q.GetContract(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	c := toContract(r.HrEmploymentContract, r.FullName, r.EmployeeNo, r.EmployeeDepartment)
	return &c, nil
}

func (s *Store) CountContractsOfType(ctx context.Context, contractType string, year int) (int, error) {
	n, err := s.q.CountContractsOfType(ctx, sqlc.CountContractsOfTypeParams{ContractType: contractType, Year: i32(year)})
	return int(n), err
}

func (s *Store) InsertContract(ctx context.Context, c domain.Contract) error {
	return wrap(s.q.InsertContract(ctx, sqlc.InsertContractParams{ID: c.ID, ContractNumber: c.ContractNumber, EmployeeID: c.EmployeeID,
		ContractType: c.ContractType, StartDate: parseDay(c.StartDate), EndDate: parseDayPtr(c.EndDate),
		ProbationEndDate: parseDayPtr(c.ProbationEndDate), ParentContractID: c.ParentContractID, Sequence: i32(c.Sequence),
		PositionTitle: c.PositionTitle, DepartmentName: c.DepartmentName, WorkLocation: c.WorkLocation,
		BaseSalary: moneyPtr(c.BaseSalary), Notes: c.Notes, CreatedBy: c.CreatedBy}))
}

func (s *Store) UpdateDraftContract(ctx context.Context, c domain.Contract, actor *string) (bool, error) {
	n, err := s.q.UpdateDraftContract(ctx, sqlc.UpdateDraftContractParams{ID: c.ID, ContractType: c.ContractType,
		StartDate: parseDay(c.StartDate), EndDate: parseDayPtr(c.EndDate), ProbationEndDate: parseDayPtr(c.ProbationEndDate),
		PositionTitle: c.PositionTitle, DepartmentName: c.DepartmentName, WorkLocation: c.WorkLocation,
		BaseSalary: moneyPtr(c.BaseSalary), Notes: c.Notes, UpdatedBy: actor})
	return n > 0, err
}

func (s *Store) SetContractStatus(ctx context.Context, u domain.ContractStatusUpdate) (bool, error) {
	n, err := s.q.SetContractStatus(ctx, sqlc.SetContractStatusParams{ID: u.ID, FromStatus: u.From, Status: u.To,
		EndDate: parseDayPtr(u.EndDate), SignedAt: parseDayPtr(u.SignedAt), CompensationAmount: moneyPtr(u.Compensation),
		TerminatedReason: u.TerminatedReason, Actor: u.Actor})
	return n > 0, wrap(err)
}

func (s *Store) ActiveContractNumber(ctx context.Context, employeeID, exceptID string) (string, error) {
	n, err := s.q.ActiveContractNumber(ctx, sqlc.ActiveContractNumberParams{EmployeeID: employeeID, ID: exceptID})
	if pg.IsNoRows(err) {
		return "", nil
	}
	return n, err
}

func (s *Store) PkwtChain(ctx context.Context, employeeID string) ([][2]*string, error) {
	rows, err := s.q.PkwtChain(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	out := make([][2]*string, len(rows))
	for i, r := range rows {
		start := day(r.StartDate)
		out[i] = [2]*string{&start, dayPtr(r.EndDate)}
	}
	return out, nil
}

func (s *Store) DeleteDraftContract(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteDraftContract(ctx, id)
	return n > 0, err
}

func (s *Store) EmployeesWithoutContract(ctx context.Context) ([]domain.Employee, error) {
	rows, err := s.q.ActiveEmployeesWithoutContract(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Employee, len(rows))
	for i, r := range rows {
		out[i] = domain.Employee{ID: r.ID, FullName: r.FullName, EmployeeNo: r.EmployeeNo, DepartmentName: r.DepartmentName}
	}
	return out, nil
}

func (s *Store) ProbationsEnding(ctx context.Context, from, to string) ([]domain.Employee, error) {
	rows, err := s.q.ProbationsEnding(ctx, sqlc.ProbationsEndingParams{DateFrom: parseDayPtr(&from), DateTo: parseDayPtr(&to)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Employee, len(rows))
	for i, r := range rows {
		out[i] = domain.Employee{ID: r.ID, FullName: r.FullName, EmployeeNo: r.EmployeeNo, DepartmentName: r.DepartmentName, ProbationEndDate: dayPtr(r.ProbationEndDate)}
	}
	return out, nil
}

func (s *Store) ActiveBirthdays(ctx context.Context) ([]domain.Employee, error) {
	rows, err := s.q.ActiveBirthdays(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Employee, len(rows))
	for i, r := range rows {
		out[i] = domain.Employee{ID: r.ID, FullName: r.FullName, BirthDate: dayPtr(r.BirthDate)}
	}
	return out, nil
}

// ─── Announcements ───────────────────────────────────────────

func toAnnouncement(a sqlc.HrAnnouncement) domain.Announcement {
	return domain.Announcement{ID: a.ID, Title: a.Title, Body: a.Body, Tags: a.Tags, Status: a.Status, IsPinned: a.IsPinned,
		TargetScope: a.TargetScope, PublishAt: a.PublishAt, ExpiresAt: a.ExpiresAt, CreatedBy: a.CreatedBy, CreatedAt: a.CreatedAt}
}

func (s *Store) ListAnnouncements(ctx context.Context, status string) ([]domain.Announcement, error) {
	rows, err := s.q.ListAnnouncements(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Announcement, len(rows))
	for i, r := range rows {
		a := toAnnouncement(r.HrAnnouncement)
		a.DepartmentIDs, a.ReadCount = r.DepartmentIds, int(r.ReadCount)
		out[i] = a
	}
	return out, nil
}

func (s *Store) GetAnnouncement(ctx context.Context, id string) (*domain.Announcement, error) {
	r, err := one(s.q.GetAnnouncement(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	a := toAnnouncement(r.HrAnnouncement)
	a.DepartmentIDs, a.ReadCount = r.DepartmentIds, int(r.ReadCount)
	return &a, nil
}

func (s *Store) AnnouncementFeed(ctx context.Context, employeeID string, now time.Time) ([]domain.Announcement, error) {
	rows, err := s.q.AnnouncementFeed(ctx, sqlc.AnnouncementFeedParams{EmployeeID: employeeID, Now: now})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Announcement, len(rows))
	for i, r := range rows {
		a := toAnnouncement(r.HrAnnouncement)
		a.IsRead = r.IsRead
		out[i] = a
	}
	return out, nil
}

// SaveAnnouncement inserts (insert=true) or updates it with its departments.
func (s *Store) SaveAnnouncement(ctx context.Context, a domain.Announcement, insert bool, actor *string) (bool, error) {
	if insert {
		if err := s.q.InsertAnnouncement(ctx, sqlc.InsertAnnouncementParams{ID: a.ID, Title: a.Title, Body: a.Body, Tags: nonNil(a.Tags),
			Status: a.Status, IsPinned: a.IsPinned, TargetScope: a.TargetScope, PublishAt: a.PublishAt, ExpiresAt: a.ExpiresAt, CreatedBy: actor}); err != nil {
			return false, err
		}
	} else {
		n, err := s.q.UpdateAnnouncement(ctx, sqlc.UpdateAnnouncementParams{ID: a.ID, Title: a.Title, Body: a.Body, Tags: nonNil(a.Tags),
			Status: a.Status, IsPinned: a.IsPinned, TargetScope: a.TargetScope, PublishAt: a.PublishAt, ExpiresAt: a.ExpiresAt, UpdatedBy: actor})
		if err != nil || n == 0 {
			return false, err
		}
	}
	if err := s.q.ClearAnnouncementDepartments(ctx, a.ID); err != nil {
		return false, err
	}
	for _, d := range a.DepartmentIDs {
		if err := s.q.AddAnnouncementDepartment(ctx, sqlc.AddAnnouncementDepartmentParams{AnnouncementID: a.ID, DepartmentID: d}); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Store) DeleteAnnouncement(ctx context.Context, id string) (bool, error) {
	n, err := s.q.SoftDeleteAnnouncement(ctx, id)
	return n > 0, err
}

func (s *Store) MarkAnnouncementRead(ctx context.Context, id, employeeID string) error {
	return s.q.MarkAnnouncementRead(ctx, sqlc.MarkAnnouncementReadParams{AnnouncementID: id, EmployeeID: employeeID})
}

// ─── Onboarding / offboarding ────────────────────────────────

func toTask(t sqlc.HrOnboardingTask, by string) domain.OnboardingTask {
	return domain.OnboardingTask{ID: t.ID, EmployeeID: t.EmployeeID, TaskName: t.TaskName, Category: t.Category, Priority: int(t.Priority),
		DueDate: dayPtr(t.DueDate), IsCompleted: t.IsCompleted, CompletedAt: t.CompletedAt, CompletedBy: t.CompletedBy,
		CompletedByName: by, Notes: t.Notes}
}

func (s *Store) ListOnboardingTasks(ctx context.Context, employeeID string) ([]domain.OnboardingTask, error) {
	rows, err := s.q.ListOnboardingTasks(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.OnboardingTask, len(rows))
	for i, r := range rows {
		out[i] = toTask(r.HrOnboardingTask, r.CompletedByName)
	}
	return out, nil
}

func (s *Store) GetOnboardingTask(ctx context.Context, id string) (*domain.OnboardingTask, error) {
	r, err := one(s.q.GetOnboardingTask(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	t := toTask(r.HrOnboardingTask, r.CompletedByName)
	return &t, nil
}

func (s *Store) InsertOnboardingTask(ctx context.Context, t domain.OnboardingTask, actor *string) error {
	return s.q.InsertOnboardingTask(ctx, sqlc.InsertOnboardingTaskParams{ID: t.ID, EmployeeID: t.EmployeeID, TaskName: t.TaskName,
		Category: t.Category, Priority: i32(t.Priority), DueDate: parseDayPtr(t.DueDate), CreatedBy: actor})
}

func (s *Store) UpdateOnboardingTask(ctx context.Context, t domain.OnboardingTask) (bool, error) {
	n, err := s.q.UpdateOnboardingTask(ctx, sqlc.UpdateOnboardingTaskParams{ID: t.ID, TaskName: t.TaskName, DueDate: parseDayPtr(t.DueDate),
		IsCompleted: t.IsCompleted, Notes: t.Notes, CompletedBy: t.CompletedBy})
	return n > 0, err
}

func (s *Store) DeleteOnboardingTask(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteOnboardingTask(ctx, id)
	return n > 0, err
}

func toOffboarding(o sqlc.HrOffboarding) domain.Offboarding {
	return domain.Offboarding{ID: o.ID, EmployeeID: o.EmployeeID, ResignationType: o.ResignationType, Status: o.Status,
		ResignationDate: day(o.ResignationDate), LastWorkingDay: day(o.LastWorkingDay), Reason: o.Reason,
		ExitInterviewNotes: o.ExitInterviewNotes, ClearanceHR: o.ClearanceHr, ClearanceIT: o.ClearanceIt,
		ClearanceFinance: o.ClearanceFinance, ClearanceManager: o.ClearanceManager, Notes: o.Notes,
		CompletedAt: o.CompletedAt, CreatedAt: o.CreatedAt}
}

func (s *Store) GetOffboarding(ctx context.Context, employeeID string) (*domain.Offboarding, error) {
	r, err := one(s.q.GetOffboarding(ctx, employeeID))
	if r == nil || err != nil {
		return nil, err
	}
	o := toOffboarding(*r)
	return &o, nil
}

func (s *Store) InsertOffboarding(ctx context.Context, o domain.Offboarding, actor *string) error {
	return wrap(s.q.InsertOffboarding(ctx, sqlc.InsertOffboardingParams{ID: o.ID, EmployeeID: o.EmployeeID, ResignationType: o.ResignationType,
		ResignationDate: parseDay(o.ResignationDate), LastWorkingDay: parseDay(o.LastWorkingDay), Reason: o.Reason, CreatedBy: actor}))
}

func (s *Store) UpdateOffboarding(ctx context.Context, o domain.Offboarding, actor *string) error {
	return s.q.UpdateOffboarding(ctx, sqlc.UpdateOffboardingParams{EmployeeID: o.EmployeeID, Status: o.Status,
		ExitInterviewNotes: o.ExitInterviewNotes, ClearanceHr: o.ClearanceHR, ClearanceIt: o.ClearanceIT,
		ClearanceFinance: o.ClearanceFinance, ClearanceManager: o.ClearanceManager, Notes: o.Notes,
		CompletedAt: o.CompletedAt, UpdatedBy: actor})
}
