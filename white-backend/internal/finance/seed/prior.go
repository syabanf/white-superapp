package seed

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

// The prior fiscal year gives the dashboard a year of history to compare
// against. WHITE was smaller then: six clients growing to nine, retainers
// at 85% and a 20-person team. Every client pays the next month (December
// before year end, except one slow payer), vendor bills, taxes and BPJS
// are settled within the month, laptops are bought in March and a dividend
// is paid in December. The opening balance is solved backwards so the year
// closes exactly at the current year's posted opening balance, then every
// period is closed.

// openingLines is the current year's beginning balance.
func (s *demo) openingLines() []domain.Line {
	line := func(code, side string, rupiah int64) domain.Line {
		return domain.Line{AccountID: s.id(code), Side: side, Amount: domain.Rupiah(rupiah)}
	}
	return []domain.Line{
		line("1101001", domain.Debit, 5_000_000), line("1102001", domain.Debit, 420_000_000), line("1102002", domain.Debit, 135_000_000),
		line("1103001", domain.Debit, 85_000_000), line("1201001", domain.Debit, 180_000_000), line("1201002", domain.Credit, 40_000_000),
		line("2101001", domain.Credit, 30_000_000), line("3101001", domain.Credit, 500_000_000), line("3201001", domain.Credit, 255_000_000),
	}
}

