package app

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

const (
	methodWriteOff  = "write_off"
	eventWriteOff   = "WRITE_OFF"
	taxPayableGroup = "2103"
)

// ─── Separation of duties ───────────────────────────────────

// checkSeparateApprover refuses a person posting, paying, crediting or writing off
// their own document while the company
// requires a second person.
func checkSeparateApprover(ctx context.Context, s domain.Store, createdBy, userID string) error {
	on, err := s.SeparateApprover(ctx)
	if err != nil || !on || createdBy == "" || createdBy != userID {
		return err
	}
	return domain.Conflict("Pembuat dokumen tidak boleh memproses dokumennya sendiri. Minta rekan lain memeriksanya.")
}

func (l *Ledgers) SeparateApprover(ctx context.Context) (bool, error) {
	return l.Store.SeparateApprover(ctx)
}

func (l *Ledgers) SetSeparateApprover(ctx context.Context, userID string, on bool) error {
	return l.Store.SetSeparateApprover(ctx, on, userID)
}

// ─── Write-off ──────────────────────────────────────────────

// WriteOffInput gives up on what a client still owes.
type WriteOffInput struct {
	Date, ExpenseAccountID, Reason string
	Amount                         domain.Money
}

// WriteOffInvoice moves an uncollectible receivable to expense. It allocates like a
// payment without cash, and voiding it brings the receivable back when the client
// pays after all. VAT stays as it was: a write-off does not cancel the tax invoice.
func (l *Ledgers) WriteOffInvoice(ctx context.Context, userID, invoiceID string, in WriteOffInput) (*domain.Invoice, error) {
	reason := trim(in.Reason)
	if len([]rune(reason)) < 5 {
		return nil, apperr.Field("reason", "Alasan minimal 5 karakter")
	}
	day, err := parseDay("write_off_date", in.Date)
	if err != nil {
		return nil, err
	}
	acc, err := l.Store.GetAccount(ctx, in.ExpenseAccountID)
	if err != nil {
		return nil, err
	}
	if acc == nil || !acc.IsPostable || !slices.Contains([]string{domain.TypeExpense, domain.TypeOtherExpense}, acc.TypeCode) {
		return nil, apperr.Field("expense_account_id", "Akun harus akun beban yang postable")
	}
	inv, err := l.Store.GetInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil || inv.Kind != domain.KindAR {
		return nil, domain.NotFound("Tagihan piutang tidak ditemukan")
	}
	if err := checkSeparateApprover(ctx, l.Store, inv.CreatedBy, userID); err != nil {
		return nil, err
	}
	if inv.Status != domain.StatusPosted || inv.JournalEntryID == "" {
		return nil, domain.Conflict("Write-off butuh tagihan yang sudah diposting dan punya jurnal")
	}
	amount := in.Amount
	if amount == 0 {
		amount = inv.Outstanding()
	}
	if amount <= 0 || amount > inv.Outstanding() {
		return nil, apperr.Field("amount", "Nilai write-off melebihi sisa tagihan")
	}
	if day < inv.InvoiceDate {
		return nil, apperr.Field("write_off_date", "Tanggal tidak boleh sebelum tanggal tagihan")
	}
	control, err := controlAccounts(ctx, l.Store)
	if err != nil {
		return nil, err
	}
	receivable := ""
	for id, kind := range control {
		if kind == domain.KindAR {
			receivable = id
		}
	}
	if receivable == "" {
		return nil, domain.Conflict("Akun piutang pada journal mapping belum diisi")
	}
	p := domain.Payment{ID: ids.New(), Kind: inv.Kind, PaymentDate: day, Method: methodWriteOff, Amount: amount, Reference: inv.InvoiceNo, Notes: reason, Status: domain.StatusPosted, PartyName: inv.PartyName, CreatedBy: userID,
		Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: amount}}}
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if p.PaymentNo, err = l.nextNo(ctx, tx, "WO"); err != nil {
			return err
		}
		if err := tx.InsertPayment(ctx, p); err != nil {
			return err
		}
		e, err := createEntry(ctx, tx, NewEntry{TagID: inv.TagID, UserID: userID, EntryDate: day, Description: "Write-off " + p.PaymentNo + " atas " + inv.InvoiceNo + ": " + reason, Post: true,
			EntryType: domain.EntryAuto, SourceModule: kinds[inv.Kind].module, SourceEventCode: eventWriteOff, SourceDocumentType: DocPayment, SourceDocumentID: p.ID, Lines: []domain.Line{
				{AccountID: in.ExpenseAccountID, Side: domain.Debit, Amount: amount, Memo: inv.InvoiceNo},
				{AccountID: receivable, Side: domain.Credit, Amount: amount, Memo: inv.InvoiceNo, SortOrder: 1},
			}})
		if err != nil {
			return err
		}
		return tx.SetPaymentJournal(ctx, p.ID, e.ID)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetInvoice(ctx, inv.ID)
}

