// Package postgres implements the publishing ports on top of sqlc queries.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
)

type Store struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: sqlc.New(pool)}
}

func (s *Store) WithTx(ctx context.Context, fn func(tx domain.Store) error) error {
	return pg.Tx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&Store{pool: s.pool, q: s.q.WithTx(tx)})
	})
}

func wrap(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ErrUnique{Constraint: pgErr.ConstraintName}
	}
	return err
}

func statuses(xs []domain.PostStatus) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, string(x))
	}
	return out
}

// ─── Media ───────────────────────────────────────────────────

func toMedia(r sqlc.GetMediaRow) domain.MediaAsset {
	return domain.MediaAsset{ID: r.ID, ClientID: r.ClientID, Kind: domain.MediaKind(r.Kind), URL: r.Url, ThumbnailURL: r.ThumbnailUrl, Filename: r.Filename, MimeType: r.MimeType, SizeBytes: r.SizeBytes, Width: r.Width, Height: r.Height, DurationSec: r.DurationSec, AltText: r.AltText, Tags: r.Tags, UploadedByID: r.UploadedByID, CreatedAt: r.CreatedAt, UsageCount: r.UsageCount}
}

func toMediaBare(m sqlc.PublishingMediaAsset, usage int32) domain.MediaAsset {
	return domain.MediaAsset{ID: m.ID, ClientID: m.ClientID, Kind: domain.MediaKind(m.Kind), URL: m.Url, ThumbnailURL: m.ThumbnailUrl, Filename: m.Filename, MimeType: m.MimeType, SizeBytes: m.SizeBytes, Width: m.Width, Height: m.Height, DurationSec: m.DurationSec, AltText: m.AltText, Tags: m.Tags, UploadedByID: m.UploadedByID, CreatedAt: m.CreatedAt, UsageCount: usage}
}

func (s *Store) ListMedia(ctx context.Context, clientID string) ([]domain.MediaAsset, error) {
	rows, err := s.q.ListMedia(ctx, clientID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.MediaAsset, 0, len(rows))
	for _, r := range rows {
		out = append(out, toMedia(sqlc.GetMediaRow(r)))
	}
	return out, nil
}

func (s *Store) GetMedia(ctx context.Context, id string) (*domain.MediaAsset, error) {
	r, err := s.q.GetMedia(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m := toMedia(r)
	return &m, nil
}

func (s *Store) ListMediaOwned(ctx context.Context, clientID string, ids []string) ([]domain.MediaAsset, error) {
	rows, err := s.q.ListMediaOwned(ctx, sqlc.ListMediaOwnedParams{ClientID: clientID, Ids: ids})
	if err != nil {
		return nil, err
	}
	out := make([]domain.MediaAsset, 0, len(rows))
	for _, r := range rows {
		out = append(out, toMediaBare(r, 0))
	}
	return out, nil
}

func (s *Store) InsertMedia(ctx context.Context, m domain.MediaAsset) (*domain.MediaAsset, error) {
	row, err := s.q.InsertMedia(ctx, sqlc.InsertMediaParams{ID: m.ID, ClientID: m.ClientID, Kind: sqlc.PublishingMediaKind(m.Kind), Url: m.URL, ThumbnailUrl: m.ThumbnailURL, Filename: m.Filename, MimeType: m.MimeType, SizeBytes: m.SizeBytes, Width: m.Width, Height: m.Height, DurationSec: m.DurationSec, AltText: m.AltText, Tags: m.Tags, UploadedByID: m.UploadedByID})
	if err != nil {
		return nil, wrap(err)
	}
	out := toMediaBare(row, 0)
	return &out, nil
}

func (s *Store) UpdateMedia(ctx context.Context, id string, altText *string, tags []string) (*domain.MediaAsset, error) {
	row, err := s.q.UpdateMedia(ctx, sqlc.UpdateMediaParams{ID: id, AltText: altText, Tags: tags})
	if err != nil {
		return nil, err
	}
	return s.GetMedia(ctx, row.ID)
}

func (s *Store) UpdateMediaAlt(ctx context.Context, id string, altText *string) error {
	return s.q.UpdateMediaAlt(ctx, sqlc.UpdateMediaAltParams{ID: id, AltText: altText})
}

func (s *Store) DeleteMedia(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteMedia(ctx, id)
	return n > 0, err
}

func (s *Store) ListMediaURLsByClient(ctx context.Context, clientID string) ([]string, error) {
	return s.q.ListMediaURLsByClient(ctx, clientID)
}

func (s *Store) DeleteMediaByClient(ctx context.Context, clientID string) (int64, error) {
	return s.q.DeleteMediaByClient(ctx, clientID)
}

func (s *Store) ReplacePostMedia(ctx context.Context, postID string, rows []domain.PostMedia) error {
	if err := s.q.DeletePostMedia(ctx, postID); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	params := make([]sqlc.InsertPostMediaParams, 0, len(rows))
	for _, r := range rows {
		params = append(params, sqlc.InsertPostMediaParams{PostID: postID, AssetID: r.AssetID, SortOrder: r.Order})
	}
	_, err := s.q.InsertPostMedia(ctx, params)
	return err
}

func (s *Store) ListPostMedia(ctx context.Context, postID string) ([]domain.MediaAsset, error) {
	rows, err := s.q.ListPostMedia(ctx, postID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.MediaAsset, 0, len(rows))
	for _, r := range rows {
		out = append(out, toMedia(sqlc.GetMediaRow(r)))
	}
	return out, nil
}

func (s *Store) ListPostMediaRows(ctx context.Context, postID string) ([]domain.PostMedia, error) {
	rows, err := s.q.ListPostMediaRows(ctx, postID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PostMedia, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.PostMedia{PostID: r.PostID, AssetID: r.AssetID, Order: r.SortOrder})
	}
	return out, nil
}

