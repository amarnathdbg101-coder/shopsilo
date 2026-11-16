package bm25

// Config defines the mathematical parameters for Okapi BM25 scoring.
type Config struct {
	// K1 controls non-linear term frequency saturation.
	// Typically between 1.2 and 2.0. Default is 1.5.
	K1 float64

	// B controls the degree of document length normalization penalty.
	// 0.0 means no length normalization, 1.0 means fully normalized. Default is 0.75.
	B float64

	// MinTokenLength drops tokens shorter than this length. Default is 1.
	MinTokenLength int
}

// Option is a functional option for configuring the BM25 index.
type Option func(*Config)

// DefaultConfig returns optimal information retrieval defaults for BM25.
func DefaultConfig() Config {
	return Config{
		K1:             1.5,
		B:              0.75,
		MinTokenLength: 1,
	}
}

// WithK1 sets the term frequency saturation parameter.
func WithK1(k1 float64) Option {
	return func(c *Config) {
		if k1 > 0 {
			c.K1 = k1
		}
	}
}

// WithB sets the document length normalization penalty.
func WithB(b float64) Option {
	return func(c *Config) {
		if b >= 0 && b <= 1.0 {
			c.B = b
		}
	}
}
