package app_test

import (
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

func TestJournalLifecycle(t *testing.T) {
	h := newHarness(t)
	j := app.NewJournals(h.deps)
	kas, beban := h.id("1101001"), h.id("6301004")

	_, err := j.Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-10-05", Lines: []domain.Line{line(beban, domain.Debit, 1500), line(kas, domain.Credit, 1000)}})
	h.wantErr(err, "Jurnal tidak balance: Debit 1500 ≠ Credit 1000")
	_, err = j.Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-09-05", Lines: []domain.Line{line(beban, domain.Debit, 1000), line(kas, domain.Credit, 1000)}})
	h.wantErr(err, "Tidak ada fiscal period OPEN untuk tanggal jurnal. Konfigurasi Fiscal Years terlebih dahulu.")
	_, err = j.Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-10-05", Lines: []domain.Line{line(h.id("1101000"), domain.Debit, 1000), line(kas, domain.Credit, 1000)}})
	h.wantErr(err, "Akun jurnal harus postable (bukan header)")

	e, err := j.Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-10-05", Description: "ATK", Lines: []domain.Line{line(beban, domain.Debit, 1000), line(kas, domain.Credit, 1000)}})
	h.ok(err)
	if e.EntryNo != "JE-202610-0001" || e.Status != domain.StatusDraft || !e.CanEdit() {
		t.Fatalf("draft %+v", e)
	}
	e2, err := j.Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-10-06", Lines: []domain.Line{line(beban, domain.Debit, 10), line(kas, domain.Credit, 10)}})
	h.ok(err)
	if e2.EntryNo != "JE-202610-0002" {
		t.Fatal(e2.EntryNo)
	}
	h.ok(j.Delete(h.ctx, "u1", e2.ID))

	e, err = j.Update(h.ctx, "u1", e.ID, app.EntryInput{EntryDate: "2026-10-07", Description: "ATK revisi", Lines: []domain.Line{line(beban, domain.Debit, 2000), line(kas, domain.Credit, 2000)}})
	h.ok(err)
	if debit, _ := e.Totals(); debit != rp(2000) || e.EntryDate != "2026-10-07" {
		t.Fatalf("update %+v", e)
	}
	e, err = j.Post(h.ctx, "u2", e.ID)
	h.ok(err)
	if e.Status != domain.StatusPosted || e.PostedBy != "u2" {
		t.Fatalf("post %+v", e)
	}
	_, err = j.Post(h.ctx, "u2", e.ID)
	h.wantErr(err, "Journal entry sudah POSTED")
	h.wantErr(j.Delete(h.ctx, "u1", e.ID), "Journal entry POSTED tidak bisa diubah/dihapus")
}

func TestFiscalPeriods(t *testing.T) {
	h := newHarness(t)
	f := app.NewFiscal(h.deps)
	years, err := f.ListYears(h.ctx, "", nil)
	h.ok(err)
	year := years[0]
	if year.OpenCount() != 1 || year.Periods[9].Status != domain.PeriodOpen {
		t.Fatalf("october open %+v", year.Periods)
	}
	nov := year.Periods[10]
	_, err = f.OpenPeriod(h.ctx, nov.ID, false)
	h.wantErr(err, "Tutup period Oktober 2026 terlebih dahulu sebelum OPEN November 2026")

	// A draft blocks closing October.
	_, err = app.NewJournals(h.deps).Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-10-10", Lines: []domain.Line{line(h.id("6301004"), domain.Debit, 1), line(h.id("1101001"), domain.Credit, 1)}})
	h.ok(err)
	pv, err := f.ClosePreview(h.ctx, year.Periods[9].ID)
	h.ok(err)
	if pv.CanClose || len(pv.DraftEntries) != 1 || pv.Blockers[0] != "Masih ada 1 jurnal DRAFT. Posting atau hapus dulu sebelum closing." {
		t.Fatalf("preview %+v", pv)
	}
	_, err = f.ClosePeriod(h.ctx, year.Periods[9].ID)
	h.wantErr(err, "Masih ada 1 jurnal DRAFT. Posting atau hapus dulu sebelum closing.")

	cov, err := f.Coverage(h.ctx, "2026-11-03")
	h.ok(err)
	if cov.Ready || cov.Suggestion == nil || !cov.Suggestion.CanOpen || len(cov.Suggestion.ClosePrevious) != 1 {
		t.Fatalf("coverage %+v", cov)
	}
	p, err := f.OpenPeriod(h.ctx, nov.ID, true)
	h.ok(err)
	if p.Status != domain.PeriodOpen {
		t.Fatal(p.Status)
	}

	// A new year with an OPEN period needs every earlier year closed.
	_, err = f.CreateYear(h.ctx, "u1", app.YearInput{Code: "FY2027", Name: "2027", StartDate: "2027-01-01", EndDate: "2027-12-31", IsActive: true})
	h.wantErr(err, "Tidak bisa OPEN fiscal: fiscal year FY2026 (Tahun Buku 2026) masih punya 1 period OPEN. Closing semua period fiscal sebelumnya terlebih dahulu.")
	_, err = f.CreateYear(h.ctx, "u1", app.YearInput{Code: "FY2026", Name: "dup", StartDate: "2026-01-01", EndDate: "2026-12-31",
		Periods: []domain.PeriodInput{{PeriodNo: 1, Name: "Jan", StartDate: "2026-01-01", EndDate: "2026-01-31", Status: domain.PeriodClosed}}})
	h.wantErr(err, "Kode fiscal year sudah dipakai")
	h.wantErr(f.DeleteYear(h.ctx, "u1", year.ID), "Fiscal year masih dipakai journal entry dan tidak bisa dihapus")
}

func TestBeginningBalanceFromPriorYear(t *testing.T) {
	h := newHarness(t)
	f := app.NewFiscal(h.deps)
	years, _ := f.ListYears(h.ctx, "FY2026", nil)
	fy26 := years[0]

	saved, err := f.SaveBeginningBalance(h.ctx, "u1", fy26.ID, []domain.Line{
		line(h.id("1102001"), domain.Debit, 500_000), line(h.id("3101001"), domain.Credit, 400_000), line(h.id("1101001"), domain.Debit, 0),
	}, false)
	h.wantErr(err, "Saldo awal tidak balance: Debit 500000 ≠ Credit 400000")
	saved, err = f.SaveBeginningBalance(h.ctx, "u1", fy26.ID, []domain.Line{line(h.id("1102001"), domain.Debit, 500_000), line(h.id("3101001"), domain.Credit, 500_000)}, true)
	h.ok(err)
	if saved.EntryNo != "OB-2026-0001" || saved.EntryType != domain.EntryOpening || saved.Status != domain.StatusPosted {
		t.Fatalf("opening %+v", saved)
	}
	_, err = f.SaveBeginningBalance(h.ctx, "u1", fy26.ID, []domain.Line{line(h.id("1102001"), domain.Debit, 1), line(h.id("3101001"), domain.Credit, 1)}, true)
	h.wantErr(err, "Beginning balance sudah POSTED dan tidak bisa diubah")

	// October activity: revenue 300k cash, expense 100k. FY2027's suggestion
	// carries the balance sheet and folds the 200k profit into Laba Ditahan.
	j := app.NewJournals(h.deps)
	for _, in := range []app.EntryInput{
		{EntryDate: "2026-10-01", Post: true, Lines: []domain.Line{line(h.id("1102001"), domain.Debit, 300_000), line(h.id("4101001"), domain.Credit, 300_000)}},
		{EntryDate: "2026-10-02", Post: true, Lines: []domain.Line{line(h.id("6301001"), domain.Debit, 100_000), line(h.id("1102001"), domain.Credit, 100_000)}},
	} {
		_, err := j.Create(h.ctx, "u1", in)
		h.ok(err)
	}
	periods := make([]domain.PeriodInput, 0, 12)
	for _, p := range fy26.Periods {
		periods = append(periods, domain.PeriodInput{PeriodNo: p.PeriodNo, Name: p.Name, StartDate: p.StartDate, EndDate: p.EndDate, Status: domain.PeriodClosed})
	}
	_, err = f.UpdateYear(h.ctx, "u1", fy26.ID, app.YearInput{Code: "FY2026", Name: "Tahun Buku 2026", StartDate: "2026-01-01", EndDate: "2026-12-31", IsActive: true, Periods: periods})
	h.ok(err)
	fy27, err := f.CreateYear(h.ctx, "u1", app.YearInput{Code: "FY2027", Name: "Tahun Buku 2027", StartDate: "2027-01-01", EndDate: "2027-12-31", IsActive: true})
	h.ok(err)
	sug, err := f.BeginningBalance(h.ctx, fy27.ID)
	h.ok(err)
	if sug.Prior == nil || sug.Prior.Code != "FY2026" || sug.TotalDebit != sug.TotalCredit || sug.TotalDebit != rp(700_000) {
		t.Fatalf("suggestion %+v", sug)
	}
	var re *app.OpeningLine
	for i := range sug.Lines {
		if sug.Lines[i].Source == "RETAINED_EARNINGS" {
			re = &sug.Lines[i]
		}
	}
	if re == nil || re.Amount != rp(200_000) || re.Side != domain.Credit || re.Account.Code != "3201001" {
		t.Fatalf("retained earnings %+v", re)
	}
}

func TestAutoPostPayroll(t *testing.T) {
	h := newHarness(t)
	auto := app.NewAutoPost(h.deps)
	run := domain.PayrollPaid{RunID: "run-oct", PeriodYear: 2026, PeriodMonth: 10, Label: "Gaji Oktober", PaidAt: ptime("2026-10-24T20:00:00Z"),
		TotalGross: rp(1_000_000), TotalNet: rp(850_000), TotalPph21: rp(50_000), TotalBpjsTkEmployee: rp(30_000), TotalBpjsKesEmployee: rp(10_000),
		TotalLoanDeduction: rp(60_000), TotalBpjsTkEmployer: rp(57_000), TotalBpjsKesEmployer: rp(40_000)}

	// Break PAYROLL_LOAN_DEDUCTION: a required account removed.
	maps, err := app.NewAccounts(h.deps).ListMappings(h.ctx, domain.MappingFilter{Search: domain.EventPayrollLoanDeduction})
	h.ok(err)
	loan := maps[0]
	broken := loan.Template()
	broken[1].AccountID = ""
	h.ok(h.deps.Store.ReplaceMappingLines(h.ctx, loan.ID, broken))

	results, err := auto.PayrollPaid(h.ctx, run)
	h.ok(err)
	if len(results) != 8 {
		t.Fatalf("results %+v", results)
	}
	if got := domain.Summarize(results...); got != "jurnal posted; jurnal draft: Journal mapping belum lengkap (akun COA wajib belum diisi)" {
		t.Fatal(got)
	}
	accrual, err := h.deps.Store.FindEntryBySource(h.ctx, domain.EventPayrollAccrual, "run-oct")
	h.ok(err)
	// Paid 03:00 WIB on the 25th: dated in Jakarta.
	if accrual == nil || accrual.EntryDate != "2026-10-25" || accrual.EntryType != domain.EntryAuto || accrual.Description != "Payroll 10/2026: beban gaji" {
		t.Fatalf("accrual %+v", accrual)
	}
	// Redelivery is a no-op.
	again, err := auto.PayrollPaid(h.ctx, run)
	h.ok(err)
	if again[0].Reason != domain.AlreadyExists {
		t.Fatalf("repeat %+v", again[0])
	}
	logs, err := app.NewJournals(h.deps).ListPostingLogs(h.ctx, domain.PostDraft, "", 0)
	h.ok(err)
	if len(logs) != 1 || logs[0].EventCode != domain.EventPayrollLoanDeduction {
		t.Fatalf("draft logs %+v", logs)
	}
	// Fix the mapping and retry the draft.
	h.ok(h.deps.Store.ReplaceMappingLines(h.ctx, loan.ID, loan.Template()))
	fixed, err := app.NewJournals(h.deps).RetryPosting(h.ctx, "u-fin", logs[0].ID)
	h.ok(err)
	if fixed.Status != domain.PostPosted || fixed.EntryID == "" || fixed.Attempts != 3 {
		t.Fatalf("retry %+v", fixed)
	}
	// Salary payable clears: gross - deductions - net = 0.
	tb, err := app.NewReports(h.deps).TrialBalance(h.ctx, "")
	h.ok(err)
	for _, r := range tb.Rows {
		if r.Code == "2102001" && (r.Debit != 0 || r.Credit != 0) {
			t.Fatalf("hutang gaji not cleared %+v", r)
		}
	}
	if tb.TotalDebit != tb.TotalCredit {
		t.Fatal("trial balance")
	}
}

func TestSalesEventsCreateReceivables(t *testing.T) {
	h := newHarness(t)
	auto := app.NewAutoPost(h.deps)
	ledgers := app.NewLedgers(h.deps)
	issued := domain.SalesInvoiceIssued{InvoiceID: "si-1", InvoiceNumber: "INV/2026/10/001", CustomerName: "PT Kopi Nusantara", IssueDate: "2026-10-03",
		DueDate: "2026-10-20", Subtotal: rp(10_000_000), TaxAmount: rp(1_100_000), TotalAmount: rp(11_100_000)}

	_, err := auto.SalesPaymentRecorded(h.ctx, domain.SalesPaymentRecorded{PaymentID: "sp-0", InvoiceID: "si-1", Amount: rp(1), PaidOn: "2026-10-05"})
	if err == nil {
		t.Fatal("payment before invoice must be retried")
	}
	inv, err := auto.SalesInvoiceIssued(h.ctx, issued)
	h.ok(err)
	if inv.Status != domain.StatusPosted || inv.Kind != domain.KindAR || inv.JournalEntryID == "" || inv.InvoiceNo != "AR-20261026-0001" {
		t.Fatalf("ar %+v", inv)
	}
	again, err := auto.SalesInvoiceIssued(h.ctx, issued)
	h.ok(err)
	if again.ID != inv.ID {
		t.Fatal("not idempotent")
	}
	entry, err := h.deps.Store.GetJournalEntry(h.ctx, inv.JournalEntryID)
	h.ok(err)
	if len(entry.Lines) != 3 || entry.SourceEventCode != domain.EventSaleARInvoice {
		t.Fatalf("journal %+v", entry.Lines)
	}
	if ledgers.PaymentStatus(*inv) != "overdue" {
		t.Fatal(ledgers.PaymentStatus(*inv))
	}
	res, err := auto.SalesPaymentRecorded(h.ctx, domain.SalesPaymentRecorded{PaymentID: "sp-1", InvoiceID: "si-1", Amount: rp(5_000_000), PaidOn: "2026-10-21", Method: "bank_transfer"})
	h.ok(err)
	if res.Payment.Kind != domain.KindAR || res.Payment.JournalEntryID == "" || res.Invoices[0].Outstanding() != rp(6_100_000) {
		t.Fatalf("receipt %+v", res)
	}
	// An overpayment is capped at the outstanding amount.
	res, err = auto.SalesPaymentRecorded(h.ctx, domain.SalesPaymentRecorded{PaymentID: "sp-2", InvoiceID: "si-1", Amount: rp(9_000_000), PaidOn: "2026-10-22"})
	h.ok(err)
	if res.Payment.Amount != rp(6_100_000) || ledgers.PaymentStatus(res.Invoices[0]) != "paid" {
		t.Fatalf("capped %+v", res.Payment)
	}
	aging, err := ledgers.Aging(h.ctx, domain.KindAR, "")
	h.ok(err)
	if aging.Total != 0 {
		t.Fatalf("aging %+v", aging)
	}
}

func TestPayablesPaymentAndVoid(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	inv, err := l.CreateInvoice(h.ctx, domain.KindAP, "u1", app.InvoiceInput{InvoiceDate: "2026-10-02", DueDate: "2026-09-01", PartyName: "PT Telkom"})
	if err == nil || inv != nil {
		t.Fatal("validation expected")
	}
	inv, err = l.CreateInvoice(h.ctx, domain.KindAP, "u1", app.InvoiceInput{InvoiceDate: "2026-10-02", DueDate: "2026-10-30", PartyName: "PT Telkom",
		Reference: "TLK-01", AccountID: h.id("6301002"), Subtotal: rp(4_500_000), TaxAmount: rp(495_000)})
	h.ok(err)
	if inv.Status != domain.StatusDraft || inv.TotalAmount != rp(4_995_000) {
		t.Fatalf("draft %+v", inv)
	}
	_, err = l.CreatePayment(h.ctx, domain.KindAP, "u1", app.PaymentInput{Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: rp(1)}}})
	h.wantErr(err, "Hanya invoice POSTED yang bisa dibayar")
	inv, note, err := l.PostInvoice(h.ctx, domain.KindAP, "u2", inv.ID)
	h.ok(err)
	if note != "jurnal posted" || inv.JournalEntryID == "" {
		t.Fatalf("post %q %+v", note, inv)
	}
	entry, _ := h.deps.Store.GetJournalEntry(h.ctx, inv.JournalEntryID)
	if entry.Lines[0].AccountID != h.id("6301002") {
		t.Fatalf("expense override %+v", entry.Lines)
	}
	_, err = l.CreatePayment(h.ctx, domain.KindAP, "u1", app.PaymentInput{Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: rp(5_000_000)}}})
	h.wantErr(err, "Amount pembayaran tidak boleh melebihi outstanding")
	res, err := l.CreatePayment(h.ctx, domain.KindAP, "u1", app.PaymentInput{PaymentDate: "2026-10-10", CashAccountID: h.id("1102002"),
		Allocations: []domain.Allocation{{InvoiceID: inv.ID, Amount: rp(4_995_000)}}})
	h.ok(err)
	if res.Payment.PaymentNo != "APP-20261026-0001" || res.Note != "jurnal posted" || l.PaymentStatus(res.Invoices[0]) != "paid" {
		t.Fatalf("payment %+v", res)
	}
	payJournal, _ := h.deps.Store.GetJournalEntry(h.ctx, res.Payment.JournalEntryID)
	if payJournal.Lines[1].AccountID != h.id("1102002") {
		t.Fatalf("cash override %+v", payJournal.Lines)
	}

	_, err = l.VoidPayment(h.ctx, domain.KindAP, "u2", res.Payment.ID, " abc ")
	h.wantErr(err, "Alasan void wajib diisi (minimal 5 karakter)")
	voided, err := l.VoidPayment(h.ctx, domain.KindAP, "u2", res.Payment.ID, "salah rekening")
	h.ok(err)
	if voided.Payment.Status != domain.StatusVoid || voided.Note != "jurnal pembalik diposting" || voided.Invoices[0].Outstanding() != rp(4_995_000) {
		t.Fatalf("void %+v", voided)
	}
	_, err = l.VoidPayment(h.ctx, domain.KindAP, "u2", res.Payment.ID, "salah rekening")
	h.wantErr(err, "Pembayaran ini sudah di-void")
	reversal, err := h.deps.Store.FindEntryBySource(h.ctx, domain.EventPurchasePayment+"_VOID", res.Payment.ID)
	h.ok(err)
	if reversal == nil || reversal.Lines[0].Side != domain.Credit {
		t.Fatalf("reversal %+v", reversal)
	}
	sub, err := l.SubsidiaryLedger(h.ctx, domain.KindAP, "PT Telkom", "", "")
	h.ok(err)
	if sub.Closing != rp(4_995_000) || len(sub.Lines) != 1 {
		t.Fatalf("subsidiary %+v", sub)
	}
}

