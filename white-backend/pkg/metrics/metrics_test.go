package metrics

import "testing"

func TestComputeDelta(t *testing.T) {
	d := ComputeDelta(120, 100)
	if d.Abs != 20 || d.Pct == nil || *d.Pct != 20 || d.Direction != Up {
		t.Fatalf("%+v", d)
	}
	z := ComputeDelta(5, 0)
	if z.Pct != nil || z.Direction != Up {
		t.Fatalf("%+v", z)
	}
	n := ComputeDelta(50, -100)
	if n.Pct == nil || *n.Pct != 150 {
		t.Fatalf("abs(prev) must be used: %+v", n)
	}
	f := ComputeDelta(1, 1)
	if f.Direction != Flat || IsGoodChange(f, false) != nil {
		t.Fatalf("%+v", f)
	}
	if g := IsGoodChange(ComputeDelta(3, 5), true); g == nil || !*g {
		t.Fatal("lower is better: a drop is good")
	}
}

func TestWeightedMean(t *testing.T) {
	if WeightedMean([]float64{1, 3}, []float64{1, 3}) != 2.5 {
		t.Fatal("weighted mean")
	}
	if WeightedMean(nil, nil) != 0 {
		t.Fatal("empty")
	}
	if Round(2.345, 2) != 2.35 || Round(1.005, 2) != 1 {
		t.Fatal("round")
	}
}
