package app

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// naturalSides is where VAT sits on an invoice and where withholding sits: a
// receivable credits output VAT and debits PPh the client held back; a payable
// debits input VAT and credits PPh we hold back.
func naturalSides(kind string) (natural, other string) {
	if kind == domain.KindAP {
		return domain.Debit, domain.Credit
	}
	return domain.Credit, domain.Debit
}

// resolveTaxes turns chosen tax ids into copies for a party on a date: the party's
// fiscal position swaps codes first, then each code is checked.
func resolveTaxes(ctx context.Context, s domain.Store, taxIDs []string, side, date, party string) ([]domain.DocTax, error) {
	profile, err := s.GetPartyProfile(ctx, side, party)
	if err != nil {
		return nil, err
	}
	ids := taxIDs
	if profile != nil && profile.FiscalPositionID != "" {
		fp, err := s.GetFiscalPosition(ctx, profile.FiscalPositionID)
		if err != nil {
			return nil, err
		}
		if fp != nil && fp.IsActive {
			ids = applyFiscalPosition(taxIDs, fp.Rules)
		}
	}
	taxes := make([]domain.DocTax, 0, len(ids))
	seen := map[string]bool{}
	vat := 0
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		t, err := s.GetTaxCode(ctx, id)
		if err != nil {
			return nil, err
		}
		if t == nil || !t.IsActive {
			return nil, apperr.Field("tax_ids", "Kode pajak tidak ditemukan atau nonaktif")
		}
		if !domain.ScopeFits(t.Scope, side) {
			return nil, apperr.Field("tax_ids", t.Code+" tidak berlaku untuk pihak ini")
		}
		if (t.ValidFrom != "" && date < t.ValidFrom) || (t.ValidTo != "" && date > t.ValidTo) {
			return nil, apperr.Field("tax_ids", t.Code+" tidak berlaku pada tanggal dokumen")
		}
		if t.Kind == domain.TaxVAT {
			if vat++; vat > 1 {
				return nil, apperr.Field("tax_ids", "Satu dokumen hanya boleh memakai satu kode PPN")
			}
			// Only a PKP may charge VAT. A fiscal position can drop the code instead.
			if side == domain.SideSupplier && profile != nil && !profile.IsPKP {
				return nil, apperr.Field("tax_ids", party+" bukan PKP, jadi tidak bisa memungut PPN")
			}
		}
		taxes = append(taxes, domain.DocTax{TaxID: t.ID, Code: t.Code, Name: t.Name, Kind: t.Kind, Rate: t.Rate, BaseNumerator: t.BaseNumerator, BaseDenominator: t.BaseDenominator,
			PriceIncluded: t.PriceIncluded, AccountID: t.AccountFor(side), ReportCode: t.ReportCode, Brackets: t.Brackets, CashBasis: t.CashBasis, TransitionAccountID: t.TransitionAccountID})
	}
	return taxes, nil
}

func applyFiscalPosition(taxIDs []string, rules []domain.FiscalRule) []string {
	out := make([]string, 0, len(taxIDs))
	for _, id := range taxIDs {
		mapped := id
		for _, r := range rules {
			if r.FromTaxID == id {
				mapped = r.ToTaxID
				break
			}
		}
		if mapped != "" {
			out = append(out, mapped)
		}
	}
	return out
}

// taxPostings turns copied taxes into invoice journal lines.
func taxPostings(taxes []domain.DocTax, kind string) ([]domain.InvoicePosting, error) {
	natural, other := naturalSides(kind)
	out := []domain.InvoicePosting{}
	for _, t := range taxes {
		if t.Amount == 0 {
			continue
		}
		if t.AccountID == "" {
			return nil, apperr.Field("account_id", "Kode pajak "+t.Code+" belum punya akun untuk pihak ini. Lengkapi di Master pajak.")
		}
		side := natural
		if t.Kind == domain.TaxWithholding {
			side = other
		}
		p := domain.InvoicePosting{AccountID: t.AccountID, Side: side, Amount: t.Amount, Memo: t.Name, SortOrder: len(out), TaxCode: t.Code, TaxName: t.Name, TaxKind: t.Kind, TaxBase: t.Base,
			TaxID: t.TaxID, TaxRate: t.Rate, ReportCode: t.ReportCode}
		if t.CashBasis && t.TransitionAccountID != "" {
			p.CashBasis, p.AccountID, p.FinalAccountID = true, t.TransitionAccountID, t.AccountID
		}
		out = append(out, p)
	}
	return out, nil
}

