// Package postgres implements the content ports on top of sqlc queries.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/content/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/content/domain"
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

// one maps no rows to nil and converts the row.
func one[R any, T any](r R, err error, conv func(R) T) (*T, error) {
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := conv(r)
	return &v, nil
}

func many[R any, T any](rows []R, err error, conv func(R) T) ([]T, error) {
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(rows))
	for _, r := range rows {
		out = append(out, conv(r))
	}
	return out, nil
}

// when maps the zero time to nil so inserts default to now().
func when(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ─── Plans ───────────────────────────────────────────────────

func toPlan(p sqlc.ContentPlan) domain.Plan {
	freq := []domain.Frequency{}
	_ = json.Unmarshal(p.Frequency, &freq)
	return domain.Plan{
		ID: p.ID, ClientID: p.ClientID, Month: domain.MonthOf(p.Month), Title: p.Title, Status: domain.PlanStatus(p.Status),
		Brief:   domain.Brief{Goals: p.Goals, Pillars: p.Pillars, Frequency: freq, Tone: p.Tone, Notes: p.Notes},
		AIModel: p.AiModel, GeneratedAt: p.GeneratedAt, SubmittedBy: p.SubmittedBy, SubmittedAt: p.SubmittedAt,
		ApprovedBy: p.ApprovedBy, ApprovedAt: p.ApprovedAt, ReviewNote: p.ReviewNote,
		CreatedBy: p.CreatedBy, UpdatedBy: p.UpdatedBy, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func frequencyJSON(f []domain.Frequency) []byte {
	if f == nil {
		f = []domain.Frequency{}
	}
	b, _ := json.Marshal(f)
	return b
}

func (s *Store) InsertPlan(ctx context.Context, p domain.Plan) error {
	return wrap(s.q.InsertPlan(ctx, sqlc.InsertPlanParams{
		ID: p.ID, ClientID: p.ClientID, Month: p.Month.Time, Title: p.Title, Status: sqlc.ContentPlanStatus(p.Status),
		Goals: p.Brief.Goals, Pillars: nonNil(p.Brief.Pillars), Tone: p.Brief.Tone, Notes: p.Brief.Notes, Frequency: frequencyJSON(p.Brief.Frequency), CreatedBy: p.CreatedBy, CreatedAt: when(p.CreatedAt),
	}))
}

func (s *Store) GetPlan(ctx context.Context, clientID, id string) (*domain.Plan, error) {
	r, err := s.q.GetPlan(ctx, sqlc.GetPlanParams{ID: id, ClientID: clientID})
	return one(r, err, toPlan)
}

func (s *Store) GetPlanByMonth(ctx context.Context, clientID string, month domain.Month) (*domain.Plan, error) {
	r, err := s.q.GetPlanByMonth(ctx, sqlc.GetPlanByMonthParams{ClientID: clientID, Month: month.Time})
	return one(r, err, toPlan)
}

func (s *Store) ListPlans(ctx context.Context, clientID string) ([]domain.Plan, error) {
	rows, err := s.q.ListPlans(ctx, clientID)
	return many(rows, err, func(r sqlc.ListPlansRow) domain.Plan {
		p := toPlan(r.ContentPlan)
		p.ItemCount, p.PublishedCount = r.ItemCount, r.PublishedCount
		return p
	})
}

func (s *Store) UpdatePlan(ctx context.Context, p domain.Plan) error {
	return s.q.UpdatePlan(ctx, sqlc.UpdatePlanParams{
		ID: p.ID, Title: p.Title, Status: sqlc.ContentPlanStatus(p.Status), Goals: p.Brief.Goals, Pillars: nonNil(p.Brief.Pillars),
		Tone: p.Brief.Tone, Notes: p.Brief.Notes, Frequency: frequencyJSON(p.Brief.Frequency), AiModel: p.AIModel, GeneratedAt: p.GeneratedAt,
		SubmittedBy: p.SubmittedBy, SubmittedAt: p.SubmittedAt, ApprovedBy: p.ApprovedBy, ApprovedAt: p.ApprovedAt, ReviewNote: p.ReviewNote, UpdatedBy: p.UpdatedBy,
	})
}

func (s *Store) SoftDeletePlan(ctx context.Context, id, userID string) error {
	return s.q.SoftDeletePlan(ctx, sqlc.SoftDeletePlanParams{ID: id, UpdatedBy: strOrNil(userID)})
}

func (s *Store) InsertPlanComment(ctx context.Context, c domain.PlanComment) error {
	return s.q.InsertPlanComment(ctx, sqlc.InsertPlanCommentParams{ID: c.ID, PlanID: c.PlanID, UserID: c.UserID, Body: c.Body, CreatedAt: when(c.CreatedAt)})
}

func (s *Store) ListPlanComments(ctx context.Context, planID string) ([]domain.PlanComment, error) {
	rows, err := s.q.ListPlanComments(ctx, planID)
	return many(rows, err, func(c sqlc.ContentPlanComment) domain.PlanComment {
		return domain.PlanComment{ID: c.ID, PlanID: c.PlanID, UserID: c.UserID, Body: c.Body, CreatedAt: c.CreatedAt}
	})
}

// ─── Items ───────────────────────────────────────────────────

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

func toItem(i sqlc.ContentItem) domain.Item {
	return domain.Item{
		ID: i.ID, PlanID: i.PlanID, ClientID: i.ClientID, PlannedDate: i.PlannedDate, PublishAt: i.PublishAt,
		Platform: domain.Platform(i.Platform), Format: domain.Format(i.Format), Pillar: i.Pillar, Title: i.Title, Hook: i.Hook,
		Caption: i.Caption, Hashtags: i.Hashtags, VisualBrief: i.VisualBrief, CopywriterID: i.CopywriterID, DesignerID: i.DesignerID,
		CopyDue: i.CopyDue, DesignDue: i.DesignDue, ReviewDue: i.ReviewDue, Status: domain.ItemStatus(i.Status),
		RevisionRound: i.RevisionRound, CaptionVersion: i.CaptionVersion, CopyDoneAt: i.CopyDoneAt, DesignDoneAt: i.DesignDoneAt,
		ApprovedAt: i.ApprovedAt, ApprovedBy: i.ApprovedBy, ScheduledBy: i.ScheduledBy, PublishingPostID: i.PublishingPostID,
		PublishedAt: i.PublishedAt, Permalink: i.Permalink, ExternalIDs: i.ExternalIds, PublishError: i.PublishError,
		CreatedBy: i.CreatedBy, UpdatedBy: i.UpdatedBy, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func (s *Store) InsertItem(ctx context.Context, it domain.Item) error {
	return s.q.InsertItem(ctx, sqlc.InsertItemParams{
		ID: it.ID, PlanID: it.PlanID, ClientID: it.ClientID, PlannedDate: it.PlannedDate, PublishAt: it.PublishAt,
		Platform: sqlc.ContentPlatform(it.Platform), Format: sqlc.ContentContentFormat(it.Format), Pillar: it.Pillar, Title: it.Title,
		Hook: it.Hook, Caption: it.Caption, Hashtags: nonNil(it.Hashtags), VisualBrief: it.VisualBrief, CopywriterID: it.CopywriterID,
		DesignerID: it.DesignerID, CopyDue: it.CopyDue, DesignDue: it.DesignDue, ReviewDue: it.ReviewDue,
		Status: sqlc.ContentItemStatus(it.Status), CaptionVersion: it.CaptionVersion, CreatedBy: it.CreatedBy, CreatedAt: when(it.CreatedAt),
	})
}

func (s *Store) GetItem(ctx context.Context, clientID, id string) (*domain.Item, error) {
	r, err := s.q.GetItem(ctx, sqlc.GetItemParams{ID: id, ClientID: clientID})
	return one(r, err, func(r sqlc.GetItemRow) domain.Item {
		it := toItem(r.ContentItem)
		it.HasFinalDesign = r.HasFinalDesign
		return it
	})
}

func (s *Store) ListItems(ctx context.Context, clientID string, f domain.ItemFilter) ([]domain.Item, error) {
	p := sqlc.ListItemsParams{ClientID: clientID, PlanID: strOrNil(f.PlanID), Status: strOrNil(string(f.Status)), AssigneeID: strOrNil(f.AssigneeID)}
	if !f.Month.IsZero() {
		from, to := f.Month.Time, f.Month.Last()
		p.FromDay, p.ToDay = &from, &to
	}
	rows, err := s.q.ListItems(ctx, p)
	return many(rows, err, func(r sqlc.ListItemsRow) domain.Item {
		it := toItem(r.ContentItem)
		it.HasFinalDesign = r.HasFinalDesign
		return it
	})
}

func (s *Store) UpdateItem(ctx context.Context, it domain.Item) error {
	return s.q.UpdateItem(ctx, sqlc.UpdateItemParams{
		ID: it.ID, PlannedDate: it.PlannedDate, PublishAt: it.PublishAt, Platform: sqlc.ContentPlatform(it.Platform),
		Format: sqlc.ContentContentFormat(it.Format), Pillar: it.Pillar, Title: it.Title, Hook: it.Hook, Caption: it.Caption,
		Hashtags: nonNil(it.Hashtags), VisualBrief: it.VisualBrief, CopywriterID: it.CopywriterID, DesignerID: it.DesignerID,
		CopyDue: it.CopyDue, DesignDue: it.DesignDue, ReviewDue: it.ReviewDue, Status: sqlc.ContentItemStatus(it.Status),
		RevisionRound: it.RevisionRound, CaptionVersion: it.CaptionVersion, CopyDoneAt: it.CopyDoneAt, DesignDoneAt: it.DesignDoneAt,
		ApprovedAt: it.ApprovedAt, ApprovedBy: it.ApprovedBy, ScheduledBy: it.ScheduledBy, PublishingPostID: it.PublishingPostID,
		PublishedAt: it.PublishedAt, Permalink: it.Permalink, ExternalIds: nonNil(it.ExternalIDs), PublishError: it.PublishError,
		UpdatedBy: it.UpdatedBy,
	})
}

func (s *Store) SoftDeleteItem(ctx context.Context, id, userID string) error {
	return s.q.SoftDeleteItem(ctx, sqlc.SoftDeleteItemParams{ID: id, UpdatedBy: strOrNil(userID)})
}

func (s *Store) SoftDeleteIdeaItems(ctx context.Context, planID, userID string) error {
	return s.q.SoftDeleteIdeaItems(ctx, sqlc.SoftDeleteIdeaItemsParams{PlanID: planID, UpdatedBy: strOrNil(userID)})
}

func (s *Store) ListScheduledDue(ctx context.Context, before time.Time, limit int32) ([]domain.Item, error) {
	rows, err := s.q.ListScheduledDue(ctx, sqlc.ListScheduledDueParams{PublishAt: &before, Limit: limit})
	return many(rows, err, toItem)
}

func (s *Store) GetItemByPublishingPost(ctx context.Context, clientID, postID string) (*domain.Item, error) {
	r, err := s.q.GetItemByPublishingPost(ctx, sqlc.GetItemByPublishingPostParams{PublishingPostID: &postID, ClientID: clientID})
	return one(r, err, toItem)
}

func (s *Store) ListOpenItemRefs(ctx context.Context) ([]domain.ItemRef, error) {
	rows, err := s.q.ListOpenItemRefs(ctx)
	return many(rows, err, func(r sqlc.ListOpenItemRefsRow) domain.ItemRef {
		return domain.ItemRef{ID: r.ID, ClientID: r.ClientID, Title: r.Title, Status: domain.ItemStatus(r.Status), CopywriterID: r.CopywriterID, DesignerID: r.DesignerID, CopyDue: r.CopyDue, DesignDue: r.DesignDue, ReviewDue: r.ReviewDue}
	})
}

func (s *Store) InsertRevision(ctx context.Context, r domain.CaptionRevision) error {
	return wrap(s.q.InsertRevision(ctx, sqlc.InsertRevisionParams{ID: r.ID, ItemID: r.ItemID, Version: r.Version, Caption: r.Caption, Hook: r.Hook, Hashtags: nonNil(r.Hashtags), Note: r.Note, CreatedBy: r.CreatedBy, CreatedAt: when(r.CreatedAt)}))
}

func (s *Store) ListRevisions(ctx context.Context, itemID string) ([]domain.CaptionRevision, error) {
	rows, err := s.q.ListRevisions(ctx, itemID)
	return many(rows, err, func(r sqlc.ContentCaptionRevision) domain.CaptionRevision {
		return domain.CaptionRevision{ID: r.ID, ItemID: r.ItemID, Version: r.Version, Caption: r.Caption, Hook: r.Hook, Hashtags: r.Hashtags, Note: r.Note, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt}
	})
}

func (s *Store) InsertItemComment(ctx context.Context, c domain.ItemComment) error {
	return s.q.InsertItemComment(ctx, sqlc.InsertItemCommentParams{ID: c.ID, ItemID: c.ItemID, UserID: c.UserID, Body: c.Body, Kind: c.Kind, Round: c.Round, CreatedAt: when(c.CreatedAt)})
}

func (s *Store) ListItemComments(ctx context.Context, itemID string) ([]domain.ItemComment, error) {
	rows, err := s.q.ListItemComments(ctx, itemID)
	return many(rows, err, func(c sqlc.ContentItemComment) domain.ItemComment {
		return domain.ItemComment{ID: c.ID, ItemID: c.ItemID, UserID: c.UserID, Body: c.Body, Kind: c.Kind, Round: c.Round, CreatedAt: c.CreatedAt}
	})
}

func (s *Store) InsertActivity(ctx context.Context, a domain.ItemActivity) error {
	return s.q.InsertActivity(ctx, sqlc.InsertActivityParams{ID: a.ID, ItemID: a.ItemID, UserID: a.UserID, Type: a.Type, FromStatus: a.FromStatus, ToStatus: a.ToStatus, Note: a.Note, CreatedAt: when(a.CreatedAt)})
}

func (s *Store) ListActivities(ctx context.Context, itemID string) ([]domain.ItemActivity, error) {
	rows, err := s.q.ListActivities(ctx, itemID)
	return many(rows, err, func(a sqlc.ContentItemActivity) domain.ItemActivity {
		return domain.ItemActivity{ID: a.ID, ItemID: a.ItemID, UserID: a.UserID, Type: a.Type, FromStatus: a.FromStatus, ToStatus: a.ToStatus, Note: a.Note, CreatedAt: a.CreatedAt}
	})
}

func (s *Store) BackdateItem(ctx context.Context, itemID string, created, updated time.Time) error {
	return s.q.BackdateItem(ctx, sqlc.BackdateItemParams{ID: itemID, CreatedAt: created, UpdatedAt: updated})
}

func (s *Store) MarkReminder(ctx context.Context, itemID, kind string, day time.Time) (bool, error) {
	n, err := s.q.MarkReminder(ctx, sqlc.MarkReminderParams{ItemID: itemID, Kind: kind, Day: day})
	return n > 0, err
}

// ─── Design ──────────────────────────────────────────────────

func toTask(t sqlc.ContentDesignTask) domain.DesignTask {
	return domain.DesignTask{
		ID: t.ID, ItemID: t.ItemID, ClientID: t.ClientID, Title: t.Title, Brief: t.Brief, References: t.Refs,
		SizePreset: t.SizePreset, Width: t.Width, Height: t.Height, DesignerID: t.DesignerID, DueDate: t.DueDate,
		Status: domain.DesignStatus(t.Status), CurrentVersion: t.CurrentVersion, CreatedBy: t.CreatedBy, UpdatedBy: t.UpdatedBy,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toFile(f sqlc.ContentDesignFile) domain.DesignFile {
	return domain.DesignFile{ID: f.ID, TaskID: f.TaskID, Version: f.Version, SortOrder: f.SortOrder, FileKey: f.FileKey, FileName: f.FileName, MimeType: f.MimeType, SizeBytes: f.SizeBytes, UploadedBy: f.UploadedBy, CreatedAt: f.CreatedAt}
}

func (s *Store) InsertDesignTask(ctx context.Context, t domain.DesignTask) error {
	return s.q.InsertDesignTask(ctx, sqlc.InsertDesignTaskParams{
		ID: t.ID, ItemID: t.ItemID, ClientID: t.ClientID, Title: t.Title, Brief: t.Brief, Refs: nonNil(t.References),
		SizePreset: t.SizePreset, Width: t.Width, Height: t.Height, DesignerID: t.DesignerID, DueDate: t.DueDate,
		Status: sqlc.ContentDesignStatus(t.Status), CreatedBy: t.CreatedBy, CreatedAt: when(t.CreatedAt),
	})
}

func (s *Store) GetDesignTask(ctx context.Context, clientID, id string) (*domain.DesignTask, error) {
	r, err := s.q.GetDesignTask(ctx, sqlc.GetDesignTaskParams{ID: id, ClientID: clientID})
	return one(r, err, toTask)
}

func (s *Store) ListDesignTasks(ctx context.Context, clientID, itemID, designerID string, status domain.DesignStatus) ([]domain.DesignTask, error) {
	rows, err := s.q.ListDesignTasks(ctx, sqlc.ListDesignTasksParams{ClientID: clientID, ItemID: strOrNil(itemID), DesignerID: strOrNil(designerID), Status: strOrNil(string(status))})
	return many(rows, err, toTask)
}

func (s *Store) UpdateDesignTask(ctx context.Context, t domain.DesignTask) error {
	return s.q.UpdateDesignTask(ctx, sqlc.UpdateDesignTaskParams{
		ID: t.ID, Title: t.Title, Brief: t.Brief, Refs: nonNil(t.References), SizePreset: t.SizePreset, Width: t.Width, Height: t.Height,
		DesignerID: t.DesignerID, DueDate: t.DueDate, Status: sqlc.ContentDesignStatus(t.Status), CurrentVersion: t.CurrentVersion, UpdatedBy: t.UpdatedBy,
	})
}

func (s *Store) InsertDesignFile(ctx context.Context, f domain.DesignFile) error {
	return s.q.InsertDesignFile(ctx, sqlc.InsertDesignFileParams{ID: f.ID, TaskID: f.TaskID, Version: f.Version, SortOrder: f.SortOrder, FileKey: f.FileKey, FileName: f.FileName, MimeType: f.MimeType, SizeBytes: f.SizeBytes, UploadedBy: f.UploadedBy, CreatedAt: when(f.CreatedAt)})
}

func (s *Store) GetDesignFile(ctx context.Context, clientID, id string) (*domain.DesignFile, error) {
	r, err := s.q.GetDesignFile(ctx, sqlc.GetDesignFileParams{ID: id, ClientID: clientID})
	return one(r, err, toFile)
}

func (s *Store) ListDesignFiles(ctx context.Context, taskID string) ([]domain.DesignFile, error) {
	rows, err := s.q.ListDesignFiles(ctx, taskID)
	return many(rows, err, toFile)
}

func (s *Store) FinalDesignFiles(ctx context.Context, itemID string) ([]domain.DesignFile, error) {
	rows, err := s.q.FinalDesignFiles(ctx, itemID)
	return many(rows, err, toFile)
}

func (s *Store) InsertDesignFeedback(ctx context.Context, f domain.DesignFeedback) error {
	return s.q.InsertDesignFeedback(ctx, sqlc.InsertDesignFeedbackParams{ID: f.ID, TaskID: f.TaskID, FileID: f.FileID, Version: f.Version, UserID: f.UserID, Decision: f.Decision, Body: f.Body, CreatedAt: when(f.CreatedAt)})
}

func (s *Store) ListDesignFeedback(ctx context.Context, taskID string) ([]domain.DesignFeedback, error) {
	rows, err := s.q.ListDesignFeedback(ctx, taskID)
	return many(rows, err, func(f sqlc.ContentDesignFeedback) domain.DesignFeedback {
		return domain.DesignFeedback{ID: f.ID, TaskID: f.TaskID, FileID: f.FileID, Version: f.Version, UserID: f.UserID, Decision: f.Decision, Body: f.Body, CreatedAt: f.CreatedAt}
	})
}

// ─── Assets ──────────────────────────────────────────────────

func toFolder(f sqlc.ContentAssetFolder) domain.AssetFolder {
	return domain.AssetFolder{ID: f.ID, ClientID: f.ClientID, ParentID: f.ParentID, Name: f.Name, CreatedBy: f.CreatedBy, CreatedAt: f.CreatedAt, UpdatedAt: f.UpdatedAt}
}

func toAsset(a sqlc.ContentAsset) domain.Asset {
	return domain.Asset{ID: a.ID, ClientID: a.ClientID, FolderID: a.FolderID, Kind: domain.AssetKind(a.Kind), Name: a.Name, Description: a.Description, Tags: a.Tags, CurrentVersion: a.CurrentVersion, CreatedBy: a.CreatedBy, UpdatedBy: a.UpdatedBy, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
}

func toVersion(v sqlc.ContentAssetVersion) domain.AssetVersion {
	return domain.AssetVersion{ID: v.ID, AssetID: v.AssetID, Version: v.Version, FileKey: v.FileKey, FileName: v.FileName, MimeType: v.MimeType, SizeBytes: v.SizeBytes, Note: v.Note, UploadedBy: v.UploadedBy, CreatedAt: v.CreatedAt}
}

func (s *Store) InsertFolder(ctx context.Context, f domain.AssetFolder) error {
	return wrap(s.q.InsertFolder(ctx, sqlc.InsertFolderParams{ID: f.ID, ClientID: f.ClientID, ParentID: f.ParentID, Name: f.Name, CreatedBy: f.CreatedBy}))
}

func (s *Store) GetFolder(ctx context.Context, clientID, id string) (*domain.AssetFolder, error) {
	r, err := s.q.GetFolder(ctx, sqlc.GetFolderParams{ID: id, ClientID: clientID})
	return one(r, err, toFolder)
}

func (s *Store) ListFolders(ctx context.Context, clientID string) ([]domain.AssetFolder, error) {
	rows, err := s.q.ListFolders(ctx, clientID)
	return many(rows, err, func(r sqlc.ListFoldersRow) domain.AssetFolder {
		f := toFolder(r.ContentAssetFolder)
		f.AssetCount = r.AssetCount
		return f
	})
}

func (s *Store) RenameFolder(ctx context.Context, id, name string) error {
	return wrap(s.q.RenameFolder(ctx, sqlc.RenameFolderParams{ID: id, Name: name}))
}

func (s *Store) DeleteFolder(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteEmptyFolder(ctx, id)
	return n > 0, err
}

func (s *Store) InsertAsset(ctx context.Context, a domain.Asset) error {
	return s.q.InsertAsset(ctx, sqlc.InsertAssetParams{ID: a.ID, ClientID: a.ClientID, FolderID: a.FolderID, Kind: sqlc.ContentAssetKind(a.Kind), Name: a.Name, Description: a.Description, Tags: nonNil(a.Tags), CurrentVersion: a.CurrentVersion, CreatedBy: a.CreatedBy})
}

func (s *Store) GetAsset(ctx context.Context, clientID, id string) (*domain.Asset, error) {
	r, err := s.q.GetAsset(ctx, sqlc.GetAssetParams{ID: id, ClientID: clientID})
	return one(r, err, toAsset)
}

func (s *Store) SearchAssets(ctx context.Context, clientID string, f domain.AssetFilter) ([]domain.Asset, error) {
	rows, err := s.q.SearchAssets(ctx, sqlc.SearchAssetsParams{ClientID: clientID, Kind: strOrNil(string(f.Kind)), FolderID: strOrNil(f.FolderID), Tag: strOrNil(f.Tag), Query: strOrNil(f.Query)})
	return many(rows, err, func(r sqlc.SearchAssetsRow) domain.Asset {
		a := toAsset(r.ContentAsset)
		v := toVersion(r.ContentAssetVersion)
		a.Current = &v
		return a
	})
}

func (s *Store) UpdateAsset(ctx context.Context, a domain.Asset) error {
	return s.q.UpdateAsset(ctx, sqlc.UpdateAssetParams{ID: a.ID, FolderID: a.FolderID, Kind: sqlc.ContentAssetKind(a.Kind), Name: a.Name, Description: a.Description, Tags: nonNil(a.Tags), CurrentVersion: a.CurrentVersion, UpdatedBy: a.UpdatedBy})
}

func (s *Store) SoftDeleteAsset(ctx context.Context, id, userID string) error {
	return s.q.SoftDeleteAsset(ctx, sqlc.SoftDeleteAssetParams{ID: id, UpdatedBy: strOrNil(userID)})
}

func (s *Store) InsertAssetVersion(ctx context.Context, v domain.AssetVersion) error {
	return wrap(s.q.InsertAssetVersion(ctx, sqlc.InsertAssetVersionParams{ID: v.ID, AssetID: v.AssetID, Version: v.Version, FileKey: v.FileKey, FileName: v.FileName, MimeType: v.MimeType, SizeBytes: v.SizeBytes, Note: v.Note, UploadedBy: v.UploadedBy, CreatedAt: when(v.CreatedAt)}))
}

func (s *Store) ListAssetVersions(ctx context.Context, assetID string) ([]domain.AssetVersion, error) {
	rows, err := s.q.ListAssetVersions(ctx, assetID)
	return many(rows, err, toVersion)
}

// ─── Reports ─────────────────────────────────────────────────

func toReport(r sqlc.ContentReport) domain.Report {
	var data domain.ReportData
	_ = json.Unmarshal(r.Data, &data)
	return domain.Report{ID: r.ID, ClientID: r.ClientID, Month: domain.MonthOf(r.Month), Data: data, Summary: r.Summary, AIModel: r.AiModel, CreatedBy: r.CreatedBy, UpdatedBy: r.UpdatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func (s *Store) UpsertReport(ctx context.Context, r domain.Report) (*domain.Report, error) {
	data, err := json.Marshal(r.Data)
	if err != nil {
		return nil, err
	}
	row, err := s.q.UpsertReport(ctx, sqlc.UpsertReportParams{ID: r.ID, ClientID: r.ClientID, Month: r.Month.Time, Data: data, Summary: r.Summary, AiModel: r.AIModel, CreatedBy: r.CreatedBy})
	return one(row, err, toReport)
}

func (s *Store) GetReport(ctx context.Context, clientID, id string) (*domain.Report, error) {
	r, err := s.q.GetReport(ctx, sqlc.GetReportParams{ID: id, ClientID: clientID})
	return one(r, err, toReport)
}

func (s *Store) ListReports(ctx context.Context, clientID string) ([]domain.Report, error) {
	rows, err := s.q.ListReports(ctx, clientID)
	return many(rows, err, toReport)
}

// ─── Maintenance ─────────────────────────────────────────────

func (s *Store) DeleteClientData(ctx context.Context, clientID string) error {
	return s.WithTx(ctx, func(tx domain.Store) error {
		q := tx.(*Store).q
		for _, del := range []func(context.Context, string) error{q.DeleteClientPlans, q.DeleteClientAssets, q.DeleteClientFolders, q.DeleteClientReports} {
			if err := del(ctx, clientID); err != nil {
				return err
			}
		}
		return q.DeleteOrphanReminders(ctx)
	})
}

func (s *Store) ForgetUser(ctx context.Context, userID string) error {
	uid := &userID
	if err := s.q.ForgetItemAssignees(ctx, uid); err != nil {
		return err
	}
	return s.q.ForgetDesignAssignee(ctx, uid)
}

// ─── Dashboard ───────────────────────────────────────────────

func (s *Store) DashboardItems(ctx context.Context, clientID string, f domain.DashboardFilter) ([]domain.DashItem, error) {
	rows, err := s.q.DashboardItems(ctx, sqlc.DashboardItemsParams{
		ClientID: clientID, FromDay: f.From, ToDay: f.To, Platform: strOrNil(string(f.Platform)), Format: strOrNil(string(f.Format)),
		Pillar: strOrNil(f.Pillar), PlanID: strOrNil(f.PlanID), AssigneeID: strOrNil(f.AssigneeID),
	})
	return many(rows, err, func(r sqlc.DashboardItemsRow) domain.DashItem {
		return domain.DashItem{
			ID: r.ID, PlanID: r.PlanID, PlannedDate: r.PlannedDate, Platform: domain.Platform(r.Platform), Format: domain.Format(r.Format),
			Pillar: r.Pillar, Title: r.Title, Status: domain.ItemStatus(r.Status), RevisionRound: r.RevisionRound,
			CopywriterID: r.CopywriterID, DesignerID: r.DesignerID, CopyDue: r.CopyDue, DesignDue: r.DesignDue, ReviewDue: r.ReviewDue,
			CreatedAt: r.CreatedAt, CopyDoneAt: r.CopyDoneAt, DesignDoneAt: r.DesignDoneAt, ApprovedAt: r.ApprovedAt, PublishedAt: r.PublishedAt,
			Permalink: r.Permalink, ExternalIDs: r.ExternalIds, ThumbnailFileID: r.ThumbnailFileID,
		}
	})
}

func (s *Store) DashboardDesignTasks(ctx context.Context, clientID string, f domain.DashboardFilter) ([]domain.DashDesignTask, error) {
	rows, err := s.q.DashboardDesignTasks(ctx, sqlc.DashboardDesignTasksParams{
		ClientID: clientID, FromDay: f.From, ToDay: f.To, Platform: strOrNil(string(f.Platform)), Format: strOrNil(string(f.Format)),
		Pillar: strOrNil(f.Pillar), PlanID: strOrNil(f.PlanID), AssigneeID: strOrNil(f.AssigneeID),
	})
	return many(rows, err, func(r sqlc.DashboardDesignTasksRow) domain.DashDesignTask {
		t := domain.DashDesignTask{ID: r.ID, ItemID: r.ItemID, DesignerID: r.DesignerID, Status: domain.DesignStatus(r.Status),
			Versions: r.CurrentVersion, CreatedAt: r.CreatedAt, ChangesRequested: r.ChangesRequested}
		// sqlc cannot type a nullable aggregate from a LEFT JOIN subquery.
		if at, ok := r.ApprovedAt.(time.Time); ok {
			t.ApprovedAt = &at
		}
		return t
	})
}

func (s *Store) DashboardAssetVersions(ctx context.Context, clientID string, from, to time.Time) ([]domain.DashAssetVersion, error) {
	rows, err := s.q.DashboardAssetVersions(ctx, sqlc.DashboardAssetVersionsParams{ClientID: clientID, FromAt: from, ToAt: to})
	return many(rows, err, func(r sqlc.DashboardAssetVersionsRow) domain.DashAssetVersion {
		return domain.DashAssetVersion{AssetID: r.AssetID, Name: r.Name, Kind: domain.AssetKind(r.Kind), CurrentVersion: r.CurrentVersion, Version: r.Version, CreatedAt: r.CreatedAt}
	})
}

func (s *Store) ClientPillars(ctx context.Context, clientID string) ([]string, error) {
	return s.q.ClientPillars(ctx, clientID)
}

func (s *Store) ItemMetrics(ctx context.Context, itemIDs []string) (map[string]domain.PostMetrics, error) {
	out := map[string]domain.PostMetrics{}
	if len(itemIDs) == 0 {
		return out, nil
	}
	rows, err := s.q.ListItemMetrics(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.ItemID] = domain.PostMetrics{Likes: r.Likes, Comments: r.Comments, Shares: r.Shares, Saves: r.Saves, Reach: r.Reach, Views: r.Views}
	}
	return out, nil
}

func (s *Store) UpsertItemMetrics(ctx context.Context, itemID string, m domain.PostMetrics) error {
	return s.q.UpsertItemMetrics(ctx, sqlc.UpsertItemMetricsParams{ItemID: itemID, Likes: m.Likes, Comments: m.Comments, Shares: m.Shares, Saves: m.Saves, Reach: m.Reach, Views: m.Views})
}
