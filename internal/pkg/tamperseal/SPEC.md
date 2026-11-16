# SPEC: go-tamperseal

## 1. Metadata
- **Module**: `go-tamperseal`
- **Package**: `tamperseal`
- **Location**: `C:\Library\go-tamperseal`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `crypto/hmac`, `crypto/sha256`, `crypto/subtle`, `encoding/base64`, `encoding/hex`, `encoding/json`, `time`, `errors`).

---

## 2. Core Capabilities
1. **Cryptographic Tamper-Proof Sealing**:
   - Encodes JSON payloads into URL-safe strings paired with an HMAC-SHA256 digital signature to verify integrity without database lookups.
2. **Timing-Attack Resilient Verification**:
   - Compares HMAC signatures using `crypto/subtle.ConstantTimeCompare`.
3. **Temporal Validity & TTL Enforcement**:
   - Embeds expiration timestamps in tokens, rejecting expired payloads with `ErrTokenExpired`.
4. **Lightweight Token Structure**:
   - Format: `<base64_payload>.<timestamp>.<hex_hmac>`. Ideal for email verification links, password resets, and presigned webhook callbacks.

---

## 3. Public API
```go
func Seal(secret []byte, payload any, ttl time.Duration) (string, error)
func Unseal(secret []byte, token string, dest any) error
func IsExpired(token string) (bool, error)
```
