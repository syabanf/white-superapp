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
)

// Renderer implements app.Renderer.
type Renderer struct{}

func New() Renderer { return Renderer{} }

// Letterhead details printed on every document.
const (
	brandName    = "Mixist"
	brandTagline = "Digital Marketing Agency"
	legalName    = "PT Mixist Digital Kreatif"
	brandAddress = "Jl. Senopati No. 8, Kebayoran Baru, Jakarta Selatan 12190"
	brandContact = "hello@mixist.id  ·  +62 21 5000 8800  ·  mixist.id"
)

type rgb struct{ r, g, b int }

var (
	ink    = rgb{0x11, 0x11, 0x11}
	muted  = rgb{0x6b, 0x6b, 0x70}
	faint  = rgb{0xe5, 0xe5, 0xea}
	panel  = rgb{0xf5, 0xf5, 0xf7}
	accent = rgb{0x00, 0x28, 0x1a}
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
		case r == ' ' || r == ' ' || r == ' ':
			b.WriteByte(' ')
		case r == ' ' || r == ' ':
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
}

func newDoc(footer func(d *doc)) *doc {
	f := fpdf.New("P", "pt", "A4", "")
	f.SetMargins(margin, margin, margin)
	f.SetAutoPageBreak(true, 64)
	f.AliasNbPages("{nb}")
	d := &doc{f: f, tr: f.UnicodeTranslatorFromDescriptor("")}
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

func (d *doc) letterhead() {
	f := d.f
	d.font("B", 24)
	d.color(accent)
	f.SetXY(margin, margin)
	f.CellFormat(200, 26, brandName, "", 0, "L", false, 0, "")
	d.font("", 8)
	d.color(muted)
	f.SetXY(margin, margin+26)
	f.CellFormat(200, 10, d.t(brandTagline), "", 0, "L", false, 0, "")
	right := []string{legalName, brandAddress, brandContact}
	for i, line := range right {
		style := ""
		if i == 0 {
			style = "B"
		}
		d.font(style, 8)
		d.color(ink)
		f.SetXY(margin, margin+4+float64(i)*11)
		f.CellFormat(contentW, 11, d.t(line), "", 0, "R", false, 0, "")
	}
	f.SetDrawColor(accent.r, accent.g, accent.b)
	f.SetLineWidth(1.2)
	f.Line(margin, margin+44, pageW-margin, margin+44)
	f.SetLineWidth(0.5)
	f.SetY(margin + 58)
}

func (d *doc) title(text, number string) {
	d.font("B", 15)
	d.color(ink)
	d.f.CellFormat(contentW, 18, d.t(text), "", 1, "L", false, 0, "")
	d.font("", 9)
	d.color(muted)
	d.f.CellFormat(contentW, 12, d.t(number), "", 1, "L", false, 0, "")
	d.f.Ln(8)
}

// grid draws label/value pairs in two columns on a light panel.
func (d *doc) grid(pairs [][2]string) {
	f := d.f
	rows := (len(pairs) + 1) / 2
	y := f.GetY()
	f.SetFillColor(panel.r, panel.g, panel.b)
	f.Rect(margin, y, contentW, float64(rows)*30+8, "F")
	for i, p := range pairs {
		col, row := i%2, i/2
		x := margin + 10 + float64(col)*(contentW/2)
		f.SetXY(x, y+6+float64(row)*30)
		d.font("", 7.5)
		d.color(muted)
		f.CellFormat(contentW/2-20, 10, d.t(strings.ToUpper(p[0])), "", 2, "L", false, 0, "")
		f.SetX(x)
		d.font("B", 9.5)
		d.color(ink)
		f.CellFormat(contentW/2-20, 13, d.t(p[1]), "", 0, "L", false, 0, "")
	}
	f.SetY(y + float64(rows)*30 + 18)
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
		f.SetFillColor(accent.r, accent.g, accent.b)
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
		f.SetDrawColor(faint.r, faint.g, faint.b)
		f.Line(margin, y+height, pageW-margin, y+height)
		f.SetY(y + height)
	}
	f.Ln(6)
}

