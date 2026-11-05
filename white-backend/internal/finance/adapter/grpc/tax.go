package grpcapi

import (
	"context"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type taxServer struct {
	financev1.UnimplementedTaxSetupServiceServer
	svc Services
}

func taxCodePB(t *domain.TaxCode) *financev1.TaxCode {
	steps := make([]*financev1.TaxBracketMsg, len(t.Brackets))
	for i, b := range t.Brackets {
		steps[i] = &financev1.TaxBracketMsg{UpTo: b.UpTo.String(), Rate: b.Rate.String()}
	}
	return &financev1.TaxCode{Brackets: steps, CashBasis: t.CashBasis, TransitionAccountId: t.TransitionAccountID, Id: t.ID, Code: t.Code, Name: t.Name, Kind: t.Kind, Scope: t.Scope, Rate: t.Rate.String(), BaseNumerator: int32(t.BaseNumerator), BaseDenominator: int32(t.BaseDenominator),
		PriceIncluded: t.PriceIncluded, AccountSaleId: t.AccountSaleID, AccountPurchaseId: t.AccountPurchaseID, ReportCode: t.ReportCode, ValidFrom: t.ValidFrom, ValidTo: t.ValidTo,
		IsDefaultSale: t.IsDefaultSale, IsDefaultPurchase: t.IsDefaultPurchase, IsActive: t.IsActive, Notes: t.Notes}
}

func chargeTypePB(c *domain.ChargeType) *financev1.ChargeType {
	return &financev1.ChargeType{Id: c.ID, Code: c.Code, Name: c.Name, Scope: c.Scope, Computation: c.Computation, Value: c.Value.String(), Taxable: c.Taxable, IsDeduction: c.IsDeduction,
		AccountSaleId: c.AccountSaleID, AccountPurchaseId: c.AccountPurchaseID, IsActive: c.IsActive, Notes: c.Notes}
}

// Sales picks tax codes on quotations, so it reads them without finance access.
func (s *taxServer) ListTaxCodes(ctx context.Context, r *financev1.ListTaxMasterRequest) (*financev1.ListTaxCodesResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	if !p.Can(authn.PermFinanceRead) && !p.Can(authn.PermSalesRead) {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	rows, err := s.svc.TaxSetup.ListTaxCodes(ctx, r.ActiveOnly)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListTaxCodesResponse{}
	for i := range rows {
		out.Taxes = append(out.Taxes, taxCodePB(&rows[i]))
	}
	return out, nil
}

func (s *taxServer) SaveTaxCode(ctx context.Context, r *financev1.TaxCode) (*financev1.TaxCode, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	rate, err := money("rate", orZero(r.Rate))
	if err != nil {
		return nil, err
	}
	steps := make([]domain.TaxBracket, len(r.Brackets))
	for i, b := range r.Brackets {
		if steps[i].UpTo, err = money("brackets", orZero(b.UpTo)); err != nil {
			return nil, err
		}
		if steps[i].Rate, err = money("brackets", orZero(b.Rate)); err != nil {
			return nil, err
		}
	}
	t, err := s.svc.TaxSetup.SaveTaxCode(ctx, p.UserID, domain.TaxCode{Brackets: steps, CashBasis: r.CashBasis, TransitionAccountID: r.TransitionAccountId, ID: r.Id, Code: r.Code, Name: r.Name, Kind: r.Kind, Scope: r.Scope, Rate: rate, BaseNumerator: int(r.BaseNumerator), BaseDenominator: int(r.BaseDenominator),
		PriceIncluded: r.PriceIncluded, AccountSaleID: r.AccountSaleId, AccountPurchaseID: r.AccountPurchaseId, ReportCode: r.ReportCode, ValidFrom: r.ValidFrom, ValidTo: r.ValidTo,
		IsDefaultSale: r.IsDefaultSale, IsDefaultPurchase: r.IsDefaultPurchase, IsActive: r.IsActive, Notes: r.Notes})
	if err != nil {
		return nil, err
	}
	return taxCodePB(t), nil
}

func (s *taxServer) ListChargeTypes(ctx context.Context, r *financev1.ListTaxMasterRequest) (*financev1.ListChargeTypesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.TaxSetup.ListChargeTypes(ctx, r.ActiveOnly)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListChargeTypesResponse{}
	for i := range rows {
		out.Charges = append(out.Charges, chargeTypePB(&rows[i]))
	}
	return out, nil
}

func (s *taxServer) SaveChargeType(ctx context.Context, r *financev1.ChargeType) (*financev1.ChargeType, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	value, err := money("value", orZero(r.Value))
	if err != nil {
		return nil, err
	}
	c, err := s.svc.TaxSetup.SaveChargeType(ctx, p.UserID, domain.ChargeType{ID: r.Id, Code: r.Code, Name: r.Name, Scope: r.Scope, Computation: r.Computation, Value: value, Taxable: r.Taxable, IsDeduction: r.IsDeduction,
		AccountSaleID: r.AccountSaleId, AccountPurchaseID: r.AccountPurchaseId, IsActive: r.IsActive, Notes: r.Notes})
	if err != nil {
		return nil, err
	}
	return chargeTypePB(c), nil
}

func (s *taxServer) GetTaxRegister(ctx context.Context, r *financev1.PeriodRequest) (*financev1.TaxRegister, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	reg, err := s.svc.TaxSetup.TaxRegister(ctx, r.DateFrom, r.DateTo)
	if err != nil {
		return nil, err
	}
	out := &financev1.TaxRegister{DateFrom: reg.From, DateTo: reg.To, OutputVat: reg.OutputVAT.String(), InputVat: reg.InputVAT.String(), VatPosition: reg.VATPosition.String(),
		WithheldByClients: reg.WithheldByClients.String(), WithheldFromVendors: reg.WithheldFromVendors.String()}
	for _, row := range reg.Rows {
		out.Rows = append(out.Rows, &financev1.TaxRegisterRow{Kind: row.Kind, TaxCode: row.TaxCode, TaxName: row.TaxName, TaxKind: row.TaxKind, InvoiceCount: int32(row.InvoiceCount), Base: row.Base.String(), Amount: row.Amount.String()})
	}
	return out, nil
}

func (s *taxServer) ExportTaxData(ctx context.Context, r *financev1.TaxExportRequest) (*financev1.TaxExportFile, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	name, data, err := s.svc.TaxSetup.Export(ctx, r.Kind, r.DateFrom, r.DateTo)
	if err != nil {
		return nil, err
	}
	return &financev1.TaxExportFile{FileName: name, ContentType: "text/csv; charset=utf-8", Data: data}, nil
}

func fiscalPB(f *domain.FiscalPosition) *financev1.FiscalPosition {
	out := &financev1.FiscalPosition{Id: f.ID, Name: f.Name, IsActive: f.IsActive, Notes: f.Notes}
	for _, r := range f.Rules {
		out.Rules = append(out.Rules, &financev1.FiscalRuleMsg{FromTaxId: r.FromTaxID, ToTaxId: r.ToTaxID})
	}
	return out
}

func (s *taxServer) ListFiscalPositions(ctx context.Context, _ *emptypb.Empty) (*financev1.ListFiscalPositionsResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.TaxSetup.ListFiscalPositions(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListFiscalPositionsResponse{}
	for i := range rows {
		out.Positions = append(out.Positions, fiscalPB(&rows[i]))
	}
	return out, nil
}

func (s *taxServer) SaveFiscalPosition(ctx context.Context, r *financev1.FiscalPosition) (*financev1.FiscalPosition, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	in := domain.FiscalPosition{ID: r.Id, Name: r.Name, IsActive: r.IsActive, Notes: r.Notes}
	for _, x := range r.Rules {
		in.Rules = append(in.Rules, domain.FiscalRule{FromTaxID: x.FromTaxId, ToTaxID: x.ToTaxId})
	}
	f, err := s.svc.TaxSetup.SaveFiscalPosition(ctx, p.UserID, in)
	if err != nil {
		return nil, err
	}
	return fiscalPB(f), nil
}

func profilePB(x domain.PartyProfile) *financev1.PartyTaxProfile {
	return &financev1.PartyTaxProfile{Id: x.ID, Side: x.Side, Name: x.Name, Npwp: x.NPWP, Address: x.Address, IsPkp: x.IsPKP, FiscalPositionId: x.FiscalPositionID, Notes: x.Notes}
}

func (s *taxServer) ListPartyProfiles(ctx context.Context, _ *emptypb.Empty) (*financev1.ListPartyProfilesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
	}
	rows, err := s.svc.TaxSetup.ListPartyProfiles(ctx)
	if err != nil {
		return nil, err
	}
	out := &financev1.ListPartyProfilesResponse{}
	for _, x := range rows {
		out.Profiles = append(out.Profiles, profilePB(x))
	}
	return out, nil
}

func (s *taxServer) SavePartyProfile(ctx context.Context, r *financev1.PartyTaxProfile) (*financev1.PartyTaxProfile, error) {
	p, err := write(ctx)
	if err != nil {
		return nil, err
	}
	x, err := s.svc.TaxSetup.SavePartyProfile(ctx, p.UserID, domain.PartyProfile{ID: r.Id, Side: r.Side, Name: r.Name, NPWP: r.Npwp, Address: r.Address, IsPKP: r.IsPkp, FiscalPositionID: r.FiscalPositionId, Notes: r.Notes})
	if err != nil {
		return nil, err
	}
	return profilePB(*x), nil
}

func (s *taxServer) CreateCreditNote(ctx context.Context, r *financev1.CreditNoteRequest) (*financev1.Invoice, error) {
	p, err := approve(ctx)
	if err != nil {
		return nil, err
	}
	amount, err := money("amount", orZero(r.Amount))
	if err != nil {
		return nil, err
	}
	inv, err := s.svc.Ledgers.CreateCreditNote(ctx, p.UserID, r.InvoiceId, r.NoteDate, amount, r.Reason)
	if err != nil {
		return nil, err
	}
	return invoicePB(inv, s.svc.Ledgers.PaymentStatus(*inv)), nil
}

func (s *taxServer) DeleteTaxCode(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.TaxSetup.DeleteTaxCode(ctx, r.Id)
}

func (s *taxServer) DeleteChargeType(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.TaxSetup.DeleteChargeType(ctx, r.Id)
}

func (s *taxServer) DeleteFiscalPosition(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.TaxSetup.DeleteFiscalPosition(ctx, r.Id)
}

func (s *taxServer) DeletePartyProfile(ctx context.Context, r *financev1.IdRequest) (*emptypb.Empty, error) {
	if _, err := approve(ctx); err != nil {
		return nil, err
	}
	return empty, s.svc.TaxSetup.DeletePartyProfile(ctx, r.Id)
}
