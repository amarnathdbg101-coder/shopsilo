package reuse

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

// ValidateStruct validates a struct and returns formatted error message
func ValidateStruct(s interface{}) error {
	if err := Validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var errMsgs []string
			for _, fieldErr := range validationErrors {
				switch fieldErr.Tag() {
				case "required":
					errMsgs = append(errMsgs, fmt.Sprintf("%s is required", fieldErr.Field()))
				case "email":
					errMsgs = append(errMsgs, fmt.Sprintf("%s must be a valid email address", fieldErr.Field()))
				case "min":
					errMsgs = append(errMsgs, fmt.Sprintf("%s must be at least %s characters long", fieldErr.Field(), fieldErr.Param()))
				case "max":
					errMsgs = append(errMsgs, fmt.Sprintf("%s cannot exceed %s characters", fieldErr.Field(), fieldErr.Param()))
				default:
					errMsgs = append(errMsgs, fmt.Sprintf("%s is invalid (%s)", fieldErr.Field(), fieldErr.Tag()))
				}
			}
			return fmt.Errorf("%s", strings.Join(errMsgs, ", "))
		}
		return err
	}
	return nil
}

// IsValidIndiaCoordinates checks whether a given lat/lng pair lies within India's mainland bounds.
// Latitude: 8.0° N to 37.5° N
// Longitude: 68.0° E to 97.5° E
func IsValidIndiaCoordinates(lat, lng float64) bool {
	if lat < 8.0 || lat > 37.5 {
		return false
	}
	if lng < 68.0 || lng > 97.5 {
		return false
	}
	return true
}

// SanitizeReviewText strips external promotional links or dangerous spam URLs from customer reviews.
func SanitizeReviewText(comment string) string {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return ""
	}

	// Simple and ultra-fast string replacements for typical link patterns without heavy regex overhead
	lowered := strings.ToLower(comment)
	spamKeywords := []string{"http://", "https://", "www.", "t.me/", "bit.ly/", "wa.me/"}
	hasSpam := false
	for _, kw := range spamKeywords {
		if strings.Contains(lowered, kw) {
			hasSpam = true
			break
		}
	}

	if !hasSpam {
		return comment
	}

	// Tokenize words and replace spam links
	words := strings.Fields(comment)
	for i, w := range words {
		lw := strings.ToLower(w)
		for _, kw := range spamKeywords {
			if strings.Contains(lw, kw) {
				words[i] = "[link removed]"
				break
			}
		}
	}
	return strings.Join(words, " ")
}

// MaskPhoneNumber masks a phone number for DPDP compliance (e.g. "+919876543210" -> "+9198*****210")
func MaskPhoneNumber(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if len(trimmed) <= 4 {
		return trimmed
	}
	if strings.HasPrefix(trimmed, "+") && len(trimmed) >= 12 {
		return trimmed[:5] + "*****" + trimmed[len(trimmed)-3:]
	}
	if len(trimmed) >= 10 {
		return trimmed[:2] + "*****" + trimmed[len(trimmed)-3:]
	}
	return trimmed[:1] + "***" + trimmed[len(trimmed)-2:]
}
