package app

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// Leaves is leave requests and balances (lib/hris/leaves-repo,
// leave-balances-repo, leaves-export) with Mixist's manager-then-HR
// approval.
type Leaves struct{ d Deps }

func NewLeaves(d Deps) *Leaves { return &Leaves{d: d} }

// LeaveTypes in display order; Annual marks the types that consume the
// annual quota.
var LeaveTypes = []struct {
	Code, Name string
	Annual     bool
}{
	{"annual", "Cuti Tahunan", true}, {"sick", "Sakit", false}, {"maternity", "Cuti Melahirkan", false},
	{"paternity", "Cuti Ayah", false}, {"unpaid", "Izin Tanpa Gaji", false}, {"emergency", "Izin Darurat", false},
	{"pilgrimage", "Cuti Ibadah", false}, {"menstrual", "Cuti Haid", false}, {"marriage", "Cuti Menikah", false},
	{"bereavement", "Cuti Duka", false},
}

func isLeaveType(code string) bool {
	for _, t := range LeaveTypes {
		if t.Code == code {
			return true
		}
	}
	return false
}

// NotifLeave* are the notification types sent through identity.
const (
	NotifLeaveRequested = "HR_LEAVE_REQUESTED"
	NotifLeaveDecided   = "HR_LEAVE_DECIDED"
)

type LeaveInput struct {
	EmployeeID, LeaveType, StartDate, EndDate, Reason string
}

// Create is createLeave: weekends and national holidays do not consume
// leave (cuti bersama does); a range without a working day is refused.
func (l *Leaves) Create(ctx context.Context, a Actor, in LeaveInput) (*domain.Leave, error) {
	employeeID, err := employeeOrSelf(a, in.EmployeeID, authn.PermHRWrite)
	if err != nil {
		return nil, err
	}
	if !isLeaveType(in.LeaveType) {
		return nil, invalid("leaveType", "Jenis cuti tidak valid.")
	}
	start, err := reqDate("startDate", in.StartDate)
	if err != nil {
		return nil, err
	}
	end, err := reqDate("endDate", in.EndDate)
	if err != nil {
		return nil, err
	}
	if end < start {
		return nil, invalid("endDate", "Tanggal selesai tidak boleh sebelum tanggal mulai.")
	}
	if strings.TrimSpace(in.Reason) == "" {
		return nil, invalid("reason", "Alasan wajib diisi.")
	}
	emp, err := l.d.Store.GetEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if emp == nil || !emp.IsActive {
		return nil, notFound("Karyawan tidak ditemukan atau sudah tidak aktif.")
	}
	days, excluded, err := l.workingDays(ctx, start, end)
	if err != nil {
		return nil, err
	}
	if days == 0 {
		return nil, invalid("startDate", hris.NoWorkingDayMessage(excluded))
	}
	overlap, err := l.d.Store.LeaveOverlaps(ctx, employeeID, start, end)
	if err != nil {
		return nil, err
	}
	if overlap {
		return nil, conflict("Sudah ada pengajuan cuti di rentang tanggal ini.")
	}
	if in.LeaveType == "annual" {
		year := yearOf(start)
		bal, err := l.ensureBalance(ctx, l.d.Store, *emp, year, a.UserRef())
		if err != nil {
			return nil, err
		}
		pending, err := l.d.Store.PendingAnnualDays(ctx, employeeID, year)
		if err != nil {
			return nil, err
		}
		if left := bal.AnnualRemaining() - pending; left < days {
			return nil, invalid("startDate", fmt.Sprintf("Sisa cuti tahunan tidak cukup (sisa %d hari, diajukan %d hari).", max(left, 0), days))
		}
	}
	leave := domain.Leave{ID: newID(), EmployeeID: employeeID, LeaveType: in.LeaveType, StartDate: start, EndDate: end,
		TotalDays: days, Reason: strings.TrimSpace(in.Reason), Status: hris.LeavePending, CreatedBy: a.UserRef()}
	if err := l.d.Store.InsertLeave(ctx, leave); err != nil {
		return nil, err
	}
	if emp.ManagerID != nil {
		l.notifyEmployee(ctx, *emp.ManagerID, NotifLeaveRequested, "Pengajuan "+hris.LeaveTypeLabel(in.LeaveType)+": "+emp.FullName,
			fmt.Sprintf("%s s.d. %s (%d hari kerja). %s", start, end, days, leave.Reason))
	}
	return l.d.Store.GetLeave(ctx, leave.ID)
}

