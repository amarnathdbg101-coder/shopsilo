// Package reuse provides reusable application helpers and utilities.
package reuse

import (
	"context"
	"fmt"
	"os"
	"strings"

	goclos "github.com/amarnathdbg101-coder/go/go-cloudflare"
	"shopMe/internal/utils"
)

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

// RemoveBackgroundCloudflareAI delegates background removal to the zero-dependency go-cloudflare SDK.
func RemoveBackgroundCloudflareAI(ctx context.Context, imgBytes []byte) ([]byte, error) {
	accountID := GetCloudflareAccountID()
	apiToken := GetCloudflareAPIToken()

	if accountID == "" || apiToken == "" {
		return nil, fmt.Errorf("cloudflare AI credentials not configured (set CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN in .env)")
	}

	cfClient, err := goclos.New(goclos.Config{
		AccountID: accountID,
		APIToken:  apiToken,
	})
	if err != nil {
		return nil, err
	}

	return cfClient.AI.RemoveBackground(ctx, imgBytes)
}
