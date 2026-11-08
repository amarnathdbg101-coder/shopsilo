// Package middleware to reduce repetition code.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	gojwt "github.com/amarnathdbg101-coder/go/go-jwt"
)

type contextKey string

const ContextKeyUser = contextKey("user")

// Claims represents user authentication credentials in request context.
type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
}

// JWTAuth validates JWT bearer token and injects verified *Claims into request context.
func JWTAuth(secret string) func(http.Handler) http.Handler {
	secBytes := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				tokenStr := r.URL.Query().Get("token")
				if tokenStr == "" {
					tokenStr = r.URL.Query().Get("access_token")
				}
				if tokenStr != "" {
					authHeader = "Bearer " + tokenStr
				}
			}

			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"error":"missing authorization header"}`))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"error":"invalid authorization header format"}`))
				return
			}

			tokenStr := strings.TrimSpace(parts[1])
			claims := &Claims{}

			if err := gojwt.VerifyCustom(secBytes, tokenStr, claims); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"error":"invalid or expired token"}`))
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUser, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalJWTAuth validates JWT token if present, attaching Claims to request context.
// Allows unauthenticated guest requests to proceed with claims = nil.
func OptionalJWTAuth(secret string) func(http.Handler) http.Handler {
	secBytes := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				tokenStr := r.URL.Query().Get("token")
				if tokenStr == "" {
					tokenStr = r.URL.Query().Get("access_token")
				}
				if tokenStr != "" {
					authHeader = "Bearer " + tokenStr
				}
			}

			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				next.ServeHTTP(w, r)
				return
			}

			tokenStr := strings.TrimSpace(parts[1])
			claims := &Claims{}

			if err := gojwt.VerifyCustom(secBytes, tokenStr, claims); err == nil {
				ctx := context.WithValue(r.Context(), ContextKeyUser, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext retrieves authenticated user Claims from request context.
func GetUserFromContext(ctx context.Context) *Claims {
	if claims, ok := ctx.Value(ContextKeyUser).(*Claims); ok {
		return claims
	}
	return nil
}

// GenerateToken creates an HMAC-SHA256 signed JWT token valid for 24 hours.
func GenerateToken(userID, email, role, secret string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		ExpiresAt: now.Add(24 * time.Hour).Unix(),
		IssuedAt:  now.Unix(),
	}

	return gojwt.SignCustom([]byte(secret), claims)
}