// workingDays counts the days that consume leave and the holidays skipped.
func (l *Leaves) workingDays(ctx context.Context, start, end string) (int, []hris.ExcludedHoliday, error) {
	hs, err := l.d.Store.ListHolidays(ctx, start, end, true)
	if err != nil {
		return 0, nil, err
	}
	rows := make([]hris.Holiday, len(hs))
	for i, h := range hs {
		rows[i] = hris.Holiday{Date: h.Date, Name: h.Name, Type: h.Type, DeductsLeave: h.DeductsLeave}
	}
	days, excluded := hris.DescribeLeaveDays(start, end, hris.IndexHolidays(rows))
	return days, excluded, nil
}

// Excluded lists the holidays inside a leave that did not consume it.
func (l *Leaves) Excluded(ctx context.Context, leave domain.Leave) []string {
	_, excluded, err := l.workingDays(ctx, leave.StartDate, leave.EndDate)
	if err != nil {
		return nil
	}
	out := make([]string, len(excluded))
	for i, h := range excluded {
		out[i] = h.Date + " " + h.Name
	}
	return out
}

func (l *Leaves) Get(ctx context.Context, a Actor, id string) (*domain.Leave, error) {
	leave, err := l.d.Store.GetLeave(ctx, id)
	if err != nil {
		return nil, err
	}
	if leave == nil {
		return nil, notFound("Pengajuan cuti tidak ditemukan.")
	}
	f := l.actorOf(a, *leave)
	if !f.IsHR && !f.IsOwner && !f.IsManager && !a.Can(authn.PermHRRead) {
		return nil, require(a, authn.PermHRRead)
	}
	return leave, nil
}

type LeaveQuery struct {
	domain.LeaveFilter
	EmployeeID string
	Scope      string
}

// List: hr.read sees everyone; scope=approvals lists the caller's direct
// reports' requests waiting for them; otherwise own requests.
func (l *Leaves) List(ctx context.Context, a Actor, q LeaveQuery) ([]domain.Leave, int, error) {
	f := q.LeaveFilter
	switch {
	case q.Scope == "approvals":
		ids, err := l.reportIDs(ctx, a)
		if err != nil || len(ids) == 0 {
			return nil, 0, err
		}
		f.EmployeeIDs = ids
		if f.Status == "" {
			f.Statuses = []string{hris.LeavePending}
		}
	case a.Can(authn.PermHRRead):
		if q.EmployeeID != "" {
			f.EmployeeIDs = []string{q.EmployeeID}
		}
	default:
		id, err := self(a)
		if err != nil {
			return nil, 0, err
		}
		f.EmployeeIDs = []string{id}
	}
	return l.d.Store.ListLeaves(ctx, f)
}

func (l *Leaves) reportIDs(ctx context.Context, a Actor) ([]string, error) {
	if a.EmployeeID == "" {
		return nil, nil
	}
	reports, _, err := l.d.Store.ListEmployees(ctx, domain.EmployeeFilter{ManagerID: a.EmployeeID})
	return employeeIDs(reports), err
}

func (l *Leaves) actorOf(a Actor, leave domain.Leave) hris.LeaveActor {
	return hris.LeaveActor{
		IsHR:      a.Can(authn.PermHRApprove),
		IsManager: a.EmployeeID != "" && deref(leave.EmployeeManagerID) == a.EmployeeID,
		IsOwner:   a.EmployeeID != "" && leave.EmployeeID == a.EmployeeID,
	}
}

// Flags are what the caller may do with a request now.
func (l *Leaves) Flags(a Actor, leave domain.Leave) (canDecide, canCancel bool) {
	f := l.actorOf(a, leave)
	next, _ := hris.DecideLeave(leave.Status, f, true)
	canCancel, _ = hris.CanCancelLeave(leave.Status, f)
	return next != "", canCancel
}

// Decide approves or rejects: the direct manager first, then hr.approve;
// the final approval consumes the quota in the same transaction.
func (l *Leaves) Decide(ctx context.Context, a Actor, id string, approve bool, reason string) (*domain.Leave, error) {
	leave, err := l.d.Store.GetLeave(ctx, id)
	if err != nil {
		return nil, err
	}
	if leave == nil {
		return nil, notFound("Pengajuan cuti tidak ditemukan.")
	}
	reason = strings.TrimSpace(reason)
	if !approve && reason == "" {
		return nil, invalid("rejectionReason", "Alasan penolakan wajib diisi.")
	}
	next, why := hris.DecideLeave(leave.Status, l.actorOf(a, *leave), approve)
	if next == "" {
		return nil, forbidden(why)
	}
	u := domain.LeaveStatusUpdate{ID: id, From: leave.Status, To: next, Actor: a.UserRef()}
	switch next {
	case hris.LeaveManagerApproved:
		u.ManagerApprovedBy = a.EmployeeRef()
	case hris.LeaveApproved:
		u.ApprovedBy = a.EmployeeRef()
		if leave.Status == hris.LeavePending && deref(leave.EmployeeManagerID) == a.EmployeeID {
			u.ManagerApprovedBy = a.EmployeeRef()
		}
	case hris.LeaveRejected:
		u.RejectedBy, u.RejectionReason = a.EmployeeRef(), reason
	}
	emp, err := l.d.Store.GetEmployee(ctx, leave.EmployeeID)
	if err != nil || emp == nil {
		return nil, err
	}
	err = l.d.Store.WithTx(ctx, func(tx domain.Store) error {
		ok, err := tx.SetLeaveStatus(ctx, u)
		if err != nil {
			return err
		}
		if !ok {
			return conflict("Pengajuan sudah diproses oleh orang lain.")
		}
		if next != hris.LeaveApproved {
			return nil
		}
		year := yearOf(leave.StartDate)
		if _, err := l.ensureBalance(ctx, tx, *emp, year, a.UserRef()); err != nil {
			return err
		}
		return tx.AddLeaveUsage(ctx, leave.EmployeeID, year, usageOf(leave.LeaveType, leave.TotalDays))
	})
	if err != nil {
		return nil, err
	}
	l.notifyDecision(ctx, *emp, *leave, next, reason)
	return l.d.Store.GetLeave(ctx, id)
}

