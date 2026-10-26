package domain

import (
	"testing"
	"time"
)

func TestResolveDashPeriod(t *testing.T) {
	now := time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC) // 10:00 WIB
	day := func(t time.Time) string { return t.Format("2006-01-02") }

	p, bad := ResolveDashPeriod("", "", "", now)
	eq(t, bad, "")
	eq(t, []string{day(p.From), day(p.To), p.Compare, day(p.CompareFrom), day(p.CompareTo)},
		[]string{"2026-10-01", "2026-10-26", "previous", "2026-09-05", "2026-09-30"})
	eq(t, len(p.Months), 12)
	eq(t, []string{p.Months[0], p.Months[11], p.CompareMonths[0], p.CompareMonths[11]}, []string{"2025-11", "2026-10", "2024-11", "2025-10"})

	p, _ = ResolveDashPeriod("2026-01-01", "2026-03-31", "year", now)
	eq(t, []string{day(p.CompareFrom), day(p.CompareTo)}, []string{"2025-01-01", "2025-03-31"})
	eq(t, []string{p.Months[0], p.Months[11], p.CompareMonths[0]}, []string{"2025-04", "2026-03", "2024-04"})

	p, _ = ResolveDashPeriod("2024-10-01", "2026-09-30", "previous", now)
	eq(t, len(p.Months), 24)
	eq(t, p.CompareMonths[23], "2024-09")

	from, to := p.Bounds()
	eq(t, to.Sub(from), time.Duration(730)*24*time.Hour)

	for _, in := range [][2]string{{"2026-13-01", ""}, {"2026-10-05", "2026-10-01"}, {"2020-01-01", "2026-01-01"}} {
		if _, bad := ResolveDashPeriod(in[0], in[1], "", now); bad == "" {
			t.Fatalf("%v accepted", in)
		}
	}
}

func TestDashTargets(t *testing.T) {
	rows := []DashTarget{
		{UserID: "", PeriodMonth: "2026-09", TargetValue: 300},
		{UserID: "a", PeriodMonth: "2026-09", TargetValue: 100},
		{UserID: "a", PeriodMonth: "2026-10", TargetValue: 310},
		{UserID: "b", PeriodMonth: "2026-10", TargetValue: 62},
	}
	eq(t, MonthTarget(rows, "2026-09", ""), 300.0) // company wins
	eq(t, MonthTarget(rows, "2026-10", ""), 372.0) // no company: sum of people
	eq(t, MonthTarget(rows, "2026-10", "a"), 310.0)
	eq(t, MonthTarget(rows, "2026-09", "b"), 0.0)

	d := func(s string) time.Time { v, _ := time.ParseInLocation("2006-01-02", s, Jakarta); return v }
	eq(t, ProratedTarget(rows, d("2026-10-01"), d("2026-10-10"), "a"), 100.0)
	eq(t, ProratedTarget(rows, d("2026-09-16"), d("2026-10-31"), "a"), 50.0+310.0)
}

func TestDashBuckets(t *testing.T) {
	eq(t, []string{StageAgeBucket(0), StageAgeBucket(7), StageAgeBucket(8), StageAgeBucket(30), StageAgeBucket(31)}, []string{"0-7", "0-7", "8-14", "15-30", "31+"})
	eq(t, []string{ReceivableBucket(-3), ReceivableBucket(0), ReceivableBucket(1), ReceivableBucket(61), ReceivableBucket(91)}, []string{"current", "current", "1-30", "61-90", "90+"})
	eq(t, Pct(1, 4), 25.0)
	eq(t, Pct(1, 0), 0.0)
	eq(t, len(MonthEnds([]string{"2026-09", "2026-10"}, time.Date(2026, 10, 26, 0, 0, 0, 0, Jakarta))), 2)
}
