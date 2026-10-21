// Package app holds the identity use cases. Each service struct takes its
// ports through the constructor and returns *apperr.Error for user-facing
// failures.
package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/authn"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
	"github.com/syabanf/white-superapp/white-backend/pkg/password"
)

const (
	msgInvalidLogin = "Email atau kata sandi salah"
	RefreshTTL      = 14 * 24 * time.Hour // same as the old Auth.js session maxAge
)

type Tokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	User             domain.User
}

type Auth struct {
	store  domain.Store
	issuer *authn.Issuer_
	clock  clock.Clock
}

func NewAuth(store domain.Store, issuer *authn.Issuer_, clk clock.Clock) *Auth {
	return &Auth{store: store, issuer: issuer, clock: clk}
}

// Login mirrors the credentials provider: lowercase+trim the email, reject
// unknown or inactive users with the same message, then issue tokens.
func (a *Auth) Login(ctx context.Context, email, pw, userAgent, ip string) (*Tokens, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || pw == "" {
		return nil, apperr.New(apperr.Validation, msgInvalidLogin)
	}
	u, err := a.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive || !password.Verify(u.PasswordHash, pw) {
		return nil, apperr.New(apperr.Unauthenticated, msgInvalidLogin)
	}
	return a.issue(ctx, *u, userAgent, ip)
}

func (a *Auth) issue(ctx context.Context, u domain.User, userAgent, ip string) (*Tokens, error) {
	now := a.clock.Now()
	access, exp, err := a.issuer.Issue(authn.Principal{UserID: u.ID, Role: string(u.Role), Name: u.Name, Email: u.Email}, now)
	if err != nil {
		return nil, err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	refresh := base64.RawURLEncoding.EncodeToString(raw)
	sess := domain.RefreshSession{ID: ids.New(), UserID: u.ID, TokenHash: hashToken(refresh), ExpiresAt: now.Add(RefreshTTL)}
	if err := a.store.InsertSession(ctx, sess, userAgent, ip); err != nil {
		return nil, err
	}
	return &Tokens{AccessToken: access, AccessExpiresAt: exp, RefreshToken: refresh, RefreshExpiresAt: sess.ExpiresAt, User: u}, nil
}

// Refresh rotates the refresh token: the old one is revoked, a new pair is
// issued. Deactivated users are cut off here even with a valid token.
func (a *Auth) Refresh(ctx context.Context, refreshToken, userAgent, ip string) (*Tokens, error) {
	sess, err := a.store.GetSession(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, err
	}
	now := a.clock.Now()
	if sess == nil || sess.RevokedAt != nil || !sess.ExpiresAt.After(now) {
		return nil, apperr.New(apperr.Unauthenticated, "Sesi berakhir. Silakan masuk lagi.")
	}
	u, err := a.store.GetUser(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive {
		return nil, apperr.New(apperr.Unauthenticated, "Akun Anda dinonaktifkan. Hubungi admin.")
	}
	if err := a.store.RevokeSession(ctx, sess.TokenHash); err != nil {
		return nil, err
	}
	return a.issue(ctx, *u, userAgent, ip)
}

func (a *Auth) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return a.store.RevokeSession(ctx, hashToken(refreshToken))
}

// Me re-reads the user row so a deactivation takes effect on the next call.
func (a *Auth) Me(ctx context.Context, userID string) (*domain.User, error) {
	u, err := a.store.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive {
		return nil, apperr.New(apperr.Unauthenticated, "")
	}
	return u, nil
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}
