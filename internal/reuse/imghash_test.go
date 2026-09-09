package reuse

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func createTestImage(width, height int, col color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, col)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestComputeDHash(t *testing.T) {
	imgBytes := createTestImage(50, 50, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	hash, err := ComputeDHash(bytes.NewReader(imgBytes))
	if err != nil {
		t.Fatalf("ComputeDHash failed: %v", err)
	}

	if len(hash) != 16 {
		t.Fatalf("expected 16 hex characters, got %d (%s)", len(hash), hash)
	}

	// Same image produces identical hash
	hash2, err := ComputeDHash(bytes.NewReader(imgBytes))
	if err != nil {
		t.Fatalf("ComputeDHash second run failed: %v", err)
	}

	if hash != hash2 {
		t.Fatalf("hashes should be identical: %s vs %s", hash, hash2)
	}

	dist, err := HammingDistance(hash, hash2)
	if err != nil || dist != 0 {
		t.Fatalf("expected distance 0, got %d, err %v", dist, err)
	}

	if !IsImageHashSimilar(hash, hash2, MaxAllowedHammingDistance) {
		t.Fatalf("expected IsImageHashSimilar to return true")
	}
}
