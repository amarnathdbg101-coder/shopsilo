package imghash

import "errors"

var (
	// ErrInvalidImage indicates that the image payload could not be decoded or has zero dimensions.
	ErrInvalidImage = errors.New("imghash: invalid image format or dimensions")

	// ErrInvalidHashFormat indicates that the provided hex hash string is not a valid 64-bit hex value.
	ErrInvalidHashFormat = errors.New("imghash: invalid 64-bit hexadecimal hash string")
)
