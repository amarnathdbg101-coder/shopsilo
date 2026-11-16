# go-slidingwindow ⏱️

> **High-Precision Rolling Time Window Rate Limiter & Counter for Go (Go 1.23+).**  
> Throughput tracking (requests per second/minute), error rate spike detection, aur multi-tenant API rate limiting ke liye constant-time $O(1)$ sliding window ring buffer. Bina kisi background timer thread ke current time ke according automatic expire hota hai.

---

## 🚀 Features

- **Ring Buffer Sub-Buckets**: Ek window ko $N$ slices me divide karta hai (e.g. 60 seconds window with 60 1-second buckets).
- **$O(1)$ Time Complexity**: Counter increment aur aggregate sum calculation constant time me execute hote hain.
- **Zero-Allocation**: Pre-allocated slice me ring-buffer index modulo arithmetic se bucket reuse hoti hai.
- **Keyed Multi-Tenant Limiter**: User ID, IP address ya API key ke basis par per-client sliding window rate limits.
- **Zero External Dependencies**: Pure Go Standard Library!

---

## 📦 Quickstart

```go
package main

import (
	"fmt"
	"go-slidingwindow"
	"time"
)

func main() {
	// 1-minute rolling window, 60 buckets (1 sec resolution), limit = 100 req/min
	limiter, _ := slidingwindow.NewKeyedLimiter(1*time.Minute, 60, 100)

	apiKey := "client_xyz_99"

	if limiter.Allow(apiKey) {
		fmt.Println("✅ Request Allowed")
	} else {
		fmt.Println("❌ 429 Too Many Requests")
	}

	fmt.Printf("Current requests in last 60s: %d\n", limiter.Count(apiKey))
}
```
