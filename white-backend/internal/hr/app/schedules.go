package app

import (
	"context"
	"regexp"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Schedules manages shifts, weekly schedule versions and public holidays.
type Schedules struct{ d Deps }

func NewSchedules(d Deps) *Schedules { return &Schedules{d: d} }

var clockRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d(:[0-5]\d)?$`)

func (s *Schedules) Shifts(ctx context.Context, a Actor, includeInactive bool) ([]domain.Shift, error) {
	if err := require(a, authn.PermHRRead, authn.PermSelfService); err != nil {
		return nil, err
	}
	return s.d.Store.ListShifts(ctx, includeInactive)
}

func (s *Schedules) SaveShift(ctx context.Context, a Actor, sh domain.Shift) (*domain.Shift, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	v := validate.New()
	v.Required("name", sh.Name, "Nama shift wajib diisi.")
	v.Match("startTime", sh.StartTime, clockRe, "Jam mulai tidak valid (HH:MM).")
	v.Match("endTime", sh.EndTime, clockRe, "Jam selesai tidak valid (HH:MM).")
	if sh.BreakMinutes < 0 || sh.BreakMinutes > 240 {
		v.Add("breakMinutes", "Istirahat 0 sampai 240 menit.")
	}
	if sh.LateToleranceMinutes < 0 || sh.LateToleranceMinutes > 120 {
		v.Add("lateToleranceMinutes", "Toleransi terlambat 0 sampai 120 menit.")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	sh.Name = strings.TrimSpace(sh.Name)
	var err error
	if sh.ID == "" {
		sh.ID = newID()
		err = s.d.Store.InsertShift(ctx, sh, a.UserRef())
	} else {
		var ok bool
		ok, err = s.d.Store.UpdateShift(ctx, sh, a.UserRef())
		if err == nil && !ok {
			return nil, notFound("Shift tidak ditemukan.")
		}
	}
	if err != nil {
		return nil, err
	}
	return s.d.Store.GetShift(ctx, sh.ID)
}

// DeleteShift deactivates a shift already used by a schedule or attendance.
func (s *Schedules) DeleteShift(ctx context.Context, a Actor, id string) (string, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return "", err
	}
	deactivated, found, err := s.d.Store.DeleteShift(ctx, id)
	switch {
	case err != nil:
		return "", err
	case !found:
		return "", notFound("Shift tidak ditemukan.")
	case deactivated:
		return "Shift sudah dipakai jadwal/absensi, jadi dinonaktifkan (tidak dihapus).", nil
	}
	return "Shift dihapus.", nil
}

// EmployeeSchedule is the pattern in force today and every version.
func (s *Schedules) EmployeeSchedule(ctx context.Context, a Actor, employeeID string) (current, history []domain.ScheduleRow, err error) {
	if employeeID != a.EmployeeID || !a.Can(authn.PermSelfService) {
		if err := require(a, authn.PermHRRead); err != nil {
			return nil, nil, err
		}
	}
	rows, err := s.d.Store.ListScheduleRows(ctx, []string{employeeID})
	if err != nil {
		return nil, nil, err
	}
	today := s.d.today()
	for _, r := range rows {
		if r.EffectiveFrom <= today && (r.EffectiveTo == nil || *r.EffectiveTo >= today) {
			current = append(current, r)
		}
	}
	return current, rows, nil
}

// SetSchedule starts a new weekly pattern on from; days without an entry
// are days off.
func (s *Schedules) SetSchedule(ctx context.Context, a Actor, employeeID, from string, days []domain.ScheduleRow) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	from, err := reqDate("effectiveFrom", from)
	if err != nil {
		return err
	}
	if _, err := s.d.Store.GetEmployee(ctx, employeeID); err != nil {
		return err
	}
	byDay := map[int]*string{}
	for _, d := range days {
		if d.DayOfWeek < 1 || d.DayOfWeek > 7 {
			return invalid("days", "Hari harus 1 (Senin) sampai 7 (Minggu).")
		}
		if d.ShiftID != nil && *d.ShiftID != "" {
			sh, err := s.d.Store.GetShift(ctx, *d.ShiftID)
			if err != nil {
				return err
			}
			if sh == nil || !sh.IsActive {
				return invalid("days", "Shift tidak ditemukan atau nonaktif.")
			}
			byDay[d.DayOfWeek] = d.ShiftID
		}
	}
	rows := make([]domain.ScheduleRow, 0, 7)
	for dow := 1; dow <= 7; dow++ {
		rows = append(rows, domain.ScheduleRow{ID: newID(), DayOfWeek: dow, ShiftID: byDay[dow]})
	}
	return s.d.Store.WithTx(ctx, func(tx domain.Store) error {
		return tx.ReplaceSchedule(ctx, employeeID, from, hris.AddDaysISO(from, -1), rows, a.UserRef())
	})
}

// TeamDay is one cell of the team week.
type TeamDay struct {
	Date, Status       string
	ShiftName          string
	StartTime, EndTime string
}

type TeamRow struct {
	Employee domain.Employee
	Days     []TeamDay
}

// TeamSchedule is the Monday to Sunday grid of a department: hr.read for
// any department, otherwise the caller's own.
func (s *Schedules) TeamSchedule(ctx context.Context, a Actor, departmentID, weekOf string) (string, []TeamRow, error) {
	if !a.Can(authn.PermHRRead) {
		id, err := self(a)
		if err != nil {
			return "", nil, err
		}
		me, err := s.d.Store.GetEmployee(ctx, id)
		if err != nil || me == nil {
			return "", nil, err
		}
		departmentID = deref(me.DepartmentID)
	}
	if weekOf == "" {
		weekOf = s.d.today()
	} else if _, err := reqDate("weekOf", weekOf); err != nil {
		return "", nil, err
	}
	monday := weekStart(weekOf)
	sunday := hris.AddDaysISO(monday, 6)
	emps, _, err := s.d.Store.ListEmployees(ctx, domain.EmployeeFilter{DepartmentID: departmentID})
	if err != nil {
		return "", nil, err
	}
	ids := employeeIDs(emps)
	rows, err := s.d.Store.ListScheduleRows(ctx, ids)
	if err != nil {
		return "", nil, err
	}
	leaves, _, err := s.d.Store.ListLeaves(ctx, domain.LeaveFilter{EmployeeIDs: ids, Status: hris.LeaveApproved, From: monday, To: sunday})
	if err != nil {
		return "", nil, err
	}
	holidays, err := s.d.Store.ListHolidays(ctx, monday, sunday, true)
	if err != nil {
		return "", nil, err
	}
	out := make([]TeamRow, 0, len(emps))
	for _, e := range emps {
		out = append(out, TeamRow{Employee: e, Days: weekDays(scheduleOf(rows, e.ID), leavesOf(leaves, e.ID), holidays, monday)})
	}
	return monday, out, nil
}

// weekDays fills seven days from monday: leave beats a public holiday,
// which beats the pattern.
func weekDays(rows []domain.ScheduleRow, leaves []domain.Leave, holidays []domain.Holiday, monday string) []TeamDay {
	plain := plainRows(rows)
	days := make([]TeamDay, 7)
	for i := range days {
		date := hris.AddDaysISO(monday, i)
		day := TeamDay{Date: date, Status: "none"}
		switch {
		case onLeave(leaves, date):
			day.Status = "cuti"
		case isPublicHoliday(holidays, date):
			day.Status = "libur_nasional"
		default:
			if idx := hris.ResolveScheduleRow(plain, date); idx >= 0 {
				day.Status = "libur"
				if sh := rows[idx].Shift; sh != nil {
					day.Status, day.ShiftName = "shift", sh.Name
					day.StartTime, day.EndTime = domain.HHMM(sh.StartTime), domain.HHMM(sh.EndTime)
				}
			}
		}
		days[i] = day
	}
	return days
}

func leavesOf(leaves []domain.Leave, employeeID string) []domain.Leave {
	var out []domain.Leave
	for _, l := range leaves {
		if l.EmployeeID == employeeID {
			out = append(out, l)
		}
	}
	return out
}

func onLeave(leaves []domain.Leave, date string) bool {
	for _, l := range leaves {
		if l.StartDate <= date && l.EndDate >= date {
			return true
		}
	}
	return false
}

func isPublicHoliday(holidays []domain.Holiday, date string) bool {
	for _, h := range holidays {
		if h.Date == date && !h.DeductsLeave {
			return true
		}
	}
	return false
}

// ─── Holidays ────────────────────────────────────────────────

func (s *Schedules) Holidays(ctx context.Context, a Actor, year int) ([]domain.Holiday, error) {
	if err := require(a, authn.PermHRRead, authn.PermSelfService); err != nil {
		return nil, err
	}
	if year == 0 {
		year = s.d.now().Year()
	}
	from, _ := hris.PeriodBounds(1, year)
	_, to := hris.PeriodBounds(12, year)
	return s.d.Store.ListHolidays(ctx, from, to, false)
}

type HolidayInput struct {
	Date, Name, Type, Status, Note string
	DeductsLeave                   *bool
}

func (s *Schedules) SaveHoliday(ctx context.Context, a Actor, id string, in HolidayInput) (*domain.Holiday, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	body := hris.HolidayInput{HolidayDate: &in.Date, Name: &in.Name, Type: strPtr(in.Type), Status: strPtr(in.Status)}
	if msg := hris.ValidateHolidayBody(body); msg != "" {
		return nil, invalid("holidayDate", msg)
	}
	typ := hris.HolidayTypeOrDefault(body.Type)
	deducts := hris.DefaultDeductsLeave(typ)
	if in.DeductsLeave != nil {
		deducts = *in.DeductsLeave
	}
	status := in.Status
	if status == "" {
		status = "aktif"
	}
	h := domain.Holiday{ID: id, Date: in.Date, Name: strings.TrimSpace(in.Name), Type: typ, DeductsLeave: deducts, Status: status, Note: strings.TrimSpace(in.Note)}
	if id == "" {
		h.ID = newID()
		if err := s.d.Store.InsertHoliday(ctx, h, a.UserRef()); err != nil {
			return nil, err
		}
		return &h, nil
	}
	ok, err := s.d.Store.UpdateHoliday(ctx, h, a.UserRef())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, notFound("Hari libur tidak ditemukan.")
	}
	return &h, nil
}

func (s *Schedules) DeleteHoliday(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ok, err := s.d.Store.DeleteHoliday(ctx, id)
	if err == nil && !ok {
		return notFound("Hari libur tidak ditemukan.")
	}
	return err
}
