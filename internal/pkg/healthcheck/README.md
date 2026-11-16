# go-healthcheck 🩺

Lightweight, concurrent Kubernetes & Docker liveness (`/livez`) and readiness (`/readyz`) health check coordinator for Go microservices. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- ☸️ **Kubernetes Ready**: Direct drop-in handlers for `/livez` (liveness) and `/readyz` (readiness) with standard HTTP 200/503 status codes.
- ⚡ **Concurrent Execution**: Saare dependency checks parallel goroutines me execute hote hain with strict timeout enforcement.
- 🛑 **Database Protection (TTL Cache)**: High frequency polling ya prometheus scrapers se database overload na ho isliye built-in result caching.
- 🔌 **Pluggable Architecture**: Native support for `database/sql` (`*sql.DB`), Redis, Memory stats, and custom functions.
- 📦 **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. Integration with Chi / net/http
```go
package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	gohealth "go-healthcheck"
	_ "github.com/lib/pq"
)

func main() {
	// Connect to PostgreSQL
	db, _ := sql.Open("postgres", "postgres://user:pass@localhost:5432/shopsilo?sslmode=disable")

	// Initialize Health Coordinator
	h := gohealth.New(gohealth.Config{
		Timeout:  2 * time.Second,
		CacheTTL: 1 * time.Second, // 1s TTL protects DB from scraper storms
	})

	// Register dependencies
	h.RegisterReadiness(gohealth.NewPingChecker("postgres", db))
	h.RegisterReadiness(gohealth.NewMemoryChecker("memory", 500*1024*1024)) // 500MB alert

	// Setup HTTP Endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", h.LiveHandler())
	mux.HandleFunc("/readyz", h.ReadyHandler())

	http.ListenAndServe(":8080", mux)
}
```

### 2. Sample Output (`/readyz`)
```json
{
  "status": "UP",
  "timestamp": "2026-11-07T23:14:13Z",
  "total_duration_ms": 1.45,
  "details": {
    "postgres": {
      "status": "UP",
      "duration_ms": 0.85
    },
    "memory": {
      "status": "UP",
      "duration_ms": 0.05
    }
  }
}
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
