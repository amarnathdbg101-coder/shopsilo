// Package middleware provides HTTP interceptors.
package middleware

import (
	"net"
	"net/http"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"strings"
)

// ExtractClientIP obtains clean client IP address from proxy headers or remote address.
func ExtractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// BanGuard checks whether the connecting client IP or Device Fingerprint is in the banned_entities table.
func BanGuard(modRepo *repository.ModerationRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := ExtractClientIP(r)
			deviceFP := strings.TrimSpace(r.Header.Get("X-Device-Fingerprint"))

			// Fast check IP
			if clientIP != "" && clientIP != "127.0.0.1" && clientIP != "::1" {
				banned, err := modRepo.IsEntityBanned(r.Context(), model.EntityTypeIP, clientIP)
				if err == nil && banned {
					http.Error(
						w,
						`{"success":false,"error":"Access denied: This network has been suspended due to community safety violations."}`,
						http.StatusForbidden,
					)
					return
				}
			}

			// Fast check Device Fingerprint
			if deviceFP != "" {
				banned, err := modRepo.IsEntityBanned(r.Context(), model.EntityTypeDeviceID, deviceFP)
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
