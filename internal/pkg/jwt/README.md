# go-jwt 🔑

Ultra-fast, zero-dependency HMAC-SHA256 (HS256) JWT engine with constant-time verification, clock skew tolerance, and custom claims support. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- ⚡ **Zero External Dependencies**: 100% pure Go standard library (no bloated 3rd-party auth deps).
- 🛡️ **Side-Channel Resistant**: `subtle.ConstantTimeCompare` signature verification prevents timing side-channel attacks.
- ⏱️ **Clock Skew Leeway**: Microservices ke beech 1-5 second time drift ke bawajood false expiration prevent karta hai.
- 📜 **RFC 7519 Compliant**: Standard HS256 JWT tokens compatible with React, Flutter, and iOS/Android clients.

---

## Quick Start

### 1. Generating & Verifying Tokens
```go
package main

import (
	"fmt"
	"log"
	"time"
	gojwt "go-jwt"
)

func main() {
	secret := []byte("my_super_secret_auth_key")
	engine, _ := gojwt.New(secret, 5*time.Second) // 5s clock skew leeway

	// Sign
	token, err := engine.Sign(gojwt.Claims{
		Subject:   "user_102",
		Issuer:    "ShopSilo-Auth",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		Custom: map[string]any{
			"role": "admin",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Signed JWT:", token)

	// Verify
	claims, err := engine.Verify(token)
	if err != nil {
		log.Fatal("Invalid token:", err)
	}
	fmt.Printf("Authenticated User: %s (Role: %s)\n", claims.Subject, claims.Custom["role"])
}
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
