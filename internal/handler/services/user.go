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
	repo       *repository.UserRepo
	khataRepo  *repository.KhataRepo
	otpRepo    *repository.PhoneVerificationRepo
	smsService SMSService
}

func NewUserService(repo *repository.UserRepo) *UserService {
	return &UserService{
		repo:       repo,
		smsService: NewDefaultSMSService(),
	}
}

func (s *UserService) SetKhataRepo(khataRepo *repository.KhataRepo) {
	s.khataRepo = khataRepo
}

func (s *UserService) SetPhoneVerificationRepo(otpRepo *repository.PhoneVerificationRepo) {
	s.otpRepo = otpRepo
}

func (s *UserService) SetSMSService(smsService SMSService) {
	s.smsService = smsService
}

func (s *UserService) SendRegistrationOTP(ctx context.Context, input dto.SendPhoneOTPRequest) (*dto.SendPhoneOTPResponse, error) {
	cleanPhone, err := utils.NormalizeIndianPhone(input.Phone)
	if err != nil {
		return nil, utils.ErrInvalidPhoneNumber
	}

	// 1. Check if user with this phone already exists
	existingUser, err := s.repo.FindByPhone(ctx, cleanPhone)
	if err == nil && existingUser != nil {
		return nil, ErrPhoneTaken
	}

	if s.otpRepo != nil {
		// 2. Enforce 60-second cooldown per phone to prevent spamming
		latest, err := s.otpRepo.GetLatestByPhone(ctx, cleanPhone)
		if err == nil && latest != nil {
			if time.Since(latest.CreatedAt) < 60*time.Second {
				return nil, ErrOTPCooldown
			}
		}

		// 3. Enforce rolling hourly limit (max 5 requests per hour) to protect against SMS bombing
		recentCount, err := s.otpRepo.CountRecentInWindow(ctx, cleanPhone, 1*time.Hour)
		if err == nil && recentCount >= 5 {
			return nil, ErrOTPHourlyLimitReached
		}
	}

	// 4. Generate cryptographically secure 6-digit OTP
	otp, err := reuse.GenerateCryptoSecureOTP()
	if err != nil {
		return nil, errors.New("failed to generate secure OTP")
	}

	// 5. Store keyed SHA-256 hash in database
	jwtSecret := utils.MustLoad().Jwt
	otpHash := reuse.HashOTP(cleanPhone, otp, jwtSecret)

	if s.otpRepo != nil {
		record := &model.PhoneVerification{
			Phone:       cleanPhone,
			OTPHash:     otpHash,
			Attempts:    0,
			MaxAttempts: 5,
			IsVerified:  false,
			IsConsumed:  false,
			ExpiresAt:   time.Now().Add(5 * time.Minute),
		}
		if err := s.otpRepo.Create(ctx, record); err != nil {
			return nil, errors.New("failed to initiate phone verification")
		}
	}

	channel := strings.ToLower(strings.TrimSpace(input.Channel))
	if channel == "" {
		channel = "whatsapp" // Default to WhatsApp
	}

	// 6. Dispatch OTP via chosen channel (WhatsApp or SMS)
	if s.smsService != nil {
		go func(ch string) {
			if ch == "sms" {
				_ = s.smsService.SendRegistrationOTP(context.Background(), cleanPhone, otp)
			} else {
				_ = s.smsService.SendRegistrationWhatsAppOTP(context.Background(), cleanPhone, otp)
			}
		}(channel)
	}

	channelName := "WhatsApp"
	if channel == "sms" {
		channelName = "SMS"
	}

	return &dto.SendPhoneOTPResponse{
		Message:   fmt.Sprintf("OTP sent successfully to your %s", channelName),
		Phone:     cleanPhone,
		Channel:   channel,
		ExpiresIn: 300, // 5 minutes
		Cooldown:  60,  // 60 seconds
	}, nil
}

