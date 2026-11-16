package goenv

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetString retrieves an environment variable or returns defaultVal if empty/unset.
func GetString(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

// GetInt retrieves an integer environment variable or returns defaultVal if missing/invalid.
func GetInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

// GetInt64 retrieves an int64 environment variable or returns defaultVal.
func GetInt64(key string, defaultVal int64) int64 {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return defaultVal
}

// GetFloat64 retrieves a float64 environment variable or returns defaultVal.
func GetFloat64(key string, defaultVal float64) float64 {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

// GetBool retrieves a boolean environment variable.
// Understands "true", "1", "yes", "on" as true; "false", "0", "no", "off" as false.
func GetBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		v := strings.ToLower(strings.TrimSpace(val))
		switch v {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultVal
}

// GetDuration retrieves a time.Duration (e.g. "5s", "10m", "24h") or returns defaultVal.
func GetDuration(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

// GetSlice retrieves a delimited string slice (e.g. "localhost,127.0.0.1") trimmed of whitespace.
func GetSlice(key string, defaultVal []string, sep string) []string {
	if sep == "" {
		sep = ","
	}
	if val, ok := os.LookupEnv(key); ok && val != "" {
		parts := strings.Split(val, sep)
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultVal
}

// MustGetString returns the value of key or ErrMissingEnv if missing/empty.
func MustGetString(key string) (string, error) {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val, nil
	}
	return "", fmt.Errorf("%w: key=%s", ErrMissingEnv, key)
}

// MustGetInt returns the integer value or error if missing/non-numeric.
func MustGetInt(key string) (int, error) {
	val, err := MustGetString(key)
	if err != nil {
		return 0, err
	}
	i, parseErr := strconv.Atoi(val)
	if parseErr != nil {
		return 0, fmt.Errorf("%w: key=%s value=%s is not an integer", ErrInvalidType, key, val)
	}
	return i, nil
}

// MustGetBool returns boolean value or error if missing/invalid.
func MustGetBool(key string) (bool, error) {
	val, err := MustGetString(key)
	if err != nil {
		return false, err
	}
	v := strings.ToLower(strings.TrimSpace(val))
	switch v {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%w: key=%s value=%s is not a valid boolean", ErrInvalidType, key, val)
	}
}

// Validate checks that all specified keys exist and are non-empty.
func Validate(requiredKeys ...string) error {
	var missing []string
	for _, k := range requiredKeys {
		if val, ok := os.LookupEnv(k); !ok || strings.TrimSpace(val) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return &MissingKeysError{Keys: missing}
	}
	return nil
}

// DumpMasked returns a map of all current environment variables with sensitive secrets masked.
// Safe for printing application startup configurations in server logs.
func DumpMasked() map[string]string {
	envList := os.Environ()
	result := make(map[string]string, len(envList))

	for _, item := range envList {
		eqIdx := strings.Index(item, "=")
		if eqIdx <= 0 {
			continue
		}
		k := item[:eqIdx]
		v := item[eqIdx+1:]

		if isSensitive(k) {
			result[k] = "********"
		} else {
			result[k] = v
		}
	}

	return result
}

func isSensitive(k string) bool {
	upper := strings.ToUpper(k)
	sensitivePatterns := []string{
		"SECRET", "PASSWORD", "PASS", "TOKEN", "KEY",
		"CREDENTIAL", "PRIVATE", "AUTH", "DSN",
	}
	for _, p := range sensitivePatterns {
		if strings.Contains(upper, p) {
			return true
		}
	}
	return false
}
