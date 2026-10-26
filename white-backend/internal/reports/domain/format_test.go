package domain

import (
	"math"
	"testing"
	"time"
)

// Expected strings were produced by lib/format.ts under Node (id-ID ICU).

func TestFormatNumber(t *testing.T) {
	cases := []struct {
		v      float64
		digits int
		want   string
	}{
		{1234567, 0, "1.234.567"},
		{2.25, 1, "2,3"},
		{2.35, 1, "2,4"},
		{1.005, 2, "1,01"},
		{-1234.5, 0, "-1.235"},
		{0.145, 2, "0,15"},
		{-0.4, 0, "-0"},
		{999.5, 0, "1.000"},
		{12345.675, 2, "12.345,68"},
		{math.NaN(), 0, "\u2013"},
	}
	for _, c := range cases {
		if got := FormatNumber(c.v, c.digits); got != c.want {
			t.Errorf("FormatNumber(%v, %d) = %q, want %q", c.v, c.digits, got, c.want)
		}
	}
}

func TestFormatCompact(t *testing.T) {
	cases := []struct {
		v    float64
		want string
	}{
		{0, "0"}, {999, "999"}, {999.4, "999,4"}, {1000, "1\u00a0rb"}, {1250, "1,3\u00a0rb"}, {1284, "1,3\u00a0rb"},
		{1050, "1,1\u00a0rb"}, {9999, "10\u00a0rb"}, {12345, "12,3\u00a0rb"}, {999950, "1000\u00a0rb"},
		{1_000_000, "1\u00a0jt"}, {1_250_000, "1,3\u00a0jt"}, {125_400, "125,4\u00a0rb"}, {8_412, "8,4\u00a0rb"},
		{45_200_000, "45,2\u00a0jt"}, {2_100_000_000, "2,1\u00a0M"}, {-1500, "-1,5\u00a0rb"}, {0.5, "0,5"},
		{12.34, "12,3"}, {-12.5, "-12,5"},
	}
	for _, c := range cases {
		if got := FormatCompact(c.v); got != c.want {
			t.Errorf("FormatCompact(%v) = %q, want %q", c.v, got, c.want)
		}
	}
}

func TestFormatCurrency(t *testing.T) {
	cases := []struct {
		v        float64
		currency string
		compact  bool
		want     string
	}{
		{1500000, "IDR", false, "Rp 1.500.000"},
		{-1500, "IDR", false, "-Rp 1.500"},
		{1234.5, "USD", false, "US$1.234,50"},
		{1234.5, "SGD", false, "SGD 1.234,50"},
		{-1234.5, "EUR", false, "-€1.234,50"},
		{45_200_000, "IDR", true, "Rp 45,2\u00a0jt"},
		{1234.5, "USD", true, "USD 1,2\u00a0rb"},
		{37542.4, "IDR", false, "Rp 37.542"},
	}
	for _, c := range cases {
		if got := FormatCurrency(c.v, c.currency, c.compact); got != c.want {
			t.Errorf("FormatCurrency(%v, %s, %v) = %q, want %q", c.v, c.currency, c.compact, got, c.want)
		}
	}
}

func TestFormatPercentAndDeltas(t *testing.T) {
	checks := [][2]string{
		{FormatPercent(4.32, 2), "4,32%"},
		{FormatPercent(75, 1), "75,0%"},
		{FormatPercent(2.095, 2), "2,10%"},
		{FormatDeltaPercent(12.34), "+12,3%"},
		{FormatDeltaPercent(-4), "\u22124,0%"},
		{FormatDeltaPercent(0), "0,0%"},
		{FormatDeltaPercent(0.05), "+0,1%"},
		{FormatDeltaPercent(-0.04), "\u22120,0%"},
		{FormatDeltaNumber(120), "+120"},
		{FormatDeltaNumber(-35), "\u221235"},
		{FormatDeltaNumber(0), "0"},
		{FormatDeltaNumber(1234), "+1.234"},
	}
	for _, c := range checks {
		if got, want := c[0], c[1]; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
	if got := FormatDeltaPercent(math.Inf(1)); got != Dash {
		t.Errorf("infinite delta = %q", got)
	}
}

func TestDateLabels(t *testing.T) {
	d := time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC)
	checks := [][2]string{
		{FormatDate(d), "19 Agu 2026"},
		{FormatDateShort(d), "19 Agu"},
		{FormatDateLong(d), "19 Agustus 2026"},
		{FormatDateTime(time.Date(2026, 8, 19, 7, 5, 0, 0, time.UTC)), "19 Agu 2026, 14.05"},
		{FormatDateTime(time.Date(2026, 8, 19, 17, 0, 0, 0, time.UTC)), "20 Agu 2026, 00.00"},
		{FormatDateRange(d, time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)), "19 Agu \u2013 15 Sep 2026"},
		{FormatDateRange(time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)), "20 Des 2025 \u2013 16 Jan 2026"},
	}
	for _, c := range checks {
		if got, want := c[0], c[1]; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	checks := [][2]string{
		{Truncate("Kopi single origin Gayo tiba di toko hari ini", 20), "Kopi single origin…"},
		{Truncate("abc", 3), "abc"},
		{Truncate("abcd", 3), "ab…"},
		{Truncate("hello world  x", 13), "hello world…"},
		{Truncate("ab😀cd", 4), "ab…"}, // the emoji is two UTF-16 units and never split
	}
	for _, c := range checks {
		if got, want := c[0], c[1]; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
