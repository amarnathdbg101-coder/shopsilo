package reuse_test

import (
	"strings"
	"testing"
	"time"

	"shopMe/internal/reuse"
)

type ResetTokenPayload struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Action string `json:"action"`
}

func TestSealData_ValidUnseal(t *testing.T) {
	orig := ResetTokenPayload{
		UserID: "usr_9981",
		Email:  "test@shopsilo.com",
		Action: "password_reset",
	}

	token, err := reuse.SealData(orig, 15*time.Minute)
	if err != nil {
		t.Fatalf("SealData failed: %v", err)
	}

	if len(strings.Split(token, ".")) != 3 {
		t.Fatalf("expected 3-part token, got %s", token)
	}

	var decoded ResetTokenPayload
	if err := reuse.UnsealData(token, &decoded); err != nil {
		t.Fatalf("UnsealData failed: %v", err)
	}

	if decoded.UserID != orig.UserID || decoded.Email != orig.Email || decoded.Action != orig.Action {
		t.Fatalf("decoded payload mismatch: %+v vs %+v", decoded, orig)
	}
}

func TestSealData_TamperDetection(t *testing.T) {
	orig := ResetTokenPayload{
		UserID: "usr_100",
		Email:  "user@example.com",
		Action: "statement_view",
	}

	token, err := reuse.SealData(orig, 15*time.Minute)
	if err != nil {
		t.Fatalf("SealData failed: %v", err)
	}

	// Tamper with token payload by replacing a character
	parts := strings.Split(token, ".")
	tamperedToken := parts[0] + "tampered" + "." + parts[1] + "." + parts[2]

	var decoded ResetTokenPayload
	if err := reuse.UnsealData(tamperedToken, &decoded); err == nil {
		t.Fatal("expected unseal to fail for tampered token, but succeeded")
	}
}
