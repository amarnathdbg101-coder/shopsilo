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
	"runtime"
	"strings"
	"sync"
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

	// Auto-lighting, white-balance neutralization & vibrance enhancement
	enhanceProductLighting(scaledProduct)

	// Soft sub-pixel alpha feathering to eliminate jagged cutout edges
	featherProductEdges(scaledProduct)

	// 4. Center horizontally, place product resting naturally on bottom third
	posX := (canvasSize - scaledW) / 2
	// Slightly anchor lower than exact center (offset by 3%) for natural grounded feel
	posY := (canvasSize-scaledH)/2 + int(float64(canvasSize)*0.02)
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

// enhanceProductLighting applies subtle auto-levels, white-balance neutralization,
// and vibrance enhancement exclusively on the product pixels (alpha > 20).
func enhanceProductLighting(img *image.RGBA) {
	pix := img.Pix
	n := len(pix)
	if n < 4 {
		return
	}

	// Pass 1: Sample average and min/max luminance of opaque pixels
	var sumR, sumG, sumB uint64
	var count uint64
	minL := 255.0
	maxL := 0.0

	for i := 0; i < n; i += 4 {
		a := pix[i+3]
		if a > 30 {
			r := float64(pix[i])
			g := float64(pix[i+1])
			b := float64(pix[i+2])
			lum := 0.299*r + 0.587*g + 0.114*b

			if lum < minL {
				minL = lum
			}
			if lum > maxL {
				maxL = lum
			}
			sumR += uint64(pix[i])
			sumG += uint64(pix[i+1])
			sumB += uint64(pix[i+2])
			count++
		}
	}

	if count < 100 || maxL <= minL+10 {
		return
	}

	// Auto-contrast stretch range (target range [10, 245])
	lumRange := maxL - minL
	contrastScale := 235.0 / lumRange
	if contrastScale > 1.35 {
		contrastScale = 1.35 // Don't over-amplify noise in very dark photos
	}

	// Subtle white balance correction factor (cool down yellow indoor light by up to 6%)
	avgR := float64(sumR) / float64(count)
	avgB := float64(sumB) / float64(count)
	rGain := 1.0
	bGain := 1.0
	if avgR > avgB*1.15 {
		// Image is warm/yellow from shop incandescent/halogen bulbs
		rGain = 0.96
		bGain = 1.06
	}

	// Pass 2: Apply contrast stretch, white balance, and vibrance
	for i := 0; i < n; i += 4 {
		a := pix[i+3]
		if a <= 20 {
			continue
		}

		r := float64(pix[i]) * rGain
		g := float64(pix[i+1])
		b := float64(pix[i+2]) * bGain

		// 1. Contrast stretch
		r = (r-minL)*contrastScale + 10.0
		g = (g-minL)*contrastScale + 10.0
		b = (b-minL)*contrastScale + 10.0

		// 2. Vibrance boost (+12% saturation on midtones)
		maxRGB := math.Max(r, math.Max(g, b))
		minRGB := math.Min(r, math.Min(g, b))
		sat := 0.0
		if maxRGB > 0 {
			sat = (maxRGB - minRGB) / maxRGB
		}
		// Boost more if saturation is low/moderate, less if already saturated
		vibranceBoost := (1.0 - sat) * 0.12
		lum := 0.299*r + 0.587*g + 0.114*b

		r = lum + (r-lum)*(1.0+vibranceBoost)
		g = lum + (g-lum)*(1.0+vibranceBoost)
		b = lum + (b-lum)*(1.0+vibranceBoost)

		// Clamp to [0, 255]
		if r < 0 {
			r = 0
		} else if r > 255 {
			r = 255
		}
		if g < 0 {
			g = 0
		} else if g > 255 {
			g = 255
		}
		if b < 0 {
			b = 0
		} else if b > 255 {
			b = 255
		}

		pix[i] = uint8(r)
		pix[i+1] = uint8(g)
		pix[i+2] = uint8(b)
	}
}

