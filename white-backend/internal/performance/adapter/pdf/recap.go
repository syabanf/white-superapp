// Package pdf renders the monthly work recap as an A4 PDF with the
// built-in Helvetica faces.
package pdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

const margin = 18.0

// safe drops what WinAnsi (Helvetica) cannot encode.
func safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\u2013' || r == '\u2014' || r == '\u2212':
			b.WriteByte('-')
		case r == '\u2022':
			b.WriteByte('*')
		case r <= 0xff:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func pct(v *float64) string {
	if v == nil {
		return "-"
	}
	return domain.FormatDecimal(*v*100, 1) + "%"
}

func num(v *float64) string {
	if v == nil {
		return "-"
	}
	return domain.FormatDecimal(*v, 2)
}

// RenderMonthlyRecap implements app.RecapPDF.
func RenderMonthlyRecap(r domain.MonthlyRecap, subjectName, departmentName string) ([]byte, error) {
	f := fpdf.New("P", "mm", "A4", "")
	tr := f.UnicodeTranslatorFromDescriptor("")
	text := func(s string) string { return tr(safe(s)) }
	f.SetMargins(margin, margin, margin)
	f.SetAutoPageBreak(true, margin)
	f.SetFooterFunc(func() {
		f.SetY(-12)
		f.SetFont("Helvetica", "", 8)
		f.SetTextColor(120, 120, 125)
		f.CellFormat(0, 5, text(fmt.Sprintf("WHITE ERP - dibuat %s - halaman %d", r.GeneratedAt.In(domain.Jakarta).Format("02/01/2006 15:04"), f.PageNo())),
			"", 0, "C", false, 0, "")
	})
	f.AddPage()

	f.SetFont("Helvetica", "B", 18)
	f.SetTextColor(17, 17, 17)
	scope := "Rekap Kerja Bulanan"
	if r.Scope == domain.ScopeDepartment {
		scope = "Rekap Departemen Bulanan"
	}
	f.CellFormat(0, 9, text(scope), "", 1, "L", false, 0, "")
	f.SetFont("Helvetica", "", 11)
	f.SetTextColor(107, 107, 112)
	sub := subjectName
	if departmentName != "" && departmentName != subjectName {
		sub += " - " + departmentName
	}
	f.CellFormat(0, 6, text(sub+" | "+r.Period.Label()), "", 1, "L", false, 0, "")
	f.Ln(4)

	m := r.Metrics
	rows := [][2]string{
		{"Tugas terjadwal", fmt.Sprint(m.TasksPlanned)},
		{"Tugas disetujui", fmt.Sprintf("%d (%s)", m.TasksDone, pct(m.CompletionRate))},
		{"Tepat waktu", fmt.Sprintf("%d (%s)", m.TasksOnTime, pct(m.OnTimeRate))},
		{"Tugas terlambat", fmt.Sprint(m.TasksOverdue)},
		{"Skor KPI", num(m.KPIScore)},
		{"Logbook", fmt.Sprintf("%d entri, kepatuhan %s", m.LogbookEntries, pct(m.LogbookCompliance))},
		{"Item perbaikan terbuka", fmt.Sprint(m.ImprovementsOpen)},
	}
	if m.Members > 0 {
		rows = append([][2]string{{"Anggota", fmt.Sprint(m.Members)}}, rows...)
	}
	if a := m.Attendance; a != nil {
		rows = append(rows, [2]string{"Kehadiran", fmt.Sprintf("%d/%d hari, %d terlambat, %d cuti, %d absen", a.PresentDays, a.WorkingDays, a.LateDays, a.LeaveDays, a.AbsentDays)})
	}
	f.SetFont("Helvetica", "B", 12)
	f.SetTextColor(17, 17, 17)
	f.CellFormat(0, 7, text("Ringkasan angka"), "", 1, "L", false, 0, "")
	f.SetDrawColor(229, 229, 234)
	for _, row := range rows {
		f.SetFont("Helvetica", "", 10)
		f.SetTextColor(107, 107, 112)
		f.CellFormat(60, 7, text(row[0]), "B", 0, "L", false, 0, "")
		f.SetTextColor(17, 17, 17)
		f.CellFormat(0, 7, text(row[1]), "B", 1, "L", false, 0, "")
	}
	f.Ln(5)

	if len(r.Accomplishments) > 0 {
		f.SetFont("Helvetica", "B", 12)
		f.CellFormat(0, 7, text("Capaian utama"), "", 1, "L", false, 0, "")
		f.SetFont("Helvetica", "", 10)
		for _, a := range r.Accomplishments {
			f.MultiCell(0, 5.5, text("* "+a), "", "L", false)
		}
		f.Ln(4)
	}

	f.SetFont("Helvetica", "B", 12)
	f.CellFormat(0, 7, text("Narasi"), "", 1, "L", false, 0, "")
	f.SetFont("Helvetica", "", 10)
	for _, para := range strings.Split(strings.TrimSpace(r.Narrative), "\n") {
		if strings.TrimSpace(para) == "" {
			f.Ln(2)
			continue
		}
		f.MultiCell(0, 5.5, text(strings.TrimSpace(strings.TrimLeft(para, "#*- "))), "", "L", false)
	}
	if f.Err() {
		return nil, f.Error()
	}
	var buf bytes.Buffer
	if err := f.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
