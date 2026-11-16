# Technical Specification: go-humanize

## 1. Domain & Purpose
`go-humanize` is a high-efficiency text humanization and data formatting library for Go. It converts raw integers, timestamps, metrics, and quantities into clean, user-friendly strings supporting both standard international systems and the Indian numbering convention (Lakhs & Crores).

## 2. Architecture & Design Principles
- **Dual Numbering Standards**:
  - `FormatComma`: International grouping (e.g. `1,250,000`).
  - `FormatIndian`: Indian banking & currency grouping (e.g. `12,50,000`, `1,00,00,000`).
- **Compact Metric & Indian Notation**:
  - Compact SI: `1.5K`, `2.4M`, `1.2B`, `3.8T`.
  - Compact Indian: `1.5 Lakh`, `2.5 Cr`.
- **Ordinal Numbers**: Appends English ordinal suffixes (`1st`, `2nd`, `3rd`, `4th`, `11th`, `21st`) taking into account teen exceptions.
- **Relative Durations**: Converts timestamps to relative phrases (`just now`, `5 minutes ago`, `yesterday`, `3 months ago`, `2 years ago`).
- **Singular/Plural Quantities**: Helper for formatting counts with grammar (`Plural(count, singular, plural)`).
- **Zero Heavy Dependencies**: Pure standard library (`fmt`, `strings`, `strconv`, `math`, `time`).

## 3. API Surface
- `func FormatComma(n int64) string`
- `func FormatIndian(n int64) string`
- `func Compact(n int64) string`
- `func CompactIndian(n int64) string`
- `func Ordinal(n int) string`
- `func TimeAgo(t time.Time) string`
- `func Plural(count int, singular, plural string) string`

## 4. Error Handling
- Robust fallback representations without panics on edge cases (negative values, zeroes, future timestamps).

## 5. Verification & Testing
- Unit tests verify comma grouping, Indian grouping up to Crores, compact abbreviations, ordinal grammar rules, time deltas, and pluralization.
