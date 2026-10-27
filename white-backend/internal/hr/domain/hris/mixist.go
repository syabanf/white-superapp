package hris

import (
	"math"
	"time"
)

// Rules Mixist adds on top of the NüHabit port: the two-step leave
// approval, work hours at clock-out (the calculate_work_hours trigger moved
// into Go), reminder windows for the daily tick and birthdays.

// Leave statuses. NüHabit decided in one step (HR or the manager);
// Mixist asks the direct manager first, then HR.
const (
	LeavePending         = "pending"
	LeaveManagerApproved = "manager_approved"
	LeaveApproved        = "approved"
	LeaveRejected        = "rejected"
	LeaveCancelled       = "cancelled"
)

// LeaveActor is who acts on a leave request.
type LeaveActor struct {
	IsHR      bool // holds hr.approve
	IsManager bool // the employee's direct manager
	IsOwner   bool
}

// DecideLeave returns the status a decision moves the request to, or a
// user-facing reason when the actor may not decide. A manager approval on a
// pending request waits for HR, unless the manager also holds hr.approve;
// HR may approve at either step (an employee without a manager goes
// straight to HR). Nobody decides their own request.
func DecideLeave(current string, a LeaveActor, approve bool) (string, string) {
	if current != LeavePending && current != LeaveManagerApproved {
		return "", "Pengajuan cuti sudah diproses"
	}
	if a.IsOwner {
		return "", "Tidak bisa memutuskan pengajuan cuti sendiri"
	}
	managerStep := current == LeavePending && a.IsManager
	if !a.IsHR && !managerStep {
		if current == LeaveManagerApproved && a.IsManager {
			return "", "Sudah disetujui atasan, menunggu persetujuan HRD"
		}
		return "", "Hanya HRD atau atasan langsung yang bisa memproses pengajuan ini"
	}
	if !approve {
		return LeaveRejected, ""
	}
	if a.IsHR {
		return LeaveApproved, ""
	}
	return LeaveManagerApproved, ""
}

// CanCancelLeave: the owner while it waits for a decision, HR also once
// approved (the annual quota is refunded).
func CanCancelLeave(current string, a LeaveActor) (bool, string) {
	switch current {
	case LeavePending, LeaveManagerApproved:
		if a.IsOwner || a.IsHR {
			return true, ""
		}
	case LeaveApproved:
		if a.IsHR {
			return true, ""
		}
		return false, "Cuti yang sudah disetujui hanya bisa dibatalkan oleh HRD/admin"
	}
	return false, "Pengajuan ini tidak bisa dibatalkan"
}

// WorkHours is public.calculate_work_hours: hours between clock-in and
// clock-out minus the break (60 minutes when unset), two decimals, never
// negative.
func WorkHours(clockIn, clockOut time.Time, breakMinutes *int) float64 {
	brk := 60
	if breakMinutes != nil {
		brk = *breakMinutes
	}
	h := math.Round((clockOut.Sub(clockIn).Hours()-float64(brk)/60)*100) / 100
	return math.Max(0, h)
}

// ContractReminderDays are the days before a contract ends on which HR is
// reminded.
var ContractReminderDays = []int{30, 14, 7}

// ProbationReminderDays are the days before a probation ends on which HR
// is reminded.
var ProbationReminderDays = []int{14, 7, 0}

// DaysUntil is the whole days from today to date (negative when past);
// ok is false for an unreadable date.
func DaysUntil(today, date string) (int, bool) {
	t, ok1 := ParseDate(today)
	d, ok2 := ParseDate(date)
	if !ok1 || !ok2 {
		return 0, false
	}
	return int(d.Sub(t).Hours() / 24), true
}

// ReminderDue reports whether date is exactly one of the reminder offsets
// away from today, and which.
func ReminderDue(today, date string, offsets []int) (int, bool) {
	n, ok := DaysUntil(today, date)
	if !ok {
		return 0, false
	}
	for _, o := range offsets {
		if n == o {
			return n, true
		}
	}
	return 0, false
}

// IsBirthday: same month and day; 29 February birthdays fall on 28
// February in common years.
func IsBirthday(birthDate, today string) bool {
	b, ok1 := ParseDate(birthDate)
	t, ok2 := ParseDate(today)
	if !ok1 || !ok2 {
		return false
	}
	if b.Month() == time.February && b.Day() == 29 && !isLeap(t.Year()) {
		return t.Month() == time.February && t.Day() == 28
	}
	return b.Month() == t.Month() && b.Day() == t.Day()
}

func isLeap(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }
