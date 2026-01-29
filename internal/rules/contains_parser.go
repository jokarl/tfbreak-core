package rules

import (
	"regexp"
	"strings"
)

// ContainsPattern represents a parsed contains([list], var.name) pattern
type ContainsPattern struct {
	Values   []string // The literal string values in the list
	VarName  string   // The variable name being checked
	RawExpr  string   // The original expression
}

// containsPatternRe matches contains([...], var.name) patterns
// This is a simplified regex that captures the list contents
var containsPatternRe = regexp.MustCompile(`contains\s*\(\s*\[([^\]]*)\]\s*,\s*var\.(\w+)\s*\)`)

// ParseContainsPattern attempts to parse a contains([list], var.name) pattern
// from a validation condition expression.
// Returns nil if the expression doesn't match the pattern or uses dynamic lists.
func ParseContainsPattern(condition string) *ContainsPattern {
	matches := containsPatternRe.FindStringSubmatch(condition)
	if matches == nil {
		return nil
	}

	listContent := strings.TrimSpace(matches[1])
	varName := matches[2]

	// Parse the list values
	values := parseStringList(listContent)
	if values == nil {
		// Could not parse as a literal string list (might have variables or expressions)
		return nil
	}

	return &ContainsPattern{
		Values:  values,
		VarName: varName,
		RawExpr: condition,
	}
}

// parseStringList parses a comma-separated list of string literals
// Returns nil if any element is not a string literal
func parseStringList(content string) []string {
	if content == "" {
		return []string{}
	}

	var values []string
	// Simple state machine to parse quoted strings
	var current strings.Builder
	inString := false
	stringChar := byte(0)
	escaped := false

	for i := 0; i < len(content); i++ {
		c := content[i]

		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if !inString {
			if c == '"' || c == '\'' {
				inString = true
				stringChar = c
				continue
			}
			if c == ',' {
				// End of element - but we should have captured something
				val := strings.TrimSpace(current.String())
				if val != "" {
					values = append(values, val)
				}
				current.Reset()
				continue
			}
			// Skip whitespace outside strings
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				continue
			}
			// Non-string content (variable reference, function call, etc.)
			return nil
		}

		// Inside a string
		if c == stringChar {
			inString = false
			stringChar = 0
			continue
		}

		current.WriteByte(c)
	}

	// Handle last element
	if inString {
		// Unclosed string
		return nil
	}

	val := strings.TrimSpace(current.String())
	if val != "" {
		values = append(values, val)
	}

	return values
}

// FindRemovedValues compares old and new contains patterns and returns removed values.
// A value is "removed" if it was in the old list but not in the new list.
func FindRemovedValues(oldPattern, newPattern *ContainsPattern) []string {
	if oldPattern == nil || newPattern == nil {
		return nil
	}

	// Build set of new values
	newSet := make(map[string]bool)
	for _, v := range newPattern.Values {
		newSet[v] = true
	}

	// Find values in old that are not in new
	var removed []string
	for _, v := range oldPattern.Values {
		if !newSet[v] {
			removed = append(removed, v)
		}
	}

	return removed
}
