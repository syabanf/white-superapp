package postgres

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
)

func (s *Store) DashAttendanceDaily(ctx context.Context, from, to string, f domain.DashFilter) ([]domain.DashAttendanceDay, error) {
	rows, err := s.q.DashAttendanceDaily(ctx, sqlc.DashAttendanceDailyParams{DateFrom: parseDay(from), DateTo: parseDay(to),
		DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashAttendanceDay, len(rows))
	for i, r := range rows {
		out[i] = domain.DashAttendanceDay{Date: day(r.Date), DepartmentID: r.DepartmentID, Records: int(r.Records), Present: int(r.Present),
			Late: int(r.Late), Absent: int(r.Absent), LateMinutes: int(r.LateMinutes), WorkHours: r.WorkHours}
	}
	return out, nil
}

func (s *Store) DashAttendanceByEmployee(ctx context.Context, from, to string, f domain.DashFilter) ([]domain.DashEmployeeAttendance, error) {
	rows, err := s.q.DashAttendanceByEmployee(ctx, sqlc.DashAttendanceByEmployeeParams{DateFrom: parseDay(from), DateTo: parseDay(to),
		DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashEmployeeAttendance, len(rows))
	for i, r := range rows {
		out[i] = domain.DashEmployeeAttendance{EmployeeID: r.EmployeeID, FullName: r.FullName, DepartmentName: r.DepartmentName,
			PositionName: r.PositionName, Records: int(r.Records), Late: int(r.Late), LateMinutes: int(r.LateMinutes), Absent: int(r.Absent)}
	}
	return out, nil
}

func (s *Store) DashClockInOffsets(ctx context.Context, from, to string, f domain.DashFilter) ([]domain.DashOffsetBin, error) {
	rows, err := s.q.DashClockInOffsets(ctx, sqlc.DashClockInOffsetsParams{DateFrom: parseDay(from), DateTo: parseDay(to),
		DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashOffsetBin, len(rows))
	for i, r := range rows {
		out[i] = domain.DashOffsetBin{Bin: int(r.Bin), Records: int(r.Records)}
	}
	return out, nil
}

func (s *Store) DashLeaves(ctx context.Context, from, to string, f domain.DashFilter) ([]domain.DashLeave, error) {
	rows, err := s.q.DashLeaves(ctx, sqlc.DashLeavesParams{DateFrom: parseDay(from), DateTo: parseDay(to),
		DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashLeave, len(rows))
	for i, r := range rows {
		out[i] = domain.DashLeave{EmployeeID: r.EmployeeID, DepartmentID: r.DepartmentID, LeaveType: r.LeaveType,
			StartDate: day(r.StartDate), EndDate: day(r.EndDate)}
	}
	return out, nil
}

func (s *Store) DashPending(ctx context.Context, f domain.DashFilter) ([]domain.DashPending, error) {
	rows, err := s.q.DashPending(ctx, sqlc.DashPendingParams{DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashPending, len(rows))
	for i, r := range rows {
		out[i] = domain.DashPending{Kind: r.Kind, ID: r.ID, EmployeeID: r.EmployeeID, FullName: r.FullName, DepartmentName: r.DepartmentName,
			Type: r.Type, Date: day(r.Date), Status: r.Status, Amount: r.Amount, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

func (s *Store) DashLeaveBalances(ctx context.Context, year int, f domain.DashFilter) ([]domain.DashBalance, error) {
	rows, err := s.q.DashLeaveBalances(ctx, sqlc.DashLeaveBalancesParams{Year: i32(year),
		DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashBalance, len(rows))
	for i, r := range rows {
		out[i] = domain.DashBalance{DepartmentID: r.DepartmentID, Quota: int(r.Quota), Used: int(r.Used)}
	}
	return out, nil
}

func (s *Store) DashOvertime(ctx context.Context, from, to string, f domain.DashFilter) ([]domain.DashOvertime, error) {
	rows, err := s.q.DashOvertime(ctx, sqlc.DashOvertimeParams{DateFrom: parseDay(from), DateTo: parseDay(to),
		DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashOvertime, len(rows))
	for i, r := range rows {
		out[i] = domain.DashOvertime{Date: day(r.Date), EmployeeID: r.EmployeeID, FullName: r.FullName, DepartmentID: r.DepartmentID,
			DepartmentName: r.DepartmentName, PositionName: r.PositionName, Hours: r.Hours, Cost: r.Cost}
	}
	return out, nil
}

func (s *Store) DashEmployees(ctx context.Context, f domain.DashFilter) ([]domain.DashEmployee, error) {
	rows, err := s.q.DashEmployees(ctx, sqlc.DashEmployeesParams{DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashEmployee, len(rows))
	for i, r := range rows {
		out[i] = domain.DashEmployee{ID: r.ID, FullName: r.FullName, DepartmentID: r.DepartmentID, DepartmentName: r.DepartmentName,
			PositionID: r.PositionID, PositionName: r.PositionName, Level: r.Level, EmploymentStatus: r.EmploymentStatus,
			StatusName: r.StatusName, Gender: r.Gender, ContractType: r.ContractType, JoinDate: day(r.JoinDate),
			BirthDate: dayPtr(r.BirthDate), EndDate: dayPtr(r.EndDate), ProbationEndDate: dayPtr(r.ProbationEndDate), IsActive: r.IsActive}
	}
	return out, nil
}

func (s *Store) DashContractsEnding(ctx context.Context, from, to string, f domain.DashFilter) ([]domain.DashContractEnding, error) {
	rows, err := s.q.DashContractsEnding(ctx, sqlc.DashContractsEndingParams{DateFrom: parseDay(from), DateTo: parseDay(to),
		DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashContractEnding, len(rows))
	for i, r := range rows {
		out[i] = domain.DashContractEnding{ID: r.ID, ContractNumber: r.ContractNumber, ContractType: r.ContractType,
			EndDate: *dayPtr(r.EndDate), EmployeeID: r.EmployeeID, FullName: r.FullName, DepartmentName: r.DepartmentName}
	}
	return out, nil
}

func (s *Store) DashPayroll(ctx context.Context, from, to string, f domain.DashFilter) ([]domain.DashPayroll, error) {
	rows, err := s.q.DashPayroll(ctx, sqlc.DashPayrollParams{DateFrom: parseDay(from), DateTo: parseDay(to),
		DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashPayroll, len(rows))
	for i, r := range rows {
		out[i] = domain.DashPayroll{Month: day(r.Month), DepartmentID: r.DepartmentID, Employees: int(r.Employees), Gross: r.Gross,
			Net: r.Net, Pph21: r.Pph21, BpjsEmployee: r.BpjsEmployee, BpjsEmployer: r.BpjsEmployer}
	}
	return out, nil
}

func (s *Store) DashPayrollByLevel(ctx context.Context, from, to string, f domain.DashFilter) ([]domain.DashLevel, error) {
	rows, err := s.q.DashPayrollByLevel(ctx, sqlc.DashPayrollByLevelParams{DateFrom: parseDay(from), DateTo: parseDay(to),
		DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashLevel, len(rows))
	for i, r := range rows {
		out[i] = domain.DashLevel{Level: r.Level, Employees: int(r.Employees), AvgBase: r.AvgBase, AvgGross: r.AvgGross,
			MinGross: r.MinGross, MaxGross: r.MaxGross}
	}
	return out, nil
}

func (s *Store) DashLoans(ctx context.Context, f domain.DashFilter) ([]domain.DashLoan, error) {
	rows, err := s.q.DashLoans(ctx, sqlc.DashLoansParams{DepartmentID: f.DepartmentID, EmploymentStatus: f.EmploymentStatus, PositionID: f.PositionID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DashLoan, len(rows))
	for i, r := range rows {
		out[i] = domain.DashLoan{ID: r.ID, LoanType: r.LoanType, EmployeeID: r.EmployeeID, FullName: r.FullName,
			DepartmentName: r.DepartmentName, Principal: r.PrincipalAmount, Remaining: r.RemainingBalance, Monthly: r.MonthlyInstallment}
	}
	return out, nil
}
