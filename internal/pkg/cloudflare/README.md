# go-cloudflare ☁️

> **Zero-Dependency Cloudflare SDK for Go (Workers AI & R2 Object Storage).**  
> Pure Go standard library client for running serverless LLMs (Llama 3, Mistral), AI background removal (RMBG), Stable Diffusion text-to-image, BGE vector embeddings, and S3-compatible R2 storage with pure Go AWS SigV4 signing.

---

## ✨ Features

- 🚀 **Zero External Dependencies**: 100% Pure Go Standard Library — no massive 50MB+ AWS SDK or third-party wrappers required!
- 🤖 **Workers AI Serverless Inference**:
  - **LLM Chat & Text**: Llama 3.1 8B, Mistral 7B, Gemma 2.
  - **Background Removal**: RMBG-1.4 and RMBG-2.0 transparent cutout extraction.
  - **Text-to-Image**: Stable Diffusion XL Lightning.
  - **Vector Embeddings**: BGE-Base / BGE-Small for semantic vector search & RAG.
- 🗄️ **R2 Object Storage (Pure Go SigV4)**:
  - Instant file uploads with content-type detection.
  - Public CDN URL and custom domain path resolution.
  - File download, inspection, and deletion.
- ⏱️ **Context-Aware**: Built-in cancellation and timeout handling.

---

## 📦 Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"

	goclos "github.com/amarnathdbg101-coder/go/go-cloudflare"
)

func main() {
	client, err := goclos.New(goclos.Config{
		AccountID:      "your_cloudflare_account_id",
		APIToken:       "your_cloudflare_api_token",
		R2AccessKeyID:  "your_r2_access_key_id",
		R2SecretAccess: "your_r2_secret_access_key",
		R2Bucket:       "my-app-uploads",
		R2PublicURL:    "https://cdn.myapp.com",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 1. Run LLM Chat (Llama 3)
	reply, err := client.AI.GenerateText(ctx, goclos.ModelLlama38BInstruct, "Explain the transactional outbox pattern in 2 sentences.")
	if err == nil {
		fmt.Println("Llama 3:", reply)
	}

	// 2. AI Background Removal (RMBG-2.0)
	rawImageBytes := []byte{ ... }
	transparentPNG, err := client.AI.RemoveBackground(ctx, rawImageBytes)
	if err == nil {
		fmt.Printf("Cutout generated: %d bytes\n", len(transparentPNG))
	}

	// 3. Upload File to Cloudflare R2
	publicURL, err := client.R2.Upload(ctx, "my-app-uploads", "avatars/user_101.png", transparentPNG, "image/png")
	if err == nil {
		fmt.Println("Uploaded to R2:", publicURL)
		// Output: https://cdn.myapp.com/avatars/user_101.png
	}
}
```

---

## 🧪 Testing

```bash
cd C:\Library\go-cloudflare
go test -v -count=1 ./...
```
