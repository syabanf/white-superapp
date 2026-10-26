package app

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Offers manages offer versions, manual responses and the offer letter.
type Offers struct{ core }

func NewOffers(d Deps) *Offers { return &Offers{core{d}} }

const msgOfferNotFound = "Offer tidak ditemukan"

// Panel is the HR offer panel: salary references and every version.
func (o *Offers) Panel(ctx context.Context, candidateID string) (*domain.SalaryReference, []domain.Offer, error) {
	a, err := staff(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	cand, err := o.requireCandidate(ctx, candidateID)
	if err != nil {
		return nil, nil, err
	}
	ref := &domain.SalaryReference{ExpectedSalary: cand.ExpectedSalary, PositionTitle: cand.JobOpeningTitle}
	if cand.JobOpeningID != nil {
		opening, err := o.Store.GetJobOpening(ctx, *cand.JobOpeningID)
		if err != nil {
			return nil, nil, err
		}
		if opening != nil {
			ref.SalaryMin, ref.SalaryMax = opening.SalaryMin, opening.SalaryMax
		}
	}
	screening, err := o.Store.GetScreening(ctx, candidateID)
	if err != nil {
		return nil, nil, err
	}
	if screening != nil {
		ref.ConfirmedSalary = screening.ConfirmedSalary
	}
	sessions, err := o.Store.ListAiSessions(ctx, candidateID)
	if err != nil {
		return nil, nil, err
	}
	ref.InterviewExpectation = latestSalaryExpectation(sessions)
	offers, err := o.Store.ListOffers(ctx, candidateID)
	if err != nil {
		return nil, nil, err
	}
	if !a.CanWrite {
		for i := range offers {
			offers[i].Token = ""
		}
	}
	return ref, offers, nil
}

// latestSalaryExpectation is ai_summary -> ekspektasi_gaji of the latest
// completed interview: the value, else the note.
func latestSalaryExpectation(sessions []domain.AiSession) string {
	var latest *domain.AiSession
	for i := range sessions {
		s := &sessions[i]
		if s.Status != "completed" || s.Summary == nil || s.CompletedAt == nil {
			continue
		}
		if latest == nil || s.CompletedAt.After(*latest.CompletedAt) {
			latest = s
		}
	}
	if latest == nil || !latest.Summary.Salary.Mentioned {
		return ""
	}
	if v := latest.Summary.Salary.Value; v != nil {
		return *v
	}
	return latest.Summary.Salary.Note
}

// Create writes a new offer version (open ones expire) and its activity.
func (o *Offers) Create(ctx context.Context, candidateID string, in domain.OfferInput) (*domain.Offer, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	salary, err := in.Validate()
	if err != nil {
		return nil, err
	}
	cand, err := o.requireCandidate(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	title := in.PositionTitle
	if title == "" {
		title = cand.JobOpeningTitle
	}
	now := o.now()
	offer := domain.Offer{ID: newID(), CandidateID: candidateID, Token: newPortalToken(), Status: "sent", PositionTitle: title,
		BaseSalary: salary, Benefits: in.Benefits, StartDate: in.StartDate, Notes: in.Notes, SentAt: &now,
		ExpiresAt: timePtr(now.Add(time.Duration(in.ExpiresDays) * 24 * time.Hour)), CreatedBy: a.id(), CreatedByName: a.Name}
	err = o.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.ExpireOpenOffers(ctx, candidateID); err != nil {
			return err
		}
		v, err := tx.NextOfferVersion(ctx, candidateID)
		if err != nil {
			return err
		}
		offer.Version = v
		if err := tx.InsertOffer(ctx, offer); err != nil {
			return err
		}
		return o.staffActivity(ctx, tx, candidateID, "offer_sent", domain.OfferCreatedDescription(v, in.ExpiresDays), a)
	})
	if err != nil {
		return nil, err
	}
	return o.Store.GetOffer(ctx, offer.ID)
}

