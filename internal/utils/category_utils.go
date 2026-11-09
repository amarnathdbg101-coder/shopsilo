package utils

import (
	"github.com/amarnathdbg101-coder/go/go-retailtaxo"
)

// CanonicalCategoryInfo defines standard metadata and synonyms for a canonical category.
type CanonicalCategoryInfo = retailtaxo.Category

// CanonicalCategories holds the 21 standard retail categories.
var CanonicalCategories = retailtaxo.CanonicalCategories

// CleanCategorySlug strips non-alphanumeric characters for clean fast matching.
func CleanCategorySlug(input string) string {
	return retailtaxo.CleanCategorySlug(input)
}

// ResolveCanonicalCategorySlug maps any category name, slug, or synonym into its canonical category slug.
// Falls back to high-confidence fuzzy matching (Levenshtein/JaroWinkler ratio >= 0.70) for typos and colloquial variations.
func ResolveCanonicalCategorySlug(raw string) (string, bool) {
	return retailtaxo.ResolveCanonicalCategorySlug(raw)
}

// ResolveFuzzyCategorySlug resolves raw category input to a canonical slug with similarity score.
// Returns canonical slug, similarity score (0.0 - 1.0), and whether a match was accepted.
func ResolveFuzzyCategorySlug(raw string, minScore float64) (string, float64, bool) {
	return retailtaxo.ResolveFuzzyCategorySlug(raw, minScore)
}

// GetCategorySynonymSlugs returns all slug aliases for a given canonical or synonym slug.
func GetCategorySynonymSlugs(raw string) []string {
	return retailtaxo.GetCategorySynonymSlugs(raw)
}

// GetCategorySynonymNames returns all name aliases for a given canonical or synonym slug.
func GetCategorySynonymNames(raw string) []string {
	return retailtaxo.GetCategorySynonymNames(raw)
}

// FormatCategoryTitle formats any category string into clean Title Case where the first
// letter of each word is capitalized and all subsequent letters are lowercased.
func FormatCategoryTitle(s string) string {
	return retailtaxo.FormatCategoryTitle(s)
}
