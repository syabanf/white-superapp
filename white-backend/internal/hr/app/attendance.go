package app

import (
	"context"
	"math"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// Attendance is clock-in/out, the daily roster, HR validation and the
// "Rekap Absensi" export (lib/hris/attendance-repo, daily-roster,
// attendance-report).
type Attendance struct {
	d    Deps
	docs Documents
}

func NewAttendance(d Deps, docs Documents) *Attendance { return &Attendance{d: d, docs: docs} }

const maxSelfieBytes = 2 << 20

var selfieMIME = []string{"image/jpeg", "image/png", "image/webp"}

var attendanceStatuses = []string{"present", "late", "absent", "half_day", "remote"}

type ClockInput struct {
	EmployeeID   string
	AttendanceID string
	Date         string
	Photo        []byte
	Location     *domain.Location
	Notes        string
}

// ClockIn records the day's attendance. A selfie is optional (NüHabit
// required one); when sent it is magic-byte checked. The scheduled shift
// and its lateness are snapshotted on the row.
func (s *Attendance) ClockIn(ctx context.Context, a Actor, in ClockInput) (*domain.Attendance, error) {
	employeeID, err := employeeOrSelf(a, in.EmployeeID, authn.PermHRWrite)
	if err != nil {
		return nil, err
	}
	today := s.d.today()
	date := today
	if in.Date != "" && in.Date != today {
		if err := require(a, authn.PermHRWrite); err != nil {
			return nil, err
		}
		if date, err = reqDate("date", in.Date); err != nil {
			return nil, err
		}
	}
	emp, err := s.d.Store.GetEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if emp == nil || !emp.IsActive {
		return nil, notFound("Karyawan tidak ditemukan atau sudah tidak aktif.")
	}
	existing, err := s.d.Store.AttendanceIDOn(ctx, employeeID, date)
	if err != nil {
		return nil, err
	}
	if existing != "" {
		return nil, conflict("Sudah clock-in untuk tanggal ini.")
	}
	photo, err := s.saveSelfie(ctx, employeeID, in.Photo)
	if err != nil {
		return nil, err
	}
	now := s.d.now()
	att := domain.Attendance{ID: newID(), EmployeeID: employeeID, Date: date, ClockIn: &now, ClockInLocation: in.Location,
		ClockInPhotoKey: photo, Status: "present", BreakMinutes: 60, Notes: strings.TrimSpace(in.Notes)}
	rows, err := s.d.Store.ListScheduleRows(ctx, []string{employeeID})
	if err != nil {
		return nil, err
	}
	if sh := ShiftOn(rows, date); sh != nil {
		times := shiftTimes(*sh)
		start, end := hris.ScheduledWindow(date, times)
		late, minutes := hris.Lateness(now, date, times)
		att.ShiftID, att.ScheduledStart, att.ScheduledEnd = &sh.ID, &start, &end
		att.IsLate, att.LateMinutes, att.BreakMinutes = late, minutes, sh.BreakMinutes
		if late {
			att.Status = "late"
		}
	}
	if err := s.d.Store.InsertAttendance(ctx, att, a.UserRef()); err != nil {
		return nil, uniqueErr(err, "Sudah clock-in untuk tanggal ini.")
	}
	return s.d.Store.GetAttendance(ctx, att.ID)
}

// ClockOut closes the row: work hours as calculate_work_hours did, and the
// hours past the scheduled end as realized overtime (the cap payroll uses).
func (s *Attendance) ClockOut(ctx context.Context, a Actor, in ClockInput) (*domain.Attendance, error) {
	att, err := s.d.Store.GetAttendance(ctx, in.AttendanceID)
	if err != nil {
		return nil, err
	}
	if att == nil {
		return nil, notFound("Data absensi tidak ditemukan.")
	}
	if att.EmployeeID != a.EmployeeID || !a.Can(authn.PermSelfService) {
		if err := require(a, authn.PermHRWrite); err != nil {
			return nil, err
		}
	}
	if att.ClockOut != nil {
		return nil, conflict("Sudah clock-out untuk absensi ini.")
	}
	photo, err := s.saveSelfie(ctx, att.EmployeeID, in.Photo)
	if err != nil {
		return nil, err
	}
	now := s.d.now()
	att.ClockOut, att.ClockOutLocation, att.ClockOutPhotoKey = &now, in.Location, photo
	if att.ClockIn != nil {
		brk := att.BreakMinutes
		att.WorkHours = hris.WorkHours(*att.ClockIn, now, &brk)
	}
	if att.ScheduledEnd != nil && now.After(*att.ScheduledEnd) {
		att.OvertimeHours = math.Round(now.Sub(*att.ScheduledEnd).Hours()*100) / 100
	}
	if n := strings.TrimSpace(in.Notes); n != "" {
		att.Notes = strings.TrimSpace(att.Notes + "\n" + n)
	}
	ok, err := s.d.Store.ClockOut(ctx, *att)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, conflict("Sudah clock-out untuk absensi ini.")
	}
	return s.d.Store.GetAttendance(ctx, att.ID)
}

