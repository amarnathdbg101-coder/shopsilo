# go-workerpool 👷

High-throughput, bounded goroutine worker pool with non-blocking backpressure, synchronous `SubmitWait`, panic isolation, and graceful shutdown. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- 🛑 **OOM Shield (Bounded Concurrency)**: Naked `go func()` ko rokk kar fixed workers me jobs execute karta hai, saving RAM under high traffic spikes.
- ⚡ **Non-Blocking Backpressure (`TrySubmit`)**: Agar queue full ho, to caller ko wait karaye bina `ErrQueueFull` deta hai (ideal for shedding load in 429 scenarios).
- ⏱️ **Synchronous Completion (`SubmitWait`)**: Job execute karke caller context timeout ke saath result wait karta hai.
- 🛡️ **Panic Immunity**: Job ke andar aayi panic worker goroutine ko kill nahi karti.
- 📦 **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. High-Concurrency Batch Processing
```go
package main

import (
	"context"
	"fmt"
	goworkerpool "go-workerpool"
)

func main() {
	// 8 worker threads, buffer queue size 1000
	pool := goworkerpool.New(8, 1000)
	defer pool.Stop()

	for i := 0; i < 50; i++ {
		jobID := i
		_ = pool.Submit(func(ctx context.Context) {
			fmt.Printf("Processing image resize for job #%d\n", jobID)
		})
	}
}
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
