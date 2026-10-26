package domain

import (
	"bytes"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Candidate is a stored candidate with its job opening title.
type Candidate struct {
	ID              string
	FullName        string
	Email           string
	Phone           string
	Domicile        string
	Source          string
	JobOpeningID    *string
	JobOpeningTitle string
	Status          string
	Notes           string
	LastExperience  string
	LastEducation   string
	Availability    string
	ExpectedSalary  *int64
	CVKey           *string
	CVFileName      string
	CVContentType   string
	HiredAt         *time.Time
	HiredEventAt    *time.Time
	CreatedBy       *string
	UpdatedBy       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CandidateFields is the editable profile (candidateCreateSchema).
type CandidateFields struct {
	FullName       string
	Email          string
	Phone          string
	Domicile       string
	Source         string
	JobOpeningID   *string
	Status         string
	Notes          string
	LastExperience string
	LastEducation  string
	Availability   string
	ExpectedSalary *int64
}

// Fields returns the editable profile of a stored candidate.
func (c Candidate) Fields() CandidateFields {
	return CandidateFields{FullName: c.FullName, Email: c.Email, Phone: c.Phone, Domicile: c.Domicile, Source: c.Source,
		JobOpeningID: c.JobOpeningID, Status: c.Status, Notes: c.Notes, LastExperience: c.LastExperience,
		LastEducation: c.LastEducation, Availability: c.Availability, ExpectedSalary: c.ExpectedSalary}
}

// CandidatePatch carries only the fields an update sends.
type CandidatePatch struct {
	FullName, Email, Phone, Domicile, Source, JobOpeningID *string
	Notes, LastExperience, LastEducation, Availability     *string
	ExpectedSalary                                         *int64
	ClearExpectedSalary                                    bool
}

// Apply merges the patch into f.
func (p CandidatePatch) Apply(f CandidateFields) CandidateFields {
	set := func(dst *string, v *string) {
		if v != nil {
			*dst = *v
		}
	}
	set(&f.FullName, p.FullName)
	set(&f.Email, p.Email)
	set(&f.Phone, p.Phone)
	set(&f.Domicile, p.Domicile)
	set(&f.Source, p.Source)
	set(&f.Notes, p.Notes)
	set(&f.LastExperience, p.LastExperience)
	set(&f.LastEducation, p.LastEducation)
	set(&f.Availability, p.Availability)
	if p.JobOpeningID != nil {
		f.JobOpeningID = emptyToNil(p.JobOpeningID)
	}
	if p.ExpectedSalary != nil {
		f.ExpectedSalary = p.ExpectedSalary
	}
	if p.ClearExpectedSalary {
		f.ExpectedSalary = nil
	}
	return f
}

// emailPattern is zod v4's email regex minus its two lookaheads (checked in
// IsEmail): no leading dot and no "..".
var emailPattern = regexp.MustCompile(`^[A-Za-z0-9_'+\-.]*[A-Za-z0-9_+-]@([A-Za-z0-9][A-Za-z0-9\-]*\.)+[A-Za-z]{2,}$`)

// IsEmail is the staff form email rule.
func IsEmail(s string) bool {
	return !strings.HasPrefix(s, ".") && !strings.Contains(s, "..") && emailPattern.MatchString(s)
}

var phonePattern = regexp.MustCompile(`^[0-9+\-\s()]{8,20}$`)

// IsPhone is the candidate phone rule: 8 to 20 digits, spaces, + - ( ).
func IsPhone(s string) bool { return phonePattern.MatchString(s) }

// MaxExpectedSalary caps expected_salary (candidateCreateSchema).
const MaxExpectedSalary = 2_000_000_000

// NormalizeCandidate trims the profile, applies the defaults (source
// walk_in, status applied) and the schema messages. Status is only checked
// against the creatable statuses when create is true.
func NormalizeCandidate(f CandidateFields, create bool) (CandidateFields, error) {
	trim := func(s *string) { *s = strings.TrimSpace(*s) }
	for _, s := range []*string{&f.FullName, &f.Email, &f.Phone, &f.Domicile, &f.Notes, &f.LastExperience, &f.LastEducation} {
		trim(s)
	}
	f.JobOpeningID = emptyToNil(f.JobOpeningID)
	if f.Source == "" {
		f.Source = "walk_in"
	}
	v := validate.New()
	switch n := UTF16Len(f.FullName); {
	case n < 2:
		v.Add("full_name", "Nama minimal 2 karakter")
	case n > 100:
		v.Add("full_name", "Nama maksimal 100 karakter")
	}
	if !IsEmail(f.Email) {
		v.Add("email", "Email tidak valid")
	}
	if !IsPhone(f.Phone) {
		v.Add("phone", "Nomor telepon tidak valid")
	}
	switch n := UTF16Len(f.Domicile); {
	case n < 1:
		v.Add("domicile", "Domisili wajib diisi")
	case n > 100:
		v.Add("domicile", "Domisili maksimal 100 karakter")
	}
	if !slices.Contains(CandidateSources, f.Source) {
		v.Add("source", "Sumber kandidat tidak valid")
	}
	if create {
		if f.Status == "" {
			f.Status = "applied"
		}
		if !slices.Contains(CreatableStatuses, f.Status) {
			v.Add("status", "Status awal kandidat harus applied atau screening")
		}
	}
	for _, c := range []struct {
		field, value string
		max          int
		msg          string
	}{
		{"notes", f.Notes, 1000, "Catatan maksimal 1000 karakter"},
		{"last_experience", f.LastExperience, 300, "Pengalaman terakhir maksimal 300 karakter"},
		{"last_education", f.LastEducation, 300, "Pendidikan terakhir maksimal 300 karakter"},
	} {
		if UTF16Len(c.value) > c.max {
			v.Add(c.field, c.msg)
		}
	}
	if f.Availability != "" && !slices.Contains(Availabilities, f.Availability) {
		v.Add("availability", "Ketersediaan tidak valid")
	}
	if s := f.ExpectedSalary; s != nil && (*s < 0 || *s > MaxExpectedSalary) {
		v.Add("expected_salary", "Ekspektasi gaji tidak valid")
	}
	return f, v.Err()
}

// ValidateNote is parseNote: 1..2000 characters.
func ValidateNote(content string) (string, error) {
	content = strings.TrimSpace(content)
	switch {
	case content == "":
		return "", apperr.Field("content", "Catatan tidak boleh kosong")
	case UTF16Len(content) > 2000:
		return "", apperr.Field("content", "Catatan maksimal 2000 karakter")
	}
	return content, nil
}

// ─── screening ───────────────────────────────────────────────

// Screening is the HR phone screening of a candidate.
type Screening struct {
	CandidateID      string
	Contacted        bool
	Interested       *bool
	AvailabilityNote string
	ConfirmedSalary  *int64
	WillingShift     *bool
	WillingPlacement *bool
	Notes            string
	Recommendation   string
	UpdatedBy        *string
	UpdatedByName    string
	UpdatedAt        time.Time
}

// Validate is parseScreening.
func (s *Screening) Validate() error {
	v := validate.New()
	s.AvailabilityNote = strings.TrimSpace(s.AvailabilityNote)
	s.Notes = strings.TrimSpace(s.Notes)
	if UTF16Len(s.AvailabilityNote) > 200 {
		v.Add("availability_note", "Ketersediaan maksimal 200 karakter")
	}
	if c := s.ConfirmedSalary; c != nil {
		if *c < 0 {
			v.Add("confirmed_salary", "Gaji tidak boleh negatif")
		} else if *c > 1_000_000_000_000 {
			v.Add("confirmed_salary", "Nominal gaji tidak wajar")
		}
	}
	if UTF16Len(s.Notes) > 2000 {
		v.Add("notes", "Catatan maksimal 2000 karakter")
	}
	if s.Recommendation != "" && !slices.Contains(Recommendations, s.Recommendation) {
		v.Add("recommendation", "Rekomendasi tidak valid")
	}
	return v.Err()
}

// PsikotesSummary is HR's psikotes verdict on a candidate.
type PsikotesSummary struct {
	CandidateID    string
	Recommendation string
	Notes          string
	UpdatedBy      *string
	UpdatedByName  string
	UpdatedAt      time.Time
}

// Validate is parsePsikotesSummary.
func (s *PsikotesSummary) Validate() error {
	v := validate.New()
	s.Notes = strings.TrimSpace(s.Notes)
	if s.Recommendation != "" && !slices.Contains(Recommendations, s.Recommendation) {
		v.Add("recommendation", "Rekomendasi tidak valid")
	}
	if UTF16Len(s.Notes) > 2000 {
		v.Add("notes", "Catatan maksimal 2000 karakter")
	}
	return v.Err()
}

// Note is an internal note on a candidate.
type Note struct {
	ID            string
	CandidateID   string
	Content       string
	CreatedBy     *string
	CreatedByName string
	CreatedAt     time.Time
}

// Activity is one line of a candidate's pipeline history.
type Activity struct {
	ID            string
	CandidateID   string
	Type          string
	Description   string
	CreatedBy     *string
	CreatedByName string
	CreatedAt     time.Time
}

// ─── portal applications ─────────────────────────────────────

// Application is the public career form.
type Application struct {
	FullName, Email, Phone, Domicile, Source string
	LastExperience, LastEducation            string
	Availability, Notes                      string
	ExpectedSalary                           *int64
}

var applicantEmail = regexp.MustCompile(`^[^\s@<>"]+@[^\s@<>"]+\.[^\s@<>"]+$`)

// Portal messages.
const (
	MsgFieldsRequired = "Kolom wajib belum diisi"
	MsgEmailFormat    = "Format email tidak valid"
	MsgAlreadyApplied = "Anda sudah melamar lowongan ini"
	MsgJobClosed      = "Lowongan ini sudah tidak menerima lamaran"
	MsgApplied        = "Lamaran berhasil dikirim"
	MsgApplyTooMany   = "Terlalu banyak lamaran dari jaringan ini. Coba lagi beberapa menit lagi."
)

// NormalizeApplication is parseApplicationFields with the staff profile
// rules on top: a missing required field wins over a format error.
func NormalizeApplication(a Application) (CandidateFields, error) {
	for _, s := range []*string{&a.FullName, &a.Email, &a.Phone, &a.Domicile} {
		*s = strings.TrimSpace(*s)
		if *s == "" {
			return CandidateFields{}, apperr.New(apperr.Validation, MsgFieldsRequired)
		}
	}
	if UTF16Len(a.Email) > 255 || !applicantEmail.MatchString(a.Email) {
		return CandidateFields{}, apperr.Field("email", MsgEmailFormat)
	}
	if a.Source == "" {
		a.Source = "portal"
	}
	f, err := NormalizeCandidate(CandidateFields{FullName: a.FullName, Email: a.Email, Phone: a.Phone, Domicile: a.Domicile,
		Source: a.Source, Status: "applied", Notes: a.Notes, LastExperience: a.LastExperience, LastEducation: a.LastEducation,
		Availability: a.Availability, ExpectedSalary: a.ExpectedSalary}, true)
	return f, err
}

// ─── uploads ─────────────────────────────────────────────────

// MaxCVBytes is the CV limit (10 MB).
const MaxCVBytes = 10 * 1024 * 1024

// Content types this service stores.
const (
	MimePDF  = "application/pdf"
	MimeDOC  = "application/msword"
	MimeDOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MimeJPEG = "image/jpeg"
	MimePNG  = "image/png"
	MimeWEBP = "image/webp"
)

// CVExtensions are the staff upload extensions (readCvFile).
var CVExtensions = []string{"pdf", "doc", "docx", "jpg", "jpeg", "png"}

// PortalCVExtensions are what the career form accepts.
var PortalCVExtensions = []string{"pdf", "doc", "docx"}

// SniffFile detects the content type from the magic bytes ("" when unknown).
// A ZIP container counts as DOCX only when the name says so.
func SniffFile(data []byte, name string) string {
	switch {
	case bytes.HasPrefix(data, []byte("%PDF-")):
		return MimePDF
	case bytes.HasPrefix(data, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}):
		return MimeDOC
	case bytes.HasPrefix(data, []byte("PK\x03\x04")) && FileExt(name) == "docx":
		return MimeDOCX
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return MimeJPEG
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return MimePNG
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return MimeWEBP
	}
	return ""
}

// FileExt is name.split(".").pop().toLowerCase(): the whole name when it
// has no dot.
func FileExt(name string) string {
	return strings.ToLower(name[strings.LastIndex(name, ".")+1:])
}

var extMime = map[string][]string{
	"pdf": {MimePDF}, "doc": {MimeDOC}, "docx": {MimeDOCX}, "jpg": {MimeJPEG}, "jpeg": {MimeJPEG}, "png": {MimePNG}, "webp": {MimeWEBP},
}

// CheckUpload validates a file against allowed extensions, size and magic
// bytes and returns the sniffed content type. typeMsg is the extension
// error; the size message names the limit in MB.
func CheckUpload(name string, data []byte, allowed []string, maxBytes int, typeMsg, sizeMsg string) (string, error) {
	ext := FileExt(path.Base(name))
	if len(data) == 0 {
		return "", apperr.Field("file", "File tidak ditemukan")
	}
	if !slices.Contains(allowed, ext) {
		return "", apperr.Field("file", typeMsg)
	}
	if len(data) > maxBytes {
		return "", apperr.Field("file", sizeMsg)
	}
	mime := SniffFile(data, name)
	if mime == "" || !slices.Contains(extMime[ext], mime) {
		return "", apperr.Field("file", "Isi file tidak sesuai dengan tipenya")
	}
	return mime, nil
}

// CheckCV is the staff CV rule (readCvFile).
func CheckCV(name string, data []byte) (string, error) {
	return CheckUpload(name, data, CVExtensions, MaxCVBytes,
		"Tipe file tidak valid. Allowed: "+strings.Join(CVExtensions, ", "), "Ukuran file maksimal 10MB")
}

// CheckPortalCV is the career form CV rule: PDF or Word only.
func CheckPortalCV(name string, data []byte) (string, error) {
	return CheckUpload(name, data, PortalCVExtensions, MaxCVBytes, "CV harus berupa file PDF atau DOC", "Ukuran CV maksimal 10MB")
}

// CheckDrawing is the drawing upload rule (jpg, png, webp; 8 MB).
func CheckDrawing(name string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", apperr.Field("file", "File tidak ditemukan di form")
	}
	if len(data) > MaxDrawingBytes {
		return "", apperr.Field("file", "Ukuran file maksimal 8MB")
	}
	switch mime := SniffFile(data, name); mime {
	case MimeJPEG, MimePNG, MimeWEBP:
		return mime, nil
	}
	return "", apperr.Field("file", "Isi file bukan gambar JPG/PNG/WebP yang valid")
}

// SafeFileName keeps the base name and drops path and control characters.
func SafeFileName(name, fallback string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == '"' {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." || name == "/" {
		return fallback
	}
	return JSSlice(name, 120)
}