// totals draws right-aligned label/value rows; the last row is bold.
func (d *doc) totals(rows [][2]string) {
	f := d.f
	for i, r := range rows {
		bold := ""
		size := 9.0
		if i == len(rows)-1 {
			bold, size = "B", 11
		}
		d.font(bold, size)
		d.color(ink)
		f.SetX(margin + contentW - 260)
		f.CellFormat(140, 16, d.t(r[0]), "", 0, "R", false, 0, "")
		f.CellFormat(120, 16, d.t(r[1]), "", 1, "R", false, 0, "")
	}
	f.Ln(6)
}

func (d *doc) section(title string) {
	if d.f.GetY() > 720 {
		d.f.AddPage()
		d.f.SetY(margin)
	}
	d.font("B", 10)
	d.color(accent)
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

// QuotationPDF is quotationPDF: letterhead, info grid, item table, totals,
// payment terms, notes and the signature block.
func (Renderer) QuotationPDF(q domain.Quotation, deal domain.Deal) ([]byte, error) {
	d := newDoc(nil)
	d.title("PENAWARAN HARGA", q.QuoteNumber+"  ·  Versi "+fmt.Sprint(q.Version))
	d.grid([][2]string{
		{"Kepada", q.ClientName},
		{"Tanggal", domain.FormatDateID(q.CreatedAt.In(domain.Jakarta))},
		{"Up.", firstNonEmpty(deal.PicName, "-")},
		{"Berlaku s.d.", dateID(q.ValidUntil)},
		{"Proyek", deal.Title},
		{"Status", strings.ToUpper(q.Status)},
	})
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
	tot = append(tot, [2]string{"TOTAL", q.Total.Rupiah()})
	d.totals(tot)
	if len(q.Terms) > 0 {
		d.section("Termin Pembayaran")
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
	if q.Notes != nil && strings.TrimSpace(*q.Notes) != "" {
		d.section("Catatan")
		d.paragraph(*q.Notes, 8.5)
	}
	d.signature("Hormat kami,", legalName)
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
func (Renderer) InvoicePDF(inv domain.Invoice, deal domain.Deal) ([]byte, error) {
	d := newDoc(nil)
	d.title("INVOICE", inv.InvoiceNumber)
	status := map[string]string{"diajukan": "Diajukan", "terkirim": "Terbit", "batal": "Batal"}[inv.Status]
	d.grid([][2]string{
		{"Ditagihkan kepada", inv.ClientName},
		{"Tanggal", domain.FormatDateID(firstTime(inv.SentAt, inv.CreatedAt).In(domain.Jakarta))},
		{"Up.", firstNonEmpty(deal.PicName, "-")},
		{"Jatuh tempo", dateID(inv.DueDate)},
		{"Proyek", inv.DealTitle},
		{"Status", status},
	})
	d.table([]column{{"Keterangan", 380, "L"}, {"Jumlah", 131.28, "R"}}, [][]string{{inv.Label, inv.Amount.Rupiah()}})
	d.totals([][2]string{{"Sudah dibayar", inv.PaidTotal.Rupiah()}, {"TOTAL TAGIHAN", inv.Amount.Rupiah()}})
	d.section("Pembayaran")
	d.paragraph("Transfer ke rekening "+legalName+", Bank BCA 123-456-7890. Cantumkan nomor invoice pada berita transfer.", 8.5)
	d.signature("Hormat kami,", legalName)
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
func (Renderer) ContractPDF(c domain.Contract) ([]byte, error) {
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
	d := newDoc(footer)
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
				d.color(accent)
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
			f.SetDrawColor(faint.r, faint.g, faint.b)
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
	f.CellFormat(colW, 12, d.t(legalName), "", 0, "L", false, 0, "")
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
	f.SetDrawColor(ink.r, ink.g, ink.b)
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
