package auditlog

import "time"

// Config holds runtime options for the asynchronous audit logging engine.
type Config struct {
	// BatchSize triggers an immediate database flush when buffered entries reach this count.
	// Default: 50.
	BatchSize int

	// FlushInterval forces a flush of pending entries periodically even if BatchSize is not reached.
	// Default: 1 second.
	FlushInterval time.Duration

	// BufferSize defines the capacity of the non-blocking channel queue.
	// Default: 2000.
	BufferSize int

	// SensitiveKeys customizes which object fields are automatically masked with "********".
	SensitiveKeys []string
}

// Option customizes Config settings.
type Option func(*Config)

// DefaultConfig provides high-throughput production defaults.
func DefaultConfig() Config {
	return Config{
		BatchSize:     50,
		FlushInterval: 1 * time.Second,
		BufferSize:    2000,
		SensitiveKeys: DefaultSensitiveKeys,
	}
}

// WithBatchSize sets maximum items accumulated before auto-flushing.
func WithBatchSize(size int) Option {
	return func(c *Config) {
		if size > 0 {
			c.BatchSize = size
		}
	}
}

// WithFlushInterval sets ticker duration for periodic batch flushing.
func WithFlushInterval(d time.Duration) Option {
	return func(c *Config) {
		if d > 0 {
			c.FlushInterval = d
		}
	}
}

// WithBufferSize sets channel buffer capacity.
func WithBufferSize(size int) Option {
	return func(c *Config) {
		if size > 0 {
			c.BufferSize = size
		}
	}
}

// WithSensitiveKeys appends custom keys to the automatic masking filter.
func WithSensitiveKeys(keys ...string) Option {
	return func(c *Config) {
		c.SensitiveKeys = append(c.SensitiveKeys, keys...)
	}
}
