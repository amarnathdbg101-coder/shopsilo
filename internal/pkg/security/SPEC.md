# SPEC: go-security

## 1. Scope & Purpose
Unified, high-performance security suite for Go applications.
Consolidates Auth, JWT, OTP, Bcrypt, AES-GCM Crypto, PII Masking, and Cookie Encryption into one clean, modular, zero-bloat package.

## 2. Public API Sub-packages
- `github.com/shopsilo/go-security/auth`: Authentication, UserStore, Login/Register/OTP handlers
- `github.com/shopsilo/go-security/jwt`: HS256 Token Signing & Claims Verification
- `github.com/shopsilo/go-security/otp`: TOTP, HOTP, and Crypto 6-digit OTP engine
- `github.com/shopsilo/go-security/crypto`: AES-256-GCM, HMAC, PBKDF2, UUIDv7
- `github.com/shopsilo/go-security/mask`: PII Data Redactor & Masker

## 3. Concurrency & Memory Guarantees
- Sub-millisecond execution.
- Thread-safe storage with sync.RWMutex.
