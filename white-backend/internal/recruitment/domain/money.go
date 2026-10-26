package domain

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Money is an amount in sen (1/100 rupiah). SQL stores numeric(18,2), the
// proto carries decimal strings.
type Money int64

var ErrInvalidMoney = errors.New("nominal tidak valid")

// ParseMoney accepts "1250000", "1250000.5", "1250000.50" and surrounding
// spaces. More than two decimals are rounded half away from zero.
func ParseMoney(s string) (Money, error) {
	s = strings.TrimSpace(s)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimLeft(s, "+-")
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
	f, _ := strconv.ParseInt((frac + "00")[:2], 10, 64)
	v := w*100 + f
	if roundUp {
		v++
	}
	if neg {
		v = -v
	}
	return Money(v), nil
}

// Rupiah builds a whole-rupiah amount.
func Rupiah(n int64) Money { return Money(n * 100) }

// String is the proto form: "1250000.00".
func (m Money) String() string {
	v := int64(m)
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}

// FormatIDR is "Rp 5.000.000" (whole rupiah, dot thousands).
func (m Money) FormatIDR() string { return "Rp " + Thousands(int64(m)/100) }

// Thousands groups digits with dots: 5000000 -> "5.000.000".
func Thousands(n int64) string {
	sign := ""
	if n < 0 {
		sign, n = "-", -n
	}
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return sign + b.String()
}

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
	}
	return fmt.Errorf("money: cannot scan %T", src)
}
