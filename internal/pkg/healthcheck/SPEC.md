# SPEC: go-healthcheck

## 1. Metadata
- **Module**: `go-healthcheck`
- **Package**: `gohealthcheck`
- **Location**: `C:\Library\go-healthcheck`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `net/http`, `context`, `encoding/json`, `time`, `sync`, `runtime`, `errors`, `fmt`).

---

## 2. Core Capabilities
1. **Kubernetes Liveness & Readiness Probes**:
   - `/livez`: Liveness probe for process alive verification (returns 200 OK).
   - `/readyz`: Readiness probe for dependency checks (returns 200 OK if all dependencies are healthy, 503 Service Unavailable if any dependency fails).
2. **Concurrent Asynchronous Execution**:
   - Executes all registered checkers in parallel via goroutines with individual timeout enforcement.
3. **Built-in Throttling & TTL Caching**:
   - Caches readiness check results for a configurable TTL (e.g. 1s) to prevent hammering downstream databases during traffic spikes.
4. **Standard Ecosystem Checkers**:
   - `PingChecker`: compatible with any struct implementing `PingContext(ctx)` (`*sql.DB` PostgreSQL/MySQL/SQLite, Redis clients).
   - `MemoryChecker`: heap allocation threshold alert.
   - `NewCheckFunc`: wrap any custom Go closure or microservice ping.

---

## 3. Public API
```go
type Status string
const (
    StatusUp   Status = "UP"
    StatusDown Status = "DOWN"
)

type CheckResult struct {
    Status     Status  `json:"status"`
    DurationMs float64 `json:"duration_ms"`
    Error      string  `json:"error,omitempty"`
}

type Checker interface {
    Name() string
    Check(ctx context.Context) error
}

func NewPingChecker(name string, pinger Pinger) Checker
func NewMemoryChecker(name string, maxAllocBytes uint64) Checker
func NewCheckFunc(name string, fn func(ctx context.Context) error) Checker

type Health struct { ... }
func New(cfg ...Config) *Health
func (h *Health) RegisterReadiness(checker Checker)
func (h *Health) RegisterLiveness(checker Checker)
func (h *Health) CheckReadiness(ctx context.Context) Report
func (h *Health) CheckLiveness(ctx context.Context) Report
func (h *Health) ReadyHandler() http.HandlerFunc
func (h *Health) LiveHandler() http.HandlerFunc
```
