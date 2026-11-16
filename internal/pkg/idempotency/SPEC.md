# SPEC: go-idempotency

## 1. Scope & Purpose
`go-idempotency` guarantees exactly-once execution semantics for mutating HTTP endpoints (`POST`, `PUT`, `PATCH`, `DELETE`). It shields payment gateways, checkout routes, and ledger transactions from duplicate button clicks, network retry storms, and race conditions. It does not intercept read-only requests (`GET`, `HEAD`, `OPTIONS`).

## 2. Public API Contract
- `type Storage interface`: Pluggable driver contract for distributed backends (Redis, PostgreSQL) and local dev.
- `type MemoryStorage struct`: Built-in zero-dependency thread-safe store with background TTL eviction and panic safety.
- `type Engine struct`: Orchestrator handling header extraction, concurrency locking, and response replay.
- `func New(opts ...Option) (*Engine, error)`: Engine constructor with functional options.
- `func (e *Engine) Middleware(next http.Handler) http.Handler`: Standard `net/http` middleware.

## 3. Concurrency & Memory Guarantees
- Single-Flight Locking: Prevents two simultaneous requests with the same idempotency key from running in parallel.
- Status 409 Conflict: Returned automatically if a key is currently in-flight.
- Memory leak protection: Background janitor runs on a clean ticker with `stopCleanup` channel, fully freed on `Close()`.
- Mutex contention minimization: Read/write locks (`sync.RWMutex`) partitioned by key.

## 4. How to Extend
To attach Redis or PostgreSQL, implement the `Storage` interface (`Lock`, `Unlock`, `Get`, `Set`, `Delete`, `Close`) and pass via `idempotency.WithStorage(myStore)`.
