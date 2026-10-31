package app

import (
	"context"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// Loans are employee loans and salary advances (NüHabit loans.go): flat
// interest, installment and active-loan limits, approval by hr.approve,
// repayment through payroll.
type Loans struct{ d Deps }

func NewLoans(d Deps) *Loans { return &Loans{d: d} }

const invalidLoan = "Jenis pinjaman, jumlah (> 0), dan tenor (1 sampai 60 bulan) wajib valid."

var loanTypes = []string{"kasbon", "loan", "emergency"}

func canManageLoans(a Actor) bool { return a.Can(authn.PermPayroll) || a.Can(authn.PermHRApprove) }

func (l *Loans) List(ctx context.Context, a Actor, employeeID, status string) ([]domain.Loan, error) {
	if !canManageLoans(a) || (employeeID != "" && employeeID == a.EmployeeID) {
		id, err := self(a)
		if err != nil {
			return nil, err
		}
		employeeID = id
	}
	return l.d.Store.ListLoans(ctx, employeeID, status)
}

func (l *Loans) Get(ctx context.Context, a Actor, id string) (*domain.Loan, error) {
	loan, err := l.d.Store.GetLoan(ctx, id)
	if err != nil {
		return nil, err
	}
	if loan == nil || (!canManageLoans(a) && loan.EmployeeID != a.EmployeeID) {
		return nil, notFound("Pinjaman tidak ditemukan.")
	}
	return loan, nil
}

type LoanInput struct {
	EmployeeID, LoanType, Principal, InterestRate, Purpose string
	Tenor                                                  int
}

// Create files a loan: hr.payroll for anyone (with interest), otherwise
// self.service for oneself, interest free.
func (l *Loans) Create(ctx context.Context, a Actor, in LoanInput) (*domain.Loan, error) {
	full := a.Can(authn.PermPayroll)
	target := in.EmployeeID
	if !full || target == "" {
		id, err := self(a)
		if err != nil {
			return nil, err
		}
		target = id
	}
	principal, err := amount("principalAmount", in.Principal)
	if err != nil || principal <= 0 || in.Tenor < 1 || in.Tenor > 60 || !slices.Contains(loanTypes, in.LoanType) {
		return nil, invalid("principalAmount", invalidLoan)
	}
	rate := 0.0
	if full && strings.TrimSpace(in.InterestRate) != "" {
		if rate, err = amount("interestRate", in.InterestRate); err != nil || rate > 100 {
			return nil, invalid("interestRate", invalidLoan)
		}
	}
	emp, err := l.d.Store.GetEmployee(ctx, target)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, notFound(msgEmployeeNotFound)
	}
	if !emp.IsActive {
		return nil, invalid("employeeId", "Karyawan sudah tidak aktif.")
	}
	raw, monthly, total := payroll.LoanTerms(principal, rate, in.Tenor)
	if msg, err := l.checkLimits(ctx, target, raw, true); err != nil {
		return nil, err
	} else if msg != "" {
		return nil, invalid("principalAmount", msg)
	}
	loan := domain.Loan{ID: newID(), EmployeeID: target, LoanType: in.LoanType, PrincipalAmount: principal, InterestRate: rate,
		TenorMonths: in.Tenor, MonthlyInstallment: monthly, RemainingBalance: total, Purpose: strings.TrimSpace(in.Purpose), CreatedBy: a.UserRef()}
	if err := l.d.Store.InsertLoan(ctx, loan); err != nil {
		return nil, err
	}
	notifyHR(ctx, l.d, NotifyInput{Type: "HR_LOAN_REQUESTED", Title: "Pengajuan pinjaman: " + emp.FullName,
		Body: payroll.FormatRupiah(principal) + ", " + strconv.Itoa(in.Tenor) + " bulan", Href: "/erp/hr/loans"})
	return l.d.Store.GetLoan(ctx, loan.ID)
}

