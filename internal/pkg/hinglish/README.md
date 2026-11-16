# go-hinglish 🇮🇳

High-performance, zero-dependency Indian colloquial and Hinglish phonetic search expansion engine for Go.

[![Go Report Card](https://goreportcard.com/badge/github.com/amarnathdbg101-coder/go/go-hinglish)](https://goreportcard.com/report/github.com/amarnathdbg101-coder/go/go-hinglish)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## What is go-hinglish? (Hinglish me samjhein)

Jab Indian customers ya Kirana merchants search karte hain, wo kabhi bookish English nahi likhte:
- Customer likhta hai: **"doodh"** ya **"dudh"** ➡️ System ko match karna chahiye **"Amul Taaza Milk 500ml"**.
- Customer likhta hai: **"amul ka doodh aur cheeni"** ➡️ Filler word **"ka"**, **"aur"** hatna chahiye aur **"milk"** + **"sugar"** match hona chahiye.
- Customer likhta hai: **"tel"** ➡️ System ko **"Mustard Oil"**, **"Refined Oil"**, aur **"Sunflower Oil"** dikhana chahiye.

`go-hinglish` ek universal, pure-Go library hai jo colloquial Indian search terms ko sub-microsecond me expand karta hai, phonetic transliteration karta hai, aur SQL queries ke liye clean synonym groups produce karta hai.

---

## Features
- 🚀 **Sub-Microsecond Speed**: ~407 ns per match query, zero cgo.
- 📚 **Curated Indian Retail Dictionary**: 100+ colloquial retail categories (Dairy, Oils, Grains, Atta, Pulses, Masala, Chai, Personal Care, Detergents, Apparel).
- 🧹 **Stopword Sanitizer**: Filters conversational fillers like `"ka"`, `"ki"`, `"ke"`, `"chahiye"`, `"bhaiya"`.
- 🔤 **Phonetic Normalizer**: Indian soundex transliteration (`"cheeni"` ➡️ `"chini"`, `"zeera"` ➡️ `"jeera"`).
- 💡 **Levenshtein Spell Suggestion**: Auto-corrects minor typographical errors (`"dood"` ➡️ `"doodh"`).
- 🔒 **Concurrency-Safe**: Dynamic synonym registration using `sync.RWMutex`.

---

## Installation

```bash
go get github.com/amarnathdbg101-coder/go/go-hinglish@v1.0.0
```

---

## Quick Start

```go
package main

import (
	"fmt"
	hinglish "github.com/amarnathdbg101-coder/go/go-hinglish"
)

func main() {
	// 1. Check if colloquial query matches an English product catalog title
	matched := hinglish.Matches("doodh", "Amul Taaza Milk 500ml Pouch")
	fmt.Println("Matched:", matched) // true

	// 2. Expand search query into SQL-ready term groups
	groups := hinglish.Expand("amul ka doodh")
	for i, g := range groups {
		fmt.Printf("Group %d: %v\n", i+1, g.Terms)
	}
	// Group 1: ["amul"]
	// Group 2: ["doodh", "milk", "dudh", "dairy"]

	// 3. Get flat deduplicated synonym list
	synonyms := hinglish.ExpandFlat("fortune tel")
	fmt.Println("Synonyms:", synonyms)
	// ["fortune", "oil", "tel", "mustard", "sunflower", "refined", "sarson"]

	// 4. Fuzzy spelling correction
	suggestions := hinglish.Suggest("dood", 2)
	fmt.Println("Suggestions:", suggestions) // ["doodh", "dudh"]
}
```

---

## Benchmarks

```
BenchmarkMatches-8               2942031         407.0 ns/op          96 B/op          4 allocs/op
BenchmarkLevenshteinDistance-8   6848947         178.6 ns/op          48 B/op          1 allocs/op
BenchmarkExpand-8                 979143        1100.0 ns/op         456 B/op          9 allocs/op
```

---

## License
MIT License. Crafted with precision for high-scale retail & e-commerce applications.
