package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Tasks ports NüHabit payroll/depttasks*.go: recurring department tasks,
// the occurrence state machine with weighted sub-tasks, history,
// attachments, head-of-division review, KPI contributions and the views
// (today/week/month, team board, KPI view, monitor).
type Tasks struct {
	Deps
	improvements *Improvements
}

func NewTasks(d Deps, imp *Improvements) *Tasks { return &Tasks{d, imp} }

const (
	msgOccurrenceGone = "Kemunculan task tidak ditemukan"
	msgTaskGone       = "Task tidak ditemukan"
	msgBadMonth       = "Bulan tidak valid, pakai format YYYY-MM"
	// MaxAttachmentBytes is the largest file an occurrence takes.
	MaxAttachmentBytes = 10 << 20
)

// headsDept: the hr department head, or (NüHabit's rule) a member of the
// department with active direct reports.
func (a *Actor) headsDept(departmentID string) bool {
	return a.HeadOf[departmentID] || (len(a.Reports) > 0 && a.DepartmentID() == departmentID)
}

// managesDept: HR readers see every department's board.
func (a *Actor) managesDept(departmentID string) bool { return a.HRRead || a.headsDept(departmentID) }

// TaskView is one occurrence with names and what the reader may do.
type TaskView struct {
	domain.TaskItem
	DepartmentName string
	AssigneeName   string
	Progress       int
	Overdue        bool
	Mine           bool
	CanWork        bool
	CanReview      bool
	CanManage      bool
}

func (t *Tasks) finish(a *Actor, ppl *people, it domain.TaskItem) TaskView {
	v := TaskView{TaskItem: it, DepartmentName: ppl.deptName(&it.DepartmentID), AssigneeName: ppl.name(it.AssigneeID)}
	var manual *float64
	if it.ProgressPercent != nil {
		p := float64(*it.ProgressPercent)
		manual = &p
	}
	v.Progress = domain.EffectiveProgress(it.Status, manual, it.SubtaskPercent())
	v.Overdue = domain.OccurrenceOverdue(it.Date, it.Status, domain.DayKey(t.now(), domain.Zone(it.Timezone)))
	if a == nil {
		return v
	}
	assignedToMe := it.AssigneeID != nil && a.IsEmployee(*it.AssigneeID)
	inMyDept := a.DepartmentID() == it.DepartmentID && a.DepartmentID() != ""
	mineFiled := it.CreatorUserID != nil && *it.CreatorUserID == a.UserID
	v.Mine = assignedToMe || (it.AssigneeID == nil && inMyDept) || mineFiled
	v.CanWork = a.HRWrite || assignedToMe || (it.AssigneeID == nil && inMyDept)
	v.CanManage = a.HRWrite || a.headsDept(it.DepartmentID) || (mineFiled && !it.ReviewRequired)
	v.CanReview = it.ReviewRequired && (a.HRWrite || a.headsDept(it.DepartmentID))
	return v
}

// ensureOccurrences generates the missing occurrences of the departments'
// active tasks in [start, end] (idempotent; nil departments is all).
func (t *Tasks) ensureOccurrences(ctx context.Context, departmentIDs []string, start, end string) error {
	if departmentIDs != nil && len(departmentIDs) == 0 {
		return nil
	}
	tasks, err := t.Store.ActiveTaskSchedules(ctx, departmentIDs)
	if err != nil {
		return err
	}
	var taskIDs, dates []string
	for _, task := range tasks {
		for _, d := range domain.OccurrenceDates(task.TaskSchedule, start, end) {
			taskIDs, dates = append(taskIDs, task.TaskID), append(dates, d)
		}
	}
	return t.Store.InsertOccurrences(ctx, taskIDs, dates)
}

// Materialize creates yesterday's to tomorrow's occurrences of every
// active task so a daily task exists each morning (hourly tick).
func (t *Tasks) Materialize(ctx context.Context, now time.Time) error {
	start := domain.DayKey(now.AddDate(0, 0, -1), domain.Jakarta)
	end := domain.DayKey(now.AddDate(0, 0, 1), domain.Zone("Asia/Jayapura"))
	return t.ensureOccurrences(ctx, nil, start, end)
}

// ─── create, edit, deactivate ────────────────────────────────

// TaskInput is POST /tasks.
type TaskInput struct {
	DepartmentID    string
	AssigneeID      string
	Body            domain.TaskBody
	Priority        string
	KPIIndicatorID  *string
	KPIContribution *float64
}