func (s *Attendance) saveSelfie(ctx context.Context, employeeID string, photo []byte) (*string, error) {
	if len(photo) == 0 {
		return nil, nil
	}
	key, _, err := s.d.Files.Put(ctx, "attendance/"+employeeID, photo, selfieMIME, maxSelfieBytes)
	if err != nil {
		return nil, fileErr(err, "Ukuran foto maksimal 2 MB.")
	}
	return &key, nil
}

// List: hr.read sees everyone; otherwise only the caller's rows.
func (s *Attendance) List(ctx context.Context, a Actor, f domain.AttendanceFilter) ([]domain.Attendance, int, error) {
	if !a.Can(authn.PermHRRead) {
		id, err := self(a)
		if err != nil {
			return nil, 0, err
		}
		f.EmployeeIDs = []string{id}
	}
	return s.d.Store.ListAttendance(ctx, f)
}

func (s *Attendance) Validate(ctx context.Context, a Actor, id, status, notes string) (*domain.Attendance, error) {
	if err := require(a, authn.PermHRApprove); err != nil {
		return nil, err
	}
	att, err := s.d.Store.GetAttendance(ctx, id)
	if err != nil {
		return nil, err
	}
	if att == nil {
		return nil, notFound("Data absensi tidak ditemukan.")
	}
	if status == "" {
		status = att.Status
	}
	if !slices.Contains(attendanceStatuses, status) {
		return nil, invalid("status", "Status absensi tidak valid.")
	}
	if _, err := s.d.Store.ValidateAttendance(ctx, id, status, a.EmployeeRef(), strings.TrimSpace(notes)); err != nil {
		return nil, err
	}
	return s.d.Store.GetAttendance(ctx, id)
}

// Photo returns a selfie: hr.read for anyone, the owner for their own.
func (s *Attendance) Photo(ctx context.Context, a Actor, id, kind string) ([]byte, string, error) {
	att, err := s.d.Store.GetAttendance(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if att == nil {
		return nil, "", notFound("Data absensi tidak ditemukan.")
	}
	if !a.Can(authn.PermHRRead) && att.EmployeeID != a.EmployeeID {
		return nil, "", require(a, authn.PermHRRead)
	}
	key := att.ClockInPhotoKey
	if kind == "out" {
		key = att.ClockOutPhotoKey
	}
	if key == nil {
		return nil, "", notFound("Foto tidak tersedia.")
	}
	return s.d.Files.Get(ctx, *key)
}

// ─── Daily roster ────────────────────────────────────────────

type RosterEntry struct {
	Employee   domain.Employee
	Shift      *domain.Shift
	Status     string
	IsOverdue  bool
	Attendance *domain.Attendance
	LeaveType  string
}

type Roster struct {
	Date        string
	HolidayName string
	Entries     []RosterEntry
	Summary     map[string]int
}

// Roster is getDailyRoster: every active employee with the shift of the
// day; attendance beats leave, leave beats a public holiday.
func (s *Attendance) Roster(ctx context.Context, a Actor, date, departmentID string) (*Roster, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, err
	}
	if date == "" {
		date = s.d.today()
	} else if _, err := reqDate("date", date); err != nil {
		return nil, err
	}
	emps, _, err := s.d.Store.ListEmployees(ctx, domain.EmployeeFilter{DepartmentID: departmentID})
	if err != nil {
		return nil, err
	}
	ids := employeeIDs(emps)
	rows, err := s.d.Store.ListScheduleRows(ctx, ids)
	if err != nil {
		return nil, err
	}
	atts, _, err := s.d.Store.ListAttendance(ctx, domain.AttendanceFilter{EmployeeIDs: ids, From: date, To: date})
	if err != nil {
		return nil, err
	}
	byEmp := map[string]*domain.Attendance{}
	for i := range atts {
		byEmp[atts[i].EmployeeID] = &atts[i]
	}
	leaves, err := s.d.Store.ApprovedLeavesOn(ctx, date)
	if err != nil {
		return nil, err
	}
	holidays, err := s.d.Store.ListHolidays(ctx, date, date, true)
	if err != nil {
		return nil, err
	}
	out := &Roster{Date: date, Summary: map[string]int{}}
	publicHoliday := false
	for _, h := range holidays {
		if !h.DeductsLeave {
			publicHoliday, out.HolidayName = true, h.Name
		}
	}
	now := s.d.now()
	past := date < s.d.today()
	for _, e := range emps {
		mine := scheduleOf(rows, e.ID)
		idx := hris.ResolveScheduleRow(plainRows(mine), date)
		var shiftID *string
		if idx >= 0 {
			shiftID = mine[idx].ShiftID
		}
		sh := ShiftOn(mine, date)
		att := byEmp[e.ID]
		leaveType, onLeave := leaves[e.ID]
		status := hris.DeriveRosterStatus(hris.RosterStatusInput{HasAttendance: att != nil, IsLate: att != nil && att.IsLate,
			OnApprovedLeave: onLeave, HasSchedule: idx >= 0, ShiftID: shiftID, IsPastDate: past, IsPublicHoliday: publicHoliday})
		entry := RosterEntry{Employee: e, Shift: sh, Status: status, Attendance: att, LeaveType: leaveType}
		if sh != nil {
			t := shiftTimes(*sh)
			entry.IsOverdue = hris.IsOverdue(now, date, &t, status)
		}
		out.Summary[status]++
		out.Entries = append(out.Entries, entry)
	}
	return out, nil
}

