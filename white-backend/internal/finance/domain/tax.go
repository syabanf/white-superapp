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

// DocTax is the copy of a tax on a document, with its result.
type DocTax struct {
	TaxID           string `json:"taxId"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Rate            Money  `json:"rate"`
	BaseNumerator   int    `json:"baseNumerator"`
	BaseDenominator int    `json:"baseDenominator"`
	PriceIncluded   bool   `json:"priceIncluded"`
	AccountID       string `json:"accountId"`
	Base            Money  `json:"base"`
	Amount          Money  `json:"amount"`
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
	ListChargeTypes(ctx context.Context, activeOnly bool) ([]ChargeType, error)
	GetChargeType(ctx context.Context, id string) (*ChargeType, error)
	SaveChargeType(ctx context.Context, c ChargeType) error
	InsertInvoicePosting(ctx context.Context, p InvoicePosting) error
	ListInvoicePostings(ctx context.Context, invoiceID string) ([]InvoicePosting, error)
	IsDocumentInvoice(ctx context.Context, invoiceID string) (bool, error)
	TaxRegister(ctx context.Context, from, to string) ([]TaxRegisterRow, error)
}
