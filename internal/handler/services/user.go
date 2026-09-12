// Package services handle business logic.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
	"shopMe/internal/utils"
	"strings"
	"time"
)


type UserService struct {
	repo *repository.UserRepo
}

func NewUserService(repo *repository.UserRepo) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) Register(ctx context.Context, input dto.UserRegisterRequest) (*dto.RegisterResponse, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	// Check if user already exists
	existingUser, err := s.repo.FindByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailTaken
	}

	hashedPassword, err := reuse.HashPassword(input.Password)
	if err != nil {
		return nil, errors.New("failed to secure password")
	}

	user := &model.User{
		Email:        email,
		PasswordHash: hashedPassword,
		FullName:     strings.TrimSpace(input.FullName),
		Phone:        strings.TrimSpace(input.Phone),
		Role:         "customer",
		IsActive:     true,
	}

	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	token, err := reuse.GenerateJwt(createdUser.ID, createdUser.Email, createdUser.Role)
	if err != nil {
		return &dto.RegisterResponse{User: createdUser}, nil
	}

	return &dto.RegisterResponse{
		User:        createdUser,
		AccessToken: token,
	}, nil
}

func (s *UserService) Login(ctx context.Context, input dto.UserLoginRequest) (*dto.TokenResponse, error) {
	identifier := strings.TrimSpace(input.Email)

	user, err := s.repo.FindByEmailOrPhone(ctx, identifier)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	if !reuse.CheckPasswordHash(input.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, err := reuse.GenerateJwt(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	return &dto.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400, // 24 hours in seconds
		User:        user,
	}, nil
}

type googleTokenInfo struct {
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"`
	Error         string `json:"error_description"`
}

func (s *UserService) GoogleLogin(ctx context.Context, input dto.GoogleLoginRequest) (*dto.TokenResponse, error) {
	idToken := strings.TrimSpace(input.IDToken)
	if idToken == "" {
		return nil, errors.New("google id token is required")
	}

	var info googleTokenInfo

	// Support local development mock tokens seamlessly
	if strings.HasPrefix(idToken, "dev_mock_") || idToken == "sample_google_id_token" {
		info = googleTokenInfo{
			Email:   "google.dev@shopme.com",
			Name:    "Google Dev User",
			Picture: "https://lh3.googleusercontent.com/a/default-avatar",
			Sub:     "mock_sub_123456",
		}
	} else {
		// Verify real Google ID Token with Google OAuth2 TokenInfo API
		tokenInfoURL := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", url.QueryEscape(idToken))

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenInfoURL, nil)
		if err != nil {
			return nil, errors.New("failed to build google verification request")
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, errors.New("failed to connect to google verification server")
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("invalid or expired google id token")
		}

		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return nil, errors.New("failed to parse google token info")
		}
	}

	if info.Email == "" {
		return nil, errors.New("google token does not contain a valid email")
	}

	email := strings.ToLower(strings.TrimSpace(info.Email))

	// Find or create user in DB
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			// Auto-register new Google user
			randomSecret := fmt.Sprintf("google_oauth_%s_%d", info.Sub, time.Now().UnixNano())
			dummyHash, hashErr := reuse.HashPassword(randomSecret)
			if hashErr != nil {
				return nil, errors.New("failed to secure user account")
			}

			fullName := strings.TrimSpace(info.Name)
			if fullName == "" {
				fullName = strings.Split(email, "@")[0]
			}

			newUser := &model.User{
				Email:        email,
				PasswordHash: dummyHash,
				FullName:     fullName,
				Role:         "customer",
				IsActive:     true,
			}

			user, err = s.repo.Create(ctx, newUser)
			if err != nil {
				return nil, errors.New("failed to create user from google account")
			}

			if info.Picture != "" {
				_ = s.repo.UpdateAvatar(ctx, user.ID, info.Picture)
				user.AvatarURL = info.Picture
			}
		} else {
			return nil, err
		}
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	// If existing user doesn't have avatar set, update with Google avatar
	if user.AvatarURL == "" && info.Picture != "" {
		_ = s.repo.UpdateAvatar(ctx, user.ID, info.Picture)
		user.AvatarURL = info.Picture
	}

	token, err := reuse.GenerateJwt(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	return &dto.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400,
		User:        user,
	}, nil
}

func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("refresh token is required")
	}

	userID, err := reuse.ParseTokenForRefresh(refreshToken)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	token, err := reuse.GenerateJwt(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	return &dto.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400,
		User:        user,
	}, nil
}

