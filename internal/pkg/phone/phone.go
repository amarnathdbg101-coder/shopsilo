package phone

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Known standard country calling codes
var knownCountryCodes = []string{
	"+880", "+971", "+977",
	"+91", "+44", "+61", "+65", "+49", "+33", "+81", "+86", "+55",
	"+27", "+92", "+94", "+60", "+62", "+63", "+41", "+31", "+34", "+39",
	"+1", "+7",
}

func init() {
	// Sort by length descending so longer prefixes match first (e.g. +880 before +88)
	sort.Slice(knownCountryCodes, func(i, j int) bool {
		return len(knownCountryCodes[i]) > len(knownCountryCodes[j])
	})
}

// PhoneNumber holds the parsed and normalized components of a telephone number.
type PhoneNumber struct {
	Raw            string
	CountryCode    string // e.g. "+91"
	NationalNumber string // e.g. "9876543210"
	E164           string // e.g. "+919876543210"
}

// CleanDigits extracts only digits from input, stripping spaces, dashes, parentheses, etc.
func CleanDigits(raw string) string {
	var sb strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// Parse normalizes and parses a telephone number string into a structured PhoneNumber.
// If the input does not include an international calling code prefix, defaultCountryCode is applied.
func Parse(raw string, defaultCountryCode string) (*PhoneNumber, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, ErrEmptyPhone
	}

	defaultCC := normalizeCountryCode(defaultCountryCode)

	hasPlus := strings.HasPrefix(trimmed, "+")
	digits := CleanDigits(trimmed)
	if len(digits) == 0 {
		return nil, ErrEmptyPhone
	}

	var countryCode string
	var nationalNumber string

	if hasPlus {
		withPlus := "+" + digits
		found := false
		for _, cc := range knownCountryCodes {
			if strings.HasPrefix(withPlus, cc) {
				countryCode = cc
				nationalNumber = strings.TrimPrefix(withPlus, cc)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: unrecognized prefix for %s", ErrInvalidCountryCode, trimmed)
		}
	} else {
		// No leading plus. Check if digits start with trunk prefix '0' (e.g. 09876543210)
		if strings.HasPrefix(digits, "0") && len(digits) > 7 {
			digits = strings.TrimPrefix(digits, "0")
		}

		if defaultCC != "" {
			countryCode = defaultCC
			// If digits already start with the default country code digits (e.g. 919876543210 where len is 12)
			ccDigits := CleanDigits(defaultCC)
			if strings.HasPrefix(digits, ccDigits) && len(digits) >= len(ccDigits)+9 && len(digits) > 10 {
				nationalNumber = strings.TrimPrefix(digits, ccDigits)
			} else {
				nationalNumber = digits
			}
		} else {
			return nil, fmt.Errorf("%w: number has no '+' prefix and no default country code specified", ErrInvalidCountryCode)
		}
	}

	// Validate lengths according to E.164: national number 6-14 digits, total E.164 max 15 digits
	totalDigits := CleanDigits(countryCode) + nationalNumber
	if len(totalDigits) < 7 || len(totalDigits) > 15 {
		return nil, fmt.Errorf("%w: total digit count is %d", ErrInvalidLength, len(totalDigits))
	}

	return &PhoneNumber{
		Raw:            raw,
		CountryCode:    countryCode,
		NationalNumber: nationalNumber,
		E164:           countryCode + nationalNumber,
	}, nil
}

// FormatE164 parses and formats an input string directly to standard E.164 notation (+[country][national]).
func FormatE164(raw string, defaultCountryCode string) (string, error) {
	pn, err := Parse(raw, defaultCountryCode)
	if err != nil {
		return "", err
	}
	return pn.E164, nil
}

// IsValidIndianMobile checks if a phone number is a valid 10-digit Indian mobile number.
// Valid Indian mobiles start with 6, 7, 8, or 9 and have exactly 10 national digits.
func IsValidIndianMobile(raw string) bool {
	pn, err := Parse(raw, "+91")
	if err != nil {
		return false
	}

	if pn.CountryCode != "+91" {
		return false
	}

	if len(pn.NationalNumber) != 10 {
		return false
	}

	firstDigit := pn.NationalNumber[0]
	return firstDigit >= '6' && firstDigit <= '9'
}

// Mask returns a privacy-safe masked representation (e.g. "+91 ******3210").
func Mask(raw string, defaultCountryCode string) string {
	pn, err := Parse(raw, defaultCountryCode)
	if err != nil {
		// Fallback masking on clean digits if parse fails
		digits := CleanDigits(raw)
		if len(digits) <= 4 {
			return "****"
		}
		return strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
	}

	nat := pn.NationalNumber
	if len(nat) <= 4 {
		return pn.CountryCode + " " + strings.Repeat("*", len(nat))
	}

	maskedNat := strings.Repeat("*", len(nat)-4) + nat[len(nat)-4:]
	return pn.CountryCode + " " + maskedNat
}

func normalizeCountryCode(cc string) string {
	cc = strings.TrimSpace(cc)
	if cc == "" {
		return ""
	}
	if !strings.HasPrefix(cc, "+") {
		return "+" + cc
	}
	return cc
}
