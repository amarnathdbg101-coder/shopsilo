package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultGeminiTimeout          = 35 * time.Second
	defaultGeminiMaxResponseBytes = 4 << 20 // 4 MiB
)

// CallOptions configures a single Gemini generateContent request with key rotation and model fallback.
type CallOptions struct {
	ExplicitKey            string
	Timeout                time.Duration
	MaxResponseBytes       int64
	AllowExhaustedFallback bool
}

// GeminiClient centralizes Gemini HTTP transport, API key rotation, rate-limit cooldowns,
// and multi-model fallback across services.
type GeminiClient struct {
	httpClient *http.Client
	rotator    *KeyRotator
}

// NewGeminiClient creates a GeminiClient using the provided HTTP client and key rotator.
// If httpClient is nil, a pooled default client with a 45s timeout is created.
func NewGeminiClient(httpClient *http.Client, rotator *KeyRotator) *GeminiClient {
	if httpClient == nil {
		httpClient = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 25,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: 45 * time.Second,
		}
	}
	return &GeminiClient{
		httpClient: httpClient,
		rotator:    rotator,
	}
}

// Rotator returns the underlying KeyRotator for key availability checks.
func (c *GeminiClient) Rotator() *KeyRotator {
	return c.rotator
}

// DefaultModels returns the ordered Gemini model fallback list, prepending GEMINI_MODEL if configured.
func DefaultModels() []string {
	custom := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	models := []string{
		"gemini-flash-lite-latest",
		"gemini-flash-latest",
		"gemini-2.5-flash",
		"gemini-2.0-flash",
		"gemini-1.5-flash",
	}
	if custom != "" && custom != "gemini-3.1-pro-preview" && custom != "gemini-2.5-flash" {
		return append([]string{custom}, models...)
	}
	return models
}

// GenerateContent marshals payload and invokes Gemini's generateContent endpoint across rotated keys
// and fallback models until parseResponse succeeds.
func (c *GeminiClient) GenerateContent(
	ctx context.Context,
	models []string,
	payload any,
	opts CallOptions,
	parseResponse func(body []byte) error,
) error {
	if c.rotator == nil || (!c.rotator.HasKeys() && strings.TrimSpace(opts.ExplicitKey) == "") {
		return ErrNoKeysConfigured
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal gemini payload: %w", err)
	}

	if len(models) == 0 {
		models = DefaultModels()
	}

	keyAttempts := c.rotator.KeyCount()
	if keyAttempts <= 0 {
		keyAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < keyAttempts; attempt++ {
		apiKey := strings.TrimSpace(opts.ExplicitKey)
		if apiKey == "" || attempt > 0 {
			k, keyErr := c.rotator.GetNextKey()
			if keyErr != nil && (!opts.AllowExhaustedFallback || !errors.Is(keyErr, ErrAllKeysExhausted)) {
				return keyErr
			}
			apiKey = k
		}
		if apiKey == "" {
			return ErrNoKeysConfigured
		}

		for _, model := range models {
			respBody, statusCode, callErr := c.callSingleModel(ctx, apiKey, model, bodyBytes, opts)
			if callErr == nil {
				if parseErr := parseResponse(respBody); parseErr == nil {
					c.rotator.MarkSuccess(apiKey)
					return nil
				} else {
					lastErr = parseErr
					continue
				}
			}

			lastErr = callErr
			if statusCode == http.StatusTooManyRequests {
				c.rotator.MarkRateLimited(apiKey, time.Minute)
				break
			}
			if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
				c.rotator.MarkRateLimited(apiKey, 10*time.Minute)
				break
			}
		}
	}

	return lastErr
}

func (c *GeminiClient) callSingleModel(
	ctx context.Context,
	apiKey, model string,
	bodyBytes []byte,
	opts CallOptions,
) ([]byte, int, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultGeminiTimeout
	}
	maxBytes := opts.MaxResponseBytes
	if maxBytes <= 0 {
		maxBytes = defaultGeminiMaxResponseBytes
	}

	modelCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	req, err := http.NewRequestWithContext(modelCtx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("gemini api request failed for model %s: %w", model, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read gemini response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("gemini api error (model %s, status %d): %s", model, resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}
