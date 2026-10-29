package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/salesevents"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Billing is EPIC-025 "Opsi B" of salesfunnel billing.go: sales submits an
// invoice (diajukan), finance issues it (terkirim, published for the AR
// side) or cancels it (batal, refused once paid); payments fill the
// quotation terms in order.
type Billing struct {
	*Deps
	onboarding *Onboarding
}

func NewBilling(d *Deps, onboarding *Onboarding) *Billing {
	return &Billing{Deps: d, onboarding: onboarding}
}

// Invoice statuses.
const (
	InvoiceSubmitted = "diajukan"
	InvoiceSent      = "terkirim"
	InvoiceCancelled = "batal"
)

func (b *Billing) ListInvoices(ctx context.Context, f domain.InvoiceFilter) ([]domain.Invoice, error) {
	return b.Store.ListInvoices(ctx, f)
}

type InvoiceInput struct {
	DealID, QuotationID, TermID, Label, Amount, DueDate string
}

func (b *Billing) SubmitInvoice(ctx context.Context, actor Actor, in InvoiceInput) (*domain.Invoice, error) {
	var out *domain.Invoice
	err := b.Store.WithTx(ctx, func(st domain.Store) error {
		inv, err := b.submit(ctx, st, actor, in)
		out = inv
		return err
	})
	if err != nil {
		return nil, err
	}
	return b.Store.GetInvoice(ctx, out.ID)
}

func (b *Billing) submit(ctx context.Context, st domain.Store, actor Actor, in InvoiceInput) (*domain.Invoice, error) {
	deal, err := st.GetDeal(ctx, in.DealID)
	if err != nil {
		return nil, err
	}
	if deal == nil {
		return nil, notFound(msgDealNotFound)
	}
	due, err := domain.ParseDate(in.DueDate)
	if err != nil {
		return nil, apperr.Field("dueDate", "Tanggal jatuh tempo tidak valid")
	}
	inv := &domain.Invoice{ID: ids.New(), DealID: deal.ID, Label: domain.JSTrim(in.Label), DueDate: due, Status: InvoiceSubmitted}
	if in.QuotationID != "" {
		qt, err := st.GetQuotation(ctx, in.QuotationID)
		if err != nil {
			return nil, err
		}
		if qt == nil || qt.DealID != deal.ID {
			return nil, notFound(msgQuoteNotFound)
		}
		if qt.Status != domain.QuoteAccepted {
			return nil, conflict(msgQuoteNeedAccepted)
		}
		inv.QuotationID = &qt.ID
		if in.TermID != "" {
			percents := make([]float64, len(qt.Terms))
			idx := -1
			for i, t := range qt.Terms {
				percents[i] = t.Percent
				if t.ID == in.TermID {
					idx = i
				}
			}
			if idx < 0 {
				return nil, notFound("Termin tidak ditemukan")
			}
			existing, err := st.ListInvoices(ctx, domain.InvoiceFilter{DealID: deal.ID, Limit: 500})
			if err != nil {
				return nil, err
			}
			for _, e := range existing {
				if e.TermID != nil && *e.TermID == in.TermID && e.Status != InvoiceCancelled {
					return nil, conflict("Termin ini sudah ditagihkan")
				}
			}
			term := qt.Terms[idx]
			inv.TermID, inv.Amount = &term.ID, domain.AllocateTermAmounts(qt.Total, percents)[idx]
			if inv.Label == "" {
				inv.Label = term.Label
			}
			if inv.DueDate == nil {
				inv.DueDate = term.DueDate
			}
		}
	}
	if inv.TermID == nil {
		amount, err := domain.ParseMoney(in.Amount)
		if err != nil || amount <= 0 {
			return nil, apperr.Field("amount", "Nominal invoice harus lebih dari 0")
		}
		inv.Amount = amount
	}
	if inv.Label == "" {
		return nil, apperr.Field("label", "Label invoice wajib diisi")
	}
	if inv.Amount <= 0 {
		return nil, apperr.Field("amount", "Nominal invoice harus lebih dari 0")
	}
	prefix := domain.DocumentPrefix(domain.NumberInvoice, b.now())
	n, err := st.NextNumber(ctx, prefix)
	if err != nil {
		return nil, err
	}
	now := b.now()
	inv.InvoiceNumber = domain.DocumentNumber(prefix, n)
	inv.Audit = domain.Audit{CreatedBy: actor.ref(), UpdatedBy: actor.ref(), CreatedAt: now, UpdatedAt: now}
	return inv, st.SaveInvoice(ctx, inv)
}

