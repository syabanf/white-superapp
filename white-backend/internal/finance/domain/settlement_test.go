package domain

import "testing"

func TestSettlementLines(t *testing.T) {
	fee := PaymentAdjustment{Kind: AdjustFee, AccountID: "fee", Amount: Rupiah(6_500)}
	pph := PaymentAdjustment{Kind: AdjustWithholding, AccountID: "pph", Amount: Rupiah(200_000)}
	full := Rupiah(10_000_000)

	// A client's transfer arrives short by the fee and the tax it held back.
	splits, extra := SettlementLines(KindAR, full, []PaymentAdjustment{fee, pph})
	if len(extra) != 0 || splits[0].Amount != Rupiah(9_793_500) || splits[1].AccountID != "fee" || splits[2].AccountID != "pph" {
		t.Fatalf("receipt %+v %+v", splits, extra)
	}
	// Paying a vendor: tax held back lowers the transfer, the fee raises it.
	splits, extra = SettlementLines(KindAP, full, []PaymentAdjustment{fee, pph})
	if splits[0].Amount != Rupiah(9_806_500) || len(splits) != 2 || splits[1].AccountID != "pph" || len(extra) != 1 || extra[0].AccountID != "fee" || extra[0].Side != Debit {
		t.Fatalf("payment %+v %+v", splits, extra)
	}
}
