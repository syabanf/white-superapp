package app

import (
	"context"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

// Dashboard tabs (new for WHITE): every tab compares the range with a
// comparison range and plots monthly trends. Range totals come from exact
// day windows; trends from monthly sums and month-end balances.

const (
	ComparePrevious = "previous"
	CompareYear     = "year"
)

// AnalyticsFilter is the dashboard filter bar.
type AnalyticsFilter struct {
	From, To, Compare                                    string
	AccountType, AccountGroup, Department, CashAccountID string
}

// Window is the resolved range, comparison range and trend months.
type Window struct {
	From, To, Compare, PrevFrom, PrevTo string
	Months, PrevMonths                  []string
}

// Kpi is one value over the range and the comparison range as decimal
// strings ("" when not computable), with its monthly trend.
type Kpi struct {
	Key             string
	Value, Previous string
	Trend           []string
}

// Series is a monthly series aligned with Window.Months.
type Series struct {
	Key, Label, Code       string
	Values, PrevValues     []domain.Money
	Total                  domain.Money
	FloatValues, FloatPrev []string
}

// Amount is one ranked slice.
type Amount struct {
	Key, Label, Code, Group string
	Value, Previous         domain.Money
	Count                   int
	// IsDecimal slices (lead time days) carry Decimal instead of Value;
	// "" means no data.
	IsDecimal bool
	Decimal   string
}

type WaterfallStep struct {
	Key, Label string
	Value      domain.Money
	Total      bool
}

func addMonths(ym string, n int) string {
	t, _ := time.Parse("2006-01", ym)
	return t.AddDate(0, n, 0).Format("2006-01")
}

func monthsBetween(a, b string) int {
	ta, _ := time.Parse("2006-01", a)
	tb, _ := time.Parse("2006-01", b)
	return (tb.Year()-ta.Year())*12 + int(tb.Month()-ta.Month())
}

// monthEnd is the month's last day, clipped to clip when it falls inside.
func monthEnd(ym, clip string) string {
	t, _ := time.Parse("2006-01", ym)
	end := t.AddDate(0, 1, -1).Format(time.DateOnly)
	if clip != "" && clip < end {
		return clip
	}
	return end
}

func daysIn(from, to string) int {
	f, _ := time.Parse(time.DateOnly, from)
	t, _ := time.Parse(time.DateOnly, to)
	return int(t.Sub(f).Hours()/24) + 1
}

// window resolves the filter: comparison = same length before from, or the
// same dates a year earlier; trend = the range's months when it spans three
// or more (at most 24), else the twelve months ending at to.
func (r *Reports) window(f AnalyticsFilter) (Window, error) {
	from, to, err := r.period(f.From, f.To)
	if err != nil {
		return Window{}, err
	}
	w := Window{From: from, To: to, Compare: ComparePrevious}
	ft, _ := time.Parse(time.DateOnly, from)
	tt, _ := time.Parse(time.DateOnly, to)
	if f.Compare == CompareYear {
		w.Compare = CompareYear
		w.PrevFrom, w.PrevTo = ft.AddDate(-1, 0, 0).Format(time.DateOnly), tt.AddDate(-1, 0, 0).Format(time.DateOnly)
	} else {
		w.PrevTo = ft.AddDate(0, 0, -1).Format(time.DateOnly)
		w.PrevFrom = ft.AddDate(0, 0, -daysIn(from, to)).Format(time.DateOnly)
	}
	first, last := from[:7], to[:7]
	n := monthsBetween(first, last) + 1
	switch {
	case n < 3:
		n = 12
	case n > 24:
		n = 24
	}
	first = addMonths(last, -(n - 1))
	shift := -n
	if w.Compare == CompareYear {
		shift = -12
	}
	for i := 0; i < n; i++ {
		m := addMonths(first, i)
		w.Months = append(w.Months, m)
		w.PrevMonths = append(w.PrevMonths, addMonths(m, shift))
	}
	return w, nil
}

// spanStart is the first day any trend or range needs.
func (w Window) spanStart() string {
	s := w.Months[0] + "-01"
	if p := w.PrevMonths[0] + "-01"; p < s {
		s = p
	}
	if w.PrevFrom < s {
		s = w.PrevFrom
	}
	return s
}

// prevClip clips the comparison month that mirrors to's month.
func (w Window) prevClip(ym string) string {
	if w.Compare == CompareYear && ym == w.PrevTo[:7] {
		return w.PrevTo
	}
	return ""
}

// ─── shared loaders ─────────────────────────────────────────

type pnl struct{ Revenue, COGS, Expense, OtherIncome, OtherExpense domain.Money }

func (p pnl) GrossProfit() domain.Money     { return p.Revenue - p.COGS }
func (p pnl) OperatingIncome() domain.Money { return p.GrossProfit() - p.Expense }
func (p pnl) NetIncome() domain.Money       { return p.OperatingIncome() + p.OtherIncome - p.OtherExpense }
func (p pnl) TotalExpense() domain.Money    { return p.COGS + p.Expense + p.OtherExpense }

func (p *pnl) add(b domain.AccountBalance) {
	switch b.TypeCode {
	case domain.TypeRevenue:
		p.Revenue += b.Net()
	case domain.TypeCOGS:
		p.COGS += b.Net()
	case domain.TypeExpense:
		p.Expense += b.Net()
	case domain.TypeOtherIncome:
		p.OtherIncome += b.Net()
	case domain.TypeOtherExpense:
		p.OtherExpense += b.Net()
	}
}

func pnlOf(rows []domain.AccountBalance) pnl {
	var p pnl
	for _, b := range rows {
		p.add(b)
	}
	return p
}

// monthly groups movements by month.
type monthly map[string][]domain.MonthMovement

func (r *Reports) monthly(ctx context.Context, w Window) (monthly, error) {
	rows, err := r.Store.MonthlyMovements(ctx, w.spanStart(), w.To)
	if err != nil {
		return nil, err
	}
	out := monthly{}
	for _, m := range rows {
		out[m.Month] = append(out[m.Month], m)
	}
	return out, nil
}

func (m monthly) pnl(month string, keep func(domain.MonthMovement) bool) pnl {
	var p pnl
	for _, mv := range m[month] {
		if keep == nil || keep(mv) {
			p.add(mv.AccountBalance)
		}
	}
	return p
}

// series maps every month through pick.
func series(months []string, pick func(string) domain.Money) []domain.Money {
	out := make([]domain.Money, len(months))
	for i, m := range months {
		out[i] = pick(m)
	}
	return out
}

// balancesAt caches as-of balances per date.
type balancesAt struct {
	r     *Reports
	ctx   context.Context
	cache map[string][]domain.AccountBalance
	today string
}

func (r *Reports) balancesAt(ctx context.Context) *balancesAt {
	return &balancesAt{r: r, ctx: ctx, cache: map[string][]domain.AccountBalance{}, today: r.today()}
}

func (b *balancesAt) at(date string) ([]domain.AccountBalance, error) {
	if date > b.today {
		date = b.today
	}
	if rows, ok := b.cache[date]; ok {
		return rows, nil
	}
	rows, err := b.r.Store.AccountBalances(b.ctx, domain.BalanceWindow{AsOf: date})
	if err != nil {
		return nil, err
	}
	b.cache[date] = rows
	return rows, nil
}

// monthEnds returns pick(balances) at each month end (clipped by clip).
func (b *balancesAt) monthEnds(months []string, clip func(string) string, pick func([]domain.AccountBalance) domain.Money) ([]domain.Money, error) {
	out := make([]domain.Money, len(months))
	for i, m := range months {
		rows, err := b.at(monthEnd(m, clip(m)))
		if err != nil {
			return nil, err
		}
		out[i] = pick(rows)
	}
	return out, nil
}

func cashOf(rows []domain.AccountBalance, accountID string) domain.Money {
	var s domain.Money
	for _, r := range rows {
		if r.IsCashBank && (accountID == "" || r.ID == accountID) {
			s += r.Balance
		}
	}
	return s
}

func (w Window) clipCur(m string) string {
	if m == w.To[:7] {
		return w.To
	}
	return ""
}

func pct(n, d domain.Money) string {
	if d == 0 {
		return ""
	}
	return num(n.Float() / d.Float() * 100)
}

func num(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return ""
	}
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', 2, 64)
}

