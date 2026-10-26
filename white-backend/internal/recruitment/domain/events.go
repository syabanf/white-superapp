package domain

import "time"

// CandidateHiredPayload is the JSON of events.RecruitmentCandidateHired,
// published when a candidate accepts an offer or HR moves them to hired. Offer fields
// are empty when HR hired the candidate without an accepted offer.
type CandidateHiredPayload struct {
	CandidateID  string  `json:"candidateId"`
	FullName     string  `json:"fullName"`
	Email        string  `json:"email"`
	Phone        string  `json:"phone"`
	JobTitle     string  `json:"jobTitle"`
	JobOpeningID *string `json:"jobOpeningId,omitempty"`
	// hr department id of the job opening.
	DepartmentID *string `json:"departmentId,omitempty"`
	// YYYY-MM-DD from the accepted offer.
	StartDate *string `json:"startDate,omitempty"`
	OfferID   *string `json:"offerId,omitempty"`
	// Decimal string ("8500000.00") from the accepted offer.
	BaseSalary *string   `json:"baseSalary,omitempty"`
	HiredAt    time.Time `json:"hiredAt"`
}

// HiredPayload builds the event from the candidate, their opening and the
// accepted offer (either may be nil).
func HiredPayload(c Candidate, opening *JobOpening, offer *Offer, hiredAt time.Time) CandidateHiredPayload {
	p := CandidateHiredPayload{CandidateID: c.ID, FullName: c.FullName, Email: c.Email, Phone: c.Phone,
		JobTitle: c.JobOpeningTitle, JobOpeningID: c.JobOpeningID, HiredAt: hiredAt}
	if opening != nil {
		p.JobTitle, p.DepartmentID = opening.Title, opening.DepartmentID
	}
	if offer != nil {
		if offer.PositionTitle != "" {
			p.JobTitle = offer.PositionTitle
		}
		salary := offer.BaseSalary.String()
		p.StartDate, p.OfferID, p.BaseSalary = offer.StartDate, &offer.ID, &salary
	}
	return p
}
