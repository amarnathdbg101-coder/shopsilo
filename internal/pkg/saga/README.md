# go-saga 🔄

> **Distributed Transaction Saga Orchestrator for Go Microservices (Go 1.23+).**  
> Microservices me distributed ACID transactions impossible hote hain. Saga pattern me agar Step 3 (Payment) fail ho jaye, toh Coordinator automatically Step 2 (Inventory) aur Step 1 (Order) ke Compensating Actions ko strict reverse order me call karta hai taaki data inconsistent na ho.

---

## 🚀 Features

- **Forward Execution & Backward Rollback**: Sequential steps execute hote hain; fail hote hi completed steps reverse sequence me compensate ho jate hain.
- **Panic Protection**: Kisi bhi step me runtime panic aane par use catch karke rollback trigger kiya jata hai.
- **Observability Hooks**: `OnStepEvent` se har step ka live status (`executing`, `failed`, `compensating`) log aur trace kiya ja sakta hai.
- **Compensation Failure Alerting**: Agar koi rollback action network ya database issue se fail ho jaye, toh detailed audit error report milta hai.
- **Zero External Dependencies**: Pure Go Standard Library!

---

## 📦 Quickstart

```go
package main

import (
	"context"
	"fmt"
	"go-saga"
)

func main() {
	coordinator := saga.New()

	// 1. Order Step
	coordinator.AddStep("create_order",
		func(ctx context.Context) error {
			fmt.Println("1. Order created in Database")
			return nil
		},
		func(ctx context.Context) error {
			fmt.Println("🔙 Compensating: Cancel Order")
			return nil
		},
	)

	// 2. Payment Step (Simulated Failure)
	coordinator.AddStep("process_payment",
		func(ctx context.Context) error {
			return fmt.Errorf("bank gateway declined card")
		},
		func(ctx context.Context) error {
			fmt.Println("🔙 Compensating: Refund Payment")
			return nil
		},
	)

	// 3. Run Saga
	err := coordinator.Run(context.Background())
	if err != nil {
		fmt.Printf("Saga Completed with Error: %v\n", err)
	}
}
```
