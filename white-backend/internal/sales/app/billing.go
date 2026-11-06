package app

import (
	"context"
	"net/http"
	"time"
	"unicode/utf8"

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

// GetInvoice returns one invoice with its paid total, or NotFound.
func (b *Billing) GetInvoice(ctx context.Context, id string) (*domain.Invoice, error) {
	return b.load(ctx, id)
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
	taxCode := ""
	if inv.QuotationID != nil {
		qt, err := b.Store.GetQuotation(ctx, *inv.QuotationID)
		if err != nil {
			return salesevents.InvoiceIssued{}, err
		}
		if qt != nil && qt.UsePpn && qt.Total > 0 {
			tax = domain.Money(roundDiv(int64(inv.Amount)*int64(qt.PpnNominal), int64(qt.Total)))
			taxCode = qt.TaxCode
		}
	}
	p := salesevents.InvoiceIssued{InvoiceID: inv.ID, InvoiceNumber: inv.InvoiceNumber, DealID: inv.DealID, CustomerName: inv.ClientName,
		IssueDate: now.In(domain.Jakarta).Format("2006-01-02"), DueDate: domain.FormatDay(inv.DueDate),
		Subtotal: (inv.Amount - tax).String(), TaxAmount: tax.String(), TotalAmount: inv.Amount.String(),
		Description: inv.Label + " · " + inv.DealTitle, IssuedBy: actor.UserID, TaxCode: taxCode}
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
	booked := inv.Status != InvoiceSubmitted
	inv.Status, inv.CancelledAt, inv.CancelledBy, inv.CancelReason = InvoiceCancelled, &now, actor.ref(), domain.TrimmedOrNil(reason)
	inv.UpdatedBy, inv.UpdatedAt = actor.ref(), now
	if err := b.Store.SaveInvoice(ctx, inv); err != nil {
		return nil, err
	}
	// Finance booked it when it was issued, so finance has to take it back out.
	if booked {
		if err := b.publish(ctx, events.SalesInvoiceCancelled, salesevents.InvoiceCancelled{InvoiceID: inv.ID, Reason: reason, CancelledBy: actor.UserID}); err != nil {
			return nil, err
		}
	}
	return b.Store.GetInvoice(ctx, id)
}

// InvoiceRevision is what finance may change on a submitted invoice.
type InvoiceRevision struct {
	Label, Amount, DueDate, Note string
}

// UpdateInvoice revises a submitted invoice. An issued invoice is already
// booked as a receivable and a paid one has money against its amount, so
// neither can change. A term invoice keeps the amount allocated to its term.
func (b *Billing) UpdateInvoice(ctx context.Context, actor Actor, id string, in InvoiceRevision) (*domain.Invoice, error) {
	inv, err := b.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != InvoiceSubmitted {
		return nil, conflict("Hanya invoice berstatus diajukan yang bisa direvisi")
	}
	if inv.PaidTotal > 0 {
		return nil, conflict("Invoice yang sudah ada pembayarannya tidak bisa direvisi")
	}
	errs := map[string]string{}
	label := domain.JSTrim(in.Label)
	if label == "" || utf8.RuneCountInString(label) > 150 {
		errs["label"] = "Keterangan wajib diisi, maksimal 150 karakter"
	}
	amount, err := domain.ParseMoney(in.Amount)
	if err != nil || amount <= 0 {
		errs["amount"] = "Nominal invoice harus lebih dari 0"
	} else if inv.TermID != nil && amount != inv.Amount {
		errs["amount"] = "Nominal invoice termin mengikuti quotation"
	}
	due, err := domain.ParseDate(in.DueDate)
	if err != nil {
		errs["dueDate"] = "Tanggal jatuh tempo tidak valid"
	}
	if utf8.RuneCountInString(in.Note) > 300 {
		errs["note"] = "Catatan maksimal 300 karakter"
	}
	if err := fieldErrs(errs); err != nil {
		return nil, err
	}
	inv.Label, inv.Amount, inv.DueDate, inv.Note = label, amount, due, domain.TrimmedOrNil(in.Note)
	inv.UpdatedBy, inv.UpdatedAt = actor.ref(), b.now()
	if err := b.Store.SaveInvoice(ctx, inv); err != nil {
		return nil, err
	}
	return b.Store.GetInvoice(ctx, id)
}

// fakturPajakMaxBytes caps the tax invoice attachment.
const fakturPajakMaxBytes = 10 << 20

var fakturPajakTypes = []string{"application/pdf", "image/jpeg", "image/png", "image/webp"}

// UploadFakturPajak attaches the tax invoice. The key is fixed per invoice, so
// a new file replaces the old one.
func (b *Billing) UploadFakturPajak(ctx context.Context, actor Actor, id, name string, data []byte) (*domain.Invoice, error) {
	inv, err := b.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv.Status == InvoiceCancelled {
		return nil, conflict("Invoice sudah dibatalkan")
	}
	if len(data) == 0 || len(data) > fakturPajakMaxBytes {
		return nil, apperr.Field("file", "Ukuran file maksimal 10 MB")
	}
	if !domain.Contains(fakturPajakTypes, http.DetectContentType(data)) {
		return nil, apperr.Field("file", "Format file harus PDF, JPG, PNG, atau WebP")
	}
	key := "invoices/" + inv.ID + "/faktur-pajak"
	if err := b.Files.Put(ctx, key, data); err != nil {
		return nil, err
	}
	safe := domain.SafeFileName(name)
	inv.FakturPajakKey, inv.FakturPajakName = &key, &safe
	inv.UpdatedBy, inv.UpdatedAt = actor.ref(), b.now()
	if err := b.Store.SaveInvoice(ctx, inv); err != nil {
		return nil, err
	}
	return b.Store.GetInvoice(ctx, id)
}

// FakturPajak returns the attached file's name, type and bytes.
func (b *Billing) FakturPajak(ctx context.Context, id string) (string, string, []byte, error) {
	inv, err := b.load(ctx, id)
	if err != nil {
		return "", "", nil, err
	}
	if inv.FakturPajakKey == nil {
		return "", "", nil, notFound("Faktur pajak belum diunggah")
	}
	data, err := b.Files.Get(ctx, *inv.FakturPajakKey)
	if err != nil {
		return "", "", nil, err
	}
	return deref(inv.FakturPajakName), http.DetectContentType(data), data, nil
}

func (b *Billing) DeleteFakturPajak(ctx context.Context, actor Actor, id string) (*domain.Invoice, error) {
	inv, err := b.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv.FakturPajakKey == nil {
		return inv, nil
	}
	key := *inv.FakturPajakKey
	inv.FakturPajakKey, inv.FakturPajakName = nil, nil
	inv.UpdatedBy, inv.UpdatedAt = actor.ref(), b.now()
	if err := b.Store.SaveInvoice(ctx, inv); err != nil {
		return nil, err
	}
	if err := b.Files.Delete(ctx, key); err != nil {
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
	quoteNumber := ""
	if inv.QuotationID != nil {
		if qt, err := b.Store.GetQuotation(ctx, *inv.QuotationID); err != nil {
			return "", nil, err
		} else if qt != nil {
			quoteNumber = qt.QuoteNumber
		}
	}
	data, err := b.Renderer.InvoicePDF(*inv, *deal, quoteNumber)
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
	// BankFee and Withheld are the parts of Amount that never reached the bank.
	BankFee, Withheld, WithheldTaxCode string
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
	bankFee, withheld := parseSettlement(in, amount, errs)
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
		payload := salesevents.PaymentRecorded{PaymentID: p.ID, InvoiceID: inv.ID, Amount: amount.String(), PaidOn: p.PaidOn.Format("2006-01-02"), Method: p.Method, Reference: deref(p.Note), RecordedBy: actor.UserID, WithheldTaxCode: in.WithheldTaxCode}
		if bankFee > 0 {
			payload.BankFee = bankFee.String()
		}
		if withheld > 0 {
			payload.Withheld = withheld.String()
		}
		if err := b.publish(ctx, events.SalesPaymentRecorded, payload); err != nil {
			return nil, err
		}
	}
	b.onboarding.OnPayment(ctx, deal.ID)
	return p, nil
}

// parseSettlement reads the parts of a payment that did not reach the bank.
func parseSettlement(in PaymentInput, amount domain.Money, errs map[string]string) (bankFee, withheld domain.Money) {
	read := func(field, value string) domain.Money {
		if value == "" {
			return 0
		}
		v, err := domain.ParseMoney(value)
		if err != nil || v < 0 {
			errs[field] = "Nominal tidak valid"
			return 0
		}
		return v
	}
	bankFee, withheld = read("bankFee", in.BankFee), read("withheld", in.Withheld)
	if bankFee+withheld > 0 && in.InvoiceID == "" {
		errs["invoiceId"] = "Biaya bank dan PPh dipotong butuh invoice"
	}
	if withheld > 0 && in.WithheldTaxCode == "" {
		errs["withheldTaxCode"] = "Pilih kode PPh"
	}
	if amount > 0 && bankFee+withheld >= amount {
		errs["amount"] = "Biaya bank dan PPh dipotong harus lebih kecil dari nominal"
	}
	return bankFee, withheld
}

// DeletePayment voids a payment and, for one recorded against an invoice, tells
// finance to reverse its receipt.
func (b *Billing) DeletePayment(ctx context.Context, actor Actor, id string) error {
	p, err := b.Store.GetPayment(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return notFound("Pembayaran tidak ditemukan")
	}
	if err := b.Store.DeletePayment(ctx, id, b.now()); err != nil {
		return err
	}
	if p.InvoiceID == nil {
		return nil
	}
	return b.publish(ctx, events.SalesPaymentDeleted, salesevents.PaymentDeleted{PaymentID: p.ID, DeletedBy: actor.UserID})
}

// ─── follow-ups ────────────────────────────────

func (b *Billing) WhatsAppEnabled() bool { return b.WhatsApp != nil && b.WhatsApp.Enabled() }

func (b *Billing) ListFollowups(ctx context.Context, status string, limit int) ([]domain.InvoiceFollowup, error) {
	return b.Store.ListFollowups(ctx, status, limit)
}

// QueueFollowups drafts one message per stage for issued, unpaid invoices. The
// draft waits for finance; it never goes out by itself.
func (b *Billing) QueueFollowups(ctx context.Context) (int, error) {
	invoices, err := b.Store.ListInvoices(ctx, domain.InvoiceFilter{Status: InvoiceSent, Limit: 2000})
	if err != nil {
		return 0, err
	}
	queued := 0
	for _, inv := range invoices {
		if inv.DueDate == nil || inv.PaidTotal >= inv.Amount {
			continue
		}
		stage, ok := domain.FollowupStageDue(*inv.DueDate, b.now())
		if !ok {
			continue
		}
		pic, phone, err := b.billingContact(ctx, inv.DealID)
		if err != nil {
			return queued, err
		}
		added, err := b.Store.InsertFollowup(ctx, domain.InvoiceFollowup{ID: ids.New(), InvoiceID: inv.ID, Stage: stage, Phone: phone,
			Message: domain.FollowupMessage(pic, inv.InvoiceNumber, inv.Amount-inv.PaidTotal, *inv.DueDate, stage)})
		if err != nil {
			return queued, err
		}
		if added {
			queued++
		}
	}
	return queued, nil
}

// billingContact prefers the lead's PIC and falls back to the account's phone.
func (b *Billing) billingContact(ctx context.Context, dealID string) (string, string, error) {
	deal, err := b.Store.GetDeal(ctx, dealID)
	if err != nil || deal == nil {
		return "", "", err
	}
	pic, phone := "", ""
	if deal.LeadID != nil {
		lead, err := b.Store.GetLead(ctx, *deal.LeadID)
		if err != nil {
			return "", "", err
		}
		if lead != nil {
			pic, phone = lead.PicName, deref(lead.PicPhone)
		}
	}
	if phone == "" && deal.AccountID != nil {
		acc, err := b.Store.GetAccount(ctx, *deal.AccountID)
		if err != nil {
			return "", "", err
		}
		if acc != nil {
			phone = deref(acc.Phone)
		}
	}
	return pic, phone, nil
}

func (b *Billing) openFollowup(ctx context.Context, id string) (*domain.InvoiceFollowup, error) {
	f, err := b.Store.GetFollowup(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, notFound("Follow-up tidak ditemukan")
	}
	if f.Status == domain.FollowupSent || f.Status == domain.FollowupSkipped {
		return nil, conflict("Follow-up ini sudah selesai")
	}
	return f, nil
}

// ApproveFollowup authorizes a message, optionally with an edited phone or text,
// and sends it when the window and the gateway allow. Otherwise it stays
// approved and the next tick sends it.
func (b *Billing) ApproveFollowup(ctx context.Context, actor Actor, id, phone, message string) (*domain.InvoiceFollowup, error) {
	f, err := b.openFollowup(ctx, id)
	if err != nil {
		return nil, err
	}
	if p := domain.JSTrim(phone); p != "" {
		f.Phone = p
	}
	if m := domain.JSTrim(message); m != "" {
		f.Message = m
	}
	if f.Phone == "" {
		return nil, apperr.Field("phone", "Nomor WhatsApp wajib diisi")
	}
	now := b.now()
	f.Status, f.ApprovedBy, f.ApprovedAt, f.LastError = domain.FollowupApproved, actor.ref(), &now, ""
	if err := b.Store.UpdateFollowup(ctx, *f); err != nil {
		return nil, err
	}
	if err := b.sendFollowup(ctx, f); err != nil {
		return nil, err
	}
	return b.Store.GetFollowup(ctx, id)
}

func (b *Billing) SkipFollowup(ctx context.Context, id string) (*domain.InvoiceFollowup, error) {
	f, err := b.openFollowup(ctx, id)
	if err != nil {
		return nil, err
	}
	f.Status = domain.FollowupSkipped
	if err := b.Store.UpdateFollowup(ctx, *f); err != nil {
		return nil, err
	}
	return f, nil
}

// SendApprovedFollowups retries what waited for the window or the gateway.
func (b *Billing) SendApprovedFollowups(ctx context.Context) (int, error) {
	rows, err := b.Store.ListFollowups(ctx, domain.FollowupApproved, 200)
	if err != nil {
		return 0, err
	}
	sent := 0
	for i := range rows {
		if err := b.sendFollowup(ctx, &rows[i]); err != nil {
			return sent, err
		}
		if rows[i].Status == domain.FollowupSent {
			sent++
		}
	}
	return sent, nil
}

// sendFollowup rechecks the invoice right before sending, so a client who paid
// after approval gets no reminder. A delivery failure stays on the row as text.
func (b *Billing) sendFollowup(ctx context.Context, f *domain.InvoiceFollowup) error {
	inv, err := b.Store.GetInvoice(ctx, f.InvoiceID)
	if err != nil {
		return err
	}
	switch {
	case inv == nil || inv.Status != InvoiceSent || inv.PaidTotal >= inv.Amount:
		f.Status, f.LastError = domain.FollowupSkipped, "Invoice sudah lunas atau dibatalkan"
	case !b.WhatsAppEnabled():
		f.LastError = "Layanan WhatsApp belum aktif. Pesan terkirim otomatis setelah aktif."
	case !domain.FollowupSendable(b.now()):
		f.LastError = "Menunggu jam kirim: Senin sampai Jumat, 08.00 sampai 18.00 WIB."
	default:
		id, err := b.WhatsApp.Send(ctx, f.Phone, f.Message)
		if err != nil {
			f.LastError = "Gagal mengirim: " + apperr.As(err).Message
			break
		}
		now := b.now()
		f.Status, f.SentAt, f.MessageID, f.LastError = domain.FollowupSent, &now, &id, ""
	}
	return b.Store.UpdateFollowup(ctx, *f)
}