// Create files a task: HR anywhere, a head in their own department, anyone
// else a personal to-do (assigned to themselves, no review).
func (t *Tasks) Create(ctx context.Context, in TaskInput) (string, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return "", err
	}
	task, msg := domain.NormalizeTask(in.Body)
	if msg != "" {
		return "", apperr.New(apperr.Validation, msg)
	}
	dept := a.DepartmentID()
	if a.HRWrite && in.DepartmentID != "" {
		dept = in.DepartmentID
	}
	if dept == "" {
		return "", apperr.New(apperr.Validation, "Departemen tidak diketahui")
	}
	self := in.AssigneeID != "" && a.IsEmployee(in.AssigneeID)
	if !a.HRWrite {
		if !a.headsDept(dept) && !self {
			return "", apperr.New(apperr.Unauthorized, "Hanya HRD atau atasan (kepala tim) yang boleh membuat task departemen")
		}
		if a.DepartmentID() != dept && !a.HeadOf[dept] {
			return "", apperr.New(apperr.Unauthorized, "Hanya boleh membuat task untuk departemen sendiri")
		}
	}
	ppl, err := t.people(ctx)
	if err != nil {
		return "", err
	}
	if _, ok := ppl.depts[dept]; !ok {
		return "", apperr.New(apperr.NotFound, msgDeptNotFound)
	}
	var assignee *string
	if in.AssigneeID != "" {
		e, ok := ppl.employee(in.AssigneeID)
		if !ok || !e.Active || !e.InDepartment(dept) {
			return "", apperr.New(apperr.Validation, "Penanggung jawab harus karyawan aktif di departemen yang sama")
		}
		assignee = &e.ID
	}
	if in.Priority == "" {
		in.Priority = "normal"
	}
	if !domain.ValidPriority(in.Priority) {
		return "", apperr.New(apperr.Validation, "Prioritas tidak valid")
	}
	if err := t.checkIndicator(ctx, in.KPIIndicatorID); err != nil {
		return "", err
	}
	createdBy := a.Name
	if createdBy == "" {
		createdBy = "-"
	}
	var creatorEmp *string
	if a.EmployeeID() != "" {
		creatorEmp = strPtr(a.EmployeeID())
	}
	row := domain.Task{ID: newID(), DepartmentID: dept, AssigneeEmployeeID: assignee, Title: task.Title, Description: task.Description,
		Recurrence: task.Schedule.Recurrence, WeeklyDay: task.Schedule.WeeklyDay, MonthlyDay: task.Schedule.MonthlyDay,
		DueDate: task.Schedule.DueDate, Priority: in.Priority, KPIIndicatorID: trimToNil(in.KPIIndicatorID), KPIContribution: in.KPIContribution,
		Timezone: "Asia/Jakarta", ReviewRequired: !self, CreatedByEmployeeID: creatorEmp, CreatedByUserID: a.by(),
		CreatedByName: createdBy, CreatedAt: t.now()}
	err = t.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertTask(ctx, row); err != nil {
			return err
		}
		for i, st := range task.Subtasks {
			if err := tx.InsertSubtask(ctx, row.ID, st, i); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	today := t.today()
	if err := t.ensureOccurrences(ctx, []string{dept}, today, today); err != nil {
		return "", err
	}
	return row.ID, nil
}

func (t *Tasks) checkIndicator(ctx context.Context, id *string) error {
	if id == nil || *id == "" {
		return nil
	}
	ind, err := t.Store.GetIndicator(ctx, *id)
	if err != nil {
		return err
	}
	if ind == nil || !ind.IsActive {
		return apperr.New(apperr.Validation, msgIndicatorGone)
	}
	return nil
}

// assertManager: HR, the head of the task's department, or the creator
// of a personal (review-free) task.
func (t *Tasks) assertManager(ctx context.Context, a *Actor, id, verb string) (*domain.Task, error) {
	task, err := t.Store.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil || !task.IsActive {
		return nil, apperr.New(apperr.NotFound, msgTaskGone)
	}
	if a.HRWrite || (!task.ReviewRequired && task.CreatedByUserID != nil && *task.CreatedByUserID == a.UserID) || a.headsDept(task.DepartmentID) {
		return task, nil
	}
	return nil, apperr.New(apperr.Unauthorized, "Hanya HRD atau atasan departemen ini yang boleh "+verb+" task")
}

// TaskPatch edits a task; nil keeps a value, the *Set flags clear one.
type TaskPatch struct {
	Title, Description, Priority *string
	AssigneeID                   *string
	AssigneeSet                  bool
	KPIIndicatorID               *string
	KPISet                       bool
	KPIContribution              *float64
	KPIContributionSet           bool
}

func (t *Tasks) Update(ctx context.Context, id string, p TaskPatch) error {
	a, err := t.actor(ctx)
	if err != nil {
		return err
	}
	task, err := t.assertManager(ctx, a, id, "mengubah")
	if err != nil {
		return err
	}
	if p.Title != nil {
		title := strings.TrimSpace(*p.Title)
		if len([]rune(title)) < 3 {
			return apperr.New(apperr.Validation, "Judul task minimal 3 karakter")
		}
		task.Title = title
	}
	if p.Description != nil {
		task.Description = trimToNil(p.Description)
	}
	if p.Priority != nil {
		if !domain.ValidPriority(*p.Priority) {
			return apperr.New(apperr.Validation, "Prioritas tidak valid")
		}
		task.Priority = *p.Priority
	}
	if p.AssigneeSet {
		task.AssigneeEmployeeID = trimToNil(p.AssigneeID)
		if task.AssigneeEmployeeID != nil {
			ppl, err := t.people(ctx)
			if err != nil {
				return err
			}
			e, ok := ppl.employee(*task.AssigneeEmployeeID)
			if !ok || !e.Active || !e.InDepartment(task.DepartmentID) {
				return apperr.New(apperr.Validation, "Penanggung jawab harus karyawan aktif di departemen yang sama")
			}
		}
	}
	if p.KPISet {
		if err := t.checkIndicator(ctx, p.KPIIndicatorID); err != nil {
			return err
		}
		task.KPIIndicatorID = trimToNil(p.KPIIndicatorID)
	}
	if p.KPIContributionSet {
		task.KPIContribution = p.KPIContribution
	}
	task.UpdatedBy = a.by()
	return t.Store.UpdateTask(ctx, *task)
}

