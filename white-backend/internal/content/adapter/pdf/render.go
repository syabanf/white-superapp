// Package pdf renders the monthly content report as an A4 PDF with the
// built-in Helvetica faces: KPIs, breakdowns, best and weakest posts and
// the summary with recommendations.
package pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
)

const margin = 40.0

type Renderer struct{}

// safe maps characters Helvetica (WinAnsi) cannot encode: dashes and
// quotes to ASCII; everything else outside Latin-1 except the bullet is dropped.
func safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '–' || r == '—' || r == '−':
			b.WriteByte('-')
		case r == '‘' || r == '’':
			b.WriteByte('\'')
		case r == '“' || r == '”':
			b.WriteByte('"')
		case r == ' ' || r == ' ':
			b.WriteByte(' ')
		case r <= 0xff || r == '\u2022':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// thousands formats 12345 as "12.345".
func thousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func decimal1(v float64) string { return strings.Replace(fmt.Sprintf("%.1f", v), ".", ",", 1) }

type doc struct {
	f  *fpdf.Fpdf
	tr func(string) string
	w  float64
}

func (d *doc) text(style string, size float64, h float64, s string) {
	d.f.SetFont("Helvetica", style, size)
	d.f.MultiCell(0, h, d.tr(safe(s)), "", "L", false)
}

func (d *doc) heading(s string) {
	d.f.Ln(10)
	d.f.SetTextColor(17, 17, 17)
	d.text("B", 13, 18, s)
	d.f.Ln(2)
}

func (d *doc) table(header []string, widths []float64, rows [][]string) {
	d.f.SetFont("Helvetica", "B", 9)
	d.f.SetFillColor(245, 245, 247)
	for i, h := range header {
		d.f.CellFormat(widths[i], 18, d.tr(h), "B", 0, "L", true, 0, "")
	}
	d.f.Ln(-1)
	d.f.SetFont("Helvetica", "", 9)
	for _, row := range rows {
		for i, c := range row {
			c = safe(c)
			if r := []rune(c); len(r) > int(widths[i]/4.6) {
				c = string(r[:int(widths[i]/4.6)-1]) + "..."
			}
			d.f.CellFormat(widths[i], 16, d.tr(c), "B", 0, "L", false, 0, "")
		}
		d.f.Ln(-1)
	}
}

func (d *doc) kpis(items [][2]string) {
	w := d.w / float64(len(items))
	y := d.f.GetY()
	for i, kv := range items {
		x := margin + float64(i)*w
		d.f.SetFillColor(245, 245, 247)
		d.f.Rect(x+2, y, w-4, 46, "F")
		d.f.SetXY(x+8, y+6)
		d.f.SetFont("Helvetica", "", 8)
		d.f.SetTextColor(107, 107, 112)
		d.f.CellFormat(w-16, 10, d.tr(kv[0]), "", 0, "L", false, 0, "")
		d.f.SetXY(x+8, y+20)
		d.f.SetFont("Helvetica", "B", 15)
		d.f.SetTextColor(17, 17, 17)
		d.f.CellFormat(w-16, 18, d.tr(kv[1]), "", 0, "L", false, 0, "")
	}
	d.f.SetXY(margin, y+54)
}

func breakdownRows(bs []domain.Breakdown, label func(string) string) [][]string {
	rows := make([][]string, 0, len(bs))
	for _, b := range bs {
		rows = append(rows, []string{label(b.Key), strconv.Itoa(int(b.Planned)), strconv.Itoa(int(b.Published))})
	}
	return rows
}

func postRows(posts []domain.ReportPost) [][]string {
	rows := make([][]string, 0, len(posts))
	for _, p := range posts {
		day := p.PlannedDate
		if p.PublishedAt != nil {
			day = p.PublishedAt.In(domain.AgencyLocation()).Format("02-01-2006")
		}
		rows = append(rows, []string{p.Title, domain.PlatformLabel(p.Platform) + " " + domain.FormatLabel(p.Format), day, thousands(p.Engagements), thousands(p.Reach)})
	}
	return rows
}

