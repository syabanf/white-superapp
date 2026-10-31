package app

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/performance/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// KPI ports NüHabit payroll/kpi.go and kpi_snapshot.go: indicator and
// department configuration, targets, the monthly snapshot, scorecards,
// the supervisor rubric and recommendations.
type KPI struct{ Deps }

func NewKPI(d Deps) *KPI { return &KPI{d} }

const (
	msgPeriodInvalid   = "Periode tidak valid"
	msgScorecardGone   = "Scorecard tidak ditemukan"
	msgDeptNotFound    = "Departemen tidak ditemukan"
	msgIndicatorGone   = "Indikator KPI tidak ditemukan"
	msgEmployeeGone    = "Karyawan tidak ditemukan"
	msgRubricForbidden = "Hanya HRD atau atasan langsung yang boleh menilai"
)

func (k *KPI) ListIndicators(ctx context.Context, includeInactive bool) ([]domain.Indicator, error) {
	if _, err := k.actor(ctx); err != nil {
		return nil, err
	}
	return k.Store.ListIndicators(ctx, includeInactive)
}

// IndicatorInput creates (ID empty) or updates an indicator.
type IndicatorInput struct {
	ID            string
	Code          string
	Name          string
	Description   string
	Unit          string
	Direction     string
	SourceKind    string
	DefaultTarget *float64
	IsActive      bool
}

var codeRE = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)

func (k *KPI) SaveIndicator(ctx context.Context, in IndicatorInput) (*domain.Indicator, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireHRWrite(a); err != nil {
		return nil, err
	}
	in.Code, in.Name = strings.TrimSpace(in.Code), strings.TrimSpace(in.Name)
	switch {
	case !codeRE.MatchString(in.Code):
		return nil, apperr.Field("code", "Kode indikator huruf kecil, angka, atau garis bawah (2-50 karakter)")
	case len([]rune(in.Name)) < 3:
		return nil, apperr.Field("name", "Nama indikator minimal 3 karakter")
	case !domain.ValidDirection(in.Direction):
		return nil, apperr.Field("direction", "Arah indikator tidak dikenal")
	case in.SourceKind != domain.SourceAuto && in.SourceKind != domain.SourceManual:
		return nil, apperr.Field("sourceKind", "Sumber data indikator tidak dikenal")
	case in.DefaultTarget != nil && (!domain.Finite(*in.DefaultTarget) || *in.DefaultTarget < 0):
		return nil, apperr.Field("defaultTarget", "Target bawaan harus angka ≥ 0")
	}
	if in.ID == "" {
		in.ID = newID()
	} else if cur, err := k.Store.GetIndicator(ctx, in.ID); err != nil {
		return nil, err
	} else if cur == nil {
		return nil, apperr.New(apperr.NotFound, msgIndicatorGone)
	}
	ind := domain.Indicator{ID: in.ID, Code: in.Code, Name: in.Name, Description: strings.TrimSpace(in.Description), Unit: strings.TrimSpace(in.Unit),
		Direction: in.Direction, SourceKind: in.SourceKind, DefaultTarget: in.DefaultTarget, IsActive: in.IsActive, UpdatedBy: a.by()}
	if err := k.Store.UpsertIndicator(ctx, ind); err != nil {
		if _, dup := err.(*domain.ErrUnique); dup {
			return nil, apperr.Field("code", "Kode indikator sudah dipakai")
		}
		return nil, err
	}
	return k.Store.GetIndicator(ctx, ind.ID)
}

// DeleteIndicator removes an indicator nobody has used yet. One with
// snapshots, task contributions or linked tasks is deactivated instead, so
// past scorecards keep their detail rows; the message says which happened.
func (k *KPI) DeleteIndicator(ctx context.Context, id string) (string, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return "", err
	}
	if err := requireHRWrite(a); err != nil {
		return "", err
	}
	cur, err := k.Store.GetIndicator(ctx, id)
	if err != nil {
		return "", err
	}
	if cur == nil {
		return "", apperr.New(apperr.NotFound, msgIndicatorGone)
	}
	deleted, err := k.Store.DeleteUnusedIndicator(ctx, id)
	if err != nil {
		return "", err
	}
	if deleted {
		return "Indikator " + cur.Name + " dihapus", nil
	}
	if err := k.Store.DeactivateIndicator(ctx, id, a.by()); err != nil {
		return "", err
	}
	return "Indikator " + cur.Name + " sudah punya riwayat KPI, jadi dinonaktifkan agar scorecard lama tetap utuh", nil
}

