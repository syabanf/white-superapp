package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Pure accounting rules ported from NüHabit accounting/domain (account
// codes, journal validation and state machine, fiscal sequencing, report
// balances, AP/AR status and aging). Messages are kept verbatim except that
// em dashes became colons or commas.

// Bad, NotFound and Conflict build the user-facing errors of the rules.
func Bad(msg string) *apperr.Error      { return apperr.New(apperr.Validation, msg) }
func NotFound(msg string) *apperr.Error { return apperr.New(apperr.NotFound, msg) }
func Conflict(msg string) *apperr.Error { return apperr.New(apperr.Conflict, msg) }

// ─── Account codes (accountcode.go) ──────────────────────────

var (
	segmentPattern = regexp.MustCompile(`^(\d) (\d) (\d{2}) (\d{3})$`)
	compactPattern = regexp.MustCompile(`^(\d)(\d)(\d{2})(\d{3})$`)
)

func isSpace(r rune) bool { return unicode.IsSpace(r) || r == 0xFEFF }

func collapseSpaces(s string) string {
	var b strings.Builder
	inSpace := false
	for _, r := range s {
		if isSpace(r) {
			if !inSpace {
				b.WriteByte(' ')
			}
			inSpace = true
			continue
		}
		inSpace = false
		b.WriteRune(r)
	}
	return b.String()
}

// NormalizeAccountCode returns the compact 7-digit code ("1 1 01 001" ->
// "1101001"), or "" when raw is not a recognised format. Separators - _ .
// are accepted.
func NormalizeAccountCode(raw string) string {
	trimmed := strings.TrimFunc(raw, isSpace)
	trimmed = strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(trimmed)
	trimmed = collapseSpaces(trimmed)
	if trimmed == "" {
		return ""
	}
	if m := segmentPattern.FindStringSubmatch(trimmed); m != nil {
		return m[1] + m[2] + m[3] + m[4]
	}
	digits := strings.ReplaceAll(trimmed, " ", "")
	if compactPattern.MatchString(digits) {
		return digits
	}
	return ""
}

type codeParts struct{ class, group, sub, detail string }

func parseAccountCode(code string) (codeParts, bool) {
	m := compactPattern.FindStringSubmatch(NormalizeAccountCode(code))
	if m == nil {
		return codeParts{}, false
	}
	return codeParts{m[1], m[2], m[3], m[4]}, true
}

// FormatAccountCodeDisplay is "1 1 01 001"; unparseable codes come back unchanged.
func FormatAccountCodeDisplay(code string) string {
	p, ok := parseAccountCode(code)
	if !ok {
		return code
	}
	return p.class + " " + p.group + " " + p.sub + " " + p.detail
}

// InferAccountLevel is 1..4 from the trailing zeros, 0 when unrecognised.
func InferAccountLevel(code string) int {
	p, ok := parseAccountCode(code)
	switch {
	case !ok:
		return 0
	case p.group == "0" && p.sub == "00" && p.detail == "000":
		return 1
	case p.sub == "00" && p.detail == "000":
		return 2
	case p.detail == "000":
		return 3
	}
	return 4
}

// ResolveAccountCode validates a code from a form.
func ResolveAccountCode(raw string) (string, int, error) {
	code := NormalizeAccountCode(raw)
	if code == "" {
		return "", 0, Bad("Format kode akun tidak valid")
	}
	level := InferAccountLevel(code)
	if level == 0 {
		return "", 0, Bad("Level kode akun tidak dikenali")
	}
	return code, level, nil
}

// ─── Journals (journal.go) ───────────────────────────────────

const (
	Debit  = "DEBIT"
	Credit = "CREDIT"

	StatusDraft  = "DRAFT"
	StatusPosted = "POSTED"
	StatusVoid   = "VOID"

	EntryManual  = "MANUAL"
	EntryOpening = "OPENING"
	EntryAuto    = "AUTO"
)

// Line is one journal line to write.
type Line struct {
	AccountID string
	Side      string
	Amount    Money
	Memo      string
	SortOrder int
}

// ValidateBalanced: at least two lines, both sides present, every amount
// > 0, debit = credit.
func ValidateBalanced(lines []Line) error {
	if len(lines) < 2 {
		return Bad("Minimal 2 baris jurnal (debit dan credit)")
	}
	var debit, credit Money
	var hasDebit, hasCredit bool
	for _, l := range lines {
		if l.AccountID == "" {
			return Bad("Setiap baris harus punya akun COA")
		}
		if l.Side != Debit && l.Side != Credit {
			return Bad("entry_side harus DEBIT atau CREDIT")
		}
		if l.Amount <= 0 {
			return Bad("Amount harus lebih dari 0")
		}
		if l.Side == Debit {
			debit += l.Amount
			hasDebit = true
		} else {
			credit += l.Amount
			hasCredit = true
		}
	}
	if !hasDebit || !hasCredit {
		return Bad("Harus ada minimal satu baris Debit dan satu baris Credit")
	}
	if debit != credit {
		return Bad("Jurnal tidak balance: Debit " + debit.Number() + " ≠ Credit " + credit.Number())
	}
	return nil
}

// NormalizeEntryType maps a stored entry_type to MANUAL/OPENING/AUTO.
func NormalizeEntryType(t string) string {
	switch t {
	case EntryOpening, EntryAuto:
		return t
	}
	return EntryManual
}

// CanEdit: only manual, non-recon drafts are editable from the form.
func CanEdit(status string, isRecon bool, entryType string) bool {
	return status == StatusDraft && !isRecon && entryType == EntryManual
}

// AssertMutable guards update and delete of a journal entry.
func AssertMutable(entryType, status string, isRecon bool) error {
	switch {
	case entryType == EntryOpening:
		return Bad("Beginning balance (OPENING) dikelola lewat menu Beginning Balance")
	case entryType == EntryAuto:
		return Bad("Journal entry AUTO dari modul operasional tidak bisa diubah dari sini")
	case status == StatusPosted:
		return Bad("Journal entry POSTED tidak bisa diubah/dihapus")
	case isRecon:
		return Bad("Journal entry dengan flag recon tidak bisa diubah/dihapus")
	}
	return nil
}

// AssertPostable guards DRAFT -> POSTED.
func AssertPostable(status string, lineCount int, totalDebit, totalCredit Money) error {
	switch {
	case status == StatusPosted:
		return Bad("Journal entry sudah POSTED")
	case lineCount < 2:
		return Bad("Minimal 2 baris jurnal sebelum post")
	case totalDebit != totalCredit:
		return Bad("Jurnal tidak balance")
	}
	return nil
}

// AccountCheck is what the postable-accounts guard reads per account.
type AccountCheck struct {
	ID         string
	IsPostable bool
	Deleted    bool
}

// CheckPostableAccounts: all found, not deleted, postable. headerMsg
// differs between journal entries and mappings.
func CheckPostableAccounts(want int, rows []AccountCheck, headerMsg string) error {
	if len(rows) != want {
		return NotFound("Satu atau lebih akun COA tidak ditemukan")
	}
	for _, r := range rows {
		if r.Deleted {
			return Bad("Akun COA sudah dihapus")
		}
		if !r.IsPostable {
			return Bad(headerMsg)
		}
	}
	return nil
}

// UniqueIDs drops empty and duplicate ids, keeping first-seen order.
func UniqueIDs(ids []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// SequenceNo formats "<prefix>NNNN" (padded to 4 digits).
func SequenceNo(prefix string, n int) string { return fmt.Sprintf("%s%04d", prefix, n) }

// EntryNoPrefix is "JE-YYYYMM-" for a YYYY-MM-DD date.
func EntryNoPrefix(entryDate string) string {
	ym := entryDate
	if len(ym) > 7 {
		ym = ym[:7]
	}
	return "JE-" + strings.Replace(ym, "-", "", 1) + "-"
}

// OpeningNoPrefix is "OB-YYYY-".
func OpeningNoPrefix(startDate string) string {
	y := startDate
	if len(y) > 4 {
		y = y[:4]
	}
	return "OB-" + y + "-"
}

// DailyNoPrefix is "<PREFIX>-YYYYMMDD-" for AP/AR documents.
func DailyNoPrefix(prefix string, day time.Time) string {
	return prefix + "-" + day.Format("20060102") + "-"
}

// Totals sums the debit and credit sides.
func Totals(lines []Line) (debit, credit Money) {
	for _, l := range lines {
		switch l.Side {
		case Debit:
			debit += l.Amount
		case Credit:
			credit += l.Amount
		}
	}
	return debit, credit
}

// ReverseLines flips every side for a reversing journal.
func ReverseLines(lines []Line, memo string) []Line {
	out := make([]Line, len(lines))
	for i, l := range lines {
		side := Debit
		if l.Side == Debit {
			side = Credit
		}
		out[i] = Line{AccountID: l.AccountID, Side: side, Amount: l.Amount, Memo: memo, SortOrder: l.SortOrder}
	}
	return out
}

// ─── Fiscal (fiscal.go) ──────────────────────────────────────

const (
	PeriodOpen   = "OPEN"
	PeriodClosed = "CLOSED"
)

// PeriodInput is one period of a fiscal year payload.
type PeriodInput struct {
	PeriodNo  int
	Name      string
	StartDate string // YYYY-MM-DD
	EndDate   string
	Status    string
}

// OpenSequenceViolation: period N may be OPEN only when every earlier
// period is CLOSED. Returns the first violation, or "".
func OpenSequenceViolation(periods []PeriodInput) string {
	sorted := append([]PeriodInput(nil), periods...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].PeriodNo < sorted[j].PeriodNo })
	for _, p := range sorted {
		if p.Status != PeriodOpen {
			continue
		}
		var prev *PeriodInput
		for i := range sorted {
			if sorted[i].PeriodNo < p.PeriodNo && sorted[i].Status == PeriodOpen {
				prev = &sorted[i]
			}
		}
		if prev != nil {
			name := ""
			if prev.Name != "" {
				name = " (" + prev.Name + ")"
			}
			return fmt.Sprintf("Tidak bisa OPEN period %d: period %d%s belum CLOSED. Tutup period sebelumnya terlebih dahulu.",
				p.PeriodNo, prev.PeriodNo, name)
		}
	}
	return ""
}

