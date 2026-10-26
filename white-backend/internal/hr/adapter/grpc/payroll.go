package grpcapi

import (
	"context"
	"fmt"

	hrv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/hr/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"google.golang.org/protobuf/types/known/emptypb"
)

type payrollServer struct {
	hrv1.UnimplementedPayrollServiceServer
	s *Services
}

func (h *payrollServer) GetPayrollSettings(ctx context.Context, _ *emptypb.Empty) (*hrv1.PayrollSettings, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	st, err := h.s.Payroll.Settings(ctx, a)
	if err != nil {
		return nil, err
	}
	return settingsPB(*st), nil
}

// settingsFrom parses the decimal strings; field names follow the proto.
func settingsFrom(in *hrv1.PayrollSettings) (domain.PayrollSettings, error) {
	var s domain.PayrollSettings
	var bad string
	num := func(field, v string) float64 {
		f, ok := domain.ParseMoney(v)
		if !ok && bad == "" {
			bad = field
		}
		return f
	}
	s.BpjsTkJhtEmployee, s.BpjsTkJhtEmployer = num("bpjsTkJhtEmployee", in.BpjsTkJhtEmployee), num("bpjsTkJhtEmployer", in.BpjsTkJhtEmployer)
	s.BpjsTkJpEmployee, s.BpjsTkJpEmployer = num("bpjsTkJpEmployee", in.BpjsTkJpEmployee), num("bpjsTkJpEmployer", in.BpjsTkJpEmployer)
	s.BpjsTkJkk, s.BpjsTkJkm = num("bpjsTkJkk", in.BpjsTkJkk), num("bpjsTkJkm", in.BpjsTkJkm)
	s.BpjsKesEmployee, s.BpjsKesEmployer = num("bpjsKesEmployee", in.BpjsKesEmployee), num("bpjsKesEmployer", in.BpjsKesEmployer)
	s.BpjsKesMaxUpah = num("bpjsKesMaxUpah", in.BpjsKesMaxUpah)
	s.TaperaEmployee, s.TaperaEmployer = num("taperaEmployee", in.TaperaEmployee), num("taperaEmployer", in.TaperaEmployer)
	s.PTKP = map[string]float64{}
	for _, st := range domain.PTKPStatuses {
		s.PTKP[st] = num("ptkp", in.Ptkp[st])
	}
	if len(in.Pph21BracketLimits) != 4 {
		return s, apperr.Field("pph21BracketLimits", "Isi empat batas lapisan PPh 21.")
	}
	for i, b := range in.Pph21BracketLimits {
		s.BracketLimits[i] = num("pph21BracketLimits", b)
	}
	s.OvertimeMultiplier = num("overtimeMultiplier", in.OvertimeMultiplier)
	s.OvertimeMultiplierHoliday = num("overtimeMultiplierHoliday", in.OvertimeMultiplierHoliday)
	s.OvertimeHourlyDivisor = num("overtimeHourlyDivisor", in.OvertimeHourlyDivisor)
	s.LateDeductionMode, s.LateDeductionAmount = in.LateDeductionMode, num("lateDeductionAmount", in.LateDeductionAmount)
	s.LoanMaxInstallmentPercent = num("loanMaxInstallmentPercent", in.LoanMaxInstallmentPercent)
	s.LoanMaxActivePerEmployee, s.PayrollDay = int(in.LoanMaxActivePerEmployee), int(in.PayrollDay)
	if bad != "" {
		return s, apperr.Field(bad, "Nilai harus angka desimal.")
	}
	return s, nil
}

func (h *payrollServer) UpdatePayrollSettings(ctx context.Context, req *hrv1.PayrollSettings) (*hrv1.PayrollSettings, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	in, err := settingsFrom(req)
	if err != nil {
		return nil, err
	}
	st, err := h.s.Payroll.UpdateSettings(ctx, a, in)
	if err != nil {
		return nil, err
	}
	return settingsPB(*st), nil
}