func (t *Tasks) Deactivate(ctx context.Context, id string) error {
	a, err := t.actor(ctx)
	if err != nil {
		return err
	}
	task, err := t.assertManager(ctx, a, id, "menonaktifkan")
	if err != nil {
		return err
	}
	task.IsActive, task.UpdatedBy = false, a.by()
	return t.Store.UpdateTask(ctx, *task)
}

// ─── occurrence actions ──────────────────────────────────────

// OccurrenceAction is PATCH /occurrences/{id}.
type OccurrenceAction struct {
	Action          string
	Notes           *string
	Note            *string
	ProgressPercent *int
	SubtaskID       string
	Checked         bool
}

// ActionResult carries the message and, for sub-task ticks, the progress.
type ActionResult struct {
	Message  string
	Progress *float64
	Complete bool
}

// assertWorker: the assignee, or any active department member when the
// task has none; HR writers as the administrative fallback. Approved is final.
func (t *Tasks) assertWorker(a *Actor, occ domain.Occurrence) error {
	if occ.Status == domain.StatusApproved {
		return apperr.New(apperr.Validation, "Sudah disetujui, tidak bisa diubah")
	}
	if a.HRWrite {
		return nil
	}
	if a.EmployeeID() == "" {
		return errNoEmployee
	}
	if occ.AssigneeID != nil {
		if !a.IsEmployee(*occ.AssigneeID) {
			return apperr.New(apperr.Unauthorized, "Hanya penanggung jawab task ini yang boleh mengerjakannya")
		}
		return nil
	}
	if a.DepartmentID() != occ.DepartmentID {
		return apperr.New(apperr.Unauthorized, "Hanya anggota departemen ini yang boleh mengerjakannya")
	}
	return nil
}

// assertParticipant: whoever may work it (approved or not), its head, HR.
func (t *Tasks) assertParticipant(a *Actor, occ domain.Occurrence) error {
	open := occ
	open.Status = domain.StatusPending
	if t.assertWorker(a, open) == nil || a.headsDept(occ.DepartmentID) {
		return nil
	}
	return apperr.New(apperr.Unauthorized, "Hanya yang terlibat di task ini yang bisa berkomentar")
}

// assertReopener: the worker reopens their own done or sent-back row; an
// approved row only HR or the department head.
func (t *Tasks) assertReopener(a *Actor, occ domain.Occurrence) error {
	if occ.Status != domain.StatusApproved {
		return t.assertWorker(a, occ)
	}
	if a.HRWrite || a.headsDept(occ.DepartmentID) {
		return nil
	}
	return apperr.New(apperr.Unauthorized, "Hanya Head Division departemen ini (atau HRD) yang boleh membuka task yang sudah disetujui")
}

