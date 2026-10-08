package reuse

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"strings"
)

// GenerateETag computes a MD5 hex hash for a given data version or payload string
func GenerateETag(dataVersion string) string {
	hash := md5.Sum([]byte(dataVersion))
	return `"` + hex.EncodeToString(hash[:]) + `"`
}

// CheckAndSetETag checks incoming 'If-None-Match' header against dataVersion.
// If matched, it sets 304 Not Modified and returns true.
// If not matched, it sets the ETag header and returns false.
func CheckAndSetETag(w http.ResponseWriter, r *http.Request, dataVersion string) bool {
	if dataVersion == "" {
		return false
	}
	etag := GenerateETag(dataVersion)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, no-cache, must-revalidate")

	ifNoneMatch := r.Header.Get("If-None-Match")
	if ifNoneMatch != "" && (ifNoneMatch == etag || strings.Contains(ifNoneMatch, etag)) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}
