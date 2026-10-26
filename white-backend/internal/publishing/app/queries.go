package app

import (
	"context"
	"sort"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/publishing/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
)

// Queries are the read models of queries.ts. Every method checks access.
type Queries struct{ Deps }

func NewQueries(d Deps) *Queries { return &Queries{d} }

type TargetView struct {
	domain.PostTarget
	Username    string
	DisplayName string
	AvatarURL   *string
}

type CommentView struct {
	domain.PostComment
	UserName *string
}

type ActivityView struct {
	domain.PostActivity
	UserName *string
}

type PostDetail struct {
	Post       domain.ContentPost
	CreatedBy  *string
	ApprovedBy *string
	Targets    []TargetView
	Media      []domain.MediaAsset
	Comments   []CommentView
	Activities []ActivityView
}

func (q *Queries) userName(ctx context.Context, cache map[string]*string, id *string) *string {
	if id == nil {
		return nil
	}
	if v, ok := cache[*id]; ok {
		return v
	}
	var out *string
	if name := q.Identity.UserName(ctx, *id); name != "" {
		out = &name
	}
	cache[*id] = out
	return out
}

func (q *Queries) GetPost(ctx context.Context, userID, clientID, postID string) (*PostDetail, error) {
	if _, err := q.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	post, err := q.Store.GetPostInClient(ctx, clientID, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, apperr.New(apperr.NotFound, msgNotFound)
	}
	targets, err := q.Store.ListTargets(ctx, postID)
	if err != nil {
		return nil, err
	}
	media, err := q.Store.ListPostMedia(ctx, postID)
	if err != nil {
		return nil, err
	}
	comments, err := q.Store.ListComments(ctx, postID)
	if err != nil {
		return nil, err
	}
	activities, err := q.Store.ListActivities(ctx, postID)
	if err != nil {
		return nil, err
	}
	accounts, err := q.Analytics.OwnAccounts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	byID := accountsByID(accounts)
	names := map[string]*string{}
	d := &PostDetail{Post: *post, Media: media, CreatedBy: q.userName(ctx, names, post.CreatedByID), ApprovedBy: q.userName(ctx, names, post.ApprovedByID)}
	for _, t := range targets {
		v := TargetView{PostTarget: t}
		if acc, ok := byID[t.SocialAccountID]; ok {
			v.Platform, v.Username, v.DisplayName, v.AvatarURL = acc.Platform, acc.Username, acc.DisplayName, acc.AvatarURL
		}
		d.Targets = append(d.Targets, v)
	}
	sort.SliceStable(d.Targets, func(i, j int) bool { return d.Targets[i].Platform < d.Targets[j].Platform })
	for _, c := range comments {
		d.Comments = append(d.Comments, CommentView{PostComment: c, UserName: q.userName(ctx, names, c.UserID)})
	}
	for _, a := range activities {
		d.Activities = append(d.Activities, ActivityView{PostActivity: a, UserName: q.userName(ctx, names, a.UserID)})
	}
	return d, nil
}

type PostRow struct {
	domain.PostSummaryRow
	Author *string
}

type PostList struct {
	Rows   []PostRow
	Counts map[domain.PostStatus]int32
}

// ListPosts returns the tab's rows plus counts for every status.
func (q *Queries) ListPosts(ctx context.Context, userID, clientID string, tab domain.PostsTab) (*PostList, error) {
	if _, err := q.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	if !domain.IsPostsTab(string(tab)) {
		tab = domain.TabAll
	}
	rows, err := q.Store.ListPosts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	statuses := make([]domain.PostStatus, 0, len(rows))
	names := map[string]*string{}
	out := &PostList{Rows: []PostRow{}}
	for _, r := range rows {
		statuses = append(statuses, r.Post.Status)
		if !domain.TabIncludes(tab, r.Post.Status) {
			continue
		}
		out.Rows = append(out.Rows, PostRow{PostSummaryRow: r, Author: q.userName(ctx, names, r.Post.CreatedByID)})
	}
	out.Counts = domain.PostStatusCounts(statuses)
	return out, nil
}

type CalendarResult struct {
	Posts []domain.PostSummaryRow
	From  time.Time
	To    time.Time
}

// Calendar lists posts placed on the client-timezone days fromDay..toDay
// (inclusive); the UTC bounds come from rangeForDays.
func (q *Queries) Calendar(ctx context.Context, userID, clientID, fromDay, toDay string) (*CalendarResult, error) {
	access, err := q.access(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	if !domain.IsValidDateKey(fromDay) || !domain.IsValidDateKey(toDay) || toDay < fromDay {
		return nil, apperr.New(apperr.Validation, "Rentang tanggal tidak valid.")
	}
	days := domain.DaysBetween(fromDay, toDay)
	if len(days) > 62 {
		return nil, apperr.New(apperr.Validation, "Rentang tanggal maksimal 62 hari.")
	}
	from, to, err := domain.RangeForDays(days, access.Timezone)
	if err != nil {
		return nil, apperr.Wrap(apperr.Validation, "Rentang tanggal tidak valid.", err)
	}
	rows, err := q.Store.ListCalendarPosts(ctx, clientID, from, to)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Post.Instant().Before(rows[j].Post.Instant()) })
	return &CalendarResult{Posts: rows, From: from, To: to}, nil
}

func (q *Queries) Stats(ctx context.Context, userID, clientID string) (*domain.PublishingStats, error) {
	if _, err := q.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	return q.Store.GetPublishingStats(ctx, clientID, q.now())
}

type BestTimeSlot struct {
	Day         int
	Hour        int
	At          time.Time
	FromHistory bool
}

// bestTimesHistory is how many recent synced posts feed the heatmap.
const bestTimesHistory = 400

// BestTimes ranks the client's posting history (defaults when thin) and
// resolves each slot to its next occurrence at least 5 minutes ahead.
func (q *Queries) BestTimes(ctx context.Context, userID, clientID, timeZone string) ([]BestTimeSlot, error) {
	access, err := q.access(ctx, userID, clientID)
	if err != nil {
		return nil, err
	}
	if timeZone == "" {
		timeZone = access.Timezone
	}
	posts, err := q.Analytics.SyncedPosts(ctx, clientID, nil, bestTimesHistory)
	if err != nil {
		return nil, err
	}
	engaged := make([]domain.EngagedPost, 0, len(posts))
	for _, p := range posts {
		engaged = append(engaged, domain.EngagedPost{PublishedAt: p.PublishedAt, Engagements: float64(p.Engagements)})
	}
	best := domain.BestPostingTimes(domain.PostingHeatmap(engaged, timeZone), 3, 2)
	slots := make([]domain.Slot, 0, len(best))
	for _, b := range best {
		slots = append(slots, domain.Slot{Day: b.Day, Hour: b.Hour})
	}
	ats := domain.NextOccurrences(slots, timeZone, q.now(), domain.MinScheduleLead)
	out := make([]BestTimeSlot, 0, len(best))
	for i, b := range best {
		out = append(out, BestTimeSlot{Day: b.Day, Hour: b.Hour, At: ats[i], FromHistory: b.Score > 0})
	}
	return out, nil
}

func (q *Queries) SyncJobs(ctx context.Context, userID, clientID string, limit int32) ([]domain.SyncJob, error) {
	if _, err := q.access(ctx, userID, clientID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return q.Store.ListSyncJobs(ctx, clientID, limit)
}