// ─── Export ──────────────────────────────────────────────────

type ExportFilter struct {
	EmployeeID, StartDate, EndDate, Status string
}

// ExportXLSX is the "Rekap Absensi" workbook (buildAttendanceXlsx).
func (s *Attendance) ExportXLSX(ctx context.Context, a Actor, f ExportFilter) ([]byte, string, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, "", err
	}
	filter := domain.AttendanceFilter{From: f.StartDate, To: f.EndDate, Status: f.Status}
	if f.EmployeeID != "" {
		filter.EmployeeIDs = []string{f.EmployeeID}
	}
	atts, _, err := s.d.Store.ListAttendance(ctx, filter)
	if err != nil {
		return nil, "", err
	}
	if len(atts) == 0 {
		return nil, "", notFound("Tidak ada data absensi untuk diekspor.")
	}
	rows := make([]hris.ReportRow, len(atts))
	for i, r := range atts {
		row := hris.ReportRow{Date: r.Date, EmployeeName: r.EmployeeName, NIP: strPtr(r.EmployeeNo), Department: strPtr(r.DepartmentName),
			Position: strPtr(r.PositionName), ClockIn: r.ClockIn, ClockOut: r.ClockOut, Status: &atts[i].Status,
			IsLate: r.IsLate, LateMinutes: r.LateMinutes, Notes: strPtr(r.Notes)}
		if r.ClockOut != nil {
			h := r.WorkHours
			row.WorkHours = &h
		}
		rows[i] = row
	}
	hris.SortReportRows(rows)
	meta := AttendanceReportMeta{CompanyName: "Mixist", PeriodLabel: "Semua tanggal", GeneratedAt: s.d.now()}
	if f.StartDate != "" && f.EndDate != "" {
		meta.PeriodLabel = hris.PeriodLabel(f.StartDate, f.EndDate)
	}
	if f.EmployeeID != "" {
		meta.EmployeeLabel = rows[0].EmployeeName
	}
	data, err := s.docs.AttendanceXLSX(rows, meta)
	return data, "rekap-absensi-" + domain.DateOf(hris.NowWIB(meta.GeneratedAt)) + ".xlsx", err
}

// ─── schedule helpers ────────────────────────────────────────

func scheduleOf(rows []domain.ScheduleRow, employeeID string) []domain.ScheduleRow {
	var out []domain.ScheduleRow
	for _, r := range rows {
		if r.EmployeeID == employeeID {
			out = append(out, r)
		}
	}
	return out
}

// plainRows is a pattern in the shape the ported rules use.
func plainRows(rows []domain.ScheduleRow) []hris.ScheduleRow {
	out := make([]hris.ScheduleRow, len(rows))
	for i, r := range rows {
		out[i] = hris.ScheduleRow{DayOfWeek: r.DayOfWeek, ShiftID: r.ShiftID, EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo}
	}
	return out
}

// ShiftOn is the shift that applies on date (nil on a day off or without
// a pattern). rows must belong to one employee.
func ShiftOn(rows []domain.ScheduleRow, date string) *domain.Shift {
	idx := hris.ResolveScheduleRow(plainRows(rows), date)
	if idx < 0 || rows[idx].Shift == nil || rows[idx].Shift.StartTime == "" {
		return nil
	}
	return rows[idx].Shift
}

func shiftTimes(sh domain.Shift) hris.ShiftTimes {
	return hris.ShiftTimes{StartTime: sh.StartTime, EndTime: sh.EndTime, IsOvernight: sh.IsOvernight, LateToleranceMinutes: sh.LateToleranceMinutes}
}

// workingDays counts scheduled days of [start, end]; without a pattern
// every weekday counts.
func workingDays(rows []hris.ScheduleRow, start, end string) int {
	n := 0
	for _, d := range hris.EachDateISO(start, end) {
		if len(rows) == 0 {
			if !hris.IsWeekend(d) {
				n++
			}
			continue
		}
		if i := hris.ResolveScheduleRow(rows, d); i >= 0 && rows[i].ShiftID != nil {
			n++
		}
	}
	return n
}

func employeeIDs(emps []domain.Employee) []string {
	out := make([]string, len(emps))
	for i, e := range emps {
		out[i] = e.ID
	}
	return out
}

// weekStart is the Monday of the week containing date.
func weekStart(date string) string {
	return hris.AddDaysISO(date, -(hris.IsoDayOfWeek(date) - 1))
}
