package tinylfu

// Metrics contains operational cache telemetry.
type Metrics struct {
	Hits      uint64  `json:"hits"`
	Misses    uint64  `json:"misses"`
	Evictions uint64  `json:"evictions"`
	HitRatio  float64 `json:"hit_ratio"`
	Size      int     `json:"size"`
	Capacity  int     `json:"capacity"`
}

// Config defines the sizing and segmentation ratios for Window TinyLFU.
type Config struct {
	// Capacity is the total maximum entries allowed across window and main segments.
	Capacity int

	// WindowPercent is the fraction allocated to the admission window (LRU). Default is 0.01 (1%).
	WindowPercent float64

	// ProtectedPercent is the fraction of Main cache allocated to the protected segment. Default is 0.80 (80%).
	ProtectedPercent float64
}

// Option is a functional option for configuring the cache.
type Option func(*Config)

// DefaultConfig returns production defaults for Window TinyLFU.
func DefaultConfig(capacity int) Config {
	if capacity < 10 {
		capacity = 10
	}
	return Config{
		Capacity:         capacity,
		WindowPercent:    0.01,
		ProtectedPercent: 0.80,
	}
}

// WithWindowPercent sets the admission window capacity ratio.
func WithWindowPercent(pct float64) Option {
	return func(c *Config) {
		if pct > 0 && pct < 0.5 {
			c.WindowPercent = pct
		}
	}
}

// WithProtectedPercent sets the protected segment ratio of the main cache.
func WithProtectedPercent(pct float64) Option {
	return func(c *Config) {
		if pct > 0 && pct < 1.0 {
			c.ProtectedPercent = pct
		}
	}
}
