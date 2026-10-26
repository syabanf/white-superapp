package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
)

// Vocabularies of lib/sales-funnel/server.ts, adapted to an agency: the
// organisation types and lead sources a digital agency sees, and service
// lines in place of NüHabit's event types.
var (
	LeadOrgTypes     = []string{"corporate", "umkm", "startup", "brand", "pemerintah", "pendidikan", "perorangan", "lainnya"}
	LeadSources      = []string{"wa", "instagram", "referral", "google", "linkedin", "pameran", "website", "chatbot", "form", "lainnya"}
	LeadTemperatures = []string{"panas", "hangat", "dingin"}
	LeadStatuses     = []string{"baru", "dihubungi", "qualified", "tidak-cocok"}
	ActivityTypes    = []string{"telepon", "wa", "meeting", "catatan", "tugas", "email"}
	AccountTypes     = LeadOrgTypes
	DealMemberRoles  = []string{"owner", "support", "pre_sales", "account_manager", "finance"}
	PaymentMethods   = []string{"cash", "transfer", "qris", "edc", "lainnya"}
	// ForecastCategories are the manual overrides of a deal.
	ForecastCategories = []string{"pipeline", "best_case", "commit"}
)

// Service lines a deal can sell (replaces NüHabit's event_type).
const (
	LineSocialMedia = "social_media_management"
	LineSEO         = "seo"
	LineAds         = "ads_management"
	LineContent     = "content_production"
	LineWebsite     = "website"
	LineBranding    = "branding"
)

var ServiceLines = []string{LineSocialMedia, LineSEO, LineAds, LineContent, LineWebsite, LineBranding}

// ServiceLineLabels are the Indonesian labels used in documents.
var ServiceLineLabels = map[string]string{
	LineSocialMedia: "Social Media Management",
	LineSEO:         "SEO",
	LineAds:         "Ads Management",
	LineContent:     "Produksi Konten",
	LineWebsite:     "Pembuatan Website",
	LineBranding:    "Branding",
}

// ServiceLineLabel falls back to the key for unknown lines.
func ServiceLineLabel(line string) string {
	if l, ok := ServiceLineLabels[line]; ok {
		return l
	}
	return line
}

func Contains(list []string, s string) bool { return slices.Contains(list, s) }

// NormalizePhone is normalizePhone: canonical 62... digits.
func NormalizePhone(raw string) string {
	var b strings.Builder
	for _, c := range raw {
		if c >= '0' && c <= '9' {
			b.WriteRune(c)
		}
	}
	digits := b.String()
	switch {
	case strings.HasPrefix(digits, "0"):
		return "62" + digits[1:]
	case strings.HasPrefix(digits, "8"):
		return "62" + digits
	}
	return digits
}

// IsValidNormalizedPhone is isValidNormalizedPhone.
func IsValidNormalizedPhone(phone string) bool {
	return len(phone) >= 10 && strings.HasPrefix(phone, "62")
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func IsEmail(s string) bool { return emailRe.MatchString(s) }

// IsValidCalendarDate is isValidCalendarDate for "YYYY-MM-DD".
func IsValidCalendarDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

// ParseDate parses "YYYY-MM-DD"; empty is nil.
func ParseDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// FormatDay formats a nullable date as "YYYY-MM-DD" ("" when nil).
func FormatDay(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// JSTrim is String.prototype.trim.
func JSTrim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == 0xFEFF })
}

// SliceUTF16 is s.slice(0, n) in UTF-16 code units.
func SliceUTF16(s string, n int) string {
	units := utf16.Encode([]rune(s))
	if len(units) <= n {
		return s
	}
	return string(utf16.Decode(units[:n]))
}

// TrimmedOrNil is `value?.trim() || null`.
func TrimmedOrNil(value string) *string {
	if s := JSTrim(value); s != "" {
		return &s
	}
	return nil
}

var (
	monthsLong = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	weekdays   = [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
)

// FormatDateLong is formatDateLong: "Sabtu, 7 November 2026".
func FormatDateLong(t time.Time) string {
	return weekdays[t.Weekday()] + ", " + strconv.Itoa(t.Day()) + " " + monthsLong[t.Month()-1] + " " + strconv.Itoa(t.Year())
}

// FormatDateID is "7 November 2026".
func FormatDateID(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + monthsLong[t.Month()-1] + " " + strconv.Itoa(t.Year())
}

// Jakarta is Asia/Jakarta (WIB).
var Jakarta = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}()

// DocumentPrefix is "QT-2610" for a quotation issued in October 2026 (WIB).
func DocumentPrefix(kind string, at time.Time) string {
	return kind + "-" + at.In(Jakarta).Format("0601")
}

// DocumentNumber is the numbering format QT-YYMM-#### / KTR-YYMM-####.
func DocumentNumber(prefix string, n int) string {
	return fmt.Sprintf("%s-%04d", prefix, n)
}

// Document number kinds.
const (
	NumberQuotation = "QT"
	NumberContract  = "KTR"
	NumberInvoice   = "INV"
)

// StageFlags is the target stage of a categorisation or stage move.
type StageFlags struct {
	IsWon, IsLost bool
	Probability   int
}

// CategoryFromStage is categoryFromStage.
func CategoryFromStage(s StageFlags) string {
	switch {
	case s.IsWon:
		return "closed_won"
	case s.IsLost:
		return "closed_lost"
	case s.Probability >= 75:
		return "commit"
	case s.Probability >= 50:
		return "best_case"
	}
	return "pipeline"
}

// StageMove is the outcome of PlanStageMove.
type StageMove struct {
	ForecastCategory string
	ClosedAt         *time.Time
	LostReasonID     *string
	ValueFinal       *Money
}

// PlanStageMove is planStageMove adapted to an agency: a won stage needs a
// final value (NüHabit also required a fixed event date; agencies sell
// retainers, so that rule is dropped), a lost stage needs a reason, an open
// stage clears closed_at and the reason. The forecast category follows the
// stage unless the caller overrides it with a manual category.
func PlanStageMove(stage StageFlags, currentFinal, newFinal *Money, lostReasonID *string, override string, now time.Time) (StageMove, error) {
	mv := StageMove{ForecastCategory: CategoryFromStage(stage)}
	if override != "" && !stage.IsWon && !stage.IsLost {
		mv.ForecastCategory = override
	}
	final := currentFinal
	if newFinal != nil {
		final = newFinal
	}
	mv.ValueFinal = final
	switch {
	case stage.IsWon:
		if final == nil {
			return mv, ErrWonNeedsValue
		}
		mv.ClosedAt = &now
	case stage.IsLost:
		if lostReasonID == nil || *lostReasonID == "" {
			return mv, ErrLostNeedsReason
		}
		mv.LostReasonID = lostReasonID
		mv.ClosedAt = &now
	}
	return mv, nil
}

// Rule errors with the NüHabit messages.
var (
	ErrWonNeedsValue   = RuleError("Deal Menang wajib diisi nilai final")
	ErrLostNeedsReason = RuleError("Deal Kalah wajib pilih alasan kalah")
)

// RuleError is a business rule violation shown to the user as is.
type RuleError string

func (e RuleError) Error() string { return string(e) }

// SplitTotalOK checks that member splits never exceed 100%.
func SplitTotalOK(splits []float64) bool {
	sum := 0.0
	for _, s := range splits {
		if s < 0 || s > 100 {
			return false
		}
		sum += s
	}
	return sum <= 100.0001
}
