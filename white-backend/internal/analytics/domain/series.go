package domain

import (
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
)

// DailyPoint is one day of a filled series. A nil value means the source
// reported nothing that day; callers decide whether that stays null or
// becomes 0.
type DailyPoint struct {
	Date   string
	Values []*float64
}

func (p DailyPoint) Get(i int) *float64 {
	if i < 0 || i >= len(p.Values) {
		return nil
	}
	return p.Values[i]
}

// FillDaily mirrors lib/metrics/series.ts fillDaily: rows are summed per day
// and every day of the range gets a point. Missing days are 0 when
// missingZero is set, otherwise nil.
func FillDaily[T any](r dates.Range, rows []T, day func(T) time.Time, fields []func(T) float64, missingZero bool) []DailyPoint {
	sums := map[string][]float64{}
	for _, row := range rows {
		k := dates.Format(day(row))
		acc, ok := sums[k]
		if !ok {
			acc = make([]float64, len(fields))
			sums[k] = acc
		}
		for i, f := range fields {
			acc[i] += f(row)
		}
	}
	out := make([]DailyPoint, 0, r.Days())
	r.Each(func(d time.Time) {
		k := dates.Format(d)
		p := DailyPoint{Date: k, Values: make([]*float64, len(fields))}
		if acc, ok := sums[k]; ok {
			for i := range fields {
				v := acc[i]
				p.Values[i] = &v
			}
		} else if missingZero {
			for i := range fields {
				z := 0.0
				p.Values[i] = &z
			}
		}
		out = append(out, p)
	})
	return out
}

// Sparkline takes the last n values of one field; nil becomes 0 (as before).
func Sparkline(points []DailyPoint, field, n int) []float64 {
	if len(points) > n {
		points = points[len(points)-n:]
	}
	out := make([]float64, 0, len(points))
	for _, p := range points {
		if v := p.Get(field); v != nil {
			out = append(out, *v)
		} else {
			out = append(out, 0)
		}
	}
	return out
}

// AlignPrevious pairs the previous series by index; nil when it has no point there.
func AlignPrevious(current, previous []DailyPoint, field int) []*float64 {
	out := make([]*float64, len(current))
	for i := range current {
		if i < len(previous) {
			out[i] = previous[i].Get(field)
		}
	}
	return out
}

func F64(n int) float64 { return float64(n) }

// Int64Ptr converts an optional float back to an integer pointer.
func Int64Ptr(v *float64) *int64 {
	if v == nil {
		return nil
	}
	n := int64(*v)
	return &n
}
