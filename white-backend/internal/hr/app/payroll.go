package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
)

// Payroll is salaries, payroll settings, runs, payslips and additions
// (NüHabit payroll: salary.go, settings.go, service_runs.go, additions.go,
// payslip_pdf.go). Calculation uses the ported domain/payroll calculator.
type Payroll struct {
	d    Deps
	docs Documents
}

func NewPayroll(d Deps, docs Documents) *Payroll { return &Payroll{d: d, docs: docs} }

// PayrollPaidEvent is the white.hr.payroll.paid payload, in the shape
// finance decodes (internal/finance/domain.PayrollPaid). Amounts are
// decimal strings in rupiah with two decimals.
type PayrollPaidEvent struct {
	// RunID is the hr.payroll_runs id; finance keys its journals on it so a
	// redelivery posts nothing twice.
	RunID string `json:"runId"`
	// PeriodYear and PeriodMonth are the payroll period (1..12).
	PeriodYear  int `json:"periodYear"`
	PeriodMonth int `json:"periodMonth"`
	// Label is the run name, e.g. "Payroll Oktober 2026".
	Label string `json:"label"`
	// PaidAt is when the run was marked paid (UTC).
	PaidAt time.Time `json:"paidAt"`
	// PaidBy is the identity user who marked it paid.
	PaidBy string `json:"paidBy,omitempty"`
	// EmployeeCount is the number of payslips in the run.
	EmployeeCount int `json:"employeeCount"`
	// TotalGross is gross pay less unpaid-leave and late deductions, so
	// TotalGross = TotalNet + TotalPph21 + TotalLoanDeduction +
	// TotalBpjsTkEmployee + TotalBpjsKesEmployee + TotalTaperaEmployee.
	TotalGross string `json:"totalGross"`
	// TotalNet is take-home pay transferred to employees.
	TotalNet string `json:"totalNet"`
	// TotalPph21 is PPh 21 withheld.
	TotalPph21 string `json:"totalPph21"`
	// TotalLoanDeduction is the loan installments settled by this run.
	TotalLoanDeduction string `json:"totalLoanDeduction"`
	// TotalBpjsTkEmployee is the employee JHT + JP deduction.
	TotalBpjsTkEmployee string `json:"totalBpjsTkEmployee"`
	// TotalBpjsKesEmployee is the employee BPJS Kesehatan deduction.
	TotalBpjsKesEmployee string `json:"totalBpjsKesEmployee"`
	// TotalTaperaEmployee is the employee Tapera deduction.
	TotalTaperaEmployee string `json:"totalTaperaEmployee"`
	// TotalBpjsTkEmployer is the employer JHT + JP + JKK + JKM share.
	TotalBpjsTkEmployer string `json:"totalBpjsTkEmployer"`
	// TotalBpjsKesEmployer is the employer BPJS Kesehatan share.
	TotalBpjsKesEmployer string `json:"totalBpjsKesEmployer"`
	// TotalTaperaEmployer is the employer Tapera share.
	TotalTaperaEmployer string `json:"totalTaperaEmployer"`
}

// ─── Settings ────────────────────────────────────────────────

func (p *Payroll) Settings(ctx context.Context, a Actor) (*domain.PayrollSettings, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	return p.d.Store.GetPayrollSettings(ctx)
}

func (p *Payroll) UpdateSettings(ctx context.Context, a Actor, s domain.PayrollSettings) (*domain.PayrollSettings, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	if !slices.Contains([]string{"off", "per_minute", "flat"}, s.LateDeductionMode) {
		return nil, invalid("lateDeductionMode", "Mode potongan keterlambatan tidak valid.")
	}
	for name, rate := range map[string]float64{"bpjsTkJhtEmployee": s.BpjsTkJhtEmployee, "bpjsTkJhtEmployer": s.BpjsTkJhtEmployer,
		"bpjsTkJpEmployee": s.BpjsTkJpEmployee, "bpjsTkJpEmployer": s.BpjsTkJpEmployer, "bpjsTkJkk": s.BpjsTkJkk, "bpjsTkJkm": s.BpjsTkJkm,
		"bpjsKesEmployee": s.BpjsKesEmployee, "bpjsKesEmployer": s.BpjsKesEmployer, "taperaEmployee": s.TaperaEmployee,
		"taperaEmployer": s.TaperaEmployer, "loanMaxInstallmentPercent": s.LoanMaxInstallmentPercent} {
		if rate < 0 || rate > 100 {
			return nil, invalid(name, "Persentase harus 0 sampai 100.")
		}
	}
	for i := 1; i < 4; i++ {
		if s.BracketLimits[i] <= s.BracketLimits[i-1] {
			return nil, invalid("pph21BracketLimits", "Batas lapisan PPh 21 harus naik.")
		}
	}
	if s.OvertimeHourlyDivisor <= 0 || s.LoanMaxActivePerEmployee < 0 || s.PayrollDay < 1 || s.PayrollDay > 31 {
		return nil, invalid("overtimeHourlyDivisor", "Nilai pengaturan payroll tidak valid.")
	}
	if err := p.d.Store.UpdatePayrollSettings(ctx, s, a.UserRef()); err != nil {
		return nil, err
	}
	return p.d.Store.GetPayrollSettings(ctx)
}

