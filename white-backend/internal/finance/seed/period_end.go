package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// periodEnd fills the settlement and closing screens: a receipt with a bank fee,
// withheld tax and excess cash, a refund, a credit note on a paid invoice, a
// write-off, recognition schedules, a VAT offset and tax remittances. It posts in
// the latest open period. The revenue schedule marks it done.
func (s *demo) periodEnd(ctx context.Context) error {
	const marker = "Retainer tahunan PT Arunika Properti"
	done, err := s.d.Store.ListDeferrals(ctx, false)
	if err != nil {
		return err
	}
	for _, d := range done {
		if d.Name == marker {
			return nil
		}
	}
	year, err := s.fiscalYear(ctx)
	if err != nil {
		return err
	}
	var open *domain.FiscalPeriod
	for i := range year.Periods {
		p := &year.Periods[i]
		start, err := time.Parse(time.DateOnly, p.StartDate)
		if err != nil {
			return err
		}
		if p.Status == domain.PeriodOpen && !start.After(s.today) {
			open = p
		}
	}
	if open == nil {
		return nil
	}
	start, err := time.Parse(time.DateOnly, open.StartDate)
	if err != nil {
		return err
	}
	day := func(n int) string {
		t := start.AddDate(0, 0, n-1)
		if t.After(s.today) {
			t = s.today
		}
		return t.Format(time.DateOnly)
	}
	bca, liability := s.id("1102001"), s.id("2105001")
	paidInvoice := func(party, reference, description string, rupiah int64) (*domain.Invoice, error) {
		inv, err := s.ledgers.CreateInvoice(ctx, domain.KindAR, "", app.InvoiceInput{InvoiceDate: day(2), PartyName: party, Reference: reference, AccountID: s.id("4101003"), Subtotal: domain.Rupiah(rupiah), Description: description})
		if err != nil {
			return nil, err
		}
		inv, _, err = s.ledgers.PostInvoice(ctx, domain.KindAR, "", inv.ID)
		return inv, err
	}

	// A client transfer short of a fee and withheld tax, plus money above the invoice.
	inv, err := paidInvoice("PT Arunika Properti", "PRJ/SETTLE/01", "Retainer SEO kuartal IV", 20_000_000)
	if err != nil {
		return err
	}
	adjustments := []app.AdjustmentInput{{Kind: domain.AdjustFee, AccountID: s.id("8101001"), Amount: domain.Rupiah(6_500)}}
	if pph, err := s.d.Store.GetTaxCodeByCode(ctx, "PPH23-2"); err != nil {
		return err
	} else if pph != nil && pph.AccountSaleID != "" {
		adjustments = append(adjustments, app.AdjustmentInput{Kind: domain.AdjustWithholding, TaxID: pph.ID, Amount: domain.Rupiah(400_000)})
	}
	receipt, err := s.ledgers.CreatePayment(ctx, domain.KindAR, "", app.PaymentInput{PaymentDate: day(5), Method: "transfer", CashAccountID: bca, Reference: "BCA 88120", Adjustments: adjustments,
		Excess: domain.Rupiah(2_500_000), ExcessAccountID: liability, Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: inv.TotalAmount}}})
	if err != nil {
		return fmt.Errorf("penerimaan: %w", err)
	}
	advance, err := s.d.Store.AdvanceBySourcePayment(ctx, receipt.Payment.ID)
	if err != nil {
		return err
	}
	if _, err := s.ledgers.RefundAdvance(ctx, "", advance.ID, app.RefundInput{Date: day(8), CashAccountID: bca, Reason: "Klien minta sisa dikembalikan", Amount: domain.Rupiah(1_000_000)}); err != nil {
		return fmt.Errorf("pengembalian: %w", err)
	}

	// A return after the invoice was paid leaves the client a balance.
	returned, err := paidInvoice("PT Batik Prameswari", "PRJ/SETTLE/02", "Produksi video katalog", 15_000_000)
	if err != nil {
		return err
	}
	if _, err := s.ledgers.CreatePayment(ctx, domain.KindAR, "", app.PaymentInput{PaymentDate: day(6), Method: "transfer", CashAccountID: bca, Allocations: []domain.Allocation{{InvoiceID: returned.ID, Amount: returned.TotalAmount}}}); err != nil {
		return err
	}
	if _, err := s.ledgers.CreateCreditNote(ctx, "", returned.ID, app.CreditNoteInput{Date: day(10), Amount: domain.Rupiah(3_000_000), Reason: "Satu video dibatalkan klien", AdvanceAccountID: liability}); err != nil {
		return fmt.Errorf("credit note: %w", err)
	}

	// A client that will not pay.
	lost, err := paidInvoice("CV Warung Kopi Senja", "PRJ/SETTLE/03", "Kelola media sosial", 4_500_000)
	if err != nil {
		return err
	}
	if _, err := s.ledgers.WriteOffInvoice(ctx, "", lost.ID, app.WriteOffInput{Date: day(12), ExpenseAccountID: s.id("8102001"), Reason: "Usaha klien tutup dan tidak bisa ditagih"}); err != nil {
		return fmt.Errorf("write-off: %w", err)
	}

	// Money taken up front, then recognized month by month.
	for _, e := range []app.EntryInput{
		{EntryDate: day(1), Description: "Terima retainer tahunan PT Arunika Properti", Post: true, Lines: []domain.Line{
			{AccountID: bca, Side: domain.Debit, Amount: domain.Rupiah(36_000_000), Memo: "Retainer 12 bulan"}, {AccountID: liability, Side: domain.Credit, Amount: domain.Rupiah(36_000_000), Memo: "Retainer 12 bulan"}}},
		{EntryDate: day(1), Description: "Bayar sewa kantor 12 bulan", Post: true, Lines: []domain.Line{
			{AccountID: s.id("1105001"), Side: domain.Debit, Amount: domain.Rupiah(60_000_000), Memo: "Sewa 12 bulan"}, {AccountID: bca, Side: domain.Credit, Amount: domain.Rupiah(60_000_000), Memo: "Sewa 12 bulan"}}},
	} {
		if _, err := s.journals.Create(ctx, "", e); err != nil {
			return err
		}
	}
	for _, d := range []domain.Deferral{
		{Kind: domain.DeferRevenue, Name: marker, PartyName: "PT Arunika Properti", Reference: "KONTRAK/2026/ARP", DeferredAccountID: liability, TargetAccountID: s.id("4101003"), StartDate: day(1), Months: 12, Amount: domain.Rupiah(36_000_000)},
		{Kind: domain.DeferExpense, Name: "Sewa kantor Kemang", PartyName: "PT Graha Kemang", Reference: "SEWA/2026", DeferredAccountID: s.id("1105001"), TargetAccountID: s.id("6301001"), StartDate: day(1), Months: 12, Amount: domain.Rupiah(60_000_000)},
	} {
		if _, err := s.ledgers.CreateDeferral(ctx, "", d); err != nil {
			return fmt.Errorf("jadwal pengakuan: %w", err)
		}
	}

	// VAT is offset first, so what stays on the output account is what gets paid.
	period := open.StartDate[:7]
	// A database with no input VAT has nothing to offset, and that is fine here.
	if _, err := s.ledgers.CloseVAT(ctx, "", period); err != nil && apperr.As(err).Code != apperr.Conflict {
		return fmt.Errorf("kompensasi PPN: %w", err)
	}
	if _, err := s.ledgers.RemitTax(ctx, "", domain.TaxRemittance{Period: period, Date: day(15), AccountID: s.id("2103003"), CashAccountID: bca, Amount: domain.Rupiah(5_000_000), NTPN: "0A1B2C3D4E5F6G7H", Notes: "Setoran PPN masa " + period}); err != nil {
		return fmt.Errorf("setor pajak: %w", err)
	}
	mistaken, err := s.ledgers.RemitTax(ctx, "", domain.TaxRemittance{Period: period, Date: day(16), AccountID: s.id("2103003"), CashAccountID: bca, Amount: domain.Rupiah(1_000_000), NTPN: "SALAH-NTPN", Notes: "Salah catat"})
	if err != nil {
		return err
	}
	if _, err := s.ledgers.VoidTaxRemittance(ctx, "", mistaken.ID, "NTPN salah ketik"); err != nil {
		return err
	}
	if _, err := app.NewSchedules(s.d).RunDue(ctx); err != nil {
		return err
	}
	s.d.AuditSystem(ctx, "SeedDemo", "", map[string]string{"period": period}, "Data contoh selisih pembayaran dan tutup buku")
	return nil
}
