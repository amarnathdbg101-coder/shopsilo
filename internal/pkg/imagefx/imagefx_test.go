package goimagefx_test

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	goimagefx "shopMe/internal/pkg/imagefx"
)

func TestAdaptiveBoundaryFloodCutout(t *testing.T) {
	// 60x60 white canvas with 20x20 red square in the center
	src := image.NewRGBA(image.Rect(0, 0, 60, 60))
	draw.Draw(src, src.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	// Draw red square (20,20) to (40,40)
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	redRect := image.Rect(20, 20, 40, 40)
	draw.Draw(src, redRect, &image.Uniform{C: red}, image.Point{}, draw.Src)

	cutout, ok := goimagefx.AdaptiveBoundaryFloodCutout(src, 30.0)
	if !ok {
		t.Fatal("expected cutout to succeed")
	}

	// Corner pixel (0,0) must now be transparent (Alpha = 0)
	cornerAlpha := cutout.At(0, 0).(color.RGBA).A
	if cornerAlpha != 0 {
		t.Fatalf("expected corner pixel to be transparent, got alpha=%d", cornerAlpha)
	}

	// Center pixel (30,30) must still be red (R=255, A=255)
	center := cutout.At(30, 30).(color.RGBA)
	if center.R < 250 || center.A != 255 {
		t.Fatalf("expected center pixel to remain red, got %v", center)
	}
}

func TestTrimTransparency(t *testing.T) {
	// 100x100 transparent image with visible 30x30 block from (20,20) to (50,50)
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	draw.Draw(src, image.Rect(20, 20, 50, 50), &image.Uniform{C: blue}, image.Point{}, draw.Src)

	trimmed, cropRect := goimagefx.TrimTransparency(src, 128)

	if cropRect.Dx() != 30 || cropRect.Dy() != 30 {
		t.Fatalf("expected 30x30 cropped rectangle, got %dx%d", cropRect.Dx(), cropRect.Dy())
	}
	if trimmed.Bounds().Dx() != 30 || trimmed.Bounds().Dy() != 30 {
		t.Fatalf("expected trimmed image to be 30x30, got %dx%d", trimmed.Bounds().Dx(), trimmed.Bounds().Dy())
	}
}

func TestFitAndCenter(t *testing.T) {
	// 100x50 image
	src := image.NewRGBA(image.Rect(0, 0, 100, 50))
	draw.Draw(src, src.Bounds(), &image.Uniform{C: color.Black}, image.Point{}, draw.Src)

	// Fit on 400x400 canvas with white background
	canvas := goimagefx.FitAndCenter(src, 400, 400, 0.10, color.White)

	if canvas.Bounds().Dx() != 400 || canvas.Bounds().Dy() != 400 {
		t.Fatalf("expected 400x400 canvas, got %dx%d", canvas.Bounds().Dx(), canvas.Bounds().Dy())
	}

	// Corner (0,0) must be white
	c0 := canvas.At(0, 0).(color.RGBA)
	if c0.R != 255 || c0.G != 255 || c0.B != 255 {
		t.Fatalf("expected white corner, got %v", c0)
	}

	// Center (200,200) must be black
	mid := canvas.At(200, 200).(color.RGBA)
	if mid.R != 0 || mid.G != 0 || mid.B != 0 {
		t.Fatalf("expected black center, got %v", mid)
	}
}

func TestApplyDropShadow(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 40, 40))
	draw.Draw(src, src.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 200, B: 0, A: 255}}, image.Point{}, draw.Src)

	withShadow := goimagefx.ApplyDropShadow(src, 6, 4, 6, 120)
	if withShadow.Bounds().Dx() <= 40 || withShadow.Bounds().Dy() <= 40 {
		t.Fatalf("expected expanded canvas for drop shadow, got %dx%d", withShadow.Bounds().Dx(), withShadow.Bounds().Dy())
	}
}

func TestResizeBilinear(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 100))
	draw.Draw(src, src.Bounds(), &image.Uniform{C: color.RGBA{R: 50, G: 100, B: 150, A: 255}}, image.Point{}, draw.Src)

	resized := goimagefx.ResizeBilinear(src, 50, 25)
	if resized.Bounds().Dx() != 50 || resized.Bounds().Dy() != 25 {
		t.Fatalf("expected 50x25 resized image, got %dx%d", resized.Bounds().Dx(), resized.Bounds().Dy())
	}

	c := resized.At(25, 12).(color.RGBA)
	if c.R != 50 || c.G != 100 || c.B != 150 {
		t.Fatalf("expected color preservation, got %v", c)
	}
}
