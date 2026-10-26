package app

import (
	"context"
	"time"
	"unicode/utf8"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/validate"
)

// Posts is the composer and approval workflow (actions.ts).
type Posts struct {
	Deps
	publishing *Publishing
}

func NewPosts(d Deps, publishing *Publishing) *Posts { return &Posts{Deps: d, publishing: publishing} }

type TargetInput struct {
	SocialAccountID string
	BodyOverride    *string
}

type SaveInput struct {
	// ID empty creates a new post.
	ID           string
	Title        string
	Body         string
	Format       domain.PostFormat
	Targets      []TargetInput
	MediaIDs     []string
	MediaAlt     map[string]string
	LinkURL      *string
	UTM          domain.UTM
	FirstComment *string
	Labels       []string
	ScheduledAt  *time.Time
}

type SaveResult struct {
	ID     string
	Status domain.PostStatus
}

// validateSave applies the zod saveSchema limits.
func validateSave(in *SaveInput) error {
	v := validate.New()
	v.MaxLen("title", in.Title, 160, "Judul terlalu panjang.")
	v.MaxLen("body", in.Body, 65000, "Caption terlalu panjang.")
	if len(in.Targets) > 20 {
		v.Add("targets", "Maksimal 20 akun tujuan.")
	}
	for _, t := range in.Targets {
		if t.SocialAccountID == "" {
			v.Add("targets", "Akun tujuan tidak valid.")
		}
		if t.BodyOverride != nil {
			v.MaxLen("targets", *t.BodyOverride, 65000, "Caption khusus terlalu panjang.")
		}
	}
	if len(in.MediaIDs) > 10 {
		v.Add("mediaIds", "Maksimal 10 media.")
	}
	for _, id := range in.MediaIDs {
		if id == "" {
			v.Add("mediaIds", "Media tidak valid.")
		}
	}
	for _, alt := range in.MediaAlt {
		v.MaxLen("mediaAlt", alt, 500, "Teks alternatif terlalu panjang.")
	}
	if in.LinkURL != nil {
		v.MaxLen("linkUrl", *in.LinkURL, 2000, "Tautan terlalu panjang.")
	}
	for _, k := range domain.UTMKeys {
		v.MaxLen("utm", in.UTM[k], 100, "Parameter UTM terlalu panjang.")
	}
	if in.FirstComment != nil {
		v.MaxLen("firstComment", *in.FirstComment, 2200, "Komentar pertama terlalu panjang.")
	}
	if len(in.Labels) > 20 {
		v.Add("labels", "Maksimal 20 label.")
	}
	labels := make([]string, 0, len(in.Labels))
	for _, l := range in.Labels {
		l = trim(l)
		if n := utf8.RuneCountInString(l); n < 1 || n > 40 {
			v.Add("labels", "Label harus 1 sampai 40 karakter.")
			continue
		}
		labels = append(labels, l)
	}
	if v.HasErrors() {
		e := apperr.As(v.Err())
		e.Message = apperr.Message(apperr.Validation)
		return e
	}
	in.Labels = dedupe(labels)
	in.LinkURL = trimToNil(in.LinkURL)
	in.FirstComment = trimToNil(in.FirstComment)
	in.UTM = in.UTM.Normalize()
	in.MediaIDs = dedupe(in.MediaIDs)
	seen := map[string]bool{}
	targets := make([]TargetInput, 0, len(in.Targets))
	for _, t := range in.Targets {
		if seen[t.SocialAccountID] {
			continue
		}
		seen[t.SocialAccountID] = true
		t.BodyOverride = trimToNil(t.BodyOverride)
		targets = append(targets, t)
	}
	in.Targets = targets
	return nil
}

// Save creates or updates a post. Updating an editable post resets it to
// DRAFT and clears the approval; a planned scheduledAt is stored as-is.
func (p *Posts) Save(ctx context.Context, userID, clientID string, in SaveInput) (*SaveResult, error) {
	if err := validateSave(&in); err != nil {
		return nil, err
	}
	access, err := p.manage(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	platforms := map[string]domain.Platform{}
	if len(in.Targets) > 0 {
		own, err := p.Analytics.OwnAccounts(ctx, clientID)
		if err != nil {
			return nil, err
		}
		byID := accountsByID(own)
		for _, t := range in.Targets {
			acc, ok := byID[t.SocialAccountID]
			if !ok {
				return nil, apperr.New(apperr.Validation, msgAccountNotOwn)
			}
			platforms[t.SocialAccountID] = acc.Platform
		}
	}
	var media []domain.MediaAsset
	if len(in.MediaIDs) > 0 {
		owned, err := p.Store.ListMediaOwned(ctx, clientID, in.MediaIDs)
		if err != nil {
			return nil, err
		}
		if len(owned) != len(in.MediaIDs) {
			return nil, apperr.New(apperr.Validation, msgMediaNotOwn)
		}
		byID := map[string]domain.MediaAsset{}
		for _, m := range owned {
			byID[m.ID] = m
		}
		for _, id := range in.MediaIDs {
			media = append(media, byID[id])
		}
	}
	in.Format = domain.ToFormat(string(in.Format))
	draft := domain.DraftInput{Format: in.Format, Body: in.Body, ScheduledAt: in.ScheduledAt, LinkURL: in.LinkURL, FirstComment: in.FirstComment}
	for _, t := range in.Targets {
		draft.Targets = append(draft.Targets, domain.DraftTarget{Platform: platforms[t.SocialAccountID], BodyOverride: t.BodyOverride})
	}
	for _, m := range media {
		draft.Media = append(draft.Media, m.Kind)
	}
	if issues := domain.ValidatePost(draft, domain.ValidateOptions{Full: false, Now: p.now()}); len(issues) > 0 {
		return nil, apperr.New(apperr.Validation, domain.IssuesToMessage(msgFixIssues, issues))
	}
	mediaRows := make([]domain.PostMedia, 0, len(in.MediaIDs))
	for i, id := range in.MediaIDs {
		mediaRows = append(mediaRows, domain.PostMedia{AssetID: id, Order: int32(i)})
	}
	altUpdates := map[string]*string{}
	for _, id := range in.MediaIDs {
		if alt, ok := in.MediaAlt[id]; ok {
			altUpdates[id] = trimToNil(&alt)
		}
	}

	if in.ID != "" {
		existing, err := p.Store.GetPostInClient(ctx, clientID, in.ID)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, apperr.New(apperr.NotFound, msgNotFound)
		}
		if !domain.CanEdit(existing.Status, access.Role) {
			return nil, apperr.New(apperr.Validation, msgCannotEditStatus)
		}
		status := domain.StatusAfterEdit(existing.Status)
		err = p.Store.WithTx(ctx, func(tx domain.Store) error {
			post := domain.ContentPost{ID: existing.ID, Title: in.Title, Body: in.Body, Status: status, Format: in.Format, ScheduledAt: in.ScheduledAt, LinkURL: in.LinkURL, FirstComment: in.FirstComment, Labels: in.Labels, UTM: in.UTM}
			if _, err := tx.UpdatePostContent(ctx, post, existing.Status != domain.StatusDraft && status == domain.StatusDraft); err != nil {
				return err
			}
			if err := p.writeTargetsAndMedia(ctx, tx, existing.ID, in.Targets, platforms, status, mediaRows, altUpdates); err != nil {
				return err
			}
			return tx.InsertActivity(ctx, domain.PostActivity{ID: ids.New(), PostID: existing.ID, UserID: &userID, Type: domain.ActEdited, Meta: map[string]any{"from": existing.Status, "to": status}})
		})
		if err != nil {
			return nil, err
		}
		return &SaveResult{ID: existing.ID, Status: status}, nil
	}

	postID := ids.New()
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		post := domain.ContentPost{ID: postID, ClientID: clientID, Title: in.Title, Body: in.Body, Status: domain.StatusDraft, Format: in.Format, ScheduledAt: in.ScheduledAt, LinkURL: in.LinkURL, FirstComment: in.FirstComment, Labels: in.Labels, UTM: in.UTM, CreatedByID: &userID}
		if _, err := tx.InsertPost(ctx, post); err != nil {
			return err
		}
		if err := p.writeTargetsAndMedia(ctx, tx, postID, in.Targets, platforms, domain.StatusDraft, mediaRows, altUpdates); err != nil {
			return err
		}
		return tx.InsertActivity(ctx, domain.PostActivity{ID: ids.New(), PostID: postID, UserID: &userID, Type: domain.ActCreated})
	})
	if err != nil {
		return nil, err
	}
	return &SaveResult{ID: postID, Status: domain.StatusDraft}, nil
}

