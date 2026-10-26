package domain

import (
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// ─── psikotes sessions ───────────────────────────────────────

// PsikotesSession is a psikotes invitation with its tests.
type PsikotesSession struct {
	ID            string
	CandidateID   string
	CandidateName string
	PositionTitle string
	Token         string
	Status        string
	WebcamConsent *bool
	InvitedAt     *time.Time
	ExpiresAt     *time.Time
	StartedAt     *time.Time
	CompletedAt   *time.Time
	CreatedBy     *string
	CreatedByName string
	Tests         []SessionTest
	Proctor       ProctorTally
}

// SessionTest is one instrument of a session, with its instrument config.
type SessionTest struct {
	ID                    string
	SessionID             string
	CandidateID           string
	InstrumentID          string
	InstrumentCode        string
	InstrumentName        string
	InstrumentKind        string
	DurationSeconds       int
	QuestionCount         *int
	Shuffle               bool
	Instructions          string
	Status                string
	QuestionIDs           []string
	Answers               map[string]string
	Score                 *float64
	ScoreDetail           *ScoreDetail
	AttachmentKey         *string
	AttachmentContentType string
	ReviewNotes           string
	ReviewedByName        string
	SortOrder             int
	StartedAt             *time.Time
	CompletedAt           *time.Time
}

// Deadline is when the test time runs out (nil before start).
func (t SessionTest) Deadline() *time.Time { return TestDeadline(t.StartedAt, t.DurationSeconds) }

// ProctorEvent is one proctoring flag.
type ProctorEvent struct {
	ID          string
	SessionType string
	SessionID   string
	EventType   string
	Meta        map[string]string
	CreatedAt   time.Time
}

// ─── offers ──────────────────────────────────────────────────

// Offer is one version of a job offer.
type Offer struct {
	ID             string
	CandidateID    string
	CandidateName  string
	CandidateEmail string
	Version        int
	Token          string
	Status         string
	PositionTitle  string
	BaseSalary     Money
	Benefits       []string
	StartDate      *string
	Notes          string
	ResponseNote   string
	RespondedAt    *time.Time
	ResponseIP     *string
	ResponseSource string
	SentAt         *time.Time
	ExpiresAt      *time.Time
	CreatedBy      *string
	CreatedByName  string
	CreatedAt      time.Time
}

// OfferInput is parseOffer.
type OfferInput struct {
	BaseSalary    string
	Benefits      []string
	StartDate     *string
	Notes         string
	ExpiresDays   int
	PositionTitle string
}

// Validate applies offerSchema and returns the parsed salary.
func (in *OfferInput) Validate() (Money, error) {
	v := validate.New()
	salary, err := ParseMoney(in.BaseSalary)
	switch {
	case strings.TrimSpace(in.BaseSalary) == "":
		v.Add("base_salary", "Gaji pokok wajib diisi")
	case err != nil || salary < 0:
		v.Add("base_salary", "Gaji pokok tidak valid")
	case salary > Rupiah(MaxOfferSalary):
		v.Add("base_salary", "Nominal gaji tidak wajar")
	}
	benefits := make([]string, 0, len(in.Benefits))
	for _, b := range in.Benefits {
		b = strings.TrimSpace(b)
		if b == "" {
			v.Add("benefits", "Benefit tidak boleh kosong")
			continue
		}
		if UTF16Len(b) > MaxBenefitChars {
			v.Add("benefits", "Benefit maksimal 120 karakter")
		}
		benefits = append(benefits, b)
	}
	if len(in.Benefits) > MaxOfferBenefits {
		v.Add("benefits", "Maksimal 15 benefit")
	}
	in.Benefits = benefits
	in.StartDate = emptyToNil(in.StartDate)
	if in.StartDate != nil {
		if _, ok := ParseDay(*in.StartDate); !ok {
			v.Add("start_date", "Format tanggal harus YYYY-MM-DD")
		}
	}
	in.Notes = strings.TrimSpace(in.Notes)
	if UTF16Len(in.Notes) > 2000 {
		v.Add("notes", "Catatan maksimal 2000 karakter")
	}
	in.PositionTitle = strings.TrimSpace(in.PositionTitle)
	validateExpiresDays(v, &in.ExpiresDays, DefaultOfferExpires)
	return salary, v.Err()
}

// ValidateOfferResponse is offerRespondSchema plus the negotiation rule.
func ValidateOfferResponse(action, note string) (string, *string, error) {
	if _, ok := OfferStatusForAction[action]; !ok {
		return "", nil, apperr.Field("action", "Aksi tidak valid")
	}
	note = strings.TrimSpace(note)
	if UTF16Len(note) > 2000 {
		return "", nil, apperr.Field("note", "Catatan maksimal 2000 karakter")
	}
	var notePtr *string
	if note != "" {
		notePtr = &note
	}
	if action == "negotiate" && notePtr == nil {
		return "", nil, apperr.Field("note", MsgNegotiateNoteNeed)
	}
	return OfferStatusForAction[action], notePtr, nil
}

// SalaryReference is the offer panel's salary context.
type SalaryReference struct {
	ExpectedSalary       *int64
	ConfirmedSalary      *int64
	InterviewExpectation string
	PositionTitle        string
	SalaryMin            *Money
	SalaryMax            *Money
}

// ─── AI analysis ─────────────────────────────────────────────

// CandidateAnalysis is the AI match of a candidate against the opening.
type CandidateAnalysis struct {
	CandidateID string
	Summary     string
	MatchScore  int
	MatchReason string
	Strengths   []string
	Gaps        []string
	JobContext  string
	Model       string
	UpdatedAt   time.Time
}

// AnalysisInput is what the analysis prompt sees.
type AnalysisInput struct {
	CandidateName string
	Profile       string
	JobContext    string
}

// AnalysisPrompt is cvAnalysisPrompt adapted to the candidate profile (this
// port does not extract CV text).
const AnalysisPrompt = `Kamu adalah asisten HR yang menganalisis profil kandidat.
Balas HANYA dengan JSON valid (tanpa markdown code fence) berbentuk:
{
  "ringkasan": string,
  "skor_kecocokan": number,
  "alasan_kecocokan": string,
  "kekuatan": [string],
  "kekurangan": [string]
}
Ketentuan:
- "ringkasan": 2-4 kalimat profil kandidat dalam bahasa Indonesia.
- "skor_kecocokan": 0-100, kecocokan profil terhadap deskripsi pekerjaan yang diberikan. Jika deskripsi pekerjaan tidak tersedia, nilai berdasarkan kualitas & kelengkapan profil dan sebutkan itu di alasan.
- "alasan_kecocokan": 1-3 kalimat alasan skor.
- "kekuatan" dan "kekurangan": masing-masing maksimal 5 butir singkat.`

// JobContext is formatJobContext for a job opening ("" without one).
func JobContext(o *JobOpening) string {
	if o == nil {
		return ""
	}
	lines := []string{"Posisi: " + o.Title}
	if o.Description != "" {
		lines = append(lines, "Deskripsi: "+o.Description)
	}
	if o.Requirements != "" {
		lines = append(lines, "Persyaratan: "+o.Requirements)
	}
	return strings.Join(lines, "\n")
}

// CandidateProfile renders the profile the analysis reads.
func CandidateProfile(c Candidate) string {
	lines := []string{"Nama: " + c.FullName, "Domisili: " + c.Domicile}
	if c.LastEducation != "" {
		lines = append(lines, "Pendidikan terakhir: "+c.LastEducation)
	}
	if c.LastExperience != "" {
		lines = append(lines, "Pengalaman terakhir: "+c.LastExperience)
	}
	if c.Notes != "" {
		lines = append(lines, "Catatan: "+c.Notes)
	}
	if c.ExpectedSalary != nil {
		lines = append(lines, "Ekspektasi gaji: "+Rupiah(*c.ExpectedSalary).FormatIDR())
	}
	return strings.Join(lines, "\n")
}

// ─── reports ─────────────────────────────────────────────────

// PipelineReport is everything the pipeline report PDF prints.
type PipelineReport struct {
	Candidate       Candidate
	Analysis        *CandidateAnalysis
	Screening       *Screening
	PsikotesTests   []SessionTest
	PsikotesSummary *PsikotesSummary
	AiInterviews    []AiSession
	Interviews      []Interview
	Offers          []Offer
	Activities      []Activity
	CompanyName     string
}

// OfferLetter is what the offer letter PDF prints.
type OfferLetter struct {
	Offer       Offer
	CompanyName string
	Department  string
	Location    string
	IssuedAt    time.Time
}

// ─── summary ─────────────────────────────────────────────────

// Count is a labelled count.
type Count struct {
	Key   string
	Label string
	Count int
}

// Summary is the recruitment dashboard.
type Summary struct {
	OpenJobs         int
	ActiveCandidates int
	AppliedLast30    int
	HiredLast90      int
	InterviewsNext7  int
	OpenOffers       int
	ByStatus         []Count
	BySource         []Count
	AvgDaysToHire    *float64
}

// ─── hr directory ────────────────────────────────────────────

// Employee is the slice of an hr employee this service reads.
type Employee struct {
	ID             string
	UserID         *string
	Name           string
	DepartmentID   *string
	DepartmentName string
	Active         bool
}

// Department is an hr department.
type Department struct {
	ID     string
	Code   string
	Name   string
	Active bool
}