func TestReportsAndCashBank(t *testing.T) {
	h := newHarness(t)
	cb := app.NewCashBank(h.deps)
	r := app.NewReports(h.deps)
	j := app.NewJournals(h.deps)
	_, err := j.Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-10-01", Post: true, Lines: []domain.Line{line(h.id("1102001"), domain.Debit, 1_000_000), line(h.id("3101001"), domain.Credit, 1_000_000)}})
	h.ok(err)
	_, err = cb.CreateMovement(h.ctx, "u1", app.MovementInput{Kind: app.KindCashIn, EntryDate: "2026-10-05", Amount: rp(400_000), CashAccountID: h.id("1102001"), OffsetAccountID: h.id("4101003")})
	h.ok(err)
	_, err = cb.CreateMovement(h.ctx, "u1", app.MovementInput{Kind: app.KindCashOut, EntryDate: "2026-10-06", Amount: rp(150_000), CashAccountID: h.id("1102001"), OffsetAccountID: h.id("6301001")})
	h.ok(err)
	_, err = cb.CreateTransfer(h.ctx, "u1", app.MovementInput{EntryDate: "2026-10-07", Amount: rp(50_000), CashAccountID: h.id("1102001"), OffsetAccountID: h.id("1101001")})
	h.ok(err)
	_, err = cb.CreateTransfer(h.ctx, "u1", app.MovementInput{EntryDate: "2026-10-07", Amount: rp(1), CashAccountID: h.id("1102001"), OffsetAccountID: h.id("6301001")})
	h.wantErr(err, "Akun kas/bank tidak valid")
	// Depreciation credits a contra asset: it must shrink total assets.
	_, err = j.Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-10-08", Post: true, Lines: []domain.Line{line(h.id("6303001"), domain.Debit, 50_000), line(h.id("1201002"), domain.Credit, 50_000)}})
	h.ok(err)

	moves, total, err := cb.ListMovements(h.ctx, app.KindCashIn, domain.JournalFilter{Limit: 10})
	h.ok(err)
	if total != 1 || moves[0].Amount != rp(400_000) || moves[0].Offset.Code != "4101003" {
		t.Fatalf("movements %+v", moves)
	}
	ledger, err := cb.Ledger(h.ctx, h.id("1102001"), "2026-10-05", "")
	h.ok(err)
	if ledger.Opening != rp(1_000_000) || ledger.Closing != rp(1_200_000) || len(ledger.Lines) != 3 {
		t.Fatalf("ledger %+v", ledger)
	}
	is, err := r.IncomeStatement(h.ctx, "2026-10-01", "2026-10-26")
	h.ok(err)
	if is.TotalRevenue != rp(400_000) || is.NetIncome != rp(200_000) || is.GrossProfit != rp(400_000) {
		t.Fatalf("income %+v", is)
	}
	bs, err := r.BalanceSheet(h.ctx, "")
	h.ok(err)
	if bs.TotalAssets != rp(1_200_000) || bs.CurrentYearEarnings != rp(200_000) || bs.TotalLiabilitiesAndEquity != bs.TotalAssets {
		t.Fatalf("balance sheet %+v", bs)
	}
	cf, err := r.CashFlow(h.ctx, "2026-10-01", "2026-10-26")
	h.ok(err)
	if cf.Opening != 0 || cf.Closing != rp(1_250_000) || cf.NetChange != rp(1_250_000) || cf.Operating.Total != rp(250_000) || cf.Financing.Total != rp(1_000_000) {
		t.Fatalf("cash flow %+v", cf)
	}
	d, err := r.Dashboard(h.ctx, "", "", "")
	h.ok(err)
	if d.Cash.Closing != rp(1_250_000) || d.TrialBalanceDiff != 0 || d.PostedEntries != 5 || len(d.MonthlyTrend) != 10 {
		t.Fatalf("dashboard %+v", d)
	}
	if k := d.BalanceSheet; k.CurrentYearEarnings != rp(200_000) || k.TotalAssets != k.TotalLiabilities+k.TotalEquity {
		t.Fatalf("dashboard equity %+v", k)
	}
}

