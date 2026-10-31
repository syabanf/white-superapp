// Package render turns sales documents into bytes: quotation, invoice and
// contract PDFs (go-pdf/fpdf with the Mixist letterhead, built-in Helvetica)
// and pitch decks (PPTX written as OOXML with archive/zip).
package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/config"
)

// Company is the letterhead identity printed on every document.
type Company struct {
	LegalName, Address, Email, Phone string
	// NPWP is the tax id; the line is left out when empty.
	NPWP string
}

// Letterhead defaults; CompanyFromEnv overrides all but the brand.
const (
	brandName    = "Mixist"
	brandTagline = "Digital Marketing Agency"
	legalName    = "PT Mixist Digital Kreatif"
	brandAddress = "Jl. Senopati No. 8, Kebayoran Baru, Jakarta Selatan 12190"
	brandEmail   = "hello@mixist.id"
	brandPhone   = "+62 21 5000 8800"
)

// CompanyFromEnv reads COMPANY_LEGAL_NAME, COMPANY_ADDRESS, COMPANY_EMAIL,
// COMPANY_PHONE and COMPANY_NPWP over the defaults above.
func CompanyFromEnv() Company {
	return Company{
		LegalName: config.Env("COMPANY_LEGAL_NAME", legalName),
		Address:   config.Env("COMPANY_ADDRESS", brandAddress),
		Email:     config.Env("COMPANY_EMAIL", brandEmail),
		Phone:     config.Env("COMPANY_PHONE", brandPhone),
		NPWP:      config.Env("COMPANY_NPWP", ""),
	}
}

// Renderer implements app.Renderer.
type Renderer struct {
	Company Company
}

// New builds a renderer with the company read from the environment.
func New() Renderer { return Renderer{Company: CompanyFromEnv()} }

type rgb struct{ r, g, b int }

var (
	ink     = rgb{0x11, 0x11, 0x11}
	muted   = rgb{0x6b, 0x6b, 0x70}
	faint   = rgb{0xe5, 0xe5, 0xea}
	border  = rgb{0xd2, 0xd2, 0xda}
	tint    = rgb{0xf7, 0xf2, 0xf6}
	magenta = rgb{0xa0, 0x1d, 0x63}
	navy    = rgb{0x2b, 0x22, 0x65}
)

const (
	margin   = 42.0
	pageW    = 595.28
	contentW = pageW - 2*margin
)

// winAnsiExtra are the WinAnsi glyphs beyond Latin-1 that Helvetica encodes.
const winAnsiExtra = "€‚ƒ„…†‡ˆ‰Š‹ŒŽ‘’“”•–—˜™š›œžŸ"

