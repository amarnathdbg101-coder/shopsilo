# go-tamperseal 🔏

> **Tamper-Proof Cryptographic Data Seal with Expiration for Go (Go 1.23+).**  
> Email verification links, password reset tokens, temporary asset downloads, aur webhook callbacks ke liye tamper-evident tokens banata hai. Database me state store kiye bina payload integrity aur expiration timestamp ko HMAC-SHA256 se cryptographically seal karta hai.

---

## 🚀 Features

- **Stateless Tokens**: Database me save kiye bina user ID, email, permissions encode karke sign karta hai.
- **Constant-Time Verification**: `subtle.ConstantTimeCompare` ka use karke timing attack vulnerabilities ko prevent karta hai.
- **Built-in Expiration (TTL)**: Expiration timestamp token ke signature ke sath bound hota hai (user timestamp modify nahi kar sakta).
- **Tamper Detection**: Payload ya signature me ek single character ka change aate hi `ErrSignatureMismatch` return hota hai.
- **Zero External Dependencies**: Pure Go Standard Library!

---

## 📦 Quickstart

```go
package main

import (
	"fmt"
	"go-tamperseal"
	"time"
)

type ResetToken struct {
	UserID string `json:"user_id"`
	Action string `json:"action"`
}

func main() {
	secret := []byte("master_app_secret_key_2026")

	// 1. 15-minute password reset link banayein
	payload := ResetToken{UserID: "usr_402", Action: "password_reset"}
	token, _ := tamperseal.Seal(secret, payload, 15*time.Minute)

	fmt.Printf("Password Reset Token: %s\n", token)

	// 2. Token verify aur unseal karein
	var verified ResetToken
	err := tamperseal.Unseal(secret, token, &verified)
	if err != nil {
		fmt.Println("Invalid or expired token:", err)
		return
	}

	fmt.Printf("✅ Authorized User: %s for %s\n", verified.UserID, verified.Action)
}
```
