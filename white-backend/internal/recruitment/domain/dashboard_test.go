package domain

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, _ := ParseDay(s)
	return t
}

func TestComparePeriod(t *testing.T) {
	p := NewPeriod(day("2026-10-01"), day("2026-10-31"))
	if p.Days() != 31 || !p.FromTS.Equal(time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC)) || !p.ToTS.Equal(time.Date(2026, 10, 31, 17, 0, 0, 0, time.UTC)) {
		t.Fatal(p)
	}
	prev := ComparePeriod(p, ComparePrevious)
	if prev.From.Format(IsoDate) != "2026-08-31" || prev.To.Format(IsoDate) != "2026-09-30" {
		t.Fatal(prev)
	}
	year := ComparePeriod(p, CompareYear)
	if year.From.Format(IsoDate) != "2025-10-01" || year.To.Format(IsoDate) != "2025-10-31" {
		t.Fatal(year)
	}
}

func TestKpisFromTrend(t *testing.T) {
	k := KpisFromTrend([]TrendBucket{
		{Applications: 5, Hires: 1, HireDays: 20, OffersSent: 2, OffersAccepted: 1, InProcess: 9, OpenPositions: 3, OpenHeadcount: 4},
		{Applications: 3, Hires: 1, HireDays: 31, OffersDeclined: 1, OffersExpired: 1, InProcess: 7, OpenPositions: 2, OpenHeadcount: 2},
	})
	if k.Applicants != 8 || k.Hires != 2 || k.OffersSent != 2 || k.InProcess != 7 || k.OpenPositions != 2 || k.OpenHeadcount != 2 {
		t.Fatal(k)
	}
	if *k.AvgDaysToHire != 25.5 || *k.OfferAcceptanceRate != 33.3 {
		t.Fatal(*k.AvgDaysToHire, *k.OfferAcceptanceRate)
	}
	if empty := KpisFromTrend(nil); empty.AvgDaysToHire != nil || empty.OfferAcceptanceRate != nil {
		t.Fatal(empty)
	}
}

func TestFunnelAndSources(t *testing.T) {
	rows := []CohortRow{
		{Source: "portal", Status: "rejected", MaxRank: 0, N: 4},
		{Source: "portal", Status: "screening", MaxRank: 1, N: 2},
		{Source: "referral", Status: "talent_pool", MaxRank: 3, N: 1},
		{Source: "referral", Status: "hired", MaxRank: 5, N: 1, HireDays: 18},
	}
	f := Funnel(rows)
	want := []FunnelStage{{"applied", 8, 0, 4}, {"screening", 4, 2, 0}, {"psikotes", 2, 0, 0}, {"interview", 2, 0, 1}, {"offer", 1, 0, 0},
		{"hired", 1, 1, 0}}
	for i := range want {
		if f[i] != want[i] {
			t.Fatalf("stage %d: %+v want %+v", i, f[i], want[i])
		}
	}
	s := Sources(rows)
	if len(s) != 2 || s[0].Source != "portal" || s[0].Applications != 6 || s[0].Screened != 2 || s[0].Hires != 0 {
		t.Fatalf("%+v", s)
	}
	if r := s[1]; r.Paid || r.Interviewed != 2 || r.Offered != 1 || r.Hires != 1 || *r.AvgDaysToHire != 18 {
		t.Fatalf("%+v", r)
	}
}
