package domain

import (
	"context"
	"encoding/json"
	"time"
)

// Document kinds.
const (
	DocPR        = "PR"
	DocPO        = "PO"
	DocQuotation = "QUOTATION"
	DocInvoice   = "INVOICE"
)

// Document sides.
const (
	SideB2B      = "B2B"
	SideSupplier = "SUPPLIER"
)

// Document statuses.
const (
	DocDraft     = "DRAFT"
	DocSubmitted = "SUBMITTED"
	DocApproved  = "APPROVED"
	DocRejected  = "REJECTED"
	DocSent      = "SENT"
	DocAccepted  = "ACCEPTED"
	DocConverted = "CONVERTED"
	DocIssued    = "ISSUED"
)

// Document actions a person takes.
const (
	ActSubmit  = "submit"
	ActApprove = "approve"
	ActReject  = "reject"
	ActReopen  = "reopen"
	ActSend    = "send"
	ActAccept  = "accept"
	ActDecline = "decline"
)

type docStep struct{ from, to string }

// docFlow is every status change a person can make. Converting and issuing
// have their own use cases because they create another record.
var docFlow = map[string]map[string]docStep{
	DocPR:        {ActSubmit: {DocDraft, DocSubmitted}, ActApprove: {DocSubmitted, DocApproved}, ActReject: {DocSubmitted, DocRejected}, ActReopen: {DocRejected, DocDraft}},
	DocPO:        {ActSubmit: {DocDraft, DocSubmitted}, ActApprove: {DocSubmitted, DocApproved}, ActReject: {DocSubmitted, DocRejected}, ActReopen: {DocRejected, DocDraft}},
	DocQuotation: {ActSend: {DocDraft, DocSent}, ActAccept: {DocSent, DocAccepted}, ActDecline: {DocSent, DocRejected}, ActReopen: {DocRejected, DocDraft}},
}

// DocTransition returns the next status, or false when the action does not apply.
func DocTransition(kind, status, action string) (string, bool) {
	step, ok := docFlow[kind][action]
	if !ok || step.from != status {
		return "", false
	}
	return step.to, true
}

// DocNeedsApprover marks actions that take finance.approve and a second person.
func DocNeedsApprover(action string) bool { return action == ActApprove || action == ActReject }

// ConvertTarget is the kind a document becomes and the status it must be in.
func ConvertTarget(kind string) (target, from string, ok bool) {
	switch kind {
	case DocPR:
		return DocPO, DocApproved, true
	case DocQuotation:
		return DocInvoice, DocAccepted, true
	}
	return "", "", false
}

// IssueReady reports whether a document can become a payable or receivable.
func IssueReady(d Document) bool {
	return (d.Kind == DocPO && d.Status == DocApproved) || (d.Kind == DocInvoice && d.Status == DocDraft)
}

// DocItem is a free-text line. Qty keeps two decimals in a Money.
type DocItem struct {
	Description string `json:"description"`
	Unit        string `json:"unit"`
	Qty         Money  `json:"qty"`
	UnitPrice   Money  `json:"unitPrice"`
	LineTotal   Money  `json:"lineTotal"`
}

// Company is the letterhead on a document.
type Company struct {
	LegalName, Address, Email, Phone, NPWP string
}

// Vendor is a supplier of goods or services.
type Vendor struct {
	ID, Name, PicName, Phone, Email, Address, NPWP  string
	BankName, BankAccountNo, BankAccountName, Notes string
	PaymentTermsDays                                int
	IsActive                                        bool
	By                                              string
}

// Document is a PR, PO, quotation or invoice. Percentages keep two decimals in
// a Money: 11% is Rupiah(11).
type Document struct {
	ID, Kind, Side, Number, Status                      string
	VendorID, PartyName, PicName, Phone, Email, Address string
	Title, IssueDate, DueDate, ExternalNumber, Notes    string
	Items                                               []DocItem
	DiscountPercent, PPNPercent                         Money
	Subtotal, Discount, Tax, Total                      Money
	AccountID, SourceID, LedgerInvoiceID, RejectReason  string
	Letterhead                                          json.RawMessage
	DecidedBy                                           string
	DecidedAt                                           *time.Time
	CreatedBy, UpdatedBy                                string
	CreatedAt                                           time.Time
}

// DocumentFilter selects documents of one kind.
type DocumentFilter struct {
	Kind, Status, Query string
	Limit, Offset       int
}

func mulHundredths(a, b Money) Money { return Money((int64(a)*int64(b) + 5000) / 10000) }

// Recalculate fills line totals and document totals. The discount applies
// before tax, the way Indonesian invoices state PPN.
func (d *Document) Recalculate() {
	d.Subtotal = 0
	for i := range d.Items {
		// Qty and price both hold hundredths, so the product divides by 100.
		d.Items[i].LineTotal = Money((int64(d.Items[i].Qty)*int64(d.Items[i].UnitPrice) + 50) / 100)
		d.Subtotal += d.Items[i].LineTotal
	}
	d.Discount = mulHundredths(d.Subtotal, d.DiscountPercent)
	d.Tax = mulHundredths(d.Subtotal-d.Discount, d.PPNPercent)
	d.Total = d.Subtotal - d.Discount + d.Tax
}

// DocumentRepo stores vendors and documents.
type DocumentRepo interface {
	ListVendors(ctx context.Context, query string, activeOnly bool) ([]Vendor, error)
	GetVendor(ctx context.Context, id string) (*Vendor, error)
	SaveVendor(ctx context.Context, v Vendor) error
	ListDocuments(ctx context.Context, f DocumentFilter) ([]Document, int, error)
	GetDocument(ctx context.Context, id string) (*Document, error)
	InsertDocument(ctx context.Context, d Document) error
	UpdateDocument(ctx context.Context, d Document) error
	DeleteDocument(ctx context.Context, id string) error
}