// Cancel: the owner while it waits, HR also once approved (quota refunded).
func (l *Leaves) Cancel(ctx context.Context, a Actor, id string) (*domain.Leave, error) {
	leave, err := l.d.Store.GetLeave(ctx, id)
	if err != nil {
		return nil, err
	}
	if leave == nil {
		return nil, notFound("Pengajuan cuti tidak ditemukan.")
	}
	if ok, why := hris.CanCancelLeave(leave.Status, l.actorOf(a, *leave)); !ok {
		return nil, forbidden(why)
	}
	err = l.d.Store.WithTx(ctx, func(tx domain.Store) error {
		ok, err := tx.SetLeaveStatus(ctx, domain.LeaveStatusUpdate{ID: id, From: leave.Status, To: hris.LeaveCancelled, Actor: a.UserRef()})
		if err != nil {
			return err
		}
		if !ok {
			return conflict("Pengajuan sudah diproses oleh orang lain.")
		}
		if leave.Status != hris.LeaveApproved {
			return nil
		}
		u := usageOf(leave.LeaveType, leave.TotalDays)
		return tx.AddLeaveUsage(ctx, leave.EmployeeID, yearOf(leave.StartDate), domain.LeaveUsage{Annual: -u.Annual, Sick: -u.Sick, Unpaid: -u.Unpaid, Other: -u.Other})
	})
	if err != nil {
		return nil, err
	}
	return l.d.Store.GetLeave(ctx, id)
}

func usageOf(leaveType string, days int) domain.LeaveUsage {
	switch leaveType {
	case "annual":
		return domain.LeaveUsage{Annual: days}
	case "sick":
		return domain.LeaveUsage{Sick: days}
	case "unpaid":
		return domain.LeaveUsage{Unpaid: days}
	}
	return domain.LeaveUsage{Other: days}
}

func yearOf(date string) int {
	if t, ok := hris.ParseDate(date); ok {
		return t.Year()
	}
	return 0
}

// ─── Balances ────────────────────────────────────────────────

type Balance struct {
	domain.LeaveBalance
	PendingDays int
}

// Balance is getLeaveBalance: the owner or hr.read; the year's row is
// created with the prorated annual quota on first read.
func (l *Leaves) Balance(ctx context.Context, a Actor, employeeID string, year int) (*Balance, error) {
	employeeID, err := employeeOrSelf(a, employeeID, authn.PermHRRead)
	if err != nil {
		return nil, err
	}
	if year == 0 {
		year = l.d.now().Year()
	}
	emp, err := l.d.Store.GetEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, notFound(msgEmployeeNotFound)
	}
	b, err := l.ensureBalance(ctx, l.d.Store, *emp, year, a.UserRef())
	if err != nil {
		return nil, err
	}
	pending, err := l.d.Store.PendingAnnualDays(ctx, employeeID, year)
	if err != nil {
		return nil, err
	}
	return &Balance{LeaveBalance: *b, PendingDays: pending}, nil
}

// ensureBalance returns the year's balance row, created with the prorated
// annual quota on first read.
func (l *Leaves) ensureBalance(ctx context.Context, st domain.Store, emp domain.Employee, year int, actor *string) (*domain.LeaveBalance, error) {
	b, err := st.GetLeaveBalance(ctx, emp.ID, year)
	if err != nil || b != nil {
		return b, err
	}
	join := emp.JoinDate
	return st.EnsureLeaveBalance(ctx, domain.LeaveBalance{ID: newID(), EmployeeID: emp.ID, Year: year,
		AnnualTotal: hris.ProratedAnnualQuota(&join, year)}, actor)
}

