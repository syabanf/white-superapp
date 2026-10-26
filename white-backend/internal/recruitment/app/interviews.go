package app

import (
	"context"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Interviews covers AI interview invitations and interviews scheduled with
// a human interviewer, with their scorecards.
type Interviews struct{ core }

func NewInterviews(d Deps) *Interviews { return &Interviews{core{d}} }

const msgInterviewNotFound = "Interview tidak ditemukan"

// AiPanel is the HR AI interview panel of a candidate.
func (iv *Interviews) AiPanel(ctx context.Context, candidateID string) ([]domain.AiSession, error) {
	a, err := staff(ctx, false)
	if err != nil {
		return nil, err
	}
	if _, err := iv.requireCandidate(ctx, candidateID); err != nil {
		return nil, err
	}
	sessions, err := aiSessionsWithTurns(ctx, iv.Store, candidateID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(sessions))
	for i, s := range sessions {
		ids[i] = s.ID
	}
	tallies, err := iv.Store.ProctorTallies(ctx, "interview", ids)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		sessions[i].Proctor = tallies[sessions[i].ID]
		if !a.CanWrite {
			sessions[i].Token = ""
		}
	}
	return sessions, nil
}

// InviteAi creates an AI interview invitation and its activity.
func (iv *Interviews) InviteAi(ctx context.Context, candidateID string, expiresDays, maxQuestions int) (*domain.AiSession, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateAiInvite(&expiresDays, &maxQuestions); err != nil {
		return nil, err
	}
	if _, err := iv.requireCandidate(ctx, candidateID); err != nil {
		return nil, err
	}
	now := iv.now()
	s := domain.AiSession{ID: newID(), CandidateID: candidateID, Token: newPortalToken(), Status: "sent", MaxQuestions: maxQuestions,
		InvitedAt: &now, ExpiresAt: timePtr(now.Add(time.Duration(expiresDays) * 24 * time.Hour)), CreatedBy: a.id(), CreatedByName: a.Name}
	err = iv.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertAiSession(ctx, s); err != nil {
			return err
		}
		return iv.staffActivity(ctx, tx, candidateID, "interview_ai_invited", domain.AiInviteDescription(maxQuestions, expiresDays), a)
	})
	return &s, err
}

// ListFilter narrows the interview list by WIB days.
type ListFilter struct {
	CandidateID           string
	Status                string
	InterviewerEmployeeID string
	DateFrom              string
	DateTo                string
}

func wibDayStart(day string) (time.Time, bool) {
	t, err := time.ParseInLocation(domain.IsoDate, day, domain.Jakarta)
	return t, err == nil
}

func (iv *Interviews) List(ctx context.Context, f ListFilter) ([]domain.Interview, error) {
	if _, err := staff(ctx, false); err != nil {
		return nil, err
	}
	out := domain.InterviewFilter{CandidateID: f.CandidateID, Status: f.Status, InterviewerEmployeeID: f.InterviewerEmployeeID}
	v := validate.New()
	if f.DateFrom != "" {
		t, ok := wibDayStart(f.DateFrom)
		if !ok {
			v.Add("date_from", "Tanggal tidak valid")
		}
		out.From = &t
	}
	if f.DateTo != "" {
		t, ok := wibDayStart(f.DateTo)
		if !ok {
			v.Add("date_to", "Tanggal tidak valid")
		}
		end := t.AddDate(0, 0, 1)
		out.To = &end
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	return iv.Store.ListInterviews(ctx, out)
}

func (iv *Interviews) get(ctx context.Context, id string) (*domain.Interview, error) {
	i, err := iv.Store.GetInterview(ctx, id)
	if err == nil && i == nil {
		err = notFound(msgInterviewNotFound)
	}
	return i, err
}

// applyInterviewer fills the interviewer name and login from hr.
func (iv *Interviews) applyInterviewer(ctx context.Context, i *domain.Interview, in domain.InterviewInput) error {
	i.InterviewerEmployeeID, i.InterviewerUserID, i.InterviewerName = nil, nil, in.InterviewerName
	if in.InterviewerEmployeeID == nil {
		return nil
	}
	e, err := iv.Directory.Employee(ctx, *in.InterviewerEmployeeID)
	if err != nil {
		return err
	}
	if e == nil {
		return apperr.Field("interviewer_employee_id", "Interviewer tidak ditemukan")
	}
	i.InterviewerEmployeeID, i.InterviewerUserID, i.InterviewerName = &e.ID, e.UserID, e.Name
	return nil
}

func applyInput(i *domain.Interview, in domain.InterviewInput) {
	i.InterviewDate, i.DurationMinutes, i.Type, i.Mode = in.InterviewDate, in.DurationMinutes, in.Type, in.Mode
	i.MeetingLink, i.Location, i.Notes = in.MeetingLink, in.Location, in.Notes
	if i.Mode == "offline" {
		i.MeetingLink = ""
	}
}

func (iv *Interviews) notifyInterviewer(ctx context.Context, i domain.Interview, title string) {
	iv.notify(ctx, Notice{UserIDs: iv.recipients(ctx, i.InterviewerUserID), Type: NotifInterviewSchedule, Title: title,
		Body: "Interview " + domain.InterviewTypeLabel(i.Type) + " dengan " + i.CandidateName + " pada " + domain.FormatDateTime(i.InterviewDate) + ".",
		Href: hrefInterviews})
}

// Schedule books an interview with a human interviewer.
func (iv *Interviews) Schedule(ctx context.Context, in domain.InterviewInput) (*domain.Interview, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	if err := in.Normalize(iv.now(), true); err != nil {
		return nil, err
	}
	if _, err := iv.requireCandidate(ctx, in.CandidateID); err != nil {
		return nil, err
	}
	i := domain.Interview{ID: newID(), CandidateID: in.CandidateID, Status: "scheduled", CreatedBy: a.id()}
	if err := iv.applyInterviewer(ctx, &i, in); err != nil {
		return nil, err
	}
	applyInput(&i, in)
	err = iv.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.InsertInterview(ctx, i); err != nil {
			return err
		}
		return iv.staffActivity(ctx, tx, i.CandidateID, "interview_scheduled", domain.ScheduledDescription(i.Type, i.InterviewDate, false), a)
	})
	if err != nil {
		return nil, err
	}
	out, err := iv.get(ctx, i.ID)
	if err == nil {
		iv.notifyInterviewer(ctx, *out, "Jadwal interview baru: "+out.CandidateName)
	}
	return out, err
}

