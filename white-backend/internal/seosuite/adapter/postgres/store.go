// Package postgres implements the seosuite ports on top of sqlc queries.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/seosuite/domain"
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

// batchErr collects the first failure of a pgx batch.
type batchErr struct{ err error }

func (b *batchErr) collect(_ int, err error) {
	if b.err == nil && err != nil {
		b.err = wrap(err)
	}
}

// ─── Keywords ─────────────────────────────────────────────────

func toKeyword(k sqlc.SeosuiteTrackedKeyword) domain.TrackedKeyword {
	var intent *domain.Intent
	if k.Intent != nil {
		v := domain.Intent(*k.Intent)
		intent = &v
	}
	tags := k.Tags
	if tags == nil {
		tags = []string{}
	}
	return domain.TrackedKeyword{
		ID: k.ID, ClientID: k.ClientID, PropertyID: k.PropertyID, Keyword: k.Keyword,
		LocationCode: k.LocationCode, LanguageCode: k.LanguageCode, Device: domain.Device(k.Device), Tags: tags,
		TargetURL: k.TargetUrl, Volume: k.Volume, Difficulty: k.Difficulty, CPC: k.Cpc, Intent: intent,
		CreatedAt: k.CreatedAt, UpdatedAt: k.UpdatedAt,
	}
}

func toKeywords(rows []sqlc.SeosuiteTrackedKeyword) []domain.TrackedKeyword {
	out := make([]domain.TrackedKeyword, 0, len(rows))
	for _, r := range rows {
		out = append(out, toKeyword(r))
	}
	return out
}

