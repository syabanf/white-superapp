package grpcapi

import (
	"context"
	"time"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

// ─── Dashboard tabs ─────────────────────────────────────────

func filterFrom(req *financev1.FinanceAnalyticsRequest) app.AnalyticsFilter {
	return app.AnalyticsFilter{From: req.From, To: req.To, Compare: req.Compare, AccountType: req.AccountType,
		AccountGroup: req.AccountGroup, Department: req.Department, CashAccountID: req.CashAccountId}
}

func windowPB(w app.Window) *financev1.AnalyticsWindow {
	return &financev1.AnalyticsWindow{From: w.From, To: w.To, Compare: w.Compare, CompareFrom: w.PrevFrom, CompareTo: w.PrevTo,
		Months: w.Months, CompareMonths: w.PrevMonths}
}

func kpisPB(ks []app.Kpi) []*financev1.AnalyticsKpi {
	out := make([]*financev1.AnalyticsKpi, len(ks))
	for i, k := range ks {
		out[i] = &financev1.AnalyticsKpi{Key: k.Key, Value: k.Value, Previous: k.Previous, Trend: k.Trend}
	}
	return out
}

func moneyStrings(ms []domain.Money) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.String()
	}
	return out
}

func seriesPB(ss []app.Series) []*financev1.AnalyticsSeries {
	out := make([]*financev1.AnalyticsSeries, len(ss))
	for i, s := range ss {
		p := &financev1.AnalyticsSeries{Key: s.Key, Label: s.Label, Code: s.Code, Values: s.FloatValues, CompareValues: s.FloatPrev}
		if s.FloatValues == nil {
			p.Values, p.CompareValues, p.Total = moneyStrings(s.Values), moneyStrings(s.PrevValues), s.Total.String()
		}
		out[i] = p
	}
	return out
}

func amountsPB(as []app.Amount) []*financev1.AnalyticsAmount {
	out := make([]*financev1.AnalyticsAmount, len(as))
	for i, a := range as {
		v := a.Value.String()
		if a.IsDecimal {
			v = a.Decimal
		}
		out[i] = &financev1.AnalyticsAmount{Key: a.Key, Label: a.Label, Code: a.Code, Group: a.Group, Value: v, Previous: a.Previous.String(), Count: int32(a.Count)}
	}
	return out
}

func stepsPB(steps []app.WaterfallStep) []*financev1.WaterfallStep {
	out := make([]*financev1.WaterfallStep, len(steps))
	for i, s := range steps {
		out[i] = &financev1.WaterfallStep{Key: s.Key, Label: s.Label, Value: s.Value.String(), Total: s.Total}
	}
	return out
}

func (s *reportServer) GetFinanceSummary(ctx context.Context, req *financev1.FinanceAnalyticsRequest) (*financev1.FinanceSummary, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Reports.Summary(ctx, filterFrom(req))
	if err != nil {
		return nil, err
	}
	return &financev1.FinanceSummary{Window: windowPB(d.Window), Kpis: kpisPB(d.Kpis), Pnl: seriesPB(d.Pnl), Cash: seriesPB(d.Cash),
		Assets: amountsPB(d.Assets), Liabilities: amountsPB(d.Liabilities), Equity: amountsPB(d.Equity),
		TrialBalanceDiff: d.TrialBalanceDiff.String(), PostedEntries: int32(d.PostedEntries), DraftEntries: int32(d.Drafts)}, nil
}

func (s *reportServer) GetFinanceProfitability(ctx context.Context, req *financev1.FinanceAnalyticsRequest) (*financev1.FinanceProfitability, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Reports.Profitability(ctx, filterFrom(req))
	if err != nil {
		return nil, err
	}
	return &financev1.FinanceProfitability{Window: windowPB(d.Window), Kpis: kpisPB(d.Kpis), Waterfall: stepsPB(d.Waterfall), Pnl: seriesPB(d.Pnl),
		RevenueAccounts: amountsPB(d.RevenueAccounts), RevenueClients: amountsPB(d.RevenueClients), ExpenseGroups: amountsPB(d.ExpenseGroups),
		ExpenseAccounts: amountsPB(d.ExpenseAccounts), AccountMonths: seriesPB(d.AccountMonths)}, nil
}