const methodCreditNote = "credit_note"

// InvoiceTaxIDs lists the master codes an invoice uses, for its edit form.
func (l *Ledgers) InvoiceTaxIDs(ctx context.Context, invoiceID string) ([]string, error) {
	extras, err := l.Store.ListInvoicePostings(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, p := range extras {
		if p.TaxID != "" {
			out = append(out, p.TaxID)
		}
	}
	return out, nil
}

// CreateCreditNote lowers what a party owes on a posted invoice. It reverses the
// invoice's journal pro rata, so every line (revenue, each tax, withholding, the
// receivable) comes back in proportion, and it allocates like a payment without
// cash. Amount zero credits all that is outstanding.
func (l *Ledgers) CreateCreditNote(ctx context.Context, userID, invoiceID, date string, amount domain.Money, reason string) (*domain.Invoice, error) {
	inv, err := l.Store.GetInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, domain.NotFound("Tagihan tidak ditemukan")
	}
	if inv.Status != domain.StatusPosted || inv.JournalEntryID == "" {
		return nil, domain.Conflict("Credit note butuh tagihan yang sudah diposting dan punya jurnal")
	}
	if reason = trim(reason); reason == "" {
		return nil, apperr.Field("reason", "Alasan wajib diisi")
	}
	if amount == 0 {
		amount = inv.Outstanding()
	}
	if amount <= 0 || amount > inv.Outstanding() {
		return nil, apperr.Field("amount", "Nilai credit note melebihi sisa tagihan")
	}
	day := l.today()
	if trim(date) != "" {
		if day, err = parseDay("note_date", date); err != nil {
			return nil, err
		}
	}
	entry, err := l.Store.GetJournalEntry(ctx, inv.JournalEntryID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, domain.NotFound("Jurnal tagihan tidak ditemukan")
	}
	lines := reverseProRata(entry.Lines, amount, inv.TotalAmount)
	extras, err := l.Store.ListInvoicePostings(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	p := domain.Payment{ID: ids.New(), Kind: inv.Kind, PaymentDate: day, Method: methodCreditNote, Amount: amount, Reference: inv.InvoiceNo, Notes: reason, Status: domain.StatusPosted, PartyName: inv.PartyName, CreatedBy: userID,
		Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: amount}}}
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if p.PaymentNo, err = l.nextNo(ctx, tx, "CN"); err != nil {
			return err
		}
		if err := tx.InsertPayment(ctx, p); err != nil {
			return err
		}
		e, err := createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: day, Description: "Credit note " + p.PaymentNo + " atas " + inv.InvoiceNo + ": " + reason, Lines: lines, Post: true,
			EntryType: domain.EntryAuto, SourceModule: kinds[inv.Kind].module, SourceEventCode: "CREDIT_NOTE", SourceDocumentType: "credit_note", SourceDocumentID: p.ID})
		if err != nil {
			return err
		}
		if err := tx.SetPaymentJournal(ctx, p.ID, e.ID); err != nil {
			return err
		}
		return creditNoteMoves(ctx, tx, *inv, extras, p, amount)
	})
	if err != nil {
		return nil, err
	}
	return l.Store.GetInvoice(ctx, inv.ID)
}

