package grpcapi

import (
	"context"

	publishingv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/publishing/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/app"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/dates"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Services struct {
	Media   *app.Media
	Posts   *app.Posts
	Queries *app.Queries
	Reports *app.Reports
}

func Register(s *grpc.Server, svc Services) {
	publishingv1.RegisterMediaServiceServer(s, &mediaServer{svc: svc})
	publishingv1.RegisterPostServiceServer(s, &postServer{svc: svc})
	publishingv1.RegisterReportDataServiceServer(s, &reportServer{svc: svc})
}

// ─── MediaService ────────────────────────────────────────────

type mediaServer struct {
	publishingv1.UnimplementedMediaServiceServer
	svc Services
}

func (s *mediaServer) ListMedia(ctx context.Context, req *publishingv1.ListMediaRequest) (*publishingv1.ListMediaResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.svc.Media.List(ctx, p.UserID, req.ClientId)
	if err != nil {
		return nil, err
	}
	return &publishingv1.ListMediaResponse{Assets: mediaListPB(rows)}, nil
}

func (s *mediaServer) UploadMedia(ctx context.Context, req *publishingv1.UploadMediaRequest) (*publishingv1.UploadMediaResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	files := make([]app.UploadFile, 0, len(req.Files))
	for _, f := range req.Files {
		files = append(files, app.UploadFile{Filename: f.Filename, Content: f.Content})
	}
	rows, err := s.svc.Media.Upload(ctx, p.UserID, req.ClientId, files)
	if err != nil {
		return nil, err
	}
	return &publishingv1.UploadMediaResponse{Assets: mediaListPB(rows)}, nil
}

func (s *mediaServer) AddExternalMedia(ctx context.Context, req *publishingv1.AddExternalMediaRequest) (*publishingv1.MediaAsset, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	m, err := s.svc.Media.AddExternal(ctx, p.UserID, req.ClientId, app.ExternalMediaInput{URL: req.Url, Kind: kindFrom(req.Kind), AltText: req.AltText, Tags: req.Tags})
	if err != nil {
		return nil, err
	}
	return mediaPB(m), nil
}

func (s *mediaServer) UpdateMedia(ctx context.Context, req *publishingv1.UpdateMediaRequest) (*publishingv1.MediaAsset, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	var tags []string
	if req.Tags != nil {
		tags = req.Tags.Values
		if tags == nil {
			tags = []string{}
		}
	}
	m, err := s.svc.Media.Update(ctx, p.UserID, req.ClientId, req.MediaId, req.AltText, tags)
	if err != nil {
		return nil, err
	}
	return mediaPB(m), nil
}

func (s *mediaServer) DeleteMedia(ctx context.Context, req *publishingv1.DeleteMediaRequest) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Media.Delete(ctx, p.UserID, req.ClientId, req.MediaId)
}

// GetMediaObject is internal: the gateway serves /api/media/{key} publicly,
// as the Next.js route did (unguessable keys stand in for signed URLs).
func (s *mediaServer) GetMediaObject(ctx context.Context, req *publishingv1.GetMediaObjectRequest) (*publishingv1.MediaObject, error) {
	obj, err := s.svc.Media.Object(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, apperr.New(apperr.NotFound, "Not found")
	}
	return &publishingv1.MediaObject{Content: obj.Body, ContentType: obj.ContentType, Etag: obj.ETag, Size: int64(len(obj.Body))}, nil
}

// ─── PostService ─────────────────────────────────────────────

type postServer struct {
	publishingv1.UnimplementedPostServiceServer
	svc Services
}

func (s *postServer) SavePost(ctx context.Context, req *publishingv1.SavePostRequest) (*publishingv1.SavePostResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	in := app.SaveInput{ID: req.Id, Title: req.Title, Body: req.Body, MediaIDs: req.MediaIds, MediaAlt: req.MediaAlt, UTM: utmFrom(req.Utm), Labels: req.Labels, ScheduledAt: timePtr(req.ScheduledAt)}
	if req.LinkUrl != "" {
		in.LinkURL = &req.LinkUrl
	}
	if req.FirstComment != "" {
		in.FirstComment = &req.FirstComment
	}
	for _, t := range req.Targets {
		ti := app.TargetInput{SocialAccountID: t.SocialAccountId}
		if t.BodyOverride != "" {
			ti.BodyOverride = &t.BodyOverride
		}
		in.Targets = append(in.Targets, ti)
	}
	r, err := s.svc.Posts.Save(ctx, p.UserID, req.ClientId, in)
	if err != nil {
		return nil, err
	}
	return saveResultPB(r), nil
}