func strs(ms []domain.Money) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.String()
	}
	return out
}

func sum(ms []domain.Money) domain.Money {
	var s domain.Money
	for _, m := range ms {
		s += m
	}
	return s
}

func moneyKpi(key string, cur, prev domain.Money, trend []domain.Money) Kpi {
	return Kpi{Key: key, Value: cur.String(), Previous: prev.String(), Trend: strs(trend)}
}

// accountFilter keeps accounts of the filter's type and under its group.
type accountFilter struct {
	typeCode, group string
	parents         map[string]domain.AccountParent
}

func (r *Reports) accountFilter(ctx context.Context, f AnalyticsFilter) (accountFilter, error) {
	af := accountFilter{typeCode: trim(f.AccountType), group: trim(f.AccountGroup), parents: map[string]domain.AccountParent{}}
	rows, err := r.Store.AccountParents(ctx)
	if err != nil {
		return af, err
	}
	for _, p := range rows {
		af.parents[p.ID] = p
	}
	return af, nil
}

func (af accountFilter) keep(id, typeCode string) bool {
	if af.typeCode != "" && typeCode != af.typeCode {
		return false
	}
	if af.group == "" {
		return true
	}
	p := af.parents[id]
	return id == af.group || p.ParentID == af.group || p.GrandparentID == af.group
}

func (af accountFilter) keepMove(m domain.MonthMovement) bool { return af.keep(m.ID, m.TypeCode) }

// groupName is the parent account's name, or the account's own.
func (af accountFilter) groupName(m domain.MonthMovement) (string, string) {
	if m.ParentID == "" {
		return m.ID, m.Name
	}
	return m.ParentID, m.ParentName
}

// ─── Summary ────────────────────────────────────────────────

type FinanceSummary struct {
	Window                Window
	Kpis                  []Kpi
	Pnl, Cash             []Series
	Assets, Liabilities   []Amount
	Equity                []Amount
	TrialBalanceDiff      domain.Money
	PostedEntries, Drafts int
}

