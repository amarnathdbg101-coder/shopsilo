package middleware

import (
	"context"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

type contextKey string

const (
	UserIDKey      contextKey = "userId"
	FirebaseUIDKey contextKey = "firebaseUid"
)

func AuthMiddleware(authClient *auth.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				http.Error(w, "Missing or invalid token", http.StatusUnauthorized)
				return
			}

			idToken := strings.TrimPrefix(authHeader, "Bearer ")
			idToken = strings.TrimPrefix(idToken, "bearer ")

			// Verify Firebase ID Token
			token, err := authClient.VerifyIDToken(context.Background(), idToken)
			if err != nil {
				http.Error(w, "Unauthorized: invalid firebase token", http.StatusUnauthorized)
				return
			}

			// Add Firebase UID to context
			ctx := context.WithValue(r.Context(), FirebaseUIDKey, token.UID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
