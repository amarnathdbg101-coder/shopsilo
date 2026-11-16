# go-cors 🌐

Zero-allocation, high-speed CORS middleware with preflight short-circuiting, wildcard subdomain matching, and credentials support for Flutter Web and React SPAs. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- 🚀 **Zero Memory Allocation**: Pre-computes saare CORS header strings (`Allow-Methods`, `Headers`, `Max-Age`) startup par hi.
- ⚡ **Instant Preflight (`OPTIONS`)**: `OPTIONS` preflight requests ko 204 No Content ke saath turant short-circuit karta hai without executing heavy database routes.
- 🔒 **Subdomain Wildcards**: `https://*.shopsilo.com` secure wildcard matching without opening security holes.
- 🍪 **Credentials Ready**: Browser rule ke anusaar `AllowCredentials: true` hone par dynamic matching origin reflect karta hai (not `*`).
- 📦 **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. Attaching to Chi / net/http Router
```go
package main

import (
	"net/http"
	gocors "go-cors"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/products", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id": 1, "name": "Wireless Mouse"}]`))
	})

	// Setup high-speed CORS middleware
	corsMW := gocors.Middleware(gocors.Config{
		AllowedOrigins:   []string{"https://*.shopsilo.com", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours browser preflight cache
	})

	http.ListenAndServe(":8080", corsMW(mux))
}
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
