# go-w3ctrace 🌐

> **Zero-Dependency Distributed Tracing and W3C TraceContext Recommendation Engine for Go (Go 1.23+).**  
> Propagate, parse, and inject standard `traceparent` and `tracestate` headers across microservices without the massive OpenTelemetry dependency footprint.

---

## 🚀 Key Features

1. **Official W3C TraceContext Standard Compliant**:
   - Parses and formats `00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01` per the W3C Recommendation.
   - Enforces strict validation: forbids all-zero IDs, enforces 32-char / 16-char hex boundaries.
2. **Sub-Microsecond Context Propagation**:
   - Propagates trace context through standard `context.Context` without heap leaks or thread locks.
3. **Parent-Child Span Hierarchy**:
   - Child spans inherit root `TraceID` and link to parent `SpanID`.
4. **Plug-and-Play HTTP Middleware**:
   - Automatically extracts incoming `traceparent` headers from upstream services or API gateways and injects outgoing headers in downstream responses.
5. **Pluggable Exporters**:
   - Standard `Exporter` interface. Includes `MemoryExporter` for in-memory testing and buffering.
6. **Zero Dependencies**:
   - 100% standard library (`crypto/rand`, `encoding/hex`, `net/http`, `context`, `sync`, `time`).

---

## 📦 Quickstart

### 1. HTTP Middleware Setup

```go
package main

import (
	"net/http"
	"go-w3ctrace"
)

func main() {
	tracer := w3ctrace.NewTracer("order-service", w3ctrace.NewMemoryExporter())
	mux := http.NewServeMux()

	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		// Active span is automatically present in request context
		span := w3ctrace.SpanFromContext(r.Context())
		span.SetAttribute("customer.tier", "VIP")

		// Create child span for database operation
		_, dbSpan := tracer.StartSpan(r.Context(), "db.query_orders")
		// ... query database ...
		dbSpan.End()

		w.Write([]byte(`{"status":"success"}`))
	})

	// Wrap with W3C Trace middleware
	http.ListenAndServe(":8080", w3ctrace.Middleware(tracer)(mux))
}
```