func (r *Reports) Summary(ctx context.Context, f AnalyticsFilter) (*FinanceSummary, error) {
	w, err := r.window(f)
	if err != nil {
		return nil, err
	}
	mon, err := r.monthly(ctx, w)
	if err != nil {
		return nil, err
	}
	cur, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: w.From, To: w.To})
	if err != nil {
		return nil, err
	}
	prev, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: w.PrevFrom, To: w.PrevTo})
	if err != nil {
		return nil, err
	}
	pc, pp := pnlOf(cur), pnlOf(prev)
	out := &FinanceSummary{Window: w}
	trendOf := func(pick func(pnl) domain.Money) ([]domain.Money, []domain.Money) {
		return series(w.Months, func(m string) domain.Money { return pick(mon.pnl(m, nil)) }),
			series(w.PrevMonths, func(m string) domain.Money { return pick(mon.pnl(m, nil)) })
	}
	rev, revPrev := trendOf(func(p pnl) domain.Money { return p.Revenue })
	exp, expPrev := trendOf(pnl.TotalExpense)
	net, netPrev := trendOf(pnl.NetIncome)
	out.Pnl = []Series{
		{Key: "revenue", Label: "Pendapatan", Values: rev, PrevValues: revPrev, Total: pc.Revenue},
		{Key: "expense", Label: "Beban", Values: exp, PrevValues: expPrev, Total: pc.TotalExpense()},
		{Key: "net_income", Label: "Laba bersih", Values: net, PrevValues: netPrev, Total: pc.NetIncome()},
	}
	margins := make([]string, len(w.Months))
	for i := range w.Months {
		margins[i] = pct(net[i], rev[i])
	}

	bal := r.balancesAt(ctx)
	cashTrend, err := bal.monthEnds(w.Months, w.clipCur, func(rows []domain.AccountBalance) domain.Money { return cashOf(rows, "") })
	if err != nil {
		return nil, err
	}
	cashPrevTrend, err := bal.monthEnds(w.PrevMonths, w.prevClip, func(rows []domain.AccountBalance) domain.Money { return cashOf(rows, "") })
	if err != nil {
		return nil, err
	}
	out.Cash = []Series{{Key: "cash", Label: "Kas dan bank", Values: cashTrend, PrevValues: cashPrevTrend}}
	prevBal, err := bal.at(w.PrevTo)
	if err != nil {
		return nil, err
	}

	inv, err := r.invoiceBook(ctx, w.To)
	if err != nil {
		return nil, err
	}
	arTrend := series(w.Months, func(m string) domain.Money { return inv.outstanding(domain.KindAR, monthEnd(m, w.clipCur(m))) })
	apTrend := series(w.Months, func(m string) domain.Money { return inv.outstanding(domain.KindAP, monthEnd(m, w.clipCur(m))) })
	// A one-day window: only the claims still waiting matter here.
	claims, err := r.Store.ClaimFacts(ctx, w.To, w.To, "")
	if err != nil {
		return nil, err
	}
	var pending domain.Money
	for _, c := range claims {
		if isPendingClaim(c.Status) {
			pending += c.Amount
		}
	}
	out.Kpis = []Kpi{
		moneyKpi("revenue", pc.Revenue, pp.Revenue, rev),
		moneyKpi("expense", pc.TotalExpense(), pp.TotalExpense(), exp),
		moneyKpi("net_income", pc.NetIncome(), pp.NetIncome(), net),
		{Key: "net_margin", Value: pct(pc.NetIncome(), pc.Revenue), Previous: pct(pp.NetIncome(), pp.Revenue), Trend: margins},
		moneyKpi("cash", cashTrend[len(cashTrend)-1], cashOf(prevBal, ""), cashTrend),
		moneyKpi("receivable", inv.outstanding(domain.KindAR, w.To), inv.outstanding(domain.KindAR, w.PrevTo), arTrend),
		moneyKpi("payable", inv.outstanding(domain.KindAP, w.To), inv.outstanding(domain.KindAP, w.PrevTo), apTrend),
		{Key: "pending_reimbursement", Value: pending.String()},
	}

	bs, err := r.BalanceSheet(ctx, w.To)
	if err != nil {
		return nil, err
	}
	af, err := r.accountFilter(ctx, AnalyticsFilter{})
	if err != nil {
		return nil, err
	}
	out.Assets, out.Liabilities, out.Equity = composition(bs.Assets, af), composition(bs.Liabilities, af), composition(bs.Equity, af)
	if bs.CurrentYearEarnings != 0 {
		out.Equity = append(out.Equity, Amount{Key: "current_year_earnings", Label: "Laba tahun berjalan", Value: bs.CurrentYearEarnings})
	}
	tb, err := r.TrialBalance(ctx, w.To)
	if err != nil {
		return nil, err
	}
	out.TrialBalanceDiff = (tb.TotalDebit - tb.TotalCredit).Abs()
	if out.PostedEntries, out.Drafts, err = r.Store.CountEntries(ctx, w.From, w.To); err != nil {
		return nil, err
	}
	return out, nil
}

// composition sums balance sheet rows by parent account, largest first.
func composition(rows []domain.AccountBalance, af accountFilter) []Amount {
	idx := map[string]int{}
	out := []Amount{}
	for _, b := range rows {
		p := af.parents[b.ID]
		key, label := p.ParentID, p.ParentName
		if key == "" {
			key, label = b.ID, b.Name
		}
		i, ok := idx[key]
		if !ok {
			i = len(out)
			idx[key] = i
			out = append(out, Amount{Key: key, Label: label})
		}
		out[i].Value += b.Net()
		out[i].Count++
	}
	kept := out[:0]
	for _, a := range out {
		if a.Value != 0 {
			kept = append(kept, a)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].Value.Abs() > kept[j].Value.Abs() })
	return kept
}

// ─── Profitability ──────────────────────────────────────────

type FinanceProfitability struct {
	Window                          Window
	Kpis                            []Kpi
	Waterfall                       []WaterfallStep
	Pnl                             []Series
	RevenueAccounts, RevenueClients []Amount
	ExpenseGroups, ExpenseAccounts  []Amount
	AccountMonths                   []Series
}

