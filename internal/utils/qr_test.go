package utils_test

import (
	"bytes"
	"image/png"
	"shopMe/internal/utils"
	"testing"
)

func TestGenerateQRCodePNG(t *testing.T) {
	pngBytes, err := utils.GenerateQRCodePNG("https://shopme.app/shops/delhi-electronics", 256)
	if err != nil {
		t.Fatalf("unexpected error generating QR code: %v", err)
	}

	if len(pngBytes) == 0 {
		t.Fatalf("expected non-empty png bytes")
	}

	// Verify valid PNG image format
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("failed to decode PNG image: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 256 || bounds.Dy() != 256 {
		t.Fatalf("expected 256x256 image, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}
