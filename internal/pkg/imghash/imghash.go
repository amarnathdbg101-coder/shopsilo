// Package imghash provides pure Go perceptual image hashing (dHash) and similarity checks.
// It detects duplicate or slightly modified catalog images, logos, and photos without external C libraries.
package imghash

import (
	"image"
	"io"
)

var defaultEngine = NewEngine()

// Compute reads and hashes an image stream, returning a 16-character hex dHash string.
func Compute(r io.Reader) (string, error) {
	return defaultEngine.HashReader(r)
}

// ComputeImage hashes an in-memory image.
func ComputeImage(img image.Image) (string, error) {
	return defaultEngine.HashImage(img)
}

// Distance returns the Hamming distance between two hex fingerprints.
func Distance(hex1, hex2 string) (int, error) {
	return HammingDistanceHex(hex1, hex2)
}

// Similar checks if two hex fingerprints represent perceptually similar images.
// Default threshold is 5 (on a scale of 0 to 64).
func Similar(hex1, hex2 string, maxDistance ...int) bool {
	if hex1 == hex2 {
		return true
	}
	threshold := 5
	if len(maxDistance) > 0 && maxDistance[0] >= 0 {
		threshold = maxDistance[0]
	}
	dist, err := Distance(hex1, hex2)
	if err != nil {
		return false
	}
	return dist <= threshold
}
