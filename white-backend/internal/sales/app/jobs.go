package app

import (
	"context"
	"errors"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
)

// Jobs runs the scheduled work off the ERP ticks (the Next.js watchers of
// NüHabit: followup reminders, workflow scheduled actions and time-based
// rules, CS SLA, quotation expiry; plus onboarding reminders).
type Jobs struct {
	*Deps
	auto       *Automation
	pipeline   *Pipeline
	chat       *Chat
	onboarding *Onboarding
}

func NewJobs(d *Deps, auto *Automation, pipeline *Pipeline, chat *Chat, onboarding *Onboarding) *Jobs {
	return &Jobs{Deps: d, auto: auto, pipeline: pipeline, chat: chat, onboarding: onboarding}
}

// Every15m is white.erp.tick.15m. Each step runs even when another fails.
func (j *Jobs) Every15m(ctx context.Context) error {
	_, e1 := j.auto.RunDueScheduledActions(ctx)
	e2 := j.auto.ScanTimeBasedRules(ctx)
	_, e3 := j.pipeline.SendDueReminders(ctx)
	_, e4 := j.chat.FlagSLABreaches(ctx)
	return errors.Join(e1, e2, e3, e4)
}

// Daily is white.erp.tick.daily.
func (j *Jobs) Daily(ctx context.Context) error {
	_, e1 := j.QuotationExpiryReminders(ctx)
	_, e2 := j.onboarding.SendReminders(ctx)
	return errors.Join(e1, e2)
}

// QuotationExpiryReminders tells the deal owner once when a sent
// quotation's validity ends within three days (quotation-expiry-watcher).
func (j *Jobs) QuotationExpiryReminders(ctx context.Context) (int, error) {
	now := j.now()
	qs, err := j.Store.ExpiringQuotations(ctx, domain.DueDate(now, 3))
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, q := range qs {
		if !domain.ExpiresSoon(q, now) {
			continue
		}
		if q.OwnerUserID != "" {
			j.notify(ctx, Notification{UserIDs: []string{q.OwnerUserID}, Type: NotifQuoteExpiry, Title: "Quotation " + q.QuoteNumber + " segera berakhir", Body: q.ClientName + " · berlaku s.d. " + domain.FormatDateLong(*q.ValidUntil), Href: "/erp/sales/pipeline?deal=" + q.DealID})
		}
		if err := j.Store.MarkQuotationReminded(ctx, q.ID, now); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}