// configOf is loadPayrollConfig over the settings row.
func configOf(s *domain.PayrollSettings) payroll.Config {
	row := payroll.Row{
		"bpjs_tk_jht_employee": s.BpjsTkJhtEmployee, "bpjs_tk_jht_employer": s.BpjsTkJhtEmployer,
		"bpjs_tk_jp_employee": s.BpjsTkJpEmployee, "bpjs_tk_jp_employer": s.BpjsTkJpEmployer,
		"bpjs_tk_jkk": s.BpjsTkJkk, "bpjs_tk_jkm": s.BpjsTkJkm, "bpjs_kes_employee": s.BpjsKesEmployee,
		"bpjs_kes_employer": s.BpjsKesEmployer, "bpjs_kes_max_upah": s.BpjsKesMaxUpah,
		"tapera_employee": s.TaperaEmployee, "tapera_employer": s.TaperaEmployer,
		"pph21_bracket_1": s.BracketLimits[0], "pph21_bracket_2": s.BracketLimits[1],
		"pph21_bracket_3": s.BracketLimits[2], "pph21_bracket_4": s.BracketLimits[3],
		"overtime_multiplier": s.OvertimeMultiplier, "overtime_multiplier_holiday": s.OvertimeMultiplierHoliday,
		"overtime_hourly_divisor": s.OvertimeHourlyDivisor, "late_deduction_mode": s.LateDeductionMode,
		"late_deduction_amount": s.LateDeductionAmount, "loan_max_installment_percent": s.LoanMaxInstallmentPercent,
		"loan_max_active_per_employee": float64(s.LoanMaxActivePerEmployee),
	}
	for _, st := range domain.PTKPStatuses {
		row["ptkp_"+strings.ToLower(strings.ReplaceAll(st, "/", "_"))] = s.PTKP[st]
	}
	return payroll.BuildConfig(row, nil)
}

// ─── Salaries ────────────────────────────────────────────────

type SalaryInput struct {
	EmployeeID, BaseSalary, FixedAllowance, VariableAllowance  string
	TransportAllowance, MealAllowance, HousingAllowance        string
	PTKPStatus, EffectiveDate, Notes                           string
	IsTaxable, BpjsTkEnrolled, BpjsKesEnrolled, TaperaEnrolled *bool
}

func (p *Payroll) Salaries(ctx context.Context, a Actor, employeeID string, activeOnly bool) ([]domain.Salary, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	return p.d.Store.ListSalaries(ctx, employeeID, activeOnly)
}

// salaryOf validates in onto base (the current version on update).
func salaryOf(in SalaryInput, base domain.Salary) (domain.Salary, error) {
	s := base
	var err error
	if s.BaseSalary, err = amount("baseSalary", in.BaseSalary); err != nil {
		return s, err
	}
	if s.BaseSalary <= 0 {
		return s, invalid("baseSalary", "Gaji pokok wajib diisi.")
	}
	for _, f := range []struct {
		field, raw string
		dst        *float64
	}{{"fixedAllowance", in.FixedAllowance, &s.FixedAllowance}, {"variableAllowance", in.VariableAllowance, &s.VariableAllowance},
		{"transportAllowance", in.TransportAllowance, &s.TransportAllowance}, {"mealAllowance", in.MealAllowance, &s.MealAllowance},
		{"housingAllowance", in.HousingAllowance, &s.HousingAllowance}} {
		if *f.dst, err = amount(f.field, f.raw); err != nil {
			return s, err
		}
	}
	s.PTKPStatus = strings.ToUpper(strings.TrimSpace(in.PTKPStatus))
	if s.PTKPStatus == "" {
		s.PTKPStatus = "TK/0"
	}
	if !slices.Contains(domain.PTKPStatuses, s.PTKPStatus) {
		return s, invalid("ptkpStatus", "Status PTKP tidak valid.")
	}
	for _, f := range []struct {
		src *bool
		dst *bool
	}{{in.IsTaxable, &s.IsTaxable}, {in.BpjsTkEnrolled, &s.BpjsTkEnrolled}, {in.BpjsKesEnrolled, &s.BpjsKesEnrolled}, {in.TaperaEnrolled, &s.TaperaEnrolled}} {
		if f.src != nil {
			*f.dst = *f.src
		}
	}
	s.Notes = strings.TrimSpace(in.Notes)
	return s, nil
}

