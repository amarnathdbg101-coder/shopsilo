// Package reuse reduce repetition task.
package reuse

import (
	"regexp"
	"strings"
)

var (
	nonAlphaNumericRegex = regexp.MustCompile(`[^a-z0-9]+`)
	multipleDashRegex    = regexp.MustCompile(`-+`)
)

// Slugify converts an arbitrary string into a URL-friendly slug
func Slugify(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	// Replace non-alphanumeric characters with dashes
	withDashes := nonAlphaNumericRegex.ReplaceAllString(lower, "-")
	// Clean multiple consecutive dashes
	cleanDashes := multipleDashRegex.ReplaceAllLiteralString(withDashes, "-")
	// Trim dashes from start and end
	trimmed := strings.Trim(cleanDashes, "-")
	if trimmed == "" {
		return "shop"
	}
	return trimmed
}
