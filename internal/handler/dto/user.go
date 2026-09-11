// Package dto handler request and response struct.
package dto

import "shopMe/internal/handler/model"

type UserRegisterRequest struct {
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=72"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,max=20"`
}

type RegisterResponse struct {
	User        *model.User `json:"user"`
	AccessToken string      `json:"access_token,omitempty"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required"` // supports email or mobile phone
	Password string `json:"password" validate:"required"`
}

type TokenResponse struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int64       `json:"expires_in"`
	User        *model.User `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6,max=72"`
}

type UpdateUserProfileRequest struct {
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,max=20"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      string `json:"role"`
}

type AdminUpdateUserStatusRequest struct {
	IsActive *bool   `json:"is_active,omitempty"`
	Role     *string `json:"role,omitempty" validate:"omitempty,oneof=customer merchant admin"`
}