// markdown prints the summary: "##" headings, bullets and numbered lines;
// ** markers are dropped.
func (d *doc) markdown(md string) {
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(strings.ReplaceAll(line, "**", ""))
		switch {
		case line == "":
			d.f.Ln(4)
		case strings.HasPrefix(line, "#"):
			d.heading(strings.TrimSpace(strings.TrimLeft(line, "#")))
		case strings.HasPrefix(line, "- "):
			d.text("", 10, 14, "  \u2022 "+strings.TrimPrefix(line, "- "))
		default:
			d.text("", 10, 14, line)
		}
	}
}

func (Renderer) Render(r domain.Report) ([]byte, error) {
	f := fpdf.New("P", "pt", "A4", "")
	f.SetMargins(margin, margin, margin)
	f.SetAutoPageBreak(true, 50)
	d := &doc{f: f, tr: f.UnicodeTranslatorFromDescriptor(""), w: 595.28 - 2*margin}
	f.SetFooterFunc(func() {
		f.SetY(-34)
		f.SetFont("Helvetica", "", 8)
		f.SetTextColor(107, 107, 112)
		f.CellFormat(0, 10, d.tr(fmt.Sprintf("Laporan konten %s  |  Halaman %d", safe(r.Data.ClientName), f.PageNo())), "", 0, "C", false, 0, "")
	})
	f.AddPage()
	data := r.Data

	f.SetTextColor(107, 107, 112)
	d.text("", 10, 14, "LAPORAN KONTEN BULANAN")
	f.SetTextColor(17, 17, 17)
	d.text("B", 22, 28, data.ClientName)
	d.text("", 12, 16, r.Month.Label())
	f.Ln(12)

	d.kpis([][2]string{
		{"Direncanakan", strconv.Itoa(int(data.Planned))},
		{"Tayang", strconv.Itoa(int(data.Published))},
		{"Gagal", strconv.Itoa(int(data.Failed))},
		{"Tepat waktu", decimal1(data.OnTimeRate) + "%"},
		{"Revisi/konten", decimal1(data.RevisionAvg)},
	})
	f.SetTextColor(107, 107, 112)
	d.text("", 9, 12, fmt.Sprintf("%d dari %d deadline terpenuhi. %d putaran revisi, %d versi caption. %d konten terjadwal, %d masih dikerjakan.", data.DeadlinesMet, data.DeadlinesTotal, data.RevisionsTotal, data.CaptionVersions, data.Scheduled, data.InProgress))
	f.SetTextColor(17, 17, 17)

	half := []float64{d.w * 0.5, d.w * 0.25, d.w * 0.25}
	d.heading("Per platform")
	d.table([]string{"Platform", "Rencana", "Tayang"}, half, breakdownRows(data.PerPlatform, func(k string) string { return domain.PlatformLabel(domain.Platform(k)) }))
	d.heading("Per format")
	d.table([]string{"Format", "Rencana", "Tayang"}, half, breakdownRows(data.PerFormat, func(k string) string { return domain.FormatLabel(domain.Format(k)) }))

	cols := []float64{d.w * 0.42, d.w * 0.2, d.w * 0.14, d.w * 0.12, d.w * 0.12}
	header := []string{"Konten", "Kanal", "Tayang", "Engagement", "Jangkauan"}
	d.heading("Engagement")
	if data.EngagementAvailable {
		d.text("", 10, 14, fmt.Sprintf("Total engagement %s, jangkauan %s.", thousands(data.TotalEngagements), thousands(data.TotalReach)))
		d.heading("Konten terbaik")
		d.table(header, cols, postRows(data.Top))
		if len(data.Bottom) > 0 {
			d.heading("Konten terlemah")
			d.table(header, cols, postRows(data.Bottom))
		}
	} else {
		d.text("", 10, 14, "Data engagement belum tersedia dari analytics.")
	}
	if len(data.Posts) > 0 {
		d.heading("Semua konten tayang")
		d.table(header, cols, postRows(data.Posts))
	}
	if strings.TrimSpace(r.Summary) != "" {
		d.markdown(r.Summary)
	}
	f.Ln(8)
	f.SetTextColor(107, 107, 112)
	d.text("", 8, 10, "Dibuat "+r.UpdatedAt.In(domain.AgencyLocation()).Format("02-01-2006 15:04")+" WIB")

	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
