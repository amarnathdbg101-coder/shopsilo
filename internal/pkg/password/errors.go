package password

import "errors"

var (
	// ErrTooShort is returned when password length is below the minimum required length.
	ErrTooShort = errors.New("password: password is too short")

	// ErrTooLong is returned when password length exceeds the maximum allowed length.
	ErrTooLong = errors.New("password: password is too long")

	// ErrCommonPassword is returned when the password matches a known common or leaked password.
	ErrCommonPassword = errors.New("password: password is too common or easily guessed")

	// ErrRepeatingChars is returned when the password contains too many consecutive identical characters.
	ErrRepeatingChars = errors.New("password: password contains too many repeating characters")

	// ErrSequentialChars is returned when the password contains predictable sequential sequences like '12345' or 'abcde'.
	ErrSequentialChars = errors.New("password: password contains sequential character patterns")

	// ErrInsufficientEntropy is returned when the password bit entropy is below the policy threshold.
	ErrInsufficientEntropy = errors.New("password: password entropy is insufficient")
)
