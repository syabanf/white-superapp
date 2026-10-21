// Package metrics has the arithmetic shared by every dashboard: period
// deltas, safe division, weighted means and rounding.
package metrics

import (
	"math"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
)

type Direction string

const (
	Up   Direction = "up"
	Down Direction = "down"
	Flat Direction = "flat"
)

type Delta struct {
	Current  float64
	Previous float64
	Abs      float64
	// Pct is nil when the previous value is 0 (the UI hides the badge).
	Pct       *float64
	Direction Direction
}

// ComputeDelta mirrors lib/metrics/delta.ts.
func ComputeDelta(current, previous float64) Delta {
	abs := current - previous
	d := Delta{Current: current, Previous: previous, Abs: abs, Direction: Flat}
	if previous != 0 {
		pct := abs / math.Abs(previous) * 100
		d.Pct = &pct
	}
	switch {
	case math.Abs(abs) < 1e-9:
		d.Direction = Flat
	case abs > 0:
		d.Direction = Up
	default:
		d.Direction = Down
	}
	return d
}

// IsGoodChange is nil for flat; up is good unless lower is better.
func IsGoodChange(d Delta, lowerIsBetter bool) *bool {
	if d.Direction == Flat {
		return nil
	}
	good := d.Direction == Up
	if lowerIsBetter {
		good = !good
	}
	return &good
}

func (d Delta) Proto() *commonv1.Delta {
	p := &commonv1.Delta{Current: d.Current, Previous: d.Previous, Abs: d.Abs, Pct: d.Pct}
	switch d.Direction {
	case Up:
		p.Direction = commonv1.Delta_UP
	case Down:
		p.Direction = commonv1.Delta_DOWN
	default:
		p.Direction = commonv1.Delta_FLAT
	}
	return p
}

func SafeDiv(n, d float64) float64 {
	if d == 0 {
		return 0
	}
	return n / d
}

// WeightedMean of value/weight pairs; 0 when the weights sum to 0.
func WeightedMean(values, weights []float64) float64 {
	var num, den float64
	for i := range values {
		num += values[i] * weights[i]
		den += weights[i]
	}
	return SafeDiv(num, den)
}

func Mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func Clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// Round to d decimals, half away from zero like Math.round for positives.
func Round(v float64, d int) float64 {
	p := math.Pow(10, float64(d))
	return math.Round(v*p) / p
}

func Pct(n, d float64) float64 { return SafeDiv(n, d) * 100 }