func (r *Reports) Profitability(ctx context.Context, f AnalyticsFilter) (*FinanceProfitability, error) {
	w, err := r.window(f)
	if err != nil {
		return nil, err
	}
	af, err := r.accountFilter(ctx, f)
	if err != nil {
		return nil, err
	}
	mon, err := r.monthly(ctx, w)
	if err != nil {
		return nil, err
	}
	cur, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: w.From, To: w.To})
	if err != nil {
		return nil, err
	}
	prev, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: w.PrevFrom, To: w.PrevTo})
	if err != nil {
		return nil, err
	}
	pc, pp := pnlOf(cur), pnlOf(prev)
	out := &FinanceProfitability{Window: w}
	out.Waterfall = []WaterfallStep{
		{Key: "revenue", Label: "Pendapatan", Value: pc.Revenue, Total: true},
		{Key: "cogs", Label: "HPP", Value: -pc.COGS},
		{Key: "gross_profit", Label: "Laba kotor", Value: pc.GrossProfit(), Total: true},
		{Key: "expense", Label: "Beban operasional", Value: -pc.Expense},
		{Key: "operating_income", Label: "Laba operasi", Value: pc.OperatingIncome(), Total: true},
		{Key: "other_income", Label: "Pendapatan lain", Value: pc.OtherIncome},
		{Key: "other_expense", Label: "Beban lain", Value: -pc.OtherExpense},
		{Key: "net_income", Label: "Laba bersih", Value: pc.NetIncome(), Total: true},
	}
	defs := []struct {
		key, label string
		pick       func(pnl) domain.Money
	}{
		{"revenue", "Pendapatan", func(p pnl) domain.Money { return p.Revenue }},
		{"cogs", "HPP", func(p pnl) domain.Money { return p.COGS }},
		{"expense", "Beban operasional", func(p pnl) domain.Money { return p.Expense }},
		{"other_income", "Pendapatan lain", func(p pnl) domain.Money { return p.OtherIncome }},
		{"other_expense", "Beban lain", func(p pnl) domain.Money { return p.OtherExpense }},
		{"gross_profit", "Laba kotor", pnl.GrossProfit},
		{"operating_income", "Laba operasi", pnl.OperatingIncome},
		{"net_income", "Laba bersih", pnl.NetIncome},
	}
	monthsPnl := map[string]pnl{}
	for _, m := range append(append([]string{}, w.Months...), w.PrevMonths...) {
		monthsPnl[m] = mon.pnl(m, nil)
	}
	byKey := map[string][]domain.Money{}
	for _, d := range defs {
		s := Series{Key: d.key, Label: d.label, Total: d.pick(pc),
			Values:     series(w.Months, func(m string) domain.Money { return d.pick(monthsPnl[m]) }),
			PrevValues: series(w.PrevMonths, func(m string) domain.Money { return d.pick(monthsPnl[m]) })}
		byKey[d.key] = s.Values
		out.Pnl = append(out.Pnl, s)
	}
	ratioTrend := func(n string) []string {
		t := make([]string, len(w.Months))
		for i := range w.Months {
			t[i] = pct(byKey[n][i], byKey["revenue"][i])
		}
		return t
	}
	out.Kpis = []Kpi{
		moneyKpi("revenue", pc.Revenue, pp.Revenue, byKey["revenue"]),
		moneyKpi("gross_profit", pc.GrossProfit(), pp.GrossProfit(), byKey["gross_profit"]),
		{Key: "gross_margin", Value: pct(pc.GrossProfit(), pc.Revenue), Previous: pct(pp.GrossProfit(), pp.Revenue), Trend: ratioTrend("gross_profit")},
		{Key: "operating_margin", Value: pct(pc.OperatingIncome(), pc.Revenue), Previous: pct(pp.OperatingIncome(), pp.Revenue), Trend: ratioTrend("operating_income")},
		{Key: "net_margin", Value: pct(pc.NetIncome(), pc.Revenue), Previous: pct(pp.NetIncome(), pp.Revenue), Trend: ratioTrend("net_income")},
		{Key: "opex_ratio", Value: pct(pc.Expense, pc.Revenue), Previous: pct(pp.Expense, pp.Revenue), Trend: ratioTrend("expense")},
	}

	prevByID := map[string]domain.AccountBalance{}
	for _, b := range prev {
		prevByID[b.ID] = b
	}
	groups := map[string]*Amount{}
	for _, b := range cur {
		if !domain.IsProfitLossType(b.TypeCode) || !af.keep(b.ID, b.TypeCode) {
			continue
		}
		p := prevByID[b.ID]
		if b.Net() == 0 && p.Net() == 0 {
			continue
		}
		a := Amount{Key: b.ID, Label: b.Name, Code: b.Code, Value: b.Net(), Previous: p.Net(), Count: b.MovementCount}
		parent := af.parents[b.ID]
		a.Group = parent.ParentName
		switch {
		case b.TypeCode == domain.TypeRevenue || b.TypeCode == domain.TypeOtherIncome:
			out.RevenueAccounts = append(out.RevenueAccounts, a)
		case domain.IsExpenseType(b.TypeCode):
			out.ExpenseAccounts = append(out.ExpenseAccounts, a)
			key, label := parent.ParentID, parent.ParentName
			if key == "" {
				key, label = b.ID, b.Name
			}
			g, ok := groups[key]
			if !ok {
				g = &Amount{Key: key, Label: label}
				groups[key] = g
			}
			g.Value += a.Value
			g.Previous += a.Previous
			g.Count++
		}
	}
	for _, g := range groups {
		out.ExpenseGroups = append(out.ExpenseGroups, *g)
	}
	for _, list := range [][]Amount{out.RevenueAccounts, out.ExpenseAccounts, out.ExpenseGroups} {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Value > list[j].Value })
	}
	if out.RevenueClients, err = r.revenueClients(ctx, w); err != nil {
		return nil, err
	}

	rowsByID := map[string]*Series{}
	var order []string
	for i, m := range w.Months {
		for _, mv := range mon[m] {
			if !domain.IsProfitLossType(mv.TypeCode) || !af.keepMove(mv) {
				continue
			}
			s, ok := rowsByID[mv.ID]
			if !ok {
				s = &Series{Key: mv.ID, Label: mv.Name, Code: mv.Code, Values: make([]domain.Money, len(w.Months))}
				rowsByID[mv.ID] = s
				order = append(order, mv.ID)
			}
			s.Values[i] += mv.Net()
			s.Total += mv.Net()
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return rowsByID[order[i]].Code < rowsByID[order[j]].Code })
	for _, id := range order {
		out.AccountMonths = append(out.AccountMonths, *rowsByID[id])
	}
	return out, nil
}

// revenueClients ranks clients by posted AR invoice subtotal in the range.
func (r *Reports) revenueClients(ctx context.Context, w Window) ([]Amount, error) {
	rows, err := r.Store.InvoiceFacts(ctx, w.To)
	if err != nil {
		return nil, err
	}
	idx := map[string]*Amount{}
	for _, inv := range rows {
		if inv.Kind != domain.KindAR {
			continue
		}
		cur := inv.InvoiceDate >= w.From && inv.InvoiceDate <= w.To
		prev := inv.InvoiceDate >= w.PrevFrom && inv.InvoiceDate <= w.PrevTo
		if !cur && !prev {
			continue
		}
		a, ok := idx[inv.PartyName]
		if !ok {
			a = &Amount{Key: inv.PartyName, Label: inv.PartyName}
			idx[inv.PartyName] = a
		}
		if cur {
			a.Value += inv.Subtotal
			a.Count++
		} else {
			a.Previous += inv.Subtotal
		}
	}
	out := []Amount{}
	for _, a := range idx {
		out = append(out, *a)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return out[i].Label < out[j].Label
	})
	return out, nil
}

// ─── Cash ───────────────────────────────────────────────────

type CashAccountSummary struct {
	AccountID, Code, Name             string
	Opening, Inflow, Outflow, Closing domain.Money
}

type FinanceCash struct {
	Window          Window
	Kpis            []Kpi
	Flows, Balances []Series
	Bridge          []WaterfallStep
	Accounts        []CashAccountSummary
}

var cashCategories = []struct{ key, category, label string }{
	{"operating", "OPERATING", "Operasi"}, {"investing", "INVESTING", "Investasi"},
	{"financing", "FINANCING", "Pendanaan"}, {"non_cash", "NON_CASH", "Non kas"},
}

// flowOf is the indirect-lite cash effect per category of non-cash lines;
// uncategorized accounts count as non cash.
func flowOf(rows []domain.MonthMovement) map[string]domain.Money {
	out := map[string]domain.Money{}
	for _, m := range rows {
		if m.IsCashBank {
			out["net"] += m.Debit - m.Credit
			continue
		}
		key := "non_cash"
		for _, c := range cashCategories {
			if c.category == m.CashFlowCategory {
				key = c.key
			}
		}
		out[key] += domain.CashFlowContribution(m.TypeCode, m.Net())
	}
	return out
}

