// Package reuse provides reusable application helpers and utilities.
package reuse

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	gootp "github.com/amarnathdbg101-coder/go/go-otp"
)

// GenerateCryptoSecureOTP generates an unpredictable 6-digit numeric OTP (100000 - 999999)
// using cryptographic random bytes to prevent pseudo-random prediction attacks.
func GenerateCryptoSecureOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", fmt.Errorf("failed to generate cryptographically secure OTP: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

// HashOTP computes a SHA-256 keyed hash combining the normalized phone number, OTP, and secret key.
// This prevents database-leak attacks (plain OTP exposure) and rainbow-table precomputations.
func HashOTP(phone, otp, secret string) string {
	h := sha256.New()
	h.Write([]byte(phone + ":" + otp + ":" + secret))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyOTPHash validates an input OTP against an expected hash in constant time
// to eliminate side-channel timing attack vectors.
func VerifyOTPHash(phone, otp, secret, expectedHash string) bool {
	computed := HashOTP(phone, otp, secret)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(expectedHash)) == 1
}

// Generate2FASecret creates a new cryptographically random Base32 secret for Google Authenticator 2FA.
func Generate2FASecret() (string, error) {
	return gootp.GenerateSecret(20) // 160-bit standard recommendation
}

// Generate2FAQRCodeURI builds an RFC-compliant otpauth:// URI ready for QR code rendering in Authenticator apps.
func Generate2FAQRCodeURI(accountEmail, secret string) string {
	cfg := gootp.DefaultConfig()
	cfg.Issuer = "ShopSilo"
	cfg.AccountName = accountEmail
	return gootp.ProvisioningURI(secret, cfg)
}

// GenerateCurrentTOTP generates the current 6-digit TOTP code for testing and automated validation.
func GenerateCurrentTOTP(secret string) (string, error) {
	return gootp.GenerateTOTP(secret, time.Now(), gootp.DefaultConfig())
}

// Verify2FACode verifies an Authenticator 6-digit code with +/-30s clock drift tolerance.
func Verify2FACode(secret, code string) bool {
	return gootp.VerifyTOTP(secret, code, time.Now(), 1, gootp.DefaultConfig())
}
