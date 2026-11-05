package domain

// Ways an invoice settles without the full amount moving through cash.
const (
	AdjustFee         = "FEE"
	AdjustDiscount    = "DISCOUNT"
	AdjustWithholding = "WITHHOLDING"
	// AdjustExcess is cash above the invoice, kept for the party as an advance.
	AdjustExcess = "EXCESS"
)

// PaymentAdjustment is one such part of a payment.
type PaymentAdjustment struct {
	Kind, AccountID, AccountName, Memo string
	TaxCode, TaxName, ReportCode       string
	TaxRate, TaxBase, Amount           Money
}

// raisesCash reports whether the adjustment adds to the cash moved: excess cash and
// a bank fee we pay on top of a vendor payment. A fee the bank takes from a client's
// transfer, a discount and withheld tax all leave less cash.
func (a PaymentAdjustment) raisesCash(kind string) bool {
	return a.Kind == AdjustExcess || (a.Kind == AdjustFee && kind == KindAP)
}

// CashMoved is what left or entered the bank to settle amount.
func CashMoved(kind string, amount Money, adjustments []PaymentAdjustment) Money {
	for _, a := range adjustments {
		if a.raisesCash(kind) {
			amount += a.Amount
		} else {
			amount -= a.Amount
		}
	}
	return amount
}

// SettlementLines splits the mapping's cash line. On a receipt every adjustment
// sits with the cash on the debit side. On a vendor payment a discount and withheld
// tax sit with the cash on the credit side, and a fee is an extra debit. Excess cash
// is an extra line on the advance account: a credit on a receipt, a debit on a vendor
// payment. The bank shows one line for the whole transfer.
func SettlementLines(kind string, amount Money, adjustments []PaymentAdjustment) ([]Split, []Line) {
	splits := []Split{{Amount: CashMoved(kind, amount, adjustments)}}
	var extra []Line
	for i, a := range adjustments {
		if a.raisesCash(kind) {
			side := Debit
			if a.Kind == AdjustExcess && kind == KindAR {
				side = Credit
			}
			extra = append(extra, Line{AccountID: a.AccountID, Side: side, Amount: a.Amount, Memo: a.Memo, SortOrder: 100 + i})
			continue
		}
		splits = append(splits, Split{AccountID: a.AccountID, Amount: a.Amount})
	}
	return splits, extra
}

// AdvanceRefund returns unused advance money.
type AdvanceRefund struct {
	ID, AdvanceID, RefundNo, Date, CashAccountID, CashAccountName string
	Reason, Status, JournalEntryID, VoidReason, CreatedBy         string
	Amount                                                        Money
}