// featherProductEdges applies a gentle 1-pixel alpha feathering along the silhouette edges
// so the product cutout blends smoothly into the canvas background without jagged artifacts.
func featherProductEdges(img *image.RGBA) {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	stride := img.Stride

	// Create a copy of the alpha channel to read from while writing smoothed values
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
			// Only smooth edge transition pixels (partially transparent or adjacent to transparent)
			if curA > 0 && curA < 250 {
				// 3x3 neighborhood average for alpha
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

// RenderProductBokeh takes raw product photo bytes and produces a DSLR-grade portrait bokeh effect:
// The central product remains 100% sharp and vibrant, while the surrounding background clutter is softly blurred.
// Uses multi-core parallel box blur, content-aware background saliency, and specular bokeh highlight rendering.
func RenderProductBokeh(imgBytes []byte, blurRadius int) ([]byte, error) {
	if len(imgBytes) == 0 {
		return nil, fmt.Errorf("empty image bytes")
	}

	srcImg, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := srcImg.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	origRGBA := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(origRGBA, origRGBA.Bounds(), srcImg, bounds.Min, draw.Src)

	if blurRadius < 8 {
		blurRadius = 14
	}
	blurredRGBA := fastBoxBlur(origRGBA, blurRadius)

	// Sample 4 border margins to detect the ambient background color & texture distribution
	bgMeanR, bgMeanG, bgMeanB := sampleBorderBackground(origRGBA, w, h)

	cx := float64(w) / 2.0
	cy := float64(h) / 2.0
	rx := (float64(w) * 0.70) / 2.0
	ry := (float64(h) * 0.74) / 2.0

	outRGBA := image.NewRGBA(image.Rect(0, 0, w, h))
	stride := origRGBA.Stride

	// Multi-core parallel composite blending with content-aware saliency & specular highlights
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	rowsPerWorker := (h + workers - 1) / workers

	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		startY := wi * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > h {
			endY = h
		}
		if startY >= endY {
			continue
		}

		wg.Add(1)
		go func(sY, eY int) {
			defer wg.Done()

			for y := sY; y < eY; y++ {
				dy := (float64(y) - cy) / ry
				dySq := dy * dy
				rowOffset := y * stride

				for x := 0; x < w; x++ {
					dx := (float64(x) - cx) / rx
					distSq := dx*dx + dySq
					idx := rowOffset + x*4

					origR := float64(origRGBA.Pix[idx])
					origG := float64(origRGBA.Pix[idx+1])
					origB := float64(origRGBA.Pix[idx+2])

					// Color contrast from border background (0.0 to 1.0)
					dR := math.Abs(origR - bgMeanR)
					dG := math.Abs(origG - bgMeanG)
					dB := math.Abs(origB - bgMeanB)
					colorDiff := (dR + dG + dB) / (3.0 * 255.0)

					// Content-aware effective focal distance:
					// If the pixel differs significantly from the border background (packaging text, colorful label),
					// it belongs to the product and its corners are kept sharp!
					effectiveDist := distSq - (colorDiff * 1.25)

					if effectiveDist <= 0.85 {
						// Razor sharp product focus zone
						outRGBA.Pix[idx] = uint8(origR)
						outRGBA.Pix[idx+1] = uint8(origG)
						outRGBA.Pix[idx+2] = uint8(origB)
						outRGBA.Pix[idx+3] = 255
					} else if effectiveDist >= 1.6 {
						// Deep optical bokeh
						blurR := float64(blurredRGBA.Pix[idx])
						blurG := float64(blurredRGBA.Pix[idx+1])
						blurB := float64(blurredRGBA.Pix[idx+2])

						// Specular highlight boost: bright light reflections expand into glowing bokeh discs
						lum := 0.299*blurR + 0.587*blurG + 0.114*blurB
						if lum > 180.0 {
							factor := math.Min(255.0, lum*1.10) / lum
							outRGBA.Pix[idx] = uint8(math.Min(255.0, blurR*factor))
							outRGBA.Pix[idx+1] = uint8(math.Min(255.0, blurG*factor))
							outRGBA.Pix[idx+2] = uint8(math.Min(255.0, blurB*factor))
						} else {
							outRGBA.Pix[idx] = uint8(blurR)
							outRGBA.Pix[idx+1] = uint8(blurG)
							outRGBA.Pix[idx+2] = uint8(blurB)
						}
						outRGBA.Pix[idx+3] = 255
					} else {
						// Smooth feather transition between sharp foreground and creamy bokeh
						t := (effectiveDist - 0.85) / 0.75
						if t < 0.0 {
							t = 0.0
						} else if t > 1.0 {
							t = 1.0
						}

						blurR := float64(blurredRGBA.Pix[idx])
						blurG := float64(blurredRGBA.Pix[idx+1])
						blurB := float64(blurredRGBA.Pix[idx+2])

						r := origR*(1.0-t) + blurR*t
						g := origG*(1.0-t) + blurG*t
						b := origB*(1.0-t) + blurB*t

						outRGBA.Pix[idx] = uint8(math.Min(255.0, math.Max(0.0, r)))
						outRGBA.Pix[idx+1] = uint8(math.Min(255.0, math.Max(0.0, g)))
						outRGBA.Pix[idx+2] = uint8(math.Min(255.0, math.Max(0.0, b)))
						outRGBA.Pix[idx+3] = 255
					}
				}
			}
		}(startY, endY)
	}
	wg.Wait()

	enhanceProductLighting(outRGBA)

	var outBuf bytes.Buffer
	if err := jpeg.Encode(&outBuf, outRGBA, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("failed to encode bokeh image: %w", err)
	}

	return outBuf.Bytes(), nil
}

// sampleBorderBackground calculates the average RGB of perimeter borders to infer the ambient background
func sampleBorderBackground(img *image.RGBA, w, h int) (float64, float64, float64) {
	borderX := int(float64(w) * 0.05)
	if borderX < 2 {
		borderX = 2
	}
	borderY := int(float64(h) * 0.05)
	if borderY < 2 {
		borderY = 2
	}

	stepX := w / 40
	if stepX < 1 {
		stepX = 1
	}
	stepY := h / 40
	if stepY < 1 {
		stepY = 1
	}

	var sumR, sumG, sumB, count int64
	stride := img.Stride

	// Top and Bottom borders
	for x := 0; x < w; x += stepX {
		for y := 0; y < borderY; y += stepY {
			idx := y*stride + x*4
			sumR += int64(img.Pix[idx])
			sumG += int64(img.Pix[idx+1])
			sumB += int64(img.Pix[idx+2])
			count++
		}
		for y := h - borderY; y < h; y += stepY {
			idx := y*stride + x*4
			sumR += int64(img.Pix[idx])
			sumG += int64(img.Pix[idx+1])
			sumB += int64(img.Pix[idx+2])
			count++
		}
	}

	// Left and Right borders
	for y := borderY; y < h-borderY; y += stepY {
		for x := 0; x < borderX; x += stepX {
			idx := y*stride + x*4
			sumR += int64(img.Pix[idx])
			sumG += int64(img.Pix[idx+1])
			sumB += int64(img.Pix[idx+2])
			count++
		}
		for x := w - borderX; x < w; x += stepX {
			idx := y*stride + x*4
			sumR += int64(img.Pix[idx])
			sumG += int64(img.Pix[idx+1])
			sumB += int64(img.Pix[idx+2])
			count++
		}
	}

	if count == 0 {
		return 128.0, 128.0, 128.0
	}
	return float64(sumR) / float64(count), float64(sumG) / float64(count), float64(sumB) / float64(count)
}

func fastBoxBlur(src *image.RGBA, radius int) *image.RGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	copy(dst.Pix, src.Pix)

	temp := image.NewRGBA(image.Rect(0, 0, w, h))
	for pass := 0; pass < 2; pass++ {
		boxBlurH(dst, temp, w, h, radius)
		boxBlurV(temp, dst, w, h, radius)
	}
	return dst
}

