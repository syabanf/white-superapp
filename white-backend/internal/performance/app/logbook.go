package app

import (
	"context"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Logbook ports NüHabit hris/logbook.go: checklist templates per
// department and dated entries per employee. Everyone works in their own
// department; HR readers see every department. Reviews are done by HR
// approvers, the department head or the employee's manager; a rejection
// files an improvement item.
type Logbook struct {
	Deps
	improvements *Improvements
}

func NewLogbook(d Deps, imp *Improvements) *Logbook { return &Logbook{d, imp} }

const (
	msgLogbookGone   = "Logbook tidak ditemukan"
	msgTemplateGone  = "Template tidak ditemukan"
	msgDeptForbidden = "Anda tidak berhak mengakses department ini"
)

// scope resolves the department filter: HR readers may pick any (or all),
// heads their headed departments, everyone else only their own.
func (l *Logbook) scope(a *Actor, requested string) (string, error) {
	if a.HRRead || (requested != "" && a.HeadOf[requested]) {
		return requested, nil
	}
	own := a.DepartmentID()
	if own == "" || (requested != "" && requested != own) {
		return "", apperr.New(apperr.Unauthorized, msgDeptForbidden)
	}
	return own, nil
}

func (l *Logbook) canReview(a *Actor, ppl *people, entry domain.LogbookEntry) bool {
	if a.HRApprove || a.HeadOf[entry.DepartmentID] {
		return true
	}
	e, ok := ppl.employee(entry.EmployeeID)
	return ok && a.Reports[e.ID]
}

// LogbookMe is the page header: the caller's department and rights.
type LogbookMe struct {
	Actor        *Actor
	CanReview    bool
	IsFullAccess bool
}

func (l *Logbook) Me(ctx context.Context) (*LogbookMe, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, err
	}
	return &LogbookMe{Actor: a, CanReview: a.HRApprove || a.IsLead(), IsFullAccess: a.HRRead}, nil
}

func (l *Logbook) ListTemplates(ctx context.Context, departmentID string, includeInactive bool) ([]domain.LogbookTemplate, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, err
	}
	dept, err := l.scope(a, departmentID)
	if err != nil {
		return nil, err
	}
	list, err := l.Store.ListTemplates(ctx, dept, includeInactive)
	if err != nil {
		return nil, err
	}
	if ppl := l.directory(ctx); ppl != nil {
		for i := range list {
			list[i].DepartmentName = ppl.depts[list[i].DepartmentID].Name
		}
	}
	return list, nil
}

// directory is the name lookup for logbook rows. Names are cosmetic, so a
// directory outage is logged and the rows go out unnamed.
func (l *Logbook) directory(ctx context.Context) *people {
	ppl, err := l.people(ctx)
	if err != nil {
		l.log().Warn("logbook names unavailable", "err", err)
		return nil
	}
	return ppl
}

func (p *people) labelEntry(e *domain.LogbookEntry) {
	e.EmployeeName = p.byID[e.EmployeeID].Name
	e.DepartmentName = p.depts[e.DepartmentID].Name
}

// entryOut reloads an entry with its names for the response.
func (l *Logbook) entryOut(ctx context.Context, id string) (*domain.LogbookEntry, error) {
	e, err := l.Store.GetEntry(ctx, id)
	if err != nil || e == nil {
		return e, err
	}
	if ppl := l.directory(ctx); ppl != nil {
		ppl.labelEntry(e)
	}
	return e, nil
}

// TemplateItemInput is one checklist line of a new template.
type TemplateItemInput struct {
	Title       string
	Description *string
	Weight      *float64
	IsRequired  *bool
}

// TemplateInput is POST /logbook/templates.
type TemplateInput struct {
	DepartmentID string
	Name         string
	Description  *string
	Frequency    string
	Items        []TemplateItemInput
}

