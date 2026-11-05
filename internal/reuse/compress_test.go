package reuse

import (
	"bytes"
	"encoding/base64"
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

func TestCompressBase64Image(t *testing.T) {
	// Create an 800x600 test image (exceeds 500px)
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			img.Set(x, y, color.RGBA{R: 120, G: 180, B: 240, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("failed to encode test jpeg: %v", err)
	}

	rawBase64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	// Resize down to 400px
	compressedBase64, mimeType, err := CompressBase64Image(rawBase64, 400, 75)
	if err != nil {
		t.Fatalf("CompressBase64Image error: %v", err)
	}
	if mimeType != "image/jpeg" {
		t.Errorf("expected mimeType image/jpeg, got %s", mimeType)
	}
	if len(compressedBase64) == 0 {
		t.Fatalf("expected non-empty compressed base64")
	}

	// Verify resized dimensions
	decBytes, err := base64.StdEncoding.DecodeString(compressedBase64)
	if err != nil {
		t.Fatalf("failed to decode resulting base64: %v", err)
	}
	decImg, _, err := image.Decode(bytes.NewReader(decBytes))
	if err != nil {
		t.Fatalf("failed to decode resized image: %v", err)
	}
	decBounds := decImg.Bounds()
	if decBounds.Dx() > 400 || decBounds.Dy() > 400 {
		t.Errorf("expected resized image to be <= 400px, got %dx%d", decBounds.Dx(), decBounds.Dy())
	}

	// Test fallback on invalid string
	fallback, _, err := CompressBase64Image("invalid_base64_not_image", 400, 80)
	if err != nil {
		t.Fatalf("expected no error on fallback: %v", err)
	}
	if fallback != "invalid_base64_not_image" {
		t.Errorf("expected fallback to return original string")
	}
}
