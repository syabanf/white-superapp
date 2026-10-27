package domain

import (
	"fmt"
	"math"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Scheduled interviews with a human interviewer (recruitment.interviews in
// NüHabit: type hrd | hiring_manager, mode offline | online, a scorecard
// jsonb and a recommendation proceed | pool | reject). NüHabit kept the
// table without routes; the scheduling flow and scorecard rules are Mixist's.

var (
	InterviewTypes           = []string{"hrd", "hiring_manager"}
	InterviewModes           = []string{"offline", "online"}
	InterviewRecommendations = []string{"proceed", "pool", "reject"}
)

var interviewTypeLabels = map[string]string{"hrd": "HRD", "hiring_manager": "Hiring Manager"}

var interviewRecLabels = map[string]string{"proceed": "Lanjut", "pool": "Talent Pool", "reject": "Tolak"}

// InterviewTypeLabel is the label of an interview type.
func InterviewTypeLabel(t string) string {
	if l, ok := interviewTypeLabels[t]; ok {
		return l
	}
	return t
}

// InterviewRecommendationLabel is the label of a scorecard verdict.
func InterviewRecommendationLabel(r string) string {
	if l, ok := interviewRecLabels[r]; ok {
		return l
	}
	return r
}

// DefaultScorecardCriteria are suggested to the interviewer.
var DefaultScorecardCriteria = []string{"Komunikasi", "Keahlian teknis", "Pengalaman relevan", "Kecocokan budaya kerja", "Motivasi"}

// Interview is a scheduled interview.
type Interview struct {
	ID                    string
	CandidateID           string
	CandidateName         string
	JobOpeningTitle       string
	InterviewerEmployeeID *string
	InterviewerUserID     *string
	InterviewerName       string
	InterviewDate         time.Time
	DurationMinutes       int
	Type                  string
	Mode                  string
	MeetingLink           string
	Location              string
	Status                string
	Scorecard             []ScoreItem
	OverallScore          *float64
	Recommendation        string
	Notes                 string
	CancelReason          string
	CreatedBy             *string
	UpdatedBy             *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ScoreItem is one scorecard criterion.
type ScoreItem struct {
	Criterion string `json:"criterion"`
	Score     int    `json:"score"`
	Note      string `json:"note,omitempty"`
}

// InterviewInput is a schedule or reschedule request.
type InterviewInput struct {
	CandidateID           string
	InterviewerEmployeeID *string
	InterviewerName       string
	InterviewDate         time.Time
	DurationMinutes       int
	Type                  string
	Mode                  string
	MeetingLink           string
	Location              string
	Notes                 string
}

// Normalize trims the input, applies the defaults (60 minutes, offline)
// and checks it. A new interview must not be in the past.
func (in *InterviewInput) Normalize(now time.Time, isNew bool) error {
	in.InterviewerName = strings.TrimSpace(in.InterviewerName)
	in.MeetingLink = strings.TrimSpace(in.MeetingLink)
	in.Location = strings.TrimSpace(in.Location)
	in.Notes = strings.TrimSpace(in.Notes)
	in.InterviewerEmployeeID = emptyToNil(in.InterviewerEmployeeID)
	if in.DurationMinutes == 0 {
		in.DurationMinutes = 60
	}
	if in.Mode == "" {
		in.Mode = "offline"
	}
	v := validate.New()
	if in.CandidateID == "" {
		v.Add("candidate_id", "Kandidat wajib dipilih")
	}
	if in.InterviewDate.IsZero() {
		v.Add("interview_date", "Jadwal interview wajib diisi")
	} else if isNew && in.InterviewDate.Before(now.Add(-time.Hour)) {
		v.Add("interview_date", "Jadwal interview tidak boleh di masa lalu")
	}
	if in.DurationMinutes < 15 || in.DurationMinutes > 480 {
		v.Add("duration_minutes", "Durasi interview 15 sampai 480 menit")
	}
	if !slices.Contains(InterviewTypes, in.Type) {
		v.Add("type", "Tipe interview tidak valid")
	}
	if !slices.Contains(InterviewModes, in.Mode) {
		v.Add("mode", "Mode interview tidak valid")
	}
	if in.Mode == "online" {
		if in.MeetingLink == "" {
			v.Add("meeting_link", "Link meeting wajib diisi untuk interview online")
		} else if u, err := url.Parse(in.MeetingLink); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			v.Add("meeting_link", "Link meeting tidak valid")
		}
	}
	if in.InterviewerEmployeeID == nil && in.InterviewerName == "" {
		v.Add("interviewer_name", "Interviewer wajib dipilih")
	}
	if UTF16Len(in.Notes) > 2000 {
		v.Add("notes", "Catatan maksimal 2000 karakter")
	}
	return v.Err()
}

// ScheduledDescription is the activity of a scheduled interview.
func ScheduledDescription(typ string, at time.Time, rescheduled bool) string {
	verb := "dijadwalkan"
	if rescheduled {
		verb = "dijadwalkan ulang ke"
	}
	return fmt.Sprintf("Interview %s %s %s", InterviewTypeLabel(typ), verb, FormatDateTime(at))
}

// ValidateScorecard checks 1..20 criteria scored 1..5 and the verdict, and
// returns the average score rounded to two decimals.
func ValidateScorecard(items []ScoreItem, recommendation, notes string) ([]ScoreItem, float64, error) {
	v := validate.New()
	clean := make([]ScoreItem, 0, len(items))
	sum := 0
	for _, it := range items {
		it.Criterion = strings.TrimSpace(it.Criterion)
		it.Note = strings.TrimSpace(it.Note)
		if it.Criterion == "" {
			v.Add("scorecard", "Nama kriteria wajib diisi")
			continue
		}
		if it.Score < 1 || it.Score > 5 {
			v.Add("scorecard", "Nilai kriteria harus 1 sampai 5")
			continue
		}
		if UTF16Len(it.Note) > 500 {
			v.Add("scorecard", "Catatan kriteria maksimal 500 karakter")
		}
		sum += it.Score
		clean = append(clean, it)
	}
	switch {
	case len(items) == 0:
		v.Add("scorecard", "Isi minimal satu kriteria penilaian")
	case len(items) > 20:
		v.Add("scorecard", "Maksimal 20 kriteria penilaian")
	}
	if !slices.Contains(InterviewRecommendations, recommendation) {
		v.Add("recommendation", "Rekomendasi wajib dipilih")
	}
	if UTF16Len(strings.TrimSpace(notes)) > 2000 {
		v.Add("notes", "Catatan maksimal 2000 karakter")
	}
	if err := v.Err(); err != nil {
		return nil, 0, err
	}
	avg := math.Round(float64(sum)/float64(len(clean))*100) / 100
	return clean, avg, nil
}

// ScorecardDescription is the activity of a submitted scorecard.
func ScorecardDescription(typ string, overall float64, recommendation string) string {
	return fmt.Sprintf("Scorecard interview %s disimpan (nilai %.2f; rekomendasi: %s)",
		InterviewTypeLabel(typ), overall, InterviewRecommendationLabel(recommendation))
}