func (l *Leaves) UpdateBalance(ctx context.Context, a Actor, employeeID string, year, total int, used *int) (*Balance, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	if total < 0 || total > 365 || (used != nil && *used < 0) {
		return nil, invalid("annualLeaveTotal", "Jumlah hari cuti tidak valid.")
	}
	b, err := l.Balance(ctx, a, employeeID, year)
	if err != nil {
		return nil, err
	}
	u := b.AnnualUsed
	if used != nil {
		u = *used
	}
	nb, err := l.d.Store.SetLeaveBalance(ctx, employeeID, b.Year, total, u, a.UserRef())
	if err != nil {
		return nil, err
	}
	return &Balance{LeaveBalance: *nb, PendingDays: b.PendingDays}, nil
}

// ExportCSV is the HR leave export (formula-injection safe).
func (l *Leaves) ExportCSV(ctx context.Context, a Actor, q LeaveQuery) ([]byte, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, err
	}
	q.Limit, q.Offset = 10000, 0
	rows, _, err := l.List(ctx, a, q)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, notFound("Tidak ada data cuti untuk diekspor.")
	}
	out := make([]hris.LeaveCSVRow, len(rows))
	for i, r := range rows {
		total := fmt.Sprint(r.TotalDays)
		row := hris.LeaveCSVRow{ID: r.ID, LeaveType: r.LeaveType, Status: r.Status, StartDate: r.StartDate, EndDate: r.EndDate,
			TotalDays: &total, Reason: strPtr(r.Reason), RejectionReason: strPtr(r.RejectionReason), ApprovedAt: r.ApprovedAt,
			CreatedAt: r.CreatedAt, EmployeeName: strPtr(r.EmployeeName), EmployeeNip: strPtr(r.EmployeeNo), Department: strPtr(r.DepartmentName)}
		if r.ApprovedByName != "" {
			row.HasApprover, row.ApproverName = true, strPtr(r.ApprovedByName)
		}
		out[i] = row
	}
	return []byte(hris.LeavesCSV(out)), nil
}

// ─── notifications ───────────────────────────────────────────

// notifyEmployee notifies the user linked to an employee, best effort.
func (l *Leaves) notifyEmployee(ctx context.Context, employeeID, typ, title, body string) {
	notifyEmployee(ctx, l.d, employeeID, NotifyInput{Type: typ, Title: title, Body: body, Href: "/erp/hr/leaves"})
}

func (l *Leaves) notifyDecision(ctx context.Context, emp domain.Employee, leave domain.Leave, status, reason string) {
	label := hris.LeaveTypeLabel(leave.LeaveType)
	rangeLabel := fmt.Sprintf("%s s.d. %s (%d hari)", leave.StartDate, leave.EndDate, leave.TotalDays)
	switch status {
	case hris.LeaveManagerApproved:
		notifyHR(ctx, l.d, NotifyInput{Type: NotifLeaveRequested, Title: label + " menunggu persetujuan HRD: " + emp.FullName, Body: rangeLabel, Href: "/erp/hr/leaves"})
		l.notifyEmployee(ctx, emp.ID, NotifLeaveDecided, label+" disetujui atasan", rangeLabel+". Menunggu persetujuan HRD.")
	case hris.LeaveApproved:
		l.notifyEmployee(ctx, emp.ID, NotifLeaveDecided, label+" disetujui", rangeLabel+". Selamat beristirahat!")
	case hris.LeaveRejected:
		l.notifyEmployee(ctx, emp.ID, NotifLeaveDecided, label+" ditolak", rangeLabel+". Alasan: "+reason)
	}
}

// HRDepartmentCode is the department whose linked users receive HR
// notifications, together with every hr.approve holder from identity.
const HRDepartmentCode = "HR"

func notifyEmployee(ctx context.Context, d Deps, employeeID string, in NotifyInput) {
	e, err := d.Store.GetEmployee(ctx, employeeID)
	if err != nil || e == nil || e.UserID == nil {
		return
	}
	in.UserIDs = []string{*e.UserID}
	if err := d.Identity.Notify(ctx, in); err != nil {
		d.Logger.Warn("notify failed", "type", in.Type, "err", err)
	}
}

// notifyHR reaches the HR department's linked users and the hr.approve
// holders; either lookup failing leaves the other.
func notifyHR(ctx context.Context, d Deps, in NotifyInput) {
	ids, err := d.Store.LinkedUserIDsInDepartment(ctx, HRDepartmentCode)
	if err != nil {
		d.Logger.Warn("hr department users lookup failed", "err", err)
	}
	holders, err := d.Identity.PermissionHolders(ctx, authn.PermHRApprove)
	if err != nil {
		d.Logger.Warn("hr approvers lookup failed", "err", err)
	}
	for _, id := range holders {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	in.UserIDs = ids
	if err := d.Identity.Notify(ctx, in); err != nil {
		d.Logger.Warn("notify failed", "type", in.Type, "err", err)
	}
}
