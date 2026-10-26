package hris

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

/* ── Overtime (lib/hris/overtime-rules, payroll/period) ───────────────── */

// OvertimeActor is who decides.
type OvertimeActor struct {
	EmployeeID *string
	IsHR       bool
}

// OvertimeRequest is the request being decided.
type OvertimeRequest struct {
	EmployeeID  string
	RequestedBy *string
	Source      string // "employee" | "company"
	ReportingTo *string
}

func samePerson(actor *string, other *string) bool {
	return actor != nil && other != nil && *actor == *other
}

// CanDecideOvertime: an employee request is decided by HR or the direct
// manager but never by the requester; a company assignment is confirmed
// only by the assigned employee; cancel by the requester (or HR for a
// company assignment). "" reason means allowed.
func CanDecideOvertime(actor OvertimeActor, req OvertimeRequest, action string) (bool, string) {
	target := req.EmployeeID
	isTarget := samePerson(actor.EmployeeID, &target)
	isRequester := samePerson(actor.EmployeeID, req.RequestedBy)
	isManager := samePerson(actor.EmployeeID, req.ReportingTo)

	if action == "cancel" {
		if isRequester || (req.Source == "company" && actor.IsHR) {
			return true, ""
		}
		return false, "Hanya pembuat pengajuan yang bisa membatalkan"
	}
	if req.Source == "company" {
		if isTarget {
			return true, ""
		}
		return false, "Hanya karyawan yang ditugaskan yang bisa mengonfirmasi penugasan ini"
	}
	if isTarget {
		return false, "Tidak bisa memutuskan pengajuan lembur sendiri"
	}
	if !actor.IsHR && !isManager {
		return false, "Hanya HRD/atasan langsung yang bisa memproses pengajuan ini"
	}
	return true, ""
}

// OvertimeHoursFromTimes: HH:MM to HH:MM, past midnight when the end is not
// after the start, rounded to two decimals.
func OvertimeHoursFromTimes(start, end string) float64 {
	part := func(s string, i int) int {
		p := strings.Split(s, ":")
		if i >= len(p) {
			return 0
		}
		n, _ := strconv.Atoi(p[i])
		return n
	}
	minutes := part(end, 0)*60 + part(end, 1) - (part(start, 0)*60 + part(start, 1))
	if minutes <= 0 {
		minutes += 24 * 60
	}
	return Round2(float64(minutes) / 60)
}

// PeriodBounds is the first and last day of a month.
func PeriodBounds(month, year int) (string, string) {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	return first.Format(DateLayout), first.AddDate(0, 1, -1).Format(DateLayout)
}

/* ── Leave (lib/hris/leave-balances-repo, leave-wa, recruitment/wa) ────── */

// ProratedAnnualQuota: 12 days, or 12 × remaining months / 12 (join month
// included, floored) for an employee who joined in that year.
func ProratedAnnualQuota(joinDate *string, year int) int {
	if joinDate == nil || *joinDate == "" {
		return 12
	}
	t, ok := parseContractDate(*joinDate)
	if !ok || t.Year() != year {
		return 12
	}
	return 13 - int(t.Month()) // months left in the year, join month included
}

var leaveTypeLabels = map[string]string{
	"annual": "Cuti Tahunan", "sick": "Sakit", "maternity": "Cuti Melahirkan", "paternity": "Cuti Ayah",
	"unpaid": "Izin Tanpa Gaji", "emergency": "Izin Darurat", "pilgrimage": "Cuti Ibadah",
	"menstrual": "Cuti Haid", "marriage": "Cuti Menikah", "bereavement": "Cuti Duka",
}

// LeaveTypeLabel names a leave type for the WhatsApp message.
func LeaveTypeLabel(t string) string {
	if l, ok := leaveTypeLabels[t]; ok {
		return l
	}
	return t
}

var (
	idWeekdays = []string{"Min", "Sen", "Sel", "Rab", "Kam", "Jum", "Sab"}
	idMonths   = []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
)

