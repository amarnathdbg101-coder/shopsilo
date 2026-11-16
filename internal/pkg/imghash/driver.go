package imghash

import (
	"image"
	"io"
)

// Hasher defines algorithm interfaces for computing perceptual image fingerprints.
type Hasher interface {
	// ComputeFromImage calculates a 64-bit perceptual hash from an in-memory image.
	ComputeFromImage(img image.Image) (uint64, error)

	// ComputeFromReader decodes and hashes an image from an io.Reader stream.
	ComputeFromReader(r io.Reader) (uint64, error)
}