// KPIConfig is GET /kpi-config: departments, active indicators, weights.
type KPIConfig struct {
	Departments []domain.Department
	Indicators  []domain.Indicator
	Weights     []domain.DepartmentWeight
}

func (k *KPI) Config(ctx context.Context) (*KPIConfig, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !a.HRRead && len(a.HeadOf) == 0 {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	depts, err := k.Directory.Departments(ctx)
	if err != nil {
		return nil, err
	}
	inds, err := k.Store.ListIndicators(ctx, false)
	if err != nil {
		return nil, err
	}
	weights, err := k.Store.ListDepartmentWeights(ctx)
	if err != nil {
		return nil, err
	}
	return &KPIConfig{Departments: depts, Indicators: inds, Weights: weights}, nil
}

// WeightInput is one enabled indicator of a department.
type WeightInput struct {
	IndicatorID string
	Weight      float64
}

// SaveDepartmentConfig replaces a department's weights (putKPIConfig).
func (k *KPI) SaveDepartmentConfig(ctx context.Context, departmentID string, items []WeightInput) (string, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return "", err
	}
	if err := requireHRWrite(a); err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", apperr.Field("items", "Minimal satu indikator harus aktif untuk departemen ini")
	}
	for _, it := range items {
		if !domain.Finite(it.Weight) || it.Weight <= 0 || it.Weight > 100 {
			return "", apperr.Field("items", "Bobot indikator aktif harus 1–100")
		}
		ind, err := k.Store.GetIndicator(ctx, it.IndicatorID)
		if err != nil {
			return "", err
		}
		if ind == nil || !ind.IsActive {
			return "", apperr.Field("items", "ID indikator tidak valid")
		}
	}
	depts, err := k.Directory.Departments(ctx)
	if err != nil {
		return "", err
	}
	name := ""
	for _, d := range depts {
		if d.ID == departmentID {
			name = d.Name
		}
	}
	if name == "" {
		return "", apperr.New(apperr.NotFound, msgDeptNotFound)
	}
	editor := a.Name
	if editor == "" {
		editor = "Pengelola KPI"
	}
	err = k.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.DeleteDepartmentWeights(ctx, departmentID); err != nil {
			return err
		}
		for _, it := range items {
			if err := tx.InsertDepartmentWeight(ctx, domain.DepartmentWeight{DepartmentID: departmentID, IndicatorID: it.IndicatorID,
				Weight: it.Weight, UpdatedBy: &editor}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return "Konfigurasi KPI departemen " + name + " disimpan, berlaku mulai snapshot bulan berikutnya", nil
}

func (k *KPI) ListTargets(ctx context.Context) ([]domain.KPITarget, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !a.HRRead {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	return k.Store.ListTargets(ctx)
}

const targetRequired = "indicator_id dan target (angka ≥ 0) wajib"

func (k *KPI) CreateTarget(ctx context.Context, t domain.KPITarget) (*domain.KPITarget, error) {
	a, err := k.validTarget(ctx, &t)
	if err != nil {
		return nil, err
	}
	t.ID = newID()
	if err := k.Store.InsertTarget(ctx, t, a.UserID); err != nil {
		return nil, err
	}
	return &t, nil
}

func (k *KPI) UpdateTarget(ctx context.Context, t domain.KPITarget) (*domain.KPITarget, error) {
	if _, err := k.validTarget(ctx, &t); err != nil {
		return nil, err
	}
	found, err := k.Store.UpdateTarget(ctx, t)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, apperr.New(apperr.NotFound, "Target tidak ditemukan")
	}
	return &t, nil
}

// validTarget checks the caller and the fields shared by create and update,
// trimming the scope ids in place.
func (k *KPI) validTarget(ctx context.Context, t *domain.KPITarget) (*Actor, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, err
	}
	if err := requireHRWrite(a); err != nil {
		return nil, err
	}
	if t.IndicatorID == "" || !domain.Finite(t.Target) || t.Target < 0 {
		return nil, apperr.New(apperr.Validation, targetRequired)
	}
	if (t.PeriodYear != nil && (*t.PeriodYear < 2020 || *t.PeriodYear > 2100)) || (t.PeriodMonth != nil && (*t.PeriodMonth < 1 || *t.PeriodMonth > 12)) {
		return nil, apperr.New(apperr.Validation, msgPeriodInvalid)
	}
	t.DepartmentID, t.EmployeeID = trimToNil(t.DepartmentID), trimToNil(t.EmployeeID)
	if t.DepartmentID != nil && t.EmployeeID != nil {
		return nil, apperr.New(apperr.Validation, "Pilih satu scope saja (department ATAU karyawan)")
	}
	ind, err := k.Store.GetIndicator(ctx, t.IndicatorID)
	if err != nil {
		return nil, err
	}
	if ind == nil {
		return nil, apperr.New(apperr.Validation, "Indikator/department/karyawan tidak ditemukan")
	}
	return a, nil
}

