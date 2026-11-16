# SPEC: go-retry

## 1. Metadata
- **Module**: `go-retry`
- **Package**: `goretry`
- **Location**: `C:\Library\go-retry`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `context`, `time`, `math/rand`, `errors`, `fmt`).

---

## 2. Core Capabilities
1. **Universal Function Retry Engine**:
   - `Do(ctx, action)` wraps any Go operation.
   - `DoWithData[T](ctx, action)` provides type-safe generics for returning values without pointer boilerplate.
2. **Pluggable Backoff Algorithms**:
   - `Exponential(initial, max, multiplier)`
   - `Constant(interval)`
   - `Linear(initial, step, max)`
3. **Thundering Herd Jitter Prevention**:
   - `FullJitter` (uniform random delay from 0 to calculated backoff).
   - `EqualJitter` (half fixed, half random).
   - `NoJitter` (exact deterministic backoff).
4. **Conditional Retries (`RetryIf`)**:
   - Only retries transient errors; stops immediately on non-retriable business or validation errors.
5. **Context-Aware Safety**:
   - Cancellation or deadline expiry halts pending retry waits instantly.

---

## 3. Public API
```go
type JitterMode int
const (
    FullJitter JitterMode = iota
    EqualJitter
    NoJitter
)

type Backoff interface {
    NextDelay(attempt int) time.Duration
}

func Exponential(initial, max time.Duration, multiplier float64) Backoff
func Constant(interval time.Duration) Backoff
func Linear(initial, step, max time.Duration) Backoff
func ApplyJitter(d time.Duration, mode JitterMode) time.Duration

type Option func(*Config)
func WithMaxAttempts(n int) Option
func WithBackoff(b Backoff) Option
func WithJitter(mode JitterMode) Option
func WithRetryIf(fn func(err error) bool) Option
func WithOnRetry(fn func(attempt int, err error, delay time.Duration)) Option

func Do(ctx context.Context, action func(ctx context.Context) error, opts ...Option) error
func DoWithData[T any](ctx context.Context, action func(ctx context.Context) (T, error), opts ...Option) (T, error)
```
