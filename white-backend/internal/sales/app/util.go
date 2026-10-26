package app

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/syabanf/white-superapp/white-backend/internal/sales/domain"
	"github.com/syabanf/white-superapp/white-backend/pkg/apperr"
	"github.com/syabanf/white-superapp/white-backend/pkg/clock"
)

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// NewToken returns an unguessable URL-safe token (256 bits).
func NewToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// RateLimiter counts requests per key in a fixed window, in memory and per
// instance like the share PIN limiter in identity.
type RateLimiter struct {
	clock   clock.Clock
	window  time.Duration
	max     int
	mu      sync.Mutex
	buckets map[string]*bucket
}

type bucket struct {
	count   int
	resetAt time.Time
}

func NewRateLimiter(clk clock.Clock, window time.Duration, max int) *RateLimiter {
	return &RateLimiter{clock: clk, window: window, max: max, buckets: map[string]*bucket{}}
}

// Allow counts one request and reports whether it is within the limit.
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.clock.Now()
	if len(r.buckets) > 10000 {
		for k, b := range r.buckets {
			if !now.Before(b.resetAt) {
				delete(r.buckets, k)
			}
		}
	}
	b := r.buckets[key]
	if b == nil || !now.Before(b.resetAt) {
		r.buckets[key] = &bucket{count: 1, resetAt: now.Add(r.window)}
		return true
	}
	b.count++
	return b.count <= r.max
}

// Check returns a RATE_LIMIT error once the key is over the limit.
func (r *RateLimiter) Check(key string) error {
	if r == nil || r.Allow(key) {
		return nil
	}
	return apperr.New(apperr.RateLimit, msgTooManyRequests)
}

// asUnique reports a unique constraint violation from the store.
func asUnique(err error, target **domain.ErrUnique) bool { return errors.As(err, target) }

// decodeJSONAnswer reads the JSON object inside a model answer (code
// fences and surrounding prose are ignored).
func decodeJSONAnswer(content string, v any) error {
	raw := domain.StripCodeFence(content)
	if i, j := strings.Index(raw, "{"), strings.LastIndex(raw, "}"); i >= 0 && j > i {
		raw = raw[i : j+1]
	}
	return json.Unmarshal([]byte(raw), v)
}
