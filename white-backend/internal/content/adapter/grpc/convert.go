// Package grpcapi exposes the content use cases over gRPC. Handlers only
// translate between proto and Go values; rules live in app.
package grpcapi

import (
	"time"

	commonv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/common/v1"
	contentv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/content/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
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

func dayPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.DateOnly)
	return &s
}

func name(names map[string]string, id *string) string {
	if id == nil {
		return ""
	}
	return names[*id]
}

// ─── enums ───────────────────────────────────────────────────

var platformPB = map[domain.Platform]commonv1.SocialPlatform{
	domain.PlatformInstagram: commonv1.SocialPlatform_INSTAGRAM,
	domain.PlatformFacebook:  commonv1.SocialPlatform_FACEBOOK,
	domain.PlatformTikTok:    commonv1.SocialPlatform_TIKTOK,
}

func platformFrom(p commonv1.SocialPlatform) domain.Platform {
	for k, v := range platformPB {
		if v == p {
			return k
		}
	}
	return ""
}

var formatPB = map[domain.Format]contentv1.ContentFormat{
	domain.FormatFeed:     contentv1.ContentFormat_FEED,
	domain.FormatCarousel: contentv1.ContentFormat_CAROUSEL,
	domain.FormatReel:     contentv1.ContentFormat_REEL,
	domain.FormatStory:    contentv1.ContentFormat_STORY,
}

func formatFrom(f contentv1.ContentFormat) domain.Format {
	for k, v := range formatPB {
		if v == f {
			return k
		}
	}
	return ""
}

var planStatusPB = map[domain.PlanStatus]contentv1.PlanStatus{
	domain.PlanDraft:    contentv1.PlanStatus_PLAN_DRAFT,
	domain.PlanInReview: contentv1.PlanStatus_PLAN_IN_REVIEW,
	domain.PlanApproved: contentv1.PlanStatus_PLAN_APPROVED,
}

// itemStatusPB relies on the proto names matching the domain values.
func itemStatusPB(s domain.ItemStatus) contentv1.ItemStatus {
	return contentv1.ItemStatus(contentv1.ItemStatus_value[string(s)])
}

func itemStatusFrom(s contentv1.ItemStatus) domain.ItemStatus {
	if s == contentv1.ItemStatus_ITEM_STATUS_UNSPECIFIED {
		return ""
	}
	return domain.ItemStatus(s.String())
}

func designStatusPB(s domain.DesignStatus) contentv1.DesignStatus {
	return contentv1.DesignStatus(contentv1.DesignStatus_value["DESIGN_"+string(s)])
}

func designStatusFrom(s contentv1.DesignStatus) domain.DesignStatus {
	if s == contentv1.DesignStatus_DESIGN_STATUS_UNSPECIFIED {
		return ""
	}
	return domain.DesignStatus(s.String()[len("DESIGN_"):])
}

var decisionPB = map[string]contentv1.ReviewDecision{
	domain.DecisionComment:          contentv1.ReviewDecision_COMMENT,
	domain.DecisionChangesRequested: contentv1.ReviewDecision_REQUEST_CHANGES,
	domain.DecisionApproved:         contentv1.ReviewDecision_APPROVE,
}

func decisionFrom(d contentv1.ReviewDecision) string {
	for k, v := range decisionPB {
		if v == d {
			return k
		}
	}
	return ""
}

func assetKindPB(k domain.AssetKind) contentv1.AssetKind {
	return contentv1.AssetKind(contentv1.AssetKind_value[string(k)])
}

func assetKindFrom(k contentv1.AssetKind) domain.AssetKind {
	if k == contentv1.AssetKind_ASSET_KIND_UNSPECIFIED {
		return ""
	}
	return domain.AssetKind(k.String())
}

// ─── plans ───────────────────────────────────────────────────

func briefPB(b domain.Brief) *contentv1.PlanBrief {
	out := &contentv1.PlanBrief{Goals: b.Goals, Pillars: b.Pillars, Tone: b.Tone, Notes: b.Notes}
	for _, f := range b.Frequency {
		out.Frequency = append(out.Frequency, &contentv1.PlatformFrequency{Platform: platformPB[f.Platform], PostsPerWeek: f.PerWeek})
	}
	return out
}

