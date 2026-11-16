# SPEC: go-slug

## 1. Metadata
- **Module**: `go-slug`
- **Package**: `goslug`
- **Location**: `C:\Library\go-slug`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `strings`, `unicode`).

---

## 2. Core Capabilities
1. **High Performance**:
   - Sub-4 microsecond conversion, pre-allocated `strings.Builder`.
2. **Multi-Language & Unicode Transliteration**:
   - European Latin diacritics normalization (é, è, ê -> e; ü, ö, ä -> u, o, a).
   - Hindi/Devanagari phonetics transliteration map (अ -> a, क -> k, ा -> a, etc.).
   - Currency & common symbols (& -> and, @ -> at, ₹ -> inr, $ -> usd, % -> percent).
3. **Word-Boundary Aware Truncation**:
   - `MaxLength` does not cut words midway; trims neatly at the preceding separator boundary.
4. **Validation**:
   - `IsValidSlug(slug, separator)` validates slug syntax against double separators, forbidden punctuation, and casing rules.

---

## 3. Public API
```go
type Config struct {
    Separator               string
    MaxLength               int
    Lowercase               bool
    TransliterateSymbols    bool
    TransliterateDevanagari bool
    CustomReplacements      map[rune]string
}

var DefaultConfig Config

func Make(s string) string
func MakeWithConfig(s string, cfg Config) string
func IsValidSlug(s string, separator string) bool
```
