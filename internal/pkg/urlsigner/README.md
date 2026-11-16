# go-urlsigner

Cryptographically signed temporary URLs and presigned links with HMAC-SHA256 for Go.

---

## Features
- **Presigned Temporary URLs**: Generate secure, time-limited download links for private files (e.g. S3/CloudStorage presigned downloads).
- **Tamper-Proof**: Modifying any URL path or query parameter instantly breaks verification.
- **Timing Attack Safe**: Validated using constant-time `hmac.Equal`.
- **Zero External Dependencies**: Pure Go standard library.

---

## Quickstart

```go
package main

import (
	"fmt"
	"time"

	gourlsigner "go-urlsigner"
)

func main() {
	secret := []byte("your-master-secret-key-32bytes!")
	signer, _ := gourlsigner.New(secret)

	// 1. Generate 15-minute temporary presigned download link
	downloadURL := "https://cdn.example.com/invoices/inv_99812.pdf?view=inline"
	signedURL, err := signer.Sign(downloadURL, 15*time.Minute)
	if err != nil {
		panic(err)
	}

	fmt.Println("Presigned URL:", signedURL)
	// Output: https://cdn.example.com/invoices/inv_99812.pdf?download=true&expires=1700000900&signature=a9f8c...

	// 2. Verify incoming request
	if err := signer.Verify(signedURL); err != nil {
		fmt.Println("Access denied:", err)
		return
	}

	fmt.Println("Access granted! Serving file...")
}
```

---

## Testing
Run unit tests with full race and count coverage:
```bash
go test -v -count=1 ./...
```
