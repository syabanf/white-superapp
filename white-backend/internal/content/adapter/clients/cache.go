// Package clients wraps the generated gRPC clients of identity,
// integrations, analytics and publishing behind the app ports, with short
// caches (access 30 s, credentials 60 s, service tokens until expiry).
package clients

import (
	"sync"
	"time"
)

type entry[T any] struct {
	value   T
	expires time.Time
}

// ttlCache is a small map cache keyed by string.
type ttlCache[T any] struct {
	mu   sync.Mutex
	ttl  time.Duration
	data map[string]entry[T]
}

func newCache[T any](ttl time.Duration) *ttlCache[T] {
	return &ttlCache[T]{ttl: ttl, data: map[string]entry[T]{}}
}

func (c *ttlCache[T]) get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.data[key]
	if !ok || time.Now().After(e.expires) {
		var zero T
		return zero, false
	}
	return e.value, true
}

func (c *ttlCache[T]) put(key string, v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = entry[T]{value: v, expires: time.Now().Add(c.ttl)}
}
