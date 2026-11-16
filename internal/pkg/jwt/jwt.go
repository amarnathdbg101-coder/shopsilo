package gojwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var defaultHeaderB64 = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// Engine signs and verifies HMAC-SHA256 (HS256) JSON Web Tokens.
type Engine struct {
	secret []byte
	leeway time.Duration
}

// New constructs an Engine with secret key and optional clock skew tolerance.
func New(secret []byte, leeway ...time.Duration) (*Engine, error) {
	if len(secret) == 0 {
		return nil, ErrMissingSecret
	}

	var l time.Duration
	if len(leeway) > 0 {
		l = leeway[0]
	}

	secCopy := make([]byte, len(secret))
	copy(secCopy, secret)

	return &Engine{
		secret: secCopy,
		leeway: l,
	}, nil
}

// Sign serializes and cryptographically signs a Claims struct into a JWT string.
func (e *Engine) Sign(claims Claims) (string, error) {
	if claims.IssuedAt == 0 {
		claims.IssuedAt = time.Now().Unix()
	}

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("jwt: marshal claims: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signingInput := defaultHeaderB64 + "." + payloadB64

	h := hmac.New(sha256.New, e.secret)
	h.Write([]byte(signingInput))
	signature := h.Sum(nil)
	sigB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + sigB64, nil
}

// SignCustom serializes any claims struct or map and cryptographically signs it into a JWT string.
func (e *Engine) SignCustom(claims any) (string, error) {
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("jwt: marshal custom claims: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signingInput := defaultHeaderB64 + "." + payloadB64

	h := hmac.New(sha256.New, e.secret)
	h.Write([]byte(signingInput))
	signature := h.Sum(nil)
	sigB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + sigB64, nil
}

// Verify decodes, authenticates, and validates a JWT token string into *Claims.
func (e *Engine) Verify(tokenStr string) (*Claims, error) {
	payloadBytes, err := e.VerifyRaw(tokenStr)
	if err != nil {
		return nil, err
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("jwt: unmarshal claims: %w", err)
	}

	return &claims, nil
}

// VerifyRaw authenticates a JWT token string, verifies signature and temporal validity,
// and returns the raw decoded payload bytes.
func (e *Engine) VerifyRaw(tokenStr string) ([]byte, error) {
	payloadBytes, err := e.verifySignatureBytes(tokenStr)
	if err != nil {
		return nil, err
	}

	// Validate temporal validity if "exp" or "nbf" exists
	var temporal struct {
		ExpiresAt int64 `json:"exp,omitempty"`
		NotBefore int64 `json:"nbf,omitempty"`
	}
	if err := json.Unmarshal(payloadBytes, &temporal); err == nil {
		now := time.Now().Unix()
		leewaySec := int64(e.leeway.Seconds())
		if temporal.ExpiresAt > 0 && now-leewaySec > temporal.ExpiresAt {
			return nil, ErrTokenExpired
		}
		if temporal.NotBefore > 0 && now+leewaySec < temporal.NotBefore {
			return nil, ErrTokenNotValidYet
		}
	}

	return payloadBytes, nil
}

// VerifyCustom authenticates a JWT token and unmarshals its payload into claimsDst.
func (e *Engine) VerifyCustom(tokenStr string, claimsDst any) error {
	payloadBytes, err := e.VerifyRaw(tokenStr)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(payloadBytes, claimsDst); err != nil {
		return fmt.Errorf("jwt: unmarshal claims: %w", err)
	}
	return nil
}

// VerifySignatureOnly validates HMAC-SHA256 signature without enforcing token expiration.
func (e *Engine) VerifySignatureOnly(tokenStr string, claimsDst any) error {
	payloadBytes, err := e.verifySignatureBytes(tokenStr)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(payloadBytes, claimsDst); err != nil {
		return fmt.Errorf("jwt: unmarshal claims: %w", err)
	}
	return nil
}

func (e *Engine) verifySignatureBytes(tokenStr string) ([]byte, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	headerB64, payloadB64, sigB64 := parts[0], parts[1], parts[2]
	signingInput := headerB64 + "." + payloadB64

	expectedSigBytes := hmacSHA256([]byte(signingInput), e.secret)
	actualSigBytes, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, ErrInvalidSignature
	}

	if subtle.ConstantTimeCompare(expectedSigBytes, actualSigBytes) != 1 {
		return nil, ErrInvalidSignature
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return payloadBytes, nil
}

// DecodeUnverified decodes the claims payload of a JWT string without verifying cryptographic signature.
// WARNING: This does NOT guarantee token authenticity. Always verify signatures before trusting sensitive data.
func DecodeUnverified(tokenStr string, claimsDst any) error {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrInvalidToken
	}

	if err := json.Unmarshal(payloadBytes, claimsDst); err != nil {
		return fmt.Errorf("jwt: unmarshal unverified claims: %w", err)
	}
	return nil
}

func hmacSHA256(message, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(message)
	return h.Sum(nil)
}

// Sign creates a signed JWT using Default settings.
func Sign(secret []byte, claims Claims) (string, error) {
	e, err := New(secret)
	if err != nil {
		return "", err
	}
	return e.Sign(claims)
}

// Verify authenticates a JWT using Default settings.
func Verify(secret []byte, tokenStr string) (*Claims, error) {
	e, err := New(secret)
	if err != nil {
		return nil, err
	}
	return e.Verify(tokenStr)
}

// SignCustom creates a signed JWT from any claims struct or map using default settings.
func SignCustom(secret []byte, claims any) (string, error) {
	e, err := New(secret)
	if err != nil {
		return "", err
	}
	return e.SignCustom(claims)
}

// VerifyCustom verifies and unmarshals a JWT into claimsDst using default settings.
func VerifyCustom(secret []byte, tokenStr string, claimsDst any) error {
	e, err := New(secret)
	if err != nil {
		return err
	}
	return e.VerifyCustom(tokenStr, claimsDst)
}
