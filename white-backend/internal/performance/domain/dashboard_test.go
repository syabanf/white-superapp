package domain

import (
	"slices"
	"testing"
	"time"
)

func TestResolveWindow(t *testing.T) {
	cases := []struct {
		from, to, compare            string
		wantFrom, wantCFrom, wantCTo string
		granularity                  string
	}{
		{"", "", "", "2026-10-01", "2026-09-10", "2026-09-30", BucketDay},
		{"2026-01-01", "2026-10-21", "year", "2026-01-01", "2025-01-01", "2025-10-21", BucketMonth},
		{"2026-07-01", "2026-09-30", "previous", "2026-07-01", "2026-03-31", "2026-06-30", BucketWeek},
		{"2026-10-01", "2026-10-21", "none", "2026-10-01", "", "", BucketDay},
	}
	for _, c := range cases {
		w, ok := ResolveWindow(c.from, c.to, c.compare, "2026-10-21")
		if !ok || w.From != c.wantFrom || w.CompareFrom != c.wantCFrom || w.CompareTo != c.wantCTo || w.Granularity != c.granularity {
			t.Errorf("%s..%s %s: %+v", c.from, c.to, c.compare, w)
		}
	}
	if _, ok := ResolveWindow("2026-10-21", "2026-10-01", "", "2026-10-21"); ok {
		t.Error("reversed range accepted")
	}
}

func TestBuckets(t *testing.T) {
	if got := Buckets("2026-10-01", "2026-10-14", BucketWeek); !slices.Equal(got, []string{"2026-09-28", "2026-10-05", "2026-10-12"}) {
		t.Fatal(got)
	}
	if got := Buckets("2026-08-15", "2026-10-02", BucketMonth); !slices.Equal(got, []string{"2026-08-01", "2026-09-01", "2026-10-01"}) {
		t.Fatal(got)
	}
	if Workdays("2026-10-19", "2026-10-25") != 5 || OverdueAgeIndex(15) != 3 || ScoreBinIndex(89.9) != 4 || ScoreBinIndex(100) != 5 {
		t.Fatal("bins")
	}
}

func TestImprovementOpenAt(t *testing.T) {
	created := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	done := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
	m := DashImprovement{Status: ImpDone, CreatedAt: created, CompletedAt: &done, UpdatedAt: done}
	if !m.OpenAt("2026-09-10") || m.OpenAt("2026-09-20") || m.OpenAt("2026-08-31") {
		t.Fatal("done item")
	}
	m.Status = ImpSuggested
	if m.OpenAt("2026-09-10") {
		t.Fatal("suggestion counted as open")
	}
}