func briefFrom(b *contentv1.PlanBrief) domain.Brief {
	out := domain.Brief{Goals: b.GetGoals(), Pillars: b.GetPillars(), Tone: b.GetTone(), Notes: b.GetNotes()}
	for _, f := range b.GetFrequency() {
		out.Frequency = append(out.Frequency, domain.Frequency{Platform: platformFrom(f.Platform), PerWeek: f.PostsPerWeek})
	}
	return out
}

func planPB(p domain.Plan) *contentv1.Plan {
	return &contentv1.Plan{
		Id: p.ID, ClientId: p.ClientID, Month: p.Month.String(), Title: p.Title, Status: planStatusPB[p.Status], Brief: briefPB(p.Brief),
		ItemCount: p.ItemCount, PublishedCount: p.PublishedCount, AiModel: p.AIModel, GeneratedAt: tsPtr(p.GeneratedAt),
		SubmittedBy: p.SubmittedBy, SubmittedAt: tsPtr(p.SubmittedAt), ApprovedBy: p.ApprovedBy, ApprovedAt: tsPtr(p.ApprovedAt),
		ReviewNote: p.ReviewNote, CreatedBy: p.CreatedBy, CreatedAt: ts(p.CreatedAt), UpdatedAt: ts(p.UpdatedAt),
	}
}

func planDetailPB(d *app.PlanDetail) *contentv1.PlanDetail {
	out := &contentv1.PlanDetail{Plan: planPB(d.Plan)}
	for _, it := range d.Items {
		out.Items = append(out.Items, itemPB(it))
	}
	for _, c := range d.Comments {
		out.Comments = append(out.Comments, planCommentPB(c, name(d.Names, c.UserID)))
	}
	return out
}

func planCommentPB(c domain.PlanComment, userName string) *contentv1.PlanComment {
	return &contentv1.PlanComment{Id: c.ID, UserId: c.UserID, UserName: userName, Body: c.Body, CreatedAt: ts(c.CreatedAt)}
}

// ─── items ───────────────────────────────────────────────────

func itemPB(it domain.Item) *contentv1.Item {
	return &contentv1.Item{
		Id: it.ID, PlanId: it.PlanID, ClientId: it.ClientID, PlannedDate: it.PlannedDate.Format(time.DateOnly), PublishAt: tsPtr(it.PublishAt),
		Platform: platformPB[it.Platform], Format: formatPB[it.Format], Pillar: it.Pillar, Title: it.Title, Hook: it.Hook, Caption: it.Caption,
		Hashtags: it.Hashtags, VisualBrief: it.VisualBrief, CopywriterId: it.CopywriterID, DesignerId: it.DesignerID,
		CopyDue: dayPtr(it.CopyDue), DesignDue: dayPtr(it.DesignDue), ReviewDue: dayPtr(it.ReviewDue), Status: itemStatusPB(it.Status),
		RevisionRound: it.RevisionRound, CaptionVersion: it.CaptionVersion, PublishingPostId: it.PublishingPostID,
		PublishedAt: tsPtr(it.PublishedAt), Permalink: it.Permalink, PublishError: it.PublishError, HasFinalDesign: it.HasFinalDesign,
		ApprovedBy: it.ApprovedBy, ApprovedAt: tsPtr(it.ApprovedAt), CreatedBy: it.CreatedBy, UpdatedBy: it.UpdatedBy,
		CreatedAt: ts(it.CreatedAt), UpdatedAt: ts(it.UpdatedAt),
	}
}

func itemInputFrom(in *contentv1.ItemInput) app.ItemInput {
	return app.ItemInput{
		PlannedDate: in.GetPlannedDate(), PublishAt: timePtr(in.GetPublishAt()), Platform: platformFrom(in.GetPlatform()), Format: formatFrom(in.GetFormat()),
		Pillar: in.GetPillar(), Title: in.GetTitle(), VisualBrief: in.GetVisualBrief(), CopywriterID: in.GetCopywriterId(), DesignerID: in.GetDesignerId(),
		CopyDue: in.GetCopyDue(), DesignDue: in.GetDesignDue(), ReviewDue: in.GetReviewDue(),
	}
}

