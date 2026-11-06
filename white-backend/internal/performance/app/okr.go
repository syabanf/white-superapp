package app

import (
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// OKR keeps quarterly objectives and their key results. Everyone signed in
// sees the whole board; company objectives are written by HR, department
// ones by HR or the head, employee ones by the employee, their leads or HR.
type OKR struct{ Deps }

func NewOKR(d Deps) *OKR { return &OKR{d} }

var periodRe = regexp.MustCompile(`^\d{4}-Q[1-4]$`)

const (
	msgObjectiveGone = "Objective tidak ditemukan"
	msgKeyResultGone = "Key result tidak ditemukan"
	msgOKRDenied     = "Anda tidak bisa mengubah OKR ini"
)

// ObjectiveView is an objective with its owner's name, key results and progress.
type ObjectiveView struct {
	domain.Objective
	OwnerName  string
	KeyResults []KeyResultView
	Progress   float64
	CanEdit    bool
}

type KeyResultView struct {
	domain.KeyResult
	Progress float64
	// Confidence is the last check-in's, 0 before the first one.
	Confidence int
}

func (o *OKR) canEdit(a *Actor, obj domain.Objective, ppl *people) bool {
	switch obj.Level {
	case domain.OKRCompany:
		return a.HRWrite
	case domain.OKRDepartment:
		return a.HRWrite || a.HeadOf[deref(obj.DepartmentID)]
	default:
		if a.HRWrite || a.IsEmployee(deref(obj.EmployeeID)) {
			return true
		}
		e, ok := ppl.byID[deref(obj.EmployeeID)]
		return ok && a.Leads(e)
	}
}

// Owner is a department or employee the caller may write OKRs for.
type Owner struct{ ID, Name string }

// BoardView is the period's objectives plus where the caller may add more.
type BoardView struct {
	Objectives  []ObjectiveView
	CanCompany  bool
	Departments []Owner
	Employees   []Owner
}

// Board lists the period's objectives, company first.
func (o *OKR) Board(ctx context.Context, period string) (*BoardView, error) {
	a, err := o.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !periodRe.MatchString(period) {
		return nil, apperr.Field("period", "Periode harus berformat YYYY-Qn")
	}
	ppl, err := o.people(ctx)
	if err != nil {
		return nil, err
	}
	view := &BoardView{CanCompany: a.HRWrite}
	for _, d := range ppl.depts {
		if d.Active && (a.HRWrite || a.HeadOf[d.ID]) {
			view.Departments = append(view.Departments, Owner{ID: d.ID, Name: d.Name})
		}
	}
	for _, e := range ppl.byID {
		if e.Active && o.canEdit(a, domain.Objective{Level: domain.OKREmployee, EmployeeID: &e.ID}, ppl) {
			view.Employees = append(view.Employees, Owner{ID: e.ID, Name: e.Name})
		}
	}
	byName := func(x, y Owner) int { return strings.Compare(x.Name, y.Name) }
	slices.SortFunc(view.Departments, byName)
	slices.SortFunc(view.Employees, byName)
	objs, err := o.Store.ListObjectives(ctx, period)
	if err != nil {
		return nil, err
	}
	krs, err := o.Store.ListKeyResults(ctx, period)
	if err != nil {
		return nil, err
	}
	conf, err := o.Store.LatestConfidence(ctx, period)
	if err != nil {
		return nil, err
	}
	out := make([]ObjectiveView, 0, len(objs))
	for _, obj := range objs {
		v := ObjectiveView{Objective: obj, CanEdit: o.canEdit(a, obj, ppl)}
		switch obj.Level {
		case domain.OKRDepartment:
			v.OwnerName = ppl.deptName(obj.DepartmentID)
		case domain.OKREmployee:
			v.OwnerName = ppl.name(obj.EmployeeID)
		}
		for _, k := range krs {
			if k.ObjectiveID == obj.ID {
				v.KeyResults = append(v.KeyResults, KeyResultView{KeyResult: k, Progress: k.Progress(), Confidence: conf[k.ID]})
				v.Progress += k.Progress()
			}
		}
		if len(v.KeyResults) > 0 {
			v.Progress /= float64(len(v.KeyResults))
		}
		out = append(out, v)
	}
	view.Objectives = out
	return view, nil
}

type ObjectiveInput struct {
	ID, Period, Level, DepartmentID, EmployeeID, ParentID, Title, Description, Status string
}

// SaveObjective creates or edits an objective. Period, level and owner are
// fixed once created.
func (o *OKR) SaveObjective(ctx context.Context, in ObjectiveInput) (*domain.Objective, error) {
	a, err := o.actor(ctx)
	if err != nil {
		return nil, err
	}
	ppl, err := o.people(ctx)
	if err != nil {
		return nil, err
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, apperr.Field("title", "Judul objective wajib diisi")
	}
	if in.Status == "" {
		in.Status = "active"
	}
	if !slices.Contains([]string{"active", "done", "cancelled"}, in.Status) {
		return nil, apperr.Field("status", "Status tidak valid")
	}
	var obj domain.Objective
	if in.ID != "" {
		cur, err := o.Store.GetObjective(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if cur == nil {
			return nil, apperr.New(apperr.NotFound, msgObjectiveGone)
		}
		obj = *cur
	} else {
		if !periodRe.MatchString(in.Period) {
			return nil, apperr.Field("period", "Periode harus berformat YYYY-Qn")
		}
		obj = domain.Objective{ID: newID(), Period: in.Period, Level: in.Level, CreatedBy: a.by()}
		switch in.Level {
		case domain.OKRCompany:
		case domain.OKRDepartment:
			if _, ok := ppl.depts[in.DepartmentID]; !ok {
				return nil, apperr.Field("department_id", "Pilih departemen")
			}
			obj.DepartmentID = strPtr(in.DepartmentID)
		case domain.OKREmployee:
			if in.EmployeeID == "" {
				in.EmployeeID = a.EmployeeID()
			}
			if _, ok := ppl.byID[in.EmployeeID]; !ok {
				return nil, apperr.Field("employee_id", "Pilih karyawan")
			}
			obj.EmployeeID = strPtr(in.EmployeeID)
		default:
			return nil, apperr.Field("level", "Level tidak valid")
		}
	}
	if !o.canEdit(a, obj, ppl) {
		return nil, apperr.New(apperr.Unauthorized, msgOKRDenied)
	}
	obj.ParentID = nil
	if in.ParentID != "" && in.ParentID != obj.ID {
		parent, err := o.Store.GetObjective(ctx, in.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.Period != obj.Period {
			return nil, apperr.Field("parent_id", "Objective induk harus dari periode yang sama")
		}
		obj.ParentID = &parent.ID
	}
	obj.Title, obj.Description, obj.Status = in.Title, strings.TrimSpace(in.Description), in.Status
	if err := o.Store.SaveObjective(ctx, obj); err != nil {
		return nil, err
	}
	return o.Store.GetObjective(ctx, obj.ID)
}

// editable loads an objective the caller may change.
func (o *OKR) editable(ctx context.Context, a *Actor, id string) (*domain.Objective, error) {
	obj, err := o.Store.GetObjective(ctx, id)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, apperr.New(apperr.NotFound, msgObjectiveGone)
	}
	ppl, err := o.people(ctx)
	if err != nil {
		return nil, err
	}
	if !o.canEdit(a, *obj, ppl) {
		return nil, apperr.New(apperr.Unauthorized, msgOKRDenied)
	}
	return obj, nil
}

func (o *OKR) DeleteObjective(ctx context.Context, id string) error {
	a, err := o.actor(ctx)
	if err != nil {
		return err
	}
	if _, err := o.editable(ctx, a, id); err != nil {
		return err
	}
	_, err = o.Store.DeleteObjective(ctx, id)
	return err
}

type KeyResultInput struct {
	ID, ObjectiveID, Title, Metric, Unit string
	Start, Target                        float64
	SortOrder                            int
}

func (o *OKR) SaveKeyResult(ctx context.Context, in KeyResultInput) (*domain.KeyResult, error) {
	a, err := o.actor(ctx)
	if err != nil {
		return nil, err
	}
	k := domain.KeyResult{ID: in.ID, ObjectiveID: in.ObjectiveID}
	if in.ID != "" {
		cur, err := o.Store.GetKeyResult(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if cur == nil {
			return nil, apperr.New(apperr.NotFound, msgKeyResultGone)
		}
		k = *cur
	} else {
		k.ID, k.Current = newID(), in.Start
	}
	if _, err := o.editable(ctx, a, k.ObjectiveID); err != nil {
		return nil, err
	}
	in.Title = strings.TrimSpace(in.Title)
	switch {
	case in.Title == "":
		return nil, apperr.Field("title", "Judul key result wajib diisi")
	case !slices.Contains([]string{"number", "percent", "currency"}, in.Metric):
		return nil, apperr.Field("metric", "Jenis ukuran tidak valid")
	case in.Target == in.Start:
		return nil, apperr.Field("target_value", "Target harus berbeda dari nilai awal")
	}
	k.Title, k.Metric, k.Unit, k.Start, k.Target, k.SortOrder = in.Title, in.Metric, strings.TrimSpace(in.Unit), in.Start, in.Target, in.SortOrder
	if err := o.Store.SaveKeyResult(ctx, k); err != nil {
		return nil, err
	}
	return o.Store.GetKeyResult(ctx, k.ID)
}

func (o *OKR) DeleteKeyResult(ctx context.Context, id string) error {
	a, err := o.actor(ctx)
	if err != nil {
		return err
	}
	k, err := o.Store.GetKeyResult(ctx, id)
	if err != nil {
		return err
	}
	if k == nil {
		return apperr.New(apperr.NotFound, msgKeyResultGone)
	}
	if _, err := o.editable(ctx, a, k.ObjectiveID); err != nil {
		return err
	}
	_, err = o.Store.DeleteKeyResult(ctx, id)
	return err
}

// CheckIn records a new value and confidence and moves the key result.
func (o *OKR) CheckIn(ctx context.Context, keyResultID string, value float64, confidence int, note string) (*domain.KeyResult, error) {
	a, err := o.actor(ctx)
	if err != nil {
		return nil, err
	}
	k, err := o.Store.GetKeyResult(ctx, keyResultID)
	if err != nil {
		return nil, err
	}
	if k == nil {
		return nil, apperr.New(apperr.NotFound, msgKeyResultGone)
	}
	obj, err := o.editable(ctx, a, k.ObjectiveID)
	if err != nil {
		return nil, err
	}
	if obj.Status != "active" {
		return nil, apperr.New(apperr.Conflict, "Objective ini sudah ditutup")
	}
	if confidence < 1 || confidence > 10 {
		return nil, apperr.Field("confidence", "Keyakinan diisi 1 sampai 10")
	}
	err = o.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertCheckin(ctx, domain.Checkin{ID: newID(), KeyResultID: k.ID, Value: value, Confidence: confidence, Note: strings.TrimSpace(note), CreatedBy: a.by()}); err != nil {
			return err
		}
		return tx.SetKeyResultValue(ctx, k.ID, value)
	})
	if err != nil {
		return nil, err
	}
	return o.Store.GetKeyResult(ctx, k.ID)
}

func (o *OKR) Checkins(ctx context.Context, keyResultID string) ([]domain.Checkin, error) {
	if _, err := o.actor(ctx); err != nil {
		return nil, err
	}
	return o.Store.ListCheckins(ctx, keyResultID)
}