// CreateTemplate: HR writers for any department, heads for theirs.
func (l *Logbook) CreateTemplate(ctx context.Context, in TemplateInput) (*domain.LogbookTemplate, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, err
	}
	dept := in.DepartmentID
	if dept == "" {
		dept = a.DepartmentID()
	}
	if dept == "" || strings.TrimSpace(in.Name) == "" {
		return nil, apperr.New(apperr.Validation, "department_id dan nama template wajib diisi")
	}
	if !a.HRWrite && !a.HeadOf[dept] {
		return nil, apperr.New(apperr.Unauthorized, msgDeptForbidden)
	}
	freq := in.Frequency
	if freq == "" {
		freq = "daily"
	}
	if !slices.Contains(domain.LogbookFrequencies, freq) {
		return nil, apperr.Field("frequency", "Frekuensi template tidak dikenal")
	}
	t := domain.LogbookTemplate{ID: newID(), DepartmentID: dept, Name: strings.TrimSpace(in.Name), Description: trimToNil(in.Description),
		Frequency: freq, IsActive: true, CreatedBy: a.by()}
	for _, it := range in.Items {
		title := strings.TrimSpace(it.Title)
		if title == "" {
			continue
		}
		required := true
		if it.IsRequired != nil {
			required = *it.IsRequired
		}
		t.Items = append(t.Items, domain.LogbookTemplateItem{ID: newID(), TemplateID: t.ID, Title: title, Description: trimToNil(it.Description),
			Weight: domain.TemplateItemWeight(it.Weight), IsRequired: required, SortOrder: len(t.Items)})
	}
	if len(t.Items) == 0 {
		return nil, apperr.New(apperr.Validation, "Minimal satu checklist item harus diisi")
	}
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertTemplate(ctx, t); err != nil {
			return err
		}
		for _, it := range t.Items {
			if err := tx.InsertTemplateItem(ctx, it); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out, err := l.Store.GetTemplate(ctx, t.ID)
	if out != nil {
		if ppl := l.directory(ctx); ppl != nil {
			out.DepartmentName = ppl.depts[out.DepartmentID].Name
		}
	}
	return out, err
}

// DeleteTemplate archives a template entries use, else deletes it.
func (l *Logbook) DeleteTemplate(ctx context.Context, id string) (string, bool, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return "", false, err
	}
	t, err := l.Store.GetTemplate(ctx, id)
	if err != nil {
		return "", false, err
	}
	if t == nil {
		return "", false, apperr.New(apperr.NotFound, msgTemplateGone)
	}
	if !a.HRWrite && !a.HeadOf[t.DepartmentID] {
		return "", false, apperr.New(apperr.Unauthorized, msgDeptForbidden)
	}
	used, err := l.Store.CountTemplateEntries(ctx, id)
	if err != nil {
		return "", false, err
	}
	if used > 0 {
		return "Template diarsipkan (sudah dipakai logbook)", true, l.Store.ArchiveTemplate(ctx, id)
	}
	return "Template dihapus", false, l.Store.DeleteTemplate(ctx, id)
}

// EntryQuery is GET /logbook/entries.
type EntryQuery struct {
	DepartmentID, EmployeeID, Status, Date, From, To string
	Mine                                             bool
	Page, Limit                                      int
}

// ListEntries: HR readers and heads see their departments, managers their
// reports, everyone their own entries.
func (l *Logbook) ListEntries(ctx context.Context, q EntryQuery) ([]domain.LogbookEntry, int, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, 0, err
	}
	f := domain.EntryFilter{Status: q.Status, Date: q.Date, From: q.From, To: q.To}
	switch {
	case q.Mine:
		if a.EmployeeID() == "" {
			return []domain.LogbookEntry{}, 0, nil
		}
		f.EmployeeIDs = []string{a.EmployeeID()}
	case a.HRRead || (q.DepartmentID != "" && a.HeadOf[q.DepartmentID]):
		f.DepartmentID = q.DepartmentID
	default:
		// Own entries plus direct reports.
		f.EmployeeIDs = []string{a.EmployeeID()}
		for id := range a.Reports {
			f.EmployeeIDs = append(f.EmployeeIDs, id)
		}
		if len(a.HeadOf) > 0 {
			ppl, err := l.people(ctx)
			if err != nil {
				return nil, 0, err
			}
			for dept := range a.HeadOf {
				for _, m := range ppl.members(dept) {
					f.EmployeeIDs = append(f.EmployeeIDs, m.ID)
				}
			}
		}
	}
	if q.EmployeeID != "" {
		if f.EmployeeIDs != nil && !slices.Contains(f.EmployeeIDs, q.EmployeeID) {
			return nil, 0, apperr.New(apperr.Unauthorized, msgDeptForbidden)
		}
		f.EmployeeIDs = []string{q.EmployeeID}
	}
	f.Limit = q.Limit
	if f.Limit == 0 {
		f.Limit = 50
	}
	f.Limit = min(100, max(1, f.Limit))
	f.Offset = (max(1, q.Page) - 1) * f.Limit
	list, total, err := l.Store.ListEntries(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	if ppl := l.directory(ctx); ppl != nil {
		for i := range list {
			ppl.labelEntry(&list[i])
		}
	}
	return list, total, nil
}

