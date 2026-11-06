package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg" // Decoders for ImageFingerprint.
	_ "image/png"
	"math/bits"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Statement line statuses.
const (
	LineUnmatched = "UNMATCHED"
	LineSuggested = "SUGGESTED"
	LineMatched   = "MATCHED"
	LineIgnored   = "IGNORED"
)

// Suggestion kinds.
const (
	SuggestJournal = "JOURNAL"
	SuggestInvoice = "INVOICE"
)

// StatementLine is one row of an uploaded bank or card statement.
// Amount is positive for money in, negative for money out or a card charge.
type StatementLine struct {
	ID, StatementID, AccountID, LineDate, Description, SenderName, DedupeKey string
	Amount                                                                   Money
	Status                                                                   string
	SuggestedKind, SuggestedRef, SuggestedLabel, SuggestedReason             string
	SuggestedScore                                                           int
	MatchedLineID, MatchedEntryID, MatchedEntryNo                            string
	Note, VerifiedBy                                                         string
	VerifiedAt                                                               *time.Time
}

// Side is the journal side this row settles on the reconciled account: money
// in debits a bank and money out credits it; a card charge credits the card
// liability and a card payment debits it.
func (l StatementLine) Side() string {
	if l.Amount > 0 {
		return Debit
	}
	return Credit
}

// Abs is the row's amount without sign.
func (l StatementLine) Abs() Money {
	if l.Amount < 0 {
		return -l.Amount
	}
	return l.Amount
}

// MatchableLine is a journal line with what a match has to check.
type MatchableLine struct {
	LedgerLine
	AccountID, Status string
}

// StatementFilter selects statement rows.
type StatementFilter struct {
	AccountID, Status, DateFrom, DateTo string
	Page, PageSize                      int
}

// ParsedRow is a statement row before it is stored.
type ParsedRow struct {
	Date, Description, Sender string
	Amount                    Money
	DedupeKey                 string
}

var (
	dateColumns   = []string{"tanggal", "tgl", "date", "tanggal transaksi", "transaction date", "posting date"}
	descColumns   = []string{"keterangan", "uraian", "description", "deskripsi", "remark", "berita", "transaction description"}
	senderColumns = []string{"nama pengirim", "pengirim", "sender", "nama", "counterparty"}
	debitColumns  = []string{"debit", "debet", "mutasi debit", "keluar", "uang keluar", "db"}
	creditColumns = []string{"kredit", "credit", "mutasi kredit", "masuk", "uang masuk", "cr"}
	amountColumns = []string{"mutasi", "amount", "nominal", "jumlah"}
	typeColumns   = []string{"dk", "d/k", "db/cr", "tipe", "type"}

	moneySuffix = regexp.MustCompile(`(?i)\s*(CR|DB|K|D)$`)
)

// ParseStatementCSV reads a statement export. It finds the header row by
// column names, so bank preambles above it are skipped, and accepts
// comma, semicolon or tab separators and both debit/credit columns and one
// amount column with a CR/DB marker.
func ParseStatementCSV(data []byte) ([]ParsedRow, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	records, head, col := findHeader(data)
	if head < 0 {
		return nil, Bad("Kolom mutasi tidak dikenali. File perlu punya kolom tanggal, keterangan, dan debit/kredit atau mutasi.")
	}
	var out []ParsedRow
	seen := map[string]int{}
	for i, rec := range records[head+1:] {
		date := cell(rec, col.date)
		if date == "" {
			continue // Footer and summary rows carry no date.
		}
		day, err := parseStatementDate(date)
		if err != nil {
			if len(out) > 0 {
				continue // Summary rows after the data.
			}
			return nil, Bad(fmt.Sprintf("Baris %d: tanggal %q tidak dikenali.", head+i+2, date))
		}
		amount, err := col.amount(rec)
		if err != nil {
			return nil, Bad(fmt.Sprintf("Baris %d: nominal tidak dikenali.", head+i+2))
		}
		if amount == 0 {
			continue
		}
		row := ParsedRow{Date: day, Description: cell(rec, col.desc), Sender: cell(rec, col.sender), Amount: amount}
		// Two identical rows on one day are two real transactions, so the
		// key counts occurrences inside the file.
		base := row.Date + "|" + strings.ToLower(row.Description) + "|" + row.Amount.String()
		seen[base]++
		sum := sha256.Sum256([]byte(base + "|" + strconv.Itoa(seen[base])))
		row.DedupeKey = hex.EncodeToString(sum[:16])
		out = append(out, row)
	}
	if len(out) == 0 {
		return nil, Bad("File mutasi tidak berisi transaksi.")
	}
	return out, nil
}

