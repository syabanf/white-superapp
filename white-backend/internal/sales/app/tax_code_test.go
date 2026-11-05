package app

import (
	"context"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

type fakeTaxes struct{ code, def *TaxInfo }

func (f fakeTaxes) VATByCode(_ context.Context, code string) (*TaxInfo, error) {
	if f.code != nil && f.code.Code == code {
		return f.code, nil
	}
	return nil, nil
}
func (f fakeTaxes) DefaultSaleVAT(context.Context) (*TaxInfo, error) { return f.def, nil }

func TestApplyTaxCode(t *testing.T) {
	ppn := &TaxInfo{Code: "PPN-11", Name: "PPN DPP 11/12", EffectivePercent: 11}
	q := &Quotes{Deps: &Deps{Taxes: fakeTaxes{code: ppn, def: ppn}}}

	p := domain.QuotationPayload{UsePpn: true}
	if err := q.applyTaxCode(context.Background(), "PPN-11", &p); err != nil || p.TaxCode != "PPN-11" || p.PpnPercent != 11 {
		t.Fatalf("chosen %+v %v", p, err)
	}
	if err := q.applyTaxCode(context.Background(), "HILANG", &domain.QuotationPayload{UsePpn: true}); err == nil {
		t.Fatal("unknown code accepted")
	}
	def := domain.QuotationPayload{UsePpn: true}
	if err := q.applyTaxCode(context.Background(), "", &def); err != nil || def.TaxCode != "PPN-11" {
		t.Fatalf("default %+v %v", def, err)
	}
	typed := domain.QuotationPayload{UsePpn: true, PpnPercent: 5}
	if err := q.applyTaxCode(context.Background(), "", &typed); err != nil || typed.TaxCode != "" || typed.PpnPercent != 5 {
		t.Fatalf("typed %+v %v", typed, err)
	}
	off := domain.QuotationPayload{}
	if err := q.applyTaxCode(context.Background(), "PPN-11", &off); err != nil || off.TaxCode != "" {
		t.Fatalf("off %+v %v", off, err)
	}
	none := &Quotes{Deps: &Deps{}}
	if err := none.applyTaxCode(context.Background(), "PPN-11", &domain.QuotationPayload{UsePpn: true}); err != nil {
		t.Fatal(err)
	}
}
