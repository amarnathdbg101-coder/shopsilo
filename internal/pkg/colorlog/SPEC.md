# SPEC: go-colorlog

## 1. Metadata
- **Module**: `go-colorlog`
- **Package**: `colorlog`
- **Location**: `C:\Library\go-colorlog`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `os`, `io`, `fmt`, `time`, `sync`, `runtime`, `path/filepath`, `bytes`, `strings`).

---

## 2. Core Capabilities
1. **Zero-Allocation Terminal Logging**:
   - Reuses write buffers via `sync.Pool` to avoid heap allocations on frequent logging statements.
2. **Distinct ANSI Level Badges & Key-Value Colors**:
   - Formats level badges with prominent colors (Green INFO, Yellow WARN, Red ERROR, Magenta DEBUG).
   - Cyan keys and neutral values for high CLI readability.
3. **Automatic Caller File & Line Tracing**:
   - Inspects `runtime.Caller` to log `[file.go:line]` without third-party stack unwrappers.
4. **Context Chaining & NO_COLOR Standard**:
   - `logger.With("req_id", 123)` creates lightweight child loggers.
   - Automatically disables escape codes when `NO_COLOR` is present.

---

## 3. Public API
```go
type Level int
const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
    LevelFatal
)

type Logger struct
func New(opts ...Option) *Logger
func (l *Logger) SetLevel(lvl Level)
func (l *Logger) With(keyvals ...any) *Logger
func (l *Logger) Debug(msg string, keyvals ...any)
func (l *Logger) Info(msg string, keyvals ...any)
func (l *Logger) Warn(msg string, keyvals ...any)
func (l *Logger) Error(msg string, keyvals ...any)

func Debug(msg string, keyvals ...any)
func Info(msg string, keyvals ...any)
func Warn(msg string, keyvals ...any)
func Error(msg string, keyvals ...any)
func With(keyvals ...any) *Logger
```
