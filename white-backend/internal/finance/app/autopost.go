package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
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
		if p.DealID != "" {
			if inv.TagID, err = tx.EnsureDealTag(ctx, domain.ProjectTag{ID: ids.New(), DealID: p.DealID, ClientName: inv.PartyName, Name: dealTagName(inv.Description)}); err != nil {
				return err
			}
		}
		if err := tx.InsertInvoice(ctx, inv); err != nil {
			return err
		}
		if err := a.tagSalesTax(ctx, tx, inv, p.TaxCode); err != nil {
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

// dealTagName takes the deal title from "label · deal title".
func dealTagName(description string) string {
	if _, title, ok := strings.Cut(description, " · "); ok && trim(title) != "" {
		return trim(title)
	}
	return description
}

// tagSalesTax books a sales invoice's PPN under the code the quotation chose. An
// unknown code leaves the mapping's PPN line, so the invoice still posts.
func (a *AutoPost) tagSalesTax(ctx context.Context, tx domain.Store, inv domain.Invoice, code string) error {
	if code == "" || inv.TaxAmount <= 0 {
		return nil
	}
	t, err := tx.GetTaxCodeByCode(ctx, code)
	if err != nil || t == nil || t.AccountSaleID == "" {
		return err
	}
	dt := domain.DocTax{TaxID: t.ID, Code: t.Code, Name: t.Name, Kind: t.Kind, Rate: t.Rate, ReportCode: t.ReportCode, AccountID: t.AccountSaleID, CashBasis: t.CashBasis, TransitionAccountID: t.TransitionAccountID, Amount: inv.TaxAmount,
		Base: domain.MulDiv(inv.Subtotal, domain.Money(t.BaseNumerator), domain.Money(max(1, t.BaseDenominator)))}
	postings, err := taxPostings([]domain.DocTax{dt}, domain.KindAR)
	if err != nil {
		return err
	}
	return insertPostings(ctx, tx, inv.ID, postings)
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
	accounts, err := a.Store.AccountsByCode(ctx)
	if err != nil {
		return nil, err
	}
	excessAccount := accounts[salesAdvanceAccountCode].ID
	if amount <= 0 {
		return a.salesPaymentAdvance(ctx, p, *inv, excessAccount)
	}
	method := p.Method
	if !slices.Contains(kinds[domain.KindAR].methods, method) {
		method = "other"
	}
	in := PaymentInput{PaymentDate: p.PaidOn, Method: method, Reference: p.Reference, Notes: "via Sales " + p.PaymentID, SalesPaymentID: p.PaymentID}
	if excessAccount != "" {
		in.Excess, in.ExcessAccountID = p.Amount-amount, excessAccount
	}
	l := NewLedgers(a.Deps)
	in.Adjustments, amount, err = a.salesAdjustments(ctx, l, p, accounts[salesBankFeeAccountCode].ID, inv.ID, amount)
	if err != nil {
		return nil, err
	}
	in.Allocations = []domain.Allocation{{InvoiceID: inv.ID, Amount: amount}}
	res, err := l.CreatePayment(ctx, domain.KindAR, p.RecordedBy, in)
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

// Default accounts for what Sales cannot pick itself.
const (
	salesAdvanceAccountCode = "2105001"
	salesBankFeeAccountCode = "8101001"
)

// salesAdjustments turns the event's fee and withheld tax into adjustments. Sales
// already took the money, so one Finance cannot accept is left out and its share
// stays open on the invoice for Finance to settle.
func (a *AutoPost) salesAdjustments(ctx context.Context, l *Ledgers, p domain.SalesPaymentRecorded, feeAccount, invoiceID string, amount domain.Money) ([]AdjustmentInput, domain.Money, error) {
	var wanted []AdjustmentInput
	if p.BankFee > 0 {
		wanted = append(wanted, AdjustmentInput{Kind: domain.AdjustFee, AccountID: feeAccount, Amount: p.BankFee})
	}
	if p.Withheld > 0 {
		t, err := a.Store.GetTaxCodeByCode(ctx, p.WithheldTaxCode)
		if err != nil {
			return nil, 0, err
		}
		taxID := ""
		if t != nil {
			taxID = t.ID
		}
		wanted = append(wanted, AdjustmentInput{Kind: domain.AdjustWithholding, TaxID: taxID, Amount: p.Withheld})
	}
	var kept []AdjustmentInput
	for _, adj := range wanted {
		trial := PaymentInput{Adjustments: append(slices.Clone(kept), adj), Allocations: []domain.Allocation{{InvoiceID: invoiceID, Amount: amount}}}
		if _, err := l.resolveAdjustments(ctx, domain.KindAR, trial); err != nil {
			if apperr.As(err).Code == apperr.Unknown {
				return nil, 0, err
			}
			if amount > adj.Amount {
				amount -= adj.Amount
			}
			continue
		}
		kept = append(kept, adj)
	}
	return kept, amount, nil
}

// salesPaymentAdvance keeps a Sales payment that reached an invoice with nothing
// left to settle. The money is real, so it waits as an advance for the party.
func (a *AutoPost) salesPaymentAdvance(ctx context.Context, p domain.SalesPaymentRecorded, inv domain.Invoice, advanceAccount string) (*PaymentResult, error) {
	existing, err := a.Store.AdvanceBySalesPayment(ctx, p.PaymentID)
	if err != nil || existing != nil {
		return &PaymentResult{Note: "uang muka sudah tercatat"}, err
	}
	if advanceAccount == "" {
		return nil, permanent("akun uang muka %s tidak ada untuk pembayaran %s", salesAdvanceAccountCode, p.PaymentID)
	}
	mapping, err := a.Store.GetActiveMapping(ctx, kinds[domain.KindAR].paymentEvent)
	if err != nil {
		return nil, err
	}
	cash := ""
	if mapping != nil {
		for _, line := range mapping.Template() {
			if line.LineRole == "CASH" {
				cash = line.AccountID
			}
		}
	}
	created, err := NewLedgers(a.Deps).CreateAdvance(ctx, p.RecordedBy, AdvanceInput{Kind: domain.KindAR, Date: p.PaidOn, PartyName: inv.PartyName, CashAccountID: cash, AdvanceAccountID: advanceAccount, Reference: p.Reference,
		Notes: "Kelebihan bayar " + inv.InvoiceNo + " via Sales " + p.PaymentID, Amount: p.Amount, SalesPaymentID: p.PaymentID})
	if err != nil {
		return nil, err
	}
	return &PaymentResult{Note: "dicatat sebagai uang muka " + created.AdvanceNo}, nil
}