func (r *Reports) CashAnalytics(ctx context.Context, f AnalyticsFilter) (*FinanceCash, error) {
	w, err := r.window(f)
	if err != nil {
		return nil, err
	}
	mon, err := r.monthly(ctx, w)
	if err != nil {
		return nil, err
	}
	flows := map[string]map[string]domain.Money{}
	for _, m := range append(append([]string{}, w.Months...), w.PrevMonths...) {
		flows[m] = flowOf(mon[m])
	}
	out := &FinanceCash{Window: w}
	for _, c := range append(cashCategories, struct{ key, category, label string }{"net", "", "Perubahan kas"}) {
		key := c.key
		s := Series{Key: key, Label: c.label,
			Values:     series(w.Months, func(m string) domain.Money { return flows[m][key] }),
			PrevValues: series(w.PrevMonths, func(m string) domain.Money { return flows[m][key] })}
		s.Total = sum(s.Values)
		out.Flows = append(out.Flows, s)
	}

	bal := r.balancesAt(ctx)
	cashID := trim(f.CashAccountID)
	closingRows, err := bal.at(w.To)
	if err != nil {
		return nil, err
	}
	for _, b := range closingRows {
		if !b.IsCashBank || (cashID != "" && b.ID != cashID) {
			continue
		}
		id := b.ID
		vals, err := bal.monthEnds(w.Months, w.clipCur, func(rows []domain.AccountBalance) domain.Money { return cashOf(rows, id) })
		if err != nil {
			return nil, err
		}
		out.Balances = append(out.Balances, Series{Key: b.ID, Label: b.Name, Code: b.Code, Values: vals, Total: b.Balance})
	}

	cf, err := r.CashFlow(ctx, w.From, w.To)
	if err != nil {
		return nil, err
	}
	prevCf, err := r.CashFlow(ctx, w.PrevFrom, w.PrevTo)
	if err != nil {
		return nil, err
	}
	activities := cf.Operating.Total + cf.Investing.Total + cf.Financing.Total
	out.Bridge = []WaterfallStep{
		{Key: "opening", Label: "Kas awal", Value: cf.Opening, Total: true},
		{Key: "operating", Label: "Operasi", Value: cf.Operating.Total},
		{Key: "investing", Label: "Investasi", Value: cf.Investing.Total},
		{Key: "financing", Label: "Pendanaan", Value: cf.Financing.Total},
		{Key: "other", Label: "Lainnya", Value: cf.NetChange - activities},
		{Key: "closing", Label: "Kas akhir", Value: cf.Closing, Total: true},
	}

	openRows, err := bal.at(dayBefore(w.From))
	if err != nil {
		return nil, err
	}
	period, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: w.From, To: w.To})
	if err != nil {
		return nil, err
	}
	moves := map[string]domain.AccountBalance{}
	for _, b := range period {
		moves[b.ID] = b
	}
	for _, b := range closingRows {
		if !b.IsCashBank || (cashID != "" && b.ID != cashID) {
			continue
		}
		out.Accounts = append(out.Accounts, CashAccountSummary{AccountID: b.ID, Code: b.Code, Name: b.Name,
			Opening: cashOf(openRows, b.ID), Inflow: moves[b.ID].Debit, Outflow: moves[b.ID].Credit, Closing: b.Balance})
	}

	prevClose, err := bal.at(w.PrevTo)
	if err != nil {
		return nil, err
	}
	prevOpen, err := bal.at(dayBefore(w.PrevFrom))
	if err != nil {
		return nil, err
	}
	cashTrend, err := bal.monthEnds(w.Months, w.clipCur, func(rows []domain.AccountBalance) domain.Money { return cashOf(rows, cashID) })
	if err != nil {
		return nil, err
	}
	closing, opening := cashOf(closingRows, cashID), cashOf(openRows, cashID)
	pClosing, pOpening := cashOf(prevClose, cashID), cashOf(prevOpen, cashID)
	burn := func(open, close domain.Money, days int) domain.Money {
		if open <= close {
			return 0
		}
		return domain.FromFloat((open - close).Float() / (math.Max(float64(days), 1) / 30.4375))
	}
	b, pb := burn(opening, closing, daysIn(w.From, w.To)), burn(pOpening, pClosing, daysIn(w.PrevFrom, w.PrevTo))
	runway := func(cash, burn domain.Money) string {
		if burn <= 0 {
			return ""
		}
		return num(cash.Float() / burn.Float())
	}
	flowTrend := func(key string) []domain.Money {
		return series(w.Months, func(m string) domain.Money { return flows[m][key] })
	}
	out.Kpis = []Kpi{
		moneyKpi("cash", closing, pClosing, cashTrend),
		moneyKpi("net_change", closing-opening, pClosing-pOpening, flowTrend("net")),
		moneyKpi("operating", cf.Operating.Total, prevCf.Operating.Total, flowTrend("operating")),
		moneyKpi("investing", cf.Investing.Total, prevCf.Investing.Total, flowTrend("investing")),
		moneyKpi("financing", cf.Financing.Total, prevCf.Financing.Total, flowTrend("financing")),
		moneyKpi("burn_rate", b, pb, nil),
		{Key: "runway", Value: runway(closing, b), Previous: runway(pClosing, pb)},
	}
	return out, nil
}

// ─── Receivables and payables ───────────────────────────────

// invoiceBook replays posted invoices and payments to any date.
type invoiceBook struct {
	invoices []domain.InvoiceFact
	paid     map[string][]domain.AllocationFact
}

func (r *Reports) invoiceBook(ctx context.Context, asOf string) (*invoiceBook, error) {
	inv, err := r.Store.InvoiceFacts(ctx, asOf)
	if err != nil {
		return nil, err
	}
	alloc, err := r.Store.AllocationFacts(ctx, asOf)
	if err != nil {
		return nil, err
	}
	b := &invoiceBook{invoices: inv, paid: map[string][]domain.AllocationFact{}}
	for _, a := range alloc {
		b.paid[a.InvoiceID] = append(b.paid[a.InvoiceID], a)
	}
	return b, nil
}

