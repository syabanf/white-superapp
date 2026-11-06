package app

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// ─── Reversals from Sales ───────────────────────────────────

const salesPaymentDeletedReason = "Pembayaran dihapus di Sales"

// SalesPaymentDeleted voids the receipt of a payment Sales removed. Sales has already
// removed it, so a void Finance cannot do yet, such as in a closed period, stays
// pending: approvers get told and the daily run tries again.
func (a *AutoPost) SalesPaymentDeleted(ctx context.Context, p domain.SalesPaymentDeleted) error {
	if p.PaymentID == "" {
		return permanent("sales payment deleted tanpa paymentId")
	}
	pay, err := a.Store.GetPaymentBySalesID(ctx, p.PaymentID)
	if err != nil {
		return err
	}
	if pay != nil {
		if pay.Status != domain.StatusPosted {
			return nil
		}
		return a.reverseSalesPayment(ctx, p.DeletedBy, pay.ID)
	}
	advance, err := a.Store.AdvanceBySalesPayment(ctx, p.PaymentID)
	if err != nil {
		return err
	}
	if advance == nil {
		// The payment event may still be on its way, so this one is redelivered.
		return fmt.Errorf("penerimaan untuk pembayaran sales %s belum ada", p.PaymentID)
	}
	if advance.Status != domain.StatusPosted {
		return nil
	}
	if _, err := NewLedgers(a.Deps).VoidAdvance(ctx, p.DeletedBy, advance.ID, salesPaymentDeletedReason); err != nil {
		if apperr.As(err).Code == apperr.Unknown {
			return err
		}
		a.tellFinance(ctx, "Uang muka "+advance.AdvanceNo+" perlu dibatalkan manual", salesPaymentDeletedReason+": "+apperr.As(err).Message)
	}
	return nil
}

func (a *AutoPost) reverseSalesPayment(ctx context.Context, userID, paymentID string) error {
	_, voidErr := NewLedgers(a.Deps).VoidPayment(ctx, domain.KindAR, userID, paymentID, salesPaymentDeletedReason)
	if voidErr != nil && apperr.As(voidErr).Code == apperr.Unknown {
		return voidErr
	}
	failure := ""
	if voidErr != nil {
		failure = apperr.As(voidErr).Message
	}
	if err := a.Store.SetPaymentSalesReversal(ctx, paymentID, salesPaymentDeletedReason, failure); err != nil {
		return err
	}
	if failure != "" {
		a.tellFinance(ctx, "Penerimaan dari Sales belum bisa dibatalkan", failure)
	}
	return nil
}

// SalesInvoiceCancelled credits what is still open on an invoice Sales cancelled.
// Money Finance recorded on it stays, and approvers get told to handle it.
func (a *AutoPost) SalesInvoiceCancelled(ctx context.Context, p domain.SalesInvoiceCancelled) error {
	if p.InvoiceID == "" {
		return permanent("sales invoice cancelled tanpa invoiceId")
	}
	inv, err := a.Store.GetInvoiceBySalesID(ctx, p.InvoiceID)
	if err != nil {
		return err
	}
	if inv == nil {
		return fmt.Errorf("AR invoice untuk sales invoice %s belum ada", p.InvoiceID)
	}
	reason := orDefault(trim(p.Reason), "Invoice dibatalkan di Sales")
	return a.reverseSalesInvoice(ctx, p.CancelledBy, inv, reason)
}

func (a *AutoPost) reverseSalesInvoice(ctx context.Context, userID string, inv *domain.Invoice, reason string) error {
	if inv.Outstanding() == 0 {
		a.tellFinance(ctx, "Invoice "+inv.InvoiceNo+" dibatalkan di Sales, padahal sudah lunas", reason)
		return a.Store.SetInvoiceSalesReversal(ctx, inv.ID, reason, "", true)
	}
	_, noteErr := NewLedgers(a.Deps).CreateCreditNote(ctx, userID, inv.ID, CreditNoteInput{Reason: reason})
	if noteErr != nil && apperr.As(noteErr).Code == apperr.Unknown {
		return noteErr
	}
	failure := ""
	if noteErr != nil {
		failure = apperr.As(noteErr).Message
		a.tellFinance(ctx, "Invoice "+inv.InvoiceNo+" dibatalkan di Sales, jurnalnya belum terbalik", failure)
	} else if inv.Allocated > 0 {
		a.tellFinance(ctx, "Invoice "+inv.InvoiceNo+" dibatalkan di Sales, ada pembayaran tercatat", "Sisa tagihan sudah dikredit. Pembayarannya perlu diputuskan: "+reason)
	}
	return a.Store.SetInvoiceSalesReversal(ctx, inv.ID, reason, failure, noteErr == nil)
}

