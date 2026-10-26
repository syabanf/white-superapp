// Package authn issues and verifies the EdDSA access tokens that identify a
// user across services. Identity signs; every service verifies with the
// public key and reads the principal from the request context.
package authn

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/ids"
)

const (
	Issuer         = "white-identity"
	RoleAdmin      = "ADMIN"
	RoleMember     = "MEMBER"
	MetadataHeader = "authorization"
)

type Principal struct {
	UserID string
	Role   string
	Name   string
	Email  string
	// ERP permission keys (see perms.go). Empty for marketing-only users.
	Perms []string
}

func (p Principal) IsAdmin() bool { return p.Role == RoleAdmin }

type claims struct {
	Role  string   `json:"role"`
	Name  string   `json:"name,omitempty"`
	Email string   `json:"email,omitempty"`
	Perms []string `json:"perms,omitempty"`
	jwt.RegisteredClaims
}

// KeyPair derives from JWT_PRIVATE_KEY (base64 of a 32-byte seed or 64-byte
// key). With no key set, the seed is SHA-256(AUTH_SECRET) so every local
// service agrees without extra setup; production must set an explicit key.
func KeyPair(privateKeyEnv, authSecret string) (ed25519.PrivateKey, error) {
	if privateKeyEnv != "" {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(privateKeyEnv))
		if err != nil {
			return nil, fmt.Errorf("JWT_PRIVATE_KEY bukan base64: %w", err)
		}
		switch len(raw) {
		case ed25519.SeedSize:
			return ed25519.NewKeyFromSeed(raw), nil
		case ed25519.PrivateKeySize:
			return ed25519.PrivateKey(raw), nil
		}
		return nil, fmt.Errorf("JWT_PRIVATE_KEY harus 32 atau 64 byte, dapat %d", len(raw))
	}
	if authSecret == "" {
		return nil, errors.New("JWT_PRIVATE_KEY atau AUTH_SECRET wajib diisi")
	}
	seed := sha256.Sum256([]byte("white-jwt:" + authSecret))
	return ed25519.NewKeyFromSeed(seed[:]), nil
}

// PublicKey reads JWT_PUBLIC_KEY (base64, 32 bytes) or derives it from the
// same fallback as KeyPair.
func PublicKey(publicKeyEnv, privateKeyEnv, authSecret string) (ed25519.PublicKey, error) {
	if publicKeyEnv != "" {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKeyEnv))
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return nil, errors.New("JWT_PUBLIC_KEY harus base64 dari 32 byte")
		}
		return ed25519.PublicKey(raw), nil
	}
	priv, err := KeyPair(privateKeyEnv, authSecret)
	if err != nil {
		return nil, err
	}
	return priv.Public().(ed25519.PublicKey), nil
}

type Issuer_ struct {
	key ed25519.PrivateKey
	ttl time.Duration
}

func NewIssuer(key ed25519.PrivateKey, ttl time.Duration) *Issuer_ {
	return &Issuer_{key: key, ttl: ttl}
}

// Issue returns a signed access token and its expiry.
func (i *Issuer_) Issue(p Principal, now time.Time) (string, time.Time, error) {
	exp := now.Add(i.ttl)
	c := claims{
		Role:  p.Role,
		Name:  p.Name,
		Email: p.Email,
		Perms: p.Perms,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   p.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        ids.New(),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, c).SignedString(i.key)
	return tok, exp, err
}

type Verifier struct {
	pub ed25519.PublicKey
}

func NewVerifier(pub ed25519.PublicKey) *Verifier { return &Verifier{pub: pub} }

func (v *Verifier) Verify(token string) (Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, errors.New("metode tanda tangan tidak didukung")
		}
		return v.pub, nil
	}, jwt.WithIssuer(Issuer), jwt.WithExpirationRequired(), jwt.WithLeeway(30*time.Second))
	if err != nil {
		return Principal{}, apperr.Wrap(apperr.Unauthenticated, "", err)
	}
	if c.Subject == "" {
		return Principal{}, apperr.New(apperr.Unauthenticated, "")
	}
	role := c.Role
	if role == "" {
		role = RoleMember
	}
	return Principal{UserID: c.Subject, Role: role, Name: c.Name, Email: c.Email, Perms: c.Perms}, nil
}

type ctxKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}

// Require returns the caller or an UNAUTHENTICATED error.
func Require(ctx context.Context) (Principal, error) {
	p, ok := FromContext(ctx)
	if !ok {
		return Principal{}, apperr.New(apperr.Unauthenticated, "")
	}
	return p, nil
}

// RequireAdmin returns UNAUTHORIZED for signed-in non-admins.
func RequireAdmin(ctx context.Context) (Principal, error) {
	p, err := Require(ctx)
	if err != nil {
		return p, err
	}
	if !p.IsAdmin() {
		return p, apperr.New(apperr.Unauthorized, "")
	}
	return p, nil
}

// BearerToken strips the scheme from an Authorization value.
func BearerToken(header string) (string, bool) {
	const prefix = "bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):]), true
	}
	return "", false
}
