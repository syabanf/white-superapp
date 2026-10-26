package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Money is an amount in sen (1/100 rupiah). NüHabit kept amounts in
// float64 and rounded to cents on every step (round2); integer sen gives the
// same results without the rounding noise. SQL stores numeric(18,2), the
// proto carries decimal strings.
type Money int64

var ErrInvalidMoney = errors.New("nominal tidak valid")

// ParseMoney accepts "1250000", "1250000.5", "1250000.50", "-10.25" and
// surrounding spaces. More than two decimals are rounded half away from zero.
func ParseMoney(s string) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	neg := false
	switch s[0] {
	case '-':
		neg = true
		s = s[1:]
	case '+':
		s = s[1:]
	}
	whole, frac, hasDot := strings.Cut(s, ".")
	if whole == "" && (!hasDot || frac == "") {
		return 0, ErrInvalidMoney
	}
	if whole == "" {
		whole = "0"
	}
	for _, r := range whole + frac {
		if r < '0' || r > '9' {
			return 0, ErrInvalidMoney
		}
	}
	if len(whole) > 16 {
		return 0, ErrInvalidMoney
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrInvalidMoney
	}
	roundUp := len(frac) > 2 && frac[2] >= '5'
	frac = (frac + "00")[:2]
	f, _ := strconv.ParseInt(frac, 10, 64)
	v := w*100 + f
	if roundUp {
		v++
	}
	if neg {
		v = -v
	}
	return Money(v), nil
}

// MustMoney parses a literal (seeds and tests).
func MustMoney(s string) Money {
	m, err := ParseMoney(s)
	if err != nil {
		panic(err)
	}
	return m
}

// Rupiah builds a whole-rupiah amount.
func Rupiah(n int64) Money { return Money(n * 100) }

// String is the proto form: "1250000.00".
func (m Money) String() string {
	v := int64(m)
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}

// Number is String(n) of the JavaScript number NüHabit printed in messages
// ("Debit 1500 ≠ Credit 1499.5").
func (m Money) Number() string {
	s := m.String()
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// Float is for ratios and charts only.
func (m Money) Float() float64 { return float64(m) / 100 }

func (m Money) Abs() Money {
	if m < 0 {
		return -m
	}
	return m
}

// FromFloat rounds a float amount to sen (analysis inputs only).
func FromFloat(f float64) Money { return Money(math.Round(f * 100)) }

// Value stores the decimal text; pgx encodes it into numeric.
func (m Money) Value() (driver.Value, error) { return m.String(), nil }

// Scan reads numeric columns, which pgx hands over as text.
func (m *Money) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*m = 0
		return nil
	case string:
		p, err := ParseMoney(v)
		*m = p
		return err
	case []byte:
		p, err := ParseMoney(string(v))
		*m = p
		return err
	case int64:
		*m = Money(v * 100)
		return nil
	case float64:
		*m = FromFloat(v)
		return nil
	}
	return fmt.Errorf("money: cannot scan %T", src)
}

// UnmarshalJSON accepts a decimal string or a JSON number, so event
// payloads may use either.
func (m *Money) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		*m = 0
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return err
		}
		s = str
	}
	p, err := ParseMoney(s)
	*m = p
	return err
}

func (m Money) MarshalJSON() ([]byte, error) { return json.Marshal(m.String()) }
