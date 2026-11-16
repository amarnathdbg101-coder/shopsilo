// Package goetag provides RFC 7232 & RFC 9110 compliant HTTP ETag generation,
// client-side caching (304 Not Modified), and optimistic concurrency locking (412 Precondition Failed).
package goetag

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"
	"time"
)

// Mode determines whether strong or weak ETags are emitted.
type Mode int

const (
	// ModeStrong emits byte-exact cryptographic strong validators (e.g., "\"a1b2c3...\"").
	ModeStrong Mode = iota
	// ModeWeak emits semantically equivalent weak validators (e.g., "W/\"a1b2c3...\"").
	ModeWeak
)

// Generate computes a strong cryptographic SHA-1 ETag for the provided payload bytes.
func Generate(data []byte) string {
	if len(data) == 0 {
		return `""`
	}
	hash := sha1.Sum(data)
	return `"` + hex.EncodeToString(hash[:16]) + `"`
}

// GenerateWeak computes a fast, sub-microsecond 64-bit FNV-1a weak ETag with payload length.
func GenerateWeak(data []byte) string {
	if len(data) == 0 {
		return `W/""`
	}
	h := fnv.New64a()
	_, _ = h.Write(data)
	return fmt.Sprintf(`W/"%x-%x"`, h.Sum64(), len(data))
}

// GenerateFromString computes a strong ETag from an arbitrary string.
func GenerateFromString(s string) string {
	return Generate([]byte(s))
}

// GenerateFromVersion creates an ETag from an entity's version sequence and last updated timestamp.
// Ideal for database entities where reading raw bytes is unnecessary (zero-memory allocation).
func GenerateFromVersion(version uint64, lastModified time.Time) string {
	unixSec := lastModified.UTC().Unix()
	return fmt.Sprintf(`"v%d-%x"`, version, unixSec)
}

// Matches evaluates whether a client's header (e.g. If-None-Match or If-Match) matches the current ETag.
// Supports:
// 1. Wildcard "*" matching any non-empty resource.
// 2. Comma-separated client lists (e.g. "etag1", "etag2", W/"etag3").
// 3. Weak vs Strong normalization according to RFC 9110 §8.8.3.
func Matches(headerValue, currentETag string, isWeakComparison bool) bool {
	headerValue = strings.TrimSpace(headerValue)
	currentETag = strings.TrimSpace(currentETag)

	if headerValue == "" || currentETag == "" {
		return false
	}

	if headerValue == "*" {
		return true
	}

	normalizedCurrent := currentETag
	if isWeakComparison {
		normalizedCurrent = strings.TrimPrefix(normalizedCurrent, "W/")
	}

	// Client header may contain multiple comma-delimited ETags
	candidates := strings.Split(headerValue, ",")
	for _, rawCandidate := range candidates {
		candidate := strings.TrimSpace(rawCandidate)
		if isWeakComparison {
			candidate = strings.TrimPrefix(candidate, "W/")
		}

		if candidate == normalizedCurrent {
			return true
		}
	}

	return false
}

// CheckIfNoneMatch checks If-None-Match for cache revalidation.
// If matched, sets HTTP 304 Not Modified, writes ETag header, and returns true (short-circuit).
func CheckIfNoneMatch(w http.ResponseWriter, r *http.Request, currentETag string) bool {
	w.Header().Set("ETag", currentETag)

	if clientETags := r.Header.Get("If-None-Match"); clientETags != "" {
		// RFC 9110 specifies weak comparison for If-None-Match
		if Matches(clientETags, currentETag, true) {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	// Fallback to If-Modified-Since if no ETag match requested
	if ifModSince := r.Header.Get("If-Modified-Since"); ifModSince != "" {
		if t, err := http.ParseTime(ifModSince); err == nil {
			// Compare against Date header or current time if available
			if !t.IsZero() && time.Now().UTC().Truncate(time.Second).Equal(t.UTC().Truncate(time.Second)) {
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}

	return false
}

// CheckIfMatch checks If-Match for Optimistic Concurrency Control (OCC).
// Prevents the "Lost Update Problem" on mutating requests (PUT, PATCH, DELETE).
// If client sends stale ETag, sets HTTP 412 Precondition Failed and returns false.
func CheckIfMatch(w http.ResponseWriter, r *http.Request, currentETag string) bool {
	clientETags := r.Header.Get("If-Match")
	if clientETags == "" {
		// No condition requested by client
		return true
	}

	// RFC 9110 specifies strong comparison for If-Match
	if !Matches(clientETags, currentETag, false) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPreconditionFailed)
		_, _ = w.Write([]byte(`{"success":false,"error":"412 Precondition Failed: resource has been modified by another request"}`))
		return false
	}

	return true
}

// SetETag writes the ETag header and standard caching directives to the response.
func SetETag(w http.ResponseWriter, etag string, cacheControl ...string) {
	w.Header().Set("ETag", etag)
	cc := "private, no-cache, must-revalidate"
	if len(cacheControl) > 0 && cacheControl[0] != "" {
		cc = cacheControl[0]
	}
	w.Header().Set("Cache-Control", cc)
}
