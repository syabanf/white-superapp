// Package postgres implements the identity ports on top of sqlc queries.
package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/adapter/postgres/sqlc"
	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	pg "github.com/syabanf/white-superapp/white-backend/pkg/postgres"
	"time"
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

// wrap turns pgx errors into domain errors; sql.ErrNoRows becomes nil result.
func wrap(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return &domain.ErrUnique{Constraint: pgErr.ConstraintName}
	}
	return err
}

// ─── Users ───────────────────────────────────────────────────

func toUser(u sqlc.IdentityUser) *domain.User {
	return &domain.User{ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: u.PasswordHash, Role: domain.UserRole(u.Role), IsActive: u.IsActive, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}
}

func (s *Store) GetUser(ctx context.Context, id string) (*domain.User, error) {
	u, err := s.q.GetUser(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toUser(u), nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := s.q.GetUserByEmail(ctx, email)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toUser(u), nil
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		out = append(out, *toUser(r))
	}
	return out, nil
}

func (s *Store) ListUsersByIDs(ctx context.Context, ids []string) ([]domain.User, error) {
	rows, err := s.q.ListUsersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		out = append(out, *toUser(r))
	}
	return out, nil
}

func (s *Store) ListActiveUserOptions(ctx context.Context) ([]domain.User, error) {
	rows, err := s.q.ListActiveUserOptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.User{ID: r.ID, Name: r.Name, Email: r.Email, IsActive: true})
	}
	return out, nil
}

func (s *Store) InsertUser(ctx context.Context, u domain.User) (*domain.User, error) {
	row, err := s.q.InsertUser(ctx, sqlc.InsertUserParams{ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: u.PasswordHash, Role: sqlc.IdentityUserRole(u.Role), IsActive: u.IsActive})
	if err != nil {
		return nil, wrap(err)
	}
	return toUser(row), nil
}

func (s *Store) UpdateUser(ctx context.Context, u domain.User, passwordHash *string) (*domain.User, error) {
	row, err := s.q.UpdateUser(ctx, sqlc.UpdateUserParams{ID: u.ID, Name: u.Name, Email: u.Email, Role: sqlc.IdentityUserRole(u.Role), IsActive: u.IsActive, PasswordHash: passwordHash})
	if err != nil {
		return nil, wrap(err)
	}
	return toUser(row), nil
}

func (s *Store) SetUserActive(ctx context.Context, id string, active bool) (*domain.User, error) {
	row, err := s.q.SetUserActive(ctx, sqlc.SetUserActiveParams{ID: id, IsActive: active})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toUser(row), nil
}

func (s *Store) DeleteUser(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteUser(ctx, id)
	return n > 0, err
}

func (s *Store) ListMembershipsOfUsers(ctx context.Context, userIDs []string) ([]domain.MembershipWithClient, error) {
	rows, err := s.q.ListMembershipsOfUsers(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	out := make([]domain.MembershipWithClient, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.MembershipWithClient{UserID: r.UserID, ClientID: r.ClientID, ClientName: r.ClientName, ClientSlug: r.ClientSlug, Role: domain.MemberRole(r.Role)})
	}
	return out, nil
}

func (s *Store) ListMembershipsForUser(ctx context.Context, userID string) ([]domain.Membership, error) {
	rows, err := s.q.ListMembershipsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Membership, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Membership{UserID: userID, ClientID: r.ClientID, Role: domain.MemberRole(r.Role)})
	}
	return out, nil
}

func (s *Store) ReplaceMemberships(ctx context.Context, userID string, memberships []domain.Membership) error {
	if err := s.q.DeleteMembershipsOfUser(ctx, userID); err != nil {
		return err
	}
	for _, m := range memberships {
		if err := s.q.InsertMembership(ctx, sqlc.InsertMembershipParams{UserID: userID, ClientID: m.ClientID, Role: sqlc.IdentityClientMemberRole(m.Role)}); err != nil {
			return wrap(err)
		}
	}
	return nil
}

func (s *Store) GetMembership(ctx context.Context, userID, clientID string) (*domain.Membership, error) {
	m, err := s.q.GetMembership(ctx, sqlc.GetMembershipParams{UserID: userID, ClientID: clientID})
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.Membership{UserID: m.UserID, ClientID: m.ClientID, Role: domain.MemberRole(m.Role), CreatedAt: m.CreatedAt}, nil
}