// ─── Deferrals ──────────────────────────────────────────────

func (l *Ledgers) ListDeferrals(ctx context.Context) ([]domain.Deferral, error) {
	return l.Store.ListDeferrals(ctx, false)
}

// CreateDeferral schedules money already on the deferred account. An invoice or a
// journal puts it there, so the schedule posts only the monthly recognition.
func (l *Ledgers) CreateDeferral(ctx context.Context, userID string, d domain.Deferral) (*domain.Deferral, error) {
	d.Name = trim(d.Name)
	if d.Name == "" {
		return nil, apperr.Field("name", "Nama jadwal wajib diisi")
	}
	if d.Months < 1 || d.Months > 120 {
		return nil, apperr.Field("months", "Jumlah bulan harus 1 sampai 120")
	}
	if d.Amount <= 0 {
		return nil, apperr.Field("amount", "Nilai harus lebih dari 0")
	}
	start, err := parseDay("start_date", d.StartDate)
	if err != nil {
		return nil, err
	}
	deferred, targets := domain.TypeLiability, []string{domain.TypeRevenue, domain.TypeOtherIncome}
	switch d.Kind {
	case domain.DeferRevenue:
	case domain.DeferExpense:
		deferred, targets = domain.TypeAsset, []string{domain.TypeExpense, domain.TypeOtherExpense}
	default:
		return nil, apperr.Field("kind", "Jenis jadwal tidak valid")
	}
	acc, err := l.Store.GetAccount(ctx, d.DeferredAccountID)
	if err != nil {
		return nil, err
	}
	if acc == nil || !acc.IsPostable || acc.IsCashBank || acc.TypeCode != deferred {
		return nil, apperr.Field("deferred_account_id", "Akun di muka tidak sesuai dengan jenis jadwal")
	}
	target, err := l.Store.GetAccount(ctx, d.TargetAccountID)
	if err != nil {
		return nil, err
	}
	if target == nil || !target.IsPostable || !slices.Contains(targets, target.TypeCode) {
		return nil, apperr.Field("target_account_id", "Akun tujuan tidak sesuai dengan jenis jadwal")
	}
	d.ID, d.StartDate, d.CreatedBy, d.PartyName, d.Reference = ids.New(), start, userID, trim(d.PartyName), trim(d.Reference)
	if err := l.Store.InsertDeferral(ctx, d); err != nil {
		return nil, err
	}
	return l.Store.GetDeferral(ctx, d.ID)
}

func (l *Ledgers) SetDeferralActive(ctx context.Context, id string, active bool) error {
	d, err := l.Store.GetDeferral(ctx, id)
	if err != nil {
		return err
	}
	if d == nil {
		return domain.NotFound("Jadwal tidak ditemukan")
	}
	if active && d.PostedMonths >= d.Months {
		return domain.Conflict("Jadwal ini sudah selesai")
	}
	return l.Store.SetDeferralActive(ctx, id, active)
}

func (l *Ledgers) DeleteDeferral(ctx context.Context, id string) error {
	d, err := l.Store.GetDeferral(ctx, id)
	if err != nil || d == nil {
		return err
	}
	if d.PostedMonths > 0 {
		return domain.Conflict("Jadwal yang sudah berjalan tidak bisa dihapus. Hentikan saja.")
	}
	return l.Store.DeleteDeferral(ctx, id)
}

func (s *Schedules) runDeferral(ctx context.Context, d domain.Deferral, today string, res *RunResult) {
	for range maxCatchUp {
		n := d.PostedMonths + 1
		end, err := d.MonthEnd(n)
		if n > d.Months || err != nil || end > today {
			return
		}
		before := d
		err = s.Store.WithTx(ctx, func(tx domain.Store) error {
			e, err := createEntry(ctx, tx, NewEntry{TagID: d.TagID, EntryDate: end, Description: fmt.Sprintf("Pengakuan %s %s", d.Name, end[:7]), Post: true, EntryType: domain.EntryAuto,
				SourceModule: "finance", SourceEventCode: "DEFERRAL", SourceDocumentType: "deferral", SourceDocumentID: fmt.Sprintf("%s:%d", d.ID, n), Lines: d.Lines(d.AmountFor(n))})
			if err != nil {
				return err
			}
			d.PostedMonths, d.LastEntryID, d.LastError, d.IsActive = n, e.ID, "", n < d.Months
			return tx.UpdateDeferralRun(ctx, d)
		})
		if err != nil {
			res.Failed++
			before.LastError = apperr.As(err).Message
			s.recordRunError(ctx, s.Store.UpdateDeferralRun(ctx, before), "deferral", d.ID)
			return
		}
		res.Posted++
		if !d.IsActive {
			return
		}
	}
}

