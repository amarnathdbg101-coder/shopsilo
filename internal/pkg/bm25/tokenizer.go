package bm25

import (
	"strings"
	"unicode"
)

// Tokenize processes raw text into normalized, clean token slices.
func Tokenize(text string, minLen int) []string {
	if minLen <= 0 {
		minLen = 1
	}

	var tokens []string
	var b strings.Builder

	flush := func() {
		if b.Len() >= minLen {
			tokens = append(tokens, b.String())
		}
		b.Reset()
	}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		} else {
			flush()
		}
	}
	flush()

	return tokens
}