// findHeader tries each separator, because a bank's preamble rows may not use
// the same one as its table.
func findHeader(data []byte) ([][]string, int, statementColumns) {
	for _, sep := range []rune{',', ';', '\t'} {
		r := csv.NewReader(bytes.NewReader(data))
		r.Comma, r.FieldsPerRecord, r.LazyQuotes, r.TrimLeadingSpace = sep, -1, true, true
		records, err := r.ReadAll()
		if err != nil {
			continue
		}
		for i, rec := range records {
			if c, ok := findColumns(rec); ok {
				return records, i, c
			}
		}
	}
	return nil, -1, statementColumns{}
}

type statementColumns struct{ date, desc, sender, debit, credit, single, kind int }

func findColumns(rec []string) (statementColumns, bool) {
	find := func(names []string) int {
		return slices.IndexFunc(rec, func(c string) bool {
			return slices.Contains(names, strings.ToLower(strings.TrimSpace(c)))
		})
	}
	c := statementColumns{date: find(dateColumns), desc: find(descColumns), sender: find(senderColumns),
		debit: find(debitColumns), credit: find(creditColumns), single: find(amountColumns), kind: find(typeColumns)}
	return c, c.date >= 0 && c.desc >= 0 && ((c.debit >= 0 && c.credit >= 0) || c.single >= 0)
}

func cell(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

func (c statementColumns) amount(rec []string) (Money, error) {
	if c.debit >= 0 && c.credit >= 0 {
		out, err := parseBankMoney(cell(rec, c.debit))
		if err != nil {
			return 0, err
		}
		in, err := parseBankMoney(cell(rec, c.credit))
		if err != nil {
			return 0, err
		}
		return abs(in) - abs(out), nil
	}
	raw := cell(rec, c.single)
	marker := strings.ToUpper(cell(rec, c.kind))
	if m := moneySuffix.FindStringSubmatch(raw); m != nil {
		marker = strings.ToUpper(m[1])
		raw = strings.TrimSpace(raw[:len(raw)-len(m[0])])
	}
	v, err := parseBankMoney(raw)
	if err != nil {
		return 0, err
	}
	switch marker {
	case "DB", "D", "DEBIT", "DEBET":
		return -abs(v), nil
	case "CR", "K", "C", "KREDIT", "CREDIT":
		return abs(v), nil
	}
	return v, nil
}

func abs(m Money) Money {
	if m < 0 {
		return -m
	}
	return m
}

// parseBankMoney reads "1.250.000,50", "1,250,000.50" and "1250000". A lone
// separator followed by three digits is a thousands mark.
func parseBankMoney(s string) (Money, error) {
	s = strings.NewReplacer("Rp", "", "IDR", "", " ", "", "(", "-", ")", "").Replace(s)
	if s == "" || s == "-" {
		return 0, nil
	}
	dot, comma := strings.LastIndex(s, "."), strings.LastIndex(s, ",")
	decimal := max(dot, comma)
	switch {
	case dot >= 0 && comma >= 0:
	case decimal >= 0 && (len(s)-decimal-1 == 3 || strings.Count(s, s[decimal:decimal+1]) > 1):
		decimal = -1
	}
	var b strings.Builder
	for i, r := range s {
		switch {
		case r == '-' || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case i == decimal:
			b.WriteRune('.')
		case r != '.' && r != ',':
			return 0, fmt.Errorf("bad money %q", s)
		}
	}
	return ParseMoney(b.String())
}

func parseStatementDate(s string) (string, error) {
	for _, layout := range []string{"2006-01-02", "02/01/2006", "02-01-2006", "02/01/06", "02-01-06", "2/1/2006", "02 Jan 2006", "2006/01/02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("bad date %q", s)
}

// ─── Matching ────────────────────────────────

// MatchCandidate is something a statement row may settle.
type MatchCandidate struct {
	Kind, Ref, Label, Date, Text string
	Amount                       Money
	Score                        int
	Reason                       string
}

// Scoring: the amount must be equal, then date and name raise confidence. Only
// a single journal candidate at autoMatchScore is matched without a person.
const autoMatchScore = 90

// ScoreCandidate fills Score and Reason; amount equality is the caller's filter.
func ScoreCandidate(l StatementLine, c MatchCandidate) MatchCandidate {
	reasons := []string{"nominal sama"}
	c.Score = 50
	if c.Date != "" {
		switch d := dayDistance(l.LineDate, c.Date); {
		case d == 0:
			c.Score += 30
			reasons = append(reasons, "tanggal sama")
		case d <= 3:
			c.Score += 20
			reasons = append(reasons, fmt.Sprintf("selisih %d hari", d))
		case d <= 7:
			c.Score += 10
			reasons = append(reasons, fmt.Sprintf("selisih %d hari", d))
		}
	}
	if sharesName(l.SenderName+" "+l.Description, c.Text) {
		c.Score += 20
		reasons = append(reasons, "nama cocok")
	}
	c.Reason = strings.Join(reasons, ", ")
	return c
}

// Decide ranks candidates and returns the status with the suggestion. Equal
// top scores mean the system is not sure, so the row stays unmatched.
func Decide(cands []MatchCandidate) (string, *MatchCandidate) {
	if len(cands) == 0 {
		return LineUnmatched, nil
	}
	slices.SortStableFunc(cands, func(a, b MatchCandidate) int { return b.Score - a.Score })
	top := cands[0]
	if len(cands) > 1 && cands[1].Score == top.Score {
		return LineUnmatched, nil
	}
	if top.Kind == SuggestJournal && top.Score >= autoMatchScore {
		return LineMatched, &top
	}
	return LineSuggested, &top
}

func dayDistance(a, b string) int {
	ta, e1 := time.Parse("2006-01-02", a)
	tb, e2 := time.Parse("2006-01-02", b)
	if e1 != nil || e2 != nil {
		return 1 << 20
	}
	d := int(ta.Sub(tb).Hours() / 24)
	if d < 0 {
		return -d
	}
	return d
}

var nameNoise = map[string]bool{"transfer": true, "trf": true, "bank": true, "dari": true, "ke": true, "pt": true, "cv": true,
	"mandiri": true, "bca": true, "invoice": true, "pembayaran": true, "payment": true, "untuk": true, "biaya": true}

func nameTokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, t := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-'
	}) {
		if len(t) >= 3 && !nameNoise[t] {
			out[t] = true
		}
	}
	return out
}