func (s *Store) ListKeywords(ctx context.Context, propertyID string) ([]domain.TrackedKeyword, error) {
	rows, err := s.q.ListKeywords(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	return toKeywords(rows), nil
}

func (s *Store) ListKeywordsByIDs(ctx context.Context, propertyID string, ids []string) ([]domain.TrackedKeyword, error) {
	rows, err := s.q.ListKeywordsByIDs(ctx, sqlc.ListKeywordsByIDsParams{PropertyID: propertyID, Ids: ids})
	if err != nil {
		return nil, err
	}
	return toKeywords(rows), nil
}

func (s *Store) ListKeywordIDs(ctx context.Context, propertyID string) ([]string, error) {
	return s.q.ListKeywordIDs(ctx, propertyID)
}

func (s *Store) GetKeyword(ctx context.Context, id string) (*domain.TrackedKeyword, error) {
	k, err := s.q.GetKeyword(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := toKeyword(k)
	return &out, nil
}

func (s *Store) InsertKeywords(ctx context.Context, rows []domain.TrackedKeyword) error {
	params := make([]sqlc.InsertKeywordParams, 0, len(rows))
	for _, k := range rows {
		var intent *sqlc.SeosuiteKeywordIntent
		if k.Intent != nil {
			v := sqlc.SeosuiteKeywordIntent(*k.Intent)
			intent = &v
		}
		tags := k.Tags
		if tags == nil {
			tags = []string{}
		}
		params = append(params, sqlc.InsertKeywordParams{
			ID: k.ID, ClientID: k.ClientID, PropertyID: k.PropertyID, Keyword: k.Keyword,
			LocationCode: k.LocationCode, LanguageCode: k.LanguageCode, Device: sqlc.SeosuiteDeviceType(k.Device),
			Tags: tags, Volume: k.Volume, Difficulty: k.Difficulty, Cpc: k.CPC, Intent: intent,
		})
	}
	var be batchErr
	s.q.InsertKeyword(ctx, params).Exec(be.collect)
	return be.err
}

func (s *Store) UpdateKeywordTags(ctx context.Context, id string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	return s.q.UpdateKeywordTags(ctx, sqlc.UpdateKeywordTagsParams{ID: id, Tags: tags})
}

func (s *Store) DeleteKeyword(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteKeyword(ctx, id)
	return n > 0, err
}

// ─── Rank snapshots ──────────────────────────────────────────

func toRankSnapshot(r sqlc.SeosuiteRankSnapshot) domain.RankSnapshot {
	out := domain.RankSnapshot{ID: r.ID, ClientID: r.ClientID, KeywordID: r.KeywordID, Date: r.Date.UTC(), Position: r.Position, URL: r.Url, SerpFeatures: r.SerpFeatures}
	if out.SerpFeatures == nil {
		out.SerpFeatures = []string{}
	}
	if len(r.TopResults) > 0 {
		_ = json.Unmarshal(r.TopResults, &out.TopResults)
	}
	return out
}

func (s *Store) ListRankSnapshots(ctx context.Context, keywordIDs []string, from, to time.Time) ([]domain.RankSnapshot, error) {
	rows, err := s.q.ListRankSnapshots(ctx, sqlc.ListRankSnapshotsParams{Ids: keywordIDs, FromDate: from, ToDate: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.RankSnapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, toRankSnapshot(r))
	}
	return out, nil
}

func (s *Store) ListRankPositionsOn(ctx context.Context, keywordIDs []string, date time.Time) (map[string]*int32, error) {
	rows, err := s.q.ListRankPositionsOn(ctx, sqlc.ListRankPositionsOnParams{Ids: keywordIDs, Date: date})
	if err != nil {
		return nil, err
	}
	out := make(map[string]*int32, len(rows))
	for _, r := range rows {
		out[r.KeywordID] = r.Position
	}
	return out, nil
}

func (s *Store) UpsertRankSnapshots(ctx context.Context, rows []domain.RankSnapshot) error {
	params := make([]sqlc.UpsertRankSnapshotParams, 0, len(rows))
	for _, r := range rows {
		top, err := json.Marshal(r.TopResults)
		if err != nil {
			return err
		}
		features := r.SerpFeatures
		if features == nil {
			features = []string{}
		}
		params = append(params, sqlc.UpsertRankSnapshotParams{ID: r.ID, ClientID: r.ClientID, KeywordID: r.KeywordID, Date: r.Date, Position: r.Position, Url: r.URL, SerpFeatures: features, TopResults: top})
	}
	var be batchErr
	s.q.UpsertRankSnapshot(ctx, params).Exec(be.collect)
	return be.err
}

func (s *Store) DeleteRankSnapshotsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.q.DeleteRankSnapshotsBefore(ctx, cutoff)
}

// ─── Research cache ──────────────────────────────────────────

func (s *Store) GetLatestResearch(ctx context.Context, clientID, seed string, locationCode int32, languageCode string) (*domain.KeywordResearch, error) {
	r, err := s.q.GetLatestResearch(ctx, sqlc.GetLatestResearchParams{ClientID: clientID, Seed: seed, LocationCode: locationCode, LanguageCode: languageCode})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := domain.KeywordResearch{ID: r.ID, ClientID: r.ClientID, Seed: r.Seed, LocationCode: r.LocationCode, LanguageCode: r.LanguageCode, Provider: r.Provider, FetchedAt: r.FetchedAt, Results: []domain.KeywordIdea{}}
	if err := json.Unmarshal(r.Results, &out.Results); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Store) InsertResearch(ctx context.Context, r domain.KeywordResearch) error {
	results, err := json.Marshal(r.Results)
	if err != nil {
		return err
	}
	return s.q.InsertResearch(ctx, sqlc.InsertResearchParams{ID: r.ID, ClientID: r.ClientID, Seed: r.Seed, LocationCode: r.LocationCode, LanguageCode: r.LanguageCode, Provider: r.Provider, Results: results, FetchedAt: r.FetchedAt})
}

func (s *Store) DeleteOtherResearch(ctx context.Context, clientID, seed string, locationCode int32, languageCode, keepID string) error {
	return s.q.DeleteOtherResearch(ctx, sqlc.DeleteOtherResearchParams{ClientID: clientID, Seed: seed, LocationCode: locationCode, LanguageCode: languageCode, ID: keepID})
}

func (s *Store) DeleteResearchBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.q.DeleteResearchBefore(ctx, cutoff)
}