func TestAnalysisAndInsight(t *testing.T) {
	h := newHarness(t)
	j := app.NewJournals(h.deps)
	for _, in := range []app.EntryInput{
		{EntryDate: "2026-10-01", Post: true, Lines: []domain.Line{line(h.id("1102001"), domain.Debit, 1_000_000), line(h.id("3101001"), domain.Credit, 1_000_000)}},
		{EntryDate: "2026-10-02", Post: true, Lines: []domain.Line{line(h.id("1103001"), domain.Debit, 600_000), line(h.id("4101001"), domain.Credit, 600_000)}},
		{EntryDate: "2026-10-03", Post: true, Lines: []domain.Line{line(h.id("6301001"), domain.Debit, 300_000), line(h.id("2101001"), domain.Credit, 300_000)}},
		{EntryDate: "2026-10-04", Post: true, Lines: []domain.Line{line(h.id("5101001"), domain.Debit, 120_000), line(h.id("1102001"), domain.Credit, 120_000)}},
	} {
		_, err := j.Create(h.ctx, "u1", in)
		h.ok(err)
	}
	fa, err := app.NewAnalysis(h.deps).Analyze(h.ctx, "2026-10-01", "2026-10-30", true, "")
	h.ok(err)
	metric := func(group []app.Metric, key string) app.Metric {
		for _, m := range group {
			if m.Key == key {
				return m
			}
		}
		t.Fatalf("metric %s missing", key)
		return app.Metric{}
	}
	if gm := metric(fa.Profitability, "gross_margin"); !gm.Current.OK || gm.Current.Value != 80 {
		t.Fatalf("gross margin %+v", gm)
	}
	// CA 880k + 600k = 1.48m over CL 300k.
	if cr := metric(fa.Liquidity, "current_ratio"); cr.Current.Value < 4.93 || cr.Current.Value > 4.94 {
		t.Fatalf("current ratio %+v", cr)
	}
	if dso := metric(fa.Efficiency, "dso"); dso.Current.Value != 30 {
		t.Fatalf("dso %+v", dso)
	}
	if fa.PrevFrom != "2026-09-01" || fa.PrevTo != "2026-09-30" || metric(fa.Profitability, "revenue_growth").Current.OK {
		t.Fatalf("previous %s %s", fa.PrevFrom, fa.PrevTo)
	}
	if len(fa.Expenses) != 2 || fa.Expenses[0].Code != "6301001" || len(fa.TopMovers) != 2 {
		t.Fatalf("expenses %+v", fa.Expenses)
	}
	ins, err := app.NewAnalysis(h.deps).GenerateInsight(h.ctx, "u1", "2026-10-01", "2026-10-30", "")
	h.ok(err)
	if ins.Model != "mock" || ins.Language != "id" || h.insights.last.Data["rasio"] == nil {
		t.Fatalf("insight %+v", ins)
	}
	list, err := app.NewAnalysis(h.deps).ListInsights(h.ctx, 0)
	h.ok(err)
	if len(list) != 1 {
		t.Fatal(len(list))
	}
}