func (p *Posts) writeTargetsAndMedia(ctx context.Context, tx domain.Store, postID string, targets []TargetInput, platforms map[string]domain.Platform, status domain.PostStatus, media []domain.PostMedia, alt map[string]*string) error {
	keep := make([]string, 0, len(targets))
	for _, t := range targets {
		keep = append(keep, t.SocialAccountID)
	}
	if err := tx.DeleteTargetsNotIn(ctx, postID, keep); err != nil {
		return err
	}
	for _, t := range targets {
		if err := tx.UpsertTarget(ctx, domain.PostTarget{ID: ids.New(), PostID: postID, SocialAccountID: t.SocialAccountID, Platform: platforms[t.SocialAccountID], BodyOverride: t.BodyOverride, Status: status}); err != nil {
			return err
		}
	}
	if err := tx.ReplacePostMedia(ctx, postID, media); err != nil {
		return err
	}
	for id, text := range alt {
		if err := tx.UpdateMediaAlt(ctx, id, text); err != nil {
			return err
		}
	}
	return nil
}

// loaded is a post with everything the transitions check.
type loaded struct {
	post     domain.ContentPost
	targets  []domain.PostTarget
	media    []domain.MediaAsset
	access   Access
	isAuthor bool
}

// load mirrors actions.ts loadPost: a missing post or missing access is NOT_FOUND.
func (p *Posts) load(ctx context.Context, userID, clientID, postID string) (*loaded, error) {
	post, err := p.Store.GetPostInClient(ctx, clientID, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, apperr.New(apperr.NotFound, msgNotFound)
	}
	access, err := p.Identity.CheckAccess(ctx, userID, clientID)
	if err != nil {
		if apperr.Is(err, apperr.NotFound) || apperr.Is(err, apperr.Unauthorized) {
			return nil, apperr.New(apperr.NotFound, msgNotFound)
		}
		return nil, err
	}
	targets, err := p.Store.ListTargets(ctx, postID)
	if err != nil {
		return nil, err
	}
	media, err := p.Store.ListPostMedia(ctx, postID)
	if err != nil {
		return nil, err
	}
	return &loaded{post: *post, targets: targets, media: media, access: *access, isAuthor: post.CreatedByID != nil && *post.CreatedByID == userID}, nil
}

func (l *loaded) ctx() domain.TransitionCtx {
	return domain.TransitionCtx{Role: l.access.Role, IsAuthor: l.isAuthor}
}

func (l *loaded) validateFull(now time.Time, requireSchedule bool, scheduledAt *time.Time) error {
	in := draftInput(l.post, l.targets, l.media)
	if scheduledAt != nil {
		in.ScheduledAt = scheduledAt
	}
	if issues := domain.ValidatePost(in, domain.ValidateOptions{Full: true, RequireSchedule: requireSchedule, Now: now}); len(issues) > 0 {
		return apperr.New(apperr.Validation, domain.IssuesToMessage(msgFixIssues, issues))
	}
	return nil
}

