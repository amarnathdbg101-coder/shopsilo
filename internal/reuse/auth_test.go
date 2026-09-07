package reuse

import (
	"os"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestHashPasswordAndCheck(t *testing.T) {
	password := "SecretPass123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if hash == password {
		t.Fatalf("hash should not equal plain password")
	}

	if !CheckPasswordHash(password, hash) {
		t.Fatalf("expected password to match hash")
	}

	if CheckPasswordHash("WrongPassword", hash) {
		t.Fatalf("expected wrong password to fail matching hash")
	}
}

func TestValidateStruct(t *testing.T) {
	type Sample struct {
		Email    string `validate:"required,email"`
		Password string `validate:"required,min=6"`
	}

	valid := Sample{
		Email:    "test@example.com",
		Password: "password123",
	}
	if err := ValidateStruct(&valid); err != nil {
		t.Fatalf("expected valid struct to pass, got: %v", err)
	}

	invalid := Sample{
		Email:    "not-an-email",
		Password: "123",
	}
	err := ValidateStruct(&invalid)
	if err == nil {
		t.Fatalf("expected invalid struct to fail validation")
	}
}

func TestGenerateAndVerifyResetToken(t *testing.T) {
	secret := "test-secret-key-123"
	userID := "user-uuid-1234-5678"
	email := "user@example.com"
	initialPasswordHash := "$2a$10$abcdefghijklmnopqrstuvwxyz123456"

	// 1. Generate reset token
	token, err := GenerateResetToken(userID, email, initialPasswordHash, secret)
	if err != nil {
		t.Fatalf("failed to generate reset token: %v", err)
	}

	// 2. Extract user ID
	extractedID, err := ExtractUserIDFromResetToken(token)
	if err != nil {
		t.Fatalf("failed to extract user id: %v", err)
	}
	if extractedID != userID {
		t.Fatalf("expected extracted user id %s, got %s", userID, extractedID)
	}

	// 3. Verify valid token with current password hash
	claims, err := VerifyResetToken(token, secret, initialPasswordHash)
	if err != nil {
		t.Fatalf("failed to verify valid reset token: %v", err)
	}
	if claims.UserID != userID || claims.Email != email {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// 4. Verify token invalidation when password hash changes
	newPasswordHash := "$2a$10$updatedhash9876543210zyxwvutsrqpo"
	_, err = VerifyResetToken(token, secret, newPasswordHash)
	if err == nil {
		t.Fatalf("expected token to be invalid after password hash change")
	}

	// 5. Verify tampered token fails
	tamperedToken := token + "tamper"
	_, err = VerifyResetToken(tamperedToken, secret, initialPasswordHash)
	if err == nil {
		t.Fatalf("expected tampered token to fail verification")
	}
}

func TestGenerateJwt(t *testing.T) {
	_ = os.Setenv("PORT", "8080")
	_ = os.Setenv("DATABASE_URL", "postgres://localhost/test")
	_ = os.Setenv("JWT_SECRET", "super-secret-jwt-key")
	_ = os.Setenv("R2_BUCKET", "bucket")
	_ = os.Setenv("R2_ACCESS_KEY", "access")
	_ = os.Setenv("R2_SECRET_KEY", "secret")
	_ = os.Setenv("R2_ENDPOINT", "endpoint")

	tokenStr, err := GenerateJwt("test-id", "test@example.com", "customer")
	if err != nil {
		t.Fatalf("failed to generate jwt: %v", err)
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte("super-secret-jwt-key"), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("failed to parse generated jwt: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatalf("failed to cast claims")
	}
	if claims["user_id"] != "test-id" || claims["email"] != "test@example.com" || claims["role"] != "customer" {
		t.Fatalf("unexpected claims in token: %+v", claims)
	}
}
