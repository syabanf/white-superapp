package domain

import (
	"math"
	"reflect"
	"testing"
)

// Ported from NüHabit payroll/domain/hris_test.go and kpi_collect_test.go.

func att(direction string, actual, target float64) *float64 {
	return Attainment(direction, &actual, &target)
}

func fp(v float64) *float64 { return &v }

func TestAttainment(t *testing.T) {
	cases := []struct {
		name    string
		got     *float64
		want    float64
		wantNil bool
	}{
		{"meets", att(HigherBetter, 100, 100), 1, false},
		{"half", att(HigherBetter, 50, 100), 0.5, false},
		{"cap", att(HigherBetter, 500, 100), AttainmentCap, false},
		{"negative clamps", att(HigherBetter, -5, 100), 0, false},
		{"zero target", att(HigherBetter, 10, 0), 0, true},
		{"lower meets", att(LowerBetter, 5, 5), 1, false},
		{"lower better", att(LowerBetter, 1, 10), AttainmentCap, false},
		{"lower worse", att(LowerBetter, 20, 10), 0.5, false},
		{"lower perfect", att(LowerBetter, 0, 10), AttainmentCap, false},
		{"lower zero target ok", att(LowerBetter, 0, 0), 1, false},
		{"lower zero target miss", att(LowerBetter, 2, 0), 0, false},
		{"boolean", att(Boolean, 1, 0), 1, false},
		{"boolean miss", att(Boolean, 0.5, 0), 0, false},
	}
	for _, c := range cases {
		if c.wantNil != (c.got == nil) || (c.got != nil && *c.got != c.want) {
			t.Errorf("%s = %v", c.name, c.got)
		}
	}
	if Attainment(HigherBetter, nil, nil) != nil {
		t.Fatal("no actual")
	}
}

func TestComposeScore(t *testing.T) {
	if s := ComposeScore([]ScoreComponent{{"a", 60, fp(1)}, {"b", 40, fp(1)}}); *s.Score != 100 || len(s.Excluded) != 0 {
		t.Fatal(s)
	}
	if s := ComposeScore([]ScoreComponent{{"a", 50, fp(0.8)}, {"b", 50, fp(0.6)}}); *s.Score != 70 {
		t.Fatal(*s.Score)
	}
	if s := ComposeScore([]ScoreComponent{{"a", 50, fp(1.2)}, {"b", 50, fp(0.9)}}); *s.Score != 100 || *s.RawScore != 105 {
		t.Fatal(*s.Score, *s.RawScore)
	}
	s := ComposeScore([]ScoreComponent{{"a", 50, fp(0.9)}, {"b", 25, nil}, {"c", 25, fp(0.6)}})
	if *s.Score != 80 || !reflect.DeepEqual(s.Excluded, []string{"b"}) || math.Abs(s.Breakdown[0].EffectiveWeight-66.67) > 0.01 {
		t.Fatal(s)
	}
	if s := ComposeScore([]ScoreComponent{{"a", 100, nil}}); s.Score != nil || s.Excluded[0] != "a" {
		t.Fatal("all excluded")
	}
	if ComposeScore(nil).Score != nil {
		t.Fatal("empty")
	}
	if s := ComposeScore([]ScoreComponent{{"a", 100, fp(0.5)}, {"b", 0, fp(1)}}); *s.Score != 50 || len(s.Breakdown) != 2 || s.Excluded[0] != "b" {
		t.Fatal(s)
	}
	if s := ComposeScore([]ScoreComponent{{"a", 33, fp(1)}, {"b", 33, fp(1)}, {"c", 34, fp(0.5)}}); *s.Score != 83 {
		t.Fatal(*s.Score)
	}
}

