# Technical Specification: go-otp

## 1. Domain & Purpose
`go-otp` is a zero-external-dependency, standard RFC-compliant implementation of Time-based One-Time Passwords (TOTP - RFC 6238) and HMAC-based One-Time Passwords (HOTP - RFC 4226). It powers Two-Factor Authentication (2FA) and Multi-Factor Authentication (MFA), fully compatible with Google Authenticator, Microsoft Authenticator, 1Password, and Authy.

## 2. Architecture & Design Principles
- **Zero Heavy Dependencies**: Pure Go standard library (`crypto/hmac`, `crypto/rand`, `crypto/sha1`, `crypto/sha256`, `crypto/sha512`, `encoding/base32`, `encoding/binary`).
- **RFC 4226 & RFC 6238 Compliance**: Validated against official IETF Appendix D test vectors.
- **Clock Drift Tolerance**: Configurable skew windows (e.g. ±30s) to accommodate client clock discrepancies.
- **Timing Attack Resistance**: Uses constant-time bitwise comparison to eliminate side-channel timing leaks.
- **Authenticator App Ready**: Built-in Base32 key generator and `otpauth://totp/...` URI builder for seamless QR code provisioning.

## 3. API Surface
- `type Algorithm string` (`SHA1`, `SHA256`, `SHA512`)
- `type Config struct { Digits int; Period time.Duration; Algorithm Algorithm; Issuer, AccountName string }`
- `func DefaultConfig() Config`
- `func GenerateSecret(byteLength int) (string, error)`
- `func GenerateHOTP(secretBase32 string, counter uint64, digits int, algo Algorithm) (string, error)`
- `func GenerateTOTP(secretBase32 string, t time.Time, cfg Config) (string, error)`
- `func VerifyTOTP(secretBase32 string, code string, t time.Time, skewWindows uint, cfg Config) bool`
- `func ProvisioningURI(secretBase32 string, cfg Config) string`

## 4. Error Handling
- `ErrInvalidSecret`: Corrupted or invalid Base32 string.
- `ErrInvalidCodeLength`: Digit length outside RFC 6-8 bounds.
- `ErrInvalidPeriod`: Time-step duration is non-positive.
- `ErrCodeMismatch`: Verification failed after checking skew window.

## 5. Verification & Testing
- Unit tests verify RFC 4226 Appendix D vectors (counters 0-9), dynamic secret generation, real-time TOTP generation and verification, clock-skew boundaries, and QR code URI generation.
