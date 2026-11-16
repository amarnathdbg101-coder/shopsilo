package imghash

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sync"
	"testing"
)

// Helper to create test images in-memory
func createGradientImage(width, height int, reverse bool) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var val uint8
			if reverse {
				val = uint8(255 - (x * 255 / width))
			} else {
				val = uint8(x * 255 / width)
			}
			img.Set(x, y, color.RGBA{R: val, G: val, B: val, A: 255})
		}
	}
	return img
}

func encodePNG(img image.Image) *bytes.Buffer {
	buf := new(bytes.Buffer)
	_ = png.Encode(buf, img)
	return buf
}

func TestDHash_IdenticalImages(t *testing.T) {
	img1 := createGradientImage(100, 100, false)
	img2 := createGradientImage(200, 200, false) // Different size, same content gradient

	hash1, err := ComputeImage(img1)
	if err != nil {
		t.Fatalf("unexpected error hashing img1: %v", err)
	}

	hash2, err := ComputeImage(img2)
	if err != nil {
		t.Fatalf("unexpected error hashing img2: %v", err)
	}

	dist, err := Distance(hash1, hash2)
	if err != nil {
		t.Fatalf("unexpected error calculating distance: %v", err)
	}

	if dist != 0 {
		t.Fatalf("expected distance 0 for identical gradient across different dimensions, got %d", dist)
	}
	if !Similar(hash1, hash2) {
		t.Fatal("expected Similar() to be true")
	}
}

func TestDHash_OppositeGradients(t *testing.T) {
	imgNormal := createGradientImage(100, 100, false)
	imgReversed := createGradientImage(100, 100, true)

	hash1, _ := ComputeImage(imgNormal)
	hash2, _ := ComputeImage(imgReversed)

	dist, err := Distance(hash1, hash2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Inverted gradient should have maximum or high distance (near 64)
	if dist < 50 {
		t.Fatalf("expected high hamming distance for opposite gradients, got %d", dist)
	}
	if Similar(hash1, hash2) {
		t.Fatal("expected Similar() to be false for opposite gradients")
	}
}

func TestDHash_ReaderStream(t *testing.T) {
	img := createGradientImage(80, 80, false)
	buf := encodePNG(img)

	hash, err := Compute(buf)
	if err != nil {
		t.Fatalf("unexpected error from reader compute: %v", err)
	}
	if len(hash) != 16 {
		t.Fatalf("expected 16-hex character hash, got length %d: %s", len(hash), hash)
	}
}

func TestDHash_InvalidInputs(t *testing.T) {
	_, err := Compute(bytes.NewReader([]byte("not an image")))
	if err == nil {
		t.Fatal("expected error on invalid image stream")
	}

	_, err = Distance("invalid", "1234567890abcdef")
	if err == nil {
		t.Fatal("expected error on invalid hex hash")
	}
}

func TestDHash_ConcurrentStress(t *testing.T) {
	img := createGradientImage(50, 50, false)

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h, err := ComputeImage(img)
			if err != nil || len(h) != 16 {
				t.Errorf("concurrent hashing error: %v", err)
			}
		}()
	}
	wg.Wait()
}
