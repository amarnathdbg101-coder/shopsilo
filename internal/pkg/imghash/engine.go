package imghash

import (
	"fmt"
	"image"
	"io"
)

// Engine provides image similarity comparisons and fingerprinting.
type Engine struct {
	hasher Hasher
	cfg    *Config
}

// NewEngine creates an imghash Engine with options.
func NewEngine(opts ...Option) *Engine {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &Engine{
		hasher: NewDHash(),
		cfg:    cfg,
	}
}

// HashImage computes the 16-character hexadecimal fingerprint for an image.
func (e *Engine) HashImage(img image.Image) (string, error) {
	val, err := e.hasher.ComputeFromImage(img)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%016x", val), nil
}

// HashReader reads, decodes, and fingerprints an image stream.
func (e *Engine) HashReader(r io.Reader) (string, error) {
	val, err := e.hasher.ComputeFromReader(r)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%016x", val), nil
}

// IsSimilar checks if two hex fingerprints are within the default similarity threshold.
func (e *Engine) IsSimilar(hex1, hex2 string) (bool, int, error) {
	dist, err := HammingDistanceHex(hex1, hex2)
	if err != nil {
		return false, -1, err
	}
	return dist <= e.cfg.DefaultThreshold, dist, nil
}
