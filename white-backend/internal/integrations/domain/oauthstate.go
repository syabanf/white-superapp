package domain

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"
)

// OAuthState is the signed `state` parameter of an OAuth round-trip:
// base64url(JSON{slug,nonce,ts}) + "." + base64url(HMAC-SHA256(body, secret)).
// The nonce is mirrored in an httpOnly cookie and compared on callback.
type OAuthState struct {
	Slug  string `json:"slug"`
	Nonce string `json:"nonce"`
	// Ts is Unix milliseconds, the way the Next.js app wrote it.
	Ts int64 `json:"ts"`
}

const (
	OAuthStateMaxAge = 10 * time.Minute
	// oauthClockSkew is how far in the future a timestamp may sit.
	oauthClockSkew = 60 * time.Second
)

var SlugRe = regexp.MustCompile(`^[a-z0-9-]+$`)

var ErrNoStateSecret = errors.New("OAuth state secret is missing (AUTH_SECRET)")

func RandomNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func stateMAC(body, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(body))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func SignOAuthState(slug, nonce, secret string, now time.Time) (string, error) {
	if secret == "" {
		return "", ErrNoStateSecret
	}
	raw, err := json.Marshal(OAuthState{Slug: slug, Nonce: nonce, Ts: now.UnixMilli()})
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	return body + "." + stateMAC(body, secret), nil
}

// VerifyOAuthState checks signature, shape and age. It returns nil on any
// failure: tampering, malformed input, expiry, or a future timestamp beyond
// the clock skew.
func VerifyOAuthState(state, secret string, now time.Time) *OAuthState {
	if state == "" || secret == "" {
		return nil
	}
	idx := strings.LastIndex(state, ".")
	if idx <= 0 || idx == len(state)-1 {
		return nil
	}
	body, sig := state[:idx], state[idx+1:]
	if !hmac.Equal([]byte(sig), []byte(stateMAC(body, secret))) {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil
	}
	var parsed struct {
		Slug  any `json:"slug"`
		Nonce any `json:"nonce"`
		Ts    any `json:"ts"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil
	}
	slug, ok := parsed.Slug.(string)
	if !ok || !SlugRe.MatchString(slug) {
		return nil
	}
	nonce, ok := parsed.Nonce.(string)
	if !ok || nonce == "" {
		return nil
	}
	ts, ok := parsed.Ts.(float64)
	if !ok || math.IsNaN(ts) || math.IsInf(ts, 0) {
		return nil
	}
	issued := time.UnixMilli(int64(ts))
	if issued.After(now.Add(oauthClockSkew)) {
		return nil
	}
	if now.Sub(issued) > OAuthStateMaxAge {
		return nil
	}
	return &OAuthState{Slug: slug, Nonce: nonce, Ts: int64(ts)}
}
