// Package docs renders the files hr hands out: the "Rekap Absensi"
// workbook (a minimal SpreadsheetML writer, no dependency) and the payslip
// PDF (go-pdf/fpdf with the built-in Helvetica faces).
package docs

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
)

// Renderer implements app.Documents.
type Renderer struct {
	// CompanyName heads the documents.
	CompanyName string
}

var _ app.Documents = Renderer{}

// ─── XLSX ────────────────────────────────────────────────────

// AttendanceXLSX is buildAttendanceXlsx: four header lines merged across
// the table, a blank line, the column header, one row per record.
func (r Renderer) AttendanceXLSX(rows []hris.ReportRow, m app.AttendanceReportMeta) ([]byte, error) {
	header := []any{"Tanggal", "Nama Karyawan", "NIP", "Departemen", "Jabatan", "Jam Masuk", "Jam Pulang", "Jam Kerja",
		"Status", "Terlambat (menit)", "Catatan"}
	employee := m.EmployeeLabel
	if employee == "" {
		employee = "Semua Karyawan"
	}
	company := m.CompanyName
	if company == "" {
		company = r.CompanyName
	}
	aoa := [][]any{
		{company + ": Rekap Absensi"},
		{"Periode: " + m.PeriodLabel},
		{"Karyawan: " + employee},
		{"Dicetak: " + hris.DateTimeWIB(&m.GeneratedAt, "") + " WIB"},
		{},
		header,
	}
	orDash := func(s *string) string {
		if s == nil || *s == "" {
			return "-"
		}
		return *s
	}
	for _, row := range rows {
		hours, late, notes := 0.0, 0, ""
		if row.WorkHours != nil {
			hours = *row.WorkHours
		}
		if row.IsLate {
			late = row.LateMinutes
		}
		if row.Notes != nil {
			notes = *row.Notes
		}
		aoa = append(aoa, []any{hris.TanggalShort(row.Date), row.EmployeeName, orDash(row.NIP), orDash(row.Department),
			orDash(row.Position), hris.ClockWIB(row.ClockIn, "-"), hris.ClockWIB(row.ClockOut, "-"), hours,
			hris.StatusLabel(row.Status), late, notes})
	}
	return buildXLSX("Rekap Absensi", aoa, []float64{18, 26, 20, 18, 18, 10, 11, 9, 12, 16, 30}, 4, len(header), 5)
}

func colName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

