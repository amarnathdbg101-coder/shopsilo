// Package reuse provides reusable application helpers and utilities.
package reuse

import (
	"net/http"

	goetag "github.com/amarnathdbg101-coder/go/go-etag"
)

// GenerateETag computes a strong cryptographic ETag for a given data version or payload string.
func GenerateETag(dataVersion string) string {
	return goetag.GenerateFromString(dataVersion)
}

// CheckAndSetETag checks incoming 'If-None-Match' header against dataVersion.
// If matched, it sets 304 Not Modified and returns true (saving egress bandwidth).
// If not matched, it sets the ETag header and returns false.
func CheckAndSetETag(w http.ResponseWriter, r *http.Request, dataVersion string) bool {
	if dataVersion == "" {
		return false
	}
	etag := GenerateETag(dataVersion)
	return goetag.CheckIfNoneMatch(w, r, etag)
}

// CheckOptimisticLock verifies incoming 'If-Match' header for safe concurrent updates (Lost Update Prevention).
// If client has a stale ETag, it writes HTTP 412 Precondition Failed and returns false.
func CheckOptimisticLock(w http.ResponseWriter, r *http.Request, currentVersion string) bool {
	etag := GenerateETag(currentVersion)
	return goetag.CheckIfMatch(w, r, etag)
}
