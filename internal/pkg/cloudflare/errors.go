package goclos

import "errors"

var (
	// ErrMissingCredentials is returned when AccountID or APIToken is omitted.
	ErrMissingCredentials = errors.New("go-cloudflare: account ID and API token are required")

	// ErrModelRequired is returned when an AI model identifier is not specified.
	ErrModelRequired = errors.New("go-cloudflare: AI model identifier is required")

	// ErrEmptyPayload is returned when image or text payload is empty.
	ErrEmptyPayload = errors.New("go-cloudflare: payload cannot be empty")

	// ErrR2CredentialsRequired is returned when R2 access key or secret key is missing.
	ErrR2CredentialsRequired = errors.New("go-cloudflare: R2 access key ID and secret access key are required")

	// ErrR2BucketRequired is returned when R2 bucket name is omitted.
	ErrR2BucketRequired = errors.New("go-cloudflare: R2 bucket name is required")

	// ErrAPIError is returned when Cloudflare returns a non-200 HTTP response.
	ErrAPIError = errors.New("go-cloudflare: API request failed")
)
