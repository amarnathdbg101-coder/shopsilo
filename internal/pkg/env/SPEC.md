# SPEC: go-env

## 1. Metadata
- **Module**: `go-env`
- **Package**: `goenv`
- **Location**: `C:\Library\go-env`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `os`, `strconv`, `strings`, `time`, `bufio`, `fmt`, `errors`).

---

## 2. Core Capabilities
1. **Zero Reflection / High Speed**:
   - Direct type parsing without expensive struct reflection.
2. **Standard `.env` Loader**:
   - Supports comments (`#`), `export KEY=VALUE`, quoted values with newline unescaping, and preserves OS env by default (12-Factor App compliant).
   - `Overload()` available when explicit override is required.
3. **Type-Safe Accessors**:
   - `GetString`, `GetInt`, `GetInt64`, `GetFloat64`, `GetBool`, `GetDuration`, `GetSlice`.
4. **Validation & Fail-Fast Boot**:
   - `MustGet*` variants.
   - `Validate(requiredKeys ...)` returns aggregated missing variables report.
5. **Secret Masking for Startup Logs**:
   - `DumpMasked()` auto-redacts sensitive keys (`SECRET`, `PASSWORD`, `KEY`, `TOKEN`, `CREDENTIAL`, `DSN`) to prevent accidental log leaks.

---

## 3. Public API
```go
// Loader
func Load(filenames ...string) error
func Overload(filenames ...string) error
func ParseBytes(data []byte, override bool) error

// Accessors with Defaults
func GetString(key, defaultVal string) string
func GetInt(key string, defaultVal int) int
func GetInt64(key string, defaultVal int64) int64
func GetFloat64(key string, defaultVal float64) float64
func GetBool(key string, defaultVal bool) bool
func GetDuration(key string, defaultVal time.Duration) time.Duration
func GetSlice(key string, defaultVal []string, sep string) []string

// Fail-Fast Getters
func MustGetString(key string) (string, error)
func MustGetInt(key string) (int, error)
func MustGetBool(key string) (bool, error)

// Validation & Safety
func Validate(requiredKeys ...string) error
func DumpMasked() map[string]string
```
