package gourlsigner

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestURLSigner_ValidRoundtrip(t *testing.T) {
	secret := []byte("secret-key-123")
	signer, err := New(secret)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	rawURL := "https://api.shopsilo.com/v1/invoices/inv_998.pdf?download=true"
	signedURL, err := signer.Sign(rawURL, 10*time.Minute)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if !strings.Contains(signedURL, "signature=") || !strings.Contains(signedURL, "expires=") {
		t.Errorf("signed URL missing parameters: %s", signedURL)
	}

	err = signer.Verify(signedURL)
	if err != nil {
		t.Errorf("expected valid verification, got error: %v", err)
	}
}

func TestURLSigner_TamperedQuery(t *testing.T) {
	secret := []byte("secret")
	signer, _ := New(secret)

	rawURL := "https://api.example.com/reset?user_id=101"
	signedURL, _ := signer.Sign(rawURL, 5*time.Minute)

	// Tamper user_id
	tamperedURL := strings.Replace(signedURL, "user_id=101", "user_id=999", 1)

	err := signer.Verify(tamperedURL)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature on tampered query, got %v", err)
	}
}

func TestURLSigner_TamperedPath(t *testing.T) {
	secret := []byte("secret")
	signer, _ := New(secret)

	rawURL := "https://api.example.com/files/doc1.pdf"
	signedURL, _ := signer.Sign(rawURL, 5*time.Minute)

	// Tamper path
	tamperedURL := strings.Replace(signedURL, "/files/doc1.pdf", "/files/doc2.pdf", 1)

	err := signer.Verify(tamperedURL)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature on tampered path, got %v", err)
	}
}

func TestURLSigner_Expired(t *testing.T) {
	secret := []byte("secret")
	signer, _ := New(secret)

	rawURL := "https://api.example.com/temp"
	// Sign with negative TTL (already expired in past)
	signedURL, _ := signer.Sign(rawURL, -5*time.Second)

	err := signer.Verify(signedURL)
	if !errors.Is(err, ErrExpired) {
		t.Errorf("expected ErrExpired, got %v", err)
	}
}

func TestURLSigner_MissingSignature(t *testing.T) {
	signer, _ := New([]byte("secret"))
	err := signer.Verify("https://api.example.com/test?expires=123")
	if !errors.Is(err, ErrMissingSignature) {
		t.Errorf("expected ErrMissingSignature, got %v", err)
	}
}
