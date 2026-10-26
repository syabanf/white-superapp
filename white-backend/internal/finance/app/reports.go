package app

import (
	"context"
	"sort"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

// Reports builds the financial statements and the dashboard from posted
// journal lines of active postable accounts (NüHabit service_reports.go).
type Reports struct{ Deps }

func NewReports(d Deps) *Reports { return &Reports{d} }

func filterType(rows []domain.AccountBalance, code string) []domain.AccountBalance {
	out := []domain.AccountBalance{}
	for _, r := range rows {
		if r.TypeCode == code {
			out = append(out, r)
		}
	}
	return out
}

func sumBalance(rows []domain.AccountBalance) domain.Money {
	var s domain.Money
	for _, r := range rows {
		s += r.Net()
	}
	return s
}

func (r *Reports) asOf(s string) (string, error) {
	if trim(s) == "" {
		return r.today(), nil
	}
	return parseDay("as_of", s)
}

// period defaults to the first of the current month through today.
func (r *Reports) period(from, to string) (string, string, error) {
	today := r.today()
	if trim(to) == "" {
		to = today
	}
	if trim(from) == "" {
		from = to[:8] + "01"
	}
	from, err := parseDay("date_from", from)
	if err != nil {
		return "", "", err
	}
	if to, err = parseDay("date_to", to); err != nil {
		return "", "", err
	}
	if to < from {
		return "", "", domain.Bad("Tanggal akhir tidak boleh sebelum tanggal awal")
	}
	return from, to, nil
}

type TrialBalance struct {
	AsOf                    string
	Rows                    []domain.AccountBalance
	TotalDebit, TotalCredit domain.Money
}

func (r *Reports) TrialBalance(ctx context.Context, asOf string) (*TrialBalance, error) {
	asOf, err := r.asOf(asOf)
	if err != nil {
		return nil, err
	}
	rows, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{AsOf: asOf})
	if err != nil {
		return nil, err
	}
	out := &TrialBalance{AsOf: asOf, Rows: rows}
	for i := range rows {
		rows[i].Debit, rows[i].Credit = domain.TrialBalanceColumns(rows[i].Debit, rows[i].Credit)
		out.TotalDebit += rows[i].Debit
		out.TotalCredit += rows[i].Credit
	}
	return out, nil
}

type BalanceSheet struct {
	AsOf                                   string
	Assets, Liabilities, Equity            []domain.AccountBalance
	CurrentYearEarnings                    domain.Money
	TotalAssets, TotalLiabilities          domain.Money
	TotalEquity, TotalLiabilitiesAndEquity domain.Money
}

// BalanceSheet: current-year earnings run from the covering fiscal year's
// start (else 1 January) and count as equity.
func (r *Reports) BalanceSheet(ctx context.Context, asOf string) (*BalanceSheet, error) {
	asOf, err := r.asOf(asOf)
	if err != nil {
		return nil, err
	}
	all, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{AsOf: asOf})
	if err != nil {
		return nil, err
	}
	fyStart, err := r.Store.CoveringFiscalYearStart(ctx, asOf)
	if err != nil {
		return nil, err
	}
	if fyStart == "" {
		fyStart = asOf[:4] + "-01-01"
	}
	pl, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: fyStart, To: asOf})
	if err != nil {
		return nil, err
	}
	var earnings domain.Money
	for _, b := range pl {
		if domain.IsProfitLossType(b.TypeCode) {
			earnings += domain.EarningsContribution(b.TypeCode, b.Net())
		}
	}
	bs := &BalanceSheet{AsOf: asOf, Assets: filterType(all, domain.TypeAsset), Liabilities: filterType(all, domain.TypeLiability),
		Equity: filterType(all, domain.TypeEquity), CurrentYearEarnings: earnings}
	bs.TotalAssets = sumBalance(bs.Assets)
	bs.TotalLiabilities = sumBalance(bs.Liabilities)
	bs.TotalEquity = sumBalance(bs.Equity) + earnings
	bs.TotalLiabilitiesAndEquity = bs.TotalLiabilities + bs.TotalEquity
	return bs, nil
}

type IncomeStatement struct {
	From, To                                        string
	Revenue, COGS, Expenses, OtherIncome, OtherExp  []domain.AccountBalance
	TotalRevenue, TotalCOGS, GrossProfit            domain.Money
	TotalExpenses, OperatingIncome                  domain.Money
	TotalOtherIncome, TotalOtherExpenses, NetIncome domain.Money
}

