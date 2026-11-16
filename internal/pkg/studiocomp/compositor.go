package studiocomp

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

// Theme represents the visual backdrop and lighting ambiance for the studio render.
type Theme string

const (
	ThemeCosmetics      Theme = "cosmetics"       // Luxury soft pearl/glass background + ground contact reflection
	ThemeEthnicFashion  Theme = "ethnic_fashion"  // Warm gallery aesthetic canvas + soft diffused ambient shadow
	ThemePackagedRetail Theme = "packaged_retail" // Crisp pure catalog white (#FFFFFF) + tight base contact shadow
	ThemeMinimalistDark Theme = "minimalist_dark" // Modern sleek obsidian/charcoal background (#18181B)
)

// DetermineStudioTheme infers the optimal visual studio theme from category and subcategory text.
func DetermineStudioTheme(categoryHint, subcategory string) Theme {
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
		return ThemeCosmetics

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
		return ThemeEthnicFashion

	default:
		return ThemePackagedRetail
	}
}

// Options configures the studio canvas, dimensions, themes, and padding margins.
type Options struct {
	Theme        Theme
	CanvasSize   int     // Standard square dimension (e.g. 1080)
	PaddingRatio float64 // Proportion of canvas reserved for margin (default 0.10)
	JpegQuality  int     // Output JPEG quality 1-100 (default 90)
}

// RenderProductStudio takes transparent PNG or standard image bytes
// and composites a studio-grade catalog image with category-tailored
// backdrops, soft ambient shadows, and centered visual balance.
func RenderProductStudio(cutoutBytes []byte, opts Options) ([]byte, error) {
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
		theme = ThemePackagedRetail
	}

	quality := opts.JpegQuality
	if quality <= 0 || quality > 100 {
		quality = 90
	}

	// 1. Detect tight non-transparent bounding box around product
	minX, minY, maxX, maxY, hasAlpha := FindOpaqueBounds(srcImg)
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
	BilinearScale(scaledProduct, cropped)

	// Soft sub-pixel alpha feathering to eliminate jagged cutout edges
	FeatherEdges(scaledProduct)

	// 4. Center horizontally, place product resting naturally on bottom third
	posX := (canvasSize - scaledW) / 2
	// Slightly anchor lower than exact center (offset by 2%) for natural grounded feel
	posY := (canvasSize - scaledH) / 2 + int(float64(canvasSize)*0.02)
	if posY+scaledH > canvasSize-int(float64(canvasSize)*0.05) {
		posY = canvasSize - scaledH - int(float64(canvasSize)*0.05)
	}

	// 5. Create studio canvas with category backdrop
	canvas := image.NewRGBA(image.Rect(0, 0, canvasSize, canvasSize))
	FillCanvasBackdrop(canvas, theme)

	// 6. Draw Category-Tailored Contact & Ambient Shadow
	DrawContactShadow(canvas, theme, posX, posY, scaledW, scaledH)

	// 7. Composite Product Cutout on top of Canvas and Shadow
	draw.Draw(canvas, image.Rect(posX, posY, posX+scaledW, posY+scaledH), scaledProduct, image.Point{0, 0}, draw.Over)

	// 8. Encode to high-quality JPEG
	var outBuf bytes.Buffer
	err = jpeg.Encode(&outBuf, canvas, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode studio image: %w", err)
	}

	return outBuf.Bytes(), nil
}

// FindOpaqueBounds locates the tight bounding rectangle of visible pixels (Alpha > ~25).
func FindOpaqueBounds(src image.Image) (int, int, int, int, bool) {
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
	return minX, minY, maxX + 1, maxY + 1, true
}

