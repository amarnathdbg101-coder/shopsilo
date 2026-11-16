# go-envvalidator ⚙️

> **Zero-Reflection Schema-Driven Startup Environment Validator for Go (Go 1.23+).**  
> Production Go applications me missing ya invalid environment variable ki wajah se runtime par crashes hote hain. `go-envvalidator` boot time par saare environment rules validate karta hai aur ek sath saari missing/invalid variables ki report deta hai (taaki developer ko bar-bar ek-ek error fix na karna pade).

---

## 🚀 Features

- **Aggregated Error Reporting**: Pehle error par fail hone ke bajaye saare invalid/missing variables ek clean list me batata hai.
- **Type-Safe Casts**: `AsInt()`, `AsBool()`, `AsDuration()`, `AsURL()`, `Between(min, max)` auto-parsed aur validated milte hain.
- **Whitelist & Regex**: `OneOf("dev", "prod")` aur `Regex(...)` se strictly allowed values enforce karein.
- **Secret Masking**: `Masked()` lagane par `Dump()` diagnostic output me passwords aur keys `*** [REDACTED]` ho jaate hain.
- **Zero External Dependencies**: Pure Go Standard Library!

---

## 📦 Quickstart

```go
package main

import (
	"fmt"
	"go-envvalidator"
	"log"
)

func main() {
	schema := envvalidator.New()

	// 1. Declarative Schema Rules
	schema.Require("PORT").AsInt().Between(1024, 65535)
	schema.Require("DATABASE_URL").AsURL()
	schema.Require("JWT_SECRET").MinLength(32).Masked()
	schema.Require("ENVIRONMENT").OneOf("dev", "staging", "prod")
	schema.Optional("WORKER_TIMEOUT", "30s").AsDuration()

	// 2. Startup validation
	env, err := schema.Validate(nil) // nil uses os.Getenv
	if err != nil {
		log.Fatalf("❌ Environment Boot Error:\n%v", err)
	}

	// 3. Type-safe configuration access
	fmt.Printf("Server listening on port: %d\n", env.GetInt("PORT"))
	fmt.Printf("Worker timeout: %v\n", env.GetDuration("WORKER_TIMEOUT"))

	// Masked configuration log dump
	fmt.Println(env.Dump())
}
```
