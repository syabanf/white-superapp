package hris

import (
	"strconv"
	"strings"
)

// Holiday is an active hris.public_holidays row.
type Holiday struct {
	Date         string
	Name         string
	Type         string
	DeductsLeave bool
}

// HolidayIndex groups holidays by date.
type HolidayIndex map[string][]Holiday

// IndexHolidays is indexHolidays.
func IndexHolidays(rows []Holiday) HolidayIndex {
	idx := HolidayIndex{}
	for _, h := range rows {
		idx[h.Date] = append(idx[h.Date], h)
	}
	return idx
}

// IsWeekend: Saturday or Sunday.
func IsWeekend(dateISO string) bool { return IsoDayOfWeek(dateISO) >= 6 }

// ExcludedHoliday names a holiday that kept a date from consuming leave.
type ExcludedHoliday struct {
	Date string `json:"date"`
	Name string `json:"name"`
}

// DescribeLeaveDays counts the days in [start, end] that consume annual
// leave: weekends and holidays with deducts_leave = false are skipped (a
// national holiday beats cuti bersama on the same date); cuti bersama still
// counts. It also lists the holidays that were skipped.
func DescribeLeaveDays(startISO, endISO string, idx HolidayIndex) (int, []ExcludedHoliday) {
	excluded := []ExcludedHoliday{}
	total := 0
	for _, d := range EachDateISO(startISO, endISO) {
		var exempt *Holiday
		for i := range idx[d] {
			if !idx[d][i].DeductsLeave {
				exempt = &idx[d][i]
				break
			}
		}
		if exempt != nil {
			excluded = append(excluded, ExcludedHoliday{Date: d, Name: exempt.Name})
			continue
		}
		if IsWeekend(d) {
			continue
		}
		total++
	}
	return total, excluded
}

// NoWorkingDayMessage explains a leave range with zero working days.
func NoWorkingDayMessage(excluded []ExcludedHoliday) string {
	alasan := "jatuh pada akhir pekan"
	if len(excluded) > 0 {
		names := make([]string, len(excluded))
		for i, h := range excluded {
			names[i] = h.Name
		}
		alasan = "sudah hari libur (" + strings.Join(names, ", ") + ")"
	}
	return "Rentang tanggal ini " + alasan + ", tidak perlu mengajukan cuti"
}

/* ── Input rules (lib/hris/holidays-input) ───────────────────────────── */

// HolidayTypes are the hris.public_holidays.type values.
var HolidayTypes = []string{"nasional", "cuti_bersama", "perusahaan"}

// HolidayInput is the optional-field body of the holiday routes.
type HolidayInput struct {
	HolidayDate  *string
	Name         *string
	Type         *string
	DeductsLeave *bool
	Status       *string
	Note         *string
	NoteSent     bool
	SourceRef    *string
}

func isHolidayType(t string) bool {
	for _, v := range HolidayTypes {
		if v == t {
			return true
		}
	}
	return false
}

func typeOrStatusError(in HolidayInput) string {
	if in.Type != nil && !isHolidayType(*in.Type) {
		return "Tipe libur tidak valid"
	}
	if in.Status != nil && *in.Status != "draft" && *in.Status != "aktif" {
		return "Status tidak valid"
	}
	return ""
}

// jsDateValid is !isNaN(new Date(`${d}T00:00:00Z`)) for a YYYY-MM-DD string:
// V8 accepts any day 1..31 regardless of the month.
func jsDateValid(d string) bool {
	m, _ := strconv.Atoi(d[5:7])
	day, _ := strconv.Atoi(d[8:10])
	return m >= 1 && m <= 12 && day >= 1 && day <= 31
}

func blank(s *string) bool { return s == nil || JSTrim(*s) == "" }

// ValidateHolidayBody is validateHolidayBody (POST); "" means valid.
func ValidateHolidayBody(in HolidayInput) string {
	if in.HolidayDate == nil || !DateRe.MatchString(*in.HolidayDate) {
		return "Tanggal wajib diisi (YYYY-MM-DD)"
	}
	if !jsDateValid(*in.HolidayDate) {
		return "Tanggal tidak valid"
	}
	if blank(in.Name) {
		return "Nama libur wajib diisi"
	}
	return typeOrStatusError(in)
}

// HolidayTypeOrDefault: unknown or missing types are "nasional".
func HolidayTypeOrDefault(t *string) string {
	if t != nil && isHolidayType(*t) {
		return *t
	}
	return "nasional"
}

// DefaultDeductsLeave: only cuti bersama consumes annual leave (SKB).
func DefaultDeductsLeave(t string) bool { return t == "cuti_bersama" }
