// Package salesevents defines the JSON payloads the sales service
// publishes. Consumers (finance for invoices and payments, identity for
// onboarding) decode these shapes; keep the field names stable. Amounts
// are decimal strings ("1250000.00"), days are "YYYY-MM-DD".
package salesevents

import "time"

// InvoiceIssued is published on events.SalesInvoiceIssued when finance
// issues a submitted invoice (diajukan -> terkirim). It matches
// finance/domain.SalesInvoiceIssued: finance creates one AR invoice per
// invoiceId (the idempotency key).
type InvoiceIssued struct {
	InvoiceID     string `json:"invoiceId"`
	InvoiceNumber string `json:"invoiceNumber"`
	DealID        string `json:"dealId,omitempty"`
	// ClientID is the marketing-platform client once onboarding created it.
	ClientID     string `json:"clientId,omitempty"`
	CustomerName string `json:"customerName"`
	IssueDate    string `json:"issueDate"`
	DueDate      string `json:"dueDate,omitempty"`
	// Subtotal + TaxAmount = TotalAmount; the PPN share is pro rata from
	// the quotation.
	Subtotal    string `json:"subtotal"`
	TaxAmount   string `json:"taxAmount"`
	TotalAmount string `json:"totalAmount"`
	Description string `json:"description,omitempty"`
	IssuedBy    string `json:"issuedBy,omitempty"`
}

// PaymentRecorded is published on events.SalesPaymentRecorded for a
// payment recorded against an invoice. It matches
// finance/domain.SalesPaymentRecorded: one AR receipt per paymentId. A
// payment may arrive before finance has processed its invoice.
type PaymentRecorded struct {
	PaymentID  string `json:"paymentId"`
	InvoiceID  string `json:"invoiceId"`
	Amount     string `json:"amount"`
	PaidOn     string `json:"paidOn"`
	Method     string `json:"method,omitempty"`
	Reference  string `json:"reference,omitempty"`
	RecordedBy string `json:"recordedBy,omitempty"`
}

// DealWon is published on events.SalesDealWon when a deal enters a won
// stage. Sales starts the client onboarding from it.
type DealWon struct {
	DealID       string    `json:"dealId"`
	Title        string    `json:"title"`
	LeadID       string    `json:"leadId,omitempty"`
	AccountID    string    `json:"accountId,omitempty"`
	ClientName   string    `json:"clientName"`
	PicName      string    `json:"picName,omitempty"`
	PicEmail     string    `json:"picEmail,omitempty"`
	PicPhone     string    `json:"picPhone,omitempty"`
	ValueFinal   string    `json:"valueFinal"`
	ServiceLines []string  `json:"serviceLines"`
	OwnerUserID  string    `json:"ownerUserId,omitempty"`
	WonAt        time.Time `json:"wonAt"`
}
