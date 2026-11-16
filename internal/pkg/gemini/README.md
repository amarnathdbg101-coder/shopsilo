# go-gemini 💎

> **Zero-Dependency Google Gemini SDK for Go with Multi-Key Rotation & Model Fallback.**  
> Pure Go standard library client for Gemini 2.0 Flash, 1.5 Flash, and 1.5 Pro with automatic round-robin key rotation, 429 rate-limit cooldown quarantine, multi-model fallback, vision OCR, and structured JSON parsing.

---

## ✨ Features

- 🚀 **Zero External Dependencies**: Pure Go Standard Library — no massive Google Cloud gRPC or protobuf dependencies!
- 🔄 **Multi-Key Round-Robin Rotation**: Load balances requests across multiple API keys.
- 🛑 **Rate-Limit Quarantine**: Automatically cools down keys that hit HTTP `429 Too Many Requests` or `RESOURCE_EXHAUSTED` for 60 seconds.
- 🛡️ **Multi-Model Fallback**: Automatically tries secondary models (e.g. `gemini-1.5-flash`) if the primary model fails or runs out of quota.
- 👁️ **Vision OCR**: Pass raw image bytes (`image/jpeg`, `image/png`, `image/webp`) for receipt scanning, invoice parsing, or visual Q&A.
- 🧹 **Automatic Markdown Stripper**: Extracts clean, parseable JSON from model outputs wrapped in ````json ... ```` fences.

---

## 📦 Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"

	gogemini "github.com/amarnathdbg101-coder/go/go-gemini"
)

type Campaign struct {
	Headline        string `json:"headline"`
	WhatsAppMessage string `json:"whatsapp_message"`
}

func main() {
	client, err := gogemini.New(gogemini.Config{
		APIKeys: []string{
			"AIzaSyKey1...",
			"AIzaSyKey2...",
			"AIzaSyKey3...",
		},
		DefaultModel:   gogemini.ModelGemini20Flash,
		FallbackModels: []string{gogemini.ModelGemini15Flash},
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 1. Plain Text Generation
	reply, err := client.GenerateText(ctx, "Explain double-entry bookkeeping in 1 sentence.")
	if err == nil {
		fmt.Println("Gemini:", reply)
	}

	// 2. Structured JSON Extraction (Auto-strips markdown fences!)
	var campaign Campaign
	prompt := `Generate a Diwali promotional campaign. Return JSON: {"headline":"...","whatsapp_message":"..."}`
	err = gogemini.GenerateJSON(ctx, client, prompt, &campaign)
	if err == nil {
		fmt.Printf("Headline: %s\nWhatsApp: %s\n", campaign.Headline, campaign.WhatsAppMessage)
	}
}
```

---

## 🧪 Testing

```bash
cd C:\Library\go-gemini
go test -v -count=1 ./...
```