// setStatus writes the post status, moves non-published targets along and
// records an activity, in one transaction.
func (p *Posts) setStatus(ctx context.Context, l *loaded, status domain.PostStatus, activity string, patch domain.StatusPatch, meta map[string]any) error {
	if meta == nil {
		meta = map[string]any{"from": l.post.Status, "to": status}
	}
	return p.Store.WithTx(ctx, func(tx domain.Store) error {
		if _, err := tx.UpdatePostStatus(ctx, l.post.ID, status, patch); err != nil {
			return err
		}
		if err := tx.SetTargetsStatusExceptPublished(ctx, l.post.ID, status); err != nil {
			return err
		}
		return tx.InsertActivity(ctx, domain.PostActivity{ID: ids.New(), PostID: l.post.ID, UserID: &l.access.UserID, Type: activity, Meta: meta})
	})
}

func (p *Posts) notifyAuthor(ctx context.Context, l *loaded, typ, title, body string) {
	if l.post.CreatedByID == nil || *l.post.CreatedByID == l.access.UserID {
		return
	}
	p.notify(ctx, NotifyInput{UserIDs: []string{*l.post.CreatedByID}, ClientID: l.post.ClientID, Type: typ, Title: title, Body: body, Href: postHref(l.access.Slug, l.post.ID)})
}

func (p *Posts) notify(ctx context.Context, in NotifyInput) {
	if err := p.Identity.Notify(ctx, in); err != nil {
		p.log().Warn("notify failed", "type", in.Type, "err", err)
	}
}

func (p *Posts) SubmitForReview(ctx context.Context, userID, clientID, postID string) (*SaveResult, error) {
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransition(l.post.Status, domain.StatusInReview, l.ctx()) {
		return nil, apperr.New(apperr.Unauthorized, msgTransition)
	}
	if err := l.validateFull(p.now(), false, nil); err != nil {
		return nil, err
	}
	if err := p.setStatus(ctx, l, domain.StatusInReview, domain.ActSubmitted, domain.StatusPatch{SetReview: true}, nil); err != nil {
		return nil, err
	}
	p.notify(ctx, NotifyInput{ClientID: clientID, Audience: AudienceClientApprovers, ExceptUserID: userID, Type: NotifPostReview, Title: notifReview(domain.NotificationTitle(l.post)), Body: msgReviewBody, Href: postHref(l.access.Slug, postID)})
	return &SaveResult{ID: postID, Status: domain.StatusInReview}, nil
}

func reviewDenied(l *loaded) error {
	if l.access.Role == domain.RoleManager && l.isAuthor {
		return apperr.New(apperr.Unauthorized, msgSelfApprove)
	}
	return apperr.New(apperr.Unauthorized, msgTransition)
}

func (p *Posts) Approve(ctx context.Context, userID, clientID, postID string) (*SaveResult, error) {
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransition(l.post.Status, domain.StatusApproved, l.ctx()) {
		return nil, reviewDenied(l)
	}
	now := p.now()
	if err := l.validateFull(now, false, nil); err != nil {
		return nil, err
	}
	patch := domain.StatusPatch{SetApproval: true, ApprovedByID: &userID, ApprovedAt: &now, SetReview: true}
	if err := p.setStatus(ctx, l, domain.StatusApproved, domain.ActApproved, patch, nil); err != nil {
		return nil, err
	}
	p.notifyAuthor(ctx, l, NotifPostApproved, notifApproved(domain.NotificationTitle(l.post)), "")
	return &SaveResult{ID: postID, Status: domain.StatusApproved}, nil
}

func (p *Posts) Reject(ctx context.Context, userID, clientID, postID, note string) (*SaveResult, error) {
	note = trim(note)
	if n := utf8.RuneCountInString(note); n < 1 || n > 1000 {
		return nil, apperr.Field("note", "Catatan wajib diisi (maksimal 1.000 karakter).")
	}
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransition(l.post.Status, domain.StatusRejected, l.ctx()) {
		return nil, reviewDenied(l)
	}
	if err := p.setStatus(ctx, l, domain.StatusRejected, domain.ActRejected, domain.StatusPatch{SetReview: true, ReviewNote: &note}, map[string]any{"note": note}); err != nil {
		return nil, err
	}
	p.notifyAuthor(ctx, l, NotifPostRejected, notifRejected(domain.NotificationTitle(l.post)), note)
	return &SaveResult{ID: postID, Status: domain.StatusRejected}, nil
}

