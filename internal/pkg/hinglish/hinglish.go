package hinglish

import (
	"sort"
	"strings"
	"sync"
	"unicode"
)

// MaxQueryLength defines the safety boundary for query sanitation to prevent ReDoS / CPU buffer abuse.
const MaxQueryLength = 120

// TermGroup represents a set of bilingual and phonetic synonyms for a single search token.
// Suitable for SQL generation: (col ILIKE '%term1%' OR col ILIKE '%term2%').
type TermGroup struct {
	Terms []string
}

// Engine coordinates Hinglish and colloquial Hindi phonetic term expansion and dictionary matching.
type Engine struct {
	mu        sync.RWMutex
	dict      map[string][]string
	stopwords map[string]struct{}
}

var (
	defaultEngineOnce sync.Once
	defaultEngine     *Engine
)

// Default returns the process-wide shared Hinglish engine loaded with the standard FMCG/Retail dictionary.
func Default() *Engine {
	defaultEngineOnce.Do(func() {
		defaultEngine = New()
	})
	return defaultEngine
}

// New creates an independent, isolated Hinglish engine with default dictionary entries.
func New() *Engine {
	e := &Engine{
		dict:      make(map[string][]string, len(defaultDictionary)),
		stopwords: make(map[string]struct{}, len(defaultStopwords)),
	}

	for k, v := range defaultDictionary {
		copied := make([]string, len(v))
		copy(copied, v)
		e.dict[k] = copied
	}

	for k := range defaultStopwords {
		e.stopwords[k] = struct{}{}
	}

	return e
}