func itemCommentPB(c domain.ItemComment, userName string) *contentv1.ItemComment {
	return &contentv1.ItemComment{Id: c.ID, UserId: c.UserID, UserName: userName, Body: c.Body, Kind: c.Kind, Round: c.Round, CreatedAt: ts(c.CreatedAt)}
}

func itemDetailPB(d *app.ItemDetail) *contentv1.ItemDetail {
	out := &contentv1.ItemDetail{Item: itemPB(d.Item)}
	for _, r := range d.Revisions {
		out.Revisions = append(out.Revisions, &contentv1.CaptionRevision{Id: r.ID, Version: r.Version, Caption: r.Caption, Hook: r.Hook, Hashtags: r.Hashtags, Note: r.Note, CreatedBy: r.CreatedBy, CreatedByName: name(d.Names, r.CreatedBy), CreatedAt: ts(r.CreatedAt)})
	}
	for _, c := range d.Comments {
		out.Comments = append(out.Comments, itemCommentPB(c, name(d.Names, c.UserID)))
	}
	for _, a := range d.Activities {
		out.Activities = append(out.Activities, &contentv1.ItemActivity{Id: a.ID, UserId: a.UserID, UserName: name(d.Names, a.UserID), Type: a.Type, FromStatus: a.FromStatus, ToStatus: a.ToStatus, Note: a.Note, CreatedAt: ts(a.CreatedAt)})
	}
	for _, t := range d.Tasks {
		out.DesignTasks = append(out.DesignTasks, taskPB(t))
	}
	for _, s := range d.Allowed {
		out.AllowedMoves = append(out.AllowedMoves, itemStatusPB(s))
	}
	return out
}

// ─── design ──────────────────────────────────────────────────

func taskPB(t domain.DesignTask) *contentv1.DesignTask {
	return &contentv1.DesignTask{
		Id: t.ID, ItemId: t.ItemID, ClientId: t.ClientID, Title: t.Title, Brief: t.Brief, References: t.References,
		SizePreset: t.SizePreset, Width: t.Width, Height: t.Height, DesignerId: t.DesignerID, DueDate: dayPtr(t.DueDate),
		Status: designStatusPB(t.Status), CurrentVersion: t.CurrentVersion, CreatedBy: t.CreatedBy, CreatedAt: ts(t.CreatedAt), UpdatedAt: ts(t.UpdatedAt),
	}
}

func designInputFrom(in *contentv1.DesignTaskInput) app.DesignInput {
	return app.DesignInput{Title: in.GetTitle(), Brief: in.GetBrief(), References: in.GetReferences(), SizePreset: in.GetSizePreset(), DesignerID: in.GetDesignerId(), DueDate: in.GetDueDate()}
}

func designDetailPB(d *app.DesignDetail) *contentv1.DesignTaskDetail {
	out := &contentv1.DesignTaskDetail{Task: taskPB(d.Task)}
	for _, f := range d.Files {
		out.Files = append(out.Files, &contentv1.DesignFile{Id: f.ID, TaskId: f.TaskID, Version: f.Version, SortOrder: f.SortOrder, FileName: f.FileName, MimeType: f.MimeType, SizeBytes: f.SizeBytes, UploadedBy: f.UploadedBy, CreatedAt: ts(f.CreatedAt)})
	}
	for _, f := range d.Feedback {
		out.Feedback = append(out.Feedback, &contentv1.DesignFeedback{Id: f.ID, FileId: f.FileID, Version: f.Version, UserId: f.UserID, UserName: name(d.Names, f.UserID), Decision: decisionPB[f.Decision], Body: f.Body, CreatedAt: ts(f.CreatedAt)})
	}
	return out
}

func uploadsFrom(files []*contentv1.UploadFile) []app.UploadFile {
	out := make([]app.UploadFile, 0, len(files))
	for _, f := range files {
		out = append(out, app.UploadFile{Filename: f.Filename, Content: f.Content})
	}
	return out
}

func fileBytesPB(f *app.File) *contentv1.FileBytes {
	return &contentv1.FileBytes{FileName: f.Name, ContentType: f.ContentType, Data: f.Data}
}

