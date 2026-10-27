package app

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Analysis computes financial ratios from the ledger and writes the AI
// CFO memo (new for Mixist).
type Analysis struct{ Deps }

func NewAnalysis(d Deps) *Analysis { return &Analysis{d} }

// Metric is one ratio for the current period and, when compared, the
// previous one.
type Metric struct {
	Key, Label, Unit string
	LowerIsBetter    bool
	Current          domain.Ratio
	Previous         *domain.Ratio
}

// ExpenseItem is one expense account's amount in both periods.
type ExpenseItem struct {
	AccountID, Code, Name string
	Amount, Previous      domain.Money
	SharePct              float64
}

type FinancialAnalysis struct {
	From, To, PrevFrom, PrevTo string
	Compare                    bool
	Profitability, Liquidity   []Metric
	Leverage, Efficiency, Cash []Metric
	Expenses, TopMovers        []ExpenseItem
	Current, Prev              domain.LedgerSnapshot
}

// snapshot reads one period's income statement and the balance sheet at
// its end.
func (a *Analysis) snapshot(ctx context.Context, from, to string) (domain.LedgerSnapshot, []domain.AccountBalance, error) {
	reports := NewReports(a.Deps)
	f, _ := time.Parse(time.DateOnly, from)
	t, _ := time.Parse(time.DateOnly, to)
	s := domain.LedgerSnapshot{Days: int(t.Sub(f).Hours()/24) + 1}
	period, err := a.Store.AccountBalances(ctx, domain.BalanceWindow{From: from, To: to})
	if err != nil {
		return s, nil, err
	}
	is := incomeStatement(from, to, period)
	s.Revenue, s.COGS, s.Expenses = is.TotalRevenue, is.TotalCOGS, is.TotalExpenses
	s.OtherIncome, s.OtherExpenses = is.TotalOtherIncome, is.TotalOtherExpenses
	bs, err := reports.BalanceSheet(ctx, to)
	if err != nil {
		return s, nil, err
	}
	s.TotalLiabilities, s.TotalEquity = bs.TotalLiabilities, bs.TotalEquity
	for _, b := range bs.Assets {
		if b.IsCashBank {
			s.Cash += b.Balance
		}
		if domain.IsCurrentAsset(b.Code) {
			s.CurrentAssets += b.Net()
		}
		if domain.IsReceivable(b.Code) {
			s.Receivables += b.Net()
		}
	}
	for _, b := range bs.Liabilities {
		if domain.IsCurrentLiability(b.Code) {
			s.CurrentLiabilities += b.Net()
		}
		if domain.IsTradePayable(b.Code) {
			s.TradePayables += b.Net()
		}
	}
	opening, err := a.Store.AccountBalances(ctx, domain.BalanceWindow{AsOf: dayBefore(from)})
	if err != nil {
		return s, nil, err
	}
	s.CashOpening = cashTotal(opening)
	var expenses []domain.AccountBalance
	for _, b := range period {
		if domain.IsExpenseType(b.TypeCode) {
			expenses = append(expenses, b)
		}
	}
	return s, expenses, nil
}

// ratioDef is one ratio of the analysis and the trend.
type ratioDef struct {
	key, label, unit, group string
	lower                   bool
	pick                    func(domain.Ratios) domain.Ratio
}

var ratioDefs = []ratioDef{
	{"gross_margin", "Gross margin", "pct", "profitability", false, func(x domain.Ratios) domain.Ratio { return x.GrossMargin }},
	{"operating_margin", "Operating margin", "pct", "profitability", false, func(x domain.Ratios) domain.Ratio { return x.OperatingMargin }},
	{"net_margin", "Net margin", "pct", "profitability", false, func(x domain.Ratios) domain.Ratio { return x.NetMargin }},
	{"current_ratio", "Current ratio", "ratio", "liquidity", false, func(x domain.Ratios) domain.Ratio { return x.CurrentRatio }},
	{"quick_ratio", "Quick ratio", "ratio", "liquidity", false, func(x domain.Ratios) domain.Ratio { return x.QuickRatio }},
	{"cash_ratio", "Cash ratio", "ratio", "liquidity", false, func(x domain.Ratios) domain.Ratio { return x.CashRatio }},
	{"debt_to_equity", "Debt to equity", "ratio", "leverage", true, func(x domain.Ratios) domain.Ratio { return x.DebtToEquity }},
	{"dso", "Days sales outstanding (DSO)", "days", "efficiency", true, func(x domain.Ratios) domain.Ratio { return x.DSO }},
	{"dpo", "Days payable outstanding (DPO)", "days", "efficiency", false, func(x domain.Ratios) domain.Ratio { return x.DPO }},
	{"burn_rate", "Burn rate per bulan", "idr", "cash", true, func(x domain.Ratios) domain.Ratio { return domain.Ratio{Value: x.BurnRate.Float(), OK: true} }},
	{"runway_months", "Runway", "months", "cash", false, func(x domain.Ratios) domain.Ratio { return x.Runway }},
}

