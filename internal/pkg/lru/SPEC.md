# Technical Specification: go-lru

## 1. Domain & Purpose
`go-lru` is a high-speed, thread-safe, generic Least Recently Used (LRU) cache with per-item TTL expiration and eviction notifications in pure Go. It delivers strict memory bounds by discarding the oldest, least accessed elements when capacity is saturated.

## 2. Architecture & Design Principles
- **$O(1)$ Hash Table + Doubly-Linked List**: Combines a hash map for instantaneous key lookups with a sentinel-bounded doubly linked list to track access recency.
- **Custom Doubly-Linked Node**: Avoids `container/list` interface boxing and type assertions, achieving zero-allocation node traversals.
- **Per-Item TTL Expiration**: Supports independent expiration durations per entry, evaluated lazily upon access.
- **Eviction Lifecycle Hooks**: Invokes `OnEvict(key, val)` when items are pruned due to capacity limits or expired TTL.
- **Zero Heavy Dependencies**: Pure Go standard library (`sync`, `time`).

## 3. API Surface
- `type OnEvictFunc[K comparable, V any] func(key K, val V)`
- `type Cache[K comparable, V any] struct`
- `func New[K comparable, V any](capacity int, onEvict OnEvictFunc[K, V]) (*Cache[K, V], error)`
- `func (c *Cache[K, V]) Get(key K) (V, bool)`
- `func (c *Cache[K, V]) Set(key K, val V, ttl time.Duration)`
- `func (c *Cache[K, V]) Peek(key K) (V, bool)`
- `func (c *Cache[K, V]) Contains(key K) bool`
- `func (c *Cache[K, V]) Remove(key K) bool`
- `func (c *Cache[K, V]) Purge()`
- `func (c *Cache[K, V]) Len() int`
- `func (c *Cache[K, V]) Cap() int`
- `func (c *Cache[K, V]) Keys() []K`

## 4. Error Handling
- `ErrInvalidCapacity`: Capacity <= 0.

## 5. Verification & Testing
- Unit tests verify $O(1)$ capacity clamping, MRU/LRU sequence updates upon access, Peek preservation of order, TTL lazy eviction, and concurrent thread-safe execution across goroutines.
