// Package render prints finance documents on the company letterhead with
// go-pdf/fpdf and built-in Helvetica. The identity comes from the same COMPANY_*
// settings as the sales documents.
package render

import (
	"bytes"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/syabanf/white-superapp/white-backend/internal/finance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
)

// CompanyFromEnv reads the letterhead settings.
func CompanyFromEnv() domain.Company {
	return domain.Company{
		LegalName: config.Env("COMPANY_LEGAL_NAME", "PT Mixist Digital Kreatif"),
		Address:   config.Env("COMPANY_ADDRESS", "Jl. Senopati No. 8, Kebayoran Baru, Jakarta Selatan 12190"),
		Email:     config.Env("COMPANY_EMAIL", "hello@mixist.id"),
		Phone:     config.Env("COMPANY_PHONE", "+62 21 5000 8800"),
		NPWP:      config.Env("COMPANY_NPWP", ""),
	}
}

var titles = map[string]string{domain.DocPR: "PURCHASE REQUEST", domain.DocPO: "PURCHASE ORDER", domain.DocQuotation: "QUOTATION", domain.DocInvoice: "INVOICE"}

func idDate(day string) string {
	t, err := time.Parse(time.DateOnly, day)
	if err != nil {
		return "-"
	}
	months := []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return t.Format("2") + " " + months[t.Month()-1] + " " + t.Format("2006")
}

// rupiah formats whole rupiah with dots, rounding the cents.
func rupiah(m domain.Money) string {
	n := (int64(m) + 50) / 100
	if m < 0 {
		n = (int64(m) - 50) / 100
	}
	sign := ""
	if n < 0 {
		sign, n = "-", -n
	}
	digits := []byte{}
	for i := 0; n > 0 || i == 0; i++ {
		if i > 0 && i%3 == 0 {
			digits = append(digits, '.')
		}
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	for l, r := 0, len(digits)-1; l < r; l, r = l+1, r-1 {
		digits[l], digits[r] = digits[r], digits[l]
	}
	return sign + "Rp " + string(digits)
}

func qty(m domain.Money) string {
	return strings.TrimSuffix(strings.TrimRight(m.String(), "0"), ".")
}

func percent(m domain.Money) string { return qty(m) + "%" }

// DocumentPDF prints one document.
func DocumentPDF(d domain.Document, co domain.Company) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	t := func(s string) string { return tr(strings.NewReplacer("–", "-", "—", "-", "’", "'").Replace(s)) }
	pdf.SetMargins(18, 16, 18)
	pdf.SetAutoPageBreak(true, 18)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-13)
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(120, 120, 120)
		pdf.CellFormat(0, 5, t(d.Number), "", 0, "R", false, 0, "")
	})
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 6, t(co.LegalName), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(90, 90, 90)
	pdf.MultiCell(0, 4.5, t(co.Address), "", "L", false)
	contact := []string{}
	for _, v := range []string{co.Phone, co.Email} {
		if v != "" {
			contact = append(contact, v)
		}
	}
	if co.NPWP != "" {
		contact = append(contact, "NPWP "+co.NPWP)
	}
	pdf.CellFormat(0, 4.5, t(strings.Join(contact, " | ")), "", 1, "L", false, 0, "")
	pdf.SetDrawColor(40, 40, 40)
	pdf.Ln(2)
	pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
	pdf.Ln(5)

	pdf.SetTextColor(20, 20, 20)
	pdf.SetFont("Helvetica", "B", 15)
	pdf.CellFormat(100, 7, titles[d.Kind], "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 7, t(d.Number), "", 1, "R", false, 0, "")
	pdf.Ln(2)

	info := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(110, 110, 110)
		pdf.CellFormat(34, 5, t(label), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(20, 20, 20)
		pdf.MultiCell(0, 5, t(value), "", "L", false)
	}
	info("Perihal", d.Title)
	info("Kepada", d.PartyName)
	info("PIC", d.PicName)
	info("Kontak", strings.Trim(d.Phone+" "+d.Email, " "))
	info("Alamat", d.Address)
	info("Tanggal", idDate(d.IssueDate))
	if d.DueDate != "" {
		label := "Jatuh tempo"
		switch d.Kind {
		case domain.DocQuotation:
			label = "Berlaku sampai"
		case domain.DocPR:
			label = "Dibutuhkan"
		}
		info(label, idDate(d.DueDate))
	}
	info("No. referensi", d.ExternalNumber)
	pdf.Ln(4)

	widths := []float64{84, 16, 18, 28, 28}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(240, 240, 240)
	for i, h := range []string{"Keterangan", "Jumlah", "Satuan", "Harga", "Total"} {
		align := "L"
		if i == 1 || i >= 3 {
			align = "R"
		}
		pdf.CellFormat(widths[i], 7, h, "B", 0, align, true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Helvetica", "", 9)
	for _, it := range d.Items {
		lines := pdf.SplitText(t(it.Description), widths[0]-2)
		height := max(5.5, 4.5*float64(len(lines))+1)
		if pdf.GetY()+height > 270 {
			pdf.AddPage()
		}
		x, y := pdf.GetXY()
		pdf.MultiCell(widths[0], 4.5, strings.Join(lines, "\n"), "", "L", false)
		pdf.SetXY(x+widths[0], y)
		pdf.CellFormat(widths[1], height, qty(it.Qty), "", 0, "R", false, 0, "")
		pdf.CellFormat(widths[2], height, t(it.Unit), "", 0, "L", false, 0, "")
		pdf.CellFormat(widths[3], height, rupiah(it.UnitPrice), "", 0, "R", false, 0, "")
		pdf.CellFormat(widths[4], height, rupiah(it.LineTotal), "", 1, "R", false, 0, "")
		pdf.SetDrawColor(225, 225, 225)
		pdf.Line(18, pdf.GetY(), 192, pdf.GetY())
	}
	pdf.Ln(3)

	total := func(label, value string, bold bool) {
		style := ""
		if bold {
			style = "B"
		}
		pdf.SetFont("Helvetica", style, 10)
		pdf.CellFormat(84, 6, "", "", 0, "L", false, 0, "")
		pdf.CellFormat(62, 6, t(label), "", 0, "L", false, 0, "")
		pdf.CellFormat(28, 6, value, "", 1, "R", false, 0, "")
	}
	total("Subtotal", rupiah(d.Subtotal), false)
	if d.Discount > 0 {
		total("Diskon "+percent(d.DiscountPercent), "-"+rupiah(d.Discount), false)
	}
	for _, c := range d.Charges {
		total(c.Name, rupiah(c.Amount), false)
	}
	for _, t := range d.Taxes {
		if t.Kind == domain.TaxVAT {
			total(t.Name, rupiah(t.Amount), false)
		}
	}
	total("Total", rupiah(d.Total), true)
	for _, t := range d.Taxes {
		if t.Kind == domain.TaxWithholding {
			total("Dipotong "+t.Name, "-"+rupiah(t.Amount), false)
		}
	}
	if d.Withholding > 0 {
		total("Dibayar", rupiah(d.NetDue()), true)
	}

	if strings.TrimSpace(d.Notes) != "" {
		pdf.Ln(5)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(0, 5, "Catatan", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(0, 4.5, t(d.Notes), "", "L", false)
	}
	pdf.Ln(12)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 5, "Hormat kami,", "", 1, "L", false, 0, "")
	pdf.Ln(16)
	pdf.CellFormat(0, 5, t(co.LegalName), "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
