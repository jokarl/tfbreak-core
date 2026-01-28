package annotation

import (
	"testing"
	"time"
)

// testResolver is a resolver for testing that maps rule names to IDs
type testResolver struct {
	nameToID map[string]string
}

func (r *testResolver) ResolveRuleID(name string) (string, bool) {
	if id, ok := r.nameToID[name]; ok {
		return id, true
	}
	return "", false
}

// newTestResolver creates a test resolver with common rule mappings
func newTestResolver() *testResolver {
	return &testResolver{
		nameToID: map[string]string{
			"required-input-added":     "BC001",
			"input-removed":            "BC002",
			"input-type-changed":       "BC004",
			"input-default-removed":    "BC005",
			"output-removed":           "BC009",
			"resource-removed-no-moved": "BC100",
			"module-removed-no-moved":  "BC101",
			"input-default-changed":    "RC006",
			"input-nullable-changed":   "RC007",
			"input-sensitive-changed":  "RC008",
			"output-sensitive-changed": "RC011",
		},
	}
}

func TestParseFile(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected int // number of annotations
	}{
		{
			name: "no annotations",
			src: `
variable "foo" {
  type = string
}
`,
			expected: 0,
		},
		{
			name: "file level ignore",
			src: `# tfbreak:ignore-file
variable "foo" {
  type = string
}
`,
			expected: 1,
		},
		{
			name: "block level ignore with rule name",
			src: `
# tfbreak:ignore required-input-added
variable "foo" {
  type = string
}
`,
			expected: 1,
		},
		{
			name: "multiple annotations",
			src: `# tfbreak:ignore-file

# tfbreak:ignore required-input-added
variable "foo" {
  type = string
}

# tfbreak:ignore input-removed, input-type-changed
variable "bar" {
  type = string
}
`,
			expected: 3,
		},
		{
			name: "with trailing comment reason",
			src: `# tfbreak:ignore required-input-added # intentional change
variable "foo" {
  type = string
}
`,
			expected: 1,
		},
		{
			name: "double slash comment",
			src: `// tfbreak:ignore required-input-added
variable "foo" {
  type = string
}
`,
			expected: 1,
		},
		{
			name: "ignore all keyword",
			src: `# tfbreak:ignore all
variable "foo" {
  type = string
}
`,
			expected: 1,
		},
	}

	parser := NewParser(newTestResolver())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anns, err := parser.ParseFile("test.tf", []byte(tt.src))
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			if len(anns) != tt.expected {
				t.Errorf("expected %d annotations, got %d", tt.expected, len(anns))
			}
		})
	}
}

func TestParseAnnotation(t *testing.T) {
	resolver := newTestResolver()
	parser := NewParser(resolver)

	tests := []struct {
		name           string
		text           string
		expectedScope  Scope
		expectedRules  []string
		expectedReason string
		expectedTicket string
		hasExpires     bool
	}{
		{
			name:          "ignore-file no rules",
			text:          "tfbreak:ignore-file",
			expectedScope: ScopeFile,
			expectedRules: nil,
		},
		{
			name:           "ignore-file with trailing comment reason",
			text:           `tfbreak:ignore-file # generated file`,
			expectedScope:  ScopeFile,
			expectedRules:  nil,
			expectedReason: "generated file",
		},
		{
			name:          "ignore single rule by name",
			text:          "tfbreak:ignore required-input-added",
			expectedScope: ScopeBlock,
			expectedRules: []string{"BC001"},
		},
		{
			name:          "ignore multiple rules by name",
			text:          "tfbreak:ignore required-input-added,input-removed,input-type-changed",
			expectedScope: ScopeBlock,
			expectedRules: []string{"BC001", "BC002", "BC004"},
		},
		{
			name:          "ignore with spaces between rules",
			text:          "tfbreak:ignore required-input-added, input-removed",
			expectedScope: ScopeBlock,
			expectedRules: []string{"BC001", "BC002"},
		},
		{
			name:           "rule name with trailing comment reason",
			text:           `tfbreak:ignore required-input-added # intentional change`,
			expectedScope:  ScopeBlock,
			expectedRules:  []string{"BC001"},
			expectedReason: "intentional change",
		},
		{
			name:          "ignore all keyword",
			text:          "tfbreak:ignore all",
			expectedScope: ScopeBlock,
			expectedRules: nil, // empty means match all
		},
		{
			name:           "ignore all with reason",
			text:           "tfbreak:ignore all # temporary workaround",
			expectedScope:  ScopeBlock,
			expectedRules:  nil,
			expectedReason: "temporary workaround",
		},
		{
			name:           "legacy metadata format still works",
			text:           `tfbreak:ignore required-input-added reason="intentional change" ticket="JIRA-123"`,
			expectedScope:  ScopeBlock,
			expectedRules:  []string{"BC001"},
			expectedReason: "intentional change",
			expectedTicket: "JIRA-123",
		},
		{
			name:          "legacy expires still works",
			text:          `tfbreak:ignore required-input-added expires="2030-12-31"`,
			expectedScope: ScopeBlock,
			expectedRules: []string{"BC001"},
			hasExpires:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ann, err := parser.parseAnnotation(tt.text, "test.tf", 1)
			if err != nil {
				t.Fatalf("parseAnnotation failed: %v", err)
			}
			if ann == nil {
				t.Fatal("expected annotation, got nil")
			}

			if ann.Scope != tt.expectedScope {
				t.Errorf("expected scope %v, got %v", tt.expectedScope, ann.Scope)
			}

			if len(ann.RuleIDs) != len(tt.expectedRules) {
				t.Errorf("expected %d rules, got %d: %v", len(tt.expectedRules), len(ann.RuleIDs), ann.RuleIDs)
			} else {
				for i, expected := range tt.expectedRules {
					if ann.RuleIDs[i] != expected {
						t.Errorf("rule %d: expected %s, got %s", i, expected, ann.RuleIDs[i])
					}
				}
			}

			if ann.Reason != tt.expectedReason {
				t.Errorf("expected reason %q, got %q", tt.expectedReason, ann.Reason)
			}

			if ann.Ticket != tt.expectedTicket {
				t.Errorf("expected ticket %q, got %q", tt.expectedTicket, ann.Ticket)
			}

			if tt.hasExpires && ann.Expires == nil {
				t.Error("expected expires to be set")
			}
			if !tt.hasExpires && ann.Expires != nil {
				t.Error("expected expires to be nil")
			}
		})
	}
}

