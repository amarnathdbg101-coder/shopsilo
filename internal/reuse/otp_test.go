package reuse_test

import (
	"strconv"
	"strings"
	"testing"

	"shopMe/internal/reuse"
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

func TestTOTP2FA_Flow(t *testing.T) {
	// 1. Generate Base32 2FA secret
	secret, err := reuse.Generate2FASecret()
	if err != nil {
		t.Fatalf("Generate2FASecret failed: %v", err)
	}
	if len(secret) < 20 {
		t.Fatalf("expected Base32 secret length >= 20, got %s", secret)
	}

	// 2. Generate Provisioning QR Code URI
	uri := reuse.Generate2FAQRCodeURI("admin@shopsilo.com", secret)
	if !strings.HasPrefix(uri, "otpauth://totp/") || !strings.Contains(uri, "ShopSilo") {
		t.Fatalf("invalid provisioning URI: %s", uri)
	}

	// 3. Generate current code
	code, err := reuse.GenerateCurrentTOTP(secret)
	if err != nil {
		t.Fatalf("GenerateCurrentTOTP failed: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit TOTP code, got %s", code)
	}

	// 4. Verify code succeeds
	if !reuse.Verify2FACode(secret, code) {
		t.Fatal("expected legitimate 2FA code to verify successfully")
	}

	// 5. Invalid code fails
	if reuse.Verify2FACode(secret, "000000") && code != "000000" {
		t.Fatal("expected invalid 2FA code to fail")
	}
}
