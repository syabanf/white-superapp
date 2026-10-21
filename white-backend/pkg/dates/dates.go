// Package dates holds the calendar rules of the platform. Every metric day is
// a UTC date; presets end on the latest complete day (yesterday UTC); the
// comparison period has the same length and ends the day before the range.
package dates

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
)

const (
	DefaultPreset = "28d"
	Layout        = "2006-01-02"
)

var presets = map[string]bool{"7d": true, "14d": true, "28d": true, "30d": true, "90d": true, "thisMonth": true, "lastMonth": true, "12m": true}

func IsPreset(p string) bool { return presets[p] }

// Day truncates to UTC midnight.
func Day(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func UTC(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func AddDays(t time.Time, n int) time.Time { return t.AddDate(0, 0, n) }

func Today(now time.Time) time.Time { return Day(now) }

// LatestCompleteDay is yesterday in UTC: the last day providers report fully.
func LatestCompleteDay(now time.Time) time.Time { return AddDays(Day(now), -1) }

func Format(t time.Time) string { return t.UTC().Format(Layout) }

var isoRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// Parse accepts strict YYYY-MM-DD and normalises overflow the way Date.UTC
// did (2026-02-31 becomes 2026-03-03).
func Parse(s string) (time.Time, error) {
	if !isoRe.MatchString(s) {
		return time.Time{}, fmt.Errorf("tanggal tidak valid: %q", s)
	}
	var y, m, d int
	if _, err := fmt.Sscanf(s, "%d-%d-%d", &y, &m, &d); err != nil {
		return time.Time{}, err
	}
	return UTC(y, time.Month(m), d), nil
}

// Range is an inclusive day span.
type Range struct {
	From time.Time
	To   time.Time
}

func (r Range) Days() int { return int(r.To.Sub(r.From).Hours()/24) + 1 }

func (r Range) Contains(t time.Time) bool {
	d := Day(t)
	return !d.Before(r.From) && !d.After(r.To)
}

// Each calls fn for every day in the range, in order.
func (r Range) Each(fn func(day time.Time)) {
	for d := r.From; !d.After(r.To); d = AddDays(d, 1) {
		fn(d)
	}
}

func (r Range) Proto() *commonv1.DateRange {
	return &commonv1.DateRange{From: Format(r.From), To: Format(r.To)}
}

// Previous is the same number of days ending right before r.
func Previous(r Range) Range {
	n := r.Days()
	to := AddDays(r.From, -1)
	return Range{From: AddDays(to, -(n - 1)), To: to}
}

func ForPreset(preset string, now time.Time) Range {
	end := LatestCompleteDay(now)
	today := Day(now)
	switch preset {
	case "7d":
		return Range{AddDays(end, -6), end}
	case "14d":
		return Range{AddDays(end, -13), end}
	case "30d":
		return Range{AddDays(end, -29), end}
	case "90d":
		return Range{AddDays(end, -89), end}
	case "thisMonth":
		from := UTC(today.Year(), today.Month(), 1)
		to := end
		if end.Before(from) {
			to = today
		}
		return Range{from, to}
	case "lastMonth":
		first := UTC(today.Year(), today.Month(), 1)
		from := first.AddDate(0, -1, 0)
		return Range{from, AddDays(first, -1)}
	case "12m":
		return Range{AddDays(end.AddDate(-1, 0, 0), 1), end}
	default: // 28d
		return Range{AddDays(end, -27), end}
	}
}

// Resolved is what every analytics query works with.
type Resolved struct {
	Current  Range
	Previous Range
	// Empty when a custom from/to was used.
	Preset  string
	Compare bool
}

func (r Resolved) Proto() *commonv1.ResolvedRange {
	return &commonv1.ResolvedRange{
		Current:  r.Current.Proto(),
		Previous: r.Previous.Proto(),
		Days:     int32(r.Current.Days()),
		Preset:   r.Preset,
		Compare:  r.Compare,
	}
}

// Resolve mirrors lib/dates.ts resolveRange: a valid custom window wins,
// otherwise the preset (falling back to 28d). Compare defaults to true.
func Resolve(q *commonv1.RangeQuery, now time.Time) Resolved {
	compare := true
	var from, to time.Time
	var fromErr, toErr error
	preset := DefaultPreset
	if q != nil {
		compare = q.Compare
		from, fromErr = Parse(q.From)
		to, toErr = Parse(q.To)
		if IsPreset(q.Preset) {
			preset = q.Preset
		}
	}
	if q != nil && fromErr == nil && toErr == nil && !from.After(to) {
		r := Range{from, to}
		return Resolved{Current: r, Previous: Previous(r), Compare: compare}
	}
	r := ForPreset(preset, now)
	return Resolved{Current: r, Previous: Previous(r), Preset: preset, Compare: compare}
}

// FromProto validates an explicit DateRange from a request.
func FromProto(dr *commonv1.DateRange) (Range, error) {
	if dr == nil {
		return Range{}, errors.New("rentang tanggal wajib diisi")
	}
	from, err := Parse(dr.From)
	if err != nil {
		return Range{}, err
	}
	to, err := Parse(dr.To)
	if err != nil {
		return Range{}, err
	}
	if from.After(to) {
		return Range{}, errors.New("tanggal mulai setelah tanggal akhir")
	}
	return Range{from, to}, nil
}

// MondayWeekKey is the ISO date of the Monday that starts t's week.
func MondayWeekKey(t time.Time) string {
	d := Day(t)
	wd := int(d.Weekday()+6) % 7
	return Format(AddDays(d, -wd))
}
