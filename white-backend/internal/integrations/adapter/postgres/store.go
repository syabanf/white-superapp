// Package postgres implements the integrations ports on top of sqlc queries.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/integrations/domain"
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

// wrap turns unique violations into domain errors.
func wrap(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ErrUnique{Constraint: pgErr.ConstraintName}
	}
	return err
}

// ─── Connections ─────────────────────────────────────────────

func toConnection(c sqlc.IntegrationsConnection) *domain.Connection {
	return &domain.Connection{
		ID: c.ID, ClientID: c.ClientID, Provider: domain.Provider(c.Provider), AccountID: c.AccountID,
		DisplayName: c.DisplayName, AccessTokenEnc: c.AccessTokenEnc, RefreshTokenEnc: c.RefreshTokenEnc,
		ExpiresAt: c.ExpiresAt, Scopes: c.Scopes, Meta: c.Meta, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func (s *Store) GetConnection(ctx context.Context, id string) (*domain.Connection, error) {
	c, err := s.q.GetConnection(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toConnection(c), nil
}

func (s *Store) ListConnections(ctx context.Context, clientID string) ([]domain.Connection, error) {
	rows, err := s.q.ListConnections(ctx, clientID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Connection, 0, len(rows))
	for _, r := range rows {
		out = append(out, *toConnection(r))
	}
	return out, nil
}

func (s *Store) FindConnection(ctx context.Context, clientID string, provider domain.Provider) (*domain.Connection, error) {
	c, err := s.q.FindConnection(ctx, sqlc.FindConnectionParams{ClientID: clientID, Provider: sqlc.IntegrationsConnectionProvider(provider)})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toConnection(c), nil
}

func (s *Store) UpsertConnection(ctx context.Context, c domain.Connection, keepRefreshToken bool) (*domain.Connection, error) {
	row, err := s.q.UpsertConnection(ctx, sqlc.UpsertConnectionParams{
		ID: c.ID, ClientID: c.ClientID, Provider: sqlc.IntegrationsConnectionProvider(c.Provider), AccountID: c.AccountID,
		DisplayName: c.DisplayName, AccessTokenEnc: c.AccessTokenEnc, RefreshTokenEnc: c.RefreshTokenEnc,
		ExpiresAt: c.ExpiresAt, Scopes: c.Scopes, KeepRefreshToken: keepRefreshToken,
	})
	if err != nil {
		return nil, wrap(err)
	}
	return toConnection(row), nil
}

func (s *Store) UpdateAccessToken(ctx context.Context, id, accessTokenEnc string, expiresAt *time.Time) error {
	return s.q.UpdateAccessToken(ctx, sqlc.UpdateAccessTokenParams{ID: id, AccessTokenEnc: accessTokenEnc, ExpiresAt: expiresAt})
}

func (s *Store) DeleteConnection(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteConnection(ctx, id)
	return n > 0, err
}

func (s *Store) DeleteClientConnections(ctx context.Context, clientID string) (int64, error) {
	return s.q.DeleteClientConnections(ctx, clientID)
}

// ─── Settings ────────────────────────────────────────────────

func toSetting(r sqlc.IntegrationsIntegrationSetting) *domain.IntegrationSetting {
	return &domain.IntegrationSetting{
		Provider: domain.IntegrationKey(r.Provider), PublicID: r.PublicID, SecretEnc: r.SecretEnc, Meta: r.Meta,
		VerifiedAt: r.VerifiedAt, UpdatedByID: r.UpdatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func (s *Store) GetSetting(ctx context.Context, key domain.IntegrationKey) (*domain.IntegrationSetting, error) {
	r, err := s.q.GetIntegrationSetting(ctx, string(key))
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toSetting(r), nil
}

func (s *Store) ListSettings(ctx context.Context) ([]domain.IntegrationSetting, error) {
	rows, err := s.q.ListIntegrationSettings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.IntegrationSetting, 0, len(rows))
	for _, r := range rows {
		out = append(out, *toSetting(r))
	}
	return out, nil
}

func (s *Store) UpsertSetting(ctx context.Context, key domain.IntegrationKey, p domain.SettingPatch) (*domain.IntegrationSetting, error) {
	row, err := s.q.UpsertIntegrationSetting(ctx, sqlc.UpsertIntegrationSettingParams{
		Provider: string(key), PublicID: p.PublicID, SecretEnc: p.SecretEnc, Meta: p.Meta, VerifiedAt: p.VerifiedAt, UpdatedByID: p.UpdatedByID,
		SetPublicID: p.SetPublicID, SetSecret: p.SetSecret, SetMeta: p.SetMeta, SetVerifiedAt: p.SetVerifiedAt,
	})
	if err != nil {
		return nil, wrap(err)
	}
	return toSetting(row), nil
}

func (s *Store) DeleteSetting(ctx context.Context, key domain.IntegrationKey) (bool, error) {
	n, err := s.q.DeleteIntegrationSetting(ctx, string(key))
	return n > 0, err
}