func (a *AutoPost) tellFinance(ctx context.Context, title, body string) {
	a.notify(ctx, NotifyInput{UserIDs: a.financeApprovers(ctx), Type: "finance_sales_reversal", Title: title, Body: body, Href: "/erp/finance/period-end?tab=controls"})
}

// retrySalesReversals runs in the daily schedule. A failure stays on the record.
func (s *Schedules) retrySalesReversals(ctx context.Context) {
	rows, err := s.Store.PendingSalesReversals(ctx)
	if err != nil {
		s.recordRunError(ctx, err, "sales reversal", "")
		return
	}
	auto := NewAutoPost(s.Deps)
	for _, r := range rows {
		if r.Kind == "PAYMENT" {
			s.recordRunError(ctx, auto.reverseSalesPayment(ctx, "", r.ID), "sales reversal", r.ID)
			continue
		}
		inv, err := s.Store.GetInvoice(ctx, r.ID)
		if err == nil && inv != nil {
			err = auto.reverseSalesInvoice(ctx, "", inv, r.Reason)
		}
		s.recordRunError(ctx, err, "sales reversal", r.ID)
	}
}

func (l *Ledgers) PendingSalesReversals(ctx context.Context) ([]domain.SalesReversal, error) {
	return l.Store.PendingSalesReversals(ctx)
}

// ─── Journal reversal ───────────────────────────────────────

const eventJournalReversal = "JOURNAL_REVERSAL"

// Reverse posts the opposite of a posted manual journal or VAT offset. Journals from
// a document reverse through that document, so its records stay in step.
func (j *Journals) Reverse(ctx context.Context, userID, id, date, reason string) (*domain.JournalEntry, error) {
	if reason = trim(reason); len([]rune(reason)) < 5 {
		return nil, apperr.Field("reason", "Alasan minimal 5 karakter")
	}
	e, err := j.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if e.Status != domain.StatusPosted {
		return nil, domain.Conflict("Hanya jurnal POSTED yang bisa dibalik")
	}
	if domain.NormalizeEntryType(e.EntryType) != domain.EntryManual && e.SourceEventCode != "VAT_CLOSING" {
		return nil, domain.Conflict("Jurnal otomatis dibalik lewat dokumen sumbernya")
	}
	if err := checkSeparateApprover(ctx, j.Store, e.CreatedBy, userID); err != nil {
		return nil, err
	}
	done, err := j.Store.FindEntryBySource(ctx, eventJournalReversal, id)
	if err != nil {
		return nil, err
	}
	if done != nil {
		return nil, domain.Conflict("Jurnal ini sudah dibalik oleh " + done.EntryNo)
	}
	day := j.today()
	if trim(date) != "" {
		if day, err = parseDay("reversal_date", date); err != nil {
			return nil, err
		}
	}
	if day < e.EntryDate {
		return nil, apperr.Field("reversal_date", "Tanggal pembalik tidak boleh sebelum tanggal jurnal")
	}
	return createEntry(ctx, j.Store, NewEntry{TagID: e.TagID, UserID: userID, EntryDate: day, Description: "Pembalik " + e.EntryNo + ": " + reason, Lines: domain.ReverseLines(e.DomainLines(), "Pembalik "+e.EntryNo), Post: true,
		EntryType: domain.EntryAuto, SourceModule: "finance", SourceEventCode: eventJournalReversal, SourceDocumentType: "journal_entry", SourceDocumentID: id})
}

// ─── Asset disposal ─────────────────────────────────────────

// DisposalInput sells or scraps an asset. CashAccountID is needed with proceeds.
type DisposalInput struct {
	Date, AssetAccountID, CashAccountID, GainLossAccountID, Reason string
	Proceeds                                                       domain.Money
}

