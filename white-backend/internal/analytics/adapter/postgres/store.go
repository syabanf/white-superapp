// Package postgres implements the analytics ports on top of sqlc queries.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/analytics/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
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

// wrap turns a unique violation into *domain.ErrUnique.
func wrap(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ErrUnique{Constraint: pgErr.ConstraintName}
	}
	return err
}

func i32(n int) int32 { return int32(n) }

func i32p(n *int) *int32 {
	if n == nil {
		return nil
	}
	v := int32(*n)
	return &v
}

func intp(n *int32) *int {
	if n == nil {
		return nil
	}
	v := int(*n)
	return &v
}

func day(t time.Time) time.Time { return t.UTC() }

// ─── Social accounts ─────────────────────────────────────────

func toAccount(a sqlc.AnalyticsSocialAccount) domain.SocialAccount {
	return domain.SocialAccount{ID: a.ID, ClientID: a.ClientID, Platform: domain.Platform(a.Platform), ExternalID: a.ExternalID, Username: a.Username, DisplayName: a.DisplayName, AvatarURL: a.AvatarUrl, Biography: a.Biography, IsCompetitor: a.IsCompetitor, ConnectionID: a.ConnectionID, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
}

func (s *Store) ListSocialAccounts(ctx context.Context, clientID string, includeCompetitors bool, platform *domain.Platform) ([]domain.SocialAccount, error) {
	var p *sqlc.AnalyticsSocialPlatform
	if platform != nil {
		v := sqlc.AnalyticsSocialPlatform(*platform)
		p = &v
	}
	rows, err := s.q.ListSocialAccounts(ctx, sqlc.ListSocialAccountsParams{ClientID: clientID, IncludeCompetitors: includeCompetitors, Platform: p})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SocialAccount, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAccount(r))
	}
	return out, nil
}

