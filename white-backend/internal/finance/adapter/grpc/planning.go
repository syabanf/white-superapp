package grpcapi

import (
	"context"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

type projectTagServer struct {
	financev1.UnimplementedProjectTagServiceServer
	svc Services
}

func projectTagPB(t *domain.ProjectTag) *financev1.ProjectTag {
	return &financev1.ProjectTag{Id: t.ID, Name: t.Name, ClientName: t.ClientName, DealId: t.DealID, IsActive: t.IsActive, Notes: t.Notes}
}

func (s *projectTagServer) ListProjectTags(ctx context.Context, r *financev1.ListProjectTagsRequest) (*financev1.ListProjectTagsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.ProjectTags.List(ctx, r.ActiveOnly)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListProjectTagsResponse{}
	for i := range rows {
		out.Tags = append(out.Tags, projectTagPB(&rows[i]))
	}
	return out, nil
}

func (s *projectTagServer) SaveProjectTag(ctx context.Context, r *financev1.ProjectTag) (*financev1.ProjectTag, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	t, err := s.svc.ProjectTags.Save(ctx, p.UserID, domain.ProjectTag{ID: r.Id, Name: r.Name, ClientName: r.ClientName, IsActive: r.IsActive, Notes: r.Notes})
	if err != nil {
		return nil, err
	}
	return projectTagPB(t), nil
}

func (s *projectTagServer) DeleteProjectTag(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := write(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.ProjectTags.Delete(ctx, r.Id)
}

func (s *projectTagServer) GetProjectProfit(ctx context.Context, r *financev1.ProjectProfitRequest) (*financev1.ProjectProfit, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, from, to, err := s.svc.ProjectTags.ProjectProfit(ctx, r.DateFrom, r.DateTo, r.ClientName)
	if err != nil {
		return nil, err
	}
	out := &financev1.ProjectProfit{DateFrom: from, DateTo: to}
	var revenue, cost domain.Money
	for _, row := range rows {
		revenue += row.Revenue
		cost += row.Cost
		out.Rows = append(out.Rows, &financev1.ProjectProfitRow{TagId: row.TagID, TagName: row.TagName, ClientName: row.ClientName, Revenue: row.Revenue.String(), Cost: row.Cost.String(), Profit: (row.Revenue - row.Cost).String()})
	}
	out.TotalRevenue, out.TotalCost, out.TotalProfit = revenue.String(), cost.String(), (revenue - cost).String()
	return out, nil
}

type advanceServer struct {
	financev1.UnimplementedAdvanceServiceServer
	svc Services
}

func advancePB(a *domain.Advance) *financev1.Advance {
	return &financev1.Advance{Id: a.ID, Kind: a.Kind, AdvanceNo: a.AdvanceNo, AdvanceDate: a.Date, PartyName: a.PartyName, Amount: a.Amount.String(), Applied: a.Applied.String(), Remaining: a.Remaining().String(),
		CashAccountId: a.CashAccountID, CashAccountName: a.CashAccountName, AdvanceAccountId: a.AdvanceAccountID, AdvanceAccountName: a.AdvanceAccountName, Reference: a.Reference, Notes: a.Notes,
		Status: a.Status, JournalEntryId: a.JournalEntryID, VoidReason: a.VoidReason}
}

func (s *advanceServer) ListAdvances(ctx context.Context, r *financev1.ListAdvancesRequest) (*financev1.ListAdvancesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Ledgers.ListAdvances(ctx, r.Kind)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListAdvancesResponse{}
	for i := range rows {
		out.Advances = append(out.Advances, advancePB(&rows[i]))
	}
	return out, nil
}

func (s *advanceServer) CreateAdvance(ctx context.Context, r *financev1.AdvanceInput) (*financev1.Advance, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", r.Amount)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Ledgers.CreateAdvance(ctx, p.UserID, app.AdvanceInput{Kind: r.Kind, Date: r.AdvanceDate, PartyName: r.PartyName, CashAccountID: r.CashAccountId, AdvanceAccountID: r.AdvanceAccountId, Reference: r.Reference, Notes: r.Notes, Amount: amount})
	if err != nil {
		return nil, err
	}
	return advancePB(a), nil
}

func (s *advanceServer) ApplyAdvance(ctx context.Context, r *financev1.ApplyAdvanceRequest) (*financev1.Invoice, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", r.Amount)
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Ledgers.ApplyAdvance(ctx, p.UserID, r.Id, r.InvoiceId, r.Date, amount)
	if err != nil {
		return nil, err
	}
	inv := &res.Invoices[0]
	return invoicePB(inv, s.svc.Ledgers.PaymentStatus(*inv)), nil
}

func (s *advanceServer) VoidAdvance(ctx context.Context, r *financev1.VoidAdvanceRequest) (*financev1.Advance, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	a, err := s.svc.Ledgers.VoidAdvance(ctx, p.UserID, r.Id, r.Reason)
	if err != nil {
		return nil, err
	}
	return advancePB(a), nil
}