type transition func(ctx context.Context, userID, clientID, postID string) (*app.SaveResult, error)

func (s *postServer) run(ctx context.Context, req *publishingv1.PostRef, fn transition) (*publishingv1.PostStatusResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, err := fn(ctx, p.UserID, req.ClientId, req.PostId)
	if err != nil {
		return nil, err
	}
	return statusResultPB(r), nil
}

func (s *postServer) SubmitForReview(ctx context.Context, req *publishingv1.PostRef) (*publishingv1.PostStatusResponse, error) {
	return s.run(ctx, req, s.svc.Posts.SubmitForReview)
}

func (s *postServer) ApprovePost(ctx context.Context, req *publishingv1.PostRef) (*publishingv1.PostStatusResponse, error) {
	return s.run(ctx, req, s.svc.Posts.Approve)
}

func (s *postServer) RejectPost(ctx context.Context, req *publishingv1.RejectPostRequest) (*publishingv1.PostStatusResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Posts.Reject(ctx, p.UserID, req.ClientId, req.PostId, req.Note)
	if err != nil {
		return nil, err
	}
	return statusResultPB(r), nil
}

func (s *postServer) SchedulePost(ctx context.Context, req *publishingv1.SchedulePostRequest) (*publishingv1.PostStatusResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	if req.ScheduledAt == nil {
		return nil, apperr.New(apperr.Validation, domain.MsgScheduleRequired)
	}
	r, err := s.svc.Posts.Schedule(ctx, p.UserID, req.ClientId, req.PostId, req.ScheduledAt.AsTime())
	if err != nil {
		return nil, err
	}
	return statusResultPB(r), nil
}

func (s *postServer) UnschedulePost(ctx context.Context, req *publishingv1.PostRef) (*publishingv1.PostStatusResponse, error) {
	return s.run(ctx, req, s.svc.Posts.Unschedule)
}

func (s *postServer) ReschedulePost(ctx context.Context, req *publishingv1.SchedulePostRequest) (*publishingv1.PostStatusResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	if req.ScheduledAt == nil {
		return nil, apperr.New(apperr.Validation, domain.MsgScheduleRequired)
	}
	r, err := s.svc.Posts.Reschedule(ctx, p.UserID, req.ClientId, req.PostId, req.ScheduledAt.AsTime())
	if err != nil {
		return nil, err
	}
	return statusResultPB(r), nil
}

func (s *postServer) BackToDraft(ctx context.Context, req *publishingv1.PostRef) (*publishingv1.PostStatusResponse, error) {
	return s.run(ctx, req, s.svc.Posts.BackToDraft)
}

func (s *postServer) PublishNow(ctx context.Context, req *publishingv1.PostRef) (*publishingv1.PublishResult, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Posts.PublishNow(ctx, p.UserID, req.ClientId, req.PostId)
	if err != nil {
		return nil, err
	}
	return publishResultPB(r), nil
}

func (s *postServer) RetryPost(ctx context.Context, req *publishingv1.PostRef) (*publishingv1.PublishResult, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Posts.Retry(ctx, p.UserID, req.ClientId, req.PostId)
	if err != nil {
		return nil, err
	}
	return publishResultPB(r), nil
}

func (s *postServer) DuplicatePost(ctx context.Context, req *publishingv1.PostRef) (*publishingv1.SavePostResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	r, err := s.svc.Posts.Duplicate(ctx, p.UserID, req.ClientId, req.PostId)
	if err != nil {
		return nil, err
	}
	return saveResultPB(r), nil
}

func (s *postServer) DeletePost(ctx context.Context, req *publishingv1.PostRef) (*emptypb.Empty, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, s.svc.Posts.Delete(ctx, p.UserID, req.ClientId, req.PostId)
}

