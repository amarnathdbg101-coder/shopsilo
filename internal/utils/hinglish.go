package utils

import (
	hinglish "github.com/amarnathdbg101-coder/go/go-hinglish"
)

// SearchTermGroup represents a group of synonyms for a given search word.
// Type alias ensuring 100% backward compatibility with existing repository callers.
type SearchTermGroup = hinglish.TermGroup

// CleanSearchQuery removes punctuation, strips null bytes, truncates excessive input,
// and normalizes whitespace using the go-hinglish engine.
func CleanSearchQuery(q string) string {
	return hinglish.CleanQuery(q)
}

// IsHinglishStopword checks if a word is an everyday Hinglish conversational filler.
func IsHinglishStopword(w string) bool {
	return hinglish.IsStopword(w)
}

// NormalizeHinglishPhonetic applies standard phonetic soundex rules to Indian colloquial spellings.
func NormalizeHinglishPhonetic(w string) string {
	return hinglish.NormalizePhonetic(w)
}

// ExpandHinglishSearchGroups breaks a multi-word or single-word search query into
// phonetic/bilingual synonym groups using go-hinglish.
func ExpandHinglishSearchGroups(rawQuery string) []SearchTermGroup {
	return hinglish.Expand(rawQuery)
}

// ExpandHinglishSearchFlat returns a unique deduplicated slice of all expanded synonyms.
func ExpandHinglishSearchFlat(rawQuery string) []string {
	return hinglish.ExpandFlat(rawQuery)
}

// MatchesHinglish checks whether candidate text matches query terms using go-hinglish.
func MatchesHinglish(query, candidate string) bool {
	return hinglish.Matches(query, candidate)
}