func (p *Payroll) CreateSalary(ctx context.Context, a Actor, in SalaryInput) (*domain.Salary, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	emp, err := p.d.Store.GetEmployee(ctx, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, notFound(msgEmployeeNotFound)
	}
	s, err := salaryOf(in, domain.Salary{IsTaxable: true, BpjsTkEnrolled: true, BpjsKesEnrolled: true, TaperaEnrolled: true})
	if err != nil {
		return nil, err
	}
	s.EffectiveDate = p.d.today()
	if d, err := optDate("effectiveDate", in.EffectiveDate); err != nil {
		return nil, err
	} else if d != nil {
		s.EffectiveDate = *d
	}
	s.ID, s.EmployeeID = newID(), emp.ID
	err = p.d.Store.WithTx(ctx, func(tx domain.Store) error {
		return tx.InsertSalary(ctx, s, hris.AddDaysISO(s.EffectiveDate, -1), a.UserRef())
	})
	if err != nil {
		return nil, err
	}
	return p.d.Store.GetSalary(ctx, s.ID)
}

func (p *Payroll) UpdateSalary(ctx context.Context, a Actor, id string, in SalaryInput) (*domain.Salary, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	cur, err := p.d.Store.GetSalary(ctx, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, notFound("Data gaji tidak ditemukan.")
	}
	s, err := salaryOf(in, *cur)
	if err != nil {
		return nil, err
	}
	if d, err := optDate("effectiveDate", in.EffectiveDate); err != nil {
		return nil, err
	} else if d != nil {
		s.EffectiveDate = *d
	}
	if _, err := p.d.Store.UpdateSalary(ctx, s, a.UserRef()); err != nil {
		return nil, err
	}
	return p.d.Store.GetSalary(ctx, id)
}

func (p *Payroll) DeactivateSalary(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermPayroll); err != nil {
		return err
	}
	ok, err := p.d.Store.DeactivateSalary(ctx, id, p.d.today(), a.UserRef())
	if err == nil && !ok {
		return notFound("Data gaji aktif tidak ditemukan.")
	}
	return err
}

// ─── Runs ────────────────────────────────────────────────────

func (p *Payroll) Runs(ctx context.Context, a Actor, year int, status string) ([]domain.PayrollRun, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	return p.d.Store.ListPayrollRuns(ctx, year, status)
}

func (p *Payroll) CreateRun(ctx context.Context, a Actor, month, year int, name string) (*domain.PayrollRun, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	if month < 1 || month > 12 || year < 2000 || year > 2100 {
		return nil, invalid("periodMonth", "Periode payroll tidak valid.")
	}
	if name = strings.TrimSpace(name); name == "" {
		name = "Payroll " + payroll.PeriodLabelID(&month, year)
	}
	run := domain.PayrollRun{ID: newID(), RunName: name, PeriodMonth: month, PeriodYear: year, CreatedBy: a.UserRef()}
	if err := p.d.Store.InsertPayrollRun(ctx, run); err != nil {
		return nil, uniqueErr(err, "Payroll untuk periode ini sudah ada.")
	}
	return p.d.Store.GetPayrollRun(ctx, run.ID)
}

func (p *Payroll) Run(ctx context.Context, a Actor, id string) (*domain.PayrollRun, []domain.Payslip, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, nil, err
	}
	run, err := p.run(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	slips, err := p.d.Store.ListPayslips(ctx, domain.PayslipFilter{RunID: id})
	return run, slips, err
}

func (p *Payroll) run(ctx context.Context, id string) (*domain.PayrollRun, error) {
	run, err := p.d.Store.GetPayrollRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, notFound("Payroll run tidak ditemukan.")
	}
	return run, nil
}

type Skipped struct{ EmployeeID, FullName, Reason string }