func (s *Store) UpsertMembership(ctx context.Context, m domain.Membership) error {
	return wrap(s.q.UpsertMembership(ctx, sqlc.UpsertMembershipParams{UserID: m.UserID, ClientID: m.ClientID, Role: sqlc.IdentityClientMemberRole(m.Role)}))
}

func (s *Store) DeleteMembership(ctx context.Context, userID, clientID string) (bool, error) {
	n, err := s.q.DeleteMembership(ctx, sqlc.DeleteMembershipParams{UserID: userID, ClientID: clientID})
	return n > 0, err
}

func (s *Store) ListClientMembers(ctx context.Context, clientID string) ([]domain.ClientMember, error) {
	rows, err := s.q.ListClientMembers(ctx, clientID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ClientMember, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.ClientMember{UserID: r.UserID, Name: r.Name, Email: r.Email, IsActive: r.IsActive, Role: domain.MemberRole(r.Role), CreatedAt: r.CreatedAt})
	}
	return out, nil
}

func (s *Store) ListAudience(ctx context.Context, clientID string, managersOnly bool) ([]string, error) {
	return s.q.ListAudienceUserIDs(ctx, sqlc.ListAudienceUserIDsParams{ClientID: clientID, ManagersOnly: managersOnly})
}

// ─── Clients ─────────────────────────────────────────────────

func toClient(c sqlc.IdentityClient) *domain.Client {
	return &domain.Client{ID: c.ID, Name: c.Name, Slug: c.Slug, Description: c.Description, LogoURL: c.LogoUrl, WebsiteURL: c.WebsiteUrl, Industry: c.Industry, Currency: c.Currency, Timezone: c.Timezone, ShareEnabled: c.ShareEnabled, SharePinHash: c.SharePinHash, ShareModules: c.ShareModules, CreatedByID: c.CreatedByID, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

func toClients(rows []sqlc.IdentityClient) []domain.Client {
	out := make([]domain.Client, 0, len(rows))
	for _, r := range rows {
		out = append(out, *toClient(r))
	}
	return out
}

func (s *Store) GetClient(ctx context.Context, id string) (*domain.Client, error) {
	c, err := s.q.GetClient(ctx, id)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toClient(c), nil
}

func (s *Store) GetClientBySlug(ctx context.Context, slug string) (*domain.Client, error) {
	c, err := s.q.GetClientBySlug(ctx, slug)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toClient(c), nil
}

func (s *Store) ListAllClients(ctx context.Context) ([]domain.Client, error) {
	rows, err := s.q.ListAllClients(ctx)
	if err != nil {
		return nil, err
	}
	return toClients(rows), nil
}

func (s *Store) ListClientsForUser(ctx context.Context, userID string) ([]domain.Client, error) {
	rows, err := s.q.ListClientsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toClients(rows), nil
}

func (s *Store) ListClientIDsForUser(ctx context.Context, userID string) ([]string, error) {
	return s.q.ListClientIDsForUser(ctx, userID)
}

func (s *Store) CountMembers(ctx context.Context, clientIDs []string) (map[string]int32, error) {
	rows, err := s.q.CountMembersByClient(ctx, clientIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int32, len(rows))
	for _, r := range rows {
		out[r.ClientID] = r.Members
	}
	return out, nil
}

func (s *Store) InsertClient(ctx context.Context, c domain.Client) (*domain.Client, error) {
	row, err := s.q.InsertClient(ctx, sqlc.InsertClientParams{ID: c.ID, Name: c.Name, Slug: c.Slug, Description: c.Description, Industry: c.Industry, WebsiteUrl: c.WebsiteURL, Currency: c.Currency, Timezone: c.Timezone, ShareModules: c.ShareModules, CreatedByID: c.CreatedByID})
	if err != nil {
		return nil, wrap(err)
	}
	return toClient(row), nil
}

func (s *Store) UpdateClient(ctx context.Context, c domain.Client) (*domain.Client, error) {
	row, err := s.q.UpdateClient(ctx, sqlc.UpdateClientParams{ID: c.ID, Name: c.Name, Slug: c.Slug, Description: c.Description, Industry: c.Industry, WebsiteUrl: c.WebsiteURL, Currency: c.Currency, Timezone: c.Timezone})
	if err != nil {
		return nil, wrap(err)
	}
	return toClient(row), nil
}

func (s *Store) UpdateShareSettings(ctx context.Context, id string, enabled bool, pinHash *string, modules []string) (*domain.Client, error) {
	row, err := s.q.UpdateShareSettings(ctx, sqlc.UpdateShareSettingsParams{ID: id, ShareEnabled: enabled, SharePinHash: pinHash, ShareModules: modules})
	if err != nil {
		return nil, err
	}
	return toClient(row), nil
}

func (s *Store) DeleteClient(ctx context.Context, id string) (bool, error) {
	n, err := s.q.DeleteClient(ctx, id)
	return n > 0, err
}

func (s *Store) CountClients(ctx context.Context) (int32, error) {
	return s.q.CountClients(ctx)
}

// ─── Notifications ───────────────────────────────────────────

func (s *Store) ListNotifications(ctx context.Context, userID string, limit int32) ([]domain.Notification, int32, error) {
	rows, err := s.q.ListNotifications(ctx, sqlc.ListNotificationsParams{UserID: userID, Limit: limit})
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.q.CountUnreadNotifications(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Notification, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.Notification{ID: r.ID, UserID: r.UserID, ClientID: r.ClientID, Type: r.Type, Title: r.Title, Body: r.Body, Href: r.Href, ReadAt: r.ReadAt, CreatedAt: r.CreatedAt})
	}
	return out, unread, nil
}

func (s *Store) MarkRead(ctx context.Context, id, userID string) error {
	_, err := s.q.MarkNotificationRead(ctx, sqlc.MarkNotificationReadParams{ID: id, UserID: userID})
	return err
}

func (s *Store) MarkAllRead(ctx context.Context, userID string) (int32, error) {
	n, err := s.q.MarkAllNotificationsRead(ctx, userID)
	return int32(n), err
}

func (s *Store) InsertNotifications(ctx context.Context, rows []domain.Notification) (int32, error) {
	params := make([]sqlc.InsertNotificationParams, 0, len(rows))
	for _, r := range rows {
		params = append(params, sqlc.InsertNotificationParams{ID: r.ID, UserID: r.UserID, ClientID: r.ClientID, Type: r.Type, Title: r.Title, Body: r.Body, Href: r.Href})
	}
	n, err := s.q.InsertNotification(ctx, params)
	return int32(n), err
}

func (s *Store) DeleteReadBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.q.DeleteReadNotificationsBefore(ctx, &cutoff)
}

