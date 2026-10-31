package app_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/docs"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/postgres"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/adapter/storage"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/app"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/seed"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

// ─── fakes ───────────────────────────────────────────────────

type fakeIdentity struct {
	mu    sync.Mutex
	users map[string]app.UserInfo
	sent  []app.NotifyInput
	// approvers are the hr.approve holders identity reports.
	approvers []string
}

func (f *fakeIdentity) PermissionHolders(_ context.Context, perm string) ([]string, error) {
	if perm != authn.PermHRApprove {
		return nil, nil
	}
	return f.approvers, nil
}

func (f *fakeIdentity) GetUser(_ context.Context, id string) (*app.UserInfo, error) {
	if u, ok := f.users[id]; ok {
		return &u, nil
	}
	return nil, nil
}

func (f *fakeIdentity) Notify(_ context.Context, in app.NotifyInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, in)
	return nil
}

func (f *fakeIdentity) count(typ string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, s := range f.sent {
		if s.Type == typ {
			n++
		}
	}
	return n
}

type recorder struct {
	mu   sync.Mutex
	msgs []struct {
		subject string
		data    []byte
	}
}

func (r *recorder) Publish(_ context.Context, subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, struct {
		subject string
		data    []byte
	}{subject, data})
	return nil
}

// ─── harness ─────────────────────────────────────────────────

// now is Wednesday 21 October 2026, 08:20 WIB.
var now = clock.At("2026-10-21T01:20:00Z")

type env struct {
	t        *testing.T
	ctx      context.Context
	d        app.Deps
	identity *fakeIdentity
	events   *recorder
	people   *app.People
	att      *app.Attendance
	sched    *app.Schedules
	leaves   *app.Leaves
	ot       *app.Overtime
	contract *app.Contracts
	payroll  *app.Payroll
	loans    *app.Loans
	maint    *app.Maintenance
	admin    app.Actor
}

