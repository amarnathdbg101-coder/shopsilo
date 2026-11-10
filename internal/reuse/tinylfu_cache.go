// Package reuse provides reusable foundational wrappers and shared infrastructure utilities.
package reuse

import (
	"time"

	"github.com/amarnathdbg101-coder/go/go-tinylfu"
)

// ExpiringValue wraps cached values with an absolute expiration deadline.
type ExpiringValue[T any] struct {
	Value     T
	ExpiresAt time.Time
}

// IsExpired checks if the cached item has expired.
func (e ExpiringValue[T]) IsExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt)
}

// TinyLFUCache provides a thread-safe Window TinyLFU cache with frequency admission and scan resistance (powered by go-tinylfu).
type TinyLFUCache[K comparable, V any] struct {
	inner *tinylfu.Cache[K, ExpiringValue[V]]
}

// NewTinyLFUCache constructs a new Window TinyLFU cache with specified capacity.
func NewTinyLFUCache[K comparable, V any](capacity int) (*TinyLFUCache[K, V], error) {
	cache, err := tinylfu.New[K, ExpiringValue[V]](capacity)
	if err != nil {
		return nil, err
	}
	return &TinyLFUCache[K, V]{inner: cache}, nil
}

// Get retrieves an item from cache if present and unexpired.
func (c *TinyLFUCache[K, V]) Get(key K) (V, bool) {
	if c == nil || c.inner == nil {
		var zero V
		return zero, false
	}

	entry, found := c.inner.Get(key)
	if !found || entry.IsExpired() {
		var zero V
		if found && entry.IsExpired() {
			c.inner.Delete(key)
		}
		return zero, false
	}
	return entry.Value, true
}

// Set inserts or updates an item with TTL.
func (c *TinyLFUCache[K, V]) Set(key K, value V, ttl time.Duration) {
	if c == nil || c.inner == nil {
		return
	}

	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	c.inner.Set(key, ExpiringValue[V]{
		Value:     value,
		ExpiresAt: expiresAt,
	})
}

// Delete removes an item from cache.
func (c *TinyLFUCache[K, V]) Delete(key K) {
	if c == nil || c.inner == nil {
		return
	}
	c.inner.Delete(key)
}

// GlobalAICache is the centralized Window TinyLFU cache for Gemini responses, OCR lookups, and Parchi parse results.
var GlobalAICache, _ = NewTinyLFUCache[string, string](2000)
