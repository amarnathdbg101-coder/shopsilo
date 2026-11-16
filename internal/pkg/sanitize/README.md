# go-sanitize 🧼

High-speed, zero-dependency HTML sanitizer, XSS neutralizer, SQL wildcard escaper, and invisible Unicode cleaner for user-generated content. Adheres strictly to the [Master Library Blueprint](../BLUEPRINT.md).

---

## Features
- 🛡️ **XSS Protection**: Neutralizes `<script>`, `<iframe>`, and `onerror` exploits in user reviews and blog comments.
- 🔗 **Protocol Sanitizer**: `javascript:`, `data:`, aur `vbscript:` schemes ko drop karta hai jabki valid `https:` links ko preserve karta hai.
- 🧹 **Plain Text Extraction (`StripHTML`)**: Kisi bhi rich HTML content ko clean, entity-decoded text me convert karta hai.
- 🔍 **SQL LIKE Wildcard Defense**: `%` aur `_` ko automatically escape karta hai taaki search input se full table scans na hon.
- 🥷 **Anti-Stealth Unicode Cleaner**: Invisible zero-width spaces (`\u200B`) aur right-to-left override (`\u202E`) attacks ko strip karta hai.
- 📦 **Zero External Dependencies**: 100% pure Go standard library.

---

## Quick Start

### 1. Sanitizing User Reviews (Allowing Safe Tags Only)
```go
package main

import (
	"fmt"
	gosanitize "go-sanitize"
)

func main() {
	userInput := `<p>Great product! <b>Highly recommended</b>. <script>alert('steal-cookie')</script></p>`

	cleanHTML := gosanitize.SanitizeHTML(userInput, gosanitize.UGCPolicy)
	// Output: <p>Great product! <b>Highly recommended</b>. </p>
	fmt.Println("Sanitized HTML:", cleanHTML)

	plainText := gosanitize.StripHTML(userInput)
	// Output: Great product! Highly recommended.
	fmt.Println("Plain Text:", plainText)
}
```

### 2. Escaping SQL Wildcards in Search Queries
```go
// User enters "100% original_leather"
searchQuery := gosanitize.EscapeSQLWildcards(userInput)
// Safe for: db.Query("SELECT * FROM products WHERE name ILIKE $1", "%" + searchQuery + "%")
```

---

## Running Tests
```bash
go test -v -count=1 ./...
```