// DisposalLines clears the asset: accumulated depreciation and any proceeds on the
// debit side, cost on the credit side, and the difference as gain or loss.
func DisposalLines(a domain.FixedAsset, in DisposalInput) []domain.Line {
	lines := []domain.Line{{AccountID: in.AssetAccountID, Side: domain.Credit, Amount: a.Cost, Memo: a.Name, SortOrder: 10}}
	if acc := a.Accumulated(); acc > 0 {
		lines = append(lines, domain.Line{AccountID: a.AccumAccountID, Side: domain.Debit, Amount: acc, Memo: a.Name})
	}
	if in.Proceeds > 0 {
		lines = append(lines, domain.Line{AccountID: in.CashAccountID, Side: domain.Debit, Amount: in.Proceeds, Memo: a.Name, SortOrder: 1})
	}
	switch diff := in.Proceeds - (a.Cost - a.Accumulated()); {
	case diff > 0:
		lines = append(lines, domain.Line{AccountID: in.GainLossAccountID, Side: domain.Credit, Amount: diff, Memo: "Laba pelepasan aset", SortOrder: 11})
	case diff < 0:
		lines = append(lines, domain.Line{AccountID: in.GainLossAccountID, Side: domain.Debit, Amount: -diff, Memo: "Rugi pelepasan aset", SortOrder: 2})
	}
	return lines
}

// DisposeAsset takes an asset off the books. Depreciation must be current to the month
// before the disposal, so the book value in the journal is the real one.
func (s *Schedules) DisposeAsset(ctx context.Context, userID, id string, in DisposalInput) (*domain.FixedAsset, error) {
	if trim(in.Reason) == "" {
		return nil, apperr.Field("reason", "Alasan wajib diisi")
	}
	day, err := parseDay("disposal_date", in.Date)
	if err != nil {
		return nil, err
	}
	a, err := s.Store.GetFixedAsset(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, domain.NotFound("Aset tidak ditemukan")
	}
	if a.DisposedOn != "" {
		return nil, domain.Conflict("Aset ini sudah dilepas")
	}
	if day < a.AcquiredOn {
		return nil, apperr.Field("disposal_date", "Tanggal pelepasan tidak boleh sebelum tanggal perolehan")
	}
	if n := a.DepreciatedMonths + 1; n <= a.LifeMonths {
		end, err := a.MonthEnd(n)
		if err != nil {
			return nil, err
		}
		if end < day[:8]+"01" {
			return nil, domain.Conflict("Penyusutan belum sampai bulan sebelum pelepasan. Jalankan jadwal penyusutan dulu.")
		}
	}
	check := func(field, accountID string, types ...string) error {
		acc, err := s.Store.GetAccount(ctx, accountID)
		if err != nil {
			return err
		}
		if acc == nil || !acc.IsPostable || !slices.Contains(types, acc.TypeCode) {
			return apperr.Field(field, "Akun tidak sesuai untuk pelepasan aset")
		}
		return nil
	}
	if err := check("asset_account_id", in.AssetAccountID, domain.TypeAsset); err != nil {
		return nil, err
	}
	if in.Proceeds < 0 {
		return nil, apperr.Field("proceeds", "Nilai jual tidak boleh negatif")
	}
	if in.Proceeds > 0 {
		cash, err := s.Store.GetAccount(ctx, in.CashAccountID)
		if err != nil {
			return nil, err
		}
		if cash == nil || !cash.IsCashBank || !cash.IsPostable {
			return nil, apperr.Field("cash_account_id", "Akun kas/bank tidak valid")
		}
	}
	if in.Proceeds != a.Cost-a.Accumulated() {
		if err := check("gain_loss_account_id", in.GainLossAccountID, domain.TypeOtherExpense, domain.TypeOtherIncome, domain.TypeExpense, domain.TypeRevenue); err != nil {
			return nil, err
		}
	}
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		e, err := createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: day, Description: "Pelepasan aset " + a.Name + ": " + trim(in.Reason), Lines: DisposalLines(*a, in), Post: true, EntryType: domain.EntryAuto,
			SourceModule: "finance", SourceEventCode: "ASSET_DISPOSAL", SourceDocumentType: "fixed_asset", SourceDocumentID: a.ID})
		if err != nil {
			return err
		}
		return tx.DisposeFixedAsset(ctx, id, day, e.ID, userID, in.Proceeds)
	})
	if err != nil {
		return nil, err
	}
	return s.Store.GetFixedAsset(ctx, id)
}

