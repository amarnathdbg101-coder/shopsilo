package imghash

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math/bits"
	"strconv"
)

// DHash implements a 64-bit gradient Difference Hash algorithm.
type DHash struct{}

// NewDHash creates a new DHash algorithm instance.
func NewDHash() *DHash {
	return &DHash{}
}

// ComputeFromImage generates a 64-bit difference hash from decoded image.
// High-performance path: fast-paths *image.RGBA, *image.NRGBA, and *image.Gray
// to eliminate interface boxing, achieving 0 heap allocations on image inspection.
func (d *DHash) ComputeFromImage(img image.Image) (uint64, error) {
	if img == nil {
		return 0, ErrInvalidImage
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return 0, ErrInvalidImage
	}

	// Downscale into 9x8 matrix (9 columns, 8 rows = 72 sample points)
	const targetWidth = 9
	const targetHeight = 8

	var grayMatrix [targetHeight][targetWidth]uint8

	switch typed := img.(type) {
	case *image.RGBA:
		for y := 0; y < targetHeight; y++ {
			srcY := bounds.Min.Y + (y * height / targetHeight)
			for x := 0; x < targetWidth; x++ {
				srcX := bounds.Min.X + (x * width / targetWidth)
				offset := typed.PixOffset(srcX, srcY)
				r := uint32(typed.Pix[offset])
				g := uint32(typed.Pix[offset+1])
				b := uint32(typed.Pix[offset+2])
				grayMatrix[y][x] = uint8((299*r + 587*g + 114*b) / 1000)
			}
		}
	case *image.NRGBA:
		for y := 0; y < targetHeight; y++ {
			srcY := bounds.Min.Y + (y * height / targetHeight)
			for x := 0; x < targetWidth; x++ {
				srcX := bounds.Min.X + (x * width / targetWidth)
				offset := typed.PixOffset(srcX, srcY)
				r := uint32(typed.Pix[offset])
				g := uint32(typed.Pix[offset+1])
				b := uint32(typed.Pix[offset+2])
				grayMatrix[y][x] = uint8((299*r + 587*g + 114*b) / 1000)
			}
		}
	case *image.Gray:
		for y := 0; y < targetHeight; y++ {
			srcY := bounds.Min.Y + (y * height / targetHeight)
			for x := 0; x < targetWidth; x++ {
				srcX := bounds.Min.X + (x * width / targetWidth)
				offset := typed.PixOffset(srcX, srcY)
				grayMatrix[y][x] = typed.Pix[offset]
			}
		}
	default:
		// Fallback for custom image.Image types (avoids GrayModel.Convert allocations)
		for y := 0; y < targetHeight; y++ {
			srcY := bounds.Min.Y + (y * height / targetHeight)
			for x := 0; x < targetWidth; x++ {
				srcX := bounds.Min.X + (x * width / targetWidth)
				r, g, b, _ := img.At(srcX, srcY).RGBA()
				grayMatrix[y][x] = uint8((299*(r>>8) + 587*(g>>8) + 114*(b>>8)) / 1000)
			}
		}
	}

	// Compute 64-bit difference bit by bit
	// Each row (8 rows) has 8 adjacent horizontal comparisons (left > right)
	var hash uint64
	bitIndex := 0

	for y := 0; y < targetHeight; y++ {
		for x := 0; x < targetWidth-1; x++ {
			if grayMatrix[y][x] > grayMatrix[y][x+1] {
				hash |= 1 << (63 - bitIndex)
			}
			bitIndex++
		}
	}

	return hash, nil
}

// ComputeFromReader decodes an image from stream and computes its 64-bit hash.
func (d *DHash) ComputeFromReader(r io.Reader) (uint64, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	return d.ComputeFromImage(img)
}

// HammingDistance calculates the number of differing bits between two 64-bit hashes.
func HammingDistance(hash1, hash2 uint64) int {
	return bits.OnesCount64(hash1 ^ hash2)
}

// HammingDistanceHex calculates distance from two 16-character hex strings.
func HammingDistanceHex(hex1, hex2 string) (int, error) {
	val1, err := strconv.ParseUint(hex1, 16, 64)
	if err != nil {
		return -1, ErrInvalidHashFormat
	}
	val2, err := strconv.ParseUint(hex2, 16, 64)
	if err != nil {
		return -1, ErrInvalidHashFormat
	}
	return HammingDistance(val1, val2), nil
}
