package domain

import (
	"sort"
	"strings"
)

// Journal mappings: an operational event plus an amount bag becomes journal
// lines (NüHabit accounting/domain/mapping.go).

// Journal modules and amount sources.
var (
	JournalModules       = []string{"PAYROLL", "SALES", "PURCHASING", "REIMBURSEMENT", "CASH_BANK"}
	JournalAmountSources = []string{"TOTAL", "SUBTOTAL", "TAX", "COGS", "PAID", "DISCOUNT", "SERVICE_CHARGE"}
)

// Event codes Mixist posts automatically.
const (
	EventPayrollAccrual         = "PAYROLL_ACCRUAL"
	EventPayrollPayment         = "PAYROLL_PAYMENT"
	EventPayrollPPh21           = "PAYROLL_PPH21_WITHHOLDING"
	EventPayrollLoanDeduction   = "PAYROLL_LOAN_DEDUCTION"
	EventPayrollBpjsTkEmployer  = "PAYROLL_BPJS_TK_EMPLOYER"
	EventPayrollBpjsKesEmployer = "PAYROLL_BPJS_KES_EMPLOYER"
	EventPayrollBpjsTkEmployee  = "PAYROLL_BPJS_TK_EMPLOYEE"
	EventPayrollBpjsKesEmployee = "PAYROLL_BPJS_KES_EMPLOYEE"
	EventPayrollTaperaEmployer  = "PAYROLL_TAPERA_EMPLOYER"
	EventPayrollTaperaEmployee  = "PAYROLL_TAPERA_EMPLOYEE"
	EventSaleARInvoice          = "SALE_AR_INVOICE"
	EventSaleARReceipt          = "SALE_AR_RECEIPT"
	EventPurchaseAPInvoice      = "PURCHASE_AP_INVOICE"
	EventPurchasePayment        = "PURCHASE_PAYMENT"
	EventReimbursementPayment   = "REIMBURSEMENT_PAYMENT"
)

// JournalEvent is the catalog entry of an event code.
type JournalEvent struct {
	Code, Name, Description, Module string
}

// JournalEvents is the catalog the mapping editor offers (names and
// descriptions from NüHabit JOURNAL_EVENT_META for the codes Mixist uses).
var JournalEvents = []JournalEvent{
	{EventPayrollAccrual, "Payroll Accrual", "Pengakuan beban gaji (expense) dan hutang gaji", "PAYROLL"},
	{EventPayrollPPh21, "Payroll PPh 21 Withholding", "Potongan PPh 21 dari payroll", "PAYROLL"},
	{EventPayrollLoanDeduction, "Payroll Loan Deduction", "Potongan cicilan pinjaman lewat payroll", "PAYROLL"},
	{EventPayrollBpjsTkEmployee, "Payroll BPJS Ketenagakerjaan (Karyawan)", "Potongan BPJS Ketenagakerjaan karyawan (hutang gaji ke hutang BPJS)", "PAYROLL"},
	{EventPayrollBpjsKesEmployee, "Payroll BPJS Kesehatan (Karyawan)", "Potongan BPJS Kesehatan karyawan (hutang gaji ke hutang BPJS)", "PAYROLL"},
	{EventPayrollTaperaEmployee, "Payroll Tapera (Karyawan)", "Potongan Tapera karyawan (hutang gaji ke hutang Tapera)", "PAYROLL"},
	{EventPayrollPayment, "Payroll Payment", "Pembayaran gaji bersih ke karyawan via bank/kas", "PAYROLL"},
	{EventPayrollBpjsTkEmployer, "Payroll BPJS Ketenagakerjaan (Pemberi Kerja)", "Iuran BPJS Ketenagakerjaan bagian perusahaan (beban dan hutang BPJS)", "PAYROLL"},
	{EventPayrollBpjsKesEmployer, "Payroll BPJS Kesehatan (Pemberi Kerja)", "Iuran BPJS Kesehatan bagian perusahaan (beban dan hutang BPJS)", "PAYROLL"},
	{EventPayrollTaperaEmployer, "Payroll Tapera (Pemberi Kerja)", "Iuran Tapera bagian perusahaan (beban dan hutang Tapera)", "PAYROLL"},
	{EventSaleARInvoice, "Sale AR Invoice", "Pengakuan piutang dari invoice B2B / sales", "SALES"},
	{EventSaleARReceipt, "Sale AR Receipt", "Penerimaan pembayaran piutang customer", "SALES"},
	{EventPurchaseAPInvoice, "Purchase AP Invoice", "Invoice hutang vendor", "PURCHASING"},
	{EventPurchasePayment, "Purchase Payment", "Pembayaran hutang vendor", "PURCHASING"},
	{EventReimbursementPayment, "Reimbursement Payment", "Pembayaran klaim reimbursement karyawan dari kas/bank", "REIMBURSEMENT"},
}