func (k *KPI) DeleteTarget(ctx context.Context, id string) error {
	a, err := k.actor(ctx)
	if err != nil {
		return err
	}
	if err := requireHRWrite(a); err != nil {
		return err
	}
	_, err = k.Store.DeleteTarget(ctx, id)
	return err
}

// ─── Monthly snapshot ────────────────────────────────────────

// SnapshotSummary is what a snapshot run reports.
type SnapshotSummary struct {
	Period             domain.Period
	Employees          int
	SnapshotsUpserted  int
	ScorecardsUpserted int
	SkippedFinal       int
	Collectors         map[string]int
}

// RunSnapshot is POST /kpi/snapshot for HR writers.
func (k *KPI) RunSnapshot(ctx context.Context, p domain.Period) (*SnapshotSummary, string, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, "", err
	}
	if err := requireHRWrite(a); err != nil {
		return nil, "", err
	}
	if !p.Valid() {
		return nil, "", apperr.New(apperr.Validation, msgPeriodInvalid)
	}
	sum, err := k.Snapshot(ctx, p)
	if err != nil {
		return nil, "", err
	}
	skipped := ""
	if sum.SkippedFinal > 0 {
		skipped = fmt.Sprintf(", %d final dilewati", sum.SkippedFinal)
	}
	return sum, fmt.Sprintf("Snapshot KPI %d/%d: %d scorecard diperbarui%s", p.Month, p.Year, sum.ScorecardsUpserted, skipped), nil
}

