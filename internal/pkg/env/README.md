# go-env ⚙️

Zero-reflection, high-speed environment configuration loader and parser for production Go backends. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- 🚀 **Zero Reflection**: Extreme performance, no heavy struct tags or magic reflection overhead.
- 📄 **Native `.env` Loader**: Full support for quoted strings, escaped newlines (`\n`), comments (`#`), and `export` prefixes.
- 🛡️ **12-Factor Compliant**: Preserves existing OS environment variables by default.
- 🔒 **Automated Secret Masking**: `DumpMasked()` hides passwords, tokens, API keys, and database DSNs in startup logs.
- ⚡ **Strong Type Safety**: Direct getters for `string`, `int`, `int64`, `float64`, `bool` (understands `1`, `true`, `on`, `yes`), `time.Duration`, and slices.
- 📦 **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. Load `.env` & Read Config
```go
package main

import (
	"fmt"
	"log"
	"time"
	goenv "go-env"
)

func main() {
	// Automatically loads .env if present
	_ = goenv.Load()

	// Safe type getters with fallbacks
	port := goenv.GetInt("PORT", 8080)
	appName := goenv.GetString("APP_NAME", "ShopSilo Core")
	debug := goenv.GetBool("DEBUG", false)
	timeout := goenv.GetDuration("HTTP_TIMEOUT", 10*time.Second)
	allowedOrigins := goenv.GetSlice("CORS_ORIGINS", []string{"*"}, ",")

	fmt.Printf("%s running on :%d (Debug: %v, Timeout: %s, Origins: %v)\n",
		appName, port, debug, timeout, allowedOrigins)
}
```

### 2. Startup Validation (Fail-Fast)
```go
// Server boot hone se pehle check karo ki mandatory keys maujood hain ya nahi
err := goenv.Validate("DATABASE_URL", "JWT_SECRET", "AWS_S3_BUCKET")
if err != nil {
    log.Fatalf("Server startup blocked: %v", err)
}
```

### 3. Safe Configuration Logging (Zero Secret Leaks)
```go
// Print all env vars safely without exposing DB password or JWT keys
for k, v := range goenv.DumpMasked() {
    log.Printf("%s=%s\n", k, v)
}
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
