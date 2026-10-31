package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
)

// ─── Settings ────────────────────────────────────────────────

func (s *Store) GetPayrollSettings(ctx context.Context) (*domain.PayrollSettings, error) {
	r, err := s.q.GetPayrollSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &domain.PayrollSettings{
		BpjsTkJhtEmployee: num(r.BpjsTkJhtEmployee), BpjsTkJhtEmployer: num(r.BpjsTkJhtEmployer),
		BpjsTkJpEmployee: num(r.BpjsTkJpEmployee), BpjsTkJpEmployer: num(r.BpjsTkJpEmployer),
		BpjsTkJkk: num(r.BpjsTkJkk), BpjsTkJkm: num(r.BpjsTkJkm),
		BpjsKesEmployee: num(r.BpjsKesEmployee), BpjsKesEmployer: num(r.BpjsKesEmployer), BpjsKesMaxUpah: num(r.BpjsKesMaxUpah),
		TaperaEmployee: num(r.TaperaEmployee), TaperaEmployer: num(r.TaperaEmployer),
		PTKP: map[string]float64{
			"TK/0": num(r.PtkpTk0), "TK/1": num(r.PtkpTk1), "TK/2": num(r.PtkpTk2), "TK/3": num(r.PtkpTk3),
			"K/0": num(r.PtkpK0), "K/1": num(r.PtkpK1), "K/2": num(r.PtkpK2), "K/3": num(r.PtkpK3),
		},
		BracketLimits:      [4]float64{num(r.Pph21Bracket1), num(r.Pph21Bracket2), num(r.Pph21Bracket3), num(r.Pph21Bracket4)},
		OvertimeMultiplier: num(r.OvertimeMultiplier), OvertimeMultiplierHoliday: num(r.OvertimeMultiplierHoliday),
		OvertimeHourlyDivisor: num(r.OvertimeHourlyDivisor), LateDeductionMode: r.LateDeductionMode,
		LateDeductionAmount: num(r.LateDeductionAmount), LoanMaxInstallmentPercent: num(r.LoanMaxInstallmentPercent),
		LoanMaxActivePerEmployee: int(r.LoanMaxActivePerEmployee), PayrollDay: int(r.PayrollDay),
		UpdatedBy: r.UpdatedBy, UpdatedAt: r.UpdatedAt,
	}, nil
}

func (s *Store) UpdatePayrollSettings(ctx context.Context, p domain.PayrollSettings, actor *string) error {
	m := money
	_, err := s.q.UpdatePayrollSettings(ctx, sqlc.UpdatePayrollSettingsParams{
		BpjsTkJhtEmployee: m(p.BpjsTkJhtEmployee), BpjsTkJhtEmployer: m(p.BpjsTkJhtEmployer),
		BpjsTkJpEmployee: m(p.BpjsTkJpEmployee), BpjsTkJpEmployer: m(p.BpjsTkJpEmployer),
		BpjsTkJkk: m(p.BpjsTkJkk), BpjsTkJkm: m(p.BpjsTkJkm), BpjsKesEmployee: m(p.BpjsKesEmployee),
		BpjsKesEmployer: m(p.BpjsKesEmployer), BpjsKesMaxUpah: m(p.BpjsKesMaxUpah),
		TaperaEmployee: m(p.TaperaEmployee), TaperaEmployer: m(p.TaperaEmployer),
		PtkpTk0: m(p.PTKP["TK/0"]), PtkpTk1: m(p.PTKP["TK/1"]), PtkpTk2: m(p.PTKP["TK/2"]), PtkpTk3: m(p.PTKP["TK/3"]),
		PtkpK0: m(p.PTKP["K/0"]), PtkpK1: m(p.PTKP["K/1"]), PtkpK2: m(p.PTKP["K/2"]), PtkpK3: m(p.PTKP["K/3"]),
		Pph21Bracket1: m(p.BracketLimits[0]), Pph21Bracket2: m(p.BracketLimits[1]),
		Pph21Bracket3: m(p.BracketLimits[2]), Pph21Bracket4: m(p.BracketLimits[3]),
		OvertimeMultiplier: m(p.OvertimeMultiplier), OvertimeMultiplierHoliday: m(p.OvertimeMultiplierHoliday),
		OvertimeHourlyDivisor: m(p.OvertimeHourlyDivisor), LateDeductionMode: p.LateDeductionMode,
		LateDeductionAmount: m(p.LateDeductionAmount), LoanMaxInstallmentPercent: m(p.LoanMaxInstallmentPercent),
		LoanMaxActivePerEmployee: i32(p.LoanMaxActivePerEmployee), PayrollDay: i32(p.PayrollDay), UpdatedBy: actor,
	})
	return err
}