func incomeStatement(from, to string, all []domain.AccountBalance) *IncomeStatement {
	is := &IncomeStatement{From: from, To: to,
		Revenue: filterType(all, domain.TypeRevenue), COGS: filterType(all, domain.TypeCOGS), Expenses: filterType(all, domain.TypeExpense),
		OtherIncome: filterType(all, domain.TypeOtherIncome), OtherExp: filterType(all, domain.TypeOtherExpense)}
	is.TotalRevenue = sumBalance(is.Revenue)
	is.TotalCOGS = sumBalance(is.COGS)
	is.GrossProfit = is.TotalRevenue - is.TotalCOGS
	is.TotalExpenses = sumBalance(is.Expenses)
	is.OperatingIncome = is.GrossProfit - is.TotalExpenses
	is.TotalOtherIncome = sumBalance(is.OtherIncome)
	is.TotalOtherExpenses = sumBalance(is.OtherExp)
	is.NetIncome = is.OperatingIncome + is.TotalOtherIncome - is.TotalOtherExpenses
	return is
}

func (r *Reports) IncomeStatement(ctx context.Context, from, to string) (*IncomeStatement, error) {
	from, to, err := r.period(from, to)
	if err != nil {
		return nil, err
	}
	all, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: from, To: to})
	if err != nil {
		return nil, err
	}
	return incomeStatement(from, to, all), nil
}

type CashFlowSection struct {
	Category string
	Rows     []domain.AccountBalance
	Total    domain.Money
}

type CashFlow struct {
	From, To                                 string
	Operating, Investing, Financing, NonCash CashFlowSection
	NetChange, Opening, Closing              domain.Money
}

func cashSection(category string, rows []domain.AccountBalance) CashFlowSection {
	s := CashFlowSection{Category: category, Rows: []domain.AccountBalance{}}
	for _, r := range rows {
		if r.CashFlowCategory != category {
			continue
		}
		s.Rows = append(s.Rows, r)
		if !r.IsCashBank {
			s.Total += domain.CashFlowContribution(r.TypeCode, r.Net())
		}
	}
	return s
}

func cashTotal(rows []domain.AccountBalance) domain.Money {
	var s domain.Money
	for _, r := range rows {
		if r.IsCashBank {
			s += r.Balance
		}
	}
	return s
}

func dayBefore(date string) string {
	t, _ := time.Parse(time.DateOnly, date)
	return t.AddDate(0, 0, -1).Format(time.DateOnly)
}

// CashFlow is indirect-lite: non-cash accounts grouped by their cash flow
// category; cash opening and closing are the cash/bank balances the day
// before from and at to.
func (r *Reports) CashFlow(ctx context.Context, from, to string) (*CashFlow, error) {
	from, to, err := r.period(from, to)
	if err != nil {
		return nil, err
	}
	period, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: from, To: to})
	if err != nil {
		return nil, err
	}
	opening, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{AsOf: dayBefore(from)})
	if err != nil {
		return nil, err
	}
	closing, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{AsOf: to})
	if err != nil {
		return nil, err
	}
	cf := &CashFlow{From: from, To: to,
		Operating: cashSection("OPERATING", period), Investing: cashSection("INVESTING", period),
		Financing: cashSection("FINANCING", period), NonCash: cashSection("NON_CASH", period),
		Opening: cashTotal(opening), Closing: cashTotal(closing)}
	cf.NetChange = cf.Closing - cf.Opening
	return cf, nil
}

// GeneralLedgerAccounts are every postable account with its balance.
func (r *Reports) GeneralLedgerAccounts(ctx context.Context, asOf string) ([]domain.AccountBalance, error) {
	asOf, err := r.asOf(asOf)
	if err != nil {
		return nil, err
	}
	return r.Store.AccountBalances(ctx, domain.BalanceWindow{AsOf: asOf})
}

func (r *Reports) GeneralLedger(ctx context.Context, accountID, from, to string) (*Ledger, error) {
	from, to, err := ledgerRange(from, to)
	if err != nil {
		return nil, err
	}
	acc, err := r.Store.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if acc == nil || !acc.IsPostable {
		return nil, domain.NotFound("Akun tidak ditemukan")
	}
	return buildLedger(ctx, r.Store, acc, from, to, acc.IsContra)
}

