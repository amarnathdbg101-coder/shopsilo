# go-disruptor

`go-disruptor` provides a high-throughput, ultra-low-latency inter-thread messaging circular ring buffer in pure Go, inspired by the LMAX Disruptor pattern and Linux kernel `kfifo`. It eliminates standard Go channel lock contention and memory allocation overhead.

## Key Features

- **Cache-Line Padding (64-byte isolation)**: Producer and consumer sequence cursors are explicitly padded with 56 bytes (`[7]uint64`) to completely eliminate CPU false sharing (L1/L2 cache line bouncing).
- **Power-of-Two Ring Buffer**: Index wrapping uses hardware bitwise masking (`seq & (Cap - 1)`), avoiding costly integer division.
- **Pluggable Wait Strategies**:
  - `WaitStrategyYield`: Calls `runtime.Gosched()` for balanced CPU efficiency.
  - `WaitStrategyBusySpin`: Tight loop for sub-microsecond algorithmic trading.
  - `WaitStrategySleeping`: Microsecond sleep for energy-saving backpressure.
  - `WaitStrategyBlocking`: `sync.Cond` condition variable signaling.
- **Batch Processing**: Reserve batches of sequence numbers for bulk publishing and draining.
- **Zero Allocations**: Steady-state publish and consume operations produce 0 heap allocations and 0 GC pressure.
- **Zero Dependencies**: Pure Go standard library.

## Quick Start

```go
package main

import (
	"fmt"
	"go-disruptor"
)

func main() {
	// 4096-slot high-speed event ring buffer
	rb, err := disruptor.New[string](4096, disruptor.WithWaitStrategy(disruptor.WaitStrategyYield))
	if err != nil {
		panic(err)
	}
	defer rb.Close()

	// Publish events
	_ = rb.Publish("order_created_101")
	_ = rb.Publish("order_created_102")

	// Batch drain
	events := rb.Drain(10)
	for _, e := range events {
		fmt.Printf("Processed event: %s\n", e)
	}
}
```
