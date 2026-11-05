package domain

import (
	"context"
	"math/big"
)

// Tax kinds.
const (
	TaxVAT         = "VAT"
	TaxWithholding = "WITHHOLDING"
)

// Where a tax or charge applies.
const (
	ScopeSale     = "SALE"
	ScopePurchase = "PURCHASE"
	ScopeBoth     = "BOTH"
)

// Charge computations.
const (
	ChargeFixed   = "FIXED"
	ChargePercent = "PERCENT"
)

// TaxCode is a tax as master data. Rate holds hundredths of a percent in a
// Money: 12% is Rupiah(12). BaseNumerator/BaseDenominator is the share of the base
// the rate applies to, such as 11/12.
type TaxCode struct {
	ID, Code, Name, Kind, Scope      string
	Rate                             Money
	BaseNumerator, BaseDenominator   int
	PriceIncluded                    bool
	AccountSaleID, AccountPurchaseID string
	ReportCode, ValidFrom, ValidTo   string
	IsDefaultSale, IsDefaultPurchase bool
	IsActive                         bool
	Notes, UpdatedBy                 string
	// Brackets replace the flat rate when present.
	Brackets []TaxBracket
	// CashBasis holds VAT on TransitionAccountID until the invoice is paid.
	CashBasis           bool
	TransitionAccountID string
}

// ScopeFits reports whether a tax or charge can be used on a side.
func ScopeFits(scope, side string) bool {
	if scope == ScopeBoth {
		return true
	}
	return (scope == ScopeSale) == (side == SideB2B)
}

// AccountFor is the posting account on a side.
func (t TaxCode) AccountFor(side string) string {
	if side == SideB2B {
		return t.AccountSaleID
	}
	return t.AccountPurchaseID
}

// ChargeType is an extra charge as master data.
type ChargeType struct {
	ID, Code, Name, Scope, Computation string
	Value                              Money
	Taxable, IsDeduction, IsActive     bool
	AccountSaleID, AccountPurchaseID   string
	Notes, UpdatedBy                   string
}

// AccountFor is the posting account on a side; empty posts with the main line.
func (c ChargeType) AccountFor(side string) string {
	if side == SideB2B {
		return c.AccountSaleID
	}
	return c.AccountPurchaseID
}

// TaxBracket is one step of a progressive rate. UpTo is the top of the step on
// the whole base; zero means no limit.
type TaxBracket struct {
	UpTo Money `json:"upTo"`
	Rate Money `json:"rate"`
}

// ProgressiveTax charges each slice of the base at its own step's rate.
func ProgressiveTax(base Money, brackets []TaxBracket) Money {
	var tax, floor Money
	for _, b := range brackets {
		top := b.UpTo
		if top == 0 || top > base {
			top = base
		}
		if top > floor {
			tax += mulHundredths(top-floor, b.Rate)
			floor = top
		}
	}
	return tax
}

// ValidBrackets requires rising limits and an open last step.
func ValidBrackets(brackets []TaxBracket) bool {
	var last Money
	for i, b := range brackets {
		if b.Rate < 0 || b.Rate > Rupiah(100) {
			return false
		}
		if i == len(brackets)-1 {
			return b.UpTo == 0
		}
		if b.UpTo <= last {
			return false
		}
		last = b.UpTo
	}
	return true
}

// FiscalRule swaps a tax for a party: an empty target removes it.
type FiscalRule struct {
	FromTaxID string `json:"fromTaxId"`
	ToTaxID   string `json:"toTaxId"`
}

// FiscalPosition is a named set of tax swaps assigned to parties.
type FiscalPosition struct {
	ID, Name, Notes, UpdatedBy string
	Rules                      []FiscalRule
	IsActive                   bool
}

// PartyProfile is the tax identity of a client or vendor, matched by name
// because invoices hold the party as text.
type PartyProfile struct {
	ID, Side, Name, NPWP, Address, FiscalPositionID, Notes, UpdatedBy string
	IsPKP                                                             bool
}

// TaxMove is one signed line of the tax ledger.
type TaxMove struct {
	Kind, SourceType, SourceID, InvoiceID, TaxCode, TaxName, TaxKind, ReportCode, Date string
	TaxRate, Base, Amount                                                              Money
}

// Tax move sources.
const (
	MoveInvoice     = "INVOICE"
	MoveCreditNote  = "CREDIT_NOTE"
	MoveCash        = "CASH"
	MovePayment     = "PAYMENT"
	MovePaymentVoid = "PAYMENT_VOID"
	MoveCashVoid    = "CASH_VOID"
)

// TaxExportRow is a tax ledger line with the invoice and the party's identity.
type TaxExportRow struct {
	Date, TaxCode, TaxName, ReportCode, InvoiceNo, Reference, Party, Description, NPWP, Address string
	TaxRate, Base, Amount, Subtotal                                                             Money
}