// ─── Backlinks ───────────────────────────────────────────────

func toBacklink(b sqlc.SeosuiteBacklink) domain.Backlink {
	return domain.Backlink{ID: b.ID, ClientID: b.ClientID, PropertyID: b.PropertyID, SourceURL: b.SourceUrl, SourceDomain: b.SourceDomain, TargetURL: b.TargetUrl, Anchor: b.Anchor, Dofollow: b.Dofollow, DomainRank: b.DomainRank, SpamScore: b.SpamScore, FirstSeen: b.FirstSeen.UTC(), LastSeen: b.LastSeen.UTC(), IsLost: b.IsLost}
}

func toBacklinkSnapshot(b sqlc.SeosuiteBacklinkSnapshot) domain.BacklinkSnapshot {
	return domain.BacklinkSnapshot{ID: b.ID, ClientID: b.ClientID, PropertyID: b.PropertyID, Date: b.Date.UTC(), Backlinks: b.Backlinks, ReferringDomains: b.ReferringDomains, Dofollow: b.Dofollow, Nofollow: b.Nofollow, NewBacklinks: b.NewBacklinks, LostBacklinks: b.LostBacklinks, DomainRank: b.DomainRank, ToxicShare: b.ToxicShare}
}

func (s *Store) ListBacklinks(ctx context.Context, propertyID string) ([]domain.Backlink, error) {
	rows, err := s.q.ListBacklinks(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Backlink, 0, len(rows))
	for _, r := range rows {
		out = append(out, toBacklink(r))
	}
	return out, nil
}

func (s *Store) ListBacklinkKeys(ctx context.Context, propertyID string) ([]domain.BacklinkKey, error) {
	rows, err := s.q.ListBacklinkKeys(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.BacklinkKey, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.BacklinkKey{SourceURL: r.SourceUrl, TargetURL: r.TargetUrl, IsLost: r.IsLost})
	}
	return out, nil
}

func (s *Store) UpsertBacklinks(ctx context.Context, rows []domain.Backlink) error {
	params := make([]sqlc.UpsertBacklinkParams, 0, len(rows))
	for _, b := range rows {
		params = append(params, sqlc.UpsertBacklinkParams{ID: b.ID, ClientID: b.ClientID, PropertyID: b.PropertyID, SourceUrl: b.SourceURL, SourceDomain: b.SourceDomain, TargetUrl: b.TargetURL, Anchor: b.Anchor, Dofollow: b.Dofollow, DomainRank: b.DomainRank, SpamScore: b.SpamScore, FirstSeen: b.FirstSeen, LastSeen: b.LastSeen, IsLost: b.IsLost})
	}
	var be batchErr
	s.q.UpsertBacklink(ctx, params).Exec(be.collect)
	return be.err
}

func (s *Store) UpsertBacklinkSnapshot(ctx context.Context, b domain.BacklinkSnapshot) error {
	return s.q.UpsertBacklinkSnapshot(ctx, sqlc.UpsertBacklinkSnapshotParams{ID: b.ID, ClientID: b.ClientID, PropertyID: b.PropertyID, Date: b.Date, Backlinks: b.Backlinks, ReferringDomains: b.ReferringDomains, Dofollow: b.Dofollow, Nofollow: b.Nofollow, NewBacklinks: b.NewBacklinks, LostBacklinks: b.LostBacklinks, DomainRank: b.DomainRank, ToxicShare: b.ToxicShare})
}

func (s *Store) GetLatestBacklinkSnapshot(ctx context.Context, propertyID string) (*domain.BacklinkSnapshot, error) {
	r, err := s.q.GetLatestBacklinkSnapshot(ctx, propertyID)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := toBacklinkSnapshot(r)
	return &out, nil
}

func (s *Store) GetBacklinkSnapshotOnOrBefore(ctx context.Context, propertyID string, date time.Time) (*domain.BacklinkSnapshot, error) {
	r, err := s.q.GetBacklinkSnapshotOnOrBefore(ctx, sqlc.GetBacklinkSnapshotOnOrBeforeParams{PropertyID: propertyID, Date: date})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := toBacklinkSnapshot(r)
	return &out, nil
}

