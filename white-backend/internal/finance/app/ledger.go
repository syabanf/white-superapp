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
	f.Search, f.PartyName = trim(f.Search), trim(f.PartyName)
	var err error
	if f.From, err = optionalDay("date_from", f.From); err != nil {
		return nil, 0, err
	}
	if f.To, err = optionalDay("date_to", f.To); err != nil {
		return nil, 0, err
	}
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
	// Withholding comes off what the party pays, so off the invoice's total.
	Withholding domain.Money
	// TaxIDs picks master tax codes; they replace a typed TaxAmount.
	TaxIDs []string
	TagID  string
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
	if err := checkTag(ctx, l.Store, in.TagID); err != nil {
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
// applyInvoiceTaxes computes the amounts of chosen tax codes on a typed invoice.
// A price-included code takes the tax out of the typed subtotal.
func (l *Ledgers) applyInvoiceTaxes(ctx context.Context, kind string, in *InvoiceInput) ([]domain.InvoicePosting, error) {
	if len(in.TaxIDs) == 0 {
		return nil, nil
	}
	side := domain.SideSupplier
	if kind == domain.KindAR {
		side = domain.SideB2B
	}
	taxes, err := resolveTaxes(ctx, l.Store, in.TaxIDs, side, in.InvoiceDate, in.PartyName)
	if err != nil {
		return nil, err
	}
	doc := domain.Document{Items: []domain.DocItem{{Qty: domain.Rupiah(1), UnitPrice: in.Subtotal}}, Taxes: taxes}
	doc.Recalculate()
	in.Subtotal, in.TaxAmount, in.Withholding = doc.Total-doc.Tax, doc.Tax, doc.Withholding
	return taxPostings(doc.Taxes, kind)
}

func insertPostings(ctx context.Context, tx domain.Store, invoiceID string, postings []domain.InvoicePosting) error {
	for _, p := range postings {
		p.InvoiceID = invoiceID
		if err := tx.InsertInvoicePosting(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledgers) CreateInvoice(ctx context.Context, kind, userID string, in InvoiceInput) (*domain.Invoice, error) {
	if err := l.normalizeInvoice(ctx, &in); err != nil {
		return nil, err
	}
	postings, err := l.applyInvoiceTaxes(ctx, kind, &in)
	if err != nil {
		return nil, err
	}
	inv := domain.Invoice{
		ID: ids.New(), Kind: kind, InvoiceDate: in.InvoiceDate, DueDate: in.DueDate, PartyName: in.PartyName, Reference: in.Reference,
		AccountID: in.AccountID, Currency: "IDR", Subtotal: in.Subtotal, TaxAmount: in.TaxAmount, TotalAmount: in.Subtotal + in.TaxAmount - in.Withholding,
		Status: domain.StatusDraft, Description: in.Description, ClientID: in.ClientID, TagID: in.TagID, CreatedBy: userID,
	}
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		no, err := l.nextNo(ctx, tx, kinds[kind].invoicePrefix)
		if err != nil {
			return err
		}
		inv.InvoiceNo = no
		if err := tx.InsertInvoice(ctx, inv); err != nil {
			return err
		}
		return insertPostings(ctx, tx, inv.ID, postings)
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

// documentGuard keeps an invoice made from a document in step with it: its tax
// lines and the document's status would otherwise stop matching.
func (l *Ledgers) documentGuard(ctx context.Context, id string) error {
	linked, err := l.Store.IsDocumentInvoice(ctx, id)
	if err != nil {
		return err
	}
	if linked {
		return domain.Conflict("Tagihan ini dibuat dari dokumen, jadi tidak bisa diubah atau dihapus di sini")
	}
	return nil
}

func (l *Ledgers) UpdateInvoice(ctx context.Context, kind, userID, id string, in InvoiceInput) (*domain.Invoice, error) {
	if err := l.documentGuard(ctx, id); err != nil {
		return nil, err
	}
	inv, err := l.draftInvoice(ctx, kind, id)
	if err != nil {
		return nil, err
	}
	if err := l.normalizeInvoice(ctx, &in); err != nil {
		return nil, err
	}
	postings, err := l.applyInvoiceTaxes(ctx, kind, &in)
	if err != nil {
		return nil, err
	}
	inv.InvoiceDate, inv.DueDate, inv.PartyName, inv.Reference = in.InvoiceDate, in.DueDate, in.PartyName, in.Reference
	inv.AccountID, inv.Subtotal, inv.TaxAmount, inv.TotalAmount = in.AccountID, in.Subtotal, in.TaxAmount, in.Subtotal+in.TaxAmount-in.Withholding
	inv.Description, inv.ClientID, inv.TagID = in.Description, in.ClientID, in.TagID
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateInvoice(ctx, *inv, userID); err != nil {
			return err
		}
		if err := tx.DeleteInvoicePostings(ctx, id); err != nil {
			return err
		}
		return insertPostings(ctx, tx, id, postings)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetInvoice(ctx, id)
}

func (l *Ledgers) DeleteInvoice(ctx context.Context, kind, userID, id string) error {
	if err := l.documentGuard(ctx, id); err != nil {
		return err
	}
	if _, err := l.draftInvoice(ctx, kind, id); err != nil {
		return err
	}
	return l.Store.SoftDeleteInvoice(ctx, id, userID)
}

// mainLineAmount is what the mapping's revenue or expense line must carry so
// the journal balances with the invoice's extra lines. The natural side is where
// VAT sits: credit on a receivable, debit on a payable.
func mainLineAmount(inv domain.Invoice, extras []domain.InvoicePosting) domain.Money {
	natural := domain.Credit
	if inv.Kind == domain.KindAP {
		natural = domain.Debit
	}
	amount := inv.TotalAmount
	for _, p := range extras {
		if p.Side == natural {
			amount -= p.Amount
		} else {
			amount += p.Amount
		}
	}
	return amount
}

func (l *Ledgers) invoicePost(inv domain.Invoice, userID string, extras []domain.InvoicePosting) MappingPost {
	cfg := kinds[inv.Kind]
	desc := cfg.label + " " + inv.InvoiceNo + " " + inv.PartyName
	if inv.Reference != "" {
		desc += " (" + inv.Reference + ")"
	}
	post := MappingPost{
		TagID: inv.TagID, UserID: userID, EventCode: cfg.invoiceEvent, DocumentType: DocInvoice, DocumentID: inv.ID, DocumentLabel: inv.InvoiceNo,
		EntryDate: inv.InvoiceDate, Description: desc, Module: cfg.module,
		Amounts: domain.Amounts{"SUBTOTAL": inv.Subtotal, "TAX": inv.TaxAmount, "TOTAL": inv.TotalAmount},
	}
	if inv.AccountID != "" {
		post.Overrides = map[string]string{cfg.accountRole: inv.AccountID}
	}
	if len(extras) > 0 {
		post.Amounts = domain.Amounts{"SUBTOTAL": mainLineAmount(inv, extras), "TOTAL": inv.TotalAmount}
		for _, p := range extras {
			post.Extra = append(post.Extra, domain.Line{AccountID: p.AccountID, Side: p.Side, Amount: p.Amount, Memo: p.Memo, SortOrder: 10 + p.SortOrder})
		}
	}
	return post
}

// postInvoiceJournal posts the invoice's mapping and links the journal.
func (l *Ledgers) postInvoiceJournal(ctx context.Context, tx domain.Store, inv domain.Invoice, userID string) (domain.PostResult, error) {
	extras, err := tx.ListInvoicePostings(ctx, inv.ID)
	if err != nil {
		return domain.PostResult{}, err
	}
	res, err := PostFromMapping(ctx, tx, l.invoicePost(inv, userID, extras))
	if err != nil || res.EntryID == "" {
		return res, err
	}
	if err := insertInvoiceTaxMoves(ctx, tx, inv, extras); err != nil {
		return res, err
	}
	return res, tx.SetInvoiceJournal(ctx, inv.ID, res.EntryID)
}

// insertInvoiceTaxMoves enters the tax ledger once the invoice has its journal,
// so the register and the exports always agree with the books.
func insertInvoiceTaxMoves(ctx context.Context, tx domain.Store, inv domain.Invoice, extras []domain.InvoicePosting) error {
	for _, p := range extras {
		if p.TaxCode == "" || p.CashBasis {
			continue
		}
		if err := tx.InsertTaxMove(ctx, domain.TaxMove{Kind: inv.Kind, SourceType: domain.MoveInvoice, SourceID: inv.ID, InvoiceID: inv.ID, TaxCode: p.TaxCode, TaxName: p.TaxName, TaxKind: p.TaxKind,
			TaxRate: p.TaxRate, ReportCode: p.ReportCode, Date: inv.InvoiceDate, Base: p.TaxBase, Amount: p.Amount}); err != nil {
			return err
		}
	}
	return nil
}

// on is the same service inside a transaction.
func (l *Ledgers) on(tx domain.Store) *Ledgers {
	d := l.Deps
	d.Store = tx
	return NewLedgers(d)
}

// needJournal refuses a manual transaction whose journal could not post, which
// rolls it back. A record without its journal would split the subledger from the
// general ledger.
func needJournal(res domain.PostResult) error {
	if res.EntryID == "" {
		return domain.Conflict("Jurnal tidak bisa diposting: " + res.Reason)
	}
	return nil
}

// PostInvoice moves DRAFT -> POSTED together with its journal. A journal that
// cannot post, for a closed period or an incomplete mapping, refuses the whole step.
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
		if res, err = l.postInvoiceJournal(ctx, tx, *inv, userID); err != nil {
			return err
		}
		return needJournal(res)
	})
	if err != nil {
		return nil, "", err
	}
	inv, err = l.Store.GetInvoice(ctx, id)
	return inv, domain.Summarize(res), err
}

// OpenInvoices are POSTED invoices with an outstanding amount. One still waiting
// for its journal is not in the books yet, so it stays out of the balance.
func (l *Ledgers) OpenInvoices(ctx context.Context, kind string) ([]domain.Invoice, error) {
	rows, _, err := l.Store.ListInvoices(ctx, domain.InvoiceFilter{Kind: kind, Status: domain.StatusPosted, Limit: 1000})
	if err != nil {
		return nil, err
	}
	out := []domain.Invoice{}
	for _, r := range rows {
		if r.Outstanding() > 0 && r.JournalEntryID != "" {
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
	// Sales already took this money, so its event cannot be refused. It records
	// the payment and leaves the journal to the posting log when it cannot post.
	if in.SalesPaymentID != "" {
		return l.recordSalesPayment(ctx, kind, userID, in)
	}
	var p domain.Payment
	var invoiceIDs []string
	var note string
	err := l.Store.WithTx(ctx, func(tx domain.Store) error {
		var err error
		if p, invoiceIDs, err = l.insertPayment(ctx, tx, kind, userID, in); err != nil {
			return err
		}
		note, err = l.postPaymentStrict(ctx, tx, p, userID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return l.paymentResult(ctx, p.ID, invoiceIDs, note)
}

func (l *Ledgers) recordSalesPayment(ctx context.Context, kind, userID string, in PaymentInput) (*PaymentResult, error) {
	var p domain.Payment
	var invoiceIDs []string
	err := l.Store.WithTx(ctx, func(tx domain.Store) error {
		var err error
		p, invoiceIDs, err = l.insertPayment(ctx, tx, kind, userID, in)
		return err
	})
	if err != nil {
		return nil, err
	}
	res, err := l.postPaymentJournal(ctx, l.Store, p, userID)
	if err != nil {
		return nil, err
	}
	note := domain.Summarize(res)
	if extra := l.recognizeCashTaxes(ctx, p, userID, false); extra != "" {
		note += "; " + extra
	}
	return l.paymentResult(ctx, p.ID, invoiceIDs, note)
}

// postPaymentStrict posts the payment's journal and cash basis VAT inside the
// caller's transaction; any failure rolls the payment back.
func (l *Ledgers) postPaymentStrict(ctx context.Context, tx domain.Store, p domain.Payment, userID string) (string, error) {
	res, err := l.postPaymentJournal(ctx, tx, p, userID)
	if err != nil {
		return "", err
	}
	if err := needJournal(res); err != nil {
		return "", err
	}
	if extra := l.on(tx).recognizeCashTaxes(ctx, p, userID, false); extra != "" {
		return "", domain.Conflict(extra)
	}
	return domain.Summarize(res), nil
}

// insertPayment validates the allocations and stores the payment. The caller
// has checked the method and the account.
func (l *Ledgers) insertPayment(ctx context.Context, tx domain.Store, kind, userID string, in PaymentInput) (domain.Payment, []string, error) {
	cfg := kinds[kind]
	day := l.today()
	if trim(in.PaymentDate) != "" {
		var err error
		if day, err = parseDay("payment_date", in.PaymentDate); err != nil {
			return domain.Payment{}, nil, err
		}
	}
	p := domain.Payment{
		ID: ids.New(), Kind: kind, PaymentDate: day, Method: orDefault(in.Method, cfg.defaultMethod), CashAccountID: in.CashAccountID, Reference: trim(in.Reference),
		Notes: trim(in.Notes), Status: domain.StatusPosted, SalesPaymentID: in.SalesPaymentID, CreatedBy: userID,
	}
	var invoiceIDs []string
	for _, a := range in.Allocations {
		if slices.Contains(invoiceIDs, a.InvoiceID) {
			return p, nil, domain.Bad("Invoice yang sama dipilih lebih dari sekali")
		}
		inv, err := tx.GetInvoice(ctx, a.InvoiceID)
		if err != nil {
			return p, nil, err
		}
		if inv == nil || inv.Kind != kind {
			return p, nil, domain.NotFound(cfg.label + " invoice tidak ditemukan")
		}
		if inv.Status != domain.StatusPosted {
			return p, nil, domain.Bad("Hanya invoice POSTED yang bisa dibayar")
		}
		if inv.JournalEntryID == "" && in.SalesPaymentID == "" {
			return p, nil, domain.Conflict("Tagihan " + inv.InvoiceNo + " belum berjurnal. Selesaikan dulu di log posting.")
		}
		if day < inv.InvoiceDate && in.SalesPaymentID == "" {
			return p, nil, apperr.Field("payment_date", "Tanggal pembayaran tidak boleh sebelum tanggal tagihan")
		}
		if a.Amount <= 0 {
			return p, nil, domain.Bad("Amount pembayaran harus > 0")
		}
		if a.Amount > inv.Outstanding() {
			return p, nil, domain.Bad("Amount pembayaran tidak boleh melebihi outstanding")
		}
		if p.PartyName == "" {
			p.PartyName = inv.PartyName
		} else if !strings.EqualFold(p.PartyName, inv.PartyName) {
			return p, nil, domain.Bad("Semua invoice dalam satu pembayaran harus dari pihak yang sama")
		}
		p.Amount += a.Amount
		invoiceIDs = append(invoiceIDs, a.InvoiceID)
	}
	no, err := l.nextNo(ctx, tx, cfg.paymentPrefix)
	if err != nil {
		return p, nil, err
	}
	p.PaymentNo, p.Allocations = no, in.Allocations
	return p, invoiceIDs, tx.InsertPayment(ctx, p)
}

func (l *Ledgers) postPaymentJournal(ctx context.Context, s domain.Store, p domain.Payment, userID string) (domain.PostResult, error) {
	cfg := kinds[p.Kind]
	post := MappingPost{
		UserID: userID, EventCode: cfg.paymentEvent, DocumentType: DocPayment, DocumentID: p.ID, DocumentLabel: p.PaymentNo,
		EntryDate: p.PaymentDate, Description: cfg.docLabel + " " + p.PaymentNo + " " + p.PartyName, Module: cfg.module,
		Amounts: domain.PaymentAmounts(p.Amount),
	}
	if p.CashAccountID != "" {
		post.Overrides = map[string]string{"CASH": p.CashAccountID}
	}
	res, err := PostFromMapping(ctx, s, post)
	if err != nil || res.EntryID == "" {
		return res, err
	}
	return res, s.SetPaymentJournal(ctx, p.ID, res.EntryID)
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
	if p.Method == methodCreditNote {
		return nil, domain.Conflict("Credit note tidak bisa dibatalkan. Buat tagihan baru bila perlu.")
	}
	if len([]rune(reason)) > 500 {
		return nil, apperr.Field("reason", "Alasan void maksimal 500 karakter")
	}
	var note string
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.VoidPayment(ctx, id, reason, userID); err != nil {
			return err
		}
		t := l.on(tx)
		var err error
		if note, err = t.reversePaymentJournal(ctx, *p, cfg, userID, reason); err != nil {
			return err
		}
		if extra := t.recognizeCashTaxes(ctx, *p, userID, true); extra != "" {
			return domain.Conflict(extra)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	invoiceIDs := make([]string, len(p.Allocations))
	for i, a := range p.Allocations {
		invoiceIDs[i] = a.InvoiceID
	}
	return l.paymentResult(ctx, id, invoiceIDs, note)
}

// reversePaymentJournal: a draft journal is deleted, a posted one gets a reversing
// AUTO entry dated today. A payment that never got its journal only drops its
// pending posting, so a retry cannot book a voided payment. Any failure refuses
// the void.
func (l *Ledgers) reversePaymentJournal(ctx context.Context, p domain.Payment, cfg kindConfig, userID, reason string) (string, error) {
	entry, err := l.Store.FindEntryBySource(ctx, cfg.paymentEvent, p.ID)
	if err != nil {
		return "", err
	}
	if entry == nil {
		return "tidak ada jurnal pembayaran untuk dibalik", l.Store.DeletePostingLog(ctx, cfg.paymentEvent, p.ID)
	}
	if entry.Status != domain.StatusPosted {
		return "jurnal draft pembayaran dihapus", l.Store.SoftDeleteJournalEntry(ctx, entry.ID, userID)
	}
	_, err = createEntry(ctx, l.Store, NewEntry{
		UserID: userID, EntryDate: l.today(), Description: "Void " + strings.ToLower(cfg.docLabel) + " " + p.PaymentNo + ": " + reason,
		Lines: domain.ReverseLines(entry.DomainLines(), "Pembalik "+entry.EntryNo), Post: true, EntryType: domain.EntryAuto,
		SourceModule: cfg.module, SourceEventCode: cfg.paymentEvent + "_VOID", SourceDocumentType: DocPayment, SourceDocumentID: p.ID,
	})
	if err != nil {
		return "", domain.Conflict("Jurnal pembalik tidak bisa diposting: " + apperr.As(err).Message)
	}
	return "jurnal pembalik diposting", nil
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
	Kind, Party, AccountID  string
	From, To                string
	Opening, Closing        domain.Money
	TotalDebit, TotalCredit domain.Money
	Lines                   []SubsidiaryLine
}

// SubsidiaryLedger: invoices debit and allocations credit the party's
// running balance.
func (l *Ledgers) SubsidiaryLedger(ctx context.Context, kind, party, accountID, from, to string) (*SubsidiaryLedger, error) {
	party, accountID = trim(party), trim(accountID)
	if party == "" && accountID == "" {
		return nil, apperr.Field("party_name", "Pilih pihak atau akun COA")
	}
	from, to, err := ledgerRange(from, to)
	if err != nil {
		return nil, err
	}
	out := &SubsidiaryLedger{Kind: kind, Party: party, AccountID: accountID, From: from, To: to}
	if from != "" {
		if out.Opening, err = l.Store.SubsidiaryOpening(ctx, kind, party, accountID, from); err != nil {
			return nil, err
		}
	}
	moves, err := l.Store.SubsidiaryMoves(ctx, kind, party, accountID, from, to)
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
