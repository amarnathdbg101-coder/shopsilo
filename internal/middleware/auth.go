package middleware

import (
	"context"
	"net/http"
	"shopMe/internal/utils"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "userId"

// ---------------- MIDDLEWARE ----------------
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
            http.Error(w, "Missing or invalid token", http.StatusUnauthorized)
            return
        }

        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        tokenStr = strings.TrimPrefix(tokenStr, "bearer ")

        token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
            return []byte(utils.MustLoad().JwtSecret), nil
        })

        if err != nil || !token.Valid {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }

        userIdVal, exists := claims["userId"]
        if !exists {
            http.Error(w, "Unauthorized: userId missing", http.StatusUnauthorized)
            return
        }

        // float64 is the default type for numbers in JWT claims
        userIdFloat, ok := userIdVal.(float64)
        if !ok {
            http.Error(w, "Unauthorized: invalid userId format", http.StatusUnauthorized)
            return
        }
        userId := int(userIdFloat)

        ctx := context.WithValue(r.Context(), UserIDKey, userId)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
