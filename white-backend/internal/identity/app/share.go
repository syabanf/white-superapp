package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/identity/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
	"github.com/syabanf/white-superapp/white-backend/pkg/password"
)

const (
	ShareCookieMaxAge = 12 * time.Hour
	pinWindow         = 10 * time.Minute
	pinMaxAttempts    = 5

	msgPinWrong    = "PIN salah"
	msgPinTooMany  = "Terlalu banyak percobaan. Coba lagi beberapa menit."
	msgShareHidden = "Dashboard tidak ditemukan"
)

type ShareStatus string

const (
	ShareOK       ShareStatus = "OK"
	ShareLocked   ShareStatus = "LOCKED"
	ShareDisabled ShareStatus = "DISABLED"
	ShareNotFound ShareStatus = "NOT_FOUND"
)

type ShareContext struct {
	Status ShareStatus
	Client *domain.Client
}

// Share resolves public dashboards and verifies PINs. The unlock token is
// HMAC(secret, pinHash): changing the PIN invalidates every cookie.
type Share struct {
	store   domain.Store
	secret  []byte
	clock   clock.Clock
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	count   int
	resetAt time.Time
}

func NewShare(store domain.Store, secret string, clk clock.Clock) *Share {
	if secret == "" {
		secret = "white-share-dev-secret"
	}
	return &Share{store: store, secret: []byte(secret), clock: clk, buckets: map[string]*bucket{}}
}

func (s *Share) UnlockToken(pinHash string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(pinHash))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Share) Context(ctx context.Context, slug, unlockToken string) (*ShareContext, error) {
	c, err := s.store.GetClientBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	switch {
	case c == nil:
		return &ShareContext{Status: ShareNotFound}, nil
	case !c.ShareEnabled:
		return &ShareContext{Status: ShareDisabled, Client: c}, nil
	case c.SharePinHash != nil && !hmac.Equal([]byte(unlockToken), []byte(s.UnlockToken(*c.SharePinHash))):
		return &ShareContext{Status: ShareLocked, Client: c}, nil
	}
	return &ShareContext{Status: ShareOK, Client: c}, nil
}

// allow counts every attempt per slug+ip inside a 10 minute window; the
// sixth attempt is blocked. In-memory and per instance, like before.
func (s *Share) allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock.Now()
	b := s.buckets[key]
	if b == nil || !now.Before(b.resetAt) {
		s.buckets[key] = &bucket{count: 1, resetAt: now.Add(pinWindow)}
		return true
	}
	b.count++
	return b.count <= pinMaxAttempts
}

// VerifyPin returns the unlock token for the cookie. A client without a PIN
// returns an empty token (nothing to unlock).
func (s *Share) VerifyPin(ctx context.Context, slug, pin, ip string) (clientID, token string, err error) {
	if !s.allow(slug + ":" + ip) {
		return "", "", apperr.New(apperr.RateLimit, msgPinTooMany)
	}
	if !pinRe.MatchString(pin) {
		return "", "", apperr.New(apperr.Validation, msgPinWrong)
	}
	c, err := s.store.GetClientBySlug(ctx, slug)
	if err != nil {
		return "", "", err
	}
	if c == nil || !c.ShareEnabled {
		return "", "", apperr.New(apperr.NotFound, msgShareHidden)
	}
	if c.SharePinHash == nil {
		return c.ID, "", nil
	}
	if !password.Verify(*c.SharePinHash, pin) {
		return "", "", apperr.New(apperr.Validation, msgPinWrong)
	}
	return c.ID, s.UnlockToken(*c.SharePinHash), nil
}
