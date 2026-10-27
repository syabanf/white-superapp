package payroll

import "slices"

// Run statuses. NüHabit used draft -> processing -> completed -> paid and
// calculated inside draft; Mixist names the steps after what happened:
// draft (created), calculated, approved, paid.
const (
	RunDraft      = "draft"
	RunCalculated = "calculated"
	RunApproved   = "approved"
	RunPaid       = "paid"
)

var runTransitions = map[string][]string{
	RunDraft:      {RunCalculated},
	RunCalculated: {RunApproved},
	RunApproved:   {RunPaid},
	RunPaid:       {},
}

// CanTransitionRun is canTransitionRunStatus: only forward, one step.
func CanTransitionRun(from, to string) bool { return slices.Contains(runTransitions[from], to) }

// CanDeleteRun is canDeleteRun: a paid run is a final financial record.
func CanDeleteRun(status string) bool { return status != RunPaid }

// CanCalculateRun: a run is (re)calculated until it is approved.
func CanCalculateRun(status string) bool { return status == RunDraft || status == RunCalculated }

// SlipShare is what one payslip contributes to the paid-run totals finance
// posts (RunSlipShare without the company split: Mixist is one company).
// Gross is the slip's gross less its unpaid-leave and late deductions, so
// Gross = Net + Pph21 + Loan + every employee BPJS and Tapera share and
// finance's salary payable clears.
type SlipShare struct {
	Gross, Net, Pph21               float64
	BpjsTkEmployer, BpjsKesEmployer float64
	BpjsTkEmployee, BpjsKesEmployee float64
	TaperaEmployer, TaperaEmployee  float64
	Loan                            float64
}

// ShareOf is a slip's contribution to the paid-run totals.
func ShareOf(r Result) SlipShare {
	return SlipShare{
		Gross: r.GrossSalary - r.UnpaidLeaveDeduction - r.LateDeduction - r.OtherDeduction,
		Net:   r.NetSalary, Pph21: r.Pph21Deduction, Loan: r.LoanDeduction,
		BpjsTkEmployee:  r.BpjsTkJhtDeduction + r.BpjsTkJpDeduction,
		BpjsKesEmployee: r.BpjsKesDeduction, TaperaEmployee: r.TaperaDeduction,
		BpjsTkEmployer:  r.BpjsTkJhtEmployer + r.BpjsTkJpEmployer + r.BpjsTkJkkEmployer + r.BpjsTkJkmEmployer,
		BpjsKesEmployer: r.BpjsKesEmployer, TaperaEmployer: r.TaperaEmployer,
	}
}

// SumSlipShares adds the slips of a run, rounded to cents like
// SplitRunByCompany did per company.
func SumSlipShares(slips []SlipShare) SlipShare {
	var t SlipShare
	for _, s := range slips {
		t.Gross += s.Gross
		t.Net += s.Net
		t.Pph21 += s.Pph21
		t.BpjsTkEmployer += s.BpjsTkEmployer
		t.BpjsKesEmployer += s.BpjsKesEmployer
		t.BpjsTkEmployee += s.BpjsTkEmployee
		t.BpjsKesEmployee += s.BpjsKesEmployee
		t.TaperaEmployer += s.TaperaEmployer
		t.TaperaEmployee += s.TaperaEmployee
		t.Loan += s.Loan
	}
	for _, v := range []*float64{&t.Gross, &t.Net, &t.Pph21, &t.BpjsTkEmployer, &t.BpjsKesEmployer, &t.BpjsTkEmployee,
		&t.BpjsKesEmployee, &t.TaperaEmployer, &t.TaperaEmployee, &t.Loan} {
		*v = Round(*v*100) / 100
	}
	return t
}
