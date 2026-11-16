# go-mask 🎭

High-speed PII data masking engine for Indian regulations (Aadhaar, PAN) and international compliance (PCI-DSS credit cards, phones, emails, recursive JSON redaction). Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- 🇮🇳 **Indian Fintech Compliant**: Standard masks for Aadhaar (`XXXX-XXXX-1234`) aur PAN cards (`XXXXX1234X`).
- 💳 **PCI-DSS Credit Card Masking**: `4111-XXXX-XXXX-1234` format for safe logging.
- 📱 **Phone & Email Privacy**: `+91-98****3210` aur `a****a@example.com` masking.
- 🪵 **Recursive JSON Redactor**: Server logs me JSON dump karne se pehle passwords, tokens, CVVs ko automatically `********` bana deta hai.
- ⚡ **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. Masking Sensitive Customer PII
```go
package main

import (
	"fmt"
	gomask "go-mask"
)

func main() {
	fmt.Println("Aadhaar:", gomask.Aadhaar("5482 9182 3491")) // Output: XXXX-XXXX-3491
	fmt.Println("PAN:", gomask.PAN("ABCDE1234F"))            // Output: XXXXX1234X
	fmt.Println("Card:", gomask.CreditCard("4111222233334444")) // Output: 4111-XXXX-XXXX-4444
	fmt.Println("Phone:", gomask.Phone("+919876543210"))     // Output: +91-98****3210
	fmt.Println("Email:", gomask.Email("aditya@example.com")) // Output: a****a@example.com
}
```

### 2. Redacting Sensitive JSON Payloads in Logs
```go
safeJSON, _ := gomask.JSONRecursively(rawPayloadBytes)
log.Println("Received Webhook:", string(safeJSON))
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
