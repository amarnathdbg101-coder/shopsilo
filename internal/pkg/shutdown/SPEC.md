# SPEC: go-shutdown

## 1. Metadata
- **Module**: `go-shutdown`
- **Package**: `goshutdown`
- **Location**: `C:\Library\go-shutdown`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `os`, `os/signal`, `syscall`, `sync`, `context`, `time`, `sort`, `fmt`).

---

## 2. Core Capabilities
1. **Phased Graceful Shutdown**:
   - Executes teardown in ascending phase priority:
     - `Phase 1 (PhaseHTTP)`: Stops HTTP/gRPC ingress, allowing in-flight requests to complete without accepting new traffic.
     - `Phase 2 (PhaseWorkers)`: Drains worker pools, event buses, and background queues.
     - `Phase 3 (PhaseStorage)`: Closes database pools, Redis connections, and open file descriptors.
2. **Global Timeout Enforcement**:
   - Caps total shutdown duration (e.g. 15s) using `context.WithTimeout`, preventing frozen containers in Kubernetes deployments.
3. **OS Signal Interception**:
   - Traps `SIGTERM`, `SIGINT`, and `os.Interrupt`.

---

## 3. Public API
```go
const (
    PhaseHTTP    = 1
    PhaseWorkers = 2
    PhaseStorage = 3
)

type Hook func(ctx context.Context) error

type Coordinator struct { ... }
func New(timeout time.Duration, signals ...os.Signal) *Coordinator
func (c *Coordinator) Register(phase int, name string, fn Hook)
func (c *Coordinator) RegisterHTTP(name string, fn Hook)
func (c *Coordinator) RegisterWorker(name string, fn Hook)
func (c *Coordinator) RegisterStorage(name string, fn Hook)
func (c *Coordinator) Trigger(ctx context.Context) error
func (c *Coordinator) Wait() error
```