// ValidateFiscalYear returns the 400 message, or "".
func ValidateFiscalYear(start, end string, periods []PeriodInput) string {
	if end < start {
		return "end_date harus >= start_date"
	}
	if len(periods) < 1 || len(periods) > 12 {
		return "Fiscal year harus punya 1-12 period"
	}
	seen := map[int]bool{}
	for _, p := range periods {
		if p.PeriodNo < 1 || p.PeriodNo > 12 {
			return "period_no harus antara 1-12"
		}
		if seen[p.PeriodNo] {
			return fmt.Sprintf("period_no %d duplikat", p.PeriodNo)
		}
		seen[p.PeriodNo] = true
		if p.EndDate < p.StartDate {
			return fmt.Sprintf("Period %d: end_date < start_date", p.PeriodNo)
		}
		if p.StartDate < start || p.EndDate > end {
			return fmt.Sprintf("Period %d harus berada dalam rentang fiscal year", p.PeriodNo)
		}
		if p.Status != PeriodOpen && p.Status != PeriodClosed {
			return fmt.Sprintf("Period %d: status tidak valid", p.PeriodNo)
		}
	}
	return OpenSequenceViolation(periods)
}

// HasOpenPeriod reports whether any period is OPEN.
func HasOpenPeriod(periods []PeriodInput) bool {
	for _, p := range periods {
		if p.Status == PeriodOpen {
			return true
		}
	}
	return false
}

