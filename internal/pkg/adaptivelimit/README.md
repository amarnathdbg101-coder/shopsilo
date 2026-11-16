# go-adaptivelimit 🚦

> **Latency-Driven, Self-Tuning Adaptive Concurrency Limiter for Go (Go 1.23+).**  
> Inspired by TCP Vegas and Netflix Concurrency Limits, it automatically expands throughput during healthy traffic and throttles incoming load when downstream latency inflates.

---

## 🚀 Key Features

1. **TCP Vegas Congestion Avoidance**:
   - Calculates gradient $\frac{\text{minRTT}}{\text{currentRTT}}$ to dynamically estimate internal queuing delay without needing external CPU/RAM agents.
2. **AIMD Tuning (Additive Increase, Multiplicative Decrease)**:
   - Expands limit $+1$ during healthy low-latency cycles.
   - Rapidly cuts limit (e.g. $20\%$ backoff) when downstream queries or external APIs spike in latency.
3. **Fail-Fast Backpressure**:
   - Immediately returns `ErrLimitExceeded` (or HTTP 503) instead of accumulating queued threads that exhaust host memory.
4. **Standard `net/http` Middleware**:
   - Plugs directly into `chi`, `net/http`, or any Go web framework.
5. **Zero Dependencies**:
   - 100% pure standard library (`sync`, `time`, `net/http`, `context`).

---

## 📦 Quickstart

```go
package main

import (
	"net/http"
	"go-adaptivelimit"
)

func main() {
	// 1. Initialize adaptive limiter
	limiter := adaptivelimit.New(
		adaptivelimit.WithInitialLimit(10),
		adaptivelimit.WithLimits(2, 100),
	)
	defer limiter.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/checkout", func(w http.ResponseWriter, r *http.Request) {
		// Business logic...
		w.Write([]byte(`{"status":"success"}`))
	})

	// 2. Wrap with adaptive limiter middleware
	http.ListenAndServe(":8080", adaptivelimit.Middleware(limiter)(mux))
}
```
