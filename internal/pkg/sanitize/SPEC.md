# SPEC: go-sanitize

## 1. Metadata
- **Module**: `go-sanitize`
- **Package**: `gosanitize`
- **Location**: `C:\Library\go-sanitize`
- **Go Version**: `1.23+`
- **Dependencies**: Zero external dependencies (100% Go Standard Library: `strings`, `unicode`, `unicode/utf8`, `bytes`, `net/url`, `html`).

---

## 2. Core Capabilities
1. **XSS Neutralization & HTML Sanitization**:
   - `SanitizeHTML(input, policy)` retains whitelisted HTML tags (`<b>`, `<i>`, `<p>`, `<a>`) while discarding unauthorized elements.
   - Drops entire dangerous elements (`<script>`, `<iframe>`, `<object>`, `<embed>`, `<style>`) along with their inner contents.
   - Validates URLs on `href` and `src`, blocking pseudo-protocols like `javascript:`, `data:`, `vbscript:`.
2. **Plain Text Extraction (`StripHTML`)**:
   - Strips all markup and decodes HTML entities into clean plain text.
3. **SQL Wildcard Defense (`EscapeSQLWildcards`)**:
   - Escapes `%` and `_` with `\%` and `\_` to prevent accidental full-table scans in user search queries.
4. **Unicode Homoglyph & Invisible Character Sanitization (`CleanUnicode`)**:
   - Strips zero-width spaces (`\u200B`), soft hyphens, and bidi override markers (`\u202E`).

---

## 3. Public API
```go
type Policy struct {
    AllowedTags        map[string]bool
    AllowedAttributes  map[string]map[string]bool
    AllowedURLSchemes  map[string]bool
    DropEntireElements map[string]bool
}

var UGCPolicy Policy

func StripHTML(input string) string
func SanitizeHTML(input string, policy Policy) string
func EscapeSQLWildcards(input string) string
func CleanUnicode(input string) string
```
