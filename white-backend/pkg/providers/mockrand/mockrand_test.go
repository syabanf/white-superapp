package mockrand

import "testing"

// Values match hashSeed/mulberry32 in the Next.js mocks (seed "demo").
func TestDeterministic(t *testing.T) {
	if Seed("demo") != Seed("demo") || Seed("demo") == Seed("demo2") {
		t.Fatal("seed")
	}
	a, b := New("x"), New("x")
	for i := 0; i < 5; i++ {
		if a.Float64() != b.Float64() {
			t.Fatal("streams differ")
		}
	}
	s := New("y")
	for i := 0; i < 1000; i++ {
		v := s.Float64()
		if v < 0 || v >= 1 {
			t.Fatal(v)
		}
	}
}
