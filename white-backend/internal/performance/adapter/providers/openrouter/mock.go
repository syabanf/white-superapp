package openrouter

import (
	"fmt"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
)

// The mock fills templates with the recap numbers: same input, same text.

func pct(v *float64) string {
	if v == nil {
		return "-"
	}
	return domain.FormatDecimal(*v*100, 1) + "%"
}

func listTitles(titles []string, limit int) string {
	if len(titles) > limit {
		return strings.Join(titles[:limit], ", ") + fmt.Sprintf(" dan %d lainnya", len(titles)-limit)
	}
	return strings.Join(titles, ", ")
}

// MockDailySummary describes the day in a few sentences.
func MockDailySummary(in domain.DailyRecapInput) string {
	if in.Items.Empty() && strings.TrimSpace(in.Notes) == "" {
		return "Belum ada aktivitas tugas atau logbook yang tercatat hari ini."
	}
	var done, open []string
	for _, t := range in.Items.Tasks {
		if t.Completed {
			done = append(done, t.Title)
		} else {
			open = append(open, fmt.Sprintf("%s (%d%%)", t.Title, t.Progress))
		}
	}
	var parts []string
	switch {
	case len(done) > 0:
		parts = append(parts, fmt.Sprintf("Hari ini %s menyelesaikan %d tugas: %s.", in.EmployeeName, len(done), listTitles(done, 3)))
	case len(in.Items.Tasks) > 0:
		parts = append(parts, fmt.Sprintf("Hari ini %s mengerjakan %d tugas tanpa ada yang ditutup.", in.EmployeeName, len(in.Items.Tasks)))
	}
	if len(open) > 0 {
		parts = append(parts, fmt.Sprintf("Masih berjalan: %s.", listTitles(open, 3)))
	}
	for _, l := range in.Items.Logbook {
		parts = append(parts, fmt.Sprintf("Logbook %s terisi %s%% (status %s).", l.Title, domain.FormatDecimal(l.Completion, 0), l.Status))
	}
	if n := strings.TrimSpace(in.Notes); n != "" {
		parts = append(parts, "Catatan: "+n)
	}
	return strings.Join(parts, " ")
}

// MockMonthlyNarrative writes two short paragraphs from the metrics.
func MockMonthlyNarrative(in domain.MonthlyRecapInput) string {
	m := in.Metrics
	first := fmt.Sprintf("Pada %s, %s menyelesaikan %d dari %d tugas terjadwal (%s) dengan ketepatan waktu %s.",
		in.Period.Label(), in.SubjectName, m.TasksDone, m.TasksPlanned, pct(m.CompletionRate), pct(m.OnTimeRate))
	if m.KPIScore != nil {
		first += fmt.Sprintf(" Skor KPI tercatat %s.", domain.FormatDecimal(*m.KPIScore, 2))
	}
	if m.LogbookEntries > 0 {
		first += fmt.Sprintf(" Ada %d logbook dengan kepatuhan checklist %s.", m.LogbookEntries, pct(m.LogbookCompliance))
	}
	if a := m.Attendance; a != nil {
		first += fmt.Sprintf(" Kehadiran %d dari %d hari kerja, %d kali terlambat.", a.PresentDays, a.WorkingDays, a.LateDays)
	}
	if len(in.Accomplishments) > 0 {
		first += " Capaian utama: " + listTitles(in.Accomplishments, 3) + "."
	}
	var focus []string
	if m.TasksOverdue > 0 {
		focus = append(focus, fmt.Sprintf("%d tugas masih terlambat", m.TasksOverdue))
	}
	if m.OnTimeRate != nil && *m.OnTimeRate < 0.8 {
		focus = append(focus, "ketepatan waktu penyelesaian di bawah 80%")
	}
	if m.LogbookCompliance != nil && *m.LogbookCompliance < 0.9 {
		focus = append(focus, "checklist logbook belum lengkap")
	}
	if m.ImprovementsOpen > 0 {
		focus = append(focus, fmt.Sprintf("%d item perbaikan masih terbuka", m.ImprovementsOpen))
	}
	second := "Bulan depan, pertahankan ritme kerja dan dokumentasikan praktik yang berhasil."
	if len(focus) > 0 {
		second = "Perlu perhatian bulan depan: " + strings.Join(focus, ", ") + "."
	}
	return first + "\n\n" + second
}

// MockSuggestions derives up to five improvements from the metrics.
func MockSuggestions(in domain.MonthlyRecapInput) []domain.ImprovementSuggestion {
	m := in.Metrics
	var out []domain.ImprovementSuggestion
	add := func(title, desc string) {
		if len(out) < domain.MaxSuggestions {
			out = append(out, domain.ImprovementSuggestion{Title: title, Description: desc})
		}
	}
	if m.TasksOverdue > 0 {
		add(fmt.Sprintf("Tuntaskan %d tugas yang terlambat", m.TasksOverdue),
			"Masih ada tugas lewat jatuh tempo. Jadwalkan penyelesaiannya di minggu pertama dan kabarkan kendala ke atasan.")
	}
	if m.OnTimeRate != nil && *m.OnTimeRate < 0.8 {
		add("Tingkatkan ketepatan waktu penyelesaian tugas",
			fmt.Sprintf("Hanya %s tugas selesai tepat waktu. Pecah tugas besar menjadi sub-task dan cek progres setiap pagi.", pct(m.OnTimeRate)))
	}
	if m.CompletionRate != nil && *m.CompletionRate < 0.85 {
		add("Naikkan rasio tugas yang disetujui",
			fmt.Sprintf("Rasio penyelesaian %s. Tandai selesai lebih awal agar sempat direview sebelum akhir bulan.", pct(m.CompletionRate)))
	}
	if m.LogbookCompliance != nil && *m.LogbookCompliance < 0.9 {
		add("Lengkapi checklist logbook harian",
			fmt.Sprintf("Kepatuhan logbook %s. Isi checklist sebelum pulang dan kirim di hari yang sama.", pct(m.LogbookCompliance)))
	}
	if a := m.Attendance; a != nil && a.LateDays > 2 {
		add("Kurangi keterlambatan hadir", fmt.Sprintf("Tercatat %d kali terlambat bulan ini. Atur ulang jadwal berangkat.", a.LateDays))
	}
	if m.KPIScore != nil && *m.KPIScore < 75 {
		add("Susun rencana peningkatan KPI bersama atasan",
			fmt.Sprintf("Skor KPI %s. Pilih dua indikator dengan capaian terendah dan sepakati target mingguan.", domain.FormatDecimal(*m.KPIScore, 2)))
	}
	if len(out) == 0 {
		add("Dokumentasikan praktik terbaik bulan ini", "Kinerja bulan ini sehat. Tulis satu praktik yang berhasil agar bisa dipakai tim.")
	}
	return out
}