func (h *payrollServer) ListSalaries(ctx context.Context, req *hrv1.ListSalariesRequest) (*hrv1.ListSalariesResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Payroll.Salaries(ctx, a, req.EmployeeId, req.ActiveOnly)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListSalariesResponse{}
	for _, r := range rows {
		out.Salaries = append(out.Salaries, salaryPB(r))
	}
	return out, nil
}

func (h *payrollServer) CreateSalary(ctx context.Context, req *hrv1.SalaryInput) (*hrv1.Salary, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	s, err := h.s.Payroll.CreateSalary(ctx, a, salaryInput(req))
	if err != nil {
		return nil, err
	}
	return salaryPB(*s), nil
}

func (h *payrollServer) UpdateSalary(ctx context.Context, req *hrv1.UpdateSalaryRequest) (*hrv1.Salary, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	s, err := h.s.Payroll.UpdateSalary(ctx, a, req.SalaryId, salaryInput(req.Input))
	if err != nil {
		return nil, err
	}
	return salaryPB(*s), nil
}

func (h *payrollServer) DeactivateSalary(ctx context.Context, req *hrv1.SalaryRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.Payroll.DeactivateSalary(ctx, a, req.SalaryId)
}

func (h *payrollServer) ListPayrollRuns(ctx context.Context, req *hrv1.ListPayrollRunsRequest) (*hrv1.ListPayrollRunsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Payroll.Runs(ctx, a, int(req.Year), req.Status)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListPayrollRunsResponse{}
	for _, r := range rows {
		out.Runs = append(out.Runs, runPB(r))
	}
	return out, nil
}

func (h *payrollServer) CreatePayrollRun(ctx context.Context, req *hrv1.CreatePayrollRunRequest) (*hrv1.PayrollRun, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.s.Payroll.CreateRun(ctx, a, int(req.PeriodMonth), int(req.PeriodYear), req.RunName)
	if err != nil {
		return nil, err
	}
	return runPB(*r), nil
}

func (h *payrollServer) GetPayrollRun(ctx context.Context, req *hrv1.PayrollRunRequest) (*hrv1.PayrollRunDetail, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, slips, err := h.s.Payroll.Run(ctx, a, req.RunId)
	if err != nil {
		return nil, err
	}
	return &hrv1.PayrollRunDetail{Run: runPB(*r), Payslips: payslipList(slips).Payslips}, nil
}

func (h *payrollServer) CalculatePayrollRun(ctx context.Context, req *hrv1.CalculatePayrollRunRequest) (*hrv1.CalculatePayrollRunResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, skipped, err := h.s.Payroll.Calculate(ctx, a, req.RunId, req.IncludeThr)
	if err != nil {
		return nil, err
	}
	out := &hrv1.CalculatePayrollRunResponse{Run: runPB(*r), Message: fmt.Sprintf("Payroll berhasil dihitung untuk %d karyawan", r.TotalEmployees)}
	if len(skipped) > 0 {
		out.Message += fmt.Sprintf(", %d dilewati", len(skipped))
	}
	for _, s := range skipped {
		out.Skipped = append(out.Skipped, &hrv1.SkippedEmployee{EmployeeId: s.EmployeeID, FullName: s.FullName, Reason: s.Reason})
	}
	return out, nil
}

func (h *payrollServer) ApprovePayrollRun(ctx context.Context, req *hrv1.PayrollRunRequest) (*hrv1.PayrollRun, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, err := h.s.Payroll.Approve(ctx, a, req.RunId)
	if err != nil {
		return nil, err
	}
	return runPB(*r), nil
}

func (h *payrollServer) MarkPayrollRunPaid(ctx context.Context, req *hrv1.MarkPayrollRunPaidRequest) (*hrv1.MarkPayrollRunPaidResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, settled, err := h.s.Payroll.MarkPaid(ctx, a, req.RunId, req.Notes)
	if err != nil {
		return nil, err
	}
	msg := "Payroll ditandai dibayar"
	if settled > 0 {
		msg += fmt.Sprintf(": %d cicilan pinjaman dipotong dari saldo", settled)
	}
	return &hrv1.MarkPayrollRunPaidResponse{Run: runPB(*r), SettledLoans: int32(settled), Message: msg}, nil
}