// ─── Tax remittance ─────────────────────────────────────────

// TaxPositions lists what is owed to the state and what is credited against it.
// Tax payables are the tax group plus the accounts of the tax master, so PPh 21 from
// payroll appears without a tax code.
func (l *Ledgers) TaxPositions(ctx context.Context, asOf string) ([]domain.TaxPosition, error) {
	if trim(asOf) == "" {
		asOf = l.today()
	}
	codes, err := l.Store.ListTaxCodes(ctx, false)
	if err != nil {
		return nil, err
	}
	master := map[string]bool{}
	for _, t := range codes {
		master[t.AccountSaleID], master[t.AccountPurchaseID] = true, true
	}
	balances, err := l.Store.AccountBalances(ctx, domain.BalanceWindow{AsOf: asOf})
	if err != nil {
		return nil, err
	}
	out := []domain.TaxPosition{}
	for _, b := range balances {
		kind := ""
		switch {
		case b.TypeCode == domain.TypeLiability && (master[b.ID] || strings.HasPrefix(b.Code, taxPayableGroup)):
			kind = "PAYABLE"
		case b.TypeCode == domain.TypeAsset && master[b.ID]:
			kind = "PREPAID"
		}
		if kind != "" {
			out = append(out, domain.TaxPosition{AccountID: b.ID, Code: b.Code, Name: b.Name, Kind: kind, Balance: b.Balance})
		}
	}
	return out, nil
}

var taxPeriod = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func (l *Ledgers) ListTaxRemittances(ctx context.Context) ([]domain.TaxRemittance, error) {
	return l.Store.ListTaxRemittances(ctx)
}

// RemitTax pays a tax payable to the state. NTPN is the state's proof of payment.
func (l *Ledgers) RemitTax(ctx context.Context, userID string, r domain.TaxRemittance) (*domain.TaxRemittance, error) {
	if !taxPeriod.MatchString(r.Period) {
		return nil, apperr.Field("period", "Masa pajak harus berformat YYYY-MM")
	}
	day, err := parseDay("remit_date", r.Date)
	if err != nil {
		return nil, err
	}
	if r.Amount <= 0 {
		return nil, apperr.Field("amount", "Nilai harus lebih dari 0")
	}
	cash, err := l.Store.GetAccount(ctx, r.CashAccountID)
	if err != nil {
		return nil, err
	}
	if cash == nil || !cash.IsCashBank || !cash.IsPostable {
		return nil, apperr.Field("cash_account_id", "Akun kas/bank tidak valid")
	}
	positions, err := l.TaxPositions(ctx, day)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(positions, func(p domain.TaxPosition) bool { return p.AccountID == r.AccountID && p.Kind == "PAYABLE" })
	if i < 0 {
		return nil, apperr.Field("account_id", "Akun harus akun hutang pajak")
	}
	if r.Amount > positions[i].Balance {
		return nil, apperr.Field("amount", "Nilai melebihi saldo hutang pajak")
	}
	r.ID, r.Date, r.Status, r.CreatedBy, r.NTPN, r.Notes = ids.New(), day, domain.StatusPosted, userID, trim(r.NTPN), trim(r.Notes)
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		var err error
		if r.RemitNo, err = l.nextNo(ctx, tx, "SP"); err != nil {
			return err
		}
		e, err := createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: day, Description: "Setor pajak " + r.RemitNo + " masa " + r.Period + " " + positions[i].Name, Post: true, EntryType: domain.EntryAuto,
			SourceModule: "finance", SourceEventCode: "TAX_REMITTANCE", SourceDocumentType: "tax_remittance", SourceDocumentID: r.ID, Lines: []domain.Line{
				{AccountID: r.AccountID, Side: domain.Debit, Amount: r.Amount, Memo: r.NTPN},
				{AccountID: r.CashAccountID, Side: domain.Credit, Amount: r.Amount, Memo: r.NTPN, SortOrder: 1},
			}})
		if err != nil {
			return err
		}
		r.JournalEntryID = e.ID
		return tx.InsertTaxRemittance(ctx, r)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetTaxRemittance(ctx, r.ID)
}

