package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Improvements are the "things to improve" items: filed by people, by the
// review, feedback, task and logbook flows, by retro actions and by the AI
// pass over monthly recaps (status suggested until accepted).
type Improvements struct{ Deps }

func NewImprovements(d Deps) *Improvements { return &Improvements{d} }

const (
	msgImprovementGone = "Item perbaikan tidak ditemukan"
	msgRetroGone       = "Retrospektif tidak ditemukan"
	maxRetroLines      = 30
)

// ─── automatic filing ────────────────────────────────────────

// file inserts an automated item, skipping a source_ref already filed;
// failures are logged so the triggering flow never fails because of it.
func (m *Improvements) file(ctx context.Context, imp domain.Improvement) bool {
	if imp.ID == "" {
		imp.ID = newID()
	}
	if imp.Status == "" {
		imp.Status = domain.ImpOpen
	}
	ok, err := m.Store.InsertImprovement(ctx, imp)
	if err != nil {
		m.log().Warn("improvement not filed", "source", imp.Source, "ref", deref(imp.SourceRef), "err", err)
		return false
	}
	return ok
}

func employeeItem(e domain.Employee, source, ref, title, description, by string) domain.Improvement {
	return domain.Improvement{Scope: domain.ScopeEmployee, EmployeeID: strPtr(e.ID), DepartmentID: e.DepartmentID, Source: source,
		SourceRef: strPtr(ref), Title: title, Description: description, OwnerEmployeeID: strPtr(e.ID), CreatedBy: strPtr(by)}
}

// fromTaskReject files the review note of a rejected occurrence.
func (m *Improvements) fromTaskReject(ctx context.Context, occ domain.Occurrence, notes *string, by string) {
	if notes == nil || occ.AssigneeID == nil {
		return
	}
	e := domain.Employee{ID: *occ.AssigneeID, DepartmentID: strPtr(occ.DepartmentID)}
	m.file(ctx, employeeItem(e, domain.SourceTaskReject, occ.ID+":"+occ.Date,
		domain.ImprovementTitle("Perbaiki: ", occ.Title), *notes, by))
}

// fromLogbookReject files the review note of a rejected logbook entry.
func (m *Improvements) fromLogbookReject(ctx context.Context, entry domain.LogbookEntry, notes *string, by string) {
	if notes == nil {
		return
	}
	e := domain.Employee{ID: entry.EmployeeID, DepartmentID: strPtr(entry.DepartmentID)}
	m.file(ctx, employeeItem(e, domain.SourceLogbookReject, entry.ID,
		domain.ImprovementTitle("Logbook: ", *notes), "Catatan review logbook "+entry.Title+":\n"+*notes, by))
}

// fromReview files the reviewer and manager notes of a finalized review.
func (m *Improvements) fromReview(ctx context.Context, rv domain.Review, by string) {
	e := domain.Employee{ID: rv.EmployeeID, DepartmentID: rv.DepartmentID}
	for _, n := range []struct {
		key, label string
		text       *string
	}{{"reviewer", "reviewer", rv.ReviewerNotes}, {"manager", "manajer", rv.ManagerNotes}} {
		if n.text == nil {
			continue
		}
		m.file(ctx, employeeItem(e, domain.SourceReview, rv.ID+":"+n.key, domain.ImprovementTitle("", *n.text),
			fmt.Sprintf("Catatan %s review %s:\n%s", n.label, rv.PeriodLabel, *n.text), by))
	}
}

// fromFeedback files each 360 weakness of a summary.
func (m *Improvements) fromFeedback(ctx context.Context, s domain.FeedbackSummary, e domain.Employee, cycleName, by string) int {
	n := 0
	for i, w := range s.Weaknesses {
		if m.file(ctx, employeeItem(e, domain.SourceFeedback, fmt.Sprintf("%s:%d", s.ID, i), domain.ImprovementTitle("Tingkatkan ", w),
			"Hasil feedback 360 "+cycleName+": "+w, by)) {
			n++
		}
	}
	return n
}

