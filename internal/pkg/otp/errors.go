package gootp

import "errors"

var (
	// ErrInvalidSecret indicates the Base32 secret is empty or invalid.
	ErrInvalidSecret = errors.New("gootp: invalid Base32 secret")

	// ErrInvalidCodeLength indicates code length is outside valid range (typically 6-8 digits).
	ErrInvalidCodeLength = errors.New("gootp: code length must be between 6 and 8")

	// ErrInvalidPeriod indicates TOTP step period must be positive (typically 30s).
	ErrInvalidPeriod = errors.New("gootp: period must be greater than zero")

	// ErrCodeMismatch indicates the OTP code did not match within tolerance window.
	ErrCodeMismatch = errors.New("gootp: verification code does not match")
)
