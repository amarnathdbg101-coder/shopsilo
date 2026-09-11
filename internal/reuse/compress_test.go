package reuse

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestCompressImage_JPEG(t *testing.T) {
	// Create a 200x200 test image
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 100, A: 255})
		}
	}

	var rawBuf bytes.Buffer
	err := jpeg.Encode(&rawBuf, img, &jpeg.Options{Quality: 100})
	if err != nil {
		t.Fatalf("failed to encode raw test image: %v", err)
	}

	compressed, cType, err := CompressImage(bytes.NewReader(rawBuf.Bytes()), "image/jpeg")
	if err != nil {
		t.Fatalf("CompressImage failed: %v", err)
	}

	if cType != "image/jpeg" {
		t.Errorf("expected contentType image/jpeg, got %s", cType)
	}

	if len(compressed) >= rawBuf.Len() {
		t.Errorf("expected compressed size (%d) to be smaller than raw 100%% quality size (%d)", len(compressed), rawBuf.Len())
	}
}

func TestCompressImage_PNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}

	var rawBuf bytes.Buffer
	if err := png.Encode(&rawBuf, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}

	compressed, _, err := CompressImage(bytes.NewReader(rawBuf.Bytes()), "image/png")
	if err != nil {
		t.Fatalf("CompressImage failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Errorf("compressed output is empty")
	}
}

func TestCompressImage_UnsupportedFallback(t *testing.T) {
	unsupportedData := []byte("plain text not an image")
	compressed, cType, err := CompressImage(bytes.NewReader(unsupportedData), "text/plain")
	if err != nil {
		t.Fatalf("expected no error on fallback, got: %v", err)
	}
	if !bytes.Equal(compressed, unsupportedData) {
		t.Errorf("expected original data returned on fallback")
	}
	if cType != "text/plain" {
		t.Errorf("expected original content-type returned")
	}
}
