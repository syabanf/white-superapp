package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

type bill struct {
	vendor, account string
	amount          int64
	ppn             bool
	paid            func(month int, current bool) bool
}

func always(int, bool) bool           { return true }
func notCurrent(_ int, cur bool) bool { return !cur }

func (s *demo) bills(month int) []bill {
	return []bill{
		{"PT Graha Perkantoran Sudirman", "6301001", 35_000_000, false, always},
		{"PT Telkom Indonesia", "6301002", 4_500_000, true, always},
		{"PT Solusi Lisensi Software", "6301003", 12_500_000, true, always},
		{"Meta Platforms Ireland", "5101001", 60_000_000 + int64(month)*2_500_000, false, notCurrent},
		{"CV Kreatif Lepas Studio", "5101002", 15_000_000, false, func(m int, cur bool) bool { return m < int(s.today.Month())-1 }},
		{"Google Asia Pacific", "6201001", 8_000_000, false, always},
	}
}

// month seeds one period; its depreciation journal, written last, marks it done.
func (s *demo) month(ctx context.Context, p domain.FiscalPeriod) error {
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
	current := start.Year() == s.today.Year() && start.Month() == s.today.Month()
	day := func(d int) string {
		t := time.Date(start.Year(), start.Month(), d, 0, 0, 0, 0, time.UTC)
		if t.After(s.today) {
			t = s.today
		}
		return t.Format(time.DateOnly)
	}
	plus := func(date string, days int) string {
		t, _ := time.Parse(time.DateOnly, date)
		return t.AddDate(0, 0, days).Format(time.DateOnly)
	}

	// Client invoices through the sales event, receipts for last month's.
	for i, c := range clients {
		subtotal := domain.Rupiah(c.retainer + int64((m+i)%3)*2_500_000)
		tax := subtotal * 11 / 100
		issue := day(3)
		if _, err := s.auto.SalesInvoiceIssued(ctx, domain.SalesInvoiceIssued{
			InvoiceID: fmt.Sprintf("seed-si-%s-%02d", ym, i+1), InvoiceNumber: fmt.Sprintf("INV/%s/%03d", ym, i+1), CustomerName: c.name,
			IssueDate: issue, DueDate: plus(issue, 30), Subtotal: subtotal, TaxAmount: tax, TotalAmount: subtotal + tax,
			Description: "Retainer jasa digital marketing " + monthNames[m-1] + " " + ym[:4],
		}); err != nil {
			return err
		}
	}
	if m > 1 {
		prev := start.AddDate(0, -1, 0).Format("2006-01")
		recent := m >= int(s.today.Month())-1
		for i := range clients {
			if i == 6 && recent {
				continue // a slow payer keeps the aging buckets honest
			}
			inv, err := s.d.Store.GetInvoiceBySalesID(ctx, fmt.Sprintf("seed-si-%s-%02d", prev, i+1))
			if err != nil || inv == nil {
				return err
			}
			amount := inv.TotalAmount
			if i == 7 {
				amount /= 2
			}
			if _, err := s.auto.SalesPaymentRecorded(ctx, domain.SalesPaymentRecorded{
				PaymentID: fmt.Sprintf("seed-sp-%s-%02d", ym, i+1), InvoiceID: inv.SalesInvoiceID, Amount: amount,
				PaidOn: day(8 + i), Method: "transfer", Reference: fmt.Sprintf("BCA %s%02d", ym, i+1),
			}); err != nil {
				return err
			}
		}
	}

	// Vendor bills, posted and mostly paid from BCA.
	for i, b := range s.bills(m) {
		subtotal := domain.Rupiah(b.amount)
		var tax domain.Money
		if b.ppn {
			tax = subtotal * 11 / 100
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
		if b.paid(m, current) {
			if _, err := s.ledgers.CreatePayment(ctx, domain.KindAP, "", app.PaymentInput{
				PaymentDate: day(12), Method: "bank_transfer", CashAccountID: s.id("1102001"),
				Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: inv.TotalAmount}},
			}); err != nil {
				return err
			}
		}
	}

	// Cash and bank movements.
	movements := []app.MovementInput{
		{Kind: app.KindCashOut, EntryDate: day(28), Amount: domain.Rupiah(250_000), CashAccountID: s.id("1102001"), OffsetAccountID: s.id("8101001"), Description: "Biaya administrasi bank " + ym},
		{Kind: app.KindCashIn, EntryDate: day(28), Amount: domain.Rupiah(1_150_000), CashAccountID: s.id("1102001"), OffsetAccountID: s.id("8201001"), Description: "Bunga jasa giro " + ym},
		{Kind: app.KindCashOut, EntryDate: day(15), Amount: domain.Rupiah(3_200_000), CashAccountID: s.id("1101001"), OffsetAccountID: s.id("6301004"), Description: "Belanja ATK dan konsumsi kantor"},
	}
	if m > 1 {
		movements = append(movements,
			app.MovementInput{Kind: app.KindCashOut, EntryDate: day(10), Amount: domain.Rupiah(9_800_000), CashAccountID: s.id("1102001"), OffsetAccountID: s.id("2103001"), Description: "Setor PPh 21 masa " + monthNames[m-2]},
			app.MovementInput{Kind: app.KindCashOut, EntryDate: day(10), Amount: domain.Rupiah(29_400_000), CashAccountID: s.id("1102001"), OffsetAccountID: s.id("2104001"), Description: "Bayar iuran BPJS " + monthNames[m-2]},
		)
	}
	for _, mv := range movements {
		if _, err := s.cash.CreateMovement(ctx, "", mv); err != nil {
			return err
		}
	}
	if _, err := s.cash.CreateTransfer(ctx, "", app.MovementInput{EntryDate: day(2), Amount: domain.Rupiah(5_000_000), CashAccountID: s.id("1102001"), OffsetAccountID: s.id("1101001"), Description: "Isi ulang kas kecil"}); err != nil {
		return err
	}

	// Payroll through the hr event, paid on the 25th.
	if !current || s.today.Day() >= 25 {
		paidAt := time.Date(start.Year(), start.Month(), 25, 3, 0, 0, 0, time.UTC)
		if _, err := s.auto.PayrollPaid(ctx, domain.PayrollPaid{
			RunID: "seed-payroll-" + ym, PeriodYear: start.Year(), PeriodMonth: m, Label: "Gaji " + monthNames[m-1] + " " + ym[:4],
			PaidAt: paidAt, EmployeeCount: len(employees), TotalGross: domain.Rupiah(210_000_000), TotalNet: domain.Rupiah(190_600_000),
			TotalPph21: domain.Rupiah(9_800_000), TotalLoanDeduction: domain.Rupiah(1_200_000), TotalBpjsTkEmployee: domain.Rupiah(6_300_000),
			TotalBpjsKesEmployee: domain.Rupiah(2_100_000), TotalBpjsTkEmployer: domain.Rupiah(12_600_000), TotalBpjsKesEmployer: domain.Rupiah(8_400_000),
		}); err != nil {
			return err
		}
	}

	_, err = s.journals.Create(ctx, "", app.EntryInput{EntryDate: day(28), Description: marker, Post: true, Lines: []domain.Line{
		{AccountID: s.id("6303001"), Side: domain.Debit, Amount: domain.Rupiah(3_750_000), Memo: "Penyusutan bulanan"},
		{AccountID: s.id("1201002"), Side: domain.Credit, Amount: domain.Rupiah(3_750_000), Memo: "Penyusutan bulanan"},
	}})
	return err
}