func boxBlurH(src, dst *image.RGBA, w, h, r int) {
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	rowsPerWorker := (h + workers - 1) / workers

	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		startY := wi * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > h {
			endY = h
		}
		if startY >= endY {
			continue
		}

		wg.Add(1)
		go func(sY, eY int) {
			defer wg.Done()
			for y := sY; y < eY; y++ {
				rowOffset := y * src.Stride
				var sumR, sumG, sumB int
				windowSize := 0

				for x := -r; x <= r; x++ {
					px := clamp(x, 0, w-1)
					idx := rowOffset + px*4
					sumR += int(src.Pix[idx])
					sumG += int(src.Pix[idx+1])
					sumB += int(src.Pix[idx+2])
					windowSize++
				}

				for x := 0; x < w; x++ {
					outIdx := rowOffset + x*4
					dst.Pix[outIdx] = uint8(sumR / windowSize)
					dst.Pix[outIdx+1] = uint8(sumG / windowSize)
					dst.Pix[outIdx+2] = uint8(sumB / windowSize)
					dst.Pix[outIdx+3] = 255

					leftPx := clamp(x-r, 0, w-1)
					rightPx := clamp(x+r+1, 0, w-1)
					leftIdx := rowOffset + leftPx*4
					rightIdx := rowOffset + rightPx*4

					sumR += int(src.Pix[rightIdx]) - int(src.Pix[leftIdx])
					sumG += int(src.Pix[rightIdx+1]) - int(src.Pix[leftIdx+1])
					sumB += int(src.Pix[rightIdx+2]) - int(src.Pix[leftIdx+2])
				}
			}
		}(startY, endY)
	}
	wg.Wait()
}

