// Package reuse provides reusable application helpers and utilities.
package reuse

import (
	"errors"
	"time"

	tamperseal "github.com/amarnathdbg101-coder/go/go-tamperseal"
	"shopMe/internal/utils"
)

func getMasterSecret() []byte {
	cfg := utils.MustLoad()
	if cfg.Jwt != "" {
		return []byte(cfg.Jwt)
	}
	return []byte("shopsilo-master-cryptographic-fallback-secret-2026")
}

// SealData cryptographically signs and seals an arbitrary data struct with HMAC-SHA256 and expiration.
// Used for tamper-proof stateless action tokens (password resets, customer khata download tokens, email verifications).
func SealData(payload any, ttl time.Duration) (string, error) {
	return tamperseal.Seal(getMasterSecret(), payload, ttl)
}

// UnsealData verifies signature authenticity in constant time and checks expiration before decoding payload.
// If payload was tampered with or expired, it returns an explicit error.
func UnsealData(token string, dest any) error {
	if token == "" {
		return errors.New("empty token")
	}
	return tamperseal.Unseal(getMasterSecret(), token, dest)
}
