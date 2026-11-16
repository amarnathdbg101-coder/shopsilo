# Technical Specification: go-urlsigner

## 1. Domain & Purpose
`go-urlsigner` provides cryptographically signed temporary URLs and presigned links in pure Go. It enables secure, time-limited access to private files (e.g. invoice PDFs, user media downloads), password reset links, and tamper-proof webhook endpoints without requiring authenticated session cookies.

## 2. Architecture & Design Principles
- **HMAC-SHA256 Integrity**: Binds the full URL path, scheme, host, and sorted query parameters with an expiration timestamp into an HMAC digest.
- **Timing Attack Resistance**: Uses constant-time `hmac.Equal` byte comparisons to prevent side-channel timing attacks.
- **Tamper Protection**: Any modification to URL path or query parameters invalidates the signature.
- **Expiration Enforcement**: Rejects requests that have surpassed the presigned Unix timestamp.
- **Zero Heavy Dependencies**: Pure Go standard library (`crypto/hmac`, `crypto/sha256`, `encoding/hex`, `net/url`, `strconv`, `time`).

## 3. API Surface
- `type Config struct { Secret []byte; ExpiresParam, SignatureParam string }`
- `type Signer struct`
- `func New(secret []byte) (*Signer, error)`
- `func NewWithConfig(cfg Config) (*Signer, error)`
- `func (s *Signer) Sign(rawURL string, ttl time.Duration) (string, error)`
- `func (s *Signer) Verify(signedURL string) error`

## 4. Error Handling
- `ErrExpired`: URL validity window has expired.
- `ErrInvalidSignature`: Signature mismatch or URL tampering.
- `ErrMissingSignature`: Signature parameter absent.
- `ErrMissingExpires`: Expiration timestamp absent.
- `ErrEmptySecret`: Secret key not supplied.

## 5. Verification & Testing
- Unit tests verify valid sign/verify roundtrips, tampered query parameter rejection, tampered path rejection, expired link rejection, and missing signature detection.
