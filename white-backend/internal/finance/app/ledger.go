package app

import (
	"context"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Ledgers covers accounts payable and receivable: invoices, the open
// register, aging, payments with allocations and void with a reversing
// journal (NüHabit service_ap.go, service_ar.go, service_subledger.go).
// AP bills are entered by finance; AR invoices come from sales events or
// are entered by finance.
type Ledgers struct{ Deps }

func NewLedgers(d Deps) *Ledgers { return &Ledgers{d} }

type kindConfig struct {
	label, invoicePrefix, paymentPrefix  string
	invoiceEvent, paymentEvent, module   string
	accountRole, defaultMethod, docLabel string
	methods                              []string
}

var kinds = map[string]kindConfig{
	domain.KindAP: {
		label: "AP", invoicePrefix: "AP", paymentPrefix: "APP", invoiceEvent: domain.EventPurchaseAPInvoice,
		paymentEvent: domain.EventPurchasePayment, module: "PURCHASING", accountRole: "EXPENSE", defaultMethod: "bank_transfer",
		docLabel: "Pembayaran AP", methods: []string{"cash", "bank_transfer", "giro", "qris", "other"},
	},
	domain.KindAR: {
		label: "AR", invoicePrefix: "AR", paymentPrefix: "ARR", invoiceEvent: domain.EventSaleARInvoice,
		paymentEvent: domain.EventSaleARReceipt, module: "SALES", accountRole: "REVENUE", defaultMethod: "transfer",
		docLabel: "AR Receipt", methods: []string{"cash", "transfer", "qris", "edc", "lainnya", "bank_transfer", "other"},
	},
}

// ─── Invoices ───────────────────────────────────────────────

// PaymentStatus is paid / partial / overdue / unpaid on the Jakarta today.
func (l *Ledgers) PaymentStatus(i domain.Invoice) string {
	return domain.PaymentStatus(i.TotalAmount, i.Allocated, i.DueDate, l.today())
}

// ListInvoices pages invoices; a payment status filter is applied in Go
// because it depends on today.
func (l *Ledgers) ListInvoices(ctx context.Context, f domain.InvoiceFilter, paymentStatus string) ([]domain.Invoice, int, error) {
	f.Search = trim(f.Search)
	if paymentStatus == "" || paymentStatus == "all" {
		return l.Store.ListInvoices(ctx, f)
	}
	limit, offset := f.Limit, f.Offset
	f.Limit, f.Offset = 1000, 0
	rows, _, err := l.Store.ListInvoices(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	var out []domain.Invoice
	for _, r := range rows {
		if l.PaymentStatus(r) == paymentStatus {
			out = append(out, r)
		}
	}
	total := len(out)
	if offset > total {
		offset = total
	}
	return out[offset:min(total, offset+limit)], total, nil
}

func (l *Ledgers) GetInvoice(ctx context.Context, kind, id string) (*domain.Invoice, error) {
	inv, err := l.Store.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv == nil || inv.Kind != kind {
		return nil, domain.NotFound(kinds[kind].label + " invoice tidak ditemukan")
	}
	return inv, nil
}

func (l *Ledgers) InvoiceBySales(ctx context.Context, salesInvoiceID string) (*domain.Invoice, error) {
	inv, err := l.Store.GetInvoiceBySalesID(ctx, salesInvoiceID)
	if err == nil && inv == nil {
		err = domain.NotFound("AR invoice tidak ditemukan")
	}
	return inv, err
}

// InvoiceInput is the invoice form.
type InvoiceInput struct {
	InvoiceDate, DueDate, PartyName, Reference, AccountID, Description, ClientID string
	Subtotal, TaxAmount                                                          domain.Money
}

func (l *Ledgers) normalizeInvoice(ctx context.Context, in *InvoiceInput) error {
	day, err := parseDay("invoice_date", in.InvoiceDate)
	if err != nil {
		return err
	}
	due, err := optionalDay("due_date", in.DueDate)
	if err != nil {
		return err
	}
	in.InvoiceDate, in.DueDate, in.PartyName = day, due, trim(in.PartyName)
	in.Reference, in.Description = trim(in.Reference), trim(in.Description)
	v := validate.New().Required("party_name", in.PartyName, "Nama vendor/klien wajib diisi")
	if in.Subtotal <= 0 {
		v.Add("subtotal", "Subtotal harus lebih dari 0")
	}
	if in.TaxAmount < 0 {
		v.Add("tax_amount", "Pajak tidak boleh negatif")
	}
	if due != "" && due < day {
		v.Add("due_date", "Jatuh tempo tidak boleh sebelum tanggal invoice")
	}
	if err := v.Err(); err != nil {
		return err
	}
	if in.AccountID != "" {
		return checkAccounts(ctx, l.Store, []string{in.AccountID}, "", "Akun harus postable (bukan header)")
	}
	return nil
}

func (l *Ledgers) nextNo(ctx context.Context, s domain.Store, prefix string) (string, error) {
	p := domain.DailyNoPrefix(prefix, l.now().In(Jakarta))
	n, err := s.NextSequence(ctx, p)
	if err != nil {
		return "", err
	}
	return domain.SequenceNo(p, n), nil
}

// CreateInvoice saves a DRAFT invoice.
func (l *Ledgers) CreateInvoice(ctx context.Context, kind, userID string, in InvoiceInput) (*domain.Invoice, error) {
	if err := l.normalizeInvoice(ctx, &in); err != nil {
		return nil, err
	}
	inv := domain.Invoice{
		ID: ids.New(), Kind: kind, InvoiceDate: in.InvoiceDate, DueDate: in.DueDate, PartyName: in.PartyName, Reference: in.Reference,
		AccountID: in.AccountID, Currency: "IDR", Subtotal: in.Subtotal, TaxAmount: in.TaxAmount, TotalAmount: in.Subtotal + in.TaxAmount,
		Status: domain.StatusDraft, Description: in.Description, ClientID: in.ClientID, CreatedBy: userID,
	}
	err := l.Store.WithTx(ctx, func(tx domain.Store) error {
		no, err := l.nextNo(ctx, tx, kinds[kind].invoicePrefix)
		if err != nil {
			return err
		}
		inv.InvoiceNo = no
		return tx.InsertInvoice(ctx, inv)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetInvoice(ctx, inv.ID)
}

func (l *Ledgers) draftInvoice(ctx context.Context, kind, id string) (*domain.Invoice, error) {
	inv, err := l.GetInvoice(ctx, kind, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != domain.StatusDraft {
		return nil, domain.Bad("Hanya invoice DRAFT yang bisa diubah atau dihapus")
	}
	return inv, nil
}

func (l *Ledgers) UpdateInvoice(ctx context.Context, kind, userID, id string, in InvoiceInput) (*domain.Invoice, error) {
	inv, err := l.draftInvoice(ctx, kind, id)
	if err != nil {
		return nil, err
	}
	if err := l.normalizeInvoice(ctx, &in); err != nil {
		return nil, err
	}
	inv.InvoiceDate, inv.DueDate, inv.PartyName, inv.Reference = in.InvoiceDate, in.DueDate, in.PartyName, in.Reference
	inv.AccountID, inv.Subtotal, inv.TaxAmount, inv.TotalAmount = in.AccountID, in.Subtotal, in.TaxAmount, in.Subtotal+in.TaxAmount
	inv.Description, inv.ClientID = in.Description, in.ClientID
	if err := l.Store.UpdateInvoice(ctx, *inv, userID); err != nil {
		return nil, err
	}
	return l.Store.GetInvoice(ctx, id)
}

func (l *Ledgers) DeleteInvoice(ctx context.Context, kind, userID, id string) error {
	if _, err := l.draftInvoice(ctx, kind, id); err != nil {
		return err
	}
	return l.Store.SoftDeleteInvoice(ctx, id, userID)
}

func (l *Ledgers) invoicePost(inv domain.Invoice, userID string) MappingPost {
	cfg := kinds[inv.Kind]
	desc := cfg.label + " " + inv.InvoiceNo + " " + inv.PartyName
	if inv.Reference != "" {
		desc += " (" + inv.Reference + ")"
	}
	post := MappingPost{
		UserID: userID, EventCode: cfg.invoiceEvent, DocumentType: DocInvoice, DocumentID: inv.ID, DocumentLabel: inv.InvoiceNo,
		EntryDate: inv.InvoiceDate, Description: desc, Module: cfg.module,
		Amounts: domain.Amounts{"SUBTOTAL": inv.Subtotal, "TAX": inv.TaxAmount, "TOTAL": inv.TotalAmount},
	}
	if inv.AccountID != "" {
		post.Overrides = map[string]string{cfg.accountRole: inv.AccountID}
	}
	return post
}

// postInvoiceJournal posts the invoice's mapping and links the journal.
func (l *Ledgers) postInvoiceJournal(ctx context.Context, tx domain.Store, inv domain.Invoice, userID string) (domain.PostResult, error) {
	res, err := PostFromMapping(ctx, tx, l.invoicePost(inv, userID))
	if err != nil || res.EntryID == "" {
		return res, err
	}
	return res, tx.SetInvoiceJournal(ctx, inv.ID, res.EntryID)
}

// PostInvoice moves DRAFT -> POSTED and posts its journal. The invoice
// stays POSTED when the mapping is not ready; the note says why.
func (l *Ledgers) PostInvoice(ctx context.Context, kind, userID, id string) (*domain.Invoice, string, error) {
	inv, err := l.GetInvoice(ctx, kind, id)
	if err != nil {
		return nil, "", err
	}
	if inv.Status != domain.StatusDraft {
		return nil, "", domain.Bad("Invoice sudah " + inv.Status)
	}
	var res domain.PostResult
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.MarkInvoicePosted(ctx, id, userID, l.now()); err != nil {
			return err
		}
		res, err = l.postInvoiceJournal(ctx, tx, *inv, userID)
		return err
	})
	if err != nil {
		return nil, "", err
	}
	inv, err = l.Store.GetInvoice(ctx, id)
	return inv, domain.Summarize(res), err
}

// OpenInvoices are POSTED invoices with an outstanding amount.
func (l *Ledgers) OpenInvoices(ctx context.Context, kind string) ([]domain.Invoice, error) {
	rows, _, err := l.Store.ListInvoices(ctx, domain.InvoiceFilter{Kind: kind, Status: domain.StatusPosted, Limit: 1000})
	if err != nil {
		return nil, err
	}
	out := []domain.Invoice{}
	for _, r := range rows {
		if r.Outstanding() > 0 {
			out = append(out, r)
		}
	}
	return out, nil
}

type AgingBucket struct {
	Bucket string
	Count  int
	Amount domain.Money
}

type Aging struct {
	AsOf     string
	Buckets  []AgingBucket
	Invoices []domain.Invoice
	Total    domain.Money
}

func (l *Ledgers) Aging(ctx context.Context, kind, asOf string) (*Aging, error) {
	if asOf == "" {
		asOf = l.today()
	}
	asOf, err := parseDay("as_of", asOf)
	if err != nil {
		return nil, err
	}
	open, err := l.OpenInvoices(ctx, kind)
	if err != nil {
		return nil, err
	}
	out := &Aging{AsOf: asOf, Invoices: open}
	idx := map[string]int{}
	for i, b := range domain.AgingBuckets {
		out.Buckets = append(out.Buckets, AgingBucket{Bucket: b})
		idx[b] = i
	}
	for _, inv := range open {
		o := inv.Outstanding()
		b := domain.BucketAging(o, inv.DueDate, asOf)
		if b == "" {
			continue
		}
		out.Buckets[idx[b]].Count++
		out.Buckets[idx[b]].Amount += o
		out.Total += o
	}
	return out, nil
}

// ─── Payments ───────────────────────────────────────────────

func (l *Ledgers) ListPayments(ctx context.Context, f domain.PaymentFilter) ([]domain.Payment, int, error) {
	f.Search = trim(f.Search)
	return l.Store.ListPayments(ctx, f)
}

// PaymentInput is a payment or receipt allocated to one or more invoices.
type PaymentInput struct {
	PaymentDate, Method, CashAccountID, Reference, Notes string
	Allocations                                          []domain.Allocation
	SalesPaymentID                                       string
}

// PaymentResult is the payment, its refreshed invoices and the journal note.
type PaymentResult struct {
	Payment  domain.Payment
	Invoices []domain.Invoice
	Note     string
}

// CreatePayment records a POSTED payment with its allocations, then posts
// the payment journal (NüHabit posts after the commit too).
func (l *Ledgers) CreatePayment(ctx context.Context, kind, userID string, in PaymentInput) (*PaymentResult, error) {
	cfg := kinds[kind]
	day := l.today()
	if trim(in.PaymentDate) != "" {
		var err error
		if day, err = parseDay("payment_date", in.PaymentDate); err != nil {
			return nil, err
		}
	}
	method := orDefault(in.Method, cfg.defaultMethod)
	if !slices.Contains(cfg.methods, method) {
		return nil, apperr.Field("method", "Metode pembayaran tidak valid")
	}
	if len(in.Allocations) == 0 {
		return nil, apperr.Field("allocations", "Pilih minimal satu invoice")
	}
	if in.CashAccountID != "" {
		acc, err := l.Store.GetAccount(ctx, in.CashAccountID)
		if err != nil {
			return nil, err
		}
		if acc == nil || !acc.IsCashBank || !acc.IsPostable {
			return nil, domain.Bad("Akun kas/bank tidak valid")
		}
	}
	p := domain.Payment{
		ID: ids.New(), Kind: kind, PaymentDate: day, Method: method, CashAccountID: in.CashAccountID, Reference: trim(in.Reference),
		Notes: trim(in.Notes), Status: domain.StatusPosted, SalesPaymentID: in.SalesPaymentID, CreatedBy: userID,
	}
	var invoiceIDs []string
	err := l.Store.WithTx(ctx, func(tx domain.Store) error {
		for _, a := range in.Allocations {
			if slices.Contains(invoiceIDs, a.InvoiceID) {
				return domain.Bad("Invoice yang sama dipilih lebih dari sekali")
			}
			inv, err := tx.GetInvoice(ctx, a.InvoiceID)
			if err != nil {
				return err
			}
			if inv == nil || inv.Kind != kind {
				return domain.NotFound(cfg.label + " invoice tidak ditemukan")
			}
			if inv.Status != domain.StatusPosted {
				return domain.Bad("Hanya invoice POSTED yang bisa dibayar")
			}
			if a.Amount <= 0 {
				return domain.Bad("Amount pembayaran harus > 0")
			}
			if a.Amount > inv.Outstanding() {
				return domain.Bad("Amount pembayaran tidak boleh melebihi outstanding")
			}
			if p.PartyName == "" {
				p.PartyName = inv.PartyName
			} else if !strings.EqualFold(p.PartyName, inv.PartyName) {
				return domain.Bad("Semua invoice dalam satu pembayaran harus dari pihak yang sama")
			}
			p.Amount += a.Amount
			invoiceIDs = append(invoiceIDs, a.InvoiceID)
		}
		no, err := l.nextNo(ctx, tx, cfg.paymentPrefix)
		if err != nil {
			return err
		}
		p.PaymentNo, p.Allocations = no, in.Allocations
		return tx.InsertPayment(ctx, p)
	})
	if err != nil {
		return nil, err
	}
	res, err := l.postPaymentJournal(ctx, p, userID)
	if err != nil {
		return nil, err
	}
	return l.paymentResult(ctx, p.ID, invoiceIDs, domain.Summarize(res))
}

func (l *Ledgers) postPaymentJournal(ctx context.Context, p domain.Payment, userID string) (domain.PostResult, error) {
	cfg := kinds[p.Kind]
	post := MappingPost{
		UserID: userID, EventCode: cfg.paymentEvent, DocumentType: DocPayment, DocumentID: p.ID, DocumentLabel: p.PaymentNo,
		EntryDate: p.PaymentDate, Description: cfg.docLabel + " " + p.PaymentNo + " " + p.PartyName, Module: cfg.module,
		Amounts: domain.PaymentAmounts(p.Amount),
	}
	if p.CashAccountID != "" {
		post.Overrides = map[string]string{"CASH": p.CashAccountID}
	}
	res, err := PostFromMapping(ctx, l.Store, post)
	if err != nil || res.EntryID == "" {
		return res, err
	}
	return res, l.Store.SetPaymentJournal(ctx, p.ID, res.EntryID)
}

func (l *Ledgers) paymentResult(ctx context.Context, paymentID string, invoiceIDs []string, note string) (*PaymentResult, error) {
	p, err := l.Store.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	out := &PaymentResult{Payment: *p, Note: note}
	for _, id := range invoiceIDs {
		inv, err := l.Store.GetInvoice(ctx, id)
		if err != nil {
			return nil, err
		}
		if inv != nil {
			out.Invoices = append(out.Invoices, *inv)
		}
	}
	return out, nil
}

// VoidPayment voids a POSTED payment with a reason (>= 5 characters). Its
// allocations stay as history but stop counting; a posted journal gets a
// reversing AUTO entry dated today.
func (l *Ledgers) VoidPayment(ctx context.Context, kind, userID, id, reason string) (*PaymentResult, error) {
	cfg := kinds[kind]
	reason = trim(reason)
	p, err := l.Store.GetPayment(ctx, id)
	if err != nil {
		return nil, err
	}
	found := p != nil && p.Kind == kind
	status := ""
	if found {
		status = p.Status
	}
	if err := domain.EvaluatePaymentVoid(found, status, reason, cfg.label); err != nil {
		return nil, err
	}
	if len([]rune(reason)) > 500 {
		return nil, apperr.Field("reason", "Alasan void maksimal 500 karakter")
	}
	if err := l.Store.VoidPayment(ctx, id, reason, userID); err != nil {
		return nil, err
	}
	note := l.reversePaymentJournal(ctx, *p, cfg, userID, reason)
	invoiceIDs := make([]string, len(p.Allocations))
	for i, a := range p.Allocations {
		invoiceIDs[i] = a.InvoiceID
	}
	return l.paymentResult(ctx, id, invoiceIDs, note)
}

// reversePaymentJournal: a draft journal is deleted, a posted one gets a
// reversing entry. Failures become the note.
func (l *Ledgers) reversePaymentJournal(ctx context.Context, p domain.Payment, cfg kindConfig, userID, reason string) string {
	entry, err := l.Store.FindEntryBySource(ctx, cfg.paymentEvent, p.ID)
	if err != nil {
		return "jurnal pembalik gagal: " + apperr.As(err).Message
	}
	if entry == nil {
		return "tidak ada jurnal pembayaran untuk dibalik"
	}
	if entry.Status != domain.StatusPosted {
		if err := l.Store.SoftDeleteJournalEntry(ctx, entry.ID, userID); err != nil {
			return "jurnal pembalik gagal: " + apperr.As(err).Message
		}
		return "jurnal draft pembayaran dihapus"
	}
	_, err = createEntry(ctx, l.Store, NewEntry{
		UserID: userID, EntryDate: l.today(), Description: "Void " + strings.ToLower(cfg.docLabel) + " " + p.PaymentNo + ": " + reason,
		Lines: domain.ReverseLines(entry.DomainLines(), "Pembalik "+entry.EntryNo), Post: true, EntryType: domain.EntryAuto,
		SourceModule: cfg.module, SourceEventCode: cfg.paymentEvent + "_VOID", SourceDocumentType: DocPayment, SourceDocumentID: p.ID,
	})
	if err != nil {
		if l.Logger != nil {
			l.Logger.Error("payment void reversal failed", "payment", p.ID, "err", err)
		}
		return "jurnal pembalik gagal: " + apperr.As(err).Message
	}
	return "jurnal pembalik diposting"
}

// ─── Subsidiary ledger ──────────────────────────────────────

func (l *Ledgers) SubsidiaryParties(ctx context.Context, kind string) ([]domain.SubsidiaryParty, error) {
	return l.Store.SubsidiaryParties(ctx, kind)
}

type SubsidiaryLine struct {
	domain.SubsidiaryMove
	Debit, Credit, Running domain.Money
}

type SubsidiaryLedger struct {
	Kind, Party, From, To   string
	Opening, Closing        domain.Money
	TotalDebit, TotalCredit domain.Money
	Lines                   []SubsidiaryLine
}

// SubsidiaryLedger: invoices debit and allocations credit the party's
// running balance.
func (l *Ledgers) SubsidiaryLedger(ctx context.Context, kind, party, from, to string) (*SubsidiaryLedger, error) {
	party = trim(party)
	if party == "" {
		return nil, apperr.Field("party_name", "Pihak wajib dipilih")
	}
	from, to, err := ledgerRange(from, to)
	if err != nil {
		return nil, err
	}
	out := &SubsidiaryLedger{Kind: kind, Party: party, From: from, To: to}
	if from != "" {
		if out.Opening, err = l.Store.SubsidiaryOpening(ctx, kind, party, from); err != nil {
			return nil, err
		}
	}
	moves, err := l.Store.SubsidiaryMoves(ctx, kind, party, from, to)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(moves, func(a, b domain.SubsidiaryMove) int {
		if a.Date != b.Date {
			return strings.Compare(a.Date, b.Date)
		}
		return strings.Compare(a.DocNo, b.DocNo)
	})
	running := out.Opening
	for _, m := range moves {
		line := SubsidiaryLine{SubsidiaryMove: m}
		if m.IsInvoice {
			line.Debit = m.Amount
		} else {
			line.Credit = m.Amount
		}
		out.TotalDebit += line.Debit
		out.TotalCredit += line.Credit
		running += line.Debit - line.Credit
		line.Running = running
		out.Lines = append(out.Lines, line)
	}
	out.Closing = running
	return out, nil
}