// fromSuggestions stores AI proposals for a monthly recap as suggested.
func (m *Improvements) fromSuggestions(ctx context.Context, recap domain.MonthlyRecap, suggestions []domain.ImprovementSuggestion) int {
	n := 0
	for i, s := range suggestions {
		if i == domain.MaxSuggestions {
			break
		}
		title := domain.ImprovementTitle("", s.Title)
		if len([]rune(title)) < domain.ImprovementTitleMin {
			continue
		}
		imp := domain.Improvement{Source: domain.SourceAI, SourceRef: strPtr(fmt.Sprintf("%s:%d", recap.ID, i)), Title: title,
			Description: strings.TrimSpace(s.Description), Status: domain.ImpSuggested, DepartmentID: recap.DepartmentID}
		if recap.Scope == domain.ScopeEmployee {
			imp.Scope, imp.EmployeeID, imp.OwnerEmployeeID = domain.ScopeEmployee, strPtr(recap.SubjectID), strPtr(recap.SubjectID)
		} else {
			imp.Scope, imp.DepartmentID = domain.ScopeDepartment, strPtr(recap.SubjectID)
		}
		if m.file(ctx, imp) {
			n++
		}
	}
	return n
}

// ─── access ──────────────────────────────────────────────────

// canSee: HR readers, the employee, its owner, its leads; department items
// for members and the head.
func (m *Improvements) canSee(a *Actor, ppl *people, imp domain.Improvement) bool {
	if a.HRRead || (imp.OwnerEmployeeID != nil && a.IsEmployee(*imp.OwnerEmployeeID)) {
		return true
	}
	if imp.Scope == domain.ScopeEmployee && imp.EmployeeID != nil {
		e, _ := ppl.employee(*imp.EmployeeID)
		e.ID = *imp.EmployeeID
		return a.IsEmployee(e.ID) || a.Leads(e)
	}
	return imp.DepartmentID != nil && (a.HeadOf[*imp.DepartmentID] || a.DepartmentID() == *imp.DepartmentID)
}

// canManage: HR writers, the owner, the employee, its leads; for
// department items the head.
func (m *Improvements) canManage(a *Actor, ppl *people, imp domain.Improvement) bool {
	if a.HRWrite || (imp.OwnerEmployeeID != nil && a.IsEmployee(*imp.OwnerEmployeeID)) {
		return true
	}
	if imp.Scope == domain.ScopeEmployee && imp.EmployeeID != nil {
		e, _ := ppl.employee(*imp.EmployeeID)
		e.ID = *imp.EmployeeID
		return a.IsEmployee(e.ID) || a.Leads(e)
	}
	return imp.DepartmentID != nil && a.HeadOf[*imp.DepartmentID]
}

// ImprovementQuery filters the list; Mine keeps items of or owned by the
// caller.
type ImprovementQuery struct {
	Scope        string
	EmployeeID   string
	DepartmentID string
	Status       string
	Source       string
	Mine         bool
}

// ImprovementView is an item with names.
type ImprovementView struct {
	domain.Improvement
	EmployeeName   string
	DepartmentName string
	OwnerName      string
}

func (m *Improvements) view(ppl *people, imp domain.Improvement) ImprovementView {
	return ImprovementView{Improvement: imp, EmployeeName: ppl.name(imp.EmployeeID), DepartmentName: ppl.deptName(imp.DepartmentID),
		OwnerName: ppl.name(imp.OwnerEmployeeID)}
}

func (m *Improvements) List(ctx context.Context, q ImprovementQuery) ([]ImprovementView, error) {
	a, err := m.actor(ctx)
	if err != nil {
		return nil, err
	}
	ppl, err := m.people(ctx)
	if err != nil {
		return nil, err
	}
	f := domain.ImprovementFilter{Scope: q.Scope, Source: q.Source}
	if q.Status != "" {
		f.Statuses = []string{q.Status}
	}
	if q.EmployeeID != "" {
		f.EmployeeIDs = []string{q.EmployeeID}
	}
	if q.DepartmentID != "" {
		f.DepartmentIDs = []string{q.DepartmentID}
	}
	if q.Mine {
		if a.EmployeeID() == "" {
			return []ImprovementView{}, nil
		}
		f.PersonID = a.EmployeeID()
	}
	rows, err := m.Store.ListImprovements(ctx, f)
	if err != nil {
		return nil, err
	}
	out := []ImprovementView{}
	for _, imp := range rows {
		if m.canSee(a, ppl, imp) {
			out = append(out, m.view(ppl, imp))
		}
	}
	return out, nil
}

