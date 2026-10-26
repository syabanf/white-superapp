package domain

import (
	"fmt"
	"math"
	"time"
)

// IsoDate is the YYYY-MM-DD layout every day string in this service uses.
const IsoDate = "2006-01-02"

// Jakarta is WIB, the agency's zone: "today", digests and recaps follow it.
var Jakarta = mustZone("Asia/Jakarta")

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}

// Zone resolves an IANA name, falling back to Jakarta.
func Zone(name string) *time.Location {
	if name == "" {
		return Jakarta
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return Jakarta
	}
	return loc
}

// DayKey is t's calendar day in loc (nil is Jakarta) as YYYY-MM-DD.
func DayKey(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = Jakarta
	}
	return t.In(loc).Format(IsoDate)
}

// ParseDay parses YYYY-MM-DD as a UTC midnight.
func ParseDay(s string) (time.Time, bool) {
	t, err := time.Parse(IsoDate, s)
	return t, err == nil
}

// IsWorkday reports Monday to Friday.
func IsWorkday(day time.Time) bool {
	wd := day.Weekday()
	return wd != time.Saturday && wd != time.Sunday
}

// ReminderDue reports whether now has reached hour:minute of its day in
// loc. Together with the per-day sent log it makes a job that runs every
// 15 minutes fire once a day, on the first run after that time.
func ReminderDue(now time.Time, hour, minute int, loc *time.Location) bool {
	if loc == nil {
		loc = Jakarta
	}
	l := now.In(loc)
	return l.Hour() > hour || (l.Hour() == hour && l.Minute() >= minute)
}

// Period is a calendar month.
type Period struct {
	Year  int
	Month int
}

func (p Period) Valid() bool {
	return p.Year >= 2020 && p.Year <= 2100 && p.Month >= 1 && p.Month <= 12
}

// Key is YYYY-MM.
func (p Period) Key() string { return fmt.Sprintf("%04d-%02d", p.Year, p.Month) }

// Span is the first and last day of the month.
func (p Period) Span() DaySpan {
	first := time.Date(p.Year, time.Month(p.Month), 1, 0, 0, 0, 0, time.UTC)
	return DaySpan{first.Format(IsoDate), first.AddDate(0, 1, -1).Format(IsoDate)}
}

// Previous is the month before.
func (p Period) Previous() Period {
	t := time.Date(p.Year, time.Month(p.Month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	return Period{t.Year(), int(t.Month())}
}

// Next is the month after.
func (p Period) Next() Period {
	t := time.Date(p.Year, time.Month(p.Month), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	return Period{t.Year(), int(t.Month())}
}

// PeriodOf is the month of a day string (YYYY-MM-DD).
func PeriodOf(day string) Period {
	t, _ := ParseDay(day)
	return Period{t.Year(), int(t.Month())}
}

// ParsePeriod parses YYYY-MM.
func ParsePeriod(s string) (Period, bool) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return Period{}, false
	}
	return Period{t.Year(), int(t.Month())}, true
}

// Label is "Oktober 2026".
func (p Period) Label() string {
	return monthNames[p.Month-1] + " " + fmt.Sprint(p.Year)
}

var monthNames = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// DaySpan is an inclusive span of days as YYYY-MM-DD.
type DaySpan struct{ Start, End string }

// EachDate lists the days of [start, end].
func EachDate(start, end string) []string {
	s, ok1 := ParseDay(start)
	e, ok2 := ParseDay(end)
	if !ok1 || !ok2 {
		return nil
	}
	var out []string
	for d := s; !d.After(e); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format(IsoDate))
	}
	return out
}

// Round is JS Math.round: halves go toward +Infinity.
func Round(x float64) float64 { return math.Floor(x + 0.5) }

// Round2 rounds to two decimals.
func Round2(v float64) float64 { return Round(v*100) / 100 }
