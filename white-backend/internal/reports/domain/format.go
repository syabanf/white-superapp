package domain

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
)

// Formatting helpers ported from lib/format.ts (id-ID locale). Intl rounds
// the shortest decimal form of a double half away from zero; toFixed rounds
// the exact binary value, ties up. Both are reproduced here because the
// strings reach the PDF and the report sections verbatim.

// Dash is what every formatter prints for a missing value (U+2013).
const Dash = "\u2013"

const nbsp = "\u00a0"

var monthsShort = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

var monthsLong = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// FormatNumber is formatNumber(v, digits): "1.234.567", "2,5".
func FormatNumber(v float64, digits int) string {
	if math.IsNaN(v) {
		return Dash
	}
	if math.IsInf(v, 0) {
		if v < 0 {
			return "-∞"
		}
		return "∞"
	}
	neg := v < 0 || (v == 0 && math.Signbit(v))
	intPart, frac := roundHalfUp(strconv.FormatFloat(math.Abs(v), 'f', -1, 64), digits)
	out := groupThousands(intPart)
	if digits > 0 {
		out += "," + frac
	}
	if neg {
		return "-" + out
	}
	return out
}

// FormatCompact is formatCompact(v, 1): "1,3\u00a0rb", "2,1\u00a0M".
func FormatCompact(v float64) string {
	const digits = 1
	if math.IsNaN(v) {
		return Dash
	}
	abs := math.Abs(v)
	sign := ""
	if v < 0 {
		sign = "-"
	}
	switch {
	case abs >= 1e9:
		return sign + trimZero(toFixed(abs/1e9, digits)) + nbsp + "M"
	case abs >= 1e6:
		return sign + trimZero(toFixed(abs/1e6, digits)) + nbsp + "jt"
	case abs >= 1e3:
		return sign + trimZero(toFixed(abs/1e3, digits)) + nbsp + "rb"
	}
	d := digits
	if abs == math.Trunc(abs) {
		d = 0
	}
	return sign + FormatNumber(abs, d)
}

// Currency symbols the id-ID locale prints instead of the ISO code.
var currencySymbols = map[string]string{
	"IDR": "Rp", "USD": "US$", "EUR": "€", "GBP": "£", "JPY": "JP¥", "AUD": "AU$", "THB": "฿",
	"VND": "₫", "CNY": "CN¥", "HKD": "HK$", "KRW": "₩", "INR": "Rs", "CAD": "CA$", "NZD": "NZ$",
	"TWD": "NT$", "BRL": "R$",
}

// FormatCurrency is formatCurrency(v, currency, {compact}). IDR has no
// decimals, other currencies two; compact uses "Rp" or the ISO code.
func FormatCurrency(v float64, currency string, compact bool) string {
	if math.IsNaN(v) {
		return Dash
	}
	if currency == "" {
		currency = "IDR"
	}
	if compact {
		symbol := currency
		if currency == "IDR" {
			symbol = "Rp"
		}
		return symbol + " " + FormatCompact(v)
	}
	digits := 2
	if currency == "IDR" {
		digits = 0
	}
	num := FormatNumber(v, digits)
	sign := ""
	if strings.HasPrefix(num, "-") {
		sign, num = "-", num[1:]
	}
	symbol, ok := currencySymbols[currency]
	if !ok {
		symbol = currency
	}
	// ICU currency spacing: a symbol that ends in a letter gets a space.
	if r := []rune(symbol); unicode.IsLetter(r[len(r)-1]) {
		symbol += " "
	}
	return sign + symbol + num
}

// FormatPercent is formatPercent(v, digits); v is already 0 to 100.
func FormatPercent(v float64, digits int) string {
	if math.IsNaN(v) {
		return Dash
	}
	return FormatNumber(v, digits) + "%"
}

// FormatDeltaPercent is formatDeltaPercent(v, 1): "+12,3%", "\u22124,0%".
func FormatDeltaPercent(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return Dash
	}
	return deltaSign(v) + FormatNumber(math.Abs(v), 1) + "%"
}

