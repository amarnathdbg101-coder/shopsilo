# SPEC: go-circuitbreaker

## 1. Metadata
- **Module**: `go-circuitbreaker`
- **Package**: `gocb`
- **Location**: `C:\Library\go-circuitbreaker`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `sync`, `time`, `context`, `fmt`, `errors`).

---

## 2. Core Capabilities
1. **Universal Execution Protection**:
   - Wraps any arbitrary Go function (SQL query, Redis command, gRPC, HTTP call, payment gateway).
2. **Three-State Lifecycle (`Closed`, `Open`, `HalfOpen`)**:
   - Consecutive failure tracking trips breaker to `Open`.
   - Rejects traffic immediately with `ErrCircuitOpen` while open (fast-fail).
   - Cooldown period transitions to `HalfOpen` for canary probe.
   - Successful canary immediately resets to `Closed`.
3. **Graceful Fallbacks**:
   - `ExecuteWithFallback` allows running degraded cached results or queueing operations when external service is down.
4. **Isolated Panic Recovery**:
   - Traps inner panics, converts them to structured errors, and records a failure event.
5. **Named Central Registry**:
   - `Registry` organizes circuit breakers across distinct downstream resources (`payment`, `email`, `inventory`).

---

## 3. Public API
```go
type State int
const (
    StateClosed State = iota
    StateHalfOpen
    StateOpen
)

type Config struct {
    Name                   string
    MaxConsecutiveFailures int
    CooldownDuration       time.Duration
    HalfOpenMaxProbes      int
    OnStateChange          func(name string, from, to State)
}

type CircuitBreaker struct { ... }
func New(cfg Config) *CircuitBreaker
func (cb *CircuitBreaker) Execute(ctx context.Context, action func(ctx context.Context) error) error
func (cb *CircuitBreaker) ExecuteWithFallback(ctx context.Context, action func(ctx context.Context) error, fallback func(ctx context.Context, err error) error) error
func (cb *CircuitBreaker) State() State
func (cb *CircuitBreaker) Reset()

type Registry struct { ... }
func NewRegistry() *Registry
func (r *Registry) Get(name string) (*CircuitBreaker, bool)
func (r *Registry) GetOrRegister(name string, cfg Config) *CircuitBreaker
func (r *Registry) All() map[string]*CircuitBreaker
```