func (p *Posts) Schedule(ctx context.Context, userID, clientID, postID string, at time.Time) (*SaveResult, error) {
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransition(l.post.Status, domain.StatusScheduled, l.ctx()) {
		return nil, apperr.New(apperr.Unauthorized, msgTransition)
	}
	at = at.UTC()
	if err := l.validateFull(p.now(), true, &at); err != nil {
		return nil, err
	}
	meta := map[string]any{"scheduledAt": at.Format(time.RFC3339Nano)}
	if err := p.setStatus(ctx, l, domain.StatusScheduled, domain.ActScheduled, domain.StatusPatch{SetSchedule: true, ScheduledAt: &at}, meta); err != nil {
		return nil, err
	}
	return &SaveResult{ID: postID, Status: domain.StatusScheduled}, nil
}

func (p *Posts) Unschedule(ctx context.Context, userID, clientID, postID string) (*SaveResult, error) {
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return nil, err
	}
	if l.post.Status != domain.StatusScheduled || !domain.CanTransition(l.post.Status, domain.StatusApproved, l.ctx()) {
		return nil, apperr.New(apperr.Unauthorized, msgTransition)
	}
	if err := p.setStatus(ctx, l, domain.StatusApproved, domain.ActUnscheduled, domain.StatusPatch{}, nil); err != nil {
		return nil, err
	}
	return &SaveResult{ID: postID, Status: domain.StatusApproved}, nil
}

// Reschedule moves the planned instant; the status stays as it is.
func (p *Posts) Reschedule(ctx context.Context, userID, clientID, postID string, at time.Time) (*SaveResult, error) {
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return nil, err
	}
	if !domain.CanReschedule(l.post.Status, l.access.Role) {
		return nil, apperr.New(apperr.Unauthorized, msgTransition)
	}
	at = at.UTC()
	if l.post.Status == domain.StatusScheduled && at.Before(p.now().Add(domain.MinScheduleLead)) {
		return nil, apperr.New(apperr.Validation, msgFixIssues+" "+domain.MsgScheduleTooSoon)
	}
	var from any
	if l.post.ScheduledAt != nil {
		from = l.post.ScheduledAt.UTC().Format(time.RFC3339Nano)
	}
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		if err := tx.UpdatePostScheduledAt(ctx, postID, &at); err != nil {
			return err
		}
		return tx.InsertActivity(ctx, domain.PostActivity{ID: ids.New(), PostID: postID, UserID: &userID, Type: domain.ActRescheduled, Meta: map[string]any{"from": from, "to": at.Format(time.RFC3339Nano)}})
	})
	if err != nil {
		return nil, err
	}
	return &SaveResult{ID: postID, Status: l.post.Status}, nil
}

func (p *Posts) BackToDraft(ctx context.Context, userID, clientID, postID string) (*SaveResult, error) {
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransition(l.post.Status, domain.StatusDraft, l.ctx()) {
		return nil, apperr.New(apperr.Unauthorized, msgTransition)
	}
	if err := p.setStatus(ctx, l, domain.StatusDraft, domain.ActBackToDraft, domain.StatusPatch{SetApproval: true}, nil); err != nil {
		return nil, err
	}
	return &SaveResult{ID: postID, Status: domain.StatusDraft}, nil
}

// PublishNow claims the post and publishes it in the request. A lost claim
// is reported as a transition error.
func (p *Posts) PublishNow(ctx context.Context, userID, clientID, postID string) (*PublishOutcome, error) {
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return nil, err
	}
	if !domain.CanTransition(l.post.Status, domain.StatusPublishing, l.ctx()) {
		return nil, apperr.New(apperr.Unauthorized, msgTransition)
	}
	if len(l.targets) == 0 {
		return nil, apperr.New(apperr.Validation, msgNothingToPublish)
	}
	if err := l.validateFull(p.now(), false, nil); err != nil {
		return nil, err
	}
	res, err := p.publishing.PublishPost(ctx, postID, PublishOptions{Now: p.now(), Manual: true, UserID: userID, SyncJob: true})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, apperr.New(apperr.Unauthorized, msgTransition)
	}
	return res, nil
}

