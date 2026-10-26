package clients

import (
	"context"
	"time"

	publishingv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/publishing/v1"
	"github.com/syabanf/white-superapp/white-backend/internal/content/app"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/grpcx"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Publishing calls the publishing service with whatever user the context
// carries (the forwarded caller or an AsUser token).
type Publishing struct {
	media publishingv1.MediaServiceClient
	posts publishingv1.PostServiceClient
}

func NewPublishing(conn *grpc.ClientConn) *Publishing {
	return &Publishing{media: publishingv1.NewMediaServiceClient(conn), posts: publishingv1.NewPostServiceClient(conn)}
}

func formatPB(f domain.Format) publishingv1.PostFormat {
	switch f {
	case domain.FormatStory:
		return publishingv1.PostFormat_STORY
	case domain.FormatReel:
		return publishingv1.PostFormat_REEL
	}
	return publishingv1.PostFormat_FEED
}

func (c *Publishing) UploadMedia(ctx context.Context, clientID string, files []app.MediaFile) ([]string, error) {
	req := &publishingv1.UploadMediaRequest{ClientId: clientID}
	for _, f := range files {
		req.Files = append(req.Files, &publishingv1.UploadMediaRequest_File{Filename: f.Filename, Content: f.Content})
	}
	res, err := c.media.UploadMedia(ctx, req)
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	ids := make([]string, 0, len(res.Assets))
	for _, a := range res.Assets {
		ids = append(ids, a.Id)
	}
	return ids, nil
}

func (c *Publishing) SavePost(ctx context.Context, in app.PostInput) (string, error) {
	req := &publishingv1.SavePostRequest{ClientId: in.ClientID, Id: in.PostID, Title: in.Title, Body: in.Body, MediaIds: in.MediaIDs, Format: formatPB(in.Format), Labels: []string{"content"}}
	for _, id := range in.AccountIDs {
		req.Targets = append(req.Targets, &publishingv1.SavePostRequest_TargetInput{SocialAccountId: id})
	}
	res, err := c.posts.SavePost(ctx, req)
	if err != nil {
		return "", grpcx.FromStatus(err)
	}
	return res.Id, nil
}

func (c *Publishing) ApprovePost(ctx context.Context, clientID, postID string) error {
	_, err := c.posts.ApprovePost(ctx, &publishingv1.PostRef{ClientId: clientID, PostId: postID})
	if err != nil {
		return grpcx.FromStatus(err)
	}
	return nil
}

func (c *Publishing) SchedulePost(ctx context.Context, clientID, postID string, at time.Time) error {
	_, err := c.posts.SchedulePost(ctx, &publishingv1.SchedulePostRequest{ClientId: clientID, PostId: postID, ScheduledAt: timestamppb.New(at)})
	if err != nil {
		return grpcx.FromStatus(err)
	}
	return nil
}

func (c *Publishing) GetPost(ctx context.Context, clientID, postID string) (*app.PostState, error) {
	res, err := c.posts.GetPost(ctx, &publishingv1.PostRef{ClientId: clientID, PostId: postID})
	if err != nil {
		return nil, grpcx.FromStatus(err)
	}
	st := &app.PostState{Status: res.Status.String()}
	if res.PublishedAt != nil {
		t := res.PublishedAt.AsTime()
		st.PublishedAt = &t
	}
	for _, t := range res.Targets {
		st.Targets = append(st.Targets, app.PostTarget{ExternalID: t.ExternalId, Permalink: t.Permalink, Error: t.Error})
	}
	return st, nil
}
