# go-shutdown 🛑

Phased, graceful application shutdown coordinator for Kubernetes, Docker, and systemd. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- 🚦 **Phased Execution**: Database ko HTTP requests complete hone se pehle close hone se bachata hai.
  - **Phase 1 (HTTP/Ingress)**: `server.Shutdown(ctx)` stops accepting new requests.
  - **Phase 2 (Workers)**: `pool.Stop()` drains background task queues.
  - **Phase 3 (Storage)**: `db.Close()` safely releases database connections.
- ⏱️ **Hard Timeout Guard**: Kubernetes pod termination deadline (e.g. 15s) par hanging tasks ko forcefully exit karta hai.
- 📡 **Signal Trapping**: Intercepts `SIGTERM`, `SIGINT`, and `os.Interrupt`.
- 📦 **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. Production Shutdown Pipeline
```go
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"
	goshutdown "go-shutdown"
)

func main() {
	db, _ := sql.Open("postgres", "...")
	server := &http.Server{Addr: ":8080"}

	// 15 seconds global shutdown timeout
	shutdown := goshutdown.New(15 * time.Second)

	// Phase 1: Stop HTTP traffic
	shutdown.RegisterHTTP("http_server", func(ctx context.Context) error {
		log.Println("Stopping HTTP server...")
		return server.Shutdown(ctx)
	})

	// Phase 2: Close database connections
	shutdown.RegisterStorage("database", func(ctx context.Context) error {
		log.Println("Closing database pool...")
		return db.Close()
	})

	go server.ListenAndServe()

	// Blocks until SIGTERM or Ctrl+C
	if err := shutdown.Wait(); err != nil {
		log.Printf("Shutdown finished with error: %v\n", err)
	}
	log.Println("Application stopped cleanly.")
}
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