// tanggalID is toLocaleDateString("id-ID", weekday short, 2-digit day,
// short month, numeric year) of a WIB calendar date: "Sen, 05 Okt 2026".
func tanggalID(iso string) string {
	t, ok := ParseDate(iso)
	if !ok {
		return iso
	}
	return fmt.Sprintf("%s, %02d %s %d", idWeekdays[t.Weekday()], t.Day(), idMonths[t.Month()-1], t.Year())
}

/* ── Leave CSV export (lib/hris/leaves-export) ───────────────────────── */

var csvLeaveTypes = map[string]string{
	"annual": "Cuti Tahunan", "sick": "Cuti Sakit", "maternity": "Cuti Melahirkan", "paternity": "Cuti Ayah",
	"unpaid": "Cuti Tanpa Upah", "emergency": "Cuti Darurat", "pilgrimage": "Cuti Haji/Umrah", "menstrual": "Cuti Haid",
}

var csvLeaveStatuses = map[string]string{
	"pending": "Menunggu Persetujuan", "approved": "Disetujui", "rejected": "Ditolak", "cancelled": "Dibatalkan",
}

// LeaveCSVHeader is the export header line.
const LeaveCSVHeader = "ID Cuti,NIP,Nama Karyawan,Departemen,Jabatan,Jenis Cuti,Tanggal Mulai,Tanggal Selesai,Total Hari,Alasan,Status,Disetujui Oleh,Tanggal Disetujui,Alasan Penolakan,Tanggal Pengajuan"

// LeaveCSVRow is one exported leave.
type LeaveCSVRow struct {
	ID, LeaveType, Status, StartDate, EndDate string
	TotalDays                                 *string // numeric text; nil is 0
	Reason, RejectionReason                   *string
	ApprovedAt                                *time.Time
	CreatedAt                                 time.Time
	EmployeeName, EmployeeNip                 *string
	Department, JobTitle                      *string
	ApproverName                              *string
	ApproverNip                               *string
	HasApprover                               bool
}

// WIBDateTime is toLocaleString("id-ID", Asia/Jakarta): "4/10/2026, 10.00.00".
func WIBDateTime(t time.Time) string {
	w := t.UTC().Add(WIBOffset)
	return fmt.Sprintf("%d/%d/%d, %02d.%02d.%02d", w.Day(), int(w.Month()), w.Year(), w.Hour(), w.Minute(), w.Second())
}

// CSVCell quotes a cell; text starting with = + - @ tab or CR is prefixed
// with ' so spreadsheets do not run it as a formula. The "-" placeholder of
// an empty cell is left as is.
func CSVCell(text string) string {
	if text != "-" && text != "" && strings.ContainsRune("=+-@\t\r", rune(text[0])) {
		text = "'" + text
	}
	return `"` + strings.ReplaceAll(text, `"`, `""`) + `"`
}

func orDash(s *string) string {
	if s == nil || *s == "" {
		return "-"
	}
	return *s
}

// LeavesCSV renders the export.
func LeavesCSV(rows []LeaveCSVRow) string {
	lines := []string{LeaveCSVHeader}
	for _, r := range rows {
		typeLabel := r.LeaveType
		if l, ok := csvLeaveTypes[r.LeaveType]; ok {
			typeLabel = l
		}
		statusLabel := r.Status
		if l, ok := csvLeaveStatuses[r.Status]; ok {
			statusLabel = l
		}
		approver := "-"
		if r.HasApprover {
			approver = jsString(r.ApproverName) + " (" + jsString(r.ApproverNip) + ")"
		}
		approvedAt := "-"
		if r.ApprovedAt != nil {
			approvedAt = WIBDateTime(*r.ApprovedAt)
		}
		total := "0"
		if r.TotalDays != nil && *r.TotalDays != "" {
			total = *r.TotalDays
		}
		cells := []string{
			r.ID, orDash(r.EmployeeNip), orDash(r.EmployeeName), orDash(r.Department), orDash(r.JobTitle),
			typeLabel, r.StartDate, r.EndDate, total, orDash(r.Reason), statusLabel, approver, approvedAt,
			orDash(r.RejectionReason), WIBDateTime(r.CreatedAt),
		}
		for i, c := range cells {
			cells[i] = CSVCell(c)
		}
		lines = append(lines, strings.Join(cells, ","))
	}
	return strings.Join(lines, "\n")
}