func boxBlurV(src, dst *image.RGBA, w, h, r int) {
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	colsPerWorker := (w + workers - 1) / workers

	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		startX := wi * colsPerWorker
		endX := startX + colsPerWorker
		if endX > w {
			endX = w
		}
		if startX >= endX {
			continue
		}

		wg.Add(1)
		go func(sX, eX int) {
			defer wg.Done()
			for x := sX; x < eX; x++ {
				var sumR, sumG, sumB int
				windowSize := 0

				for y := -r; y <= r; y++ {
					py := clamp(y, 0, h-1)
					idx := py*src.Stride + x*4
					sumR += int(src.Pix[idx])
					sumG += int(src.Pix[idx+1])
					sumB += int(src.Pix[idx+2])
					windowSize++
				}

				for y := 0; y < h; y++ {
					outIdx := y*dst.Stride + x*4
					dst.Pix[outIdx] = uint8(sumR / windowSize)
					dst.Pix[outIdx+1] = uint8(sumG / windowSize)
					dst.Pix[outIdx+2] = uint8(sumB / windowSize)
					dst.Pix[outIdx+3] = 255

					topPy := clamp(y-r, 0, h-1)
					botPy := clamp(y+r+1, 0, h-1)
					topIdx := topPy*src.Stride + x*4
					botIdx := botPy*src.Stride + x*4

					sumR += int(src.Pix[botIdx]) - int(src.Pix[topIdx])
					sumG += int(src.Pix[botIdx+1]) - int(src.Pix[topIdx+1])
					sumB += int(src.Pix[botIdx+2]) - int(src.Pix[topIdx+2])
				}
			}
		}(startX, endX)
	}
	wg.Wait()
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// RenderProductBokeh takes raw product photo bytes and produces a DSLR-grade portrait bokeh effect:
// The central product remains 100% sharp and vibrant, while the surrounding background clutter is softly blurred.
func RenderProductBokeh(imgBytes []byte, blurRadius int) ([]byte, error) {
	if len(imgBytes) == 0 {
		return nil, fmt.Errorf("empty image bytes")
	}

	srcImg, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := srcImg.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	origRGBA := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(origRGBA, origRGBA.Bounds(), srcImg, bounds.Min, draw.Src)

	if blurRadius < 8 {
		blurRadius = 14
	}
	blurredRGBA := fastBoxBlur(origRGBA, blurRadius)

	cx := float64(w) / 2.0
	cy := float64(h) / 2.0
	rx := (float64(w) * 0.70) / 2.0
	ry := (float64(h) * 0.74) / 2.0

	outRGBA := image.NewRGBA(image.Rect(0, 0, w, h))
	stride := origRGBA.Stride

	// Multi-core parallel composite blending with content-aware saliency & specular highlights
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	rowsPerWorker := (h + workers - 1) / workers

	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		startY := wi * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > h {
			endY = h
		}
		if startY >= endY {
			continue
		}

		wg.Add(1)
		go func(sY, eY int) {
			defer wg.Done()

			for y := sY; y < eY; y++ {
				dy := (float64(y) - cy) / ry
				dySq := dy * dy
				rowOffset := y * stride

				for x := 0; x < w; x++ {
					dx := (float64(x) - cx) / rx
					distSq := dx*dx + dySq
					idx := rowOffset + x*4

					origR := float64(origRGBA.Pix[idx])
					origG := float64(origRGBA.Pix[idx+1])
					origB := float64(origRGBA.Pix[idx+2])

					// Color contrast from border background (0.0 to 1.0)
					dR := math.Abs(origR - bgMeanR)
					dG := math.Abs(origG - bgMeanG)
					dB := math.Abs(origB - bgMeanB)
					colorDiff := (dR + dG + dB) / (3.0 * 255.0)

					// Content-aware effective focal distance:
					// If the pixel differs significantly from the border background (packaging text, colorful label),
					// it belongs to the product and its corners are kept sharp!
					effectiveDist := distSq - (colorDiff * 1.25)

					if effectiveDist <= 0.85 {
						// Razor sharp product focus zone
						outRGBA.Pix[idx] = uint8(origR)
						outRGBA.Pix[idx+1] = uint8(origG)
						outRGBA.Pix[idx+2] = uint8(origB)
						outRGBA.Pix[idx+3] = 255
					} else if effectiveDist >= 1.6 {
						// Deep optical bokeh
						blurR := float64(blurredRGBA.Pix[idx])
						blurG := float64(blurredRGBA.Pix[idx+1])
						blurB := float64(blurredRGBA.Pix[idx+2])

						// Specular highlight boost: bright light reflections expand into glowing bokeh discs
						lum := 0.299*blurR + 0.587*blurG + 0.114*blurB
						if lum > 180.0 {
							factor := math.Min(255.0, lum*1.10) / lum
							outRGBA.Pix[idx] = uint8(math.Min(255.0, blurR*factor))
							outRGBA.Pix[idx+1] = uint8(math.Min(255.0, blurG*factor))
							outRGBA.Pix[idx+2] = uint8(math.Min(255.0, blurB*factor))
						} else {
							outRGBA.Pix[idx] = uint8(blurR)
							outRGBA.Pix[idx+1] = uint8(blurG)
							outRGBA.Pix[idx+2] = uint8(blurB)
						}
						outRGBA.Pix[idx+3] = 255
					} else {
						// Smooth feather transition between sharp foreground and creamy bokeh
						t := (effectiveDist - 0.85) / 0.75
						if t < 0.0 {
							t = 0.0
						} else if t > 1.0 {
							t = 1.0
						}

						blurR := float64(blurredRGBA.Pix[idx])
						blurG := float64(blurredRGBA.Pix[idx+1])
						blurB := float64(blurredRGBA.Pix[idx+2])

						r := origR*(1.0-t) + blurR*t
						g := origG*(1.0-t) + blurG*t
						b := origB*(1.0-t) + blurB*t

						outRGBA.Pix[idx] = uint8(math.Min(255.0, math.Max(0.0, r)))
						outRGBA.Pix[idx+1] = uint8(math.Min(255.0, math.Max(0.0, g)))
						outRGBA.Pix[idx+2] = uint8(math.Min(255.0, math.Max(0.0, b)))
						outRGBA.Pix[idx+3] = 255
					}
				}
			}
		}(startY, endY)
	}
	wg.Wait()

	enhanceProductLighting(outRGBA)

	var outBuf bytes.Buffer
	if err := jpeg.Encode(&outBuf, outRGBA, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("failed to encode bokeh image: %w", err)
	}

	return outBuf.Bytes(), nil
}