// SubmitFromTerms submits one invoice per term of an accepted quotation
// that has no active invoice yet.
func (b *Billing) SubmitFromTerms(ctx context.Context, actor Actor, quotationID string) ([]domain.Invoice, error) {
	qt, err := b.Store.GetQuotation(ctx, quotationID)
	if err != nil {
		return nil, err
	}
	if qt == nil {
		return nil, notFound(msgQuoteNotFound)
	}
	if qt.Status != domain.QuoteAccepted {
		return nil, conflict(msgQuoteNeedAccepted)
	}
	if len(qt.Terms) == 0 {
		return nil, invalid("Quotation tidak punya termin pembayaran")
	}
	err = b.Store.WithTx(ctx, func(st domain.Store) error {
		existing, err := st.ListInvoices(ctx, domain.InvoiceFilter{DealID: qt.DealID, Limit: 500})
		if err != nil {
			return err
		}
		billed := map[string]bool{}
		for _, e := range existing {
			if e.TermID != nil && e.Status != InvoiceCancelled {
				billed[*e.TermID] = true
			}
		}
		for _, t := range qt.Terms {
			if billed[t.ID] {
				continue
			}
			if _, err := b.submit(ctx, st, actor, InvoiceInput{DealID: qt.DealID, QuotationID: qt.ID, TermID: t.ID}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return b.Store.ListInvoices(ctx, domain.InvoiceFilter{DealID: qt.DealID, Limit: 500})
}

func (b *Billing) load(ctx context.Context, id string) (*domain.Invoice, error) {
	inv, err := b.Store.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, notFound(msgInvoiceNotFound)
	}
	return inv, nil
}

// IssueInvoice is finance issuing a submitted invoice; it publishes
// white.sales.invoice.issued so finance books the receivable.
func (b *Billing) IssueInvoice(ctx context.Context, actor Actor, id string) (*domain.Invoice, error) {
	inv, err := b.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != InvoiceSubmitted {
		return nil, conflict("Hanya invoice berstatus diajukan yang bisa diterbitkan")
	}
	now := b.now()
	inv.Status, inv.SentAt, inv.IssuedBy, inv.UpdatedBy, inv.UpdatedAt = InvoiceSent, &now, actor.ref(), actor.ref(), now
	if err := b.Store.SaveInvoice(ctx, inv); err != nil {
		return nil, err
	}
	payload, err := b.issuedPayload(ctx, *inv, actor, now)
	if err != nil {
		return nil, err
	}
	if err := b.publish(ctx, events.SalesInvoiceIssued, payload); err != nil {
		return nil, err
	}
	if deal, err := b.Store.GetDeal(ctx, inv.DealID); err == nil && deal != nil && deal.OwnerUserID != nil {
		b.notify(ctx, Notification{UserIDs: []string{*deal.OwnerUserID}, Type: NotifInvoice, Title: "Invoice " + inv.InvoiceNumber + " diterbitkan", Body: inv.Label + " · " + inv.Amount.Rupiah(), Href: "/erp/sales/pipeline/" + inv.DealID})
	}
	return b.Store.GetInvoice(ctx, id)
}

// issuedPayload is the finance contract (finance/domain.SalesInvoiceIssued).
func (b *Billing) issuedPayload(ctx context.Context, inv domain.Invoice, actor Actor, now time.Time) (salesevents.InvoiceIssued, error) {
	var tax domain.Money
	if inv.QuotationID != nil {
		qt, err := b.Store.GetQuotation(ctx, *inv.QuotationID)
		if err != nil {
			return salesevents.InvoiceIssued{}, err
		}
		if qt != nil && qt.UsePpn && qt.Total > 0 {
			tax = domain.Money(roundDiv(int64(inv.Amount)*int64(qt.PpnNominal), int64(qt.Total)))
		}
	}
	p := salesevents.InvoiceIssued{InvoiceID: inv.ID, InvoiceNumber: inv.InvoiceNumber, DealID: inv.DealID, CustomerName: inv.ClientName,
		IssueDate: now.In(domain.Jakarta).Format("2006-01-02"), DueDate: domain.FormatDay(inv.DueDate),
		Subtotal: (inv.Amount - tax).String(), TaxAmount: tax.String(), TotalAmount: inv.Amount.String(),
		Description: inv.Label + " · " + inv.DealTitle, IssuedBy: actor.UserID}
	if ob, err := b.Store.GetOnboardingByDeal(ctx, inv.DealID); err != nil {
		return p, err
	} else if ob != nil {
		p.ClientID = deref(ob.MarketingClientID)
	}
	return p, nil
}

func roundDiv(a, b int64) int64 {
	if b == 0 {
		return 0
	}
	return (a*2 + b) / (2 * b)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func (b *Billing) CancelInvoice(ctx context.Context, actor Actor, id, reason string) (*domain.Invoice, error) {
	inv, err := b.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv.Status == InvoiceCancelled {
		return nil, conflict("Invoice sudah dibatalkan")
	}
	if inv.PaidTotal > 0 {
		return nil, conflict("Invoice yang sudah ada pembayarannya tidak bisa dibatalkan")
	}
	now := b.now()
	inv.Status, inv.CancelledAt, inv.CancelledBy, inv.CancelReason = InvoiceCancelled, &now, actor.ref(), domain.TrimmedOrNil(reason)
	inv.UpdatedBy, inv.UpdatedAt = actor.ref(), now
	if err := b.Store.SaveInvoice(ctx, inv); err != nil {
		return nil, err
	}
	return b.Store.GetInvoice(ctx, id)
}

func (b *Billing) RenderInvoicePDF(ctx context.Context, id string) (string, []byte, error) {
	inv, err := b.load(ctx, id)
	if err != nil {
		return "", nil, err
	}
	deal, err := b.Store.GetDeal(ctx, inv.DealID)
	if err != nil {
		return "", nil, err
	}
	if deal == nil {
		return "", nil, notFound(msgDealNotFound)
	}
	data, err := b.Renderer.InvoicePDF(*inv, *deal)
	if err != nil {
		return "", nil, err
	}
	return fileName(inv.InvoiceNumber, inv.ClientName, "pdf"), data, nil
}

// ─── payments ────────────────────────────────────────────────

func (b *Billing) ListPayments(ctx context.Context, dealID string) ([]domain.Payment, domain.Money, error) {
	ps, err := b.Store.ListPayments(ctx, dealID)
	if err != nil {
		return nil, 0, err
	}
	total, err := b.Store.PaidTotal(ctx, dealID)
	return ps, total, err
}

type PaymentInput struct {
	DealID, InvoiceID, Amount, Method, PaidOn, Note string
}

// RecordPayment stores a payment and publishes white.sales.payment.recorded.
func (b *Billing) RecordPayment(ctx context.Context, actor Actor, in PaymentInput) (*domain.Payment, error) {
	deal, err := b.Store.GetDeal(ctx, in.DealID)
	if err != nil {
		return nil, err
	}
	if deal == nil {
		return nil, notFound(msgDealNotFound)
	}
	errs := map[string]string{}
	amount, err := domain.ParseMoney(in.Amount)
	if err != nil || amount <= 0 {
		errs["amount"] = "Nominal pembayaran harus lebih dari 0"
	}
	if !domain.Contains(domain.PaymentMethods, in.Method) {
		errs["method"] = "Metode pembayaran tidak valid"
	}
	paidOn, err := domain.ParseDate(in.PaidOn)
	if err != nil {
		errs["paidOn"] = "Tanggal bayar tidak valid"
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	if paidOn == nil {
		d := domain.DueDate(b.now(), 0)
		paidOn = &d
	}
	var inv *domain.Invoice
	if in.InvoiceID != "" {
		if inv, err = b.Store.GetInvoice(ctx, in.InvoiceID); err != nil {
			return nil, err
		}
		if inv == nil || inv.DealID != deal.ID {
			return nil, notFound(msgInvoiceNotFound)
		}
		if inv.Status == InvoiceCancelled {
			return nil, conflict("Invoice sudah dibatalkan")
		}
	}
	now := b.now()
	p := &domain.Payment{ID: ids.New(), DealID: deal.ID, InvoiceID: domain.TrimmedOrNil(in.InvoiceID), Amount: amount, Method: in.Method, PaidOn: *paidOn, Note: domain.TrimmedOrNil(in.Note), CreatedBy: actor.ref(), CreatedAt: now}
	if err := b.Store.SavePayment(ctx, p); err != nil {
		return nil, err
	}
	// Finance books receipts against AR invoices, so only invoiced
	// payments are published.
	if inv != nil {
		payload := salesevents.PaymentRecorded{PaymentID: p.ID, InvoiceID: inv.ID, Amount: amount.String(), PaidOn: p.PaidOn.Format("2006-01-02"), Method: p.Method, Reference: deref(p.Note), RecordedBy: actor.UserID}
		if err := b.publish(ctx, events.SalesPaymentRecorded, payload); err != nil {
			return nil, err
		}
	}
	b.onboarding.OnPayment(ctx, deal.ID)
	return p, nil
}

// DeletePayment voids a payment. Finance has no receipt reversal event
// yet, so nothing is published.
func (b *Billing) DeletePayment(ctx context.Context, id string) error {
	p, err := b.Store.GetPayment(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return notFound("Pembayaran tidak ditemukan")
	}
	return b.Store.DeletePayment(ctx, id, b.now())
}