// ─── Purchase order matching ────────────────────────────────

// POMatch compares what was ordered, delivered and billed, all before tax.
type POMatch struct {
	Document                  domain.Document
	Ordered, Received, Billed domain.Money
	Receipts                  []domain.GoodsReceipt
	Invoices                  []domain.POInvoice
}

func receivedTotal(rows []domain.GoodsReceipt) domain.Money {
	var total domain.Money
	for _, r := range rows {
		if r.Status == domain.StatusPosted {
			total += r.Amount
		}
	}
	return total
}

func billedTotal(rows []domain.POInvoice) domain.Money {
	var total domain.Money
	for _, i := range rows {
		if i.Status == domain.StatusPosted {
			total += i.Subtotal
		}
	}
	return total
}

// ListPOMatches lists approved purchase orders with their receipts and invoices.
func (l *Ledgers) ListPOMatches(ctx context.Context) ([]POMatch, error) {
	docs, _, err := l.Store.ListDocuments(ctx, domain.DocumentFilter{Kind: domain.DocPO, Status: domain.DocApproved, Limit: 200})
	if err != nil {
		return nil, err
	}
	out := make([]POMatch, 0, len(docs))
	for _, d := range docs {
		receipts, err := l.Store.ListGoodsReceipts(ctx, d.ID)
		if err != nil {
			return nil, err
		}
		invoices, err := l.Store.POInvoices(ctx, d.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, POMatch{Document: d, Ordered: d.Subtotal - d.Discount, Received: receivedTotal(receipts), Billed: billedTotal(invoices), Receipts: receipts, Invoices: invoices})
	}
	return out, nil
}

// ReceiveGoods records a delivery, up to what the order has left.
func (l *Ledgers) ReceiveGoods(ctx context.Context, userID string, r domain.GoodsReceipt) (*domain.GoodsReceipt, error) {
	if r.Amount <= 0 {
		return nil, apperr.Field("amount", "Nilai harus lebih dari 0")
	}
	day, err := parseDay("receipt_date", r.Date)
	if err != nil {
		return nil, err
	}
	doc, err := l.Store.GetDocument(ctx, r.DocumentID)
	if err != nil {
		return nil, err
	}
	if doc == nil || doc.Kind != domain.DocPO || doc.Status != domain.DocApproved {
		return nil, domain.NotFound("Purchase order yang disetujui tidak ditemukan")
	}
	existing, err := l.Store.ListGoodsReceipts(ctx, r.DocumentID)
	if err != nil {
		return nil, err
	}
	if receivedTotal(existing)+r.Amount > doc.Subtotal-doc.Discount {
		return nil, apperr.Field("amount", "Nilai melebihi sisa purchase order")
	}
	r.ID, r.Date, r.Status, r.CreatedBy, r.Notes = ids.New(), day, domain.StatusPosted, userID, trim(r.Notes)
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		var err error
		if r.ReceiptNo, err = l.nextNo(ctx, tx, "GR"); err != nil {
			return err
		}
		return tx.InsertGoodsReceipt(ctx, r)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetGoodsReceipt(ctx, r.ID)
}

// VoidGoodsReceipt cancels a delivery, unless posted invoices already rely on it.
func (l *Ledgers) VoidGoodsReceipt(ctx context.Context, id, reason string) error {
	if reason = trim(reason); len([]rune(reason)) < 5 {
		return apperr.Field("reason", "Alasan minimal 5 karakter")
	}
	r, err := l.Store.GetGoodsReceipt(ctx, id)
	if err != nil {
		return err
	}
	if r == nil || r.Status != domain.StatusPosted {
		return domain.NotFound("Penerimaan aktif tidak ditemukan")
	}
	receipts, err := l.Store.ListGoodsReceipts(ctx, r.DocumentID)
	if err != nil {
		return err
	}
	invoices, err := l.Store.POInvoices(ctx, r.DocumentID)
	if err != nil {
		return err
	}
	if receivedTotal(receipts)-r.Amount < billedTotal(invoices) {
		return domain.Conflict("Tagihan yang sudah diposting memakai penerimaan ini")
	}
	return l.Store.VoidGoodsReceipt(ctx, id, reason)
}