// ─── Posts ───────────────────────────────────────────────────

func toPost(p sqlc.PublishingContentPost) domain.ContentPost {
	var raw map[string]any
	if len(p.Utm) > 0 {
		_ = json.Unmarshal(p.Utm, &raw)
	}
	return domain.ContentPost{ID: p.ID, ClientID: p.ClientID, Title: p.Title, Body: p.Body, Status: domain.PostStatus(p.Status), Format: domain.ToFormat(p.Format), ScheduledAt: p.ScheduledAt, PublishedAt: p.PublishedAt, LinkURL: p.LinkUrl, FirstComment: p.FirstComment, Labels: p.Labels, UTM: domain.ParseUTM(raw), ReviewNote: p.ReviewNote, CreatedByID: p.CreatedByID, ApprovedByID: p.ApprovedByID, ApprovedAt: p.ApprovedAt, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

func utmJSON(u domain.UTM) []byte {
	u = u.Normalize()
	if !u.Has() {
		return nil
	}
	b, _ := json.Marshal(u)
	return b
}

func (s *Store) onePost(p sqlc.PublishingContentPost, err error) (*domain.ContentPost, error) {
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, wrap(err)
	}
	out := toPost(p)
	return &out, nil
}

func (s *Store) GetPost(ctx context.Context, id string) (*domain.ContentPost, error) {
	return s.onePost(s.q.GetPost(ctx, id))
}

func (s *Store) GetPostInClient(ctx context.Context, clientID, id string) (*domain.ContentPost, error) {
	return s.onePost(s.q.GetPostInClient(ctx, sqlc.GetPostInClientParams{ID: id, ClientID: clientID}))
}

func (s *Store) InsertPost(ctx context.Context, p domain.ContentPost) (*domain.ContentPost, error) {
	return s.onePost(s.q.InsertPost(ctx, sqlc.InsertPostParams{ID: p.ID, ClientID: p.ClientID, Title: p.Title, Body: p.Body, Status: sqlc.PublishingPostStatus(p.Status), ScheduledAt: p.ScheduledAt, LinkUrl: p.LinkURL, FirstComment: p.FirstComment, Labels: p.Labels, Utm: utmJSON(p.UTM), CreatedByID: p.CreatedByID, Format: string(domain.ToFormat(string(p.Format)))}))
}

func (s *Store) UpdatePostContent(ctx context.Context, p domain.ContentPost, clearApproval bool) (*domain.ContentPost, error) {
	return s.onePost(s.q.UpdatePostContent(ctx, sqlc.UpdatePostContentParams{ID: p.ID, Title: p.Title, Body: p.Body, Status: sqlc.PublishingPostStatus(p.Status), ScheduledAt: p.ScheduledAt, LinkUrl: p.LinkURL, FirstComment: p.FirstComment, Labels: p.Labels, Utm: utmJSON(p.UTM), Format: string(domain.ToFormat(string(p.Format))), ClearApproval: clearApproval}))
}

func (s *Store) UpdatePostStatus(ctx context.Context, id string, status domain.PostStatus, patch domain.StatusPatch) (*domain.ContentPost, error) {
	return s.onePost(s.q.UpdatePostStatus(ctx, sqlc.UpdatePostStatusParams{
		ID: id, Status: sqlc.PublishingPostStatus(status),
		SetSchedule: patch.SetSchedule, ScheduledAt: patch.ScheduledAt,
		SetReview: patch.SetReview, ReviewNote: patch.ReviewNote,
		SetApproval: patch.SetApproval, ApprovedByID: patch.ApprovedByID, ApprovedAt: patch.ApprovedAt,
		SetPublished: patch.SetPublished, PublishedAt: patch.PublishedAt,
	}))
}

func (s *Store) UpdatePostScheduledAt(ctx context.Context, id string, at *time.Time) error {
	return s.q.UpdatePostScheduledAt(ctx, sqlc.UpdatePostScheduledAtParams{ID: id, ScheduledAt: at})
}

func (s *Store) ClaimPost(ctx context.Context, id string, from []domain.PostStatus) (bool, error) {
	n, err := s.q.ClaimPost(ctx, sqlc.ClaimPostParams{ID: id, Statuses: statuses(from)})
	return n == 1, err
}

func (s *Store) ClaimDuePost(ctx context.Context, id string, now time.Time) (bool, error) {
	n, err := s.q.ClaimDuePost(ctx, sqlc.ClaimDuePostParams{ID: id, NowAt: now})
	return n == 1, err
}

func toPosts(rows []sqlc.PublishingContentPost) []domain.ContentPost {
	out := make([]domain.ContentPost, 0, len(rows))
	for _, r := range rows {
		out = append(out, toPost(r))
	}
	return out
}

func (s *Store) ListDuePosts(ctx context.Context, now time.Time, limit int32) ([]domain.ContentPost, error) {
	rows, err := s.q.ListDuePosts(ctx, sqlc.ListDuePostsParams{NowAt: now, MaxRows: limit})
	if err != nil {
		return nil, err
	}
	return toPosts(rows), nil
}

func (s *Store) ListPostsPublishing(ctx context.Context, ids []string) ([]domain.ContentPost, error) {
	rows, err := s.q.ListPostsPublishing(ctx, ids)
	if err != nil {
		return nil, err
	}
	return toPosts(rows), nil
}

func (s *Store) DeletePost(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeletePost(ctx, id)
	return n > 0, err
}

// summaries joins posts with their target platforms, failed counts and first media.
func (s *Store) summaries(ctx context.Context, posts []domain.ContentPost) ([]domain.PostSummaryRow, error) {
	ids := make([]string, 0, len(posts))
	for _, p := range posts {
		ids = append(ids, p.ID)
	}
	out := make([]domain.PostSummaryRow, 0, len(posts))
	if len(ids) == 0 {
		return out, nil
	}
	targets, err := s.q.ListTargetsForPosts(ctx, ids)
	if err != nil {
		return nil, err
	}
	media, err := s.q.FirstMediaForPosts(ctx, ids)
	if err != nil {
		return nil, err
	}
	platforms := map[string][]domain.Platform{}
	failed := map[string]int32{}
	for _, t := range targets {
		p := domain.ToPlatform(t.Platform)
		dup := false
		for _, x := range platforms[t.PostID] {
			if x == p {
				dup = true
				break
			}
		}
		if !dup {
			platforms[t.PostID] = append(platforms[t.PostID], p)
		}
		if t.Status == sqlc.PublishingPostStatusFAILED {
			failed[t.PostID]++
		}
	}
	thumbs := map[string]*string{}
	for _, m := range media {
		url := m.Url
		if m.ThumbnailUrl != nil {
			url = *m.ThumbnailUrl
		}
		thumbs[m.PostID] = &url
	}
	for _, p := range posts {
		out = append(out, domain.PostSummaryRow{Post: p, Platforms: platforms[p.ID], FailedTargets: failed[p.ID], ThumbnailURL: thumbs[p.ID]})
	}
	return out, nil
}

func (s *Store) ListPosts(ctx context.Context, clientID string) ([]domain.PostSummaryRow, error) {
	rows, err := s.q.ListPosts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return s.summaries(ctx, toPosts(rows))
}

func (s *Store) CountPostsByStatus(ctx context.Context, clientID string) (map[domain.PostStatus]int32, error) {
	rows, err := s.q.CountPostsByStatus(ctx, clientID)
	if err != nil {
		return nil, err
	}
	out := domain.PostStatusCounts(nil)
	for _, r := range rows {
		out[domain.PostStatus(r.Status)] = r.Count
	}
	return out, nil
}

func (s *Store) ListCalendarPosts(ctx context.Context, clientID string, from, to time.Time) ([]domain.PostSummaryRow, error) {
	rows, err := s.q.ListCalendarPosts(ctx, sqlc.ListCalendarPostsParams{ClientID: clientID, FromAt: from, ToAt: to})
	if err != nil {
		return nil, err
	}
	return s.summaries(ctx, toPosts(rows))
}

func (s *Store) GetPublishingStats(ctx context.Context, clientID string, now time.Time) (*domain.PublishingStats, error) {
	r, err := s.q.PublishingStats(ctx, sqlc.PublishingStatsParams{ClientID: clientID, NowAt: now, In7d: now.Add(7 * 24 * time.Hour), Ago30d: now.Add(-30 * 24 * time.Hour)})
	if err != nil {
		return nil, err
	}
	return &domain.PublishingStats{Scheduled7d: r.Scheduled7d, InReview: r.InReview, Published30: r.Published30d, Failed: r.Failed, Drafts: r.Drafts}, nil
}

func (s *Store) DeletePostsByClient(ctx context.Context, clientID string) (int64, error) {
	return s.q.DeletePostsByClient(ctx, clientID)
}

func (s *Store) NullUserRefs(ctx context.Context, userID string) error {
	for _, fn := range []func(context.Context, *string) error{s.q.NullPostCreatedBy, s.q.NullPostApprovedBy, s.q.NullMediaUploader, s.q.NullCommentUser, s.q.NullActivityUser} {
		if err := fn(ctx, &userID); err != nil {
			return err
		}
	}
	return nil
}

// ─── Targets ─────────────────────────────────────────────────

func toTarget(t sqlc.PublishingPostTarget) domain.PostTarget {
	return domain.PostTarget{ID: t.ID, PostID: t.PostID, SocialAccountID: t.SocialAccountID, Platform: domain.ToPlatform(t.Platform), BodyOverride: t.BodyOverride, Status: domain.PostStatus(t.Status), ExternalID: t.ExternalID, Permalink: t.Permalink, Error: t.Error, Attempts: t.Attempts, PublishedAt: t.PublishedAt}
}

func (s *Store) ListTargets(ctx context.Context, postID string) ([]domain.PostTarget, error) {
	rows, err := s.q.ListTargets(ctx, postID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PostTarget, 0, len(rows))
	for _, r := range rows {
		out = append(out, toTarget(r))
	}
	return out, nil
}

func (s *Store) UpsertTarget(ctx context.Context, t domain.PostTarget) error {
	return wrap(s.q.UpsertTarget(ctx, sqlc.UpsertTargetParams{ID: t.ID, PostID: t.PostID, SocialAccountID: t.SocialAccountID, Platform: string(t.Platform), BodyOverride: t.BodyOverride, Status: sqlc.PublishingPostStatus(t.Status)}))
}

func (s *Store) DeleteTargetsNotIn(ctx context.Context, postID string, keep []string) error {
	if keep == nil {
		keep = []string{}
	}
	return s.q.DeleteTargetsNotIn(ctx, sqlc.DeleteTargetsNotInParams{PostID: postID, Keep: keep})
}

func (s *Store) SetTargetsStatusExceptPublished(ctx context.Context, postID string, status domain.PostStatus) error {
	return s.q.SetTargetsStatusExceptPublished(ctx, sqlc.SetTargetsStatusExceptPublishedParams{PostID: postID, Status: sqlc.PublishingPostStatus(status)})
}

func (s *Store) SetTargetsStatus(ctx context.Context, ids []string, status domain.PostStatus) error {
	if len(ids) == 0 {
		return nil
	}
	return s.q.SetTargetsStatus(ctx, sqlc.SetTargetsStatusParams{Ids: ids, Status: sqlc.PublishingPostStatus(status)})
}

func (s *Store) UpdateTargetResult(ctx context.Context, id string, r domain.TargetResult) error {
	return s.q.UpdateTargetResult(ctx, sqlc.UpdateTargetResultParams{ID: id, Status: sqlc.PublishingPostStatus(r.Status), ExternalID: r.ExternalID, Permalink: r.Permalink, Error: r.Error, Attempts: r.Attempts, PublishedAt: r.PublishedAt})
}

func (s *Store) ListPublishedTargets(ctx context.Context, clientID string, from, to time.Time) ([]domain.PublishedTarget, error) {
	rows, err := s.q.ListPublishedTargets(ctx, sqlc.ListPublishedTargetsParams{ClientID: clientID, FromAt: from, ToAt: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.PublishedTarget, 0, len(rows))
	for _, r := range rows {
		var at time.Time
		if r.PublishedAt != nil {
			at = *r.PublishedAt
		}
		out = append(out, domain.PublishedTarget{PostID: r.PostID, Title: r.Title, Body: r.Body, Platform: domain.ToPlatform(r.Platform), SocialAccountID: r.SocialAccountID, ExternalID: r.ExternalID, PublishedAt: at})
	}
	return out, nil
}

func (s *Store) CountFailedTargets(ctx context.Context, clientID string, from, to time.Time) (int32, error) {
	return s.q.CountFailedTargets(ctx, sqlc.CountFailedTargetsParams{ClientID: clientID, FromAt: from, ToAt: to})
}

// ─── Comments and activities ─────────────────────────────────

func (s *Store) InsertComment(ctx context.Context, c domain.PostComment) (*domain.PostComment, error) {
	r, err := s.q.InsertComment(ctx, sqlc.InsertCommentParams{ID: c.ID, PostID: c.PostID, UserID: c.UserID, Body: c.Body})
	if err != nil {
		return nil, err
	}
	return &domain.PostComment{ID: r.ID, PostID: r.PostID, UserID: r.UserID, Body: r.Body, CreatedAt: r.CreatedAt}, nil
}

func (s *Store) ListComments(ctx context.Context, postID string) ([]domain.PostComment, error) {
	rows, err := s.q.ListComments(ctx, postID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PostComment, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.PostComment{ID: r.ID, PostID: r.PostID, UserID: r.UserID, Body: r.Body, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

func (s *Store) InsertActivity(ctx context.Context, a domain.PostActivity) error {
	var meta []byte
	if a.Meta != nil {
		b, err := json.Marshal(a.Meta)
		if err != nil {
			return err
		}
		meta = b
	}
	return s.q.InsertActivity(ctx, sqlc.InsertActivityParams{ID: a.ID, PostID: a.PostID, UserID: a.UserID, Type: a.Type, Meta: meta})
}

func (s *Store) ListActivities(ctx context.Context, postID string) ([]domain.PostActivity, error) {
	rows, err := s.q.ListActivities(ctx, postID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PostActivity, 0, len(rows))
	for _, r := range rows {
		var meta map[string]any
		if len(r.Meta) > 0 {
			_ = json.Unmarshal(r.Meta, &meta)
		}
		out = append(out, domain.PostActivity{ID: r.ID, PostID: r.PostID, UserID: r.UserID, Type: r.Type, Meta: meta, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

// ─── Sync jobs ───────────────────────────────────────────────

func (s *Store) InsertSyncJob(ctx context.Context, j domain.SyncJob) error {
	return s.q.InsertSyncJob(ctx, sqlc.InsertSyncJobParams{ID: j.ID, ClientID: j.ClientID, Kind: j.Kind, Status: sqlc.PublishingJobStatus(j.Status), StartedAt: j.StartedAt, FinishedAt: j.FinishedAt, Message: j.Message, Error: j.Error})
}

func (s *Store) ListSyncJobs(ctx context.Context, clientID string, limit int32) ([]domain.SyncJob, error) {
	rows, err := s.q.ListSyncJobs(ctx, sqlc.ListSyncJobsParams{ClientID: &clientID, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SyncJob, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.SyncJob{ID: r.ID, ClientID: r.ClientID, Kind: r.Kind, Status: domain.JobStatus(r.Status), StartedAt: r.StartedAt, FinishedAt: r.FinishedAt, Message: r.Message, Error: r.Error})
	}
	return out, nil
}

func (s *Store) DeleteSyncJobsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.q.DeleteSyncJobsBefore(ctx, cutoff)
}

func (s *Store) DeleteSyncJobsByClient(ctx context.Context, clientID string) (int64, error) {
	return s.q.DeleteSyncJobsByClient(ctx, &clientID)
}
