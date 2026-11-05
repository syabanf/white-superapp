package domain

import "testing"

func tax(kind string, rate int64, num, den int, included bool) DocTax {
	return DocTax{Kind: kind, Rate: Rupiah(rate), BaseNumerator: num, BaseDenominator: den, PriceIncluded: included}
}

func TestDocumentTotals(t *testing.T) {
	d := Document{Items: []DocItem{{Qty: Rupiah(2), UnitPrice: Rupiah(1_000_000)}, {Qty: 150, UnitPrice: Rupiah(200_000)}},
		DiscountPercent: Rupiah(10), Taxes: []DocTax{tax(TaxVAT, 11, 1, 1, false)}}
	d.Recalculate()
	if d.Items[1].LineTotal != Rupiah(300_000) || d.Subtotal != Rupiah(2_300_000) {
		t.Fatalf("lines %+v subtotal %d", d.Items, d.Subtotal)
	}
	if d.Discount != Rupiah(230_000) || d.Tax != Rupiah(227_700) || d.Total != Rupiah(2_297_700) {
		t.Fatalf("totals %d %d %d", d.Discount, d.Tax, d.Total)
	}
}

func TestDPPNilaiLainIsEffectivelyEleven(t *testing.T) {
	d := Document{Items: []DocItem{{Qty: Rupiah(1), UnitPrice: Rupiah(10_000_000)}}, Taxes: []DocTax{tax(TaxVAT, 12, 11, 12, false)}}
	d.Recalculate()
	if d.Tax != Rupiah(1_100_000) || d.Total != Rupiah(11_100_000) {
		t.Fatalf("tax %d total %d", d.Tax, d.Total)
	}
	if d.Taxes[0].Base != mulDiv(int64(Rupiah(10_000_000)), 11, 12) {
		t.Fatalf("base %d", d.Taxes[0].Base)
	}
}

func TestWithholdingUsesBaseWithoutVAT(t *testing.T) {
	d := Document{Items: []DocItem{{Qty: Rupiah(1), UnitPrice: Rupiah(10_000_000)}}, Taxes: []DocTax{tax(TaxVAT, 11, 1, 1, false), tax(TaxWithholding, 2, 1, 1, false)}}
	d.Recalculate()
	if d.Withholding != Rupiah(200_000) || d.Total != Rupiah(11_100_000) || d.NetDue() != Rupiah(10_900_000) {
		t.Fatalf("wh %d total %d net %d", d.Withholding, d.Total, d.NetDue())
	}
}

func TestPriceIncludedVAT(t *testing.T) {
	d := Document{Items: []DocItem{{Qty: Rupiah(1), UnitPrice: Rupiah(1_110_000)}}, Taxes: []DocTax{tax(TaxVAT, 11, 1, 1, true)}}
	d.Recalculate()
	if d.Subtotal != Rupiah(1_000_000) || d.Tax != Rupiah(110_000) || d.Total != Rupiah(1_110_000) {
		t.Fatalf("subtotal %d tax %d total %d", d.Subtotal, d.Tax, d.Total)
	}
	// A price that does not divide evenly still totals the price typed.
	odd := Document{Items: []DocItem{{Qty: Rupiah(1), UnitPrice: Rupiah(100_001)}}, Taxes: []DocTax{tax(TaxVAT, 12, 11, 12, true)}}
	odd.Recalculate()
	if odd.Total != Rupiah(100_001) {
		t.Fatalf("odd total %d", odd.Total)
	}
}

func TestChargesAroundTheTaxBase(t *testing.T) {
	d := Document{Items: []DocItem{{Qty: Rupiah(1), UnitPrice: Rupiah(10_000_000)}}, Taxes: []DocTax{tax(TaxVAT, 11, 1, 1, false)}, Charges: []DocCharge{
		{Computation: ChargePercent, Value: Rupiah(10), Taxable: true},                       // Agency fee 1.000.000.
		{Computation: ChargeFixed, Value: Rupiah(10_000)},                                    // Materai, after tax.
		{Computation: ChargeFixed, Value: Rupiah(500_000), Taxable: true, IsDeduction: true}, // Potongan.
	}}
	d.Recalculate()
	wantBase := Rupiah(10_500_000)
	if d.Taxes[0].Base != wantBase || d.Tax != Rupiah(1_155_000) {
		t.Fatalf("base %d tax %d", d.Taxes[0].Base, d.Tax)
	}
	if d.Total != wantBase+Rupiah(1_155_000)+Rupiah(10_000) || d.ChargesAmount != Rupiah(510_000) {
		t.Fatalf("total %d charges %d", d.Total, d.ChargesAmount)
	}
	if d.Charges[2].Amount != Rupiah(-500_000) {
		t.Fatalf("deduction %d", d.Charges[2].Amount)
	}
}

func TestDocTransition(t *testing.T) {
	for _, c := range []struct {
		kind, status, action, want string
		ok                         bool
	}{
		{DocPO, DocDraft, ActSubmit, DocSubmitted, true},
		{DocPO, DocDraft, ActApprove, "", false},
		{DocPO, DocSubmitted, ActApprove, DocApproved, true},
		{DocPR, DocRejected, ActReopen, DocDraft, true},
		{DocQuotation, DocDraft, ActSubmit, "", false},
		{DocQuotation, DocSent, ActAccept, DocAccepted, true},
		{DocInvoice, DocDraft, ActApprove, "", false},
	} {
		got, ok := DocTransition(c.kind, c.status, c.action)
		if got != c.want || ok != c.ok {
			t.Errorf("%+v = %q %v", c, got, ok)
		}
	}
}

func TestProgressiveTax(t *testing.T) {
	steps := []TaxBracket{{UpTo: Rupiah(60_000_000), Rate: Rupiah(5)}, {UpTo: Rupiah(250_000_000), Rate: Rupiah(15)}, {Rate: Rupiah(25)}}
	for base, want := range map[int64]int64{0: 0, 10_000_000: 500_000, 60_000_000: 3_000_000, 100_000_000: 9_000_000, 300_000_000: 44_000_000} {
		if got := ProgressiveTax(Rupiah(base), steps); got != Rupiah(want) {
			t.Errorf("base %d = %d", base, got)
		}
	}
	if !ValidBrackets(steps) {
		t.Fatal("rising steps with an open top are valid")
	}
	if ValidBrackets([]TaxBracket{{UpTo: Rupiah(10), Rate: 1}, {UpTo: Rupiah(5), Rate: 1}, {Rate: 1}}) || ValidBrackets([]TaxBracket{{UpTo: Rupiah(10), Rate: 1}}) {
		t.Fatal("falling steps or a capped top are invalid")
	}
	d := Document{Items: []DocItem{{Qty: Rupiah(1), UnitPrice: Rupiah(100_000_000)}}, Taxes: []DocTax{{Kind: TaxWithholding, Rate: Rupiah(25), BaseNumerator: 1, BaseDenominator: 1, Brackets: steps}}}
	d.Recalculate()
	if d.Withholding != Rupiah(9_000_000) {
		t.Fatalf("withholding %d", d.Withholding)
	}
}
