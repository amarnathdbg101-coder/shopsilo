package imghash

// Option configures image hashing parameters.
type Option func(*Config)

// Config defines perceptual matching rules.
type Config struct {
	// DefaultThreshold defines maximum Hamming distance to consider two images identical (Default: 5).
	DefaultThreshold int
}

// DefaultConfig provides recommended production settings.
func DefaultConfig() *Config {
	return &Config{
		DefaultThreshold: 5,
	}
}

// WithThreshold sets the default maximum Hamming distance threshold.
func WithThreshold(threshold int) Option {
	return func(c *Config) {
		if threshold >= 0 && threshold <= 64 {
			c.DefaultThreshold = threshold
		}
	}
}
