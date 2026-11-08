// Package reuse reduces repetitive tasks across ShopSilo.
package reuse

import (
	goslug "github.com/amarnathdbg101-coder/go/go-slug"
)

// Slugify converts an arbitrary string into a URL-friendly slug.
// Powered by zero-dep go-slug with Devanagari transliteration,
// symbol conversion (& -> and, @ -> at, ₹ -> inr), and Unicode normalization.
func Slugify(text string) string {
	res := goslug.Make(text)
	if res == "" {
		return "shop"
	}
	return res
}
