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

	// CompanyProfile is nil until someone saves it.
	CompanyProfile(ctx context.Context) (*Company, error)
	SaveCompanyProfile(ctx context.Context, c Company, by string) error
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

// SalesReversal is a payment or invoice Sales removed that Finance has not reversed.
type SalesReversal struct {
	Kind, ID, DocNo, PartyName, Reason, Error string
	Amount                                    Money
}

// SalesPaymentDeleted is the payload of white.sales.payment.deleted.
type SalesPaymentDeleted struct {
	PaymentID string `json:"paymentId"`
	DeletedBy string `json:"deletedBy,omitempty"`
}

// SalesInvoiceCancelled is the payload of white.sales.invoice.cancelled, sent only
// for an invoice Finance already booked.
type SalesInvoiceCancelled struct {
	InvoiceID   string `json:"invoiceId"`
	Reason      string `json:"reason,omitempty"`
	CancelledBy string `json:"cancelledBy,omitempty"`
}

// GoodsReceipt records what a vendor delivered against a purchase order, before tax.
type GoodsReceipt struct {
	ID, ReceiptNo, DocumentID, Date, Notes, Status, VoidReason, CreatedBy string
	Amount                                                                Money
}

// POInvoice is an AP invoice matched to a purchase order.
type POInvoice struct {
	DocumentID, InvoiceID, InvoiceNo, PartyName, Status string
	Subtotal                                            Money
}

// Kinds of cost allocated to a project.
const (
	AllocPayroll       = "PAYROLL"
	AllocReimbursement = "REIMBURSEMENT"
)

// ProjectAllocation assigns payroll or reimbursement cost to a project for a month.
type ProjectAllocation struct {
	ID, TagID, TagName, ClientName, Month, Kind, Notes, CreatedBy string
	Amount                                                        Money
	// Source is MANUAL or TIMESHEET (replaced on each recalculation).
	Source string
}

const (
	AllocManual    = "MANUAL"
	AllocTimesheet = "TIMESHEET"
)

// ControlRepo stores the links and records of the remaining ERP controls.
type ControlRepo interface {
	SetPaymentSalesReversal(ctx context.Context, id, reason, failure string) error
	SetInvoiceSalesReversal(ctx context.Context, id, reason, failure string, settled bool) error
	PendingSalesReversals(ctx context.Context) ([]SalesReversal, error)
	DisposeFixedAsset(ctx context.Context, id, date, entryID, by string, proceeds Money) error
	ListGoodsReceipts(ctx context.Context, documentID string) ([]GoodsReceipt, error)
	GetGoodsReceipt(ctx context.Context, id string) (*GoodsReceipt, error)
	InsertGoodsReceipt(ctx context.Context, r GoodsReceipt) error
	VoidGoodsReceipt(ctx context.Context, id, reason string) error
	SetInvoicePOLink(ctx context.Context, invoiceID, documentID string) error
	InvoicePOLink(ctx context.Context, invoiceID string) (string, error)
	POInvoices(ctx context.Context, documentID string) ([]POInvoice, error)
	ListProjectAllocations(ctx context.Context, fromMonth, toMonth string) ([]ProjectAllocation, error)
	InsertProjectAllocation(ctx context.Context, a ProjectAllocation) error
	DeleteProjectAllocation(ctx context.Context, id string) error
	DeleteTimesheetAllocations(ctx context.Context, month string) error
	InsertInvoiceReminder(ctx context.Context, invoiceID string, stage int) (bool, error)
}
