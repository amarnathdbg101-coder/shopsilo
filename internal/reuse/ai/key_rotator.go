// Package ai provides thread-safe API key rotation and health tracking for Gemini AI.
package ai

import (
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoKeysConfigured = errors.New("no api keys configured for gemini")
	ErrAllKeysExhausted = errors.New("all api keys are currently rate limited or exhausted")
)

type KeyHealth struct {
	Key               string    `json:"key"`
	ConsecutiveErrors int       `json:"consecutive_errors"`
	IsRateLimited     bool      `json:"is_rate_limited"`
	RateLimitedUntil  time.Time `json:"rate_limited_until"`
	TotalCalls        int64     `json:"total_calls"`
	TotalSuccess      int64     `json:"total_success"`
}

// KeyRotator manages thread-safe round-robin API key rotation and health tracking.
type KeyRotator struct {
	mu           sync.RWMutex
	keys         []string
	currentIndex int
	health       map[string]*KeyHealth
}

// NewKeyRotator initializes a new key rotator with the given keys.
func NewKeyRotator(keys []string) *KeyRotator {
	cleanKeys := make([]string, 0, len(keys))
	health := make(map[string]*KeyHealth)

	for _, k := range keys {
		clean := strings.TrimSpace(strings.Trim(k, `"'`))
		if clean != "" {
			cleanKeys = append(cleanKeys, clean)
			health[clean] = &KeyHealth{
				Key: clean,
			}
		}
	}

	return &KeyRotator{
		keys:         cleanKeys,
		currentIndex: 0,
		health:       health,
	}
}

// HasKeys returns true if there is at least one key configured.
func (r *KeyRotator) HasKeys() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.keys) > 0
}

// GetNextKey retrieves the next healthy, non-rate-limited key in round-robin sequence.
func (r *KeyRotator) GetNextKey() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	total := len(r.keys)
	if total == 0 {
		return "", ErrNoKeysConfigured
	}

	now := time.Now()

	// 1. Try to find a healthy key starting from current index
	for i := 0; i < total; i++ {
		idx := (r.currentIndex + i) % total
		k := r.keys[idx]
		h := r.health[k]

		// Check if cooldown has elapsed
		if h.IsRateLimited && now.After(h.RateLimitedUntil) {
			h.IsRateLimited = false
			h.ConsecutiveErrors = 0
		}

		if !h.IsRateLimited {
			r.currentIndex = (idx + 1) % total
			h.TotalCalls++
			return k, nil
		}
	}

	// 2. If all are rate limited, pick the key that will recover earliest
	var earliestKey string
	var earliestTime time.Time

	for _, k := range r.keys {
		h := r.health[k]
		if earliestKey == "" || h.RateLimitedUntil.Before(earliestTime) {
			earliestKey = k
			earliestTime = h.RateLimitedUntil
		}
	}

	return earliestKey, ErrAllKeysExhausted
}

// MarkRateLimited sets a cooldown period for a key (e.g. after receiving HTTP 429).
func (r *KeyRotator) MarkRateLimited(key string, cooldown time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if h, ok := r.health[key]; ok {
		h.IsRateLimited = true
		h.RateLimitedUntil = time.Now().Add(cooldown)
		h.ConsecutiveErrors++
	}
}

// MarkSuccess clears errors on a key after a successful response.
func (r *KeyRotator) MarkSuccess(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if h, ok := r.health[key]; ok {
		h.ConsecutiveErrors = 0
		h.IsRateLimited = false
		h.TotalSuccess++
	}
}

// MarkError records a general error for a key. If error threshold is met, puts key on short cooldown.
func (r *KeyRotator) MarkError(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if h, ok := r.health[key]; ok {
		h.ConsecutiveErrors++
		if h.ConsecutiveErrors >= 3 {
			h.IsRateLimited = true
			h.RateLimitedUntil = time.Now().Add(1 * time.Minute)
		}
	}
}

// KeyCount returns total active keys configured.
func (r *KeyRotator) KeyCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.keys)
}

// ParseCommaList splits a comma-separated string into clean slice of keys.
func ParseCommaList(raw string) []string {
	if raw == "" {
		return []string{}
	}
	parts := []string{}
	for _, p := range strings.Split(raw, ",") {
		clean := strings.TrimSpace(strings.Trim(p, `"'`))
		if clean != "" {
			parts = append(parts, clean)
		}
	}
	return parts
}
