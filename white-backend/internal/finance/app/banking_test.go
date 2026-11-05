package app_test

import (
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

func lineOn(e *domain.JournalEntry, account string) domain.JournalLine {
	for _, l := range e.Lines {
		if l.AccountID == account {
			return l
		}
	}
	return domain.JournalLine{}
}

func statusOf(h *harness, b *app.Banking, account, description string) domain.StatementLine {
	h.t.Helper()
	rows, _, err := b.Lines(h.ctx, domain.StatementFilter{AccountID: account, PageSize: 100})
	h.ok(err)
	for _, r := range rows {
		if r.Description == description {
			return r
		}
	}
	h.t.Fatalf("line %q missing", description)
	return domain.StatementLine{}
}

func TestBankReconciliation(t *testing.T) {
	h := newHarness(t)
	b := app.NewBanking(h.deps)
	kas, beban := h.id("1101001"), h.id("6301004")
	j := app.NewJournals(h.deps)
	e, err := j.Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-10-05", Description: "Setoran Kopi Senja", Lines: []domain.Line{line(kas, domain.Debit, 1_000_000), line(beban, domain.Credit, 1_000_000)}})
	h.ok(err)
	_, err = j.Post(h.ctx, "u1", e.ID)
	h.ok(err)

	csv := []byte("Tanggal,Keterangan,Debit,Kredit\n" +
		"05/10/2026,TRF DARI KOPI SENJA,,1.000.000\n" +
		"06/10/2026,SETORAN TUNAI,,1.000.000\n" +
		"07/10/2026,BIAYA ADMIN,15.000,\n")
	res, err := b.Import(h.ctx, "u1", kas, "mutasi.csv", csv)
	h.ok(err)
	if res.Inserted != 3 || res.Matched != 1 || res.Unmatched != 2 {
		t.Fatalf("import %+v", res)
	}

	// The same file twice adds nothing.
	again, err := b.Import(h.ctx, "u1", kas, "mutasi.csv", csv)
	h.ok(err)
	if again.Inserted != 0 || again.Skipped != 3 {
		t.Fatalf("second import %+v", again)
	}

	// A journal line settles one row only.
	settled := statusOf(h, b, kas, "TRF DARI KOPI SENJA")
	if settled.Status != domain.LineMatched || settled.MatchedEntryNo != e.EntryNo || settled.VerifiedBy != "" {
		t.Fatalf("auto match %+v", settled)
	}
	other := statusOf(h, b, kas, "SETORAN TUNAI")
	_, err = b.MatchJournal(h.ctx, "u1", other.ID, settled.MatchedLineID)
	h.wantErr(err, "Jurnal ini sudah dipakai oleh baris mutasi lain.")

	// A bank fee the books do not have yet becomes a journal.
	fee := statusOf(h, b, kas, "BIAYA ADMIN")
	_, err = b.Record(h.ctx, "u1", fee.ID, kas, "")
	h.wantErr(err, "Akun lawan harus berbeda dari rekening")
	got, err := b.Record(h.ctx, "u2", fee.ID, beban, "Biaya admin Oktober")
	h.ok(err)
	if got.Status != domain.LineMatched || got.VerifiedBy != "u2" {
		t.Fatalf("record %+v", got)
	}
	posted, err := j.Get(h.ctx, got.MatchedEntryID)
	h.ok(err)
	if posted.Status != domain.StatusPosted || lineOn(posted, kas).Side != domain.Credit || lineOn(posted, kas).Amount != rp(15_000) || lineOn(posted, beban).Side != domain.Debit {
		t.Fatalf("fee journal %+v", posted)
	}
	_, err = b.Ignore(h.ctx, "u1", got.ID, "x")
	h.wantErr(err, "Baris mutasi ini sudah diselesaikan. Batalkan dulu untuk mengubahnya.")

	// Exceptions: an ignore needs a reason and can be undone.
	_, err = b.Ignore(h.ctx, "u1", other.ID, " ")
	h.wantErr(err, "Alasan wajib diisi")
	ignored, err := b.Ignore(h.ctx, "u1", other.ID, "Transfer antar rekening")
	h.ok(err)
	if ignored.Status != domain.LineIgnored {
		t.Fatalf("ignore %+v", ignored)
	}
	reopened, err := b.Reopen(h.ctx, other.ID)
	h.ok(err)
	if reopened.Status != domain.LineUnmatched || reopened.Note != "" {
		t.Fatalf("reopen %+v", reopened)
	}
}

func TestNamelessMoneyIn(t *testing.T) {
	h := newHarness(t)
	b := app.NewBanking(h.deps)
	l := app.NewLedgers(h.deps)
	kas, revenue := h.id("1101001"), h.id("4101001")
	inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: "2026-10-01", PartyName: "Kopi Senja", AccountID: revenue, Subtotal: rp(2_500_000)})
	h.ok(err)
	_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
	h.ok(err)

	res, err := b.Import(h.ctx, "u1", kas, "m.csv", []byte("Tanggal,Keterangan,Debit,Kredit\n10/10/2026,SETORAN TANPA NAMA,,2.500.000\n11/10/2026,TRANSFER MASUK,,99.000\n"))
	h.ok(err)
	if res.Suggested != 1 || res.Unmatched != 1 {
		t.Fatalf("import %+v", res)
	}
	line := statusOf(h, b, kas, "SETORAN TANPA NAMA")
	if line.Status != domain.LineSuggested || line.SuggestedKind != domain.SuggestInvoice || line.SuggestedRef != inv.ID {
		t.Fatalf("suggestion %+v", line)
	}

	// Money out cannot settle a receivable.
	out, err := b.Import(h.ctx, "u1", kas, "k.csv", []byte("Tanggal,Keterangan,Debit,Kredit\n12/10/2026,KELUAR,1.000,\n"))
	h.ok(err)
	if out.Inserted != 1 {
		t.Fatalf("%+v", out)
	}
	_, err = b.MatchInvoice(h.ctx, "u1", statusOf(h, b, kas, "KELUAR").ID, inv.ID)
	h.wantErr(err, "Uang masuk dicocokkan ke piutang, uang keluar ke hutang")

	// A person confirms the suggestion: the receipt is recorded and the invoice closes.
	done, err := b.MatchInvoice(h.ctx, "u2", line.ID, inv.ID)
	h.ok(err)
	if done.Status != domain.LineMatched || done.VerifiedBy != "u2" || done.MatchedEntryID == "" {
		t.Fatalf("match %+v", done)
	}
	after, err := l.GetInvoice(h.ctx, domain.KindAR, inv.ID)
	h.ok(err)
	if after.Outstanding() != 0 {
		t.Fatalf("outstanding %d", after.Outstanding())
	}
}

