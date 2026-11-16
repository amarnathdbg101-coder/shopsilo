package golru

import (
	"sync"
	"time"
)

type node[K comparable, V any] struct {
	key       K
	val       V
	expiresAt int64 // Unix nanoseconds, 0 = no expiration
	prev      *node[K, V]
	next      *node[K, V]
}

func (n *node[K, V]) isExpired(now int64) bool {
	return n.expiresAt > 0 && now >= n.expiresAt
}

// OnEvictFunc defines a callback triggered when an item is evicted from cache.
type OnEvictFunc[K comparable, V any] func(key K, val V)

// Cache is a thread-safe, high-speed generic Least Recently Used (LRU) cache with per-item TTL.
type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]*node[K, V]
	head     *node[K, V] // Sentinel head (head.next is MRU)
	tail     *node[K, V] // Sentinel tail (tail.prev is LRU)
	onEvict  OnEvictFunc[K, V]
	mu       sync.RWMutex
}

// New creates an LRU Cache with specified capacity and optional onEvict callback.
func New[K comparable, V any](capacity int, onEvict OnEvictFunc[K, V]) (*Cache[K, V], error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}

	c := &Cache[K, V]{
		capacity: capacity,
		items:    make(map[K]*node[K, V], capacity),
		onEvict:  onEvict,
		head:     &node[K, V]{},
		tail:     &node[K, V]{},
	}

	// Link sentinels
	c.head.next = c.tail
	c.tail.prev = c.head

	return c, nil
}

// Get retrieves a key's value and marks it as Most Recently Used.
// Returns false if key is absent or has expired.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero V
	n, ok := c.items[key]
	if !ok {
		return zero, false
	}

	now := time.Now().UnixNano()
	if n.isExpired(now) {
		c.removeNode(n)
		if c.onEvict != nil {
			c.onEvict(n.key, n.val)
		}
		return zero, false
	}

	c.moveToFront(n)
	return n.val, true
}

// Set adds or updates an entry. If capacity is saturated, evicts the Least Recently Used item.
func (c *Cache[K, V]) Set(key K, val V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt int64
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).UnixNano()
	}

	if n, ok := c.items[key]; ok {
		n.val = val
		n.expiresAt = expiresAt
		c.moveToFront(n)
		return
	}

	// Evict LRU if capacity exceeded
	if len(c.items) >= c.capacity {
		c.evictOldest()
	}

	newNode := &node[K, V]{
		key:       key,
		val:       val,
		expiresAt: expiresAt,
	}

	c.insertAtFront(newNode)
	c.items[key] = newNode
}

// Peek returns the value without updating the LRU position or checking expiration.
func (c *Cache[K, V]) Peek(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var zero V
	n, ok := c.items[key]
	if !ok {
		return zero, false
	}
	return n.val, true
}

// Contains checks if key is present and non-expired.
func (c *Cache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	n, ok := c.items[key]
	if !ok {
		return false
	}
	return !n.isExpired(time.Now().UnixNano())
}

// Remove deletes a specific key from cache.
func (c *Cache[K, V]) Remove(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	n, ok := c.items[key]
	if !ok {
		return false
	}

	c.removeNode(n)
	if c.onEvict != nil {
		c.onEvict(n.key, n.val)
	}
	return true
}

// Purge removes all keys from cache.
func (c *Cache[K, V]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.onEvict != nil {
		for _, n := range c.items {
			c.onEvict(n.key, n.val)
		}
	}

	c.items = make(map[K]*node[K, V], c.capacity)
	c.head.next = c.tail
	c.tail.prev = c.head
}

// Len returns current item count.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Cap returns cache maximum capacity.
func (c *Cache[K, V]) Cap() int {
	return c.capacity
}

// Keys returns all active keys in Most-to-Least recently used order.
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, len(c.items))
	curr := c.head.next
	for curr != c.tail {
		keys = append(keys, curr.key)
		curr = curr.next
	}
	return keys
}

// Internal node manipulation helpers (assumes caller holds write lock)

func (c *Cache[K, V]) insertAtFront(n *node[K, V]) {
	n.next = c.head.next
	n.prev = c.head
	c.head.next.prev = n
	c.head.next = n
}

func (c *Cache[K, V]) removeNode(n *node[K, V]) {
	n.prev.next = n.next
	n.next.prev = n.prev
	delete(c.items, n.key)
}

func (c *Cache[K, V]) moveToFront(n *node[K, V]) {
	// Unlink
	n.prev.next = n.next
	n.next.prev = n.prev
	// Insert at front
	c.insertAtFront(n)
}

func (c *Cache[K, V]) evictOldest() {
	oldest := c.tail.prev
	if oldest == c.head {
		return
	}
	c.removeNode(oldest)
	if c.onEvict != nil {
		c.onEvict(oldest.key, oldest.val)
	}
}
