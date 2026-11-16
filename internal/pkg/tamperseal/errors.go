package tamperseal

import "errors"

var (
	// ErrMissingSecret is returned when the HMAC secret is empty.
	ErrMissingSecret = errors.New("tamperseal: secret key cannot be empty")
	// ErrInvalidToken is returned when the token format does not have 3 parts.
	ErrInvalidToken = errors.New("tamperseal: invalid token format")
	// ErrSignatureMismatch is returned when the token's HMAC signature is invalid or tampered with.
	ErrSignatureMismatch = errors.New("tamperseal: signature verification failed; data has been tampered")
	// ErrTokenExpired is returned when the token timestamp exceeds the permitted TTL.
	ErrTokenExpired = errors.New("tamperseal: token has expired")
)
