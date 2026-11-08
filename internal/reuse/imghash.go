// Package reuse provides reusable utility functions.
package reuse

import (
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/amarnathdbg101-coder/go/go-imghash"
)

const (
	// MaxAllowedHammingDistance defines threshold for perceptual image similarity (0-64).
	// Distance <= 5 indicates the same image with minor edits, compression, or resizing.
	MaxAllowedHammingDistance = 5
)

// ComputeDHash calculates a 64-bit Difference Hash (dHash) from an image stream using go-imghash.
func ComputeDHash(r io.Reader) (string, error) {
	return imghash.Compute(r)
}

// ComputeRawSHA256 returns standard SHA256 hex digest for exact binary tracking.
func ComputeRawSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// HammingDistance calculates the bitwise difference between two 64-bit dHash strings.
func HammingDistance(hash1, hash2 string) (int, error) {
	return imghash.Distance(hash1, hash2)
}

// IsImageHashSimilar checks whether two hashes are within the threshold distance.
func IsImageHashSimilar(hash1, hash2 string, maxDistance int) bool {
	return imghash.Similar(hash1, hash2, maxDistance)
}
