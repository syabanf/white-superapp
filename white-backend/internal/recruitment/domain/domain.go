// Package domain holds the recruitment rules ported from NüHabit: candidate
// statuses and sources, portal link validity, psikotes scoring, offer
// responses, job opening normalization, the AI interview flow, interview
// scorecards and upload checks. Pure Go, no database or gRPC.
package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// CandidateStatuses are the keys of CANDIDATE_STATUS_LABELS
// (lib/recruitment/status.ts), in pipeline order. "archived" is the talent
// pool archive.
var CandidateStatuses = []string{
	"applied", "screening", "psikotes", "interview", "offer", "hired", "talent_pool", "rejected", "archived",
}

var statusLabels = map[string]string{
	"applied":     "Applied",
	"screening":   "Screening",
	"psikotes":    "Psikotes",
	"interview":   "Interview",
	"offer":       "Offer",
	"hired":       "Hired",
	"talent_pool": "Talent Pool",
	"rejected":    "Tolak",
	"archived":    "Diarsipkan",
}

// StatusLabel returns the UI label of a candidate status.
func StatusLabel(status string) (string, bool) {
	label, ok := statusLabels[status]
	return label, ok
}

// IsCandidateStatus reports a known pipeline status.
func IsCandidateStatus(s string) bool { return slices.Contains(CandidateStatuses, s) }

// CreatableStatuses are the statuses a new candidate may start in.
var CreatableStatuses = []string{"applied", "screening"}

// ActiveStatuses are the statuses still moving through the pipeline.
var ActiveStatuses = []string{"applied", "screening", "psikotes", "interview", "offer"}

// TalentPoolStatuses are the talent pool and its archive.
var TalentPoolStatuses = []string{"talent_pool", "archived"}

// CandidateSources are the keys of CANDIDATE_SOURCE_LABELS.
var CandidateSources = []string{
	"portal", "internal", "referral", "jobstreet", "instagram", "jobfair", "walk_in", "internal_referral", "headhunter", "other",
}

var sourceLabels = map[string]string{
	"portal":            "Portal Karir",
	"internal":          "Internal",
	"referral":          "Referral",
	"jobstreet":         "JobStreet",
	"instagram":         "Instagram",
	"jobfair":           "Job Fair",
	"walk_in":           "Walk-in",
	"internal_referral": "Referral Internal",
	"headhunter":        "Headhunter",
	"other":             "Lainnya",
}

// SourceLabel is the display label of a source (the key when unknown).
func SourceLabel(s string) string {
	if l, ok := sourceLabels[s]; ok {
		return l
	}
	return s
}

// Availabilities are CANDIDATE_AVAILABILITIES.
var Availabilities = []string{"immediate", "1_week", "2_weeks", "1_month"}

// Recommendations are the screening and psikotes summary verdicts.
var Recommendations = []string{"lolos", "hold", "tidak_lolos"}

var recommendationLabels = map[string]string{"lolos": "Lolos", "hold": "Hold", "tidak_lolos": "Tidak Lolos"}

// RecommendationLabel is the label of a verdict ("-" when empty).
func RecommendationLabel(rec string) string {
	if l, ok := recommendationLabels[rec]; ok {
		return l
	}
	if rec == "" {
		return "-"
	}
	return rec
}

// RecommendationSuffix is the activity suffix " (rekomendasi: Lolos)", or ""
// without a recommendation.
func RecommendationSuffix(rec *string) string {
	if rec == nil || *rec == "" {
		return ""
	}
	return " (rekomendasi: " + recommendationLabels[*rec] + ")"
}

// StageChangeDescription is the activity text of a stage move.
func StageChangeDescription(from, to string) string {
	fromLabel, ok := StatusLabel(from)
	if !ok {
		fromLabel = from
	}
	toLabel, _ := StatusLabel(to)
	return fmt.Sprintf("Tahap diubah: %s → %s", fromLabel, toLabel)
}

var portalTokenPattern = regexp.MustCompile(`(?i)^[a-f0-9]{48,128}$`)

// IsPortalToken reports whether s looks like a candidate portal link token
// (psikotes, interview, offer): 48 to 128 hex characters.
func IsPortalToken(s string) bool { return portalTokenPattern.MatchString(s) }

// LikePattern escapes LIKE wildcards so a search matches literally.
func LikePattern(search string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(search) + "%"
}

// Jakarta is WIB, the agency's time zone for days and reminders.
var Jakarta = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}()

const IsoDate = "2006-01-02"

// ParseDay reads a YYYY-MM-DD day.
func ParseDay(s string) (time.Time, bool) {
	t, err := time.ParseInLocation(IsoDate, s, time.UTC)
	return t, err == nil && t.Format(IsoDate) == s
}

// DayKey is the WIB day of an instant.
func DayKey(t time.Time) string { return t.In(Jakarta).Format(IsoDate) }

var idMonths = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus",
	"September", "Oktober", "November", "Desember"}

// FormatDate is toLocaleDateString("id-ID", {day, month: "long", year}) in
// Jakarta: "15 Juli 2026".
func FormatDate(t time.Time) string {
	t = t.In(Jakarta)
	return fmt.Sprintf("%d %s %d", t.Day(), idMonths[t.Month()-1], t.Year())
}

// FormatDateTime adds the 2-digit time: "15 Juli 2026 pukul 16.30".
func FormatDateTime(t time.Time) string {
	return FormatDate(t) + " pukul " + t.In(Jakarta).Format("15.04")
}

// ErrUnique is a unique violation the store translated.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }
