# Technical Specification: go-gemini

## 1. Domain & Purpose
`go-gemini` is a zero-dependency, production-grade Go SDK for Google Gemini models (`gemini-2.0-flash`, `gemini-1.5-flash`, `gemini-1.5-pro`). It features thread-safe multi-key round-robin rotation, automated HTTP 429 rate-limit cooldown quarantine, multi-model fallback resiliency, vision OCR, and structured JSON extraction.

## 2. Architecture & Design Principles
- **Zero Third-Party Dependencies**: Pure Go standard library (`net/http`, `encoding/json`, `encoding/base64`, `sync`, `time`, `regexp`). Eliminates heavy Google Cloud gRPC and protobuf dependencies.
- **Resilient Key Rotation**: Distributes API calls evenly across an array of API keys. Automatically isolates keys receiving `429 Too Many Requests` or `RESOURCE_EXHAUSTED` into a temporary cooldown pool.
- **Model Fallback Chain**: If the primary model fails or encounters quota limits, the request automatically falls back to secondary models (e.g. `gemini-2.0-flash` -> `gemini-1.5-flash` -> `gemini-1.5-pro`).
- **Structured JSON Extraction**: Model responses enclosed in markdown code fences (` ```json ... ``` `) are automatically stripped, cleaned, and unmarshaled into target structs via `GenerateJSON`.
- **Vision Multimodal Support**: Accepts raw image bytes (JPEG, PNG, WebP) and automatically packages them into base64 `inline_data` parts.

## 3. API Surface
- `type Config struct`
- `func New(cfg Config) (*Client, error)`
- `func (c *Client) GenerateText(ctx context.Context, prompt string) (string, error)`
- `func (c *Client) GenerateWithVision(ctx context.Context, prompt string, imgBytes []byte, mimeType string) (string, error)`
- `func GenerateJSON(ctx context.Context, c *Client, prompt string, dest any) error`
- `func ExtractJSON(input string) string`
- `func NewKeyRotator(keys []string, cooldown time.Duration) (*KeyRotator, error)`

## 4. Error Handling
- `ErrNoAPIKeys`: No valid API keys provided.
- `ErrAllKeysExhausted`: All keys are currently quarantined due to 429 rate limits.
- `ErrEmptyPrompt`: Prompt string is empty.
- `ErrEmptyPayload`: Image or payload bytes are empty.
- `ErrModelFailed`: All models in the fallback chain were exhausted without success.
- `ErrJSONExtractionFailed`: Model response did not contain valid parseable JSON.

## 5. Verification & Testing
- Unit tests verify round-robin sequence, rate-limit cooldown quarantine timing, markdown code block stripping, vision payload assembly, and fallback model recovery.
