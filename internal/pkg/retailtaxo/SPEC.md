# SPEC: `go-retailtaxo` — Universal Indian Retail Taxonomy & Category Normalization Engine

## 1. Overview & Problem Statement
Modern multi-store retail, ERPs, quick-commerce, and kirana management platforms deal with unstructured, colloquial category inputs:
- Misspellings & typos: `"kirna"`, `"snaks"`, `"vegitables"`, `"footwer"`
- Synonym variations: `"Grocery & Staples"`, `"Dal-Pulses"`, `"Household & Cleaning"`, `"Dairy, Bakery & Frozen"`
- Bilingual naming: English (`"Kirana & Grocery"`) vs Hindi (`"किराना व राशन"`)
- Inconsistent casing: `"pooja samagri"`, `"spices"`, `"oil/ghee"`

`go-retailtaxo` solves this by providing:
1. 21 canonical standardized retail categories covering grocery, dairy, personal care, fashion, baby care, hardware, and electronics.
2. 3-tier resolution engine:
   - Tier 1: Zero-alloc O(1) dictionary lookup for exact canonical and alias matches (~1.2 μs).
   - Tier 2: Hyphen-stripped normalization.
   - Tier 3: Sub-millisecond typo-tolerant fuzzy matching (Levenshtein + Jaro-Winkler via `go-fuzzy`) for OCR and user typing errors.
3. Clean Title casing formatter (`"oil/ghee"` -> `"Oil/Ghee"`, `"kirana (ration)"` -> `"Kirana (Ration)"`).

---

## 2. Architecture & Design Principles
- **Monorepo Integration**: Leverages `go-fuzzy` for typo tolerance.
- **Zero cgo**: 100% pure Go standard library + `go-fuzzy`.
- **Precomputed Search Space**: Candidate slug slices and hash maps pre-indexed on initialization for sub-microsecond lookups.
- **Bilingual & Departmental Classifications**: Built-in support for Hindi scripts, department tags (`grocery`, `dairy`, `fashion`, etc.), and emoji icons.

---

## 3. Public API Specification
```go
package retailtaxo

type Category struct {
    Slug         string   `json:"slug"`
    NameEn       string   `json:"name_en"`
    NameHi       string   `json:"name_hi"`
    Department   string   `json:"department"`
    Icon         string   `json:"icon"`
    SynonymSlugs []string `json:"synonym_slugs"`
    SynonymNames []string `json:"synonym_names"`
}

var CanonicalCategories []Category

func CleanSlug(input string) string
func CleanCategorySlug(input string) string // Alias
func ResolveSlug(raw string) (string, bool)
func ResolveCanonicalCategorySlug(raw string) (string, bool) // Alias
func ResolveFuzzySlug(raw string, minScore float64) (string, float64, bool)
func ResolveFuzzyCategorySlug(raw string, minScore float64) (string, float64, bool) // Alias
func GetSynonymSlugs(raw string) []string
func GetCategorySynonymSlugs(raw string) []string // Alias
func GetSynonymNames(raw string) []string
func GetCategorySynonymNames(raw string) []string // Alias
func FindBySlug(slug string) (Category, bool)
func AllCategories() []Category
func FormatTitle(s string) string
func FormatCategoryTitle(s string) string // Alias
```

---

## 4. Benchmarks
- `ResolveSlug (Exact)`: ~1.2 μs / op
- `FormatTitle`: ~497 ns / op
- `ResolveSlug (Fuzzy Typo)`: ~173 μs / op