// BilinearScale performs fast, pure-Go bilinear interpolation scaling from src into dst.
func BilinearScale(dst *image.RGBA, src image.Image) {
	dstBounds := dst.Bounds()
	srcBounds := src.Bounds()

	if dstBounds.Dx() == 0 || dstBounds.Dy() == 0 || srcBounds.Dx() == 0 || srcBounds.Dy() == 0 {
		return
	}

	dx := float64(srcBounds.Dx()) / float64(dstBounds.Dx())
	dy := float64(srcBounds.Dy()) / float64(dstBounds.Dy())

	for y := dstBounds.Min.Y; y < dstBounds.Max.Y; y++ {
		srcY := float64(y)*dy + float64(srcBounds.Min.Y)
		yFloor := int(srcY)
		yFrac := srcY - float64(yFloor)
		yNext := yFloor + 1
		if yNext >= srcBounds.Max.Y {
			yNext = srcBounds.Max.Y - 1
		}

		for x := dstBounds.Min.X; x < dstBounds.Max.X; x++ {
			srcX := float64(x)*dx + float64(srcBounds.Min.X)
			xFloor := int(srcX)
			xFrac := srcX - float64(xFloor)
			xNext := xFloor + 1
			if xNext >= srcBounds.Max.X {
				xNext = srcBounds.Max.X - 1
			}

			r00, g00, b00, a00 := src.At(xFloor, yFloor).RGBA()
			r10, g10, b10, a10 := src.At(xNext, yFloor).RGBA()
			r01, g01, b01, a01 := src.At(xFloor, yNext).RGBA()
			r11, g11, b11, a11 := src.At(xNext, yNext).RGBA()

			w00 := (1.0 - xFrac) * (1.0 - yFrac)
			w10 := xFrac * (1.0 - yFrac)
			w01 := (1.0 - xFrac) * yFrac
			w11 := xFrac * yFrac

			r := uint8((float64(r00)*w00 + float64(r10)*w10 + float64(r01)*w01 + float64(r11)*w11) / 257.0)
			g := uint8((float64(g00)*w00 + float64(g10)*w10 + float64(g01)*w01 + float64(g11)*w11) / 257.0)
			b := uint8((float64(b00)*w00 + float64(b10)*w10 + float64(b01)*w01 + float64(b11)*w11) / 257.0)
			a := uint8((float64(a00)*w00 + float64(a10)*w10 + float64(a01)*w01 + float64(a11)*w11) / 257.0)

			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
}

// FeatherEdges applies a gentle 1-pixel alpha feathering along silhouette edges
// so the product cutout blends smoothly into the canvas background without jagged artifacts.
func FeatherEdges(img *image.RGBA) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	stride := img.Stride

	alphas := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		rowOffset := y * stride
		for x := 0; x < w; x++ {
			alphas[y*w+x] = img.Pix[rowOffset+x*4+3]
		}
	}

	for y := 1; y < h-1; y++ {
		rowOffset := y * stride
		for x := 1; x < w-1; x++ {
			curA := alphas[y*w+x]
			if curA > 0 && curA < 250 {
				var sumA int
				sumA += int(alphas[(y-1)*w+(x-1)]) + int(alphas[(y-1)*w+x]) + int(alphas[(y-1)*w+(x+1)])
				sumA += int(alphas[y*w+(x-1)]) + int(curA)*2 + int(alphas[y*w+(x+1)])
				sumA += int(alphas[(y+1)*w+(x-1)]) + int(alphas[(y+1)*w+x]) + int(alphas[(y+1)*w+(x+1)])
				smoothed := uint8(sumA / 10)
				img.Pix[rowOffset+x*4+3] = smoothed
			}
		}
	}
}

// FillCanvasBackdrop fills the background with subtle luxury studio tint.
func FillCanvasBackdrop(canvas *image.RGBA, theme Theme) {
	bounds := canvas.Bounds()
	var baseColor color.RGBA

	switch theme {
	case ThemeCosmetics:
		// Luxury off-white with cool pearl tint (#F6F8FB)
		baseColor = color.RGBA{R: 246, G: 248, B: 251, A: 255}
	case ThemeEthnicFashion:
		// Warm studio aesthetic gallery off-white (#FAF9F6)
		baseColor = color.RGBA{R: 250, G: 249, B: 246, A: 255}
	case ThemeMinimalistDark:
		// Obsidian charcoal dark mode (#18181B)
		baseColor = color.RGBA{R: 24, G: 24, B: 27, A: 255}
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

// DrawContactShadow paints a smooth elliptical contact shadow under the product base.
func DrawContactShadow(canvas *image.RGBA, theme Theme, posX, posY, pWidth, pHeight int) {
	baseY := posY + pHeight

	var shadowRadiusX, shadowRadiusY float64
	var maxAlpha float64
	var offsetY int

	switch theme {
	case ThemeCosmetics:
		shadowRadiusX = float64(pWidth) * 0.42
		shadowRadiusY = float64(pHeight) * 0.035
		if shadowRadiusY < 12 {
			shadowRadiusY = 12
		}
		maxAlpha = 0.38
		offsetY = -2

	case ThemeEthnicFashion:
		shadowRadiusX = float64(pWidth) * 0.48
		shadowRadiusY = float64(pHeight) * 0.050
		if shadowRadiusY < 16 {
			shadowRadiusY = 16
		}
		maxAlpha = 0.28
		offsetY = 4

	case ThemeMinimalistDark:
		shadowRadiusX = float64(pWidth) * 0.45
		shadowRadiusY = float64(pHeight) * 0.040
		if shadowRadiusY < 12 {
			shadowRadiusY = 12
		}
		maxAlpha = 0.50
		offsetY = 1

	default: // Packaged Retail
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

			distSq := (dx*dx)/(shadowRadiusX*shadowRadiusX) + (dy*dy)/(shadowRadiusY*shadowRadiusY)
			if distSq < 1.0 {
				factor := math.Cos(math.Sqrt(distSq) * (math.Pi / 2.0))
				alphaFactor := factor * factor * maxAlpha

				cur := canvas.RGBAAt(x, y)
				shadowR, shadowG, shadowB := 26.0, 26.0, 36.0

				newR := uint8(float64(cur.R)*(1.0-alphaFactor) + shadowR*alphaFactor)
				newG := uint8(float64(cur.G)*(1.0-alphaFactor) + shadowG*alphaFactor)
				newB := uint8(float64(cur.B)*(1.0-alphaFactor) + shadowB*alphaFactor)

				canvas.SetRGBA(x, y, color.RGBA{R: newR, G: newG, B: newB, A: 255})
			}
		}
	}
}
