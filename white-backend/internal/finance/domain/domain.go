// Package domain holds the finance entities, repository ports and pure
// rules: account codes, journal validation and state machine, fiscal
// sequencing, journal mappings, AP/AR status and aging, report balances,
// reimbursement claims and financial ratios. Dates are "YYYY-MM-DD"
// strings on the Asia/Jakarta calendar; amounts are Money (sen).
package domain

import (
	"context"
	"time"
)

// ─── Chart of accounts ──────────────────────────────────────

type AccountType struct {
	ID            string
	Code          string
	Name          string
	NormalBalance string
	SortOrder     int
	IsActive      bool
	AccountCount  int
}

// StandardAccountTypes are the eight NüHabit account types.
var StandardAccountTypes = []AccountType{
	{ID: "at-asset", Code: TypeAsset, Name: "Aset", NormalBalance: Debit, SortOrder: 10, IsActive: true},
	{ID: "at-liability", Code: TypeLiability, Name: "Liabilitas", NormalBalance: Credit, SortOrder: 20, IsActive: true},
	{ID: "at-equity", Code: TypeEquity, Name: "Ekuitas", NormalBalance: Credit, SortOrder: 30, IsActive: true},
	{ID: "at-revenue", Code: TypeRevenue, Name: "Pendapatan", NormalBalance: Credit, SortOrder: 40, IsActive: true},
	{ID: "at-cogs", Code: TypeCOGS, Name: "Harga Pokok Penjualan", NormalBalance: Debit, SortOrder: 50, IsActive: true},
	{ID: "at-expense", Code: TypeExpense, Name: "Beban Operasional", NormalBalance: Debit, SortOrder: 60, IsActive: true},
	{ID: "at-other-income", Code: TypeOtherIncome, Name: "Pendapatan Lain-lain", NormalBalance: Credit, SortOrder: 70, IsActive: true},
	{ID: "at-other-expense", Code: TypeOtherExpense, Name: "Beban Lain-lain", NormalBalance: Debit, SortOrder: 80, IsActive: true},
}

