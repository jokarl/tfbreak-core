package annotation

import (
	"testing"
	"time"
)

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
			name: "block level ignore",
			src: `
# tfbreak:ignore BC001
variable "foo" {
  type = string
}
`,
			expected: 1,
		},
		{
			name: "multiple annotations",
			src: `# tfbreak:ignore-file

# tfbreak:ignore BC001
variable "foo" {
  type = string
}

# tfbreak:ignore BC002,BC003
variable "bar" {
  type = string
}
`,
			expected: 3,
		},
		{
			name: "with metadata",
			src: `# tfbreak:ignore BC001 reason="intentional" ticket="JIRA-123"
variable "foo" {
  type = string
}
`,
			expected: 1,
		},
		{
			name: "double slash comment",
			src: `// tfbreak:ignore BC001
variable "foo" {
  type = string
}
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anns, err := ParseFile("test.tf", []byte(tt.src))
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
			name:          "ignore-file with reason",
			text:          `tfbreak:ignore-file reason="generated file"`,
			expectedScope: ScopeFile,
			expectedRules:  nil,
			expectedReason: "generated file",
		},
		{
			name:          "ignore single rule",
			text:          "tfbreak:ignore BC001",
			expectedScope: ScopeBlock,
			expectedRules: []string{"BC001"},
		},
		{
			name:          "ignore multiple rules",
			text:          "tfbreak:ignore BC001,BC002,BC003",
			expectedScope: ScopeBlock,
			expectedRules: []string{"BC001", "BC002", "BC003"},
		},
		{
			name:          "ignore with spaces",
			text:          "tfbreak:ignore BC001, BC002",
			expectedScope: ScopeBlock,
			expectedRules: []string{"BC001", "BC002"},
		},
		{
			name:           "full metadata",
			text:           `tfbreak:ignore BC001 reason="intentional change" ticket="JIRA-123"`,
			expectedScope:  ScopeBlock,
			expectedRules:  []string{"BC001"},
			expectedReason: "intentional change",
			expectedTicket: "JIRA-123",
		},
		{
			name:          "with expires",
			text:          `tfbreak:ignore BC001 expires="2030-12-31"`,
			expectedScope: ScopeBlock,
			expectedRules: []string{"BC001"},
			hasExpires:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ann, err := parseAnnotation(tt.text, "test.tf", 1)
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

	// Specific rules
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