func (s *reportServer) GetFinanceCashAnalytics(ctx context.Context, req *financev1.FinanceAnalyticsRequest) (*financev1.FinanceCashAnalytics, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Reports.CashAnalytics(ctx, filterFrom(req))
	if err != nil {
		return nil, err
	}
	out := &financev1.FinanceCashAnalytics{Window: windowPB(d.Window), Kpis: kpisPB(d.Kpis), Flows: seriesPB(d.Flows), Balances: seriesPB(d.Balances), Bridge: stepsPB(d.Bridge)}
	for _, a := range d.Accounts {
		out.Accounts = append(out.Accounts, &financev1.CashAccountSummary{AccountId: a.AccountID, Code: a.Code, Name: a.Name, Opening: a.Opening.String(),
			Inflow: a.Inflow.String(), Outflow: a.Outflow.String(), Closing: a.Closing.String()})
	}
	return out, nil
}

func partiesPB(ps []app.PartyBalance) []*financev1.PartyBalance {
	out := make([]*financev1.PartyBalance, len(ps))
	for i, p := range ps {
		out[i] = &financev1.PartyBalance{PartyName: p.Party, Outstanding: p.Outstanding.String(), Overdue: p.Overdue.String(), InvoiceCount: int32(p.Count),
			OldestDays: int32(p.OldestDays), Billed: p.Billed.String()}
	}
	return out
}

func (s *reportServer) GetFinanceArAp(ctx context.Context, req *financev1.FinanceAnalyticsRequest) (*financev1.FinanceArAp, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Reports.ArAp(ctx, filterFrom(req))
	if err != nil {
		return nil, err
	}
	out := &financev1.FinanceArAp{Window: windowPB(d.Window), Kpis: kpisPB(d.Kpis), ArAgingTrend: seriesPB(d.ArTrend), ApAgingTrend: seriesPB(d.ApTrend),
		Days: seriesPB(d.Days), Customers: partiesPB(d.Customers), Vendors: partiesPB(d.Vendors)}
	for _, a := range d.Aging {
		out.Aging = append(out.Aging, &financev1.AgingRow{Kind: a.Kind, Bucket: a.Bucket, Amount: a.Amount.String(), Count: int32(a.Count)})
	}
	for _, o := range d.Overdue {
		out.Overdue = append(out.Overdue, &financev1.OverdueInvoice{Id: o.ID, Kind: o.Kind, InvoiceNo: o.InvoiceNo, PartyName: o.PartyName, InvoiceDate: o.InvoiceDate,
			DueDate: o.DueDate, Total: o.Total.String(), Outstanding: o.Outstanding.String(), DaysOverdue: int32(o.DaysOverdue)})
	}
	return out, nil
}

func (s *reportServer) GetFinanceSpend(ctx context.Context, req *financev1.FinanceAnalyticsRequest) (*financev1.FinanceSpend, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	d, err := s.svc.Reports.Spend(ctx, filterFrom(req))
	if err != nil {
		return nil, err
	}
	out := &financev1.FinanceSpend{Window: windowPB(d.Window), Kpis: kpisPB(d.Kpis), ExpenseGroups: seriesPB(d.ExpenseGroups), ClaimCategories: seriesPB(d.ClaimCategories),
		ByCategory: amountsPB(d.ByCategory), ByDepartment: amountsPB(d.ByDepartment), ByStatus: amountsPB(d.ByStatus), LeadTime: amountsPB(d.LeadTime),
		LeadTimeTrend: seriesPB(d.LeadTimeTrend), Departments: d.Departments}
	now := time.Now()
	for _, c := range d.Pending {
		age := 0
		if c.SubmittedAt != nil {
			age = int(now.Sub(*c.SubmittedAt).Hours() / 24)
		}
		out.Pending = append(out.Pending, &financev1.ClaimSummary{Id: c.ID, ClaimNo: c.ClaimNo, EmployeeName: c.EmployeeName, DepartmentName: c.DepartmentName,
			Category: c.Category, TransactionDate: c.TransactionDate, Amount: c.Amount.String(), Status: c.Status, AgeDays: int32(age)})
	}
	return out, nil
}

func (s *analysisServer) GetRatioTrend(ctx context.Context, req *financev1.RatioTrendRequest) (*financev1.RatioTrend, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	months, series, err := s.svc.Analysis.RatioTrend(ctx, req.To, int(req.Months))
	if err != nil {
		return nil, err
	}
	out := &financev1.RatioTrend{Months: months}
	for _, rs := range series {
		p := &financev1.RatioTrendSeries{Key: rs.Key, Label: rs.Label, Group: rs.Group, Unit: rs.Unit, LowerIsBetter: rs.LowerIsBetter, Benchmark: rs.Benchmark}
		for i, pt := range rs.Points {
			p.Points = append(p.Points, &financev1.RatioPoint{Month: months[i], Value: pt.Value, Unavailable: !pt.OK})
		}
		out.Series = append(out.Series, p)
	}
	return out, nil
}
