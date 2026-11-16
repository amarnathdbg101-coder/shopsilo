# SPEC: go-adaptivelimit

## 1. Scope & Purpose
`go-adaptivelimit` protects Go microservices and backends from server death under unexpected load or degraded downstream dependencies. It replaces brittle static QPS thresholds with self-tuning concurrency limits derived from real-time round-trip latency.

It does NOT provide client IP rate limiting or token buckets (use `go-ratelimit`).

## 2. Public API Contract
```go
type Limiter struct { ... }
type ReleaseFunc func()

func New(opts ...Option) *Limiter
func (l *Limiter) Acquire(ctx context.Context) (ReleaseFunc, error)
func (l *Limiter) Execute(ctx context.Context, fn func() error) error
func (l *Limiter) Limit() float64
func (l *Limiter) Metrics() Metrics
func (l *Limiter) Close() error

func Middleware(limiter *Limiter) func(http.Handler) http.Handler
```

## 3. Concurrency & Memory Guarantees
- Thread-safe state managed by `sync.Mutex`.
- Mathematical moving average and gradient calculations are $O(1)$ and allocate zero heap memory.
- `ReleaseFunc` is guarded by `sync.Once` to prevent double-release corruption.

## 4. How to Extend
- Add Gradient2 or BBR (Bottleneck Bandwidth and RTT) algorithms.
- Export metrics directly into Prometheus via `go-metrics`.