// VoidTaxRemittance reverses a remittance entered by mistake, dated today.
func (l *Ledgers) VoidTaxRemittance(ctx context.Context, userID, id, reason string) (*domain.TaxRemittance, error) {
	if reason = trim(reason); len([]rune(reason)) < 5 {
		return nil, apperr.Field("reason", "Alasan minimal 5 karakter")
	}
	r, err := l.Store.GetTaxRemittance(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, domain.NotFound("Setoran tidak ditemukan")
	}
	if r.Status != domain.StatusPosted {
		return nil, domain.Conflict("Setoran ini sudah dibatalkan")
	}
	entry, err := l.Store.GetJournalEntry(ctx, r.JournalEntryID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, domain.NotFound("Jurnal setoran tidak ditemukan")
	}
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if _, err := createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: l.today(), Description: "Batal setor pajak " + r.RemitNo + ": " + reason, Lines: domain.ReverseLines(entry.DomainLines(), "Pembalik "+entry.EntryNo), Post: true,
			EntryType: domain.EntryAuto, SourceModule: "finance", SourceEventCode: "TAX_REMITTANCE_VOID", SourceDocumentType: "tax_remittance", SourceDocumentID: r.ID}); err != nil {
			return err
		}
		return tx.VoidTaxRemittance(ctx, id, reason)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetTaxRemittance(ctx, id)
}

// VATClosing is what one month's closing offset.
type VATClosing struct {
	Offset, Payable, Compensation domain.Money
	JournalEntryID                string
}

// CloseVAT offsets input VAT against output VAT at the end of a month. What stays on
// the output account is kurang bayar, to remit. What stays on the input account is
// lebih bayar, carried to the next month as compensation. The offset works from
// balances, so running it again after a late invoice books only the new difference.
func (l *Ledgers) CloseVAT(ctx context.Context, userID, period string) (*VATClosing, error) {
	if !taxPeriod.MatchString(period) {
		return nil, apperr.Field("period", "Masa pajak harus berformat YYYY-MM")
	}
	first, err := time.Parse(time.DateOnly, period+"-01")
	if err != nil {
		return nil, apperr.Field("period", "Masa pajak tidak valid")
	}
	end := first.AddDate(0, 1, -1).Format(time.DateOnly)
	codes, err := l.Store.ListTaxCodes(ctx, false)
	if err != nil {
		return nil, err
	}
	output, input := map[string]bool{}, map[string]bool{}
	for _, t := range codes {
		if t.Kind == domain.TaxVAT {
			output[t.AccountSaleID], input[t.AccountPurchaseID] = true, true
		}
	}
	balances, err := l.Store.AccountBalances(ctx, domain.BalanceWindow{AsOf: end})
	if err != nil {
		return nil, err
	}
	var payable, credit domain.Money
	for _, b := range balances {
		if b.Balance <= 0 {
			continue
		}
		if output[b.ID] {
			payable += b.Balance
		} else if input[b.ID] {
			credit += b.Balance
		}
	}
	offset := min(payable, credit)
	if offset == 0 {
		return nil, domain.Conflict("Tidak ada PPN untuk dikompensasi pada masa ini")
	}
	lines, leftOutput, leftInput := []domain.Line{}, offset, offset
	for _, b := range balances {
		if b.Balance <= 0 {
			continue
		}
		if output[b.ID] && leftOutput > 0 {
			part := min(leftOutput, b.Balance)
			leftOutput -= part
			lines = append(lines, domain.Line{AccountID: b.ID, Side: domain.Debit, Amount: part, Memo: "Kompensasi PPN " + period, SortOrder: len(lines)})
		} else if input[b.ID] && leftInput > 0 {
			part := min(leftInput, b.Balance)
			leftInput -= part
			lines = append(lines, domain.Line{AccountID: b.ID, Side: domain.Credit, Amount: part, Memo: "Kompensasi PPN " + period, SortOrder: len(lines)})
		}
	}
	e, err := createEntry(ctx, l.Store, NewEntry{UserID: userID, EntryDate: end, Description: "Kompensasi PPN masa " + period, Lines: lines, Post: true, EntryType: domain.EntryAuto,
		SourceModule: "finance", SourceEventCode: "VAT_CLOSING", SourceDocumentType: "vat_closing", SourceDocumentID: period + ":" + ids.New()})
	if err != nil {
		return nil, err
	}
	return &VATClosing{Offset: offset, Payable: payable - offset, Compensation: credit - offset, JournalEntryID: e.ID}, nil
}