func TestCreditCardReconciliation(t *testing.T) {
	h := newHarness(t)
	b := app.NewBanking(h.deps)
	card := h.acc["2101001"]
	card.IsCreditCard = true
	h.ok(h.deps.Store.UpdateAccount(h.ctx, card, "u1"))

	accounts, err := b.Accounts(h.ctx)
	h.ok(err)
	found := false
	for _, a := range accounts {
		found = found || (a.ID == card.ID && a.IsCreditCard)
	}
	if !found {
		t.Fatal("card account not listed")
	}
	// A card charge credits the liability and debits the expense.
	res, err := b.Import(h.ctx, "u1", card.ID, "cc.csv", []byte("Tanggal,Keterangan,Debit,Kredit\n08/10/2026,ADOBE SUBSCRIPTION,750.000,\n"))
	h.ok(err)
	if res.Inserted != 1 {
		t.Fatalf("%+v", res)
	}
	got, err := b.Record(h.ctx, "u1", statusOf(h, b, card.ID, "ADOBE SUBSCRIPTION").ID, h.id("6301004"), "")
	h.ok(err)
	e, err := app.NewJournals(h.deps).Get(h.ctx, got.MatchedEntryID)
	h.ok(err)
	if lineOn(e, card.ID).Side != domain.Credit || lineOn(e, h.id("6301004")).Side != domain.Debit {
		t.Fatalf("card journal %+v", e.Lines)
	}
}