func (s *Store) ListBacklinkSnapshots(ctx context.Context, propertyID string, from, to time.Time) ([]domain.BacklinkSnapshot, error) {
	rows, err := s.q.ListBacklinkSnapshots(ctx, sqlc.ListBacklinkSnapshotsParams{PropertyID: propertyID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.BacklinkSnapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, toBacklinkSnapshot(r))
	}
	return out, nil
}

func (s *Store) DeleteBacklinkSnapshotsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.q.DeleteBacklinkSnapshotsBefore(ctx, cutoff)
}

// ─── Competitor domains ──────────────────────────────────────

func toCompetitor(c sqlc.SeosuiteCompetitorDomain) domain.CompetitorDomain {
	return domain.CompetitorDomain{ID: c.ID, ClientID: c.ClientID, PropertyID: c.PropertyID, Domain: c.Domain, CreatedAt: c.CreatedAt}
}

func toDomainSnapshot(d sqlc.SeosuiteDomainSnapshot) domain.DomainSnapshot {
	return domain.DomainSnapshot{ID: d.ID, ClientID: d.ClientID, PropertyID: d.PropertyID, Domain: d.Domain, Date: d.Date.UTC(), OrganicKeywords: d.OrganicKeywords, OrganicTraffic: d.OrganicTraffic, OrganicCost: d.OrganicCost, Top3: d.Top3, Top10: d.Top10, Top100: d.Top100, Backlinks: d.Backlinks, ReferringDomains: d.ReferringDomains, DomainRank: d.DomainRank}
}

func (s *Store) ListCompetitorDomains(ctx context.Context, propertyID string) ([]domain.CompetitorDomain, error) {
	rows, err := s.q.ListCompetitorDomains(ctx, propertyID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CompetitorDomain, 0, len(rows))
	for _, r := range rows {
		out = append(out, toCompetitor(r))
	}
	return out, nil
}

func (s *Store) GetCompetitorDomain(ctx context.Context, id string) (*domain.CompetitorDomain, error) {
	c, err := s.q.GetCompetitorDomain(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := toCompetitor(c)
	return &out, nil
}

func (s *Store) FindCompetitorDomain(ctx context.Context, propertyID, dom string) (*domain.CompetitorDomain, error) {
	c, err := s.q.FindCompetitorDomain(ctx, sqlc.FindCompetitorDomainParams{PropertyID: propertyID, Domain: dom})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := toCompetitor(c)
	return &out, nil
}

func (s *Store) CountCompetitorDomains(ctx context.Context, propertyID string) (int32, error) {
	return s.q.CountCompetitorDomains(ctx, propertyID)
}

func (s *Store) InsertCompetitorDomain(ctx context.Context, c domain.CompetitorDomain) error {
	return wrap(s.q.InsertCompetitorDomain(ctx, sqlc.InsertCompetitorDomainParams{ID: c.ID, ClientID: c.ClientID, PropertyID: c.PropertyID, Domain: c.Domain, CreatedAt: c.CreatedAt}))
}

func (s *Store) DeleteCompetitorDomain(ctx context.Context, c domain.CompetitorDomain) error {
	return s.WithTx(ctx, func(tx domain.Store) error {
		t := tx.(*Store)
		if err := t.q.DeleteCompetitorDomain(ctx, c.ID); err != nil {
			return err
		}
		return t.q.DeleteDomainSnapshotsOf(ctx, sqlc.DeleteDomainSnapshotsOfParams{PropertyID: c.PropertyID, Domain: c.Domain})
	})
}

func (s *Store) UpsertDomainSnapshot(ctx context.Context, d domain.DomainSnapshot) error {
	return s.q.UpsertDomainSnapshot(ctx, sqlc.UpsertDomainSnapshotParams{ID: d.ID, ClientID: d.ClientID, PropertyID: d.PropertyID, Domain: d.Domain, Date: d.Date, OrganicKeywords: d.OrganicKeywords, OrganicTraffic: d.OrganicTraffic, OrganicCost: d.OrganicCost, Top3: d.Top3, Top10: d.Top10, Top100: d.Top100, Backlinks: d.Backlinks, ReferringDomains: d.ReferringDomains, DomainRank: d.DomainRank})
}

func (s *Store) ListDomainSnapshots(ctx context.Context, propertyID string, domains []string, from, to time.Time) ([]domain.DomainSnapshot, error) {
	rows, err := s.q.ListDomainSnapshots(ctx, sqlc.ListDomainSnapshotsParams{PropertyID: propertyID, Domains: domains, FromDate: from, ToDate: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DomainSnapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, toDomainSnapshot(r))
	}
	return out, nil
}

func (s *Store) ListLatestDomainSnapshots(ctx context.Context, propertyID string, domains []string) ([]domain.DomainSnapshot, error) {
	rows, err := s.q.ListLatestDomainSnapshots(ctx, sqlc.ListLatestDomainSnapshotsParams{PropertyID: propertyID, Domains: domains})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DomainSnapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, toDomainSnapshot(r))
	}
	return out, nil
}

func (s *Store) DeleteDomainSnapshotsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.q.DeleteDomainSnapshotsBefore(ctx, cutoff)
}

// ─── Sync jobs ───────────────────────────────────────────────

func toJob(j sqlc.SeosuiteSyncJob) domain.SyncJob {
	return domain.SyncJob{ID: j.ID, ClientID: j.ClientID, Kind: j.Kind, Status: domain.JobStatus(j.Status), StartedAt: j.StartedAt, FinishedAt: j.FinishedAt, Message: j.Message, Error: j.Error}
}

func (s *Store) GetLatestJob(ctx context.Context, clientID, kind string) (*domain.SyncJob, error) {
	j, err := s.q.GetLatestJob(ctx, sqlc.GetLatestJobParams{ClientID: clientID, Kind: kind})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := toJob(j)
	return &out, nil
}

func (s *Store) InsertJob(ctx context.Context, j domain.SyncJob) error {
	return s.q.InsertJob(ctx, sqlc.InsertJobParams{ID: j.ID, ClientID: j.ClientID, Kind: j.Kind, Status: sqlc.SeosuiteJobStatus(j.Status), StartedAt: j.StartedAt, Message: j.Message})
}

func (s *Store) FinishJob(ctx context.Context, id string, status domain.JobStatus, finishedAt time.Time, message, errText *string) error {
	return s.q.FinishJob(ctx, sqlc.FinishJobParams{ID: id, Status: sqlc.SeosuiteJobStatus(status), FinishedAt: &finishedAt, Message: message, Error: errText})
}

func (s *Store) ListJobs(ctx context.Context, clientID string, kinds []string, limit int32) ([]domain.SyncJob, error) {
	rows, err := s.q.ListJobs(ctx, sqlc.ListJobsParams{ClientID: clientID, Kinds: kinds, RowLimit: limit})
	if err != nil {
		return nil, err
	}
	out := make([]domain.SyncJob, 0, len(rows))
	for _, r := range rows {
		out = append(out, toJob(r))
	}
	return out, nil
}

func (s *Store) DeleteJobsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.q.DeleteJobsBefore(ctx, cutoff)
}

// PurgeClient removes every row of a client inside one transaction.
func (s *Store) PurgeClient(ctx context.Context, clientID string) error {
	return s.WithTx(ctx, func(tx domain.Store) error {
		q := tx.(*Store).q
		for _, del := range []func(context.Context, string) error{
			q.PurgeRankSnapshots, q.PurgeKeywords, q.PurgeResearch, q.PurgeBacklinks,
			q.PurgeBacklinkSnapshots, q.PurgeDomainSnapshots, q.PurgeCompetitorDomains, q.PurgeJobs,
		} {
			if err := del(ctx, clientID); err != nil {
				return err
			}
		}
		return nil
	})
}
