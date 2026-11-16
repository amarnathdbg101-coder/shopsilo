# SPEC: go-saga

## 1. Metadata
- **Module**: `go-saga`
- **Package**: `saga`
- **Location**: `C:\Library\go-saga`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `context`, `sync`, `fmt`, `strings`, `errors`).

---

## 2. Core Capabilities
1. **Saga Distributed Transaction Orchestration**:
   - Executes multi-service business transactions sequentially. If any step fails, automatically triggers compensating rollback actions in strict reverse order ($N-1 \to 1$).
2. **Panic & Failure Isolation**:
   - Catches panics inside step execution or compensation functions without crashing the host process.
3. **Observability Hooks**:
   - `OnStepEvent(fn)` surfaces phase transitions (`executing`, `completed`, `failed`, `compensating`, `compensated`).
4. **Compensation Failure Tracking**:
   - Aggregates any errors during rollback and surfaces `ErrCompensationFailed` with a detailed audit trail.

---

## 3. Public API
```go
type StepAction func(ctx context.Context) error

type Step struct {
    Name       string
    Execute    StepAction
    Compensate StepAction
}

type Coordinator struct
func New() *Coordinator
func (c *Coordinator) AddStep(name string, execute, compensate StepAction) *Coordinator
func (c *Coordinator) OnStepEvent(fn func(name string, phase string, err error)) *Coordinator
func (c *Coordinator) Run(ctx context.Context) error
```