// ─── Salaries ────────────────────────────────────────────────

func toSalary(x sqlc.HrEmployeeSalary, name, no, dept string) domain.Salary {
	return domain.Salary{ID: x.ID, EmployeeID: x.EmployeeID, EmployeeName: name, EmployeeNo: no, DepartmentName: dept,
		BaseSalary: num(x.BaseSalary), FixedAllowance: num(x.FixedAllowance), VariableAllowance: num(x.VariableAllowance),
		TransportAllowance: num(x.TransportAllowance), MealAllowance: num(x.MealAllowance), HousingAllowance: num(x.HousingAllowance),
		PTKPStatus: x.PtkpStatus, IsTaxable: x.IsTaxable, BpjsTkEnrolled: x.BpjsTkEnrolled, BpjsKesEnrolled: x.BpjsKesEnrolled,
		TaperaEnrolled: x.TaperaEnrolled, EffectiveDate: day(x.EffectiveDate), EndDate: dayPtr(x.EndDate), IsActive: x.IsActive,
		Notes: x.Notes, CreatedBy: x.CreatedBy}
}

func (s *Store) ListSalaries(ctx context.Context, employeeID string, activeOnly bool) ([]domain.Salary, error) {
	rows, err := s.q.ListSalaries(ctx, sqlc.ListSalariesParams{EmployeeID: employeeID, ActiveOnly: activeOnly})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Salary, len(rows))
	for i, r := range rows {
		out[i] = toSalary(r.HrEmployeeSalary, r.FullName, r.EmployeeNo, r.DepartmentName)
	}
	return out, nil
}

