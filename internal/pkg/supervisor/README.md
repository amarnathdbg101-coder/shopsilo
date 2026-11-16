# go-supervisor 🛡️

> **Erlang/OTP-Style Goroutine Supervision, Panic Isolation, and Crash-Loop Protection for Go (Go 1.23+).**  
> Protects background goroutines from terminating the entire Go OS process upon unexpected panics, and orchestrates automatic restarts with `OneForOne` and `OneForAll` strategies.

---

## 🚀 Key Features

1. **Panic Isolation & Zero-Downtime Recovery**:
   - Isolates unexpected nil pointers and panics within the worker enclosure, recording stack traces and automatically restarting.
2. **Erlang/OTP Supervision Strategies**:
   - `OneForOne`: Only restarts the failed worker.
   - `OneForAll`: Restarts all supervised workers if one critical worker fails.
3. **Flexible Restart Policies**:
   - `PERMANENT`: Always restart.
   - `TRANSIENT`: Restart only upon error or panic (clean `nil` exits are preserved).
   - `TEMPORARY`: Never restart.
4. **Crash Loop Circuit Breaker**:
   - Halts restarts if a worker exceeds `MaxRestarts` within `RestartWindow` to avoid CPU spinning.
5. **Zero Dependencies**:
   - 100% pure Go standard library (`sync`, `context`, `time`, `runtime/debug`).

---

## 📦 Quickstart

```go
package main

import (
	"context"
	"fmt"
	"time"

	"go-supervisor"
)

func main() {
	sup := supervisor.New(
		supervisor.WithStrategy(supervisor.StrategyOneForOne),
		supervisor.WithCrashLoopProtection(3, 10*time.Second),
		supervisor.WithOnPanic(func(name string, panicVal any, stack []byte) {
			fmt.Printf("Worker %s panicked: %v\n", name, panicVal)
		}),
	)

	// Supervise background poller
	sup.Supervise("metrics_poller", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(1 * time.Second):
				// Even if this panics, supervisor catches it and restarts it!
				fmt.Println("Polling metrics...")
			}
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = sup.Start(ctx)
}
```
