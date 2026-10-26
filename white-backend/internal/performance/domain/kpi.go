package domain

import (
	"math"
	"sort"
)

// KPI engine ported from NüHabit payroll/domain/kpi.go and kpi_collect.go
// (lib/kpi/*): attainment capped at 120%, weighted 0-100 scorecards with
// excluded weights redistributed, target resolution and the collectors
// that make sense for an agency (attendance from hr, logbook, tasks).

// Indicator directions and source kinds.
const (
	HigherBetter = "higher_better"
	LowerBetter  = "lower_better"
	Boolean      = "boolean"

	SourceAuto   = "auto"
	SourceManual = "manual"

	ScorecardDraft = "draft"
	ScorecardFinal = "final"
)

// Codes of indicators with an automatic collector, and the rubric.
const (
	CodeAttOntime         = "att_ontime"
	CodeAttLateRatio      = "att_late_ratio"
	CodeTeamOntime        = "team_ontime"
	CodeLogbookCompliance = "logbook_compliance"
	CodeTaskCompletion    = "task_completion"
	CodeSupervisorRubric  = "supervisor_rubric"
)

// RubricMax is the top of the supervisor rubric (1-5).
const RubricMax = 5.0

// AttainmentCap caps over-achievement at 120%.
const AttainmentCap = 1.2

// Finite reports Number.isFinite.
func Finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// ValidDirection reports a known indicator direction.
func ValidDirection(d string) bool { return d == HigherBetter || d == LowerBetter || d == Boolean }

// Attainment is computeAttainment: nil excludes the indicator from the score.
func Attainment(direction string, actual, target *float64) *float64 {
	if actual == nil || !Finite(*actual) {
		return nil
	}
	v := func(x float64) *float64 { return &x }
	clamp := func(x float64) *float64 { return v(math.Min(AttainmentCap, math.Max(0, x))) }
	if direction == Boolean {
		if *actual >= 1 {
			return v(1)
		}
		return v(0)
	}
	if target == nil || !Finite(*target) {
		return nil
	}
	if direction == HigherBetter {
		if *target <= 0 {
			return nil
		}
		return clamp(*actual / *target)
	}
	switch {
	case *target < 0:
		return nil
	case *target == 0:
		if *actual <= 0 {
			return v(1)
		}
		return v(0)
	case *actual <= 0:
		return v(AttainmentCap)
	}
	return clamp(*target / *actual)
}

// ScoreComponent is one weighted indicator of a scorecard.
type ScoreComponent struct {
	Code       string
	Weight     float64
	Attainment *float64
}

// BreakdownRow is stored in kpi_scorecards.breakdown.
type BreakdownRow struct {
	Code            string   `json:"code"`
	Weight          float64  `json:"weight"`
	EffectiveWeight float64  `json:"effectiveWeight"`
	Attainment      *float64 `json:"attainment"`
	Contribution    float64  `json:"contribution"`
}

// Score is composeScore's result.
type Score struct {
	Score      *float64
	RawScore   *float64
	UsedWeight float64
	Excluded   []string
	Breakdown  []BreakdownRow
}

// ComposeScore is composeScore: Σ weight × attainment on a 0-100 scale,
// weights of excluded components redistributed proportionally, capped at
// 100 with the raw score kept.
func ComposeScore(components []ScoreComponent) Score {
	included := func(c ScoreComponent) bool { return c.Attainment != nil && Finite(*c.Attainment) && c.Weight > 0 }
	s := Score{Excluded: []string{}, Breakdown: []BreakdownRow{}}
	for _, c := range components {
		if included(c) {
			s.UsedWeight += c.Weight
		} else {
			s.Excluded = append(s.Excluded, c.Code)
		}
	}
	if s.UsedWeight <= 0 {
		s.UsedWeight = 0
		for _, c := range components {
			s.Breakdown = append(s.Breakdown, BreakdownRow{Code: c.Code, Weight: c.Weight, Attainment: c.Attainment})
		}
		return s
	}
	scale := 100 / s.UsedWeight
	raw := 0.0
	for _, c := range components {
		eff, contrib := 0.0, 0.0
		if included(c) {
			eff = c.Weight * scale
			contrib = eff * *c.Attainment
		}
		raw += contrib
		s.Breakdown = append(s.Breakdown, BreakdownRow{Code: c.Code, Weight: c.Weight,
			EffectiveWeight: Round2(eff), Attainment: c.Attainment, Contribution: Round2(contrib)})
	}
	score, rawScore := Round2(math.Min(100, raw)), Round2(raw)
	s.Score, s.RawScore, s.UsedWeight = &score, &rawScore, Round2(s.UsedWeight)
	return s
}