func (l *Logbook) guard(ctx context.Context, a *Actor, id string) (*domain.LogbookEntry, *people, error) {
	e, err := l.Store.GetEntry(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if e == nil {
		return nil, nil, apperr.New(apperr.NotFound, msgLogbookGone)
	}
	ppl, err := l.people(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !a.HRRead && !a.IsEmployee(e.EmployeeID) && !l.canReview(a, ppl, *e) {
		return nil, nil, apperr.New(apperr.NotFound, msgLogbookGone)
	}
	return e, ppl, nil
}

func (l *Logbook) GetEntry(ctx context.Context, id string) (*domain.LogbookEntry, bool, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, false, err
	}
	e, ppl, err := l.guard(ctx, a, id)
	if err != nil {
		return nil, false, err
	}
	ppl.labelEntry(e)
	return e, l.canReview(a, ppl, *e), nil
}

// CreateEntry fills a dated entry for the caller from an active template
// of their department.
func (l *Logbook) CreateEntry(ctx context.Context, templateID, entryDate string, title, notes *string) (*domain.LogbookEntry, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, err
	}
	if a.EmployeeID() == "" {
		return nil, errNoEmployee
	}
	if !validDay(entryDate) {
		return nil, apperr.Field("entryDate", "Tanggal logbook tidak valid")
	}
	t, err := l.Store.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, apperr.New(apperr.NotFound, msgTemplateGone)
	}
	if t.DepartmentID != a.DepartmentID() && !a.HeadOf[t.DepartmentID] {
		return nil, apperr.New(apperr.Unauthorized, msgDeptForbidden)
	}
	if !t.IsActive {
		return nil, apperr.New(apperr.Conflict, "Template sudah diarsipkan")
	}
	note, ok := domain.NormalizeLogbookNote(deref(notes))
	if !ok {
		return nil, apperr.New(apperr.Validation, "Catatan tidak valid/terlalu panjang")
	}
	e := domain.LogbookEntry{ID: newID(), TemplateID: t.ID, DepartmentID: t.DepartmentID, EmployeeID: a.EmployeeID(), EntryDate: entryDate,
		Title: domain.LogbookTitle(t.Name, entryDate, deref(title)), Notes: note, CreatedBy: a.by()}
	err = l.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertEntry(ctx, e); err != nil {
			return err
		}
		return tx.CopyTemplateItems(ctx, e.ID, t.ID)
	})
	if _, dup := err.(*domain.ErrUnique); dup {
		return nil, apperr.New(apperr.Conflict, "Logbook untuk template & tanggal ini sudah ada")
	}
	if err != nil {
		return nil, err
	}
	return l.entryOut(ctx, e.ID)
}

// UpdateItem ticks a checklist item or edits its note while draft (owner).
func (l *Logbook) UpdateItem(ctx context.Context, itemID string, checked *bool, notes *string) (*domain.LogbookEntry, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.Store.GetEntryItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, apperr.New(apperr.NotFound, "Item tidak ditemukan")
	}
	e, _, err := l.guard(ctx, a, item.EntryID)
	if err != nil {
		return nil, err
	}
	if !a.IsEmployee(e.EmployeeID) && !a.HRWrite {
		return nil, apperr.New(apperr.Unauthorized, "Hanya pemilik logbook yang boleh mengisi checklist")
	}
	if e.Status != domain.LogbookDraft {
		return nil, apperr.New(apperr.Conflict, "Checklist hanya bisa diubah selama logbook masih draft")
	}
	if checked != nil {
		item.IsChecked, item.CheckedBy, item.CheckedAt = *checked, nil, nil
		if *checked {
			item.CheckedBy, item.CheckedAt = a.by(), timePtr(l.now())
		}
	}
	if notes != nil {
		n, ok := domain.NormalizeLogbookNote(*notes)
		if !ok {
			return nil, apperr.New(apperr.Validation, "Catatan terlalu panjang")
		}
		item.Notes = n
	}
	if err := l.Store.UpdateEntryItem(ctx, *item); err != nil {
		return nil, err
	}
	if err := l.rescore(ctx, e.ID); err != nil {
		return nil, err
	}
	return l.entryOut(ctx, e.ID)
}

func (l *Logbook) rescore(ctx context.Context, entryID string) error {
	e, err := l.Store.GetEntry(ctx, entryID)
	if err != nil || e == nil {
		return err
	}
	states := make([]domain.LogbookItemState, len(e.Items))
	for i, it := range e.Items {
		states[i] = domain.LogbookItemState{Checked: it.IsChecked, Weight: it.Weight}
	}
	completion, kpi := domain.LogbookScore(states)
	return l.Store.SetEntryScore(ctx, entryID, completion, kpi)
}