func setup(t *testing.T) *env {
	pool := pg.TestPool(t, hr.Migrations(), "migrations", hr.Schema)
	pg.TruncateSchema(t, pool, hr.Schema)
	ctx := context.Background()
	// payroll_settings and employment_statuses are seeded by the migration.
	if _, err := pool.Exec(ctx, `INSERT INTO hr.payroll_settings (id) VALUES (1) ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO hr.employment_statuses (code, name, sort_order) VALUES
		('permanent','Karyawan Tetap',1),('contract','Kontrak (PKWT)',2),('probation','Masa Percobaan',3),
		('intern','Magang',4),('resigned','Mengundurkan Diri',5),('terminated','Diberhentikan',6) ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	id := &fakeIdentity{users: map[string]app.UserInfo{"user-ayu": {ID: "user-ayu", Email: "ayu@mixist.id"}, "user-budi": {ID: "user-budi"}, "user-wulan": {ID: "user-wulan"}}}
	rec := &recorder{}
	d := app.Deps{Store: postgres.NewStore(pool), Identity: id, Files: storage.NewLocal(t.TempDir()), Events: rec, Clock: now,
		Logger: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))}
	r := docs.Renderer{CompanyName: "Mixist"}
	return &env{t: t, ctx: ctx, d: d, identity: id, events: rec, people: app.NewPeople(d), att: app.NewAttendance(d, r),
		sched: app.NewSchedules(d), leaves: app.NewLeaves(d), ot: app.NewOvertime(d), contract: app.NewContracts(d),
		payroll: app.NewPayroll(d, r), loans: app.NewLoans(d), maint: app.NewMaintenance(d),
		admin: app.Actor{Principal: authn.Principal{UserID: "user-admin", Role: authn.RoleAdmin}}}
}

func (e *env) must(err error) {
	e.t.Helper()
	if err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) actor(userID string, perms ...string) app.Actor {
	e.t.Helper()
	a, err := e.d.ActorFor(e.ctx, authn.Principal{UserID: userID, Role: authn.RoleMember, Perms: perms})
	e.must(err)
	return a
}

func wantCode(t *testing.T, err error, code apperr.Code) {
	t.Helper()
	if err == nil || !apperr.Is(err, code) {
		t.Fatalf("want %s, got %v", code, err)
	}
}

// org creates HR and Creative with Wulan (HR manager), Budi (Creative
// lead) and Ayu (designer reporting to Budi), all on a 09:00 shift.
type org struct{ hrDept, creative, wulan, budi, ayu, shift string }

func (e *env) org() org {
	var o org
	hrd, err := e.people.SaveDepartment(e.ctx, e.admin, "", app.DepartmentInput{Code: "HR", Name: "HR", IsActive: true})
	e.must(err)
	crv, err := e.people.SaveDepartment(e.ctx, e.admin, "", app.DepartmentInput{Code: "CRV", Name: "Creative", IsActive: true})
	e.must(err)
	o.hrDept, o.creative = hrd.ID, crv.ID
	mk := func(name, email, dept, manager, user, status string) string {
		emp, err := e.people.Create(e.ctx, e.admin, app.EmployeeInput{FullName: name, Email: email, DepartmentID: dept, ManagerID: manager,
			EmploymentStatus: status, JoinDate: "2024-01-15", UserID: user})
		e.must(err)
		return emp.ID
	}
	o.wulan = mk("Wulan Sari", "wulan@mixist.id", o.hrDept, "", "user-wulan", "permanent")
	o.budi = mk("Budi Santoso", "budi@mixist.id", o.creative, "", "user-budi", "permanent")
	o.ayu = mk("Ayu Lestari", "ayu@mixist.id", o.creative, o.budi, "user-ayu", "permanent")
	sh, err := e.sched.SaveShift(e.ctx, e.admin, domain.Shift{Name: "Reguler", StartTime: "09:00", EndTime: "18:00", BreakMinutes: 60, LateToleranceMinutes: 15, IsActive: true})
	e.must(err)
	o.shift = sh.ID
	week := []domain.ScheduleRow{}
	for d := 1; d <= 5; d++ {
		week = append(week, domain.ScheduleRow{DayOfWeek: d, ShiftID: &o.shift})
	}
	for _, id := range []string{o.wulan, o.budi, o.ayu} {
		e.must(e.sched.SetSchedule(e.ctx, e.admin, id, "2024-01-15", week))
	}
	return o
}

// ─── people ──────────────────────────────────────────────────

func TestPeopleRecordsHistoryAndRefusesCycles(t *testing.T) {
	e := setup(t)
	o := e.org()
	ayu, err := e.people.Get(e.ctx, o.ayu)
	e.must(err)
	if ayu.EmployeeNo != "EMP-2024-00003" || ayu.ManagerName != "Budi Santoso" || ayu.UserID == nil {
		t.Fatalf("employee %+v", ayu)
	}
	_, err = e.people.Update(e.ctx, e.admin, o.budi, app.EmployeeInput{FullName: "Budi Santoso", JoinDate: "2024-01-15", ManagerID: o.ayu, EmploymentStatus: "permanent"})
	wantCode(t, err, apperr.Validation)

	_, err = e.people.Update(e.ctx, e.admin, o.ayu, app.EmployeeInput{FullName: "Ayu Lestari", Email: "ayu@mixist.id", JoinDate: "2024-01-15",
		DepartmentID: o.hrDept, ManagerID: o.wulan, EmploymentStatus: "contract"})
	e.must(err)
	hist, err := e.people.History(e.ctx, e.admin, o.ayu)
	e.must(err)
	if len(hist) != 1 || !strings.Contains(hist[0].Notes, "permanent → contract") || hist[0].NewDepartment != "HR" {
		t.Fatalf("history %+v", hist)
	}
	_, err = e.people.Create(e.ctx, e.admin, app.EmployeeInput{FullName: "Dup", Email: "AYU@mixist.id", JoinDate: "2025-01-01"})
	wantCode(t, err, apperr.Conflict)
	_, err = e.people.LinkUser(e.ctx, e.admin, o.wulan, "user-ghost")
	wantCode(t, err, apperr.Validation)
	_, err = e.people.Create(e.ctx, e.actor("user-ayu", authn.PermSelfService), app.EmployeeInput{FullName: "X", JoinDate: "2025-01-01"})
	wantCode(t, err, apperr.Unauthorized)

	roots, total, err := e.people.OrgChart(e.ctx, e.admin, "")
	e.must(err)
	if total != 3 || len(roots) != 2 {
		t.Fatalf("org chart %d roots of %d", len(roots), total)
	}
	e.must(e.maint.ForgetUser(e.ctx, "user-ayu"))
	ayu, _ = e.people.Get(e.ctx, o.ayu)
	if ayu.UserID != nil {
		t.Fatal("user.deleted must unlink")
	}
}

// ─── attendance ──────────────────────────────────────────────

func TestAttendanceClockInLatenessAndClockOut(t *testing.T) {
	e := setup(t)
	o := e.org()
	ayu := e.actor("user-ayu", authn.PermSelfService)
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0x10, 'J', 'F', 'I', 'F'}

	// 08:20 WIB: early.
	in, err := e.att.ClockIn(e.ctx, ayu, app.ClockInput{Photo: jpeg, Location: &domain.Location{Latitude: -6.26, Longitude: 106.81}})
	e.must(err)
	if in.Date != "2026-10-21" || in.IsLate || in.ShiftName != "Reguler" || in.ClockInPhotoKey == nil || in.ClockInLocation == nil {
		t.Fatalf("clock in %+v", in)
	}
	_, err = e.att.ClockIn(e.ctx, ayu, app.ClockInput{})
	wantCode(t, err, apperr.Conflict)
	_, err = e.att.ClockIn(e.ctx, e.actor("user-wulan", authn.PermSelfService), app.ClockInput{Photo: []byte("<?php echo 1;")})
	wantCode(t, err, apperr.Validation)

	// Budi clocks in at 09:31 WIB: past the 15 minute tolerance.
	late := app.NewAttendance(withClock(e.d, "2026-10-21T02:31:00Z"), docs.Renderer{})
	b, err := late.ClockIn(e.ctx, e.actor("user-budi", authn.PermSelfService), app.ClockInput{})
	e.must(err)
	if !b.IsLate || b.LateMinutes != 31 || b.Status != "late" {
		t.Fatalf("late %+v", b)
	}
	_, err = e.att.ClockIn(e.ctx, ayu, app.ClockInput{EmployeeID: o.budi})
	wantCode(t, err, apperr.Unauthorized)

	// Clock out at 19:30 WIB: 11h10 minus the hour break, 1.5h past 18:00.
	out, err := app.NewAttendance(withClock(e.d, "2026-10-21T12:30:00Z"), docs.Renderer{}).ClockOut(e.ctx, ayu, app.ClockInput{AttendanceID: in.ID, Notes: "Lembur revisi"})
	e.must(err)
	if out.WorkHours != 10.17 || out.OvertimeHours != 1.5 || out.Notes != "Lembur revisi" {
		t.Fatalf("clock out %+v", out)
	}
	_, err = e.att.ClockOut(e.ctx, ayu, app.ClockInput{AttendanceID: in.ID})
	wantCode(t, err, apperr.Conflict)
	_, err = e.att.ClockOut(e.ctx, e.actor("user-wulan", authn.PermSelfService), app.ClockInput{AttendanceID: b.ID})
	wantCode(t, err, apperr.Unauthorized)

	photo, ct, err := e.att.Photo(e.ctx, ayu, in.ID, "in")
	e.must(err)
	if ct != "image/jpeg" || len(photo) != len(jpeg) {
		t.Fatal("photo round trip")
	}
	_, _, err = e.att.Photo(e.ctx, e.actor("user-budi", authn.PermSelfService), in.ID, "in")
	wantCode(t, err, apperr.Unauthorized)

	roster, err := e.att.Roster(e.ctx, e.admin, "", "")
	e.must(err)
	if roster.Summary[hris.RosterHadir] != 1 || roster.Summary[hris.RosterTerlambat] != 1 || roster.Summary[hris.RosterBelumAbsen] != 1 {
		t.Fatalf("roster %+v", roster.Summary)
	}
	v, err := e.att.Validate(e.ctx, e.actor("user-wulan", authn.PermHRApprove), b.ID, "present", "Macet, sudah izin atasan")
	e.must(err)
	if v.ValidatedAt == nil || v.Status != "present" || v.ValidatedByName != "Wulan Sari" {
		t.Fatalf("validated %+v", v)
	}
	mine, total, err := e.att.List(e.ctx, ayu, domain.AttendanceFilter{})
	e.must(err)
	if total != 1 || mine[0].EmployeeID != o.ayu {
		t.Fatal("self list must be own rows")
	}
	xlsx, name, err := e.att.ExportXLSX(e.ctx, e.admin, app.ExportFilter{StartDate: "2026-10-01", EndDate: "2026-10-31"})
	e.must(err)
	if !strings.HasPrefix(name, "rekap-absensi-") || len(xlsx) < 500 {
		t.Fatal("export")
	}
	sum, err := e.people.AttendanceSummary(e.ctx, []string{o.ayu, o.budi}, "2026-10")
	e.must(err)
	if sum[0].WorkingDays != 22 || sum[0].PresentDays != 1 || sum[1].LateDays != 1 || sum[1].LateMinutes != 31 {
		t.Fatalf("summary %+v", sum)
	}
}

func withClock(d app.Deps, at string) app.Deps {
	d.Clock = clock.At(at)
	return d
}

// ─── leave ───────────────────────────────────────────────────

func TestLeaveManagerThenHR(t *testing.T) {
	e := setup(t)
	o := e.org()
	_, err := e.sched.SaveHoliday(e.ctx, e.admin, "", app.HolidayInput{Date: "2026-10-28", Name: "Libur Nasional Uji"})
	e.must(err)
	ayu := e.actor("user-ayu", authn.PermSelfService)
	budi := e.actor("user-budi", authn.PermSelfService)
	wulan := e.actor("user-wulan", authn.PermSelfService, authn.PermHRApprove, authn.PermHRRead)

	// Mon 26 to Fri 30 Oct: the Wednesday holiday does not count.
	l, err := e.leaves.Create(e.ctx, ayu, app.LeaveInput{LeaveType: "annual", StartDate: "2026-10-26", EndDate: "2026-10-30", Reason: "Liburan"})
	e.must(err)
	if l.TotalDays != 4 || l.Status != hris.LeavePending {
		t.Fatalf("leave %+v", l)
	}
	if e.identity.count(app.NotifLeaveRequested) != 1 {
		t.Fatal("manager notified")
	}
	_, err = e.leaves.Create(e.ctx, ayu, app.LeaveInput{LeaveType: "annual", StartDate: "2026-10-31", EndDate: "2026-11-01", Reason: "x"})
	wantCode(t, err, apperr.Validation) // weekend only
	_, err = e.leaves.Create(e.ctx, ayu, app.LeaveInput{LeaveType: "annual", StartDate: "2026-10-29", EndDate: "2026-10-29", Reason: "x"})
	wantCode(t, err, apperr.Conflict) // overlap
	_, err = e.leaves.Create(e.ctx, ayu, app.LeaveInput{LeaveType: "annual", StartDate: "2026-11-02", EndDate: "2026-11-20", Reason: "x"})
	wantCode(t, err, apperr.Validation) // 12 - 4 pending < 15

	_, err = e.leaves.Decide(e.ctx, ayu, l.ID, true, "")
	wantCode(t, err, apperr.Unauthorized)
	l, err = e.leaves.Decide(e.ctx, budi, l.ID, true, "")
	e.must(err)
	if l.Status != hris.LeaveManagerApproved || l.ManagerApprovedByName != "Budi Santoso" {
		t.Fatalf("manager step %+v", l)
	}
	_, err = e.leaves.Decide(e.ctx, budi, l.ID, true, "")
	wantCode(t, err, apperr.Unauthorized)
	l, err = e.leaves.Decide(e.ctx, wulan, l.ID, true, "")
	e.must(err)
	if l.Status != hris.LeaveApproved || l.ApprovedByName != "Wulan Sari" {
		t.Fatalf("hr step %+v", l)
	}
	bal, err := e.leaves.Balance(e.ctx, ayu, "", 2026)
	e.must(err)
	if bal.AnnualTotal != 12 || bal.AnnualUsed != 4 || bal.PendingDays != 0 {
		t.Fatalf("balance %+v", bal)
	}
	_, err = e.leaves.Cancel(e.ctx, ayu, l.ID)
	wantCode(t, err, apperr.Unauthorized)
	_, err = e.leaves.Cancel(e.ctx, wulan, l.ID)
	e.must(err)
	bal, _ = e.leaves.Balance(e.ctx, ayu, "", 2026)
	if bal.AnnualUsed != 0 {
		t.Fatal("cancel refunds the quota")
	}

	sick, err := e.leaves.Create(e.ctx, ayu, app.LeaveInput{LeaveType: "sick", StartDate: "2026-11-02", EndDate: "2026-11-02", Reason: "Demam"})
	e.must(err)
	_, err = e.leaves.Decide(e.ctx, budi, sick.ID, false, "")
	wantCode(t, err, apperr.Validation)
	sick, err = e.leaves.Decide(e.ctx, budi, sick.ID, false, "Mohon lampirkan surat dokter")
	e.must(err)
	if sick.Status != hris.LeaveRejected {
		t.Fatal("rejected")
	}
	approvals, _, err := e.leaves.List(e.ctx, budi, app.LeaveQuery{Scope: "approvals"})
	e.must(err)
	if len(approvals) != 0 {
		t.Fatal("nothing pending for Budi")
	}
	csv, err := e.leaves.ExportCSV(e.ctx, wulan, app.LeaveQuery{})
	e.must(err)
	if !strings.HasPrefix(string(csv), hris.LeaveCSVHeader) {
		t.Fatal("csv header")
	}
	_ = o
}

// ─── overtime ────────────────────────────────────────────────

func TestOvertimeRequestAndAssignment(t *testing.T) {
	e := setup(t)
	o := e.org()
	ayu := e.actor("user-ayu", authn.PermSelfService)
	budi := e.actor("user-budi", authn.PermSelfService)
	ot, err := e.ot.Create(e.ctx, ayu, app.OvertimeInput{Date: "2026-10-20", StartTime: "18:00", EndTime: "20:30", Reason: "Revisi"})
	e.must(err)
	if ot.Hours != 2.5 || ot.Source != "employee" {
		t.Fatalf("overtime %+v", ot)
	}
	_, err = e.ot.Create(e.ctx, ayu, app.OvertimeInput{Date: "2026-10-20", StartTime: "21:00", EndTime: "22:00"})
	wantCode(t, err, apperr.Conflict)
	_, err = e.ot.Decide(e.ctx, ayu, ot.ID, "approve", "")
	wantCode(t, err, apperr.Unauthorized)
	ot, err = e.ot.Decide(e.ctx, budi, ot.ID, "approve", "")
	e.must(err)
	if ot.Status != "approved" || ot.DecidedByName != "Budi Santoso" {
		t.Fatalf("decided %+v", ot)
	}
	night, err := e.ot.Create(e.ctx, e.admin, app.OvertimeInput{EmployeeID: o.budi, Date: "2026-10-22", StartTime: "22:00", EndTime: "02:00"})
	e.must(err)
	if night.Hours != 4 || night.Source != "company" {
		t.Fatalf("assignment %+v", night)
	}
	_, err = e.ot.Decide(e.ctx, e.admin, night.ID, "approve", "")
	wantCode(t, err, apperr.Unauthorized) // only the assigned employee confirms
	_, err = e.ot.Decide(e.ctx, budi, night.ID, "approve", "")
	e.must(err)
}

func TestOvertimeHRUpdateAndDelete(t *testing.T) {
	e := setup(t)
	e.org()
	ayu := e.actor("user-ayu", authn.PermSelfService)
	budi := e.actor("user-budi", authn.PermSelfService)
	hrw := e.actor("user-wulan", authn.PermSelfService, authn.PermHRWrite, authn.PermHRRead)
	ot, err := e.ot.Create(e.ctx, ayu, app.OvertimeInput{Date: "2026-10-20", StartTime: "18:00", EndTime: "20:30", Reason: "Revisi"})
	e.must(err)
	_, err = e.ot.Update(e.ctx, ayu, ot.ID, app.OvertimeInput{Date: "2026-10-20", StartTime: "19:00", EndTime: "21:00"})
	wantCode(t, err, apperr.Unauthorized)
	ot, err = e.ot.Update(e.ctx, hrw, ot.ID, app.OvertimeInput{Date: "2026-10-19", StartTime: "19:00", EndTime: "23:30", Reason: "Revisi klien"})
	e.must(err)
	if ot.Date != "2026-10-19" || ot.Hours != 4.5 || ot.Source != "employee" || ot.Status != "pending" || ot.Reason != "Revisi klien" {
		t.Fatalf("updated %+v", ot)
	}
	_, err = e.ot.Update(e.ctx, hrw, ot.ID, app.OvertimeInput{Date: "2026-10-19", StartTime: "10:00", EndTime: "23:30"})
	wantCode(t, err, apperr.Validation) // over 12 hours

	other, err := e.ot.Create(e.ctx, ayu, app.OvertimeInput{Date: "2026-10-20", StartTime: "18:00", EndTime: "19:00"})
	e.must(err)
	_, err = e.ot.Update(e.ctx, hrw, other.ID, app.OvertimeInput{Date: "2026-10-19", StartTime: "18:00", EndTime: "19:00"})
	wantCode(t, err, apperr.Conflict) // one active entry per day

	_, err = e.ot.Decide(e.ctx, budi, ot.ID, "approve", "")
	e.must(err)
	_, err = e.ot.Update(e.ctx, hrw, ot.ID, app.OvertimeInput{Date: "2026-10-19", StartTime: "19:00", EndTime: "20:00"})
	wantCode(t, err, apperr.Conflict)
	wantCode(t, e.ot.Delete(e.ctx, hrw, ot.ID), apperr.Conflict) // approved hours feed payroll

	_, err = e.ot.Decide(e.ctx, ayu, other.ID, "cancel", "")
	e.must(err)
	wantCode(t, e.ot.Delete(e.ctx, ayu, other.ID), apperr.Unauthorized)
	e.must(e.ot.Delete(e.ctx, hrw, other.ID))
	wantCode(t, e.ot.Delete(e.ctx, hrw, other.ID), apperr.NotFound)
}

func TestAttendanceManualRecords(t *testing.T) {
	e := setup(t)
	o := e.org()
	hrw := e.actor("user-wulan", authn.PermSelfService, authn.PermHRWrite, authn.PermHRRead)
	in := app.RecordInput{EmployeeID: o.budi, Date: "2026-10-20", ClockIn: "09:40", ClockOut: "18:30", Notes: "Lupa absen"}
	_, err := e.att.CreateRecord(e.ctx, e.actor("user-ayu", authn.PermSelfService), in)
	wantCode(t, err, apperr.Unauthorized)

	// Tuesday on the 09:00 shift: 40 minutes late, 8h50 minus the break,
	// half an hour past 18:00.
	r, err := e.att.CreateRecord(e.ctx, hrw, in)
	e.must(err)
	if r.Status != "late" || !r.IsLate || r.LateMinutes != 40 || r.WorkHours != 7.83 || r.OvertimeHours != 0.5 ||
		r.ShiftName != "Reguler" || r.Notes != "Lupa absen" || hris.NowWIB(*r.ClockIn).Format("15:04") != "09:40" {
		t.Fatalf("manual record %+v", r)
	}
	_, err = e.att.CreateRecord(e.ctx, hrw, in)
	wantCode(t, err, apperr.Conflict)
	_, err = e.att.CreateRecord(e.ctx, hrw, app.RecordInput{EmployeeID: o.ayu, Date: "2026-10-22", ClockIn: "09:00"})
	wantCode(t, err, apperr.Validation) // future date
	_, err = e.att.CreateRecord(e.ctx, hrw, app.RecordInput{EmployeeID: o.ayu, Date: "2026-10-19"})
	wantCode(t, err, apperr.Validation) // clock-in required unless absent
	_, err = e.att.CreateRecord(e.ctx, hrw, app.RecordInput{EmployeeID: o.ayu, Date: "2026-10-19", ClockIn: "09:00", Status: "sick"})
	wantCode(t, err, apperr.Validation)
	absent, err := e.att.CreateRecord(e.ctx, hrw, app.RecordInput{EmployeeID: o.ayu, Date: "2026-10-19", Status: "absent"})
	e.must(err)
	if absent.Status != "absent" || absent.ClockIn != nil || absent.WorkHours != 0 {
		t.Fatalf("absent %+v", absent)
	}

	// Corrected to an on-time start and a clock-out past midnight.
	r, err = e.att.UpdateRecord(e.ctx, hrw, r.ID, app.RecordInput{ClockIn: "08:55", ClockOut: "02:00", Notes: "Koreksi"})
	e.must(err)
	if r.Status != "present" || r.IsLate || r.LateMinutes != 0 || r.WorkHours != 16.08 || r.OvertimeHours != 8 ||
		r.Date != "2026-10-20" || r.EmployeeID != o.budi || r.Notes != "Koreksi" {
		t.Fatalf("corrected %+v", r)
	}
	r, err = e.att.UpdateRecord(e.ctx, hrw, r.ID, app.RecordInput{ClockIn: "09:30", Status: "remote"})
	e.must(err)
	if r.Status != "remote" || !r.IsLate || r.ClockOut != nil || r.WorkHours != 0 || r.OvertimeHours != 0 {
		t.Fatalf("explicit status %+v", r)
	}
	_, err = e.att.UpdateRecord(e.ctx, hrw, r.ID, app.RecordInput{ClockOut: "17:00"})
	wantCode(t, err, apperr.Validation)

	wantCode(t, e.att.DeleteRecord(e.ctx, e.actor("user-budi", authn.PermSelfService), r.ID), apperr.Unauthorized)
	e.must(e.att.DeleteRecord(e.ctx, hrw, r.ID))
	wantCode(t, e.att.DeleteRecord(e.ctx, hrw, r.ID), apperr.NotFound)
	_, err = e.att.UpdateRecord(e.ctx, hrw, r.ID, in)
	wantCode(t, err, apperr.NotFound)
}

func TestLoanCancelPendingOnly(t *testing.T) {
	e := setup(t)
	o := e.org()
	ayu := e.actor("user-ayu", authn.PermSelfService)
	payrollAdmin := e.actor("user-wulan", authn.PermPayroll, authn.PermHRApprove, authn.PermSelfService)
	for _, id := range []string{o.ayu, o.budi} {
		_, err := e.payroll.CreateSalary(e.ctx, payrollAdmin, app.SalaryInput{EmployeeID: id, BaseSalary: "10000000", PTKPStatus: "TK/0", EffectiveDate: "2024-01-15"})
		e.must(err)
	}
	loan, err := e.loans.Create(e.ctx, ayu, app.LoanInput{LoanType: "kasbon", Principal: "1000000", Tenor: 2})
	e.must(err)
	_, err = e.loans.Cancel(e.ctx, e.actor("user-budi", authn.PermSelfService), loan.ID)
	wantCode(t, err, apperr.NotFound) // someone else's loan
	loan, err = e.loans.Cancel(e.ctx, ayu, loan.ID)
	e.must(err)
	if loan.Status != "cancelled" || loan.IsActive {
		t.Fatalf("cancelled %+v", loan)
	}
	_, err = e.loans.Cancel(e.ctx, ayu, loan.ID)
	wantCode(t, err, apperr.Conflict)

	// The cancelled loan no longer takes the one active slot.
	next, err := e.loans.Create(e.ctx, ayu, app.LoanInput{LoanType: "kasbon", Principal: "1000000", Tenor: 2})
	e.must(err)
	_, err = e.loans.Decide(e.ctx, payrollAdmin, next.ID, true, "")
	e.must(err)
	_, err = e.loans.Cancel(e.ctx, ayu, next.ID)
	wantCode(t, err, apperr.Conflict)
	_, err = e.loans.Cancel(e.ctx, payrollAdmin, next.ID)
	wantCode(t, err, apperr.Conflict) // approved loans are repaid through payroll

	forBudi, err := e.loans.Create(e.ctx, payrollAdmin, app.LoanInput{EmployeeID: o.budi, LoanType: "loan", Principal: "2000000", Tenor: 4})
	e.must(err)
	forBudi, err = e.loans.Cancel(e.ctx, payrollAdmin, forBudi.ID)
	e.must(err)
	if forBudi.Status != "cancelled" {
		t.Fatalf("hr cancel %+v", forBudi)
	}
}

// ─── contracts ───────────────────────────────────────────────

func TestContractLifecycle(t *testing.T) {
	e := setup(t)
	o := e.org()
	c, err := e.contract.Create(e.ctx, e.admin, app.ContractInput{EmployeeID: o.ayu, ContractType: "pkwt", StartDate: "2026-01-01", EndDate: "2026-11-10", BaseSalary: "8000000"})
	e.must(err)
	if c.ContractNumber != "0001/PKWT/X/2026" || c.Status != "draft" || c.DepartmentName != "Creative" {
		t.Fatalf("draft %+v", c)
	}
	_, err = e.contract.Create(e.ctx, e.admin, app.ContractInput{EmployeeID: o.ayu, ContractType: "pkwt", StartDate: "2026-01-01"})
	wantCode(t, err, apperr.Validation)
	res, err := e.contract.Action(e.ctx, e.admin, c.ID, app.ContractActionInput{Action: "activate", SignedAt: "2025-12-20"})
	e.must(err)
	if res.Contract.Status != "active" {
		t.Fatal("active")
	}
	ayu, _ := e.people.Get(e.ctx, o.ayu)
	sal, _ := e.d.Store.ActiveSalary(e.ctx, o.ayu)
	if ayu.EmploymentStatus != "contract" || sal == nil || sal.BaseSalary != 8_000_000 {
		t.Fatalf("activation side effects %+v %+v", ayu, sal)
	}
	ex, err := e.contract.Expiring(e.ctx, e.admin, 30)
	e.must(err)
	if len(ex.Contracts) != 1 || len(ex.WithoutContract) != 2 {
		t.Fatalf("expiring %+v", ex)
	}
	renew, err := e.contract.Action(e.ctx, e.admin, c.ID, app.ContractActionInput{Action: "renew", EndDate: "2027-11-10"})
	e.must(err)
	if renew.Contract.StartDate != "2026-11-11" || renew.Contract.Sequence != 2 || renew.Contract.Status != "draft" {
		t.Fatalf("renewal %+v", renew.Contract)
	}
	end, err := e.contract.Action(e.ctx, e.admin, c.ID, app.ContractActionInput{Action: "end", EndDate: "2026-07-01"})
	e.must(err)
	if end.Compensation == nil || *end.Compensation != 4_000_000 {
		t.Fatalf("compensation %v", end.Compensation)
	}
}

// ─── payroll ─────────────────────────────────────────────────

func TestPayrollRunPublishesBalancedTotals(t *testing.T) {
	e := setup(t)
	o := e.org()
	payrollAdmin := e.actor("user-wulan", authn.PermPayroll, authn.PermHRApprove, authn.PermSelfService)
	for _, s := range []app.SalaryInput{
		{EmployeeID: o.wulan, BaseSalary: "18000000", FixedAllowance: "2000000", PTKPStatus: "K/1", EffectiveDate: "2024-01-15"},
		{EmployeeID: o.budi, BaseSalary: "15000000", PTKPStatus: "TK/0", EffectiveDate: "2024-01-15"},
		{EmployeeID: o.ayu, BaseSalary: "10000000", FixedAllowance: "1000000", TransportAllowance: "500000", PTKPStatus: "TK/0", EffectiveDate: "2024-01-15"},
	} {
		_, err := e.payroll.CreateSalary(e.ctx, payrollAdmin, s)
		e.must(err)
	}
	_, err := e.payroll.CreateSalary(e.ctx, e.actor("user-ayu", authn.PermSelfService), app.SalaryInput{EmployeeID: o.ayu, BaseSalary: "1"})
	wantCode(t, err, apperr.Unauthorized)

	// Ayu borrows 3,000,000 over 3 months, approved in August: the first
	// installment falls in September.
	loan, err := app.NewLoans(withClock(e.d, "2026-08-10T03:00:00Z")).Create(e.ctx, e.actor("user-ayu", authn.PermSelfService),
		app.LoanInput{LoanType: "kasbon", Principal: "3000000", InterestRate: "5", Tenor: 3, Purpose: "Servis motor"})
	e.must(err)
	if loan.InterestRate != 0 || loan.MonthlyInstallment != 1_000_000 {
		t.Fatalf("self loans are interest free %+v", loan)
	}
	_, err = e.loans.Create(e.ctx, e.actor("user-ayu", authn.PermSelfService), app.LoanInput{LoanType: "kasbon", Principal: "1000000", Tenor: 2})
	wantCode(t, err, apperr.Validation) // one active loan
	_, err = app.NewLoans(withClock(e.d, "2026-08-10T03:00:00Z")).Decide(e.ctx, payrollAdmin, loan.ID, true, "")
	e.must(err)

	add, err := e.payroll.CreateAddition(e.ctx, payrollAdmin, app.AdditionInput{EmployeeID: o.budi, Amount: "5000000", Label: "Bonus pitching", Month: 9, Year: 2026})
	e.must(err)
	if add.PeriodMonth != 9 {
		t.Fatal("explicit period")
	}
	run, err := e.payroll.CreateRun(e.ctx, payrollAdmin, 9, 2026, "")
	e.must(err)
	if run.RunName != "Payroll September 2026" {
		t.Fatal(run.RunName)
	}
	_, err = e.payroll.CreateRun(e.ctx, payrollAdmin, 9, 2026, "")
	wantCode(t, err, apperr.Conflict)
	_, err = e.payroll.Approve(e.ctx, payrollAdmin, run.ID)
	wantCode(t, err, apperr.Conflict)

	run, skipped, err := e.payroll.Calculate(e.ctx, payrollAdmin, run.ID, false)
	e.must(err)
	if run.Status != "calculated" || run.TotalEmployees != 3 || len(skipped) != 0 {
		t.Fatalf("calculated %+v %v", run, skipped)
	}
	_, slips, err := e.payroll.Run(e.ctx, payrollAdmin, run.ID)
	e.must(err)
	var ayuSlip, budiSlip domain.Payslip
	for _, s := range slips {
		switch s.EmployeeID {
		case o.ayu:
			ayuSlip = s
		case o.budi:
			budiSlip = s
		}
	}
	if ayuSlip.LoanDeduction != 1_000_000 || ayuSlip.WorkingDays != 22 || budiSlip.Bonus != 5_000_000 {
		t.Fatalf("slips ayu %+v budi %+v", ayuSlip, budiSlip)
	}

	// An addition after calculation flags the run.
	extra, err := e.payroll.CreateAddition(e.ctx, payrollAdmin, app.AdditionInput{EmployeeID: o.ayu, Amount: "250000", Label: "Insentif", Month: 9, Year: 2026})
	e.must(err)
	_, err = e.payroll.Approve(e.ctx, payrollAdmin, run.ID)
	wantCode(t, err, apperr.Conflict)
	e.must(e.payroll.DeleteAddition(e.ctx, payrollAdmin, extra.ID))
	_, _, err = e.payroll.Calculate(e.ctx, payrollAdmin, run.ID, false)
	e.must(err)
	run, err = e.payroll.Approve(e.ctx, payrollAdmin, run.ID)
	e.must(err)
	if run.Status != "approved" {
		t.Fatal("approved")
	}
	run, settled, err := e.payroll.MarkPaid(e.ctx, payrollAdmin, run.ID, "BCA")
	e.must(err)
	if run.Status != "paid" || settled != 1 || run.PaidAt == nil {
		t.Fatalf("paid %+v settled %d", run, settled)
	}
	got, _ := e.d.Store.GetLoan(e.ctx, loan.ID)
	if got.RemainingBalance != 2_000_000 || got.PaidAmount != 1_000_000 {
		t.Fatalf("loan after payroll %+v", got)
	}
	_, _, err = e.payroll.MarkPaid(e.ctx, payrollAdmin, run.ID, "")
	wantCode(t, err, apperr.Conflict)
	wantCode(t, e.payroll.DeleteRun(e.ctx, payrollAdmin, run.ID), apperr.Conflict)

	if len(e.events.msgs) != 1 || e.events.msgs[0].subject != events.HRPayrollPaid {
		t.Fatalf("events %+v", e.events.msgs)
	}
	var ev map[string]any
	e.must(json.Unmarshal(e.events.msgs[0].data, &ev))
	num := func(k string) float64 {
		v, ok := domain.ParseMoney(ev[k].(string))
		if !ok {
			t.Fatalf("%s is not a decimal string: %v", k, ev[k])
		}
		return v
	}
	deductions := num("totalPph21") + num("totalLoanDeduction") + num("totalBpjsTkEmployee") + num("totalBpjsKesEmployee") + num("totalTaperaEmployee")
	if ev["runId"] != run.ID || ev["periodMonth"] != 9.0 || ev["employeeCount"] != 3.0 || num("totalGross") != num("totalNet")+deductions || num("totalLoanDeduction") != 1_000_000 {
		t.Fatalf("event %v", ev)
	}

	// Employees see their own slips of paid runs only.
	mine, err := e.payroll.Payslips(e.ctx, e.actor("user-ayu", authn.PermSelfService), domain.PayslipFilter{})
	e.must(err)
	if len(mine) != 1 || mine[0].EmployeeID != o.ayu {
		t.Fatal("own payslips")
	}
	pdf, name, err := e.payroll.PayslipPDF(e.ctx, e.actor("user-ayu", authn.PermSelfService), mine[0].ID)
	e.must(err)
	if name != "Slip-Gaji-Ayu-Lestari-September-2026.pdf" || !strings.HasPrefix(string(pdf), "%PDF-") {
		t.Fatal("payslip pdf", name)
	}
	_, _, err = e.payroll.PayslipPDF(e.ctx, e.actor("user-budi", authn.PermSelfService), mine[0].ID)
	wantCode(t, err, apperr.NotFound)
}

// ─── daily reminders ─────────────────────────────────────────

func TestDailyRemindersAreSentOnce(t *testing.T) {
	e := setup(t)
	o := e.org()
	c, err := e.contract.Create(e.ctx, e.admin, app.ContractInput{EmployeeID: o.ayu, ContractType: "pkwt", StartDate: "2026-01-01", EndDate: "2026-11-04"})
	e.must(err)
	_, err = e.contract.Action(e.ctx, e.admin, c.ID, app.ContractActionInput{Action: "activate"})
	e.must(err)
	runAt := time.Date(2026, 10, 20, 23, 0, 0, 0, time.UTC) // 21 Oct WIB: 14 days before 4 Nov
	rep, err := e.maint.Daily(e.ctx, runAt)
	e.must(err)
	if rep.ContractReminders != 1 {
		t.Fatalf("report %+v", rep)
	}
	if e.identity.count("HR_CONTRACT_EXPIRING") != 1 {
		t.Fatal("HR (Wulan, department HR) is notified")
	}
	rep, err = e.maint.Daily(e.ctx, runAt)
	e.must(err)
	if rep.ContractReminders != 0 || e.identity.count("HR_CONTRACT_EXPIRING") != 1 {
		t.Fatal("a redelivered tick sends nothing")
	}
}

// ─── seed ────────────────────────────────────────────────────

func TestSeedIsIdempotent(t *testing.T) {
	e := setup(t)
	e.d.Events = events.Noop{}
	users := []app.UserInfo{{ID: "user-admin", Email: "admin@mixist.id"}, {ID: "user-tim", Email: "tim@mixist.id"}}
	e.must(seed.New(e.d, docs.Renderer{CompanyName: "Mixist"}).Run(e.ctx, users))
	count := func() []int {
		emps, _, err := e.d.Store.ListEmployees(e.ctx, domain.EmployeeFilter{IncludeInactive: true})
		e.must(err)
		_, atts, err := e.d.Store.ListAttendance(e.ctx, domain.AttendanceFilter{Limit: 1})
		e.must(err)
		runs, err := e.d.Store.ListPayrollRuns(e.ctx, 0, "")
		e.must(err)
		return []int{len(emps), atts, len(runs)}
	}
	first := count()
	e.must(seed.New(e.d, docs.Renderer{CompanyName: "Mixist"}).Run(e.ctx, users))
	second := count()
	if first[0] != 34 || first[1] < 5000 || first[2] != 15 || first[0] != second[0] || first[1] != second[1] || first[2] != second[2] {
		t.Fatalf("seed counts %v then %v", first, second)
	}
	runs, _ := e.d.Store.ListPayrollRuns(e.ctx, 0, "")
	states := map[string]int{}
	for _, r := range runs {
		states[r.Status]++
	}
	if states["paid"] != 12 || states["approved"] != 1 || states["calculated"] != 1 || states["draft"] != 1 {
		t.Fatalf("seeded run states %v", states)
	}
	admin, err := e.d.Store.EmployeeIDByUser(e.ctx, "user-admin")
	e.must(err)
	tim, err := e.d.Store.EmployeeIDByUser(e.ctx, "user-tim")
	e.must(err)
	a, _ := e.d.Store.GetEmployee(e.ctx, admin)
	g, _ := e.d.Store.GetEmployee(e.ctx, tim)
	if a == nil || a.DepartmentName != "Management" || g == nil || g.DepartmentName != "Social Media" {
		t.Fatal("seed links admin@mixist.id and tim@mixist.id")
	}
}

func TestCandidateHiredCreatesProbationEmployee(t *testing.T) {
	e := setup(t)
	o := e.org()
	e.identity.approvers = []string{"user-hrboss", "user-wulan"}
	pos, err := e.people.SavePosition(e.ctx, e.admin, "", app.PositionInput{Title: "Graphic Designer", DepartmentID: o.creative, IsActive: true})
	e.must(err)
	life := app.NewLifecycle(e.d)
	start, salary := "2026-11-02", "8500000.00"
	hired := app.CandidateHired{CandidateID: "cand-1", FullName: "Rina Putri", Email: "Rina@Mail.com", Phone: "0812", JobTitle: "graphic designer",
		DepartmentID: &o.creative, StartDate: &start, BaseSalary: &salary, HiredAt: time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC)}
	e.must(life.OnCandidateHired(e.ctx, hired))
	e.must(life.OnCandidateHired(e.ctx, hired)) // redelivery

	list, total, err := e.people.List(e.ctx, domain.EmployeeFilter{Search: "Rina"})
	e.must(err)
	if total != 1 || len(list) != 1 {
		t.Fatalf("want one employee, got %d", total)
	}
	emp := list[0]
	if emp.EmploymentStatus != "probation" || !emp.IsActive || emp.JoinDate != start || emp.Email != "rina@mail.com" ||
		deref(emp.DepartmentID) != o.creative || deref(emp.PositionID) != pos.ID || emp.EmployeeNo == "" {
		t.Fatalf("employee %+v", emp)
	}
	tasks, err := life.Tasks(e.ctx, e.admin, emp.ID)
	e.must(err)
	if len(tasks) != len(app.OnboardingTemplate) || tasks[0].DueDate == nil {
		t.Fatalf("onboarding tasks %d", len(tasks))
	}
	salaries, err := e.d.Store.ListSalaries(e.ctx, emp.ID, false)
	e.must(err)
	if len(salaries) != 0 {
		t.Fatal("salary must not be auto-created")
	}
	if n := e.identity.count(app.NotifNewHire); n != 1 {
		t.Fatalf("new hire notices %d", n)
	}
	last := e.identity.sent[len(e.identity.sent)-1]
	if strings.Join(last.UserIDs, ",") != "user-wulan,user-hrboss" || !strings.Contains(last.Body, "Rp8.500.000") {
		t.Fatalf("notice %+v", last)
	}

	// Unknown department, no matching position, no start date: still created.
	ghost := "dept-gone"
	e.must(life.OnCandidateHired(e.ctx, app.CandidateHired{CandidateID: "cand-2", FullName: "Dodi", JobTitle: "Copywriter", DepartmentID: &ghost,
		HiredAt: time.Date(2026, 10, 20, 18, 0, 0, 0, time.UTC)}))
	list, _, err = e.people.List(e.ctx, domain.EmployeeFilter{Search: "Dodi"})
	e.must(err)
	if len(list) != 1 || list[0].DepartmentID != nil || list[0].PositionID != nil || list[0].JoinDate != "2026-10-21" {
		t.Fatalf("employee %+v", list)
	}
	// Malformed payloads are skipped.
	e.must(life.OnCandidateHired(e.ctx, app.CandidateHired{FullName: "Tanpa ID"}))
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
