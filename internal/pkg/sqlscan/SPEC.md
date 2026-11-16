# SPEC: go-sqlscan

## 1. Metadata
- **Module**: `go-sqlscan`
- **Package**: `sqlscan`
- **Location**: `C:\Library\go-sqlscan`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `database/sql`, `reflect`, `sync`, `strings`, `errors`, `fmt`).

---

## 2. Core Capabilities
1. **Reflection-Cached Struct Scanning**:
   - Analyzes struct field tags (`db:"col_name"`) once and caches field index trees inside a thread-safe map (`sync.RWMutex`), achieving sub-microsecond scanning speeds over large SQL result sets.
2. **Type-Safe Generics Accessors**:
   - `ScanOne[T](rows)`: Scans the first matching row into `*T` and auto-closes `rows`.
   - `ScanAll[T](rows)`: Scans the entire result set into a slice `[]T` and auto-closes `rows`.
3. **Embedded Struct & Discard Tolerant**:
   - Recursively maps anonymous embedded structs.
   - Automatically skips columns returned by SQL queries that do not have corresponding struct fields without returning errors.

---

## 3. Public API
```go
func ScanRowStruct(rows *sql.Rows, dest any) error
func ScanOne[T any](rows *sql.Rows) (*T, error)
func ScanAll[T any](rows *sql.Rows) ([]T, error)
```