func (t *Tasks) Act(ctx context.Context, id string, in OccurrenceAction) (*ActionResult, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return nil, err
	}
	occ, err := t.Store.GetOccurrence(ctx, id)
	if err != nil {
		return nil, err
	}
	if occ == nil {
		return nil, apperr.New(apperr.NotFound, msgOccurrenceGone)
	}
	myName := a.Name
	if myName == "" {
		myName = "HRD"
	}
	var me *string
	if a.EmployeeID() != "" {
		me = strPtr(a.EmployeeID())
	}
	now := t.now()
	log := func(tx domain.Store, kind string, note *string, percent *int, detail map[string]any) error {
		return tx.InsertTaskUpdate(ctx, domain.TaskUpdate{ID: newID(), OccurrenceID: id, Kind: kind, Note: note, ProgressPercent: percent,
			AuthorEmployeeID: me, AuthorUserID: a.by(), AuthorName: myName, Detail: detail, CreatedAt: now})
	}
	save := func(kind string, note *string, percent *int, detail map[string]any, after func(tx domain.Store) error) error {
		return t.Store.WithTx(ctx, func(tx domain.Store) error {
			if err := tx.SaveOccurrence(ctx, *occ); err != nil {
				return err
			}
			if after != nil {
				if err := after(tx); err != nil {
					return err
				}
			}
			return log(tx, kind, note, percent, detail)
		})
	}
	record := func(tx domain.Store) error {
		return tx.RecordContribution(ctx, occ.ID, occ.TaskID, domain.PeriodOf(occ.Date))
	}
	drop := func(tx domain.Store) error { return tx.DropContribution(ctx, occ.ID) }

	switch in.Action {
	case domain.ActComment:
		if err := t.assertParticipant(a, *occ); err != nil {
			return nil, err
		}
		note := trimToNil(in.Note)
		if note == nil {
			return nil, apperr.New(apperr.Validation, "Komentar wajib diisi")
		}
		return &ActionResult{Message: "Komentar dikirim"}, log(t.Store, domain.UpdateComment, note, nil, nil)

	case domain.ActCheckSubtask:
		if err := t.assertWorker(a, *occ); err != nil {
			return nil, err
		}
		return t.checkSubtask(ctx, a, occ, in, myName, me)

	case domain.ActStart:
		if err := t.assertWorker(a, *occ); err != nil {
			return nil, err
		}
		if occ.Status == domain.StatusDone {
			return nil, apperr.New(apperr.Validation, "Task sudah ditandai selesai, menunggu review")
		}
		occ.Status = domain.StatusInProgress
		if occ.StartedAt == nil {
			occ.StartedAt = &now
		}
		return &ActionResult{Message: "Task mulai dikerjakan"}, save(domain.UpdateStatus, nil, nil, map[string]any{"status": occ.Status}, nil)

	case domain.ActProgress:
		if err := t.assertWorker(a, *occ); err != nil {
			return nil, err
		}
		if occ.Status == domain.StatusDone {
			return nil, apperr.New(apperr.Validation, "Task sudah ditandai selesai, menunggu review")
		}
		note := trimToNil(in.Note)
		if note == nil {
			note = trimToNil(in.Notes)
		}
		if in.ProgressPercent != nil {
			n, err := t.Store.CountSubtasks(ctx, occ.TaskID)
			if err != nil {
				return nil, err
			}
			if n > 0 {
				return nil, apperr.New(apperr.Validation, "Task ini memakai sub-task, progres dihitung dari ceklis sub-task")
			}
			if *in.ProgressPercent < 0 || *in.ProgressPercent > 99 {
				return nil, apperr.New(apperr.Validation, "Progres 0–99% (untuk 100% pakai Tandai Selesai)")
			}
			occ.ProgressPercent = in.ProgressPercent
		}
		if note == nil && in.ProgressPercent == nil {
			return nil, apperr.New(apperr.Validation, "Isi catatan atau persen progres")
		}
		occ.Status = domain.StatusInProgress
		if occ.StartedAt == nil {
			occ.StartedAt = &now
		}
		return &ActionResult{Message: "Progres dicatat, status Sedang dikerjakan"}, save(domain.UpdateProgress, note, in.ProgressPercent, nil, nil)

	case domain.ActReopen:
		if err := t.assertReopener(a, *occ); err != nil {
			return nil, err
		}
		occ.Status, occ.DoneAt, occ.DoneBy, occ.DoneByName, occ.ProgressPercent = domain.StatusPending, nil, nil, nil, nil
		occ.ReviewNotes, occ.ReviewedBy, occ.ReviewedByName, occ.ReviewedAt = nil, nil, nil, nil
		return &ActionResult{Message: "Task dibuka lagi"}, save(domain.UpdateStatus, trimToNil(in.Note), nil, map[string]any{"status": occ.Status}, drop)

	case domain.ActDone:
		if err := t.assertWorker(a, *occ); err != nil {
			return nil, err
		}
		occ.DoneAt, occ.DoneBy, occ.DoneByName, occ.ReviewNotes = &now, me, &myName, nil
		if !occ.ReviewRequired {
			// A personal task: done is final, as if its owner approved it.
			occ.Status, occ.ReviewedBy, occ.ReviewedByName, occ.ReviewedAt = domain.StatusApproved, me, &myName, &now
			return &ActionResult{Message: "Tugas selesai"}, save(domain.UpdateStatus, nil, nil, map[string]any{"status": occ.Status}, record)
		}
		occ.Status, occ.ReviewedBy, occ.ReviewedByName, occ.ReviewedAt = domain.StatusDone, nil, nil, nil
		if err := save(domain.UpdateStatus, nil, nil, map[string]any{"status": occ.Status}, nil); err != nil {
			return nil, err
		}
		t.notifyReviewers(ctx, *occ, myName)
		return &ActionResult{Message: "Task ditandai selesai, menunggu review Head Division"}, nil

	case domain.ActApprove, domain.ActReject:
		if a.EmployeeID() == "" && !a.HRWrite {
			return nil, apperr.New(apperr.Unauthorized, "Akun ini tidak terhubung ke data karyawan, tidak bisa mereview task")
		}
		if !a.HRWrite && !a.headsDept(occ.DepartmentID) {
			return nil, apperr.New(apperr.Unauthorized, "Hanya Head Division departemen ini (atau HRD) yang boleh mereview task")
		}
		if occ.Status != domain.StatusDone && !(in.Action == domain.ActReject && occ.Status == domain.StatusApproved) {
			return nil, apperr.New(apperr.Validation, "Hanya task berstatus selesai yang bisa direview")
		}
		notes := trimToNil(in.Notes)
		occ.ReviewNotes, occ.ReviewedBy, occ.ReviewedByName, occ.ReviewedAt = notes, me, &myName, &now
		if in.Action == domain.ActApprove {
			occ.Status = domain.StatusApproved
			return &ActionResult{Message: "Task disetujui"}, save(domain.UpdateReview, notes, nil, map[string]any{"status": occ.Status}, record)
		}
		occ.Status = domain.StatusRejected
		if err := save(domain.UpdateReview, notes, nil, map[string]any{"status": occ.Status}, drop); err != nil {
			return nil, err
		}
		t.improvements.fromTaskReject(ctx, *occ, notes, a.UserID)
		if occ.AssigneeID != nil {
			if ppl, err := t.people(ctx); err == nil {
				t.notify(ctx, Notice{UserIDs: ppl.userIDs(*occ.AssigneeID), Type: NotifTaskReview, Href: hrefTasks,
					Title: "Task ditolak: " + occ.Title, Body: deref(notes)})
			}
		}
		return &ActionResult{Message: "Task ditolak, penanggung jawab bisa memperbaiki"}, nil
	}
	return nil, apperr.New(apperr.Validation, "Aksi tidak dikenal")
}

