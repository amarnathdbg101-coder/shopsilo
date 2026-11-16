package goslug

import (
	"strings"
	"unicode"
)

// Config controls the behavior of slug generation.
type Config struct {
	// Separator used between words (default "-").
	Separator string

	// MaxLength limits slug length. If > 0, cuts safely at word boundary (default 0 = unlimited).
	MaxLength int

	// Lowercase converts all output characters to lowercase (default true).
	Lowercase bool

	// TransliterateSymbols converts symbols like &, @, ₹ to "and", "at", "inr" (default true).
	TransliterateSymbols bool

	// TransliterateDevanagari transliterates Hindi/Devanagari script to Latin characters (default true).
	TransliterateDevanagari bool

	// CustomReplacements allows overriding or extending character substitutions.
	CustomReplacements map[rune]string
}

// DefaultConfig provides standard SEO-friendly configuration.
var DefaultConfig = Config{
	Separator:               "-",
	MaxLength:               0,
	Lowercase:               true,
	TransliterateSymbols:    true,
	TransliterateDevanagari: true,
	CustomReplacements:      nil,
}

// Make converts any input string into a clean, URL-safe slug using DefaultConfig.
func Make(s string) string {
	return MakeWithConfig(s, DefaultConfig)
}

// MakeWithConfig converts an input string into a slug according to the provided config.
func MakeWithConfig(s string, cfg Config) string {
	sep := cfg.Separator
	if sep == "" {
		sep = "-"
	}

	var sb strings.Builder
	sb.Grow(len(s) + 16)

	// Pre-process runes
	runes := []rune(s)
	for _, r := range runes {
		// 1. Check custom replacements
		if cfg.CustomReplacements != nil {
			if repl, ok := cfg.CustomReplacements[r]; ok {
				sb.WriteString(repl)
				continue
			}
		}

		// 2. Check symbol transliteration
		if cfg.TransliterateSymbols {
			if repl, ok := SymbolReplacements[r]; ok {
				sb.WriteString(" ")
				sb.WriteString(repl)
				sb.WriteString(" ")
				continue
			}
		}

		// 3. Check Devanagari transliteration
		if cfg.TransliterateDevanagari {
			if repl, ok := DevanagariReplacements[r]; ok {
				sb.WriteString(repl)
				continue
			}
		}

		// 4. Check Latin diacritics
		if repl, ok := DiacriticReplacements[r]; ok {
			r = repl
		}

		// 5. Letter, Digit or Space
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if cfg.Lowercase {
				sb.WriteRune(unicode.ToLower(r))
			} else {
				sb.WriteRune(r)
			}
		} else {
			// Any punctuation, whitespace or unhandled rune becomes a space placeholder
			sb.WriteRune(' ')
		}
	}

	raw := sb.String()

	// Normalize spaces: split into words and join with separator
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return ""
	}

	slug := strings.Join(fields, sep)

	// Apply MaxLength with word boundary preservation
	if cfg.MaxLength > 0 && len(slug) > cfg.MaxLength {
		truncated := slug[:cfg.MaxLength]
		// Find last separator to avoid cutting words in half
		lastSep := strings.LastIndex(truncated, sep)
		if lastSep > 0 {
			slug = truncated[:lastSep]
		} else {
			slug = strings.TrimRight(truncated, sep)
		}
	}

	return slug
}

// IsValidSlug checks if a string is a valid slug (only lowercase alphanumeric and configured separator).
func IsValidSlug(s string, separator string) bool {
	if s == "" {
		return false
	}
	if separator == "" {
		separator = "-"
	}
	sepRune := rune(separator[0])

	// Cannot start or end with separator
	if rune(s[0]) == sepRune || rune(s[len(s)-1]) == sepRune {
		return false
	}

	var prevWasSep bool
	for _, r := range s {
		if r == sepRune {
			if prevWasSep {
				// Consecutive separators disallowed
				return false
			}
			prevWasSep = true
			continue
		}
		prevWasSep = false

		if !unicode.IsDigit(r) && !unicode.IsLower(r) {
			return false
		}
	}

	return true
}
