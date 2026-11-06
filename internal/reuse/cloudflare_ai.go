package reuse

import (
	"bytes"
	"context"
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
// If Cloudflare credentials are not configured, it returns ErrCloudflareNotConfigured.
func RemoveBackgroundCloudflareAI(ctx context.Context, imgBytes []byte) ([]byte, error) {
	accountID := GetCloudflareAccountID()
	apiToken := GetCloudflareAPIToken()

	if accountID == "" || apiToken == "" {
		return nil, fmt.Errorf("cloudflare AI credentials not configured (set CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN in .env)")
	}

	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/run/@cf/bria/rmbg-2.0", accountID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudflare ai request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/octet-stream")

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

	// Verify we received a PNG image
	if len(respBytes) < 8 || !bytes.Equal(respBytes[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		// Sometimes Cloudflare wraps response in JSON if an error occurred
		return nil, fmt.Errorf("cloudflare ai returned invalid PNG payload: %s", string(respBytes[:minInt(len(respBytes), 200)]))
	}

	return respBytes, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