// RecordResponse stores a response HR took by phone or WhatsApp.
func (o *Offers) RecordResponse(ctx context.Context, id, status, note string) (*domain.Offer, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(domain.ManualOfferStatuses, status) {
		return nil, apperr.Field("status", "Status tidak valid")
	}
	note = strings.TrimSpace(note)
	if domain.UTF16Len(note) > 2000 {
		return nil, apperr.Field("note", "Catatan maksimal 2000 karakter")
	}
	offer, err := o.Store.GetOffer(ctx, id)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return nil, notFound(msgOfferNotFound)
	}
	if domain.IsOfferFinal(offer.Status) {
		return nil, conflict(domain.MsgOfferFinal)
	}
	var notePtr *string
	if note != "" {
		notePtr = &note
	}
	err = o.Store.WithTx(ctx, func(tx domain.Store) error {
		ok, err := tx.RecordManualResponse(ctx, id, status, notePtr)
		if err != nil {
			return err
		}
		if !ok {
			return conflict(domain.MsgOfferFinal)
		}
		if err := o.staffActivity(ctx, tx, offer.CandidateID, "offer_response",
			domain.ManualOfferResponseDescription(status, offer.Version, note), a); err != nil {
			return err
		}
		if status == "accepted" {
			return o.hireOnAccept(ctx, tx, offer.CandidateID, a.id(), a.Name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	o.afterOfferResponse(ctx, *offer, status, note)
	return o.Store.GetOffer(ctx, id)
}

// hireOnAccept moves the candidate to hired inside the response
// transaction; by nil and name "Sistem" for a portal acceptance.
func (c core) hireOnAccept(ctx context.Context, tx domain.Store, candidateID string, by *string, name string) error {
	cand, err := tx.GetCandidate(ctx, candidateID)
	if err != nil || cand == nil || cand.Status == "hired" {
		return err
	}
	if _, err := tx.SetCandidateStatus(ctx, candidateID, "hired", by); err != nil {
		return err
	}
	return c.activity(ctx, tx, candidateID, "status_change", domain.StageChangeDescription(cand.Status, "hired"), by, name)
}

// afterOfferResponse notifies recruiters, the offer author and the hiring
// manager, and publishes the hire on acceptance.
func (c core) afterOfferResponse(ctx context.Context, offer domain.Offer, status, note string) {
	cand, err := c.Store.GetCandidate(ctx, offer.CandidateID)
	if err != nil || cand == nil {
		return
	}
	body := cand.FullName + " " + domain.OfferResponseVerb(status) + " offer v" + itoa(offer.Version) + "."
	if note != "" {
		body += ` Catatan: "` + domain.JSSlice(note, 200) + `"`
	}
	c.notify(ctx, Notice{UserIDs: c.recipients(ctx, offer.CreatedBy, c.hiringManager(ctx, cand)), Type: NotifOfferResponse,
		Title: "Respons offer: " + domain.OfferStatusLabel(status), Body: body, Href: candidateHref(cand.ID)})
	if status == "accepted" {
		c.afterStatusChange(ctx, cand.ID, "hired")
	}
}

// DownloadLetter renders the offer letter PDF of a version.
func (o *Offers) DownloadLetter(ctx context.Context, id string) (*File, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	offer, err := o.Store.GetOffer(ctx, id)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return nil, notFound(msgOfferNotFound)
	}
	return o.letter(ctx, *offer)
}

func (c core) letter(ctx context.Context, offer domain.Offer) (*File, error) {
	l := domain.OfferLetter{Offer: offer, CompanyName: c.company(), IssuedAt: c.now()}
	if offer.SentAt != nil {
		l.IssuedAt = *offer.SentAt
	}
	if cand, err := c.Store.GetCandidate(ctx, offer.CandidateID); err == nil && cand != nil && cand.JobOpeningID != nil {
		if opening, err := c.Store.GetJobOpening(ctx, *cand.JobOpeningID); err == nil && opening != nil {
			l.Department, l.Location = opening.Department, opening.Location
		}
	}
	pdf, err := c.PDF.OfferLetter(l)
	if err != nil {
		return nil, err
	}
	name := "surat-penawaran-" + fileSlug(offer.CandidateName, "kandidat") + "-v" + itoa(offer.Version) + ".pdf"
	return &File{Name: name, ContentType: domain.MimePDF, Data: pdf}, nil
}
