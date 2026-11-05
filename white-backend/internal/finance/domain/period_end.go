package domain

import "context"

// Which side of the books a deferral sits on.
const (
	DeferRevenue = "REVENUE"
	DeferExpense = "EXPENSE"
)

// Deferral recognizes an amount evenly over its months, ending each month.
type Deferral struct {
	ID, Kind, Name, PartyName, Reference   string
	DeferredAccountID, DeferredAccountName string
	TargetAccountID, TargetAccountName     string
	TagID, StartDate, LastEntryID          string
	LastError, CreatedBy                   string
	Months, PostedMonths                   int
	Amount                                 Money
	IsActive                               bool
}

// AmountFor is month n's share (1-based).
func (d Deferral) AmountFor(n int) Money {
	return straightLineTotal(d.Amount, d.Months, n) - straightLineTotal(d.Amount, d.Months, n-1)
}

// Recognized is what has reached the target account so far.
func (d Deferral) Recognized() Money { return straightLineTotal(d.Amount, d.Months, d.PostedMonths) }

// MonthEnd is the posting date of month n (1-based). The start month is month 1.
func (d Deferral) MonthEnd(n int) (string, error) { return monthEndAfter(d.StartDate, n) }

// Lines moves one month out of the deferred account.
func (d Deferral) Lines(amount Money) []Line {
	first, second := Line{AccountID: d.DeferredAccountID, Side: Debit, Amount: amount, Memo: d.Name}, Line{AccountID: d.TargetAccountID, Side: Credit, Amount: amount, Memo: d.Name, SortOrder: 1}
	if d.Kind == DeferExpense {
		first.Side, second.Side = Credit, Debit
	}
	return []Line{first, second}
}

// TaxPosition is one tax account's balance: PAYABLE owed to the state, PREPAID
// credited against it.
type TaxPosition struct {
	AccountID, Code, Name, Kind string
	Balance                     Money
}

// TaxRemittance is tax paid to the state.
type TaxRemittance struct {
	ID, RemitNo, Period, Date, AccountID, AccountName string
	CashAccountID, CashAccountName, NTPN, Notes       string
	Status, JournalEntryID, VoidReason, CreatedBy     string
	Amount                                            Money
}

// PeriodEndRepo stores deferrals, tax remittances and control settings.
type PeriodEndRepo interface {
	ListDeferrals(ctx context.Context, activeOnly bool) ([]Deferral, error)
	GetDeferral(ctx context.Context, id string) (*Deferral, error)
	InsertDeferral(ctx context.Context, d Deferral) error
	UpdateDeferralRun(ctx context.Context, d Deferral) error
	SetDeferralActive(ctx context.Context, id string, active bool) error
	DeleteDeferral(ctx context.Context, id string) error

	ListTaxRemittances(ctx context.Context) ([]TaxRemittance, error)
	GetTaxRemittance(ctx context.Context, id string) (*TaxRemittance, error)
	InsertTaxRemittance(ctx context.Context, r TaxRemittance) error
	VoidTaxRemittance(ctx context.Context, id, reason string) error

	SeparateApprover(ctx context.Context) (bool, error)
	SetSeparateApprover(ctx context.Context, on bool, by string) error
}

// Results of an audited action.
const (
	AuditOK      = "OK"
	AuditRefused = "REFUSED"
)

// AuditEntry is one change someone or the system asked Finance to make.
type AuditEntry struct {
	ID, ActorID, ActorName, Action, DocID, DocNo string
	Result, Message, Request, Response, At       string
}

// AuditFilter narrows the trail.
type AuditFilter struct {
	Search, Result, From, To string
	Limit, Offset            int
}

// AuditRepo appends to the trail and reads it. The table refuses edits and deletes.
type AuditRepo interface {
	InsertAuditEntry(ctx context.Context, e AuditEntry) error
	ListAuditEntries(ctx context.Context, f AuditFilter) ([]AuditEntry, int, error)
}
