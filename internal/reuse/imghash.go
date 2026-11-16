// Package reuse provides reusable utility functions.
package reuse

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math/bits"
	"strconv"
)

const (
	// MaxAllowedHammingDistance defines threshold for perceptual image similarity (0-64).
	// Distance <= 5 indicates the same image with minor edits, compression, or resizing.
	MaxAllowedHammingDistance = 5
)

// ComputeDHash calculates a 64-bit Difference Hash (dHash) from an image stream.
// If the image format cannot be decoded by image.Decode (e.g., custom format),
// it falls back to a deterministic SHA256 hex string.
func ComputeDHash(r io.Reader) (string, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return "", err
	}

	// 1. Downscale to 9x8 grayscale
	// 9 columns, 8 rows = 72 pixels
	const targetWidth = 9
	const targetHeight = 8

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width == 0 || height == 0 {
		return "", fmt.Errorf("invalid image dimensions: %dx%d", width, height)
	}

	var grayMatrix [targetHeight][targetWidth]uint8

	for y := 0; y < targetHeight; y++ {
		srcY := bounds.Min.Y + (y * height / targetHeight)
		for x := 0; x < targetWidth; x++ {
			srcX := bounds.Min.X + (x * width / targetWidth)
			c := img.At(srcX, srcY)
			gray := color.GrayModel.Convert(c).(color.Gray)
			grayMatrix[y][x] = gray.Y
		}
	}

	// 2. Compute 64-bit difference hash
	// For each row (8 rows), compare 8 adjacent column pairs (col 0 vs 1, 1 vs 2, ..., 7 vs 8)
	var hash uint64
	bitIndex := 0

	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth-1; x++ {
			if grayMatrix[y][x] > grayMatrix[y][x+1] {
				hash |= 1 << (63 - bitIndex)
			}
			bitIndex++
		}
	}

	return fmt.Sprintf("%016x", hash), nil
}

// ComputeRawSHA256 returns standard SHA256 hex digest for exact binary tracking.
func ComputeRawSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// HammingDistance calculates the bitwise difference between two 64-bit dHash strings.
func HammingDistance(hash1, hash2 string) (int, error) {
	val1, err := strconv.ParseUint(hash1, 16, 64)
	if err != nil {
		return -1, fmt.Errorf("invalid hash1: %w", err)
	}
	val2, err := strconv.ParseUint(hash2, 16, 64)
	if err != nil {
		return -1, fmt.Errorf("invalid hash2: %w", err)
	}

	xor := val1 ^ val2
	return bits.OnesCount64(xor), nil
}

// IsImageHashSimilar checks whether two hashes are within the threshold distance.
func IsImageHashSimilar(hash1, hash2 string, maxDistance int) bool {
	if hash1 == hash2 {
		return true
	}
	dist, err := HammingDistance(hash1, hash2)
	if err != nil {
		return false
	}
	return dist <= maxDistance
}
