# Technical Specification: go-cloudflare

## 1. Domain & Purpose
`go-cloudflare` is a zero-dependency, production-grade Go client library providing unified access to Cloudflare Workers AI (LLMs, RMBG background removal, Stable Diffusion, BGE embeddings) and Cloudflare R2 S3-compatible Object Storage with pure Go standard library AWS SigV4 request signing.

## 2. Architecture & Design Principles
- **Zero Third-Party Dependencies**: 100% Go standard library (`net/http`, `crypto/hmac`, `crypto/sha256`, `encoding/hex`, `encoding/json`). Eliminates massive 50MB+ AWS SDK dependencies for S3/R2 usage.
- **Workers AI Engine**: Supports multi-turn chat (`ChatRequest`), 1-liner prompt execution (`GenerateText`), Text-to-Image generation (`GenerateImage`), transparent PNG background removal (`RemoveBackground`), and vector embeddings (`Embeddings`).
- **R2 Storage Engine**: Pure Go implementation of AWS Signature Version 4 (SigV4) against the Cloudflare R2 regional endpoints (`https://<account_id>.r2.cloudflarestorage.com`), supporting upload, download, deletion, and public CDN URL resolution.
- **Context-Aware**: Full support for deadlines, timeouts, and cancellation propagation.

## 3. API Surface
- `type Config struct`
- `func New(cfg Config) (*Client, error)`
- `func (c *Client) AccountID() string`
- `func (s *AIService) Chat(ctx context.Context, req ChatRequest) (string, error)`
- `func (s *AIService) GenerateText(ctx context.Context, model, prompt string) (string, error)`
- `func (s *AIService) GenerateImage(ctx context.Context, model, prompt string) ([]byte, error)`
- `func (s *AIService) RemoveBackground(ctx context.Context, imgBytes []byte) ([]byte, error)`
- `func (s *AIService) Embeddings(ctx context.Context, model string, texts []string) ([][]float64, error)`
- `func (s *R2Service) Endpoint() string`
- `func (s *R2Service) PublicURL(key string) string`
- `func (s *R2Service) Upload(ctx context.Context, bucket, key string, data []byte, contentType string) (string, error)`
- `func (s *R2Service) Download(ctx context.Context, bucket, key string) ([]byte, string, error)`
- `func (s *R2Service) Delete(ctx context.Context, bucket, key string) error`

## 4. Error Handling
- `ErrMissingCredentials`: Account ID or API Token missing.
- `ErrModelRequired`: Model identifier missing.
- `ErrEmptyPayload`: Payload empty.
- `ErrR2CredentialsRequired`: R2 access key or secret key missing.
- `ErrR2BucketRequired`: Bucket name missing.
- `ErrAPIError`: Cloudflare returned error response.

## 5. Verification & Testing
- Validates credential checking, Workers AI envelope response parsing, binary image pass-through, R2 endpoint generation, public CDN URL mapping, and SigV4 authentication headers.