func TestRecurringJournals(t *testing.T) {
	h := newHarness(t)
	s := app.NewSchedules(h.deps)
	kas, beban := h.id("1101001"), h.id("6301004")
	lines := []domain.Line{line(beban, domain.Debit, 300_000), line(kas, domain.Credit, 300_000)}

	_, err := s.CreateRecurring(h.ctx, "u1", app.RecurringInput{Name: "Sewa", Frequency: domain.FreqMonthly, StartDate: "2026-10-05", Lines: []domain.Line{line(beban, domain.Debit, 1), line(kas, domain.Credit, 2)}})
	if err == nil {
		t.Fatal("unbalanced template accepted")
	}
	r, err := s.CreateRecurring(h.ctx, "u1", app.RecurringInput{Name: "Sewa", Frequency: domain.FreqMonthly, StartDate: "2026-10-05", Lines: lines})
	h.ok(err)

	res, err := s.RunDue(h.ctx)
	h.ok(err)
	if res.Posted != 1 || res.Failed != 0 {
		t.Fatalf("run %+v", res)
	}
	r, err = h.deps.Store.GetRecurringJournal(h.ctx, r.ID)
	h.ok(err)
	if r.NextRun != "2026-11-05" || r.LastEntryID == "" {
		t.Fatalf("after run %+v", r)
	}
	// Nothing posts twice.
	again, err := s.RunDue(h.ctx)
	h.ok(err)
	if again.Posted != 0 {
		t.Fatalf("second run %+v", again)
	}

	// A closed period keeps the date due and shows the reason.
	closed, err := s.CreateRecurring(h.ctx, "u1", app.RecurringInput{Name: "Lama", Frequency: domain.FreqMonthly, StartDate: "2026-08-31", Lines: lines})
	h.ok(err)
	failed, err := s.RunDue(h.ctx)
	h.ok(err)
	if failed.Failed != 1 {
		t.Fatalf("closed period %+v", failed)
	}
	closed, err = h.deps.Store.GetRecurringJournal(h.ctx, closed.ID)
	h.ok(err)
	if closed.NextRun != "2026-08-31" || closed.LastError == "" {
		t.Fatalf("closed %+v", closed)
	}

	// An end date stops the schedule.
	ending, err := s.CreateRecurring(h.ctx, "u1", app.RecurringInput{Name: "Selesai", Frequency: domain.FreqMonthly, StartDate: "2026-10-01", EndDate: "2026-10-20", Lines: lines})
	h.ok(err)
	_, err = s.RunDue(h.ctx)
	h.ok(err)
	ending, err = h.deps.Store.GetRecurringJournal(h.ctx, ending.ID)
	h.ok(err)
	if ending.IsActive {
		t.Fatalf("still active %+v", ending)
	}
}

func TestFixedAssetDepreciation(t *testing.T) {
	h := newHarness(t)
	s := app.NewSchedules(h.deps)
	in := app.AssetInput{Name: "Kamera", AcquiredOn: "2026-10-01", Cost: rp(50_000_000), LifeMonths: 48, ExpenseAccountID: h.id("6303001"), AccumAccountID: h.id("1201002")}
	_, err := s.CreateAsset(h.ctx, "u1", app.AssetInput{Name: "Salah", AcquiredOn: "2026-10-01", Cost: rp(10), Salvage: rp(10), LifeMonths: 12, ExpenseAccountID: in.ExpenseAccountID, AccumAccountID: in.AccumAccountID})
	if err == nil {
		t.Fatal("salvage equal to cost accepted")
	}
	a, err := s.CreateAsset(h.ctx, "u1", in)
	h.ok(err)

	// October has not ended on the pinned date.
	res, err := s.RunDue(h.ctx)
	h.ok(err)
	if res.Posted != 0 {
		t.Fatalf("too early %+v", res)
	}
	h.ok(s.DeleteAsset(h.ctx, a.ID))

	// September's month end has passed, but its period is closed.
	in.AcquiredOn = "2026-09-10"
	a, err = s.CreateAsset(h.ctx, "u1", in)
	h.ok(err)
	res, err = s.RunDue(h.ctx)
	h.ok(err)
	if res.Failed != 1 {
		t.Fatalf("closed period %+v", res)
	}
	a, err = h.deps.Store.GetFixedAsset(h.ctx, a.ID)
	h.ok(err)
	if a.DepreciatedMonths != 0 || a.LastError == "" {
		t.Fatalf("after failure %+v", a)
	}

	f := app.NewFiscal(h.deps)
	period, err := h.periodFor("2026-09")
	h.ok(err)
	_, err = f.OpenPeriod(h.ctx, period, true)
	h.ok(err)
	res, err = s.RunDue(h.ctx)
	h.ok(err)
	if res.Posted != 1 {
		t.Fatalf("after opening %+v", res)
	}
	a, err = h.deps.Store.GetFixedAsset(h.ctx, a.ID)
	h.ok(err)
	if a.DepreciatedMonths != 1 || a.LastError != "" || a.Accumulated() != a.DepreciationFor(1) {
		t.Fatalf("after run %+v", a)
	}
	e, err := app.NewJournals(h.deps).Get(h.ctx, a.LastEntryID)
	h.ok(err)
	if e.EntryDate != "2026-09-30" || e.Status != domain.StatusPosted || lineOn(e, a.AccumAccountID).Side != domain.Credit || lineOn(e, a.AccumAccountID).Amount != a.DepreciationFor(1) {
		t.Fatalf("journal %+v", e)
	}
	h.wantErr(s.DeleteAsset(h.ctx, a.ID), "Aset yang sudah disusutkan tidak bisa dihapus. Nonaktifkan saja.")
}

