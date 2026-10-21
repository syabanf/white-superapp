// Package clients wraps the generated gRPC clients of identity,
// integrations and analytics behind the app ports, with the short caches
// the conventions ask for (access 30 s, credentials 60 s, accounts 15 s).
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

func (c *ttlCache[T]) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = map[string]entry[T]{}
}
