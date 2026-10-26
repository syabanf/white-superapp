package domain

import (
	"context"
	"time"
)

// DashFilter narrows the HR dashboard to a slice of employees ("" = any).
type DashFilter struct {
	DepartmentID     string
	EmploymentStatus string
	PositionID       string
}

// DashAttendanceDay is one department's attendance on one date.
type DashAttendanceDay struct {
	Date, DepartmentID                          string
	Records, Present, Late, Absent, LateMinutes int
	WorkHours                                   float64
}

// DashEmployeeAttendance is one employee's attendance over a period.
type DashEmployeeAttendance struct {
	EmployeeID, FullName, DepartmentName, PositionName string
	Records, Late, LateMinutes, Absent                 int
}

// DashOffsetBin counts clock-ins starting Bin minutes after the shift start.
type DashOffsetBin struct{ Bin, Records int }

// DashLeave is one approved leave overlapping the period.
type DashLeave struct {
	EmployeeID, DepartmentID, LeaveType string
	StartDate, EndDate                  string
}

// DashPending is a leave or overtime request waiting for a decision.
type DashPending struct {
	Kind, ID, EmployeeID, FullName, DepartmentName string
	Type, Date, Status                             string
	Amount                                         float64
	CreatedAt                                      time.Time
}

// DashBalance is a department's annual leave quota and use for a year.
type DashBalance struct {
	DepartmentID string
	Quota, Used  int
}

// DashOvertime is one approved overtime request with its estimated cost.
type DashOvertime struct {
	Date, EmployeeID, FullName   string
	DepartmentID, DepartmentName string
	PositionName                 string
	Hours, Cost                  float64
}

// DashEmployee is an employee as the dashboard groups them.
type DashEmployee struct {
	ID, FullName                         string
	DepartmentID, DepartmentName         string
	PositionID, PositionName, Level      string
	EmploymentStatus, StatusName, Gender string
	ContractType                         string
	JoinDate                             string
	BirthDate, EndDate, ProbationEndDate *string
	IsActive                             bool
}

// DashContractEnding is an active contract ending soon.
type DashContractEnding struct {
	ID, ContractNumber, ContractType, EndDate string
	EmployeeID, FullName, DepartmentName      string
}

// DashPayroll sums one department's payslips of a run month.
type DashPayroll struct {
	Month, DepartmentID                           string
	Employees                                     int
	Gross, Net, Pph21, BpjsEmployee, BpjsEmployer float64
}

// DashLevel is pay by position level over approved and paid runs.
type DashLevel struct {
	Level                                 string
	Employees                             int
	AvgBase, AvgGross, MinGross, MaxGross float64
}

// DashLoan is an approved loan still being repaid.
type DashLoan struct {
	ID, LoanType, EmployeeID, FullName, DepartmentName string
	Principal, Remaining, Monthly                      float64
}

// DashboardRepo reads the aggregates behind the HR dashboards.
type DashboardRepo interface {
	DashAttendanceDaily(ctx context.Context, from, to string, f DashFilter) ([]DashAttendanceDay, error)
	DashAttendanceByEmployee(ctx context.Context, from, to string, f DashFilter) ([]DashEmployeeAttendance, error)
	DashClockInOffsets(ctx context.Context, from, to string, f DashFilter) ([]DashOffsetBin, error)
	DashLeaves(ctx context.Context, from, to string, f DashFilter) ([]DashLeave, error)
	DashPending(ctx context.Context, f DashFilter) ([]DashPending, error)
	DashLeaveBalances(ctx context.Context, year int, f DashFilter) ([]DashBalance, error)
	DashOvertime(ctx context.Context, from, to string, f DashFilter) ([]DashOvertime, error)
	DashEmployees(ctx context.Context, f DashFilter) ([]DashEmployee, error)
	DashContractsEnding(ctx context.Context, from, to string, f DashFilter) ([]DashContractEnding, error)
	// DashPayroll and DashPayrollByLevel cover runs whose month starts in [from, to].
	DashPayroll(ctx context.Context, from, to string, f DashFilter) ([]DashPayroll, error)
	DashPayrollByLevel(ctx context.Context, from, to string, f DashFilter) ([]DashLevel, error)
	DashLoans(ctx context.Context, f DashFilter) ([]DashLoan, error)
}
