package app

import (
	"context"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
)

// ─── Announcements ───────────────────────────────────────────

// Announcements are HR posts to everyone or to departments.
type Announcements struct{ d Deps }

func NewAnnouncements(d Deps) *Announcements { return &Announcements{d: d} }

func (s *Announcements) List(ctx context.Context, a Actor, status string) ([]domain.Announcement, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, err
	}
	return s.d.Store.ListAnnouncements(ctx, status)
}

func (s *Announcements) Get(ctx context.Context, a Actor, id string) (*domain.Announcement, error) {
	if err := require(a, authn.PermHRRead); err != nil {
		return nil, err
	}
	an, err := s.d.Store.GetAnnouncement(ctx, id)
	if err != nil {
		return nil, err
	}
	if an == nil {
		return nil, notFound("Pengumuman tidak ditemukan.")
	}
	return an, nil
}

func (s *Announcements) Save(ctx context.Context, a Actor, an domain.Announcement) (*domain.Announcement, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	an.Title = strings.TrimSpace(an.Title)
	if an.Title == "" {
		return nil, invalid("title", "Judul pengumuman wajib diisi.")
	}
	if an.Status == "" {
		an.Status = "draft"
	}
	if !slices.Contains([]string{"draft", "published", "archived"}, an.Status) {
		return nil, invalid("status", "Status pengumuman tidak valid.")
	}
	if an.TargetScope == "" {
		an.TargetScope = "global"
	}
	switch an.TargetScope {
	case "global":
		an.DepartmentIDs = nil
	case "departments":
		if len(an.DepartmentIDs) == 0 {
			return nil, invalid("departmentIds", "Pilih minimal satu departemen atau ubah target ke global.")
		}
	default:
		return nil, invalid("targetScope", "Target pengumuman tidak valid.")
	}
	if an.PublishAt != nil && an.ExpiresAt != nil && !an.ExpiresAt.After(*an.PublishAt) {
		return nil, invalid("expiresAt", "Tanggal berakhir harus setelah tanggal tayang.")
	}
	insert := an.ID == ""
	if insert {
		an.ID = newID()
	}
	ok, err := s.d.Store.SaveAnnouncement(ctx, an, insert, a.UserRef())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, notFound("Pengumuman tidak ditemukan.")
	}
	return s.d.Store.GetAnnouncement(ctx, an.ID)
}

func (s *Announcements) Delete(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ok, err := s.d.Store.DeleteAnnouncement(ctx, id)
	if err == nil && !ok {
		return notFound("Pengumuman tidak ditemukan.")
	}
	return err
}

// Feed is what the caller sees: published, live and targeted at them.
func (s *Announcements) Feed(ctx context.Context, a Actor) ([]domain.Announcement, int, error) {
	id, err := self(a)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.d.Store.AnnouncementFeed(ctx, id, s.d.now())
	if err != nil {
		return nil, 0, err
	}
	unread := 0
	for _, r := range rows {
		if !r.IsRead {
			unread++
		}
	}
	return rows, unread, nil
}

// Open returns a feed item and marks it read.
func (s *Announcements) Open(ctx context.Context, a Actor, id string) (*domain.Announcement, error) {
	rows, _, err := s.Feed(ctx, a)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.ID == id {
			if err := s.d.Store.MarkAnnouncementRead(ctx, id, a.EmployeeID); err != nil {
				return nil, err
			}
			r.IsRead = true
			return &r, nil
		}
	}
	return nil, notFound("Pengumuman tidak ditemukan.")
}

// ─── Onboarding / offboarding ────────────────────────────────

// Lifecycle is the onboarding checklist and the offboarding process.
type Lifecycle struct{ d Deps }

func NewLifecycle(d Deps) *Lifecycle { return &Lifecycle{d: d} }

var onboardingCategories = []string{"admin", "it", "hr", "manager", "general"}

// OnboardingTemplate is WHITE's default first-month checklist (days after
// the join date).
var OnboardingTemplate = []struct {
	Name, Category string
	Priority, Days int
}{
	{"Tanda tangan kontrak kerja", "hr", 1, 0},
	{"Kumpulkan KTP, NPWP, KK dan rekening bank", "admin", 1, 3},
	{"Daftarkan BPJS Ketenagakerjaan dan Kesehatan", "hr", 1, 14},
	{"Buat akun email, Slack dan akses WHITE Platform", "it", 1, 1},
	{"Serahkan laptop dan perangkat kerja", "it", 2, 1},
	{"Perkenalan tim dan budaya kerja WHITE", "manager", 2, 2},
	{"Briefing klien dan proyek yang dipegang", "manager", 2, 5},
	{"Atur jadwal shift dan absensi", "hr", 2, 1},
	{"Review masa percobaan bulan pertama", "manager", 3, 30},
}

