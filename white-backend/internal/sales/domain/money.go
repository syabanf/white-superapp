package domain

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// Money is an amount in sen (1/100 rupiah). SQL stores numeric(18,2) and
// the API carries decimal strings; arithmetic never goes through float for
// stored amounts.
type Money int64

var ErrInvalidMoney = errors.New("nominal tidak valid")

// ParseMoney reads "1250000", "1250000.5" or "-12.30". Empty is zero.
// Grouping separators are not accepted: the API speaks plain decimals.
func ParseMoney(s string) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	neg := false
	if s[0] == '-' || s[0] == '+' {
		neg = s[0] == '-'
		s = s[1:]
	}
	intPart, frac, hasFrac := strings.Cut(s, ".")
	if intPart == "" || !digitsOnly(intPart) || (hasFrac && (frac == "" || !digitsOnly(frac))) || len(intPart) > 16 {
		return 0, ErrInvalidMoney
	}
	whole, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, ErrInvalidMoney
	}
	cents := whole * 100
	if hasFrac {
		// Round half up on the third decimal, like numeric(18,2) does.
		padded := (frac + "000")[:3]
		n, _ := strconv.Atoi(padded)
		cents += int64((n + 5) / 10)
	}
	if neg {
		cents = -cents
	}
	return Money(cents), nil
}

// MustMoney is ParseMoney for values the database produced.
func MustMoney(s string) Money {
	m, err := ParseMoney(s)
	if err != nil {
		return 0
	}
	return m
}

// MoneyPtr parses a nullable column.
func MoneyPtr(s *string) *Money {
	if s == nil {
		return nil
	}
	m := MustMoney(*s)
	return &m
}

func digitsOnly(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// String is the canonical decimal with two places: "1250000.50".
func (m Money) String() string {
	sign := ""
	v := int64(m)
	if v < 0 {
		sign, v = "-", -v
	}
	return sign + strconv.FormatInt(v/100, 10) + "." + leftPad(strconv.FormatInt(v%100, 10), 2)
}

// StringPtr formats a nullable amount.
func (m *Money) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := m.String()
	return &s
}

func leftPad(s string, n int) string {
	for len(s) < n {
		s = "0" + s
	}
	return s
}

// roundHalfUp rounds to the nearest integer, halves away from zero (JS
// Math.round matches this for the non-negative amounts sales deals in).
func roundHalfUp(f float64) int64 {
	if f < 0 {
		return -int64(math.Floor(-f + 0.5))
	}
	return int64(math.Floor(f + 0.5))
}

// Percent returns pct percent of m rounded to the sen.
func (m Money) Percent(pct float64) Money {
	return Money(roundHalfUp(float64(m) * pct / 100))
}

// Times multiplies by a quantity (numeric(12,3)) rounded to the sen.
func (m Money) Times(qty float64) Money {
	return Money(roundHalfUp(float64(m) * qty))
}

// Rupiah is formatRupiah: "Rp1.250.000", rounded to whole rupiah.
func (m Money) Rupiah() string {
	v := int64(m)
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	return sign + "Rp" + GroupThousands(strconv.FormatInt((v+50)/100, 10))
}

// GroupThousands puts "." between groups of three digits.
func GroupThousands(intPart string) string {
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// FormatNumber is formatNumber (toLocaleString "id-ID", at most
// maxFractionDigits decimals).
func FormatNumber(v float64, maxFractionDigits int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		v = 0
	}
	scale := math.Pow10(maxFractionDigits)
	rounded := math.Round(v*scale) / scale
	sign := ""
	if rounded < 0 {
		sign, rounded = "-", -rounded
	}
	intPart, frac, _ := strings.Cut(strconv.FormatFloat(rounded, 'f', maxFractionDigits, 64), ".")
	frac = strings.TrimRight(frac, "0")
	out := sign + GroupThousands(intPart)
	if frac != "" {
		out += "," + frac
	}
	return out
}

// ParsePercent reads a numeric(5,2) column.
func ParsePercent(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return f
}

// PercentString formats a percent for numeric(5,2).
func PercentString(f float64) string { return strconv.FormatFloat(math.Round(f*100)/100, 'f', -1, 64) }
