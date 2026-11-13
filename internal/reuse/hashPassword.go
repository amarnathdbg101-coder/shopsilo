package reuse

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	gopw "github.com/amarnathdbg101-coder/go/go-password"
	"golang.org/x/crypto/argon2"
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

// Argon2id Parameters (OWASP Recommended Profile)
const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

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

// HashPassword hashes a raw password using GPU-resistant Argon2id profile.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		// Fallback to bcrypt if crypto/rand fails
		bytes, errB := bcrypt.GenerateFromPassword([]byte(password), 12)
		return string(bytes), errB
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads, saltB64, hashB64)

	return encoded, nil
}

// CheckPasswordHash compares a plaintext password with its hash (supports both Argon2id and Bcrypt).
func CheckPasswordHash(password, hash string) bool {
	valid, _ := CheckPasswordHashAndNeedsUpgrade(password, hash)
	return valid
}

// CheckPasswordHashAndNeedsUpgrade checks password validity and indicates if hash should be auto-upgraded to Argon2id.
func CheckPasswordHashAndNeedsUpgrade(password, hash string) (valid bool, needsUpgrade bool) {
	if strings.HasPrefix(hash, "$argon2id$") {
		parts := strings.Split(hash, "$")
		if len(parts) != 6 {
			return false, false
		}

		var memory uint32
		var time uint32
		var threads uint8
		_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
		if err != nil {
			return false, false
		}

		salt, err := base64.RawStdEncoding.DecodeString(parts[4])
		if err != nil {
			return false, false
		}

		expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
		if err != nil {
			return false, false
		}

		computedHash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expectedHash)))

		if subtle.ConstantTimeCompare(computedHash, expectedHash) == 1 {
			return true, false // Valid & already on Argon2id
		}
		return false, false
	}

	// Fallback to legacy Bcrypt verification
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return true, true // Valid, but needs upgrade from Bcrypt to Argon2id!
	}

	return false, false
}
