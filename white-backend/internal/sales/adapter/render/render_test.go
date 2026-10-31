package render

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"image"
	"image/png"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

func quote() domain.Quotation {
	vu := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	notes := "Harga belum termasuk ad spend."
	return domain.Quotation{QuoteNumber: "QT-2610-0001", Version: 1, Status: "draft", ClientName: "PT Maju Jaya", UsePpn: true, PpnPercent: 11,
		Subtotal: domain.MustMoney("10000000"), DiscountPercent: 5, DiscountNominal: domain.MustMoney("500000"), PpnNominal: domain.MustMoney("1045000"), Total: domain.MustMoney("10545000"),
		ValidUntil: &vu, Notes: &notes, Audit: domain.Audit{CreatedAt: time.Now()},
		Items: []domain.QuotationItem{{Description: "SEO Bulanan (Standard)", Scope: strings.Repeat("Riset kata kunci dan artikel. ", 12), Qty: 6, Unit: "bulan", UnitPrice: domain.MustMoney("1666666.67"), LineTotal: domain.MustMoney("10000000")}},
		Terms: []domain.QuotationTerm{{Label: "DP", Percent: 50}, {Label: "Pelunasan", Percent: 50}}}
}

func invoice() domain.Invoice {
	due := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	return domain.Invoice{InvoiceNumber: "INV-2610-0001", Label: "DP", Amount: domain.MustMoney("5272500"), PaidTotal: domain.MustMoney("1000000"), DueDate: &due,
		ClientName: "PT Maju", DealTitle: "SEO PT Maju", Status: "terkirim", Audit: domain.Audit{CreatedAt: time.Now()}}
}

func TestQuotationAndInvoicePDF(t *testing.T) {
	deal := domain.Deal{Title: "SEO PT Maju", PicName: "Andi", PicPhone: "0812 3456 7890", PicEmail: "andi@majujaya.co.id"}
	cases := map[string]Company{
		"defaults": CompanyFromEnv(),
		"npwp":     {LegalName: legalName, Address: brandAddress, Email: brandEmail, Phone: brandPhone, NPWP: "01.234.567.8-901.000"},
		"no npwp, long address": {LegalName: legalName, Email: brandEmail,
			Address: "Gedung Mixist Tower Lantai 12, Jl. Jenderal Sudirman Kav. 52-53, Senayan, Kebayoran Baru, Jakarta Selatan, DKI Jakarta 12190"},
		"bare": {LegalName: legalName},
	}
	for name, co := range cases {
		t.Run(name, func(t *testing.T) {
			r := Renderer{Company: co}
			pdf, err := r.QuotationPDF(quote(), deal, "Sari Sales")
			if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
				t.Fatalf("quotation pdf: %v", err)
			}
			pdf, err = r.InvoicePDF(invoice(), deal, "QT-2610-0001")
			if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
				t.Fatalf("invoice pdf: %v", err)
			}
		})
	}
	// Unknown owner and no source quotation leave their rows out.
	if _, err := New().QuotationPDF(quote(), domain.Deal{}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := New().InvoicePDF(domain.Invoice{InvoiceNumber: "INV-2610-0002", Status: "diajukan"}, domain.Deal{}, ""); err != nil {
		t.Fatal(err)
	}
}

func TestCompanyFromEnv(t *testing.T) {
	t.Setenv("COMPANY_LEGAL_NAME", "PT Lain")
	t.Setenv("COMPANY_NPWP", "")
	co := CompanyFromEnv()
	if co.LegalName != "PT Lain" || co.Address != brandAddress || co.NPWP != "" {
		t.Fatalf("company %+v", co)
	}
}

func signature(t *testing.T) []byte {
	var buf bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 60, 20))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestContractPDF(t *testing.T) {
	q := quote()
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 6, -1)
	vals := domain.ContractValues(domain.ContractContext{AgencyName: legalName, ContractNumber: "KTR-2610-0001", Today: start, ClientName: "PT Maju", Quotation: &q, StartDate: &start, EndDate: &end})
	body := domain.RenderContract("# PERJANJIAN\n\nAntara **{{agency_name}}** dan {{client_name}}.\n\n## Pasal 1\n\n{{services_table}}\n\n{{payment_terms}}\n\n1. satu\n2. dua\n\n---\n", vals)
	signed := time.Now()
	name, ip := "Andi Wijaya", "203.0.113.5"
	c := domain.Contract{ID: "c1", ContractNumber: "KTR-2610-0001", Title: "Perjanjian Retainer", ClientName: "PT Maju", Body: body, SignedAt: &signed, SignerName: &name, SignerIP: &ip, SignaturePNG: signature(t)}
	pdf, err := New().ContractPDF(c)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("contract pdf: %v", err)
	}
	c.SignaturePNG, c.SignedAt = nil, nil
	if _, err := New().ContractPDF(c); err != nil {
		t.Fatal(err)
	}
}

func TestPitchDeckIsValidOOXML(t *testing.T) {
	slides := domain.BuildSlides(domain.DeckInput{ClientName: "PT <Maju> & Co", ServiceLines: []string{domain.LineSEO}, Options: domain.DeckOptions{IncludePricing: true}}, domain.MockDeckCopy(domain.DeckInput{ClientName: "PT <Maju> & Co", ServiceLines: []string{domain.LineSEO}}))
	data, err := New().PitchDeck("Proposal PT <Maju> & Co", slides)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
		rc, _ := f.Open()
		raw, _ := io.ReadAll(rc)
		rc.Close()
		dec := xml.NewDecoder(bytes.NewReader(raw))
		for {
			if _, err := dec.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s: %v", f.Name, err)
			}
		}
	}
	for _, want := range []string{"[Content_Types].xml", "ppt/presentation.xml", "ppt/slides/slide1.xml", "ppt/theme/theme1.xml"} {
		if !names[want] {
			t.Fatalf("missing %s", want)
		}
	}
	if !names["ppt/slides/slide"+itoa(len(slides))+".xml"] {
		t.Fatal("one part per slide")
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