func (s *UserService) VerifyRegistrationOTP(ctx context.Context, input dto.VerifyPhoneOTPRequest) (*dto.VerifyPhoneOTPResponse, error) {
	cleanPhone, err := utils.NormalizeIndianPhone(input.Phone)
	if err != nil {
		return nil, utils.ErrInvalidPhoneNumber
	}

	otp := strings.TrimSpace(input.OTP)
	if len(otp) != 6 {
		return nil, ErrInvalidOTP
	}

	if s.otpRepo == nil {
		return nil, errors.New("phone verification service unavailable")
	}

	record, err := s.otpRepo.GetLatestByPhone(ctx, cleanPhone)
	if err != nil || record == nil {
		return nil, ErrOTPNotFoundOrExpired
	}

	// Check if already expired
	if time.Now().After(record.ExpiresAt) {
		return nil, ErrOTPNotFoundOrExpired
	}

	// Check if already reached max attempts
	if record.Attempts >= record.MaxAttempts {
		return nil, ErrOTPMaxAttemptsExceeded
	}

	// Verify keyed hash in constant time
	jwtSecret := utils.MustLoad().Jwt
	valid := reuse.VerifyOTPHash(cleanPhone, otp, jwtSecret, record.OTPHash)
	if !valid {
		newAttempts, _ := s.otpRepo.IncrementAttempts(ctx, record.ID)
		if newAttempts >= record.MaxAttempts {
			return nil, ErrOTPMaxAttemptsExceeded
		}
		return nil, ErrInvalidOTP
	}

	// Mark verified in DB
	if err := s.otpRepo.MarkVerified(ctx, record.ID); err != nil {
		return nil, errors.New("failed to finalize phone verification")
	}

	// Generate 15-minute single-use verification token
	token, err := reuse.GeneratePhoneVerificationToken(cleanPhone, record.ID, jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate verification token")
	}

	return &dto.VerifyPhoneOTPResponse{
		Message:           "Phone verified successfully",
		Phone:             cleanPhone,
		VerificationToken: token,
		ExpiresIn:         900, // 15 minutes
	}, nil
}

