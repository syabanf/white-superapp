package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

// Publishing is the publish flow (scheduler.ts): claim, send each pending
// target through the real or mock publisher, resolve the outcome, record
// activity, sync job and notifications.
type Publishing struct{ Deps }

func NewPublishing(d Deps) *Publishing { return &Publishing{d} }

type PublishOptions struct {
	Now time.Time
	// Manual "publish now" or retry: no automatic retry, actor on the activity.
	Manual bool
	UserID string
	// Claimed: the caller already moved the post to PUBLISHING (cron claim).
	Claimed bool
	// SyncJob writes a per-post row; the cron writes one per client instead.
	SyncJob bool
}

type PublishError struct {
	Target string
	Error  string
}

type PublishOutcome struct {
	Status    domain.PostStatus
	Published int
	Failed    int
	Errors    []PublishError
}

var claimable = []domain.PostStatus{domain.StatusApproved, domain.StatusScheduled, domain.StatusFailed}

// targetError formats a publisher failure the way the old scheduler did:
// "{generic message} ({provider message})" for provider errors.
func targetError(err error) string {
	e := apperr.As(err)
	if e.Provider != "" {
		return apperr.Message(e.Code) + " (" + e.Message + ")"
	}
	if e.Code == apperr.Unknown && e.Unwrap() != nil {
		return e.Unwrap().Error()
	}
	return e.Message
}

