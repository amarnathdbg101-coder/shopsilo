package password

import (
	"crypto/rand"
	"math"
	"math/big"
	"strings"
	"unicode"
)

// Strength defines the evaluated security tier of a password.
type Strength int

const (
	StrengthVeryWeak Strength = iota
	StrengthWeak
	StrengthMedium
	StrengthStrong
	StrengthVeryStrong
)

func (s Strength) String() string {
	switch s {
	case StrengthVeryWeak:
		return "Very Weak"
	case StrengthWeak:
		return "Weak"
	case StrengthMedium:
		return "Medium"
	case StrengthStrong:
		return "Strong"
	case StrengthVeryStrong:
		return "Very Strong"
	default:
		return "Unknown"
	}
}

// Policy specifies rules for validating passwords.
type Policy struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireDigit   bool
	RequireSymbol  bool
	MaxRepeats     int // max consecutive identical characters allowed (0 to disable)
	DisallowCommon bool
	MinEntropy     float64
}

// DefaultPolicy returns an OWASP/NIST-recommended default password policy.
func DefaultPolicy() Policy {
	return Policy{
		MinLength:      8,
		MaxLength:      128,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSymbol:  false,
		MaxRepeats:     3,
		DisallowCommon: true,
		MinEntropy:     35.0,
	}
}

var commonPasswords = map[string]struct{}{
	"password": {}, "123456": {}, "12345678": {}, "123456789": {}, "qwerty": {},
	"admin": {}, "welcome": {}, "login": {}, "secret": {}, "iloveyou": {},
	"password123": {}, "admin123": {}, "abc123": {}, "letmein": {}, "monkey": {},
	"dragon": {}, "master": {}, "sunshine": {}, "princess": {}, "football": {},
	"shadow": {}, "superman": {}, "trustno1": {}, "charlie": {},
}

// Entropy calculates the Shannon information entropy of the password in bits.
func Entropy(pw string) float64 {
	length := len(pw)
	if length == 0 {
		return 0
	}

	hasLower, hasUpper, hasDigit, hasSymbol := false, false, false, false
	for _, r := range pw {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSymbol = true
		}
	}

	poolSize := 0
	if hasLower {
		poolSize += 26
	}
	if hasUpper {
		poolSize += 26
	}
	if hasDigit {
		poolSize += 10
	}
	if hasSymbol {
		poolSize += 33
	}

	if poolSize == 0 {
		return 0
	}

	return float64(length) * (math.Log(float64(poolSize)) / math.Log(2))
}

// Score evaluates and classifies the strength of a password.
func Score(pw string) Strength {
	ent := Entropy(pw)
	length := len(pw)

	switch {
	case length < 6 || ent < 25:
		return StrengthVeryWeak
	case length < 8 || ent < 40:
		return StrengthWeak
	case length < 12 || ent < 60:
		return StrengthMedium
	case ent < 80:
		return StrengthStrong
	default:
		return StrengthVeryStrong
	}
}

// Validate checks whether a password satisfies the specified policy constraints.
func Validate(pw string, p Policy) error {
	length := len(pw)

	if p.MinLength > 0 && length < p.MinLength {
		return ErrTooShort
	}
	if p.MaxLength > 0 && length > p.MaxLength {
		return ErrTooLong
	}

	if p.DisallowCommon {
		lower := strings.ToLower(pw)
		if _, exists := commonPasswords[lower]; exists {
			return ErrCommonPassword
		}
	}

	hasLower, hasUpper, hasDigit, hasSymbol := false, false, false, false
	var lastRune rune
	repeatCount := 1

	for _, r := range pw {
		if r == lastRune {
			repeatCount++
			if p.MaxRepeats > 0 && repeatCount > p.MaxRepeats {
				return ErrRepeatingChars
			}
		} else {
			lastRune = r
			repeatCount = 1
		}

		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSymbol = true
		}
	}

	if p.RequireLower && !hasLower {
		return ErrTooShort // Or policy violation
	}
	if p.RequireUpper && !hasUpper {
		return ErrTooShort
	}
	if p.RequireDigit && !hasDigit {
		return ErrTooShort
	}
	if p.RequireSymbol && !hasSymbol {
		return ErrTooShort
	}

	if p.MinEntropy > 0 && Entropy(pw) < p.MinEntropy {
		return ErrInsufficientEntropy
	}

	return nil
}

const (
	lowerChars  = "abcdefghijklmnopqrstuvwxyz"
	upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars  = "0123456789"
	symbolChars = "!@#$%^&*()-_=+[]{}<>~?"
)

// GenerateOptions configures the password generator.
type GenerateOptions struct {
	Length         int
	IncludeUpper   bool
	IncludeLower   bool
	IncludeDigits  bool
	IncludeSymbols bool
}

// Generate creates a cryptographically secure random password compliant with the options.
func Generate(opts GenerateOptions) (string, error) {
	if opts.Length <= 0 {
		opts.Length = 16
	}

	var charset string
	var mandatory []byte

	if opts.IncludeLower {
		charset += lowerChars
		b, err := cryptoPick(lowerChars)
		if err != nil {
			return "", err
		}
		mandatory = append(mandatory, b)
	}

	if opts.IncludeUpper {
		charset += upperChars
		b, err := cryptoPick(upperChars)
		if err != nil {
			return "", err
		}
		mandatory = append(mandatory, b)
	}

	if opts.IncludeDigits {
		charset += digitChars
		b, err := cryptoPick(digitChars)
		if err != nil {
			return "", err
		}
		mandatory = append(mandatory, b)
	}

	if opts.IncludeSymbols {
		charset += symbolChars
		b, err := cryptoPick(symbolChars)
		if err != nil {
			return "", err
		}
		mandatory = append(mandatory, b)
	}

	if charset == "" {
		charset = lowerChars + digitChars
	}

	result := make([]byte, opts.Length)
	copy(result, mandatory)

	for i := len(mandatory); i < opts.Length; i++ {
		b, err := cryptoPick(charset)
		if err != nil {
			return "", err
		}
		result[i] = b
	}

	// Shuffle result so mandatory characters are not always in prefix positions
	for i := len(result) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		j := int(jBig.Int64())
		result[i], result[j] = result[j], result[i]
	}

	return string(result), nil
}

func cryptoPick(pool string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))
	if err != nil {
		return 0, err
	}
	return pool[n.Int64()], nil
}