func TestResolveTarget(t *testing.T) {
	s := func(v string) *string { return &v }
	i := func(v int) *int { return &v }
	ctx := TargetContext{EmployeeID: "emp-1", DepartmentID: s("dept-1"), Month: 7, Year: 2026}
	val := func(p *float64) any {
		if p == nil {
			return nil
		}
		return *p
	}
	for name, c := range map[string]struct {
		rows []KPITarget
		ctx  TargetContext
		def  *float64
		want any
	}{
		"default":       {nil, ctx, fp(0.95), 0.95},
		"no default":    {nil, ctx, nil, nil},
		"employee wins": {[]KPITarget{{Target: 1}, {DepartmentID: s("dept-1"), Target: 3}, {EmployeeID: s("emp-1"), Target: 4}}, ctx, nil, 4.0},
		"period wins":   {[]KPITarget{{DepartmentID: s("dept-1"), Target: 2}, {DepartmentID: s("dept-1"), PeriodYear: i(2026), PeriodMonth: i(7), Target: 5}}, ctx, nil, 5.0},
		"others ignored": {[]KPITarget{{EmployeeID: s("x"), Target: 9}, {DepartmentID: s("y"), Target: 9},
			{PeriodYear: i(2026), PeriodMonth: i(6), Target: 9}}, ctx, fp(0.9), 0.9},
		"year only":     {[]KPITarget{{Target: 1}, {PeriodYear: i(2026), Target: 2}}, ctx, nil, 2.0},
		"month only":    {[]KPITarget{{Target: 1}, {PeriodMonth: i(7), Target: 6}}, ctx, nil, 6.0},
		"scope first":   {[]KPITarget{{PeriodYear: i(2026), PeriodMonth: i(7), Target: 1}, {EmployeeID: s("emp-1"), Target: 7}}, ctx, nil, 7.0},
		"no department": {[]KPITarget{{DepartmentID: s("dept-1"), Target: 3}}, TargetContext{EmployeeID: "e", Month: 7, Year: 2026}, nil, nil},
	} {
		if got := val(ResolveTarget(c.rows, c.ctx, c.def)); got != c.want {
			t.Errorf("%s: %v, want %v", name, got, c.want)
		}
	}
}

func TestSafeRatio(t *testing.T) {
	if SafeRatio(1, 0) != nil || SafeRatio(1, -1) != nil || SafeRatio(math.NaN(), 1) != nil || *SafeRatio(-3, 2) != 0 || *SafeRatio(3, 2) != 1.5 {
		t.Fatal("SafeRatio")
	}
}

func TestTaskCompletion(t *testing.T) {
	d := "d1"
	s := func(v string) *string { return &v }
	employees := []KPIEmployee{{ID: "a", DepartmentID: &d}, {ID: "b", DepartmentID: &d}, {ID: "c"}}
	got := TaskCompletion(employees, []TaskCount{
		{DepartmentID: d, AssigneeEmployeeID: s("a"), Status: "approved", Count: 3},
		{DepartmentID: d, AssigneeEmployeeID: s("a"), Status: "done", Count: 1},
		{DepartmentID: d, DoneBy: s("b"), Status: "approved", Count: 2},
		{DepartmentID: d, Status: "pending", Count: 2}, // nobody did it: due for every member
	})
	if *got["a"].Actual != 0.5 || got["a"].SampleSize != 6 || *got["b"].Actual != 0.5 || got["b"].SampleSize != 4 {
		t.Fatalf("%+v", got)
	}
	if _, ok := got["c"]; ok {
		t.Fatal("no tasks, no value")
	}
}

func TestTeamOntime(t *testing.T) {
	d := "d1"
	emps := []KPIEmployee{{ID: "a", DepartmentID: &d}, {ID: "b", DepartmentID: &d}, {ID: "c", DepartmentID: &d}}
	got := TeamOntime(emps, map[string]CollectorValue{"a": {Actual: fp(1)}, "b": {Actual: fp(0.5)}, "c": {}})
	if *got["c"].Actual != 0.75 || got["a"].SampleSize != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestAttendanceCollectors(t *testing.T) {
	v := AttendanceOntime(Attendance{WorkingDays: 22, LeaveDays: 2, PresentDays: 20, LateDays: 4})
	if *v.Actual != 0.8 || v.SampleSize != 20 || v.Detail["basis"] != "working_days" {
		t.Fatalf("%+v", v)
	}
	if v := AttendanceOntime(Attendance{PresentDays: 10, LateDays: 1}); *v.Actual != 0.9 || v.Detail["basis"] != "attendance" {
		t.Fatalf("%+v", v)
	}
	if v := AttendanceOntime(Attendance{}); v.Actual != nil {
		t.Fatal("no data")
	}
	late, ok := AttendanceLateRatio(Attendance{WorkingDays: 21, LeaveDays: 1, LateMinutes: 96})
	if !ok || math.Abs(*late.Actual-0.01) > 1e-9 {
		t.Fatalf("%+v", late)
	}
	if _, ok := AttendanceLateRatio(Attendance{WorkingDays: 2, LeaveDays: 2}); ok {
		t.Fatal("no scheduled days")
	}
	lc := LogbookCompliance(map[string]LogbookCount{"a": {Checked: 9, Total: 10}, "b": {}})
	if *lc["a"].Actual != 0.9 || lc["b"].Actual != nil {
		t.Fatalf("%+v", lc)
	}
}