// sampleBorderBackground calculates the average RGB of perimeter borders to infer the ambient background
func sampleBorderBackground(img *image.RGBA, w, h int) (float64, float64, float64) {
	borderX := int(float64(w) * 0.05)
	if borderX < 2 {
		borderX = 2
	}
	borderY := int(float64(h) * 0.05)
	if borderY < 2 {
		borderY = 2
	}

	stepX := w / 40
	if stepX < 1 {
		stepX = 1
	}
	stepY := h / 40
	if stepY < 1 {
		stepY = 1
	}

	var sumR, sumG, sumB, count int64
	stride := img.Stride

	// Top and Bottom borders
	for x := 0; x < w; x += stepX {
		for y := 0; y < borderY; y += stepY {
			idx := y*stride + x*4
			sumR += int64(img.Pix[idx])
			sumG += int64(img.Pix[idx+1])
			sumB += int64(img.Pix[idx+2])
			count++
		}
		for y := h - borderY; y < h; y += stepY {
			idx := y*stride + x*4
			sumR += int64(img.Pix[idx])
			sumG += int64(img.Pix[idx+1])
			sumB += int64(img.Pix[idx+2])
			count++
		}
	}

	// Left and Right borders
	for y := borderY; y < h-borderY; y += stepY {
		for x := 0; x < borderX; x += stepX {
			idx := y*stride + x*4
			sumR += int64(img.Pix[idx])
			sumG += int64(img.Pix[idx+1])
			sumB += int64(img.Pix[idx+2])
			count++
		}
		for x := w - borderX; x < w; x += stepX {
			idx := y*stride + x*4
			sumR += int64(img.Pix[idx])
			sumG += int64(img.Pix[idx+1])
			sumB += int64(img.Pix[idx+2])
			count++
		}
	}

	if count == 0 {
		return 128.0, 128.0, 128.0
	}
	return float64(sumR) / float64(count), float64(sumG) / float64(count), float64(sumB) / float64(count)
}

