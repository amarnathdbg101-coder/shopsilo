# go-retry 🔁

Standalone, context-aware function retry engine with full jitter, exponential backoff, conditional error filtering, and Go Generics support. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- 🔄 **Universal & Generics Ready**: `Do()` kisi bhi function ke liye, aur `DoWithData[T]()` direct return value receive karne ke liye.
- ⚡ **Anti-Thundering Herd (Full Jitter)**: Uniform random jitter sabhi clients ko ek saath server par retry karne se rokta hai.
- 🛑 **Conditional Retries (`WithRetryIf`)**: Only network drops ya 503 errors retry karo; 400 Bad Request ya Validation errors ko attempt 1 par hi stop karo.
- ⏱️ **Context Cancellation Safe**: Agar user ne HTTP request cancel kar di, to background sleep turant terminate ho jati hai.
- 📦 **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. Retrying a Database Operation with Exponential Backoff
```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"
	goretry "go-retry"
)

func main() {
	ctx := context.Background()

	// Generic function execution with automatic retry
	userEmail, err := goretry.DoWithData(
		ctx,
		func(ctx context.Context) (string, error) {
			// Transient DB query simulation
			return queryUserEmail(42)
		},
		goretry.WithMaxAttempts(4),
		goretry.WithBackoff(goretry.Exponential(100*time.Millisecond, 2*time.Second, 2.0)),
		goretry.WithJitter(goretry.FullJitter),
		goretry.WithOnRetry(func(attempt int, err error, delay time.Duration) {
			log.Printf("Attempt %d failed (%v). Retrying in %v...\n", attempt, err, delay)
		}),
	)

	if err != nil {
		log.Fatalf("All retries exhausted: %v", err)
	}

	fmt.Println("Fetched User Email:", userEmail)
}

func queryUserEmail(id int) (string, error) {
	return "aman@example.com", nil
}
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