func TestParseAnnotation_UnknownRuleName(t *testing.T) {
	resolver := newTestResolver()
	parser := NewParser(resolver)

	// Unknown rule names should result in empty RuleIDs (not resolved)
	ann, err := parser.parseAnnotation("tfbreak:ignore unknown-rule", "test.tf", 1)
	if err != nil {
		t.Fatalf("parseAnnotation failed: %v", err)
	}
	if ann == nil {
		t.Fatal("expected annotation, got nil")
	}

	// Unknown rules are not added to RuleIDs
	if len(ann.RuleIDs) != 0 {
		t.Errorf("expected 0 rules for unknown rule name, got %d: %v", len(ann.RuleIDs), ann.RuleIDs)
	}
}

func TestAnnotationIsExpired(t *testing.T) {
	// Not expired
	future := time.Now().Add(24 * time.Hour)
	ann := &Annotation{Expires: &future}
	if ann.IsExpired() {
		t.Error("expected annotation to not be expired")
	}

	// Expired
	past := time.Now().Add(-24 * time.Hour)
	ann = &Annotation{Expires: &past}
	if !ann.IsExpired() {
		t.Error("expected annotation to be expired")
	}

	// No expiration
	ann = &Annotation{}
	if ann.IsExpired() {
		t.Error("expected annotation without expires to not be expired")
	}
}

func TestAnnotationMatchesRule(t *testing.T) {
	// Empty rules matches all
	ann := &Annotation{RuleIDs: nil}
	if !ann.MatchesRule("BC001") {
		t.Error("expected empty rules to match any rule")
	}

	// Specific rules (using resolved IDs)
	ann = &Annotation{RuleIDs: []string{"BC001", "BC002"}}
	if !ann.MatchesRule("BC001") {
		t.Error("expected BC001 to match")
	}
	if !ann.MatchesRule("BC002") {
		t.Error("expected BC002 to match")
	}
	if ann.MatchesRule("BC003") {
		t.Error("expected BC003 to not match")
	}
}

func TestFindBlockStarts(t *testing.T) {
	src := `
variable "foo" {
  type = string
}

resource "aws_s3_bucket" "main" {
  bucket = "test"
}

output "bar" {
  value = "test"
}
`
	blocks, err := FindBlockStarts("test.tf", []byte(src))
	if err != nil {
		t.Fatalf("FindBlockStarts failed: %v", err)
	}

	// Should have 3 blocks
	if len(blocks) != 3 {
		t.Errorf("expected 3 blocks, got %d: %v", len(blocks), blocks)
	}

	// Check block types (lines may vary based on parsing)
	hasVariable := false
	hasResource := false
	hasOutput := false
	for _, blockType := range blocks {
		switch blockType {
		case "variable":
			hasVariable = true
		case "resource":
			hasResource = true
		case "output":
			hasOutput = true
		}
	}

	if !hasVariable {
		t.Error("expected to find variable block")
	}
	if !hasResource {
		t.Error("expected to find resource block")
	}
	if !hasOutput {
		t.Error("expected to find output block")
	}
}
