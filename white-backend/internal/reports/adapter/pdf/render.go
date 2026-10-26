// Package pdf renders the branded A4 report (report-document.tsx) with the
// built-in Helvetica faces: cover, one page per module, the optional
// strategic summary and a closing page.
package pdf

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/syabanf/white-superapp/white-backend/internal/reports/domain"
)

type rgb struct{ r, g, b int }

var (
	ink    = rgb{0x11, 0x11, 0x11}
	muted  = rgb{0x6b, 0x6b, 0x70}
	faint  = rgb{0xe5, 0xe5, 0xea}
	pillBg = rgb{0xf5, 0xf5, 0xf7}
	barPos = rgb{0x35, 0x35, 0x35} // ink at 85% opacity on white
	barNeg = rgb{0x98, 0x98, 0x9d}
	good   = rgb{0x2f, 0x6b, 0x3a}
	bad    = rgb{0x8a, 0x34, 0x34}
	white  = rgb{0xff, 0xff, 0xff}
)

const (
	fontReg     = ""
	fontB       = "B"
	margin      = 48.0
	bottomSpace = 64.0 // keeps body text clear of the footer
	chartHeight = 72.0
)

// winAnsiExtra are the WinAnsi glyphs beyond Latin-1 that Helvetica encodes.
const winAnsiExtra = "€‚ƒ„…†‡ˆ‰Š‹ŒŽ‘’“”•–—˜™š›œžŸ"

// Safe turns U+2212 into "-", NBSP and thin spaces into spaces, line and
// paragraph separators into newlines, and drops what WinAnsi cannot encode.
func Safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\u2212':
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
	f          *fpdf.Fpdf
	tr         func(string) string
	data       domain.PdfData
	labels     domain.Labels
	totalPages int
	footer     bool
	pageW      float64
	pageH      float64
	contentW   float64
}

