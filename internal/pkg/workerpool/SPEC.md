# SPEC: go-workerpool

## 1. Metadata
- **Module**: `go-workerpool`
- **Package**: `goworkerpool`
- **Location**: `C:\Library\go-workerpool`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `sync`, `context`, `errors`).

---

## 2. Core Capabilities
1. **Bounded Goroutine Control**:
   - Spawns a fixed number of long-lived worker goroutines consuming from a buffered job queue, preventing runaway goroutine spawning (OOM defense).
2. **Backpressure Modes**:
   - `Submit(job)`: Blocking submission until space frees up.
   - `TrySubmit(job)`: Non-blocking submission, instantly returning `ErrQueueFull` if the buffer is at capacity.
3. **Synchronous Execution Support (`SubmitWait`)**:
   - Submits a closure and blocks until it completes or the caller context expires.
4. **Panic Isolation**:
   - Traps inner panics, keeps worker goroutines alive, and forwards error to `WithOnPanic` callback.
5. **Graceful Draining & Fast Stop**:
   - `Stop()` drains the queue before exit.
   - `StopNow()` halts processing immediately via context cancellation.

---

## 3. Public API
```go
type Job func(ctx context.Context)
type Option func(*Pool)
func WithOnPanic(fn func(recovered any)) Option

type Pool struct { ... }
func New(workers, queueSize int, opts ...Option) *Pool
func (p *Pool) Submit(job Job) error
func (p *Pool) TrySubmit(job Job) error
func (p *Pool) SubmitWait(ctx context.Context, action func() error) error
func (p *Pool) Stop() error
func (p *Pool) StopNow() error
func (p *Pool) WorkerCount() int
```
