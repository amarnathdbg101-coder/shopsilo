package goclos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// R2Service provides S3-compatible Cloudflare R2 object storage operations with pure Go AWS SigV4.
type R2Service struct {
	client *Client
}

// Endpoint returns the S3-compatible API endpoint for this Cloudflare account.
func (s *R2Service) Endpoint() string {
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", s.client.cfg.AccountID)
}

// PublicURL constructs the public CDN or custom domain URL for an object key.
func (s *R2Service) PublicURL(key string) string {
	key = strings.TrimPrefix(key, "/")
	if s.client.cfg.R2PublicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.client.cfg.R2PublicURL, "/"), key)
	}
	return fmt.Sprintf("/images/%s", key)
}

// Upload uploads raw bytes to Cloudflare R2 with automatic SigV4 signing and returns the accessible public URL.
func (s *R2Service) Upload(ctx context.Context, bucket, key string, data []byte, contentType string) (string, error) {
	if bucket == "" {
		bucket = s.client.cfg.R2Bucket
	}
	if bucket == "" {
		return "", ErrR2BucketRequired
	}
	if len(data) == 0 {
		return "", ErrEmptyPayload
	}
	if s.client.cfg.R2AccessKeyID == "" || s.client.cfg.R2SecretAccess == "" {
		return "", ErrR2CredentialsRequired
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	key = strings.TrimPrefix(key, "/")
	rawURL := fmt.Sprintf("%s/%s/%s", s.Endpoint(), bucket, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, rawURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("go-cloudflare: create upload request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)

	// Sign request via pure Go AWS SigV4
	now := time.Now().UTC()
	signSigV4(req, data, s.client.cfg.R2AccessKeyID, s.client.cfg.R2SecretAccess, "auto", "s3", now)

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("go-cloudflare: r2 upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: r2 upload status %d: %s", ErrAPIError, resp.StatusCode, string(body))
	}

	return s.PublicURL(key), nil
}

// Download retrieves an object from Cloudflare R2 and returns its bytes and MIME Content-Type.
func (s *R2Service) Download(ctx context.Context, bucket, key string) ([]byte, string, error) {
	if bucket == "" {
		bucket = s.client.cfg.R2Bucket
	}
	if bucket == "" {
		return nil, "", ErrR2BucketRequired
	}
	if s.client.cfg.R2AccessKeyID == "" || s.client.cfg.R2SecretAccess == "" {
		return nil, "", ErrR2CredentialsRequired
	}

	key = strings.TrimPrefix(key, "/")
	rawURL := fmt.Sprintf("%s/%s/%s", s.Endpoint(), bucket, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("go-cloudflare: create download request: %w", err)
	}

	now := time.Now().UTC()
	signSigV4(req, nil, s.client.cfg.R2AccessKeyID, s.client.cfg.R2SecretAccess, "auto", "s3", now)

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("go-cloudflare: r2 download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("%w: r2 download status %d: %s", ErrAPIError, resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("go-cloudflare: read download body: %w", err)
	}

	return data, resp.Header.Get("Content-Type"), nil
}

// Delete removes an object from Cloudflare R2.
func (s *R2Service) Delete(ctx context.Context, bucket, key string) error {
	if bucket == "" {
		bucket = s.client.cfg.R2Bucket
	}
	if bucket == "" {
		return ErrR2BucketRequired
	}
	if s.client.cfg.R2AccessKeyID == "" || s.client.cfg.R2SecretAccess == "" {
		return ErrR2CredentialsRequired
	}

	key = strings.TrimPrefix(key, "/")
	rawURL := fmt.Sprintf("%s/%s/%s", s.Endpoint(), bucket, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, rawURL, nil)
	if err != nil {
		return fmt.Errorf("go-cloudflare: create delete request: %w", err)
	}

	now := time.Now().UTC()
	signSigV4(req, nil, s.client.cfg.R2AccessKeyID, s.client.cfg.R2SecretAccess, "auto", "s3", now)

	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("go-cloudflare: r2 delete failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: r2 delete status %d: %s", ErrAPIError, resp.StatusCode, string(body))
	}

	return nil
}

// -------------------------------------------------------------------------
// Pure Go AWS SigV4 Implementation (Zero external AWS dependencies!)
// -------------------------------------------------------------------------

func signSigV4(req *http.Request, payload []byte, accessKey, secretKey, region, service string, t time.Time) {
	dateStr := t.Format("20060102")
	dateTimeStr := t.Format("20060102T150405Z")

	payloadHash := sha256Hex(payload)

	req.Header.Set("x-amz-date", dateTimeStr)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	host := req.URL.Host
	req.Header.Set("Host", host)

	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		host, payloadHash, dateTimeStr,
	)

	// Canonical Request
	canonicalReq := strings.Join([]string{
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	// String to Sign
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStr, region, service)
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		dateTimeStr,
		credentialScope,
		sha256Hex([]byte(canonicalReq)),
	}, "\n")

	// Derive signing key
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStr))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))

	// Signature
	signature := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature,
	)

	req.Header.Set("Authorization", authHeader)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
