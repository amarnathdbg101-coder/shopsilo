package reuse_test

import (
	"strings"
	"testing"

	"shopMe/internal/reuse"
)

func TestPasswordFeatures(t *testing.T) {
	t.Run("Rejects Trivial Common Passwords", func(t *testing.T) {
		err := reuse.ValidatePassword("123456")
		if err == nil {
			t.Fatalf("expected error for common password 123456, got nil")
		}
	})

	t.Run("Rejects Repetitive Characters", func(t *testing.T) {
		err := reuse.ValidatePassword("aaaaaaa")
		if err == nil {
			t.Fatalf("expected error for repetitive password aaaaaaa, got nil")
		}
	})

	t.Run("Evaluates Password Strength", func(t *testing.T) {
		tier, entropy := reuse.EvaluatePasswordStrength("ShopSilo@2026!Secure")
		if tier != "Strong" && tier != "Very Strong" {
			t.Fatalf("expected strong password tier, got %s", tier)
		}
		if entropy < 40.0 {
			t.Fatalf("expected entropy >= 40 bits, got %.2f", entropy)
		}
	})

	t.Run("Generates High Entropy Staff Password", func(t *testing.T) {
		pw, err := reuse.GenerateSecureStaffPassword(12)
		if err != nil {
			t.Fatalf("unexpected error generating staff password: %v", err)
		}
		if len(pw) != 12 {
			t.Fatalf("expected 12 characters, got %d (%s)", len(pw), pw)
		}
		if strings.TrimSpace(pw) != pw {
			t.Fatalf("password should not contain whitespace: %s", pw)
		}
	})
}
