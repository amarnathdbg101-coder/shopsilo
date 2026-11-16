package gogemini

import (
	"net/http"
	"time"
)

// Default model constants
const (
	ModelGemini20Flash = "gemini-2.0-flash"
	ModelGemini15Flash = "gemini-1.5-flash"
	ModelGemini15Pro   = "gemini-1.5-pro"
)

// Config configures multi-key rotation and multi-model fallback for Gemini.
type Config struct {
	APIKeys          []string      // List of Google Gemini API keys for round-robin rotation
	DefaultModel     string        // Primary model (default: "gemini-2.0-flash")
	FallbackModels   []string      // Backup models if primary encounters quota or errors
	CooldownDuration time.Duration // Cooldown period after receiving 429 Too Many Requests (default: 60s)
	HTTPClient       *http.Client  // Custom HTTP client (default: 30s timeout)
	Temperature      *float64      // Generation temperature (0.0 to 2.0)
	MaxOutputTokens  int           // Maximum generation tokens
}
