# go-security 🔒

> **Unified, Production-Grade Security & Authentication Suite for Go.**  
> Combines Auth, Dual JWTs, Bcrypt, Crypto OTP, AES-256-GCM, and PII Masking into a single modular Go package.

---

## 📦 Sub-packages

- `github.com/shopsilo/go-security/auth`: Complete Authentication Engine & UserStore
- `github.com/shopsilo/go-security/jwt`: HS256 Token Signing & Claims Verification
- `github.com/shopsilo/go-security/otp`: TOTP, HOTP, and Crypto 6-digit OTP engine
- `github.com/shopsilo/go-security/crypto`: AES-256-GCM, HMAC, PBKDF2, UUIDv7
- `github.com/shopsilo/go-security/mask`: PII Data Redactor & Masker

---

## ⚡ 3-Minute Quickstart

```go
package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shopsilo/go-security/auth"
)

func main() {
	store := auth.NewMemoryUserStore()
	cfg := auth.DefaultConfig("super-secret-jwt-key-at-least-16-bytes")

	engine, err := auth.New(store, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer engine.Close()

	r := chi.NewRouter()
	r.Mount("/auth", engine.Routes())

	http.ListenAndServe(":8080", r)
}
```
