package goclos

import (
	"net/http"
	"time"
)

// Config encapsulates Cloudflare credentials and options for Workers AI and R2 storage.
type Config struct {
	AccountID       string        // Cloudflare 32-character hexadecimal Account ID
	APIToken        string        // Cloudflare API Token with Workers AI and R2 permissions
	R2AccessKeyID   string        // Optional: S3-compatible R2 Access Key ID
	R2SecretAccess  string        // Optional: S3-compatible R2 Secret Access Key
	R2Bucket        string        // Default R2 bucket name
	R2PublicURL     string        // Public CDN or custom domain URL for asset serving (e.g. "https://assets.example.com")
	HTTPClient      *http.Client  // Custom HTTP client (defaults to client with 30s timeout)
	Timeout         time.Duration // Default timeout per request
}

// DefaultModels provides quick constants for popular Cloudflare Workers AI models.
const (
	ModelLlama38BInstruct = "@cf/meta/llama-3-8b-instruct"
	ModelLlama318B        = "@cf/meta/llama-3.1-8b-instruct"
	ModelMistral7B        = "@cf/mistral/7b-instruct-v0.1"
	ModelSDXLLightning    = "@cf/bytedance/stable-diffusion-xl-lightning"
	ModelRMBG14           = "@cf/briaai/rmbg-1.4"
	ModelRMBG20           = "@cf/bria/rmbg-2.0"
	ModelBGEBase          = "@cf/baai/bge-base-en-v1.5"
	ModelWhisper          = "@cf/openai/whisper"
)