func (p *Posts) Retry(ctx context.Context, userID, clientID, postID string) (*PublishOutcome, error) {
	post, err := p.Store.GetPostInClient(ctx, clientID, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, apperr.New(apperr.NotFound, msgNotFound)
	}
	if post.Status != domain.StatusFailed {
		return nil, apperr.New(apperr.Validation, msgNoFailedTargets)
	}
	return p.PublishNow(ctx, userID, clientID, postID)
}

// Duplicate copies content, targets and media into a new DRAFT; the
// schedule is not copied.
func (p *Posts) Duplicate(ctx context.Context, userID, clientID, postID string) (*SaveResult, error) {
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return nil, err
	}
	if !l.access.canWrite() {
		return nil, apperr.New(apperr.Unauthorized, "")
	}
	mediaRows, err := p.Store.ListPostMediaRows(ctx, postID)
	if err != nil {
		return nil, err
	}
	title := ""
	if l.post.Title != "" {
		title = l.post.Title + " (salinan)"
	}
	newID := ids.New()
	err = p.Store.WithTx(ctx, func(tx domain.Store) error {
		post := domain.ContentPost{ID: newID, ClientID: clientID, Title: title, Body: l.post.Body, Status: domain.StatusDraft, Format: l.post.Format, LinkURL: l.post.LinkURL, FirstComment: l.post.FirstComment, Labels: l.post.Labels, UTM: l.post.UTM, CreatedByID: &userID}
		if _, err := tx.InsertPost(ctx, post); err != nil {
			return err
		}
		for _, t := range l.targets {
			if err := tx.UpsertTarget(ctx, domain.PostTarget{ID: ids.New(), PostID: newID, SocialAccountID: t.SocialAccountID, Platform: t.Platform, BodyOverride: t.BodyOverride, Status: domain.StatusDraft}); err != nil {
				return err
			}
		}
		copies := make([]domain.PostMedia, 0, len(mediaRows))
		for _, m := range mediaRows {
			copies = append(copies, domain.PostMedia{AssetID: m.AssetID, Order: m.Order})
		}
		if err := tx.ReplacePostMedia(ctx, newID, copies); err != nil {
			return err
		}
		return tx.InsertActivity(ctx, domain.PostActivity{ID: ids.New(), PostID: newID, UserID: &userID, Type: domain.ActCreated, Meta: map[string]any{"duplicatedFrom": postID}})
	})
	if err != nil {
		return nil, err
	}
	return &SaveResult{ID: newID, Status: domain.StatusDraft}, nil
}

// Delete removes the post and its children. Stored media files stay.
func (p *Posts) Delete(ctx context.Context, userID, clientID, postID string) error {
	l, err := p.load(ctx, userID, clientID, postID)
	if err != nil {
		return err
	}
	if !domain.CanDelete(l.post.Status, l.access.Role) {
		return apperr.New(apperr.Unauthorized, msgTransition)
	}
	_, err = p.Store.DeletePost(ctx, postID)
	return err
}

// AddComment is open to every role, VIEWER included.
func (p *Posts) AddComment(ctx context.Context, userID, clientID, postID, body string) (*domain.PostComment, error) {
	body = trim(body)
	if n := utf8.RuneCountInString(body); n < 1 || n > 2000 {
		return nil, apperr.Field("body", "Komentar wajib diisi (maksimal 2.000 karakter).")
	}
	if _, err := p.load(ctx, userID, clientID, postID); err != nil {
		return nil, err
	}
	return p.Store.InsertComment(ctx, domain.PostComment{ID: ids.New(), PostID: postID, UserID: &userID, Body: body})
}
