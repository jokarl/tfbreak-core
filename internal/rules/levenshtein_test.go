package rules

import (
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{
			name:     "identical strings",
			a:        "hello",
			b:        "hello",
			expected: 0,
		},
		{
			name:     "empty strings",
			a:        "",
			b:        "",
			expected: 0,
		},
		{
			name:     "first empty",
			a:        "",
			b:        "hello",
			expected: 5,
		},
		{
			name:     "second empty",
			a:        "hello",
			b:        "",
			expected: 5,
		},
		{
			name:     "one character difference",
			a:        "hello",
			b:        "hallo",
			expected: 1,
		},
		{
			name:     "one insertion",
			a:        "hello",
			b:        "helloo",
			expected: 1,
		},
		{
			name:     "one deletion",
			a:        "hello",
			b:        "hell",
			expected: 1,
		},
		{
			name:     "completely different",
			a:        "abc",
			b:        "xyz",
			expected: 3,
		},
		{
			name:     "kitten to sitting",
			a:        "kitten",
			b:        "sitting",
			expected: 3,
		},
		{
			name:     "variable rename simple",
			a:        "foo",
			b:        "foo_v2",
			expected: 3,
		},
		{
			name:     "variable rename prefix",
			a:        "instance_type",
			b:        "vm_instance_type",
			expected: 3,
		},
		{
			name:     "variable rename suffix",
			a:        "api_key",
			b:        "api_key_v2",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LevenshteinDistance(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
			// Test symmetry - distance should be the same in both directions
			reverse := LevenshteinDistance(tt.b, tt.a)
			if reverse != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d (symmetry check)", tt.b, tt.a, reverse, tt.expected)
			}
		})
	}
}

func TestSimilarity(t *testing.T) {
	tests := []struct {
		name        string
		a           string
		b           string
		minExpected float64
		maxExpected float64
	}{
		{
			name:        "identical strings",
			a:           "hello",
			b:           "hello",
			minExpected: 1.0,
			maxExpected: 1.0,
		},
		{
			name:        "empty strings",
			a:           "",
			b:           "",
			minExpected: 1.0,
			maxExpected: 1.0,
		},
		{
			name:        "first empty",
			a:           "",
			b:           "hello",
			minExpected: 0.0,
			maxExpected: 0.0,
		},
		{
			name:        "completely different same length",
			a:           "abc",
			b:           "xyz",
			minExpected: 0.0,
			maxExpected: 0.0,
		},
		{
			name:        "one character difference",
			a:           "hello",
			b:           "hallo",
			minExpected: 0.79,
			maxExpected: 0.81,
		},
		{
			name:        "variable rename foo to foo_v2",
			a:           "foo",
			b:           "foo_v2",
			minExpected: 0.49,
			maxExpected: 0.51,
		},
		{
			name:        "variable rename api_key to api_key_v2",
			a:           "api_key",
			b:           "api_key_v2",
			minExpected: 0.69,
			maxExpected: 0.71,
		},
		{
			name:        "variable rename instance_type to vm_instance_type",
			a:           "instance_type",
			b:           "vm_instance_type",
			minExpected: 0.80,
			maxExpected: 0.82,
		},
		{
			name:        "variable rename timeout to timeout_seconds",
			a:           "timeout",
			b:           "timeout_seconds",
			minExpected: 0.46,
			maxExpected: 0.48,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Similarity(tt.a, tt.b)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("Similarity(%q, %q) = %f, want between %f and %f", tt.a, tt.b, result, tt.minExpected, tt.maxExpected)
			}
			// Test symmetry
			reverse := Similarity(tt.b, tt.a)
			if reverse < tt.minExpected || reverse > tt.maxExpected {
				t.Errorf("Similarity(%q, %q) = %f, want between %f and %f (symmetry check)", tt.b, tt.a, reverse, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestSimilarity_BoundaryValues(t *testing.T) {
	// Test that similarity is always between 0 and 1
	testCases := []struct {
		a string
		b string
	}{
		{"", ""},
		{"a", ""},
		{"", "a"},
		{"hello", "world"},
		{"abcdefghij", "klmnopqrst"},
		{"test", "test"},
	}

	for _, tc := range testCases {
		sim := Similarity(tc.a, tc.b)
		if sim < 0.0 || sim > 1.0 {
			t.Errorf("Similarity(%q, %q) = %f, want value between 0.0 and 1.0", tc.a, tc.b, sim)
		}
	}
}

func TestFindBestMatch(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		candidates     []string
		threshold      float64
		expectedMatch  string
		expectedFound  bool
		minSimilarity  float64
	}{
		{
			name:           "exact match in candidates",
			target:         "foo",
			candidates:     []string{"bar", "foo", "baz"},
			threshold:      0.85,
			expectedMatch:  "foo",
			expectedFound:  true,
			minSimilarity:  1.0,
		},
		{
			name:           "best match above threshold",
			target:         "api_key",
			candidates:     []string{"secret", "api_key_v2", "token"},
			threshold:      0.70,
			expectedMatch:  "api_key_v2",
			expectedFound:  true,
			minSimilarity:  0.70,
		},
		{
			name:           "no match above threshold",
			target:         "foo",
			candidates:     []string{"completely", "different", "names"},
			threshold:      0.85,
			expectedMatch:  "",
			expectedFound:  false,
			minSimilarity:  0.0,
		},
		{
			name:           "empty candidates",
			target:         "foo",
			candidates:     []string{},
			threshold:      0.85,
			expectedMatch:  "",
			expectedFound:  false,
			minSimilarity:  0.0,
		},
		{
			name:           "multiple similar candidates picks best",
			target:         "database_host",
			candidates:     []string{"database_hostname", "database_url", "db_host"},
			threshold:      0.50,
			expectedMatch:  "database_hostname",
			expectedFound:  true,
			minSimilarity:  0.50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, similarity, found := FindBestMatch(tt.target, tt.candidates, tt.threshold)
			if found != tt.expectedFound {
				t.Errorf("FindBestMatch found = %v, want %v", found, tt.expectedFound)
			}
			if match != tt.expectedMatch {
				t.Errorf("FindBestMatch match = %q, want %q", match, tt.expectedMatch)
			}
			if found && similarity < tt.minSimilarity {
				t.Errorf("FindBestMatch similarity = %f, want >= %f", similarity, tt.minSimilarity)
			}
		})
	}
}
