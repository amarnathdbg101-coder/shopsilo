package reuse

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	gomask "github.com/amarnathdbg101-coder/go/go-mask"
	gosanitize "github.com/amarnathdbg101-coder/go/go-sanitize"
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

// SanitizeReviewText strips HTML XSS tags and external promotional spam URLs from customer reviews.
func SanitizeReviewText(comment string) string {
	comment = gosanitize.StripHTML(comment)
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
		return strings.Join(strings.Fields(comment), " ")
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

// SanitizeUserInput strips HTML tags and normalizes whitespace for public-facing text fields.
func SanitizeUserInput(input string) string {
	return strings.TrimSpace(gosanitize.StripHTML(input))
}

// MaskPhoneNumber masks a phone number for DPDP compliance using go-mask.
func MaskPhoneNumber(phone string) string {
	trimmed := strings.TrimSpace(phone)
	if len(trimmed) <= 4 {
		return trimmed
	}
	return gomask.Phone(phone)
}

// MaskEmail masks an email address for privacy and DPDP compliance using go-mask.
func MaskEmail(email string) string {
	return gomask.Email(email)
}
// GeoQueryParams holds validated, sanitized, and clamped spatial parameters.
type GeoQueryParams struct {
	Lat      *float64
	Lng      *float64
	RadiusKm *float64
	HasGeo   bool
}

// ParseAndValidateGeoParams extracts, validates, and clamps geospatial query parameters.
// Guarantees:
// 1. Both lat and lng must be provided together. Partial coordinates return an error.
// 2. Latitude is strictly checked: -90.0 <= lat <= 90.0, not NaN, not Inf.
// 3. Longitude is strictly checked: -180.0 <= lng <= 180.0, not NaN, not Inf.
// 4. Clamps radius between minRadius (e.g. 0.5 km) and maxRadius (e.g. 30.0 km) to prevent DoS.
// 5. Rounds coordinates to 5 decimal places (~1.1 meter accuracy) for privacy protection and cache consolidation.
func ParseAndValidateGeoParams(latStr, lngStr, radStr string, defaultRadius, minRadius, maxRadius float64) (GeoQueryParams, error) {
	latStr = strings.TrimSpace(latStr)
	lngStr = strings.TrimSpace(lngStr)
	radStr = strings.TrimSpace(radStr)

	// If neither lat nor lng provided, geo query is inactive (clean non-geo browse)
	if latStr == "" && lngStr == "" {
		return GeoQueryParams{HasGeo: false}, nil
	}

	// Partial coordinates validation: both must be present
	if latStr == "" || lngStr == "" {
		return GeoQueryParams{}, errors.New("both latitude (lat) and longitude (lng) must be provided together")
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || math.IsNaN(lat) || math.IsInf(lat, 0) {
		return GeoQueryParams{}, errors.New("invalid latitude: must be a valid decimal number")
	}
	if lat < -90.0 || lat > 90.0 {
		return GeoQueryParams{}, errors.New("latitude out of range: must be between -90.0 and +90.0 degrees")
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || math.IsNaN(lng) || math.IsInf(lng, 0) {
		return GeoQueryParams{}, errors.New("invalid longitude: must be a valid decimal number")
	}
	if lng < -180.0 || lng > 180.0 {
		return GeoQueryParams{}, errors.New("longitude out of range: must be between -180.0 and +180.0 degrees")
	}

	// Sanitize and round coordinates to 5 decimal places (~1.1 meter accuracy)
	roundedLat := math.Round(lat*100000) / 100000
	roundedLng := math.Round(lng*100000) / 100000

	// Radius parsing and DoS clamping
	radius := defaultRadius
	if radius <= 0 {
		radius = 10.0
	}
	if radStr != "" {
		if rVal, rErr := strconv.ParseFloat(radStr, 64); rErr == nil && !math.IsNaN(rVal) && !math.IsInf(rVal, 0) {
			radius = rVal
		}
	}

	if minRadius > 0 && radius < minRadius {
		radius = minRadius
	}
	if maxRadius > 0 && radius > maxRadius {
		radius = maxRadius
	}

	return GeoQueryParams{
		Lat:      &roundedLat,
		Lng:      &roundedLng,
		RadiusKm: &radius,
		HasGeo:   true,
	}, nil
}
