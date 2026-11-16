# go-colorlog 🪵

> **Structured Terminal Logger with Colors, Caller Tracing & Key-Value Fields for Go (Go 1.23+).**  
> Microservices, CLI binaries, aur backend APIs ke liye fast structured logging engine. ANSI color badges (Green INFO, Yellow WARN, Red ERROR), automatic `[file.go:line]` caller tracing, child logger context inheritance (`With`), aur zero-allocation buffer reuse.

---

## 🚀 Features

- **ANSI Color Badges**: Level badges aur key-value pairs terminal me instant visual scan provide karte hain.
- **Caller Line Tracing**: Automatically batata hai ki log statement kis file aur line number se trigger hua (`[main.go:42]`).
- **Context Inheritance (`With`)**: Request ID, tenant ID, ya trace context ko child logger me bind karke propagate karein.
- **Buffer Pool**: `sync.Pool` ke zariye write buffers reuse karta hai jisse allocations minimal rehte hain.
- **NO_COLOR Compliant**: CI/CD pipelines ya plain files me automatically escape codes disable ho jaate hain.
- **Zero External Dependencies**: Pure Go Standard Library!

---

## 📦 Quickstart

```go
package main

import (
	"go-colorlog"
)

func main() {
	// Standard structured log
	colorlog.Info("Server started successfully", "port", 8080, "env", "production")

	// Child logger with request context
	reqLogger := colorlog.With("request_id", "REQ-10928", "ip", "192.168.1.10")
	reqLogger.Warn("Slow database query detected", "duration_ms", 450)
	reqLogger.Error("Failed to charge credit card", "error", "gateway timeout")
}
```
