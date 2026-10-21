package dates

import (
	"testing"
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
)

var now = time.Date(2026, 9, 19, 10, 30, 0, 0, time.UTC)

func TestPresetsEndYesterday(t *testing.T) {
	r := ForPreset("28d", now)
	if Format(r.To) != "2026-09-18" || Format(r.From) != "2026-08-22" || r.Days() != 28 {
		t.Fatalf("28d = %s..%s (%d)", Format(r.From), Format(r.To), r.Days())
	}
	m := ForPreset("12m", now)
	if Format(m.From) != "2025-09-19" {
		t.Fatalf("12m from = %s", Format(m.From))
	}
	lm := ForPreset("lastMonth", now)
	if Format(lm.From) != "2026-08-01" || Format(lm.To) != "2026-08-31" {
		t.Fatalf("lastMonth = %s..%s", Format(lm.From), Format(lm.To))
	}
	first := ForPreset("thisMonth", time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC))
	if Format(first.From) != "2026-10-01" || Format(first.To) != "2026-10-01" {
		t.Fatalf("thisMonth on the 1st = %s..%s", Format(first.From), Format(first.To))
	}
}

func TestPreviousKeepsLength(t *testing.T) {
	r := Range{UTC(2026, 9, 1), UTC(2026, 9, 10)}
	p := Previous(r)
	if p.Days() != 10 || Format(p.To) != "2026-08-31" || Format(p.From) != "2026-08-22" {
		t.Fatalf("prev = %s..%s", Format(p.From), Format(p.To))
	}
}

func TestResolveCustomWins(t *testing.T) {
	res := Resolve(&commonv1.RangeQuery{From: "2026-09-01", To: "2026-09-05", Preset: "7d", Compare: false}, now)
	if res.Preset != "" || res.Current.Days() != 5 || res.Compare {
		t.Fatalf("%+v", res)
	}
	bad := Resolve(&commonv1.RangeQuery{From: "2026-09-10", To: "2026-09-01", Preset: "bogus", Compare: true}, now)
	if bad.Preset != "28d" || !bad.Compare {
		t.Fatalf("%+v", bad)
	}
	if d := Resolve(nil, now); d.Preset != "28d" || !d.Compare {
		t.Fatalf("%+v", d)
	}
}

func TestParseOverflow(t *testing.T) {
	d, err := Parse("2026-02-31")
	if err != nil || Format(d) != "2026-03-03" {
		t.Fatalf("%v %v", d, err)
	}
	if _, err := Parse("2026-2-3"); err == nil {
		t.Fatal("loose format accepted")
	}
}

func TestMondayWeekKey(t *testing.T) {
	if k := MondayWeekKey(UTC(2026, 9, 20)); k != "2026-09-14" { // Sunday
		t.Fatal(k)
	}
}