func sendResetEmailViaSMTP(toEmail, resetLink string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	if smtpHost == "" || smtpUser == "" || smtpPass == "" {
		return errors.New("smtp credentials not configured in environment")
	}

	if smtpPort == "" {
		smtpPort = "587"
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	fromHeader := os.Getenv("FROM_EMAIL")
	if fromHeader == "" {
		fromHeader = smtpUser
	}

	subject := "Subject: ShopMe - Reset Your Password\r\n"
	fromLine := fmt.Sprintf("From: %s\r\n", fromHeader)
	toLine := fmt.Sprintf("To: %s\r\n", toEmail)
	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; background-color: #f8fafc; padding: 20px;">
  <div style="max-width: 500px; margin: 0 auto; background: #ffffff; border-radius: 12px; padding: 24px; border: 1px solid #e2e8f0;">
    <h2 style="color: #4f46e5; margin-top: 0;">ShopMe Password Reset</h2>
    <p style="color: #334155; font-size: 15px;">Namaste,</p>
    <p style="color: #334155; font-size: 14px; line-height: 1.5;">
      Aapne apne ShopMe account ka password reset karne ke liye anurodh kiya tha.
    </p>
    <p style="text-align: center; margin: 28px 0;">
      <a href="%s" style="background-color: #4f46e5; color: #ffffff; padding: 12px 24px; border-radius: 8px; text-decoration: none; font-weight: bold; font-size: 14px; display: inline-block;">
        Naya Password Banayein
      </a>
    </p>
    <p style="color: #64748b; font-size: 12px;">
      Yeh link agle 15 minute ke liye valid hai. Agar aapne yeh request nahi ki thi, toh kripya is email ko ignore karein.
    </p>
  </div>
</body>
</html>`, resetLink)

	msg := []byte(fromLine + toLine + subject + mime + body)
	return smtp.SendMail(addr, auth, smtpUser, []string{toEmail}, msg)
}

func (s *UserService) ForgotPassword(ctx context.Context, input dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		// Generic response for privacy/security
		return &dto.ForgotPasswordResponse{
			Message: "If your email is registered, password reset instructions have been sent.",
		}, nil
	}

	jwtSecret := utils.MustLoad().Jwt
	resetToken, err := reuse.GenerateResetToken(user.ID, user.Email, user.PasswordHash, jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate reset token")
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(frontendURL, "/"), resetToken)

	// Attempt real SMTP dispatch if configured
	if err := sendResetEmailViaSMTP(user.Email, resetLink); err == nil {
		return &dto.ForgotPasswordResponse{
			Message: "Password reset link aapke registered email (" + user.Email + ") par bhej diya gaya hai. Kripya apna inbox check karein.",
		}, nil
	}

	// Fallback during local development / when SMTP credentials are not yet added to .env
	log.Printf("[SECURITY/AUTH] SMTP not configured. Reset Link for %s: %s", user.Email, resetLink)
	return &dto.ForgotPasswordResponse{
		Message: fmt.Sprintf("Reset link generate ho gaya hai! Kripya is link par click karke naya password set karein:\n%s", resetLink),
	}, nil
}

func (s *UserService) ResetPassword(ctx context.Context, input dto.ResetPasswordRequest) error {
	userID, err := reuse.ExtractUserIDFromResetToken(input.Token)
	if err != nil {
		return ErrInvalidResetToken
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return ErrInvalidResetToken
	}

	jwtSecret := utils.MustLoad().Jwt
	_, err = reuse.VerifyResetToken(input.Token, jwtSecret, user.PasswordHash)
	if err != nil {
		return ErrInvalidResetToken
	}

	newHashedPassword, err := reuse.HashPassword(input.NewPassword)
	if err != nil {
		return errors.New("failed to secure new password")
	}

	return s.repo.UpdatePassword(ctx, user.ID, newHashedPassword)
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (*dto.UserResponse, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		AvatarURL: user.AvatarURL,
		Role:      user.Role,
	}, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, input dto.UpdateUserProfileRequest) (*dto.UserResponse, error) {
	user, err := s.repo.UpdateProfile(ctx, userID, input.FullName, input.Phone)
	if err != nil {
		return nil, err
	}
	return &dto.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		AvatarURL: user.AvatarURL,
		Role:      user.Role,
	}, nil
}

func (s *UserService) ListUsersForAdmin(ctx context.Context, role, search string, page, limit int) ([]*model.User, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit
	return s.repo.ListUsersForAdmin(ctx, role, search, limit, offset)
}

func (s *UserService) UpdateUserStatusForAdmin(ctx context.Context, userID string, isActive *bool, role *string) (*model.User, error) {
	return s.repo.UpdateUserStatusForAdmin(ctx, userID, isActive, role)
}