// Snapshot (runKpiSnapshot) is idempotent: a re-run upserts snapshots and
// draft scorecards; final scorecards and their snapshots stay frozen.
func (k *KPI) Snapshot(ctx context.Context, p domain.Period) (*SnapshotSummary, error) {
	ppl, err := k.people(ctx)
	if err != nil {
		return nil, err
	}
	active := ppl.active()
	employees := make([]domain.KPIEmployee, len(active))
	for i, e := range active {
		employees[i] = domain.KPIEmployee{ID: e.ID, DepartmentID: e.DepartmentID}
	}
	cfg, err := k.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	col, err := k.collect(ctx, employees, p)
	if err != nil {
		return nil, err
	}
	finalIDs, err := k.Store.FinalScorecardEmployees(ctx, p)
	if err != nil {
		return nil, err
	}
	finals := map[string]bool{}
	for _, id := range finalIDs {
		finals[id] = true
	}
	manualRows, err := k.Store.ListManualSnapshots(ctx, p)
	if err != nil {
		return nil, err
	}
	manual := map[string]*float64{}
	for _, s := range manualRows {
		manual[s.EmployeeID+":"+s.IndicatorID] = s.Attainment
	}

	var snaps []domain.Snapshot
	var cards []domain.Scorecard
	for _, e := range employees {
		if finals[e.ID] || e.DepartmentID == nil {
			continue
		}
		comps := cfg.byDepartment[*e.DepartmentID]
		if len(comps) == 0 {
			continue // no configuration for this department
		}
		var parts []domain.ScoreComponent
		for _, c := range comps {
			ind, ok := cfg.indicators[c.IndicatorID]
			if !ok {
				continue
			}
			var attain *float64
			if v, collected := col[ind.Code][e.ID]; collected && ind.SourceKind == domain.SourceAuto {
				target := domain.ResolveTarget(cfg.targets[ind.ID], domain.TargetContext{EmployeeID: e.ID, DepartmentID: e.DepartmentID,
					Month: p.Month, Year: p.Year}, ind.DefaultTarget)
				attain = domain.Attainment(ind.Direction, v.Actual, target)
				detail, _ := json.Marshal(v.Detail)
				snaps = append(snaps, domain.Snapshot{ID: newID(), EmployeeID: e.ID, IndicatorID: ind.ID, Period: p, Actual: v.Actual,
					Target: target, Attainment: attain, SampleSize: v.SampleSize, SourceDetail: detail})
			} else {
				// Manual indicators (rubric, manual actuals) keep their stored attainment.
				attain = manual[e.ID+":"+ind.ID]
			}
			parts = append(parts, domain.ScoreComponent{Code: ind.Code, Weight: c.Weight, Attainment: attain})
		}
		score := domain.ComposeScore(parts)
		cards = append(cards, domain.Scorecard{ID: newID(), EmployeeID: e.ID, DepartmentID: e.DepartmentID, Period: p, Score: score.Score,
			RawScore: score.RawScore, UsedWeight: score.UsedWeight, Breakdown: score.Breakdown})
	}
	sum := &SnapshotSummary{Period: p, Employees: len(employees), SkippedFinal: len(finals), Collectors: map[string]int{}}
	for code, values := range col {
		sum.Collectors[code] = len(values)
	}
	err = k.Store.WithTx(ctx, func(tx domain.Store) error {
		for _, s := range snaps {
			if err := tx.UpsertSnapshot(ctx, s); err != nil {
				return err
			}
		}
		sum.SnapshotsUpserted = len(snaps)
		for _, c := range cards {
			ok, err := tx.UpsertDraftScorecard(ctx, c)
			if err != nil {
				return err
			}
			if ok {
				sum.ScorecardsUpserted++
			}
		}
		return nil
	})
	return sum, err
}

type kpiConfig struct {
	indicators   map[string]domain.Indicator
	byDepartment map[string][]domain.DepartmentWeight
	targets      map[string][]domain.KPITarget
}