func requireScheduled(i *domain.Interview) error {
	if i.Status != "scheduled" {
		return conflict("Interview yang sudah selesai atau dibatalkan tidak bisa diubah")
	}
	return nil
}

// Update reschedules or edits a scheduled interview.
func (iv *Interviews) Update(ctx context.Context, id string, in domain.InterviewInput) (*domain.Interview, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	i, err := iv.get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := requireScheduled(i); err != nil {
		return nil, err
	}
	in.CandidateID = i.CandidateID
	if err := in.Normalize(iv.now(), false); err != nil {
		return nil, err
	}
	moved := !in.InterviewDate.Equal(i.InterviewDate)
	if err := iv.applyInterviewer(ctx, i, in); err != nil {
		return nil, err
	}
	applyInput(i, in)
	i.UpdatedBy = a.id()
	err = iv.Store.WithTx(ctx, func(tx domain.Store) error {
		if _, err := tx.UpdateInterview(ctx, *i); err != nil {
			return err
		}
		if !moved {
			return nil
		}
		return iv.staffActivity(ctx, tx, i.CandidateID, "interview_rescheduled", domain.ScheduledDescription(i.Type, i.InterviewDate, true), a)
	})
	if err != nil {
		return nil, err
	}
	if moved {
		iv.notifyInterviewer(ctx, *i, "Jadwal interview berubah: "+i.CandidateName)
	}
	return iv.get(ctx, id)
}

// Cancel calls off a scheduled interview.
func (iv *Interviews) Cancel(ctx context.Context, id, reason string) (*domain.Interview, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if domain.UTF16Len(reason) > 500 {
		return nil, apperr.Field("reason", "Alasan maksimal 500 karakter")
	}
	i, err := iv.get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := requireScheduled(i); err != nil {
		return nil, err
	}
	i.Status, i.CancelReason, i.UpdatedBy = "cancelled", reason, a.id()
	desc := "Interview " + domain.InterviewTypeLabel(i.Type) + " dibatalkan"
	if reason != "" {
		desc += `: "` + reason + `"`
	}
	err = iv.Store.WithTx(ctx, func(tx domain.Store) error {
		if _, err := tx.UpdateInterview(ctx, *i); err != nil {
			return err
		}
		return iv.staffActivity(ctx, tx, i.CandidateID, "interview_cancelled", desc, a)
	})
	if err != nil {
		return nil, err
	}
	return iv.get(ctx, id)
}

// SubmitScorecard stores the scorecard and completes the interview; a
// completed interview may be re-scored.
func (iv *Interviews) SubmitScorecard(ctx context.Context, id string, items []domain.ScoreItem, recommendation, notes string) (*domain.Interview, error) {
	a, err := staff(ctx, true)
	if err != nil {
		return nil, err
	}
	clean, overall, err := domain.ValidateScorecard(items, recommendation, notes)
	if err != nil {
		return nil, err
	}
	i, err := iv.get(ctx, id)
	if err != nil {
		return nil, err
	}
	if i.Status == "cancelled" {
		return nil, conflict("Interview yang dibatalkan tidak bisa dinilai")
	}
	i.Status, i.Scorecard, i.OverallScore, i.Recommendation, i.UpdatedBy = "completed", clean, &overall, recommendation, a.id()
	if n := strings.TrimSpace(notes); n != "" {
		i.Notes = n
	}
	err = iv.Store.WithTx(ctx, func(tx domain.Store) error {
		if _, err := tx.UpdateInterview(ctx, *i); err != nil {
			return err
		}
		return iv.staffActivity(ctx, tx, i.CandidateID, "interview_scored", domain.ScorecardDescription(i.Type, overall, recommendation), a)
	})
	if err != nil {
		return nil, err
	}
	return iv.get(ctx, id)
}
