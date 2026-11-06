package reuse

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"math"
	"strings"
)

type StudioTheme string

const (
	StudioThemeCosmetics      StudioTheme = "cosmetics"       // Luxury soft pearl/glass background + ground contact reflection
	StudioThemeEthnicFashion  StudioTheme = "ethnic_fashion"  // Warm gallery aesthetic canvas + soft diffused ambient shadow
	StudioThemePackagedRetail StudioTheme = "packaged_retail" // Crisp pure catalog white (#FFFFFF) + tight base contact shadow
)

// DetermineStudioTheme infers the optimal visual studio theme from category and subcategory text.
func DetermineStudioTheme(categoryHint, subcategory string) StudioTheme {
	combined := strings.ToLower(categoryHint + " " + subcategory)
	switch {
	case strings.Contains(combined, "beauty") ||
		strings.Contains(combined, "personal care") ||
		strings.Contains(combined, "cosmetic") ||
		strings.Contains(combined, "lipstick") ||
		strings.Contains(combined, "perfume") ||
		strings.Contains(combined, "cream") ||
		strings.Contains(combined, "shampoo") ||
		strings.Contains(combined, "serum") ||
		strings.Contains(combined, "lotion") ||
		strings.Contains(combined, "makeup"):
		return StudioThemeCosmetics

	case strings.Contains(combined, "clothing") ||
		strings.Contains(combined, "fashion") ||
		strings.Contains(combined, "saree") ||
		strings.Contains(combined, "kurti") ||
		strings.Contains(combined, "ethnic") ||
		strings.Contains(combined, "apparel") ||
		strings.Contains(combined, "lehenga") ||
		strings.Contains(combined, "suit") ||
		strings.Contains(combined, "shirt") ||
		strings.Contains(combined, "jeans") ||
		strings.Contains(combined, "fabric"):
		return StudioThemeEthnicFashion

	default:
		return StudioThemePackagedRetail
	}
}

type StudioCompositorOptions struct {
	Theme        StudioTheme
	CanvasSize   int     // Standard e-commerce square dimension (default 1080)
	PaddingRatio float64 // Proportion of canvas reserved for margin (default 0.10)
}

// RenderProductStudio takes transparent PNG bytes (e.g. from Cloudflare Workers AI)
// and composites a studio-grade 1080x1080 catalog image with category-tailored
// backdrops, soft ambient shadows, and centered visual balance.
func RenderProductStudio(cutoutBytes []byte, opts StudioCompositorOptions) ([]byte, error) {
	if len(cutoutBytes) == 0 {
		return nil, fmt.Errorf("cutout bytes are empty")
	}

	srcImg, _, err := image.Decode(bytes.NewReader(cutoutBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode cutout image: %w", err)
	}

	canvasSize := opts.CanvasSize
	if canvasSize <= 0 {
		canvasSize = 1080
	}
	paddingRatio := opts.PaddingRatio
	if paddingRatio <= 0 || paddingRatio >= 0.4 {
		paddingRatio = 0.10
	}

	theme := opts.Theme
	if theme == "" {
		theme = StudioThemePackagedRetail
	}

	// 1. Detect tight non-transparent bounding box around product
	minX, minY, maxX, maxY, hasAlpha := findOpaqueBounds(srcImg)
	if !hasAlpha || minX >= maxX || minY >= maxY {
		// Fallback: Use entire image bounds if no transparent alpha was detected
		bounds := srcImg.Bounds()
		minX, minY, maxX, maxY = bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y
	}

	croppedWidth := maxX - minX
	croppedHeight := maxY - minY

	// 2. Crop to product bounds
	cropped := image.NewRGBA(image.Rect(0, 0, croppedWidth, croppedHeight))
	for y := 0; y < croppedHeight; y++ {
		for x := 0; x < croppedWidth; x++ {
			cropped.Set(x, y, srcImg.At(minX+x, minY+y))
		}
	}

	// 3. Compute optimal scaling to fit inside canvas with breathing room
	maxAllowedW := float64(canvasSize) * (1.0 - 2.0*paddingRatio)
	maxAllowedH := float64(canvasSize) * (1.0 - 2.0*paddingRatio)

	scaleW := maxAllowedW / float64(croppedWidth)
	scaleH := maxAllowedH / float64(croppedHeight)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}
	if scale > 2.0 {
		scale = 2.0 // Don't over-magnify tiny resolution crops
	}

	scaledW := int(float64(croppedWidth) * scale)
	scaledH := int(float64(croppedHeight) * scale)
	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}

	scaledProduct := image.NewRGBA(image.Rect(0, 0, scaledW, scaledH))
	bilinearScale(scaledProduct, cropped)

	// 4. Center horizontally, place product resting naturally on bottom third
	posX := (canvasSize - scaledW) / 2
	// Slightly anchor lower than exact center (offset by 3%) for natural grounded feel
	posY := (canvasSize - scaledH) / 2 + int(float64(canvasSize)*0.02)
	if posY+scaledH > canvasSize-int(float64(canvasSize)*0.05) {
		posY = canvasSize - scaledH - int(float64(canvasSize)*0.05)
	}

	// 5. Create studio canvas with category backdrop
	canvas := image.NewRGBA(image.Rect(0, 0, canvasSize, canvasSize))
	fillCanvasBackdrop(canvas, theme)

	// 6. Draw Category-Tailored Contact & Ambient Shadow
	drawCategoryShadow(canvas, theme, posX, posY, scaledW, scaledH)

	// 7. Composite Product Cutout on top of Canvas and Shadow
	draw.Draw(canvas, image.Rect(posX, posY, posX+scaledW, posY+scaledH), scaledProduct, image.Point{0, 0}, draw.Over)

	// 8. Encode to high-quality JPEG (90% quality)
	var outBuf bytes.Buffer
	err = jpeg.Encode(&outBuf, canvas, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, fmt.Errorf("failed to encode studio image: %w", err)
	}

	return outBuf.Bytes(), nil
}