func (l *Lifecycle) canSee(a Actor, employeeID string) error {
	if employeeID == a.EmployeeID && a.Can(authn.PermSelfService) {
		return nil
	}
	return require(a, authn.PermHRRead)
}

func (l *Lifecycle) Tasks(ctx context.Context, a Actor, employeeID string) ([]domain.OnboardingTask, error) {
	if err := l.canSee(a, employeeID); err != nil {
		return nil, err
	}
	return l.d.Store.ListOnboardingTasks(ctx, employeeID)
}

func (l *Lifecycle) ApplyTemplate(ctx context.Context, a Actor, employeeID string) ([]domain.OnboardingTask, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	emp, err := l.d.Store.GetEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, notFound(msgEmployeeNotFound)
	}
	err = l.d.Store.WithTx(ctx, func(tx domain.Store) error { return applyOnboardingTemplate(ctx, tx, *emp, a.UserRef()) })
	if err != nil {
		return nil, err
	}
	return l.d.Store.ListOnboardingTasks(ctx, employeeID)
}

// applyOnboardingTemplate adds the template tasks the employee lacks, due
// relative to the join date.
func applyOnboardingTemplate(ctx context.Context, tx domain.Store, emp domain.Employee, actor *string) error {
	existing, err := tx.ListOnboardingTasks(ctx, emp.ID)
	if err != nil {
		return err
	}
	for _, t := range OnboardingTemplate {
		if slices.ContainsFunc(existing, func(e domain.OnboardingTask) bool { return e.TaskName == t.Name }) {
			continue
		}
		due := hris.AddDaysISO(emp.JoinDate, t.Days)
		if err := tx.InsertOnboardingTask(ctx, domain.OnboardingTask{ID: newID(), EmployeeID: emp.ID, TaskName: t.Name,
			Category: t.Category, Priority: t.Priority, DueDate: &due}, actor); err != nil {
			return err
		}
	}
	return nil
}

func (l *Lifecycle) AddTask(ctx context.Context, a Actor, t domain.OnboardingTask) (*domain.OnboardingTask, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	t.TaskName = strings.TrimSpace(t.TaskName)
	if t.TaskName == "" {
		return nil, invalid("taskName", "Nama tugas wajib diisi.")
	}
	if t.Category == "" {
		t.Category = "general"
	}
	if !slices.Contains(onboardingCategories, t.Category) {
		return nil, invalid("category", "Kategori tidak valid.")
	}
	if t.Priority == 0 {
		t.Priority = 2
	}
	if t.Priority < 1 || t.Priority > 3 {
		return nil, invalid("priority", "Prioritas 1 sampai 3.")
	}
	if _, err := optDate("dueDate", deref(t.DueDate)); err != nil {
		return nil, err
	}
	t.ID = newID()
	if err := l.d.Store.InsertOnboardingTask(ctx, t, a.UserRef()); err != nil {
		return nil, err
	}
	return l.d.Store.GetOnboardingTask(ctx, t.ID)
}

type TaskUpdate struct {
	IsCompleted     *bool
	Notes, TaskName string
	DueDate         string
}

func (l *Lifecycle) UpdateTask(ctx context.Context, a Actor, id string, in TaskUpdate) (*domain.OnboardingTask, error) {
	t, err := l.d.Store.GetOnboardingTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, notFound("Tugas tidak ditemukan.")
	}
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	if in.IsCompleted != nil {
		t.IsCompleted = *in.IsCompleted
	}
	if n := strings.TrimSpace(in.TaskName); n != "" {
		t.TaskName = n
	}
	if d, err := optDate("dueDate", in.DueDate); err != nil {
		return nil, err
	} else if d != nil {
		t.DueDate = d
	}
	t.Notes = strings.TrimSpace(in.Notes)
	t.CompletedBy = a.EmployeeRef()
	if _, err := l.d.Store.UpdateOnboardingTask(ctx, *t); err != nil {
		return nil, err
	}
	return l.d.Store.GetOnboardingTask(ctx, id)
}

func (l *Lifecycle) DeleteTask(ctx context.Context, a Actor, id string) error {
	if err := require(a, authn.PermHRWrite); err != nil {
		return err
	}
	ok, err := l.d.Store.DeleteOnboardingTask(ctx, id)
	if err == nil && !ok {
		return notFound("Tugas tidak ditemukan.")
	}
	return err
}

