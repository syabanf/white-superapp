package grpcapi

import (
	"context"

	financev1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/finance/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

type taxServer struct {
	financev1.UnimplementedTaxSetupServiceServer
	svc Services
}

func taxCodePB(t *domain.TaxCode) *financev1.TaxCode {
	return &financev1.TaxCode{Id: t.ID, Code: t.Code, Name: t.Name, Kind: t.Kind, Scope: t.Scope, Rate: t.Rate.String(), BaseNumerator: int32(t.BaseNumerator), BaseDenominator: int32(t.BaseDenominator),
		PriceIncluded: t.PriceIncluded, AccountSaleId: t.AccountSaleID, AccountPurchaseId: t.AccountPurchaseID, ReportCode: t.ReportCode, ValidFrom: t.ValidFrom, ValidTo: t.ValidTo,
		IsDefaultSale: t.IsDefaultSale, IsDefaultPurchase: t.IsDefaultPurchase, IsActive: t.IsActive, Notes: t.Notes}
}

func chargeTypePB(c *domain.ChargeType) *financev1.ChargeType {
	return &financev1.ChargeType{Id: c.ID, Code: c.Code, Name: c.Name, Scope: c.Scope, Computation: c.Computation, Value: c.Value.String(), Taxable: c.Taxable, IsDeduction: c.IsDeduction,
		AccountSaleId: c.AccountSaleID, AccountPurchaseId: c.AccountPurchaseID, IsActive: c.IsActive, Notes: c.Notes}
}

func (s *taxServer) ListTaxCodes(ctx context.Context, r *financev1.ListTaxMasterRequest) (*financev1.ListTaxCodesResponse, error) {
	if _, err := read(ctx); err != nil {
		return nil, err
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
	t, err := s.svc.TaxSetup.SaveTaxCode(ctx, p.UserID, domain.TaxCode{ID: r.Id, Code: r.Code, Name: r.Name, Kind: r.Kind, Scope: r.Scope, Rate: rate, BaseNumerator: int(r.BaseNumerator), BaseDenominator: int(r.BaseDenominator),
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
