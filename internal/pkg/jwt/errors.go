package gojwt

import "errors"

var (
	// ErrInvalidToken indicates the token format is not 3 dot-separated Base64 parts.
	ErrInvalidToken = errors.New("jwt: invalid token format")

	// ErrInvalidSignature indicates HMAC signature verification failed.
	ErrInvalidSignature = errors.New("jwt: signature verification failed")

	// ErrTokenExpired indicates token expiration time (exp) has passed.
	ErrTokenExpired = errors.New("jwt: token has expired")

	// ErrTokenNotValidYet indicates token is used before its NotBefore (nbf) time.
	ErrTokenNotValidYet = errors.New("jwt: token not valid yet")

	// ErrMissingSecret indicates secret key was empty.
	ErrMissingSecret = errors.New("jwt: secret key cannot be empty")
)
