package reuse

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	goimagefx "github.com/amarnathdbg101-coder/go/go-imagefx"
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

// AdaptiveBoundaryFloodCutout delegates to goimagefx.AdaptiveBoundaryFloodCutout (pure Go, 0ms latency, offline).
func AdaptiveBoundaryFloodCutout(src image.Image, tolerance float64) (*image.RGBA, bool) {
	return goimagefx.AdaptiveBoundaryFloodCutout(src, tolerance)
}

// FitAndCenterStudioShot delegates to goimagefx.FitAndCenter to create a centered studio square shot.
func FitAndCenterStudioShot(src image.Image, canvasSize int, paddingPct float64, bg color.Color) *image.RGBA {
	return goimagefx.FitAndCenter(src, canvasSize, canvasSize, paddingPct, bg)
}

// ApplyStudioDropShadow delegates to goimagefx.ApplyDropShadow to add soft realistic depth.
func ApplyStudioDropShadow(src image.Image, blurRadius, offsetX, offsetY int, shadowAlpha uint8) *image.RGBA {
	return goimagefx.ApplyDropShadow(src, blurRadius, offsetX, offsetY, shadowAlpha)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
