# Technical Specification: go-fuzzy

## 1. Domain & Purpose
`go-fuzzy` is a high-performance, Unicode-aware fuzzy string matching and text similarity library for Go. It provides edit distance calculations, prefix-weighted similarity scoring, and ranking utilities for typo-tolerant product searches, autocomplete systems, and data deduplication.

## 2. Architecture & Design Principles
- **Multi-Algorithm Scoring Suite**:
  - **Levenshtein Distance**: Standard edit distance optimized with single-row `O(min(m, n))` space.
  - **Damerau-Levenshtein Distance**: Detects adjacent character transpositions (`teh` -> `the`).
  - **SimilarityRatio**: Normalized 0.0 to 1.0 scoring with case-insensitivity option.
  - **Jaro-Winkler Similarity**: Tailored for short strings and names with Winkler prefix boost (up to 4 runes).
- **Search & Ranking Utilities**: `Rank()` filters and sorts candidate lists by match quality; `FindBestMatch()` retrieves the single highest-scoring item.
- **Unicode Support**: Iterates through `[]rune` to support multilingual scripts (including Devanagari/Hindi).
- **Zero Heavy Dependencies**: Pure standard library (`math`, `strings`, `sort`).

## 3. API Surface
- `type Match struct { Target string; Score float64; Index int }`
- `func Levenshtein(s1, s2 string) int`
- `func DamerauLevenshtein(s1, s2 string) int`
- `func SimilarityRatio(s1, s2 string, ignoreCase bool) float64`
- `func JaroWinkler(s1, s2 string) float64`
- `func Rank(query string, candidates []string, minScore float64) []Match`
- `func FindBestMatch(query string, candidates []string) (Match, bool)`

## 4. Error Handling
- `ErrEmptyInput`: Returned when performing operations requiring non-empty inputs.

## 5. Verification & Testing
- Unit tests verify Levenshtein and Damerau distances, transposition scoring differences, normalized ratio ranges, Jaro-Winkler prefix boosts, Unicode strings, and candidate catalog ranking.
