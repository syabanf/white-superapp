package grpc

import (
	"context"

	salesv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/sales/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/app"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

// dashboard checks sales.read and builds the shared query.
func dashboard(ctx context.Context, r *salesv1.DashboardRequest) (app.DashQuery, error) {
	if _, err := reader(ctx); err != nil {
		return app.DashQuery{}, err
	}
	return app.DashQuery{From: r.From, To: r.To, Compare: r.Compare, Filter: domain.DashFilter{
		PipelineID: r.PipelineId, OwnerUserID: r.OwnerUserId, ServiceLine: r.ServiceLine, Source: r.Source,
	}}, nil
}

func (s *insightServer) GetDashboardSummary(ctx context.Context, r *salesv1.DashboardRequest) (*salesv1.DashboardSummary, error) {
	q, err := dashboard(ctx, r)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Dashboard.Summary(ctx, q)
	if err != nil {
		return nil, err
	}
	return &salesv1.DashboardSummary{Period: periodPB(d.Period), Kpis: kpisPB(d.Kpis), Trend: seriesPB(d.Trend), Stages: slicesPB(d.Stages)}, nil
}

func (s *insightServer) GetDashboardPipeline(ctx context.Context, r *salesv1.DashboardRequest) (*salesv1.DashboardPipeline, error) {
	q, err := dashboard(ctx, r)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Dashboard.Pipeline(ctx, q)
	if err != nil {
		return nil, err
	}
	out := &salesv1.DashboardPipeline{Period: periodPB(d.Period), PipelineId: d.PipelineID, Kpis: kpisPB(d.Kpis), Funnel: slicesPB(d.Funnel),
		StageLines: slicesPB(d.StageLines), Aging: slicesPB(d.Aging), StageDays: slicesPB(d.StageDays)}
	for _, x := range d.Stuck {
		out.Stuck = append(out.Stuck, &salesv1.DashboardDeal{Id: x.ID, Title: x.Title, AccountName: x.AccountName, StageId: x.StageID, StageName: x.StageName,
			OwnerUserId: x.OwnerUserID, OwnerName: x.OwnerName, Value: x.Value, DaysInStage: int32(x.DaysInStage), ThresholdDays: int32(x.ThresholdDays)})
	}
	return out, nil
}

func (s *insightServer) GetDashboardSales(ctx context.Context, r *salesv1.DashboardRequest) (*salesv1.DashboardSales, error) {
	q, err := dashboard(ctx, r)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Dashboard.Sales(ctx, q)
	if err != nil {
		return nil, err
	}
	return &salesv1.DashboardSales{Period: periodPB(d.Period), Kpis: kpisPB(d.Kpis), Monthly: seriesPB(d.Monthly), ServiceLines: slicesPB(d.ServiceLines),
		Industries: slicesPB(d.Industries), LostReasons: slicesPB(d.LostReasons), QuoteStatuses: slicesPB(d.QuoteStatuses), Sources: slicesPB(d.Sources)}, nil
}

func (s *insightServer) GetDashboardTeam(ctx context.Context, r *salesv1.DashboardRequest) (*salesv1.DashboardTeam, error) {
	q, err := dashboard(ctx, r)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Dashboard.Team(ctx, q)
	if err != nil {
		return nil, err
	}
	out := &salesv1.DashboardTeam{Period: periodPB(d.Period), Kpis: kpisPB(d.Kpis), WonByOwner: seriesPB(d.WonByOwner)}
	for _, m := range d.Members {
		out.Members = append(out.Members, &salesv1.DashboardMember{UserId: m.UserID, Name: m.Name, OpenDeals: int32(m.OpenDeals), PipelineValue: m.PipelineValue,
			WonDeals: int32(m.WonDeals), WonValue: m.WonValue, LostDeals: int32(m.LostDeals), WinRate: m.WinRate, Activities: int32(m.Activities),
			ActivitiesDone: int32(m.ActivitiesDone), TargetValue: m.TargetValue, AttainmentPercent: m.AttainmentPercent, PreviousWonValue: m.PreviousWonValue})
	}
	return out, nil
}

func (s *insightServer) GetDashboardLeads(ctx context.Context, r *salesv1.DashboardRequest) (*salesv1.DashboardLeads, error) {
	q, err := dashboard(ctx, r)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Dashboard.Leads(ctx, q)
	if err != nil {
		return nil, err
	}
	return &salesv1.DashboardLeads{Period: periodPB(d.Period), Kpis: kpisPB(d.Kpis), MonthlySources: slicesPB(d.MonthlySources), Sources: slicesPB(d.Sources),
		Scores: slicesPB(d.Scores), Response: seriesPB(d.Response), Channels: slicesPB(d.Channels), Forms: slicesPB(d.Forms)}, nil
}

func (s *insightServer) GetDashboardBilling(ctx context.Context, r *salesv1.DashboardRequest) (*salesv1.DashboardBilling, error) {
	q, err := dashboard(ctx, r)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Dashboard.Billing(ctx, q)
	if err != nil {
		return nil, err
	}
	out := &salesv1.DashboardBilling{Period: periodPB(d.Period), Kpis: kpisPB(d.Kpis), Monthly: seriesPB(d.Monthly), Aging: slicesPB(d.Aging)}
	for _, x := range d.Overdue {
		due := ""
		if x.DueDate != nil {
			due = x.DueDate.Format("2006-01-02")
		}
		out.Overdue = append(out.Overdue, &salesv1.DashboardInvoice{Id: x.ID, InvoiceNumber: x.InvoiceNumber, DealId: x.DealID, DealTitle: x.DealTitle,
			AccountName: x.AccountName, Amount: x.Amount, Paid: x.Paid, DueDate: due, DaysOverdue: int32(x.DaysOverdue)})
	}
	for _, t := range d.Terms {
		out.Terms = append(out.Terms, &salesv1.DashboardTerm{DealId: t.DealID, DealTitle: t.DealTitle, AccountName: t.AccountName, ContractValue: t.ContractValue,
			Invoiced: t.Invoiced, Paid: t.Paid, Terms: t.Terms, TermsInvoiced: t.TermsInvoiced})
	}
	return out, nil
}

func periodPB(p domain.DashPeriod) *salesv1.DashboardPeriod {
	const day = "2006-01-02"
	return &salesv1.DashboardPeriod{From: p.From.Format(day), To: p.To.Format(day), Compare: p.Compare, CompareFrom: p.CompareFrom.Format(day),
		CompareTo: p.CompareTo.Format(day), Months: p.Months, CompareMonths: p.CompareMonths}
}

func kpisPB(in []domain.DashKpi) []*salesv1.DashboardKpi {
	out := make([]*salesv1.DashboardKpi, len(in))
	for i, k := range in {
		out[i] = &salesv1.DashboardKpi{Key: k.Key, Value: k.Value, Previous: k.Previous, Spark: k.Spark}
	}
	return out
}

func seriesPB(in []domain.DashSeries) []*salesv1.DashboardSeries {
	out := make([]*salesv1.DashboardSeries, len(in))
	for i, s := range in {
		out[i] = &salesv1.DashboardSeries{Key: s.Key, Values: s.Values, Previous: s.Previous}
	}
	return out
}

func slicesPB(in []domain.DashSlice) []*salesv1.DashboardSlice {
	out := make([]*salesv1.DashboardSlice, len(in))
	for i, s := range in {
		out[i] = &salesv1.DashboardSlice{Key: s.Key, Label: s.Label, Group: s.Group, Value: s.Value, Count: int32(s.Count), Extra: s.Extra}
	}
	return out
}
