package fuzzy

import (
	"math"
	"sort"
	"strings"
)

// Match represents a candidate string paired with its similarity score (0.0 to 1.0).
type Match struct {
	Target string
	Score  float64
	Index  int
}

// Levenshtein calculates the minimum edit distance between two strings (insert, delete, substitute).
// Time complexity: O(m*n), Space complexity: O(min(m, n)). Unicode-aware via runes.
func Levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n1, n2 := len(r1), len(r2)

	if n1 == 0 {
		return n2
	}
	if n2 == 0 {
		return n1
	}

	// Ensure r2 is shorter to optimize memory
	if n1 < n2 {
		r1, r2 = r2, r1
		n1, n2 = n2, n1
	}

	row := make([]int, n2+1)
	for j := 0; j <= n2; j++ {
		row[j] = j
	}

	for i := 1; i <= n1; i++ {
		prev := i - 1
		row[0] = i
		for j := 1; j <= n2; j++ {
			old := row[j]
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			row[j] = min(row[j]+1, row[j-1]+1, prev+cost)
			prev = old
		}
	}

	return row[n2]
}

// DamerauLevenshtein calculates edit distance including adjacent character transpositions.
func DamerauLevenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n1, n2 := len(r1), len(r2)

	if n1 == 0 {
		return n2
	}
	if n2 == 0 {
		return n1
	}

	d := make([][]int, n1+1)
	for i := range d {
		d[i] = make([]int, n2+1)
		d[i][0] = i
	}
	for j := 0; j <= n2; j++ {
		d[0][j] = j
	}

	for i := 1; i <= n1; i++ {
		for j := 1; j <= n2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}

			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)

			// Transposition
			if i > 1 && j > 1 && r1[i-1] == r2[j-2] && r1[i-2] == r2[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}

	return d[n1][n2]
}

// SimilarityRatio returns a normalized score between 0.0 (no match) and 1.0 (exact match)
// based on Levenshtein distance. Case-insensitive comparison if ignoreCase is true.
func SimilarityRatio(s1, s2 string, ignoreCase bool) float64 {
	if ignoreCase {
		s1 = strings.ToLower(s1)
		s2 = strings.ToLower(s2)
	}

	if s1 == s2 {
		return 1.0
	}

	r1, r2 := []rune(s1), []rune(s2)
	maxLen := max(len(r1), len(r2))
	if maxLen == 0 {
		return 1.0
	}

	dist := Levenshtein(s1, s2)
	return 1.0 - (float64(dist) / float64(maxLen))
}

// JaroWinkler calculates the Jaro-Winkler similarity metric (0.0 to 1.0).
// Heavily favors strings with identical prefixes, optimal for short names and terms.
func JaroWinkler(s1, s2 string) float64 {
	r1, r2 := []rune(s1), []rune(s2)
	n1, n2 := len(r1), len(r2)

	if n1 == 0 && n2 == 0 {
		return 1.0
	}
	if n1 == 0 || n2 == 0 {
		return 0.0
	}

	matchDistance := max(n1, n2)/2 - 1
	if matchDistance < 0 {
		matchDistance = 0
	}

	s1Matches := make([]bool, n1)
	s2Matches := make([]bool, n2)

	matches := 0
	for i := 0; i < n1; i++ {
		start := max(0, i-matchDistance)
		end := min(i+matchDistance+1, n2)

		for j := start; j < end; j++ {
			if s2Matches[j] || r1[i] != r2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	transpositions := 0
	k := 0
	for i := 0; i < n1; i++ {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	jaro := ((m / float64(n1)) + (m / float64(n2)) + ((m - float64(transpositions/2)) / m)) / 3.0

	// Winkler prefix scaling (up to 4 matching prefix runes)
	prefixLen := 0
	maxPrefix := min(4, min(n1, n2))
	for i := 0; i < maxPrefix; i++ {
		if r1[i] == r2[i] {
			prefixLen++
		} else {
			break
		}
	}

	return jaro + (float64(prefixLen) * 0.1 * (1.0 - jaro))
}

// Rank scores each candidate in candidates against query, filtering by minScore and sorting descending by score.
func Rank(query string, candidates []string, minScore float64) []Match {
	var matches []Match
	qLower := strings.ToLower(query)

	for i, c := range candidates {
		cLower := strings.ToLower(c)
		score := SimilarityRatio(qLower, cLower, false)
		if score >= minScore {
			matches = append(matches, Match{
				Target: c,
				Score:  score,
				Index:  i,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if math.Abs(matches[i].Score-matches[j].Score) < 1e-9 {
			return len(matches[i].Target) < len(matches[j].Target)
		}
		return matches[i].Score > matches[j].Score
	})

	return matches
}

// FindBestMatch finds the highest scoring candidate from a slice. Returns false if candidates is empty or no match.
func FindBestMatch(query string, candidates []string) (Match, bool) {
	matches := Rank(query, candidates, 0.0)
	if len(matches) == 0 {
		return Match{}, false
	}
	return matches[0], true
}
