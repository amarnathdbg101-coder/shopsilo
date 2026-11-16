package paginate

import (
	"strings"
	"unicode"
)

// AllowedSortOrders strictly limits sort direction to prevent SQL injection.
const (
	OrderAsc  = "ASC"
	OrderDesc = "DESC"
)

// SanitizeSort verifies that the sort column is in the allowed whitelist.
// If input matches an allowed column, it returns the safe column and verified order.
// Otherwise, it returns defaultSort to prevent dynamic SQL injection attacks.
func SanitizeSort(inputSort, inputOrder, defaultSort string, allowedCols ...string) (string, string) {
	// Normalize order strictly to ASC or DESC
	cleanOrder := OrderDesc
	if strings.EqualFold(strings.TrimSpace(inputOrder), OrderAsc) {
		cleanOrder = OrderAsc
	}

	cleanDefault := cleanIdentifier(defaultSort)
	if cleanDefault == "" {
		cleanDefault = "id"
	}

	trimmedInput := strings.TrimSpace(inputSort)
	if trimmedInput == "" {
		return cleanDefault, cleanOrder
	}

	// Whitelist validation
	for _, allowed := range allowedCols {
		if strings.EqualFold(trimmedInput, strings.TrimSpace(allowed)) {
			// Return the exact whitelisted column name (preserving original casing/schema)
			return cleanIdentifier(allowed), cleanOrder
		}
	}

	// Not in whitelist: Fallback to safe default
	return cleanDefault, cleanOrder
}

// Sanitize returns a safe copy of Params with sort validated against the provided whitelist.
func (p Params) Sanitize(defaultSort string, allowedCols ...string) Params {
	cleanSort, cleanOrder := SanitizeSort(p.Sort, p.Order, defaultSort, allowedCols...)
	return Params{
		Page:    p.Page,
		PerPage: p.PerPage,
		Sort:    cleanSort,
		Order:   cleanOrder,
	}
}

// cleanIdentifier strips any character that is not alphanumeric, underscore, or period (for table.column).
// Yeh kisi bhi SQL injection attempt (quotes, spaces, semicolons, comments) ko completely strip kar deta hai.
func cleanIdentifier(s string) string {
	var sb strings.Builder
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
