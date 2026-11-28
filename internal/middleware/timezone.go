package middleware

import (
	"net/http"

	"shopMe/internal/utils"
)

// Timezone reads X-Timezone and X-Timezone-Offset headers from the user's device
// and attaches the resolved *time.Location to the request context.
func Timezone(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tzName := r.Header.Get("X-Timezone")
		tzOffset := r.Header.Get("X-Timezone-Offset")
		loc := utils.ResolveLocation(tzName, tzOffset)
		next.ServeHTTP(w, r.WithContext(utils.WithLocation(r.Context(), loc)))
	})
}