// CollectorValue is one collected measurement: Actual nil excludes the
// indicator; Detail is stored as source_detail for audit.
type CollectorValue struct {
	Actual     *float64
	SampleSize float64
	Detail     map[string]any
}

// SafeRatio is safeRatio: nil when the denominator is not positive, a
// negative numerator clamps to 0, ratios may exceed 1.
func SafeRatio(num, den float64) *float64 {
	if !Finite(num) || !Finite(den) || den <= 0 {
		return nil
	}
	v := math.Max(0, num) / den
	return &v
}

// KPIEmployee is an active employee as the snapshot sees it.
type KPIEmployee struct {
	ID           string
	DepartmentID *string
}

// Attendance is the hr monthly attendance summary of one employee.
type Attendance struct {
	EmployeeID    string
	WorkingDays   int
	PresentDays   int
	LateDays      int
	AbsentDays    int
	LeaveDays     int
	LateMinutes   int
	OvertimeHours float64
}

// ScheduledMinutesPerDay is the working time the late ratio divides by:
// hr reports totals, not shifts, so an 8 hour day is assumed.
const ScheduledMinutesPerDay = 8 * 60

// AttendanceOntime is att_ontime from the hr summary: on-time days over
// scheduled days minus leave; without working days, over days present
// (computeOntime's two bases).
func AttendanceOntime(a Attendance) CollectorValue {
	ontime := max(0, a.PresentDays-a.LateDays)
	den, basis := a.WorkingDays-a.LeaveDays, "working_days"
	if a.WorkingDays <= 0 {
		den, basis = a.PresentDays, "attendance"
	}
	return CollectorValue{Actual: SafeRatio(float64(ontime), float64(den)), SampleSize: float64(max(0, den)),
		Detail: map[string]any{"basis": basis, "workingDays": a.WorkingDays, "leaveDays": a.LeaveDays,
			"presentDays": a.PresentDays, "ontimeDays": ontime}}
}

// AttendanceLateRatio is att_late_ratio: late minutes over the scheduled
// minutes of the days that were due (working days minus leave).
func AttendanceLateRatio(a Attendance) (CollectorValue, bool) {
	days := a.WorkingDays - a.LeaveDays
	if days <= 0 {
		return CollectorValue{}, false
	}
	minutes := days * ScheduledMinutesPerDay
	return CollectorValue{Actual: SafeRatio(float64(a.LateMinutes), float64(minutes)), SampleSize: float64(days),
		Detail: map[string]any{"late_minutes": a.LateMinutes, "scheduled_minutes": minutes, "scheduled_days": days}}, true
}

// TeamOntime is collectTeamOntime: each department's average on-time ratio
// over the members that have one, shared with every member.
func TeamOntime(employees []KPIEmployee, ontime map[string]CollectorValue) map[string]CollectorValue {
	type sum struct {
		total float64
		n     int
	}
	sums := map[string]*sum{}
	for _, e := range employees {
		v, ok := ontime[e.ID]
		if e.DepartmentID == nil || !ok || v.Actual == nil {
			continue
		}
		s := sums[*e.DepartmentID]
		if s == nil {
			s = &sum{}
			sums[*e.DepartmentID] = s
		}
		s.total += *v.Actual
		s.n++
	}
	out := map[string]CollectorValue{}
	for _, e := range employees {
		if e.DepartmentID == nil || sums[*e.DepartmentID] == nil {
			continue
		}
		s := sums[*e.DepartmentID]
		avg := s.total / float64(s.n)
		out[e.ID] = CollectorValue{Actual: &avg, SampleSize: float64(s.n),
			Detail: map[string]any{"department_id": *e.DepartmentID, "members_with_data": s.n}}
	}
	return out
}