// Amounts is an amount bag: amount source -> value. A missing key is 0.
type Amounts map[string]Money

// AmountFromSource: non-positive values are 0.
func AmountFromSource(source string, amounts Amounts) Money {
	if v := amounts[source]; v > 0 {
		return v
	}
	return 0
}

// PaymentAmounts is the bag of a payment or receipt.
func PaymentAmounts(amount Money) Amounts {
	if amount < 0 {
		amount = 0
	}
	return Amounts{"PAID": amount, "TOTAL": amount, "SUBTOTAL": amount}
}

// MappingLine is one template line of a journal mapping.
type MappingLine struct {
	ID           string
	Side         string
	AccountID    string
	AmountSource string
	SortOrder    int
	IsRequired   bool
	LineRole     string
}

// HasRequiredAccounts reports whether every required line has an account.
func HasRequiredAccounts(lines []MappingLine) bool {
	for _, l := range lines {
		if l.IsRequired && l.AccountID == "" {
			return false
		}
	}
	return true
}

// WithAccountOverrides replaces the account of lines whose role is in
// overrides (an AP invoice's expense account, a payment's bank account).
// The override also fills a required line left empty in the mapping.
func WithAccountOverrides(lines []MappingLine, overrides map[string]string) []MappingLine {
	out := make([]MappingLine, len(lines))
	for i, l := range lines {
		if id := overrides[l.LineRole]; id != "" {
			l.AccountID = id
		}
		out[i] = l
	}
	return out
}

// Split spreads one mapping line over several accounts (a reimbursement
// batch debits one expense account per category). An empty AccountID uses
// the line's account.
type Split struct {
	AccountID string `json:"accountId,omitempty"`
	Amount    Money  `json:"amount"`
}

// BuildFromMapping returns the journal lines a mapping produces for an
// amount bag, or the reason it is not ready. A role listed in splits posts
// one line per split instead of reading the bag.
func BuildFromMapping(lines []MappingLine, amounts Amounts, memo string, splits map[string][]Split) ([]Line, string) {
	if !HasRequiredAccounts(lines) {
		return nil, "Journal mapping belum lengkap (akun COA wajib belum diisi)"
	}
	out := []Line{}
	for _, l := range lines {
		parts, isSplit := splits[l.LineRole]
		if !isSplit {
			parts = []Split{{Amount: AmountFromSource(l.AmountSource, amounts)}}
		}
		var total Money
		for _, p := range parts {
			if p.Amount <= 0 {
				continue
			}
			total += p.Amount
			account := p.AccountID
			if account == "" {
				account = l.AccountID
			}
			// An optional role (DISCOUNT/TAX) with a value but no account would
			// unbalance the journal, so it is not skipped silently.
			if account == "" {
				return nil, "Akun COA untuk role " + l.LineRole + " belum diisi (nilai " + l.AmountSource + "=" + p.Amount.Number() + ")"
			}
			out = append(out, Line{AccountID: account, Side: l.Side, Amount: p.Amount, Memo: memo, SortOrder: l.SortOrder})
		}
		if total <= 0 && l.IsRequired {
			return nil, "Amount source " + l.AmountSource + " untuk role " + l.LineRole + " bernilai 0"
		}
	}
	if len(out) < 2 {
		return nil, "Baris jurnal dari mapping kurang dari 2 setelah amount dihitung"
	}
	return MergeLines(out), ""
}

// Post result statuses.
const (
	PostPosted  = "posted"
	PostDraft   = "draft"
	PostSkipped = "skipped"
)

// AlreadyExists is the reason of an idempotent repeat post.
const AlreadyExists = "already_exists"

// PostResult is the outcome of posting one source document.
type PostResult struct {
	Status  string
	EntryID string
	Reason  string
}

// Summarize is the human note appended to a response, "" when there is
// nothing to say.
func Summarize(results ...PostResult) string {
	var notes []string
	seen := map[string]bool{}
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			notes = append(notes, n)
		}
	}
	for _, r := range results {
		switch {
		case r.Status == PostPosted:
			add("jurnal posted")
		case r.Status == PostDraft:
			add("jurnal draft: " + orDefault(r.Reason, "mapping belum lengkap"))
		case r.Status == PostSkipped && r.Reason != AlreadyExists:
			add("jurnal dilewati: " + orDefault(r.Reason, "tidak ada mapping"))
		}
	}
	return strings.Join(notes, "; ")
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// MergeLines combines lines on the same account and side (a reimbursement
// batch with many claims of one category posts one expense line).
func MergeLines(lines []Line) []Line {
	type key struct{ account, side string }
	idx := map[key]int{}
	var out []Line
	for _, l := range lines {
		k := key{l.AccountID, l.Side}
		if i, ok := idx[k]; ok {
			out[i].Amount += l.Amount
			continue
		}
		idx[k] = len(out)
		out = append(out, l)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].SortOrder < out[j].SortOrder })
	return out
}