// safe drops what WinAnsi cannot encode and normalises odd spaces.
func safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '−':
			b.WriteByte('-')
		case r == '\u202f' || r == '\u2009' || r == '\u00a0':
			b.WriteByte(' ')
		case r == '\u2028' || r == '\u2029':
			b.WriteByte('\n')
		case r <= 0xff || strings.ContainsRune(winAnsiExtra, r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

type doc struct {
	f  *fpdf.Fpdf
	tr func(string) string
	co Company
}

func newDoc(co Company, footer func(d *doc)) *doc {
	f := fpdf.New("P", "pt", "A4", "")
	f.SetMargins(margin, margin, margin)
	f.SetAutoPageBreak(true, 64)
	f.AliasNbPages("{nb}")
	d := &doc{f: f, tr: f.UnicodeTranslatorFromDescriptor(""), co: co}
	f.SetFooterFunc(func() {
		if footer != nil {
			footer(d)
		}
		d.font("", 7.5)
		d.color(muted)
		f.SetXY(margin, 820)
		f.CellFormat(contentW, 10, fmt.Sprintf("Halaman %d / {nb}", f.PageNo()), "", 0, "R", false, 0, "")
	})
	f.AddPage()
	d.letterhead()
	return d
}

func (d *doc) t(s string) string { return d.tr(safe(s)) }

func (d *doc) font(style string, size float64) { d.f.SetFont("Helvetica", style, size) }

func (d *doc) color(c rgb) { d.f.SetTextColor(c.r, c.g, c.b) }

func (d *doc) fill(c rgb) { d.f.SetFillColor(c.r, c.g, c.b) }

func (d *doc) stroke(c rgb) { d.f.SetDrawColor(c.r, c.g, c.b) }

// letterhead: brand mark and name on the left; legal name, address,
// contact and NPWP right-aligned on the right; a two-tone rule below.
func (d *doc) letterhead() {
	f, co := d.f, d.co
	top := margin
	d.fill(magenta)
	f.RoundedRect(margin, top, 30, 30, 6, "1234", "F")
	d.font("B", 18)
	f.SetTextColor(255, 255, 255)
	f.SetXY(margin, top)
	f.CellFormat(30, 30, "M", "", 0, "C", false, 0, "")
	d.font("B", 22)
	d.color(magenta)
	f.SetXY(margin+38, top)
	f.CellFormat(150, 20, brandName, "", 0, "L", false, 0, "")
	d.font("", 8)
	d.color(navy)
	f.SetXY(margin+38, top+20)
	f.CellFormat(150, 10, d.t(brandTagline), "", 0, "L", false, 0, "")

	const rightW = 280.0
	x, y := pageW-margin-rightW, top
	d.font("B", 12)
	d.color(navy)
	f.SetXY(x, y)
	f.CellFormat(rightW, 15, d.t(co.LegalName), "", 0, "R", false, 0, "")
	y += 16
	if co.Address != "" {
		d.font("", 7.5)
		d.color(muted)
		for _, ln := range f.SplitText(d.t(co.Address), rightW) {
			f.SetXY(x, y)
			f.CellFormat(rightW, 10, ln, "", 0, "R", false, 0, "")
			y += 10
		}
	}
	if co.Email != "" || co.Phone != "" {
		d.contactLine(y)
		y += 11
	}
	if co.NPWP != "" {
		d.font("", 7.5)
		d.color(muted)
		f.SetXY(x, y)
		f.CellFormat(rightW, 10, d.t("NPWP: "+co.NPWP), "", 0, "R", false, 0, "")
		y += 11
	}
	rule := max(y, top+34) + 8
	f.SetLineWidth(1.5)
	d.stroke(magenta)
	f.Line(margin, rule, margin+contentW*0.3, rule)
	d.stroke(navy)
	f.Line(margin+contentW*0.3, rule, pageW-margin, rule)
	f.SetLineWidth(0.5)
	f.SetY(rule + 14)
}

// contactLine prints "email · phone" right-aligned, the email in magenta.
func (d *doc) contactLine(y float64) {
	type part struct {
		text string
		c    rgb
	}
	var parts []part
	if d.co.Email != "" {
		parts = append(parts, part{d.t(d.co.Email), magenta})
	}
	if d.co.Phone != "" {
		if len(parts) > 0 {
			parts = append(parts, part{d.t("  ·  "), muted})
		}
		parts = append(parts, part{d.t(d.co.Phone), muted})
	}
	d.font("", 7.5)
	total := 0.0
	for _, p := range parts {
		total += d.f.GetStringWidth(p.text)
	}
	d.f.SetXY(pageW-margin-total, y)
	for _, p := range parts {
		d.color(p.c)
		d.f.CellFormat(d.f.GetStringWidth(p.text), 10, p.text, "", 0, "L", false, 0, "")
	}
}

// titleBand is a full-width light panel with the centred document title
// and its number.
func (d *doc) titleBand(title, number string) {
	f := d.f
	y := f.GetY()
	d.fill(tint)
	f.Rect(margin, y, contentW, 48, "F")
	d.font("B", 16)
	d.color(navy)
	f.SetXY(margin, y+8)
	f.CellFormat(contentW, 18, d.t(title), "", 0, "C", false, 0, "")
	d.font("", 9)
	d.color(ink)
	f.SetXY(margin, y+28)
	f.CellFormat(contentW, 12, d.t("Nomor : "+number), "", 0, "C", false, 0, "")
	f.SetY(y + 48 + 12)
}

// infoBox draws a bordered two-column box of "LABEL : value" rows. Empty
// values print "-"; values wrap within their column.
func (d *doc) infoBox(left, right [][2]string) {
	f := d.f
	const pad, labelW, colonW, lineH = 10.0, 84.0, 10.0, 11.0
	colW := contentW / 2
	valueW := colW - 2*pad - labelW - colonW
	column := func(x, y float64, rows [][2]string) float64 {
		for _, r := range rows {
			d.font("", 8.5)
			lines := f.SplitText(d.t(firstNonEmpty(r[1], "-")), valueW)
			d.font("I", 7)
			d.color(muted)
			f.SetXY(x, y)
			f.CellFormat(labelW, lineH, d.t(strings.ToUpper(r[0])), "", 0, "L", false, 0, "")
			f.CellFormat(colonW, lineH, ":", "", 0, "L", false, 0, "")
			d.font("", 8.5)
			d.color(ink)
			for i, ln := range lines {
				f.SetXY(x+labelW+colonW, y+float64(i)*lineH)
				f.CellFormat(valueW, lineH, ln, "", 0, "L", false, 0, "")
			}
			y += float64(max(1, len(lines)))*lineH + 4
		}
		return y
	}
	top := f.GetY()
	bottom := max(column(margin+pad, top+pad, left), column(margin+colW+pad, top+pad, right)) + pad - 4
	d.stroke(border)
	f.SetLineWidth(0.6)
	f.Rect(margin, top, contentW, bottom-top, "D")
	f.Line(margin+colW, top, margin+colW, bottom)
	f.SetLineWidth(0.5)
	f.SetY(bottom + 14)
}

type column struct {
	title string
	width float64
	align string
}

// table draws a header and rows; cells wrap and rows break pages.
func (d *doc) table(cols []column, rows [][]string) {
	f := d.f
	header := func() {
		d.fill(navy)
		f.SetTextColor(255, 255, 255)
		d.font("B", 8)
		for _, c := range cols {
			f.CellFormat(c.width, 18, d.t(c.title), "", 0, c.align, true, 0, "")
		}
		f.Ln(-1)
	}
	header()
	d.font("", 8.5)
	for _, r := range rows {
		lines := make([][]string, len(cols))
		height := 0.0
		for i, c := range cols {
			text := ""
			if i < len(r) {
				text = r[i]
			}
			var ls []string
			for _, para := range strings.Split(text, "\n") {
				ls = append(ls, f.SplitText(d.t(para), c.width-8)...)
			}
			lines[i] = ls
			height = max(height, float64(len(ls))*11+8)
		}
		if f.GetY()+height > 770 {
			f.AddPage()
			f.SetY(margin)
			header()
			d.font("", 8.5)
		}
		y := f.GetY()
		x := margin
		d.color(ink)
		for i, c := range cols {
			for j, ln := range lines[i] {
				f.SetXY(x+4, y+4+float64(j)*11)
				f.CellFormat(c.width-8, 11, ln, "", 0, c.align, false, 0, "")
			}
			x += c.width
		}
		d.stroke(faint)
		f.Line(margin, y+height, pageW-margin, y+height)
		f.SetY(y + height)
	}
	f.Ln(6)
}

// totals draws right-aligned label/value rows; the last row is bold navy
// under a rule.
func (d *doc) totals(rows [][2]string) {
	f := d.f
	const labelW, valueW = 140.0, 120.0
	x := pageW - margin - labelW - valueW
	if f.GetY()+float64(len(rows))*16+10 > 770 {
		f.AddPage()
		f.SetY(margin)
	}
	for i, r := range rows {
		style, size, c := "", 9.0, ink
		if i == len(rows)-1 {
			style, size, c = "B", 11, navy
			d.stroke(navy)
			f.Line(x, f.GetY()+2, pageW-margin, f.GetY()+2)
			f.Ln(4)
		}
		d.font(style, size)
		d.color(c)
		f.SetX(x)
		f.CellFormat(labelW, 16, d.t(r[0]), "", 0, "R", false, 0, "")
		f.CellFormat(valueW, 16, d.t(r[1]), "", 1, "R", false, 0, "")
	}
	f.Ln(8)
}

func (d *doc) section(title string) {
	if d.f.GetY() > 720 {
		d.f.AddPage()
		d.f.SetY(margin)
	}
	d.font("B", 10)
	d.color(magenta)
	d.f.CellFormat(contentW, 16, d.t(title), "", 1, "L", false, 0, "")
	d.color(ink)
}

func (d *doc) paragraph(text string, size float64) {
	d.font("", size)
	d.color(ink)
	d.f.MultiCell(contentW, size*1.45, d.t(text), "", "L", false)
	d.f.Ln(4)
}

func (d *doc) bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := d.f.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func dateID(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return domain.FormatDateID(*t)
}

// issuedID is the WIB issue day: sent, else created.
func issuedID(sent *time.Time, created time.Time) string {
	return domain.FormatDateID(firstTime(sent, created).In(domain.Jakarta))
}

// partyRows are the client-side rows shared by quotations and invoices.
func partyRows(program, client string, deal domain.Deal) [][2]string {
	return [][2]string{
		{"Program name", program},
		{"Client", client},
		{"PIC", deal.PicName},
		{"Telp / HP", deal.PicPhone},
		{"Email", deal.PicEmail},
	}
}

// QuotationPDF is quotationPDF: letterhead, title band, info box, item
// table, totals, notes with payment terms and the signature block.
func (r Renderer) QuotationPDF(q domain.Quotation, deal domain.Deal, ownerName string) ([]byte, error) {
	d := newDoc(r.Company, nil)
	d.titleBand("QUOTATION", q.QuoteNumber)
	right := [][2]string{
		{"Tanggal", issuedID(q.SentAt, q.CreatedAt)},
		{"Berlaku sampai", dateID(q.ValidUntil)},
		{"Versi", fmt.Sprint(q.Version)},
	}
	if ownerName != "" {
		right = append(right, [2]string{"Account", ownerName})
	}
	d.infoBox(partyRows(deal.Title, firstNonEmpty(q.ClientName, deal.ClientName()), deal), right)
	cols := []column{{"No", 26, "C"}, {"Layanan dan ruang lingkup", 249, "L"}, {"Qty", 70, "R"}, {"Harga", 83, "R"}, {"Jumlah", 83.28, "R"}}
	rows := make([][]string, len(q.Items))
	for i, it := range q.Items {
		desc := it.Description
		if it.Scope != "" {
			desc += "\n" + it.Scope
		}
		rows[i] = []string{fmt.Sprint(i + 1), desc, domain.FormatNumber(it.Qty, 3) + " " + it.Unit, it.UnitPrice.Rupiah(), it.LineTotal.Rupiah()}
	}
	d.table(cols, rows)
	tot := [][2]string{{"Subtotal", q.Subtotal.Rupiah()}}
	if q.DiscountNominal > 0 {
		tot = append(tot, [2]string{"Diskon " + domain.FormatNumber(q.DiscountPercent, 2) + "%", "-" + q.DiscountNominal.Rupiah()})
	}
	if q.UsePpn {
		tot = append(tot, [2]string{"PPN " + domain.FormatNumber(q.PpnPercent, 2) + "%", q.PpnNominal.Rupiah()})
	}
	tot = append(tot, [2]string{"Total", q.Total.Rupiah()})
	d.totals(tot)
	notes := ""
	if q.Notes != nil {
		notes = strings.TrimSpace(*q.Notes)
	}
	if len(q.Terms) > 0 || notes != "" {
		d.section("Catatan")
	}
	if len(q.Terms) > 0 {
		d.font("B", 8.5)
		d.color(navy)
		d.f.CellFormat(contentW, 13, d.t("Termin pembayaran"), "", 1, "L", false, 0, "")
		percents := make([]float64, len(q.Terms))
		for i, t := range q.Terms {
			percents[i] = t.Percent
		}
		amounts := domain.AllocateTermAmounts(q.Total, percents)
		trows := make([][]string, len(q.Terms))
		for i, t := range q.Terms {
			trows[i] = []string{t.Label, domain.FormatNumber(t.Percent, 2) + "%", dateID(t.DueDate), amounts[i].Rupiah()}
		}
		d.table([]column{{"Termin", 220, "L"}, {"Persen", 70, "R"}, {"Jatuh tempo", 110, "L"}, {"Nominal", 111.28, "R"}}, trows)
	}
	if notes != "" {
		d.paragraph(notes, 8.5)
	}
	d.signature("Hormat kami,", d.co.LegalName)
	return d.bytes()
}

func (d *doc) signature(greeting, name string) {
	if d.f.GetY() > 690 {
		d.f.AddPage()
		d.f.SetY(margin)
	}
	d.f.Ln(14)
	d.font("", 9)
	d.color(ink)
	d.f.CellFormat(contentW, 12, d.t(greeting), "", 1, "L", false, 0, "")
	d.f.Ln(42)
	d.font("B", 9)
	d.f.CellFormat(contentW, 12, d.t(name), "", 1, "L", false, 0, "")
}

// InvoicePDF renders a submitted or issued invoice.
func (r Renderer) InvoicePDF(inv domain.Invoice, deal domain.Deal, quoteNumber string) ([]byte, error) {
	d := newDoc(r.Company, nil)
	d.titleBand("INVOICE", inv.InvoiceNumber)
	right := [][2]string{
		{"Tanggal terbit", issuedID(inv.SentAt, inv.CreatedAt)},
		{"Jatuh tempo", dateID(inv.DueDate)},
	}
	if quoteNumber != "" {
		right = append(right, [2]string{"Referensi", quoteNumber})
	}
	status := map[string]string{"diajukan": "Diajukan", "terkirim": "Terbit", "batal": "Batal"}[inv.Status]
	right = append(right, [2]string{"Termin", inv.Label}, [2]string{"Status", status})
	d.infoBox(partyRows(firstNonEmpty(inv.DealTitle, deal.Title), firstNonEmpty(inv.ClientName, deal.ClientName()), deal), right)
	d.table([]column{{"Keterangan", 380, "L"}, {"Jumlah", 131.28, "R"}}, [][]string{{inv.Label, inv.Amount.Rupiah()}})
	d.totals([][2]string{
		{"Jumlah tagihan", inv.Amount.Rupiah()},
		{"Terbayar", inv.PaidTotal.Rupiah()},
		{"Sisa", (inv.Amount - inv.PaidTotal).Rupiah()},
	})
	d.section("Catatan")
	d.paragraph("Transfer ke rekening "+d.co.LegalName+", Bank BCA 123-456-7890. Cantumkan nomor invoice pada berita transfer.", 8.5)
	d.signature("Hormat kami,", d.co.LegalName)
	return d.bytes()
}

func firstTime(p *time.Time, t time.Time) time.Time {
	if p != nil {
		return *p
	}
	return t
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ContractPDF lays out the markdown-lite body, the signature block (with
// the client's drawn signature once signed) and an audit footer.
func (r Renderer) ContractPDF(c domain.Contract) ([]byte, error) {
	sum := sha256.Sum256([]byte(c.Body))
	digest := hex.EncodeToString(sum[:])[:16]
	footer := func(d *doc) {
		d.font("", 6.5)
		d.color(muted)
		d.f.SetXY(margin, 806)
		text := "Dokumen " + c.ContractNumber + " · sidik isi " + digest
		if c.SignedAt != nil {
			text += fmt.Sprintf(" · ditandatangani elektronik oleh %s pada %s WIB dari IP %s", deref(c.SignerName), c.SignedAt.In(domain.Jakarta).Format("02/01/2006 15:04:05"), firstNonEmpty(deref(c.SignerIP), "-"))
		}
		d.f.CellFormat(contentW, 9, d.t(text), "", 0, "L", false, 0, "")
	}
	d := newDoc(r.Company, footer)
	d.font("B", 14)
	d.color(ink)
	d.f.MultiCell(contentW, 18, d.t(strings.ToUpper(c.Title)), "", "C", false)
	d.font("", 9)
	d.color(muted)
	d.f.CellFormat(contentW, 14, d.t("Nomor: "+c.ContractNumber), "", 1, "C", false, 0, "")
	d.f.Ln(10)
	d.markdown(c.Body)
	d.contractSignatures(c)
	return d.bytes()
}

func (d *doc) spans(text string, size, lineH float64) {
	for _, s := range domain.InlineSpans(text) {
		style := ""
		if s.Bold {
			style = "B"
		}
		d.font(style, size)
		d.f.Write(lineH, d.t(s.Text))
	}
	d.f.Ln(lineH)
}

func (d *doc) markdown(body string) {
	f := d.f
	for _, b := range domain.ParseMarkdown(body) {
		d.color(ink)
		switch b.Kind {
		case domain.BlockHeading:
			size := map[int]float64{1: 13, 2: 11, 3: 10}[b.Level]
			if f.GetY() > 740 {
				f.AddPage()
				f.SetY(margin)
			}
			f.Ln(4)
			d.font("B", size)
			if b.Level == 1 {
				d.color(navy)
			}
			f.MultiCell(contentW, size*1.4, d.t(domain.PlainText(b.Text)), "", "L", false)
			f.Ln(2)
		case domain.BlockParagraph:
			f.SetX(margin)
			d.spans(b.Text, 9.5, 13.5)
			f.Ln(4)
		case domain.BlockBullets, domain.BlockNumbered:
			for i, item := range b.Items {
				marker := "•"
				if b.Kind == domain.BlockNumbered {
					marker = fmt.Sprintf("%d.", i+1)
				}
				d.font("", 9.5)
				f.SetX(margin + 8)
				f.CellFormat(16, 13.5, d.t(marker), "", 0, "L", false, 0, "")
				f.SetLeftMargin(margin + 24)
				d.spans(item, 9.5, 13.5)
				f.SetLeftMargin(margin)
			}
			f.Ln(4)
		case domain.BlockTable:
			if len(b.Rows) == 0 {
				continue
			}
			cols := tableColumns(b.Rows)
			body := make([][]string, 0, len(b.Rows)-1)
			for _, r := range b.Rows[1:] {
				row := make([]string, len(r))
				for i, cell := range r {
					row[i] = domain.PlainText(cell)
				}
				body = append(body, row)
			}
			d.table(cols, body)
		case domain.BlockRule:
			d.stroke(faint)
			f.Line(margin, f.GetY()+4, pageW-margin, f.GetY()+4)
			f.Ln(10)
		}
	}
}

// tableColumns sizes columns by their longest cell, within the page.
func tableColumns(rows [][]string) []column {
	n := len(rows[0])
	weights := make([]float64, n)
	for _, r := range rows {
		for i := 0; i < n && i < len(r); i++ {
			weights[i] = max(weights[i], min(float64(len([]rune(r[i]))), 60)+4)
		}
	}
	total := 0.0
	for _, w := range weights {
		total += w
	}
	cols := make([]column, n)
	for i, h := range rows[0] {
		if i >= n {
			break
		}
		align := "L"
		if len(rows) > 1 && i < len(rows[1]) && strings.HasPrefix(strings.TrimSpace(rows[1][i]), "Rp") {
			align = "R"
		}
		cols[i] = column{title: domain.PlainText(h), width: contentW * weights[i] / total, align: align}
	}
	return cols
}

func (d *doc) contractSignatures(c domain.Contract) {
	f := d.f
	if f.GetY() > 640 {
		f.AddPage()
		f.SetY(margin)
	}
	f.Ln(16)
	colW := contentW / 2
	y := f.GetY()
	d.font("B", 9)
	d.color(ink)
	f.SetXY(margin, y)
	f.CellFormat(colW, 12, d.t("PIHAK PERTAMA"), "", 0, "L", false, 0, "")
	f.CellFormat(colW, 12, d.t("PIHAK KEDUA"), "", 1, "L", false, 0, "")
	d.font("", 9)
	f.SetX(margin)
	f.CellFormat(colW, 12, d.t(d.co.LegalName), "", 0, "L", false, 0, "")
	f.CellFormat(colW, 12, d.t(c.ClientName), "", 1, "L", false, 0, "")
	sigY := f.GetY() + 6
	if len(c.SignaturePNG) > 0 {
		name := "sig-" + c.ID
		opts := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}
		if info := f.RegisterImageOptionsReader(name, opts, bytes.NewReader(c.SignaturePNG)); info != nil && f.Ok() {
			f.ImageOptions(name, margin+colW, sigY, 140, 0, false, opts, 0, "")
		} else {
			f.ClearError()
		}
	}
	f.SetY(sigY + 64)
	d.stroke(ink)
	f.Line(margin, f.GetY(), margin+colW-30, f.GetY())
	f.Line(margin+colW, f.GetY(), pageW-margin-10, f.GetY())
	f.Ln(4)
	d.font("B", 9)
	f.SetX(margin)
	f.CellFormat(colW, 12, "Direktur", "", 0, "L", false, 0, "")
	signer := "Nama dan tanda tangan"
	if c.SignerName != nil {
		signer = *c.SignerName
	}
	f.CellFormat(colW, 12, d.t(signer), "", 1, "L", false, 0, "")
	if c.SignedAt != nil {
		d.font("", 7.5)
		d.color(muted)
		f.SetX(margin + colW)
		f.CellFormat(colW, 10, d.t("Ditandatangani "+c.SignedAt.In(domain.Jakarta).Format("02/01/2006 15:04")+" WIB"), "", 1, "L", false, 0, "")
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
