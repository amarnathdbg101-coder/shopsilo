# SPEC: go-cors

## 1. Metadata
- **Module**: `go-cors`
- **Package**: `gocors`
- **Location**: `C:\Library\go-cors`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `net/http`, `strings`, `strconv`).

---

## 2. Core Capabilities
1. **Zero-Allocation Execution**:
   - Pre-computes header strings (`Access-Control-Allow-Methods`, `Headers`, `ExposedHeaders`, `MaxAge`) at startup to eliminate string formatting on every HTTP request.
2. **Preflight Optimization**:
   - Short-circuits `OPTIONS` requests immediately with `204 No Content`, terminating before reaching backend business logic routes.
3. **Subdomain Wildcards**:
   - Permissive yet secure matching: `https://*.domain.com` matches `https://api.domain.com` without allowing arbitrary attacker origins.
4. **Credential Support**:
   - Supports `Access-Control-Allow-Credentials: true` with strict dynamic origin reflection (satisfying the browser security requirement disallowing `*` with credentials).

---

## 3. Public API
```go
type Config struct {
    AllowedOrigins   []string
    AllowedMethods   []string
    AllowedHeaders   []string
    ExposedHeaders   []string
    AllowCredentials bool
    MaxAge           int
}

var DefaultConfig Config

type CORS struct { ... }
func New(cfg ...Config) *CORS
func (c *CORS) IsOriginAllowed(origin string) bool
func (c *CORS) Handler(next http.Handler) http.Handler
func Middleware(cfg ...Config) func(http.Handler) http.Handler
```
