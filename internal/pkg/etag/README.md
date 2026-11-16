# go-etag ⚡

> **RFC 7232 & RFC 9110 HTTP ETag Generator, Client-Side Caching Middleware & Optimistic Concurrency Engine for Go.**  
> Built with 100% Go Standard Library, Sub-Microsecond Hashing, HTTP 304 Bandwidth Reduction, and HTTP 412 Lost Update Prevention.

---

## ✨ Features

- 🚀 **Zero External Dependencies**: 100% Pure Go Standard Library (`net/http`, `crypto/sha1`, `hash/fnv`, `sync`).
- ⚡ **Bandwidth Saver (HTTP 304)**: `If-None-Match` match hone par empty body ke sath **304 Not Modified** return karke 95%+ egress bandwidth save karta hai.
- 🛡️ **Lost Update Shield (HTTP 412)**: `If-Match` check karke concurrent users ko ek dusre ke updates overwrite karne se rokta hai (Optimistic Concurrency Control).
- 🏷️ **Strong & Weak ETags**: Cryptographic SHA-1 strong validator (`"a1b2..."`) aur fast FNV-1a weak validator (`W/"x-y"`).
- 🔄 **Zero-Allocation Middleware**: `sync.Pool` memory pooling for buffering response bodies.
- 🌐 **RFC 9110 Compliant**: Supports wildcards (`*`), comma-separated headers, and weak/strong comparison semantics.

---

## 📦 Quickstart

### Method 1: Automatic HTTP Middleware (Easiest)

```go
package main

import (
	"net/http"

	goetag "github.com/amarnathdbg101-coder/go/go-etag"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id": 1, "name": "Basmati Rice", "price": 120.0}]`))
	})

	// Wrap router with go-etag Middleware
	handler := goetag.Middleware()(mux)

	http.ListenAndServe(":8080", handler)
}
```

#### Client Request / Response Flow:
1. **Client GET `/api/v1/products`**:
   - Response: `200 OK`, `ETag: "9b3c4f..."`, `Cache-Control: private, no-cache, must-revalidate`
2. **Client repeat GET `/api/v1/products` with `If-None-Match: "9b3c4f..."`**:
   - Response: `304 Not Modified` (0 bytes payload transmitted!)

---

### Method 2: Manual Optimistic Concurrency Control (OCC)

```go
func handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	currentVersion := uint64(14)
	lastUpdated := time.Now()
	currentETag := goetag.GenerateFromVersion(currentVersion, lastUpdated)

	// Check if client is updating against the latest version
	if !goetag.CheckIfMatch(w, r, currentETag) {
		// Automatically returns HTTP 412 Precondition Failed
		return
	}

	// Safe to execute mutation...
	w.WriteHeader(http.StatusOK)
}
```

---

## 🧪 Testing

```bash
cd C:\Library\go-etag
go test -v -count=1 ./...
```
