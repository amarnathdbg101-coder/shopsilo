# go-otp

Zero-dependency RFC 6238 TOTP & RFC 4226 HOTP Two-Factor Authentication (2FA) engine for Go.

---

## Features
- **RFC Compliant**: Validated against official IETF RFC 4226 Appendix D test vectors.
- **Authenticator App Compatible**: Works out-of-the-box with Google Authenticator, Microsoft Authenticator, 1Password, and Authy.
- **Clock Drift Tolerance**: Configurable skew windows (e.g. `skew=1` allows ±30s) to prevent authentication failures on client clock drift.
- **Constant-Time Verification**: Prevents side-channel timing attacks.
- **Provisioning URI Builder**: Generates `otpauth://totp/...` strings ready for QR codes.
- **Zero External Dependencies**: Pure Go standard library.

---

## Quickstart

```go
package main

import (
	"fmt"
	"time"

	gootp "go-otp"
)

func main() {
	// 1. Generate a new Base32 secret for user
	secret, err := gootp.GenerateSecret(20)
	if err != nil {
		panic(err)
	}
	fmt.Printf("User Secret: %s\n", secret)

	// 2. Generate QR code URI for Authenticator apps
	cfg := gootp.DefaultConfig()
	cfg.Issuer = "ShopSilo"
	cfg.AccountName = "user@example.com"

	uri := gootp.ProvisioningURI(secret, cfg)
	fmt.Printf("QR Code URI: %s\n", uri)
	// Output: otpauth://totp/ShopSilo:user@example.com?algorithm=SHA1&digits=6&issuer=ShopSilo&period=30&secret=...

	// 3. User enters code from Authenticator app
	code, _ := gootp.GenerateTOTP(secret, time.Now(), cfg)
	fmt.Printf("Generated 6-digit Code: %s\n", code)

	// 4. Verify code (allowing ±1 step = ±30s clock drift)
	isValid := gootp.VerifyTOTP(secret, code, time.Now(), 1, cfg)
	fmt.Printf("Is Code Valid? %v\n", isValid)
}
```

---

## Testing
Run unit tests with full race and count coverage:
```bash
go test -v -count=1 ./...
```
