package tamperseal

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Seal serializes payload into JSON, appends an expiration timestamp, and seals with an HMAC-SHA256 signature.
// Token format: <base64_payload>.<expires_unix>.<hex_hmac>
func Seal(secret []byte, payload any, ttl time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", ErrMissingSecret
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("tamperseal: marshal payload: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	var expUnix int64
	if ttl != 0 {
		expUnix = time.Now().Add(ttl).Unix()
	}

	signingInput := fmt.Sprintf("%s.%d", payloadB64, expUnix)
	signature := computeHMAC([]byte(signingInput), secret)
	sigHex := hex.EncodeToString(signature)

	return fmt.Sprintf("%s.%s", signingInput, sigHex), nil
}

// Unseal verifies the HMAC signature in constant time, checks expiration, and unmarshals payload into dest.
func Unseal(secret []byte, token string, dest any) error {
	if len(secret) == 0 {
		return ErrMissingSecret
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ErrInvalidToken
	}

	payloadB64, expStr, sigHex := parts[0], parts[1], parts[2]
	signingInput := fmt.Sprintf("%s.%s", payloadB64, expStr)

	// Verify HMAC signature in constant time
	expectedSig := computeHMAC([]byte(signingInput), secret)
	actualSig, err := hex.DecodeString(sigHex)
	if err != nil {
		return ErrSignatureMismatch
	}

	if subtle.ConstantTimeCompare(expectedSig, actualSig) != 1 {
		return ErrSignatureMismatch
	}

	// Verify expiration
	expUnix, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return ErrInvalidToken
	}

	if expUnix > 0 && time.Now().Unix() > expUnix {
		return ErrTokenExpired
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return ErrInvalidToken
	}

	if dest != nil {
		if err := json.Unmarshal(payloadBytes, dest); err != nil {
			return fmt.Errorf("tamperseal: unmarshal payload: %w", err)
		}
	}

	return nil
}

// IsExpired checks if a token's expiration timestamp has passed (without verifying the HMAC signature).
func IsExpired(token string) (bool, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false, ErrInvalidToken
	}

	expUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false, ErrInvalidToken
	}

	if expUnix == 0 {
		return false, nil // no expiration
	}

	return time.Now().Unix() > expUnix, nil
}

func computeHMAC(message, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(message)
	return h.Sum(nil)
}
