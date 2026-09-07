// Package services handle business logic.
package services

import (
	"context"
	"errors"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/handler/repository"
	"shopMe/internal/reuse"
	"shopMe/internal/utils"
	"strings"
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
	email := strings.ToLower(strings.TrimSpace(input.Email))

	user, err := s.repo.FindByEmail(ctx, email)
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

func (s *UserService) ForgotPassword(ctx context.Context, input dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		// Generic response for privacy/security
		return &dto.ForgotPasswordResponse{
			Message: "If your email is registered, password reset instructions have been generated.",
		}, nil
	}

	jwtSecret := utils.MustLoad().Jwt
	resetToken, err := reuse.GenerateResetToken(user.ID, user.Email, user.PasswordHash, jwtSecret)
	if err != nil {
		return nil, errors.New("failed to generate reset token")
	}

	return &dto.ForgotPasswordResponse{
		Message:    "Password reset instructions generated successfully. Please use the reset token to set a new password.",
		ResetToken: resetToken,
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