// reverseProRata flips each line at amount/total and puts rounding on the last line.
func reverseProRata(lines []domain.JournalLine, amount, total domain.Money) []domain.Line {
	out := make([]domain.Line, 0, len(lines))
	var debit, credit domain.Money
	for i, l := range lines {
		side := domain.Debit
		if l.Side == domain.Debit {
			side = domain.Credit
		}
		part := domain.MulDiv(l.Amount, amount, total)
		if part == 0 {
			continue
		}
		if side == domain.Debit {
			debit += part
		} else {
			credit += part
		}
		out = append(out, domain.Line{AccountID: l.AccountID, Side: side, Amount: part, Memo: l.Memo, SortOrder: i})
	}
	if n := len(out); n > 0 && debit != credit {
		diff := debit - credit
		if out[n-1].Side == domain.Debit {
			out[n-1].Amount -= diff
		} else {
			out[n-1].Amount += diff
		}
	}
	return out
}

// creditNoteMoves takes the credited share out of the tax ledger. Cash basis VAT
// reaches the ledger on payment, so an unpaid share has nothing to reverse.
func creditNoteMoves(ctx context.Context, tx domain.Store, inv domain.Invoice, extras []domain.InvoicePosting, p domain.Payment, amount domain.Money) error {
	tagged := false
	for _, x := range extras {
		if x.TaxCode == "" {
			continue
		}
		if x.TaxKind == domain.TaxVAT {
			tagged = true
		}
		if x.CashBasis {
			continue
		}
		if err := tx.InsertTaxMove(ctx, domain.TaxMove{Kind: inv.Kind, SourceType: domain.MoveCreditNote, SourceID: p.ID, InvoiceID: inv.ID, TaxCode: x.TaxCode, TaxName: x.TaxName, TaxKind: x.TaxKind, TaxRate: x.TaxRate,
			ReportCode: x.ReportCode, Date: p.PaymentDate, Base: -domain.MulDiv(x.TaxBase, amount, inv.TotalAmount), Amount: -domain.MulDiv(x.Amount, amount, inv.TotalAmount)}); err != nil {
			return err
		}
	}
	if tagged || inv.TaxAmount <= 0 {
		return nil
	}
	return tx.InsertTaxMove(ctx, domain.TaxMove{Kind: inv.Kind, SourceType: domain.MoveCreditNote, SourceID: p.ID, InvoiceID: inv.ID, TaxKind: domain.TaxVAT, Date: p.PaymentDate,
		Base: -domain.MulDiv(inv.Subtotal, amount, inv.TotalAmount), Amount: -domain.MulDiv(inv.TaxAmount, amount, inv.TotalAmount)})
}

// recognizeCashTaxes moves cash basis VAT from the transition account to its final
// account in proportion to what the payment settled, and enters the tax ledger on
// the payment date. A void reverses it. Like the payment journal it runs after the
// payment commits, so a failure comes back as a note for Finance to correct.
func (l *Ledgers) recognizeCashTaxes(ctx context.Context, p domain.Payment, userID string, void bool) string {
	source, sign := domain.MoveCash, domain.Money(1)
	if void {
		source, sign = domain.MoveCashVoid, -1
	}
	notes := ""
	for _, a := range p.Allocations {
		extras, err := l.Store.ListInvoicePostings(ctx, a.InvoiceID)
		if err != nil {
			return "PPN berbasis kas gagal dibaca: " + apperr.As(err).Message
		}
		for _, x := range extras {
			if !x.CashBasis {
				continue
			}
			inv, err := l.Store.GetInvoice(ctx, a.InvoiceID)
			if err != nil || inv == nil {
				return "PPN berbasis kas gagal: tagihan tidak terbaca"
			}
			amount := domain.MulDiv(x.Amount, a.Amount, inv.TotalAmount)
			if amount == 0 {
				continue
			}
			toFinal, fromTransition := x.Side, domain.Debit
			if x.Side == domain.Debit {
				fromTransition = domain.Credit
			}
			if void {
				toFinal, fromTransition = fromTransition, toFinal
			}
			sourceID := p.ID + ":" + a.InvoiceID
			err = l.Store.WithTx(ctx, func(tx domain.Store) error {
				_, err := createEntry(ctx, tx, NewEntry{UserID: userID, EntryDate: p.PaymentDate, Description: "Pengakuan " + x.TaxName + " " + inv.InvoiceNo, Post: true, EntryType: domain.EntryAuto,
					SourceModule: kinds[inv.Kind].module, SourceEventCode: "TAX_" + source, SourceDocumentType: "tax_recognition", SourceDocumentID: sourceID, Lines: []domain.Line{
						{AccountID: x.AccountID, Side: fromTransition, Amount: amount, Memo: x.TaxName},
						{AccountID: x.FinalAccountID, Side: toFinal, Amount: amount, Memo: x.TaxName, SortOrder: 1},
					}})
				if err != nil {
					return err
				}
				return tx.InsertTaxMove(ctx, domain.TaxMove{Kind: inv.Kind, SourceType: source, SourceID: sourceID, InvoiceID: inv.ID, TaxCode: x.TaxCode, TaxName: x.TaxName, TaxKind: x.TaxKind, TaxRate: x.TaxRate,
					ReportCode: x.ReportCode, Date: p.PaymentDate, Base: sign * domain.MulDiv(x.TaxBase, a.Amount, inv.TotalAmount), Amount: sign * amount})
			})
			if err != nil {
				notes = "PPN berbasis kas belum terjurnal: " + apperr.As(err).Message
			}
		}
	}
	return notes
}

