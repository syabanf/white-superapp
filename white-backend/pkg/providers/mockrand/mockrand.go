// Package mockrand is the deterministic random stream every demo-mode
// adapter uses, ported from the FNV-1a + mulberry32 pair in the Next.js
// mocks so seeded demo data stays stable across both backends.
package mockrand

import "math"

// Seed hashes a string with FNV-1a (32-bit).
func Seed(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

// Source is a mulberry32 generator.
type Source struct{ a uint32 }

func New(seed string) *Source { return &Source{a: Seed(seed)} }

func FromSeed(seed uint32) *Source { return &Source{a: seed} }

// Float64 returns the next value in [0, 1).
func (s *Source) Float64() float64 {
	s.a += 0x6D2B79F5
	t := s.a
	t = (t ^ (t >> 15)) * (t | 1)
	t ^= t + (t^(t>>7))*(t|61)
	return float64(t^(t>>14)) / 4294967296
}

// Between returns a value in [lo, hi).
func (s *Source) Between(lo, hi float64) float64 { return lo + s.Float64()*(hi-lo) }

// IntN returns an int in [0, n).
func (s *Source) IntN(n int) int {
	if n <= 0 {
		return 0
	}
	return int(math.Floor(s.Float64() * float64(n)))
}

// Pick returns one element of xs.
func Pick[T any](s *Source, xs []T) T { return xs[s.IntN(len(xs))] }
