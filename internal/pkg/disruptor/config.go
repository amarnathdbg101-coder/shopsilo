package disruptor

import (
	"runtime"
	"sync"
	"time"
)

// WaitStrategy defines how a consumer waits for published sequences.
type WaitStrategy int

const (
	// WaitStrategyYield calls runtime.Gosched() when waiting (balanced low-latency and CPU friendly).
	WaitStrategyYield WaitStrategy = iota
	// WaitStrategyBusySpin busy-waits in a tight loop for absolute minimal sub-microsecond latency.
	WaitStrategyBusySpin
	// WaitStrategySleeping yields initially, then sleeps for 100 microseconds.
	WaitStrategySleeping
	// WaitStrategyBlocking uses a condition variable to sleep when idle.
	WaitStrategyBlocking
)

// Config defines settings for the Disruptor ring buffer.
type Config struct {
	Capacity     int          // Must be a power of two (e.g. 1024, 4096, 65536)
	WaitStrategy WaitStrategy // Consumer wait strategy
}

// Option configures functional options.
type Option func(*Config)

// DefaultConfig returns recommended production defaults.
func DefaultConfig(capacity int) Config {
	return Config{
		Capacity:     capacity,
		WaitStrategy: WaitStrategyYield,
	}
}

// WithWaitStrategy sets the consumer wait strategy.
func WithWaitStrategy(ws WaitStrategy) Option {
	return func(c *Config) {
		c.WaitStrategy = ws
	}
}

type waiter struct {
	strategy WaitStrategy
	mu       sync.Mutex
	cond     *sync.Cond
}

func newWaiter(strategy WaitStrategy) *waiter {
	w := &waiter{strategy: strategy}
	if strategy == WaitStrategyBlocking {
		w.cond = sync.NewCond(&w.mu)
	}
	return w
}

func (w *waiter) wait(counter int) int {
	switch w.strategy {
	case WaitStrategyBusySpin:
		// No-op, tight spin loop
		return counter + 1
	case WaitStrategyYield:
		runtime.Gosched()
		return counter + 1
	case WaitStrategySleeping:
		if counter < 100 {
			runtime.Gosched()
		} else {
			time.Sleep(100 * time.Microsecond)
		}
		return counter + 1
	case WaitStrategyBlocking:
		w.mu.Lock()
		w.cond.Wait()
		w.mu.Unlock()
		return 0
	default:
		runtime.Gosched()
		return counter + 1
	}
}

func (w *waiter) signal() {
	if w.strategy == WaitStrategyBlocking && w.cond != nil {
		w.cond.Broadcast()
	}
}
