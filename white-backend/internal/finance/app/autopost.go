package app

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// AutoPost turns other services' events into journals, idempotent per
// source document (NüHabit service_payroll.go, CreateArFromSalesInvoice,
// RecordArReceipt).
type AutoPost struct{ Deps }

func NewAutoPost(d Deps) *AutoPost { return &AutoPost{d} }

// ErrPermanent marks an event that can never be processed; the worker
// acknowledges it instead of retrying.
var ErrPermanent = errors.New("event tidak bisa diproses")

func permanent(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrPermanent}, args...)...)
}

// PayrollPaid posts one journal per non-zero component of a paid run,
// dated the day it was paid in Jakarta, keyed by the run id.
func (a *AutoPost) PayrollPaid(ctx context.Context, p domain.PayrollPaid) ([]domain.PostResult, error) {
	if p.RunID == "" || p.PaidAt.IsZero() {
		return nil, permanent("payroll paid tanpa runId/paidAt")
	}
	date := p.PaidAt.In(Jakarta).Format("2006-01-02")
	label := p.Label
	if label == "" {
		label = fmt.Sprintf("Payroll %02d/%d", p.PeriodMonth, p.PeriodYear)
	}
	var results []domain.PostResult
	for _, j := range domain.PayrollJournals(p) {
		res, err := PostFromMapping(ctx, a.Store, MappingPost{
			UserID: p.PaidBy, EventCode: j.Event, DocumentType: DocPayrollRun, DocumentID: p.RunID, DocumentLabel: label,
			EntryDate: date, Description: domain.PayrollDescription(p, j.Label), Module: "PAYROLL", Amounts: domain.Amounts{j.Source: j.Amount},
		})
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}

// SalesInvoiceIssued creates one POSTED AR invoice per sales invoice and
// posts SALE_AR_INVOICE in the same transaction.
func (a *AutoPost) SalesInvoiceIssued(ctx context.Context, p domain.SalesInvoiceIssued) (*domain.Invoice, error) {
	if p.InvoiceID == "" {
		return nil, permanent("sales invoice tanpa invoiceId")
	}
	if existing, err := a.Store.GetInvoiceBySalesID(ctx, p.InvoiceID); err != nil || existing != nil {
		return existing, err
	}
	total := p.TotalAmount
	if total <= 0 {
		total = p.Subtotal + p.TaxAmount
	}
	if total <= 0 {
		return nil, permanent("nilai invoice %s 0", p.InvoiceNumber)
	}
	subtotal := p.Subtotal
	if subtotal <= 0 {
		subtotal = total - p.TaxAmount
	}
	date := p.IssueDate
	if _, err := parseDay("issueDate", date); err != nil {
		date = a.today()
	}
	due := p.DueDate
	if _, err := parseDay("dueDate", due); err != nil {
		due = ""
	}
	inv := domain.Invoice{
		ID: ids.New(), Kind: domain.KindAR, InvoiceDate: date, DueDate: due, PartyName: orDefault(p.CustomerName, "Customer"),
		Reference: p.InvoiceNumber, Currency: "IDR", Subtotal: subtotal, TaxAmount: p.TaxAmount, TotalAmount: total,
		Status: domain.StatusPosted, Description: orDefault(p.Description, "AR dari B2B "+p.InvoiceNumber), SalesInvoiceID: p.InvoiceID,
		ClientID: p.ClientID, DealID: p.DealID, CreatedBy: p.IssuedBy,
	}
	ledgers := NewLedgers(a.Deps)
	err := a.Store.WithTx(ctx, func(tx domain.Store) error {
		no, err := ledgers.nextNo(ctx, tx, kinds[domain.KindAR].invoicePrefix)
		if err != nil {
			return err
		}
		inv.InvoiceNo = no
		if err := tx.InsertInvoice(ctx, inv); err != nil {
			return err
		}
		_, err = ledgers.postInvoiceJournal(ctx, tx, inv, p.IssuedBy)
		return err
	})
	var unique *domain.ErrUnique
	if errors.As(err, &unique) {
		return a.Store.GetInvoiceBySalesID(ctx, p.InvoiceID)
	}
	if err != nil {
		return nil, err
	}
	return a.Store.GetInvoice(ctx, inv.ID)
}

// SalesPaymentRecorded records one AR receipt per sales payment against
// the AR invoice of the sales invoice and posts SALE_AR_RECEIPT. A payment
// that arrives before its invoice fails and is redelivered.
func (a *AutoPost) SalesPaymentRecorded(ctx context.Context, p domain.SalesPaymentRecorded) (*PaymentResult, error) {
	if p.PaymentID == "" || p.InvoiceID == "" {
		return nil, permanent("sales payment tanpa paymentId/invoiceId")
	}
	if existing, err := a.Store.GetPaymentBySalesID(ctx, p.PaymentID); err != nil || existing != nil {
		if existing == nil {
			return nil, err
		}
		return &PaymentResult{Payment: *existing}, nil
	}
	inv, err := a.Store.GetInvoiceBySalesID(ctx, p.InvoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, fmt.Errorf("AR invoice untuk sales invoice %s belum ada", p.InvoiceID)
	}
	amount := min(p.Amount, inv.Outstanding())
	if amount <= 0 {
		return nil, permanent("pembayaran %s tidak punya sisa tagihan", p.PaymentID)
	}
	method := p.Method
	if !slices.Contains(kinds[domain.KindAR].methods, method) {
		method = "other"
	}
	res, err := NewLedgers(a.Deps).CreatePayment(ctx, domain.KindAR, p.RecordedBy, PaymentInput{
		PaymentDate: p.PaidOn, Method: method, Reference: p.Reference, Notes: "via Sales " + p.PaymentID,
		Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: amount}}, SalesPaymentID: p.PaymentID,
	})
	var unique *domain.ErrUnique
	if errors.As(err, &unique) {
		existing, gerr := a.Store.GetPaymentBySalesID(ctx, p.PaymentID)
		if gerr != nil || existing == nil {
			return nil, err
		}
		return &PaymentResult{Payment: *existing}, nil
	}
	return res, err
}
