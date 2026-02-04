package cli

import (
	"strings"
	"testing"
)

func TestIndent(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		prefix string
		want   string
	}{
		{
			name:   "single line",
			text:   "hello",
			prefix: "  ",
			want:   "  hello",
		},
		{
			name:   "multiple lines",
			text:   "line1\nline2\nline3",
			prefix: "  ",
			want:   "  line1\n  line2\n  line3",
		},
		{
			name:   "empty lines preserved",
			text:   "line1\n\nline2",
			prefix: "  ",
			want:   "  line1\n\n  line2",
		},
		{
			name:   "empty text",
			text:   "",
			prefix: "  ",
			want:   "",
		},
		{
			name:   "only empty line",
			text:   "\n",
			prefix: "  ",
			want:   "\n",
		},
		{
			name:   "different prefix",
			text:   "text",
			prefix: ">>> ",
			want:   ">>> text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indent(tt.text, tt.prefix)
			if got != tt.want {
				t.Errorf("indent(%q, %q) = %q, want %q", tt.text, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestRunExplain_KnownRule(t *testing.T) {
	// Test that a known rule ID doesn't cause a panic
	// We can't easily capture stdout, but we can verify no error is returned
	err := runExplain(nil, []string{"BC001"})
	if err != nil {
		t.Errorf("runExplain returned error for known rule: %v", err)
	}
}

func TestRunExplain_CaseInsensitive(t *testing.T) {
	// Test that rule IDs are case-insensitive
	err := runExplain(nil, []string{"bc001"})
	if err != nil {
		t.Errorf("runExplain returned error for lowercase rule ID: %v", err)
	}
}

func TestRunExplain_ValidRuleIDs(t *testing.T) {
	// Test a few known rule IDs that should exist
	ruleIDs := []string{
		"BC001",
		"BC002",
		"BC003",
		"RC003",
	}

	for _, id := range ruleIDs {
		t.Run(id, func(t *testing.T) {
			err := runExplain(nil, []string{id})
			if err != nil {
				t.Errorf("runExplain(%s) returned error: %v", id, err)
			}
		})
	}
}

func TestIndent_WhitespaceHandling(t *testing.T) {
	// Test that leading/trailing whitespace is handled correctly
	result := indent("  text with spaces  ", ">>")
	if !strings.HasPrefix(result, ">>") {
		t.Errorf("expected prefix to be added, got: %q", result)
	}
}