// Analyze computes the ratios for [from, to] and, with compare, the
// comparison period: the same length ending the day before from, or with
// mode "year" the same dates a year earlier.
func (a *Analysis) Analyze(ctx context.Context, from, to string, compare bool, mode string) (*FinancialAnalysis, error) {
	from, to, err := NewReports(a.Deps).period(from, to)
	if err != nil {
		return nil, err
	}
	cur, curExp, err := a.snapshot(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out := &FinancialAnalysis{From: from, To: to, Compare: compare, Current: cur}
	var prevExp []domain.AccountBalance
	var prevRatios *domain.Ratios
	if compare {
		f, _ := time.Parse(time.DateOnly, from)
		out.PrevTo = f.AddDate(0, 0, -1).Format(time.DateOnly)
		out.PrevFrom = f.AddDate(0, 0, -cur.Days).Format(time.DateOnly)
		if mode == CompareYear {
			t, _ := time.Parse(time.DateOnly, to)
			out.PrevFrom, out.PrevTo = f.AddDate(-1, 0, 0).Format(time.DateOnly), t.AddDate(-1, 0, 0).Format(time.DateOnly)
		}
		if out.Prev, prevExp, err = a.snapshot(ctx, out.PrevFrom, out.PrevTo); err != nil {
			return nil, err
		}
		r := domain.ComputeRatios(out.Prev)
		prevRatios = &r
	}
	r := domain.ComputeRatios(cur)
	groups := map[string]*[]Metric{"profitability": &out.Profitability, "liquidity": &out.Liquidity, "leverage": &out.Leverage, "efficiency": &out.Efficiency, "cash": &out.Cash}
	for _, d := range ratioDefs {
		m := Metric{Key: d.key, Label: d.label, Unit: d.unit, LowerIsBetter: d.lower, Current: d.pick(r)}
		if prevRatios != nil {
			p := d.pick(*prevRatios)
			m.Previous = &p
		}
		*groups[d.group] = append(*groups[d.group], m)
	}
	growth := Metric{Key: "revenue_growth", Label: "Pertumbuhan pendapatan", Unit: "pct"}
	if compare && out.Prev.Revenue != 0 {
		growth.Current = domain.Ratio{Value: (cur.Revenue - out.Prev.Revenue).Float() / math.Abs(out.Prev.Revenue.Float()) * 100, OK: true}
	}
	out.Profitability = append(out.Profitability, growth)
	out.Expenses, out.TopMovers = expenseBreakdown(curExp, prevExp)
	return out, nil
}

// RatioSeries is one ratio month by month.
type RatioSeries struct {
	Key, Label, Unit, Group string
	LowerIsBetter           bool
	Points                  []domain.Ratio
	Benchmark               *float64
}

// RatioTrend computes every ratio for each of the months months ending at
// to's month (the last one up to to).
func (a *Analysis) RatioTrend(ctx context.Context, to string, months int) ([]string, []RatioSeries, error) {
	_, to, err := NewReports(a.Deps).period("", to)
	if err != nil {
		return nil, nil, err
	}
	if months <= 0 {
		months = 12
	}
	months = min(months, 24)
	list := make([]string, months)
	ratios := make([]domain.Ratios, months)
	for i := range list {
		m := addMonths(to[:7], i-months+1)
		list[i] = m
		snap, _, err := a.snapshot(ctx, m+"-01", monthEnd(m, to))
		if err != nil {
			return nil, nil, err
		}
		ratios[i] = domain.ComputeRatios(snap)
	}
	out := make([]RatioSeries, len(ratioDefs))
	for i, d := range ratioDefs {
		s := RatioSeries{Key: d.key, Label: d.label, Unit: d.unit, Group: d.group, LowerIsBetter: d.lower, Points: make([]domain.Ratio, months)}
		for j := range list {
			s.Points[j] = d.pick(ratios[j])
		}
		if b, ok := domain.Benchmarks[d.key]; ok {
			s.Benchmark = &b
		}
		out[i] = s
	}
	return list, out, nil
}

// expenseBreakdown lists current expense accounts by size with their share,
// and the five accounts whose amount moved most against the previous period.
func expenseBreakdown(cur, prev []domain.AccountBalance) ([]ExpenseItem, []ExpenseItem) {
	byID := map[string]*ExpenseItem{}
	var order []string
	var total domain.Money
	add := func(b domain.AccountBalance, current bool) {
		it, ok := byID[b.ID]
		if !ok {
			it = &ExpenseItem{AccountID: b.ID, Code: b.Code, Name: b.Name}
			byID[b.ID] = it
			order = append(order, b.ID)
		}
		if current {
			it.Amount = b.Balance
		} else {
			it.Previous = b.Balance
		}
	}
	for _, b := range cur {
		add(b, true)
		total += max(b.Balance, 0)
	}
	for _, b := range prev {
		add(b, false)
	}
	var breakdown, movers []ExpenseItem
	for _, id := range order {
		it := *byID[id]
		if total > 0 && it.Amount > 0 {
			it.SharePct = it.Amount.Float() / total.Float() * 100
		}
		if it.Amount > 0 {
			breakdown = append(breakdown, it)
		}
		if it.Amount != it.Previous {
			movers = append(movers, it)
		}
	}
	sort.SliceStable(breakdown, func(i, j int) bool { return breakdown[i].Amount > breakdown[j].Amount })
	sort.SliceStable(movers, func(i, j int) bool {
		return (movers[i].Amount - movers[i].Previous).Abs() > (movers[j].Amount - movers[j].Previous).Abs()
	})
	if len(movers) > 5 {
		movers = movers[:5]
	}
	return breakdown, movers
}

// insightData is the JSON the memo prompt receives: real numbers only.
func insightData(fa *FinancialAnalysis) map[string]any {
	metrics := map[string]any{}
	for _, group := range [][]Metric{fa.Profitability, fa.Liquidity, fa.Leverage, fa.Efficiency, fa.Cash} {
		for _, m := range group {
			entry := map[string]any{"label": m.Label, "unit": m.Unit}
			if m.Current.OK {
				entry["value"] = math.Round(m.Current.Value*100) / 100
			} else {
				entry["value"] = nil
			}
			if m.Previous != nil && m.Previous.OK {
				entry["previous"] = math.Round(m.Previous.Value*100) / 100
			}
			metrics[m.Key] = entry
		}
	}
	snap := func(s domain.LedgerSnapshot) map[string]any {
		return map[string]any{
			"pendapatan": s.Revenue.Float(), "hpp": s.COGS.Float(), "bebanOperasional": s.Expenses.Float(),
			"labaBersih": s.NetIncome().Float(), "kas": s.Cash.Float(), "piutang": s.Receivables.Float(),
			"hutangUsaha": s.TradePayables.Float(), "totalLiabilitas": s.TotalLiabilities.Float(), "totalEkuitas": s.TotalEquity.Float(),
		}
	}
	var top []map[string]any
	for _, e := range fa.Expenses {
		if len(top) == 8 {
			break
		}
		top = append(top, map[string]any{"akun": e.Name, "nilai": e.Amount.Float(), "sebelumnya": e.Previous.Float(), "porsiPct": math.Round(e.SharePct*10) / 10})
	}
	data := map[string]any{"periode": map[string]any{"dari": fa.From, "sampai": fa.To}, "rasio": metrics, "ringkasan": snap(fa.Current), "bebanTerbesar": top}
	if fa.Compare {
		data["pembanding"] = map[string]any{"dari": fa.PrevFrom, "sampai": fa.PrevTo, "ringkasan": snap(fa.Prev)}
	}
	return data
}

// GenerateInsight sends the analysis to the AI (mock without a key) and
// stores the memo in financial_insights.
func (a *Analysis) GenerateInsight(ctx context.Context, userID, from, to, language string) (*domain.FinancialInsight, error) {
	if language != "en" {
		language = "id"
	}
	fa, err := a.Analyze(ctx, from, to, true, ComparePrevious)
	if err != nil {
		return nil, err
	}
	data := insightData(fa)
	content, model, err := a.Insights.Generate(ctx, InsightInput{From: fa.From, To: fa.To, Language: language, Data: data})
	if err != nil {
		return nil, err
	}
	metrics, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	ins := domain.FinancialInsight{ID: ids.New(), DateFrom: fa.From, DateTo: fa.To, Language: language, Model: model, Content: content, Metrics: metrics, CreatedBy: userID, CreatedAt: a.now()}
	if err := a.Store.InsertInsight(ctx, ins); err != nil {
		return nil, err
	}
	return &ins, nil
}

func (a *Analysis) ListInsights(ctx context.Context, limit int) ([]domain.FinancialInsight, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return a.Store.ListInsights(ctx, limit)
}
