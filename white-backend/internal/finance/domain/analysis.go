package domain

import (
	"math"
	"strings"
)

// Financial ratios from ledger totals (new for Mixist; NüHabit's dashboard
// computed none).

// LedgerSnapshot holds what one period's ratios need.
type LedgerSnapshot struct {
	Days int
	// Income statement over the period.
	Revenue, COGS, Expenses, OtherIncome, OtherExpenses Money
	// Balance sheet at the period end.
	CurrentAssets, CurrentLiabilities, Cash, Receivables, TradePayables Money
	TotalLiabilities, TotalEquity                                       Money
	// Cash at the day before the period start.
	CashOpening Money
}

func (s LedgerSnapshot) GrossProfit() Money     { return s.Revenue - s.COGS }
func (s LedgerSnapshot) OperatingIncome() Money { return s.GrossProfit() - s.Expenses }
func (s LedgerSnapshot) NetIncome() Money {
	return s.OperatingIncome() + s.OtherIncome - s.OtherExpenses
}

// Months is the period length in months (30.4375 days each), at least 1/30.
func (s LedgerSnapshot) Months() float64 {
	return math.Max(float64(s.Days), 1) / 30.4375
}

// Ratio is a computed value; OK is false when the denominator is zero.
type Ratio struct {
	Value float64
	OK    bool
}

func div(n, d float64) Ratio {
	if d == 0 || math.IsNaN(n) || math.IsNaN(d) {
		return Ratio{}
	}
	return Ratio{Value: n / d, OK: true}
}

func pct(n, d Money) Ratio {
	r := div(n.Float(), d.Float())
	r.Value *= 100
	return r
}

// Ratios is the full set for one period.
type Ratios struct {
	GrossMargin, OperatingMargin, NetMargin Ratio
	CurrentRatio, QuickRatio, CashRatio     Ratio
	DebtToEquity                            Ratio
	DSO, DPO                                Ratio
	// Average monthly net cash outflow (0 when cash grew) and runway months.
	BurnRate Money
	Runway   Ratio
}

// ComputeRatios applies the textbook formulas:
//   - margins over revenue, in percent
//   - current = CA / CL, quick = (cash + receivables) / CL, cash = cash / CL
//   - debt to equity = total liabilities / total equity (incl. current earnings)
//   - DSO = receivables / revenue x days; DPO = trade payables / (COGS +
//     operating expenses) x days, since an agency books little COGS
//   - burn = (opening cash - closing cash) / months when cash fell;
//     runway = cash / burn
func ComputeRatios(s LedgerSnapshot) Ratios {
	r := Ratios{
		GrossMargin:     pct(s.GrossProfit(), s.Revenue),
		OperatingMargin: pct(s.OperatingIncome(), s.Revenue),
		NetMargin:       pct(s.NetIncome(), s.Revenue),
		CurrentRatio:    div(s.CurrentAssets.Float(), s.CurrentLiabilities.Float()),
		QuickRatio:      div((s.Cash + s.Receivables).Float(), s.CurrentLiabilities.Float()),
		CashRatio:       div(s.Cash.Float(), s.CurrentLiabilities.Float()),
		DebtToEquity:    div(s.TotalLiabilities.Float(), s.TotalEquity.Float()),
		DSO:             div(s.Receivables.Float()*float64(s.Days), s.Revenue.Float()),
		DPO:             div(s.TradePayables.Float()*float64(s.Days), (s.COGS + s.Expenses).Float()),
	}
	if fell := s.CashOpening - s.Cash; fell > 0 {
		r.BurnRate = FromFloat(fell.Float() / s.Months())
		r.Runway = div(s.Cash.Float(), r.BurnRate.Float())
	}
	return r
}

// Account classification by the 7-digit code, matching the seeded
// Indonesian chart: 11xxxxx current assets, 1103xxx receivables, 21xxxxx
// current liabilities, 2101xxx trade payables.
func IsCurrentAsset(code string) bool     { return strings.HasPrefix(code, "11") }
func IsReceivable(code string) bool       { return strings.HasPrefix(code, "1103") }
func IsCurrentLiability(code string) bool { return strings.HasPrefix(code, "21") }
func IsTradePayable(code string) bool     { return strings.HasPrefix(code, "2101") }
func IsExpenseType(typeCode string) bool {
	return typeCode == TypeExpense || typeCode == TypeCOGS || typeCode == TypeOtherExpense
}
