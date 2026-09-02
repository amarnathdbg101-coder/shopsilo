package middleware

import (
	"context"
	"net/http"
	"shopMe/internal/utils"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDContextKey contextKey = "userId"

// Load once at app start
var jwtSecret = []byte(utils.MustLoad().Jwt) // utils.MustLoad().Jwt

func JWTAuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tokenString := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
        if tokenString == "" {
            http.Error(w, "Missing token", http.StatusUnauthorized)
            return
        }

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return jwtSecret, nil
        })

        if err != nil || !token.Valid {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            http.Error(w, "Invalid claims", http.StatusUnauthorized)
            return
        }

        idVal, exists := claims["userId"]
        if !exists {
            http.Error(w, "User ID missing in token", http.StatusUnauthorized)
            return
        }

        userId := int(idVal.(float64))

        // Inject userID into context
        ctx := context.WithValue(r.Context(), UserIDContextKey, userId)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
