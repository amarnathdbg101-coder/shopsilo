package gootp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash"
	"math"
	"net/url"
	"strings"
	"time"
)

// Algorithm defines the hash algorithm used for HMAC derivation.
type Algorithm string

const (
	SHA1   Algorithm = "SHA1"
	SHA256 Algorithm = "SHA256"
	SHA512 Algorithm = "SHA512"
)

// Config specifies OTP parameters.
type Config struct {
	Digits      int           // Number of digits (default 6, range 6-8)
	Period      time.Duration // Time-step duration (default 30s)
	Algorithm   Algorithm     // Hash algorithm (default SHA1)
	Issuer      string        // Organization or service name
	AccountName string        // User account or email
}

// DefaultConfig provides standard Google Authenticator compatible settings.
func DefaultConfig() Config {
	return Config{
		Digits:    6,
		Period:    30 * time.Second,
		Algorithm: SHA1,
	}
}

// GenerateSecret generates a cryptographically random Base32 encoded secret.
func GenerateSecret(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 20 // 160-bit standard recommendation
	}

	raw := make([]byte, byteLength)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	// Base32 encoding without padding (compatible with Authenticator apps)
	return strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "="), nil
}

// GenerateHOTP implements RFC 4226 HMAC-based One-Time Password algorithm.
func GenerateHOTP(secretBase32 string, counter uint64, digits int, algo Algorithm) (string, error) {
	if digits < 6 || digits > 8 {
		return "", ErrInvalidCodeLength
	}

	secret, err := decodeBase32(secretBase32)
	if err != nil {
		return "", err
	}

	// 8-byte big-endian counter
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	var hasher func() hash.Hash
	switch algo {
	case SHA256:
		hasher = sha256.New
	case SHA512:
		hasher = sha512.New
	default:
		hasher = sha1.New
	}

	mac := hmac.New(hasher, secret)
	mac.Write(buf)
	h := mac.Sum(nil)

	// Dynamic Truncation (DT)
	offset := h[len(h)-1] & 0x0f
	binaryCode := binary.BigEndian.Uint32(h[offset : offset+4])
	binaryCode &= 0x7fffffff // clear highest bit

	mod := uint32(math.Pow10(digits))
	codeNum := binaryCode % mod

	format := fmt.Sprintf("%%0%dd", digits)
	return fmt.Sprintf(format, codeNum), nil
}

// GenerateTOTP implements RFC 6238 Time-based One-Time Password generation.
func GenerateTOTP(secretBase32 string, t time.Time, cfg Config) (string, error) {
	if cfg.Digits == 0 {
		cfg.Digits = 6
	}
	if cfg.Period <= 0 {
		cfg.Period = 30 * time.Second
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = SHA1
	}

	counter := uint64(t.Unix()) / uint64(cfg.Period.Seconds())
	return GenerateHOTP(secretBase32, counter, cfg.Digits, cfg.Algorithm)
}

// VerifyTOTP validates a user-provided OTP code with clock skew tolerance.
// skewWindows specifies how many periods before and after the current step to check (e.g. 1 allows ±30s).
func VerifyTOTP(secretBase32 string, code string, t time.Time, skewWindows uint, cfg Config) bool {
	code = strings.TrimSpace(code)
	if len(code) == 0 {
		return false
	}

	if cfg.Digits == 0 {
		cfg.Digits = 6
	}
	if cfg.Period <= 0 {
		cfg.Period = 30 * time.Second
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = SHA1
	}

	currentStep := int64(t.Unix()) / int64(cfg.Period.Seconds())
	skew := int64(skewWindows)

	for step := currentStep - skew; step <= currentStep+skew; step++ {
		expected, err := GenerateHOTP(secretBase32, uint64(step), cfg.Digits, cfg.Algorithm)
		if err != nil {
			continue
		}
		if subtleCompare(code, expected) {
			return true
		}
	}

	return false
}

// ProvisioningURI returns a standard "otpauth://" URI formatted for QR code generation.
func ProvisioningURI(secretBase32 string, cfg Config) string {
	if cfg.Digits == 0 {
		cfg.Digits = 6
	}
	if cfg.Period <= 0 {
		cfg.Period = 30 * time.Second
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = SHA1
	}

	label := cfg.AccountName
	if cfg.Issuer != "" {
		label = fmt.Sprintf("%s:%s", cfg.Issuer, cfg.AccountName)
	}

	params := url.Values{}
	params.Set("secret", strings.ToUpper(secretBase32))
	if cfg.Issuer != "" {
		params.Set("issuer", cfg.Issuer)
	}
	if cfg.Algorithm != SHA1 {
		params.Set("algorithm", string(cfg.Algorithm))
	}
	if cfg.Digits != 6 {
		params.Set("digits", fmt.Sprintf("%d", cfg.Digits))
	}
	if cfg.Period != 30*time.Second {
		params.Set("period", fmt.Sprintf("%d", int(cfg.Period.Seconds())))
	}

	return fmt.Sprintf("otpauth://totp/%s?%s", url.PathEscape(label), params.Encode())
}

func decodeBase32(s string) ([]byte, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	// Re-add padding if missing
	if rem := len(s) % 8; rem != 0 {
		s += strings.Repeat("=", 8-rem)
	}
	bytes, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrInvalidSecret
	}
	return bytes, nil
}

func subtleCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
