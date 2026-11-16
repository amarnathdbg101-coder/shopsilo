package tinylfu

import (
	"container/list"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

type cacheEntry[K comparable, V any] struct {
	key     K
	val     V
	hash    uint64
	element *list.Element
	segment int // 0: window, 1: probational, 2: protected
}

// Cache implements the Window TinyLFU eviction algorithm with Go Generics.
type Cache[K comparable, V any] struct {
	cfg Config
	mu  sync.Mutex

	sketch *cmSketch
	lookup map[K]*cacheEntry[K, V]

	windowList      *list.List
	probationalList *list.List
	protectedList   *list.List

	windowCap    int
	probationCap int
	protectCap   int

	hits      uint64
	misses    uint64
	evictions uint64
	addCount  uint64
	closed    bool
}

// New creates a new Window TinyLFU Cache instance.
func New[K comparable, V any](capacity int, opts ...Option) (*Cache[K, V], error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}

	cfg := DefaultConfig(capacity)
	for _, opt := range opts {
		opt(&cfg)
	}

	wCap := int(float64(cfg.Capacity) * cfg.WindowPercent)
	if wCap < 1 {
		wCap = 1
	}
	mainCap := cfg.Capacity - wCap
	protectCap := int(float64(mainCap) * cfg.ProtectedPercent)
	if protectCap < 1 {
		protectCap = 1
	}
	probationCap := mainCap - protectCap
	if probationCap < 1 {
		probationCap = 1
	}

	return &Cache[K, V]{
		cfg:             cfg,
		sketch:          newCMSketch(cfg.Capacity * 4),
		lookup:          make(map[K]*cacheEntry[K, V]),
		windowList:      list.New(),
		probationalList: list.New(),
		protectedList:   list.New(),
		windowCap:       wCap,
		probationCap:    probationCap,
		protectCap:      protectCap,
	}, nil
}

func hashKey[K comparable](key K) uint64 {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "%v", key)
	return h.Sum64()
}

// Get looks up a key, records frequency, and promotes entries across segments.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	var zero V
	h := hashKey(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return zero, false
	}

	// Record frequency in sketch
	c.sketch.increment(h)

	entry, exists := c.lookup[key]
	if !exists {
		atomic.AddUint64(&c.misses, 1)
		return zero, false
	}

	atomic.AddUint64(&c.hits, 1)

	// Promote/move within segments
	switch entry.segment {
	case 0:
		// Window LRU: move to front
		c.windowList.MoveToFront(entry.element)
	case 1:
		// Probational: promote to Protected on hit!
		c.probationalList.Remove(entry.element)
		entry.segment = 2
		entry.element = c.protectedList.PushFront(entry)
		// If protected overflowed, demote LRU of protected to probational
		if c.protectedList.Len() > c.protectCap {
			demoteElem := c.protectedList.Back()
			c.protectedList.Remove(demoteElem)
			demoted := demoteElem.Value.(*cacheEntry[K, V])
			demoted.segment = 1
			demoted.element = c.probationalList.PushFront(demoted)
		}
	case 2:
		// Protected: move to front
		c.protectedList.MoveToFront(entry.element)
	}

	return entry.val, true
}

// Set stores a key-value entry applying TinyLFU admission filtering.
func (c *Cache[K, V]) Set(key K, val V) {
	h := hashKey(key)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.sketch.increment(h)
	c.addCount++
	// Periodically halve sketch counters (aging factor)
	if c.addCount >= uint64(c.cfg.Capacity*10) {
		c.sketch.reset()
		c.addCount = 0
	}

	// 1. Existing entry update
	if entry, exists := c.lookup[key]; exists {
		entry.val = val
		switch entry.segment {
		case 0:
			c.windowList.MoveToFront(entry.element)
		case 1:
			c.probationalList.MoveToFront(entry.element)
		case 2:
			c.protectedList.MoveToFront(entry.element)
		}
		return
	}

	// 2. New entry: insert into Window LRU
	entry := &cacheEntry[K, V]{
		key:     key,
		val:     val,
		hash:    h,
		segment: 0,
	}
	entry.element = c.windowList.PushFront(entry)
	c.lookup[key] = entry

	// 3. If window exceeds capacity, candidate competes for main cache admission
	if c.windowList.Len() > c.windowCap {
		candElem := c.windowList.Back()
		c.windowList.Remove(candElem)
		candidate := candElem.Value.(*cacheEntry[K, V])

		c.admitCandidate(candidate)
	}
}

// admitCandidate evaluates whether candidate should displace a victim in probational segment.
func (c *Cache[K, V]) admitCandidate(candidate *cacheEntry[K, V]) {
	mainLen := c.probationalList.Len() + c.protectedList.Len()
	mainCap := c.probationCap + c.protectCap

	if mainLen < mainCap {
		// Main cache has free room: admit into Probational
		candidate.segment = 1
		candidate.element = c.probationalList.PushFront(candidate)
		return
	}

	// Main cache is full: compare candidate against victim at LRU end of Probational
	if c.probationalList.Len() == 0 {
		// If probational is empty, evict candidate directly
		delete(c.lookup, candidate.key)
		atomic.AddUint64(&c.evictions, 1)
		return
	}

	victimElem := c.probationalList.Back()
	victim := victimElem.Value.(*cacheEntry[K, V])

	candFreq := c.sketch.estimate(candidate.hash)
	victimFreq := c.sketch.estimate(victim.hash)

	// TinyLFU Admission Policy
	if candFreq > victimFreq {
		// Candidate wins: evict victim, admit candidate
		c.probationalList.Remove(victimElem)
		delete(c.lookup, victim.key)
		atomic.AddUint64(&c.evictions, 1)

		candidate.segment = 1
		candidate.element = c.probationalList.PushFront(candidate)
	} else {
		// Victim wins: reject candidate
		delete(c.lookup, candidate.key)
		atomic.AddUint64(&c.evictions, 1)
	}
}

// Delete removes an entry from the cache.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.lookup[key]
	if !exists {
		return
	}

	delete(c.lookup, key)
	switch entry.segment {
	case 0:
		c.windowList.Remove(entry.element)
	case 1:
		c.probationalList.Remove(entry.element)
	case 2:
		c.protectedList.Remove(entry.element)
	}
}

// Metrics returns real-time hit/miss metrics and calculated hit ratio.
func (c *Cache[K, V]) Metrics() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	hits := atomic.LoadUint64(&c.hits)
	misses := atomic.LoadUint64(&c.misses)
	total := hits + misses
	ratio := 0.0
	if total > 0 {
		ratio = float64(hits) / float64(total)
	}

	return Metrics{
		Hits:      hits,
		Misses:    misses,
		Evictions: atomic.LoadUint64(&c.evictions),
		HitRatio:  ratio,
		Size:      len(c.lookup),
		Capacity:  c.cfg.Capacity,
	}
}

// Close closes the cache.
func (c *Cache[K, V]) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	c.lookup = nil
	return nil
}
