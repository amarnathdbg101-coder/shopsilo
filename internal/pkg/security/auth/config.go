package auth

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	BcryptCost      int
	OTPTTL          time.Duration
	OTPMaxAttempts  int
	Issuer          string
	DefaultRole     string
	OnSendOTP       func(ctx context.Context, email, otp string) error
}

func DefaultConfig(jwtSecret string) Config {
	return Config{
		JWTSecret:       jwtSecret,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      bcrypt.DefaultCost + 2,
		OTPTTL:          5 * time.Minute,
		OTPMaxAttempts:  3,
		Issuer:          "go-security",
		DefaultRole:     "user",
	}
}

func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return errors.New("auth: JWTSecret must not be empty")
	}
	if len(c.JWTSecret) < 16 {
		return errors.New("auth: JWTSecret must be at least 16 characters for HMAC-SHA256 security")
	}
	if c.AccessTokenTTL <= 0 {
		c.AccessTokenTTL = 15 * time.Minute
	}
	if c.RefreshTokenTTL <= 0 {
		c.RefreshTokenTTL = 7 * 24 * time.Hour
	}
	if c.BcryptCost < bcrypt.MinCost || c.BcryptCost > bcrypt.MaxCost {
		c.BcryptCost = bcrypt.DefaultCost
	}
	if c.OTPTTL <= 0 {
		c.OTPTTL = 5 * time.Minute
	}
	if c.OTPMaxAttempts <= 0 {
		c.OTPMaxAttempts = 3
	}
	if c.Issuer == "" {
		c.Issuer = "go-security"
	}
	if c.DefaultRole == "" {
		c.DefaultRole = "user"
	}
	return nil
}
