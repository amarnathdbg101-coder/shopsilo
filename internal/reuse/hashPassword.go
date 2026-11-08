package reuse

import (
	gopw "github.com/amarnathdbg101-coder/go/go-password"
	"golang.org/x/crypto/bcrypt"
)

// ShopSiloPasswordPolicy defines production password rules for merchants and staff accounts.
var ShopSiloPasswordPolicy = gopw.Policy{
	MinLength:      6,
	MaxLength:      128,
	MaxRepeats:     4,
	DisallowCommon: true,
	MinEntropy:     15.0,
}

// ValidatePassword checks password against common dictionary leaks and entropy rules.
func ValidatePassword(pw string) error {
	return gopw.Validate(pw, ShopSiloPasswordPolicy)
}

// EvaluatePasswordStrength returns the evaluated security tier and Shannon entropy in bits.
func EvaluatePasswordStrength(pw string) (string, float64) {
	str := gopw.Score(pw)
	entropy := gopw.Entropy(pw)
	return str.String(), entropy
}

// GenerateSecureStaffPassword generates an unpredictable, high-entropy password for new counter staff.
func GenerateSecureStaffPassword(length int) (string, error) {
	if length < 8 {
		length = 10
	}
	return gopw.Generate(gopw.GenerateOptions{
		Length:        length,
		IncludeLower:  true,
		IncludeUpper:  true,
		IncludeDigits: true,
	})
}

// HashPassword hashes a raw password using bcrypt with standard cost 12.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPasswordHash compares a plaintext password with its bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
