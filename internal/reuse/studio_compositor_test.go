package reuse

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestDetermineStudioTheme(t *testing.T) {
	tests := []struct {
		category string
		subcat   string
		expected StudioTheme
	}{
		{"Personal Care & Beauty", "Matte Lipstick", StudioThemeCosmetics},
		{"Cosmetics", "Perfume Bottle", StudioThemeCosmetics},
		{"Beauty", "Face Serum", StudioThemeCosmetics},
		{"Clothing & Fashion", "Ethnic Wear & Sarees", StudioThemeEthnicFashion},
		{"Fashion", "Embroidered Kurti", StudioThemeEthnicFashion},
		{"Apparel", "Cotton Shirt", StudioThemeEthnicFashion},
		{"Grocery & Staples", "Atta & Flours", StudioThemePackagedRetail},
		{"General Store", "Biscuits", StudioThemePackagedRetail},
	}

	for _, tt := range tests {
		got := DetermineStudioTheme(tt.category, tt.subcat)
		if got != tt.expected {
			t.Errorf("DetermineStudioTheme(%q, %q) = %v; want %v", tt.category, tt.subcat, got, tt.expected)
		}
	}
}

func TestRenderProductStudio_Cosmetics(t *testing.T) {
	// Create a 200x400 transparent PNG with an opaque red bottle in the center
	img := image.NewRGBA(image.Rect(0, 0, 200, 400))
	for y := 50; y < 350; y++ {
		for x := 60; x < 140; x++ {
			img.Set(x, y, color.RGBA{R: 220, G: 20, B: 60, A: 255}) // Crimson lipstick/bottle
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}

	renderedBytes, err := RenderProductStudio(buf.Bytes(), StudioCompositorOptions{
		Theme:      StudioThemeCosmetics,
		CanvasSize: 600,
	})
	if err != nil {
		t.Fatalf("RenderProductStudio failed: %v", err)
	}

	if len(renderedBytes) == 0 {
		t.Fatal("RenderProductStudio produced empty output")
	}

	// Verify output is valid JPEG image
	decoded, format, err := image.Decode(bytes.NewReader(renderedBytes))
	if err != nil {
		t.Fatalf("failed to decode rendered studio image: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("expected format jpeg, got %s", format)
	}
	if decoded.Bounds().Dx() != 600 || decoded.Bounds().Dy() != 600 {
		t.Errorf("expected canvas 600x600, got %dx%d", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func TestRenderProductStudio_EthnicFashion(t *testing.T) {
	// Create a 300x500 transparent PNG with an opaque royal blue saree/dress
	img := image.NewRGBA(image.Rect(0, 0, 300, 500))
	for y := 40; y < 460; y++ {
		for x := 50; x < 250; x++ {
			img.Set(x, y, color.RGBA{R: 25, G: 25, B: 112, A: 255}) // Midnight blue dress
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}

	renderedBytes, err := RenderProductStudio(buf.Bytes(), StudioCompositorOptions{
		Theme:      StudioThemeEthnicFashion,
		CanvasSize: 800,
	})
	if err != nil {
		t.Fatalf("RenderProductStudio failed: %v", err)
	}

	decoded, _, err := image.Decode(bytes.NewReader(renderedBytes))
	if err != nil {
		t.Fatalf("failed to decode rendered studio image: %v", err)
	}
	if decoded.Bounds().Dx() != 800 || decoded.Bounds().Dy() != 800 {
		t.Errorf("expected canvas 800x800, got %dx%d", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func TestEnhanceProductLightingAndFeathering(t *testing.T) {
	// Test on a warm-lit, slightly dim rectangular object
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 20; y < 80; y++ {
		for x := 20; x < 80; x++ {
			// Dim warm yellow/brown object
			img.Set(x, y, color.RGBA{R: 160, G: 120, B: 50, A: 255})
		}
	}
	// Add a semi-transparent border to test feathering
	for x := 19; x <= 80; x++ {
		img.Set(x, 19, color.RGBA{R: 160, G: 120, B: 50, A: 120})
		img.Set(x, 80, color.RGBA{R: 160, G: 120, B: 50, A: 120})
	}

	// Must run without panic and improve pixel quality
	enhanceProductLighting(img)
	featherProductEdges(img)

	// Check center pixel is still opaque
	centerA := img.RGBAAt(50, 50).A
	if centerA != 255 {
		t.Errorf("expected center alpha 255, got %d", centerA)
	}

	// Check border pixel was smoothed
	borderA := img.RGBAAt(20, 19).A
	if borderA == 0 {
		t.Errorf("expected non-zero smoothed border alpha, got 0")
	}
}
