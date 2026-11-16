package goimagefx

import (
	"image"
	"image/draw"
	"math"
	"sort"
)

// AdaptiveBoundaryFloodCutout identifies the dominant background color along the image perimeter
// and floods contiguous background pixels to Alpha = 0 (100% transparent), creating a clean cutout.
// It returns the cutout image and true if at least 5% of pixels were recognized and cleared.
func AdaptiveBoundaryFloodCutout(src image.Image, tolerance float64) (*image.RGBA, bool) {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 16 || h < 16 {
		return nil, false
	}

	if tolerance <= 0 {
		tolerance = 42.0 // Default Euclidean color threshold in 0-255 RGB space
	}

	// 1. Copy src to image.RGBA
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), src, bounds.Min, draw.Src)

	// 2. Sample perimeter pixels to find the dominant background color
	type rgbSample struct {
		r, g, b float64
	}
	var samples []rgbSample
	step := maxInt(1, (w+h)/80)

	for x := 0; x < w; x += step {
		r, g, b, _ := rgba.At(x, 0).RGBA()
		samples = append(samples, rgbSample{float64(r >> 8), float64(g >> 8), float64(b >> 8)})
		r, g, b, _ = rgba.At(x, h-1).RGBA()
		samples = append(samples, rgbSample{float64(r >> 8), float64(g >> 8), float64(b >> 8)})
	}
	for y := 0; y < h; y += step {
		r, g, b, _ := rgba.At(0, y).RGBA()
		samples = append(samples, rgbSample{float64(r >> 8), float64(g >> 8), float64(b >> 8)})
		r, g, b, _ = rgba.At(w-1, y).RGBA()
		samples = append(samples, rgbSample{float64(r >> 8), float64(g >> 8), float64(b >> 8)})
	}

	if len(samples) == 0 {
		return nil, false
	}

	rVals := make([]float64, len(samples))
	gVals := make([]float64, len(samples))
	bVals := make([]float64, len(samples))
	for i, s := range samples {
		rVals[i] = s.r
		gVals[i] = s.g
		bVals[i] = s.b
	}
	sort.Float64s(rVals)
	sort.Float64s(gVals)
	sort.Float64s(bVals)

	medianR := rVals[len(rVals)/2]
	medianG := gVals[len(gVals)/2]
	medianB := bVals[len(bVals)/2]

	colorDist := func(r, g, b uint8) float64 {
		dr := float64(r) - medianR
		dg := float64(g) - medianG
		db := float64(b) - medianB
		return math.Sqrt(dr*dr + dg*dg + db*db)
	}

	// 3. Multi-source BFS Flood Fill from outer boundary
	visited := make([]bool, w*h)
	featherZone := 14.0 // Soft transition zone
	maxDist := tolerance + featherZone

	type point struct {
		x, y int
	}
	queue := make([]point, 0, (w+h)*4)

	enqueueIfMatch := func(x, y int) {
		idx := y*w + x
		if visited[idx] {
			return
		}
		offset := (y * rgba.Stride) + (x * 4)
		r := rgba.Pix[offset]
		g := rgba.Pix[offset+1]
		b := rgba.Pix[offset+2]
		d := colorDist(r, g, b)
		if d <= maxDist {
			visited[idx] = true
			queue = append(queue, point{x, y})
		}
	}

	// Seed perimeter pixels
	for x := 0; x < w; x++ {
		enqueueIfMatch(x, 0)
		enqueueIfMatch(x, h-1)
	}
	for y := 0; y < h; y++ {
		enqueueIfMatch(0, y)
		enqueueIfMatch(w-1, y)
	}

	transparentCount := 0

	head := 0
	for head < len(queue) {
		p := queue[head]
		head++

		offset := (p.y * rgba.Stride) + (p.x * 4)
		r := rgba.Pix[offset]
		g := rgba.Pix[offset+1]
		b := rgba.Pix[offset+2]
		d := colorDist(r, g, b)

		if d <= tolerance {
			rgba.Pix[offset+3] = 0 // Fully transparent
			transparentCount++
		} else {
			// Smooth edge feathering
			fade := (d - tolerance) / featherZone
			if fade > 1.0 {
				fade = 1.0
			}
			newAlpha := uint8(fade * 255.0)
			if newAlpha < rgba.Pix[offset+3] {
				rgba.Pix[offset+3] = newAlpha
			}
		}

		neighbors := [4]point{
			{p.x + 1, p.y},
			{p.x - 1, p.y},
			{p.x, p.y + 1},
			{p.x, p.y - 1},
		}

		for _, nb := range neighbors {
			if nb.x >= 0 && nb.x < w && nb.y >= 0 && nb.y < h {
				nbIdx := nb.y*w + nb.x
				if !visited[nbIdx] {
					nbOffset := (nb.y * rgba.Stride) + (nb.x * 4)
					nbr := rgba.Pix[nbOffset]
					nbg := rgba.Pix[nbOffset+1]
					nbb := rgba.Pix[nbOffset+2]
					nbd := colorDist(nbr, nbg, nbb)
					if nbd <= maxDist {
						visited[nbIdx] = true
						queue = append(queue, nb)
					}
				}
			}
		}
	}

	// If at least 5% of pixels were recognized and cleared as background
	if transparentCount >= (w * h * 5 / 100) {
		return rgba, true
	}

	return nil, false
}

// TrimTransparency identifies the minimum bounding rectangle containing non-transparent pixels
// and returns a cropped sub-image.
func TrimTransparency(src image.Image, alphaThreshold uint8) (image.Image, image.Rectangle) {
	bounds := src.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y

	hasVisible := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			alpha := uint8(a >> 8)
			if alpha >= alphaThreshold {
				hasVisible = true
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

	if !hasVisible {
		return src, bounds
	}

	cropRect := image.Rect(minX, minY, maxX+1, maxY+1)

	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	if si, ok := src.(subImager); ok {
		return si.SubImage(cropRect), cropRect
	}

	// Fallback: draw onto new RGBA
	dst := image.NewRGBA(image.Rect(0, 0, cropRect.Dx(), cropRect.Dy()))
	draw.Draw(dst, dst.Bounds(), src, cropRect.Min, draw.Src)
	return dst, cropRect
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