// MatchInvoiceToPO links a draft vendor invoice to an order; an empty order unlinks it.
func (l *Ledgers) MatchInvoiceToPO(ctx context.Context, invoiceID, documentID string) error {
	inv, err := l.draftInvoice(ctx, domain.KindAP, invoiceID)
	if err != nil {
		return err
	}
	if documentID != "" {
		doc, err := l.Store.GetDocument(ctx, documentID)
		if err != nil {
			return err
		}
		if doc == nil || doc.Kind != domain.DocPO || doc.Status != domain.DocApproved {
			return domain.NotFound("Purchase order yang disetujui tidak ditemukan")
		}
	}
	return l.Store.SetInvoicePOLink(ctx, inv.ID, documentID)
}

// checkPOMatch refuses a vendor invoice billing more than the order has delivered.
func (l *Ledgers) checkPOMatch(ctx context.Context, inv domain.Invoice) error {
	documentID, err := l.Store.InvoicePOLink(ctx, inv.ID)
	if err != nil || documentID == "" {
		return err
	}
	receipts, err := l.Store.ListGoodsReceipts(ctx, documentID)
	if err != nil {
		return err
	}
	invoices, err := l.Store.POInvoices(ctx, documentID)
	if err != nil {
		return err
	}
	if billedTotal(invoices)+inv.Subtotal > receivedTotal(receipts) {
		return domain.Conflict("Nilai tagihan melebihi barang atau jasa yang sudah diterima pada purchase order ini")
	}
	return nil
}

// ─── Project cost allocation ────────────────────────────────

// Allocations for one period, with the months derived from its dates.
func (p *ProjectTags) ListAllocations(ctx context.Context, from, to string) ([]domain.ProjectAllocation, error) {
	from, to, err := NewReports(p.Deps).period(from, to)
	if err != nil {
		return nil, err
	}
	return p.Store.ListProjectAllocations(ctx, from[:7], to[:7])
}

// CreateAllocation assigns payroll or reimbursement cost to a project for a month.
func (p *ProjectTags) CreateAllocation(ctx context.Context, userID string, a domain.ProjectAllocation) ([]domain.ProjectAllocation, error) {
	if !taxPeriod.MatchString(a.Month) {
		return nil, apperr.Field("month", "Bulan harus berformat YYYY-MM")
	}
	if a.Kind != domain.AllocPayroll && a.Kind != domain.AllocReimbursement {
		return nil, apperr.Field("kind", "Jenis biaya tidak valid")
	}
	if a.Amount <= 0 {
		return nil, apperr.Field("amount", "Nilai harus lebih dari 0")
	}
	tag, err := p.Store.GetProjectTag(ctx, a.TagID)
	if err != nil {
		return nil, err
	}
	if tag == nil {
		return nil, apperr.Field("tag_id", "Proyek tidak ditemukan")
	}
	a.ID, a.CreatedBy, a.Notes, a.Source = ids.New(), userID, trim(a.Notes), domain.AllocManual
	if err := p.Store.InsertProjectAllocation(ctx, a); err != nil {
		return nil, err
	}
	return p.Store.ListProjectAllocations(ctx, a.Month, a.Month)
}

// TimesheetAllocation is the outcome of AllocateFromTimesheet. Skipped names
// projects HR reported that Finance does not have as a tag.
type TimesheetAllocation struct {
	Allocations []domain.ProjectAllocation
	Skipped     []string
}

