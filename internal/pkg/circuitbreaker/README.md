# go-circuitbreaker ⚡

Universal, zero-dependency circuit breaker with named registry, canary probing, graceful fallbacks, and isolated panic safety for any Go operation. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- 🛡️ **Universal Protection**: Kisi bhi Go function (database queries, Redis, Razorpay payments, SMS gateways) ke upar wrap ho sakta hai.
- 🚦 **3-State Lifecycle**: `Closed` (normal) -> `Open` (tripped, fast-fails) -> `HalfOpen` (canary probe recovery).
- 🛑 **Cascading Outage Shield**: Jab downstream down hota hai, to requests block karke server threads exhaust hone se bachata hai.
- 🪂 **Graceful Fallbacks**: External service down hone par fallback function (e.g. read from cache or enqueue) execute karta hai.
- 🏢 **Central Registry**: Multiple named services (`"payment-service"`, `"sms-gateway"`) ko easily manage karta hai.
- 📦 **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. Wrapping an External Service Call
```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"
	gocb "go-circuitbreaker"
)

func main() {
	cb := gocb.New(gocb.Config{
		Name:                   "razorpay-payments",
		MaxConsecutiveFailures: 3,                // 3 failures par trip hoga
		CooldownDuration:       15 * time.Second, // 15s baad recovery test karega
		OnStateChange: func(name string, from, to gocb.State) {
			log.Printf("Circuit %s changed from %s to %s\n", name, from, to)
		},
	})

	ctx := context.Background()

	// Execute with automatic fallback
	err := cb.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			// External payment gateway call
			return callRazorpayAPI()
		},
		func(ctx context.Context, triggerErr error) error {
			// Fallback: order ko 'pending_verification' status me save karo
			fmt.Println("Payment service is down, queued order for retry:", triggerErr)
			return nil
		},
	)

	if err != nil {
		log.Println("Operation failed:", err)
	}
}

func callRazorpayAPI() error {
	// Simulate downstream network failure
	return fmt.Errorf("connection timeout")
}
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
