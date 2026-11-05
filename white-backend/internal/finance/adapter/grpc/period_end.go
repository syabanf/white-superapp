package grpcapi

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

type periodEndServer struct {
	financev1.UnimplementedPeriodEndServiceServer
	svc Services
}

func deferralPB(d *domain.Deferral) *financev1.Deferral {
	return &financev1.Deferral{Id: d.ID, Kind: d.Kind, Name: d.Name, PartyName: d.PartyName, Reference: d.Reference, DeferredAccountId: d.DeferredAccountID, DeferredAccountName: d.DeferredAccountName,
		TargetAccountId: d.TargetAccountID, TargetAccountName: d.TargetAccountName, TagId: d.TagID, StartDate: d.StartDate, Months: int32(d.Months), PostedMonths: int32(d.PostedMonths), Amount: d.Amount.String(),
		Recognized: d.Recognized().String(), IsActive: d.IsActive, LastError: d.LastError}
}

func (s *periodEndServer) ListDeferrals(ctx context.Context, _ *financev1.ListDeferralsRequest) (*financev1.ListDeferralsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Ledgers.ListDeferrals(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListDeferralsResponse{}
	for i := range rows {
		out.Deferrals = append(out.Deferrals, deferralPB(&rows[i]))
	}
	return out, nil
}

func (s *periodEndServer) CreateDeferral(ctx context.Context, r *financev1.Deferral) (*financev1.Deferral, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", orZero(r.Amount))
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Ledgers.CreateDeferral(ctx, p.UserID, domain.Deferral{Kind: r.Kind, Name: r.Name, PartyName: r.PartyName, Reference: r.Reference, DeferredAccountID: r.DeferredAccountId, TargetAccountID: r.TargetAccountId,
		TagID: r.TagId, StartDate: r.StartDate, Months: int(r.Months), Amount: amount})
	if err != nil {
		return nil, err
	}
	return deferralPB(d), nil
}

func (s *periodEndServer) SetDeferralActive(ctx context.Context, r *financev1.SetDeferralActiveRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.Ledgers.SetDeferralActive(ctx, r.Id, r.Active)
}

func (s *periodEndServer) DeleteDeferral(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.Ledgers.DeleteDeferral(ctx, r.Id)
}

func (s *periodEndServer) WriteOffInvoice(ctx context.Context, r *financev1.WriteOffRequest) (*financev1.Invoice, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", orZero(r.Amount))
	if err != nil {
		return nil, err
	}
	inv, err := s.svc.Ledgers.WriteOffInvoice(ctx, p.UserID, r.InvoiceId, app.WriteOffInput{Date: r.WriteOffDate, ExpenseAccountID: r.ExpenseAccountId, Reason: r.Reason, Amount: amount})
	if err != nil {
		return nil, err
	}
	return invoicePB(inv, s.svc.Ledgers.PaymentStatus(*inv)), nil
}

func (s *periodEndServer) ListTaxPositions(ctx context.Context, r *financev1.ListTaxPositionsRequest) (*financev1.ListTaxPositionsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Ledgers.TaxPositions(ctx, r.AsOf)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListTaxPositionsResponse{}
	for _, p := range rows {
		out.Positions = append(out.Positions, &financev1.TaxPosition{AccountId: p.AccountID, Code: p.Code, Name: p.Name, Kind: p.Kind, Balance: p.Balance.String()})
	}
	return out, nil
}

func taxRemittancePB(r *domain.TaxRemittance) *financev1.TaxRemittance {
	return &financev1.TaxRemittance{Id: r.ID, RemitNo: r.RemitNo, Period: r.Period, RemitDate: r.Date, AccountId: r.AccountID, AccountName: r.AccountName, CashAccountId: r.CashAccountID, CashAccountName: r.CashAccountName,
		Amount: r.Amount.String(), Ntpn: r.NTPN, Notes: r.Notes, Status: r.Status, JournalEntryId: r.JournalEntryID, VoidReason: r.VoidReason}
}

func (s *periodEndServer) ListTaxRemittances(ctx context.Context, _ *financev1.ListTaxRemittancesRequest) (*financev1.ListTaxRemittancesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.Ledgers.ListTaxRemittances(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListTaxRemittancesResponse{}
	for i := range rows {
		out.Remittances = append(out.Remittances, taxRemittancePB(&rows[i]))
	}
	return out, nil
}

func (s *periodEndServer) RemitTax(ctx context.Context, r *financev1.TaxRemittance) (*financev1.TaxRemittance, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", orZero(r.Amount))
	if err != nil {
		return nil, err
	}
	out, err := s.svc.Ledgers.RemitTax(ctx, p.UserID, domain.TaxRemittance{Period: r.Period, Date: r.RemitDate, AccountID: r.AccountId, CashAccountID: r.CashAccountId, Amount: amount, NTPN: r.Ntpn, Notes: r.Notes})
	if err != nil {
		return nil, err
	}
	return taxRemittancePB(out), nil
}

func (s *periodEndServer) VoidTaxRemittance(ctx context.Context, r *financev1.VoidAdvanceRequest) (*financev1.TaxRemittance, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	out, err := s.svc.Ledgers.VoidTaxRemittance(ctx, p.UserID, r.Id, r.Reason)
	if err != nil {
		return nil, err
	}
	return taxRemittancePB(out), nil
}

func (s *periodEndServer) CloseVat(ctx context.Context, r *financev1.CloseVatRequest) (*financev1.VatClosing, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Ledgers.CloseVAT(ctx, p.UserID, r.Period)
	if err != nil {
		return nil, err
	}
	return &financev1.VatClosing{Offset: c.Offset.String(), Payable: c.Payable.String(), Compensation: c.Compensation.String(), JournalEntryId: c.JournalEntryID}, nil
}

func (s *periodEndServer) GetControlSettings(ctx context.Context, _ *financev1.ControlSettingsRequest) (*financev1.ControlSettings, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	on, err := s.svc.Ledgers.SeparateApprover(ctx)
	if err != nil {
		return nil, err
	}
	return &financev1.ControlSettings{SeparateApprover: on}, nil
}

func (s *periodEndServer) SetControlSettings(ctx context.Context, r *financev1.ControlSettings) (*financev1.ControlSettings, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Ledgers.SetSeparateApprover(ctx, p.UserID, r.SeparateApprover); err != nil {
		return nil, err
	}
	return r, nil
}
