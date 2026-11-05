package domain

import "testing"

func TestDocumentTotals(t *testing.T) {
	d := Document{Items: []DocItem{{Qty: Rupiah(2), UnitPrice: Rupiah(1_000_000)}, {Qty: 150, UnitPrice: Rupiah(200_000)}},
		DiscountPercent: Rupiah(10), PPNPercent: Rupiah(11)}
	d.Recalculate()
	if d.Items[1].LineTotal != Rupiah(300_000) || d.Subtotal != Rupiah(2_300_000) {
		t.Fatalf("lines %+v subtotal %d", d.Items, d.Subtotal)
	}
	if d.Discount != Rupiah(230_000) || d.Tax != Rupiah(227_700) || d.Total != Rupiah(2_297_700) {
		t.Fatalf("totals %d %d %d", d.Discount, d.Tax, d.Total)
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
