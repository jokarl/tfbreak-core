package rules

// LevenshteinDistance calculates the minimum number of single-character edits
// (insertions, deletions, or substitutions) required to change one string into another.
func LevenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create a matrix of size (len(a)+1) x (len(b)+1)
	// We only need two rows at a time to save memory
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	// Initialize the first row
	for j := range prev {
		prev[j] = j
	}

	// Fill in the matrix
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			// Minimum of:
			// - deletion (prev[j] + 1)
			// - insertion (curr[j-1] + 1)
			// - substitution (prev[j-1] + cost)
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}

	return prev[len(b)]
}

// Similarity calculates a normalized similarity score between two strings.
// Returns a value between 0.0 (completely different) and 1.0 (identical).
// The formula is: 1 - (levenshtein_distance / max(len(a), len(b)))
func Similarity(a, b string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0 // Two empty strings are identical
	}

	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 1.0
	}

	distance := LevenshteinDistance(a, b)
	return 1.0 - float64(distance)/float64(maxLen)
}

// FindBestMatch finds the string in candidates that has the highest similarity to target.
// Returns the best matching string, its similarity score, and whether a match was found
// above the given threshold.
func FindBestMatch(target string, candidates []string, threshold float64) (string, float64, bool) {
	var bestMatch string
	var bestSimilarity float64

	for _, candidate := range candidates {
		sim := Similarity(target, candidate)
		if sim > bestSimilarity {
			bestSimilarity = sim
			bestMatch = candidate
		}
	}

	if bestSimilarity >= threshold {
		return bestMatch, bestSimilarity, true
	}

	return "", 0, false
}
