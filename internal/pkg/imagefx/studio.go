package goimagefx

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

// ResizeBilinear rescales an image to target dimensions using bilinear interpolation in pure Go.
func ResizeBilinear(src image.Image, targetW, targetH int) *image.RGBA {
	if targetW <= 0 || targetH <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))

	xRatio := float64(srcW-1) / float64(targetW)
	yRatio := float64(srcH-1) / float64(targetH)
	if targetW == 1 {
		xRatio = 0
	}
	if targetH == 1 {
		yRatio = 0
	}

	for y := 0; y < targetH; y++ {
		srcY := float64(y) * yRatio
		yLow := int(math.Floor(srcY))
		yHigh := int(math.Ceil(srcY))
		if yHigh >= srcH {
			yHigh = srcH - 1
		}
		yWeight := srcY - float64(yLow)

		for x := 0; x < targetW; x++ {
			srcX := float64(x) * xRatio
			xLow := int(math.Floor(srcX))
			xHigh := int(math.Ceil(srcX))
			if xHigh >= srcW {
				xHigh = srcW - 1
			}
			xWeight := srcX - float64(xLow)

			c00 := colorToNRGBA(src.At(srcBounds.Min.X+xLow, srcBounds.Min.Y+yLow))
			c10 := colorToNRGBA(src.At(srcBounds.Min.X+xHigh, srcBounds.Min.Y+yLow))
			c01 := colorToNRGBA(src.At(srcBounds.Min.X+xLow, srcBounds.Min.Y+yHigh))
			c11 := colorToNRGBA(src.At(srcBounds.Min.X+xHigh, srcBounds.Min.Y+yHigh))

			interp := func(v00, v10, v01, v11 uint8) uint8 {
				top := float64(v00)*(1.0-xWeight) + float64(v10)*xWeight
				bottom := float64(v01)*(1.0-xWeight) + float64(v11)*xWeight
				return uint8(math.Round(top*(1.0-yWeight) + bottom*yWeight))
			}

			r := interp(c00.R, c10.R, c01.R, c11.R)
			g := interp(c00.G, c10.G, c01.G, c11.G)
			b := interp(c00.B, c10.B, c01.B, c11.B)
			a := interp(c00.A, c10.A, c01.A, c11.A)

			dst.Set(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}

	return dst
}

// FitAndCenter fits an image inside target canvas dimensions preserving aspect ratio with padding.
func FitAndCenter(src image.Image, canvasW, canvasH int, paddingPct float64, bg color.Color) *image.RGBA {
	if canvasW <= 0 {
		canvasW = 800
	}
	if canvasH <= 0 {
		canvasH = 800
	}
	if paddingPct < 0 {
		paddingPct = 0
	} else if paddingPct > 0.40 {
		paddingPct = 0.40 // Max 40% padding
	}

	availW := float64(canvasW) * (1.0 - 2.0*paddingPct)
	availH := float64(canvasH) * (1.0 - 2.0*paddingPct)

	srcBounds := src.Bounds()
	srcW := float64(srcBounds.Dx())
	srcH := float64(srcBounds.Dy())

	scale := math.Min(availW/srcW, availH/srcH)
	if scale > 1.0 {
		// Do not upscale small assets beyond original resolution
		scale = 1.0
	}

	targetW := int(math.Round(srcW * scale))
	targetH := int(math.Round(srcH * scale))
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}

	resized := ResizeBilinear(src, targetW, targetH)

	// Create canvas
	dst := image.NewRGBA(image.Rect(0, 0, canvasW, canvasH))
	if bg != nil {
		draw.Draw(dst, dst.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	}

	// Calculate center offset
	offsetX := (canvasW - targetW) / 2
	offsetY := (canvasH - targetH) / 2

	destRect := image.Rect(offsetX, offsetY, offsetX+targetW, offsetY+targetH)
	draw.Draw(dst, destRect, resized, image.Point{}, draw.Over)

	return dst
}

// ApplyDropShadow renders an e-commerce studio soft drop shadow beneath the object.
func ApplyDropShadow(src image.Image, blurRadius int, offsetX, offsetY int, shadowAlpha uint8) *image.RGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if blurRadius < 1 {
		blurRadius = 8
	}
	if shadowAlpha == 0 {
		shadowAlpha = 110 // ~43% opacity
	}

	padding := blurRadius * 2
	outW := w + padding*2 + int(math.Abs(float64(offsetX)))
	outH := h + padding*2 + int(math.Abs(float64(offsetY)))

	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))

	// 1. Build alpha silhouette mask
	mask := make([][]float64, outH)
	for y := range mask {
		mask[y] = make([]float64, outW)
	}

	originX := padding
	if offsetX < 0 {
		originX += -offsetX
	}
	originY := padding
	if offsetY < 0 {
		originY += -offsetY
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			_, _, _, a := src.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			mask[originY+y+offsetY][originX+x+offsetX] = float64(a>>8) / 255.0
		}
	}

	// 2. Horizontal box blur pass
	temp := make([][]float64, outH)
	for y := range temp {
		temp[y] = make([]float64, outW)
	}

	kernelSize := blurRadius*2 + 1
	kernelWeight := 1.0 / float64(kernelSize)

	for y := 0; y < outH; y++ {
		var sum float64
		for x := 0; x < kernelSize && x < outW; x++ {
			sum += mask[y][x]
		}
		for x := 0; x < outW; x++ {
			temp[y][x] = sum * kernelWeight
			left := x - blurRadius
			right := x + blurRadius + 1
			if left >= 0 {
				sum -= mask[y][left]
			}
			if right < outW {
				sum += mask[y][right]
			}
		}
	}

	// 3. Vertical box blur pass and render shadow
	for x := 0; x < outW; x++ {
		var sum float64
		for y := 0; y < kernelSize && y < outH; y++ {
			sum += temp[y][x]
		}
		for y := 0; y < outH; y++ {
			val := sum * kernelWeight
			if val > 0.01 {
				alpha := uint8(math.Min(255.0, val*float64(shadowAlpha)))
				dst.Set(x, y, color.NRGBA{R: 20, G: 20, B: 25, A: alpha})
			}
			top := y - blurRadius
			bottom := y + blurRadius + 1
			if top >= 0 {
				sum -= temp[top][x]
			}
			if bottom < outH {
				sum += temp[bottom][x]
			}
		}
	}

	// 4. Draw original image on top of shadow
	destRect := image.Rect(originX, originY, originX+w, originY+h)
	draw.Draw(dst, destRect, src, bounds.Min, draw.Over)

	return dst
}

func colorToNRGBA(c color.Color) color.NRGBA {
	if n, ok := c.(color.NRGBA); ok {
		return n
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return color.NRGBA{0, 0, 0, 0}
	}
	// Un-premultiply
	return color.NRGBA{
		R: uint8((r * 0xFF) / a),
		G: uint8((g * 0xFF) / a),
		B: uint8((b * 0xFF) / a),
		A: uint8(a >> 8),
	}
}