// notifyReviewers tells the department head an occurrence waits for review.
func (t *Tasks) notifyReviewers(ctx context.Context, occ domain.Occurrence, doneBy string) {
	ppl, err := t.people(ctx)
	if err != nil {
		return
	}
	head := ppl.depts[occ.DepartmentID].HeadEmployeeID
	if head == nil {
		return
	}
	t.notify(ctx, Notice{UserIDs: ppl.userIDs(*head), Type: NotifTaskReview, Href: hrefTasks,
		Title: "Task menunggu review: " + occ.Title, Body: doneBy + " menandai task selesai."})
}

func (t *Tasks) checkSubtask(ctx context.Context, a *Actor, occ *domain.Occurrence, in OccurrenceAction, myName string, me *string) (*ActionResult, error) {
	ok, err := t.Store.SubtaskExists(ctx, occ.TaskID, in.SubtaskID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperr.New(apperr.Validation, "Sub-task bukan milik task ini")
	}
	now := t.now()
	var progress float64
	var complete bool
	err = t.Store.WithTx(ctx, func(tx domain.Store) error {
		var at *time.Time
		var by *string
		if in.Checked {
			at, by = &now, &myName
		}
		if err := tx.UpsertOccurrenceItem(ctx, occ.ID, in.SubtaskID, in.Checked, at, by); err != nil {
			return err
		}
		total, done, err := tx.SubtaskProgress(ctx, occ.ID, occ.TaskID)
		if err != nil {
			return err
		}
		progress = done
		complete = total > 0 && done >= total-0.01
		// All ticked: done (final for a personal task); some: in progress.
		status := domain.StatusPending
		switch {
		case complete && !occ.ReviewRequired:
			status = domain.StatusApproved
		case complete:
			status = domain.StatusDone
		case done > 0:
			status = domain.StatusInProgress
		}
		occ.Status = status
		if done > 0 && occ.StartedAt == nil {
			occ.StartedAt = &now
		}
		occ.DoneAt, occ.DoneBy, occ.DoneByName = nil, nil, nil
		if complete {
			occ.DoneAt, occ.DoneBy, occ.DoneByName = &now, me, &myName
		}
		occ.ReviewedBy, occ.ReviewedByName, occ.ReviewedAt = nil, nil, nil
		if status == domain.StatusApproved {
			occ.ReviewedBy, occ.ReviewedByName, occ.ReviewedAt = me, &myName, &now
		}
		if err := tx.SaveOccurrence(ctx, *occ); err != nil {
			return err
		}
		if status == domain.StatusApproved {
			err = tx.RecordContribution(ctx, occ.ID, occ.TaskID, domain.PeriodOf(occ.Date))
		} else {
			err = tx.DropContribution(ctx, occ.ID)
		}
		if err != nil {
			return err
		}
		pct := int(domain.Round(progress))
		return tx.InsertTaskUpdate(ctx, domain.TaskUpdate{ID: newID(), OccurrenceID: occ.ID, Kind: domain.UpdateProgress, ProgressPercent: &pct,
			AuthorEmployeeID: me, AuthorUserID: a.by(), AuthorName: myName, Detail: map[string]any{"subtask_id": in.SubtaskID, "checked": in.Checked},
			CreatedAt: now})
	})
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("Progres %v%%", domain.Round(progress))
	if complete && occ.ReviewRequired {
		msg = "Semua sub-task selesai (100%), menunggu review Head Division"
		t.notifyReviewers(ctx, *occ, myName)
	} else if complete {
		msg = "Semua sub-task selesai (100%)"
	}
	return &ActionResult{Message: msg, Progress: &progress, Complete: complete}, nil
}

// ─── views ───────────────────────────────────────────────────

// ItemFilter is GET /tasks/items: scope "mine" or "team", view
// today|week|month|all, month overrides the view.
type ItemFilter struct {
	Scope, View, Month, Q                         string
	DepartmentID, AssigneeID, Status, IndicatorID string
}

// ListItems lists occurrences; open rows past due always come along,
// first. Occurrences of the departments in reach are generated first.
func (t *Tasks) ListItems(ctx context.Context, f ItemFilter) ([]TaskView, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return nil, err
	}
	ppl, err := t.people(ctx)
	if err != nil {
		return nil, err
	}
	today := t.today()
	q := domain.ItemQuery{AssigneeID: f.AssigneeID, Status: f.Status, IndicatorID: f.IndicatorID, Q: strings.TrimSpace(f.Q)}
	var depts []string
	if f.Scope == "team" {
		switch {
		case a.HRRead && f.DepartmentID != "":
			depts = []string{f.DepartmentID}
		case a.HRRead:
			for id := range ppl.depts {
				depts = append(depts, id)
			}
		case f.DepartmentID != "":
			if !a.headsDept(f.DepartmentID) {
				return nil, apperr.New(apperr.Unauthorized, "Anda bukan kepala departemen ini")
			}
			depts = []string{f.DepartmentID}
		default:
			for id := range ppl.depts {
				if a.headsDept(id) {
					depts = append(depts, id)
				}
			}
			if len(depts) == 0 {
				return nil, apperr.New(apperr.Unauthorized, "Papan tim hanya untuk kepala departemen")
			}
		}
		q.DepartmentIDs = depts
	} else {
		if own := a.DepartmentID(); own != "" {
			depts = []string{own}
		}
		q.MineOnly, q.MineEmployeeID, q.MineDepartmentID, q.MineUserID = true, a.EmployeeID(), a.DepartmentID(), a.UserID
	}
	span, ranged := domain.ViewRange(f.View, t.now(), domain.Jakarta), true
	switch {
	case f.Month != "":
		p, ok := domain.ParsePeriod(f.Month)
		if !ok {
			return nil, apperr.New(apperr.Validation, msgBadMonth)
		}
		span = p.Span()
	case f.View == "all":
		ranged = false
	}
	if ranged {
		// From the start of the span's month so earlier open rows exist.
		from := span.Start[:7] + "-01"
		if err := t.ensureOccurrences(ctx, depts, from, span.End); err != nil {
			return nil, err
		}
		q.SpanStart, q.SpanEnd, q.OverdueBefore = span.Start, span.End, today
	}
	items, err := t.Store.QueryItems(ctx, q)
	if err != nil {
		return nil, err
	}
	out := make([]TaskView, len(items))
	for i, it := range items {
		out[i] = t.finish(a, ppl, it)
	}
	return out, nil
}

