// Package domain holds the identity entities and the ports the use cases
// need. Nothing here imports gRPC, Postgres or NATS.
package domain

import (
	"context"
	"time"
)

type UserRole string

const (
	RoleAdmin  UserRole = "ADMIN"
	RoleMember UserRole = "MEMBER"
)

type MemberRole string

const (
	MemberManager MemberRole = "MANAGER"
	MemberViewer  MemberRole = "VIEWER"
)

// Module keys a client may expose on its public dashboard.
var AllModules = []string{"SOCIAL", "SEO", "ADS"}

var Currencies = []string{"IDR", "USD", "SGD", "AUD", "EUR"}

var Timezones = []string{"Asia/Jakarta", "Asia/Makassar", "Asia/Jayapura", "Asia/Singapore", "Asia/Kuala_Lumpur", "Australia/Sydney"}

type User struct {
	ID           string
	Email        string
	Name         string
	PasswordHash string
	Role         UserRole
	IsActive     bool
	// ERP permission keys (pkg/authn/perms.go).
	Permissions []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Membership struct {
	UserID    string
	ClientID  string
	Role      MemberRole
	CreatedAt time.Time
}

// MembershipWithClient is a user's membership joined with client names.
type MembershipWithClient struct {
	UserID     string
	ClientID   string
	ClientName string
	ClientSlug string
	Role       MemberRole
}

type Client struct {
	ID           string
	Name         string
	Slug         string
	Description  string
	LogoURL      *string
	WebsiteURL   *string
	Industry     *string
	Currency     string
	Timezone     string
	ShareEnabled bool
	SharePinHash *string
	ShareModules []string
	CreatedByID  *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ClientMember is a membership row joined with the user profile.
type ClientMember struct {
	UserID    string
	Name      string
	Email     string
	IsActive  bool
	Role      MemberRole
	CreatedAt time.Time
}

type Notification struct {
	ID        string
	UserID    string
	ClientID  *string
	Type      string
	Title     string
	Body      string
	Href      *string
	ReadAt    *time.Time
	CreatedAt time.Time
}

type RefreshSession struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

// AccessRole is what a user may do with one client.
type AccessRole string

const (
	AccessAdmin   AccessRole = "ADMIN"
	AccessManager AccessRole = "MANAGER"
	AccessViewer  AccessRole = "VIEWER"
)

type ClientAccess struct {
	Client    Client
	Role      AccessRole
	CanManage bool
}

// ─── Ports ────────────────────────────────────────────────────

type UserRepo interface {
	GetUser(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	ListUsers(ctx context.Context) ([]User, error)
	ListUsersByIDs(ctx context.Context, ids []string) ([]User, error)
	ListActiveUserOptions(ctx context.Context) ([]User, error)
	ListPermissionHolders(ctx context.Context, permission string) ([]User, error)
	InsertUser(ctx context.Context, u User) (*User, error)
	// UpdateUser changes profile fields; passwordHash nil keeps the old hash.
	UpdateUser(ctx context.Context, u User, passwordHash *string) (*User, error)
	SetUserActive(ctx context.Context, id string, active bool) (*User, error)
	SetUserPermissions(ctx context.Context, id string, perms []string) (*User, error)
	// UpsertLegacyUser mirrors a Next.js user row; permissions are kept.
	UpsertLegacyUser(ctx context.Context, u User) (*User, error)
	DeleteUser(ctx context.Context, id string) (bool, error)

	ListMembershipsOfUsers(ctx context.Context, userIDs []string) ([]MembershipWithClient, error)
	ListMembershipsForUser(ctx context.Context, userID string) ([]Membership, error)
	ReplaceMemberships(ctx context.Context, userID string, memberships []Membership) error
	GetMembership(ctx context.Context, userID, clientID string) (*Membership, error)
	UpsertMembership(ctx context.Context, m Membership) error
	DeleteMembership(ctx context.Context, userID, clientID string) (bool, error)
	ListClientMembers(ctx context.Context, clientID string) ([]ClientMember, error)
	// ListAudience returns active admins plus the client's active members
	// (managers only when managersOnly is set).
	ListAudience(ctx context.Context, clientID string, managersOnly bool) ([]string, error)
}

type ClientRepo interface {
	GetClient(ctx context.Context, id string) (*Client, error)
	GetClientBySlug(ctx context.Context, slug string) (*Client, error)
	ListAllClients(ctx context.Context) ([]Client, error)
	ListClientsForUser(ctx context.Context, userID string) ([]Client, error)
	ListClientIDsForUser(ctx context.Context, userID string) ([]string, error)
	CountMembers(ctx context.Context, clientIDs []string) (map[string]int32, error)
	InsertClient(ctx context.Context, c Client) (*Client, error)
	UpdateClient(ctx context.Context, c Client) (*Client, error)
	UpdateShareSettings(ctx context.Context, id string, enabled bool, pinHash *string, modules []string) (*Client, error)
	DeleteClient(ctx context.Context, id string) (bool, error)
	CountClients(ctx context.Context) (int32, error)
}

type NotificationRepo interface {
	ListNotifications(ctx context.Context, userID string, limit int32) ([]Notification, int32, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) (int32, error)
	InsertNotifications(ctx context.Context, rows []Notification) (int32, error)
	DeleteReadBefore(ctx context.Context, cutoff time.Time) (int64, error)
	DeleteCreatedBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type SettingsRepo interface {
	GetSetting(ctx context.Context, key string, out any) (bool, error)
	PutSetting(ctx context.Context, key string, value any) error
}

type SessionRepo interface {
	InsertSession(ctx context.Context, s RefreshSession, userAgent, ip string) error
	GetSession(ctx context.Context, tokenHash string) (*RefreshSession, error)
	RevokeSession(ctx context.Context, tokenHash string) error
	RevokeUserSessions(ctx context.Context, userID string) error
	DeleteExpiredSessions(ctx context.Context, before time.Time) (int64, error)
}

// Store bundles every repository and can run a function inside one
// transaction, handing it a Store bound to that transaction.
type Store interface {
	UserRepo
	ClientRepo
	NotificationRepo
	SettingsRepo
	SessionRepo
	WithTx(ctx context.Context, fn func(tx Store) error) error
}

// ErrUnique is returned by repositories on a unique-constraint violation so
// use cases can translate it into the right field message.
type ErrUnique struct{ Constraint string }

func (e *ErrUnique) Error() string { return "unique violation: " + e.Constraint }