func (s *demo) priorYear(ctx context.Context) error {
	y := s.today.Year() - 1
	code := fmt.Sprintf("FY%d", y)
	years, err := s.fiscal.ListYears(ctx, code, nil)
	if err != nil {
		return err
	}
	var year *domain.FiscalYear
	for i := range years {
		if years[i].Code == code {
			year = &years[i]
		}
	}
	if year == nil {
		if year, err = s.fiscal.CreateYear(ctx, "", app.YearInput{Code: code, Name: fmt.Sprintf("Tahun Buku %d", y),
			StartDate: fmt.Sprintf("%d-01-01", y), EndDate: fmt.Sprintf("%d-12-31", y), IsActive: true}); err != nil {
			return err
		}
	}
	if existing, err := s.d.Store.OpeningEntry(ctx, year.ID); err != nil || existing != nil {
		return err
	}
	for _, p := range year.Periods {
		if err := s.priorMonth(ctx, p); err != nil {
			return fmt.Errorf("%s: %w", p.Name, err)
		}
	}
	if err := s.priorOpening(ctx, year); err != nil {
		return err
	}
	year, err = s.fiscal.GetYear(ctx, year.ID)
	if err != nil {
		return err
	}
	for _, p := range year.Periods {
		if p.Status == domain.PeriodOpen {
			if _, err := s.fiscal.ClosePeriod(ctx, p.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// priorMonth seeds one prior-year period; its depreciation journal, written
// last, marks it done.
func (s *demo) priorMonth(ctx context.Context, p domain.FiscalPeriod) error {
	start, _ := time.Parse(time.DateOnly, p.StartDate)
	ym := p.StartDate[:7]
	marker := "Penyusutan peralatan " + ym
	done, _, err := s.d.Store.ListJournalEntries(ctx, domain.JournalFilter{Search: marker, Limit: 1})
	if err != nil || len(done) > 0 {
		return err
	}
	if p.Status != domain.PeriodOpen {
		if _, err := s.fiscal.OpenPeriod(ctx, p.ID, true); err != nil {
			return err
		}
	}
	m := int(start.Month())
	day := func(d int) string {
		return time.Date(start.Year(), start.Month(), d, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
	}
	plus := func(date string, days int) string {
		t, _ := time.Parse(time.DateOnly, date)
		return t.AddDate(0, 0, days).Format(time.DateOnly)
	}
	// Clients 7, 8 and 9 sign in April, July and October.
	active := min(6+max(m-1, 0)/3, len(clients))
	invoiceID := func(month string, i int) string { return fmt.Sprintf("seed-si-%s-%02d", month, i+1) }
	var ppnOut, ppnIn domain.Money
	for i := 0; i < active; i++ {
		c := clients[i]
		subtotal := domain.Rupiah((c.retainer*85/100)/500_000*500_000 + int64((m+i)%3)*2_000_000)
		tax := subtotal * 11 / 100
		ppnOut += tax
		issue := day(3)
		if _, err := s.auto.SalesInvoiceIssued(ctx, domain.SalesInvoiceIssued{
			InvoiceID: invoiceID(ym, i), InvoiceNumber: fmt.Sprintf("INV/%s/%03d", ym, i+1), CustomerName: c.name,
			IssueDate: issue, DueDate: plus(issue, 30), Subtotal: subtotal, TaxAmount: tax, TotalAmount: subtotal + tax,
			Description: "Retainer jasa digital marketing " + monthNames[m-1] + " " + ym[:4],
		}); err != nil {
			return err
		}
	}
	pay := func(month string, i int, date string) error {
		inv, err := s.d.Store.GetInvoiceBySalesID(ctx, invoiceID(month, i))
		if err != nil || inv == nil {
			return err
		}
		_, err = s.auto.SalesPaymentRecorded(ctx, domain.SalesPaymentRecorded{
			PaymentID: fmt.Sprintf("seed-sp-%s-%02d", month, i+1), InvoiceID: inv.SalesInvoiceID, Amount: inv.TotalAmount,
			PaidOn: date, Method: "transfer", Reference: fmt.Sprintf("BCA %s%02d", month, i+1),
		})
		return err
	}
	if m > 1 {
		prev := start.AddDate(0, -1, 0).Format("2006-01")
		for i := 0; i < min(6+(m-2)/3, len(clients)); i++ {
			if err := pay(prev, i, day(8+i)); err != nil {
				return err
			}
		}
	}
	if m == 12 {
		for i := 0; i < active; i++ {
			if i != 6 { // a slow payer keeps a year-old invoice in the >90 bucket
				if err := pay(ym, i, day(18+i)); err != nil {
					return err
				}
			}
		}
	}

	bills := []bill{
		{"PT Graha Perkantoran Sudirman", "6301001", 30_000_000, false, always},
		{"PT Telkom Indonesia", "6301002", 4_000_000, true, always},
		{"PT Solusi Lisensi Software", "6301003", 10_000_000, true, always},
		{"Meta Platforms Ireland", "5101001", 40_000_000 + int64(m)*2_000_000, false, always},
		{"CV Kreatif Lepas Studio", "5101002", 12_000_000, false, always},
		{"Google Asia Pacific", "6201001", 6_000_000, false, always},
	}
	for i, b := range bills {
		subtotal := domain.Rupiah(b.amount)
		var tax domain.Money
		if b.ppn {
			tax = subtotal * 11 / 100
			ppnIn += tax
		}
		inv, err := s.ledgers.CreateInvoice(ctx, domain.KindAP, "", app.InvoiceInput{
			InvoiceDate: day(5), DueDate: plus(day(5), 30), PartyName: b.vendor, Reference: fmt.Sprintf("BILL/%s/%02d", ym, i+1),
			AccountID: s.id(b.account), Subtotal: subtotal, TaxAmount: tax, Description: "Tagihan " + monthNames[m-1],
		})
		if err != nil {
			return err
		}
		if _, _, err := s.ledgers.PostInvoice(ctx, domain.KindAP, "", inv.ID); err != nil {
			return err
		}
		if _, err := s.ledgers.CreatePayment(ctx, domain.KindAP, "", app.PaymentInput{
			PaymentDate: day(12), Method: "bank_transfer", CashAccountID: s.id("1102001"),
			Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: inv.TotalAmount}},
		}); err != nil {
			return err
		}
	}

	if _, err := s.auto.PayrollPaid(ctx, domain.PayrollPaid{
		RunID: "seed-payroll-" + ym, PeriodYear: start.Year(), PeriodMonth: m, Label: "Gaji " + monthNames[m-1] + " " + ym[:4],
		PaidAt: time.Date(start.Year(), start.Month(), 25, 3, 0, 0, 0, time.UTC), EmployeeCount: 20, TotalGross: domain.Rupiah(168_000_000),
		TotalNet: domain.Rupiah(152_480_000), TotalPph21: domain.Rupiah(7_600_000), TotalLoanDeduction: domain.Rupiah(1_200_000),
		TotalBpjsTkEmployee: domain.Rupiah(5_040_000), TotalBpjsKesEmployee: domain.Rupiah(1_680_000),
		TotalBpjsTkEmployer: domain.Rupiah(10_080_000), TotalBpjsKesEmployer: domain.Rupiah(6_720_000),
	}); err != nil {
		return err
	}

	bca := s.id("1102001")
	movements := []app.MovementInput{
		{Kind: app.KindCashOut, EntryDate: day(28), Amount: domain.Rupiah(225_000), CashAccountID: bca, OffsetAccountID: s.id("8101001"), Description: "Biaya administrasi bank " + ym},
		{Kind: app.KindCashIn, EntryDate: day(28), Amount: domain.Rupiah(950_000), CashAccountID: bca, OffsetAccountID: s.id("8201001"), Description: "Bunga jasa giro " + ym},
		{Kind: app.KindCashOut, EntryDate: day(15), Amount: domain.Rupiah(3_000_000), CashAccountID: s.id("1101001"), OffsetAccountID: s.id("6301004"), Description: "Belanja ATK dan konsumsi kantor"},
		{Kind: app.KindCashOut, EntryDate: day(27), Amount: domain.Rupiah(7_600_000), CashAccountID: bca, OffsetAccountID: s.id("2103001"), Description: "Setor PPh 21 masa " + monthNames[m-1]},
		{Kind: app.KindCashOut, EntryDate: day(27), Amount: domain.Rupiah(23_520_000), CashAccountID: bca, OffsetAccountID: s.id("2104001"), Description: "Bayar iuran BPJS " + monthNames[m-1]},
	}
	if m == 3 {
		movements = append(movements, app.MovementInput{Kind: app.KindCashOut, EntryDate: day(18), Amount: domain.Rupiah(45_000_000), CashAccountID: bca,
			OffsetAccountID: s.id("1201001"), Description: "Pembelian 6 laptop tim kreatif"})
	}
	for _, mv := range movements {
		if _, err := s.cash.CreateMovement(ctx, "", mv); err != nil {
			return err
		}
	}
	if _, err := s.cash.CreateTransfer(ctx, "", app.MovementInput{EntryDate: day(2), Amount: domain.Rupiah(3_000_000), CashAccountID: bca, OffsetAccountID: s.id("1101001"), Description: "Isi ulang kas kecil"}); err != nil {
		return err
	}
	// PPN for the month: output tax less input tax, paid from BCA.
	if _, err := s.journals.Create(ctx, "", app.EntryInput{EntryDate: day(28), Description: "Setor PPN masa " + monthNames[m-1] + " " + ym[:4], Post: true, Lines: []domain.Line{
		{AccountID: s.id("2103003"), Side: domain.Debit, Amount: ppnOut, Memo: "PPN keluaran"},
		{AccountID: s.id("1104001"), Side: domain.Credit, Amount: ppnIn, Memo: "PPN masukan"},
		{AccountID: bca, Side: domain.Credit, Amount: ppnOut - ppnIn, Memo: "Kurang bayar PPN"},
	}}); err != nil {
		return err
	}
	if err := s.claimMonth(ctx, start); err != nil {
		return err
	}
	_, err = s.journals.Create(ctx, "", app.EntryInput{EntryDate: day(28), Description: marker, Post: true, Lines: []domain.Line{
		{AccountID: s.id("6303001"), Side: domain.Debit, Amount: domain.Rupiah(2_500_000), Memo: "Penyusutan bulanan"},
		{AccountID: s.id("1201002"), Side: domain.Credit, Amount: domain.Rupiah(2_500_000), Memo: "Penyusutan bulanan"},
	}})
	return err
}

// priorOpening pays the December dividend, then posts the opening balance
// that makes the year close at the current year's beginning balance, with
// the year's profit folded into retained earnings.
func (s *demo) priorOpening(ctx context.Context, year *domain.FiscalYear) error {
	totals, err := s.d.Store.YearAccountTotals(ctx, year.ID)
	if err != nil {
		return err
	}
	var profit domain.Money
	for _, t := range totals {
		if domain.IsProfitLossType(t.TypeCode) {
			profit += t.Credit - t.Debit
		}
	}
	dividend := max(domain.Rupiah(75_000_000), profit-domain.Rupiah(205_000_000))
	if _, err := s.journals.Create(ctx, "", app.EntryInput{EntryDate: year.EndDate[:8] + "20", Description: "Pembagian dividen " + year.EndDate[:4], Post: true, Lines: []domain.Line{
		{AccountID: s.id("3201001"), Side: domain.Debit, Amount: dividend, Memo: "Dividen tunai"},
		{AccountID: s.id("1102001"), Side: domain.Credit, Amount: dividend, Memo: "Dividen tunai"},
	}}); err != nil {
		return err
	}
	if totals, err = s.d.Store.YearAccountTotals(ctx, year.ID); err != nil {
		return err
	}
	net := map[string]domain.Money{}
	add := func(accountID, side string, amount domain.Money) {
		if side == domain.Debit {
			net[accountID] += amount
		} else {
			net[accountID] -= amount
		}
	}
	ob, err := s.currentOpening(ctx)
	if err != nil {
		return err
	}
	if ob != nil {
		for _, l := range ob.Lines {
			add(l.AccountID, l.Side, l.Amount)
		}
	} else {
		for _, l := range s.openingLines() {
			add(l.AccountID, l.Side, l.Amount)
		}
	}
	for _, t := range totals {
		if !domain.IsProfitLossType(t.TypeCode) {
			net[t.ID] -= t.Debit - t.Credit
		}
	}
	net[s.id("3201001")] += profit
	var lines []domain.Line
	for _, code := range slices.Sorted(maps.Keys(s.acc)) {
		id := s.id(code)
		switch v := net[id]; {
		case v > 0:
			lines = append(lines, domain.Line{AccountID: id, Side: domain.Debit, Amount: v})
		case v < 0:
			lines = append(lines, domain.Line{AccountID: id, Side: domain.Credit, Amount: -v})
		}
	}
	if net[s.id("1102001")] <= 0 {
		return fmt.Errorf("prior year opening cash would be negative")
	}
	_, err = s.fiscal.SaveBeginningBalance(ctx, "", year.ID, lines, true)
	return err
}

// currentOpening is the current year's posted beginning balance, if the
// year exists already.
func (s *demo) currentOpening(ctx context.Context) (*domain.JournalEntry, error) {
	code := fmt.Sprintf("FY%d", s.today.Year())
	years, err := s.fiscal.ListYears(ctx, code, nil)
	if err != nil {
		return nil, err
	}
	for _, y := range years {
		if y.Code == code {
			return s.d.Store.OpeningEntry(ctx, y.ID)
		}
	}
	return nil, nil
}
