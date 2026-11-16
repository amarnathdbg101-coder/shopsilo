package reuse_test

import (
	"shopMe/internal/reuse"
	"strconv"
	"testing"
)

func TestCryptoSecureOTP(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		otp, err := reuse.GenerateCryptoSecureOTP()
		if err != nil {
			t.Fatalf("unexpected error generating OTP: %v", err)
		}
		if len(otp) != 6 {
			t.Fatalf("expected OTP length 6, got %d (%s)", len(otp), otp)
		}
		num, err := strconv.Atoi(otp)
		if err != nil || num < 100000 || num > 999999 {
			t.Fatalf("OTP out of expected bounds: %s", otp)
		}
		seen[otp] = true
	}
	if len(seen) < 45 {
		t.Fatalf("low entropy detected in OTP generation: %d unique out of 50", len(seen))
	}
}

func TestOTPHashAndVerify(t *testing.T) {
	phone := "9876543210"
	otp := "123456"
	secret := "super-secure-secret"

	hash := reuse.HashOTP(phone, otp, secret)
	if hash == "" {
		t.Fatalf("expected non-empty hash")
	}

	// Valid verification
	if !reuse.VerifyOTPHash(phone, otp, secret, hash) {
		t.Fatalf("expected OTP verification to succeed for matching OTP")
	}

	// Wrong OTP
	if reuse.VerifyOTPHash(phone, "654321", secret, hash) {
		t.Fatalf("expected OTP verification to fail for wrong OTP")
	}

	// Wrong Phone
	if reuse.VerifyOTPHash("9999999999", otp, secret, hash) {
		t.Fatalf("expected OTP verification to fail for wrong phone")
	}

	// Wrong Secret
	if reuse.VerifyOTPHash(phone, otp, "wrong-secret", hash) {
		t.Fatalf("expected OTP verification to fail for wrong secret")
	}
}
