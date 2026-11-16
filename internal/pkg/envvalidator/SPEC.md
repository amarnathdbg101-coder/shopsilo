# SPEC: go-envvalidator

## 1. Metadata
- **Module**: `go-envvalidator`
- **Package**: `envvalidator`
- **Location**: `C:\Library\go-envvalidator`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `os`, `strconv`, `net/url`, `regexp`, `strings`, `time`, `fmt`, `errors`).

---

## 2. Core Capabilities
1. **Schema-Driven Startup Environment Verification**:
   - Compiles declarative rules for required and optional environment variables before the HTTP server or database connections initialize.
2. **Aggregated Diagnostic Error Reporting**:
   - Avoids single-error crash loops by accumulating every missing, unparseable, or out-of-bounds variable into a single clean formatted list.
3. **Type-Safe Casts & Bounds Checking**:
   - Supports `AsInt().Between(min, max)`, `AsBool()`, `AsDuration()`, `AsURL()`, `OneOf(...)`, `Regex(...)`, and `MinLength(...)`.
4. **Secret Masking & Redaction**:
   - `Masked()` flags sensitive secrets (passwords, JWT private keys, API credentials) to prevent accidental log leaks in `Dump()`.

---

## 3. Public API
```go
type Rule struct
func (r *Rule) AsInt() *Rule
func (r *Rule) Between(min, max int) *Rule
func (r *Rule) AsBool() *Rule
func (r *Rule) AsDuration() *Rule
func (r *Rule) AsURL() *Rule
func (r *Rule) OneOf(allowed ...string) *Rule
func (r *Rule) Regex(pattern string) *Rule
func (r *Rule) MinLength(length int) *Rule
func (r *Rule) Masked() *Rule

type Schema struct
func New() *Schema
func (s *Schema) Require(key string) *Rule
func (s *Schema) Optional(key, defaultVal string) *Rule
func (s *Schema) Validate(lookup func(string) string) (*Environment, error)

type Environment struct
func (e *Environment) GetString(key string) string
func (e *Environment) GetInt(key string) int
func (e *Environment) GetBool(key string) bool
func (e *Environment) GetDuration(key string) time.Duration
func (e *Environment) Dump() string
```
