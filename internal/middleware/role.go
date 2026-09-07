package middleware

import (
	"net/http"
)

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleMap := make(map[string]bool)
	for _, r := range roles {
		roleMap[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserFromContext(r.Context())
			if claims == nil {
				http.Error(w, `{"success":false,"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			if !roleMap[claims.Role] {
				http.Error(w, `{"success":false,"error":"forbidden: insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