// AllocateFromTimesheet replaces the month's timesheet allocations with the
// payroll cost HR spread over projects by approved hours. Manual
// allocations are left alone.
func (p *ProjectTags) AllocateFromTimesheet(ctx context.Context, userID, month string) (*TimesheetAllocation, error) {
	if !taxPeriod.MatchString(month) {
		return nil, apperr.Field("month", "Bulan harus berformat YYYY-MM")
	}
	if p.Timesheet == nil {
		return nil, domain.Conflict("Layanan HR tidak tersambung")
	}
	costs, err := p.Timesheet.ProjectCosts(ctx, month)
	if err != nil {
		return nil, err
	}
	out := &TimesheetAllocation{}
	var rows []domain.ProjectAllocation
	for _, c := range costs {
		amount, err := domain.ParseMoney(c.Cost)
		if err != nil || amount <= 0 {
			continue
		}
		tag, err := p.Store.GetProjectTag(ctx, c.ProjectID)
		if err != nil {
			return nil, err
		}
		if tag == nil {
			out.Skipped = append(out.Skipped, c.ProjectName)
			continue
		}
		rows = append(rows, domain.ProjectAllocation{ID: ids.New(), TagID: tag.ID, Month: month, Kind: domain.AllocPayroll, Amount: amount,
			Notes: fmt.Sprintf("Dari timesheet: %s jam", strconv.FormatFloat(c.Hours, 'f', -1, 64)), CreatedBy: userID, Source: domain.AllocTimesheet})
	}
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.DeleteTimesheetAllocations(ctx, month); err != nil {
			return err
		}
		for _, a := range rows {
			if err := tx.InsertProjectAllocation(ctx, a); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if out.Allocations, err = p.Store.ListProjectAllocations(ctx, month, month); err != nil {
		return nil, err
	}
	return out, nil
}

func (p *ProjectTags) DeleteAllocation(ctx context.Context, id string) error {
	return p.Store.DeleteProjectAllocation(ctx, id)
}

// applyAllocations moves allocated cost from the untagged row to its project. The
// total stays equal to the income statement, because the journals do not change.
func applyAllocations(rows []domain.ProjectProfitRow, allocations []domain.ProjectAllocation, clientName string) []domain.ProjectProfitRow {
	var moved domain.Money
	for _, a := range allocations {
		if clientName != "" && !strings.EqualFold(a.ClientName, clientName) {
			continue
		}
		i := slices.IndexFunc(rows, func(r domain.ProjectProfitRow) bool { return r.TagID == a.TagID })
		if i < 0 {
			rows = append(rows, domain.ProjectProfitRow{TagID: a.TagID, TagName: a.TagName, ClientName: a.ClientName})
			i = len(rows) - 1
		}
		rows[i].Allocated += a.Amount
		rows[i].Cost += a.Amount
		moved += a.Amount
	}
	if moved == 0 {
		return rows
	}
	i := slices.IndexFunc(rows, func(r domain.ProjectProfitRow) bool { return r.TagID == "" })
	if i < 0 {
		rows = append(rows, domain.ProjectProfitRow{})
		i = len(rows) - 1
	}
	rows[i].Allocated -= moved
	rows[i].Cost -= moved
	return rows
}

// ─── Overdue reminders ──────────────────────────────────────

// overdueStages are the days past due that trigger a reminder.
var overdueStages = []int{1, 7, 14, 30}

func overdueStage(dueDate, today string) (int, bool) {
	due, err := time.Parse(time.DateOnly, dueDate)
	if err != nil {
		return 0, false
	}
	now, err := time.Parse(time.DateOnly, today)
	if err != nil {
		return 0, false
	}
	days, stage := int(now.Sub(due).Hours()/24), 0
	for _, s := range overdueStages {
		if days >= s {
			stage = s
		}
	}
	return stage, stage > 0
}

// remindOverdueInvoices tells approvers about manual receivables past due. Sales
// invoices have their own client follow-ups, so they stay out.
func (s *Schedules) remindOverdueInvoices(ctx context.Context) error {
	approvers := s.financeApprovers(ctx)
	if len(approvers) == 0 {
		return nil
	}
	invoices, err := NewLedgers(s.Deps).OpenInvoices(ctx, domain.KindAR)
	if err != nil {
		return err
	}
	today := s.today()
	for _, inv := range invoices {
		if inv.SalesInvoiceID != "" || inv.DueDate == "" {
			continue
		}
		stage, due := overdueStage(inv.DueDate, today)
		if !due {
			continue
		}
		added, err := s.Store.InsertInvoiceReminder(ctx, inv.ID, stage)
		if err != nil {
			return err
		}
		if added {
			s.notify(ctx, NotifyInput{UserIDs: approvers, Type: "finance_invoice_overdue", Title: fmt.Sprintf("Tagihan %s lewat %d hari", inv.InvoiceNo, stage),
				Body: fmt.Sprintf("%s masih punya sisa %s", inv.PartyName, inv.Outstanding().String()), Href: "/erp/finance/receivables"})
		}
	}
	return nil
}
