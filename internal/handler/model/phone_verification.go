package model

import "time"

// PhoneVerification tracks OTP dispatch, attempts, verification status, and single-use consumption.
type PhoneVerification struct {
	ID          string     `json:"id"`
	Phone       string     `json:"phone"`
	OTPHash     string     `json:"-"` // never expose OTP hash in API responses
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	IsVerified  bool       `json:"is_verified"`
	IsConsumed  bool       `json:"is_consumed"`
	ExpiresAt   time.Time  `json:"expires_at"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	ConsumedAt  *time.Time `json:"consumed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