func fastBoxBlur(src *image.RGBA, radius int) *image.RGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	copy(dst.Pix, src.Pix)

	temp := image.NewRGBA(image.Rect(0, 0, w, h))
	for pass := 0; pass < 2; pass++ {
		boxBlurH(dst, temp, w, h, radius)
		boxBlurV(temp, dst, w, h, radius)
	}
	return dst
}

func boxBlurH(src, dst *image.RGBA, w, h, r int) {
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	rowsPerWorker := (h + workers - 1) / workers

	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		startY := wi * rowsPerWorker
		endY := startY + rowsPerWorker
		if endY > h {
			endY = h
		}
		if startY >= endY {
			continue
		}

		wg.Add(1)
		go func(sY, eY int) {
			defer wg.Done()
			for y := sY; y < eY; y++ {
				rowOffset := y * src.Stride
				var sumR, sumG, sumB int
				windowSize := 0

				for x := -r; x <= r; x++ {
					px := clamp(x, 0, w-1)
					idx := rowOffset + px*4
					sumR += int(src.Pix[idx])
					sumG += int(src.Pix[idx+1])
					sumB += int(src.Pix[idx+2])
					windowSize++
				}

				for x := 0; x < w; x++ {
					outIdx := rowOffset + x*4
					dst.Pix[outIdx] = uint8(sumR / windowSize)
					dst.Pix[outIdx+1] = uint8(sumG / windowSize)
					dst.Pix[outIdx+2] = uint8(sumB / windowSize)
					dst.Pix[outIdx+3] = 255

					leftPx := clamp(x-r, 0, w-1)
					rightPx := clamp(x+r+1, 0, w-1)
					leftIdx := rowOffset + leftPx*4
					rightIdx := rowOffset + rightPx*4

					sumR += int(src.Pix[rightIdx]) - int(src.Pix[leftIdx])
					sumG += int(src.Pix[rightIdx+1]) - int(src.Pix[leftIdx+1])
					sumB += int(src.Pix[rightIdx+2]) - int(src.Pix[leftIdx+2])
				}
			}
		}(startY, endY)
	}
	wg.Wait()
}

func boxBlurV(src, dst *image.RGBA, w, h, r int) {
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	colsPerWorker := (w + workers - 1) / workers

	var wg sync.WaitGroup
	for wi := 0; wi < workers; wi++ {
		startX := wi * colsPerWorker
		endX := startX + colsPerWorker
		if endX > w {
			endX = w
		}
		if startX >= endX {
			continue
		}

		wg.Add(1)
		go func(sX, eX int) {
			defer wg.Done()
			for x := sX; x < eX; x++ {
				var sumR, sumG, sumB int
				windowSize := 0

				for y := -r; y <= r; y++ {
					py := clamp(y, 0, h-1)
					idx := py*src.Stride + x*4
					sumR += int(src.Pix[idx])
					sumG += int(src.Pix[idx+1])
					sumB += int(src.Pix[idx+2])
					windowSize++
				}

				for y := 0; y < h; y++ {
					outIdx := y*dst.Stride + x*4
					dst.Pix[outIdx] = uint8(sumR / windowSize)
					dst.Pix[outIdx+1] = uint8(sumG / windowSize)
					dst.Pix[outIdx+2] = uint8(sumB / windowSize)
					dst.Pix[outIdx+3] = 255

					topPy := clamp(y-r, 0, h-1)
					botPy := clamp(y+r+1, 0, h-1)
					topIdx := topPy*src.Stride + x*4
					botIdx := botPy*src.Stride + x*4

					sumR += int(src.Pix[botIdx]) - int(src.Pix[topIdx])
					sumG += int(src.Pix[botIdx+1]) - int(src.Pix[topIdx+1])
					sumB += int(src.Pix[botIdx+2]) - int(src.Pix[topIdx+2])
				}
			}
		}(startX, endX)
	}
	wg.Wait()
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