// PublishPost returns nil, nil when the claim is lost (someone else owns the post).
func (p *Publishing) PublishPost(ctx context.Context, postID string, opts PublishOptions) (*PublishOutcome, error) {
	now := opts.Now
	if now.IsZero() {
		now = p.now()
	}
	if !opts.Claimed {
		ok, err := p.Store.ClaimPost(ctx, postID, claimable)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}
	}
	post, err := p.Store.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, nil
	}
	targets, err := p.Store.ListTargets(ctx, postID)
	if err != nil {
		return nil, err
	}
	assets, err := p.Store.ListPostMedia(ctx, postID)
	if err != nil {
		return nil, err
	}
	accounts, err := p.Analytics.OwnAccounts(ctx, post.ClientID)
	if err != nil {
		return nil, err
	}
	byID := accountsByID(accounts)
	metaConfigured := p.Integrations.MetaConfigured(ctx)

	link := ""
	if post.LinkURL != nil {
		link = domain.BuildUTMURL(*post.LinkURL, post.UTM)
	}
	media := make([]domain.PublishMedia, 0, len(assets))
	for _, a := range assets {
		media = append(media, domain.PublishMedia{URL: a.URL, Kind: a.Kind})
	}

	var pending []domain.PostTarget
	var outcomes []domain.TargetOutcome
	for _, t := range targets {
		if t.Status == domain.StatusPublished {
			outcomes = append(outcomes, domain.TargetOutcome{Status: domain.StatusPublished, Attempts: t.Attempts})
		} else {
			pending = append(pending, t)
		}
	}
	pendingIDs := make([]string, 0, len(pending))
	for _, t := range pending {
		pendingIDs = append(pendingIDs, t.ID)
	}
	if err := p.Store.SetTargetsStatus(ctx, pendingIDs, domain.StatusPublishing); err != nil {
		return nil, err
	}

	var errs []PublishError
	for _, t := range pending {
		acc, known := byID[t.SocialAccountID]
		platform := t.Platform
		if known {
			platform = acc.Platform
		}
		text := domain.CaptionFor(post.Body, t.BodyOverride)
		if link != "" && !strings.Contains(text, link) {
			if strings.TrimSpace(text) != "" {
				text = strings.TrimSpace(text) + "\n\n" + link
			} else {
				text = link
			}
		}
		res, perr := p.sendTarget(ctx, acc, known, platform, post, text, media, link, metaConfigured)
		attempts := t.Attempts + 1
		if perr != nil {
			msg := targetError(perr)
			stored := domain.TruncateRunes(msg, 500)
			if err := p.Store.UpdateTargetResult(ctx, t.ID, domain.TargetResult{Status: domain.StatusFailed, Error: &stored, Attempts: attempts}); err != nil {
				return nil, err
			}
			outcomes = append(outcomes, domain.TargetOutcome{Status: domain.StatusFailed, Attempts: attempts})
			errs = append(errs, PublishError{Target: strings.ToLower(string(platform)) + " @" + acc.Username, Error: msg})
			continue
		}
		if err := p.Store.UpdateTargetResult(ctx, t.ID, domain.TargetResult{Status: domain.StatusPublished, ExternalID: &res.ExternalID, Permalink: res.Permalink, Attempts: attempts, PublishedAt: &now}); err != nil {
			return nil, err
		}
		outcomes = append(outcomes, domain.TargetOutcome{Status: domain.StatusPublished, Attempts: attempts})
	}

	status := domain.ResolvePostOutcome(outcomes, domain.MaxPublishAttempts, opts.Manual)
	published, failed := 0, 0
	for _, o := range outcomes {
		if o.Status == domain.StatusPublished {
			published++
		} else if o.Status == domain.StatusFailed {
			failed++
		}
	}
	var publishedAt *time.Time
	if status == domain.StatusPublished {
		publishedAt = &now
	}
	if _, err := p.Store.UpdatePostStatus(ctx, post.ID, status, domain.StatusPatch{SetPublished: true, PublishedAt: publishedAt}); err != nil {
		return nil, err
	}
	activity := domain.ActPublishRetry
	switch status {
	case domain.StatusPublished:
		activity = domain.ActPublished
	case domain.StatusFailed:
		activity = domain.ActPublishFailed
	}
	errMeta := make([]map[string]string, 0, len(errs))
	for _, e := range errs {
		errMeta = append(errMeta, map[string]string{"target": e.Target, "error": e.Error})
	}
	var actor *string
	if opts.UserID != "" {
		actor = strPtr(opts.UserID)
	}
	if err := p.Store.InsertActivity(ctx, domain.PostActivity{ID: ids.New(), PostID: post.ID, UserID: actor, Type: activity, Meta: map[string]any{"published": published, "failed": failed, "errors": errMeta, "manual": opts.Manual, "mock": !metaConfigured}}); err != nil {
		return nil, err
	}
	if opts.SyncJob {
		label := post.Title
		if label == "" {
			label = post.ID
		}
		job := domain.SyncJob{ID: ids.New(), ClientID: &post.ClientID, Kind: domain.SyncKindPublish, Status: domain.JobSuccess, StartedAt: now, FinishedAt: strTime(p.now()), Message: strPtr(fmt.Sprintf("%s: %d terbit, %d gagal%s", label, published, failed, mockSuffix(metaConfigured)))}
		if status == domain.StatusFailed {
			job.Status = domain.JobFailed
		}
		if len(errs) > 0 {
			job.Error = strPtr(errs[0].Error)
		}
		if err := p.Store.InsertSyncJob(ctx, job); err != nil {
			return nil, err
		}
	}

	title := domain.NotificationTitle(*post)
	href := ""
	if info, err := p.Identity.ClientInfo(ctx, post.ClientID); err == nil && info != nil {
		href = postHref(info.Slug, post.ID)
	}
	switch status {
	case domain.StatusPublished:
		p.notify(ctx, NotifyInput{ClientID: post.ClientID, Audience: AudienceClientAll, Type: NotifPostPublished, Title: notifPublished(title), Href: href})
	case domain.StatusFailed:
		body := ""
		if len(errs) > 0 {
			body = errs[0].Error
		}
		p.notify(ctx, NotifyInput{ClientID: post.ClientID, Audience: AudienceClientAll, Type: NotifPostFailed, Title: notifFailed(title), Body: body, Href: href})
	}
	return &PublishOutcome{Status: status, Published: published, Failed: failed, Errors: errs}, nil
}

