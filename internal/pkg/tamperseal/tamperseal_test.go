package tamperseal

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type VerificationPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Action string `json:"action"`
}

func TestTamperSeal_Success(t *testing.T) {
	secret := []byte("super_secure_secret_key_2026")
	payload := VerificationPayload{
		UserID: "usr_9981",
		Email:  "arjun@example.com",
		Action: "verify_email",
	}

	token, err := Seal(secret, payload, 1*time.Hour)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	var dest VerificationPayload
	err = Unseal(secret, token, &dest)
	if err != nil {
		t.Fatalf("Unseal failed: %v", err)
	}

	if dest.UserID != payload.UserID || dest.Email != payload.Email || dest.Action != payload.Action {
		t.Fatalf("unsealed payload mismatch: %+v", dest)
	}
}

func TestTamperSeal_TamperedPayload(t *testing.T) {
	secret := []byte("secret")
	token, _ := Seal(secret, map[string]string{"role": "user"}, 1*time.Hour)

	// Tamper payload
	parts := strings.Split(token, ".")
	tampered := parts[0] + "tamper." + parts[1] + "." + parts[2]

	var dest map[string]string
	err := Unseal(secret, tampered, &dest)
	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch, got: %v", err)
	}
}

func TestTamperSeal_TamperedSignature(t *testing.T) {
	secret := []byte("secret")
	token, _ := Seal(secret, "data", 1*time.Hour)

	parts := strings.Split(token, ".")
	lastChar := parts[2][len(parts[2])-1]
	replacement := "a"
	if lastChar == 'a' {
		replacement = "b"
	}
	tamperedSig := parts[2][:len(parts[2])-1] + replacement
	tamperedToken := parts[0] + "." + parts[1] + "." + tamperedSig

	var dest string
	err := Unseal(secret, tamperedToken, &dest)
	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch, got: %v", err)
	}
}

func TestTamperSeal_ExpiredToken(t *testing.T) {
	secret := []byte("secret")
	// Token with negative TTL (already expired 1 hour ago)
	token, _ := Seal(secret, "payload", -1*time.Hour)

	var dest string
	err := Unseal(secret, token, &dest)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got: %v", err)
	}

	expired, err := IsExpired(token)
	if err != nil || !expired {
		t.Fatalf("expected IsExpired true, got %v, err: %v", expired, err)
	}
}

func TestTamperSeal_WrongSecret(t *testing.T) {
	token, _ := Seal([]byte("secret_1"), "important", 1*time.Hour)

	var dest string
	err := Unseal([]byte("secret_2"), token, &dest)
	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch with wrong secret, got: %v", err)
	}
}
