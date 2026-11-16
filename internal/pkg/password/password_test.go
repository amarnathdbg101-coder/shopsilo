package password

import (
	"errors"
	"strings"
	"testing"
	"unicode"
)

func TestEntropyAndScore(t *testing.T) {
	// Weak
	pwWeak := "123456"
	entWeak := Entropy(pwWeak)
	if entWeak > 25 {
		t.Errorf("expected low entropy for 123456, got %f", entWeak)
	}
	if Score(pwWeak) != StrengthVeryWeak {
		t.Errorf("expected VeryWeak for 123456, got %s", Score(pwWeak))
	}

	// Strong
	pwStrong := "K#9xP$2wL!mQ"
	entStrong := Entropy(pwStrong)
	if entStrong < 70 {
		t.Errorf("expected high entropy for K#9xP$2wL!mQ, got %f", entStrong)
	}
	if Score(pwStrong) != StrengthStrong && Score(pwStrong) != StrengthVeryStrong {
		t.Errorf("expected Strong/VeryStrong, got %s", Score(pwStrong))
	}
}

func TestValidate_Policy(t *testing.T) {
	policy := DefaultPolicy()

	// Valid password
	if err := Validate("P@ssw0rd!Secure99", policy); err != nil {
		t.Errorf("expected valid password, got: %v", err)
	}

	// Too short
	if err := Validate("P@1a", policy); !errors.Is(err, ErrTooShort) {
		t.Errorf("expected ErrTooShort, got %v", err)
	}

	// Common password blacklist
	if err := Validate("password123", policy); !errors.Is(err, ErrCommonPassword) {
		t.Errorf("expected ErrCommonPassword for password123, got %v", err)
	}

	// Repeating characters
	if err := Validate("Paaaaassword123", policy); !errors.Is(err, ErrRepeatingChars) {
		t.Errorf("expected ErrRepeatingChars, got %v", err)
	}
}

func TestGenerate(t *testing.T) {
	opts := GenerateOptions{
		Length:         20,
		IncludeUpper:   true,
		IncludeLower:   true,
		IncludeDigits:  true,
		IncludeSymbols: true,
	}

	pw, err := Generate(opts)
	if err != nil {
		t.Fatalf("unexpected error generating password: %v", err)
	}

	if len(pw) != 20 {
		t.Errorf("expected length 20, got %d", len(pw))
	}

	// Verify all classes are present
	hasUpper := strings.ContainsFunc(pw, unicode.IsUpper)
	hasLower := strings.ContainsFunc(pw, unicode.IsLower)
	hasDigit := strings.ContainsFunc(pw, unicode.IsDigit)

	if !hasUpper || !hasLower || !hasDigit {
		t.Errorf("generated password missing required char classes: %s", pw)
	}

	// Score of generated 20-char password should be Very Strong
	score := Score(pw)
	if score != StrengthVeryStrong {
		t.Errorf("expected VeryStrong for 20-char generated password, got %s", score)
	}
}
