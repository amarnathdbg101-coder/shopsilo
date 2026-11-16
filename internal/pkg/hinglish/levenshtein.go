package hinglish

// LevenshteinDistance computes the minimum edit distance between two strings.
// Optimized using a single sliding array to avoid heap allocations.
func LevenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Use smaller dimension for column buffer to minimize space
	if len1 > len2 {
		r1, r2 = r2, r1
		len1, len2 = len2, len1
	}

	// Allocate a slice on stack/heap
	v0 := make([]int, len1+1)
	for i := 0; i <= len1; i++ {
		v0[i] = i
	}

	for j := 1; j <= len2; j++ {
		prev := v0[0]
		v0[0] = j
		r2Char := r2[j-1]

		for i := 1; i <= len1; i++ {
			temp := v0[i]
			cost := 1
			if r1[i-1] == r2Char {
				cost = 0
			}

			// Minimum of deletion, insertion, substitution
			minVal := v0[i] + 1 // deletion
			if insertVal := v0[i-1] + 1; insertVal < minVal {
				minVal = insertVal
			}
			if subVal := prev + cost; subVal < minVal {
				minVal = subVal
			}

			prev = temp
			v0[i] = minVal
		}
	}

	return v0[len1]
}

// Similarity returns a normalized ratio between 0.0 (completely distinct) and 1.0 (exact match).
func Similarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	maxLen := max(len([]rune(s1)), len([]rune(s2)))
	if maxLen == 0 {
		return 1.0
	}
	dist := LevenshteinDistance(s1, s2)
	return 1.0 - (float64(dist) / float64(maxLen))
}
