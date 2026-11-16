package reuse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"shopMe/internal/utils"
)

var cloudflareHTTPClient = &http.Client{
	Timeout: 20 * time.Second,
}

// GetCloudflareAccountID returns configured Cloudflare account id,
// or automatically extracts it from the R2 storage endpoint.
func GetCloudflareAccountID() string {
	if id := strings.TrimSpace(os.Getenv("CLOUDFLARE_ACCOUNT_ID")); id != "" {
		return id
	}
	cfg := utils.MustLoad()
	ep := strings.TrimPrefix(cfg.Endpoint, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	parts := strings.Split(ep, ".")
	if len(parts) > 0 && len(parts[0]) >= 16 {
		return parts[0]
	}
	return ""
}

// GetCloudflareAPIToken returns the Cloudflare API token for Workers AI access.
func GetCloudflareAPIToken() string {
	if t := strings.TrimSpace(os.Getenv("CLOUDFLARE_API_TOKEN")); t != "" {
		return t
	}
	if t := strings.TrimSpace(os.Getenv("CLOUDFLARE_AI_TOKEN")); t != "" {
		return t
	}
	return ""
}

// RemoveBackgroundCloudflareAI sends an image to Cloudflare Workers AI (@cf/bria/rmbg-2.0)
// and returns the background-removed transparent PNG bytes.
// If Cloudflare credentials are not configured, it returns an informative error.
func RemoveBackgroundCloudflareAI(ctx context.Context, imgBytes []byte) ([]byte, error) {
	accountID := GetCloudflareAccountID()
	apiToken := GetCloudflareAPIToken()

	if accountID == "" || apiToken == "" {
		return nil, fmt.Errorf("cloudflare AI credentials not configured (set CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN in .env)")
	}

	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/run/@cf/bria/rmbg-2.0", accountID)

	// Cloudflare Workers AI expects a JSON object with uint8 array: {"image": [ ... ]}
	intArr := make([]int, len(imgBytes))
	for i, b := range imgBytes {
		intArr[i] = int(b)
	}
	payloadBytes, err := json.Marshal(map[string]interface{}{
		"image": intArr,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cloudflare ai payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudflare ai request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "image/png, application/json, application/octet-stream")

	resp, err := cloudflareHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare ai connection failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read cloudflare ai response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloudflare ai error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	// Case 1: Direct binary PNG stream
	if len(respBytes) >= 8 && bytes.Equal(respBytes[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		return respBytes, nil
	}

	// Case 2: Cloudflare JSON envelope
	// e.g. {"result": [137, 80, ...]} or {"result": {"image": "..."}}
	var jsonResp struct {
		Success bool            `json:"success"`
		Result  json.RawMessage `json:"result"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBytes, &jsonResp); err == nil {
		if len(jsonResp.Errors) > 0 {
			return nil, fmt.Errorf("cloudflare ai api error: %s", jsonResp.Errors[0].Message)
		}

		// Subcase 2a: result is an array of byte integers [137, 80, ...]
		var byteArr []uint8
		if err := json.Unmarshal(jsonResp.Result, &byteArr); err == nil && len(byteArr) > 0 {
			return byteArr, nil
		}

		// Subcase 2b: result is an object {"image": "..."}
		var imgObj struct {
			Image string `json:"image"`
		}
		if err := json.Unmarshal(jsonResp.Result, &imgObj); err == nil && imgObj.Image != "" {
			if strings.HasPrefix(imgObj.Image, "http://") || strings.HasPrefix(imgObj.Image, "https://") {
				// Download temporary cutout from Cloudflare URL
				getReq, getErr := http.NewRequestWithContext(ctx, http.MethodGet, imgObj.Image, nil)
				if getErr == nil {
					getResp, doErr := cloudflareHTTPClient.Do(getReq)
					if doErr == nil {
						defer getResp.Body.Close()
						cutoutBytes, readErr := io.ReadAll(getResp.Body)
						if readErr == nil && len(cutoutBytes) >= 8 {
							return cutoutBytes, nil
						}
					}
				}
			} else {
				// Base64 string
				clean := imgObj.Image
				if idx := strings.Index(clean, ","); idx != -1 {
					clean = clean[idx+1:]
				}
				if decoded, decErr := base64.StdEncoding.DecodeString(clean); decErr == nil {
					return decoded, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("cloudflare ai returned unrecognized payload: %s", string(respBytes[:minInt(len(respBytes), 200)]))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
