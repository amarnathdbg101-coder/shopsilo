package auth

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidToken        = errors.New("invalid or malformed token")
	ErrTokenExpired        = errors.New("token has expired")
	ErrInvalidTokenType    = errors.New("invalid token type")
	ErrInvalidOTP          = errors.New("invalid OTP")
	ErrOTPExpired          = errors.New("OTP has expired")
	ErrMaxAttemptsExceeded = errors.New("maximum OTP verification attempts exceeded, request a new OTP")
	ErrWeakPassword        = errors.New("password must be at least 8 characters long")
	ErrInvalidEmail        = errors.New("invalid email format")
	ErrUnauthorized        = errors.New("unauthorized access")
	ErrForbidden           = errors.New("forbidden: insufficient permissions")
)