var (
	resignationTypes    = []string{"voluntary", "termination", "layoff", "end_of_contract"}
	offboardingStatuses = []string{"submitted", "notice_period", "exit_interview", "completed"}
)

func (l *Lifecycle) Offboarding(ctx context.Context, a Actor, employeeID string) (*domain.Offboarding, error) {
	if err := l.canSee(a, employeeID); err != nil {
		return nil, err
	}
	o, err := l.d.Store.GetOffboarding(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, notFound("Belum ada proses offboarding.")
	}
	return o, nil
}

func (l *Lifecycle) StartOffboarding(ctx context.Context, a Actor, o domain.Offboarding) (*domain.Offboarding, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	if !slices.Contains(resignationTypes, o.ResignationType) {
		return nil, invalid("resignationType", "Jenis pengunduran diri tidak valid.")
	}
	var err error
	if o.ResignationDate, err = reqDate("resignationDate", o.ResignationDate); err != nil {
		return nil, err
	}
	if o.LastWorkingDay, err = reqDate("lastWorkingDay", o.LastWorkingDay); err != nil {
		return nil, err
	}
	if o.LastWorkingDay < o.ResignationDate {
		return nil, invalid("lastWorkingDay", "Hari kerja terakhir tidak boleh sebelum tanggal pengunduran diri.")
	}
	if _, err := l.d.Store.GetEmployee(ctx, o.EmployeeID); err != nil {
		return nil, err
	}
	o.ID, o.Reason = newID(), strings.TrimSpace(o.Reason)
	if err := l.d.Store.InsertOffboarding(ctx, o, a.UserRef()); err != nil {
		return nil, uniqueErr(err, "Karyawan ini sudah dalam proses offboarding.")
	}
	return l.d.Store.GetOffboarding(ctx, o.EmployeeID)
}

type OffboardingUpdate struct {
	Status, ClearanceType, ExitInterviewNotes, Notes string
	Cleared                                          *bool
}

// UpdateOffboarding moves the status and clearances; completing it sets the
// employee resigned (or terminated) and inactive from the last working day.
func (l *Lifecycle) UpdateOffboarding(ctx context.Context, a Actor, employeeID string, in OffboardingUpdate) (*domain.Offboarding, error) {
	if err := require(a, authn.PermHRWrite); err != nil {
		return nil, err
	}
	o, err := l.d.Store.GetOffboarding(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, notFound("Belum ada proses offboarding.")
	}
	if in.Status != "" {
		if !slices.Contains(offboardingStatuses, in.Status) {
			return nil, invalid("status", "Status offboarding tidak valid.")
		}
		o.Status = in.Status
	}
	if in.Cleared != nil {
		switch in.ClearanceType {
		case "hr":
			o.ClearanceHR = *in.Cleared
		case "it":
			o.ClearanceIT = *in.Cleared
		case "finance":
			o.ClearanceFinance = *in.Cleared
		case "manager":
			o.ClearanceManager = *in.Cleared
		default:
			return nil, invalid("clearanceType", "Jenis clearance tidak valid.")
		}
	}
	if v := strings.TrimSpace(in.ExitInterviewNotes); v != "" {
		o.ExitInterviewNotes = v
	}
	if v := strings.TrimSpace(in.Notes); v != "" {
		o.Notes = v
	}
	completing := o.Status == "completed" && o.CompletedAt == nil
	if completing {
		if !(o.ClearanceHR && o.ClearanceIT && o.ClearanceFinance && o.ClearanceManager) {
			return nil, invalid("status", "Semua clearance (HR, IT, keuangan, atasan) harus selesai sebelum offboarding ditutup.")
		}
		now := l.d.now()
		o.CompletedAt = &now
	}
	err = l.d.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateOffboarding(ctx, *o, a.UserRef()); err != nil {
			return err
		}
		if !completing {
			return nil
		}
		status := "resigned"
		if o.ResignationType == "termination" || o.ResignationType == "layoff" {
			status = "terminated"
		}
		last := o.LastWorkingDay
		if err := tx.SetEmployeeStatus(ctx, employeeID, status, false, &last, a.UserRef()); err != nil {
			return err
		}
		return tx.InsertHistory(ctx, domain.HistoryEntry{ID: newID(), EmployeeID: employeeID, ChangeType: "offboarding",
			Notes: "Offboarding selesai (" + o.ResignationType + ")", NewStatus: &status, CreatedBy: a.UserRef()})
	})
	if err != nil {
		return nil, err
	}
	return l.d.Store.GetOffboarding(ctx, employeeID)
}
