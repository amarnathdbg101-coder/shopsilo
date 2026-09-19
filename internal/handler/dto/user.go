// Package dto handler request and response struct.
package dto

import "shopMe/internal/handler/model"

type SendPhoneOTPRequest struct {
	Phone   string `json:"phone" validate:"required,min=10,max=20"`
	Channel string `json:"channel,omitempty" validate:"omitempty,oneof=whatsapp sms"` // defaults to whatsapp
}

type SendPhoneOTPResponse struct {
	Message   string `json:"message"`
	Phone     string `json:"phone"`
	Channel   string `json:"channel"`
	ExpiresIn int64  `json:"expires_in"` // seconds
	Cooldown  int64  `json:"cooldown"`   // seconds
}

type VerifyPhoneOTPRequest struct {
	Phone string `json:"phone" validate:"required,min=10,max=20"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

type VerifyPhoneOTPResponse struct {
	Message           string `json:"message"`
	Phone             string `json:"phone"`
	VerificationToken string `json:"verification_token"`
	ExpiresIn         int64  `json:"expires_in"` // seconds
}

type UserRegisterRequest struct {
	FullName          string `json:"full_name" validate:"required,min=2,max=100"`
	Email             string `json:"email" validate:"required,email"`
	Password          string `json:"password" validate:"required,min=6,max=72"`
	Phone             string `json:"phone" validate:"omitempty,min=10,max=20"`
	VerificationToken string `json:"verification_token,omitempty"`
}

type RegisterResponse struct {
	User        *model.User `json:"user"`
	AccessToken string      `json:"access_token,omitempty"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required"` // supports email or mobile phone
	Password string `json:"password" validate:"required"`
}

type GoogleLoginRequest struct {
	IDToken           string `json:"id_token" validate:"required"`
	Phone             string `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	VerificationToken string `json:"verification_token,omitempty"`
	Role              string `json:"role,omitempty"`
}

type GoogleLoginResponse struct {
	RequiresPhone bool        `json:"requires_phone"`
	Email         string      `json:"email,omitempty"`
	FullName      string      `json:"full_name,omitempty"`
	AvatarURL     string      `json:"avatar_url,omitempty"`
	AccessToken   string      `json:"access_token,omitempty"`
	TokenType     string      `json:"token_type,omitempty"`
	ExpiresIn     int64       `json:"expires_in,omitempty"`
	User          *model.User `json:"user,omitempty"`
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