func TestSubsidiaryLedgerByAccount(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	first, second := h.id("4101001"), h.id("4101002")
	for _, c := range []struct {
		party, account string
		amount         int64
	}{{"A", first, 100}, {"B", first, 200}, {"A", second, 400}} {
		inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: "2026-10-02", PartyName: c.party, AccountID: c.account, Subtotal: rp(c.amount)})
		h.ok(err)
		_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
		h.ok(err)
	}
	_, err := l.SubsidiaryLedger(h.ctx, domain.KindAR, "", "", "", "")
	h.wantErr(err, "Pilih pihak atau akun COA")
	byAccount, err := l.SubsidiaryLedger(h.ctx, domain.KindAR, "", first, "", "")
	h.ok(err)
	if byAccount.TotalDebit != rp(300) || len(byAccount.Lines) != 2 {
		t.Fatalf("by account %+v", byAccount)
	}
	both, err := l.SubsidiaryLedger(h.ctx, domain.KindAR, "A", first, "", "")
	h.ok(err)
	if both.TotalDebit != rp(100) {
		t.Fatalf("party and account %+v", both)
	}
}

func TestReceiptDuplicates(t *testing.T) {
	h := newHarness(t)
	h.staff()
	r := app.NewReimbursements(h.deps)
	sekar := principal("u-sekar", authn.PermSelfService)
	taufik := principal("u-taufik", authn.PermSelfService)

	first := h.draft(r, sekar, domain.CategoryTransport, 100_000, "2026-10-20")
	if len(first.Duplicates) != 0 {
		t.Fatalf("first claim %+v", first.Duplicates)
	}
	// Another person sends the same file.
	second := h.draft(r, taufik, domain.CategoryTransport, 100_000, "2026-10-21")
	if len(second.Duplicates) != 1 || !second.Duplicates[0].Exact || second.Duplicates[0].ClaimNo != first.ClaimNo {
		t.Fatalf("duplicate %+v", second.Duplicates)
	}
	// A cancelled claim no longer counts.
	_, err := r.Cancel(h.ctx, sekar, first.ID)
	h.ok(err)
	again, err := r.Get(h.ctx, taufik, second.ID)
	h.ok(err)
	if len(again.Duplicates) != 0 {
		t.Fatalf("after cancel %+v", again.Duplicates)
	}
}

func TestClaimPreferredAccount(t *testing.T) {
	h := newHarness(t)
	h.staff()
	r := app.NewReimbursements(h.deps)
	sekar := principal("u-sekar", authn.PermSelfService)
	kas := h.id("1101001")
	_, err := r.Create(h.ctx, sekar, app.ClaimInput{Category: domain.CategoryTransport, TransactionDate: "2026-10-20", Amount: rp(1), Description: "x", PreferredCashAccountID: h.id("6301004")})
	h.wantErr(err, "Rekening tidak valid")
	c, err := r.Create(h.ctx, sekar, app.ClaimInput{Category: domain.CategoryTransport, TransactionDate: "2026-10-20", Amount: rp(1), Description: "x", PreferredCashAccountID: kas})
	h.ok(err)
	if c.PreferredCashAccountID != kas || c.PreferredCashAccountName == "" {
		t.Fatalf("preferred %+v", c)
	}
}

func TestBudgetVsActual(t *testing.T) {
	h := newHarness(t)
	r := app.NewReports(h.deps)
	j := app.NewJournals(h.deps)
	kas, beban, pendapatan := h.id("1101001"), h.id("6301004"), h.id("4101001")
	post := func(lines ...domain.Line) {
		e, err := j.Create(h.ctx, "u1", app.EntryInput{EntryDate: "2026-10-05", Lines: lines})
		h.ok(err)
		_, err = j.Post(h.ctx, "u1", e.ID)
		h.ok(err)
	}
	post(line(kas, domain.Debit, 900), line(pendapatan, domain.Credit, 900))
	post(line(beban, domain.Debit, 300), line(kas, domain.Credit, 300))

	h.ok(r.SaveBudgetYear(h.ctx, "u1", 2026, []domain.Budget{{Month: "2026-10", AccountID: pendapatan, Amount: rp(1000)}, {Month: "2026-10", AccountID: beban, Amount: rp(200)}}))
	err := r.SaveBudgetYear(h.ctx, "u1", 2026, []domain.Budget{{Month: "2027-01", AccountID: beban, Amount: rp(1)}})
	h.wantErr(err, "Bulan harus berada di tahun anggaran")
	err = r.SaveBudgetYear(h.ctx, "u1", 2026, []domain.Budget{{Month: "2026-10", AccountID: kas, Amount: rp(1)}})
	h.wantErr(err, "Anggaran hanya untuk akun pendapatan dan beban")

	// A mid-month range still compares the whole month.
	b, err := r.BudgetVsActual(h.ctx, "2026-10-10", "2026-10-20")
	h.ok(err)
	if b.From != "2026-10-01" || b.To != "2026-10-31" || b.BudgetRevenue != rp(1000) || b.ActualRevenue != rp(900) || b.BudgetCost != rp(200) || b.ActualCost != rp(300) {
		t.Fatalf("totals %+v", b)
	}
	for _, row := range b.Rows {
		if row.Variance != rp(-100) {
			t.Fatalf("both lines are 100 unfavorable: %+v", row)
		}
	}
	// Zero removes the row.
	h.ok(r.SaveBudgetYear(h.ctx, "u1", 2026, []domain.Budget{{Month: "2026-10", AccountID: beban, Amount: 0}}))
	year, err := r.BudgetYear(h.ctx, 2026)
	h.ok(err)
	if len(year) != 1 || year[0].AccountID != pendapatan {
		t.Fatalf("year %+v", year)
	}
}

