// Package domain holds the hr entities and repository ports. The ported
// NüHabit rules live in the hris (people and workforce) and payroll
// (compensation) subpackages. Dates are "YYYY-MM-DD" strings, the form the
// ported rules compute with; amounts are float64 whole rupiah in memory and
// numeric(18,2) in SQL.
package domain

import (
	"strconv"
	"time"
)

// ErrUnique is returned by the store on a unique violation.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }

// Money formats an amount as the decimal string the API and SQL use.
func Money(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

// ParseMoney reads a decimal string; ok is false for anything else.
func ParseMoney(s string) (float64, bool) {
	if s == "" {
		return 0, true
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

// HHMM trims a "15:04:05" time column to "15:04".
func HHMM(s string) string {
	if len(s) >= 5 {
		return s[:5]
	}
	return s
}

// DateOf formats a calendar date.
func DateOf(t time.Time) string { return t.Format("2006-01-02") }

// ─── People ──────────────────────────────────────────────────

type Department struct {
	ID, Code, Name string
	ParentID       *string
	HeadEmployeeID *string
	IsActive       bool
}

type Position struct {
	ID, Title            string
	DepartmentID         *string
	DepartmentName       string
	Level                string
	SalaryMin, SalaryMax float64
	IsActive             bool
}

type EmploymentStatusOption struct {
	Code, Name string
	IsActive   bool
	SortOrder  int
}

type Employee struct {
	ID                   string
	UserID               *string
	EmployeeNo           string
	FullName             string
	Email, Phone         string
	NIK, NPWP            string
	BpjsTkNo, BpjsKesNo  string
	BirthPlace           string
	BirthDate            *string
	Gender               string
	MaritalStatus        string
	Religion             string
	Address              string
	DepartmentID         *string
	DepartmentName       string
	PositionID           *string
	PositionName         string
	ManagerID            *string
	ManagerName          string
	ManagerUserID        *string
	EmploymentStatus     string
	EmploymentStatusName string
	IsActive             bool
	JoinDate             string
	EndDate              *string
	ProbationEndDate     *string
	BankName             string
	BankAccountNo        string
	BankAccountName      string
	EmergencyName        string
	EmergencyPhone       string
	EmergencyRelation    string
	CreatedBy, UpdatedBy *string
	CreatedAt, UpdatedAt time.Time
}

// EmployeeFilter narrows ListEmployees. Zero values do not filter.
type EmployeeFilter struct {
	DepartmentID    string
	ManagerID       string
	IDs             []string
	IncludeInactive bool
	Search          string
	Limit, Offset   int
}

type HistoryEntry struct {
	ID, EmployeeID, ChangeType, Notes string
	PrevStatus, NewStatus             *string
	PrevDepartmentID, NewDepartmentID *string
	PrevPositionID, NewPositionID     *string
	PrevDepartment, NewDepartment     string
	PrevPosition, NewPosition         string
	CreatedBy                         *string
	CreatedAt                         time.Time
}

type Document struct {
	ID, EmployeeID, DocType, Name string
	StorageKey, ContentType       string
	SizeBytes                     int64
	CreatedBy                     *string
	CreatedAt                     time.Time
}

// ─── Workforce ───────────────────────────────────────────────

type Shift struct {
	ID                   string
	Name                 string
	StartTime, EndTime   string // HH:MM:SS
	BreakMinutes         int
	LateToleranceMinutes int
	IsOvernight          bool
	IsActive             bool
	SortOrder            int
}

// ScheduleRow is one weekly pattern row with its shift.
type ScheduleRow struct {
	ID            string
	EmployeeID    string
	DayOfWeek     int
	ShiftID       *string
	Shift         *Shift
	EffectiveFrom string
	EffectiveTo   *string
}

type Holiday struct {
	ID, Date, Name, Type string
	DeductsLeave         bool
	Status, Note         string
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Accuracy  float64 `json:"accuracy,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type Attendance struct {
	ID                string
	EmployeeID        string
	EmployeeName      string
	EmployeeNo        string
	DepartmentName    string
	PositionName      string
	Date              string
	ClockIn, ClockOut *time.Time
	ClockInLocation   *Location
	ClockOutLocation  *Location
	ClockInPhotoKey   *string
	ClockOutPhotoKey  *string
	WorkHours         float64
	BreakMinutes      int
	Status            string
	IsLate            bool
	LateMinutes       int
	OvertimeHours     float64
	ShiftID           *string
	ShiftName         string
	ScheduledStart    *time.Time
	ScheduledEnd      *time.Time
	ValidatedBy       *string
	ValidatedByName   string
	ValidatedAt       *time.Time
	ValidationNotes   string
	Notes             string
	CreatedAt         time.Time
}

type AttendanceFilter struct {
	EmployeeIDs     []string
	From, To        string
	Status          string
	OnlyLate        bool
	OnlyUnvalidated bool
	Limit, Offset   int
}

// AttendanceTotals are one employee's attendance counts for a date range.
type AttendanceTotals struct {
	PresentDays, LateDays, AbsentDays, LateMinutes int
	OvertimeHours                                  float64
}

// LeaveUsage moves the used counters of a leave balance year.
type LeaveUsage struct{ Annual, Sick, Unpaid, Other int }

// LeaveStatusUpdate moves a leave request from one status to another.
type LeaveStatusUpdate struct {
	ID, From, To      string
	ManagerApprovedBy *string
	ApprovedBy        *string
	RejectedBy        *string
	RejectionReason   string
	Actor             *string
}

type LeaveBalance struct {
	ID, EmployeeID string
	Year           int
	AnnualTotal    int
	AnnualUsed     int
	SickUsed       int
	UnpaidUsed     int
	OtherUsed      int
}

func (b LeaveBalance) AnnualRemaining() int { return b.AnnualTotal - b.AnnualUsed }

type Leave struct {
	ID                    string
	EmployeeID            string
	EmployeeName          string
	EmployeeNo            string
	DepartmentName        string
	EmployeeManagerID     *string
	LeaveType             string
	StartDate, EndDate    string
	TotalDays             int
	Reason                string
	Status                string
	ManagerApprovedBy     *string
	ManagerApprovedByName string
	ManagerApprovedAt     *time.Time
	ApprovedBy            *string
	ApprovedByName        string
	ApprovedAt            *time.Time
	RejectedBy            *string
	RejectionReason       string
	CreatedBy             *string
	CreatedAt             time.Time
}

type LeaveFilter struct {
	EmployeeIDs   []string
	Status        string
	Statuses      []string
	LeaveType     string
	From, To      string
	Limit, Offset int
}

type Overtime struct {
	ID                 string
	EmployeeID         string
	EmployeeName       string
	EmployeeNo         string
	DepartmentName     string
	EmployeeManagerID  *string
	Date               string
	StartTime, EndTime string // HH:MM
	Hours              float64
	Source             string
	Status             string
	Reason             string
	RequestedBy        *string
	RequestedByName    string
	DecidedBy          *string
	DecidedByName      string
	DecidedAt          *time.Time
	RejectionReason    string
	CreatedAt          time.Time
}

type OvertimeFilter struct {
	EmployeeIDs []string
	Source      string
	Status      string
	From, To    string
}

type Contract struct {
	ID                 string
	ContractNumber     string
	EmployeeID         string
	EmployeeName       string
	EmployeeNo         string
	DepartmentNameNow  string
	ContractType       string
	Status             string
	StartDate          string
	EndDate            *string
	ProbationEndDate   *string
	ParentContractID   *string
	Sequence           int
	PositionTitle      string
	DepartmentName     string
	WorkLocation       string
	BaseSalary         *float64
	SignedAt           *string
	CompensationAmount *float64
	TerminatedReason   string
	Notes              string
	CreatedBy          *string
	CreatedAt          time.Time
}

// ContractStatusUpdate moves a contract from one status to another; nil
// fields keep their value.
type ContractStatusUpdate struct {
	ID, From, To     string
	EndDate          *string
	SignedAt         *string
	Compensation     *float64
	TerminatedReason *string
	Actor            *string
}

type ContractFilter struct {
	EmployeeID   string
	Status       string // "" = every status
	ContractType string
	Search       string
}

type Announcement struct {
	ID            string
	Title, Body   string
	Tags          []string
	Status        string
	IsPinned      bool
	TargetScope   string
	DepartmentIDs []string
	PublishAt     *time.Time
	ExpiresAt     *time.Time
	CreatedBy     *string
	CreatedByName string
	CreatedAt     time.Time
	IsRead        bool
	ReadCount     int
}

type OnboardingTask struct {
	ID, EmployeeID  string
	TaskName        string
	Category        string
	Priority        int
	DueDate         *string
	IsCompleted     bool
	CompletedAt     *time.Time
	CompletedBy     *string
	CompletedByName string
	Notes           string
}

type Offboarding struct {
	ID, EmployeeID     string
	ResignationType    string
	Status             string
	ResignationDate    string
	LastWorkingDay     string
	Reason             string
	ExitInterviewNotes string
	ClearanceHR        bool
	ClearanceIT        bool
	ClearanceFinance   bool
	ClearanceManager   bool
	Notes              string
	CompletedAt        *time.Time
	CreatedAt          time.Time
}

// ─── Compensation ────────────────────────────────────────────

// PayrollSettings is the hr.payroll_settings row. Rates are percentages.
type PayrollSettings struct {
	BpjsTkJhtEmployee, BpjsTkJhtEmployer float64
	BpjsTkJpEmployee, BpjsTkJpEmployer   float64
	BpjsTkJkk, BpjsTkJkm                 float64
	BpjsKesEmployee, BpjsKesEmployer     float64
	BpjsKesMaxUpah                       float64
	TaperaEmployee, TaperaEmployer       float64
	PTKP                                 map[string]float64 // "TK/0" .. "K/3"
	BracketLimits                        [4]float64
	OvertimeMultiplier                   float64
	OvertimeMultiplierHoliday            float64
	OvertimeHourlyDivisor                float64
	LateDeductionMode                    string
	LateDeductionAmount                  float64
	LoanMaxInstallmentPercent            float64
	LoanMaxActivePerEmployee             int
	PayrollDay                           int
	UpdatedBy                            *string
	UpdatedAt                            time.Time
}

// PTKPStatuses in display order.
var PTKPStatuses = []string{"TK/0", "TK/1", "TK/2", "TK/3", "K/0", "K/1", "K/2", "K/3"}

type Salary struct {
	ID                 string
	EmployeeID         string
	EmployeeName       string
	EmployeeNo         string
	DepartmentName     string
	BaseSalary         float64
	FixedAllowance     float64
	VariableAllowance  float64
	TransportAllowance float64
	MealAllowance      float64
	HousingAllowance   float64
	PTKPStatus         string
	IsTaxable          bool
	BpjsTkEnrolled     bool
	BpjsKesEnrolled    bool
	TaperaEnrolled     bool
	EffectiveDate      string
	EndDate            *string
	IsActive           bool
	Notes              string
	CreatedBy          *string
}

type PayrollRun struct {
	ID                 string
	RunName            string
	PeriodMonth        int
	PeriodYear         int
	Status             string
	TotalEmployees     int
	TotalGross         float64
	TotalDeductions    float64
	TotalNet           float64
	TotalBpjsEmployee  float64
	TotalBpjsEmployer  float64
	TotalPph21         float64
	NeedsRecalculation bool
	CalculatedAt       *time.Time
	CalculatedBy       *string
	ApprovedAt         *time.Time
	ApprovedBy         *string
	ApprovedByName     string
	PaidAt             *time.Time
	PaidBy             *string
	Notes              string
	CreatedBy          *string
	CreatedAt          time.Time
}

// Payslip is one hr.payroll_details row.
type Payslip struct {
	ID                   string
	RunID                string
	PeriodMonth          int
	PeriodYear           int
	RunStatus            string
	EmployeeID           string
	EmployeeName         string
	EmployeeNo           string
	DepartmentName       string
	PositionName         string
	BaseSalary           float64
	FullBaseSalary       float64
	FixedAllowance       float64
	VariableAllowance    float64
	TransportAllowance   float64
	MealAllowance        float64
	HousingAllowance     float64
	OvertimePay          float64
	Thr                  float64
	Bonus                float64
	GrossSalary          float64
	BpjsTkJhtDeduction   float64
	BpjsTkJpDeduction    float64
	BpjsKesDeduction     float64
	TaperaDeduction      float64
	Pph21Deduction       float64
	UnpaidLeaveDeduction float64
	LateDeduction        float64
	LoanDeduction        float64
	LoanDetails          []byte // JSON []payroll.LoanInstallmentDetail
	TotalDeductions      float64
	NetSalary            float64
	BpjsTkJhtEmployer    float64
	BpjsTkJpEmployer     float64
	BpjsTkJkkEmployer    float64
	BpjsTkJkmEmployer    float64
	BpjsKesEmployer      float64
	TaperaEmployer       float64
	TaxableIncome        float64
	PTKPAmount           float64
	Pph21Annual          float64
	WorkingDays          float64
	PresentDays          float64
	LateDays             float64
	UnpaidLeaveDays      float64
	OvertimeHours        float64
	ProrateFactor        float64
}

type PayslipFilter struct {
	EmployeeID string
	Year       int
	RunID      string
	OnlyPaid   bool
}

type PayrollAddition struct {
	ID             string
	EmployeeID     string
	EmployeeName   string
	EmployeeNo     string
	DepartmentName string
	PeriodMonth    int
	PeriodYear     int
	Amount         float64
	Label          string
	CreatedBy      *string
	CreatedAt      time.Time
}

type Loan struct {
	ID                    string
	EmployeeID            string
	EmployeeName          string
	EmployeeNo            string
	DepartmentName        string
	LoanType              string
	PrincipalAmount       float64
	InterestRate          float64
	TenorMonths           int
	MonthlyInstallment    float64
	RemainingBalance      float64
	PaidAmount            float64
	FirstInstallmentMonth *int
	FirstInstallmentYear  *int
	Status                string
	IsActive              bool
	Purpose               string
	ApprovedBy            *string
	ApprovedByName        string
	ApprovedAt            *time.Time
	RejectedBy            *string
	RejectionReason       string
	CreatedBy             *string
	CreatedAt             time.Time
}