// Calculate is calculateRun: every active employee with an active salary
// gets a payslip; old slips are replaced and the run totals written.
func (p *Payroll) Calculate(ctx context.Context, a Actor, id string, includeThr bool) (*domain.PayrollRun, []Skipped, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, nil, err
	}
	run, err := p.run(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if !payroll.CanCalculateRun(run.Status) {
		return nil, nil, conflict("Payroll yang sudah disetujui atau dibayar tidak bisa dihitung ulang.")
	}
	settings, err := p.d.Store.GetPayrollSettings(ctx)
	if err != nil {
		return nil, nil, err
	}
	config := configOf(settings)
	start, end := payroll.PeriodBounds(run.PeriodMonth, run.PeriodYear)
	all, _, err := p.d.Store.ListEmployees(ctx, domain.EmployeeFilter{IncludeInactive: true})
	if err != nil {
		return nil, nil, err
	}
	emps := employedDuring(all, start, end)
	if len(emps) == 0 {
		return nil, nil, invalid("runId", "Tidak ada karyawan aktif ditemukan.")
	}
	holidays, err := p.holidaySet(ctx, start, end)
	if err != nil {
		return nil, nil, err
	}
	var slips []domain.Payslip
	var skipped []Skipped
	var results []payroll.Result
	for _, e := range emps {
		in, err := p.input(ctx, e, run.PeriodMonth, run.PeriodYear, includeThr, holidays)
		if err != nil {
			return nil, nil, err
		}
		if in == nil {
			skipped = append(skipped, Skipped{e.ID, e.FullName, "Belum ada struktur gaji aktif"})
			continue
		}
		r := payroll.Calculate(*in, config)
		slip, err := payslipOf(run.ID, e.ID, *in, r)
		if err != nil {
			return nil, nil, err
		}
		slips = append(slips, slip)
		results = append(results, r)
	}
	totals := payroll.SumRunTotals(results)
	run.TotalEmployees, run.TotalGross, run.TotalDeductions = len(slips), totals.TotalGross, totals.TotalDeductions
	run.TotalNet, run.TotalBpjsEmployee, run.TotalBpjsEmployer = totals.TotalNet, totals.TotalBjtkEmployee, totals.TotalBjtkEmployer
	run.TotalPph21, run.CalculatedBy = totals.TotalPph21, a.UserRef()
	err = p.d.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.DeletePayrollDetails(ctx, run.ID); err != nil {
			return err
		}
		for _, s := range slips {
			if err := tx.InsertPayslip(ctx, s); err != nil {
				return err
			}
		}
		return tx.SetRunCalculated(ctx, *run)
	})
	if err != nil {
		return nil, nil, err
	}
	out, err := p.run(ctx, id)
	return out, skipped, err
}

