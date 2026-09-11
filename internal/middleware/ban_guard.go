// Package middleware provides HTTP interceptors.
package middleware

import (
	"context"
	"net/http"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"strings"
	"sync"
	"time"
)

type banCacheEntry struct {
	banned    bool
	expiresAt time.Time
}

var (
	banCacheMu sync.RWMutex
	banCache   = make(map[string]banCacheEntry)
)

// InvalidateBanCache clears cached blacklist lookups (called when admin updates blacklist)
func InvalidateBanCache() {
	banCacheMu.Lock()
	banCache = make(map[string]banCacheEntry)
	banCacheMu.Unlock()
}

func checkBannedCached(ctx context.Context, modRepo *repository.ModerationRepo, entityType, entityValue string) (bool, error) {
	key := entityType + ":" + entityValue
	now := time.Now()

	banCacheMu.RLock()
	if entry, found := banCache[key]; found && now.Before(entry.expiresAt) {
		banCacheMu.RUnlock()
		return entry.banned, nil
	}
	banCacheMu.RUnlock()

	banned, err := modRepo.IsEntityBanned(ctx, entityType, entityValue)
	if err != nil {
		return false, err
	}

	banCacheMu.Lock()
	banCache[key] = banCacheEntry{
		banned:    banned,
		expiresAt: now.Add(60 * time.Second),
	}
	if len(banCache) > 5000 {
		for k, v := range banCache {
			if now.After(v.expiresAt) {
				delete(banCache, k)
			}
		}
	}
	banCacheMu.Unlock()

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
					http.Error(
						w,
						`{"success":false,"error":"Access denied: This network has been suspended due to community safety violations."}`,
						http.StatusForbidden,
					)
					return
				}
			}

			// Fast cached check Device Fingerprint
			if deviceFP != "" {
				banned, err := checkBannedCached(r.Context(), modRepo, model.EntityTypeDeviceID, deviceFP)
				if err == nil && banned {
					http.Error(
						w,
						`{"success":false,"error":"Access denied: This device has been suspended due to community safety violations."}`,
						http.StatusForbidden,
					)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
