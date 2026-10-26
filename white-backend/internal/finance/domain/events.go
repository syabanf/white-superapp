package domain

import (
	"fmt"
	"time"
)

// Payloads of the events finance consumes and publishes. Money fields
// accept decimal strings ("1250000.00") or JSON numbers.

// PayrollPaid is the payload of white.hr.payroll.paid, published by hr
// once a payroll run is paid. Finance posts one journal per non-zero
// component, keyed by the run id so a redelivery is a no-op.
//
//	{
//	  "runId": "0192...", "periodYear": 2026, "periodMonth": 10,
//	  "label": "Gaji Oktober 2026", "paidAt": "2026-10-25T03:00:00Z",
//	  "paidBy": "<identity user id>", "employeeCount": 25,
//	  "totalGross": "250000000.00", "totalNet": "221500000.00",
//	  "totalPph21": "12000000.00", "totalLoanDeduction": "1500000.00",
//	  "totalBpjsTkEmployee": "5000000.00", "totalBpjsKesEmployee": "2500000.00",
//	  "totalTaperaEmployee": "0", "totalBpjsTkEmployer": "9500000.00",
//	  "totalBpjsKesEmployer": "10000000.00", "totalTaperaEmployer": "0"
//	}
//
// totalGross must equal totalNet plus every employee deduction so the
// salary payable clears.
type PayrollPaid struct {
	RunID                string    `json:"runId"`
	PeriodYear           int       `json:"periodYear"`
	PeriodMonth          int       `json:"periodMonth"`
	Label                string    `json:"label"`
	PaidAt               time.Time `json:"paidAt"`
	PaidBy               string    `json:"paidBy,omitempty"`
	EmployeeCount        int       `json:"employeeCount"`
	TotalGross           Money     `json:"totalGross"`
	TotalNet             Money     `json:"totalNet"`
	TotalPph21           Money     `json:"totalPph21"`
	TotalLoanDeduction   Money     `json:"totalLoanDeduction"`
	TotalBpjsTkEmployee  Money     `json:"totalBpjsTkEmployee"`
	TotalBpjsKesEmployee Money     `json:"totalBpjsKesEmployee"`
	TotalTaperaEmployee  Money     `json:"totalTaperaEmployee"`
	TotalBpjsTkEmployer  Money     `json:"totalBpjsTkEmployer"`
	TotalBpjsKesEmployer Money     `json:"totalBpjsKesEmployer"`
	TotalTaperaEmployer  Money     `json:"totalTaperaEmployer"`
}

// PayrollJournal is one journal a paid run posts.
type PayrollJournal struct {
	Event, Source, Label string
	Amount               Money
}

// PayrollJournals lists the journals of a paid run in NüHabit's order, with
// the amount source each seeded mapping reads. Zero components are left out.
func PayrollJournals(p PayrollPaid) []PayrollJournal {
	all := []PayrollJournal{
		{EventPayrollAccrual, "TOTAL", "beban gaji", p.TotalGross},
		{EventPayrollPPh21, "TAX", "potongan PPh 21", p.TotalPph21},
		{EventPayrollLoanDeduction, "PAID", "potongan cicilan pinjaman", p.TotalLoanDeduction},
		{EventPayrollBpjsTkEmployee, "TOTAL", "potongan BPJS Ketenagakerjaan karyawan", p.TotalBpjsTkEmployee},
		{EventPayrollBpjsKesEmployee, "TOTAL", "potongan BPJS Kesehatan karyawan", p.TotalBpjsKesEmployee},
		{EventPayrollTaperaEmployee, "TOTAL", "potongan Tapera karyawan", p.TotalTaperaEmployee},
		{EventPayrollPayment, "PAID", "pembayaran gaji bersih", p.TotalNet},
		{EventPayrollBpjsTkEmployer, "TOTAL", "BPJS Ketenagakerjaan pemberi kerja", p.TotalBpjsTkEmployer},
		{EventPayrollBpjsKesEmployer, "TOTAL", "BPJS Kesehatan pemberi kerja", p.TotalBpjsKesEmployer},
		{EventPayrollTaperaEmployer, "TOTAL", "Tapera pemberi kerja", p.TotalTaperaEmployer},
	}
	out := all[:0:0]
	for _, j := range all {
		if j.Amount > 0 {
			out = append(out, j)
		}
	}
	return out
}

