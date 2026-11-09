// Package reuse provides reusable application helpers and utilities.
package reuse

import (
	"errors"
	"time"

	gourlsigner "github.com/amarnathdbg101-coder/go/go-urlsigner"
	"shopMe/internal/utils"
)

// SignDownloadURL creates a tamper-proof presigned temporary URL for invoice, PDF, or file downloads.
// The URL automatically expires after the given duration and any modification to parameters will invalidate it.
func SignDownloadURL(rawURL string, ttl time.Duration) (string, error) {
	if rawURL == "" {
		return "", errors.New("cannot sign empty URL")
	}

	cfg := utils.MustLoad()
	secret := []byte(cfg.Jwt)
	if len(secret) == 0 {
		secret = []byte("shopsilo-presigned-url-signing-secret-key-2026")
	}

	signer, err := gourlsigner.New(secret)
	if err != nil {
		return "", err
	}

	return signer.Sign(rawURL, ttl)
}

// VerifyDownloadURL verifies that a presigned download URL is valid, unexpired, and has not been tampered with.
func VerifyDownloadURL(signedURL string) error {
	if signedURL == "" {
		return errors.New("empty URL")
	}

	cfg := utils.MustLoad()
	secret := []byte(cfg.Jwt)
	if len(secret) == 0 {
		secret = []byte("shopsilo-presigned-url-signing-secret-key-2026")
	}

	signer, err := gourlsigner.New(secret)
	if err != nil {
		return err
	}

	return signer.Verify(signedURL)
}
