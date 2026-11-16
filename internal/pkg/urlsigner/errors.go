package gourlsigner

import "errors"

var (
	// ErrExpired indicates the signed URL has exceeded its validity expiration time.
	ErrExpired = errors.New("gourlsigner: signed URL has expired")

	// ErrInvalidSignature indicates the signature does not match URL content or was tampered with.
	ErrInvalidSignature = errors.New("gourlsigner: invalid URL signature")

	// ErrMissingSignature indicates the URL query parameters lack the required signature.
	ErrMissingSignature = errors.New("gourlsigner: missing signature query parameter")

	// ErrMissingExpires indicates the URL query parameters lack the required expiration timestamp.
	ErrMissingExpires = errors.New("gourlsigner: missing expires query parameter")

	// ErrEmptySecret indicates secret key cannot be empty.
	ErrEmptySecret = errors.New("gourlsigner: secret key cannot be empty")
)
