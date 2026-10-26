package app

import (
	"context"
	"fmt"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/events"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Scheduling hands approved items to the publishing service and folds
// publish results back into the item.
type Scheduling struct{ Deps }

func NewScheduling(d Deps) *Scheduling { return &Scheduling{d} }

// Publishing post statuses content reacts to.
const (
	postPublished = "PUBLISHED"
	postFailed    = "FAILED"
)

// Schedule uploads the final design to publishing, creates the post (the
// copywriter is its author when a service token can be minted, so the
// scheduler can approve it), approves and schedules it.
func (s *Scheduling) Schedule(ctx context.Context, userID, clientID, itemID string, publishAt *time.Time) (*domain.Item, error) {
	if _, err := s.manage(ctx, userID, clientID); err != nil {
		return nil, err
	}
	it, err := ensureFound(s.Store.GetItem(ctx, clientID, itemID))(msgItemNotFound)
	if err != nil {
		return nil, err
	}
	if !domain.CanSchedule(it.Status) {
		return nil, apperr.New(apperr.Validation, msgNotSchedulable)
	}
	at := it.PublishAt
	if publishAt != nil {
		at = publishAt
	}
	if at == nil || !at.After(s.now()) {
		return nil, apperr.Field("publishAt", msgPublishAtRequired)
	}
	finals, err := s.Store.FinalDesignFiles(ctx, it.ID)
	if err != nil {
		return nil, err
	}
	if len(finals) == 0 {
		return nil, apperr.New(apperr.Validation, msgNoFinalDesign)
	}
	media := make([]MediaFile, 0, len(finals))
	for _, f := range finals {
		if !domain.IsMedia(f.MimeType) {
			return nil, apperr.New(apperr.Validation, msgFinalNotMedia)
		}
		data, err := s.Files.Get(ctx, f.FileKey)
		if err != nil {
			return nil, err
		}
		if data == nil {
			return nil, apperr.New(apperr.NotFound, msgFileNotFound)
		}
		media = append(media, MediaFile{Filename: f.FileName, Content: data})
	}
	accounts, err := s.Analytics.OwnAccounts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	var accountIDs []string
	for _, acc := range accounts {
		if acc.Platform == it.Platform {
			accountIDs = append(accountIDs, acc.ID)
		}
	}
	if len(accountIDs) == 0 {
		return nil, apperr.New(apperr.Validation, fmt.Sprintf(msgNoAccount, domain.PlatformLabel(it.Platform)))
	}
	mediaIDs, err := s.Publishing.UploadMedia(ctx, clientID, media)
	if err != nil {
		return nil, err
	}

	authorCtx := ctx
	if it.CopywriterID != nil && *it.CopywriterID != userID {
		if c, err := s.Identity.AsUser(ctx, *it.CopywriterID); err == nil {
			authorCtx = c
		} else {
			s.log().Info("posting as the scheduler", "item", it.ID, "err", err)
		}
	}
	in := PostInput{ClientID: clientID, Title: it.Title, Body: domain.ComposeCaption(it.Format, it.Caption, it.Hashtags), Format: domain.PublishingFormat(it.Format), AccountIDs: accountIDs, MediaIDs: mediaIDs}
	// A post left behind by an earlier attempt is still a draft and is
	// reused; after a failed publish a new post is created.
	if it.Status == domain.ItemApproved && it.PublishingPostID != nil {
		in.PostID = *it.PublishingPostID
	}
	postID, err := s.Publishing.SavePost(authorCtx, in)
	if err != nil {
		return nil, err
	}
	if it.PublishingPostID == nil || *it.PublishingPostID != postID {
		it.PublishingPostID = &postID
		if err := s.Store.UpdateItem(ctx, *it); err != nil {
			return nil, err
		}
	}
	if err := s.Publishing.ApprovePost(ctx, clientID, postID); err != nil {
		return nil, err
	}
	if err := s.Publishing.SchedulePost(ctx, clientID, postID, *at); err != nil {
		return nil, err
	}

	from := it.Status
	it.Status, it.PublishAt, it.ScheduledBy, it.PublishError, it.UpdatedBy = domain.ItemScheduled, at, &userID, nil, &userID
	err = s.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateItem(ctx, *it); err != nil {
			return err
		}
		f, t := string(from), string(it.Status)
		return tx.InsertActivity(ctx, domain.ItemActivity{ID: ids.New(), ItemID: it.ID, UserID: &userID, Type: domain.ActScheduled, FromStatus: &f, ToStatus: &t, Note: strPtr(at.UTC().Format(time.RFC3339))})
	})
	if err != nil {
		return nil, err
	}
	return it, nil
}

// Sync reads the publishing post now (as the caller).
func (s *Scheduling) Sync(ctx context.Context, userID, clientID, itemID string) error {
	a, err := s.access(ctx, userID, clientID)
	if err != nil {
		return err
	}
	it, err := ensureFound(s.Store.GetItem(ctx, clientID, itemID))(msgItemNotFound)
	if err != nil {
		return err
	}
	if it.PublishingPostID == nil {
		return apperr.New(apperr.Validation, msgNotScheduled)
	}
	state, err := s.Publishing.GetPost(ctx, clientID, *it.PublishingPostID)
	if err != nil {
		return err
	}
	return s.apply(ctx, it, state, &userID, a.Slug)
}

