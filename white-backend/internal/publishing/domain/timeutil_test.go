package domain

import (
	"testing"
	"time"
)

func mustZoned(t *testing.T, date, clock, tz string) time.Time {
	t.Helper()
	at, err := FromZoned(date, clock, tz)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

func TestJakartaRoundTrip(t *testing.T) {
	at := mustZoned(t, "2026-09-02", "14:30", "Asia/Jakarta")
	if at.Format(time.RFC3339) != "2026-09-02T07:30:00Z" {
		t.Fatal(at)
	}
	z := ToZoned(at, "Asia/Jakarta")
	if z.Date != "2026-09-02" || z.Time != "14:30" || z.Weekday != 2 {
		t.Fatalf("%+v", z)
	}
	if TzOffsetMinutes(at, "Asia/Jakarta") != 420 || TzOffsetMinutes(at, "Asia/Makassar") != 480 || TzOffsetMinutes(at, "Asia/Jayapura") != 540 {
		t.Fatal("offsets")
	}
	if Clock(at, "Asia/Jakarta") != "14.30" {
		t.Fatal(Clock(at, "Asia/Jakarta"))
	}
	if TzLabel("Asia/Jakarta", at) != "WIB" || TzLabel("Asia/Makassar", at) != "WITA" || TzLabel("Asia/Jayapura", at) != "WIT" || TzLabel("Nowhere/Nope", at) != "Nowhere/Nope" {
		t.Fatal("labels")
	}
}

func TestDSTZone(t *testing.T) {
	if mustZoned(t, "2026-03-29", "03:00", "Europe/Berlin").Format(time.RFC3339) != "2026-03-29T01:00:00Z" {
		t.Fatal("summer")
	}
	if mustZoned(t, "2026-01-15", "12:00", "Europe/Berlin").Format(time.RFC3339) != "2026-01-15T11:00:00Z" {
		t.Fatal("winter")
	}
}

func TestMoveToDayAndNextOccurrences(t *testing.T) {
	at := mustZoned(t, "2026-09-02", "19:00", "Asia/Jakarta")
	z := ToZoned(MoveToDay(at, "2026-09-10", "Asia/Jakarta"), "Asia/Jakarta")
	if z.Date != "2026-09-10" || z.Time != "19:00" {
		t.Fatalf("%+v", z)
	}
	// now is Wed 2026-09-02 10:00 WIB.
	got := NextOccurrences([]Slot{{2, 10}, {2, 11}, {0, 9}}, "Asia/Jakarta", now, 5*time.Minute)
	if ToZoned(got[0], "Asia/Jakarta").Date != "2026-09-09" {
		t.Fatal("this week's slot already passed", got[0])
	}
	if w := ToZoned(got[1], "Asia/Jakarta"); w.Date != "2026-09-02" || w.Time != "11:00" {
		t.Fatalf("%+v", w)
	}
	if ToZoned(got[2], "Asia/Jakarta").Date != "2026-09-07" {
		t.Fatal(got[2])
	}
}

func TestCalendarGrid(t *testing.T) {
	grid := MonthGrid("2026-09-15")
	if len(grid) != 6 || grid[0][0] != "2026-08-31" || grid[0][1] != "2026-09-01" || grid[5][6] != "2026-10-11" {
		t.Fatalf("%v", grid)
	}
	if StartOfWeek("2026-09-02") != "2026-08-31" || WeekDays("2026-09-02")[6] != "2026-09-06" {
		t.Fatal("week")
	}
	if ShiftMonths("2026-01-31", 1) != "2026-02-28" || ShiftMonths("2026-03-15", -1) != "2026-02-15" {
		t.Fatal("months")
	}
	if IsValidDateKey("2026-02-30") || !IsValidDateKey("2026-09-02") || IsValidDateKey("x") {
		t.Fatal("date key")
	}
	if days := DaysBetween("2026-09-01", "2026-09-03"); len(days) != 3 || days[2] != "2026-09-03" {
		t.Fatal(days)
	}
}

func TestRangeForDays(t *testing.T) {
	for _, c := range []struct{ tz, from, to string }{
		{"Asia/Jakarta", "2026-08-31T17:00:00Z", "2026-09-02T17:00:00Z"},
		{"Asia/Makassar", "2026-08-31T16:00:00Z", "2026-09-02T16:00:00Z"},
		{"Asia/Jayapura", "2026-08-31T15:00:00Z", "2026-09-02T15:00:00Z"},
	} {
		from, to, err := RangeForDays([]string{"2026-09-01", "2026-09-02"}, c.tz)
		if err != nil {
			t.Fatal(err)
		}
		if from.Format(time.RFC3339) != c.from || to.Format(time.RFC3339) != c.to {
			t.Fatalf("%s: %s %s", c.tz, from, to)
		}
	}
	if _, _, err := RangeForDays(nil, "Asia/Jakarta"); err == nil {
		t.Fatal("empty range accepted")
	}
}