// RegisterSynonyms dynamically registers or augments synonyms for a specific colloquial term (Concurrency-safe).
func (e *Engine) RegisterSynonyms(term string, synonyms ...string) {
	if len(synonyms) == 0 {
		return
	}
	cleaned := CleanQuery(term)
	if cleaned == "" {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	existing := e.dict[cleaned]
	seen := make(map[string]bool, len(existing)+len(synonyms))
	for _, s := range existing {
		seen[s] = true
	}

	for _, s := range synonyms {
		c := strings.ToLower(strings.TrimSpace(s))
		if c != "" && !seen[c] {
			seen[c] = true
			existing = append(existing, c)
		}
	}
	e.dict[cleaned] = existing
}

// RegisterStopword dynamically registers a colloquial conversational filler.
func (e *Engine) RegisterStopword(word string) {
	w := strings.ToLower(strings.TrimSpace(word))
	if w == "" {
		return
	}
	e.mu.Lock()
	e.stopwords[w] = struct{}{}
	e.mu.Unlock()
}

// CleanQuery removes null bytes, punctuation, truncates excessive input, and normalizes spacing.
func CleanQuery(q string) string {
	q = strings.TrimSpace(q)
	q = strings.ReplaceAll(q, "\x00", " ")
	if len(q) > MaxQueryLength {
		q = q[:MaxQueryLength]
	}

	var sb strings.Builder
	sb.Grow(len(q))
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			sb.WriteRune(unicode.ToLower(r))
		} else {
			sb.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

// IsStopword checks if a word is an everyday Hinglish conversational filler.
func (e *Engine) IsStopword(w string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.stopwords[strings.ToLower(strings.TrimSpace(w))]
	return ok
}

// NormalizePhonetic applies standard phonetic soundex rules to Indian colloquial spellings:
// - ee -> i (e.g. "cheeni" -> "chini", "paneer" -> "panir")
// - oo -> u (e.g. "doodh" -> "dudh", "sooji" -> "suji")
// - sh -> s (e.g. "shakkar" -> "sakkar")
// - z  -> j (e.g. "zeera"   -> "jeera")
func NormalizePhonetic(w string) string {
	w = strings.ToLower(strings.TrimSpace(w))
	if len(w) < 3 {
		return w
	}
	w = strings.ReplaceAll(w, "ee", "i")
	w = strings.ReplaceAll(w, "oo", "u")
	w = strings.ReplaceAll(w, "sh", "s")
	w = strings.ReplaceAll(w, "z", "j")
	return w
}

// Expand splits a user query into phonetic/bilingual synonym groups while stripping stopwords.
func (e *Engine) Expand(rawQuery string) []TermGroup {
	cleaned := CleanQuery(rawQuery)
	if cleaned == "" {
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// 1. Direct multi-word key match (e.g., "cold drink", "tel baal")
	if syns, ok := e.dict[cleaned]; ok {
		return []TermGroup{{Terms: syns}}
	}

	words := strings.Fields(cleaned)
	if len(words) == 0 {
		return nil
	}

	// 2. Filter conversational stopwords unless all words are stopwords
	var nonStopwords []string
	for _, w := range words {
		if _, ok := e.stopwords[w]; !ok {
			nonStopwords = append(nonStopwords, w)
		}
	}
	if len(nonStopwords) > 0 {
		words = nonStopwords
	}

	var groups []TermGroup
	for _, word := range words {
		if len(word) == 0 {
			continue
		}

		// 3. Direct dictionary match
		if syns, ok := e.dict[word]; ok {
			groups = append(groups, TermGroup{Terms: syns})
			continue
		}

		// 4. Phonetic normalized lookup
		normalized := NormalizePhonetic(word)
		if syns, ok := e.dict[normalized]; ok {
			groups = append(groups, TermGroup{Terms: syns})
			continue
		}

		// 5. Plain word fallback
		groups = append(groups, TermGroup{Terms: []string{word}})
	}

	return groups
}

// ExpandFlat returns a unique deduplicated slice of all possible synonyms and phonetic variations across the query.
func (e *Engine) ExpandFlat(rawQuery string) []string {
	groups := e.Expand(rawQuery)
	if len(groups) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var flat []string
	for _, g := range groups {
		for _, t := range g.Terms {
			if !seen[t] {
				seen[t] = true
				flat = append(flat, t)
			}
		}
	}
	return flat
}

// Matches checks whether the given candidate text satisfies the search query.
// It returns true if EVERY search token group has at least one term matching in candidate.
func (e *Engine) Matches(query, candidate string) bool {
	groups := e.Expand(query)
	if len(groups) == 0 {
		return false
	}

	candLower := strings.ToLower(candidate)

	for _, g := range groups {
		matchedAny := false
		for _, term := range g.Terms {
			if strings.Contains(candLower, term) {
				matchedAny = true
				break
			}
		}
		if !matchedAny {
			return false
		}
	}

	return true
}

// Suggest finds the closest vocabulary terms matching a potentially misspelled term using Levenshtein distance.
func (e *Engine) Suggest(term string, maxDistance int) []string {
	cleaned := CleanQuery(term)
	if cleaned == "" {
		return nil
	}
	if maxDistance <= 0 {
		maxDistance = 2
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	type match struct {
		term string
		dist int
	}

	var candidates []match
	for key := range e.dict {
		d := LevenshteinDistance(cleaned, key)
		if d <= maxDistance {
			candidates = append(candidates, match{term: key, dist: d})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].dist != candidates[j].dist {
			return candidates[i].dist < candidates[j].dist
		}
		return candidates[i].term < candidates[j].term
	})

	result := make([]string, 0, min(len(candidates), 5))
	for i := 0; i < len(candidates) && i < 5; i++ {
		result = append(result, candidates[i].term)
	}

	return result
}

// Package-level global wrappers delegating to Default()

func Expand(rawQuery string) []TermGroup {
	return Default().Expand(rawQuery)
}

func ExpandFlat(rawQuery string) []string {
	return Default().ExpandFlat(rawQuery)
}

func Matches(query, candidate string) bool {
	return Default().Matches(query, candidate)
}

func IsStopword(w string) bool {
	return Default().IsStopword(w)
}

func Suggest(term string, maxDistance int) []string {
	return Default().Suggest(term, maxDistance)
}