// CloseBlockers lists why a period cannot be closed.
func CloseBlockers(periodName, status string, yearActive bool, draftCount int) []string {
	blockers := []string{}
	if status == PeriodClosed {
		blockers = append(blockers, "Period "+periodName+" sudah CLOSED")
	}
	if !yearActive {
		blockers = append(blockers, "Fiscal year tidak aktif")
	}
	if draftCount > 0 {
		blockers = append(blockers, fmt.Sprintf("Masih ada %d jurnal DRAFT. Posting atau hapus dulu sebelum closing.", draftCount))
	}
	return blockers
}

var monthNames = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// MonthlyPeriods splits [start, end] into calendar-month periods (at most
// 12). The first period is OPEN, the rest CLOSED, which satisfies the open
// sequence rule.
func MonthlyPeriods(start, end time.Time) []PeriodInput {
	var out []PeriodInput
	cur := start
	for n := 1; n <= 12 && !cur.After(end); n++ {
		last := time.Date(cur.Year(), cur.Month()+1, 0, 0, 0, 0, 0, time.UTC)
		if last.After(end) {
			last = end
		}
		status := PeriodClosed
		if n == 1 {
			status = PeriodOpen
		}
		out = append(out, PeriodInput{
			PeriodNo: n, Name: fmt.Sprintf("%s %d", monthNames[cur.Month()-1], cur.Year()),
			StartDate: cur.Format(time.DateOnly), EndDate: last.Format(time.DateOnly), Status: status,
		})
		cur = last.AddDate(0, 0, 1)
	}
	return out
}

// ─── Report balances (reports.go) ────────────────────────────

// Account type codes.
const (
	TypeAsset        = "ASSET"
	TypeLiability    = "LIABILITY"
	TypeEquity       = "EQUITY"
	TypeRevenue      = "REVENUE"
	TypeCOGS         = "COGS"
	TypeExpense      = "EXPENSE"
	TypeOtherIncome  = "OTHER_INCOME"
	TypeOtherExpense = "OTHER_EXPENSE"
)

// AsNormal maps a stored normal_balance to DEBIT/CREDIT.
func AsNormal(v string) string {
	if v == Credit {
		return Credit
	}
	return Debit
}

// EffectiveNormal is the side an account grows on: its type's normal
// balance, flipped for contra accounts (accumulated depreciation is CREDIT).
func EffectiveNormal(normal string, isContra bool) string {
	if !isContra {
		return normal
	}
	if normal == Credit {
		return Debit
	}
	return Credit
}