func (b *invoiceBook) open(inv domain.InvoiceFact, asOf string) domain.Money {
	if inv.InvoiceDate > asOf {
		return 0
	}
	o := inv.Total
	for _, a := range b.paid[inv.ID] {
		if a.PaymentDate <= asOf {
			o -= a.Amount
		}
	}
	return max(o, 0)
}

func (b *invoiceBook) outstanding(kind, asOf string) domain.Money {
	var s domain.Money
	for _, inv := range b.invoices {
		if inv.Kind == kind {
			s += b.open(inv, asOf)
		}
	}
	return s
}

func (b *invoiceBook) overdue(kind, asOf string) domain.Money {
	var s domain.Money
	for _, inv := range b.invoices {
		if inv.Kind == kind && inv.DueDate != "" && inv.DueDate < asOf {
			s += b.open(inv, asOf)
		}
	}
	return s
}

// byBucket sums open amounts per aging bucket at asOf.
func (b *invoiceBook) byBucket(kind, asOf string) (map[string]domain.Money, map[string]int) {
	amounts, counts := map[string]domain.Money{}, map[string]int{}
	for _, inv := range b.invoices {
		if inv.Kind != kind {
			continue
		}
		o := b.open(inv, asOf)
		if bucket := domain.BucketAging(o, inv.DueDate, asOf); bucket != "" {
			amounts[bucket] += o
			counts[bucket]++
		}
	}
	return amounts, counts
}

type AgingRow struct {
	Kind, Bucket string
	Amount       domain.Money
	Count        int
}

type PartyBalance struct {
	Party                        string
	Outstanding, Overdue, Billed domain.Money
	Count, OldestDays            int
}

type OverdueInvoice struct {
	domain.InvoiceFact
	Outstanding domain.Money
	DaysOverdue int
}

type FinanceArAp struct {
	Window                 Window
	Kpis                   []Kpi
	Aging                  []AgingRow
	ArTrend, ApTrend, Days []Series
	Customers, Vendors     []PartyBalance
	Overdue                []OverdueInvoice
}

var bucketLabels = map[string]string{"current": "Belum jatuh tempo", "1_30": "1-30 hari", "31_60": "31-60 hari", "61_90": "61-90 hari", "90_plus": "> 90 hari"}

func (r *Reports) ArAp(ctx context.Context, f AnalyticsFilter) (*FinanceArAp, error) {
	w, err := r.window(f)
	if err != nil {
		return nil, err
	}
	book, err := r.invoiceBook(ctx, w.To)
	if err != nil {
		return nil, err
	}
	mon, err := r.monthly(ctx, w)
	if err != nil {
		return nil, err
	}
	out := &FinanceArAp{Window: w}
	for _, kind := range []string{domain.KindAR, domain.KindAP} {
		amounts, counts := book.byBucket(kind, w.To)
		for _, bucket := range domain.AgingBuckets {
			out.Aging = append(out.Aging, AgingRow{Kind: kind, Bucket: bucket, Amount: amounts[bucket], Count: counts[bucket]})
		}
		trend := make([]Series, len(domain.AgingBuckets))
		for i, bucket := range domain.AgingBuckets {
			trend[i] = Series{Key: bucket, Label: bucketLabels[bucket], Values: make([]domain.Money, len(w.Months))}
		}
		for mi, m := range w.Months {
			amounts, _ := book.byBucket(kind, monthEnd(m, w.clipCur(m)))
			for i, bucket := range domain.AgingBuckets {
				trend[i].Values[mi] = amounts[bucket]
			}
		}
		if kind == domain.KindAR {
			out.ArTrend = trend
		} else {
			out.ApTrend = trend
		}
	}

	// DSO and DPO per month: month-end open invoices over the month's
	// revenue, or its COGS plus operating expense, times its days.
	days := func(months []string, clip func(string) string) ([]string, []string) {
		dso, dpo := make([]string, len(months)), make([]string, len(months))
		for i, m := range months {
			end := monthEnd(m, clip(m))
			d := float64(daysIn(m+"-01", end))
			p := mon.pnl(m, nil)
			if p.Revenue != 0 {
				dso[i] = num(book.outstanding(domain.KindAR, end).Float() / p.Revenue.Float() * d)
			}
			if spend := p.COGS + p.Expense; spend != 0 {
				dpo[i] = num(book.outstanding(domain.KindAP, end).Float() / spend.Float() * d)
			}
		}
		return dso, dpo
	}
	dso, dpo := days(w.Months, w.clipCur)
	pdso, pdpo := days(w.PrevMonths, w.prevClip)
	out.Days = []Series{{Key: "dso", Label: "DSO", FloatValues: dso, FloatPrev: pdso}, {Key: "dpo", Label: "DPO", FloatValues: dpo, FloatPrev: pdpo}}

	cur, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: w.From, To: w.To})
	if err != nil {
		return nil, err
	}
	prev, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: w.PrevFrom, To: w.PrevTo})
	if err != nil {
		return nil, err
	}
	pc, pp := pnlOf(cur), pnlOf(prev)
	ratio := func(open, flow domain.Money, from, to string) string {
		if flow == 0 {
			return ""
		}
		return num(open.Float() / flow.Float() * float64(daysIn(from, to)))
	}
	ar, par := book.outstanding(domain.KindAR, w.To), book.outstanding(domain.KindAR, w.PrevTo)
	ap, pap := book.outstanding(domain.KindAP, w.To), book.outstanding(domain.KindAP, w.PrevTo)
	trendOf := func(kind string) []domain.Money {
		return series(w.Months, func(m string) domain.Money { return book.outstanding(kind, monthEnd(m, w.clipCur(m))) })
	}
	out.Kpis = []Kpi{
		moneyKpi("receivable", ar, par, trendOf(domain.KindAR)),
		moneyKpi("payable", ap, pap, trendOf(domain.KindAP)),
		moneyKpi("overdue_receivable", book.overdue(domain.KindAR, w.To), book.overdue(domain.KindAR, w.PrevTo), nil),
		moneyKpi("overdue_payable", book.overdue(domain.KindAP, w.To), book.overdue(domain.KindAP, w.PrevTo), nil),
		{Key: "dso", Value: ratio(ar, pc.Revenue, w.From, w.To), Previous: ratio(par, pp.Revenue, w.PrevFrom, w.PrevTo), Trend: dso},
		{Key: "dpo", Value: ratio(ap, pc.COGS+pc.Expense, w.From, w.To), Previous: ratio(pap, pp.COGS+pp.Expense, w.PrevFrom, w.PrevTo), Trend: dpo},
	}

	parties := map[string]map[string]*PartyBalance{domain.KindAR: {}, domain.KindAP: {}}
	for _, inv := range book.invoices {
		o := book.open(inv, w.To)
		billed := inv.InvoiceDate >= w.From && inv.InvoiceDate <= w.To
		if o == 0 && !billed {
			continue
		}
		p, ok := parties[inv.Kind][inv.PartyName]
		if !ok {
			p = &PartyBalance{Party: inv.PartyName}
			parties[inv.Kind][inv.PartyName] = p
		}
		if billed {
			p.Billed += inv.Total
		}
		if o == 0 {
			continue
		}
		p.Outstanding += o
		p.Count++
		if late := domain.DaysPastDue(inv.DueDate, w.To); inv.DueDate != "" && late > 0 {
			p.Overdue += o
			p.OldestDays = max(p.OldestDays, late)
			out.Overdue = append(out.Overdue, OverdueInvoice{InvoiceFact: inv, Outstanding: o, DaysOverdue: late})
		}
	}
	rank := func(m map[string]*PartyBalance) []PartyBalance {
		list := []PartyBalance{}
		for _, p := range m {
			list = append(list, *p)
		}
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Outstanding != list[j].Outstanding {
				return list[i].Outstanding > list[j].Outstanding
			}
			return list[i].Billed > list[j].Billed
		})
		return list
	}
	out.Customers, out.Vendors = rank(parties[domain.KindAR]), rank(parties[domain.KindAP])
	sort.SliceStable(out.Overdue, func(i, j int) bool { return out.Overdue[i].DaysOverdue > out.Overdue[j].DaysOverdue })
	if len(out.Overdue) > 50 {
		out.Overdue = out.Overdue[:50]
	}
	return out, nil
}