// PayrollDescription is "Payroll MM/YYYY: <label>" (NüHabit used an em dash).
func PayrollDescription(p PayrollPaid, label string) string {
	return fmt.Sprintf("Payroll %02d/%d: %s", p.PeriodMonth, p.PeriodYear, label)
}

// SalesInvoiceIssued is the payload of white.sales.invoice.issued. Finance
// creates one POSTED AR invoice per invoiceId and posts SALE_AR_INVOICE.
//
//	{
//	  "invoiceId": "...", "invoiceNumber": "INV/2026/10/0012",
//	  "dealId": "...", "clientId": "...", "customerName": "PT Contoh",
//	  "issueDate": "2026-10-20", "dueDate": "2026-11-19",
//	  "subtotal": "15000000.00", "taxAmount": "1650000.00",
//	  "totalAmount": "16650000.00", "description": "Retainer SEO Oktober",
//	  "issuedBy": "<user id>"
//	}
type SalesInvoiceIssued struct {
	InvoiceID     string `json:"invoiceId"`
	InvoiceNumber string `json:"invoiceNumber"`
	DealID        string `json:"dealId,omitempty"`
	ClientID      string `json:"clientId,omitempty"`
	CustomerName  string `json:"customerName"`
	IssueDate     string `json:"issueDate"`
	DueDate       string `json:"dueDate,omitempty"`
	Subtotal      Money  `json:"subtotal"`
	TaxAmount     Money  `json:"taxAmount"`
	TotalAmount   Money  `json:"totalAmount"`
	Description   string `json:"description,omitempty"`
	IssuedBy      string `json:"issuedBy,omitempty"`
}

// SalesPaymentRecorded is the payload of white.sales.payment.recorded.
// Finance records one AR receipt per paymentId against the AR invoice of
// invoiceId and posts SALE_AR_RECEIPT.
//
//	{
//	  "paymentId": "...", "invoiceId": "...", "amount": "16650000.00",
//	  "paidOn": "2026-10-28", "method": "transfer",
//	  "reference": "BCA 123456", "recordedBy": "<user id>"
//	}
type SalesPaymentRecorded struct {
	PaymentID  string `json:"paymentId"`
	InvoiceID  string `json:"invoiceId"`
	Amount     Money  `json:"amount"`
	PaidOn     string `json:"paidOn"`
	Method     string `json:"method,omitempty"`
	Reference  string `json:"reference,omitempty"`
	RecordedBy string `json:"recordedBy,omitempty"`
}

// ReimbursementPaid is the payload finance publishes on
// white.finance.reimbursement.paid after a payment batch.
type ReimbursementPaid struct {
	BatchID     string                 `json:"batchId"`
	BatchNo     string                 `json:"batchNo"`
	PaidAt      string                 `json:"paidAt"`
	TotalAmount Money                  `json:"totalAmount"`
	ClaimIDs    []string               `json:"claimIds"`
	EmployeeIDs []string               `json:"employeeIds"`
	Amounts     []Money                `json:"amounts"`
	Claims      []ReimbursementPaidRow `json:"claims"`
}

// ReimbursementPaidRow is one paid claim; ClaimIDs, EmployeeIDs and Amounts
// above are the same rows as parallel arrays.
type ReimbursementPaidRow struct {
	ClaimID    string `json:"claimId"`
	ClaimNo    string `json:"claimNo"`
	EmployeeID string `json:"employeeId"`
	Category   string `json:"category"`
	Amount     Money  `json:"amount"`
}
