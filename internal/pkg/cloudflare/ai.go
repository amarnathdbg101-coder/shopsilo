package goclos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AIService exposes Cloudflare Workers AI serverless inference.
type AIService struct {
	client *Client
}

// ChatMessage represents a single turn in a conversational exchange.
type ChatMessage struct {
	Role    string `json:"role"`    // "system", "user", or "assistant"
	Content string `json:"content"` // Text content of the prompt
}

// ChatRequest configures an LLM text generation request.
type ChatRequest struct {
	Model       string        `json:"-"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatResponse encapsulates generated LLM text.
type ChatResponse struct {
	Response string `json:"response"`
}

type cloudflareAPIResponse struct {
	Success  bool            `json:"success"`
	Errors   []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result   json.RawMessage `json:"result"`
}

// Chat executes a conversational multi-turn prompt against a Cloudflare Workers AI LLM.
func (s *AIService) Chat(ctx context.Context, req ChatRequest) (string, error) {
	model := req.Model
	if model == "" {
		model = ModelLlama38BInstruct
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("go-cloudflare: marshal chat payload: %w", err)
	}

	respBytes, err := s.executeAIRun(ctx, model, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err == nil && chatResp.Response != "" {
		return chatResp.Response, nil
	}

	// Some models return {"response": "..."} wrapped in result
	var rawMap map[string]any
	if err := json.Unmarshal(respBytes, &rawMap); err == nil {
		if val, ok := rawMap["response"].(string); ok {
			return val, nil
		}
	}

	return string(respBytes), nil
}

// GenerateText is a simple 1-liner to prompt any Cloudflare Workers AI model.
func (s *AIService) GenerateText(ctx context.Context, model, prompt string) (string, error) {
	if prompt == "" {
		return "", ErrEmptyPayload
	}

	return s.Chat(ctx, ChatRequest{
		Model: model,
		Messages: []ChatMessage{
			{Role: "user", Content: prompt},
		},
	})
}

// GenerateImage generates raw image bytes (e.g. PNG/JPEG) from a text description using Stable Diffusion.
func (s *AIService) GenerateImage(ctx context.Context, model, prompt string) ([]byte, error) {
	if model == "" {
		model = ModelSDXLLightning
	}
	if prompt == "" {
		return nil, ErrEmptyPayload
	}

	payload, _ := json.Marshal(map[string]string{
		"prompt": prompt,
	})

	return s.executeAIRun(ctx, model, "application/json", bytes.NewReader(payload))
}

// RemoveBackground sends an image to Cloudflare Workers AI (@cf/bria/rmbg-2.0)
// and returns the background-removed transparent PNG bytes.
func (s *AIService) RemoveBackground(ctx context.Context, imgBytes []byte) ([]byte, error) {
	if len(imgBytes) == 0 {
		return nil, ErrEmptyPayload
	}

	// Cloudflare Workers AI image-to-image expects a uint8 integer array: {"image": [ ... ]}
	intArr := make([]int, len(imgBytes))
	for i, b := range imgBytes {
		intArr[i] = int(b)
	}

	payload, err := json.Marshal(map[string]any{
		"image": intArr,
	})
	if err != nil {
		return nil, fmt.Errorf("go-cloudflare: marshal image payload: %w", err)
	}

	return s.executeAIRun(ctx, ModelRMBG20, "application/json", bytes.NewReader(payload))
}

// Embeddings generates vector embeddings for a batch of text strings.
func (s *AIService) Embeddings(ctx context.Context, model string, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyPayload
	}
	if model == "" {
		model = ModelBGEBase
	}

	payload, _ := json.Marshal(map[string]any{
		"text": texts,
	})

	respBytes, err := s.executeAIRun(ctx, model, "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	var embResult struct {
		Data [][]float64 `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &embResult); err == nil && len(embResult.Data) > 0 {
		return embResult.Data, nil
	}

	// Some models return direct [][]float64
	var directData [][]float64
	if err := json.Unmarshal(respBytes, &directData); err == nil {
		return directData, nil
	}

	return nil, fmt.Errorf("go-cloudflare: unexpected embeddings response structure")
}

func (s *AIService) executeAIRun(ctx context.Context, model, contentType string, body io.Reader) ([]byte, error) {
	if model == "" {
		return nil, ErrModelRequired
	}

	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/run/%s", s.client.cfg.AccountID, model)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("go-cloudflare: create http request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.client.cfg.APIToken)
	httpReq.Header.Set("Content-Type", contentType)

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("go-cloudflare: ai request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("go-cloudflare: read response body: %w", err)
	}

	// If binary image response (e.g. image/png), return bytes directly
	respContentType := resp.Header.Get("Content-Type")
	if strings.Contains(respContentType, "image/") || strings.Contains(respContentType, "octet-stream") {
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%w: status %d: %s", ErrAPIError, resp.StatusCode, string(respBytes))
		}
		return respBytes, nil
	}

	// Parse Cloudflare standard envelope: {"success": true, "result": ...}
	var apiResp cloudflareAPIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err == nil {
		if !apiResp.Success && len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("%w: %s (code %d)", ErrAPIError, apiResp.Errors[0].Message, apiResp.Errors[0].Code)
		}
		if len(apiResp.Result) > 0 {
			return apiResp.Result, nil
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", ErrAPIError, resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}