// ─── Fiscal positions and party profiles ──────

func (s *TaxSetup) ListFiscalPositions(ctx context.Context) ([]domain.FiscalPosition, error) {
	return s.Store.ListFiscalPositions(ctx)
}

func (s *TaxSetup) SaveFiscalPosition(ctx context.Context, userID string, f domain.FiscalPosition) (*domain.FiscalPosition, error) {
	f.Name, f.Notes = trim(f.Name), trim(f.Notes)
	if err := validate.New().Required("name", f.Name, "Nama wajib diisi").Err(); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, r := range f.Rules {
		if r.FromTaxID == "" || seen[r.FromTaxID] {
			return nil, apperr.Field("rules", "Setiap aturan perlu kode pajak asal yang berbeda")
		}
		seen[r.FromTaxID] = true
		for _, id := range []string{r.FromTaxID, r.ToTaxID} {
			if id == "" {
				continue
			}
			if t, err := s.Store.GetTaxCode(ctx, id); err != nil {
				return nil, err
			} else if t == nil {
				return nil, apperr.Field("rules", "Kode pajak pada aturan tidak ditemukan")
			}
		}
	}
	if f.ID == "" {
		f.ID = ids.New()
	}
	f.UpdatedBy = userID
	if err := s.Store.SaveFiscalPosition(ctx, f); err != nil {
		var unique *domain.ErrUnique
		if errors.As(err, &unique) {
			return nil, apperr.Field("name", "Nama sudah dipakai")
		}
		return nil, err
	}
	return s.Store.GetFiscalPosition(ctx, f.ID)
}

func (s *TaxSetup) ListPartyProfiles(ctx context.Context) ([]domain.PartyProfile, error) {
	return s.Store.ListPartyProfiles(ctx)
}

