package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

func errMsg(err error) string {
	if err == nil {
		return ""
	}
	if e := apperr.As(err); e != nil {
		return e.Message
	}
	return err.Error()
}

func TestMoney(t *testing.T) {
	cases := map[string]string{"": "0.00", "1250000": "1250000.00", "1250000.5": "1250000.50", " -10.25 ": "-10.25", "0.005": "0.01", ".5": "0.50", "12.344": "12.34"}
	for in, want := range cases {
		m, err := ParseMoney(in)
		if err != nil || m.String() != want {
			t.Errorf("ParseMoney(%q) = %v, %v; want %s", in, m, err, want)
		}
	}
	for _, in := range []string{"abc", "1e2", "1,000", "-", "1.2.3"} {
		if _, err := ParseMoney(in); err == nil {
			t.Errorf("ParseMoney(%q) should fail", in)
		}
	}
	if MustMoney("1500").Number() != "1500" || MustMoney("1499.5").Number() != "1499.5" || MustMoney("0").Number() != "0" {
		t.Fatal("Number")
	}
	var p struct{ A, B Money }
	if err := json.Unmarshal([]byte(`{"A":"10.50","B":2500}`), &p); err != nil || p.A != 1050 || p.B != 250000 {
		t.Fatalf("json %v %v", p, err)
	}
	if FormatRupiah(MustMoney("1250000")) != "1.250.000" || FormatRupiah(MustMoney("999.5")) != "999,50" {
		t.Fatal(FormatRupiah(MustMoney("1250000")))
	}
}

func TestAccountCodes(t *testing.T) {
	cases := []struct {
		raw, code, display string
		level              int
	}{
		{"1 1 01 001", "1101001", "1 1 01 001", 4},
		{"1101001", "1101001", "1 1 01 001", 4},
		{"1-1-01-000", "1101000", "1 1 01 000", 3},
		{"  2.1.00.000 ", "2100000", "2 1 00 000", 2},
		{"3000000", "3000000", "3 0 00 000", 1},
		{"11010", "", "11010", 0},
		{"abc", "", "abc", 0},
	}
	for _, c := range cases {
		if got := NormalizeAccountCode(c.raw); got != c.code {
			t.Errorf("NormalizeAccountCode(%q) = %q, want %q", c.raw, got, c.code)
		}
		if got := FormatAccountCodeDisplay(c.raw); got != c.display {
			t.Errorf("FormatAccountCodeDisplay(%q) = %q, want %q", c.raw, got, c.display)
		}
		if got := InferAccountLevel(c.raw); got != c.level {
			t.Errorf("InferAccountLevel(%q) = %d, want %d", c.raw, got, c.level)
		}
	}
}

func TestOpenSequenceViolation(t *testing.T) {
	if got := OpenSequenceViolation([]PeriodInput{{PeriodNo: 1, Status: "CLOSED"}, {PeriodNo: 2, Status: "OPEN"}}); got != "" {
		t.Fatalf("closed then open: %q", got)
	}
	got := OpenSequenceViolation([]PeriodInput{{PeriodNo: 2, Status: "OPEN"}, {PeriodNo: 1, Name: "Januari 2026", Status: "OPEN"}})
	want := "Tidak bisa OPEN period 2: period 1 (Januari 2026) belum CLOSED. Tutup period sebelumnya terlebih dahulu."
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestValidateFiscalYear(t *testing.T) {
	jan := PeriodInput{PeriodNo: 1, Name: "Jan", StartDate: "2026-01-01", EndDate: "2026-01-31", Status: "OPEN"}
	feb := PeriodInput{PeriodNo: 2, Name: "Feb", StartDate: "2026-02-01", EndDate: "2026-02-28", Status: "CLOSED"}
	cases := []struct {
		start, end string
		periods    []PeriodInput
		want       string
	}{
		{"2026-01-01", "2026-12-31", []PeriodInput{jan, feb}, ""},
		{"2026-12-31", "2026-01-01", []PeriodInput{jan}, "end_date harus >= start_date"},
		{"2026-01-01", "2026-12-31", []PeriodInput{jan, jan}, "period_no 1 duplikat"},
		{"2026-02-01", "2026-12-31", []PeriodInput{jan}, "Period 1 harus berada dalam rentang fiscal year"},
		{"2026-01-01", "2026-12-31", []PeriodInput{{PeriodNo: 3, StartDate: "2026-03-31", EndDate: "2026-03-01", Status: "OPEN"}}, "Period 3: end_date < start_date"},
		{"2026-01-01", "2026-12-31", []PeriodInput{jan, {PeriodNo: 2, Name: "Feb", StartDate: "2026-02-01", EndDate: "2026-02-28", Status: "OPEN"}},
			"Tidak bisa OPEN period 2: period 1 (Jan) belum CLOSED. Tutup period sebelumnya terlebih dahulu."},
	}
	for i, c := range cases {
		if got := ValidateFiscalYear(c.start, c.end, c.periods); got != c.want {
			t.Errorf("case %d: got %q, want %q", i, got, c.want)
		}
	}
	if b := CloseBlockers("Jan", "CLOSED", false, 2); len(b) != 3 || b[2] != "Masih ada 2 jurnal DRAFT. Posting atau hapus dulu sebelum closing." {
		t.Fatalf("blockers %v", b)
	}
	ps := MonthlyPeriods(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC))
	if len(ps) != 12 || ps[1].EndDate != "2026-02-28" || ps[0].Status != PeriodOpen || ps[11].Name != "Desember 2026" {
		t.Fatalf("monthly %v", ps)
	}
	if msg := ValidateFiscalYear("2026-01-01", "2026-12-31", ps); msg != "" {
		t.Fatal(msg)
	}
}

func line(side string, amount string) Line {
	return Line{AccountID: "a", Side: side, Amount: MustMoney(amount)}
}

func TestValidateBalanced(t *testing.T) {
	cases := []struct {
		name  string
		lines []Line
		want  string
	}{
		{"balanced", []Line{line(Debit, "100.10"), line(Credit, "50.05"), line(Credit, "50.05")}, ""},
		{"one line", []Line{line(Debit, "1")}, "Minimal 2 baris jurnal (debit dan credit)"},
		{"no credit", []Line{line(Debit, "1"), line(Debit, "1")}, "Harus ada minimal satu baris Debit dan satu baris Credit"},
		{"zero", []Line{line(Debit, "0"), line(Credit, "0")}, "Amount harus lebih dari 0"},
		{"bad side", []Line{line("X", "1"), line(Credit, "1")}, "entry_side harus DEBIT atau CREDIT"},
		{"no account", []Line{{Side: Debit, Amount: 100}, line(Credit, "1")}, "Setiap baris harus punya akun COA"},
		{"unbalanced", []Line{line(Debit, "1500"), line(Credit, "1499.5")}, "Jurnal tidak balance: Debit 1500 ≠ Credit 1499.5"},
	}
	for _, c := range cases {
		if got := errMsg(ValidateBalanced(c.lines)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestJournalStateMachine(t *testing.T) {
	mut := []struct {
		typ, status string
		recon       bool
		want        string
	}{
		{EntryManual, StatusDraft, false, ""},
		{EntryOpening, StatusDraft, false, "Beginning balance (OPENING) dikelola lewat menu Beginning Balance"},
		{EntryAuto, StatusDraft, false, "Journal entry AUTO dari modul operasional tidak bisa diubah dari sini"},
		{EntryManual, StatusPosted, false, "Journal entry POSTED tidak bisa diubah/dihapus"},
		{EntryManual, StatusDraft, true, "Journal entry dengan flag recon tidak bisa diubah/dihapus"},
	}
	for _, c := range mut {
		if got := errMsg(AssertMutable(c.typ, c.status, c.recon)); got != c.want {
			t.Errorf("AssertMutable(%s,%s,%v) = %q", c.typ, c.status, c.recon, got)
		}
	}
	post := []struct {
		status string
		lines  int
		d, c   Money
		want   string
	}{
		{StatusDraft, 2, 10, 10, ""},
		{StatusPosted, 2, 10, 10, "Journal entry sudah POSTED"},
		{StatusDraft, 1, 10, 10, "Minimal 2 baris jurnal sebelum post"},
		{StatusDraft, 2, 10, 9, "Jurnal tidak balance"},
	}
	for _, c := range post {
		if got := errMsg(AssertPostable(c.status, c.lines, c.d, c.c)); got != c.want {
			t.Errorf("AssertPostable(%v) = %q", c, got)
		}
	}
	if !CanEdit(StatusDraft, false, EntryManual) || CanEdit(StatusDraft, false, EntryAuto) || CanEdit(StatusPosted, false, EntryManual) {
		t.Fatal("CanEdit")
	}
	if NormalizeEntryType("weird") != EntryManual || NormalizeEntryType(EntryAuto) != EntryAuto {
		t.Fatal("NormalizeEntryType")
	}
}

func TestCheckPostableAccounts(t *testing.T) {
	ok := AccountCheck{ID: "a", IsPostable: true}
	cases := []struct {
		want int
		rows []AccountCheck
		msg  string
	}{
		{1, []AccountCheck{ok}, ""},
		{2, []AccountCheck{ok}, "Satu atau lebih akun COA tidak ditemukan"},
		{1, []AccountCheck{{ID: "a", IsPostable: true, Deleted: true}}, "Akun COA sudah dihapus"},
		{1, []AccountCheck{{ID: "a"}}, "header"},
	}
	for i, c := range cases {
		if got := errMsg(CheckPostableAccounts(c.want, c.rows, "header")); got != c.msg {
			t.Errorf("case %d: %q", i, got)
		}
	}
	if err := CheckPostableAccounts(2, nil, ""); !apperr.Is(err, apperr.NotFound) {
		t.Fatal("missing account must be NOT_FOUND")
	}
}

func TestNumbering(t *testing.T) {
	if got := SequenceNo(EntryNoPrefix("2026-10-26"), 42); got != "JE-202610-0042" {
		t.Fatal(got)
	}
	if got := SequenceNo(OpeningNoPrefix("2026-01-01"), 1); got != "OB-2026-0001" {
		t.Fatal(got)
	}
	if got := SequenceNo(DailyNoPrefix("APP", time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)), 3); got != "APP-20261026-0003" {
		t.Fatal(got)
	}
}

func ptr[T any](v T) *T { return &v }

func TestMappingPosting(t *testing.T) {
	if AmountFromSource("TOTAL", Amounts{"TOTAL": 1200, "SUBTOTAL": 1000}) != 1200 || AmountFromSource("PAID", Amounts{"PAID": 500}) != 500 {
		t.Fatal("reads by source")
	}
	if AmountFromSource("TAX", Amounts{"TOTAL": 10}) != 0 || AmountFromSource("TOTAL", Amounts{"TOTAL": -5}) != 0 {
		t.Fatal("non-positive is 0")
	}
	if !HasRequiredAccounts([]MappingLine{{AccountID: "a", IsRequired: true}, {IsRequired: false}}) ||
		HasRequiredAccounts([]MappingLine{{IsRequired: true}, {AccountID: "b", IsRequired: true}}) {
		t.Fatal("required accounts")
	}
	base := []MappingLine{
		{Side: Debit, AccountID: "inv", AmountSource: "TOTAL", SortOrder: 10, IsRequired: true, LineRole: "INVENTORY"},
		{Side: Credit, AccountID: "grni", AmountSource: "TOTAL", SortOrder: 20, IsRequired: true, LineRole: "GRNI"},
	}
	lines, reason := BuildFromMapping(base, Amounts{"TOTAL": MustMoney("1500")}, "", nil)
	if reason != "" || len(lines) != 2 || lines[0].Amount != MustMoney("1500") || lines[1].SortOrder != 20 {
		t.Fatalf("balanced: %v %q", lines, reason)
	}
	if _, reason := BuildFromMapping(base, Amounts{"SUBTOTAL": 10}, "", nil); reason != "Amount source TOTAL untuk role INVENTORY bernilai 0" {
		t.Fatal(reason)
	}
	missing := []MappingLine{{Side: Debit, AmountSource: "TOTAL", IsRequired: true, LineRole: "INVENTORY"}, base[1]}
	if _, reason := BuildFromMapping(missing, Amounts{"TOTAL": 100}, "", nil); reason != "Journal mapping belum lengkap (akun COA wajib belum diisi)" {
		t.Fatal(reason)
	}
	discount := []MappingLine{
		{Side: Debit, AccountID: "cash", AmountSource: "TOTAL", SortOrder: 10, IsRequired: true, LineRole: "CASH"},
		{Side: Debit, AmountSource: "DISCOUNT", SortOrder: 15, LineRole: "DISCOUNT"},
		{Side: Credit, AccountID: "rev", AmountSource: "SUBTOTAL", SortOrder: 20, IsRequired: true, LineRole: "REVENUE"},
	}
	if _, reason := BuildFromMapping(discount, Amounts{"TOTAL": MustMoney("36000"), "DISCOUNT": MustMoney("4000"), "SUBTOTAL": MustMoney("40000")}, "", nil); reason != "Akun COA untuk role DISCOUNT belum diisi (nilai DISCOUNT=4000)" {
		t.Fatal(reason)
	}
	// Overrides fill a role; splits post one line per account and merge.
	reimb := []MappingLine{
		{Side: Debit, AccountID: "exp-default", AmountSource: "TOTAL", SortOrder: 10, IsRequired: true, LineRole: "EXPENSE"},
		{Side: Credit, AccountID: "kas", AmountSource: "PAID", SortOrder: 20, IsRequired: true, LineRole: "CASH"},
	}
	reimb = WithAccountOverrides(reimb, map[string]string{"CASH": "bca"})
	lines, reason = BuildFromMapping(reimb, Amounts{"PAID": 600}, "m", map[string][]Split{"EXPENSE": {{AccountID: "transport", Amount: 100}, {Amount: 200}, {AccountID: "transport", Amount: 300}}})
	if reason != "" || len(lines) != 3 || lines[0].AccountID != "transport" || lines[0].Amount != 400 || lines[1].AccountID != "exp-default" || lines[2].AccountID != "bca" {
		t.Fatalf("splits: %+v %q", lines, reason)
	}
	if got := Summarize(PostResult{Status: PostPosted}, PostResult{Status: PostPosted}, PostResult{Status: PostSkipped, Reason: AlreadyExists}, PostResult{Status: PostDraft}); got != "jurnal posted; jurnal draft: mapping belum lengkap" {
		t.Fatal(got)
	}
	if got := Summarize(PostResult{Status: PostSkipped, Reason: AlreadyExists}); got != "" {
		t.Fatal(got)
	}
}

func TestPayrollJournals(t *testing.T) {
	p := PayrollPaid{PeriodYear: 2026, PeriodMonth: 3, TotalGross: 1000, TotalNet: 800, TotalPph21: 100, TotalBpjsTkEmployee: 100, TotalBpjsKesEmployer: 40}
	js := PayrollJournals(p)
	if len(js) != 5 || js[0].Event != EventPayrollAccrual || js[3].Event != EventPayrollPayment || js[4].Event != EventPayrollBpjsKesEmployer {
		t.Fatalf("%+v", js)
	}
	if PayrollDescription(p, "beban gaji") != "Payroll 03/2026: beban gaji" {
		t.Fatal(PayrollDescription(p, "beban gaji"))
	}
}

func TestReceivableStatus(t *testing.T) {
	if InvoiceOutstanding(1000, 250) != 750 || InvoiceOutstanding(100, 100) != 0 || InvoiceOutstanding(50, 80) != 0 {
		t.Fatal("outstanding")
	}
	today := "2026-08-12"
	cases := []struct {
		total, alloc Money
		due          string
		want         string
	}{
		{1000, 1000, "2026-01-01", "paid"},
		{1000, 400, "2026-12-01", "partial"},
		{1000, 0, "2026-01-01", "overdue"},
		{1000, 100, "2026-01-01", "overdue"},
		{1000, 0, "2026-12-01", "unpaid"},
		{1000, 0, "", "unpaid"},
	}
	for _, c := range cases {
		if got := PaymentStatus(c.total, c.alloc, c.due, today); got != c.want {
			t.Errorf("%v: %s", c, got)
		}
	}
	if BucketAging(0, "2026-01-01", today) != "" {
		t.Fatal("no outstanding")
	}
	buckets := map[string]string{"2026-08-20": "current", "2026-08-12": "current", "2026-07-20": "1_30", "2026-06-20": "31_60", "2026-05-20": "61_90", "2026-01-01": "90_plus", "": "current"}
	for due, want := range buckets {
		if got := BucketAging(100, due, today); got != want {
			t.Errorf("BucketAging(%s) = %s, want %s", due, got, want)
		}
	}
	if DaysPastDue("2026-08-01", today) != 11 || DaysPastDue("2026-08-20", today) != -8 {
		t.Fatal("daysPastDue")
	}
}

func TestPaymentVoid(t *testing.T) {
	cases := []struct {
		found          bool
		status, reason string
		want           apperr.Code
	}{
		{false, "", "alasan panjang", apperr.NotFound},
		{true, StatusPosted, " abc ", apperr.Validation},
		{true, StatusVoid, "salah input", apperr.Conflict},
		{true, StatusDraft, "salah input", apperr.Conflict},
		{true, StatusPosted, "salah input", ""},
	}
	for _, c := range cases {
		err := EvaluatePaymentVoid(c.found, c.status, c.reason, "AP")
		var got apperr.Code
		if e := apperr.As(err); e != nil {
			got = e.Code
		}
		if got != c.want {
			t.Errorf("%v: %s", c, got)
		}
	}
}

func TestReportBalances(t *testing.T) {
	if Balance(100, 40, Debit, false) != 60 || Balance(100, 40, Credit, false) != -60 || Balance(100, 40, Debit, true) != -60 {
		t.Fatal("Balance")
	}
	if SignedDelta(Debit, 10, Debit, false) != 10 || SignedDelta(Credit, 10, Debit, false) != -10 || SignedDelta(Credit, 10, Credit, true) != -10 {
		t.Fatal("SignedDelta")
	}
	if d, c := TrialBalanceColumns(10, 30); d != 0 || c != 20 {
		t.Fatal("TB")
	}
	if side, amt, ok := OpeningLine(5, 5); ok || side != "" || amt != 0 {
		t.Fatal("OpeningLine zero")
	}
	if side, amt, _ := OpeningLine(5, 15); side != Credit || amt != 10 {
		t.Fatal("OpeningLine credit")
	}
	if EarningsContribution(TypeRevenue, 10) != 10 || EarningsContribution(TypeExpense, 10) != -10 || CashFlowContribution(TypeLiability, 5) != 5 || CashFlowContribution(TypeAsset, 5) != -5 {
		t.Fatal("contributions")
	}
}

func TestCoaImport(t *testing.T) {
	rows, issues := ParseCoaMatrix([][]string{
		{"Kode Akun", "Nama Akun", "Tipe"},
		{"1 0 00 000", "ASET", ""},
		{"1 1 00 000", "Aset Lancar", ""},
		{"1 1 01 000", "Kas & Bank", ""},
		{"1 1 01 001", "Kas Kecil", ""},
		{"1 1 01 001", "Duplikat", ""},
		{"4 1 01 001", "Pendapatan Jasa", ""},
		{"x", "Rusak", ""},
	})
	if len(rows) != 5 || len(issues) != 2 {
		t.Fatalf("rows %d issues %v", len(rows), issues)
	}
	kas := rows[3]
	if kas.ParentCode != "1101000" || !kas.IsCashBank || kas.AccountTypeCode != TypeAsset || kas.CashFlowCategory != "OPERATING" {
		t.Fatalf("kas %+v", kas)
	}
	if rows[4].ParentCode != "" || rows[4].AccountTypeCode != TypeRevenue {
		t.Fatalf("revenue %+v", rows[4])
	}
	sulu, _ := ParseCoaMatrix([][]string{{"1101001", "Cash on hand"}, {"8301001", "", "Rounding Gain"}})
	if len(sulu) != 2 || sulu[1].Code != "8201003" || sulu[1].AccountTypeCode != TypeOtherIncome {
		t.Fatalf("sulu %+v", sulu)
	}
	types := map[string]string{TypeAsset: "at-asset", TypeRevenue: "at-revenue"}
	preview := BuildCoaImportPreview(rows, types, map[string]ExistingAccount{"1101001": {ID: "x", Name: "Kas Kecil"}, "1000000": {ID: "y", Name: "Aset"}})
	s := SummarizeCoaImport(preview, issues)
	if s.Create != 3 || s.Update != 1 || s.Skip != 1 || s.Error != 2 {
		t.Fatalf("summary %+v %+v", s, preview)
	}
}

func TestClaimRules(t *testing.T) {
	today := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	if CheckClaimAge(today.AddDate(0, 0, -30), today) != nil {
		t.Fatal("30 days is allowed")
	}
	if errMsg(CheckClaimAge(today.AddDate(0, 0, -31), today)) != MsgClaimTooOld {
		t.Fatal("31 days rejected")
	}
	if errMsg(CheckClaimAge(today.AddDate(0, 0, 1), today)) != MsgFutureDate {
		t.Fatal("future")
	}
	p := &Policy{IsActive: true, PerClaimLimit: Rupiah(500_000), MonthlyLimit: Rupiah(1_000_000)}
	if over, _ := EvaluateLimit(p, Rupiah(400_000), Rupiah(500_000)); over {
		t.Fatal("within limits")
	}
	over, note := EvaluateLimit(p, Rupiah(600_000), Rupiah(500_000))
	if !over || note != "Klaim melebihi batas per klaim Rp500.000; total bulan ini Rp1.100.000 melebihi batas bulanan Rp1.000.000." {
		t.Fatal(note)
	}
	if over, _ := EvaluateLimit(&Policy{}, Rupiah(9_000_000), 0); over {
		t.Fatal("inactive policy")
	}
	if NextApproval(ClaimSubmitted) != ClaimApprovedManager || NextApproval(ClaimApprovedManager) != ClaimApprovedFinance || NextApproval(ClaimDraft) != "" {
		t.Fatal("approvals")
	}
	if !CanCancel(ClaimSubmitted) || CanCancel(ClaimApprovedManager) || !CanReject(ClaimApprovedManager) || CanReject(ClaimPaid) {
		t.Fatal("cancel/reject")
	}
	if m, e := SniffReceipt([]byte("%PDF-1.7 ...")); m != "application/pdf" || e != "pdf" {
		t.Fatal("pdf")
	}
	if m, _ := SniffReceipt([]byte("GIF89a......")); m != "" {
		t.Fatal("gif is not accepted")
	}
	if CleanFilename("../../etc/passwd") != "passwd" || CleanFilename("") != "bukti" || !IsSafeKey("claims/abc.pdf") || IsSafeKey("../x/y") {
		t.Fatal("filenames")
	}
}

func TestRatios(t *testing.T) {
	s := LedgerSnapshot{
		Days: 30, Revenue: Rupiah(100_000_000), COGS: Rupiah(10_000_000), Expenses: Rupiah(60_000_000),
		OtherIncome: Rupiah(1_000_000), OtherExpenses: Rupiah(1_000_000),
		CurrentAssets: Rupiah(300_000_000), CurrentLiabilities: Rupiah(100_000_000), Cash: Rupiah(200_000_000),
		Receivables: Rupiah(50_000_000), TradePayables: Rupiah(14_000_000), TotalLiabilities: Rupiah(100_000_000),
		TotalEquity: Rupiah(250_000_000), CashOpening: Rupiah(230_437_500),
	}
	r := ComputeRatios(s)
	near := func(a, b float64) bool { return a-b < 0.001 && b-a < 0.001 }
	if !near(r.GrossMargin.Value, 90) || !near(r.OperatingMargin.Value, 30) || !near(r.NetMargin.Value, 30) {
		t.Fatalf("margins %+v", r)
	}
	if !near(r.CurrentRatio.Value, 3) || !near(r.QuickRatio.Value, 2.5) || !near(r.CashRatio.Value, 2) || !near(r.DebtToEquity.Value, 0.4) {
		t.Fatalf("liquidity %+v", r)
	}
	if !near(r.DSO.Value, 15) || !near(r.DPO.Value, 6) {
		t.Fatalf("efficiency %+v", r)
	}
	// 30,437,500 fell over 30 days = 30,437,500 / (30/30.4375) per month.
	if r.BurnRate != MustMoney("30881380.21") || !r.Runway.OK || !near(r.Runway.Value, 200_000_000/30_881_380.21) {
		t.Fatalf("burn %v runway %+v", r.BurnRate, r.Runway)
	}
	if empty := ComputeRatios(LedgerSnapshot{Days: 30}); empty.GrossMargin.OK || empty.CurrentRatio.OK || empty.Runway.OK {
		t.Fatal("zero denominators are unavailable")
	}
}