// sendTarget picks the publisher per target: real only with Meta app
// credentials AND a connection on the account; otherwise the mock.
func (p *Publishing) sendTarget(ctx context.Context, acc domain.SocialAccount, known bool, platform domain.Platform, post *domain.ContentPost, text string, media []domain.PublishMedia, link string, metaConfigured bool) (*domain.PublishResult, error) {
	if !known {
		return nil, apperr.New(apperr.NotFound, msgAccountMissing)
	}
	useReal := metaConfigured && acc.ConnectionID != nil && *acc.ConnectionID != ""
	publisher := p.Mock
	token := ""
	if useReal {
		publisher = p.Real
		var err error
		if token, err = p.Integrations.AccessToken(ctx, *acc.ConnectionID); err != nil {
			return nil, err
		}
	}
	in := domain.PublishInput{Platform: platform, AccessToken: token, AccountExternalID: acc.ExternalID, Text: text, Media: media, LinkURL: link}
	if platform == domain.PlatformInstagram && post.FirstComment != nil {
		in.FirstComment = *post.FirstComment
	}
	return publisher.Publish(ctx, in)
}

func (p *Publishing) notify(ctx context.Context, in NotifyInput) {
	if err := p.Identity.Notify(ctx, in); err != nil {
		p.log().Warn("notify failed", "type", in.Type, "err", err)
	}
}

func mockSuffix(metaConfigured bool) string {
	if metaConfigured {
		return ""
	}
	return " (mock)"
}

func strTime(t time.Time) *time.Time { return &t }

type DueRun struct {
	Attempted int
	Published int
	Failed    int
}

// PublishDuePosts is the tick entry point: claim due SCHEDULED posts one by
// one, publish them (retries allowed) and write one PUBLISH sync job per client.
func (p *Publishing) PublishDuePosts(ctx context.Context, now time.Time, limit int32) (*DueRun, error) {
	if limit <= 0 {
		limit = 20
	}
	candidates, err := p.Store.ListDuePosts(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	var claimedIDs []string
	for _, c := range candidates {
		ok, err := p.Store.ClaimDuePost(ctx, c.ID, now)
		if err != nil {
			return nil, err
		}
		if ok {
			claimedIDs = append(claimedIDs, c.ID)
		}
	}
	run := &DueRun{}
	if len(claimedIDs) == 0 {
		return run, nil
	}
	owned, err := p.Store.ListPostsPublishing(ctx, claimedIDs)
	if err != nil {
		return nil, err
	}
	type agg struct{ published, failed, retry int }
	perClient := map[string]*agg{}
	var order []string
	for _, post := range owned {
		a := perClient[post.ClientID]
		if a == nil {
			a = &agg{}
			perClient[post.ClientID] = a
			order = append(order, post.ClientID)
		}
		res, err := p.PublishPost(ctx, post.ID, PublishOptions{Now: now, Claimed: true})
		switch {
		case err != nil:
			run.Failed++
			a.failed++
			p.log().Error("publish crashed", "post", post.ID, "err", err)
			if _, uerr := p.Store.UpdatePostStatus(ctx, post.ID, domain.StatusFailed, domain.StatusPatch{}); uerr != nil {
				p.log().Warn("mark failed", "post", post.ID, "err", uerr)
			}
			if aerr := p.Store.InsertActivity(ctx, domain.PostActivity{ID: ids.New(), PostID: post.ID, Type: domain.ActPublishFailed, Meta: map[string]any{"error": err.Error()}}); aerr != nil {
				p.log().Warn("record failure", "post", post.ID, "err", aerr)
			}
		case res != nil && res.Status == domain.StatusPublished:
			run.Published++
			a.published++
		case res != nil && res.Status == domain.StatusFailed:
			run.Failed++
			a.failed++
		default:
			a.retry++
		}
	}
	run.Attempted = len(owned)
	metaConfigured := p.Integrations.MetaConfigured(ctx)
	for _, clientID := range order {
		a := perClient[clientID]
		status := domain.JobSuccess
		if a.failed > 0 && a.published == 0 {
			status = domain.JobFailed
		}
		cid := clientID
		job := domain.SyncJob{ID: ids.New(), ClientID: &cid, Kind: domain.SyncKindPublish, Status: status, StartedAt: now, FinishedAt: strTime(p.now()), Message: strPtr(fmt.Sprintf("%d terbit · %d gagal · %d diulang%s", a.published, a.failed, a.retry, mockSuffix(metaConfigured)))}
		if err := p.Store.InsertSyncJob(ctx, job); err != nil {
			return run, err
		}
	}
	return run, nil
}
