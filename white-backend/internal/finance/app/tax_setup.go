package app

import (
	"context"
	"errors"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// TaxSetup is the master data for taxes and extra charges, and the tax register
// built from posted invoices. Codes are archived with is_active, never deleted,
// because documents refer to them.
type TaxSetup struct{ Deps }

func NewTaxSetup(d Deps) *TaxSetup { return &TaxSetup{d} }

func (s *TaxSetup) ListTaxCodes(ctx context.Context, activeOnly bool) ([]domain.TaxCode, error) {
	return s.Store.ListTaxCodes(ctx, activeOnly)
}

func (s *TaxSetup) ListChargeTypes(ctx context.Context, activeOnly bool) ([]domain.ChargeType, error) {
	return s.Store.ListChargeTypes(ctx, activeOnly)
}

func validScope(scope string) bool {
	return scope == domain.ScopeSale || scope == domain.ScopePurchase || scope == domain.ScopeBoth
}

// checkPostable validates the optional accounts of a master record.
func (s *TaxSetup) checkPostable(ctx context.Context, accountIDs ...string) error {
	set := []string{}
	for _, id := range accountIDs {
		if id != "" {
			set = append(set, id)
		}
	}
	if len(set) == 0 {
		return nil
	}
	return checkAccounts(ctx, s.Store, set, "", "Akun harus postable (bukan header)")
}

func (s *TaxSetup) SaveTaxCode(ctx context.Context, userID string, t domain.TaxCode) (*domain.TaxCode, error) {
	t.Code, t.Name, t.ReportCode, t.Notes = trim(t.Code), trim(t.Name), trim(t.ReportCode), trim(t.Notes)
	v := validate.New().Required("code", t.Code, "Kode wajib diisi").Required("name", t.Name, "Nama wajib diisi")
	if t.Kind != domain.TaxVAT && t.Kind != domain.TaxWithholding {
		v.Add("kind", "Jenis pajak tidak valid")
	}
	if !validScope(t.Scope) {
		v.Add("scope", "Berlaku untuk tidak valid")
	}
	if t.Rate < 0 || t.Rate > domain.Rupiah(100) {
		v.Add("rate", "Tarif 0 sampai 100 persen")
	}
	if t.BaseNumerator <= 0 || t.BaseDenominator <= 0 || t.BaseNumerator > t.BaseDenominator {
		v.Add("base_numerator", "Porsi DPP harus di antara 0 dan 1, misalnya 11/12")
	}
	if t.PriceIncluded && t.Kind != domain.TaxVAT {
		v.Add("price_included", "Hanya PPN yang bisa termasuk dalam harga")
	}
	if t.IsDefaultSale && t.Scope == domain.ScopePurchase || t.IsDefaultPurchase && t.Scope == domain.ScopeSale {
		v.Add("scope", "Bawaan harus sesuai cakupan pajak")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	var err error
	if t.ValidFrom, err = optionalDay("valid_from", t.ValidFrom); err != nil {
		return nil, err
	}
	if t.ValidTo, err = optionalDay("valid_to", t.ValidTo); err != nil {
		return nil, err
	}
	if t.ValidFrom != "" && t.ValidTo != "" && t.ValidTo < t.ValidFrom {
		return nil, apperr.Field("valid_to", "Tanggal akhir harus setelah tanggal mulai")
	}
	if t.IsActive {
		missingSale := t.Scope != domain.ScopePurchase && t.AccountSaleID == ""
		missingPurchase := t.Scope != domain.ScopeSale && t.AccountPurchaseID == ""
		if missingSale || missingPurchase {
			return nil, apperr.Field("account_sale_id", "Kode pajak aktif perlu akun posting")
		}
	}
	if err := s.checkPostable(ctx, t.AccountSaleID, t.AccountPurchaseID); err != nil {
		return nil, err
	}
	if t.ID == "" {
		t.ID = ids.New()
	}
	t.UpdatedBy = userID
	if err := s.Store.SaveTaxCode(ctx, t); err != nil {
		var unique *domain.ErrUnique
		if errors.As(err, &unique) {
			return nil, apperr.Field("code", "Kode pajak sudah dipakai")
		}
		return nil, err
	}
	return s.Store.GetTaxCode(ctx, t.ID)
}

func (s *TaxSetup) SaveChargeType(ctx context.Context, userID string, c domain.ChargeType) (*domain.ChargeType, error) {
	c.Code, c.Name, c.Notes = trim(c.Code), trim(c.Name), trim(c.Notes)
	v := validate.New().Required("code", c.Code, "Kode wajib diisi").Required("name", c.Name, "Nama wajib diisi")
	if !validScope(c.Scope) {
		v.Add("scope", "Berlaku untuk tidak valid")
	}
	if c.Computation != domain.ChargeFixed && c.Computation != domain.ChargePercent {
		v.Add("computation", "Cara hitung tidak valid")
	}
	if c.Value < 0 || (c.Computation == domain.ChargePercent && c.Value > domain.Rupiah(100)) {
		v.Add("value", "Nilai tidak valid")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	if err := s.checkPostable(ctx, c.AccountSaleID, c.AccountPurchaseID); err != nil {
		return nil, err
	}
	if c.ID == "" {
		c.ID = ids.New()
	}
	c.UpdatedBy = userID
	if err := s.Store.SaveChargeType(ctx, c); err != nil {
		var unique *domain.ErrUnique
		if errors.As(err, &unique) {
			return nil, apperr.Field("code", "Kode biaya sudah dipakai")
		}
		return nil, err
	}
	return s.Store.GetChargeType(ctx, c.ID)
}

// TaxRegister is the period's taxes from posted invoices. The VAT position is
// output minus input, what a monthly return settles.
type TaxRegister struct {
	From, To                               string
	Rows                                   []domain.TaxRegisterRow
	OutputVAT, InputVAT, VATPosition       domain.Money
	WithheldByClients, WithheldFromVendors domain.Money
}

func (s *TaxSetup) TaxRegister(ctx context.Context, from, to string) (*TaxRegister, error) {
	from, to, err := NewReports(s.Deps).period(from, to)
	if err != nil {
		return nil, err
	}
	rows, err := s.Store.TaxRegister(ctx, from, to)
	if err != nil {
		return nil, err
	}
	out := &TaxRegister{From: from, To: to, Rows: rows}
	for _, r := range rows {
		switch {
		case r.TaxKind == domain.TaxVAT && r.Kind == domain.KindAR:
			out.OutputVAT += r.Amount
		case r.TaxKind == domain.TaxVAT:
			out.InputVAT += r.Amount
		case r.Kind == domain.KindAR:
			out.WithheldByClients += r.Amount
		default:
			out.WithheldFromVendors += r.Amount
		}
	}
	out.VATPosition = out.OutputVAT - out.InputVAT
	return out, nil
}
