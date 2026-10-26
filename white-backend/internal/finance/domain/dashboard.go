package domain

import (
	"context"
	"time"
)

// Read models for the dashboard tabs (new for WHITE).

// MonthMovement is one account's posted lines in one month (YYYY-MM).
type MonthMovement struct {
	Month string
	AccountBalance
	ParentID, ParentName, GrandparentID string
}

// AccountParent places an account under its parent and grandparent.
type AccountParent struct {
	ID, Code, Name                      string
	ParentID, ParentName, GrandparentID string
}

// InvoiceFact is a posted AP or AR invoice.
type InvoiceFact struct {
	ID, Kind, InvoiceNo, InvoiceDate, DueDate, PartyName string
	Subtotal, Total                                      Money
}

// AllocationFact is a posted payment applied to an invoice.
type AllocationFact struct {
	InvoiceID, PaymentDate string
	Amount                 Money
}

// ClaimFact is a reimbursement claim with its workflow timestamps.
type ClaimFact struct {
	ID, ClaimNo, EmployeeName, DepartmentName, Category, TransactionDate, Status string
	Amount                                                                       Money
	SubmittedAt, ManagerApprovedAt, FinanceApprovedAt, PaidAt, RejectedAt        *time.Time
}

type AnalyticsRepo interface {
	MonthlyMovements(ctx context.Context, from, to string) ([]MonthMovement, error)
	AccountParents(ctx context.Context) ([]AccountParent, error)
	InvoiceFacts(ctx context.Context, asOf string) ([]InvoiceFact, error)
	AllocationFacts(ctx context.Context, asOf string) ([]AllocationFact, error)
	// ClaimFacts are claims dated in [from, to] plus every claim still
	// waiting, optionally for one department.
	ClaimFacts(ctx context.Context, from, to, department string) ([]ClaimFact, error)
	ClaimDepartments(ctx context.Context) ([]string, error)
}

// Benchmarks are healthy thresholds for a services agency, by ratio key.
// They draw the reference line on the ratio trends.
var Benchmarks = map[string]float64{
	"gross_margin": 40, "operating_margin": 15, "net_margin": 10,
	"current_ratio": 1.5, "quick_ratio": 1, "cash_ratio": 0.5,
	"debt_to_equity": 1, "dso": 45, "dpo": 30, "runway_months": 6,
}
