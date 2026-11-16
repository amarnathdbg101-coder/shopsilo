package gourlsigner

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"time"
)

// Config specifies URL signing parameters.
type Config struct {
	Secret         []byte
	ExpiresParam   string
	SignatureParam string
}

// Signer coordinates signing and verification of URLs.
type Signer struct {
	secret         []byte
	expiresParam   string
	signatureParam string
}

// New creates a Signer with default parameter names ("expires", "signature").
func New(secret []byte) (*Signer, error) {
	if len(secret) == 0 {
		return nil, ErrEmptySecret
	}

	return &Signer{
		secret:         secret,
		expiresParam:   "expires",
		signatureParam: "signature",
	}, nil
}

// NewWithConfig creates a Signer with custom parameter names.
func NewWithConfig(cfg Config) (*Signer, error) {
	if len(cfg.Secret) == 0 {
		return nil, ErrEmptySecret
	}
	if cfg.ExpiresParam == "" {
		cfg.ExpiresParam = "expires"
	}
	if cfg.SignatureParam == "" {
		cfg.SignatureParam = "signature"
	}

	return &Signer{
		secret:         cfg.Secret,
		expiresParam:   cfg.ExpiresParam,
		signatureParam: cfg.SignatureParam,
	}, nil
}

// Sign appends an expiration timestamp and HMAC-SHA256 signature to the URL.
// If ttl <= 0, the URL is signed without an expiration limit.
func (s *Signer) Sign(rawURL string, ttl time.Duration) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Del(s.signatureParam)

	if ttl != 0 {
		expiresAt := time.Now().Add(ttl).Unix()
		q.Set(s.expiresParam, strconv.FormatInt(expiresAt, 10))
	} else {
		q.Del(s.expiresParam)
	}

	u.RawQuery = q.Encode()

	// Compute HMAC over scheme + host + path + encoded sorted query
	sig := s.computeHMAC(u.String())
	q.Set(s.signatureParam, sig)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// Verify validates that the signature is authentic and the URL has not expired.
func (s *Signer) Verify(signedURL string) error {
	u, err := url.Parse(signedURL)
	if err != nil {
		return err
	}

	q := u.Query()
	providedSig := q.Get(s.signatureParam)
	if providedSig == "" {
		return ErrMissingSignature
	}

	// Check expiration
	if expiresStr := q.Get(s.expiresParam); expiresStr != "" {
		expUnix, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil {
			return ErrExpired
		}
		if time.Now().Unix() > expUnix {
			return ErrExpired
		}
	}

	// Strip signature query param to reconstruct original payload string
	q.Del(s.signatureParam)
	u.RawQuery = q.Encode()

	expectedSig := s.computeHMAC(u.String())

	expectedBytes, err := hex.DecodeString(expectedSig)
	if err != nil {
		return ErrInvalidSignature
	}

	providedBytes, err := hex.DecodeString(providedSig)
	if err != nil {
		return ErrInvalidSignature
	}

	// Constant-time comparison
	if !hmac.Equal(expectedBytes, providedBytes) {
		return ErrInvalidSignature
	}

	return nil
}

func (s *Signer) computeHMAC(target string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(target))
	return hex.EncodeToString(mac.Sum(nil))
}
