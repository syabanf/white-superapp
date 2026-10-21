package domain

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

// Timezone and calendar helpers ported from features/publishing/time.ts.
// Calendar days are `YYYY-MM-DD` strings in the client's timezone; instants
// are UTC time.Time values. Weekdays are Monday = 0.

var (
	locMu    sync.Mutex
	locCache = map[string]*time.Location{}
)

// Location resolves an IANA name, falling back to UTC for unknown zones so
// a bad client setting never breaks a page.
func Location(timeZone string) *time.Location {
	locMu.Lock()
	defer locMu.Unlock()
	if loc, ok := locCache[timeZone]; ok {
		return loc
	}
	loc, err := time.LoadLocation(timeZone)
	if err != nil || timeZone == "" {
		loc = time.UTC
	}
	locCache[timeZone] = loc
	return loc
}

type ZonedParts struct {
	Date    string
	Time    string
	Year    int
	Month   int
	Day     int
	Hour    int
	Minute  int
	Weekday int
}

// ToZoned breaks an instant into wall-clock parts in timeZone.
func ToZoned(t time.Time, timeZone string) ZonedParts {
	z := t.In(Location(timeZone))
	return ZonedParts{
		Date:    z.Format("2006-01-02"),
		Time:    z.Format("15:04"),
		Year:    z.Year(),
		Month:   int(z.Month()),
		Day:     z.Day(),
		Hour:    z.Hour(),
		Minute:  z.Minute(),
		Weekday: mondayIndex(z.Weekday()),
	}
}

func mondayIndex(wd time.Weekday) int { return (int(wd) + 6) % 7 }

// TzOffsetMinutes is the zone offset from UTC at the given instant.
func TzOffsetMinutes(t time.Time, timeZone string) int {
	_, off := t.In(Location(timeZone)).Zone()
	return off / 60
}

// FromZoned turns a wall-clock date and HH:mm in timeZone into a UTC instant.
func FromZoned(date, clock, timeZone string) (time.Time, error) {
	var y, m, d, hh, mm int
	if _, err := fmt.Sscanf(date, "%d-%d-%d", &y, &m, &d); err != nil {
		return time.Time{}, fmt.Errorf("tanggal tidak valid: %q", date)
	}
	if _, err := fmt.Sscanf(clock, "%d:%d", &hh, &mm); err != nil {
		if _, err2 := fmt.Sscanf(clock, "%d", &hh); err2 != nil {
			return time.Time{}, fmt.Errorf("jam tidak valid: %q", clock)
		}
	}
	return time.Date(y, time.Month(m), d, hh, mm, 0, 0, Location(timeZone)).UTC(), nil
}

var dateKeyRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// IsValidDateKey accepts strict calendar dates (2026-02-30 is rejected).
func IsValidDateKey(v string) bool {
	if !dateKeyRe.MatchString(v) {
		return false
	}
	t, err := time.Parse("2006-01-02", v)
	return err == nil && t.Format("2006-01-02") == v
}

func keyToUTC(key string) time.Time {
	var y, m, d int
	fmt.Sscanf(key, "%d-%d-%d", &y, &m, &d)
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func utcToKey(t time.Time) string { return t.UTC().Format("2006-01-02") }

func ShiftDays(key string, n int) string { return utcToKey(keyToUTC(key).AddDate(0, 0, n)) }

// ShiftMonths clamps to the last day of the target month (Jan 31 + 1 = Feb 28).
func ShiftMonths(key string, n int) string {
	d := keyToUTC(key)
	first := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, n, 0)
	last := first.AddDate(0, 1, -1).Day()
	day := d.Day()
	if day > last {
		day = last
	}
	return utcToKey(time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, time.UTC))
}

// StartOfWeek is the Monday of the week containing key.
func StartOfWeek(key string) string {
	return ShiftDays(key, -mondayIndex(keyToUTC(key).Weekday()))
}

func WeekDays(key string) []string {
	start := StartOfWeek(key)
	out := make([]string, 7)
	for i := range out {
		out[i] = ShiftDays(start, i)
	}
	return out
}

func MonthOf(key string) string { return key[:7] }

// MonthGrid is 6 rows x 7 days, Monday first, covering the month of key.
func MonthGrid(key string) [][]string {
	cursor := StartOfWeek(MonthOf(key) + "-01")
	rows := make([][]string, 6)
	for r := range rows {
		row := make([]string, 7)
		for c := range row {
			row[c] = cursor
			cursor = ShiftDays(cursor, 1)
		}
		rows[r] = row
	}
	return rows
}

// RangeForDays returns the UTC bounds [from, to) of consecutive client-tz
// calendar days.
func RangeForDays(days []string, timeZone string) (from, to time.Time, err error) {
	if len(days) == 0 {
		return from, to, fmt.Errorf("rentang hari kosong")
	}
	from, err = FromZoned(days[0], "00:00", timeZone)
	if err != nil {
		return
	}
	to, err = FromZoned(ShiftDays(days[len(days)-1], 1), "00:00", timeZone)
	return
}

// DaysBetween lists the calendar days from..to inclusive.
func DaysBetween(from, to string) []string {
	var out []string
	for d := from; d <= to; d = ShiftDays(d, 1) {
		out = append(out, d)
	}
	return out
}

type Slot struct {
	Day  int
	Hour int
}

// NextOccurrences finds, per slot, the first instant at or after from+lead
// that falls on that weekday and hour in timeZone, looking 8 days ahead.
func NextOccurrences(slots []Slot, timeZone string, from time.Time, lead time.Duration) []time.Time {
	min := from.Add(lead)
	today := ToZoned(min, timeZone)
	out := make([]time.Time, 0, len(slots))
	for _, s := range slots {
		clock := fmt.Sprintf("%02d:00", s.Hour)
		found := false
		for i := 0; i < 8; i++ {
			key := ShiftDays(today.Date, i)
			if mondayIndex(keyToUTC(key).Weekday()) != s.Day {
				continue
			}
			at, _ := FromZoned(key, clock, timeZone)
			if !at.Before(min) {
				out = append(out, at)
				found = true
				break
			}
		}
		if !found {
			at, _ := FromZoned(ShiftDays(today.Date, 7), clock, timeZone)
			out = append(out, at)
		}
	}
	return out
}

// MoveToDay moves an instant to another calendar day, keeping its wall-clock time.
func MoveToDay(instant time.Time, dayKey, timeZone string) time.Time {
	z := ToZoned(instant, timeZone)
	at, _ := FromZoned(dayKey, z.Time, timeZone)
	return at
}

// TzLabel is the short zone label: WIB, WITA, WIT or the zone abbreviation.
func TzLabel(timeZone string, at time.Time) string {
	switch timeZone {
	case "Asia/Jakarta":
		return "WIB"
	case "Asia/Makassar":
		return "WITA"
	case "Asia/Jayapura":
		return "WIT"
	}
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		return timeZone
	}
	name, _ := at.In(loc).Zone()
	if name == "" {
		return timeZone
	}
	return name
}

// Clock renders the wall-clock time as "14.05" (Indonesian separator).
func Clock(t time.Time, timeZone string) string {
	z := ToZoned(t, timeZone)
	return fmt.Sprintf("%02d.%02d", z.Hour, z.Minute)
}