func (k *KPI) loadConfig(ctx context.Context) (*kpiConfig, error) {
	cfg := &kpiConfig{indicators: map[string]domain.Indicator{}, byDepartment: map[string][]domain.DepartmentWeight{}, targets: map[string][]domain.KPITarget{}}
	inds, err := k.Store.ListIndicators(ctx, false)
	if err != nil {
		return nil, err
	}
	for _, i := range inds {
		cfg.indicators[i.ID] = i
	}
	weights, err := k.Store.ListDepartmentWeights(ctx)
	if err != nil {
		return nil, err
	}
	for _, w := range weights {
		cfg.byDepartment[w.DepartmentID] = append(cfg.byDepartment[w.DepartmentID], w)
	}
	targets, err := k.Store.ListTargets(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range targets {
		cfg.targets[t.IndicatorID] = append(cfg.targets[t.IndicatorID], t)
	}
	return cfg, nil
}

// collect runs the collectors for a month: attendance from hr (skipped
// with a warning when hr is down), logbook compliance and task completion
// (due up to today).
func (k *KPI) collect(ctx context.Context, employees []domain.KPIEmployee, p domain.Period) (map[string]map[string]domain.CollectorValue, error) {
	out := map[string]map[string]domain.CollectorValue{}
	span := p.Span()
	ids := make([]string, len(employees))
	for i, e := range employees {
		ids[i] = e.ID
	}
	att, err := k.Directory.Attendance(ctx, ids, p)
	if err != nil {
		k.log().Warn("kpi snapshot: attendance unavailable", "period", p.Key(), "err", err)
	} else {
		ontime, late := map[string]domain.CollectorValue{}, map[string]domain.CollectorValue{}
		for id, a := range att {
			ontime[id] = domain.AttendanceOntime(a)
			if v, ok := domain.AttendanceLateRatio(a); ok {
				late[id] = v
			}
		}
		out[domain.CodeAttOntime] = ontime
		out[domain.CodeAttLateRatio] = late
		out[domain.CodeTeamOntime] = domain.TeamOntime(employees, ontime)
	}
	stats, err := k.Store.LogbookStats(ctx, span)
	if err != nil {
		return nil, err
	}
	counts := map[string]domain.LogbookCount{}
	for id, s := range stats {
		counts[id] = s.LogbookCount
	}
	out[domain.CodeLogbookCompliance] = domain.LogbookCompliance(counts)
	due := span
	if today := k.today(); today < due.End {
		due.End = today
	}
	tasks, err := k.Store.TaskCounts(ctx, due)
	if err != nil {
		return nil, err
	}
	out[domain.CodeTaskCompletion] = domain.TaskCompletion(employees, tasks)
	return out, nil
}

// recompose rebuilds one employee's draft scorecard from stored snapshots
// with the department weights; "skipped_final" or "no_components" when
// nothing changes.
func (k *KPI) recompose(ctx context.Context, e domain.Employee, p domain.Period) (*float64, string, error) {
	if e.DepartmentID == nil {
		return nil, "no_components", nil
	}
	cfg, err := k.loadConfig(ctx)
	if err != nil {
		return nil, "", err
	}
	comps := cfg.byDepartment[*e.DepartmentID]
	if len(comps) == 0 {
		return nil, "no_components", nil
	}
	cur, err := k.Store.GetScorecardFor(ctx, e.ID, p)
	if err != nil {
		return nil, "", err
	}
	if cur != nil && cur.Status == domain.ScorecardFinal {
		return nil, "skipped_final", nil
	}
	snaps, err := k.Store.ListSnapshots(ctx, e.ID, p)
	if err != nil {
		return nil, "", err
	}
	attain := map[string]*float64{}
	for _, s := range snaps {
		attain[s.IndicatorID] = s.Attainment
	}
	parts := make([]domain.ScoreComponent, 0, len(comps))
	for _, c := range comps {
		parts = append(parts, domain.ScoreComponent{Code: c.IndicatorCode, Weight: c.Weight, Attainment: attain[c.IndicatorID]})
	}
	score := domain.ComposeScore(parts)
	if _, err := k.Store.UpsertDraftScorecard(ctx, domain.Scorecard{ID: newID(), EmployeeID: e.ID, DepartmentID: e.DepartmentID, Period: p,
		Score: score.Score, RawScore: score.RawScore, UsedWeight: score.UsedWeight, Breakdown: score.Breakdown}); err != nil {
		return nil, "", err
	}
	return score.Score, "updated", nil
}

// ─── Scorecards ──────────────────────────────────────────────

// ScorecardQuery is listScorecards' query: Team lists the caller's team,
// History an employee's latest N months.
type ScorecardQuery struct {
	Period     domain.Period
	EmployeeID string
	Team       bool
	History    int
}

// ScorecardView is a scorecard with names.
type ScorecardView struct {
	domain.Scorecard
	EmployeeName   string
	DepartmentName string
}

func scorecardViews(ppl *people, cards []domain.Scorecard) []ScorecardView {
	out := make([]ScorecardView, len(cards))
	for i, c := range cards {
		out[i] = ScorecardView{Scorecard: c, EmployeeName: ppl.name(&c.EmployeeID), DepartmentName: ppl.deptName(c.DepartmentID)}
	}
	return out
}

func (k *KPI) ListScorecards(ctx context.Context, q ScorecardQuery) ([]ScorecardView, error) {
	cards, ppl, err := k.listScorecards(ctx, q)
	if err != nil {
		return nil, err
	}
	return scorecardViews(ppl, cards), nil
}

func (k *KPI) listScorecards(ctx context.Context, q ScorecardQuery) ([]domain.Scorecard, *people, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !q.Period.Valid() {
		y, m := k.CurrentPeriod()
		q.Period = domain.Period{Year: y, Month: m}
	}
	ppl, err := k.people(ctx)
	if err != nil {
		return nil, nil, err
	}
	if q.Team {
		var team []string
		for _, e := range ppl.active() {
			if a.Leads(e) && !a.IsEmployee(e.ID) {
				team = append(team, e.ID)
			}
		}
		if len(team) == 0 {
			return []domain.Scorecard{}, ppl, nil
		}
		cards, err := k.Store.ListScorecards(ctx, q.Period, team)
		return cards, ppl, err
	}
	filter := q.EmployeeID
	if filter == "me" {
		filter = a.EmployeeID()
		if filter == "" {
			return []domain.Scorecard{}, ppl, nil
		}
	}
	if filter != "" {
		e, ok := ppl.employee(filter)
		if !ok || !a.SeesEmployee(e) {
			return nil, nil, apperr.New(apperr.Unauthorized, "")
		}
	} else if !a.HRRead {
		// Non-HR is locked to their own scorecards.
		if filter = a.EmployeeID(); filter == "" {
			return []domain.Scorecard{}, ppl, nil
		}
	}
	if q.History > 0 {
		if filter == "" {
			return nil, nil, apperr.New(apperr.Validation, "history membutuhkan employee_id")
		}
		cards, err := k.Store.ScorecardHistory(ctx, filter, min(24, q.History))
		return cards, ppl, err
	}
	var ids []string
	if filter != "" {
		ids = []string{filter}
	}
	cards, err := k.Store.ListScorecards(ctx, q.Period, ids)
	return cards, ppl, err
}

// ScorecardDetail is a scorecard with its snapshots.
type ScorecardDetail struct {
	Scorecard ScorecardView
	Snapshots []domain.Snapshot
	CanRate   bool
}

func (k *KPI) GetScorecard(ctx context.Context, id string) (*ScorecardDetail, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, err
	}
	sc, err := k.Store.GetScorecard(ctx, id)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, apperr.New(apperr.NotFound, msgScorecardGone)
	}
	ppl, err := k.people(ctx)
	if err != nil {
		return nil, err
	}
	e, _ := ppl.employee(sc.EmployeeID)
	e.ID = sc.EmployeeID
	if !a.SeesEmployee(e) {
		return nil, apperr.New(apperr.NotFound, msgScorecardGone)
	}
	snaps, err := k.Store.ListSnapshots(ctx, sc.EmployeeID, sc.Period)
	if err != nil {
		return nil, err
	}
	return &ScorecardDetail{Scorecard: scorecardViews(ppl, []domain.Scorecard{*sc})[0], Snapshots: snaps, CanRate: a.HRWrite || a.Leads(e)}, nil
}

