package app

import (
	"context"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// Overtime is the two-way overtime request (hris overtime.go): employees
// ask (decided by HR or the direct manager), HR assigns (confirmed by the
// employee). Hours come from HH:MM with an overnight wrap.
type Overtime struct{ d Deps }

func NewOvertime(d Deps) *Overtime { return &Overtime{d: d} }

const overtimeDuplicate = "Sudah ada pengajuan lembur pending/approved di tanggal tersebut."

type OvertimeQuery struct {
	EmployeeID, Status, Month, Scope, DepartmentID string
}

// List: scope=approvals lists the direct reports' own requests; hr.read
// sees everyone; otherwise own rows.
func (o *Overtime) List(ctx context.Context, a Actor, q OvertimeQuery) ([]domain.Overtime, error) {
	f := domain.OvertimeFilter{Status: q.Status, DepartmentID: q.DepartmentID}
	switch {
	case q.Scope == "approvals":
		if a.EmployeeID == "" {
			return nil, nil
		}
		reports, _, err := o.d.Store.ListEmployees(ctx, domain.EmployeeFilter{ManagerID: a.EmployeeID})
		if err != nil || len(reports) == 0 {
			return nil, err
		}
		f.EmployeeIDs, f.Source = employeeIDs(reports), "employee"
	case a.Can(authn.PermHRRead):
		if q.EmployeeID != "" {
			f.EmployeeIDs = []string{q.EmployeeID}
		}
	default:
		id, err := self(a)
		if err != nil {
			return nil, err
		}
		f.EmployeeIDs = []string{id}
	}
	if q.Month != "" {
		from, to, ok := monthBounds(q.Month)
		if !ok {
			return nil, invalid("month", "Bulan tidak valid (YYYY-MM).")
		}
		f.From, f.To = from, to
	}
	return o.d.Store.ListOvertime(ctx, f)
}

type OvertimeInput struct {
	EmployeeID, Date, StartTime, EndTime, Reason string
}

// Create files a request for oneself (source employee) or, with
// hr.write, an assignment for another employee (source company).
func (o *Overtime) Create(ctx context.Context, a Actor, in OvertimeInput) (*domain.Overtime, error) {
	target, err := employeeOrSelf(a, in.EmployeeID, authn.PermHRWrite)
	if err != nil {
		return nil, err
	}
	ot := domain.Overtime{ID: newID(), EmployeeID: target, Status: "pending", RequestedBy: a.EmployeeRef(), Source: "employee"}
	if err := setSlot(&ot, in); err != nil {
		return nil, err
	}
	emp, err := o.d.Store.GetEmployee(ctx, target)
	if err != nil {
		return nil, err
	}
	if emp == nil || !emp.IsActive {
		return nil, notFound("Karyawan tidak ditemukan/nonaktif.")
	}
	if target != a.EmployeeID {
		ot.Source = "company"
	}
	if err := o.d.Store.InsertOvertime(ctx, ot, a.UserRef()); err != nil {
		return nil, uniqueErr(err, overtimeDuplicate)
	}
	slot := ot.Date + " " + ot.StartTime + "-" + ot.EndTime
	if ot.Source == "employee" && emp.ManagerID != nil {
		notifyEmployee(ctx, o.d, *emp.ManagerID, NotifyInput{Type: "HR_OVERTIME_REQUESTED", Title: "Pengajuan lembur: " + emp.FullName,
			Body: slot, Href: "/erp/hr/overtime"})
	} else if ot.Source == "company" {
		notifyEmployee(ctx, o.d, target, NotifyInput{Type: "HR_OVERTIME_ASSIGNED", Title: "Penugasan lembur " + ot.Date,
			Body: ot.StartTime + "-" + ot.EndTime + ". Mohon konfirmasi.", Href: "/erp/me/overtime"})
	}
	return o.d.Store.GetOvertime(ctx, ot.ID)
}

// setSlot validates the date and HH:MM times and sets them, the hours
// (overnight wrap, at most 12) and the reason on ot.
func setSlot(ot *domain.Overtime, in OvertimeInput) error {
	date, err := reqDate("date", in.Date)
	if err != nil {
		return err
	}
	if !clockRe.MatchString(in.StartTime) || !clockRe.MatchString(in.EndTime) {
		return invalid("startTime", "Jam lembur tidak valid (HH:MM).")
	}
	start, end := domain.HHMM(in.StartTime), domain.HHMM(in.EndTime)
	if start == end {
		return invalid("endTime", "Jam mulai dan selesai tidak boleh sama.")
	}
	hours := hris.OvertimeHoursFromTimes(start, end)
	if hours > 12 {
		return invalid("endTime", "Durasi lembur maksimal 12 jam.")
	}
	ot.Date, ot.StartTime, ot.EndTime, ot.Hours, ot.Reason = date, start, end, hours, strings.TrimSpace(in.Reason)
	return nil
}

const msgOvertimeNotFound = "Pengajuan lembur tidak ditemukan."

// Update edits a pending entry (hr.write). A decided entry is history and
// an approved one already counts toward payroll, so only pending entries
// change; the employee, source and status stay.
func (o *Overtime) Update(ctx context.Context, a Actor, id string, in OvertimeInput) (*domain.Overtime, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	ot, err := o.d.Store.GetOvertime(ctx, id)
	if err != nil {
		return nil, err
	}
	if ot == nil {
		return nil, notFound(msgOvertimeNotFound)
	}
	if ot.Status != "pending" {
		return nil, conflict("Hanya lembur berstatus pending yang bisa diubah.")
	}
	if err := setSlot(ot, in); err != nil {
		return nil, err
	}
	ok, err := o.d.Store.UpdateOvertime(ctx, *ot)
	if err != nil {
		return nil, uniqueErr(err, overtimeDuplicate)
	}
	if !ok {
		return nil, conflict("Pengajuan sudah diproses oleh orang lain.")
	}
	return o.d.Store.GetOvertime(ctx, id)
}

// Delete removes an entry (hr.write) unless it is approved: approved hours
// are paid through payroll and must stay on record.
func (o *Overtime) Delete(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ot, err := o.d.Store.GetOvertime(ctx, id)
	if err != nil {
		return err
	}
	if ot == nil {
		return notFound(msgOvertimeNotFound)
	}
	if ot.Status == "approved" {
		return conflict("Lembur yang sudah disetujui tidak bisa dihapus karena masuk penggajian.")
	}
	ok, err := o.d.Store.DeleteOvertime(ctx, id)
	if err == nil && !ok {
		return conflict("Pengajuan sudah diproses oleh orang lain.")
	}
	return err
}

var (
	overtimeDecisionStatus = map[string]string{"approve": "approved", "reject": "rejected", "cancel": "cancelled"}
)

// Decide applies domain.CanDecideOvertime.
func (o *Overtime) Decide(ctx context.Context, a Actor, id, action, reason string) (*domain.Overtime, error) {
	status, ok := overtimeDecisionStatus[action]
	if !ok {
		return nil, invalid("action", "Aksi tidak valid.")
	}
	ot, err := o.d.Store.GetOvertime(ctx, id)
	if err != nil {
		return nil, err
	}
	if ot == nil {
		return nil, notFound(msgOvertimeNotFound)
	}
	if ot.Status != "pending" {
		return nil, conflict("Pengajuan sudah " + ot.Status + ".")
	}
	allowed, why := hris.CanDecideOvertime(hris.OvertimeActor{EmployeeID: a.EmployeeRef(), IsHR: a.Can(authn.PermHRApprove)},
		hris.OvertimeRequest{EmployeeID: ot.EmployeeID, RequestedBy: ot.RequestedBy, Source: ot.Source, ReportingTo: ot.EmployeeManagerID}, action)
	if !allowed {
		return nil, forbidden(why)
	}
	reason = strings.TrimSpace(reason)
	if action == "reject" && reason == "" {
		return nil, invalid("rejectionReason", "Alasan penolakan wajib diisi.")
	}
	if action != "reject" {
		reason = ""
	}
	done, err := o.d.Store.DecideOvertime(ctx, id, status, a.EmployeeRef(), reason)
	if err != nil {
		return nil, err
	}
	if !done {
		return nil, conflict("Pengajuan sudah diproses oleh orang lain.")
	}
	if action != "cancel" && ot.Source == "employee" {
		notifyEmployee(ctx, o.d, ot.EmployeeID, NotifyInput{Type: "HR_OVERTIME_DECIDED", Title: "Lembur " + ot.Date + " " + map[string]string{"approved": "disetujui", "rejected": "ditolak"}[status],
			Body: reason, Href: "/erp/me/overtime"})
	}
	return o.d.Store.GetOvertime(ctx, id)
}