// apply folds the publishing state into a scheduled (or failed) item:
// published, failed, or back to approved when someone unscheduled the post
// in publishing.
func (s *Scheduling) apply(ctx context.Context, it *domain.Item, state *PostState, actor *string, slug string) error {
	if it.Status != domain.ItemScheduled && it.Status != domain.ItemFailed {
		return nil
	}
	from := it.Status
	act := domain.ItemActivity{ID: ids.New(), ItemID: it.ID, UserID: actor}
	switch state.Status {
	case postPublished:
		now := s.now()
		it.Status, it.PublishedAt, it.PublishError = domain.ItemPublished, state.PublishedAt, nil
		if it.PublishedAt == nil {
			it.PublishedAt = &now
		}
		it.ExternalIDs, it.Permalink = nil, nil
		for _, t := range state.Targets {
			if t.ExternalID != nil {
				it.ExternalIDs = append(it.ExternalIDs, *t.ExternalID)
			}
			if it.Permalink == nil && t.Permalink != nil {
				it.Permalink = t.Permalink
			}
		}
		act.Type = domain.ActPublished
	case postFailed:
		if from == domain.ItemFailed {
			return nil
		}
		it.Status = domain.ItemFailed
		msg := "Publikasi gagal."
		for _, t := range state.Targets {
			if t.Error != nil {
				msg = *t.Error
				break
			}
		}
		it.PublishError = &msg
		act.Type, act.Note = domain.ActFailed, &msg
	case "DRAFT", "IN_REVIEW", "APPROVED", "REJECTED":
		if from != domain.ItemScheduled {
			return nil
		}
		it.Status = domain.ItemApproved
		act.Type = domain.ActUnscheduled
	default:
		return nil
	}
	f, t := string(from), string(it.Status)
	act.FromStatus, act.ToStatus = &f, &t
	err := s.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdateItem(ctx, *it); err != nil {
			return err
		}
		return tx.InsertActivity(ctx, act)
	})
	if err != nil {
		return err
	}
	var to []string
	for _, id := range []*string{it.ScheduledBy, it.CopywriterID, it.DesignerID} {
		if id != nil && (actor == nil || *id != *actor) {
			to = append(to, *id)
		}
	}
	switch it.Status {
	case domain.ItemPublished:
		s.notify(ctx, NotifyInput{UserIDs: dedupe(to), ClientID: it.ClientID, Type: NotifItemPublished, Title: "Konten terbit: " + it.Title, Href: itemHref(slug, it.ID)})
	case domain.ItemFailed:
		s.notify(ctx, NotifyInput{UserIDs: dedupe(to), ClientID: it.ClientID, Type: NotifItemFailed, Title: "Konten gagal terbit: " + it.Title, Body: *it.PublishError, Href: itemHref(slug, it.ID)})
	}
	return nil
}

func dedupe(xs []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

type ReconcileRun struct {
	Checked int
	Changed int
}

// Reconcile checks scheduled items whose publish time has passed, acting
// as the user who scheduled them (identity service token).
func (s *Scheduling) Reconcile(ctx context.Context, now time.Time, limit int32) (*ReconcileRun, error) {
	items, err := s.Store.ListScheduledDue(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	run := &ReconcileRun{}
	for i := range items {
		it := &items[i]
		if it.ScheduledBy == nil {
			continue
		}
		uctx, err := s.Identity.AsUser(ctx, *it.ScheduledBy)
		if err != nil {
			return run, err
		}
		state, err := s.Publishing.GetPost(uctx, it.ClientID, *it.PublishingPostID)
		if err != nil {
			s.log().Warn("publishing post lookup failed", "item", it.ID, "err", err)
			continue
		}
		run.Checked++
		before := it.Status
		if err := s.apply(ctx, it, state, nil, s.clientSlug(ctx, it.ClientID)); err != nil {
			return run, err
		}
		if it.Status != before {
			run.Changed++
		}
	}
	return run, nil
}

func (s *Scheduling) clientSlug(ctx context.Context, clientID string) string {
	if info, err := s.Identity.ClientInfo(ctx, clientID); err == nil && info != nil {
		return info.Slug
	}
	return ""
}

// OnPostOutcome folds a publishing outcome event into the item that owns
// the post, without waiting for Reconcile. It reads the post's full state
// as the scheduler (every target's external id) and falls back to the
// event payload when that read fails. Reconcile stays as the safety net.
func (s *Scheduling) OnPostOutcome(ctx context.Context, ev events.PublishPostPayload) error {
	it, err := s.Store.GetItemByPublishingPost(ctx, ev.ClientID, ev.PostID)
	if err != nil || it == nil {
		return err
	}
	at := ev.At
	state := &PostState{Status: ev.Status, PublishedAt: &at, Targets: []PostTarget{{ExternalID: optID(ev.ExternalID), Permalink: optID(ev.Permalink), Error: optID(ev.Error)}}}
	if it.ScheduledBy != nil {
		if full, err := s.postAs(ctx, *it.ScheduledBy, it.ClientID, ev.PostID); err != nil {
			s.log().Warn("publishing post lookup failed, using event payload", "item", it.ID, "err", err)
		} else {
			state = full
		}
	}
	return s.apply(ctx, it, state, nil, s.clientSlug(ctx, it.ClientID))
}

func (s *Scheduling) postAs(ctx context.Context, userID, clientID, postID string) (*PostState, error) {
	uctx, err := s.Identity.AsUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.Publishing.GetPost(uctx, clientID, postID)
}
