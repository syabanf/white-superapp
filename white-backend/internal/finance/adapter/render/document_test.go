package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
)

func TestDocumentPDF(t *testing.T) {
	items := make([]domain.DocItem, 45) // Forces a page break.
	for i := range items {
		items[i] = domain.DocItem{Description: strings.Repeat("Produksi konten bulanan ", 4), Unit: "paket", Qty: 150, UnitPrice: domain.Rupiah(1_000_000)}
	}
	for _, kind := range []string{domain.DocPR, domain.DocPO, domain.DocQuotation, domain.DocInvoice} {
		d := domain.Document{Kind: kind, Number: "X-1", Title: "Uji — dokumen", PartyName: "PT Tepat", IssueDate: "2026-11-05", DueDate: "2026-12-05", Items: items,
			DiscountPercent: domain.Rupiah(10), Notes: "Catatan", Taxes: []domain.DocTax{{Name: "PPN", Kind: domain.TaxVAT, Rate: domain.Rupiah(12), BaseNumerator: 11, BaseDenominator: 12}, {Name: "PPh 23", Kind: domain.TaxWithholding, Rate: domain.Rupiah(2)}}, Charges: []domain.DocCharge{{Name: "Bea materai", Computation: domain.ChargeFixed, Value: domain.Rupiah(10_000)}}}
		d.Recalculate()
		data, err := DocumentPDF(d, domain.Company{LegalName: "PT Mixist", Address: "Jakarta", NPWP: "01.234"})
		if err != nil || !bytes.HasPrefix(data, []byte("%PDF")) {
			t.Fatalf("%s: %v", kind, err)
		}
	}
}

func TestFormats(t *testing.T) {
	if got := rupiah(domain.Rupiah(1_250_000)); got != "Rp 1.250.000" {
		t.Fatal(got)
	}
	if got := rupiah(0); got != "Rp 0" {
		t.Fatal(got)
	}
	if got := qty(150); got != "1.5" {
		t.Fatal(got)
	}
	if got := qty(domain.Rupiah(2)); got != "2" {
		t.Fatal(got)
	}
	if got := percent(domain.Rupiah(11)); got != "11%" {
		t.Fatal(got)
	}
}
