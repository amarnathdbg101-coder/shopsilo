package reuse_test

import (
	"strings"
	"testing"
	"time"

	"shopMe/internal/reuse"
)

func TestSignDownloadURL_ValidVerify(t *testing.T) {
	baseURL := "https://api.shopsilo.com/api/v1/pos/invoices/inv_78912/download?format=pdf"

	signedURL, err := reuse.SignDownloadURL(baseURL, 10*time.Minute)
	if err != nil {
		t.Fatalf("SignDownloadURL failed: %v", err)
	}

	if !strings.Contains(signedURL, "expires=") || !strings.Contains(signedURL, "signature=") {
		t.Fatalf("signed URL missing expires or signature: %s", signedURL)
	}

	if err := reuse.VerifyDownloadURL(signedURL); err != nil {
		t.Fatalf("VerifyDownloadURL failed for legitimate URL: %v", err)
	}
}

func TestSignDownloadURL_TamperRejection(t *testing.T) {
	baseURL := "https://api.shopsilo.com/api/v1/pos/invoices/inv_78912/download"

	signedURL, err := reuse.SignDownloadURL(baseURL, 10*time.Minute)
	if err != nil {
		t.Fatalf("SignDownloadURL failed: %v", err)
	}

	// Tampering: changing invoice id in URL path
	tamperedURL := strings.Replace(signedURL, "inv_78912", "inv_00001", 1)

	if err := reuse.VerifyDownloadURL(tamperedURL); err == nil {
		t.Fatal("expected VerifyDownloadURL to fail for tampered path, but it passed!")
	}
}
