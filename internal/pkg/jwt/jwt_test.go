package gojwt

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestJWT_SignAndVerify(t *testing.T) {
	secret := []byte("production_super_secret_key_2026")
	engine, err := New(secret)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	claims := Claims{
		Subject:   "usr_102",
		Issuer:    "ShopSilo-Auth",
		Audience:  "ShopSilo-Web",
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
		Custom: map[string]any{
			"role":  "admin",
			"email": "aman@example.com",
		},
	}

	tokenStr, err := engine.Sign(claims)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Must be 3 parts
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in JWT, got %d", len(parts))
	}

	// Verify
	parsed, err := engine.Verify(tokenStr)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if parsed.Subject != "usr_102" || parsed.Issuer != "ShopSilo-Auth" {
		t.Fatalf("claims mismatch: %+v", parsed)
	}
	if parsed.Custom["role"] != "admin" {
		t.Fatalf("custom claim mismatch: %+v", parsed.Custom)
	}
}

func TestJWT_TamperedSignature(t *testing.T) {
	secret := []byte("secret_1")
	engine, _ := New(secret)

	token, _ := engine.Sign(Claims{Subject: "1", ExpiresAt: time.Now().Add(time.Hour).Unix()})

	parts := strings.Split(token, ".")
	sig := parts[2]
	tamperedSig := "A" + sig[1:]
	if sig[0] == 'A' {
		tamperedSig = "B" + sig[1:]
	}
	tampered := parts[0] + "." + parts[1] + "." + tamperedSig

	_, err := engine.Verify(tampered)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestJWT_WrongSecret(t *testing.T) {
	engine1, _ := New([]byte("secret_correct"))
	engine2, _ := New([]byte("secret_wrong"))

	token, _ := engine1.Sign(Claims{Subject: "1", ExpiresAt: time.Now().Add(time.Hour).Unix()})

	_, err := engine2.Verify(token)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature on wrong secret, got %v", err)
	}
}

func TestJWT_ExpiredToken(t *testing.T) {
	secret := []byte("secret")
	engine, _ := New(secret)

	// Expired token (1 hour ago)
	token, _ := engine.Sign(Claims{
		Subject:   "1",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Unix(),
	})

	_, err := engine.Verify(token)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}

	// Token expired 2 seconds ago, with 2-minute leeway, it should be accepted
	recentExpiredToken, _ := engine.Sign(Claims{
		Subject:   "1",
		ExpiresAt: time.Now().Add(-2 * time.Second).Unix(),
	})

	engineWithLeeway, _ := New(secret, 2*time.Minute)
	parsed, err := engineWithLeeway.Verify(recentExpiredToken)
	if err != nil || parsed == nil {
		t.Fatalf("expected token to be valid within leeway, err: %v", err)
	}
}

func TestJWT_NotBefore(t *testing.T) {
	secret := []byte("secret")
	engine, _ := New(secret)

	// Valid 1 hour in future
	token, _ := engine.Sign(Claims{
		Subject:   "1",
		NotBefore: time.Now().Add(1 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(2 * time.Hour).Unix(),
	})

	_, err := engine.Verify(token)
	if !errors.Is(err, ErrTokenNotValidYet) {
		t.Fatalf("expected ErrTokenNotValidYet, got %v", err)
	}
}

type TestCustomClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp,omitempty"`
}

func TestJWT_CustomStruct(t *testing.T) {
	secret := []byte("custom_secret_key_123")
	input := TestCustomClaims{
		UserID: "usr_999",
		Email:  "test@shopsilo.in",
		Role:   "manager",
		Exp:    time.Now().Add(10 * time.Minute).Unix(),
	}

	tokenStr, err := SignCustom(secret, input)
	if err != nil {
		t.Fatalf("SignCustom failed: %v", err)
	}

	var output TestCustomClaims
	if err := VerifyCustom(secret, tokenStr, &output); err != nil {
		t.Fatalf("VerifyCustom failed: %v", err)
	}

	if output.UserID != input.UserID || output.Email != input.Email || output.Role != input.Role {
		t.Fatalf("claims mismatch: %+v vs %+v", output, input)
	}
}

func TestJWT_DecodeUnverified(t *testing.T) {
	secret := []byte("secret_xyz")
	input := TestCustomClaims{
		UserID: "usr_unverified",
		Email:  "anon@domain.com",
	}

	tokenStr, _ := SignCustom(secret, input)

	var output TestCustomClaims
	if err := DecodeUnverified(tokenStr, &output); err != nil {
		t.Fatalf("DecodeUnverified failed: %v", err)
	}
	if output.UserID != "usr_unverified" {
		t.Fatalf("expected usr_unverified, got %s", output.UserID)
	}
}

func TestJWT_VerifySignatureOnly(t *testing.T) {
	secret := []byte("secret_refresh")
	engine, _ := New(secret)

	// Already expired token
	input := TestCustomClaims{
		UserID: "usr_refresh_10",
		Exp:    time.Now().Add(-5 * time.Hour).Unix(),
	}
	tokenStr, _ := engine.SignCustom(input)

	// Verify should fail with expired
	var out1 TestCustomClaims
	err := engine.VerifyCustom(tokenStr, &out1)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}

	// VerifySignatureOnly should succeed
	var out2 TestCustomClaims
	err = engine.VerifySignatureOnly(tokenStr, &out2)
	if err != nil {
		t.Fatalf("VerifySignatureOnly failed: %v", err)
	}
	if out2.UserID != "usr_refresh_10" {
		t.Fatalf("expected usr_refresh_10, got %s", out2.UserID)
	}
}

func TestClaims_TopLevelCustomCapture(t *testing.T) {
	secret := []byte("secret_toplevel")
	rawJSONToken, _ := SignCustom(secret, map[string]any{
		"sub":     "subject_1",
		"user_id": "usr_top",
		"role":    "admin",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})

	claims, err := Verify(secret, rawJSONToken)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims.Subject != "subject_1" {
		t.Fatalf("expected subject_1, got %s", claims.Subject)
	}
	if claims.GetString("user_id") != "usr_top" {
		t.Fatalf("expected user_id usr_top, got %s", claims.GetString("user_id"))
	}
	if claims.GetString("role") != "admin" {
		t.Fatalf("expected role admin, got %s", claims.GetString("role"))
	}
}