func (s *postServer) AddPostComment(ctx context.Context, req *publishingv1.AddPostCommentRequest) (*publishingv1.PostComment, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	c, err := s.svc.Posts.AddComment(ctx, p.UserID, req.ClientId, req.PostId, req.Body)
	if err != nil {
		return nil, err
	}
	var name *string
	if p.Name != "" {
		name = &p.Name
	}
	return commentPB(*c, name), nil
}

func (s *postServer) GetPost(ctx context.Context, req *publishingv1.PostRef) (*publishingv1.PostDetail, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.svc.Queries.GetPost(ctx, p.UserID, req.ClientId, req.PostId)
	if err != nil {
		return nil, err
	}
	return detailPB(d), nil
}

func (s *postServer) ListPosts(ctx context.Context, req *publishingv1.ListPostsRequest) (*publishingv1.ListPostsResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	tab, ok := tabFrom[req.Tab]
	if !ok {
		tab = domain.TabAll
	}
	list, err := s.svc.Queries.ListPosts(ctx, p.UserID, req.ClientId, tab)
	if err != nil {
		return nil, err
	}
	out := &publishingv1.ListPostsResponse{}
	for _, r := range list.Rows {
		out.Rows = append(out.Rows, postRowPB(r))
	}
	for _, st := range domain.StatusOrder {
		out.Counts = append(out.Counts, &publishingv1.ListPostsResponse_StatusCount{Status: status(st), Count: list.Counts[st]})
	}
	return out, nil
}

func (s *postServer) GetCalendar(ctx context.Context, req *publishingv1.GetCalendarRequest) (*publishingv1.GetCalendarResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	res, err := s.svc.Queries.Calendar(ctx, p.UserID, req.ClientId, req.FromDay, req.ToDay)
	if err != nil {
		return nil, err
	}
	out := &publishingv1.GetCalendarResponse{From: ts(res.From), To: ts(res.To)}
	for _, r := range res.Posts {
		out.Posts = append(out.Posts, calendarPostPB(r))
	}
	return out, nil
}

func (s *postServer) GetPublishingStats(ctx context.Context, req *publishingv1.ClientRequest) (*publishingv1.PublishingStats, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	st, err := s.svc.Queries.Stats(ctx, p.UserID, req.ClientId)
	if err != nil {
		return nil, err
	}
	return &publishingv1.PublishingStats{Scheduled_7D: st.Scheduled7d, InReview: st.InReview, Published_30D: st.Published30, Failed: st.Failed, Drafts: st.Drafts}, nil
}

func (s *postServer) GetBestTimes(ctx context.Context, req *publishingv1.GetBestTimesRequest) (*publishingv1.GetBestTimesResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	slots, err := s.svc.Queries.BestTimes(ctx, p.UserID, req.ClientId, req.Timezone)
	if err != nil {
		return nil, err
	}
	out := &publishingv1.GetBestTimesResponse{}
	for _, sl := range slots {
		out.Slots = append(out.Slots, &publishingv1.BestTimeSlot{Day: int32(sl.Day), Hour: int32(sl.Hour), At: ts(sl.At), FromHistory: sl.FromHistory})
	}
	return out, nil
}

func (s *postServer) ListSyncJobs(ctx context.Context, req *publishingv1.ListSyncJobsRequest) (*publishingv1.ListSyncJobsResponse, error) {
	p, err := authn.Require(ctx)
	if err != nil {
		return nil, err
	}
	jobs, err := s.svc.Queries.SyncJobs(ctx, p.UserID, req.ClientId, req.Limit)
	if err != nil {
		return nil, err
	}
	out := &publishingv1.ListSyncJobsResponse{}
	for _, j := range jobs {
		out.Jobs = append(out.Jobs, syncJobPB(j))
	}
	return out, nil
}

// ─── ReportDataService (internal) ────────────────────────────

type reportServer struct {
	publishingv1.UnimplementedReportDataServiceServer
	svc Services
}

func (s *reportServer) GetPublishingSummary(ctx context.Context, req *publishingv1.GetPublishingSummaryRequest) (*publishingv1.PublishingSummary, error) {
	rng, err := dates.FromProto(req.Range)
	if err != nil {
		return nil, apperr.Wrap(apperr.Validation, err.Error(), err)
	}
	sum, err := s.svc.Reports.PublishingSummary(ctx, req.ClientId, rng)
	if err != nil {
		return nil, err
	}
	return summaryPB(sum), nil
}