// Submit moves the caller's draft to submitted. Nil notes keep the stored ones.
func (l *Logbook) Submit(ctx context.Context, id string, notes *string) (*domain.LogbookEntry, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, err
	}
	e, ppl, err := l.guard(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if !a.IsEmployee(e.EmployeeID) && !a.HRWrite {
		return nil, apperr.New(apperr.Unauthorized, "Hanya pemilik logbook yang boleh submit")
	}
	if e.Status != domain.LogbookDraft {
		return nil, apperr.New(apperr.Conflict, "Hanya logbook draft yang bisa disubmit")
	}
	if notes != nil {
		n, ok := domain.NormalizeLogbookNote(*notes)
		if !ok {
			return nil, apperr.New(apperr.Validation, "Catatan tidak valid/terlalu panjang")
		}
		e.Notes = n
	}
	e.Status, e.SubmittedBy, e.SubmittedAt = domain.LogbookSubmitted, a.by(), timePtr(l.now())
	if err := l.Store.UpdateEntry(ctx, *e); err != nil {
		return nil, err
	}
	if emp, ok := ppl.employee(e.EmployeeID); ok {
		l.notify(ctx, Notice{UserIDs: ppl.userIDs(ppl.leadsOf(emp)...), Type: NotifLogbookPending, Href: hrefLogbook,
			Title: "Logbook menunggu review", Body: emp.Name + " mengirim " + e.Title + "."})
	}
	return l.entryOut(ctx, id)
}

// Review marks a submitted entry reviewed or rejected.
func (l *Logbook) Review(ctx context.Context, id string, reject bool, reviewNotes *string) (*domain.LogbookEntry, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, err
	}
	e, ppl, err := l.guard(ctx, a, id)
	if err != nil {
		return nil, err
	}
	if !l.canReview(a, ppl, *e) || a.IsEmployee(e.EmployeeID) {
		return nil, apperr.New(apperr.Unauthorized, "Anda tidak berhak me-review logbook")
	}
	if e.Status != domain.LogbookSubmitted {
		return nil, apperr.New(apperr.Conflict, "Hanya logbook berstatus submitted yang bisa direview")
	}
	note, ok := domain.NormalizeLogbookNote(strings.TrimSpace(deref(reviewNotes)))
	if !ok {
		return nil, apperr.New(apperr.Validation, "Catatan review tidak valid/terlalu panjang")
	}
	e.Status = domain.LogbookReviewed
	if reject {
		e.Status = domain.LogbookRejected
	}
	e.ReviewNotes, e.ReviewedBy, e.ReviewedAt = note, a.by(), timePtr(l.now())
	if err := l.Store.UpdateEntry(ctx, *e); err != nil {
		return nil, err
	}
	if reject {
		l.improvements.fromLogbookReject(ctx, *e, note, a.UserID)
		l.notify(ctx, Notice{UserIDs: ppl.userIDs(e.EmployeeID), Type: NotifLogbookPending, Href: hrefLogbook,
			Title: "Logbook ditolak: " + e.Title, Body: deref(note)})
	}
	return l.entryOut(ctx, id)
}

// DeleteEntry removes the caller's draft.
func (l *Logbook) DeleteEntry(ctx context.Context, id string) error {
	a, err := l.actor(ctx)
	if err != nil {
		return err
	}
	e, _, err := l.guard(ctx, a, id)
	if err != nil {
		return err
	}
	if !a.IsEmployee(e.EmployeeID) && !a.HRWrite {
		return apperr.New(apperr.Unauthorized, "Hanya pemilik logbook yang boleh menghapus")
	}
	if e.Status != domain.LogbookDraft {
		return apperr.New(apperr.Conflict, "Hanya logbook draft yang bisa dihapus")
	}
	return l.Store.DeleteEntry(ctx, id)
}

// Summary groups entries per department (avg completion and KPI score);
// names maps department ids to names.
func (l *Logbook) Summary(ctx context.Context, departmentID, from, to string) ([]domain.LogbookDepartmentSummary, map[string]string, error) {
	a, err := l.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	dept, err := l.scope(a, departmentID)
	if err != nil {
		return nil, nil, err
	}
	rows, err := l.Store.LogbookSummaryRows(ctx, dept, from, to)
	if err != nil {
		return nil, nil, err
	}
	ppl, err := l.people(ctx)
	if err != nil {
		return nil, nil, err
	}
	names := map[string]string{}
	for id, d := range ppl.depts {
		names[id] = d.Name
	}
	return domain.SummarizeLogbookEntries(rows), names, nil
}
