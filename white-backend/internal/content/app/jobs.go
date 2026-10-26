package app

import (
	"context"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
)

// Jobs is the background work: deadline reminders and client/user cleanup.
type Jobs struct{ Deps }

func NewJobs(d Deps) *Jobs { return &Jobs{d} }

// SendReminders notifies the owner of every open item whose current
// deadline is due tomorrow or overdue, once per item, deadline and day.
func (j *Jobs) SendReminders(ctx context.Context, now time.Time) (int, error) {
	today := domain.Day(now)
	refs, err := j.Store.ListOpenItemRefs(ctx)
	if err != nil {
		return 0, err
	}
	sent := 0
	slugs := map[string]string{}
	for _, r := range domain.DueReminders(refs, today) {
		fresh, err := j.Store.MarkReminder(ctx, r.Item.ID, r.Kind, today)
		if err != nil {
			return sent, err
		}
		if !fresh {
			continue
		}
		slug, ok := slugs[r.Item.ClientID]
		if !ok {
			if info, err := j.Identity.ClientInfo(ctx, r.Item.ClientID); err == nil && info != nil {
				slug = info.Slug
			}
			slugs[r.Item.ClientID] = slug
		}
		when := "besok"
		switch {
		case r.Due.Before(today):
			when = "terlewat"
		case r.Due.Equal(today):
			when = "hari ini"
		}
		in := NotifyInput{ClientID: r.Item.ClientID, Type: NotifDeadline, Title: "Deadline " + domain.DeadlineLabel(r.Kind) + " " + when + ": " + r.Item.Title, Body: "Jatuh tempo " + r.Due.Format("02-01-2006"), Href: itemHref(slug, r.Item.ID)}
		owner := r.Item.CopywriterID
		if r.Kind == domain.DeadlineDesign {
			owner = r.Item.DesignerID
		}
		if r.Kind == domain.DeadlineReview || owner == nil {
			in.Audience = AudienceClientApprovers
		} else {
			in.UserIDs = []string{*owner}
		}
		j.notify(ctx, in)
		sent++
	}
	return sent, nil
}

// PurgeClient removes the client's rows and files (white.client.deleted).
func (j *Jobs) PurgeClient(ctx context.Context, clientID string) error {
	if err := j.Store.DeleteClientData(ctx, clientID); err != nil {
		return err
	}
	return j.Files.DeleteClient(ctx, clientID)
}

// ForgetUser clears assignments to a deleted user (white.user.deleted).
func (j *Jobs) ForgetUser(ctx context.Context, userID string) error {
	return j.Store.ForgetUser(ctx, userID)
}
