package bulkhead

import "time"

// Metrics reports real-time operational telemetry of a Bulkhead partition.
type Metrics struct {
	MaxConcurrent    int
	ActiveCount      int
	MaxQueue         int
	QueuedCount      int
	TotalExecuted    uint64
	TotalRejected    uint64
	TotalQueueTimeout uint64
}

// Config specifies the operational bounds of a Bulkhead.
type Config struct {
	// MaxConcurrent is the maximum number of concurrent executions permitted.
	// Default is 10.
	MaxConcurrent int

	// MaxQueue is the maximum number of requests allowed to wait in queue.
	// Default is 20.
	MaxQueue int

	// QueueTimeout is the maximum duration an execution will wait in the queue before being rejected.
	// Default is 1 second.
	QueueTimeout time.Duration

	// OnRejected is an optional callback triggered whenever a task is rejected.
	OnRejected func(reason error)
}

// Option is a functional option for configuring a Bulkhead.
type Option func(*Config)

// DefaultConfig returns resilient production defaults for a Bulkhead.
func DefaultConfig() Config {
	return Config{
		MaxConcurrent: 10,
		MaxQueue:      20,
		QueueTimeout:  1 * time.Second,
	}
}

// WithMaxConcurrent sets the maximum number of concurrent executions.
func WithMaxConcurrent(n int) Option {
	return func(c *Config) {
		if n > 0 {
			c.MaxConcurrent = n
		}
	}
}

// WithMaxQueue sets the maximum queue depth for waiting executions.
func WithMaxQueue(n int) Option {
	return func(c *Config) {
		if n >= 0 {
			c.MaxQueue = n
		}
	}
}

// WithQueueTimeout sets the maximum queue wait duration.
func WithQueueTimeout(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.QueueTimeout = d
		}
	}
}

// WithOnRejected configures a rejection callback hook.
func WithOnRejected(fn func(reason error)) Option {
	return func(c *Config) {
		c.OnRejected = fn
	}
}
