package auth

import "context"

type contextKey struct{}

var claimsKey = contextKey{}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func GetClaims(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

func GetUserID(ctx context.Context) (string, bool) {
	claims, ok := GetClaims(ctx)
	if !ok || claims == nil {
		return "", false
	}
	return claims.UserID, true
}

func GetUserEmail(ctx context.Context) (string, bool) {
	claims, ok := GetClaims(ctx)
	if !ok || claims == nil {
		return "", false
	}
	return claims.Email, true
}

func GetUserRole(ctx context.Context) (string, bool) {
	claims, ok := GetClaims(ctx)
	if !ok || claims == nil {
		return "", false
	}
	return claims.Role, true
}
