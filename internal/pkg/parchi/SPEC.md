# SPEC: `go-parchi` — Universal Kirana & Grocery Slip Parser

## 1. Overview & Problem Statement
In retail POS, quick-commerce, and kirana store software, customers frequently order items by:
1. Spoken voice commands: `"2 kg cheeni, 1 packet amul butter aur 50 ka tel"`
2. Handwritten paper slips (parchi) entered as multi-line text or WhatsApp lists:
   ```
   1. 2 kg cheeni 80 rs
   2) 1 packet butter 55 rs
   - 50 ka tel
   ```
Relying entirely on external LLMs (such as Gemini or OpenAI) for 100% of these inputs causes:
- Unnecessary token consumption and high latency (hundreds of milliseconds or seconds).
- Network dependency and downtime if offline or flaky internet connection.
- High cost for high-volume counter operations.

`go-parchi` provides an offline, deterministic, sub-microsecond parser that decodes conversational Indian retail units, price-first syntax, quantity-first syntax, and compound command splitters. If a slip or sentence contains conversational ambiguities or questions (e.g. `"bhaiya sasta kaunsa hai"`), `go-parchi` safely signals `ok=false` so the caller can selectively delegate only ambiguous queries to an LLM.

---

## 2. Architecture & Design Principles
- **Pure Standard Library**: Zero 3rd party runtime dependencies. Pure Go 1.22+.
- **Zero-Token Fast Path**: Parse unambiguous slips in ~14 μs.
- **Deterministic Sound Unit Mapping**: Normalizes 40+ Indian retail units across Hindi, Hinglish, and English (`kilo` -> `kg`, `pkt` -> `packet`, `ltr` -> `litre`, `bora` -> `sack`, `darjan` -> `dozen`).
- **Conversational Filter**: Proactively detects 20+ conversational intent words (`"bhaiya"`, `"chahiye"`, `"discount"`, `"sasta"`, etc.) and yields to LLMs.
- **Title Case Formatting**: Auto-normalizes product names to clean human-readable title casing.

---

## 3. Public API Specification
```go
package parchi

type Item struct {
    ProductName string  `json:"product_name"`
    Quantity    float64 `json:"quantity"`
    Unit        string  `json:"unit"`
    UnitPrice   float64 `json:"unit_price"`
    TotalPrice  float64 `json:"total_price"`
    RawLine     string  `json:"raw_line,omitempty"`
}

type UnmatchedLine struct {
    OriginalLine string `json:"original_line"`
    CleanedLine  string `json:"cleaned_line"`
    Reason       string `json:"reason"`
}

type Slip struct {
    TotalLinesParsed int             `json:"total_lines_parsed"`
    MatchedCount     int             `json:"matched_count"`
    UnmatchedCount   int             `json:"unmatched_count"`
    EstimatedTotal   float64         `json:"estimated_total"`
    Items            []Item          `json:"items"`
    UnmatchedLines   []UnmatchedLine `json:"unmatched_lines"`
}

func CleanTitle(s string) string
func SplitCommands(input string) []string
func NormalizeUnit(raw string) string
func HasConversationalStopWords(text string) bool
func ParseSingleLine(line string) (*Item, bool)
func ParseSlip(rawText string) (*Slip, bool)
```

---

## 4. Benchmarks & Performance Targets
- `ParseSingleLine`: < 10 μs / op (Achieved: ~6.6 μs)
- `ParseSlip`: < 30 μs / op (Achieved: ~14.5 μs)
- `SplitCommands`: < 5 μs / op (Achieved: ~3.1 μs)
