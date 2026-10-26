package domain

import (
	"context"
	"time"
)

// Store is every hr repository port; adapter/postgres implements it.
type Store interface {
	DashboardRepo
	WithTx(ctx context.Context, fn func(tx Store) error) error
	ListDepartments(ctx context.Context, includeInactive bool) ([]Department, error)
	GetDepartment(ctx context.Context, id string) (*Department, error)
	InsertDepartment(ctx context.Context, d Department, actor *string) error
	UpdateDepartment(ctx context.Context, d Department, actor *string) (bool, error)
	DeleteDepartment(ctx context.Context, id string, actor *string) (bool, error)
	CountDepartmentEmployees(ctx context.Context, id string) (int, error)
	ListPositions(ctx context.Context, departmentID string, includeInactive bool) ([]Position, error)
	GetPosition(ctx context.Context, id string) (*Position, error)
	InsertPosition(ctx context.Context, p Position, actor *string) error
	UpdatePosition(ctx context.Context, p Position, actor *string) (bool, error)
	DeletePosition(ctx context.Context, id string, actor *string) (bool, error)
	ListEmploymentStatuses(ctx context.Context) ([]EmploymentStatusOption, error)
	UpsertEmploymentStatus(ctx context.Context, o EmploymentStatusOption) error
	DeleteEmploymentStatus(ctx context.Context, code string) (bool, error)
	ListEmployees(ctx context.Context, f EmployeeFilter) ([]Employee, int, error)
	GetEmployee(ctx context.Context, id string) (*Employee, error)
	EmployeeIDByUser(ctx context.Context, userID string) (string, error)
	ListEmployeeNos(ctx context.Context, prefix string) ([]string, error)
	InsertEmployee(ctx context.Context, e Employee) error
	UpdateEmployee(ctx context.Context, e Employee) (bool, error)
	SetEmployeeStatus(ctx context.Context, id, status string, active bool, endDate *string, actor *string) error
	DeleteEmployee(ctx context.Context, id string, actor *string) (bool, error)
	SetEmployeeUser(ctx context.Context, id string, userID, actor *string) (bool, error)
	UnlinkUser(ctx context.Context, userID string) (int, error)
	LinkedUserIDsInDepartment(ctx context.Context, code string) ([]string, error)
	// ClaimRecruitmentHire maps a recruitment candidate to its employee;
	// false when the candidate was already mapped.
	ClaimRecruitmentHire(ctx context.Context, candidateID, employeeID string) (bool, error)
	InsertHistory(ctx context.Context, h HistoryEntry) error
	ListHistory(ctx context.Context, employeeID string) ([]HistoryEntry, error)
	InsertDocument(ctx context.Context, d Document) error
	ListDocuments(ctx context.Context, employeeID string) ([]Document, error)
	GetDocument(ctx context.Context, id string) (*Document, error)
	DeleteDocument(ctx context.Context, id string) (bool, error)
	ClaimReminder(ctx context.Context, kind, ref, d string) (bool, error)
	GetPayrollSettings(ctx context.Context) (*PayrollSettings, error)
	UpdatePayrollSettings(ctx context.Context, p PayrollSettings, actor *string) error
	ListSalaries(ctx context.Context, employeeID string, activeOnly bool) ([]Salary, error)
	GetSalary(ctx context.Context, id string) (*Salary, error)
	ActiveSalary(ctx context.Context, employeeID string) (*Salary, error)
	InsertSalary(ctx context.Context, x Salary, endPrevious string, actor *string) error
	UpdateSalary(ctx context.Context, x Salary, actor *string) (bool, error)
	DeactivateSalary(ctx context.Context, id, endDate string, actor *string) (bool, error)
	ListPayrollRuns(ctx context.Context, year int, status string) ([]PayrollRun, error)
	GetPayrollRun(ctx context.Context, id string) (*PayrollRun, error)
	GetPayrollRunByPeriod(ctx context.Context, month, year int) (*PayrollRun, error)
	InsertPayrollRun(ctx context.Context, r PayrollRun) error
	DeletePayrollDetails(ctx context.Context, runID string) error
	InsertPayslip(ctx context.Context, p Payslip) error
	SetRunCalculated(ctx context.Context, r PayrollRun) error
	ApproveRun(ctx context.Context, id string, by *string) (bool, error)
	MarkRunPaid(ctx context.Context, id string, paidAt time.Time, by *string, notes string) (*PayrollRun, error)
	FlagRunRecalculation(ctx context.Context, id string) error
	DeletePayrollRun(ctx context.Context, id string) (bool, error)
	ListPayslips(ctx context.Context, f PayslipFilter) ([]Payslip, error)
	GetPayslip(ctx context.Context, id string) (*Payslip, error)
	ListPayrollAdditions(ctx context.Context, month, year int, employeeID string) ([]PayrollAddition, error)
	GetPayrollAddition(ctx context.Context, id string) (*PayrollAddition, error)
	InsertPayrollAddition(ctx context.Context, a PayrollAddition) error
	DeletePayrollAddition(ctx context.Context, id string) (bool, error)
	SumPayrollAdditions(ctx context.Context, employeeID string, month, year int) (float64, error)
	ListLoans(ctx context.Context, employeeID, status string) ([]Loan, error)
	GetLoan(ctx context.Context, id string) (*Loan, error)
	CountOpenLoans(ctx context.Context, employeeID string, includePending bool) (int, error)
	InsertLoan(ctx context.Context, l Loan) error
	ApproveLoan(ctx context.Context, id string, by *string, at time.Time, month, year int) (bool, error)
	RejectLoan(ctx context.Context, id string, by *string, reason string) (bool, error)
	OpenLoans(ctx context.Context, employeeID string, lock bool) ([]Loan, error)
	ApplyLoanPayment(ctx context.Context, id string, paid, remaining float64) error
	OvertimeForPeriod(ctx context.Context, employeeID, from, to string) ([]Overtime, error)
	ListContracts(ctx context.Context, f ContractFilter) ([]Contract, error)
	GetContract(ctx context.Context, id string) (*Contract, error)
	CountContractsOfType(ctx context.Context, contractType string, year int) (int, error)
	InsertContract(ctx context.Context, c Contract) error
	UpdateDraftContract(ctx context.Context, c Contract, actor *string) (bool, error)
	SetContractStatus(ctx context.Context, u ContractStatusUpdate) (bool, error)
	ActiveContractNumber(ctx context.Context, employeeID, exceptID string) (string, error)
	PkwtChain(ctx context.Context, employeeID string) ([][2]*string, error)
	DeleteDraftContract(ctx context.Context, id string) (bool, error)
	EmployeesWithoutContract(ctx context.Context) ([]Employee, error)
	ProbationsEnding(ctx context.Context, from, to string) ([]Employee, error)
	ActiveBirthdays(ctx context.Context) ([]Employee, error)
	ListAnnouncements(ctx context.Context, status string) ([]Announcement, error)
	GetAnnouncement(ctx context.Context, id string) (*Announcement, error)
	AnnouncementFeed(ctx context.Context, employeeID string, now time.Time) ([]Announcement, error)
	SaveAnnouncement(ctx context.Context, a Announcement, insert bool, actor *string) (bool, error)
	DeleteAnnouncement(ctx context.Context, id string) (bool, error)
	MarkAnnouncementRead(ctx context.Context, id, employeeID string) error
	ListOnboardingTasks(ctx context.Context, employeeID string) ([]OnboardingTask, error)
	GetOnboardingTask(ctx context.Context, id string) (*OnboardingTask, error)
	InsertOnboardingTask(ctx context.Context, t OnboardingTask, actor *string) error
	UpdateOnboardingTask(ctx context.Context, t OnboardingTask) (bool, error)
	DeleteOnboardingTask(ctx context.Context, id string) (bool, error)
	GetOffboarding(ctx context.Context, employeeID string) (*Offboarding, error)
	InsertOffboarding(ctx context.Context, o Offboarding, actor *string) error
	UpdateOffboarding(ctx context.Context, o Offboarding, actor *string) error
	ListShifts(ctx context.Context, includeInactive bool) ([]Shift, error)
	GetShift(ctx context.Context, id string) (*Shift, error)
	InsertShift(ctx context.Context, sh Shift, actor *string) error
	UpdateShift(ctx context.Context, sh Shift, actor *string) (bool, error)
	DeleteShift(ctx context.Context, id string) (deactivated, found bool, err error)
	ListScheduleRows(ctx context.Context, employeeIDs []string) ([]ScheduleRow, error)
	ReplaceSchedule(ctx context.Context, employeeID, from, closeOn string, rows []ScheduleRow, actor *string) error
	ListHolidays(ctx context.Context, from, to string, onlyActive bool) ([]Holiday, error)
	InsertHoliday(ctx context.Context, h Holiday, actor *string) error
	UpdateHoliday(ctx context.Context, h Holiday, actor *string) (bool, error)
	DeleteHoliday(ctx context.Context, id string) (bool, error)
	ListAttendance(ctx context.Context, f AttendanceFilter) ([]Attendance, int, error)
	GetAttendance(ctx context.Context, id string) (*Attendance, error)
	AttendanceIDOn(ctx context.Context, employeeID, date string) (string, error)
	InsertAttendance(ctx context.Context, a Attendance, actor *string) error
	InsertAttendanceRecord(ctx context.Context, a Attendance, actor *string) error
	ClockOut(ctx context.Context, a Attendance) (bool, error)
	ValidateAttendance(ctx context.Context, id, status string, validatedBy *string, notes string) (bool, error)
	AttendanceTotals(ctx context.Context, employeeIDs []string, from, to string) (map[string]AttendanceTotals, error)
	GetLeaveBalance(ctx context.Context, employeeID string, year int) (*LeaveBalance, error)
	EnsureLeaveBalance(ctx context.Context, b LeaveBalance, actor *string) (*LeaveBalance, error)
	SetLeaveBalance(ctx context.Context, employeeID string, year, total, used int, actor *string) (*LeaveBalance, error)
	AddLeaveUsage(ctx context.Context, employeeID string, year int, u LeaveUsage) error
	PendingAnnualDays(ctx context.Context, employeeID string, year int) (int, error)
	ListLeaves(ctx context.Context, f LeaveFilter) ([]Leave, int, error)
	GetLeave(ctx context.Context, id string) (*Leave, error)
	InsertLeave(ctx context.Context, l Leave) error
	LeaveOverlaps(ctx context.Context, employeeID, start, end string) (bool, error)
	SetLeaveStatus(ctx context.Context, u LeaveStatusUpdate) (bool, error)
	ApprovedLeavesOn(ctx context.Context, date string) (map[string]string, error)
	ListOvertime(ctx context.Context, f OvertimeFilter) ([]Overtime, error)
	GetOvertime(ctx context.Context, id string) (*Overtime, error)
	InsertOvertime(ctx context.Context, o Overtime, actor *string) error
	DecideOvertime(ctx context.Context, id, status string, decidedBy *string, rejection string) (bool, error)
}
