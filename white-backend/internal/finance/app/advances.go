package app

import (
	"context"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

const methodAdvance = "advance"

// AdvanceInput is money paid to a vendor or received from a client up front.
type AdvanceInput struct {
	Kind, Date, PartyName, CashAccountID, AdvanceAccountID, Reference, Notes string
	Amount                                                                   domain.Money
	// SalesPaymentID marks an advance a Sales payment created on its own.
	SalesPaymentID string
}

func (l *Ledgers) ListAdvances(ctx context.Context, kind string) ([]domain.Advance, error) {
	return l.Store.ListAdvances(ctx, kind)
}

// checkAdvanceAccount requires a postable non-cash account of the right type: an
// asset for what we paid a vendor, a liability for what a client paid us.
func (l *Ledgers) checkAdvanceAccount(ctx context.Context, kind, id string) error {
	acc, err := l.Store.GetAccount(ctx, id)
	if err != nil {
		return err
	}
	want := domain.TypeLiability
	label := "liabilitas"
	if kind == domain.KindAP {
		want, label = domain.TypeAsset, "aset"
	}
	if acc == nil || !acc.IsPostable || acc.IsCashBank || acc.TypeCode != want {
		return apperr.Field("advance_account_id", "Akun uang muka harus akun "+label+" yang postable dan bukan kas")
	}
	return nil
}

// advanceLines moves cash against the advance account.
func advanceLines(a domain.Advance) []domain.Line {
	cashSide, advanceSide := domain.Debit, domain.Credit
	if a.Kind == domain.KindAP {
		cashSide, advanceSide = domain.Credit, domain.Debit
	}
	return []domain.Line{
		{AccountID: a.CashAccountID, Side: cashSide, Amount: a.Amount, Memo: a.AdvanceNo, SortOrder: 0},
		{AccountID: a.AdvanceAccountID, Side: advanceSide, Amount: a.Amount, Memo: a.AdvanceNo, SortOrder: 1},
	}
}

// CreateAdvance records the money and its journal together, so an advance never
// exists without its entry.
func (l *Ledgers) CreateAdvance(ctx context.Context, userID string, in AdvanceInput) (*domain.Advance, error) {
	in.PartyName = trim(in.PartyName)
	check := validate.New().Required("party_name", in.PartyName, "Nama pihak wajib diisi")
	if in.Kind != domain.KindAP && in.Kind != domain.KindAR {
		check.Add("kind", "Jenis uang muka tidak valid")
	}
	if in.Amount <= 0 {
		check.Add("amount", "Nilai uang muka harus lebih dari 0")
	}
	if err := check.Err(); err != nil {
		return nil, err
	}
	day, err := parseDay("advance_date", in.Date)
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
	if err := l.checkAdvanceAccount(ctx, in.Kind, in.AdvanceAccountID); err != nil {
		return nil, err
	}
	a := domain.Advance{ID: ids.New(), Kind: in.Kind, Date: day, PartyName: in.PartyName, Amount: in.Amount, CashAccountID: in.CashAccountID, AdvanceAccountID: in.AdvanceAccountID,
		Reference: trim(in.Reference), Notes: trim(in.Notes), Status: domain.StatusPosted, SalesPaymentID: in.SalesPaymentID, CreatedBy: userID}
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if a.AdvanceNo, err = l.nextNo(ctx, tx, "DP"); err != nil {
			return err
		}
		e, err := createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: day, Description: "Uang muka " + a.AdvanceNo + " " + a.PartyName, Lines: advanceLines(a), Post: true, EntryType: domain.EntryAuto,
			SourceModule: kinds[a.Kind].module, SourceEventCode: "ADVANCE", SourceDocumentType: "advance", SourceDocumentID: a.ID})
		if err != nil {
			return err
		}
		a.JournalEntryID = e.ID
		return tx.InsertAdvance(ctx, a)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetAdvance(ctx, a.ID)
}

// ApplyAdvance settles an invoice from an advance. It is a payment whose
// "cash" account is the advance account, so the journal, allocation and cash
// basis VAT follow the normal path, and voiding the payment frees the advance.
func (l *Ledgers) ApplyAdvance(ctx context.Context, userID, advanceID, invoiceID, date string, amount domain.Money) (*PaymentResult, error) {
	var p domain.Payment
	var invoiceIDs []string
	var note string
	err := l.Store.WithTx(ctx, func(tx domain.Store) error {
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
		if amount <= 0 {
			return apperr.Field("amount", "Nilai harus lebih dari 0")
		}
		if amount > a.Remaining() {
			return apperr.Field("amount", "Nilai melebihi sisa uang muka")
		}
		inv, err := tx.GetInvoice(ctx, invoiceID)
		if err != nil {
			return err
		}
		if inv == nil || inv.Kind != a.Kind {
			return domain.NotFound("Tagihan tidak ditemukan")
		}
		if !strings.EqualFold(inv.PartyName, a.PartyName) {
			return domain.Bad("Uang muka hanya bisa dipakai untuk tagihan pihak yang sama")
		}
		if strings.TrimSpace(date) != "" && date < a.Date {
			return apperr.Field("payment_date", "Tanggal pemakaian tidak boleh sebelum tanggal uang muka")
		}
		in := PaymentInput{PaymentDate: date, Method: methodAdvance, CashAccountID: a.AdvanceAccountID, Reference: a.AdvanceNo, Allocations: []domain.Allocation{{InvoiceID: invoiceID, Amount: amount}}}
		if p, invoiceIDs, err = l.insertPayment(ctx, tx, a.Kind, userID, in); err != nil {
			return err
		}
		if err := tx.InsertAdvanceApplication(ctx, p.ID, a.ID); err != nil {
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

// VoidAdvance reverses an advance nothing uses. To use part of it elsewhere,
// void its applications first.
func (l *Ledgers) VoidAdvance(ctx context.Context, userID, id, reason string) (*domain.Advance, error) {
	if reason = trim(reason); len([]rune(reason)) < 5 {
		return nil, apperr.Field("reason", "Alasan minimal 5 karakter")
	}
	err := l.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.LockAdvance(ctx, id); err != nil {
			return err
		}
		a, err := tx.GetAdvance(ctx, id)
		if err != nil {
			return err
		}
		if a == nil {
			return domain.NotFound("Uang muka tidak ditemukan")
		}
		if a.Status != domain.StatusPosted {
			return domain.Conflict("Uang muka ini sudah dibatalkan")
		}
		if a.SourcePaymentID != "" {
			return domain.Conflict("Uang muka ini lahir dari " + a.Reference + ". Batalkan dokumen itu.")
		}
		if a.Applied > 0 {
			return domain.Conflict("Uang muka sudah dipakai tagihan. Batalkan pemakaiannya dulu.")
		}
		if a.Refunded > 0 {
			return domain.Conflict("Uang muka sudah dikembalikan sebagian. Batalkan pengembaliannya dulu.")
		}
		_, err = createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: l.today(), Description: "Batal uang muka " + a.AdvanceNo + ": " + reason, Lines: domain.ReverseLines(advanceLines(*a), "Pembalik "+a.AdvanceNo), Post: true, EntryType: domain.EntryAuto,
			SourceModule: kinds[a.Kind].module, SourceEventCode: "ADVANCE_VOID", SourceDocumentType: "advance", SourceDocumentID: a.ID})
		if err != nil {
			return err
		}
		return tx.VoidAdvance(ctx, id, reason)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetAdvance(ctx, id)
}