// findOpaqueBounds locates the tight bounding rectangle of visible pixels (Alpha > 25)
func findOpaqueBounds(src image.Image) (int, int, int, int, bool) {
	bounds := src.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	found := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a > 0x1919 { // Alpha > ~25 (out of 255)
				found = true
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if !found {
		return 0, 0, 0, 0, false
	}
	// Add 1 to max to make bounding box inclusive of the outer pixel
	return minX, minY, maxX + 1, maxY + 1, true
}

// fillCanvasBackdrop fills the background with subtle luxury studio tint
func fillCanvasBackdrop(canvas *image.RGBA, theme StudioTheme) {
	bounds := canvas.Bounds()
	var baseColor color.RGBA

	switch theme {
	case StudioThemeCosmetics:
		// Luxury off-white with cool pearl tint (#F6F8FB)
		baseColor = color.RGBA{R: 246, G: 248, B: 251, A: 255}
	case StudioThemeEthnicFashion:
		// Warm studio aesthetic gallery off-white (#FAF9F6)
		baseColor = color.RGBA{R: 250, G: 249, B: 246, A: 255}
	default:
		// Pure Amazon/Flipkart White (#FFFFFF)
		baseColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			canvas.SetRGBA(x, y, baseColor)
		}
	}
}

// drawCategoryShadow paints a smooth elliptical contact shadow under the product base
func drawCategoryShadow(canvas *image.RGBA, theme StudioTheme, posX, posY, pWidth, pHeight int) {
	baseY := posY + pHeight

	var shadowRadiusX, shadowRadiusY float64
	var maxAlpha float64
	var offsetY int

	switch theme {
	case StudioThemeCosmetics:
		// Crisp, sleek ground reflection shadow directly underneath cosmetics bottle/jar
		shadowRadiusX = float64(pWidth) * 0.42
		shadowRadiusY = float64(pHeight) * 0.035
		if shadowRadiusY < 12 {
			shadowRadiusY = 12
		}
		maxAlpha = 0.38
		offsetY = -2

	case StudioThemeEthnicFashion:
		// Soft, wide diffused ambient floor shadow for flowing fabric, kurtis & sarees
		shadowRadiusX = float64(pWidth) * 0.48
		shadowRadiusY = float64(pHeight) * 0.050
		if shadowRadiusY < 16 {
			shadowRadiusY = 16
		}
		maxAlpha = 0.28
		offsetY = 4

	default: // Packaged Retail
		// Tight horizontal pill contact shadow
		shadowRadiusX = float64(pWidth) * 0.38
		shadowRadiusY = float64(pHeight) * 0.030
		if shadowRadiusY < 10 {
			shadowRadiusY = 10
		}
		maxAlpha = 0.42
		offsetY = 0
	}

	centerX := float64(posX) + float64(pWidth)/2.0
	centerY := float64(baseY + offsetY)

	minX := int(centerX - shadowRadiusX)
	maxX := int(centerX + shadowRadiusX)
	minY := int(centerY - shadowRadiusY)
	maxY := int(centerY + shadowRadiusY)

	cBounds := canvas.Bounds()
	if minX < cBounds.Min.X {
		minX = cBounds.Min.X
	}
	if maxX > cBounds.Max.X {
		maxX = cBounds.Max.X
	}
	if minY < cBounds.Min.Y {
		minY = cBounds.Min.Y
	}
	if maxY > cBounds.Max.Y {
		maxY = cBounds.Max.Y
	}

	for y := minY; y < maxY; y++ {
		dy := float64(y) - centerY
		for x := minX; x < maxX; x++ {
			dx := float64(x) - centerX

			// Normalized ellipse equation: (dx/rx)^2 + (dy/ry)^2
			distSq := (dx*dx)/(shadowRadiusX*shadowRadiusX) + (dy*dy)/(shadowRadiusY*shadowRadiusY)
			if distSq < 1.0 {
				// Smooth cosine/cubic falloff from center to edge
				factor := math.Cos(math.Sqrt(distSq) * (math.Pi / 2.0))
				alphaFactor := factor * factor * maxAlpha

				cur := canvas.RGBAAt(x, y)
				// Blend dark grey shadow (#1a1a24) over existing canvas color
				shadowR, shadowG, shadowB := 26.0, 26.0, 36.0

				newR := uint8(float64(cur.R)*(1.0-alphaFactor) + shadowR*alphaFactor)
				newG := uint8(float64(cur.G)*(1.0-alphaFactor) + shadowG*alphaFactor)
				newB := uint8(float64(cur.B)*(1.0-alphaFactor) + shadowB*alphaFactor)

				canvas.SetRGBA(x, y, color.RGBA{R: newR, G: newG, B: newB, A: 255})
			}
		}
	}
}