// checkLimits is validateLoanLimits against the settings and the active
// salary; "" when the installment fits.
func (l *Loans) checkLimits(ctx context.Context, employeeID string, installment float64, includePending bool) (string, error) {
	settings, err := l.d.Store.GetPayrollSettings(ctx)
	if err != nil {
		return "", err
	}
	config := configOf(settings)
	sal, err := l.d.Store.ActiveSalary(ctx, employeeID)
	if err != nil {
		return "", err
	}
	var base *float64
	if sal != nil {
		base = &sal.BaseSalary
	}
	count, err := l.d.Store.CountOpenLoans(ctx, employeeID, includePending)
	if err != nil {
		return "", err
	}
	return payroll.ValidateLoanLimits(installment, base, config.LoanMaxInstallmentPct, float64(count), config.LoanMaxActivePerPerson), nil
}

// Decide approves (first installment the month after, WIB) or rejects.
func (l *Loans) Decide(ctx context.Context, a Actor, id string, approve bool, reason string) (*domain.Loan, error) {
	if err := require(a, authn.PermHRApprove); err != nil {
		return nil, err
	}
	loan, err := l.d.Store.GetLoan(ctx, id)
	if err != nil {
		return nil, err
	}
	if loan == nil {
		return nil, notFound("Pinjaman tidak ditemukan.")
	}
	if loan.Status != "pending" {
		return nil, conflict("Pinjaman sudah diproses.")
	}
	var ok bool
	if approve {
		msg, err := l.checkLimits(ctx, loan.EmployeeID, loan.MonthlyInstallment, false)
		if err != nil {
			return nil, err
		}
		if msg != "" {
			return nil, invalid("loanId", msg)
		}
		now := l.d.now()
		month, year := payroll.FirstInstallmentPeriod(hris.NowWIB(now))
		ok, err = l.d.Store.ApproveLoan(ctx, id, a.EmployeeRef(), now, month, year)
		if err != nil {
			return nil, err
		}
	} else {
		if reason = strings.TrimSpace(reason); reason == "" {
			reason = "Tidak disetujui"
		}
		if ok, err = l.d.Store.RejectLoan(ctx, id, a.EmployeeRef(), reason); err != nil {
			return nil, err
		}
	}
	if !ok {
		return nil, conflict("Pinjaman sudah diproses oleh orang lain.")
	}
	title := "Pinjaman ditolak"
	if approve {
		title = "Pinjaman disetujui, cicilan mulai bulan depan"
	}
	notifyEmployee(ctx, l.d, loan.EmployeeID, NotifyInput{Type: "HR_LOAN_DECIDED", Title: title, Body: reason, Href: "/erp/me/loans"})
	return l.d.Store.GetLoan(ctx, id)
}

// Cancel withdraws a pending loan: hr.payroll or hr.approve for any loan,
// self.service for one's own. An approved loan is repaid through payroll
// and cannot be cancelled.
func (l *Loans) Cancel(ctx context.Context, a Actor, id string) (*domain.Loan, error) {
	if !canManageLoans(a) {
		if _, err := self(a); err != nil {
			return nil, err
		}
	}
	loan, err := l.Get(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if loan.Status != "pending" {
		return nil, conflict("Hanya pinjaman berstatus pending yang bisa dibatalkan.")
	}
	ok, err := l.d.Store.CancelLoan(ctx, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, conflict("Pinjaman sudah diproses oleh orang lain.")
	}
	return l.d.Store.GetLoan(ctx, id)
}

// ScheduleRow is one planned installment.
type ScheduleRow struct {
	No, Month, Year        int
	Amount, RemainingAfter float64
	IsPaid                 bool
}

// Schedule plans the installments from the first installment period; the
// ones covered by the paid amount are marked paid.
func Schedule(loan domain.Loan) []ScheduleRow {
	if loan.FirstInstallmentMonth == nil || loan.FirstInstallmentYear == nil || loan.MonthlyInstallment <= 0 {
		return nil
	}
	total := loan.PaidAmount + loan.RemainingBalance
	var out []ScheduleRow
	m, y := *loan.FirstInstallmentMonth, *loan.FirstInstallmentYear
	left, paid := total, loan.PaidAmount
	for i := 1; i <= loan.TenorMonths && left > 0; i++ {
		amt := math.Min(loan.MonthlyInstallment, left)
		left -= amt
		out = append(out, ScheduleRow{No: i, Month: m, Year: y, Amount: amt, RemainingAfter: left, IsPaid: paid >= amt-0.005})
		paid -= amt
		if m++; m > 12 {
			m, y = 1, y+1
		}
	}
	return out
}
