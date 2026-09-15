package services

import (
	"context"
	"errors"
	"os"
	"shopMe/internal/handler/dto"
	"shopMe/internal/reuse"
	"shopMe/internal/utils"
	"testing"
)

func TestRegisterSecurityValidations(t *testing.T) {
	_ = os.Setenv("PORT", "8080")
	_ = os.Setenv("DATABASE_URL", "postgres://localhost/test")
	_ = os.Setenv("JWT_SECRET", "super-secret-jwt-key")
	_ = os.Setenv("R2_BUCKET", "bucket")
	_ = os.Setenv("R2_ACCESS_KEY", "access")
	_ = os.Setenv("R2_SECRET_KEY", "secret")
	_ = os.Setenv("R2_ENDPOINT", "endpoint")

	jwtSecret := utils.MustLoad().Jwt
	service := NewUserService(nil)

	t.Run("Rejects Invalid Phone Format", func(t *testing.T) {
		req := dto.UserRegisterRequest{
			FullName:          "Test User",
			Email:             "test@example.com",
			Password:          "password123",
			Phone:             "12345", // invalid format
			VerificationToken: "dummy-token",
		}

		_, err := service.Register(context.Background(), req)
		if !errors.Is(err, utils.ErrInvalidPhoneNumber) {
			t.Fatalf("expected ErrInvalidPhoneNumber, got: %v", err)
		}
	})

	t.Run("Rejects Invalid or Expired Verification Token", func(t *testing.T) {
		req := dto.UserRegisterRequest{
			FullName:          "Test User",
			Email:             "test@example.com",
			Password:          "password123",
			Phone:             "9876543210",
			VerificationToken: "invalid.jwt.token",
		}

		_, err := service.Register(context.Background(), req)
		if !errors.Is(err, ErrInvalidVerificationToken) {
			t.Fatalf("expected ErrInvalidVerificationToken, got: %v", err)
		}
	})

	t.Run("Rejects Phone Mismatch Between Token and Payload (Loophole Prevention)", func(t *testing.T) {
		// Attacker generates token for phone A: 9876543210
		tokenForPhoneA, err := reuse.GeneratePhoneVerificationToken("9876543210", "rec-uuid-1", jwtSecret)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		// Attacker tries to register phone B: 9999999999 using phone A's token
		req := dto.UserRegisterRequest{
			FullName:          "Attacker",
			Email:             "attacker@example.com",
			Password:          "password123",
			Phone:             "9999999999", // mismatch!
			VerificationToken: tokenForPhoneA,
		}

		_, err = service.Register(context.Background(), req)
		if !errors.Is(err, ErrPhoneVerificationMismatch) {
			t.Fatalf("expected ErrPhoneVerificationMismatch, got: %v", err)
		}
	})
}

func TestGoogleLoginFlow(t *testing.T) {
	t.Setenv("ENV", "test")
	_ = os.Setenv("PORT", "8080")
	_ = os.Setenv("DATABASE_URL", "postgres://localhost/test")
	_ = os.Setenv("JWT_SECRET", "super-secret-jwt-key")
	_ = os.Setenv("R2_BUCKET", "bucket")
	_ = os.Setenv("R2_ACCESS_KEY", "access")
	_ = os.Setenv("R2_SECRET_KEY", "secret")
	_ = os.Setenv("R2_ENDPOINT", "endpoint")

	jwtSecret := utils.MustLoad().Jwt
	service := NewUserService(nil)

	t.Run("Rejects Empty Google ID Token", func(t *testing.T) {
		_, err := service.GoogleLogin(context.Background(), dto.GoogleLoginRequest{
			IDToken: "",
		})
		if err == nil {
			t.Fatal("expected error for empty google id token")
		}
	})

	t.Run("New Google User Without Phone Returns RequiresPhone", func(t *testing.T) {
		// Mock token starts with dev_mock_ for test environments
		resp, err := service.GoogleLogin(context.Background(), dto.GoogleLoginRequest{
			IDToken: "dev_mock_google_token_123",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.RequiresPhone {
			t.Fatal("expected RequiresPhone to be true for new google user without verified phone")
		}
		if resp.Email != "google.dev@shopme.com" {
			t.Fatalf("expected email google.dev@shopme.com, got %s", resp.Email)
		}
		if resp.AccessToken == "" {
			t.Fatal("expected non-empty access token for google login")
		}
		if resp.User == nil {
			t.Fatal("expected non-nil user object in google login response")
		}
	})

	t.Run("New Google User With Phone Mismatch Rejects", func(t *testing.T) {
		// Token issued for 9876543210
		token, err := reuse.GeneratePhoneVerificationToken("9876543210", "rec-1", jwtSecret)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		// User tries to pass different phone 9111111111
		_, err = service.GoogleLogin(context.Background(), dto.GoogleLoginRequest{
			IDToken:           "dev_mock_google_token_123",
			Phone:             "9111111111",
			VerificationToken: token,
		})
		if !errors.Is(err, ErrPhoneVerificationMismatch) {
			t.Fatalf("expected ErrPhoneVerificationMismatch, got %v", err)
		}
	})
}