// FormatDeltaNumber is formatDeltaNumber(v, 0): "+120", "\u221235".
func FormatDeltaNumber(v float64) string {
	if math.IsNaN(v) {
		return Dash
	}
	return deltaSign(v) + FormatNumber(math.Abs(v), 0)
}

func deltaSign(v float64) string {
	switch {
	case v > 0:
		return "+"
	case v < 0:
		return "\u2212"
	}
	return ""
}

// FormatDate is "19 Agu 2026" (UTC).
func FormatDate(t time.Time) string {
	t = t.UTC()
	return strconv.Itoa(t.Day()) + " " + monthsShort[t.Month()-1] + " " + strconv.Itoa(t.Year())
}

// FormatDateShort is "19 Agu" (UTC).
func FormatDateShort(t time.Time) string {
	t = t.UTC()
	return strconv.Itoa(t.Day()) + " " + monthsShort[t.Month()-1]
}

// FormatDateLong is "19 Agustus 2026" (UTC).
func FormatDateLong(t time.Time) string {
	t = t.UTC()
	return strconv.Itoa(t.Day()) + " " + monthsLong[t.Month()-1] + " " + strconv.Itoa(t.Year())
}

// FormatDateTime is "19 Agu 2026, 14.05" in Asia/Jakarta.
func FormatDateTime(t time.Time) string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	l := t.In(loc)
	return strconv.Itoa(l.Day()) + " " + monthsShort[l.Month()-1] + " " + strconv.Itoa(l.Year()) + ", " + l.Format("15.04")
}

// FormatDateRange is "19 Agu \u2013 15 Sep 2026", or both dates in full
// when the years differ.
func FormatDateRange(from, to time.Time) string {
	left := FormatDate(from)
	if from.UTC().Year() == to.UTC().Year() {
		left = FormatDateShort(from)
	}
	return left + " \u2013 " + FormatDate(to)
}

// Truncate is truncate(text, max): lengths count UTF-16 code units like JS.
func Truncate(text string, max int) string {
	units := utf16.Encode([]rune(text))
	if len(units) <= max {
		return text
	}
	cut := units[:max-1]
	if n := len(cut); n > 0 && utf16.IsSurrogate(rune(cut[n-1])) && cut[n-1] < 0xdc00 {
		cut = cut[:n-1] // never leave half an emoji behind
	}
	return strings.TrimRightFunc(string(utf16.Decode(cut)), unicode.IsSpace) + "…"
}

// roundHalfUp rounds a plain decimal string ("12.345") to `digits`
// fraction digits, ties away from zero. It returns the integer part and the
// zero-padded fraction.
func roundHalfUp(s string, digits int) (string, string) {
	intPart, frac, _ := strings.Cut(s, ".")
	for len(frac) < digits {
		frac += "0"
	}
	if len(frac) == digits {
		return intPart, frac
	}
	up := frac[digits] >= '5'
	frac = frac[:digits]
	if !up {
		return intPart, frac
	}
	b := []byte(intPart + frac)
	i := len(b) - 1
	for ; i >= 0; i-- {
		if b[i] == '9' {
			b[i] = '0'
			continue
		}
		b[i]++
		break
	}
	if i < 0 {
		b = append([]byte{'1'}, b...)
	}
	n := len(b) - digits
	return string(b[:n]), string(b[n:])
}

// toFixed is Number.prototype.toFixed for non-negative values.
func toFixed(v float64, digits int) string {
	intPart, frac := roundHalfUp(strconv.FormatFloat(v, 'f', 1080, 64), digits)
	if digits == 0 {
		return intPart
	}
	return intPart + "." + frac
}

// trimZero drops trailing fraction zeros and swaps "." for ",".
func trimZero(s string) string {
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	return strings.Replace(s, ".", ",", 1)
}

func groupThousands(intPart string) string {
	if len(intPart) <= 3 {
		return intPart
	}
	var b strings.Builder
	head := len(intPart) % 3
	if head > 0 {
		b.WriteString(intPart[:head])
	}
	for i := head; i < len(intPart); i += 3 {
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(intPart[i : i+3])
	}
	return b.String()
}
