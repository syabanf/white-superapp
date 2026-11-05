package domain

import "context"

// ProjectTag groups transactions into one project of one client.
type ProjectTag struct {
	ID, Name, ClientName, DealID, Notes, UpdatedBy string
	IsActive                                       bool
}

// ProjectProfitRow is one tag's result; an empty TagID holds untagged journals.
type ProjectProfitRow struct {
	TagID, TagName, ClientName string
	Revenue, Cost              Money
}

// RecurringInvoice issues the same invoice on a schedule.
type RecurringInvoice struct {
	ID, Kind, Name, PartyName, ClientID, AccountID, TagID, Description string
	Subtotal, TaxAmount                                                Money
	TaxIDs                                                             []string
	DueDays, AnchorDay                                                 int
	Frequency, NextRun, EndDate, LastInvoiceID, LastError              string
	AutoPost, IsActive                                                 bool
	CreatedBy                                                          string
}

// Advance is money paid or received before the invoice exists.
type Advance struct {
	ID, Kind, AdvanceNo, Date, PartyName, Reference, Notes, Status string
	CashAccountID, CashAccountName                                 string
	AdvanceAccountID, AdvanceAccountName                           string
	JournalEntryID, VoidReason, CreatedBy                          string
	Amount, Applied                                                Money
}

// Remaining is what can still be applied to invoices.
func (a Advance) Remaining() Money {
	if a.Status != StatusPosted {
		return 0
	}
	return a.Amount - a.Applied
}

// PlanningRepo stores project tags, recurring invoices and advances.
type PlanningRepo interface {
	ListProjectTags(ctx context.Context, activeOnly bool) ([]ProjectTag, error)
	GetProjectTag(ctx context.Context, id string) (*ProjectTag, error)
	SaveProjectTag(ctx context.Context, t ProjectTag) error
	EnsureDealTag(ctx context.Context, t ProjectTag) (string, error)
	ProjectTagInUse(ctx context.Context, id string) (bool, error)
	DeleteProjectTag(ctx context.Context, id string) error
	ProjectProfit(ctx context.Context, from, to, clientName string) ([]ProjectProfitRow, error)

	ListRecurringInvoices(ctx context.Context) ([]RecurringInvoice, error)
	DueRecurringInvoices(ctx context.Context, today string) ([]RecurringInvoice, error)
	GetRecurringInvoice(ctx context.Context, id string) (*RecurringInvoice, error)
	SaveRecurringInvoice(ctx context.Context, r RecurringInvoice, by string) error
	UpdateRecurringInvoiceRun(ctx context.Context, r RecurringInvoice) error
	SetRecurringInvoiceActive(ctx context.Context, id string, active bool, by string) error
	DeleteRecurringInvoice(ctx context.Context, id string) error

	ListAdvances(ctx context.Context, kind string) ([]Advance, error)
	GetAdvance(ctx context.Context, id string) (*Advance, error)
	LockAdvance(ctx context.Context, id string) error
	InsertAdvance(ctx context.Context, a Advance) error
	VoidAdvance(ctx context.Context, id, reason string) error
	InsertAdvanceApplication(ctx context.Context, paymentID, advanceID string) error
}