// Balance is signed by the normal side; contra accounts flip.
func Balance(debit, credit Money, normal string, isContra bool) Money {
	raw := debit - credit
	if normal != Debit {
		raw = credit - debit
	}
	if isContra {
		raw = -raw
	}
	return raw
}

// SignedDelta is one line's effect on a running balance.
func SignedDelta(side string, amount Money, normal string, isContra bool) Money {
	increase := (normal == Debit && side == Debit) || (normal == Credit && side == Credit)
	delta := -amount
	if increase {
		delta = amount
	}
	if isContra {
		delta = -delta
	}
	return delta
}

// TrialBalanceColumns puts the net in the debit or the credit column.
func TrialBalanceColumns(debit, credit Money) (Money, Money) {
	net := debit - credit
	switch {
	case net > 0:
		return net, 0
	case net < 0:
		return 0, -net
	}
	return 0, 0
}

// IsProfitLossType reports the P&L account types.
func IsProfitLossType(code string) bool {
	switch code {
	case TypeRevenue, TypeCOGS, TypeExpense, TypeOtherIncome, TypeOtherExpense:
		return true
	}
	return false
}

// EarningsContribution is a P&L account's contribution to equity.
func EarningsContribution(typeCode string, balance Money) Money {
	if typeCode == TypeRevenue || typeCode == TypeOtherIncome {
		return balance
	}
	return -balance
}

// CashFlowContribution is the indirect-lite cash effect of a non-cash
// account's period balance.
func CashFlowContribution(typeCode string, balance Money) Money {
	switch typeCode {
	case TypeRevenue, TypeOtherIncome, TypeLiability, TypeEquity:
		return balance
	}
	return -balance
}

// OpeningLine is the single opening line of a net position.
func OpeningLine(debit, credit Money) (side string, amount Money, ok bool) {
	net := debit - credit
	switch {
	case net > 0:
		return Debit, net, true
	case net < 0:
		return Credit, -net, true
	}
	return "", 0, false
}

// BalanceTypeOrder sorts beginning balance lines: assets, liabilities,
// equity, then the rest.
func BalanceTypeOrder(code string) int {
	switch code {
	case TypeAsset:
		return 1
	case TypeLiability:
		return 2
	case TypeEquity:
		return 3
	}
	return 9
}

// ─── AP/AR status and aging (receivables.go) ────────────────

var AgingBuckets = []string{"current", "1_30", "31_60", "61_90", "90_plus"}

// InvoiceOutstanding is total - allocated, never negative.
func InvoiceOutstanding(total, allocated Money) Money {
	if v := total - allocated; v > 0 {
		return v
	}
	return 0
}

// PaymentStatus is paid / partial / overdue / unpaid. Dates are
// "YYYY-MM-DD" and compare as strings.
func PaymentStatus(total, allocated Money, dueDate, today string) string {
	if InvoiceOutstanding(total, allocated) <= 0 {
		return "paid"
	}
	pastDue := dueDate != "" && dueDate < today
	if allocated > 0 {
		if pastDue {
			return "overdue"
		}
		return "partial"
	}
	if pastDue {
		return "overdue"
	}
	return "unpaid"
}

// DaysPastDue is negative when not yet due, 0 when a date does not parse.
func DaysPastDue(dueDate, asOf string) int {
	due, err1 := time.Parse(time.DateOnly, dueDate)
	at, err2 := time.Parse(time.DateOnly, asOf)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(at.Sub(due).Hours() / 24)
}

// BucketAging returns "" when nothing is outstanding.
func BucketAging(outstanding Money, dueDate, asOf string) string {
	if outstanding <= 0 {
		return ""
	}
	days := DaysPastDue(dueDate, asOf)
	switch {
	case days <= 0:
		return "current"
	case days <= 30:
		return "1_30"
	case days <= 60:
		return "31_60"
	case days <= 90:
		return "61_90"
	}
	return "90_plus"
}

// VoidReasonMinLength is the minimum void reason length (characters).
const VoidReasonMinLength = 5

// EvaluatePaymentVoid returns nil when a void may go ahead. found=false
// covers a missing or soft-deleted payment.
func EvaluatePaymentVoid(found bool, status, reason, label string) error {
	if !found {
		return NotFound("Pembayaran " + label + " tidak ditemukan")
	}
	if len([]rune(strings.TrimFunc(reason, isSpace))) < VoidReasonMinLength {
		return Bad(fmt.Sprintf("Alasan void wajib diisi (minimal %d karakter)", VoidReasonMinLength))
	}
	if status == StatusVoid {
		return Conflict("Pembayaran ini sudah di-void")
	}
	if status != StatusPosted {
		return Conflict("Hanya pembayaran POSTED yang bisa di-void")
	}
	return nil
}
