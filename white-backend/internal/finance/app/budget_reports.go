package app

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// BudgetRow compares one profit and loss account with its plan.
type BudgetRow struct {
	AccountID, Code, Name, TypeCode string
	Budget, Actual, Variance        domain.Money
}

// BudgetVsActual covers whole months: the range snaps to month starts and ends
// because a budget is monthly.
type BudgetVsActual struct {
	From, To                     string
	Rows                         []BudgetRow
	BudgetRevenue, ActualRevenue domain.Money
	BudgetCost, ActualCost       domain.Money
}

func isRevenueType(t string) bool { return t == domain.TypeRevenue || t == domain.TypeOtherIncome }

func (r *Reports) BudgetVsActual(ctx context.Context, from, to string) (*BudgetVsActual, error) {
	from, to, err := r.period(from, to)
	if err != nil {
		return nil, err
	}
	start, _ := time.Parse(time.DateOnly, from)
	end, _ := time.Parse(time.DateOnly, to)
	from = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
	to = time.Date(end.Year(), end.Month()+1, 0, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)

	budgets, err := r.Store.ListBudgets(ctx, from, to)
	if err != nil {
		return nil, err
	}
	planned := map[string]domain.Money{}
	for _, b := range budgets {
		planned[b.AccountID] += b.Amount
	}
	balances, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: from, To: to})
	if err != nil {
		return nil, err
	}
	out := &BudgetVsActual{From: from, To: to, Rows: []BudgetRow{}}
	for _, b := range balances {
		if !domain.IsProfitLossType(b.TypeCode) {
			continue
		}
		budget, actual := planned[b.ID], b.Net()
		if budget == 0 && actual == 0 {
			continue
		}
		out.Rows = append(out.Rows, BudgetRow{AccountID: b.ID, Code: b.Code, Name: b.Name, TypeCode: b.TypeCode, Budget: budget, Actual: actual,
			Variance: domain.BudgetVariance(b.TypeCode, budget, actual)})
		if isRevenueType(b.TypeCode) {
			out.BudgetRevenue += budget
			out.ActualRevenue += actual
		} else {
			out.BudgetCost += budget
			out.ActualCost += actual
		}
	}
	return out, nil
}

func budgetYearRange(year int) (string, string, error) {
	if year < 2000 || year > 2100 {
		return "", "", apperr.Field("year", "Tahun tidak valid")
	}
	return fmt.Sprintf("%d-01-01", year), fmt.Sprintf("%d-12-01", year), nil
}

func (r *Reports) BudgetYear(ctx context.Context, year int) ([]domain.Budget, error) {
	from, to, err := budgetYearRange(year)
	if err != nil {
		return nil, err
	}
	return r.Store.ListBudgets(ctx, from, to)
}

// SaveBudgetYear stores the given cells together. Only profit and loss accounts
// can have a budget, and every month must belong to the year.
func (r *Reports) SaveBudgetYear(ctx context.Context, userID string, year int, entries []domain.Budget) error {
	if _, _, err := budgetYearRange(year); err != nil {
		return err
	}
	balances, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{AsOf: r.today()})
	if err != nil {
		return err
	}
	plAccount := map[string]bool{}
	for _, b := range balances {
		plAccount[b.ID] = domain.IsProfitLossType(b.TypeCode)
	}
	prefix := fmt.Sprintf("%d-", year)
	for _, e := range entries {
		if t, err := time.Parse("2006-01", e.Month); err != nil || e.Month[:5] != prefix || t.IsZero() {
			return apperr.Field("month", "Bulan harus berada di tahun anggaran")
		}
		if !plAccount[e.AccountID] {
			return apperr.Field("account_id", "Anggaran hanya untuk akun pendapatan dan beban")
		}
		if e.Amount < 0 {
			return apperr.Field("amount", "Anggaran tidak boleh negatif")
		}
	}
	return r.Store.WithTx(ctx, func(tx domain.Store) error {
		for _, e := range entries {
			if err := tx.SaveBudget(ctx, e, userID); err != nil {
				return err
			}
		}
		return nil
	})
}

// EquityRow is one equity line over a period.
type EquityRow struct {
	CodeDisplay, Name          string
	Opening, Movement, Closing domain.Money
	IsEarnings                 bool
}

// EquityChanges rolls equity from the day before the period to its end. Both
// totals come from the balance sheet, so the report always ties to it.
type EquityChanges struct {
	From, To                                  string
	Rows                                      []EquityRow
	TotalOpening, TotalMovement, TotalClosing domain.Money
}

func (r *Reports) EquityChanges(ctx context.Context, from, to string) (*EquityChanges, error) {
	from, to, err := r.period(from, to)
	if err != nil {
		return nil, err
	}
	start, _ := time.Parse(time.DateOnly, from)
	opening, err := r.BalanceSheet(ctx, start.AddDate(0, 0, -1).Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	closing, err := r.BalanceSheet(ctx, to)
	if err != nil {
		return nil, err
	}
	before := map[string]domain.Money{}
	for _, b := range opening.Equity {
		before[b.ID] = b.Net()
	}
	out := &EquityChanges{From: from, To: to, Rows: []EquityRow{}, TotalOpening: opening.TotalEquity, TotalClosing: closing.TotalEquity,
		TotalMovement: closing.TotalEquity - opening.TotalEquity}
	for _, b := range closing.Equity {
		o, c := before[b.ID], b.Net()
		if o == 0 && c == 0 {
			continue
		}
		out.Rows = append(out.Rows, EquityRow{CodeDisplay: domain.FormatAccountCodeDisplay(b.Code), Name: b.Name, Opening: o, Movement: c - o, Closing: c})
	}
	out.Rows = append(out.Rows, EquityRow{Name: "Laba tahun berjalan", IsEarnings: true, Opening: opening.CurrentYearEarnings,
		Movement: closing.CurrentYearEarnings - opening.CurrentYearEarnings, Closing: closing.CurrentYearEarnings})
	return out, nil
}

// ClientEconomicsReport is the agency's view of unit economics: what each client
// is billed, how much arrives and how fast. Costs are not booked per client, so
// it carries no margin per client.
type ClientEconomicsReport struct {
	From, To                                                    string
	Rows                                                        []domain.ClientEconomics
	TotalInvoiced, TotalCollected, AveragePerClient             domain.Money
	CollectionRate, TopClientShare, TopThreeShare, AvgDaysToPay float64
	HasPayments                                                 bool
}

func (r *Reports) ClientEconomics(ctx context.Context, from, to string) (*ClientEconomicsReport, error) {
	from, to, err := r.period(from, to)
	if err != nil {
		return nil, err
	}
	rows, err := r.Store.ClientEconomics(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out := &ClientEconomicsReport{From: from, To: to, Rows: rows}
	var paidDays float64
	for _, c := range rows {
		out.TotalInvoiced += c.Invoiced
		out.TotalCollected += c.Collected
		paidDays += c.PaidDays
	}
	if len(rows) == 0 || out.TotalInvoiced == 0 {
		return out, nil
	}
	out.AveragePerClient = out.TotalInvoiced / domain.Money(len(rows))
	total := out.TotalInvoiced.Float()
	out.CollectionRate = out.TotalCollected.Float() / total * 100
	out.TopClientShare = rows[0].Invoiced.Float() / total * 100
	for _, c := range rows[:min(3, len(rows))] {
		out.TopThreeShare += c.Invoiced.Float() / total * 100
	}
	if out.TotalCollected > 0 {
		out.AvgDaysToPay, out.HasPayments = paidDays/out.TotalCollected.Float(), true
	}
	return out, nil
}