type Account struct {
	ID               string
	Code             string
	Name             string
	ParentID         string
	AccountTypeID    string
	TypeCode         string
	TypeName         string
	NormalBalance    string
	Level            int
	IsPostable       bool
	IsContra         bool
	IsCashBank       bool
	CashFlowCategory string
	Description      string
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AccountFilter struct {
	AccountTypeID    string
	IsPostable       *bool
	IsCashBank       *bool
	IsActive         *bool
	CashFlowCategory string
	Search           string
}

type JournalMapping struct {
	ID          string
	EventCode   string
	Name        string
	Description string
	Module      string
	IsActive    bool
	Lines       []MappingLineView
	UpdatedAt   time.Time
}

// MappingLineView is a mapping line with its account labels.
type MappingLineView struct {
	MappingLine
	AccountCode string
	AccountName string
}

func (m JournalMapping) Template() []MappingLine {
	out := make([]MappingLine, len(m.Lines))
	for i, l := range m.Lines {
		out[i] = l.MappingLine
	}
	return out
}

type MappingFilter struct {
	Search   string
	Module   string
	IsActive *bool
}

// ─── Fiscal ─────────────────────────────────────────────────

type FiscalYear struct {
	ID        string
	Code      string
	Name      string
	StartDate string
	EndDate   string
	IsActive  bool
	Periods   []FiscalPeriod
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (y FiscalYear) OpenCount() int {
	n := 0
	for _, p := range y.Periods {
		if p.Status == PeriodOpen {
			n++
		}
	}
	return n
}

type FiscalPeriod struct {
	ID           string
	FiscalYearID string
	PeriodNo     int
	Name         string
	StartDate    string
	EndDate      string
	Status       string
	YearCode     string
	YearName     string
	YearActive   bool
	PostedCount  int
	DraftCount   int
}

type PeriodFilter struct {
	FiscalYearID string
	Status       string
	Search       string
}

// YearOpenCount is an earlier year that still has OPEN periods.
type YearOpenCount struct {
	ID, Code, Name, EndDate string
	OpenCount               int
}

// ─── Journals ───────────────────────────────────────────────

type JournalLine struct {
	ID          string
	AccountID   string
	AccountCode string
	AccountName string
	Side        string
	Amount      Money
	Memo        string
	SortOrder   int
}

type JournalEntry struct {
	ID                 string
	EntryNo            string
	EntryDate          string
	Description        string
	FiscalPeriodID     string
	PeriodName         string
	YearCode           string
	EntryType          string
	Status             string
	IsRecon            bool
	PostedAt           *time.Time
	PostedBy           string
	SourceModule       string
	SourceEventCode    string
	SourceDocumentType string
	SourceDocumentID   string
	SourceFiscalYearID string
	Lines              []JournalLine
	CreatedBy          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (e JournalEntry) DomainLines() []Line {
	out := make([]Line, len(e.Lines))
	for i, l := range e.Lines {
		out[i] = Line{AccountID: l.AccountID, Side: l.Side, Amount: l.Amount, Memo: l.Memo, SortOrder: l.SortOrder}
	}
	return out
}

func (e JournalEntry) Totals() (Money, Money) { return Totals(e.DomainLines()) }

func (e JournalEntry) CanEdit() bool {
	return CanEdit(e.Status, e.IsRecon, NormalizeEntryType(e.EntryType))
}

type JournalFilter struct {
	Search             string
	Status             string
	EntryType          string
	DateFrom, DateTo   string
	AccountID          string
	SourceDocumentType string
	Limit, Offset      int
}

// PostingLog records one automatic posting attempt per (event, document).
type PostingLog struct {
	ID            string
	EventCode     string
	DocumentType  string
	DocumentID    string
	DocumentLabel string
	Status        string
	Reason        string
	EntryID       string
	Request       []byte
	Attempts      int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ─── Balances ───────────────────────────────────────────────

// BalanceWindow bounds the posted lines summed: AsOf (<=), From (>=), To (<=).
type BalanceWindow struct{ AsOf, From, To string }

type AccountBalance struct {
	ID               string
	Code             string
	Name             string
	TypeCode         string
	NormalBalance    string
	IsContra         bool
	IsCashBank       bool
	CashFlowCategory string
	Level            int
	Debit            Money
	Credit           Money
	Balance          Money
	MovementCount    int
}

// Net is the balance as it counts toward its section total. Contra balances
// are positive on their own side (accumulated depreciation), so they reduce it.
func (b AccountBalance) Net() Money {
	if b.IsContra {
		return -b.Balance
	}
	return b.Balance
}

type LedgerLine struct {
	LineID      string
	EntryID     string
	EntryNo     string
	EntryDate   string
	EntryType   string
	Description string
	Memo        string
	Side        string
	Amount      Money
}

type MonthPnL struct {
	Month            string
	Revenue, Expense Money
}

// ─── Invoices and payments ──────────────────────────────────

const (
	KindAP = "AP"
	KindAR = "AR"
)

type Invoice struct {
	ID             string
	Kind           string
	InvoiceNo      string
	InvoiceDate    string
	DueDate        string
	PartyName      string
	Reference      string
	AccountID      string
	AccountName    string
	Currency       string
	Subtotal       Money
	TaxAmount      Money
	TotalAmount    Money
	Status         string
	Description    string
	SalesInvoiceID string
	ClientID       string
	DealID         string
	Allocated      Money
	JournalEntryID string
	PostedAt       *time.Time
	CreatedBy      string
	CreatedAt      time.Time
}

func (i Invoice) Outstanding() Money { return InvoiceOutstanding(i.TotalAmount, i.Allocated) }

type InvoiceFilter struct {
	Kind          string
	Status        string
	Search        string
	From, To      string
	PartyName     string
	Limit, Offset int
}

type Allocation struct {
	InvoiceID string
	InvoiceNo string
	Amount    Money
}

type Payment struct {
	ID              string
	Kind            string
	PaymentNo       string
	PaymentDate     string
	Amount          Money
	Method          string
	CashAccountID   string
	CashAccountName string
	Reference       string
	Notes           string
	Status          string
	PartyName       string
	SalesPaymentID  string
	JournalEntryID  string
	VoidReason      string
	VoidedAt        *time.Time
	Allocations     []Allocation
	CreatedBy       string
	CreatedAt       time.Time
}

type PaymentFilter struct {
	Kind          string
	Search        string
	Limit, Offset int
}

type SubsidiaryParty struct {
	PartyName    string
	Outstanding  Money
	InvoiceCount int
}

// SubsidiaryMove is an invoice (debit) or an allocation (credit) of a party.
type SubsidiaryMove struct {
	ID          string
	DocNo       string
	Date        string
	Amount      Money
	Description string
	IsInvoice   bool
}

// ─── Reimbursement ──────────────────────────────────────────

type Claim struct {
	ID                string
	ClaimNo           string
	EmployeeID        string
	EmployeeUserID    string
	EmployeeName      string
	DepartmentName    string
	ManagerEmployeeID string
	ManagerUserID     string
	Category          string
	TransactionDate   string
	Amount            Money
	Description       string
	ClientID          string
	Project           string
	Status            ClaimStatus
	OverLimit         bool
	LimitNote         string
	SubmittedAt       *time.Time
	ManagerApprovedAt *time.Time
	ManagerApprovedBy string
	FinanceApprovedAt *time.Time
	FinanceApprovedBy string
	RejectedAt        *time.Time
	RejectedBy        string
	RejectedStage     string
	RejectReason      string
	CancelledAt       *time.Time
	PaidAt            *time.Time
	PaidBy            string
	PaymentBatchID    string
	LastRemindedAt    *time.Time
	Receipts          []Receipt
	History           []ClaimEvent
	CreatedBy         string
	CreatedAt         time.Time
	UpdatedBy         string
	UpdatedAt         time.Time
}

type Receipt struct {
	ID          string
	ClaimID     string
	StorageKey  string
	Filename    string
	ContentType string
	SizeBytes   int64
	CreatedBy   string
	CreatedAt   time.Time
}

type ClaimEvent struct {
	ID        string
	ClaimID   string
	Action    string
	ActorID   string
	ActorName string
	Note      string
	CreatedAt time.Time
}

type ClaimFilter struct {
	Statuses        []ClaimStatus
	Category        string
	EmployeeID      string
	EmployeeUserID  string
	ManagerUserID   string
	Department      string
	MonthFrom       string
	MonthTo         string
	Search          string
	SubmittedBefore *time.Time
	RemindedBefore  *time.Time
	Limit, Offset   int
}

type ReimbursementBatch struct {
	ID              string
	BatchNo         string
	PaidAt          string
	CashAccountID   string
	CashAccountName string
	TotalAmount     Money
	ClaimCount      int
	JournalEntryID  string
	JournalStatus   string
	JournalReason   string
	Notes           string
	CreatedBy       string
	CreatedAt       time.Time
}

type SummaryRow struct {
	EmployeeID      string
	EmployeeName    string
	DepartmentName  string
	Category        string
	ClaimCount      int
	SubmittedAmount Money
	ApprovedAmount  Money
	PaidAmount      Money
}

// ─── Analysis ───────────────────────────────────────────────

type FinancialInsight struct {
	ID        string
	DateFrom  string
	DateTo    string
	Language  string
	Model     string
	Content   string
	Metrics   []byte
	CreatedBy string
	CreatedAt time.Time
}

// ─── Ports ──────────────────────────────────────────────────

type AccountRepo interface {
	ListAccountTypes(ctx context.Context, includeInactive bool) ([]AccountType, error)
	GetAccountType(ctx context.Context, id string) (*AccountType, error)
	UpsertAccountType(ctx context.Context, t AccountType, by string) error
	DeleteAccountType(ctx context.Context, id string) error

	ListAccounts(ctx context.Context, f AccountFilter) ([]Account, error)
	GetAccount(ctx context.Context, id string) (*Account, error)
	AccountsByCode(ctx context.Context) (map[string]Account, error)
	InsertAccount(ctx context.Context, a Account, by string) error
	UpdateAccount(ctx context.Context, a Account, by string) error
	SoftDeleteAccount(ctx context.Context, id, by string) error
	CountChildren(ctx context.Context, id string) (int, error)
	// RecomputePostable makes a leaf postable and its ancestors headers.
	RecomputePostable(ctx context.Context, id string) error
	RecomputeAllPostable(ctx context.Context) error
	AccountChecks(ctx context.Context, ids []string) ([]AccountCheck, error)

	ListMappings(ctx context.Context, f MappingFilter) ([]JournalMapping, error)
	GetMapping(ctx context.Context, id string) (*JournalMapping, error)
	GetActiveMapping(ctx context.Context, eventCode string) (*JournalMapping, error)
	InsertMapping(ctx context.Context, m JournalMapping, by string) error
	UpdateMapping(ctx context.Context, m JournalMapping, by string) error
	ReplaceMappingLines(ctx context.Context, mappingID string, lines []MappingLine) error
	SoftDeleteMapping(ctx context.Context, id, by string) error
}

type FiscalRepo interface {
	ListFiscalYears(ctx context.Context, search string, active *bool) ([]FiscalYear, error)
	GetFiscalYear(ctx context.Context, id string) (*FiscalYear, error)
	InsertFiscalYear(ctx context.Context, y FiscalYear, by string) error
	UpdateFiscalYear(ctx context.Context, y FiscalYear, by string) error
	SoftDeleteFiscalYear(ctx context.Context, id, by string) error
	// UpsertPeriods writes periods by period_no and deletes the year's
	// other periods that no journal uses.
	UpsertPeriods(ctx context.Context, yearID string, periods []PeriodInput) error
	UsedPeriodNos(ctx context.Context, yearID string) ([]int, error)
	CountYearEntries(ctx context.Context, yearID string) (int, error)
	// PriorYearWithOpenPeriods is an earlier year (end < before) with OPEN
	// periods, excluding excludeID.
	PriorYearWithOpenPeriods(ctx context.Context, before, excludeID string) (*YearOpenCount, error)
	PriorFiscalYear(ctx context.Context, before string) (*YearOpenCount, error)
	CoveringFiscalYearStart(ctx context.Context, date string) (string, error)

	ListPeriods(ctx context.Context, f PeriodFilter) ([]FiscalPeriod, error)
	GetPeriod(ctx context.Context, id string) (*FiscalPeriod, error)
	SetPeriodStatus(ctx context.Context, ids []string, status string) error
	// OpenPeriodFor is the OPEN period of an active year covering date.
	OpenPeriodFor(ctx context.Context, date string) (*FiscalPeriod, error)
	CoveringPeriod(ctx context.Context, date string) (*FiscalPeriod, error)
	PreviousOpenPeriods(ctx context.Context, yearID string, periodNo int) ([]FiscalPeriod, error)
	DraftEntries(ctx context.Context, periodID string, limit int) ([]JournalEntry, error)
}

type JournalRepo interface {
	NextSequence(ctx context.Context, prefix string) (int, error)
	InsertJournalEntry(ctx context.Context, e JournalEntry) error
	UpdateJournalHeader(ctx context.Context, e JournalEntry, by string) error
	ReplaceJournalLines(ctx context.Context, entryID string, lines []Line) error
	MarkJournalPosted(ctx context.Context, id, by string) error
	SoftDeleteJournalEntry(ctx context.Context, id, by string) error
	GetJournalEntry(ctx context.Context, id string) (*JournalEntry, error)
	ListJournalEntries(ctx context.Context, f JournalFilter) ([]JournalEntry, int, error)
	FindEntryBySource(ctx context.Context, eventCode, documentID string) (*JournalEntry, error)
	OpeningEntry(ctx context.Context, yearID string) (*JournalEntry, error)
	CountEntries(ctx context.Context, from, to string) (posted, drafts int, err error)

	UpsertPostingLog(ctx context.Context, l PostingLog) (*PostingLog, error)
	GetPostingLog(ctx context.Context, id string) (*PostingLog, error)
	ListPostingLogs(ctx context.Context, status, eventCode string, limit int) ([]PostingLog, error)
}

type BalanceRepo interface {
	// AccountBalances sums posted lines per postable active account.
	AccountBalances(ctx context.Context, w BalanceWindow) ([]AccountBalance, error)
	// YearAccountTotals sums the posted lines of one fiscal year per account.
	YearAccountTotals(ctx context.Context, yearID string) ([]AccountBalance, error)
	AccountTotalsBefore(ctx context.Context, accountID, date string) (debit, credit Money, err error)
	LedgerLines(ctx context.Context, accountID, from, to string) ([]LedgerLine, error)
	MonthlyPnL(ctx context.Context, from, to string) ([]MonthPnL, error)
}

type InvoiceRepo interface {
	ListInvoices(ctx context.Context, f InvoiceFilter) ([]Invoice, int, error)
	GetInvoice(ctx context.Context, id string) (*Invoice, error)
	GetInvoiceBySalesID(ctx context.Context, salesInvoiceID string) (*Invoice, error)
	InsertInvoice(ctx context.Context, i Invoice) error
	UpdateInvoice(ctx context.Context, i Invoice, by string) error
	SoftDeleteInvoice(ctx context.Context, id, by string) error
	MarkInvoicePosted(ctx context.Context, id, by string, at time.Time) error
	SetInvoiceJournal(ctx context.Context, id, entryID string) error

	ListPayments(ctx context.Context, f PaymentFilter) ([]Payment, int, error)
	GetPayment(ctx context.Context, id string) (*Payment, error)
	GetPaymentBySalesID(ctx context.Context, salesPaymentID string) (*Payment, error)
	InsertPayment(ctx context.Context, p Payment) error
	VoidPayment(ctx context.Context, id, reason, by string) error
	SetPaymentJournal(ctx context.Context, id, entryID string) error

	SubsidiaryParties(ctx context.Context, kind string) ([]SubsidiaryParty, error)
	SubsidiaryMoves(ctx context.Context, kind, party, from, to string) ([]SubsidiaryMove, error)
	SubsidiaryOpening(ctx context.Context, kind, party, before string) (Money, error)
}

type ReimbursementRepo interface {
	ListPolicies(ctx context.Context) ([]Policy, error)
	GetPolicy(ctx context.Context, category string) (*Policy, error)
	UpsertPolicy(ctx context.Context, p Policy, by string) error

	InsertClaim(ctx context.Context, c Claim) error
	UpdateClaim(ctx context.Context, c Claim) error
	GetClaim(ctx context.Context, id string) (*Claim, error)
	ListClaims(ctx context.Context, f ClaimFilter) ([]Claim, int, Money, error)
	// MonthToDate sums the employee's live claims (submitted or later) of a
	// category in [from, to], excluding excludeID.
	MonthToDate(ctx context.Context, employeeID, category, from, to, excludeID string) (Money, error)
	InsertReceipt(ctx context.Context, r Receipt) error
	DeleteReceipt(ctx context.Context, id string) error
	GetReceipt(ctx context.Context, id string) (*Receipt, error)
	InsertClaimEvent(ctx context.Context, e ClaimEvent) error
	MarkReminded(ctx context.Context, ids []string, at time.Time) error
	InsertBatch(ctx context.Context, b ReimbursementBatch) error
	UpdateBatchJournal(ctx context.Context, id, entryID, status, reason string) error
	ListBatches(ctx context.Context, limit int) ([]ReimbursementBatch, error)
	MonthlySummary(ctx context.Context, from, to, employeeID string) ([]SummaryRow, error)
	// ForgetUser drops the user's DRAFT claims (returning their receipt
	// keys) and clears manager references to the user.
	ForgetUser(ctx context.Context, userID string) ([]string, error)
	CountPendingClaims(ctx context.Context) (int, error)
}

type InsightRepo interface {
	InsertInsight(ctx context.Context, i FinancialInsight) error
	ListInsights(ctx context.Context, limit int) ([]FinancialInsight, error)
}

// Store is every repository plus transactions. WithTx nests as a savepoint
// when called on a transactional store.
type Store interface {
	AccountRepo
	FiscalRepo
	JournalRepo
	BalanceRepo
	InvoiceRepo
	ReimbursementRepo
	InsightRepo
	AnalyticsRepo
	WithTx(ctx context.Context, fn func(tx Store) error) error
}

// ErrUnique is returned on a unique-constraint violation.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }
