package app

import (
	"context"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// AdjustmentInput is one part of a payment that is not cash.
type AdjustmentInput struct {
	Kind, AccountID, TaxID, Memo string
	Amount                       domain.Money
}

// resolveAdjustments checks each adjustment and fills its account and tax details.
func (l *Ledgers) resolveAdjustments(ctx context.Context, kind string, in PaymentInput) ([]domain.PaymentAdjustment, error) {
	if len(in.Adjustments) == 0 {
		return nil, nil
	}
	control, err := controlAccounts(ctx, l.Store)
	if err != nil {
		return nil, err
	}
	var settled domain.Money
	for _, a := range in.Allocations {
		settled += a.Amount
	}
	out := make([]domain.PaymentAdjustment, 0, len(in.Adjustments))
	for _, a := range in.Adjustments {
		if a.Amount <= 0 {
			return nil, apperr.Field("adjustments", "Nilai selisih harus lebih dari 0")
		}
		adj := domain.PaymentAdjustment{Kind: a.Kind, Amount: a.Amount, Memo: trim(a.Memo)}
		switch a.Kind {
		case domain.AdjustFee, domain.AdjustDiscount:
			acc, err := l.Store.GetAccount(ctx, a.AccountID)
			if err != nil {
				return nil, err
			}
			if _, isControl := control[a.AccountID]; acc == nil || !acc.IsPostable || acc.IsCashBank || isControl {
				return nil, apperr.Field("adjustments", "Akun selisih harus postable, bukan kas, dan bukan akun piutang atau hutang")
			}
			adj.AccountID = a.AccountID
		case domain.AdjustWithholding:
			if err := l.fillWithholding(ctx, kind, a.TaxID, in.Allocations, &adj); err != nil {
				return nil, err
			}
		default:
			return nil, apperr.Field("adjustments", "Jenis selisih tidak valid")
		}
		out = append(out, adj)
	}
	if domain.CashMoved(kind, settled, out) <= 0 {
		return nil, apperr.Field("adjustments", "Selisih tidak boleh menghabiskan seluruh nilai pembayaran")
	}
	return out, nil
}

// settlement is everything on a payment besides the cash: its adjustments and any
// excess, which the payment's own journal carries on the advance account.
func (l *Ledgers) settlement(ctx context.Context, kind string, in PaymentInput) ([]domain.PaymentAdjustment, error) {
	out, err := l.resolveAdjustments(ctx, kind, in)
	if err != nil || in.Excess == 0 {
		return out, err
	}
	if in.Excess < 0 {
		return nil, apperr.Field("excess_amount", "Kelebihan bayar harus lebih dari 0")
	}
	if err := l.checkAdvanceAccount(ctx, kind, in.ExcessAccountID); err != nil {
		return nil, err
	}
	return append(out, domain.PaymentAdjustment{Kind: domain.AdjustExcess, AccountID: in.ExcessAccountID, Amount: in.Excess, Memo: "Kelebihan bayar"}), nil
}

// linkExcessAdvance gives the party an advance for the document's excess. It has no
// journal of its own: the document's journal carries it and fills it in on posting.
func (l *Ledgers) linkExcessAdvance(ctx context.Context, tx domain.Store, userID string, p domain.Payment) error {
	for _, adj := range p.Adjustments {
		if adj.Kind != domain.AdjustExcess {
			continue
		}
		no, err := l.nextNo(ctx, tx, "DP")
		if err != nil {
			return err
		}
		return tx.InsertAdvance(ctx, domain.Advance{ID: ids.New(), Kind: p.Kind, AdvanceNo: no, Date: p.PaymentDate, PartyName: p.PartyName, Amount: adj.Amount, CashAccountID: p.CashAccountID, AdvanceAccountID: adj.AccountID,
			Reference: p.PaymentNo, Notes: adj.Memo + " " + p.PaymentNo, Status: domain.StatusPosted, JournalEntryID: p.JournalEntryID, SourcePaymentID: p.ID, CreatedBy: userID})
	}
	return nil
}

// voidExcessAdvance voids the advance a payment created. The payment's reversing
// journal undoes its money, so a used advance has to be freed first.
func voidExcessAdvance(ctx context.Context, tx domain.Store, paymentID, reason string) error {
	a, err := tx.AdvanceBySourcePayment(ctx, paymentID)
	if err != nil || a == nil || a.Status != domain.StatusPosted {
		return err
	}
	if err := tx.LockAdvance(ctx, a.ID); err != nil {
		return err
	}
	if a.Applied > 0 || a.Refunded > 0 {
		return domain.Conflict("Uang muka " + a.AdvanceNo + " dari kelebihan bayar sudah dipakai atau dikembalikan. Batalkan pemakaiannya dulu.")
	}
	return tx.VoidAdvance(ctx, a.ID, reason)
}

// fillWithholding takes the account and rate from the tax master. The tax ledger
// ties a move to one invoice, and an invoice that withheld on its own face already
// counted this tax, so both are refused.
func (l *Ledgers) fillWithholding(ctx context.Context, kind, taxID string, allocations []domain.Allocation, adj *domain.PaymentAdjustment) error {
	if len(allocations) != 1 {
		return apperr.Field("adjustments", "Potongan PPh hanya bisa untuk satu tagihan per pembayaran")
	}
	t, err := l.Store.GetTaxCode(ctx, taxID)
	if err != nil {
		return err
	}
	side := domain.SideB2B
	if kind == domain.KindAP {
		side = domain.SideSupplier
	}
	if t == nil || !t.IsActive || t.Kind != domain.TaxWithholding || !domain.ScopeFits(t.Scope, side) || t.AccountFor(side) == "" {
		return apperr.Field("adjustments", "Kode PPh tidak valid untuk transaksi ini atau akunnya belum diisi")
	}
	extras, err := l.Store.ListInvoicePostings(ctx, allocations[0].InvoiceID)
	if err != nil {
		return err
	}
	for _, x := range extras {
		if x.TaxKind == domain.TaxWithholding {
			return apperr.Field("adjustments", "Tagihan ini sudah memotong PPh, jadi tidak bisa dipotong lagi saat bayar")
		}
	}
	adj.AccountID, adj.TaxCode, adj.TaxName, adj.TaxRate, adj.ReportCode = t.AccountFor(side), t.Code, t.Name, t.Rate, t.ReportCode
	if t.Rate > 0 && t.BaseNumerator > 0 {
		adj.TaxBase = domain.MulDiv(domain.MulDiv(adj.Amount, domain.Rupiah(100), t.Rate), domain.Money(t.BaseDenominator), domain.Money(t.BaseNumerator))
	}
	return nil
}

// insertAdjustmentMoves enters withheld tax in the tax ledger. A void enters the
// same amounts negative.
func insertAdjustmentMoves(ctx context.Context, tx domain.Store, p domain.Payment, void bool) error {
	source, sign := domain.MovePayment, domain.Money(1)
	if void {
		source, sign = domain.MovePaymentVoid, -1
	}
	for _, a := range p.Adjustments {
		if a.TaxCode == "" {
			continue
		}
		if err := tx.InsertTaxMove(ctx, domain.TaxMove{Kind: p.Kind, SourceType: source, SourceID: p.ID, InvoiceID: p.Allocations[0].InvoiceID, TaxCode: a.TaxCode, TaxName: a.TaxName, TaxKind: domain.TaxWithholding,
			TaxRate: a.TaxRate, ReportCode: a.ReportCode, Date: p.PaymentDate, Base: sign * a.TaxBase, Amount: sign * a.Amount}); err != nil {
			return err
		}
	}
	return nil
}

// RefundInput returns what is left of an advance.
type RefundInput struct {
	Date, CashAccountID, Reason string
	Amount                      domain.Money
}

// RefundAdvance gives unused advance money back: to a client on AR, from a vendor
// on AP. It reverses the advance's own journal shape for the refunded amount.
func (l *Ledgers) RefundAdvance(ctx context.Context, userID, advanceID string, in RefundInput) (*domain.Advance, error) {
	if in.Reason = trim(in.Reason); len([]rune(in.Reason)) < 5 {
		return nil, apperr.Field("reason", "Alasan minimal 5 karakter")
	}
	day, err := parseDay("refund_date", in.Date)
	if err != nil {
		return nil, err
	}
	cash, err := l.Store.GetAccount(ctx, in.CashAccountID)
	if err != nil {
		return nil, err
	}
	if cash == nil || !cash.IsCashBank || !cash.IsPostable {
		return nil, apperr.Field("cash_account_id", "Akun kas/bank tidak valid")
	}
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.LockAdvance(ctx, advanceID); err != nil {
			return err
		}
		a, err := tx.GetAdvance(ctx, advanceID)
		if err != nil {
			return err
		}
		if a == nil {
			return domain.NotFound("Uang muka tidak ditemukan")
		}
		if a.JournalEntryID == "" {
			return domain.Conflict("Uang muka ini belum berjurnal. Selesaikan dulu di log posting.")
		}
		if in.Amount <= 0 {
			return apperr.Field("amount", "Nilai harus lebih dari 0")
		}
		if in.Amount > a.Remaining() {
			return apperr.Field("amount", "Nilai melebihi sisa uang muka")
		}
		if day < a.Date {
			return apperr.Field("refund_date", "Tanggal pengembalian tidak boleh sebelum tanggal uang muka")
		}
		r := domain.AdvanceRefund{ID: ids.New(), AdvanceID: a.ID, Date: day, Amount: in.Amount, CashAccountID: in.CashAccountID, Reason: in.Reason, Status: domain.StatusPosted, CreatedBy: userID}
		if r.RefundNo, err = l.nextNo(ctx, tx, "RF"); err != nil {
			return err
		}
		paid := *a
		paid.CashAccountID, paid.Amount, paid.AdvanceNo = in.CashAccountID, in.Amount, r.RefundNo
		e, err := createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: day, Description: "Pengembalian uang muka " + a.AdvanceNo + " " + a.PartyName + ": " + in.Reason, Lines: domain.ReverseLines(advanceLines(paid), r.RefundNo), Post: true,
			EntryType: domain.EntryAuto, SourceModule: kinds[a.Kind].module, SourceEventCode: "ADVANCE_REFUND", SourceDocumentType: "advance_refund", SourceDocumentID: r.ID})
		if err != nil {
			return err
		}
		r.JournalEntryID = e.ID
		return tx.InsertAdvanceRefund(ctx, r)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetAdvance(ctx, advanceID)
}

// VoidAdvanceRefund reverses a refund entered by mistake and frees the money
// again. The reversal is dated today, so a closed period refuses it.
func (l *Ledgers) VoidAdvanceRefund(ctx context.Context, userID, refundID, reason string) (*domain.Advance, error) {
	if reason = trim(reason); len([]rune(reason)) < 5 {
		return nil, apperr.Field("reason", "Alasan minimal 5 karakter")
	}
	r, err := l.Store.GetAdvanceRefund(ctx, refundID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, domain.NotFound("Pengembalian tidak ditemukan")
	}
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.LockAdvance(ctx, r.AdvanceID); err != nil {
			return err
		}
		current, err := tx.GetAdvanceRefund(ctx, refundID)
		if err != nil {
			return err
		}
		if current.Status != domain.StatusPosted {
			return domain.Conflict("Pengembalian ini sudah dibatalkan")
		}
		entry, err := tx.GetJournalEntry(ctx, current.JournalEntryID)
		if err != nil {
			return err
		}
		if entry == nil {
			return domain.NotFound("Jurnal pengembalian tidak ditemukan")
		}
		if _, err := createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: l.today(), Description: "Batal pengembalian " + current.RefundNo + ": " + reason, Lines: domain.ReverseLines(entry.DomainLines(), "Pembalik "+entry.EntryNo), Post: true,
			EntryType: domain.EntryAuto, SourceModule: entry.SourceModule, SourceEventCode: "ADVANCE_REFUND_VOID", SourceDocumentType: "advance_refund", SourceDocumentID: current.ID}); err != nil {
			return err
		}
		return tx.VoidAdvanceRefund(ctx, refundID, reason)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetAdvance(ctx, r.AdvanceID)
}
