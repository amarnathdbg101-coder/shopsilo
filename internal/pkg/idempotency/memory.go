package idempotency

import (
	"context"
	"sync"
	"time"
)

type memoryLock struct {
	expiresAt time.Time
}

// MemoryStorage provides an in-memory, zero-dependency implementation of Storage.
type MemoryStorage struct {
	mu          sync.RWMutex
	records     map[string]*Record
	locks       map[string]memoryLock
	stopCleanup chan struct{}
	closeOnce   sync.Once
	isClosed    bool
}

// NewMemoryStorage creates and initializes a new MemoryStorage instance with background TTL janitor.
func NewMemoryStorage(cleanupInterval time.Duration) *MemoryStorage {
	if cleanupInterval <= 0 {
		cleanupInterval = 30 * time.Second
	}

	m := &MemoryStorage{
		records:     make(map[string]*Record),
		locks:       make(map[string]memoryLock),
		stopCleanup: make(chan struct{}),
	}

	go m.janitor(cleanupInterval)
	return m
}

func (m *MemoryStorage) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.evictExpired()
		case <-m.stopCleanup:
			return
		}
	}
}

func (m *MemoryStorage) evictExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for k, rec := range m.records {
		if now.After(rec.ExpiresAt) {
			delete(m.records, k)
		}
	}

	for k, lk := range m.locks {
		if now.After(lk.expiresAt) {
			delete(m.locks, k)
		}
	}
}

// Lock acquires an in-flight execution lock for the key.
func (m *MemoryStorage) Lock(_ context.Context, key string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isClosed {
		return false, ErrStorageClosed
	}

	now := time.Now()
	if lk, exists := m.locks[key]; exists && now.Before(lk.expiresAt) {
		return false, nil
	}

	m.locks[key] = memoryLock{
		expiresAt: now.Add(ttl),
	}
	return true, nil
}

// Unlock releases the in-flight lock for the key.
func (m *MemoryStorage) Unlock(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isClosed {
		return ErrStorageClosed
	}

	delete(m.locks, key)
	return nil
}

// Get retrieves a cached Record.
func (m *MemoryStorage) Get(_ context.Context, key string) (*Record, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.isClosed {
		return nil, ErrStorageClosed
	}

	rec, exists := m.records[key]
	if !exists {
		return nil, nil
	}

	if time.Now().After(rec.ExpiresAt) {
		return nil, nil
	}

	return rec, nil
}

// Set stores a Record with a specified TTL.
func (m *MemoryStorage) Set(_ context.Context, key string, record *Record, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isClosed {
		return ErrStorageClosed
	}

	record.ExpiresAt = time.Now().Add(ttl)
	m.records[key] = record
	return nil
}

// Delete removes a cached record.
func (m *MemoryStorage) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isClosed {
		return ErrStorageClosed
	}

	delete(m.records, key)
	delete(m.locks, key)
	return nil
}

// Close gracefully stops the background janitor and clears storage.
func (m *MemoryStorage) Close() error {
	var err error
	m.closeOnce.Do(func() {
		m.mu.Lock()
		m.isClosed = true
		m.records = nil
		m.locks = nil
		m.mu.Unlock()

		close(m.stopCleanup)
	})
	return err
}
