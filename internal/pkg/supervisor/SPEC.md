# SPEC: go-supervisor

## 1. Scope & Purpose
`go-supervisor` provides Erlang/OTP-style lifecycle supervision for long-running goroutines in Go services. It prevents entire application termination on worker panics and coordinates automated restarts.

It does NOT provide inter-process OS process management (e.g. systemd).

## 2. Public API Contract
```go
type WorkerFunc func(ctx context.Context) error
type WorkerSpec struct { Name, Policy, Backoff, Func }
type Supervisor struct { ... }

func New(opts ...Option) *Supervisor
func (s *Supervisor) Add(spec WorkerSpec) error
func (s *Supervisor) Supervise(name string, fn WorkerFunc) error
func (s *Supervisor) Start(ctx context.Context) error
func (s *Supervisor) Stop() error
```

## 3. Concurrency & Memory Guarantees
- Individual worker lifecycles run in isolated goroutines.
- Panic stack traces are captured using `runtime/debug.Stack()`.
- Thread-safe coordination managed by `sync.Mutex` and `sync.WaitGroup`.

## 4. How to Extend
- Add dynamic supervisor tree nesting (supervisors supervising child supervisors).
- Add health check probes (heartbeat pings per worker).