func (m *Improvements) load(ctx context.Context, a *Actor, id string) (*domain.Improvement, *people, error) {
	imp, err := m.Store.GetImprovement(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	ppl, err := m.people(ctx)
	if err != nil {
		return nil, nil, err
	}
	if imp == nil || !m.canSee(a, ppl, *imp) {
		return nil, nil, apperr.New(apperr.NotFound, msgImprovementGone)
	}
	return imp, ppl, nil
}

func (m *Improvements) Get(ctx context.Context, id string) (*ImprovementView, error) {
	a, err := m.actor(ctx)
	if err != nil {
		return nil, err
	}
	imp, ppl, err := m.load(ctx, a, id)
	if err != nil {
		return nil, err
	}
	v := m.view(ppl, *imp)
	return &v, nil
}

// ImprovementInput creates or edits an item; nil fields are kept on edit.
type ImprovementInput struct {
	Scope           string
	EmployeeID      *string
	DepartmentID    *string
	Title           *string
	Description     *string
	OwnerEmployeeID *string
	DueDate         *string
	Status          *string
	LinkedTaskID    *string
}

func validateTitle(t string) error {
	n := len([]rune(t))
	if n < domain.ImprovementTitleMin || n > domain.ImprovementTitleMax {
		return apperr.Field("title", "Judul perbaikan 3 sampai 200 karakter")
	}
	return nil
}

func validDue(s *string) error {
	if s != nil && *s != "" && !validDay(*s) {
		return apperr.Field("dueDate", "Tanggal tenggat tidak valid")
	}
	return nil
}

func (m *Improvements) Create(ctx context.Context, in ImprovementInput) (*ImprovementView, error) {
	a, err := m.actor(ctx)
	if err != nil {
		return nil, err
	}
	ppl, err := m.people(ctx)
	if err != nil {
		return nil, err
	}
	imp := domain.Improvement{ID: newID(), Scope: in.Scope, Source: domain.SourceManualItem, Title: strings.TrimSpace(deref(in.Title)),
		Description: strings.TrimSpace(deref(in.Description)), OwnerEmployeeID: trimToNil(in.OwnerEmployeeID), DueDate: trimToNil(in.DueDate),
		Status: domain.ImpOpen, LinkedTaskID: trimToNil(in.LinkedTaskID), CreatedBy: a.by()}
	if err := validateTitle(imp.Title); err != nil {
		return nil, err
	}
	if err := validDue(imp.DueDate); err != nil {
		return nil, err
	}
	switch in.Scope {
	case domain.ScopeEmployee:
		id := deref(trimToNil(in.EmployeeID))
		if id == "" {
			id = a.EmployeeID()
		}
		e, ok := ppl.employee(id)
		if !ok {
			return nil, apperr.New(apperr.NotFound, msgEmployeeGone)
		}
		if !a.HRWrite && !a.IsEmployee(e.ID) && !a.Leads(e) {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
		imp.EmployeeID, imp.DepartmentID = &e.ID, e.DepartmentID
		if imp.OwnerEmployeeID == nil {
			imp.OwnerEmployeeID = &e.ID
		}
	case domain.ScopeDepartment:
		dept := deref(trimToNil(in.DepartmentID))
		if dept == "" {
			dept = a.DepartmentID()
		}
		if _, ok := ppl.depts[dept]; !ok {
			return nil, apperr.New(apperr.NotFound, msgDeptNotFound)
		}
		if !a.HRWrite && !a.HeadOf[dept] && a.DepartmentID() != dept {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
		imp.DepartmentID = &dept
	default:
		return nil, apperr.Field("scope", "Cakupan harus karyawan atau departemen")
	}
	if imp.OwnerEmployeeID != nil {
		if _, ok := ppl.employee(*imp.OwnerEmployeeID); !ok {
			return nil, apperr.Field("ownerEmployeeId", "Penanggung jawab tidak ditemukan")
		}
	}
	if _, err := m.Store.InsertImprovement(ctx, imp); err != nil {
		return nil, err
	}
	v := m.view(ppl, imp)
	return &v, nil
}

func (m *Improvements) Update(ctx context.Context, id string, in ImprovementInput) (*ImprovementView, error) {
	a, err := m.actor(ctx)
	if err != nil {
		return nil, err
	}
	imp, ppl, err := m.load(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if !m.canManage(a, ppl, *imp) {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	if in.Title != nil {
		imp.Title = strings.TrimSpace(*in.Title)
		if err := validateTitle(imp.Title); err != nil {
			return nil, err
		}
	}
	if in.Description != nil {
		imp.Description = strings.TrimSpace(*in.Description)
	}
	if in.OwnerEmployeeID != nil {
		imp.OwnerEmployeeID = trimToNil(in.OwnerEmployeeID)
		if imp.OwnerEmployeeID != nil {
			if _, ok := ppl.employee(*imp.OwnerEmployeeID); !ok {
				return nil, apperr.Field("ownerEmployeeId", "Penanggung jawab tidak ditemukan")
			}
		}
	}
	if in.DueDate != nil {
		if err := validDue(in.DueDate); err != nil {
			return nil, err
		}
		imp.DueDate = trimToNil(in.DueDate)
	}
	if in.LinkedTaskID != nil {
		imp.LinkedTaskID = trimToNil(in.LinkedTaskID)
	}
	if in.Status != nil && *in.Status != imp.Status {
		if err := m.transition(imp, *in.Status); err != nil {
			return nil, err
		}
	}
	imp.UpdatedBy = a.by()
	if err := m.Store.UpdateImprovement(ctx, *imp); err != nil {
		return nil, err
	}
	v := m.view(ppl, *imp)
	return &v, nil
}

func (m *Improvements) transition(imp *domain.Improvement, to string) error {
	if !domain.CanTransitionImprovement(imp.Status, to) {
		return apperr.New(apperr.Validation, "Perubahan status tidak diizinkan")
	}
	imp.Status = to
	imp.CompletedAt = nil
	if to == domain.ImpDone {
		imp.CompletedAt = timePtr(m.now())
	}
	return nil
}

// Decide accepts (open) or declines a suggested item.
func (m *Improvements) Decide(ctx context.Context, id string, accept bool, owner, dueDate *string) (*ImprovementView, error) {
	a, err := m.actor(ctx)
	if err != nil {
		return nil, err
	}
	imp, ppl, err := m.load(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if !m.canManage(a, ppl, *imp) {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	if imp.Status != domain.ImpSuggested {
		return nil, apperr.New(apperr.Conflict, "Hanya saran yang bisa diterima atau ditolak")
	}
	to := domain.ImpDeclined
	if accept {
		to = domain.ImpOpen
		if o := trimToNil(owner); o != nil {
			if _, ok := ppl.employee(*o); !ok {
				return nil, apperr.Field("ownerEmployeeId", "Penanggung jawab tidak ditemukan")
			}
			imp.OwnerEmployeeID = o
		}
		if err := validDue(dueDate); err != nil {
			return nil, err
		}
		if d := trimToNil(dueDate); d != nil {
			imp.DueDate = d
		}
	}
	if err := m.transition(imp, to); err != nil {
		return nil, err
	}
	imp.UpdatedBy = a.by()
	if err := m.Store.UpdateImprovement(ctx, *imp); err != nil {
		return nil, err
	}
	v := m.view(ppl, *imp)
	return &v, nil
}

func (m *Improvements) Delete(ctx context.Context, id string) error {
	a, err := m.actor(ctx)
	if err != nil {
		return err
	}
	imp, ppl, err := m.load(ctx, a, id)
	if err != nil {
		return err
	}
	if !m.canManage(a, ppl, *imp) {
		return apperr.New(apperr.Unauthorized, "")
	}
	return m.Store.SoftDeleteImprovement(ctx, id, a.UserID)
}

// ─── Retrospectives ──────────────────────────────────────────

// RetroView is a retro with its department's name.
type RetroView struct {
	domain.RetroSession
	DepartmentName string
}

func (m *Improvements) retroView(ctx context.Context, r *domain.RetroSession) (*RetroView, error) {
	ppl, err := m.people(ctx)
	if err != nil {
		return nil, err
	}
	return &RetroView{RetroSession: *r, DepartmentName: ppl.depts[r.DepartmentID].Name}, nil
}

func (m *Improvements) ListRetros(ctx context.Context, departmentID string) ([]RetroView, error) {
	a, err := m.actor(ctx)
	if err != nil {
		return nil, err
	}
	var depts []string
	switch {
	case departmentID != "":
		if !a.SeesDepartment(departmentID) && a.DepartmentID() != departmentID {
			return nil, apperr.New(apperr.Unauthorized, "")
		}
		depts = []string{departmentID}
	case !a.HRRead:
		for d := range a.HeadOf {
			depts = append(depts, d)
		}
		if own := a.DepartmentID(); own != "" && !a.HeadOf[own] {
			depts = append(depts, own)
		}
		if len(depts) == 0 {
			return []RetroView{}, nil
		}
	}
	rows, err := m.Store.ListRetros(ctx, depts)
	if err != nil {
		return nil, err
	}
	ppl, err := m.people(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RetroView, len(rows))
	for i, r := range rows {
		out[i] = RetroView{RetroSession: r, DepartmentName: ppl.depts[r.DepartmentID].Name}
	}
	return out, nil
}

func (m *Improvements) GetRetro(ctx context.Context, id string) (*RetroView, error) {
	a, err := m.actor(ctx)
	if err != nil {
		return nil, err
	}
	r, err := m.Store.GetRetro(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil || (!a.SeesDepartment(r.DepartmentID) && a.DepartmentID() != r.DepartmentID) {
		return nil, apperr.New(apperr.NotFound, msgRetroGone)
	}
	return m.retroView(ctx, r)
}

// RetroInput saves a retro draft (by department and month).
type RetroInput struct {
	DepartmentID string
	Period       domain.Period
	WentWell     []string
	ToImprove    []string
	Actions      []domain.RetroAction
}

func (m *Improvements) SaveRetro(ctx context.Context, in RetroInput) (*RetroView, error) {
	a, err := m.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !a.HRWrite && !a.HeadOf[in.DepartmentID] {
		return nil, apperr.New(apperr.Unauthorized, "Hanya kepala departemen (atau HRD) yang boleh mengisi retrospektif")
	}
	if !in.Period.Valid() {
		return nil, apperr.New(apperr.Validation, msgPeriodInvalid)
	}
	ppl, err := m.people(ctx)
	if err != nil {
		return nil, err
	}
	if _, ok := ppl.depts[in.DepartmentID]; !ok {
		return nil, apperr.New(apperr.NotFound, msgDeptNotFound)
	}
	actions := []domain.RetroAction{}
	for _, act := range in.Actions {
		act.Description = strings.TrimSpace(act.Description)
		if act.Description == "" || len(actions) == maxRetroLines {
			continue
		}
		act.OwnerEmployeeID, act.DueDate, act.ImprovementID = trimToNil(act.OwnerEmployeeID), trimToNil(act.DueDate), nil
		if err := validDue(act.DueDate); err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	cur, err := m.Store.GetRetroFor(ctx, in.DepartmentID, in.Period)
	if err != nil {
		return nil, err
	}
	if cur != nil && cur.Status == domain.RetroClosed {
		return nil, apperr.New(apperr.Conflict, "Retrospektif sudah ditutup")
	}
	r := domain.RetroSession{ID: newID(), DepartmentID: in.DepartmentID, Period: in.Period, WentWell: domain.CleanList(in.WentWell, maxRetroLines),
		ToImprove: domain.CleanList(in.ToImprove, maxRetroLines), Actions: actions, Status: domain.RetroDraft, CreatedBy: a.by(), UpdatedBy: a.by()}
	if cur == nil {
		err = m.Store.InsertRetro(ctx, r)
	} else {
		r.ID, r.CreatedBy = cur.ID, cur.CreatedBy
		err = m.Store.UpdateRetro(ctx, r)
	}
	if err != nil {
		return nil, err
	}
	saved, err := m.Store.GetRetro(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	return m.retroView(ctx, saved)
}

// CloseRetro turns every action into an open department improvement item.
func (m *Improvements) CloseRetro(ctx context.Context, id string) (*RetroView, int, error) {
	a, err := m.actor(ctx)
	if err != nil {
		return nil, 0, err
	}
	r, err := m.Store.GetRetro(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	if r == nil {
		return nil, 0, apperr.New(apperr.NotFound, msgRetroGone)
	}
	if !a.HRWrite && !a.HeadOf[r.DepartmentID] {
		return nil, 0, apperr.New(apperr.Unauthorized, "")
	}
	if r.Status == domain.RetroClosed {
		return nil, 0, apperr.New(apperr.Conflict, "Retrospektif sudah ditutup")
	}
	filed := 0
	err = m.Store.WithTx(ctx, func(tx domain.Store) error {
		for i := range r.Actions {
			act := &r.Actions[i]
			imp := domain.Improvement{ID: newID(), Scope: domain.ScopeDepartment, DepartmentID: &r.DepartmentID, Source: domain.SourceRetro,
				SourceRef: strPtr(fmt.Sprintf("%s:%d", r.ID, i)), Title: domain.ImprovementTitle("", act.Description), Description: act.Description,
				OwnerEmployeeID: act.OwnerEmployeeID, DueDate: act.DueDate, Status: domain.ImpOpen, CreatedBy: a.by()}
			ok, err := tx.InsertImprovement(ctx, imp)
			if err != nil {
				return err
			}
			if ok {
				act.ImprovementID = &imp.ID
				filed++
			}
		}
		r.Status, r.ClosedAt, r.UpdatedBy = domain.RetroClosed, timePtr(m.now()), a.by()
		return tx.UpdateRetro(ctx, *r)
	})
	if err != nil {
		return nil, 0, err
	}
	v, err := m.retroView(ctx, r)
	return v, filed, err
}