// TaskDetail is an occurrence with everything attached to it.
type TaskDetail struct {
	TaskView
	Subtasks     []domain.SubtaskState
	Updates      []domain.TaskUpdate
	Attachments  []domain.TaskAttachment
	Contribution *domain.TaskContribution
}

func (t *Tasks) loadItem(ctx context.Context, a *Actor, id string) (*TaskView, error) {
	items, err := t.Store.QueryItems(ctx, domain.ItemQuery{OccurrenceID: id})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, apperr.New(apperr.NotFound, msgOccurrenceGone)
	}
	ppl, err := t.people(ctx)
	if err != nil {
		return nil, err
	}
	v := t.finish(a, ppl, items[0])
	if !v.Mine && !v.CanWork && !v.CanManage && !a.managesDept(v.DepartmentID) {
		return nil, apperr.New(apperr.NotFound, msgOccurrenceGone)
	}
	return &v, nil
}

func (t *Tasks) GetOccurrence(ctx context.Context, id string) (*TaskDetail, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := t.loadItem(ctx, a, id)
	if err != nil {
		return nil, err
	}
	d := &TaskDetail{TaskView: *v}
	if d.Subtasks, err = t.Store.ListSubtaskStates(ctx, id, v.TaskID); err != nil {
		return nil, err
	}
	if d.Updates, err = t.Store.ListTaskUpdates(ctx, id, 200); err != nil {
		return nil, err
	}
	if d.Attachments, err = t.Store.ListAttachments(ctx, id); err != nil {
		return nil, err
	}
	if d.Contribution, err = t.Store.GetContribution(ctx, id); err != nil {
		return nil, err
	}
	return d, nil
}

// ─── attachments ─────────────────────────────────────────────

var controlChars = regexp.MustCompile(`[\x00-\x1f\x7f]`)

