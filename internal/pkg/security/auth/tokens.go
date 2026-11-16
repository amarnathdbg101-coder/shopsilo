package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func (e *Engine) GenerateToken(user *User, tokenType string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    e.cfg.Issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(e.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("auth: failed to sign token: %w", err)
	}
	return signed, nil
}

func (e *Engine) GenerateTokenPair(user *User) (*AuthResponse, error) {
	accessToken, err := e.GenerateToken(user, "access", e.cfg.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := e.GenerateToken(user, "refresh", e.cfg.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(e.cfg.AccessTokenTTL.Seconds()),
		User:         user.ToResponse(),
	}, nil
}

func (e *Engine) ValidateToken(tokenStr string, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(e.cfg.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if expectedType != "" && claims.TokenType != expectedType {
		return nil, ErrInvalidTokenType
	}

	return claims, nil
}
