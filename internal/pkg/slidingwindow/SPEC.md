# SPEC: go-slidingwindow

## 1. Metadata
- **Module**: `go-slidingwindow`
- **Package**: `slidingwindow`
- **Location**: `C:\Library\go-slidingwindow`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `sync`, `time`, `errors`).

---

## 2. Core Capabilities
1. **Rolling Time Window Ring Buffer**:
   - Divides a total time window into $N$ granular sub-buckets stored in a fixed-size ring array. Old buckets automatically overwrite without background timer goroutines.
2. **$O(1)$ Increments & Aggregate Counts**:
   - `Inc(delta)` updates the current temporal slice in constant time.
   - `Count()` and `Rate()` return event counts and instantaneous throughput (events/second) across the rolling window.
3. **Atomic Rate Limiter**:
   - `Allow(limit)` atomically evaluates window threshold and increments only if within capacity.
4. **Multi-Tenant Isolation**:
   - `KeyedLimiter` partitions sliding windows across dynamic keys (User IDs, IP addresses, API Tokens).

---

## 3. Public API
```go
type Window struct
func New(windowSize time.Duration, bucketCount int) (*Window, error)
func (w *Window) Inc(delta int64)
func (w *Window) Count() int64
func (w *Window) Rate() float64
func (w *Window) Allow(limit int64) bool
func (w *Window) Reset()

type KeyedLimiter struct
func NewKeyedLimiter(windowSize time.Duration, bucketCount int, limit int64) (*KeyedLimiter, error)
func (kl *KeyedLimiter) Allow(key string) bool
func (kl *KeyedLimiter) Count(key string) int64
```
