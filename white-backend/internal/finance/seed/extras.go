package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

// extras fills the states the month generator never produces, so every
// list and tab has rows: draft journals, an accrual with its reversal, a
// voided receipt (reversing journal), draft AR and AP invoices, a partly
// paid bill, movements on every cash and bank account, and a bonus payroll
// dated in a closed period whose journals stay DRAFT in the posting log.
// The accrual journal marks it done.
func (s *demo) extras(ctx context.Context) error {
	const marker = "Akrual bonus kinerja tim kreatif"
	done, _, err := s.d.Store.ListJournalEntries(ctx, domain.JournalFilter{Search: marker, Limit: 1})
	if err != nil || len(done) > 0 {
		return err
	}
	ym := s.today.Format("2006-01")
	day := func(d int) string {
		t := time.Date(s.today.Year(), s.today.Month(), d, 0, 0, 0, 0, time.UTC)
		if t.After(s.today) {
			t = s.today
		}
		return t.Format(time.DateOnly)
	}
	line := func(code, side string, rupiah int64, memo string) domain.Line {
		return domain.Line{AccountID: s.id(code), Side: side, Amount: domain.Rupiah(rupiah), Memo: memo}
	}

	for _, e := range []app.EntryInput{
		{EntryDate: day(20), Description: "Akrual listrik dan internet " + monthNames[s.today.Month()-1], Lines: []domain.Line{
			line("6301002", domain.Debit, 4_200_000, "Estimasi tagihan"), line("2101001", domain.Credit, 4_200_000, "Estimasi tagihan")}},
		{EntryDate: day(22), Description: "Reklasifikasi biaya produksi konten klien", Lines: []domain.Line{
			line("5101002", domain.Debit, 2_750_000, "Reklasifikasi"), line("6301004", domain.Credit, 2_750_000, "Reklasifikasi")}},
		{EntryDate: day(24), Description: "Koreksi uang muka klien PT Arunika Properti", IsRecon: true, Lines: []domain.Line{
			line("1103001", domain.Debit, 5_000_000, "Koreksi"), line("2105001", domain.Credit, 5_000_000, "Koreksi")}},
	} {
		if _, err := s.journals.Create(ctx, "", e); err != nil {
			return err
		}
	}
	accrual := []domain.Line{line("6101001", domain.Debit, 18_000_000, "Bonus kuartal"), line("2102001", domain.Credit, 18_000_000, "Bonus kuartal")}
	reversal := []domain.Line{line("2102001", domain.Debit, 18_000_000, "Batal akrual"), line("6101001", domain.Credit, 18_000_000, "Batal akrual")}
	if _, err := s.journals.Create(ctx, "", app.EntryInput{EntryDate: day(6), Description: marker, Post: true, Lines: accrual}); err != nil {
		return err
	}
	if _, err := s.journals.Create(ctx, "", app.EntryInput{EntryDate: day(9), Description: "Pembalik: " + marker + " (bonus dibayar lewat payroll)", Post: true, Lines: reversal}); err != nil {
		return err
	}

	// Every cash and bank account moves.
	bca, mandiri, kasBesar := s.id("1102001"), s.id("1102002"), s.id("1101002")
	for _, t := range []app.MovementInput{
		{EntryDate: day(4), Amount: domain.Rupiah(150_000_000), CashAccountID: bca, OffsetAccountID: mandiri, Description: "Pemindahan dana cadangan ke Mandiri"},
		{EntryDate: day(7), Amount: domain.Rupiah(12_000_000), CashAccountID: mandiri, OffsetAccountID: kasBesar, Description: "Tarik tunai kas besar event klien"},
	} {
		if _, err := s.cash.CreateTransfer(ctx, "", t); err != nil {
			return err
		}
	}
	for _, mv := range []app.MovementInput{
		{Kind: app.KindCashOut, EntryDate: day(8), Amount: domain.Rupiah(4_650_000), CashAccountID: kasBesar, OffsetAccountID: s.id("6302002"), Description: "Konsumsi event peluncuran Batik Prameswari"},
		{Kind: app.KindCashOut, EntryDate: day(9), Amount: domain.Rupiah(3_100_000), CashAccountID: kasBesar, OffsetAccountID: s.id("6302001"), Description: "Sewa kendaraan tim shooting"},
		{Kind: app.KindCashIn, EntryDate: day(21), Amount: domain.Rupiah(625_000), CashAccountID: mandiri, OffsetAccountID: s.id("8201001"), Description: "Bunga deposito Mandiri " + ym},
		{Kind: app.KindCashOut, EntryDate: day(21), Amount: domain.Rupiah(55_000), CashAccountID: mandiri, OffsetAccountID: s.id("8101001"), Description: "Biaya administrasi Mandiri " + ym},
	} {
		if _, err := s.cash.CreateMovement(ctx, "", mv); err != nil {
			return err
		}
	}

	// Invoices in every state: drafts, a partly paid bill, a voided receipt.
	if _, err := s.ledgers.CreateInvoice(ctx, domain.KindAR, "", app.InvoiceInput{InvoiceDate: day(23), DueDate: day(23)[:8] + "28",
		PartyName: "PT Nusantara Retail Group", Reference: "PRJ/" + ym + "/SEO", AccountID: s.id("4101003"),
		Subtotal: domain.Rupiah(36_000_000), TaxAmount: domain.Rupiah(3_960_000), Description: "Proyek audit SEO (draf)"}); err != nil {
		return err
	}
	if _, err := s.ledgers.CreateInvoice(ctx, domain.KindAP, "", app.InvoiceInput{InvoiceDate: day(23), PartyName: "CV Cahaya Lighting Rental",
		Reference: "BILL/" + ym + "/LGT", AccountID: s.id("5101002"), Subtotal: domain.Rupiah(6_500_000), Description: "Sewa lighting studio (draf)"}); err != nil {
		return err
	}
	bill, err := s.ledgers.CreateInvoice(ctx, domain.KindAP, "", app.InvoiceInput{InvoiceDate: day(2), DueDate: day(2)[:8] + "30",
		PartyName: "PT Kamera Pro Indonesia", Reference: "BILL/" + ym + "/CAM", AccountID: s.id("5101002"),
		Subtotal: domain.Rupiah(24_000_000), TaxAmount: domain.Rupiah(2_640_000), Description: "Sewa peralatan kamera kuartal IV"})
	if err != nil {
		return err
	}
	if _, _, err := s.ledgers.PostInvoice(ctx, domain.KindAP, "", bill.ID); err != nil {
		return err
	}
	if _, err := s.ledgers.CreatePayment(ctx, domain.KindAP, "", app.PaymentInput{PaymentDate: day(15), Method: "bank_transfer", CashAccountID: mandiri,
		Reference: "Termin 1", Allocations: []domain.Allocation{{InvoiceID: bill.ID, Amount: domain.Rupiah(10_000_000)}}}); err != nil {
		return err
	}
	inv, err := s.ledgers.CreateInvoice(ctx, domain.KindAR, "", app.InvoiceInput{InvoiceDate: day(5), DueDate: day(5)[:8] + "25",
		PartyName: "PT EduPintar Digital", Reference: "PRJ/" + ym + "/WEB", AccountID: s.id("4101005"),
		Subtotal: domain.Rupiah(18_000_000), TaxAmount: domain.Rupiah(1_980_000), Description: "Desain ulang landing page"})
	if err != nil {
		return err
	}
	if _, _, err := s.ledgers.PostInvoice(ctx, domain.KindAR, "", inv.ID); err != nil {
		return err
	}
	receipt, err := s.ledgers.CreatePayment(ctx, domain.KindAR, "", app.PaymentInput{PaymentDate: day(12), Method: "bank_transfer", CashAccountID: bca,
		Reference: "Transfer salah nominal", Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: domain.Rupiah(19_980_000)}}})
	if err != nil {
		return err
	}
	if _, err := s.ledgers.VoidPayment(ctx, domain.KindAR, "", receipt.Payment.ID, "Transfer dikembalikan bank, nominal salah"); err != nil {
		return err
	}

	// A bonus run dated last month (a closed period): its journals stay
	// DRAFT in the posting log until finance reopens the period and retries.
	last := time.Date(s.today.Year(), s.today.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -5)
	_, err = s.auto.PayrollPaid(ctx, domain.PayrollPaid{
		RunID: fmt.Sprintf("seed-bonus-%s", last.Format("2006-01")), PeriodYear: last.Year(), PeriodMonth: int(last.Month()),
		Label: "Bonus proyek " + monthNames[last.Month()-1], PaidAt: last.Add(3 * time.Hour), EmployeeCount: 6,
		TotalGross: domain.Rupiah(30_000_000), TotalNet: domain.Rupiah(28_500_000), TotalPph21: domain.Rupiah(1_500_000),
	})
	return err
}
