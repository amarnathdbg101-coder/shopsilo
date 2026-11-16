package reuse

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
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
