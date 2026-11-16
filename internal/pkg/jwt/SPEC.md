# SPEC: go-jwt

## 1. Metadata
- **Module**: `go-jwt`
- **Package**: `gojwt`
- **Location**: `C:\Library\go-jwt`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `crypto/hmac`, `crypto/sha256`, `crypto/subtle`, `encoding/base64`, `encoding/json`, `strings`, `time`, `errors`, `fmt`).

---

## 2. Core Capabilities
1. **HMAC-SHA256 (HS256) Signing & Verification**:
   - RFC 7519 compliant JSON Web Token encoding and decoding.
   - Side-channel timing attack immunity using `subtle.ConstantTimeCompare`.
2. **Standard & Custom Claims**:
   - Built-in `sub`, `iss`, `aud`, `exp`, `nbf`, `iat`, `jti` support + flexible custom map payload.
3. **Temporal Validation**:
   - Enforces expiration (`exp`) and not-before (`nbf`) with configurable clock skew leeway (`time.Duration`).

---

## 3. Public API
```go
type Claims struct {
    Subject   string         `json:"sub,omitempty"`
    Issuer    string         `json:"iss,omitempty"`
    Audience  string         `json:"aud,omitempty"`
    ExpiresAt int64          `json:"exp,omitempty"`
    NotBefore int64          `json:"nbf,omitempty"`
    IssuedAt  int64          `json:"iat,omitempty"`
    ID        string         `json:"jti,omitempty"`
    Custom    map[string]any `json:"custom,omitempty"`
}
func (c *Claims) Valid(leeway time.Duration) error

type Engine struct { ... }
func New(secret []byte, leeway ...time.Duration) (*Engine, error)
func (e *Engine) Sign(claims Claims) (string, error)
func (e *Engine) Verify(tokenStr string) (*Claims, error)

func Sign(secret []byte, claims Claims) (string, error)
func Verify(secret []byte, tokenStr string) (*Claims, error)
```