// Render lays the document out twice: the first pass counts the pages so
// the footer can print "Halaman n / N" with the real total.
func Render(data domain.PdfData) ([]byte, error) {
	first, err := layout(data, 0)
	if err != nil {
		return nil, err
	}
	second, err := layout(data, first.f.PageNo())
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := second.f.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func layout(data domain.PdfData, totalPages int) (*doc, error) {
	f := fpdf.New("P", "pt", "A4", "")
	f.SetMargins(margin, margin, margin)
	f.SetAutoPageBreak(true, bottomSpace)
	f.SetCellMargin(0)
	d := &doc{f: f, tr: f.UnicodeTranslatorFromDescriptor(""), data: data, labels: domain.LabelsFor(data.Lang), totalPages: totalPages}
	d.pageW, d.pageH = f.GetPageSize()
	d.contentW = d.pageW - 2*margin
	f.SetTitle(d.tr(Safe(data.Title)), false)
	f.SetAuthor("WHITE Analytics", false)
	f.SetCreator("WHITE Analytics", false)
	f.SetProducer("WHITE Analytics", false)
	f.SetFooterFunc(d.drawFooter)

	d.cover()
	for _, m := range data.Modules {
		d.modulePage(m)
	}
	if strings.TrimSpace(data.AiSummary) != "" {
		d.summaryPage()
	}
	d.closing()
	f.Close()
	return d, f.Error()
}

// ─── primitives ─────────────────────────────────────────────

func (d *doc) enc(s string) string { return d.tr(Safe(s)) }

func (d *doc) font(style string, size float64, c rgb) {
	d.f.SetFont("Helvetica", style, size)
	d.f.SetTextColor(c.r, c.g, c.b)
}

func (d *doc) width(s string) float64 { return d.f.GetStringWidth(d.enc(s)) }

// text draws s with its top-left corner at (x, top).
func (d *doc) text(x, top float64, s string) {
	_, size := d.f.GetFontSize()
	d.f.Text(x, top+size*0.8, d.enc(s))
}

func (d *doc) textRight(right, top float64, s string) { d.text(right-d.width(s), top, s) }

// lines wraps s to width w with the current font.
func (d *doc) lines(s string, w float64) []string {
	var out []string
	for _, l := range d.f.SplitLines([]byte(d.enc(s)), w) {
		out = append(out, string(l))
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

func (d *doc) hline(x1, x2, y, w float64, c rgb) {
	d.f.SetDrawColor(c.r, c.g, c.b)
	d.f.SetLineWidth(w)
	d.f.Line(x1, y, x2, y)
}

func (d *doc) fill(c rgb) { d.f.SetFillColor(c.r, c.g, c.b) }

// ensure starts a new page when h points do not fit above the bottom space.
func (d *doc) ensure(h float64) {
	if d.f.GetY()+h > d.pageH-bottomSpace {
		d.f.AddPage()
	}
}

func (d *doc) drawFooter() {
	if !d.footer {
		return
	}
	top := d.pageH - 24 - 17
	d.hline(margin, d.pageW-margin, top, 0.5, faint)
	d.font(fontReg, 7.5, muted)
	d.text(margin, top+8, d.labels.Footer+" · "+d.data.ClientName)
	d.textRight(d.pageW-margin, top+8, d.labels.Page+" "+strconv.Itoa(d.f.PageNo())+" / "+strconv.Itoa(d.totalPages))
}

// ─── pages ──────────────────────────────────────────────────

func (d *doc) cover() {
	f, L := d.f, d.labels
	d.footer = false
	f.AddPage()
	d.fill(ink)
	f.Circle(margin+6, margin+16, 6, "F")
	d.font(fontB, 28, ink)
	d.text(margin+20, margin, "white")

	y := margin + 34 + 140
	d.font(fontReg, 8, muted)
	d.text(margin, y, strings.ToUpper(L.Report))
	y += 18
	d.font(fontB, 24, ink)
	for _, l := range d.lines(d.data.Title, d.contentW) {
		f.Text(margin, y+24*0.8, l)
		y += 24 * 1.2
	}
	y += 28
	d.hline(margin, d.pageW-margin, y, 1, ink)
	y += 16
	d.font(fontReg, 8, muted)
	d.text(margin, y, strings.ToUpper(L.PreparedFor))
	d.font(fontB, 13, ink)
	d.text(margin, y+13, d.data.ClientName)
	y += 13 + 16 + 12

	d.font(fontReg, 8, muted)
	d.text(margin, y, strings.ToUpper(L.Period))
	d.font(fontReg, 10, ink)
	d.text(margin, y+13, d.data.PeriodLabel)
	colW := d.width(d.data.PeriodLabel)
	if d.data.ComparedLabel != "" {
		d.font(fontReg, 8, muted)
		d.text(margin, y+27, d.data.ComparedLabel)
		colW = max(colW, d.width(d.data.ComparedLabel))
	}
	d.font(fontReg, 8, muted)
	colW = max(colW, d.width(strings.ToUpper(L.Period)))
	x2 := margin + colW + 48
	d.text(x2, y, strings.ToUpper(L.GeneratedOn))
	d.font(fontReg, 10, ink)
	d.text(x2, y+13, d.data.GeneratedLabel)

	bottom := d.pageH - 40 - 10
	d.font(fontReg, 8, muted)
	d.text(margin, bottom, L.Footer)
	titles := make([]string, 0, len(d.data.Modules))
	for _, m := range d.data.Modules {
		titles = append(titles, m.Title)
	}
	d.textRight(d.pageW-margin, bottom, strings.Join(titles, " · "))
}

func (d *doc) pageHeader(title, right string) {
	f := d.f
	y := f.GetY()
	d.font(fontB, 14, ink)
	d.text(margin, y, title)
	if right != "" {
		d.font(fontReg, 8, muted)
		d.textRight(d.pageW-margin, y+4, right)
	}
	y += 14 + 10 + 8
	d.hline(margin, d.pageW-margin, y, 1, ink)
	f.SetY(y + 14)
}

func (d *doc) modulePage(m domain.PdfModule) {
	d.f.AddPage()
	d.footer = true
	d.pageHeader(m.Title, d.data.ClientName+" · "+d.data.PeriodLabel)
	d.kpis(m.Kpis)
	if m.Chart != nil && len(m.Chart.Points) > 1 {
		d.chart(*m.Chart)
	}
	for _, t := range m.Tables {
		d.table(t)
	}
	for _, s := range m.Sections {
		d.section(s)
	}
}

func (d *doc) summaryPage() {
	d.f.AddPage()
	d.footer = true
	d.pageHeader(d.labels.ExecutiveSummary, "")
	d.markdown(d.data.AiSummary)
}

func (d *doc) closing() {
	f, L := d.f, d.labels
	f.AddPage()
	d.footer = false
	cx := d.pageW / 2
	y := margin + 240
	d.fill(ink)
	f.Circle(cx, y+5, 5, "F")
	y += 20
	d.font(fontB, 20, ink)
	d.text(cx-d.width(L.Closing)/2, y, L.Closing)
	y += 30
	d.font(fontReg, 9.5, muted)
	for _, l := range d.lines(L.ClosingBody, 320) {
		f.Text(cx-f.GetStringWidth(l)/2, y+9.5*0.8, l)
		y += 9.5 * 1.5
	}
	pill := L.Footer + " · " + d.data.GeneratedLabel
	d.font(fontReg, 8, muted)
	w := d.width(pill) + 16
	top := d.pageH - 40 - 14
	d.fill(pillBg)
	f.RoundedRect(cx-w/2, top, w, 14, 3, "1234", "F")
	d.text(cx-w/2+8, top+3, pill)
}

// ─── blocks ─────────────────────────────────────────────────

func (d *doc) kpis(kpis []domain.PdfKpi) {
	f := d.f
	const gap, pad = 8.0, 10.0
	tileW := d.contentW * 0.315
	for i := 0; i < len(kpis); i += 3 {
		row := kpis[i:min(i+3, len(kpis))]
		h := pad + 9 + 4 + 17 + pad
		for _, k := range row {
			if k.Delta != "" {
				h = pad + 9 + 4 + 17 + 3 + 9 + pad
			}
		}
		if i > 0 {
			f.SetY(f.GetY() + gap)
		}
		d.ensure(h)
		y := f.GetY()
		for j, k := range row {
			x := margin + float64(j)*(tileW+gap)
			d.fill(white)
			f.SetDrawColor(faint.r, faint.g, faint.b)
			f.SetLineWidth(0.5)
			f.RoundedRect(x, y, tileW, h, 6, "1234", "FD")
			d.font(fontReg, 7.5, muted)
			d.text(x+pad, y+pad, d.fit(strings.ToUpper(k.Label), tileW-2*pad))
			d.font(fontB, 14, ink)
			d.text(x+pad, y+pad+13, d.fit(k.Value, tileW-2*pad))
			if k.Delta != "" {
				c := muted
				if k.DeltaGood != nil {
					c = bad
					if *k.DeltaGood {
						c = good
					}
				}
				d.font(fontReg, 7.5, c)
				d.text(x+pad, y+pad+13+20, k.Delta)
			}
		}
		f.SetY(y + h)
	}
}

// fit trims s with an ellipsis until it fits w with the current font.
func (d *doc) fit(s string, w float64) string {
	if d.width(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && d.width(string(r)+"…") > w {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// chart draws bars on a hairline baseline: positives above in ink,
// negatives below in grey, first and last date underneath.
func (d *doc) chart(c domain.PdfChart) {
	f := d.f
	var maxPos, maxNeg float64
	for _, p := range c.Points {
		maxPos = max(maxPos, p.Value)
		maxNeg = max(maxNeg, -p.Value)
	}
	span := maxPos + maxNeg
	posH, negH := chartHeight, 0.0
	if span > 0 {
		posH, negH = maxPos/span*chartHeight, maxNeg/span*chartHeight
	}
	f.SetY(f.GetY() + 12)
	d.ensure(13 + chartHeight + 14)
	y := f.GetY()
	d.font(fontB, 9, ink)
	d.text(margin, y, c.Title)
	d.font(fontReg, 7.5, muted)
	d.textRight(d.pageW-margin, y+1, "maks "+c.MaxLabel)
	y += 13
	gap := 2.0
	if len(c.Points) > 60 {
		gap = 1
	}
	n := float64(len(c.Points))
	barW := (d.contentW - n*gap) / n
	base := y + posH
	for i, p := range c.Points {
		x := margin + float64(i)*(barW+gap)
		switch {
		case p.Value > 0 && maxPos > 0:
			h := max(1, p.Value/maxPos*posH)
			d.fill(barPos)
			f.Rect(x, base-h, barW, h, "F")
		case p.Value < 0 && maxNeg > 0:
			h := max(1, -p.Value/maxNeg*negH)
			d.fill(barNeg)
			f.Rect(x, base, barW, h, "F")
		}
	}
	d.hline(margin, d.pageW-margin, base, 0.5, ink)
	y = base + negH + 3
	d.font(fontReg, 7, muted)
	d.text(margin, y, c.Points[0].Date)
	d.textRight(d.pageW-margin, y, c.Points[len(c.Points)-1].Date)
	f.SetY(y + 9)
}

// column is the i-th column, 20% wide when the table defines fewer.
func column(t domain.Table, i int) domain.Column {
	if i < len(t.Columns) && t.Columns[i].Width > 0 {
		return t.Columns[i]
	}
	return domain.Column{Width: 20}
}

func (d *doc) colWidth(c domain.Column) float64 { return d.contentW * float64(c.Width) / 100 }

// rowCells wraps every cell; the row is as tall as its tallest cell.
func (d *doc) rowCells(t domain.Table, row []string) ([][]string, float64) {
	cells := make([][]string, len(row))
	maxLines := 1
	for i, cell := range row {
		cells[i] = d.lines(cell, d.colWidth(column(t, i))-6)
		maxLines = max(maxLines, len(cells[i]))
	}
	return cells, float64(maxLines)*10.5 + 8
}

func (d *doc) drawRow(t domain.Table, cells [][]string, h float64) {
	f := d.f
	y := f.GetY()
	x := margin
	for i, lines := range cells {
		col := column(t, i)
		w := d.colWidth(col)
		for li, l := range lines {
			lx := x
			if col.Align == "right" {
				lx = x + w - 6 - f.GetStringWidth(l)
			}
			f.Text(lx, y+4+float64(li)*10.5+8.5*0.8, l)
		}
		x += w
	}
	d.hline(margin, d.pageW-margin, y+h, 0.5, faint)
	f.SetY(y + h)
}

// table keeps the title, the header and the first row together.
func (d *doc) table(t domain.Table) {
	f := d.f
	f.SetY(f.GetY() + 14)
	d.font(fontReg, 8.5, ink)
	var firstCells [][]string
	firstH := 0.0
	if len(t.Rows) > 0 {
		firstCells, firstH = d.rowCells(t, t.Rows[0])
	}
	d.ensure(15 + 16 + firstH)
	y := f.GetY()
	d.font(fontB, 9, ink)
	d.text(margin, y, t.Title)
	y += 15
	d.font(fontB, 8, ink)
	x := margin
	for i := range t.Columns {
		c := column(t, i)
		w := d.colWidth(c)
		label := d.fit(c.Label, w-6)
		if c.Align == "right" {
			d.textRight(x+w-6, y, label)
		} else {
			d.text(x, y, label)
		}
		x += w
	}
	y += 12
	d.hline(margin, d.pageW-margin, y, 1, ink)
	f.SetY(y + 2)
	d.font(fontReg, 8.5, ink)
	for i, row := range t.Rows {
		cells, h := firstCells, firstH
		if i > 0 {
			cells, h = d.rowCells(t, row)
			d.ensure(h)
		}
		d.drawRow(t, cells, h)
	}
}

func (d *doc) section(s domain.PdfSection) {
	f := d.f
	f.SetY(f.GetY() + 20)
	d.ensure(12 + 19 + 62)
	y := f.GetY()
	d.hline(margin, d.pageW-margin, y, 0.5, faint)
	y += 12
	d.font(fontB, 11, ink)
	d.text(margin, y, s.Title)
	f.SetY(y + 19)
	d.kpis(s.Kpis)
	if s.Table != nil && len(s.Table.Rows) > 0 {
		d.table(*s.Table)
	}
}

// markdown renders the summary with hanging indents for list items.
func (d *doc) markdown(text string) {
	f := d.f
	const size = 9.0
	spans := func(ss []domain.Span, lineH float64) {
		for _, sp := range ss {
			style := fontReg
			if sp.Bold {
				style = fontB
			}
			d.font(style, size, ink)
			f.Write(lineH, d.enc(sp.Text))
		}
	}
	for _, b := range domain.ParseMarkdownLite(text) {
		switch b.Kind {
		case domain.BlockHeading:
			f.SetY(f.GetY() + 12)
			d.ensure(20)
			f.SetX(margin)
			d.font(fontB, 10.5, ink)
			for _, sp := range b.Spans {
				f.Write(13, d.enc(sp.Text))
			}
			f.SetY(f.GetY() + 13 + 6)
		case domain.BlockBullet, domain.BlockNumbered:
			const lineH = size * 1.45
			d.ensure(lineH)
			marker, markerStyle := "•", fontReg
			if b.Kind == domain.BlockNumbered {
				marker, markerStyle = b.Number+".", fontB
			}
			d.font(markerStyle, size, ink)
			markerX := margin + 4
			indent := markerX + d.width(marker) + 5
			f.SetLeftMargin(indent)
			y := f.GetY()
			d.text(markerX, y+(lineH-size)/2, marker)
			f.SetXY(indent, y)
			spans(b.Spans, lineH)
			f.SetLeftMargin(margin)
			f.SetY(f.GetY() + lineH + 3)
		default:
			const lineH = size * 1.5
			d.ensure(lineH)
			f.SetX(margin)
			spans(b.Spans, lineH)
			f.SetY(f.GetY() + lineH + 4)
		}
	}
}