func esc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// buildXLSX writes one sheet: the first mergeRows rows merged across cols,
// row boldRow (0-based) in bold.
func buildXLSX(sheet string, rows [][]any, widths []float64, mergeRows, cols, boldRow int) ([]byte, error) {
	var sh strings.Builder
	sh.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><cols>`)
	for i, w := range widths {
		fmt.Fprintf(&sh, `<col min="%d" max="%d" width="%g" customWidth="1"/>`, i+1, i+1, w)
	}
	sh.WriteString(`</cols><sheetData>`)
	for ri, row := range rows {
		fmt.Fprintf(&sh, `<row r="%d">`, ri+1)
		for ci, v := range row {
			ref := colName(ci) + strconv.Itoa(ri+1)
			style := ""
			if ri == boldRow || ri == 0 {
				style = ` s="1"`
			}
			switch x := v.(type) {
			case int:
				fmt.Fprintf(&sh, `<c r="%s"%s><v>%d</v></c>`, ref, style, x)
			case float64:
				fmt.Fprintf(&sh, `<c r="%s"%s><v>%s</v></c>`, ref, style, strconv.FormatFloat(x, 'f', -1, 64))
			default:
				fmt.Fprintf(&sh, `<c r="%s" t="inlineStr"%s><is><t xml:space="preserve">%s</t></is></c>`, ref, style, esc(fmt.Sprint(x)))
			}
		}
		sh.WriteString(`</row>`)
	}
	sh.WriteString(`</sheetData>`)
	if mergeRows > 0 {
		fmt.Fprintf(&sh, `<mergeCells count="%d">`, mergeRows)
		for i := 1; i <= mergeRows; i++ {
			fmt.Fprintf(&sh, `<mergeCell ref="A%d:%s%d"/>`, i, colName(cols-1), i)
		}
		sh.WriteString(`</mergeCells>`)
	}
	sh.WriteString(`</worksheet>`)

	files := []struct{ name, body string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
			`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" ` +
			`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="` + esc(sheet) + `" sheetId="1" r:id="rId1"/></sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
			`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`},
		{"xl/styles.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
			`<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>` +
			`<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>` +
			`<borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs>` +
			`<cellXfs count="2"><xf fontId="0"/><xf fontId="1" applyFont="1"/></cellXfs></styleSheet>`},
		{"xl/worksheets/sheet1.xml", sh.String()},
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(f.body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ─── Payslip PDF ─────────────────────────────────────────────

// safe drops what Helvetica's WinAnsi encoding cannot show.
func safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r <= 0xff {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// PayslipPDF renders the slip from its stored snapshot: earnings, employee
// deductions (with each one's share of gross), employer contributions,
// attendance basis and take-home pay.
func (r Renderer) PayslipPDF(s domain.Payslip, loans []payroll.LoanInstallmentDetail) ([]byte, error) {
	f := fpdf.New("P", "pt", "A4", "")
	tr := f.UnicodeTranslatorFromDescriptor("")
	txt := func(v string) string { return tr(safe(v)) }
	f.SetMargins(40, 40, 40)
	f.SetAutoPageBreak(true, 40)
	f.AddPage()
	w, _ := f.GetPageSize()
	inner := w - 80

	f.SetFont("Helvetica", "B", 16)
	f.CellFormat(inner, 22, txt(r.CompanyName), "", 1, "L", false, 0, "")
	f.SetFont("Helvetica", "", 10)
	f.SetTextColor(100, 100, 105)
	f.CellFormat(inner, 14, txt("Slip Gaji "+payroll.PeriodLabelID(&s.PeriodMonth, s.PeriodYear)), "", 1, "L", false, 0, "")
	f.SetTextColor(17, 17, 17)
	f.Ln(10)

	info := [][2]string{{"Nama", s.EmployeeName}, {"No. Karyawan", s.EmployeeNo}, {"Departemen", s.DepartmentName}, {"Jabatan", s.PositionName}}
	for _, kv := range info {
		f.SetFont("Helvetica", "", 10)
		f.CellFormat(110, 15, txt(kv[0]), "", 0, "L", false, 0, "")
		f.SetFont("Helvetica", "B", 10)
		f.CellFormat(inner-110, 15, txt(kv[1]), "", 1, "L", false, 0, "")
	}
	f.Ln(8)

	section := func(title string) {
		f.SetFillColor(245, 245, 247)
		f.SetFont("Helvetica", "B", 10)
		f.CellFormat(inner, 18, txt(title), "", 1, "L", true, 0, "")
	}
	line := func(label string, amount float64, note string, bold bool) {
		style := ""
		if bold {
			style = "B"
		}
		f.SetFont("Helvetica", style, 10)
		f.CellFormat(inner-170, 16, txt(label), "B", 0, "L", false, 0, "")
		f.SetFont("Helvetica", "", 8)
		f.SetTextColor(100, 100, 105)
		f.CellFormat(60, 16, txt(note), "B", 0, "R", false, 0, "")
		f.SetTextColor(17, 17, 17)
		f.SetFont("Helvetica", style, 10)
		f.CellFormat(110, 16, txt(payroll.FormatRupiah(amount)), "B", 1, "R", false, 0, "")
	}
	share := func(v float64) string {
		if p := payroll.ShareOfGross(v, s.GrossSalary); p != nil {
			return *p
		}
		return ""
	}
	nonZero := func(label string, v float64) {
		if v != 0 {
			line(label, v, "", false)
		}
	}

	section("Pendapatan")
	line("Gaji Pokok", s.BaseSalary, "", false)
	nonZero("Tunjangan Tetap", s.FixedAllowance)
	nonZero("Tunjangan Variabel", s.VariableAllowance)
	nonZero("Tunjangan Transport", s.TransportAllowance)
	nonZero("Tunjangan Makan", s.MealAllowance)
	nonZero("Tunjangan Perumahan", s.HousingAllowance)
	nonZero("Lembur", s.OvertimePay)
	nonZero("THR", s.Thr)
	nonZero("Bonus / Tambahan", s.Bonus)
	line("Total Pendapatan", s.GrossSalary, "", true)
	f.Ln(6)

	section("Potongan")
	for _, d := range []struct {
		label string
		v     float64
	}{{"BPJS TK JHT (2%)", s.BpjsTkJhtDeduction}, {"BPJS TK JP (1%)", s.BpjsTkJpDeduction}, {"BPJS Kesehatan (1%)", s.BpjsKesDeduction},
		{"Tapera", s.TaperaDeduction}, {"PPh 21", s.Pph21Deduction}, {"Potongan Cuti Tanpa Gaji", s.UnpaidLeaveDeduction},
		{"Potongan Keterlambatan", s.LateDeduction}} {
		if d.v != 0 {
			line(d.label, d.v, share(d.v), false)
		}
	}
	for _, l := range loans {
		line(payroll.LoanInstallmentLabel(l), l.Amount, share(l.Amount), false)
	}
	if len(loans) == 0 && s.LoanDeduction != 0 {
		line("Cicilan Pinjaman", s.LoanDeduction, share(s.LoanDeduction), false)
	}
	line("Total Potongan", s.TotalDeductions, share(s.TotalDeductions), true)
	f.Ln(8)

	f.SetFillColor(17, 17, 17)
	f.SetTextColor(255, 255, 255)
	f.SetFont("Helvetica", "B", 12)
	f.CellFormat(inner-170, 24, txt("  Gaji Bersih (Take Home Pay)"), "", 0, "L", true, 0, "")
	f.CellFormat(170, 24, txt(payroll.FormatRupiah(s.NetSalary)+"  "), "", 1, "R", true, 0, "")
	f.SetTextColor(17, 17, 17)
	f.Ln(10)

	section("Kontribusi Perusahaan (tidak dipotong dari gaji)")
	nonZero("BPJS TK JHT (3,7%)", s.BpjsTkJhtEmployer)
	nonZero("BPJS TK JP (2%)", s.BpjsTkJpEmployer)
	nonZero("BPJS TK JKK (0,24%)", s.BpjsTkJkkEmployer)
	nonZero("BPJS TK JKM (0,3%)", s.BpjsTkJkmEmployer)
	nonZero("BPJS Kesehatan (4%)", s.BpjsKesEmployer)
	nonZero("Tapera (0,5%)", s.TaperaEmployer)
	f.Ln(6)

	f.SetFont("Helvetica", "", 9)
	f.SetTextColor(100, 100, 105)
	basis := fmt.Sprintf("Hari kerja %g, hadir %g, terlambat %g, cuti tanpa gaji %g, lembur %g jam. PTKP %s, penghasilan kena pajak setahun %s.",
		s.WorkingDays, s.PresentDays, s.LateDays, s.UnpaidLeaveDays, s.OvertimeHours,
		payroll.FormatRupiah(s.PTKPAmount), payroll.FormatRupiah(s.TaxableIncome))
	if s.ProrateFactor < 1 {
		basis += fmt.Sprintf(" Gaji diprorata %.0f%% dari %s.", s.ProrateFactor*100, payroll.FormatRupiah(s.FullBaseSalary))
	}
	f.MultiCell(inner, 12, txt(basis), "", "L", false)
	f.Ln(4)
	f.MultiCell(inner, 12, txt("Dokumen ini dibuat otomatis oleh sistem dan sah tanpa tanda tangan."), "", "L", false)

	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
