# go-bulkhead 🛡️

> **Netflix/Hystrix-Style Concurrency Partitioning, Queue Limiting, and Backpressure Engine for Go (Go 1.23+).**  
> Prevents cascading microservice failures by isolating resource consumption and rejecting excess load before it crashes your backend.

---

## 🚀 Key Features

1. **Dual Boundary Protection**:
   - `MaxConcurrent`: Caps simultaneous executions (e.g. 10).
   - `MaxQueue`: Caps waiting requests (e.g. 20) with a hard timeout (`QueueTimeout`).
2. **Instant Rejection (Fail-Fast Backpressure)**:
   - When capacity is exceeded, returns `ErrBulkheadFull` in under 1 microsecond.
3. **Queue Timeout Defense**:
   - Stale requests waiting in queue beyond `QueueTimeout` are rejected with `ErrQueueTimeout` to prevent thundering herds.
4. **Real-Time Operational Telemetry**:
   - Inspect `Metrics()` for active slots, queued count, total executed, rejected, and timeout events.
5. **Zero Dependencies**:
   - Pure Go standard library (`sync`, `context`, `time`).

---

## 📦 Quickstart

```go
package main

import (
	"context"
	"fmt"
	"time"

	"go-bulkhead"
)

func main() {
	// Isolate payment gateway calls to max 5 concurrent with 10 queue slots
	paymentBulkhead := bulkhead.New(
		bulkhead.WithMaxConcurrent(5),
		bulkhead.WithMaxQueue(10),
		bulkhead.WithQueueTimeout(500*time.Millisecond),
		bulkhead.WithOnRejected(func(reason error) {
			fmt.Println("ALERT: Payment gateway bulkhead saturated:", reason)
		}),
	)
	defer paymentBulkhead.Close()

	ctx := context.Background()

	// Execute with typed return value
	res, err := bulkhead.ExecuteWithResult(paymentBulkhead, ctx, func() (string, error) {
		// Call payment API...
		return "PAYMENT_TX_SUCCESS", nil
	})

	if err != nil {
		fmt.Println("Execution failed or rejected:", err)
		return
	}
	fmt.Println("Result:", res)
}
```
