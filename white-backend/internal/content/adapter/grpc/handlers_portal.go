package grpcapi

import (
	"context"
	"time"

	contentv1 "github.com/syabanf/white-superapp/white-backend/gen/go/white/content/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

// portalServer is public: the share unlock token is the only credential.
type portalServer struct {
	contentv1.UnimplementedPortalServiceServer
	svc Services
}

func (s *portalServer) GetContentReview(ctx context.Context, r *contentv1.PortalRequest) (*contentv1.ContentReview, error) {
	items, err := s.svc.Portal.Review(ctx, r.Slug, r.UnlockToken)
	if err != nil {
		return nil, err
	}
	out := &contentv1.ContentReview{Items: make([]*contentv1.PortalItem, 0, len(items))}
	for _, pi := range items {
		it := pi.Item
		item := &contentv1.PortalItem{Id: it.ID, Title: it.Title, PlannedDate: it.PlannedDate.Format(time.DateOnly), Platform: platformPB[it.Platform], Format: formatPB[it.Format],
			Hook: it.Hook, Caption: it.Caption, Hashtags: it.Hashtags, RevisionRound: it.RevisionRound}
		for _, f := range pi.Files {
			item.Files = append(item.Files, &contentv1.PortalFile{Id: f.ID, FileName: f.FileName, MimeType: f.MimeType})
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

func (s *portalServer) DecideContent(ctx context.Context, r *contentv1.DecideContentRequest) (*emptypb.Empty, error) {
	if err := s.svc.Portal.Decide(ctx, r.Slug, r.UnlockToken, r.ItemId, r.Approve, r.SignerName, r.Note); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *portalServer) GetPortalFile(ctx context.Context, r *contentv1.PortalFileRequest) (*contentv1.FileBytes, error) {
	f, data, err := s.svc.Portal.File(ctx, r.Slug, r.UnlockToken, r.FileId)
	if err != nil {
		return nil, err
	}
	return &contentv1.FileBytes{FileName: f.FileName, ContentType: f.MimeType, Data: data}, nil
}
