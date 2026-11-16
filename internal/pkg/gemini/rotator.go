package gogemini

import (
	"sync"
	"time"
)

// KeyRotator orchestrates round-robin multi-key rotation with automatic 429 rate-limit quarantine.
type KeyRotator struct {
	mu          sync.Mutex
	keys        []string
	cooldowns   map[string]time.Time
	currentIdx  int
	cooldownDur time.Duration
}

// NewKeyRotator initializes a thread-safe KeyRotator.
func NewKeyRotator(keys []string, cooldown time.Duration) (*KeyRotator, error) {
	validKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != "" {
			validKeys = append(validKeys, k)
		}
	}

	if len(validKeys) == 0 {
		return nil, ErrNoAPIKeys
	}
	if cooldown <= 0 {
		cooldown = 60 * time.Second
	}

	return &KeyRotator{
		keys:        validKeys,
		cooldowns:   make(map[string]time.Time),
		cooldownDur: cooldown,
	}, nil
}

// GetKey retrieves the next available unquarantined API key in round-robin order.
func (r *KeyRotator) GetKey() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	n := len(r.keys)

	for i := 0; i < n; i++ {
		idx := (r.currentIdx + i) % n
		key := r.keys[idx]

		cooldownUntil, onCooldown := r.cooldowns[key]
		if !onCooldown || now.After(cooldownUntil) {
			if onCooldown {
				delete(r.cooldowns, key)
			}
			r.currentIdx = (idx + 1) % n
			return key, nil
		}
	}

	return "", ErrAllKeysExhausted
}

// MarkRateLimited places an API key into temporary quarantine after receiving HTTP 429.
func (r *KeyRotator) MarkRateLimited(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cooldowns[key] = time.Now().Add(r.cooldownDur)
}