// SetScorecardStatus finalizes or reopens (hr.approve).
func (k *KPI) SetScorecardStatus(ctx context.Context, id, action string) (*ScorecardView, string, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, "", err
	}
	if err := requireHRApprove(a); err != nil {
		return nil, "", err
	}
	sc, err := k.Store.GetScorecard(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if sc == nil {
		return nil, "", apperr.New(apperr.NotFound, msgScorecardGone)
	}
	switch action {
	case "finalize":
		if sc.Status != domain.ScorecardDraft {
			return nil, "", apperr.New(apperr.Conflict, "Scorecard sudah final")
		}
		err = k.Store.SetScorecardStatus(ctx, id, domain.ScorecardFinal, a.by(), timePtr(k.now()))
	case "reopen":
		if sc.Status != domain.ScorecardFinal {
			return nil, "", apperr.New(apperr.Conflict, "Scorecard masih draft")
		}
		err = k.Store.SetScorecardStatus(ctx, id, domain.ScorecardDraft, nil, nil)
	default:
		return nil, "", apperr.New(apperr.Validation, "action (finalize|reopen) dan scorecard_id wajib")
	}
	if err != nil {
		return nil, "", err
	}
	sc, err = k.Store.GetScorecard(ctx, id)
	if err != nil {
		return nil, "", err
	}
	ppl, err := k.people(ctx)
	if err != nil {
		return nil, "", err
	}
	view := scorecardViews(ppl, []domain.Scorecard{*sc})[0]
	if action == "reopen" {
		return &view, "Scorecard dibuka kembali", nil
	}
	{
		scoreText := "belum ada data"
		if sc.Score != nil {
			scoreText = domain.FormatDecimal(*sc.Score, 2)
		}
		k.notify(ctx, Notice{UserIDs: ppl.userIDs(sc.EmployeeID), Type: NotifKPIFinal, Href: hrefKPI,
			Title: "Skor KPI " + sc.Period.Label() + " sudah final",
			Body:  fmt.Sprintf("Skor KPI Anda periode %s sudah FINAL: %s.", sc.Period.Label(), scoreText)})
	}
	return &view, "Scorecard difinalkan", nil
}

