package phone

import "errors"

var (
	// ErrEmptyPhone is returned when the phone number string is blank.
	ErrEmptyPhone = errors.New("phone: input number is empty")

	// ErrInvalidLength is returned when the digit length is too short or exceeds E.164 max length (15 digits).
	ErrInvalidLength = errors.New("phone: invalid phone number length (must be 7-15 digits)")

	// ErrInvalidCountryCode is returned when the international country calling code is unrecognized or missing.
	ErrInvalidCountryCode = errors.New("phone: missing or unrecognized country calling code")

	// ErrInvalidCharacters is returned when input contains disallowed characters.
	ErrInvalidCharacters = errors.New("phone: contains invalid characters")
)
