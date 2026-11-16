package adaptivelimit

import "time"

// Metrics reports real-time telemetry from the adaptive concurrency limiter.
type Metrics struct {
	CurrentLimit float64       `json:"current_limit"`
	InFlight     int           `json:"in_flight"`
	MinRTT       time.Duration `json:"min_rtt"`
	CurrentRTT   time.Duration `json:"current_rtt"`
	TotalAcquired uint64       `json:"total_acquired"`
	TotalRejected uint64       `json:"total_rejected"`
}

// Config specifies the mathematical tuning parameters for the TCP Vegas adaptive limiter.
type Config struct {
	// InitialLimit is the starting concurrency boundary. Default is 10.
	InitialLimit float64

	// MinLimit is the absolute minimum concurrency floor. Default is 2.
	MinLimit float64

	// MaxLimit is the maximum concurrency ceiling. Default is 200.
	MaxLimit float64

	// Alpha is the queue size lower threshold. Below this, capacity increases. Default is 3.0.
	Alpha float64

	// Beta is the queue size upper threshold. Above this, capacity decreases. Default is 6.0.
	Beta float64

	// BackoffRatio is the multiplicative reduction factor during congestion. Default is 0.8 (20% reduction).
	BackoffRatio float64

	// Smoothing factor for exponential moving average of RTT (0.0 to 1.0). Default is 0.2.
	Smoothing float64
}

// Option is a functional option for configuring the limiter.
type Option func(*Config)

// DefaultConfig returns optimal production parameters for the TCP Vegas adaptive limiter.
func DefaultConfig() Config {
	return Config{
		InitialLimit: 10.0,
		MinLimit:     2.0,
		MaxLimit:     200.0,
		Alpha:        3.0,
		Beta:         6.0,
		BackoffRatio: 0.8,
		Smoothing:    0.2,
	}
}

// WithInitialLimit sets the initial concurrency limit.
func WithInitialLimit(limit float64) Option {
	return func(c *Config) {
		if limit > 0 {
			c.InitialLimit = limit
		}
	}
}

// WithLimits sets minimum and maximum concurrency bounds.
func WithLimits(min, max float64) Option {
	return func(c *Config) {
		if min > 0 && max >= min {
			c.MinLimit = min
			c.MaxLimit = max
		}
	}
}

// WithThresholds sets Alpha and Beta queue thresholds.
func WithThresholds(alpha, beta float64) Option {
	return func(c *Config) {
		if alpha > 0 && beta > alpha {
			c.Alpha = alpha
			c.Beta = beta
		}
	}
}

// WithBackoffRatio sets multiplicative backoff ratio (e.g. 0.85).
func WithBackoffRatio(ratio float64) Option {
	return func(c *Config) {
		if ratio > 0 && ratio < 1.0 {
			c.BackoffRatio = ratio
		}
	}
}
