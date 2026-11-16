# SPEC: `go-hinglish` — Universal Indian Colloquial & Hinglish Search Expansion Engine

## 1. Overview & Problem Statement
In Indian retail, e-commerce, quick-commerce, and hyperlocal platforms (e.g. Kirana stores, Quick-commerce apps), end customers and shopkeepers rarely type exact standard English product titles. Users frequently search in colloquial Hindi/Hinglish transliterations:
- Searching `"doodh"` or `"dudh"` must effortlessly match `"Amul Taaza Milk 500ml"`.
- Searching `"cheeni"` or `"chini"` must match `"Madhur Pure Sugar 1kg"`.
- Searching `"tel"` or `"sarson"` must match `"Fortune Mustard Oil"`.
- Conversational filler words (e.g. `"ka"`, `"ki"`, `"chahiye"`, `"bhaiya"`) corrupt text indexing unless stripped deterministically.

`go-hinglish` solves this universally with a zero-dependency, sub-microsecond phonetic transliteration dictionary, soundex-style normalization, fuzzy Levenshtein spelling suggestions, and SQL synonym group generator.

---

## 2. Architecture & Design Principles
1. **Zero External Dependencies**: Standard Go library only (`strings`, `sync`, `unicode`, `sort`).
2. **Deterministic & Concurrency-Safe**: Default thread-safe engine (`sync.RWMutex`) with immutable copy isolation for custom dictionary registrations.
3. **Phonetic Normalization Rules**:
   - `ee` -> `i` (e.g. `"cheeni"` -> `"chini"`, `"paneer"` -> `"panir"`)
   - `oo` -> `u` (e.g. `"doodh"` -> `"dudh"`, `"sooji"` -> `"suji"`)
   - `sh` -> `s` (e.g. `"shakkar"` -> `"sakkar"`)
   - `z`  -> `j` (e.g. `"zeera"`   -> `"jeera"`)
4. **ReDoS / Buffer Exhaustion Guardrails**:
   - Queries capped at `MaxQueryLength = 120`.
   - Control characters and null bytes stripped.
   - Whitespace collapsed to single spaces.

---

## 3. Public API Surface

```go
type TermGroup struct {
    Terms []string
}

type Engine struct { /* ... */ }

func Default() *Engine
func New() *Engine

func (e *Engine) CleanQuery(q string) string
func (e *Engine) IsStopword(w string) bool
func (e *Engine) NormalizePhonetic(w string) string
func (e *Engine) Expand(rawQuery string) []TermGroup
func (e *Engine) ExpandFlat(rawQuery string) []string
func (e *Engine) Matches(query, candidate string) bool
func (e *Engine) Suggest(term string, maxDistance int) []string
func (e *Engine) RegisterSynonyms(term string, synonyms ...string)
func (e *Engine) RegisterStopword(word string)

// Package-level helpers
func CleanQuery(q string) string
func IsStopword(w string) bool
func NormalizePhonetic(w string) string
func Expand(rawQuery string) []TermGroup
func ExpandFlat(rawQuery string) []string
func Matches(query, candidate string) bool
func Suggest(term string, maxDistance int) []string
func LevenshteinDistance(s1, s2 string) int
func Similarity(s1, s2 string) float64
```

---

## 4. Performance Benchmarks
- `Matches`: ~407 ns/op, 96 B/op
- `LevenshteinDistance`: ~178 ns/op, 48 B/op
- `Expand`: ~1.1 μs/op, 456 B/op