func (s *UserService) Register(ctx context.Context, input dto.UserRegisterRequest) (*dto.RegisterResponse, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	// 1. Normalize and validate Indian mobile phone
	cleanPhone, err := utils.NormalizeIndianPhone(input.Phone)
	if err != nil {
		return nil, utils.ErrInvalidPhoneNumber
	}

	// 2. Validate cryptographic verification token
	jwtSecret := utils.MustLoad().Jwt
	claims, err := reuse.VerifyPhoneVerificationToken(input.VerificationToken, jwtSecret)
	if err != nil {
		return nil, ErrInvalidVerificationToken
	}

	// 3. Security Loophole Prevention: Phone binding check
	if claims.Phone != cleanPhone {
		return nil, ErrPhoneVerificationMismatch
	}

	// 4. Security Loophole Prevention: Atomic single-use consumption
	if s.otpRepo != nil {
		consumed, err := s.otpRepo.ConsumeVerification(ctx, claims.VerificationID, cleanPhone)
		if err != nil || !consumed {
			return nil, ErrTokenAlreadyUsed
		}
	}

	// 5. Check if email already exists
	existingUser, err := s.repo.FindByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailTaken
	}

	// 6. Check if phone already registered
	existingPhoneUser, err := s.repo.FindByPhone(ctx, cleanPhone)
	if err == nil && existingPhoneUser != nil {
		return nil, ErrPhoneTaken
	}

	hashedPassword, err := reuse.HashPassword(input.Password)
	if err != nil {
		return nil, errors.New("failed to secure password")
	}

	user := &model.User{
		Email:        email,
		PasswordHash: hashedPassword,
		FullName:     strings.TrimSpace(input.FullName),
		Phone:        cleanPhone,
		Role:         "customer",
		IsActive:     true,
	}

	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, ErrEmailTaken
		}
		if errors.Is(err, repository.ErrDuplicatePhone) {
			return nil, ErrPhoneTaken
		}
		return nil, err
	}

	// Auto-link any existing offline shop khatas to this user
	if s.khataRepo != nil && createdUser.Phone != "" {
		_, _ = s.khataRepo.AutoLinkCustomerKhatas(ctx, createdUser.ID, createdUser.Phone)
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

	// Auto-link any pending shop khatas on login as well
	if s.khataRepo != nil && user.Phone != "" {
		_, _ = s.khataRepo.AutoLinkCustomerKhatas(ctx, user.ID, user.Phone)
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
	Iss           string `json:"iss"`
	Aud           string `json:"aud"`
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Error         string `json:"error_description"`
}

func (s *UserService) GoogleLogin(ctx context.Context, input dto.GoogleLoginRequest) (*dto.GoogleLoginResponse, error) {
	idToken := strings.TrimSpace(input.IDToken)
	if idToken == "" {
		return nil, errors.New("google id token is required")
	}

	var info googleTokenInfo

	// Support mock tokens ONLY during automated unit tests (ENV=test)
	if strings.HasPrefix(idToken, "dev_mock_") && os.Getenv("ENV") == "test" {
		info = googleTokenInfo{
			Iss:           "https://accounts.google.com",
			Email:         "google.dev@shopme.com",
			EmailVerified: "true",
			Name:          "Google Dev User",
			Picture:       "https://lh3.googleusercontent.com/a/default-avatar",
			Sub:           "mock_sub_123456",
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

		// Security Check 1: Issuer validation
		if info.Iss != "accounts.google.com" && info.Iss != "https://accounts.google.com" {
			return nil, errors.New("invalid google token issuer")
		}

		// Security Check 2: Audience validation against configured Client ID
		expectedClientID := os.Getenv("GOOGLE_CLIENT_ID")
		if expectedClientID != "" && info.Aud != expectedClientID {
			return nil, errors.New("google token audience mismatch: token was not issued for this application")
		}

		// Security Check 3: Email verified check
		if info.EmailVerified != "true" && info.EmailVerified != "1" {
			return nil, errors.New("google account email is not verified")
		}
	}

	if info.Email == "" {
		return nil, errors.New("google token does not contain a valid email")
	}

	email := strings.ToLower(strings.TrimSpace(info.Email))
	fullName := strings.TrimSpace(info.Name)
	if fullName == "" {
		fullName = strings.Split(email, "@")[0]
	}

	// 1. Check if user already exists in DB
	var user *model.User
	if s.repo != nil {
		var err error
		user, err = s.repo.FindByEmail(ctx, email)
		if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
			return nil, err
		}
	}

	// 2. Case A: Existing user -> Log in immediately
	if user != nil {
		if !user.IsActive {
			return nil, ErrAccountInactive
		}

		// If existing user has no avatar set, update with Google picture
		if user.AvatarURL == "" && info.Picture != "" && s.repo != nil {
			_ = s.repo.UpdateAvatar(ctx, user.ID, info.Picture)
			user.AvatarURL = info.Picture
		}

		token, err := reuse.GenerateJwt(user.ID, user.Email, user.Role)
		if err != nil {
			return nil, errors.New("failed to generate access token")
		}

		return &dto.GoogleLoginResponse{
			RequiresPhone: false,
			AccessToken:   token,
			TokenType:     "Bearer",
			ExpiresIn:     86400,
			User:          user,
		}, nil
	}

	// 3. Case B: New user registering for the first time
	cleanPhone := ""
	if strings.TrimSpace(input.Phone) != "" && strings.TrimSpace(input.VerificationToken) != "" {
		var err error
		cleanPhone, err = utils.NormalizeIndianPhone(input.Phone)
		if err != nil {
			return nil, utils.ErrInvalidPhoneNumber
		}

		jwtSecret := utils.MustLoad().Jwt
		claims, err := reuse.VerifyPhoneVerificationToken(input.VerificationToken, jwtSecret)
		if err != nil {
			return nil, ErrInvalidVerificationToken
		}

		if claims.Phone != cleanPhone {
			return nil, ErrPhoneVerificationMismatch
		}

		if s.otpRepo != nil {
			consumed, err := s.otpRepo.ConsumeVerification(ctx, claims.VerificationID, cleanPhone)
			if err != nil || !consumed {
				return nil, ErrTokenAlreadyUsed
			}
		}

		// Ensure phone is not registered by another user
		if s.repo != nil {
			existingPhoneUser, err := s.repo.FindByPhone(ctx, cleanPhone)
			if err == nil && existingPhoneUser != nil {
				return nil, ErrPhoneTaken
			}
		}
	}

	randomSecret := fmt.Sprintf("google_oauth_%s_%d", info.Sub, time.Now().UnixNano())
	dummyHash, err := reuse.HashPassword(randomSecret)
	if err != nil {
		return nil, errors.New("failed to secure user account")
	}

	userRole := "customer"
	if strings.ToLower(strings.TrimSpace(input.Role)) == "shop" {
		userRole = "shop"
	}

	newUser := &model.User{
		Email:        email,
		PasswordHash: dummyHash,
		FullName:     fullName,
		Phone:        cleanPhone,
		AvatarURL:    info.Picture,
		Role:         userRole,
		IsActive:     true,
	}

	var createdUser *model.User
	if s.repo != nil {
		var err error
		createdUser, err = s.repo.Create(ctx, newUser)
		if err != nil {
			if errors.Is(err, repository.ErrDuplicateEmail) {
				return nil, ErrEmailTaken
			}
			if errors.Is(err, repository.ErrDuplicatePhone) {
				return nil, ErrPhoneTaken
			}
			return nil, errors.New("failed to create user from google account")
		}
	} else {
		createdUser = newUser
	}

	// Auto-link any existing offline shop khatas to this user
	if s.khataRepo != nil && createdUser.Phone != "" {
		_, _ = s.khataRepo.AutoLinkCustomerKhatas(ctx, createdUser.ID, createdUser.Phone)
	}

	token, err := reuse.GenerateJwt(createdUser.ID, createdUser.Email, createdUser.Role)
	if err != nil {
		return nil, errors.New("failed to generate access token")
	}

	return &dto.GoogleLoginResponse{
		RequiresPhone: cleanPhone == "",
		Email:         email,
		FullName:      fullName,
		AvatarURL:     info.Picture,
		AccessToken:   token,
		TokenType:     "Bearer",
		ExpiresIn:     86400,
		User:          createdUser,
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




