package reuse

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
)

const (
	// MaxDimension defines the maximum width or height of an uploaded image.
	// 1600px is standard HD quality for e-commerce product photos and avatars.
	MaxDimension = 1600

	// JPEGCompressionQuality defines the JPEG compression quality (1-100).
	// 80 provides 65-85% file size reduction with virtually no visible degradation.
	JPEGCompressionQuality = 80
)

// CompressImage reads an image stream, resizes it if it exceeds MaxDimension,
// compresses it using optimal encoding, and returns the compressed bytes and resulting MIME type.
// If decoding fails (e.g. unsupported format), it returns the original bytes untouched without error.
func CompressImage(r io.Reader, contentType string) ([]byte, string, error) {
	origBytes, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}

	cleanType := strings.ToLower(strings.TrimSpace(contentType))

	// Attempt to decode the image
	img, format, err := image.Decode(bytes.NewReader(origBytes))
	if err != nil {
		// Fallback: If decode fails, return original data safely
		return origBytes, cleanType, nil
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 1. Calculate target dimensions if larger than MaxDimension
	targetWidth := width
	targetHeight := height

	if width > MaxDimension || height > MaxDimension {
		if width > height {
			targetWidth = MaxDimension
			targetHeight = int(float64(height) * (float64(MaxDimension) / float64(width)))
		} else {
			targetHeight = MaxDimension
			targetWidth = int(float64(width) * (float64(MaxDimension) / float64(height)))
		}
		if targetWidth < 1 {
			targetWidth = 1
		}
		if targetHeight < 1 {
			targetHeight = 1
		}

		// Resize using bilinear interpolation
		resized := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
		bilinearScale(resized, img)
		img = resized
	}

	var buf bytes.Buffer
	outContentType := cleanType

	// 2. Encode with optimal compression based on format
	switch {
	case format == "png":
		// If PNG has no transparency, encode to JPEG for massive size reduction (often 80-90% smaller)
		if isOpaque(img) {
			err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGCompressionQuality})
			outContentType = "image/jpeg"
		} else {
			enc := png.Encoder{CompressionLevel: png.BestCompression}
			err = enc.Encode(&buf, img)
			outContentType = "image/png"
		}
	default:
		// Default to high-quality compressed JPEG (for jpeg, webp, gif, etc.)
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGCompressionQuality})
		outContentType = "image/jpeg"
	}

	if err != nil {
		// Fallback to original bytes if encoding encounters an unexpected error
		return origBytes, cleanType, nil
	}

	compressedBytes := buf.Bytes()

	// If compressed version happens to be larger than original, preserve the smaller original
	if len(compressedBytes) >= len(origBytes) {
		return origBytes, cleanType, nil
	}

	return compressedBytes, outContentType, nil
}

// isOpaque checks whether the entire image is opaque (no alpha transparency)
func isOpaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xffff {
				return false
			}
		}
	}
	return true
}

// bilinearScale performs fast, high-quality bilinear scaling from src into dst
func bilinearScale(dst *image.RGBA, src image.Image) {
	dstBounds := dst.Bounds()
	srcBounds := src.Bounds()

	dx := float64(srcBounds.Dx()) / float64(dstBounds.Dx())
	dy := float64(srcBounds.Dy()) / float64(dstBounds.Dy())

	// If the difference is simple, draw.ApproxBiLinear or manual bilinear
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
	_ = draw.Over // keep import
}