// ─── Expenses and reimbursements ────────────────────────────

type FinanceSpend struct {
	Window                             Window
	Kpis                               []Kpi
	ExpenseGroups, ClaimCategories     []Series
	ByCategory, ByDepartment, ByStatus []Amount
	LeadTime                           []Amount
	LeadTimeTrend                      []Series
	Pending                            []domain.ClaimFact
	Departments                        []string
}

var categoryLabels = map[string]string{
	domain.CategoryTransport: "Transport", domain.CategoryMakan: "Makan", domain.CategoryAkomodasi: "Akomodasi",
	domain.CategoryOperasional: "Operasional proyek", domain.CategoryLainnya: "Lainnya",
}

func isPendingClaim(status string) bool {
	switch domain.ClaimStatus(status) {
	case domain.ClaimSubmitted, domain.ClaimApprovedManager, domain.ClaimApprovedFinance:
		return true
	}
	return false
}

// claimed counts toward claimed amounts: submitted at some point and not
// withdrawn.
func claimed(status string) bool {
	return status != string(domain.ClaimDraft) && status != string(domain.ClaimCancelled)
}

func daysBetween(a, b *time.Time) (float64, bool) {
	if a == nil || b == nil || b.Before(*a) {
		return 0, false
	}
	return b.Sub(*a).Hours() / 24, true
}

func avg(total float64, n int) string {
	if n == 0 {
		return ""
	}
	return num(total / float64(n))
}

func inRange(date, from, to string) bool { return date >= from && date <= to }

