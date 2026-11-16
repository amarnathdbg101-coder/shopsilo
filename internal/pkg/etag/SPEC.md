# Technical Specification: go-etag

## 1. Domain & Purpose
`go-etag` is a high-performance, zero-dependency HTTP ETag generator, client-side caching middleware, and optimistic concurrency locking engine compliant with RFC 7232 and RFC 9110. It reduces egress bandwidth by up to 95% via HTTP `304 Not Modified` responses and eliminates mid-air update collisions via HTTP `412 Precondition Failed`.

## 2. Architecture & Design Principles
- **RFC 7232 & 9110 Compliance**: Strictly implements weak (`W/`) and strong validator semantics, wildcard `*` matching, and comma-separated header candidate parsing.
- **Dual Verification Modes**:
  - `CheckIfNoneMatch`: Short-circuits cache hits with empty body and status 304, relieving downstream network serialization.
  - `CheckIfMatch`: Solves the Lost Update Problem in REST APIs by ensuring the resource version on the client matches the current state before executing mutations.
- **Buffer Pool Recycling**: Middleware uses `sync.Pool` to avoid memory allocations while buffering response payloads.
- **Zero Third-Party Dependencies**: 100% pure Go standard library (`net/http`, `crypto/sha1`, `hash/fnv`, `sync`, `time`).

## 3. API Surface
- `type Mode int (ModeStrong, ModeWeak)`
- `func Generate(data []byte) string`
- `func GenerateWeak(data []byte) string`
- `func GenerateFromString(s string) string`
- `func GenerateFromVersion(version uint64, lastModified time.Time) string`
- `func Matches(headerValue, currentETag string, isWeakComparison bool) bool`
- `func CheckIfNoneMatch(w http.ResponseWriter, r *http.Request, currentETag string) bool`
- `func CheckIfMatch(w http.ResponseWriter, r *http.Request, currentETag string) bool`
- `func SetETag(w http.ResponseWriter, etag string, cacheControl ...string)`
- `func Middleware(opts ...Option) func(http.Handler) http.Handler`

## 4. Error Handling
- `ErrEmptyPayload`: Payload is empty.
- `ErrPreconditionFailed`: If-Match mismatch (HTTP 412).
- `ErrNotModified`: Resource unmodified (HTTP 304).

## 5. Verification & Testing
- Validates strong/weak hashing, wildcard matching, list extraction, HTTP 304 response with empty body, and HTTP 412 mutation rejection.