// tinyPNG is a valid 1x1 PNG used as the demo receipt.
var tinyPNG = []byte{
	0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R', 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00, 0x0d, 'I', 'D', 'A', 'T', 0x78, 0x9c, 0x63, 0xf8, 0xcf, 0xc0, 0xf0,
	0x1f, 0x00, 0x05, 0x00, 0x01, 0xff, 0x89, 0x99, 0x3d, 0x1d, 0x00, 0x00, 0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

type claimSpec struct {
	emp           int
	category      string
	daysAgo       int
	amount        int64
	description   string
	status        domain.ClaimStatus
	project       string
	submittedDays int
}

var claimSpecs = []claimSpec{
	{1, domain.CategoryTransport, 12, 185_000, "Grab ke kantor klien PT Kopi Nusantara Jaya untuk shooting", domain.ClaimPaid, "Kopi Nusantara Q4", 11},
	{2, domain.CategoryMakan, 11, 240_000, "Makan siang tim saat shooting konten", domain.ClaimPaid, "Kopi Nusantara Q4", 10},
	{5, domain.CategoryTransport, 10, 320_000, "Taksi ke event launching Batik Prameswari", domain.ClaimPaid, "", 9},
	{9, domain.CategoryOperasional, 9, 1_450_000, "Sewa lighting tambahan untuk foto produk", domain.ClaimPaid, "Sehat Sentosa Farma", 8},
	{12, domain.CategoryMakan, 9, 275_000, "Konsumsi meeting review campaign", domain.ClaimPaid, "", 8},
	{16, domain.CategoryTransport, 8, 410_000, "Bensin dan tol kunjungan prospek Bekasi", domain.ClaimPaid, "", 7},
	{17, domain.CategoryAkomodasi, 8, 1_850_000, "Hotel 2 malam pitching di Surabaya", domain.ClaimPaid, "Pitching Arunika", 7},
	{20, domain.CategoryLainnya, 7, 150_000, "Materai dan fotokopi dokumen kontrak", domain.ClaimPaid, "", 6},
	{3, domain.CategoryTransport, 6, 95_000, "Ojek online ke studio rekaman", domain.ClaimApprovedFinance, "", 5},
	{6, domain.CategoryMakan, 6, 180_000, "Makan malam lembur posting konten", domain.ClaimApprovedFinance, "", 5},
	{10, domain.CategoryOperasional, 5, 1_200_000, "Langganan tool riset keyword bulanan", domain.ClaimApprovedFinance, "", 4},
	{13, domain.CategoryAkomodasi, 5, 2_100_000, "Tiket pesawat workshop Meta Ads Jakarta-Bali", domain.ClaimApprovedFinance, "", 4},
	{14, domain.CategoryOperasional, 4, 3_750_000, "Pembelian aset stok foto untuk 3 klien", domain.ClaimApprovedManager, "Gizi Bunda", 3},
	{18, domain.CategoryAkomodasi, 4, 1_600_000, "Penginapan visit klien di Bandung", domain.ClaimApprovedManager, "", 3},
	{21, domain.CategoryLainnya, 3, 420_000, "Kurir dokumen tagihan klien", domain.ClaimApprovedManager, "", 2},
	{4, domain.CategoryTransport, 4, 230_000, "Transport ke lokasi shooting Jepara", domain.ClaimSubmitted, "Rumah Kayu Jepara", 3},
	{7, domain.CategoryMakan, 3, 160_000, "Konsumsi live streaming klien", domain.ClaimSubmitted, "", 3},
	{11, domain.CategoryOperasional, 2, 850_000, "Backlink placement media lokal", domain.ClaimSubmitted, "EduPintar Digital", 1},
	{15, domain.CategoryTransport, 2, 145_000, "Transport meeting di kantor Meta", domain.ClaimSubmitted, "", 1},
	{19, domain.CategoryMakan, 1, 210_000, "Jamuan makan prospek klien baru", domain.ClaimSubmitted, "", 0},
	{8, domain.CategoryLainnya, 6, 600_000, "Hadiah giveaway tanpa persetujuan klien", domain.ClaimRejected, "", 5},
	{22, domain.CategoryMakan, 5, 450_000, "Makan siang pribadi", domain.ClaimRejected, "", 4},
	{23, domain.CategoryTransport, 1, 120_000, "Transport interview kandidat", domain.ClaimDraft, "", 0},
	{24, domain.CategoryMakan, 0, 95_000, "Konsumsi onboarding karyawan baru", domain.ClaimDraft, "", 0},
	{0, domain.CategoryOperasional, 2, 780_000, "Properti foto untuk konten bulan depan", domain.ClaimDraft, "", 0},
	{2, domain.CategoryTransport, 3, 75_000, "Parkir di mall saat survei lokasi", domain.ClaimCancelled, "", 0},
}

// claims seeds reimbursement claims in every state for the current month
// and pays the PAID ones in one batch (journal plus event).
func (s *demo) claims(ctx context.Context) error {
	_, total, _, err := s.d.Store.ListClaims(ctx, domain.ClaimFilter{Search: claimSpecs[0].description, Limit: 1})
	if err != nil || total > 0 {
		return err
	}
	now := s.d.Clock.Now()
	at := func(days int) *time.Time {
		t := now.Add(-time.Duration(days) * 24 * time.Hour)
		return &t
	}
	reimb := app.NewReimbursements(s.d)
	var toPay []string
	for _, sp := range claimSpecs {
		emp := employees[sp.emp]
		trx := s.today.AddDate(0, 0, -sp.daysAgo)
		c := domain.Claim{
			ID: ids.New(), EmployeeID: emp.id, EmployeeName: emp.name, DepartmentName: emp.dept, ManagerEmployeeID: emp.manager,
			Category: sp.category, TransactionDate: trx.Format(time.DateOnly), Amount: domain.Rupiah(sp.amount), Description: sp.description,
			Project: sp.project, Status: domain.ClaimDraft,
		}
		prefix := "RB-" + trx.Format("200601") + "-"
		n, err := s.d.Store.NextSequence(ctx, prefix)
		if err != nil {
			return err
		}
		c.ClaimNo = domain.SequenceNo(prefix, n)
		if err := s.d.Store.InsertClaim(ctx, c); err != nil {
			return err
		}
		rec := domain.Receipt{ID: ids.New(), ClaimID: c.ID, Filename: "struk-" + c.ClaimNo + ".png", ContentType: "image/png", SizeBytes: int64(len(tinyPNG))}
		rec.StorageKey = "reimbursements/" + rec.ID + ".png"
		if err := s.d.Storage.Put(ctx, rec.StorageKey, tinyPNG); err != nil {
			return err
		}
		if err := s.d.Store.InsertReceipt(ctx, rec); err != nil {
			return err
		}
		history := []string{"CREATED"}
		if sp.status != domain.ClaimDraft && sp.status != domain.ClaimCancelled {
			c.SubmittedAt = at(sp.submittedDays)
			c.Status = domain.ClaimSubmitted
			history = append(history, "SUBMITTED")
			policy, err := s.d.Store.GetPolicy(ctx, c.Category)
			if err != nil {
				return err
			}
			c.OverLimit, c.LimitNote = domain.EvaluateLimit(policy, c.Amount, 0)
		}
		switch sp.status {
		case domain.ClaimApprovedManager, domain.ClaimApprovedFinance, domain.ClaimPaid:
			c.Status, c.ManagerApprovedAt = domain.ClaimApprovedManager, at(max(sp.submittedDays-1, 0))
			history = append(history, "APPROVED_MANAGER")
			if sp.status != domain.ClaimApprovedManager {
				c.Status, c.FinanceApprovedAt = domain.ClaimApprovedFinance, at(max(sp.submittedDays-2, 0))
				history = append(history, "APPROVED_FINANCE")
			}
		case domain.ClaimRejected:
			c.Status, c.RejectedAt, c.RejectedStage, c.RejectReason = domain.ClaimRejected, at(sp.submittedDays-1), string(domain.ClaimSubmitted), "Tidak sesuai kebijakan reimbursement."
			history = append(history, "REJECTED")
		case domain.ClaimCancelled:
			c.Status, c.CancelledAt = domain.ClaimCancelled, at(sp.daysAgo)
			history = append(history, "CANCELLED")
		}
		if err := s.d.Store.UpdateClaim(ctx, c); err != nil {
			return err
		}
		for _, action := range history {
			note := ""
			if action == "REJECTED" {
				note = c.RejectReason
			}
			if err := s.d.Store.InsertClaimEvent(ctx, domain.ClaimEvent{ClaimID: c.ID, Action: action, ActorName: "Seeder", Note: note}); err != nil {
				return err
			}
		}
		if sp.status == domain.ClaimPaid {
			toPay = append(toPay, c.ID)
		}
	}
	_, err = reimb.Pay(ctx, admin, app.PayInput{ClaimIDs: toPay, CashAccountID: s.id("1102001"), PaidAt: s.today.Format(time.DateOnly), Notes: "Pembayaran reimbursement mingguan"})
	return err
}

// insights store AI memos (the mock without an OpenRouter key) for the
// last three months, last month, the quarter and the year to date (the
// analysis page defaults); each range is written once.
func (s *demo) insight(ctx context.Context) error {
	existing, err := s.d.Store.ListInsights(ctx, 50)
	if err != nil {
		return err
	}
	have := map[string]bool{}
	for _, i := range existing {
		have[i.DateFrom+i.DateTo] = true
	}
	monthStart := time.Date(s.today.Year(), s.today.Month(), 1, 0, 0, 0, 0, time.UTC)
	quarter := time.Date(s.today.Year(), time.Month((int(s.today.Month())-1)/3*3+1), 1, 0, 0, 0, 0, time.UTC)
	for _, r := range [][2]time.Time{
		{monthStart.AddDate(0, -3, 0), monthStart.AddDate(0, 0, -1)},
		{monthStart.AddDate(0, -1, 0), monthStart.AddDate(0, 0, -1)},
		{quarter, s.today},
		{time.Date(s.today.Year(), 1, 1, 0, 0, 0, 0, time.UTC), s.today},
	} {
		from, to := r[0].Format(time.DateOnly), r[1].Format(time.DateOnly)
		if have[from+to] {
			continue
		}
		have[from+to] = true
		if _, err := app.NewAnalysis(s.d).GenerateInsight(ctx, "", from, to, "id"); err != nil {
			return err
		}
	}
	return nil
}