// MulDiv is a*b/c rounded half up.
func MulDiv(a, b, c Money) Money { return mulDiv(int64(a), int64(b), int64(c)) }

// DocTax is the copy of a tax on a document, with its result.
type DocTax struct {
	TaxID               string       `json:"taxId"`
	Code                string       `json:"code"`
	Name                string       `json:"name"`
	Kind                string       `json:"kind"`
	Rate                Money        `json:"rate"`
	BaseNumerator       int          `json:"baseNumerator"`
	BaseDenominator     int          `json:"baseDenominator"`
	PriceIncluded       bool         `json:"priceIncluded"`
	AccountID           string       `json:"accountId"`
	ReportCode          string       `json:"reportCode,omitempty"`
	Brackets            []TaxBracket `json:"brackets,omitempty"`
	CashBasis           bool         `json:"cashBasis,omitempty"`
	TransitionAccountID string       `json:"transitionAccountId,omitempty"`
	Base                Money        `json:"base"`
	Amount              Money        `json:"amount"`
}

// DocCharge is the copy of a charge on a document. Amount is signed: a
// deduction is negative.
type DocCharge struct {
	ChargeID    string `json:"chargeId"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Computation string `json:"computation"`
	Value       Money  `json:"value"`
	Taxable     bool   `json:"taxable"`
	IsDeduction bool   `json:"isDeduction"`
	AccountID   string `json:"accountId"`
	Amount      Money  `json:"amount"`
}

// mulDiv is a*b/c rounded half up, safe from overflow.
func mulDiv(a, b, c int64) Money {
	if c == 0 {
		return 0
	}
	n := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	d := big.NewInt(c)
	n.Mul(n, big.NewInt(2)).Add(n, d)
	n.Div(n, new(big.Int).Mul(d, big.NewInt(2)))
	return Money(n.Int64())
}

func (t DocTax) ratio() (int64, int64) {
	if t.BaseDenominator <= 0 || t.BaseNumerator <= 0 {
		return 1, 1
	}
	return int64(t.BaseNumerator), int64(t.BaseDenominator)
}

// InvoicePosting is one journal line an invoice adds to its mapping.
type InvoicePosting struct {
	ID, InvoiceID, AccountID, Side, Memo string
	Amount                               Money
	SortOrder                            int
	TaxCode, TaxName, TaxKind            string
	TaxBase                              Money
	TaxID, ReportCode, FinalAccountID    string
	TaxRate                              Money
	// CashBasis parks the tax on the transition account until payment.
	CashBasis bool
}

// TaxRegisterRow is one tax code in a period, per payable or receivable side.
type TaxRegisterRow struct {
	Kind, TaxCode, TaxName, TaxKind string
	InvoiceCount                    int
	Base, Amount                    Money
}

// TaxRepo stores tax master data and the postings made from it.
type TaxRepo interface {
	ListTaxCodes(ctx context.Context, activeOnly bool) ([]TaxCode, error)
	GetTaxCode(ctx context.Context, id string) (*TaxCode, error)
	SaveTaxCode(ctx context.Context, t TaxCode) error
	TaxCodeInUse(ctx context.Context, id, code string) (bool, error)
	DeleteTaxCode(ctx context.Context, id string) error
	ListChargeTypes(ctx context.Context, activeOnly bool) ([]ChargeType, error)
	GetChargeType(ctx context.Context, id string) (*ChargeType, error)
	SaveChargeType(ctx context.Context, c ChargeType) error
	ChargeTypeInUse(ctx context.Context, id string) (bool, error)
	DeleteChargeType(ctx context.Context, id string) error
	InsertInvoicePosting(ctx context.Context, p InvoicePosting) error
	ListInvoicePostings(ctx context.Context, invoiceID string) ([]InvoicePosting, error)
	IsDocumentInvoice(ctx context.Context, invoiceID string) (bool, error)
	TaxRegister(ctx context.Context, from, to string) ([]TaxRegisterRow, error)
	GetTaxCodeByCode(ctx context.Context, code string) (*TaxCode, error)
	DeleteInvoicePostings(ctx context.Context, invoiceID string) error
	InsertTaxMove(ctx context.Context, m TaxMove) error
	TaxExportRows(ctx context.Context, kind, taxKind, from, to string) ([]TaxExportRow, error)
	ListFiscalPositions(ctx context.Context) ([]FiscalPosition, error)
	GetFiscalPosition(ctx context.Context, id string) (*FiscalPosition, error)
	SaveFiscalPosition(ctx context.Context, f FiscalPosition) error
	FiscalPositionInUse(ctx context.Context, id string) (bool, error)
	DeleteFiscalPosition(ctx context.Context, id string) error
	ListPartyProfiles(ctx context.Context) ([]PartyProfile, error)
	GetPartyProfile(ctx context.Context, side, name string) (*PartyProfile, error)
	SavePartyProfile(ctx context.Context, p PartyProfile) error
	DeletePartyProfile(ctx context.Context, id string) error
}