func (s *Store) DeleteCreatedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.q.DeleteNotificationsBefore(ctx, cutoff)
}

// ─── Settings ────────────────────────────────────────────────

func (s *Store) GetSetting(ctx context.Context, key string, out any) (bool, error) {
	raw, err := s.q.GetAppSetting(ctx, key)
	if pg.IsNoRows(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(raw, out)
}

func (s *Store) PutSetting(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.q.UpsertAppSetting(ctx, sqlc.UpsertAppSettingParams{Key: key, Value: raw})
}

// ─── Sessions ────────────────────────────────────────────────

func (s *Store) InsertSession(ctx context.Context, r domain.RefreshSession, userAgent, ip string) error {
	return s.q.InsertRefreshSession(ctx, sqlc.InsertRefreshSessionParams{ID: r.ID, UserID: r.UserID, TokenHash: r.TokenHash, ExpiresAt: r.ExpiresAt, UserAgent: userAgent, Ip: ip})
}

func (s *Store) GetSession(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	r, err := s.q.GetRefreshSession(ctx, tokenHash)
	if pg.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.RefreshSession{ID: r.ID, UserID: r.UserID, TokenHash: r.TokenHash, ExpiresAt: r.ExpiresAt, CreatedAt: r.CreatedAt, RevokedAt: r.RevokedAt}, nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := s.q.RevokeRefreshSession(ctx, tokenHash)
	return err
}

func (s *Store) RevokeUserSessions(ctx context.Context, userID string) error {
	_, err := s.q.RevokeUserSessions(ctx, userID)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context, before time.Time) (int64, error) {
	return s.q.DeleteExpiredRefreshSessions(ctx, before)
}
