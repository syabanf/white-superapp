package app

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/recruitment/domain"
)

// Automations run off the scheduler's white.erp.tick.daily.
type Automations struct{ core }

func NewAutomations(d Deps) *Automations { return &Automations{core{d}} }

// Reminder kinds (recruitment.reminder_sends).
const (
	reminderInterviewTomorrow = "interview_tomorrow"
	reminderOfferExpiring     = "offer_expiring"
)

// TickDaily expires lapsed links, reminds interviewers about tomorrow's
// interviews and offer authors about offers expiring within a day (once
// per record and day), retries unsent hire events and prunes rate limits.
func (a *Automations) TickDaily(ctx context.Context, runAt time.Time) error {
	if runAt.IsZero() {
		runAt = a.now()
	}
	var errs []error
	if n, err := a.Store.ExpireLapsed(ctx, runAt); err != nil {
		errs = append(errs, err)
	} else if n > 0 {
		a.log().Info("expired lapsed recruitment links", "count", n)
	}
	errs = append(errs, a.remindInterviews(ctx, runAt), a.remindOffers(ctx, runAt), a.retryHired(ctx),
		a.Store.PruneRateLimits(ctx, runAt.Add(-24*time.Hour)))
	return errors.Join(errs...)
}

func (a *Automations) remindInterviews(ctx context.Context, runAt time.Time) error {
	today := domain.DayKey(runAt)
	start, _ := wibDayStart(today)
	from, to := start.AddDate(0, 0, 1), start.AddDate(0, 0, 2)
	list, err := a.Store.ListInterviews(ctx, domain.InterviewFilter{Status: "scheduled", From: &from, To: &to})
	if err != nil {
		return err
	}
	for _, i := range list {
		first, err := a.Store.ClaimReminder(ctx, reminderInterviewTomorrow, i.ID, today)
		if err != nil {
			return err
		}
		if !first {
			continue
		}
		where := i.Location
		if i.Mode == "online" {
			where = i.MeetingLink
		}
		body := "Interview " + domain.InterviewTypeLabel(i.Type) + " dengan " + i.CandidateName + " besok, " + domain.FormatDateTime(i.InterviewDate) + "."
		if where != "" {
			body += " Lokasi: " + where
		}
		a.notify(ctx, Notice{UserIDs: a.recipients(ctx, i.InterviewerUserID), Type: NotifInterviewReminder,
			Title: "Pengingat interview besok: " + i.CandidateName, Body: body, Href: hrefInterviews})
	}
	return nil
}

func (a *Automations) remindOffers(ctx context.Context, runAt time.Time) error {
	today := domain.DayKey(runAt)
	offers, err := a.Store.ExpiringOffers(ctx, runAt, runAt.Add(24*time.Hour))
	if err != nil {
		return err
	}
	for _, o := range offers {
		first, err := a.Store.ClaimReminder(ctx, reminderOfferExpiring, o.ID, today)
		if err != nil {
			return err
		}
		if !first {
			continue
		}
		a.notify(ctx, Notice{UserIDs: a.recipients(ctx, o.CreatedBy), Type: NotifOfferExpiring,
			Title: "Offer segera kedaluwarsa: " + o.CandidateName,
			Body:  "Offer v" + itoa(o.Version) + " untuk " + o.CandidateName + " berakhir " + domain.FormatDateTime(*o.ExpiresAt) + " dan belum direspons.",
			Href:  candidateHref(o.CandidateID)})
	}
	return nil
}

// retryHired publishes the hire events a failed publish left behind.
func (a *Automations) retryHired(ctx context.Context) error {
	pending, err := a.Store.PendingHiredEvents(ctx)
	if err != nil {
		return err
	}
	for _, c := range pending {
		a.publishHired(ctx, c)
	}
	return nil
}

func itoa(n int) string { return strconv.Itoa(n) }
