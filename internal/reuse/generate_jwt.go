package reuse

import (
	"errors"
	"fmt"
	"shopMe/internal/utils"
	"time"

	gojwt "github.com/amarnathdbg101-coder/go/go-jwt"
)

// ResetClaims holds password reset token claims.
type ResetClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	ExpiresAt int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
}

// GenerateJwt creates a long-lived session token (30 days) using application JWT secret.
func GenerateJwt(userID, email, role string) (string, error) {
	now := time.Now()
	claims := map[string]any{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     now.Add(30 * 24 * time.Hour).Unix(),
		"iat":     now.Unix(),
	}
	return gojwt.SignCustom([]byte(utils.MustLoad().Jwt), claims)
}

// GenerateResetToken generates a short-lived (15 min) token tied to current password hash.
func GenerateResetToken(userID, email, passwordHash, secret string) (string, error) {
	now := time.Now()
	claims := ResetClaims{
		UserID:    userID,
		Email:     email,
		ExpiresAt: now.Add(15 * time.Minute).Unix(),
		IssuedAt:  now.Unix(),
	}
	signingKey := fmt.Sprintf("%s:%s", secret, passwordHash)
	return gojwt.SignCustom([]byte(signingKey), claims)
}

// VerifyResetToken validates the reset token with the user's password hash.
func VerifyResetToken(tokenStr, secret, passwordHash string) (*ResetClaims, error) {
	signingKey := fmt.Sprintf("%s:%s", secret, passwordHash)
	claims := &ResetClaims{}
	if err := gojwt.VerifyCustom([]byte(signingKey), tokenStr, claims); err != nil {
		return nil, errors.New("invalid or expired reset token")
	}
	return claims, nil
}

// ExtractUserIDFromResetToken extracts the user_id without verifying the signature yet.
func ExtractUserIDFromResetToken(tokenStr string) (string, error) {
	claims := &ResetClaims{}
	if err := gojwt.DecodeUnverified(tokenStr, claims); err != nil {
		return "", errors.New("invalid reset token format")
	}
	if claims.UserID == "" {
		return "", errors.New("missing user_id in reset token")
	}
	return claims.UserID, nil
}

// ParseTokenForRefresh securely validates token signature (even if expired) and extracts user_id for refresh.
func ParseTokenForRefresh(tokenStr string) (string, error) {
	secret := []byte(utils.MustLoad().Jwt)
	engine, err := gojwt.New(secret)
	if err != nil {
		return "", errors.New("invalid secret")
	}

	claims := map[string]any{}
	if err := engine.VerifySignatureOnly(tokenStr, &claims); err != nil {
		return "", errors.New("invalid token signature")
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return "", errors.New("missing user_id in token")
	}

	return userID, nil
}

// PhoneVerificationClaims stores the verified phone and verification record ID.
type PhoneVerificationClaims struct {
	Phone          string `json:"phone"`
	VerificationID string `json:"verification_id"`
	Purpose        string `json:"purpose"` // "phone_registration"
	ExpiresAt      int64  `json:"exp,omitempty"`
	IssuedAt       int64  `json:"iat,omitempty"`
}

// GeneratePhoneVerificationToken generates a short-lived (15 minute) cryptographically signed token
// indicating that the specified phone number has been successfully verified via OTP.
func GeneratePhoneVerificationToken(phone, verificationID, secret string) (string, error) {
	now := time.Now()
	claims := PhoneVerificationClaims{
		Phone:          phone,
		VerificationID: verificationID,
		Purpose:        "phone_registration",
		ExpiresAt:      now.Add(15 * time.Minute).Unix(),
		IssuedAt:       now.Unix(),
	}
	return gojwt.SignCustom([]byte(secret), claims)
}

// VerifyPhoneVerificationToken verifies the signature, expiration, and claims of a phone verification token.
func VerifyPhoneVerificationToken(tokenStr, secret string) (*PhoneVerificationClaims, error) {
	claims := &PhoneVerificationClaims{}
	if err := gojwt.VerifyCustom([]byte(secret), tokenStr, claims); err != nil {
		return nil, errors.New("invalid or expired phone verification token")
	}
	if claims.Purpose != "phone_registration" || claims.Phone == "" || claims.VerificationID == "" {
		return nil, errors.New("invalid phone verification token claims")
	}
	return claims, nil
}