func (r *Reports) Spend(ctx context.Context, f AnalyticsFilter) (*FinanceSpend, error) {
	w, err := r.window(f)
	if err != nil {
		return nil, err
	}
	af, err := r.accountFilter(ctx, f)
	if err != nil {
		return nil, err
	}
	mon, err := r.monthly(ctx, w)
	if err != nil {
		return nil, err
	}
	out := &FinanceSpend{Window: w}

	groups := map[string]*Series{}
	var order []string
	for i, m := range w.Months {
		for _, mv := range mon[m] {
			if !domain.IsExpenseType(mv.TypeCode) || !af.keepMove(mv) {
				continue
			}
			key, label := af.groupName(mv)
			s, ok := groups[key]
			if !ok {
				s = &Series{Key: key, Label: label, Values: make([]domain.Money, len(w.Months))}
				groups[key] = s
				order = append(order, key)
			}
			s.Values[i] += mv.Net()
			s.Total += mv.Net()
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return groups[order[i]].Total > groups[order[j]].Total })
	for _, k := range order {
		out.ExpenseGroups = append(out.ExpenseGroups, *groups[k])
	}
	cur, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: w.From, To: w.To})
	if err != nil {
		return nil, err
	}
	prev, err := r.Store.AccountBalances(ctx, domain.BalanceWindow{From: w.PrevFrom, To: w.PrevTo})
	if err != nil {
		return nil, err
	}
	expense := func(rows []domain.AccountBalance) domain.Money {
		var s domain.Money
		for _, b := range rows {
			if domain.IsExpenseType(b.TypeCode) && af.keep(b.ID, b.TypeCode) {
				s += b.Net()
			}
		}
		return s
	}
	expenseTrend := series(w.Months, func(m string) domain.Money {
		var s domain.Money
		for _, mv := range mon[m] {
			if domain.IsExpenseType(mv.TypeCode) && af.keepMove(mv) {
				s += mv.Net()
			}
		}
		return s
	})

	start := w.Months[0] + "-01"
	if w.PrevFrom < start {
		start = w.PrevFrom
	}
	claims, err := r.Store.ClaimFacts(ctx, start, w.To, trim(f.Department))
	if err != nil {
		return nil, err
	}
	if out.Departments, err = r.Store.ClaimDepartments(ctx); err != nil {
		return nil, err
	}
	type tally struct {
		paid, pending            domain.Money
		count, rejected, decided int
		leadDays                 float64
		leadN                    int
	}
	var tc, tp tally
	cats, depts, statuses := map[string]*Amount{}, map[string]*Amount{}, map[string]*Amount{}
	slice := func(m map[string]*Amount, key, label string) *Amount {
		a, ok := m[key]
		if !ok {
			a = &Amount{Key: key, Label: label}
			m[key] = a
		}
		return a
	}
	catMonths := map[string]*Series{}
	stage := [4]struct {
		total float64
		n     int
	}{}
	leadByMonth := map[string]*struct {
		total float64
		n     int
	}{}
	for _, c := range claims {
		if isPendingClaim(c.Status) {
			tc.pending += c.Amount
			out.Pending = append(out.Pending, c)
		}
		paidOn := ""
		if c.PaidAt != nil {
			paidOn = c.PaidAt.In(Jakarta).Format(time.DateOnly)
		}
		for _, side := range []struct {
			t        *tally
			from, to string
			cur      bool
		}{{&tc, w.From, w.To, true}, {&tp, w.PrevFrom, w.PrevTo, false}} {
			if paidOn != "" && inRange(paidOn, side.from, side.to) {
				side.t.paid += c.Amount
				if d, ok := daysBetween(c.SubmittedAt, c.PaidAt); ok {
					side.t.leadDays += d
					side.t.leadN++
				}
			}
			if !inRange(c.TransactionDate, side.from, side.to) {
				continue
			}
			if side.cur {
				slice(statuses, c.Status, c.Status).Value += c.Amount
				statuses[c.Status].Count++
			}
			if !claimed(c.Status) {
				continue
			}
			side.t.count++
			switch domain.ClaimStatus(c.Status) {
			case domain.ClaimRejected:
				side.t.rejected++
				side.t.decided++
			case domain.ClaimApprovedFinance, domain.ClaimPaid:
				side.t.decided++
			}
			cat := slice(cats, c.Category, categoryLabels[c.Category])
			dept := slice(depts, c.DepartmentName, orDash(c.DepartmentName))
			if side.cur {
				cat.Value += c.Amount
				cat.Count++
				dept.Value += c.Amount
				dept.Count++
				if d, ok := daysBetween(c.SubmittedAt, c.ManagerApprovedAt); ok {
					stage[0].total, stage[0].n = stage[0].total+d, stage[0].n+1
				}
				if d, ok := daysBetween(c.ManagerApprovedAt, c.FinanceApprovedAt); ok {
					stage[1].total, stage[1].n = stage[1].total+d, stage[1].n+1
				}
				if d, ok := daysBetween(c.FinanceApprovedAt, c.PaidAt); ok {
					stage[2].total, stage[2].n = stage[2].total+d, stage[2].n+1
				}
				if d, ok := daysBetween(c.SubmittedAt, c.PaidAt); ok {
					stage[3].total, stage[3].n = stage[3].total+d, stage[3].n+1
				}
			} else {
				cat.Previous += c.Amount
				dept.Previous += c.Amount
			}
		}
		if claimed(c.Status) {
			if mi := indexOf(w.Months, c.TransactionDate[:7]); mi >= 0 {
				s, ok := catMonths[c.Category]
				if !ok {
					s = &Series{Key: c.Category, Label: categoryLabels[c.Category], Values: make([]domain.Money, len(w.Months))}
					catMonths[c.Category] = s
				}
				s.Values[mi] += c.Amount
				s.Total += c.Amount
			}
		}
		if paidOn != "" {
			if d, ok := daysBetween(c.SubmittedAt, c.PaidAt); ok {
				lm, ok := leadByMonth[paidOn[:7]]
				if !ok {
					lm = &struct {
						total float64
						n     int
					}{}
					leadByMonth[paidOn[:7]] = lm
				}
				lm.total, lm.n = lm.total+d, lm.n+1
			}
		}
	}
	for _, cat := range []string{domain.CategoryTransport, domain.CategoryMakan, domain.CategoryAkomodasi, domain.CategoryOperasional, domain.CategoryLainnya} {
		if s, ok := catMonths[cat]; ok {
			out.ClaimCategories = append(out.ClaimCategories, *s)
		}
	}
	collect := func(m map[string]*Amount, keepZero bool) []Amount {
		list := []Amount{}
		for _, a := range m {
			if keepZero || a.Value != 0 {
				list = append(list, *a)
			}
		}
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].Value != list[j].Value {
				return list[i].Value > list[j].Value
			}
			return list[i].Label < list[j].Label
		})
		return list
	}
	out.ByCategory, out.ByDepartment, out.ByStatus = collect(cats, false), collect(depts, false), collect(statuses, true)
	for i, s := range []struct{ key, label string }{
		{"manager", "Pengajuan ke persetujuan atasan"}, {"finance", "Atasan ke persetujuan finance"},
		{"payment", "Finance ke pembayaran"}, {"total", "Pengajuan ke pembayaran"},
	} {
		out.LeadTime = append(out.LeadTime, Amount{Key: s.key, Label: s.label, IsDecimal: true, Decimal: avg(stage[i].total, stage[i].n), Count: stage[i].n})
	}
	lead := make([]string, len(w.Months))
	for i, m := range w.Months {
		if lm, ok := leadByMonth[m]; ok {
			lead[i] = avg(lm.total, lm.n)
		}
	}
	out.LeadTimeTrend = []Series{{Key: "lead_days", Label: "Rata-rata hari sampai dibayar", FloatValues: lead}}
	sort.SliceStable(out.Pending, func(i, j int) bool { return submittedOrDate(out.Pending[i]) < submittedOrDate(out.Pending[j]) })
	rate := func(t tally) string {
		if t.decided == 0 {
			return ""
		}
		return num(float64(t.rejected) / float64(t.decided) * 100)
	}
	out.Kpis = []Kpi{
		moneyKpi("expense", expense(cur), expense(prev), expenseTrend),
		moneyKpi("reimburse_paid", tc.paid, tp.paid, nil),
		{Key: "reimburse_pending", Value: tc.pending.String()},
		{Key: "claims", Value: strconv.Itoa(tc.count), Previous: strconv.Itoa(tp.count)},
		{Key: "approval_days", Value: avg(tc.leadDays, tc.leadN), Previous: avg(tp.leadDays, tp.leadN), Trend: lead},
		{Key: "rejection_rate", Value: rate(tc), Previous: rate(tp)},
	}
	return out, nil
}

func submittedOrDate(c domain.ClaimFact) string {
	if c.SubmittedAt != nil {
		return c.SubmittedAt.Format(time.RFC3339)
	}
	return c.TransactionDate
}

func orDash(s string) string {
	if s == "" {
		return "Tanpa departemen"
	}
	return s
}

func indexOf(list []string, v string) int {
	for i, s := range list {
		if s == v {
			return i
		}
	}
	return -1
}