func (s *Store) GetSocialAccount(ctx context.Context, id string) (*domain.SocialAccount, error) {
	r, err := s.q.GetSocialAccount(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a := toAccount(r)
	return &a, nil
}

func (s *Store) FindCompetitor(ctx context.Context, clientID, username string) (*domain.SocialAccount, error) {
	r, err := s.q.FindCompetitorByUsername(ctx, sqlc.FindCompetitorByUsernameParams{ClientID: clientID, Username: username})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a := toAccount(r)
	return &a, nil
}

func (s *Store) InsertSocialAccount(ctx context.Context, a domain.SocialAccount) (*domain.SocialAccount, error) {
	r, err := s.q.InsertSocialAccount(ctx, sqlc.InsertSocialAccountParams{ID: a.ID, ClientID: a.ClientID, Platform: sqlc.AnalyticsSocialPlatform(a.Platform), ExternalID: a.ExternalID, Username: a.Username, DisplayName: a.DisplayName, AvatarUrl: a.AvatarURL, Biography: a.Biography, IsCompetitor: a.IsCompetitor, ConnectionID: a.ConnectionID})
	if err != nil {
		return nil, wrap(err)
	}
	out := toAccount(r)
	return &out, nil
}

func (s *Store) UpdateSocialAccountProfile(ctx context.Context, id string, p domain.ProfilePatch) error {
	return s.q.UpdateSocialAccountProfile(ctx, sqlc.UpdateSocialAccountProfileParams{ID: id, DisplayName: p.DisplayName, Username: p.Username, SetBiography: p.SetBiography, Biography: p.Biography, AvatarUrl: p.AvatarURL})
}

func (s *Store) DeleteSocialAccount(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteSocialAccount(ctx, id)
	return n > 0, err
}

// ─── Snapshots ───────────────────────────────────────────────

func toSnapshot(r sqlc.AnalyticsSocialSnapshot) domain.SocialSnapshot {
	return domain.SocialSnapshot{ID: r.ID, AccountID: r.SocialAccountID, Date: day(r.Date), Followers: int(r.Followers), Following: intp(r.Following), MediaCount: int(r.MediaCount), Reach: intp(r.Reach), Impressions: intp(r.Impressions), ProfileViews: intp(r.ProfileViews), WebsiteClicks: intp(r.WebsiteClicks)}
}

func (s *Store) LatestFollowers(ctx context.Context, accountIDs []string) (map[string]int, error) {
	rows, err := s.q.LatestFollowersByAccount(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, r := range rows {
		out[r.SocialAccountID] = int(r.Followers)
	}
	return out, nil
}

func (s *Store) ListSnapshots(ctx context.Context, accountIDs []string, from, to time.Time) ([]domain.SocialSnapshot, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := s.q.ListSnapshots(ctx, sqlc.ListSnapshotsParams{Column1: accountIDs, Date: from, Date_2: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SocialSnapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, toSnapshot(r))
	}
	return out, nil
}

func (s *Store) ListRecentSnapshots(ctx context.Context, accountID string, limit int) ([]domain.SocialSnapshot, error) {
	rows, err := s.q.ListRecentSnapshots(ctx, sqlc.ListRecentSnapshotsParams{SocialAccountID: accountID, Limit: i32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SocialSnapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, toSnapshot(r))
	}
	return out, nil
}

func (s *Store) UpsertSnapshot(ctx context.Context, c domain.SocialSnapshot, p domain.SnapshotPatch) error {
	id := c.ID
	if id == "" {
		id = ids.New()
	}
	return s.q.UpsertSnapshot(ctx, sqlc.UpsertSnapshotParams{
		ID: id, SocialAccountID: c.AccountID, Date: c.Date, Followers: i32(c.Followers), Following: i32p(c.Following), MediaCount: i32(c.MediaCount),
		Reach: i32p(c.Reach), Impressions: i32p(c.Impressions), ProfileViews: i32p(c.ProfileViews), WebsiteClicks: i32p(c.WebsiteClicks),
		UFollowers: i32p(p.Followers), UFollowing: i32p(p.Following), UMediaCount: i32p(p.MediaCount), UReach: i32p(p.Reach),
		UImpressions: i32p(p.Impressions), UProfileViews: i32p(p.ProfileViews), UWebsiteClicks: i32p(p.WebsiteClicks),
	})
}

func (s *Store) InsertSnapshotsIgnore(ctx context.Context, rows []domain.SocialSnapshot) error {
	for _, c := range rows {
		id := c.ID
		if id == "" {
			id = ids.New()
		}
		if err := s.q.InsertSnapshotIgnore(ctx, sqlc.InsertSnapshotIgnoreParams{ID: id, SocialAccountID: c.AccountID, Date: c.Date, Followers: i32(c.Followers), Following: i32p(c.Following), MediaCount: i32(c.MediaCount), Reach: i32p(c.Reach), Impressions: i32p(c.Impressions), ProfileViews: i32p(c.ProfileViews), WebsiteClicks: i32p(c.WebsiteClicks)}); err != nil {
			return err
		}
	}
	return nil
}

// ─── Posts ───────────────────────────────────────────────────

func toPost(r sqlc.AnalyticsSocialPost) domain.SocialPost {
	return domain.SocialPost{ID: r.ID, AccountID: r.SocialAccountID, ExternalID: r.ExternalID, Caption: r.Caption, MediaType: r.MediaType, ProductType: r.ProductType, Permalink: r.Permalink, MediaURL: r.MediaUrl, ThumbnailURL: r.ThumbnailUrl, PublishedAt: r.PublishedAt, Likes: int(r.Likes), Comments: int(r.Comments), Shares: int(r.Shares), Saves: int(r.Saves), Views: int(r.Views), Reach: int(r.Reach), Impressions: int(r.Impressions), SyncedAt: r.SyncedAt}
}

func (s *Store) ListPosts(ctx context.Context, accountIDs []string, from, toExclusive time.Time) ([]domain.SocialPost, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := s.q.ListPosts(ctx, sqlc.ListPostsParams{Column1: accountIDs, PublishedAt: from, PublishedAt_2: toExclusive})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SocialPost, 0, len(rows))
	for _, r := range rows {
		out = append(out, toPost(r))
	}
	return out, nil
}

func (s *Store) ListOwnPosts(ctx context.Context, clientID string, from, toExclusive time.Time) ([]domain.PostWithAccount, error) {
	rows, err := s.q.ListOwnPostsForClient(ctx, sqlc.ListOwnPostsForClientParams{ClientID: clientID, PublishedAt: from, PublishedAt_2: toExclusive})
	if err != nil {
		return nil, err
	}
	out := make([]domain.PostWithAccount, 0, len(rows))
	for _, r := range rows {
		p := toPost(sqlc.AnalyticsSocialPost{ID: r.ID, SocialAccountID: r.SocialAccountID, ExternalID: r.ExternalID, Caption: r.Caption, MediaType: r.MediaType, ProductType: r.ProductType, Permalink: r.Permalink, MediaUrl: r.MediaUrl, ThumbnailUrl: r.ThumbnailUrl, PublishedAt: r.PublishedAt, Likes: r.Likes, Comments: r.Comments, Shares: r.Shares, Saves: r.Saves, Views: r.Views, Reach: r.Reach, Impressions: r.Impressions, SyncedAt: r.SyncedAt})
		out = append(out, domain.PostWithAccount{SocialPost: p, Platform: domain.Platform(r.Platform), Username: r.Username})
	}
	return out, nil
}

func (s *Store) UpsertPost(ctx context.Context, c domain.SocialPost, p domain.PostPatch) error {
	id := c.ID
	if id == "" {
		id = ids.New()
	}
	return s.q.UpsertPost(ctx, sqlc.UpsertPostParams{
		ID: id, SocialAccountID: c.AccountID, ExternalID: c.ExternalID, Caption: c.Caption, MediaType: c.MediaType, ProductType: c.ProductType,
		Permalink: c.Permalink, MediaUrl: c.MediaURL, ThumbnailUrl: c.ThumbnailURL, PublishedAt: c.PublishedAt,
		Likes: i32(c.Likes), Comments: i32(c.Comments), Shares: i32(c.Shares), Saves: i32(c.Saves), Views: i32(c.Views), Reach: i32(c.Reach), Impressions: i32(c.Impressions),
		UCaption: p.Caption, UMediaType: p.MediaType, UProductType: p.ProductType, UPermalink: p.Permalink, UMediaUrl: p.MediaURL, UThumbnailUrl: p.ThumbnailURL,
		ULikes: i32p(p.Likes), UComments: i32p(p.Comments), UShares: i32p(p.Shares), USaves: i32p(p.Saves), UViews: i32p(p.Views), UReach: i32p(p.Reach), UImpressions: i32p(p.Impressions),
	})
}

func (s *Store) InsertPostsIgnore(ctx context.Context, rows []domain.SocialPost) error {
	for _, c := range rows {
		id := c.ID
		if id == "" {
			id = ids.New()
		}
		if err := s.q.InsertPostIgnore(ctx, sqlc.InsertPostIgnoreParams{ID: id, SocialAccountID: c.AccountID, ExternalID: c.ExternalID, Caption: c.Caption, MediaType: c.MediaType, ProductType: c.ProductType, Permalink: c.Permalink, MediaUrl: c.MediaURL, ThumbnailUrl: c.ThumbnailURL, PublishedAt: c.PublishedAt, Likes: i32(c.Likes), Comments: i32(c.Comments), Shares: i32(c.Shares), Saves: i32(c.Saves), Views: i32(c.Views), Reach: i32(c.Reach), Impressions: i32(c.Impressions)}); err != nil {
			return err
		}
	}
	return nil
}

// ─── SEO properties ──────────────────────────────────────────

func toProperty(p sqlc.AnalyticsSeoProperty) domain.SeoProperty {
	return domain.SeoProperty{ID: p.ID, ClientID: p.ClientID, SiteURL: p.SiteUrl, Ga4PropertyID: p.Ga4PropertyID, ConnectionID: p.ConnectionID, LocationCode: int(p.LocationCode), LanguageCode: p.LanguageCode, AuditCadence: domain.AuditCadence(p.AuditCadence), CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

func (s *Store) ListSeoProperties(ctx context.Context, clientID string) ([]domain.SeoProperty, error) {
	rows, err := s.q.ListSeoProperties(ctx, clientID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.SeoProperty, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProperty(r))
	}
	return out, nil
}

func (s *Store) GetSeoProperty(ctx context.Context, id string) (*domain.SeoProperty, error) {
	r, err := s.q.GetSeoProperty(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p := toProperty(r)
	return &p, nil
}

func (s *Store) InsertSeoProperty(ctx context.Context, p domain.SeoProperty) (*domain.SeoProperty, error) {
	r, err := s.q.InsertSeoProperty(ctx, sqlc.InsertSeoPropertyParams{ID: p.ID, ClientID: p.ClientID, SiteUrl: p.SiteURL, Ga4PropertyID: p.Ga4PropertyID, ConnectionID: p.ConnectionID})
	if err != nil {
		return nil, wrap(err)
	}
	out := toProperty(r)
	return &out, nil
}

func (s *Store) SetAuditCadence(ctx context.Context, id string, c domain.AuditCadence) error {
	return s.q.SetAuditCadence(ctx, sqlc.SetAuditCadenceParams{ID: id, AuditCadence: sqlc.AnalyticsAuditCadence(c)})
}

func (s *Store) DeleteSeoProperty(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteSeoProperty(ctx, id)
	return n > 0, err
}

// ─── SEO metrics ─────────────────────────────────────────────

func (s *Store) ListSeoDaily(ctx context.Context, propertyIDs []string, from, to time.Time) ([]domain.SeoDailyMetric, error) {
	if len(propertyIDs) == 0 {
		return nil, nil
	}
	rows, err := s.q.ListSeoDaily(ctx, sqlc.ListSeoDailyParams{Column1: propertyIDs, Date: from, Date_2: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SeoDailyMetric, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.SeoDailyMetric{PropertyID: r.PropertyID, Date: day(r.Date), Clicks: int(r.Clicks), Impressions: int(r.Impressions), CTR: r.Ctr, Position: r.Position})
	}
	return out, nil
}

func (s *Store) UpsertSeoDaily(ctx context.Context, rows []domain.SeoDailyMetric) error {
	for _, r := range rows {
		if err := s.q.UpsertSeoDaily(ctx, sqlc.UpsertSeoDailyParams{ID: ids.New(), PropertyID: r.PropertyID, Date: r.Date, Clicks: i32(r.Clicks), Impressions: i32(r.Impressions), Ctr: r.CTR, Position: r.Position}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SumSeoClicks(ctx context.Context, clientIDs []string, from, to time.Time) (int, error) {
	n, err := s.q.SumSeoClicksForClients(ctx, sqlc.SumSeoClicksForClientsParams{Column1: clientIDs, Date: from, Date_2: to})
	return int(n), err
}

func (s *Store) ListSeoDimension(ctx context.Context, propertyIDs []string, dim domain.SeoDimension, from, to time.Time) ([]domain.SeoDimensionMetric, error) {
	if len(propertyIDs) == 0 {
		return nil, nil
	}
	rows, err := s.q.ListSeoDimension(ctx, sqlc.ListSeoDimensionParams{Column1: propertyIDs, Dimension: sqlc.AnalyticsSeoDimension(dim), Date: from, Date_2: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SeoDimensionMetric, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.SeoDimensionMetric{PropertyID: r.PropertyID, Date: day(r.Date), Dimension: domain.SeoDimension(r.Dimension), Key: r.Key, Clicks: int(r.Clicks), Impressions: int(r.Impressions), CTR: r.Ctr, Position: r.Position})
	}
	return out, nil
}

func (s *Store) UpsertSeoDimension(ctx context.Context, rows []domain.SeoDimensionMetric) error {
	for _, r := range rows {
		if err := s.q.UpsertSeoDimension(ctx, sqlc.UpsertSeoDimensionParams{ID: ids.New(), PropertyID: r.PropertyID, Date: r.Date, Dimension: sqlc.AnalyticsSeoDimension(r.Dimension), Key: r.Key, Clicks: i32(r.Clicks), Impressions: i32(r.Impressions), Ctr: r.CTR, Position: r.Position}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListGa4Daily(ctx context.Context, propertyID string, from, to time.Time) ([]domain.Ga4DailyMetric, error) {
	rows, err := s.q.ListGa4Daily(ctx, sqlc.ListGa4DailyParams{PropertyID: propertyID, Date: from, Date_2: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Ga4DailyMetric, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Ga4DailyMetric{PropertyID: r.PropertyID, Date: day(r.Date), Sessions: int(r.Sessions), OrganicSessions: int(r.OrganicSessions), Users: int(r.Users), EngagedSessions: int(r.EngagedSessions), Conversions: int(r.Conversions), AvgEngagementSec: r.AvgEngagementSec})
	}
	return out, nil
}

func (s *Store) UpsertGa4Daily(ctx context.Context, rows []domain.Ga4DailyMetric) error {
	for _, r := range rows {
		if err := s.q.UpsertGa4Daily(ctx, sqlc.UpsertGa4DailyParams{ID: ids.New(), PropertyID: r.PropertyID, Date: r.Date, Sessions: i32(r.Sessions), OrganicSessions: i32(r.OrganicSessions), Users: i32(r.Users), EngagedSessions: i32(r.EngagedSessions), Conversions: i32(r.Conversions), AvgEngagementSec: r.AvgEngagementSec}); err != nil {
			return err
		}
	}
	return nil
}

// ─── Audits and crawls ───────────────────────────────────────

func toAudit(a sqlc.AnalyticsSeoAudit) domain.SeoAudit {
	return domain.SeoAudit{ID: a.ID, PropertyID: a.PropertyID, URL: a.Url, Strategy: domain.AuditStrategy(a.Strategy), RunAt: a.RunAt, Performance: int(a.Performance), Seo: int(a.Seo), Accessibility: int(a.Accessibility), BestPractices: int(a.BestPractices), LcpMs: a.LcpMs, InpMs: a.InpMs, Cls: a.Cls, FcpMs: a.FcpMs, TtfbMs: a.TtfbMs, TbtMs: a.TbtMs, SpeedIndexMs: a.SpeedIndexMs, Raw: a.Raw}
}

func (s *Store) LatestAudit(ctx context.Context, propertyIDs []string, strategy domain.AuditStrategy) (*domain.SeoAudit, error) {
	r, err := s.q.LatestAudit(ctx, sqlc.LatestAuditParams{Column1: propertyIDs, Strategy: sqlc.AnalyticsAuditStrategy(strategy)})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a := toAudit(r)
	return &a, nil
}

func (s *Store) LatestAuditAny(ctx context.Context, propertyID string) (*domain.SeoAudit, error) {
	r, err := s.q.LatestAuditAny(ctx, propertyID)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a := toAudit(r)
	return &a, nil
}

func (s *Store) ListMobileAuditHistory(ctx context.Context, propertyID string, limit int) ([]domain.SeoAudit, error) {
	rows, err := s.q.ListMobileAuditHistory(ctx, sqlc.ListMobileAuditHistoryParams{PropertyID: propertyID, Limit: i32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SeoAudit, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAudit(r))
	}
	return out, nil
}

func (s *Store) InsertAudit(ctx context.Context, a domain.SeoAudit) error {
	id := a.ID
	if id == "" {
		id = ids.New()
	}
	return s.q.InsertAudit(ctx, sqlc.InsertAuditParams{ID: id, PropertyID: a.PropertyID, Url: a.URL, Strategy: sqlc.AnalyticsAuditStrategy(a.Strategy), RunAt: a.RunAt, Performance: i32(a.Performance), Seo: i32(a.Seo), Accessibility: i32(a.Accessibility), BestPractices: i32(a.BestPractices), LcpMs: a.LcpMs, InpMs: a.InpMs, Cls: a.Cls, FcpMs: a.FcpMs, TtfbMs: a.TtfbMs, TbtMs: a.TbtMs, SpeedIndexMs: a.SpeedIndexMs, Raw: a.Raw})
}

func toCrawl(c sqlc.AnalyticsSeoCrawl) domain.SeoCrawl {
	return domain.SeoCrawl{ID: c.ID, PropertyID: c.PropertyID, StartedAt: c.StartedAt, FinishedAt: c.FinishedAt, Status: domain.JobStatus(c.Status), PagesCrawled: int(c.PagesCrawled), MaxPages: int(c.MaxPages), Error: c.Error}
}

func (s *Store) LatestCrawl(ctx context.Context, propertyIDs []string, status *domain.JobStatus) (*domain.SeoCrawl, error) {
	var st *sqlc.AnalyticsJobStatus
	if status != nil {
		v := sqlc.AnalyticsJobStatus(*status)
		st = &v
	}
	r, err := s.q.LatestCrawl(ctx, sqlc.LatestCrawlParams{Column1: propertyIDs, Status: st})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c := toCrawl(r)
	return &c, nil
}

func (s *Store) ListRecentSuccessCrawls(ctx context.Context, propertyID string, limit int) ([]domain.SeoCrawl, error) {
	rows, err := s.q.ListRecentSuccessCrawls(ctx, sqlc.ListRecentSuccessCrawlsParams{PropertyID: propertyID, Limit: i32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SeoCrawl, 0, len(rows))
	for _, r := range rows {
		out = append(out, toCrawl(r))
	}
	return out, nil
}

func (s *Store) InsertCrawl(ctx context.Context, c domain.SeoCrawl) error {
	return s.q.InsertCrawl(ctx, sqlc.InsertCrawlParams{ID: c.ID, PropertyID: c.PropertyID, StartedAt: c.StartedAt, Status: sqlc.AnalyticsJobStatus(c.Status), MaxPages: i32(c.MaxPages)})
}

func (s *Store) UpdateCrawl(ctx context.Context, id string, status domain.JobStatus, finishedAt time.Time, pagesCrawled int, errMsg *string) error {
	return s.q.UpdateCrawl(ctx, sqlc.UpdateCrawlParams{ID: id, Status: sqlc.AnalyticsJobStatus(status), FinishedAt: &finishedAt, PagesCrawled: i32(pagesCrawled), Error: errMsg})
}

func (s *Store) InsertCrawlPages(ctx context.Context, rows []domain.SeoCrawlPage) error {
	params := make([]sqlc.InsertCrawlPageParams, 0, len(rows))
	for _, p := range rows {
		id := p.ID
		if id == "" {
			id = ids.New()
		}
		params = append(params, sqlc.InsertCrawlPageParams{ID: id, CrawlID: p.CrawlID, Url: p.URL, StatusCode: i32(p.StatusCode), Title: p.Title, MetaDescription: p.MetaDescription, H1Count: i32(p.H1Count), WordCount: i32(p.WordCount), Canonical: p.Canonical, Indexable: p.Indexable, ImagesTotal: i32(p.ImagesTotal), ImagesMissingAlt: i32(p.ImagesMissingAlt), InternalLinks: i32(p.InternalLinks), ExternalLinks: i32(p.ExternalLinks), LoadMs: i32p(p.LoadMs), HasJsonLd: p.HasJsonLd, HasHreflang: p.HasHreflang})
	}
	_, err := s.q.InsertCrawlPage(ctx, params)
	return err
}

func (s *Store) InsertIssues(ctx context.Context, rows []domain.SeoIssue) error {
	params := make([]sqlc.InsertIssueParams, 0, len(rows))
	for _, i := range rows {
		id := i.ID
		if id == "" {
			id = ids.New()
		}
		params = append(params, sqlc.InsertIssueParams{ID: id, CrawlID: i.CrawlID, Severity: sqlc.AnalyticsIssueSeverity(i.Severity), Code: i.Code, Message: i.Message, Url: i.URL})
	}
	_, err := s.q.InsertIssue(ctx, params)
	return err
}

func (s *Store) ListCrawlPages(ctx context.Context, crawlID string) ([]domain.SeoCrawlPage, error) {
	rows, err := s.q.ListCrawlPages(ctx, crawlID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.SeoCrawlPage, 0, len(rows))
	for _, p := range rows {
		out = append(out, domain.SeoCrawlPage{ID: p.ID, CrawlID: p.CrawlID, URL: p.Url, StatusCode: int(p.StatusCode), Title: p.Title, MetaDescription: p.MetaDescription, H1Count: int(p.H1Count), WordCount: int(p.WordCount), Canonical: p.Canonical, Indexable: p.Indexable, ImagesTotal: int(p.ImagesTotal), ImagesMissingAlt: int(p.ImagesMissingAlt), InternalLinks: int(p.InternalLinks), ExternalLinks: int(p.ExternalLinks), LoadMs: intp(p.LoadMs), HasJsonLd: p.HasJsonLd, HasHreflang: p.HasHreflang})
	}
	return out, nil
}

func (s *Store) ListIssues(ctx context.Context, crawlID string) ([]domain.SeoIssue, error) {
	rows, err := s.q.ListIssues(ctx, crawlID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.SeoIssue, 0, len(rows))
	for _, i := range rows {
		out = append(out, domain.SeoIssue{ID: i.ID, CrawlID: i.CrawlID, Severity: domain.IssueSeverity(i.Severity), Code: i.Code, Message: i.Message, URL: i.Url})
	}
	return out, nil
}

func (s *Store) CountIssues(ctx context.Context, crawlIDs []string) (map[string]domain.SeverityCounts, error) {
	out := map[string]domain.SeverityCounts{}
	if len(crawlIDs) == 0 {
		return out, nil
	}
	rows, err := s.q.CountIssuesBySeverity(ctx, crawlIDs)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		c := out[r.CrawlID]
		switch domain.IssueSeverity(r.Severity) {
		case domain.SeverityError:
			c.Errors = int(r.N)
		case domain.SeverityWarning:
			c.Warnings = int(r.N)
		case domain.SeverityNotice:
			c.Notices = int(r.N)
		}
		out[r.CrawlID] = c
	}
	return out, nil
}

// ─── Ads ─────────────────────────────────────────────────────

func toAdAccount(a sqlc.AnalyticsAdAccount) domain.AdAccount {
	return domain.AdAccount{ID: a.ID, ClientID: a.ClientID, ExternalID: a.ExternalID, Name: a.Name, Currency: a.Currency, ConnectionID: a.ConnectionID, LastSyncedAt: a.LastSyncedAt, CreatedAt: a.CreatedAt}
}

func (s *Store) ListAdAccounts(ctx context.Context, clientID string) ([]domain.AdAccount, error) {
	rows, err := s.q.ListAdAccounts(ctx, clientID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AdAccount, 0, len(rows))
	for _, r := range rows {
		out = append(out, toAdAccount(r))
	}
	return out, nil
}

func (s *Store) GetAdAccount(ctx context.Context, id string) (*domain.AdAccount, error) {
	r, err := s.q.GetAdAccount(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a := toAdAccount(r)
	return &a, nil
}

func (s *Store) InsertAdAccount(ctx context.Context, a domain.AdAccount) (*domain.AdAccount, error) {
	r, err := s.q.InsertAdAccount(ctx, sqlc.InsertAdAccountParams{ID: a.ID, ClientID: a.ClientID, ExternalID: a.ExternalID, Name: a.Name, Currency: a.Currency, ConnectionID: a.ConnectionID})
	if err != nil {
		return nil, wrap(err)
	}
	out := toAdAccount(r)
	return &out, nil
}

func (s *Store) DeleteAdAccount(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteAdAccount(ctx, id)
	return n > 0, err
}

func (s *Store) TouchAdAccounts(ctx context.Context, idList []string, at time.Time) error {
	return s.q.TouchAdAccounts(ctx, sqlc.TouchAdAccountsParams{Column1: idList, LastSyncedAt: &at})
}

func (s *Store) ListCampaigns(ctx context.Context, accountIDs []string) ([]domain.AdCampaign, error) {
	rows, err := s.q.ListCampaigns(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	out := make([]domain.AdCampaign, 0, len(rows))
	for _, c := range rows {
		out = append(out, domain.AdCampaign{ID: c.ID, AccountID: c.AdAccountID, ExternalID: c.ExternalID, Name: c.Name, Objective: c.Objective, Status: c.Status, DailyBudget: c.DailyBudget, StartTime: c.StartTime})
	}
	return out, nil
}

func (s *Store) CountCampaigns(ctx context.Context, accountIDs []string) (int, error) {
	n, err := s.q.CountCampaigns(ctx, accountIDs)
	return int(n), err
}

func (s *Store) UpsertCampaign(ctx context.Context, c domain.AdCampaign, fullUpdate bool) (string, error) {
	id := c.ID
	if id == "" {
		id = ids.New()
	}
	return s.q.UpsertCampaign(ctx, sqlc.UpsertCampaignParams{ID: id, AdAccountID: c.AccountID, ExternalID: c.ExternalID, Name: c.Name, Objective: c.Objective, Status: c.Status, DailyBudget: c.DailyBudget, StartTime: c.StartTime, FullUpdate: fullUpdate})
}

func (s *Store) ListAdSets(ctx context.Context, accountIDs []string, campaignID *string) ([]domain.AdSet, error) {
	rows, err := s.q.ListAdSets(ctx, sqlc.ListAdSetsParams{Column1: accountIDs, CampaignID: campaignID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.AdSet, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AdSet{ID: r.ID, CampaignID: r.CampaignID, ExternalID: r.ExternalID, Name: r.Name, Status: r.Status, DailyBudget: r.DailyBudget})
	}
	return out, nil
}

func (s *Store) UpsertAdSet(ctx context.Context, a domain.AdSet, fullUpdate bool) (string, error) {
	id := a.ID
	if id == "" {
		id = ids.New()
	}
	return s.q.UpsertAdSet(ctx, sqlc.UpsertAdSetParams{ID: id, CampaignID: a.CampaignID, ExternalID: a.ExternalID, Name: a.Name, Status: a.Status, DailyBudget: a.DailyBudget, FullUpdate: fullUpdate})
}

func (s *Store) ListAds(ctx context.Context, accountIDs []string, campaignID *string) ([]domain.Ad, error) {
	rows, err := s.q.ListAds(ctx, sqlc.ListAdsParams{Column1: accountIDs, CampaignID: campaignID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Ad, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Ad{ID: r.ID, AdSetID: r.AdSetID, ExternalID: r.ExternalID, Name: r.Name, Status: r.Status, ThumbnailURL: r.ThumbnailUrl})
	}
	return out, nil
}

func (s *Store) UpsertAd(ctx context.Context, a domain.Ad, fullUpdate bool) (string, error) {
	id := a.ID
	if id == "" {
		id = ids.New()
	}
	return s.q.UpsertAd(ctx, sqlc.UpsertAdParams{ID: id, AdSetID: a.AdSetID, ExternalID: a.ExternalID, Name: a.Name, Status: a.Status, ThumbnailUrl: a.ThumbnailURL, FullUpdate: fullUpdate})
}

func toInsight(r sqlc.AnalyticsAdDailyInsight) domain.AdDailyInsight {
	return domain.AdDailyInsight{ID: r.ID, AccountID: r.AdAccountID, Date: day(r.Date), CampaignID: r.CampaignID, AdSetID: r.AdSetID, AdID: r.AdID, Spend: r.Spend, Impressions: int(r.Impressions), Reach: int(r.Reach), Clicks: int(r.Clicks), LinkClicks: int(r.LinkClicks), Results: int(r.Results), ResultType: r.ResultType, PurchaseValue: r.PurchaseValue}
}

func (s *Store) ListInsights(ctx context.Context, accountIDs []string, from, to time.Time, campaignID *string) ([]domain.AdDailyInsight, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := s.q.ListInsights(ctx, sqlc.ListInsightsParams{Column1: accountIDs, Date: from, Date_2: to, CampaignID: campaignID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.AdDailyInsight, 0, len(rows))
	for _, r := range rows {
		out = append(out, toInsight(r))
	}
	return out, nil
}

func (s *Store) UpsertInsights(ctx context.Context, rows []domain.AdDailyInsight) error {
	for _, r := range rows {
		id := r.ID
		if id == "" {
			id = ids.New()
		}
		if err := s.q.UpsertInsight(ctx, sqlc.UpsertInsightParams{ID: id, AdAccountID: r.AccountID, Date: r.Date, CampaignID: r.CampaignID, AdSetID: r.AdSetID, AdID: r.AdID, Spend: r.Spend, Impressions: i32(r.Impressions), Reach: i32(r.Reach), Clicks: i32(r.Clicks), LinkClicks: i32(r.LinkClicks), Results: i32(r.Results), ResultType: r.ResultType, PurchaseValue: r.PurchaseValue}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SumAdSpend(ctx context.Context, clientIDs []string, from, to time.Time) (float64, error) {
	return s.q.SumAdSpendForClients(ctx, sqlc.SumAdSpendForClientsParams{Column1: clientIDs, Date: from, Date_2: to})
}

func (s *Store) ListDemographics(ctx context.Context, accountIDs []string, from, to time.Time, campaignID *string) ([]domain.AdDemographic, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := s.q.ListDemographics(ctx, sqlc.ListDemographicsParams{Column1: accountIDs, RangeFrom: from, RangeTo: to, CampaignID: campaignID})
	if err != nil {
		return nil, err
	}
	out := make([]domain.AdDemographic, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.AdDemographic{ID: r.ID, AccountID: r.AdAccountID, DateFrom: day(r.DateFrom), DateTo: day(r.DateTo), CampaignID: r.CampaignID, Age: r.Age, Gender: r.Gender, Spend: r.Spend, Impressions: int(r.Impressions), Reach: int(r.Reach), Results: int(r.Results)})
	}
	return out, nil
}

func (s *Store) UpsertDemographics(ctx context.Context, rows []domain.AdDemographic) error {
	for _, r := range rows {
		id := r.ID
		if id == "" {
			id = ids.New()
		}
		if err := s.q.UpsertDemographic(ctx, sqlc.UpsertDemographicParams{ID: id, AdAccountID: r.AccountID, DateFrom: r.DateFrom, DateTo: r.DateTo, CampaignID: r.CampaignID, Age: r.Age, Gender: r.Gender, Spend: r.Spend, Impressions: i32(r.Impressions), Reach: i32(r.Reach), Results: i32(r.Results)}); err != nil {
			return err
		}
	}
	return nil
}

// ─── Sync jobs ───────────────────────────────────────────────

func toJob(j sqlc.AnalyticsSyncJob) domain.SyncJob {
	return domain.SyncJob{ID: j.ID, ClientID: j.ClientID, Kind: j.Kind, Status: domain.JobStatus(j.Status), StartedAt: j.StartedAt, FinishedAt: j.FinishedAt, Message: j.Message, Error: j.Error}
}

func (s *Store) InsertSyncJob(ctx context.Context, j domain.SyncJob) (*domain.SyncJob, error) {
	id := j.ID
	if id == "" {
		id = ids.New()
	}
	r, err := s.q.InsertSyncJob(ctx, sqlc.InsertSyncJobParams{ID: id, ClientID: j.ClientID, Kind: j.Kind, Status: sqlc.AnalyticsJobStatus(j.Status), StartedAt: j.StartedAt, FinishedAt: j.FinishedAt, Message: j.Message, Error: j.Error})
	if err != nil {
		return nil, err
	}
	out := toJob(r)
	return &out, nil
}

func (s *Store) UpdateSyncJob(ctx context.Context, id string, status domain.JobStatus, finishedAt time.Time, message, errMsg *string) error {
	return s.q.UpdateSyncJob(ctx, sqlc.UpdateSyncJobParams{ID: id, Status: sqlc.AnalyticsJobStatus(status), FinishedAt: &finishedAt, Message: message, Error: errMsg})
}

func (s *Store) LatestSyncJob(ctx context.Context, clientID string, kinds []string) (*domain.SyncJob, error) {
	r, err := s.q.LatestSyncJob(ctx, sqlc.LatestSyncJobParams{ClientID: clientID, Column2: kinds})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	j := toJob(r)
	return &j, nil
}

func (s *Store) ListSyncJobs(ctx context.Context, clientID string, limit int) ([]domain.SyncJob, error) {
	rows, err := s.q.ListSyncJobs(ctx, sqlc.ListSyncJobsParams{ClientID: clientID, Limit: i32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SyncJob, 0, len(rows))
	for _, r := range rows {
		out = append(out, toJob(r))
	}
	return out, nil
}

func (s *Store) LatestSyncByClients(ctx context.Context, clientIDs []string) (map[string]time.Time, error) {
	rows, err := s.q.LatestSyncByClients(ctx, clientIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]time.Time, len(rows))
	for _, r := range rows {
		out[r.ClientID] = r.StartedAt
	}
	return out, nil
}

func (s *Store) DeleteSyncJobsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.q.DeleteSyncJobsBefore(ctx, cutoff)
}

func (s *Store) PurgeClient(ctx context.Context, clientID string) error {
	if err := s.q.DeleteSocialAccountsByClient(ctx, clientID); err != nil {
		return err
	}
	if err := s.q.DeleteSeoPropertiesByClient(ctx, clientID); err != nil {
		return err
	}
	if err := s.q.DeleteAdAccountsByClient(ctx, clientID); err != nil {
		return err
	}
	return s.q.DeleteSyncJobsByClient(ctx, clientID)
}
