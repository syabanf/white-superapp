// Package grpcapi exposes the publishing use cases over gRPC. Handlers only
// translate between proto and domain; rules live in app.
package grpcapi

import (
	"encoding/json"
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	publishingv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/publishing/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/app"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ts(t time.Time) *timestamppb.Timestamp { return timestamppb.New(t) }

func tsPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func timePtr(t *timestamppb.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	v := t.AsTime().UTC()
	return &v
}

var statusPB = map[domain.PostStatus]publishingv1.PostStatus{
	domain.StatusDraft:      publishingv1.PostStatus_DRAFT,
	domain.StatusInReview:   publishingv1.PostStatus_IN_REVIEW,
	domain.StatusApproved:   publishingv1.PostStatus_APPROVED,
	domain.StatusScheduled:  publishingv1.PostStatus_SCHEDULED,
	domain.StatusPublishing: publishingv1.PostStatus_PUBLISHING,
	domain.StatusPublished:  publishingv1.PostStatus_PUBLISHED,
	domain.StatusFailed:     publishingv1.PostStatus_FAILED,
	domain.StatusRejected:   publishingv1.PostStatus_REJECTED,
}

func status(s domain.PostStatus) publishingv1.PostStatus { return statusPB[s] }

func formatPB(f domain.PostFormat) publishingv1.PostFormat {
	switch f {
	case domain.FormatStory:
		return publishingv1.PostFormat_STORY
	case domain.FormatReel:
		return publishingv1.PostFormat_REEL
	}
	return publishingv1.PostFormat_FEED
}

func formatFrom(f publishingv1.PostFormat) domain.PostFormat {
	switch f {
	case publishingv1.PostFormat_STORY:
		return domain.FormatStory
	case publishingv1.PostFormat_REEL:
		return domain.FormatReel
	}
	return domain.FormatFeed
}

func kind(k domain.MediaKind) publishingv1.MediaKind {
	if k == domain.MediaVideo {
		return publishingv1.MediaKind_VIDEO
	}
	return publishingv1.MediaKind_IMAGE
}

func kindFrom(k publishingv1.MediaKind) domain.MediaKind {
	switch k {
	case publishingv1.MediaKind_IMAGE:
		return domain.MediaImage
	case publishingv1.MediaKind_VIDEO:
		return domain.MediaVideo
	}
	return ""
}

func platform(p domain.Platform) commonv1.SocialPlatform {
	switch p {
	case domain.PlatformFacebook:
		return commonv1.SocialPlatform_FACEBOOK
	case domain.PlatformTikTok:
		return commonv1.SocialPlatform_TIKTOK
	}
	return commonv1.SocialPlatform_INSTAGRAM
}

func platforms(ps []domain.Platform) []commonv1.SocialPlatform {
	out := make([]commonv1.SocialPlatform, 0, len(ps))
	for _, p := range ps {
		out = append(out, platform(p))
	}
	return out
}

var tabFrom = map[publishingv1.PostsTab]domain.PostsTab{
	publishingv1.PostsTab_POSTS_TAB_ALL:       domain.TabAll,
	publishingv1.PostsTab_POSTS_TAB_DRAFT:     domain.TabDraft,
	publishingv1.PostsTab_POSTS_TAB_IN_REVIEW: domain.TabInReview,
	publishingv1.PostsTab_POSTS_TAB_APPROVED:  domain.TabApproved,
	publishingv1.PostsTab_POSTS_TAB_SCHEDULED: domain.TabScheduled,
	publishingv1.PostsTab_POSTS_TAB_PUBLISHED: domain.TabPublished,
	publishingv1.PostsTab_POSTS_TAB_FAILED:    domain.TabFailed,
}

func jobStatus(s domain.JobStatus) commonv1.JobStatus {
	switch s {
	case domain.JobPending:
		return commonv1.JobStatus_PENDING
	case domain.JobRunning:
		return commonv1.JobStatus_RUNNING
	case domain.JobSuccess:
		return commonv1.JobStatus_SUCCESS
	case domain.JobFailed:
		return commonv1.JobStatus_FAILED
	}
	return commonv1.JobStatus_JOB_STATUS_UNSPECIFIED
}

func mediaPB(m *domain.MediaAsset) *publishingv1.MediaAsset {
	return &publishingv1.MediaAsset{Id: m.ID, ClientId: m.ClientID, Kind: kind(m.Kind), Url: m.URL, ThumbnailUrl: m.ThumbnailURL, Filename: m.Filename, MimeType: m.MimeType, SizeBytes: m.SizeBytes, Width: m.Width, Height: m.Height, AltText: m.AltText, Tags: m.Tags, CreatedAt: ts(m.CreatedAt), UsageCount: m.UsageCount}
}

func mediaListPB(rows []domain.MediaAsset) []*publishingv1.MediaAsset {
	out := make([]*publishingv1.MediaAsset, 0, len(rows))
	for i := range rows {
		out = append(out, mediaPB(&rows[i]))
	}
	return out
}

func utmPB(u domain.UTM) *publishingv1.Utm {
	return &publishingv1.Utm{Source: u["source"], Medium: u["medium"], Campaign: u["campaign"], Term: u["term"], Content: u["content"]}
}

func utmFrom(u *publishingv1.Utm) domain.UTM {
	if u == nil {
		return domain.UTM{}
	}
	return domain.UTM{"source": u.Source, "medium": u.Medium, "campaign": u.Campaign, "term": u.Term, "content": u.Content}
}

func saveResultPB(r *app.SaveResult) *publishingv1.SavePostResponse {
	return &publishingv1.SavePostResponse{Id: r.ID, Status: status(r.Status)}
}

func statusResultPB(r *app.SaveResult) *publishingv1.PostStatusResponse {
	return &publishingv1.PostStatusResponse{Id: r.ID, Status: status(r.Status)}
}

func publishResultPB(r *app.PublishOutcome) *publishingv1.PublishResult {
	out := &publishingv1.PublishResult{Status: status(r.Status), Published: int32(r.Published), Failed: int32(r.Failed)}
	for _, e := range r.Errors {
		out.Errors = append(out.Errors, &publishingv1.PublishResult_PublishError{Target: e.Target, Error: e.Error})
	}
	return out
}

func commentPB(c domain.PostComment, userName *string) *publishingv1.PostComment {
	return &publishingv1.PostComment{Id: c.ID, Body: c.Body, CreatedAt: ts(c.CreatedAt), UserId: c.UserID, UserName: userName}
}

func detailPB(d *app.PostDetail) *publishingv1.PostDetail {
	p := d.Post
	out := &publishingv1.PostDetail{
		Id: p.ID, ClientId: p.ClientID, Title: p.Title, Body: p.Body, Status: status(p.Status),
		ScheduledAt: tsPtr(p.ScheduledAt), PublishedAt: tsPtr(p.PublishedAt), LinkUrl: p.LinkURL, FirstComment: p.FirstComment,
		Labels: p.Labels, Utm: utmPB(p.UTM), ReviewNote: p.ReviewNote, CreatedAt: ts(p.CreatedAt), UpdatedAt: ts(p.UpdatedAt),
		CreatedById: p.CreatedByID, CreatedBy: d.CreatedBy, ApprovedBy: d.ApprovedBy, ApprovedAt: tsPtr(p.ApprovedAt),
		Media: mediaListPB(d.Media), Format: formatPB(p.Format),
	}
	for _, t := range d.Targets {
		out.Targets = append(out.Targets, &publishingv1.PostTargetView{Id: t.ID, SocialAccountId: t.SocialAccountID, Platform: platform(t.Platform), Username: t.Username, DisplayName: t.DisplayName, AvatarUrl: t.AvatarURL, BodyOverride: t.BodyOverride, Status: status(t.Status), ExternalId: t.ExternalID, Permalink: t.Permalink, Error: t.Error, Attempts: t.Attempts, PublishedAt: tsPtr(t.PublishedAt)})
	}
	for _, c := range d.Comments {
		out.Comments = append(out.Comments, commentPB(c.PostComment, c.UserName))
	}
	for _, a := range d.Activities {
		meta := ""
		if a.Meta != nil {
			if b, err := json.Marshal(a.Meta); err == nil {
				meta = string(b)
			}
		}
		out.Activities = append(out.Activities, &publishingv1.PostActivity{Id: a.ID, Type: a.Type, MetaJson: meta, CreatedAt: ts(a.CreatedAt), UserName: a.UserName})
	}
	return out
}

func postRowPB(r app.PostRow) *publishingv1.PostRow {
	p := r.Post
	return &publishingv1.PostRow{Id: p.ID, Title: p.Title, Body: p.Body, BodyShort: domain.Truncate(p.Body, 90), Status: status(p.Status), At: ts(p.Instant()), ScheduledAt: tsPtr(p.ScheduledAt), PublishedAt: tsPtr(p.PublishedAt), UpdatedAt: ts(p.UpdatedAt), Platforms: platforms(r.Platforms), Author: r.Author, ThumbnailUrl: r.ThumbnailURL, FailedTargets: r.FailedTargets}
}

func calendarPostPB(r domain.PostSummaryRow) *publishingv1.CalendarPost {
	p := r.Post
	return &publishingv1.CalendarPost{Id: p.ID, Title: p.Title, BodyShort: domain.Truncate(p.Body, 60), Status: status(p.Status), At: ts(p.Instant()), Platforms: platforms(r.Platforms), ThumbnailUrl: r.ThumbnailURL}
}

func syncJobPB(j domain.SyncJob) *publishingv1.SyncJob {
	return &publishingv1.SyncJob{Id: j.ID, Kind: j.Kind, Status: jobStatus(j.Status), StartedAt: ts(j.StartedAt), FinishedAt: tsPtr(j.FinishedAt), Message: j.Message, Error: j.Error}
}

func summaryPB(s *domain.PublishingSummary) *publishingv1.PublishingSummary {
	out := &publishingv1.PublishingSummary{Posts: s.Posts, Failed: s.Failed, Mode: s.Mode}
	for _, p := range s.PerPlatform {
		out.PerPlatform = append(out.PerPlatform, &publishingv1.PublishingSummary_PlatformCount{Platform: platform(p.Platform), Count: p.Count})
	}
	for _, t := range s.Top {
		out.Top = append(out.Top, &publishingv1.PublishingSummary_PublishedTarget{PostId: t.PostID, Title: t.Title, Body: t.Body, Platform: platform(t.Platform), SocialAccountId: t.SocialAccountID, ExternalId: t.ExternalID, PublishedAt: ts(t.PublishedAt), Engagements: t.Engagements})
	}
	return out
}