func cleanFileName(name string) string {
	s := controlChars.ReplaceAllString(name, "")
	s = strings.TrimSpace(strings.NewReplacer(`\`, "-", "/", "-").Replace(s))
	if strings.Trim(s, ".") == "" {
		s = "lampiran"
	}
	if r := []rune(s); len(r) > 255 {
		s = string(r[:255])
	}
	return s
}

// resolveMime trusts the bytes first, then the client's type.
func resolveMime(data []byte, declared string) string {
	if m := http.DetectContentType(data); m != "application/octet-stream" && !strings.HasPrefix(m, "text/plain") {
		return strings.SplitN(m, ";", 2)[0]
	}
	if c := strings.ToLower(strings.TrimSpace(declared)); strings.Count(c, "/") == 1 && len(c) <= 150 {
		return c
	}
	return "application/octet-stream"
}

var extRE = regexp.MustCompile(`^\.[a-z0-9]{1,7}$`)

func extensionFor(mimeType, name string) string {
	if ext := strings.ToLower(filepath.Ext(name)); extRE.MatchString(ext) {
		return ext[1:]
	}
	if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
		return strings.TrimPrefix(exts[0], ".")
	}
	return "bin"
}

func (t *Tasks) AddAttachment(ctx context.Context, occurrenceID, name, declaredMime string, data []byte) (*domain.TaskAttachment, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := t.loadItem(ctx, a, occurrenceID)
	if err != nil {
		return nil, err
	}
	if !v.CanWork && !v.CanManage {
		return nil, apperr.New(apperr.Unauthorized, "Hanya penanggung jawab atau kepala departemen yang bisa melampirkan file")
	}
	if len(data) == 0 {
		return nil, apperr.New(apperr.Validation, "File kosong")
	}
	if len(data) > MaxAttachmentBytes {
		return nil, apperr.New(apperr.Validation, "Ukuran file maksimal 10 MB")
	}
	mt := resolveMime(data, declaredMime)
	token := make([]byte, 12)
	_, _ = rand.Read(token)
	now := t.now().In(domain.Jakarta)
	key := fmt.Sprintf("tasks/%d/%02d/%s.%s", now.Year(), int(now.Month()), hex.EncodeToString(token), extensionFor(mt, name))
	if err := t.Files.Put(ctx, key, data); err != nil {
		return nil, err
	}
	uploader := a.Name
	att := domain.TaskAttachment{ID: newID(), OccurrenceID: occurrenceID, Name: cleanFileName(name), Mime: mt, SizeBytes: int64(len(data)),
		StoragePath: key, UploadedByUserID: a.by(), UploadedByName: uploader, CreatedAt: t.now()}
	var me *string
	if a.EmployeeID() != "" {
		me = strPtr(a.EmployeeID())
	}
	err = t.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertAttachment(ctx, att); err != nil {
			return err
		}
		return tx.InsertTaskUpdate(ctx, domain.TaskUpdate{ID: newID(), OccurrenceID: occurrenceID, Kind: domain.UpdateAttachment,
			AuthorEmployeeID: me, AuthorUserID: a.by(), AuthorName: uploader, Detail: map[string]any{"name": att.Name}, CreatedAt: att.CreatedAt})
	})
	if err != nil {
		_ = t.Files.Delete(ctx, key)
		return nil, err
	}
	return &att, nil
}

func (t *Tasks) attachment(ctx context.Context, a *Actor, occurrenceID, attachmentID string) (*domain.TaskAttachment, *TaskView, error) {
	v, err := t.loadItem(ctx, a, occurrenceID)
	if err != nil {
		return nil, nil, err
	}
	att, err := t.Store.GetAttachment(ctx, attachmentID)
	if err != nil {
		return nil, nil, err
	}
	if att == nil || att.OccurrenceID != occurrenceID {
		return nil, nil, apperr.New(apperr.NotFound, "Lampiran tidak ditemukan")
	}
	return att, v, nil
}

// GetAttachment returns the file's metadata and bytes.
func (t *Tasks) GetAttachment(ctx context.Context, occurrenceID, attachmentID string) (*domain.TaskAttachment, []byte, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	att, _, err := t.attachment(ctx, a, occurrenceID, attachmentID)
	if err != nil {
		return nil, nil, err
	}
	data, err := t.Files.Get(ctx, att.StoragePath)
	if err != nil {
		return nil, nil, apperr.Wrap(apperr.NotFound, "File lampiran tidak ditemukan", err)
	}
	return att, data, nil
}

func (t *Tasks) DeleteAttachment(ctx context.Context, occurrenceID, attachmentID string) error {
	a, err := t.actor(ctx)
	if err != nil {
		return err
	}
	att, v, err := t.attachment(ctx, a, occurrenceID, attachmentID)
	if err != nil {
		return err
	}
	if !v.CanManage && (att.UploadedByUserID == nil || *att.UploadedByUserID != a.UserID) {
		return apperr.New(apperr.Unauthorized, "Hanya pengunggah atau kepala departemen yang bisa menghapus lampiran")
	}
	if err := t.Store.DeleteAttachment(ctx, attachmentID); err != nil {
		return err
	}
	if err := t.Files.Delete(ctx, att.StoragePath); err != nil {
		t.log().Warn("attachment file not deleted", "key", att.StoragePath, "err", err)
	}
	return nil
}

// ─── meta, KPI view, contributions, monitor ──────────────────

// TaskIndicator is an indicator a task can link to, with the departments
// that weight it.
type TaskIndicator struct {
	domain.Indicator
	DepartmentIDs []string
}

// TaskMeta is what the task pages need up front.
type TaskMeta struct {
	Actor       *Actor
	TeamScope   string
	Departments []domain.Department
	Members     []domain.Employee
	Indicators  []TaskIndicator
	Today       string
}

func (t *Tasks) Meta(ctx context.Context) (*TaskMeta, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return nil, err
	}
	ppl, err := t.people(ctx)
	if err != nil {
		return nil, err
	}
	m := &TaskMeta{Actor: a, TeamScope: "none", Today: t.today(), Departments: []domain.Department{}, Members: []domain.Employee{}, Indicators: []TaskIndicator{}}
	switch {
	case a.HRRead:
		m.TeamScope = "all"
	case len(a.HeadOf) > 0 || (len(a.Reports) > 0 && a.DepartmentID() != ""):
		m.TeamScope = "heads"
	}
	depts, err := t.Directory.Departments(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range depts {
		if !a.managesDept(d.ID) && a.DepartmentID() != d.ID {
			continue
		}
		m.Departments = append(m.Departments, d)
		m.Members = append(m.Members, ppl.members(d.ID)...)
	}
	inds, err := t.Store.ListIndicators(ctx, false)
	if err != nil {
		return nil, err
	}
	byInd, err := t.Store.IndicatorDepartments(ctx)
	if err != nil {
		return nil, err
	}
	for _, ind := range inds {
		m.Indicators = append(m.Indicators, TaskIndicator{Indicator: ind, DepartmentIDs: byInd[ind.ID]})
	}
	return m, nil
}

// IndicatorTasks is one indicator of a department with its occurrences of
// the month and their completion.
type IndicatorTasks struct {
	domain.KPIIndicatorWeight
	Total             int
	Approved          int
	WaitingReview     int
	Open              int
	Overdue           int
	Rate              *float64
	ExpectedTotal     float64
	ContributionTotal float64
	Tasks             []TaskView
}

// KPIView: for each indicator the department weights (or its tasks link
// to), the month's occurrences and how many were approved. Rate is
// approved over the rows due up to today.
func (t *Tasks) KPIView(ctx context.Context, departmentID, month string) ([]IndicatorTasks, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return nil, err
	}
	if month == "" {
		month = t.today()[:7]
	}
	p, ok := domain.ParsePeriod(month)
	if !ok {
		return nil, apperr.New(apperr.Validation, msgBadMonth)
	}
	if departmentID == "" {
		departmentID = a.DepartmentID()
	}
	if departmentID == "" {
		return nil, apperr.New(apperr.Validation, "Pilih departemen")
	}
	if !a.managesDept(departmentID) && a.DepartmentID() != departmentID {
		return nil, apperr.New(apperr.Unauthorized, "Anda tidak punya akses ke KPI departemen ini")
	}
	span := p.Span()
	if err := t.ensureOccurrences(ctx, []string{departmentID}, span.Start, span.End); err != nil {
		return nil, err
	}
	items, err := t.Store.QueryItems(ctx, domain.ItemQuery{DepartmentIDs: []string{departmentID}, KPIOnly: true, SpanStart: span.Start, SpanEnd: span.End})
	if err != nil {
		return nil, err
	}
	inds, err := t.Store.DepartmentKPIIndicators(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	totals, err := t.Store.ContributionTotals(ctx, departmentID, span)
	if err != nil {
		return nil, err
	}
	ppl, err := t.people(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]IndicatorTasks, len(inds))
	index := map[string]int{}
	for i, ind := range inds {
		out[i] = IndicatorTasks{KPIIndicatorWeight: ind, Tasks: []TaskView{}, ContributionTotal: totals[ind.ID]}
		index[ind.ID] = i
	}
	today := t.today()
	due := map[string]int{}
	for _, it := range items {
		i, ok := index[deref(it.KPIIndicatorID)]
		if !ok {
			continue
		}
		v := t.finish(a, ppl, it)
		row := &out[i]
		row.Tasks = append(row.Tasks, v)
		row.Total++
		if it.Date <= today {
			due[row.ID]++
		}
		switch it.Status {
		case domain.StatusApproved:
			row.Approved++
		case domain.StatusDone:
			row.WaitingReview++
		default:
			row.Open++
			if v.Overdue {
				row.Overdue++
			}
		}
		if it.KPIContribution != nil {
			row.ExpectedTotal += *it.KPIContribution
		}
	}
	for i := range out {
		if n := due[out[i].ID]; n > 0 {
			rate := float64(out[i].Approved) / float64(n)
			out[i].Rate = &rate
		}
	}
	return out, nil
}

// Contributions: KPI readers see anyone, leads their team, others themselves.
func (t *Tasks) Contributions(ctx context.Context, employeeID, month string) ([]domain.ContributionSummary, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return nil, err
	}
	p, ok := domain.ParsePeriod(month)
	if !ok {
		return nil, apperr.New(apperr.Validation, msgBadMonth)
	}
	if employeeID == "" || employeeID == "me" {
		employeeID = a.EmployeeID()
	}
	ppl, err := t.people(ctx)
	if err != nil {
		return nil, err
	}
	e, ok := ppl.employee(employeeID)
	if !ok {
		return nil, apperr.New(apperr.NotFound, msgEmployeeGone)
	}
	if !a.SeesEmployee(e) {
		return nil, apperr.New(apperr.Unauthorized, "Anda tidak punya akses ke kontribusi karyawan ini")
	}
	return t.Store.EmployeeContributions(ctx, employeeID, p.Span())
}

// MonitorData is the monitor: per department and assignee, overdue rows
// with their last note and rows waiting for review.
type MonitorData struct {
	Span          domain.DaySpan
	Today         string
	Summary       domain.MonitorSummary
	Departments   []domain.DepartmentSummary
	Overdue       []domain.MonitorRow
	WaitingReview []domain.MonitorRow
	Scope         string
}

func (t *Tasks) Monitor(ctx context.Context, departmentID, month string) (*MonitorData, error) {
	a, err := t.actor(ctx)
	if err != nil {
		return nil, err
	}
	p, ok := domain.ParsePeriod(month)
	if !ok {
		return nil, apperr.New(apperr.Validation, "Parameter month wajib (YYYY-MM)")
	}
	ppl, err := t.people(ctx)
	if err != nil {
		return nil, err
	}
	scope := "own"
	depts := []string{}
	switch {
	case a.HRRead:
		scope = "all"
		if departmentID != "" {
			depts = []string{departmentID}
		} else {
			for id := range ppl.depts {
				depts = append(depts, id)
			}
		}
	case a.DepartmentID() != "":
		depts = []string{a.DepartmentID()}
	}
	for id := range a.HeadOf {
		if !a.HRRead && id != a.DepartmentID() {
			depts = append(depts, id)
		}
	}
	span := p.Span()
	if err := t.ensureOccurrences(ctx, depts, span.Start, span.End); err != nil {
		return nil, err
	}
	var items []domain.TaskItem
	if len(depts) > 0 {
		if items, err = t.Store.QueryItems(ctx, domain.ItemQuery{DepartmentIDs: depts, SpanStart: span.Start, SpanEnd: span.End}); err != nil {
			return nil, err
		}
	}
	rows := make([]domain.MonitorRow, len(items))
	for i, it := range items {
		v := t.finish(a, ppl, it)
		name := v.AssigneeName
		rows[i] = domain.MonitorRow{OccurrenceID: it.OccurrenceID, TaskID: it.TaskID, Title: it.Title, DepartmentID: it.DepartmentID,
			DepartmentName: v.DepartmentName, AssigneeID: it.AssigneeID, Date: it.Date, Status: it.Status, Progress: v.Progress, LastNote: it.LastNote}
		if it.AssigneeID != nil {
			rows[i].AssigneeName = &name
		}
	}
	today := t.today()
	return &MonitorData{Span: span, Today: today, Summary: domain.Summarize(rows, today), Departments: domain.GroupByDepartment(rows, today),
		Overdue: domain.OverdueList(rows, today, 50), WaitingReview: domain.WaitingReview(rows, 50), Scope: scope}, nil
}