func (h *payrollServer) DeletePayrollRun(ctx context.Context, req *hrv1.PayrollRunRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.Payroll.DeleteRun(ctx, a, req.RunId)
}

func (h *payrollServer) ListPayslips(ctx context.Context, req *hrv1.ListPayslipsRequest) (*hrv1.ListPayslipsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Payroll.Payslips(ctx, a, domain.PayslipFilter{EmployeeID: req.EmployeeId, Year: int(req.Year), RunID: req.RunId})
	if err != nil {
		return nil, err
	}
	return payslipList(rows), nil
}

func (s *Services) payslipPDF(ctx context.Context, a app.Actor, id string) (*hrv1.FileBytes, error) {
	data, name, err := s.Payroll.PayslipPDF(ctx, a, id)
	if err != nil {
		return nil, err
	}
	return file(name, "application/pdf", data), nil
}

func (h *payrollServer) GetPayslipPdf(ctx context.Context, req *hrv1.PayslipRequest) (*hrv1.FileBytes, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return h.s.payslipPDF(ctx, a, req.PayslipId)
}

func (h *payrollServer) ListPayrollAdditions(ctx context.Context, req *hrv1.ListPayrollAdditionsRequest) (*hrv1.ListPayrollAdditionsResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Payroll.Additions(ctx, a, int(req.PeriodMonth), int(req.PeriodYear), req.EmployeeId)
	if err != nil {
		return nil, err
	}
	out := &hrv1.ListPayrollAdditionsResponse{}
	for _, r := range rows {
		out.Additions = append(out.Additions, additionPB(r))
	}
	return out, nil
}

func (h *payrollServer) CreatePayrollAddition(ctx context.Context, req *hrv1.PayrollAdditionInput) (*hrv1.PayrollAddition, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	add, err := h.s.Payroll.CreateAddition(ctx, a, app.AdditionInput{EmployeeID: req.EmployeeId, Amount: req.Amount, Label: req.Label,
		Month: int(req.PeriodMonth), Year: int(req.PeriodYear)})
	if err != nil {
		return nil, err
	}
	return additionPB(*add), nil
}

func (h *payrollServer) DeletePayrollAddition(ctx context.Context, req *hrv1.PayrollAdditionRequest) (*emptypb.Empty, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return empty, h.s.Payroll.DeleteAddition(ctx, a, req.AdditionId)
}

func (h *payrollServer) ListLoans(ctx context.Context, req *hrv1.ListLoansRequest) (*hrv1.ListLoansResponse, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := h.s.Loans.List(ctx, a, req.EmployeeId, req.Status)
	if err != nil {
		return nil, err
	}
	return loanList(rows), nil
}

func (h *payrollServer) GetLoan(ctx context.Context, req *hrv1.LoanRequest) (*hrv1.Loan, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	l, err := h.s.Loans.Get(ctx, a, req.LoanId)
	if err != nil {
		return nil, err
	}
	return loanPB(*l, true), nil
}

func (s *Services) createLoan(ctx context.Context, a app.Actor, req *hrv1.CreateLoanRequest) (*hrv1.Loan, error) {
	l, err := s.Loans.Create(ctx, a, app.LoanInput{EmployeeID: req.EmployeeId, LoanType: req.LoanType, Principal: req.PrincipalAmount,
		InterestRate: req.InterestRate, Purpose: req.Purpose, Tenor: int(req.TenorMonths)})
	if err != nil {
		return nil, err
	}
	return loanPB(*l, true), nil
}

func (h *payrollServer) CreateLoan(ctx context.Context, req *hrv1.CreateLoanRequest) (*hrv1.Loan, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	return h.s.createLoan(ctx, a, req)
}

func (h *payrollServer) DecideLoan(ctx context.Context, req *hrv1.DecideLoanRequest) (*hrv1.Loan, error) {
	a, err := h.s.actor(ctx)
	if err != nil {
		return nil, err
	}
	l, err := h.s.Loans.Decide(ctx, a, req.LoanId, req.Approve, req.RejectionReason)
	if err != nil {
		return nil, err
	}
	return loanPB(*l, true), nil
}
