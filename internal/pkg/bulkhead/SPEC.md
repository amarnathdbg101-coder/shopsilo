# SPEC: go-bulkhead

## 1. Scope & Purpose
`go-bulkhead` implements the Bulkhead design pattern to enforce resource isolation between disparate services, endpoints, or background tasks. It prevents slow or failing dependencies from saturating thread pools and exhausting host resources.

It does NOT handle retries or circuit breakers (use `go-circuitbreaker` or `go-retry`).

## 2. Public API Contract
```go
type Bulkhead struct { ... }
func New(opts ...Option) *Bulkhead
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error
func (b *Bulkhead) TryExecute(fn func() error) error
func (b *Bulkhead) ActiveCount() int
func (b *Bulkhead) QueuedCount() int
func (b *Bulkhead) Metrics() Metrics
func (b *Bulkhead) Close() error

func ExecuteWithResult[T any](b *Bulkhead, ctx context.Context, fn func() (T, error)) (T, error)
```

## 3. Concurrency & Memory Guarantees
- Channel-based semaphore ensures zero lock contention for active slot acquisition.
- Mutex-guarded queue counter prevents integer overflow and race conditions.
- Zero goroutine leaks: all timers are stopped via `defer timer.Stop()`.

## 4. How to Extend
- Attach to HTTP middleware router (e.g. `chi` or `net/http`) to partition incoming endpoints by customer tier.
- Expose metrics to Prometheus via `go-metrics`.
