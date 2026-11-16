# go-deadlock 🔒

> **Drop-in replacements for `sync.Mutex` and `sync.RWMutex` with real-time lock-order inversion detection and lock starvation alerts (Go 1.23+).**  
> Identifies potential deadlocks (AB-BA cycles) before they hang production services and prints caller stack traces without any CGO.

---

## 🚀 Key Features

1. **Lock-Order Reversal Detection**:
   - Maintains a directed graph of all acquired locks.
   - Detects AB-BA cycles via Depth-First Search (DFS) the moment a dangerous acquisition path is taken.
2. **Lock Starvation & Timeout Alerts**:
   - Configurable timeout (default 2s). Emits a warning if a lock is held or waited upon longer than threshold with the waiting stack trace.
3. **Drop-in `sync.Mutex` / `sync.RWMutex` Interface**:
   - Identical API: `Lock()`, `Unlock()`, `TryLock()`, `RLock()`, `RUnlock()`.
4. **Zero Dependencies**:
   - 100% pure Go standard library (`sync`, `runtime`, `time`). Zero CGO.
5. **Configurable Handlers & Bypass**:
   - Plug custom alert logging, metric counters, or disable tracking in performance-critical production paths with `deadlock.SetConfig(deadlock.Config{Disabled: true})`.

---

## 📦 Quickstart

```go
package main

import (
	"fmt"
	"go-deadlock"
)

type Account struct {
	mu      deadlock.Mutex
	balance int
}

func Transfer(from, to *Account, amount int) {
	// If one goroutine transfers A -> B and another transfers B -> A,
	// go-deadlock immediately detects the cycle and prints the exact stack traces!
	from.mu.Lock()
	defer from.mu.Unlock()

	to.mu.Lock()
	defer to.mu.Unlock()

	from.balance -= amount
	to.balance += amount
}

func main() {
	deadlock.SetConfig(deadlock.Config{
		OnPotentialDeadlock: func(msg string) {
			fmt.Println("CRITICAL CONCURRENCY WARNING:\n", msg)
		},
	})
}
```