func TestEquityChangesTiesToBalanceSheet(t *testing.T) {
	h := newHarness(t)
	r := app.NewReports(h.deps)
	j := app.NewJournals(h.deps)
	kas, modal, pendapatan := h.id("1101001"), h.id("3101001"), h.id("4101001")
	for _, c := range []struct {
		day   string
		lines []domain.Line
	}{
		{"2026-10-02", []domain.Line{line(kas, domain.Debit, 1000), line(modal, domain.Credit, 1000)}},
		{"2026-10-12", []domain.Line{line(kas, domain.Debit, 500), line(modal, domain.Credit, 500)}},
		{"2026-10-13", []domain.Line{line(kas, domain.Debit, 250), line(pendapatan, domain.Credit, 250)}},
	} {
		e, err := j.Create(h.ctx, "u1", app.EntryInput{EntryDate: c.day, Lines: c.lines})
		h.ok(err)
		_, err = j.Post(h.ctx, "u1", e.ID)
		h.ok(err)
	}
	eq, err := r.EquityChanges(h.ctx, "2026-10-10", "2026-10-20")
	h.ok(err)
	bs, err := r.BalanceSheet(h.ctx, "2026-10-20")
	h.ok(err)
	if eq.TotalClosing != bs.TotalEquity || eq.TotalOpening != rp(1000) || eq.TotalMovement != rp(750) {
		t.Fatalf("totals %+v", eq)
	}
	var movement domain.Money
	for _, row := range eq.Rows {
		movement += row.Movement
	}
	if movement != eq.TotalMovement {
		t.Fatalf("rows add up to %d, want %d", movement, eq.TotalMovement)
	}
}

func TestClientEconomics(t *testing.T) {
	h := newHarness(t)
	r := app.NewReports(h.deps)
	l := app.NewLedgers(h.deps)
	revenue := h.id("4101001")
	invoice := func(party string, amount int64) *domain.Invoice {
		inv, err := l.CreateInvoice(h.ctx, domain.KindAR, "u1", app.InvoiceInput{InvoiceDate: "2026-10-01", PartyName: party, AccountID: revenue, Subtotal: rp(amount)})
		h.ok(err)
		_, _, err = l.PostInvoice(h.ctx, domain.KindAR, "u1", inv.ID)
		h.ok(err)
		return inv
	}
	big := invoice("Besar", 750)
	invoice("Kecil", 250)
	_, err := l.CreatePayment(h.ctx, domain.KindAR, "u1", app.PaymentInput{PaymentDate: "2026-10-11", CashAccountID: h.id("1101001"), Allocations: []domain.Allocation{{InvoiceID: big.ID, Amount: rp(500)}}})
	h.ok(err)

	c, err := r.ClientEconomics(h.ctx, "2026-10-01", "2026-10-31")
	h.ok(err)
	if c.TotalInvoiced != rp(1000) || c.TotalCollected != rp(500) || c.AveragePerClient != rp(500) || c.CollectionRate != 50 || c.TopClientShare != 75 || c.TopThreeShare != 100 {
		t.Fatalf("report %+v", c)
	}
	if !c.HasPayments || c.AvgDaysToPay != 10 {
		t.Fatalf("days %+v", c)
	}
	if c.Rows[0].Party != "Besar" {
		t.Fatalf("order %+v", c.Rows)
	}
	if _, paid := c.Rows[1].AvgDaysToPay(); paid {
		t.Fatal("the small client has not paid")
	}
}
