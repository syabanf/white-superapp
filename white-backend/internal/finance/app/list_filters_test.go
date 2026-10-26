package app_test

import (
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/app"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// Dashboard drills land on the invoice and claim lists with a date range,
// a party or a department.
func TestListFiltersForDrillDown(t *testing.T) {
	h := newHarness(t)
	l := app.NewLedgers(h.deps)
	for _, in := range []app.InvoiceInput{
		{InvoiceDate: "2026-09-15", PartyName: "PT Telkom"},
		{InvoiceDate: "2026-10-02", PartyName: "PT Telkom"},
		{InvoiceDate: "2026-10-05", PartyName: "PT Telkom Akses"},
	} {
		in.DueDate, in.AccountID, in.Subtotal = "2026-10-30", h.id("6301002"), rp(100_000)
		_, err := l.CreateInvoice(h.ctx, domain.KindAP, "u1", in)
		h.ok(err)
	}
	count := func(f domain.InvoiceFilter) int {
		t.Helper()
		f.Kind, f.Limit = domain.KindAP, 50
		_, total, err := l.ListInvoices(h.ctx, f, "")
		h.ok(err)
		return total
	}
	if n := count(domain.InvoiceFilter{Search: "Telkom"}); n != 3 {
		t.Fatalf("search keeps matching partial names: %d", n)
	}
	if n := count(domain.InvoiceFilter{PartyName: "PT Telkom"}); n != 2 {
		t.Fatalf("party is exact: %d", n)
	}
	if n := count(domain.InvoiceFilter{From: "2026-10-01", To: "2026-10-31"}); n != 2 {
		t.Fatalf("date range: %d", n)
	}
	if n := count(domain.InvoiceFilter{From: "2026-10-01", PartyName: "PT Telkom"}); n != 1 {
		t.Fatalf("range and party: %d", n)
	}
	_, _, err := l.ListInvoices(h.ctx, domain.InvoiceFilter{Kind: domain.KindAP, From: "2026-13-01"}, "")
	h.wantErr(err, "Tanggal tidak valid (format YYYY-MM-DD).")

	h.staff()
	r := app.NewReimbursements(h.deps)
	h.draft(r, principal("u-sekar", authn.PermSelfService), domain.CategoryTransport, 50_000, "2026-10-20")
	h.draft(r, principal("u-sekar", authn.PermSelfService), domain.CategoryTransport, 60_000, "2026-09-20")
	h.draft(r, principal("u-taufik", authn.PermSelfService), domain.CategoryTransport, 70_000, "2026-10-21")
	claims := func(f domain.ClaimFilter, month string) int {
		t.Helper()
		_, total, _, err := r.List(h.ctx, f, month)
		h.ok(err)
		return total
	}
	if n := claims(domain.ClaimFilter{Department: "Creative"}, ""); n != 2 {
		t.Fatalf("department: %d", n)
	}
	if n := claims(domain.ClaimFilter{Department: "Creative", MonthFrom: "2026-10-01", MonthTo: "2026-10-31"}, ""); n != 1 {
		t.Fatalf("department and range: %d", n)
	}
	if n := claims(domain.ClaimFilter{MonthFrom: "2026-01-01", MonthTo: "2026-01-31"}, "2026-10"); n != 2 {
		t.Fatalf("month wins over the range: %d", n)
	}
}
