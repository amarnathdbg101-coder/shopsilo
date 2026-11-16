# SPEC: go-tinylfu

## 1. Scope & Purpose
`go-tinylfu` provides an in-memory cache engine with near-optimal hit ratios using the W-TinyLFU eviction policy. It protects memory-bounded caches against scan pollution and burst thrashing.

It does NOT provide multi-node distributed cache replication (use Redis for cross-server caching).

## 2. Public API Contract
```go
type Metrics struct { ... }
type Cache[K comparable, V any] struct { ... }

func New[K comparable, V any](capacity int, opts ...Option) (*Cache[K, V], error)
func (c *Cache[K, V]) Get(key K) (V, bool)
func (c *Cache[K, V]) Set(key K, val V)
func (c *Cache[K, V]) Delete(key K)
func (c *Cache[K, V]) Metrics() Metrics
func (c *Cache[K, V]) Close() error
```

## 3. Concurrency & Memory Guarantees
- Thread-safe via internal `sync.Mutex`.
- Fixed memory upper bound determined strictly by `capacity`.
- Sketch memory scales as a tiny fraction ($< 1\%$) of total cache capacity.

## 4. How to Extend
- Add per-item TTL expiration wheel via `go-timingwheel`.
- Add singleflight dogpile coalescing via `go-cachekit`.
