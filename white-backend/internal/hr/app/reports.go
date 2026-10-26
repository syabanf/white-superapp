package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// AttendanceReportMeta heads the "Rekap Absensi" workbook.
type AttendanceReportMeta struct {
	CompanyName, PeriodLabel, EmployeeLabel string
	GeneratedAt                             time.Time
}

// Documents renders the files hr hands out; adapter/docs implements it.
type Documents interface {
	AttendanceXLSX(rows []hris.ReportRow, meta AttendanceReportMeta) ([]byte, error)
	PayslipPDF(slip domain.Payslip, loans []payroll.LoanInstallmentDetail) ([]byte, error)
}

// Reports is the monthly HRIS recap (BuildHrisReport) and its CSV.
type Reports struct{ d Deps }

func NewReports(d Deps) *Reports { return &Reports{d: d} }

func (r *Reports) Monthly(ctx context.Context, a Actor, month, year int) (*hris.HrisReport, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, err
	}
	now := hris.NowWIB(r.d.now())
	if month == 0 {
		month = int(now.Month())
	}
	if year == 0 {
		year = now.Year()
	}
	if month < 1 || month > 12 {
		return nil, invalid("month", "Bulan tidak valid.")
	}
	emps, _, err := r.d.Store.ListEmployees(ctx, domain.EmployeeFilter{IncludeInactive: true})
	if err != nil {
		return nil, err
	}
	depts, err := r.d.Store.ListDepartments(ctx, true)
	if err != nil {
		return nil, err
	}
	start, end := payroll.PeriodBounds(month, year)
	atts, _, err := r.d.Store.ListAttendance(ctx, domain.AttendanceFilter{From: start, To: end})
	if err != nil {
		return nil, err
	}
	leaves, _, err := r.d.Store.ListLeaves(ctx, domain.LeaveFilter{From: start, To: end, Limit: 10000})
	if err != nil {
		return nil, err
	}
	re := make([]hris.ReportEmployee, len(emps))
	for i, e := range emps {
		re[i] = hris.ReportEmployee{EmploymentStatus: e.EmploymentStatus, IsActive: e.IsActive, DepartmentID: e.DepartmentID, JoinDate: e.JoinDate, EndDate: e.EndDate}
	}
	rd := make([]hris.ReportDepartment, len(depts))
	for i, d := range depts {
		rd[i] = hris.ReportDepartment{ID: d.ID, Name: d.Name}
	}
	ra := make([]hris.ReportAttendance, len(atts))
	for i, at := range atts {
		status := at.Status
		if status == "late" || status == "remote" || status == "half_day" {
			status = "present"
		}
		ra[i] = hris.ReportAttendance{Status: status, IsLate: at.IsLate, WorkHours: at.WorkHours, Date: at.Date}
	}
	rl := make([]hris.ReportLeave, 0, len(leaves))
	for _, l := range leaves {
		if l.StartDate < start || l.StartDate > end {
			continue
		}
		status := l.Status
		if status == hris.LeaveManagerApproved {
			status = hris.LeavePending
		}
		rl = append(rl, hris.ReportLeave{LeaveType: l.LeaveType, Status: status, TotalDays: float64(l.TotalDays)})
	}
	rep := hris.BuildHrisReport(re, rd, ra, rl, month, year)
	return &rep, nil
}

// MonthlyCSV is the report as one CSV with a section per block.
func (r *Reports) MonthlyCSV(ctx context.Context, a Actor, month, year int) ([]byte, string, error) {
	rep, err := r.Monthly(ctx, a, month, year)
	if err != nil {
		return nil, "", err
	}
	var b strings.Builder
	row := func(cells ...any) {
		for i, c := range cells {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(hris.CSVCell(fmt.Sprint(c)))
		}
		b.WriteByte('\n')
	}
	h, at, lv := rep.Headcount, rep.Attendance, rep.Leaves
	row("Laporan SDM", payroll.PeriodLabelID(&rep.Period.Month, rep.Period.Year))
	row()
	row("Karyawan aktif", h.TotalActive)
	row("Karyawan baru", h.NewHires)
	row("Keluar (tahun berjalan)", h.TurnoverCount)
	row("Tingkat turnover (%)", h.TurnoverRate)
	row()
	row("Status kepegawaian", "Jumlah")
	for _, s := range h.ByStatus {
		row(s.Status, s.Count)
	}
	row()
	row("Departemen", "Jumlah")
	for _, d := range h.ByDepartment {
		row(d.Name, d.Count)
	}
	row()
	row("Bulan", "Headcount")
	for _, m := range h.MonthlyTrend {
		row(m.Month, m.Count)
	}
	row()
	row("Catatan absensi", at.TotalRecords)
	row("Hadir", at.PresentCount)
	row("Tidak hadir", at.AbsentCount)
	row("Terlambat", at.LateCount)
	row("Tingkat kehadiran (%)", at.PresentRate)
	row("Tingkat keterlambatan (%)", at.LateRate)
	row("Rata-rata jam kerja", at.AvgWorkHours)
	row()
	row("Tanggal", "Hadir", "Tidak hadir", "Terlambat")
	for _, d := range at.DailyTrend {
		row(d.Date, d.Present, d.Absent, d.Late)
	}
	row()
	row("Cuti disetujui", lv.ApprovedCount)
	row("Cuti menunggu", lv.PendingCount)
	row("Total hari cuti", lv.TotalDays)
	row("Jenis cuti", "Hari")
	for _, t := range lv.ByType {
		row(t.Type, t.Days)
	}
	name := fmt.Sprintf("laporan-sdm-%d-%02d.csv", rep.Period.Year, rep.Period.Month)
	return []byte(b.String()), name, nil
}
