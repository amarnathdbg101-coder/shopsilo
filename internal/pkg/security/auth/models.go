package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}

type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email"`
	OTP         string `json:"otp"`
	NewPassword string `json:"new_password"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

type MessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