func sharesName(a, b string) bool {
	tokens := nameTokens(b)
	for t := range nameTokens(a) {
		if tokens[t] {
			return true
		}
	}
	return false
}

// ─── Receipts ────────────────────────────────

// ReceiptPrint is what identifies a stored receipt.
type ReceiptPrint struct {
	ReceiptID, ClaimID, ClaimNo, Filename, ContentHash string
	ImageHash                                          *int64
}

// DuplicateReceipt flags a receipt that also appears on another claim.
type DuplicateReceipt struct {
	ReceiptID, ClaimNo, Filename string
	// Exact is a byte-identical file; otherwise the image looks the same.
	Exact bool
}

// ImageFingerprint is a 64-bit difference hash of a JPEG or PNG, stable across
// re-compression and resizing, so a screenshot forwarded through a chat app still
// matches. It returns nil for PDFs, WebP and images it cannot read safely.
func ImageFingerprint(data []byte) *int64 {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 9 || cfg.Height < 8 || cfg.Width*cfg.Height > 50_000_000 {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	b := img.Bounds()
	luma := func(x, y int) uint32 {
		r, g, bl, _ := img.At(b.Min.X+x*b.Dx()/9, b.Min.Y+y*b.Dy()/8).RGBA()
		return 299*r + 587*g + 114*bl
	}
	var hash uint64
	for y := range 8 {
		for x := range 8 {
			hash <<= 1
			if luma(x, y) > luma(x+1, y) {
				hash |= 1
			}
		}
	}
	out := int64(hash)
	return &out
}

// ContentHash identifies a file's exact bytes.
func ContentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// imageHashDistance is how many of 64 bits two similar images may differ by.
const imageHashDistance = 4

// FindDuplicates compares a claim's receipts with receipts on other claims.
func FindDuplicates(own, others []ReceiptPrint) []DuplicateReceipt {
	out := []DuplicateReceipt{}
	for _, r := range own {
		for _, o := range others {
			exact := r.ContentHash != "" && r.ContentHash == o.ContentHash
			similar := r.ImageHash != nil && o.ImageHash != nil && bits.OnesCount64(uint64(*r.ImageHash^*o.ImageHash)) <= imageHashDistance
			if exact || similar {
				out = append(out, DuplicateReceipt{ReceiptID: r.ReceiptID, ClaimNo: o.ClaimNo, Filename: o.Filename, Exact: exact})
			}
		}
	}
	return out
}

// ─── Recurring and depreciation ──────────────

// Recurring frequencies.
const (
	FreqMonthly   = "MONTHLY"
	FreqQuarterly = "QUARTERLY"
	FreqYearly    = "YEARLY"
)

// RecurringJournal is a journal template posted on a schedule.
type RecurringJournal struct {
	TagID                                                                      string
	ID, Name, Description, Frequency, NextRun, EndDate, LastEntryID, LastError string
	AnchorDay                                                                  int
	Lines                                                                      []Line
	IsActive                                                                   bool
	CreatedBy, UpdatedBy                                                       string
}

// NextRunAfter is the next date of a schedule, clamped to the month's length so
// day 31 runs on the 30th in April and returns to the 31st in May.
func NextRunAfter(day, frequency string, anchor int) (string, error) {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return "", err
	}
	months := map[string]int{FreqMonthly: 1, FreqQuarterly: 3, FreqYearly: 12}[frequency]
	first := time.Date(t.Year(), t.Month()+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(anchor, last), 0, 0, 0, 0, time.UTC).Format("2006-01-02"), nil
}

