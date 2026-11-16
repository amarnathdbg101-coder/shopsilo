# go-fuzzy 🔍

`go-fuzzy` ek high-performance, Unicode-aware fuzzy string matching aur similarity scoring library hai Go ke liye. Ye typo-tolerant search engines, e-commerce product catalogs, autocomplete suggestions, aur customer deduplication systems ke liye banaya gaya hai.

---

## ✨ Features

- **Multiple Matching Algorithms**:
  - **Levenshtein Distance**: Standard edit distance optimized with $O(\min(m, n))$ memory.
  - **Damerau-Levenshtein Distance**: Character transpositions (`teh` -> `the`) handle karta hai.
  - **Similarity Ratio**: 0.0 se 1.0 tak normalized match score.
  - **Jaro-Winkler Metric**: Short names aur terms ke prefix-weighted similarity scoring.
- **Search & Ranking API**: Candidates list me se `Rank()` aur `FindBestMatch()` se typo-tolerant search results nikalein.
- **Full Unicode / Multilingual**: Hindi (Devanagari) aur accented characters support.
- **Zero Third-Party Dependencies**: Pure Go Standard Library.

---

## 🚀 Quick Start

### Installation
```bash
go get github.com/shopsilo/go-fuzzy
```

### Complete Example
```go
package main

import (
	"fmt"

	"github.com/shopsilo/go-fuzzy"
)

func main() {
	// 1. Calculate Distances
	dist := fuzzy.Levenshtein("kitten", "sitting")
	fmt.Printf("Levenshtein distance: %d\n", dist) // 3

	// 2. Transposition check
	damDist := fuzzy.DamerauLevenshtein("teh", "the")
	fmt.Printf("Damerau distance: %d\n", damDist) // 1

	// 3. Typo-tolerant Product Search
	catalog := []string{
		"Apple iPhone 15 Pro",
		"Samsung Galaxy S24",
		"Google Pixel 8",
		"Sony WH-1000XM5",
	}

	// User typed "iphne"
	match, found := fuzzy.FindBestMatch("iphne", catalog)
	if found {
		fmt.Printf("Found match: %s (Score: %.2f)\n", match.Target, match.Score)
	}

	// 4. Jaro-Winkler for customer names
	nameScore := fuzzy.JaroWinkler("martha", "marhta")
	fmt.Printf("Name similarity: %.4f\n", nameScore) // ~0.96
}
```

---

## 🧪 Testing

```bash
go test -v -count=1 ./...
```
Output:
```
=== RUN   TestLevenshtein
--- PASS: TestLevenshtein (0.00s)
=== RUN   TestDamerauLevenshtein
--- PASS: TestDamerauLevenshtein (0.00s)
=== RUN   TestSimilarityRatio
--- PASS: TestSimilarityRatio (0.00s)
=== RUN   TestJaroWinkler
--- PASS: TestJaroWinkler (0.00s)
=== RUN   TestRankAndFindBestMatch
--- PASS: TestRankAndFindBestMatch (0.00s)
PASS
ok  	go-fuzzy	0.491s
```

---

## 📄 License
MIT License.