// ─── Dashboard ──────────────────────────────────────────────

type NameValue struct {
	Name  string
	Value domain.Money
}

type MonthTrend struct {
	Month                 string
	Revenue, Expense, Net domain.Money
}

type Dashboard struct {
	AsOf, From, To        string
	BalanceSheet          *BalanceSheet
	Income                *IncomeStatement
	Cash                  *CashFlow
	TrialBalanceDiff      domain.Money
	PostedEntries, Drafts int
	Receivable, Payable   domain.Money
	PendingClaims         int
	BalanceComposition    []NameValue
	PnlBreakdown          []NameValue
	CashFlowBreakdown     []NameValue
	MonthlyTrend          []MonthTrend
	TopExpenses           []NameValue
}

func maxZero(m domain.Money) domain.Money { return max(m, 0) }

// Dashboard adds AR/AP outstanding and pending reimbursements to NüHabit's
// KPIs; the trend runs from the start of the period's year to asOf.
func (r *Reports) Dashboard(ctx context.Context, asOf, from, to string) (*Dashboard, error) {
	asOf, err := r.asOf(asOf)
	if err != nil {
		return nil, err
	}
	if trim(to) == "" {
		to = asOf
	}
	if trim(from) == "" {
		from = asOf[:4] + "-01-01"
	}
	bs, err := r.BalanceSheet(ctx, asOf)
	if err != nil {
		return nil, err
	}
	is, err := r.IncomeStatement(ctx, from, to)
	if err != nil {
		return nil, err
	}
	cf, err := r.CashFlow(ctx, from, to)
	if err != nil {
		return nil, err
	}
	tb, err := r.TrialBalance(ctx, asOf)
	if err != nil {
		return nil, err
	}
	d := &Dashboard{AsOf: asOf, From: is.From, To: is.To, BalanceSheet: bs, Income: is, Cash: cf, TrialBalanceDiff: (tb.TotalDebit - tb.TotalCredit).Abs()}
	if d.PostedEntries, d.Drafts, err = r.Store.CountEntries(ctx, is.From, is.To); err != nil {
		return nil, err
	}
	ledgers := NewLedgers(r.Deps)
	for kind, target := range map[string]*domain.Money{domain.KindAR: &d.Receivable, domain.KindAP: &d.Payable} {
		open, err := ledgers.OpenInvoices(ctx, kind)
		if err != nil {
			return nil, err
		}
		for _, inv := range open {
			*target += inv.Outstanding()
		}
	}
	if d.PendingClaims, err = r.Store.CountPendingClaims(ctx); err != nil {
		return nil, err
	}
	monthly, err := r.Store.MonthlyPnL(ctx, is.From, asOf)
	if err != nil {
		return nil, err
	}
	for _, m := range monthly {
		d.MonthlyTrend = append(d.MonthlyTrend, MonthTrend{Month: m.Month, Revenue: m.Revenue, Expense: m.Expense, Net: m.Revenue - m.Expense})
	}
	d.BalanceComposition = []NameValue{{"Aset", maxZero(bs.TotalAssets)}, {"Liabilitas", maxZero(bs.TotalLiabilities)}, {"Ekuitas", maxZero(bs.TotalEquity)}}
	d.PnlBreakdown = []NameValue{{"Pendapatan", is.TotalRevenue}, {"HPP", is.TotalCOGS}, {"Beban", is.TotalExpenses}, {"Pendapatan lain", is.TotalOtherIncome}, {"Beban lain", is.TotalOtherExpenses}}
	d.CashFlowBreakdown = []NameValue{{"Operasi", cf.Operating.Total}, {"Investasi", cf.Investing.Total}, {"Pendanaan", cf.Financing.Total}}
	d.TopExpenses = topExpenses(append(append(append([]domain.AccountBalance{}, is.Expenses...), is.COGS...), is.OtherExp...), 8)
	return d, nil
}

func topExpenses(rows []domain.AccountBalance, n int) []NameValue {
	var out []NameValue
	for _, r := range rows {
		if r.Balance > 0 {
			out = append(out, NameValue{Name: r.Name, Value: r.Balance})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	if len(out) > n {
		out = out[:n]
	}
	return out
}
