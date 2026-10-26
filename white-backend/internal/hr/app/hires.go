package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/hris"
	"github.com/syabanf/white-superapp/white-backend/internal/hr/domain/payroll"
)

// CandidateHired is the JSON of white.recruitment.candidate.hired, copied
// from recruitment's domain.CandidateHiredPayload (services do not import
// each other's internals).
type CandidateHired struct {
	CandidateID  string    `json:"candidateId"`
	FullName     string    `json:"fullName"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	JobTitle     string    `json:"jobTitle"`
	JobOpeningID *string   `json:"jobOpeningId,omitempty"`
	DepartmentID *string   `json:"departmentId,omitempty"`
	StartDate    *string   `json:"startDate,omitempty"`
	OfferID      *string   `json:"offerId,omitempty"`
	BaseSalary   *string   `json:"baseSalary,omitempty"`
	HiredAt      time.Time `json:"hiredAt"`
}

// NotifNewHire tells HR a hired candidate is waiting as a new employee.
const NotifNewHire = "HR_NEW_HIRE"

var errHireSeen = errors.New("candidate already hired")

// OnCandidateHired creates the hired candidate's employee record in
// probation from the start date, applies the onboarding template and tells
// HR to complete the profile, contract and salary (salaries have no draft
// state, so none is created). A candidate is processed once: redeliveries
// and payloads without a candidate or name are acked without changes.
func (l *Lifecycle) OnCandidateHired(ctx context.Context, h CandidateHired) error {
	h.CandidateID, h.FullName = strings.TrimSpace(h.CandidateID), strings.TrimSpace(h.FullName)
	if h.CandidateID == "" || h.FullName == "" {
		l.d.Logger.Warn("candidate hired without id or name, skipped", "candidate", h.CandidateID)
		return nil
	}
	joinAt := h.HiredAt
	if joinAt.IsZero() {
		joinAt = l.d.now()
	}
	join := hris.TodayWIB(joinAt)
	if h.StartDate != nil {
		if _, ok := hris.ParseDate(*h.StartDate); ok {
			join = *h.StartDate
		}
	}
	e := domain.Employee{ID: newID(), FullName: h.FullName, Email: strings.ToLower(strings.TrimSpace(h.Email)), Phone: strings.TrimSpace(h.Phone),
		EmploymentStatus: "probation", IsActive: true, JoinDate: join}
	if h.DepartmentID != nil {
		d, err := l.d.Store.GetDepartment(ctx, *h.DepartmentID)
		if err != nil {
			return err
		}
		if d != nil {
			e.DepartmentID = &d.ID
		}
	}
	positionID, err := l.positionFor(ctx, deref(e.DepartmentID), h.JobTitle)
	if err != nil {
		return err
	}
	e.PositionID = positionID
	if e.EmployeeNo, err = (&People{d: l.d}).nextEmployeeNo(ctx, join); err != nil {
		return err
	}
	err = l.d.Store.WithTx(ctx, func(tx domain.Store) error {
		// Claim first: a redelivery stops here before any unique check.
		claimed, err := tx.ClaimRecruitmentHire(ctx, h.CandidateID, e.ID)
		if err != nil {
			return err
		}
		if !claimed {
			return errHireSeen
		}
		if err := tx.InsertEmployee(ctx, e); err != nil {
			return err
		}
		return applyOnboardingTemplate(ctx, tx, e, nil)
	})
	var unique *domain.ErrUnique
	switch {
	case errors.Is(err, errHireSeen):
		return nil
	case errors.As(err, &unique) && strings.Contains(unique.Constraint, "email"):
		// Another employee owns the email (a rehire): HR links them by hand.
		notifyHR(ctx, l.d, NotifyInput{Type: NotifNewHire, Title: "Kandidat diterima: " + h.FullName,
			Body: "Data karyawan tidak dibuat otomatis karena email " + e.Email + " sudah dipakai karyawan lain.", Href: "/erp/hr/employees"})
		return nil
	case err != nil:
		// Includes a concurrent employee number clash: the retry succeeds.
		return err
	}
	body := "Mulai " + join + " (masa percobaan). Lengkapi data, kontrak dan gaji karyawan."
	if h.BaseSalary != nil {
		if v, ok := domain.ParseMoney(*h.BaseSalary); ok && v > 0 {
			body += " Gaji penawaran " + payroll.FormatRupiah(v) + "."
		}
	}
	notifyHR(ctx, l.d, NotifyInput{Type: NotifNewHire, Title: "Karyawan baru dari rekrutmen: " + h.FullName, Body: body, Href: "/erp/hr/employees/" + e.ID})
	return nil
}

// positionFor is the active position titled like the job, within the
// department when there is one.
func (l *Lifecycle) positionFor(ctx context.Context, departmentID, jobTitle string) (*string, error) {
	jobTitle = strings.TrimSpace(jobTitle)
	if jobTitle == "" {
		return nil, nil
	}
	positions, err := l.d.Store.ListPositions(ctx, departmentID, false)
	if err != nil {
		return nil, err
	}
	for _, p := range positions {
		if strings.EqualFold(strings.TrimSpace(p.Title), jobTitle) {
			return &p.ID, nil
		}
	}
	return nil, nil
}