// FixedAsset is depreciated straight-line, from the acquisition month.
type FixedAsset struct {
	ID, Name, AcquiredOn, ExpenseAccountID, AccumAccountID string
	ExpenseAccountName, AccumAccountName                   string
	LastEntryID, LastError                                 string
	DisposedOn                                             string
	Cost, Salvage, DisposalProceeds                        Money
	LifeMonths, DepreciatedMonths                          int
	IsActive                                               bool
	CreatedBy, UpdatedBy                                   string
}

// DepreciationFor is month n's charge (1-based). It takes the difference of
// rounded cumulative totals, so the months always add up to cost minus salvage.
func (a FixedAsset) DepreciationFor(n int) Money {
	return a.accumulatedAt(n) - a.accumulatedAt(n-1)
}

func (a FixedAsset) accumulatedAt(n int) Money {
	return straightLineTotal(a.Cost-a.Salvage, a.LifeMonths, n)
}

// straightLineTotal is the rounded total after n of months equal parts.
func straightLineTotal(base Money, months, n int) Money {
	life := int64(months)
	return Money((int64(base)*int64(n)*2 + life) / (2 * life))
}

// Accumulated is the depreciation booked so far.
func (a FixedAsset) Accumulated() Money { return a.accumulatedAt(a.DepreciatedMonths) }

// MonthEnd is the posting date of month n (1-based).
func (a FixedAsset) MonthEnd(n int) (string, error) { return monthEndAfter(a.AcquiredOn, n) }

// monthEndAfter is the last day of month n, counting date's own month as 1.
func monthEndAfter(date string, n int) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", err
	}
	return time.Date(t.Year(), t.Month()+time.Month(n), 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), nil
}

// ─── Budgets and client economics ────────────

// Budget is one account's plan for one month ("2026-11").
type Budget struct {
	Month, AccountID string
	Amount           Money
}

// ClientEconomics is one client's invoices in a period. PaidDays is the sum of
// amount times days to pay, in rupiah-days, for a weighted average.
type ClientEconomics struct {
	Party        string
	InvoiceCount int
	Invoiced     Money
	Collected    Money
	PaidDays     float64
}

// AvgDaysToPay weighs each payment by its amount; ok is false with no payment.
func (c ClientEconomics) AvgDaysToPay() (float64, bool) {
	if c.Collected <= 0 {
		return 0, false
	}
	return c.PaidDays / c.Collected.Float(), true
}

// BudgetVariance is favorable when positive: more revenue or less cost than planned.
func BudgetVariance(typeCode string, budget, actual Money) Money {
	if typeCode == TypeRevenue || typeCode == TypeOtherIncome {
		return actual - budget
	}
	return budget - actual
}