func (s *TaxSetup) SavePartyProfile(ctx context.Context, userID string, p domain.PartyProfile) (*domain.PartyProfile, error) {
	p.Name, p.NPWP, p.Address, p.Notes = trim(p.Name), trim(p.NPWP), trim(p.Address), trim(p.Notes)
	v := validate.New().Required("name", p.Name, "Nama pihak wajib diisi")
	if p.Side != domain.SideB2B && p.Side != domain.SideSupplier {
		v.Add("side", "Pihak tidak valid")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	if p.ID == "" {
		p.ID = ids.New()
	}
	p.UpdatedBy = userID
	if err := s.Store.SavePartyProfile(ctx, p); err != nil {
		var unique *domain.ErrUnique
		if errors.As(err, &unique) {
			return nil, apperr.Field("name", "Profil pihak ini sudah ada")
		}
		return nil, err
	}
	return s.Store.GetPartyProfile(ctx, p.Side, p.Name)
}

// Export kinds.
const (
	ExportEFaktur     = "efaktur"
	ExportBupotKeluar = "bupot-keluar"
	ExportBupotMasuk  = "bupot-masuk"
)

// Export writes a period's taxes as CSV, for Finance to carry into the DJP
// template and converter. It is a working file: nothing here reaches DJP.
func (s *TaxSetup) Export(ctx context.Context, kind, from, to string) (string, []byte, error) {
	from, to, err := NewReports(s.Deps).period(from, to)
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("\xef\xbb\xbf") // Excel reads the file as UTF-8.
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	switch kind {
	case ExportEFaktur:
		rows, err := s.Store.TaxExportRows(ctx, domain.KindAR, domain.TaxVAT, from, to)
		if err != nil {
			return "", nil, err
		}
		_ = w.Write([]string{"Tanggal faktur", "Kode transaksi", "No. invoice", "Referensi", "NPWP pembeli", "Nama pembeli", "Alamat pembeli", "Uraian", "Harga jual", "DPP", "Tarif", "PPN", "Kode pajak"})
		for _, r := range rows {
			_ = w.Write([]string{r.Date, r.ReportCode, r.InvoiceNo, r.Reference, r.NPWP, r.Party, r.Address, r.Description, r.Subtotal.String(), r.Base.String(), r.TaxRate.String(), r.Amount.String(), r.TaxCode})
		}
	case ExportBupotKeluar, ExportBupotMasuk:
		invoiceKind := domain.KindAP
		if kind == ExportBupotMasuk {
			invoiceKind = domain.KindAR
		}
		rows, err := s.Store.TaxExportRows(ctx, invoiceKind, domain.TaxWithholding, from, to)
		if err != nil {
			return "", nil, err
		}
		_ = w.Write([]string{"Tanggal", "Kode objek pajak", "No. invoice", "Referensi", "NPWP", "Nama", "Alamat", "Uraian", "Dasar pemotongan", "Tarif", "PPh dipotong", "Kode pajak"})
		for _, r := range rows {
			_ = w.Write([]string{r.Date, r.ReportCode, r.InvoiceNo, r.Reference, r.NPWP, r.Party, r.Address, r.Description, r.Base.String(), r.TaxRate.String(), r.Amount.String(), r.TaxCode})
		}
	default:
		return "", nil, apperr.Field("kind", "Jenis ekspor tidak valid")
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("%s-%s-%s.csv", kind, from, to), buf.Bytes(), nil
}

// DeleteTaxCode removes a code nothing refers to. One that has been used stays
// for the records and can be deactivated.
func (s *TaxSetup) DeleteTaxCode(ctx context.Context, id string) error {
	t, err := s.Store.GetTaxCode(ctx, id)
	if err != nil {
		return err
	}
	if t == nil {
		return nil
	}
	used, err := s.Store.TaxCodeInUse(ctx, id, t.Code)
	if err != nil {
		return err
	}
	if used {
		return domain.Conflict("Kode pajak sudah dipakai. Nonaktifkan saja.")
	}
	return s.Store.DeleteTaxCode(ctx, id)
}

func (s *TaxSetup) DeleteChargeType(ctx context.Context, id string) error {
	used, err := s.Store.ChargeTypeInUse(ctx, id)
	if err != nil {
		return err
	}
	if used {
		return domain.Conflict("Biaya sudah dipakai dokumen. Nonaktifkan saja.")
	}
	return s.Store.DeleteChargeType(ctx, id)
}

func (s *TaxSetup) DeleteFiscalPosition(ctx context.Context, id string) error {
	used, err := s.Store.FiscalPositionInUse(ctx, id)
	if err != nil {
		return err
	}
	if used {
		return domain.Conflict("Fiscal position masih dipakai profil pihak. Nonaktifkan saja.")
	}
	return s.Store.DeleteFiscalPosition(ctx, id)
}

func (s *TaxSetup) DeletePartyProfile(ctx context.Context, id string) error {
	return s.Store.DeletePartyProfile(ctx, id)
}
