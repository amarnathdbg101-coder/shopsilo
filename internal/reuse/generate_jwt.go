package reuse

import (
	"errors"
	"fmt"
	"shopMe/internal/utils"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ResetClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateJwt(userID, email, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(utils.MustLoad().Jwt))
}

// GenerateResetToken generates a short-lived (15 min) token tied to current password hash
func GenerateResetToken(userID, email, passwordHash, secret string) (string, error) {
	claims := ResetClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	signingKey := fmt.Sprintf("%s:%s", secret, passwordHash)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(signingKey))
}

// VerifyResetToken validates the reset token with the user's password hash
func VerifyResetToken(tokenStr, secret, passwordHash string) (*ResetClaims, error) {
	signingKey := fmt.Sprintf("%s:%s", secret, passwordHash)
	claims := &ResetClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(signingKey), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired reset token")
	}
	return claims, nil
}

// ExtractUserIDFromResetToken extracts the user_id without verifying the signature yet
func ExtractUserIDFromResetToken(tokenStr string) (string, error) {
	parser := jwt.NewParser()
	claims := &ResetClaims{}
	_, _, err := parser.ParseUnverified(tokenStr, claims)
	if err != nil {
		return "", errors.New("invalid reset token format")
	}
	if claims.UserID == "" {
		return "", errors.New("missing user_id in reset token")
	}
	return claims.UserID, nil
}

// ParseTokenForRefresh securely validates token signature (even if expired) and extracts user_id for refresh
func ParseTokenForRefresh(tokenStr string) (string, error) {
	secret := []byte(utils.MustLoad().Jwt)
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	}, jwt.WithoutClaimsValidation())

	if err != nil || token == nil || !token.Valid {
		return "", errors.New("invalid token signature")
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return "", errors.New("missing user_id in token")
	}

	return userID, nil
}

