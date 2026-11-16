// Package gogemini provides a zero-dependency Google Gemini REST SDK featuring
// multi-key round-robin rotation, 429 rate-limit cooldown quarantine, multi-model fallback,
// function calling tools, vision OCR, and structured JSON extraction.
package gogemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Client is the primary interface for interacting with Google Gemini models.
type Client struct {
	cfg        Config
	rotator    *KeyRotator
	httpClient *http.Client
	models     []string
}

// New constructs an initialized Gemini Client.
func New(cfg Config) (*Client, error) {
	rotator, err := NewKeyRotator(cfg.APIKeys, cfg.CooldownDuration)
	if err != nil {
		return nil, err
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	primary := cfg.DefaultModel
	if primary == "" {
		primary = ModelGemini20Flash
	}

	models := []string{primary}
	for _, fb := range cfg.FallbackModels {
		if fb != "" && fb != primary {
			models = append(models, fb)
		}
	}

	return &Client{
		cfg:        cfg,
		rotator:    rotator,
		httpClient: httpClient,
		models:     models,
	}, nil
}

// Part represents a component of a Gemini content turn (text, inline media, function call, or response).
type Part struct {
	Text             string            `json:"text,omitempty"`
	InlineData       *InlineData       `json:"inline_data,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

// InlineData carries base64-encoded media for Gemini vision models.
type InlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

// Content defines a single prompt or response turn.
type Content struct {
	Role  string `json:"role,omitempty"` // "user", "model", "function"
	Parts []Part `json:"parts"`
}

// GenerationConfig configures temperature and token limits.
type GenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
}

type geminiPayload struct {
	SystemInstruction *SystemInstruction `json:"system_instruction,omitempty"`
	Contents          []Content          `json:"contents"`
	Tools             []Tool             `json:"tools,omitempty"`
	GenerationConfig  *GenerationConfig  `json:"generationConfig,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Role  string `json:"role"`
			Parts []struct {
				Text         string        `json:"text"`
				FunctionCall *FunctionCall `json:"functionCall,omitempty"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// GenerateText sends a plain text prompt through multi-key rotation and multi-model fallback.
func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", ErrEmptyPrompt
	}

	req := ChatRequest{
		Contents: []Content{
			{
				Role:  "user",
				Parts: []Part{{Text: prompt}},
			},
		},
	}
	resp, err := c.Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// GenerateWithVision sends a text prompt along with raw image bytes (JPEG, PNG, WebP) to Gemini Vision.
func (c *Client) GenerateWithVision(ctx context.Context, prompt string, imgBytes []byte, mimeType string) (string, error) {
	if prompt == "" {
		return "", ErrEmptyPrompt
	}
	if len(imgBytes) == 0 {
		return "", ErrEmptyPayload
	}
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	b64Image := base64.StdEncoding.EncodeToString(imgBytes)

	req := ChatRequest{
		Contents: []Content{
			{
				Role: "user",
				Parts: []Part{
					{Text: prompt},
					{
						InlineData: &InlineData{
							MimeType: mimeType,
							Data:     b64Image,
						},
					},
				},
			},
		},
	}
	resp, err := c.Chat(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// Chat executes a structured or tool-augmented chat request through key rotation and model fallback.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if len(req.Contents) == 0 {
		return nil, ErrEmptyPrompt
	}

	payload := geminiPayload{
		Contents: req.Contents,
	}

	if req.SystemInstruction != "" {
		payload.SystemInstruction = &SystemInstruction{
			Parts: []Part{{Text: req.SystemInstruction}},
		}
	}

	if len(req.Tools) > 0 {
		payload.Tools = req.Tools
	}

	temp := req.Temperature
	if temp == nil {
		temp = c.cfg.Temperature
	}
	maxTokens := req.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = c.cfg.MaxOutputTokens
	}

	if temp != nil || maxTokens > 0 {
		payload.GenerationConfig = &GenerationConfig{
			Temperature:     temp,
			MaxOutputTokens: maxTokens,
		}
	}

	return c.executeChatWithFallback(ctx, payload)
}

// GenerateJSON executes a prompt and automatically deserializes the output into dest.
// Strips markdown ```json ... ``` code fences automatically before unmarshaling.
func GenerateJSON(ctx context.Context, c *Client, prompt string, dest any) error {
	raw, err := c.GenerateText(ctx, prompt)
	if err != nil {
		return err
	}

	cleanJSON := ExtractJSON(raw)
	if err := json.Unmarshal([]byte(cleanJSON), dest); err != nil {
		return fmt.Errorf("%w: %v (raw text: %q)", ErrJSONExtractionFailed, err, raw)
	}

	return nil
}

var jsonBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")

// ExtractJSON removes markdown formatting code fences from model outputs.
func ExtractJSON(input string) string {
	input = strings.TrimSpace(input)
	matches := jsonBlockRegex.FindStringSubmatch(input)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return input
}

func (c *Client) executeChatWithFallback(ctx context.Context, payload geminiPayload) (*ChatResponse, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("go-gemini: marshal payload: %w", err)
	}

	var lastErr error

	// Try each model in fallback order
	for _, model := range c.models {
		key, keyErr := c.rotator.GetKey()
		if keyErr != nil {
			return nil, keyErr
		}

		result, err := c.doPostChat(ctx, model, key, bodyBytes)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// If 429 rate limited, mark key in cooldown
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			c.rotator.MarkRateLimited(key)
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrModelFailed, lastErr)
	}
	return nil, ErrModelFailed
}

func (c *Client) doPostChat(ctx context.Context, model, key string, body []byte) (*ChatResponse, error) {
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("go-gemini: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("go-gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("go-gemini: read response: %w", err)
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("go-gemini: unmarshal response: %w (status %d)", err, resp.StatusCode)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("gemini api error (status %d %s): %s", geminiResp.Error.Code, geminiResp.Error.Status, geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("go-gemini: empty candidate response from model %s", model)
	}

	var texts []string
	var funcCalls []FunctionCall

	for _, part := range geminiResp.Candidates[0].Content.Parts {
		if strings.TrimSpace(part.Text) != "" {
			texts = append(texts, part.Text)
		}
		if part.FunctionCall != nil && part.FunctionCall.Name != "" {
			funcCalls = append(funcCalls, *part.FunctionCall)
		}
	}

	return &ChatResponse{
		Text:          strings.Join(texts, "\n"),
		FunctionCalls: funcCalls,
	}, nil
}
