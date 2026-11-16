# SPEC: go-deadlock

## 1. Scope & Purpose
`go-deadlock` monitors synchronization primitives to diagnose and prevent concurrency deadlocks in high-throughput Go applications. It targets AB-BA lock inversion, recursive lock attempts on non-reentrant mutexes, and excessive lock hold times.

It explicitly does NOT modify Go runtime thread scheduling.

## 2. Public API Contract
```go
type Mutex struct { ... }
func (m *Mutex) Lock()
func (m *Mutex) Unlock()
func (m *Mutex) TryLock() bool

type RWMutex struct { ... }
func (rw *RWMutex) Lock()
func (rw *RWMutex) Unlock()
func (rw *RWMutex) RLock()
func (rw *RWMutex) RUnlock()

type Config struct { ... }
func SetConfig(c Config)
func GetConfig() Config
```

## 3. Concurrency & Memory Guarantees
- Lock acquisition graph is guarded by an internal fast mutex.
- Goroutine ID extraction runs in constant bounded time parsing `runtime.Stack` header.
- Graph automatically purges inactive goroutines when held locks reach zero.

## 4. How to Extend
- Add exporter hooks to push deadlock metrics to Prometheus or Sentry.
- Add thread-contention histogram metrics.