// ─── assets ──────────────────────────────────────────────────

func folderPB(f domain.AssetFolder) *contentv1.AssetFolder {
	return &contentv1.AssetFolder{Id: f.ID, ClientId: f.ClientID, ParentId: f.ParentID, Name: f.Name, AssetCount: f.AssetCount, CreatedAt: ts(f.CreatedAt)}
}

func versionPB(v *domain.AssetVersion) *contentv1.AssetVersion {
	if v == nil {
		return nil
	}
	return &contentv1.AssetVersion{Id: v.ID, Version: v.Version, FileName: v.FileName, MimeType: v.MimeType, SizeBytes: v.SizeBytes, Note: v.Note, UploadedBy: v.UploadedBy, CreatedAt: ts(v.CreatedAt)}
}

func assetPB(a domain.Asset) *contentv1.Asset {
	return &contentv1.Asset{
		Id: a.ID, ClientId: a.ClientID, FolderId: a.FolderID, Kind: assetKindPB(a.Kind), Name: a.Name, Description: a.Description, Tags: a.Tags,
		CurrentVersion: a.CurrentVersion, Current: versionPB(a.Current), CreatedBy: a.CreatedBy, UpdatedBy: a.UpdatedBy, CreatedAt: ts(a.CreatedAt), UpdatedAt: ts(a.UpdatedAt),
	}
}

func assetDetailPB(d *app.AssetDetail) *contentv1.AssetDetail {
	out := &contentv1.AssetDetail{Asset: assetPB(d.Asset)}
	for i := range d.Versions {
		out.Versions = append(out.Versions, versionPB(&d.Versions[i]))
	}
	return out
}

// ─── reports ─────────────────────────────────────────────────

func breakdownsPB(bs []domain.Breakdown) []*contentv1.ReportBreakdown {
	out := make([]*contentv1.ReportBreakdown, 0, len(bs))
	for _, b := range bs {
		out = append(out, &contentv1.ReportBreakdown{Key: b.Key, Planned: b.Planned, Published: b.Published})
	}
	return out
}

func postsPB(ps []domain.ReportPost) []*contentv1.ReportPost {
	out := make([]*contentv1.ReportPost, 0, len(ps))
	for _, p := range ps {
		out = append(out, &contentv1.ReportPost{
			ItemId: p.ItemID, Title: p.Title, Platform: platformPB[p.Platform], Format: formatPB[p.Format], PlannedDate: p.PlannedDate,
			PublishedAt: tsPtr(p.PublishedAt), Permalink: p.Permalink, HasMetrics: p.HasMetrics, Likes: p.Likes, Comments: p.Comments,
			Shares: p.Shares, Saves: p.Saves, Reach: p.Reach, Views: p.Views, Engagements: p.Engagements,
		})
	}
	return out
}

func reportPB(r *domain.Report) *contentv1.ContentReport {
	d := r.Data
	return &contentv1.ContentReport{
		Id: r.ID, ClientId: r.ClientID, Month: r.Month.String(), Planned: d.Planned, Published: d.Published, Failed: d.Failed,
		Scheduled: d.Scheduled, InProgress: d.InProgress, DeadlinesMet: d.DeadlinesMet, DeadlinesTotal: d.DeadlinesTotal,
		OnTimeRate: d.OnTimeRate, RevisionsTotal: d.RevisionsTotal, RevisionAvg: d.RevisionAvg, CaptionVersions: d.CaptionVersions,
		PerPlatform: breakdownsPB(d.PerPlatform), PerFormat: breakdownsPB(d.PerFormat), EngagementAvailable: d.EngagementAvailable,
		TotalEngagements: d.TotalEngagements, TotalReach: d.TotalReach, Posts: postsPB(d.Posts), Top: postsPB(d.Top), Bottom: postsPB(d.Bottom),
		Summary: r.Summary, AiModel: r.AIModel, CreatedBy: r.CreatedBy, CreatedAt: ts(r.CreatedAt), UpdatedAt: ts(r.UpdatedAt),
	}
}

func monthErr() error { return apperr.Field("month", "Bulan harus berformat YYYY-MM.") }
