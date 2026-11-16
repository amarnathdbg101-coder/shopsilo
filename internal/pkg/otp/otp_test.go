package gootp

import (
	"strings"
	"testing"
	"time"
)

func TestHOTP_RFC4226_TestVectors(t *testing.T) {
	// RFC 4226 Section 5.3 Test Secret: "12345678901234567890"
	// In Base32: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

	expected := []string{
		"755224", // 0
		"287082", // 1
		"359152", // 2
		"969429", // 3
		"338314", // 4
		"254676", // 5
		"287922", // 6
		"162583", // 7
		"399871", // 8
		"520489", // 9
	}

	for counter, exp := range expected {
		code, err := GenerateHOTP(secret, uint64(counter), 6, SHA1)
		if err != nil {
			t.Fatalf("counter %d failed: %v", counter, err)
		}
		if code != exp {
			t.Errorf("counter %d: expected %s, got %s", counter, exp, code)
		}
	}
}

func TestTOTP_GenerateAndVerify(t *testing.T) {
	secret, err := GenerateSecret(20)
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}

	now := time.Now()
	cfg := DefaultConfig()

	code, err := GenerateTOTP(secret, now, cfg)
	if err != nil {
		t.Fatalf("GenerateTOTP failed: %v", err)
	}

	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %q", code)
	}

	// Verify exact time
	if !VerifyTOTP(secret, code, now, 0, cfg) {
		t.Errorf("expected verification to succeed at current time")
	}

	// Verify invalid code
	if VerifyTOTP(secret, "000000", now, 0, cfg) {
		t.Errorf("expected false code to fail verification")
	}
}

func TestTOTP_ClockSkew(t *testing.T) {
	secret, _ := GenerateSecret(20)
	now := time.Now()
	cfg := DefaultConfig()

	// Generate code for 30s in past
	pastTime := now.Add(-30 * time.Second)
	pastCode, _ := GenerateTOTP(secret, pastTime, cfg)

	// With skew 0 at current time -> Should FAIL
	if VerifyTOTP(secret, pastCode, now, 0, cfg) {
		t.Errorf("expected skew=0 to reject past code")
	}

	// With skew 1 at current time -> Should PASS
	if !VerifyTOTP(secret, pastCode, now, 1, cfg) {
		t.Errorf("expected skew=1 to accept past code")
	}
}

func TestProvisioningURI(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	cfg := Config{
		Digits:      6,
		Period:      30 * time.Second,
		Algorithm:   SHA1,
		Issuer:      "AcmeCorp",
		AccountName: "alice@example.com",
	}

	uri := ProvisioningURI(secret, cfg)

	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Errorf("invalid URI prefix: %s", uri)
	}
	if !strings.Contains(uri, "secret=JBSWY3DPEHPK3PXP") {
		t.Errorf("URI missing secret: %s", uri)
	}
	if !strings.Contains(uri, "issuer=AcmeCorp") {
		t.Errorf("URI missing issuer: %s", uri)
	}
}
