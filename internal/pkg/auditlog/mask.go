package auditlog

import (
	"encoding/json"
	"strings"
)

// DefaultSensitiveKeys are keys automatically masked to prevent secrets leaking into audit stores.
var DefaultSensitiveKeys = []string{
	"password",
	"password_hash",
	"token",
	"access_token",
	"refresh_token",
	"secret",
	"api_key",
	"cvv",
	"card_number",
	"credit_card",
	"private_key",
	"ssn",
}

const MaskReplacement = "********"

// MaskData deep-scans structured data (maps, structs, slices) and masks all sensitive keys.
func MaskData(input any, customKeys ...string) any {
	if input == nil {
		return nil
	}

	// Normalize data to generic map/slice structure via JSON roundtrip
	bytes, err := json.Marshal(input)
	if err != nil {
		return input
	}

	var raw any
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return input
	}

	keysToMask := DefaultSensitiveKeys
	if len(customKeys) > 0 {
		keysToMask = append(keysToMask, customKeys...)
	}

	return maskRecursive(raw, keysToMask)
}

func maskRecursive(val any, sensitiveKeys []string) any {
	switch v := val.(type) {
	case map[string]any:
		maskedMap := make(map[string]any, len(v))
		for key, subVal := range v {
			if isSensitiveKey(key, sensitiveKeys) {
				maskedMap[key] = MaskReplacement
			} else {
				maskedMap[key] = maskRecursive(subVal, sensitiveKeys)
			}
		}
		return maskedMap

	case []any:
		maskedSlice := make([]any, len(v))
		for i, item := range v {
			maskedSlice[i] = maskRecursive(item, sensitiveKeys)
		}
		return maskedSlice

	default:
		return v
	}
}

func isSensitiveKey(key string, sensitiveKeys []string) bool {
	cleanKey := strings.ToLower(strings.TrimSpace(key))
	for _, s := range sensitiveKeys {
		s = strings.ToLower(strings.TrimSpace(s))
		if cleanKey == s ||
			strings.HasPrefix(cleanKey, s+"_") ||
			strings.HasSuffix(cleanKey, "_"+s) ||
			strings.HasPrefix(cleanKey, s+"-") ||
			strings.HasSuffix(cleanKey, "-"+s) {
			return true
		}
	}
	return false
}
