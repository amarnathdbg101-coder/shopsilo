// Package middleware provides HTTP interceptors.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	golru "github.com/amarnathdbg101-coder/go/go-lru"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
)

// banLRU is a bounded, thread-safe O(1) Least Recently Used cache with per-item 60s TTL.
// Caps memory to 10,000 active entries to eliminate memory exhaustion under DDoS attacks.
var banLRU, _ = golru.New[string, bool](10000, nil)

// InvalidateBanCache clears cached blacklist lookups (called when admin updates blacklist).
func InvalidateBanCache() {
	banLRU.Purge()
}

func checkBannedCached(ctx context.Context, modRepo *repository.ModerationRepo, entityType, entityValue string) (bool, error) {
	key := entityType + ":" + entityValue

	if banned, found := banLRU.Get(key); found {
		return banned, nil
	}

	banned, err := modRepo.IsEntityBanned(ctx, entityType, entityValue)
	if err != nil {
		return false, err
	}

	// Cache lookup for 60 seconds with automatic LRU eviction and expiry
	banLRU.Set(key, banned, 60*time.Second)
	return banned, nil
}

// ExtractClientIP obtains clean client IP address from proxy headers or remote address.
func ExtractClientIP(r *http.Request) string {
	return ExtractIP(r)
}

// BanGuard checks whether the connecting client IP or Device Fingerprint is in the banned_entities table.
func BanGuard(modRepo *repository.ModerationRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := ExtractClientIP(r)
			deviceFP := strings.TrimSpace(r.Header.Get("X-Device-Fingerprint"))

			// Fast cached check IP
			if clientIP != "" && clientIP != "127.0.0.1" && clientIP != "::1" {
				banned, err := checkBannedCached(r.Context(), modRepo, model.EntityTypeIP, clientIP)
				if err == nil && banned {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"success":false,"error":"Access denied: This network has been suspended due to community safety violations."}`))
					return
				}
			}

			// Fast cached check Device Fingerprint
			if deviceFP != "" {
				banned, err := checkBannedCached(r.Context(), modRepo, model.EntityTypeDeviceID, deviceFP)
				if err == nil && banned {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"success":false,"error":"Access denied: This device has been suspended due to community safety violations."}`))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