// employedDuring keeps the employees on the books for part of [start, end]:
// joined by the last day, and either active or ended on or after the first
// day (a mid-month leaver still gets the prorated final slip).
func employedDuring(emps []domain.Employee, start, end string) []domain.Employee {
	out := make([]domain.Employee, 0, len(emps))
	for _, e := range emps {
		if e.JoinDate > end {
			continue
		}
		if e.EndDate != nil && *e.EndDate < start {
			continue
		}
		if !e.IsActive && e.EndDate == nil {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (p *Payroll) holidaySet(ctx context.Context, start, end string) (map[string]bool, error) {
	hs, err := p.d.Store.ListHolidays(ctx, start, end, true)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, h := range hs {
		if !h.DeductsLeave {
			out[h.Date] = true
		}
	}
	return out, nil
}

// input loads one employee's period records and builds the calculator
// input (loadEmployeePayrollInput); nil without an active salary.
func (p *Payroll) input(ctx context.Context, e domain.Employee, month, year int, includeThr bool, holidays map[string]bool) (*payroll.Input, error) {
	sal, err := p.d.Store.ActiveSalary(ctx, e.ID)
	if err != nil || sal == nil {
		return nil, err
	}
	start, end := payroll.PeriodBounds(month, year)
	atts, _, err := p.d.Store.ListAttendance(ctx, domain.AttendanceFilter{EmployeeIDs: []string{e.ID}, From: start, To: end})
	if err != nil {
		return nil, err
	}
	sched, err := p.d.Store.ListScheduleRows(ctx, []string{e.ID})
	if err != nil {
		return nil, err
	}
	ots, err := p.d.Store.OvertimeForPeriod(ctx, e.ID, start, end)
	if err != nil {
		return nil, err
	}
	leaves, _, err := p.d.Store.ListLeaves(ctx, domain.LeaveFilter{EmployeeIDs: []string{e.ID}, Status: hris.LeaveApproved, From: start, To: end})
	if err != nil {
		return nil, err
	}
	contracts, err := p.d.Store.ListContracts(ctx, domain.ContractFilter{EmployeeID: e.ID})
	if err != nil {
		return nil, err
	}
	loans, err := p.d.Store.OpenLoans(ctx, e.ID, false)
	if err != nil {
		return nil, err
	}
	additions, err := p.d.Store.SumPayrollAdditions(ctx, e.ID, month, year)
	if err != nil {
		return nil, err
	}
	var rec payroll.WorkforceRecords
	for _, at := range atts {
		late, mins, ot := at.IsLate, any(float64(at.LateMinutes)), any(at.OvertimeHours)
		rec.Attendance = append(rec.Attendance, payroll.AttendanceRow{Date: at.Date, Status: at.Status, HasClockOut: at.ClockOut != nil,
			IsLate: &late, LateMinutes: mins, OvertimeHours: ot})
	}
	for _, r := range sched {
		rec.Schedule = append(rec.Schedule, payroll.ShiftRow{DayOfWeek: r.DayOfWeek, ShiftID: r.ShiftID, EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo})
	}
	for _, o := range ots {
		rec.Overtime = append(rec.Overtime, payroll.OvertimeRequest{Date: o.Date, Hours: o.Hours})
	}
	for _, l := range leaves {
		s, en := l.StartDate, l.EndDate
		rec.Leaves = append(rec.Leaves, payroll.Leave{StartDate: &s, EndDate: &en, LeaveType: l.LeaveType})
	}
	slices.SortFunc(contracts, func(x, y domain.Contract) int { return strings.Compare(y.StartDate, x.StartDate) })
	for _, c := range contracts {
		if c.Status == "draft" {
			continue
		}
		typ, s := c.ContractType, c.StartDate
		rec.Contracts = append(rec.Contracts, payroll.Contract{ContractType: &typ, StartDate: &s, EndDate: c.EndDate})
	}
	loanRows := make([]payroll.LoanRow, len(loans))
	for i, l := range loans {
		loanRows[i] = loanRowOf(l)
	}
	salary := payroll.Salary{BaseSalary: sal.BaseSalary, FixedAllowance: sal.FixedAllowance, VariableAllowance: sal.VariableAllowance,
		TransportAllowance: sal.TransportAllowance, MealAllowance: sal.MealAllowance, HousingAllowance: sal.HousingAllowance,
		PTKPStatus: &sal.PTKPStatus, IsTaxable: &sal.IsTaxable, BpjsTkEnrolled: &sal.BpjsTkEnrolled,
		BpjsKesEnrolled: &sal.BpjsKesEnrolled, TaperaEnrl: &sal.TaperaEnrolled}
	in := payroll.BuildInput(payroll.Employee{ID: e.ID, EmploymentStatus: e.EmploymentStatus, JoinDate: e.JoinDate}, salary, rec, loanRows, holidays, month, year, includeThr)
	in.Bonus, in.IrregularIncome = additions, additions
	return &in, nil
}

func loanRowOf(l domain.Loan) payroll.LoanRow {
	typ, tenor := l.LoanType, l.TenorMonths
	return payroll.LoanRow{ID: l.ID, MonthlyInstallment: l.MonthlyInstallment, RemainingBalance: l.RemainingBalance,
		FirstInstallmentMonth: l.FirstInstallmentMonth, FirstInstallmentYear: l.FirstInstallmentYear, Status: l.Status,
		IsActive: l.IsActive, PrincipalAmount: l.PrincipalAmount, TenorMonths: &tenor, LoanType: &typ}
}

func payslipOf(runID, employeeID string, in payroll.Input, r payroll.Result) (domain.Payslip, error) {
	breakdown := in.LoanBreakdown
	if breakdown == nil {
		breakdown = []payroll.LoanInstallmentDetail{}
	}
	details, err := json.Marshal(breakdown)
	if err != nil {
		return domain.Payslip{}, err
	}
	factor := 1.0
	if in.ProrateFactor != nil {
		factor = *in.ProrateFactor
	}
	return domain.Payslip{ID: newID(), RunID: runID, EmployeeID: employeeID,
		BaseSalary: r.BaseSalary, FullBaseSalary: in.BaseSalary, FixedAllowance: r.FixedAllowance, VariableAllowance: r.VariableAllowance,
		TransportAllowance: r.TransportAllowance, MealAllowance: r.MealAllowance, HousingAllowance: r.HousingAllowance,
		OvertimePay: r.OvertimePay, Thr: r.Thr, Bonus: r.Bonus, GrossSalary: r.GrossSalary,
		BpjsTkJhtDeduction: r.BpjsTkJhtDeduction, BpjsTkJpDeduction: r.BpjsTkJpDeduction, BpjsKesDeduction: r.BpjsKesDeduction,
		TaperaDeduction: r.TaperaDeduction, Pph21Deduction: r.Pph21Deduction, UnpaidLeaveDeduction: r.UnpaidLeaveDeduction,
		LateDeduction: r.LateDeduction, LoanDeduction: r.LoanDeduction, LoanDetails: details, TotalDeductions: r.TotalDeductions,
		NetSalary: r.NetSalary, BpjsTkJhtEmployer: r.BpjsTkJhtEmployer, BpjsTkJpEmployer: r.BpjsTkJpEmployer,
		BpjsTkJkkEmployer: r.BpjsTkJkkEmployer, BpjsTkJkmEmployer: r.BpjsTkJkmEmployer, BpjsKesEmployer: r.BpjsKesEmployer,
		TaperaEmployer: r.TaperaEmployer, TaxableIncome: r.TaxableIncome, PTKPAmount: r.PTKPAmount, Pph21Annual: r.Pph21Annual,
		WorkingDays: in.WorkingDays, PresentDays: in.PresentDays, LateDays: in.LateDays, UnpaidLeaveDays: in.UnpaidLeaveDays,
		OvertimeHours: in.OvertimeHours + in.OvertimeHolidayHrs, ProrateFactor: factor}, nil
}

// resultOf rebuilds the calculator result a stored payslip came from.
func resultOf(s domain.Payslip) payroll.Result {
	return payroll.Result{GrossSalary: s.GrossSalary, NetSalary: s.NetSalary, Pph21Deduction: s.Pph21Deduction,
		LoanDeduction: s.LoanDeduction, UnpaidLeaveDeduction: s.UnpaidLeaveDeduction, LateDeduction: s.LateDeduction,
		BpjsTkJhtDeduction: s.BpjsTkJhtDeduction, BpjsTkJpDeduction: s.BpjsTkJpDeduction, BpjsKesDeduction: s.BpjsKesDeduction,
		TaperaDeduction: s.TaperaDeduction, BpjsTkJhtEmployer: s.BpjsTkJhtEmployer, BpjsTkJpEmployer: s.BpjsTkJpEmployer,
		BpjsTkJkkEmployer: s.BpjsTkJkkEmployer, BpjsTkJkmEmployer: s.BpjsTkJkmEmployer, BpjsKesEmployer: s.BpjsKesEmployer,
		TaperaEmployer: s.TaperaEmployer}
}

func (p *Payroll) Approve(ctx context.Context, a Actor, id string) (*domain.PayrollRun, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	run, err := p.run(ctx, id)
	if err != nil {
		return nil, err
	}
	if run.NeedsRecalculation {
		return nil, conflict("Ada perubahan tambahan payroll: hitung ulang dulu sebelum menyetujui.")
	}
	if !payroll.CanTransitionRun(run.Status, payroll.RunApproved) {
		return nil, conflict(fmt.Sprintf("Transisi status '%s' ke '%s' tidak diizinkan.", run.Status, payroll.RunApproved))
	}
	ok, err := p.d.Store.ApproveRun(ctx, id, a.UserRef())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, conflict("Run sudah diproses/berubah status, muat ulang halaman.")
	}
	return p.run(ctx, id)
}

// MarkPaid is markRunPaid: the run turns paid, the loan installments on
// its slips are settled oldest loan first, and white.hr.payroll.paid is
// published, all in one transaction so a failed publish leaves the run
// approved and the action can be retried.
func (p *Payroll) MarkPaid(ctx context.Context, a Actor, id, notes string) (*domain.PayrollRun, int, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, 0, err
	}
	run, err := p.run(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	if !payroll.CanTransitionRun(run.Status, payroll.RunPaid) {
		return nil, 0, conflict(fmt.Sprintf("Transisi status '%s' ke '%s' tidak diizinkan.", run.Status, payroll.RunPaid))
	}
	slips, err := p.d.Store.ListPayslips(ctx, domain.PayslipFilter{RunID: id})
	if err != nil {
		return nil, 0, err
	}
	settled := 0
	err = p.d.Store.WithTx(ctx, func(tx domain.Store) error {
		paid, err := tx.MarkRunPaid(ctx, id, p.d.now(), a.UserRef(), strings.TrimSpace(notes))
		if err != nil {
			return err
		}
		if paid == nil {
			return conflict("Run sudah diproses/berubah status, muat ulang halaman.")
		}
		shares := make([]payroll.SlipShare, 0, len(slips))
		for _, s := range slips {
			share := payroll.ShareOf(resultOf(s))
			if s.LoanDeduction > 0 {
				n, err := settleLoans(ctx, tx, s, run.PeriodMonth, run.PeriodYear)
				if err != nil {
					return err
				}
				settled += n
			}
			shares = append(shares, share)
		}
		t := payroll.SumSlipShares(shares)
		m := domain.Money
		return p.d.Events.Publish(ctx, events.HRPayrollPaid, PayrollPaidEvent{
			RunID: id, PeriodYear: run.PeriodYear, PeriodMonth: run.PeriodMonth, Label: run.RunName, PaidAt: *paid.PaidAt,
			PaidBy: a.UserID, EmployeeCount: len(slips), TotalGross: m(t.Gross), TotalNet: m(t.Net), TotalPph21: m(t.Pph21),
			TotalLoanDeduction: m(t.Loan), TotalBpjsTkEmployee: m(t.BpjsTkEmployee), TotalBpjsKesEmployee: m(t.BpjsKesEmployee),
			TotalTaperaEmployee: m(t.TaperaEmployee), TotalBpjsTkEmployer: m(t.BpjsTkEmployer),
			TotalBpjsKesEmployer: m(t.BpjsKesEmployer), TotalTaperaEmployer: m(t.TaperaEmployer),
		})
	})
	if err != nil {
		return nil, 0, err
	}
	out, err := p.run(ctx, id)
	return out, settled, err
}

// settleLoans allocates one slip's loan deduction over the employee's due
// loans; a shortfall (another run settled them first) aborts.
func settleLoans(ctx context.Context, tx domain.Store, s domain.Payslip, month, year int) (int, error) {
	loans, err := tx.OpenLoans(ctx, s.EmployeeID, true)
	if err != nil {
		return 0, err
	}
	var due []payroll.LoanRow
	for _, l := range loans {
		if r := loanRowOf(l); payroll.IsLoanDue(r, month, year) {
			due = append(due, r)
		}
	}
	deducted := payroll.Round(s.LoanDeduction)
	allocs := payroll.AllocateLoanPayment(due, deducted)
	total := 0.0
	for _, al := range allocs {
		total += al.Amount
	}
	if total < deducted {
		return 0, conflict("Cicilan pinjaman di slip " + s.EmployeeName + " tidak lagi cocok dengan saldo pinjaman saat ini " +
			"(kemungkinan run periode lain ditandai dibayar lebih dulu). Hitung ulang run ini sebelum menandai dibayar.")
	}
	for _, al := range allocs {
		if err := tx.ApplyLoanPayment(ctx, al.LoanID, al.Amount, al.NewRemaining); err != nil {
			return 0, err
		}
	}
	return len(allocs), nil
}

func (p *Payroll) DeleteRun(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermPayroll); err != nil {
		return err
	}
	run, err := p.run(ctx, id)
	if err != nil {
		return err
	}
	if !payroll.CanDeleteRun(run.Status) {
		return conflict("Payroll yang sudah dibayar tidak bisa dihapus.")
	}
	_, err = p.d.Store.DeletePayrollRun(ctx, id)
	return err
}

// ─── Payslips ────────────────────────────────────────────────

// Payslips: hr.payroll sees every slip; everyone else only their own slips
// of paid runs (resolvePayslipScope).
func (p *Payroll) Payslips(ctx context.Context, a Actor, f domain.PayslipFilter) ([]domain.Payslip, error) {
	if !a.Can(authn.PermPayroll) || (f.EmployeeID != "" && f.EmployeeID == a.EmployeeID) {
		id, err := self(a)
		if err != nil {
			return nil, err
		}
		f.EmployeeID, f.OnlyPaid = id, true
	}
	return p.d.Store.ListPayslips(ctx, f)
}

// PayslipPDF renders a slip from its stored snapshot.
func (p *Payroll) PayslipPDF(ctx context.Context, a Actor, id string) ([]byte, string, error) {
	s, err := p.d.Store.GetPayslip(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if s == nil {
		return nil, "", notFound("Slip gaji tidak ditemukan.")
	}
	if !a.Can(authn.PermPayroll) {
		if s.EmployeeID != a.EmployeeID || s.RunStatus != payroll.RunPaid {
			return nil, "", notFound("Slip gaji tidak ditemukan.")
		}
	}
	var details []payroll.LoanInstallmentDetail
	_ = json.Unmarshal(s.LoanDetails, &details)
	data, err := p.docs.PayslipPDF(*s, details)
	return data, payroll.PayslipFileName(s.EmployeeName, s.PeriodMonth, s.PeriodYear), err
}

// ─── Additions ───────────────────────────────────────────────

type AdditionInput struct {
	EmployeeID, Amount, Label string
	Month, Year               int
}

func (p *Payroll) Additions(ctx context.Context, a Actor, month, year int, employeeID string) ([]domain.PayrollAddition, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	return p.d.Store.ListPayrollAdditions(ctx, month, year, employeeID)
}

// CreateAddition is AddToOpenPeriod: without an explicit period it lands
// in the first month (from this one) whose run is missing or not yet
// approved; a calculated run is flagged for recalculation.
func (p *Payroll) CreateAddition(ctx context.Context, a Actor, in AdditionInput) (*domain.PayrollAddition, error) {
	if err := require(a, authn.PermPayroll); err != nil {
		return nil, err
	}
	amt, err := amount("amount", in.Amount)
	if err != nil {
		return nil, err
	}
	if amt <= 0 {
		return nil, invalid("amount", "Nominal harus lebih dari 0.")
	}
	if strings.TrimSpace(in.Label) == "" {
		return nil, invalid("label", "Keterangan wajib diisi.")
	}
	if _, err := (&People{d: p.d}).Get(ctx, in.EmployeeID); err != nil {
		return nil, err
	}
	month, year := in.Month, in.Year
	var run *domain.PayrollRun
	if month != 0 || year != 0 {
		if month < 1 || month > 12 || year < 2000 {
			return nil, invalid("periodMonth", "Periode tidak valid.")
		}
		if run, err = p.d.Store.GetPayrollRunByPeriod(ctx, month, year); err != nil {
			return nil, err
		}
		if run != nil && !payroll.CanCalculateRun(run.Status) {
			return nil, conflict("Payroll periode ini sudah disetujui atau dibayar.")
		}
	} else {
		now := hris.NowWIB(p.d.now())
		for i := range 13 {
			t := time.Date(now.Year(), now.Month()+time.Month(i), 1, 0, 0, 0, 0, time.UTC)
			month, year = int(t.Month()), t.Year()
			if run, err = p.d.Store.GetPayrollRunByPeriod(ctx, month, year); err != nil {
				return nil, err
			}
			if run == nil || payroll.CanCalculateRun(run.Status) {
				break
			}
		}
	}
	add := domain.PayrollAddition{ID: newID(), EmployeeID: in.EmployeeID, PeriodMonth: month, PeriodYear: year,
		Amount: math.Round(amt*100) / 100, Label: strings.TrimSpace(in.Label), CreatedBy: a.UserRef()}
	err = p.d.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertPayrollAddition(ctx, add); err != nil {
			return err
		}
		if run != nil {
			return tx.FlagRunRecalculation(ctx, run.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	rows, err := p.d.Store.ListPayrollAdditions(ctx, month, year, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.ID == add.ID {
			return &r, nil
		}
	}
	return &add, nil
}

func (p *Payroll) DeleteAddition(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermPayroll); err != nil {
		return err
	}
	add, err := p.d.Store.GetPayrollAddition(ctx, id)
	if err != nil {
		return err
	}
	if add == nil {
		return notFound("Tambahan payroll tidak ditemukan.")
	}
	run, err := p.d.Store.GetPayrollRunByPeriod(ctx, add.PeriodMonth, add.PeriodYear)
	if err != nil {
		return err
	}
	if run != nil && !payroll.CanCalculateRun(run.Status) {
		return conflict("Payroll periode ini sudah disetujui atau dibayar.")
	}
	return p.d.Store.WithTx(ctx, func(tx domain.Store) error {
		if _, err := tx.DeletePayrollAddition(ctx, id); err != nil {
			return err
		}
		if run != nil {
			return tx.FlagRunRecalculation(ctx, run.ID)
		}
		return nil
	})
}