// LogbookCount is an employee's checked and total checklist items.
type LogbookCount struct{ Checked, Total int }

// LogbookCompliance is logbook_compliance per employee.
func LogbookCompliance(counts map[string]LogbookCount) map[string]CollectorValue {
	out := map[string]CollectorValue{}
	for id, c := range counts {
		out[id] = CollectorValue{Actual: SafeRatio(float64(c.Checked), float64(c.Total)), SampleSize: float64(c.Total),
			Detail: map[string]any{"checked_items": c.Checked, "total_items": c.Total}}
	}
	return out
}

// TaskCount is one group of department task occurrences.
type TaskCount struct {
	DepartmentID       string
	AssigneeEmployeeID *string
	DoneBy             *string
	Status             string
	Count              int
}

// TaskCompletion is collectTaskCompletion: approved over due occurrences
// per employee. An occurrence counts for its assignee, else whoever did it;
// one nobody did is due for every member of the department.
func TaskCompletion(employees []KPIEmployee, counts []TaskCount) map[string]CollectorValue {
	out := map[string]CollectorValue{}
	if len(counts) == 0 {
		return out
	}
	type agg struct{ approved, due int }
	per := map[string]*agg{}
	bump := func(id string, approved, due int) {
		a := per[id]
		if a == nil {
			a = &agg{}
			per[id] = a
		}
		a.approved += approved
		a.due += due
	}
	members := map[string][]string{}
	for _, e := range employees {
		if e.DepartmentID != nil {
			members[*e.DepartmentID] = append(members[*e.DepartmentID], e.ID)
		}
	}
	for _, c := range counts {
		approved := 0
		if c.Status == StatusApproved {
			approved = c.Count
		}
		owner := c.AssigneeEmployeeID
		if owner == nil {
			owner = c.DoneBy
		}
		if owner != nil {
			bump(*owner, approved, c.Count)
			continue
		}
		for _, id := range members[c.DepartmentID] {
			bump(id, 0, c.Count)
		}
	}
	for _, e := range employees {
		a := per[e.ID]
		if a == nil || a.due == 0 {
			continue
		}
		v := float64(a.approved) / float64(a.due)
		out[e.ID] = CollectorValue{Actual: &v, SampleSize: float64(a.due),
			Detail: map[string]any{"approved": a.approved, "due": a.due}}
	}
	return out
}

// KPITarget is a kpi_targets row; nil scopes match anything.
type KPITarget struct {
	ID                       string
	IndicatorID              string
	PeriodYear, PeriodMonth  *int
	DepartmentID, EmployeeID *string
	Target                   float64
}

// TargetContext is who and when a target is resolved for.
type TargetContext struct {
	EmployeeID   string
	DepartmentID *string
	Month, Year  int
}

func (t KPITarget) scopeRank() int {
	switch {
	case t.EmployeeID != nil:
		return 8
	case t.DepartmentID != nil:
		return 4
	}
	return 1
}

func (t KPITarget) periodRank() int {
	r := 0
	if t.PeriodMonth != nil {
		r += 2
	}
	if t.PeriodYear != nil {
		r++
	}
	return r
}

func (t KPITarget) matches(c TargetContext) bool {
	switch {
	case t.EmployeeID != nil && *t.EmployeeID != c.EmployeeID,
		t.DepartmentID != nil && (c.DepartmentID == nil || *t.DepartmentID != *c.DepartmentID),
		t.PeriodYear != nil && *t.PeriodYear != c.Year,
		t.PeriodMonth != nil && *t.PeriodMonth != c.Month:
		return false
	}
	return true
}

// ResolveTarget is resolveTarget: the most specific matching row (scope
// employee > department > generic, then exact month > year only > any
// period), else the indicator's default.
func ResolveTarget(rows []KPITarget, c TargetContext, def *float64) *float64 {
	var candidates []KPITarget
	for _, r := range rows {
		if r.matches(c) {
			candidates = append(candidates, r)
		}
	}
	if len(candidates) == 0 {
		return def
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.scopeRank() != b.scopeRank() {
			return a.scopeRank() > b.scopeRank()
		}
		return a.periodRank() > b.periodRank()
	})
	return &candidates[0].Target
}
