// Package domain holds the integrations entities, the repository ports and
// the pure rules (credential registry, Meta review checklist, cost
// estimators, OAuth state). Nothing here imports gRPC, Postgres or NATS.
package domain

import (
	"context"
	"time"
)

type Provider string

const (
	ProviderMeta   Provider = "META"
	ProviderGoogle Provider = "GOOGLE"
)

func (p Provider) Valid() bool { return p == ProviderMeta || p == ProviderGoogle }

// Slug is the lowercase name used in URLs and the `?provider=` query.
func (p Provider) Slug() string {
	if p == ProviderMeta {
		return "meta"
	}
	return "google"
}

// IntegrationKey is the credential pair the provider app needs.
func (p Provider) IntegrationKey() IntegrationKey {
	if p == ProviderMeta {
		return KeyMeta
	}
	return KeyGoogle
}

type Connection struct {
	ID              string
	ClientID        string
	Provider        Provider
	AccountID       string
	DisplayName     string
	AccessTokenEnc  string
	RefreshTokenEnc *string
	ExpiresAt       *time.Time
	Scopes          []string
	Meta            []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// RefreshSkew is how close to expiry a token counts as expiring.
const RefreshSkew = 60 * time.Second

// Expiring reports whether the token must be refreshed before use. A nil
// expiry never expires.
func (c Connection) Expiring(now time.Time) bool {
	return c.ExpiresAt != nil && c.ExpiresAt.Sub(now) < RefreshSkew
}

type IntegrationSetting struct {
	Provider    IntegrationKey
	PublicID    *string
	SecretEnc   *string
	Meta        []byte
	VerifiedAt  *time.Time
	UpdatedByID *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SettingPatch is a partial update: a field is written only when its Set
// flag is on, so callers can clear (nil) or keep (flag off) independently.
type SettingPatch struct {
	PublicID      *string
	SetPublicID   bool
	SecretEnc     *string
	SetSecret     bool
	Meta          []byte
	SetMeta       bool
	VerifiedAt    *time.Time
	SetVerifiedAt bool
	UpdatedByID   *string
}

// ClientAccess is what identity resolved for one user on one client.
// CanManage is true for ADMIN users and MANAGER members.
type ClientAccess struct {
	ClientID  string
	Slug      string
	CanManage bool
}

// ─── Ports ────────────────────────────────────────────────────

type ConnectionRepo interface {
	GetConnection(ctx context.Context, id string) (*Connection, error)
	ListConnections(ctx context.Context, clientID string) ([]Connection, error)
	// FindConnection returns the oldest connection of a provider for a client.
	FindConnection(ctx context.Context, clientID string, provider Provider) (*Connection, error)
	// UpsertConnection inserts or updates on (client_id, provider, account_id).
	// With keepRefreshToken, a nil refresh token leaves the stored one alone.
	UpsertConnection(ctx context.Context, c Connection, keepRefreshToken bool) (*Connection, error)
	UpdateAccessToken(ctx context.Context, id, accessTokenEnc string, expiresAt *time.Time) error
	DeleteConnection(ctx context.Context, id string) (bool, error)
	DeleteClientConnections(ctx context.Context, clientID string) (int64, error)
}

type SettingsRepo interface {
	GetSetting(ctx context.Context, key IntegrationKey) (*IntegrationSetting, error)
	ListSettings(ctx context.Context) ([]IntegrationSetting, error)
	UpsertSetting(ctx context.Context, key IntegrationKey, patch SettingPatch) (*IntegrationSetting, error)
	DeleteSetting(ctx context.Context, key IntegrationKey) (bool, error)
}

type Store interface {
	ConnectionRepo
	SettingsRepo
	WithTx(ctx context.Context, fn func(tx Store) error) error
}

// ErrUnique is returned by repositories on a unique-constraint violation.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }
