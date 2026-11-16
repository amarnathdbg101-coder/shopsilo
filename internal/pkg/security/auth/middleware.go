package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func (e *Engine) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeJSONError(w, http.StatusUnauthorized, "Invalid Authorization header format. Expected 'Bearer <token>'")
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		claims, err := e.ValidateToken(tokenStr, "access")
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				writeJSONError(w, http.StatusUnauthorized, "Access token has expired")
				return
			}
			writeJSONError(w, http.StatusUnauthorized, "Invalid or unauthorized token")
			return
		}

		ctx := WithClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaims(r.Context())
			if !ok || claims == nil {
				writeJSONError(w, http.StatusUnauthorized, "Authentication required prior to role verification")
				return
			}

			userRole := claims.Role
			isAllowed := false
			for _, allowed := range allowedRoles {
				if strings.EqualFold(userRole, allowed) {
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				writeJSONError(w, http.StatusForbidden, "Forbidden: Insufficient privileges for this resource")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   http.StatusText(statusCode),
		"message": message,
	})
}
