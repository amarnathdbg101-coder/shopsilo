package studiocomp_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"shopMe/internal/pkg/studiocomp"
)

// createMockCutout generates an in-memory PNG cutout image with transparent padding.
func createMockCutout(width, height int, col color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill a central rectangle with the color, leaving 20% borders transparent
	padX := width / 5
	padY := height / 5
	for y := padY; y < height-padY; y++ {
		for x := padX; x < width-padX; x++ {
			img.SetRGBA(x, y, col)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestDetermineStudioTheme(t *testing.T) {
	tests := []struct {
		category string
		subcat   string
		expected studiocomp.Theme
	}{
		{"Personal Care & Beauty", "Matte Lipstick", studiocomp.ThemeCosmetics},
		{"Cosmetics", "Perfume Bottle", studiocomp.ThemeCosmetics},
		{"Beauty", "Face Serum", studiocomp.ThemeCosmetics},
		{"Clothing & Fashion", "Ethnic Wear & Sarees", studiocomp.ThemeEthnicFashion},
		{"Fashion", "Embroidered Kurti", studiocomp.ThemeEthnicFashion},
		{"Apparel", "Cotton Shirt", studiocomp.ThemeEthnicFashion},
		{"Grocery & Staples", "Atta & Flours", studiocomp.ThemePackagedRetail},
		{"General Store", "Biscuits", studiocomp.ThemePackagedRetail},
		{"", "", studiocomp.ThemePackagedRetail},
	}

	for _, tt := range tests {
		got := studiocomp.DetermineStudioTheme(tt.category, tt.subcat)
		if got != tt.expected {
			t.Errorf("DetermineStudioTheme(%q, %q) = %v; want %v", tt.category, tt.subcat, got, tt.expected)
		}
	}
}

func TestRenderProductStudio_Themes(t *testing.T) {
	cutout := createMockCutout(200, 300, color.RGBA{R: 220, G: 50, B: 50, A: 255})

	themes := []studiocomp.Theme{
		studiocomp.ThemeCosmetics,
		studiocomp.ThemeEthnicFashion,
		studiocomp.ThemePackagedRetail,
		studiocomp.ThemeMinimalistDark,
	}

	for _, th := range themes {
		t.Run(string(th), func(t *testing.T) {
			rendered, err := studiocomp.RenderProductStudio(cutout, studiocomp.Options{
				Theme:       th,
				CanvasSize:  512,
				JpegQuality: 85,
			})
			if err != nil {
				t.Fatalf("RenderProductStudio failed for theme %s: %v", th, err)
			}
			if len(rendered) == 0 {
				t.Fatalf("RenderProductStudio produced empty output for theme %s", th)
			}
		})
	}
}

func TestRenderProductStudio_EmptyCutout(t *testing.T) {
	_, err := studiocomp.RenderProductStudio(nil, studiocomp.Options{})
	if err == nil {
		t.Fatal("expected error for empty cutout bytes")
	}
}

func TestFindOpaqueBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 20; y < 80; y++ {
		for x := 30; x < 70; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	minX, minY, maxX, maxY, found := studiocomp.FindOpaqueBounds(img)
	if !found {
		t.Fatal("expected to find opaque bounds")
	}
	if minX != 30 || minY != 20 || maxX != 70 || maxY != 80 {
		t.Errorf("bounds = (%d,%d)-(%d,%d); want (30,20)-(70,80)", minX, minY, maxX, maxY)
	}
}

func TestBilinearScale(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			src.SetRGBA(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, 100, 100))
	studiocomp.BilinearScale(dst, src)

	c := dst.RGBAAt(50, 50)
	if c.R < 95 || c.R > 105 {
		t.Errorf("expected scaled pixel red near 100, got %d", c.R)
	}
}

func BenchmarkRenderProductStudio(b *testing.B) {
	cutout := createMockCutout(300, 400, color.RGBA{R: 180, G: 40, B: 40, A: 255})
	opts := studiocomp.Options{
		Theme:       studiocomp.ThemePackagedRetail,
		CanvasSize:  512,
		JpegQuality: 85,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = studiocomp.RenderProductStudio(cutout, opts)
	}
}

func BenchmarkBilinearScale(b *testing.B) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 200))
	dst := image.NewRGBA(image.Rect(0, 0, 400, 400))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		studiocomp.BilinearScale(dst, src)
	}
}
