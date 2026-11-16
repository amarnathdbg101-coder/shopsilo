package reuse

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

var bgRemovalHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// BackgroundRemovalProvider identifies which engine was used to remove the background.
type BackgroundRemovalProvider string

const (
	ProviderHFRmbg2    BackgroundRemovalProvider = "HuggingFace-RMBG-2.0"
	ProviderHFRmbg1    BackgroundRemovalProvider = "HuggingFace-RMBG-1.4"
	ProviderCloudflare BackgroundRemovalProvider = "Cloudflare-Workers-AI"
	ProviderRemoveBg   BackgroundRemovalProvider = "Remove.bg"
	ProviderLocalChroma BackgroundRemovalProvider = "Local-Adaptive-Chroma"
	ProviderOriginal   BackgroundRemovalProvider = "Original-Passthrough"
)

// GetHuggingFaceToken returns configured Hugging Face API token for free serverless inference.
func GetHuggingFaceToken() string {
	if t := strings.TrimSpace(os.Getenv("HUGGINGFACE_API_TOKEN")); t != "" {
		return t
	}
	if t := strings.TrimSpace(os.Getenv("HF_TOKEN")); t != "" {
		return t
	}
	return ""
}

// RemoveBackground intelligently removes the background using a multi-engine cascade:
// 1. Hugging Face Serverless Inference (briaai/RMBG-2.0 / RMBG-1.4) - ₹0, Serverless GPU
// 2. Remove.bg API (if REMOVE_BG_API_KEY is configured)
// 3. Cloudflare Workers AI (if custom model/endpoint configured)
// 4. Local Adaptive Boundary Flood Cutout (Pure Go, 0ms latency, ₹0, works offline)
// If all fail, it gracefully returns the original image.
func RemoveBackground(ctx context.Context, imgBytes []byte) ([]byte, BackgroundRemovalProvider, error) {
	if len(imgBytes) == 0 {
		return nil, ProviderOriginal, fmt.Errorf("empty image bytes")
	}

	// 1. Try Hugging Face Serverless Inference (Free Bria AI RMBG)
	if hfToken := GetHuggingFaceToken(); hfToken != "" {
		cutout, err := removeBackgroundHuggingFace(ctx, imgBytes, hfToken, "briaai/RMBG-2.0")
		if err == nil && len(cutout) > 0 {
			return cutout, ProviderHFRmbg2, nil
		}
		if err != nil {
			log.Printf("[BackgroundRemover] HF RMBG-2.0 failed: %v. Retrying with RMBG-1.4...", err)
		}

		cutout1, err1 := removeBackgroundHuggingFace(ctx, imgBytes, hfToken, "briaai/RMBG-1.4")
		if err1 == nil && len(cutout1) > 0 {
			return cutout1, ProviderHFRmbg1, nil
		}
		if err1 != nil {
			log.Printf("[BackgroundRemover] HF RMBG-1.4 notice: %v", err1)
		}
	}

	// 2. Try Remove.bg API if key is present
	if rbgKey := strings.TrimSpace(os.Getenv("REMOVE_BG_API_KEY")); rbgKey != "" {
		cutout, err := removeBackgroundRemoveBg(ctx, imgBytes, rbgKey)
		if err == nil && len(cutout) > 0 {
			return cutout, ProviderRemoveBg, nil
		}
		log.Printf("[BackgroundRemover] Remove.bg notice: %v", err)
	}

	// 3. Try Cloudflare Workers AI (if user configured custom AI model)
	if cfToken := GetCloudflareAPIToken(); cfToken != "" {
		cutout, err := RemoveBackgroundCloudflareAI(ctx, imgBytes)
		if err == nil && len(cutout) > 0 {
			return cutout, ProviderCloudflare, nil
		}
		// Expected to fail if model @cf/bria/rmbg-2.0 is not in Cloudflare's catalog
	}

	// 4. Pure Go Local Adaptive Boundary Flood Cutout (₹0, instant, offline)
	srcImg, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err == nil {
		cutoutRGBA, ok := AdaptiveBoundaryFloodCutout(srcImg, 42.0)
		if ok && cutoutRGBA != nil {
			var buf bytes.Buffer
			if encErr := png.Encode(&buf, cutoutRGBA); encErr == nil {
				return buf.Bytes(), ProviderLocalChroma, nil
			}
		}
	}

	// 5. Graceful Passthrough: Return original bytes so compositor can still enhance lighting
	return imgBytes, ProviderOriginal, nil
}

// removeBackgroundHuggingFace calls Hugging Face Serverless Inference API for RMBG
func removeBackgroundHuggingFace(ctx context.Context, imgBytes []byte, token string, model string) ([]byte, error) {
	endpoint := fmt.Sprintf("https://router.huggingface.co/hf-inference/models/%s", model)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HF request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Accept", "image/png")

	resp, err := bgRemovalHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HF connection failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HF response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HF error (status %d): %s", resp.StatusCode, string(bodyBytes[:minInt(len(bodyBytes), 180)]))
	}

	// Verify PNG magic bytes
	if len(bodyBytes) >= 8 && bytes.Equal(bodyBytes[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		return bodyBytes, nil
	}

	return nil, fmt.Errorf("HF did not return a valid PNG image")
}

// removeBackgroundRemoveBg calls remove.bg API
func removeBackgroundRemoveBg(ctx context.Context, imgBytes []byte, apiKey string) ([]byte, error) {
	endpoint := "https://api.remove.bg/v1.0/removebg"

	var body bytes.Buffer
	writer := io.MultiWriter(&body)
	// We send as binary payload in POST
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(imgBytes))
	if err != nil {
		return nil, err
	}
	_ = writer
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := bgRemovalHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remove.bg returned status %d", resp.StatusCode)
	}
	return respBytes, nil
}

// AdaptiveBoundaryFloodCutout identifies the dominant background color along the image perimeter
// and floods contiguous background pixels to Alpha = 0 (100% transparent), creating a clean cutout.
// It returns the cutout image and true if at least 6% of pixels were made transparent.
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
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			rgba.Set(x, y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}

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

	// Compute median of border samples to be resilient against the product touching one border
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

	// BFS queue traversal
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

		// 4-directional flood expansion
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

	// If at least 6% of pixels were recognized and cleared as background
	if transparentCount >= (w * h * 6 / 100) {
		return rgba, true
	}

	return nil, false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
