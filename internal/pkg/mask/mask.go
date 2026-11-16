package gomask

import (
	"encoding/json"
	"strings"
	"unicode"
)

// MaskString replaces intermediate characters with maskRune, preserving keepPrefix and keepSuffix runes.
func MaskString(s string, keepPrefix, keepSuffix int, maskRune rune) string {
	runes := []rune(s)
	total := len(runes)

	if total == 0 {
		return ""
	}
	if keepPrefix < 0 {
		keepPrefix = 0
	}
	if keepSuffix < 0 {
		keepSuffix = 0
	}

	if keepPrefix+keepSuffix >= total {
		// String is too short to mask intermediate runes; mask everything except first rune
		if total <= 2 {
			return string(maskRune)
		}
		keepPrefix = 1
		keepSuffix = 1
	}

	var sb strings.Builder
	sb.Grow(total)

	sb.WriteString(string(runes[:keepPrefix]))
	for i := keepPrefix; i < total-keepSuffix; i++ {
		sb.WriteRune(maskRune)
	}
	sb.WriteString(string(runes[total-keepSuffix:]))

	return sb.String()
}

// Email masks the username portion of an email address (e.g. "rahul.sharma@example.com" -> "r***a@example.com").
func Email(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return MaskString(email, 1, 1, '*')
	}

	user := parts[0]
	domain := parts[1]

	var maskedUser string
	uLen := len(user)
	if uLen <= 2 {
		maskedUser = string(user[0]) + "*"
	} else {
		maskedUser = string(user[0]) + strings.Repeat("*", uLen-2) + string(user[uLen-1])
	}

	return maskedUser + "@" + domain
}

// Phone masks mobile numbers, preserving country code and last 4 digits (e.g. "+919876543210" -> "+91-98****3210").
func Phone(phone string) string {
	digitsOnly := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, phone)

	dLen := len(digitsOnly)
	if dLen < 7 {
		return MaskString(phone, 2, 2, '*')
	}

	hasPlus := strings.HasPrefix(strings.TrimSpace(phone), "+")
	last4 := digitsOnly[dLen-4:]

	if dLen == 10 {
		// Standard 10-digit: "98****3210"
		return digitsOnly[:2] + "****" + last4
	}

	if dLen > 10 {
		// With country code: e.g. 12 digits (91 + 10 digits)
		cc := digitsOnly[:dLen-10]
		prefix := digitsOnly[dLen-10 : dLen-8]
		res := prefix + "****" + last4
		if hasPlus {
			return "+" + cc + "-" + res
		}
		return cc + "-" + res
	}

	return MaskString(phone, 2, 4, '*')
}

// Aadhaar masks a 12-digit Indian Aadhaar number (e.g. "5482 9182 3491" -> "XXXX-XXXX-3491").
func Aadhaar(aadhaar string) string {
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, aadhaar)

	if len(digits) != 12 {
		return MaskString(aadhaar, 0, 4, 'X')
	}

	return "XXXX-XXXX-" + digits[8:]
}

// PAN masks a 10-character Indian Permanent Account Number (e.g. "ABCDE1234F" -> "XXXXX1234X").
func PAN(pan string) string {
	clean := strings.ToUpper(strings.TrimSpace(pan))
	if len(clean) != 10 {
		return MaskString(pan, 0, 4, 'X')
	}

	return "XXXXX" + clean[5:9] + "X"
}

// CreditCard masks 16-digit payment card numbers (e.g. "4111222233334444" -> "4111-XXXX-XXXX-4444").
func CreditCard(cardNumber string) string {
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, cardNumber)

	if len(digits) < 13 || len(digits) > 19 {
		return MaskString(cardNumber, 4, 4, 'X')
	}

	first4 := digits[:4]
	last4 := digits[len(digits)-4:]

	return first4 + "-XXXX-XXXX-" + last4
}

// JSONRecursively masks sensitive key values in a JSON payload.
func JSONRecursively(jsonBytes []byte, sensitiveKeys ...string) ([]byte, error) {
	if len(sensitiveKeys) == 0 {
		sensitiveKeys = []string{
			"password", "token", "secret", "cvv", "api_key",
			"aadhaar", "pan", "card_number", "pin",
		}
	}

	keyMap := make(map[string]bool, len(sensitiveKeys))
	for _, k := range sensitiveKeys {
		keyMap[strings.ToLower(k)] = true
	}

	var data any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, err
	}

	masked := maskValue(data, keyMap)
	return json.Marshal(masked)
}

func maskValue(val any, sensitiveKeys map[string]bool) any {
	switch v := val.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for k, child := range v {
			if sensitiveKeys[strings.ToLower(k)] {
				result[k] = "********"
			} else {
				result[k] = maskValue(child, sensitiveKeys)
			}
		}
		return result

	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = maskValue(item, sensitiveKeys)
		}
		return result

	default:
		return v
	}
}