// ManualInput rates the supervisor rubric (Indicator empty) or records a
// manual indicator's actual for one employee and month.
type ManualInput struct {
	EmployeeID  string
	IndicatorID string
	Period      domain.Period
	Value       float64
	Notes       *string
}

// SaveManual stores the measurement, then recomposes the draft scorecard.
func (k *KPI) SaveManual(ctx context.Context, in ManualInput) (*float64, string, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, "", err
	}
	if in.EmployeeID == "" || !in.Period.Valid() {
		return nil, "", apperr.New(apperr.Validation, "Parameter tidak lengkap")
	}
	ppl, err := k.people(ctx)
	if err != nil {
		return nil, "", err
	}
	e, ok := ppl.employee(in.EmployeeID)
	if !ok {
		return nil, "", apperr.New(apperr.NotFound, msgEmployeeGone)
	}
	if !a.HRWrite && !a.Leads(e) {
		return nil, "", apperr.New(apperr.Unauthorized, msgRubricForbidden)
	}
	var ind *domain.Indicator
	if in.IndicatorID == "" {
		if ind, err = k.Store.GetIndicatorByCode(ctx, domain.CodeSupervisorRubric); err != nil {
			return nil, "", err
		}
		if ind == nil {
			return nil, "", apperr.New(apperr.NotFound, "Indikator rubrik tidak ditemukan")
		}
		if !domain.Finite(in.Value) || in.Value < 1 || in.Value > domain.RubricMax {
			return nil, "", apperr.Field("value", "Nilai rubrik harus 1-5")
		}
	} else {
		if ind, err = k.Store.GetIndicator(ctx, in.IndicatorID); err != nil {
			return nil, "", err
		}
		if ind == nil || !ind.IsActive {
			return nil, "", apperr.New(apperr.NotFound, msgIndicatorGone)
		}
		if ind.SourceKind != domain.SourceManual {
			return nil, "", apperr.New(apperr.Validation, "Indikator ini dihitung otomatis")
		}
		if !domain.Finite(in.Value) || in.Value < 0 {
			return nil, "", apperr.Field("value", "Nilai aktual harus angka ≥ 0")
		}
	}
	cur, err := k.Store.GetScorecardFor(ctx, e.ID, in.Period)
	if err != nil {
		return nil, "", err
	}
	if cur != nil && cur.Status == domain.ScorecardFinal {
		return nil, "", apperr.New(apperr.Conflict, "Scorecard sudah final, buka kembali dulu untuk mengubah nilai")
	}
	var target *float64
	if ind.Code == domain.CodeSupervisorRubric {
		m := domain.RubricMax
		target = &m
	} else {
		targets, err := k.Store.ListTargets(ctx)
		if err != nil {
			return nil, "", err
		}
		var rows []domain.KPITarget
		for _, t := range targets {
			if t.IndicatorID == ind.ID {
				rows = append(rows, t)
			}
		}
		target = domain.ResolveTarget(rows, domain.TargetContext{EmployeeID: e.ID, DepartmentID: e.DepartmentID,
			Month: in.Period.Month, Year: in.Period.Year}, ind.DefaultTarget)
	}
	value := in.Value
	notes := trimToNil(in.Notes)
	detail, _ := json.Marshal(map[string]any{"rated_by": a.UserID, "notes": notes, "at": k.now().Format(time.RFC3339)})
	if err := k.Store.UpsertSnapshot(ctx, domain.Snapshot{ID: newID(), EmployeeID: e.ID, IndicatorID: ind.ID, Period: in.Period,
		Actual: &value, Target: target, Attainment: domain.Attainment(ind.Direction, &value, target), SampleSize: 1, SourceDetail: detail}); err != nil {
		return nil, "", err
	}
	return k.recompose(ctx, e, in.Period)
}

func (k *KPI) Recommendations(ctx context.Context, employeeIDs []string) ([]domain.ScoreAverage, error) {
	a, err := k.actor(ctx)
	if err != nil {
		return nil, err
	}
	if !a.HRRead {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	if len(employeeIDs) > 200 {
		employeeIDs = employeeIDs[:200]
	}
	if len(employeeIDs) == 0 {
		return []domain.ScoreAverage{}, nil
	}
	return k.Store.RecentScoreAverages(ctx, employeeIDs)
}