func (s *Store) GetSalary(ctx context.Context, id string) (*domain.Salary, error) {
	r, err := one(s.q.GetSalary(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	x := toSalary(r.HrEmployeeSalary, r.FullName, r.EmployeeNo, r.DepartmentName)
	return &x, nil
}

func (s *Store) ActiveSalary(ctx context.Context, employeeID string) (*domain.Salary, error) {
	r, err := one(s.q.ActiveSalary(ctx, employeeID))
	if r == nil || err != nil {
		return nil, err
	}
	x := toSalary(*r, "", "", "")
	return &x, nil
}

// InsertSalary ends the running version on endPrevious and inserts x.
func (s *Store) InsertSalary(ctx context.Context, x domain.Salary, endPrevious string, actor *string) error {
	end := parseDay(endPrevious)
	if err := s.q.EndActiveSalary(ctx, sqlc.EndActiveSalaryParams{EmployeeID: x.EmployeeID, EndDate: &end, UpdatedBy: actor}); err != nil {
		return err
	}
	return wrap(s.q.InsertSalary(ctx, sqlc.InsertSalaryParams{ID: x.ID, EmployeeID: x.EmployeeID, BaseSalary: money(x.BaseSalary),
		FixedAllowance: money(x.FixedAllowance), VariableAllowance: money(x.VariableAllowance),
		TransportAllowance: money(x.TransportAllowance), MealAllowance: money(x.MealAllowance),
		HousingAllowance: money(x.HousingAllowance), PtkpStatus: x.PTKPStatus, IsTaxable: x.IsTaxable,
		BpjsTkEnrolled: x.BpjsTkEnrolled, BpjsKesEnrolled: x.BpjsKesEnrolled, TaperaEnrolled: x.TaperaEnrolled,
		EffectiveDate: parseDay(x.EffectiveDate), Notes: x.Notes, CreatedBy: actor}))
}

func (s *Store) UpdateSalary(ctx context.Context, x domain.Salary, actor *string) (bool, error) {
	n, err := s.q.UpdateSalary(ctx, sqlc.UpdateSalaryParams{ID: x.ID, BaseSalary: money(x.BaseSalary),
		FixedAllowance: money(x.FixedAllowance), VariableAllowance: money(x.VariableAllowance),
		TransportAllowance: money(x.TransportAllowance), MealAllowance: money(x.MealAllowance),
		HousingAllowance: money(x.HousingAllowance), PtkpStatus: x.PTKPStatus, IsTaxable: x.IsTaxable,
		BpjsTkEnrolled: x.BpjsTkEnrolled, BpjsKesEnrolled: x.BpjsKesEnrolled, TaperaEnrolled: x.TaperaEnrolled,
		EffectiveDate: parseDay(x.EffectiveDate), Notes: x.Notes, UpdatedBy: actor})
	return n > 0, err
}

func (s *Store) DeactivateSalary(ctx context.Context, id, endDate string, actor *string) (bool, error) {
	end := parseDay(endDate)
	n, err := s.q.DeactivateSalary(ctx, sqlc.DeactivateSalaryParams{ID: id, EndDate: &end, UpdatedBy: actor})
	return n > 0, err
}

// ─── Runs and payslips ───────────────────────────────────────

func toRun(r sqlc.HrPayrollRun, approver string) domain.PayrollRun {
	return domain.PayrollRun{ID: r.ID, RunName: r.RunName, PeriodMonth: int(r.PeriodMonth), PeriodYear: int(r.PeriodYear),
		Status: r.Status, TotalEmployees: int(r.TotalEmployees), TotalGross: num(r.TotalGross), TotalDeductions: num(r.TotalDeductions),
		TotalNet: num(r.TotalNet), TotalBpjsEmployee: num(r.TotalBpjsEmployee), TotalBpjsEmployer: num(r.TotalBpjsEmployer),
		TotalPph21: num(r.TotalPph21), NeedsRecalculation: r.NeedsRecalculation, CalculatedAt: r.CalculatedAt,
		CalculatedBy: r.CalculatedBy, ApprovedAt: r.ApprovedAt, ApprovedBy: r.ApprovedBy, ApprovedByName: approver,
		PaidAt: r.PaidAt, PaidBy: r.PaidBy, Notes: r.Notes, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt}
}

func (s *Store) ListPayrollRuns(ctx context.Context, year int, status string) ([]domain.PayrollRun, error) {
	rows, err := s.q.ListPayrollRuns(ctx, sqlc.ListPayrollRunsParams{Year: i32(year), Status: status})
	if err != nil {
		return nil, err
	}
	out := make([]domain.PayrollRun, len(rows))
	for i, r := range rows {
		out[i] = toRun(r.HrPayrollRun, r.ApprovedByName)
	}
	return out, nil
}

func (s *Store) GetPayrollRun(ctx context.Context, id string) (*domain.PayrollRun, error) {
	r, err := one(s.q.GetPayrollRun(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	run := toRun(r.HrPayrollRun, r.ApprovedByName)
	return &run, nil
}

func (s *Store) GetPayrollRunByPeriod(ctx context.Context, month, year int) (*domain.PayrollRun, error) {
	r, err := one(s.q.GetPayrollRunByPeriod(ctx, sqlc.GetPayrollRunByPeriodParams{PeriodYear: i32(year), PeriodMonth: i32(month)}))
	if r == nil || err != nil {
		return nil, err
	}
	run := toRun(*r, "")
	return &run, nil
}

func (s *Store) InsertPayrollRun(ctx context.Context, r domain.PayrollRun) error {
	return wrap(s.q.InsertPayrollRun(ctx, sqlc.InsertPayrollRunParams{ID: r.ID, RunName: r.RunName, PeriodMonth: i32(r.PeriodMonth),
		PeriodYear: i32(r.PeriodYear), CreatedBy: r.CreatedBy}))
}

func (s *Store) DeletePayrollDetails(ctx context.Context, runID string) error {
	return s.q.DeletePayrollDetails(ctx, runID)
}

func (s *Store) InsertPayslip(ctx context.Context, p domain.Payslip) error {
	m := money
	return s.q.InsertPayrollDetail(ctx, sqlc.InsertPayrollDetailParams{
		ID: p.ID, PayrollRunID: p.RunID, EmployeeID: p.EmployeeID, BaseSalary: m(p.BaseSalary), FullBaseSalary: m(p.FullBaseSalary),
		FixedAllowance: m(p.FixedAllowance), VariableAllowance: m(p.VariableAllowance), TransportAllowance: m(p.TransportAllowance),
		MealAllowance: m(p.MealAllowance), HousingAllowance: m(p.HousingAllowance), OvertimePay: m(p.OvertimePay), Thr: m(p.Thr),
		Bonus: m(p.Bonus), GrossSalary: m(p.GrossSalary), BpjsTkJhtDeduction: m(p.BpjsTkJhtDeduction),
		BpjsTkJpDeduction: m(p.BpjsTkJpDeduction), BpjsKesDeduction: m(p.BpjsKesDeduction), TaperaDeduction: m(p.TaperaDeduction),
		Pph21Deduction: m(p.Pph21Deduction), UnpaidLeaveDeduction: m(p.UnpaidLeaveDeduction), LateDeduction: m(p.LateDeduction),
		LoanDeduction: m(p.LoanDeduction), LoanDetails: p.LoanDetails, TotalDeductions: m(p.TotalDeductions), NetSalary: m(p.NetSalary),
		BpjsTkJhtEmployer: m(p.BpjsTkJhtEmployer), BpjsTkJpEmployer: m(p.BpjsTkJpEmployer), BpjsTkJkkEmployer: m(p.BpjsTkJkkEmployer),
		BpjsTkJkmEmployer: m(p.BpjsTkJkmEmployer), BpjsKesEmployer: m(p.BpjsKesEmployer), TaperaEmployer: m(p.TaperaEmployer),
		TaxableIncome: m(p.TaxableIncome), PtkpAmount: m(p.PTKPAmount), Pph21Annual: m(p.Pph21Annual), WorkingDays: m(p.WorkingDays),
		PresentDays: m(p.PresentDays), LateDays: m(p.LateDays), UnpaidLeaveDays: m(p.UnpaidLeaveDays), OvertimeHours: m(p.OvertimeHours),
		ProrateFactor: fixed6(p.ProrateFactor),
	})
}

func fixed6(f float64) string { return strconv.FormatFloat(f, 'f', 6, 64) }

func (s *Store) SetRunCalculated(ctx context.Context, r domain.PayrollRun) error {
	return s.q.SetRunCalculated(ctx, sqlc.SetRunCalculatedParams{ID: r.ID, TotalEmployees: i32(r.TotalEmployees),
		TotalGross: money(r.TotalGross), TotalDeductions: money(r.TotalDeductions), TotalNet: money(r.TotalNet),
		TotalBpjsEmployee: money(r.TotalBpjsEmployee), TotalBpjsEmployer: money(r.TotalBpjsEmployer),
		TotalPph21: money(r.TotalPph21), CalculatedBy: r.CalculatedBy})
}

func (s *Store) ApproveRun(ctx context.Context, id string, by *string) (bool, error) {
	n, err := s.q.ApproveRun(ctx, sqlc.ApproveRunParams{ID: id, ApprovedBy: by})
	return n > 0, err
}

// MarkRunPaid flips an approved run to paid; nil when it was not approved.
func (s *Store) MarkRunPaid(ctx context.Context, id string, paidAt time.Time, by *string, notes string) (*domain.PayrollRun, error) {
	r, err := one(s.q.MarkRunPaid(ctx, sqlc.MarkRunPaidParams{ID: id, PaidAt: &paidAt, PaidBy: by, Notes: notes}))
	if r == nil || err != nil {
		return nil, err
	}
	run := toRun(*r, "")
	return &run, nil
}

func (s *Store) FlagRunRecalculation(ctx context.Context, id string) error {
	return s.q.FlagRunRecalculation(ctx, id)
}

func (s *Store) DeletePayrollRun(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeletePayrollRun(ctx, id)
	return n > 0, err
}

func toPayslip(p sqlc.HrPayrollDetail, month, year int32, status, name, no, dept, pos string) domain.Payslip {
	return domain.Payslip{
		ID: p.ID, RunID: p.PayrollRunID, PeriodMonth: int(month), PeriodYear: int(year), RunStatus: status,
		EmployeeID: p.EmployeeID, EmployeeName: name, EmployeeNo: no, DepartmentName: dept, PositionName: pos,
		BaseSalary: num(p.BaseSalary), FullBaseSalary: num(p.FullBaseSalary), FixedAllowance: num(p.FixedAllowance),
		VariableAllowance: num(p.VariableAllowance), TransportAllowance: num(p.TransportAllowance),
		MealAllowance: num(p.MealAllowance), HousingAllowance: num(p.HousingAllowance), OvertimePay: num(p.OvertimePay),
		Thr: num(p.Thr), Bonus: num(p.Bonus), GrossSalary: num(p.GrossSalary), BpjsTkJhtDeduction: num(p.BpjsTkJhtDeduction),
		BpjsTkJpDeduction: num(p.BpjsTkJpDeduction), BpjsKesDeduction: num(p.BpjsKesDeduction),
		TaperaDeduction: num(p.TaperaDeduction), Pph21Deduction: num(p.Pph21Deduction),
		UnpaidLeaveDeduction: num(p.UnpaidLeaveDeduction), LateDeduction: num(p.LateDeduction),
		LoanDeduction: num(p.LoanDeduction), LoanDetails: p.LoanDetails, TotalDeductions: num(p.TotalDeductions),
		NetSalary: num(p.NetSalary), BpjsTkJhtEmployer: num(p.BpjsTkJhtEmployer), BpjsTkJpEmployer: num(p.BpjsTkJpEmployer),
		BpjsTkJkkEmployer: num(p.BpjsTkJkkEmployer), BpjsTkJkmEmployer: num(p.BpjsTkJkmEmployer),
		BpjsKesEmployer: num(p.BpjsKesEmployer), TaperaEmployer: num(p.TaperaEmployer), TaxableIncome: num(p.TaxableIncome),
		PTKPAmount: num(p.PtkpAmount), Pph21Annual: num(p.Pph21Annual), WorkingDays: num(p.WorkingDays),
		PresentDays: num(p.PresentDays), LateDays: num(p.LateDays), UnpaidLeaveDays: num(p.UnpaidLeaveDays),
		OvertimeHours: num(p.OvertimeHours), ProrateFactor: num(p.ProrateFactor),
	}
}

func (s *Store) ListPayslips(ctx context.Context, f domain.PayslipFilter) ([]domain.Payslip, error) {
	rows, err := s.q.ListPayslips(ctx, sqlc.ListPayslipsParams{EmployeeID: f.EmployeeID, Year: i32(f.Year), RunID: f.RunID, OnlyPaid: f.OnlyPaid})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Payslip, len(rows))
	for i, r := range rows {
		out[i] = toPayslip(r.HrPayrollDetail, r.PeriodMonth, r.PeriodYear, r.RunStatus, r.FullName, r.EmployeeNo, r.DepartmentName, r.PositionName)
	}
	return out, nil
}

func (s *Store) GetPayslip(ctx context.Context, id string) (*domain.Payslip, error) {
	r, err := one(s.q.GetPayslip(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	p := toPayslip(r.HrPayrollDetail, r.PeriodMonth, r.PeriodYear, r.RunStatus, r.FullName, r.EmployeeNo, r.DepartmentName, r.PositionName)
	return &p, nil
}

// ─── Additions ───────────────────────────────────────────────

func toAddition(a sqlc.HrPayrollAddition, name, no, dept string) domain.PayrollAddition {
	return domain.PayrollAddition{ID: a.ID, EmployeeID: a.EmployeeID, EmployeeName: name, EmployeeNo: no, DepartmentName: dept,
		PeriodMonth: int(a.PeriodMonth), PeriodYear: int(a.PeriodYear), Amount: num(a.Amount), Label: a.Label,
		CreatedBy: a.CreatedBy, CreatedAt: a.CreatedAt}
}

func (s *Store) ListPayrollAdditions(ctx context.Context, month, year int, employeeID string) ([]domain.PayrollAddition, error) {
	rows, err := s.q.ListPayrollAdditions(ctx, sqlc.ListPayrollAdditionsParams{PeriodMonth: i32(month), PeriodYear: i32(year), EmployeeID: employeeID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.PayrollAddition, len(rows))
	for i, r := range rows {
		out[i] = toAddition(r.HrPayrollAddition, r.FullName, r.EmployeeNo, r.DepartmentName)
	}
	return out, nil
}

func (s *Store) GetPayrollAddition(ctx context.Context, id string) (*domain.PayrollAddition, error) {
	r, err := one(s.q.GetPayrollAddition(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	a := toAddition(*r, "", "", "")
	return &a, nil
}

func (s *Store) InsertPayrollAddition(ctx context.Context, a domain.PayrollAddition) error {
	return s.q.InsertPayrollAddition(ctx, sqlc.InsertPayrollAdditionParams{ID: a.ID, EmployeeID: a.EmployeeID, PeriodMonth: i32(a.PeriodMonth),
		PeriodYear: i32(a.PeriodYear), Amount: money(a.Amount), Label: a.Label, CreatedBy: a.CreatedBy})
}

func (s *Store) DeletePayrollAddition(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeletePayrollAddition(ctx, id)
	return n > 0, err
}

func (s *Store) SumPayrollAdditions(ctx context.Context, employeeID string, month, year int) (float64, error) {
	v, err := s.q.SumPayrollAdditions(ctx, sqlc.SumPayrollAdditionsParams{EmployeeID: employeeID, PeriodMonth: i32(month), PeriodYear: i32(year)})
	return num(v), err
}

// ─── Loans ───────────────────────────────────────────────────

func toLoan(l sqlc.HrLoan, name, no, dept, approver string) domain.Loan {
	return domain.Loan{ID: l.ID, EmployeeID: l.EmployeeID, EmployeeName: name, EmployeeNo: no, DepartmentName: dept,
		LoanType: l.LoanType, PrincipalAmount: num(l.PrincipalAmount), InterestRate: num(l.InterestRate),
		TenorMonths: int(l.TenorMonths), MonthlyInstallment: num(l.MonthlyInstallment), RemainingBalance: num(l.RemainingBalance),
		PaidAmount: num(l.PaidAmount), FirstInstallmentMonth: intPtr(l.FirstInstallmentMonth),
		FirstInstallmentYear: intPtr(l.FirstInstallmentYear), Status: l.Status, IsActive: l.IsActive, Purpose: l.Purpose,
		ApprovedBy: l.ApprovedBy, ApprovedByName: approver, ApprovedAt: l.ApprovedAt, RejectedBy: l.RejectedBy,
		RejectionReason: l.RejectionReason, CreatedBy: l.CreatedBy, CreatedAt: l.CreatedAt}
}

func (s *Store) ListLoans(ctx context.Context, employeeID, status string) ([]domain.Loan, error) {
	rows, err := s.q.ListLoans(ctx, sqlc.ListLoansParams{EmployeeID: employeeID, Status: status})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Loan, len(rows))
	for i, r := range rows {
		out[i] = toLoan(r.HrLoan, r.FullName, r.EmployeeNo, r.DepartmentName, r.ApprovedByName)
	}
	return out, nil
}

func (s *Store) GetLoan(ctx context.Context, id string) (*domain.Loan, error) {
	r, err := one(s.q.GetLoan(ctx, id))
	if r == nil || err != nil {
		return nil, err
	}
	l := toLoan(r.HrLoan, r.FullName, r.EmployeeNo, r.DepartmentName, r.ApprovedByName)
	return &l, nil
}

func (s *Store) CountOpenLoans(ctx context.Context, employeeID string, includePending bool) (int, error) {
	n, err := s.q.CountOpenLoans(ctx, sqlc.CountOpenLoansParams{EmployeeID: employeeID, IncludePending: includePending})
	return int(n), err
}

func (s *Store) InsertLoan(ctx context.Context, l domain.Loan) error {
	return s.q.InsertLoan(ctx, sqlc.InsertLoanParams{ID: l.ID, EmployeeID: l.EmployeeID, LoanType: l.LoanType,
		PrincipalAmount: money(l.PrincipalAmount), InterestRate: money(l.InterestRate), TenorMonths: i32(l.TenorMonths),
		MonthlyInstallment: money(l.MonthlyInstallment), RemainingBalance: money(l.RemainingBalance), Purpose: l.Purpose, CreatedBy: l.CreatedBy})
}

func (s *Store) ApproveLoan(ctx context.Context, id string, by *string, at time.Time, month, year int) (bool, error) {
	n, err := s.q.ApproveLoan(ctx, sqlc.ApproveLoanParams{ID: id, ApprovedBy: by, ApprovedAt: &at, FirstInstallmentMonth: i32Ptr(&month), FirstInstallmentYear: i32Ptr(&year)})
	return n > 0, err
}

func (s *Store) RejectLoan(ctx context.Context, id string, by *string, reason string) (bool, error) {
	n, err := s.q.RejectLoan(ctx, sqlc.RejectLoanParams{ID: id, RejectedBy: by, RejectionReason: reason})
	return n > 0, err
}

// CancelLoan withdraws a pending loan; false when it is no longer pending.
func (s *Store) CancelLoan(ctx context.Context, id string) (bool, error) {
	n, err := s.q.CancelLoan(ctx, id)
	return n > 0, err
}

// OpenLoans lists the employee's approved loans with a balance, oldest
// first; lock takes row locks for the settlement transaction.
func (s *Store) OpenLoans(ctx context.Context, employeeID string, lock bool) ([]domain.Loan, error) {
	var rows []sqlc.HrLoan
	var err error
	if lock {
		rows, err = s.q.LockOpenLoans(ctx, employeeID)
	} else {
		rows, err = s.q.OpenLoans(ctx, employeeID)
	}
	if err != nil {
		return nil, err
	}
	out := make([]domain.Loan, len(rows))
	for i, r := range rows {
		out[i] = toLoan(r, "", "", "", "")
	}
	return out, nil
}

func (s *Store) ApplyLoanPayment(ctx context.Context, id string, paid, remaining float64) error {
	return s.q.ApplyLoanPayment(ctx, sqlc.ApplyLoanPaymentParams{ID: id, Paid: money(paid), Remaining: money(remaining)})
}

// OvertimeForPeriod is the approved overtime of one employee in a range.
func (s *Store) OvertimeForPeriod(ctx context.Context, employeeID, from, to string) ([]domain.Overtime, error) {
	rows, err := s.q.OvertimeForPeriod(ctx, sqlc.OvertimeForPeriodParams{EmployeeID: employeeID, DateFrom: parseDay(from), DateTo: parseDay(to)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Overtime, len(rows))
	for i, r := range rows {
		out[i] = domain.Overtime{EmployeeID: r.EmployeeID, Date: day(r.Date), Hours: num(r.Hours)}
	}
	return out, nil
}
