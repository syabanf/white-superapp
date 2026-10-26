package domain

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// JobOpeningStatuses are the allowed job opening statuses.
var JobOpeningStatuses = []string{"draft", "published", "closed"}

var (
	slugInvalid = regexp.MustCompile(`[^a-z0-9]+`)
	slugEdges   = regexp.MustCompile(`^-+|-+$`)
)

// Slugify lowercases, turns every non [a-z0-9] run into "-" and trims dashes.
func Slugify(value string) string {
	s := slugInvalid.ReplaceAllString(strings.TrimSpace(strings.ToLower(value)), "-")
	return slugEdges.ReplaceAllString(s, "")
}

// Defaults of normalizeJobOpening.
const (
	DefaultDepartment     = "Operations"
	DefaultLocation       = "Jakarta, ID"
	DefaultEmploymentType = "Full-time"
	DefaultWorkMode       = "On-site"
)

// JobOpeningInput is a create or update as HR submits it.
type JobOpeningInput struct {
	Title               string
	Slug                string
	DepartmentID        *string
	Department          string
	Location            string
	EmploymentType      string
	WorkMode            string
	Headcount           int
	Description         string
	Requirements        string
	Benefits            string
	SalaryMin           string
	SalaryMax           string
	HiringManagerUserID *string
	Status              string
	ClosingDate         *string
}

// JobOpening is a stored opening.
type JobOpening struct {
	ID                  string
	Title               string
	Slug                string
	DepartmentID        *string
	Department          string
	Location            string
	EmploymentType      string
	WorkMode            string
	Headcount           int
	Description         string
	Requirements        string
	Benefits            string
	SalaryMin           *Money
	SalaryMax           *Money
	HiringManagerUserID *string
	Status              string
	PublishedAt         *time.Time
	ClosingDate         *string
	CandidateCount      int
	HiredCount          int
	CreatedBy           *string
	UpdatedBy           *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func emptyToNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	t := strings.TrimSpace(*s)
	return &t
}

// NormalizeJobOpening is normalizeJobOpening: trimmed texts, the slug from
// the title, the NüHabit defaults, headcount 1 and status draft. A newly
// published opening gets published_at now; an update keeps the first
// publication time.
func NormalizeJobOpening(in JobOpeningInput, prev *JobOpening, now time.Time) (JobOpening, error) {
	title := strings.TrimSpace(in.Title)
	o := JobOpening{
		Title:               title,
		Slug:                strings.TrimSpace(orDefault(in.Slug, Slugify(title))),
		DepartmentID:        emptyToNil(in.DepartmentID),
		Department:          orDefault(strings.TrimSpace(in.Department), DefaultDepartment),
		Location:            orDefault(strings.TrimSpace(in.Location), DefaultLocation),
		EmploymentType:      orDefault(strings.TrimSpace(in.EmploymentType), DefaultEmploymentType),
		WorkMode:            orDefault(strings.TrimSpace(in.WorkMode), DefaultWorkMode),
		Headcount:           in.Headcount,
		Description:         strings.TrimSpace(in.Description),
		Requirements:        strings.TrimSpace(in.Requirements),
		Benefits:            strings.TrimSpace(in.Benefits),
		HiringManagerUserID: emptyToNil(in.HiringManagerUserID),
		Status:              orDefault(in.Status, "draft"),
		ClosingDate:         emptyToNil(in.ClosingDate),
	}
	if o.Headcount == 0 {
		o.Headcount = 1
	}
	v := validate.New()
	if o.Title == "" {
		v.Add("title", "Judul lowongan wajib diisi")
	}
	if o.Slug == "" {
		v.Add("slug", "Slug lowongan wajib diisi")
	}
	if !slices.Contains(JobOpeningStatuses, o.Status) {
		v.Add("status", "Status lowongan tidak valid")
	}
	if o.Headcount < 1 {
		v.Add("headcount", "Jumlah kebutuhan minimal 1")
	}
	if o.ClosingDate != nil {
		if _, ok := ParseDay(*o.ClosingDate); !ok {
			v.Add("closing_date", "Tanggal tidak valid")
		}
	}
	for _, s := range []struct {
		field, raw string
		dst        **Money
	}{{"salary_min", in.SalaryMin, &o.SalaryMin}, {"salary_max", in.SalaryMax, &o.SalaryMax}} {
		if strings.TrimSpace(s.raw) == "" {
			continue
		}
		m, err := ParseMoney(s.raw)
		if err != nil || m < 0 {
			v.Add(s.field, "Nominal gaji tidak valid")
			continue
		}
		*s.dst = &m
	}
	if o.SalaryMin != nil && o.SalaryMax != nil && *o.SalaryMin > *o.SalaryMax {
		v.Add("salary_max", "Gaji minimum tidak boleh melebihi gaji maksimum")
	}
	if prev != nil {
		o.PublishedAt = prev.PublishedAt
	}
	if o.Status == "published" && o.PublishedAt == nil {
		o.PublishedAt = &now
	}
	return o, v.Err()
}

// IsOpenForApplications reports a published opening whose closing date
// (a WIB day) has not passed.
func (o JobOpening) IsOpenForApplications(now time.Time) bool {
	if o.Status != "published" {
		return false
	}
	return o.ClosingDate == nil || *o.ClosingDate >= DayKey(now)
}
