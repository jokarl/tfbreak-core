package output

import (
	"testing"
)

func TestNewRenderer(t *testing.T) {
	tests := []struct {
		format   Format
		wantType string
	}{
		{FormatText, "*output.TextRenderer"},
		{FormatJSON, "*output.JSONRenderer"},
		{FormatCompact, "*output.CompactRenderer"},
		{FormatCheckstyle, "*output.CheckstyleRenderer"},
		{FormatJUnit, "*output.JUnitRenderer"},
		{FormatSARIF, "*output.SARIFRenderer"},
		{"unknown", "*output.TextRenderer"}, // Default
		{"", "*output.TextRenderer"},        // Empty defaults to text
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			renderer := NewRenderer(tt.format, false)
			gotType := getTypeName(renderer)
			if gotType != tt.wantType {
				t.Errorf("NewRenderer(%q) = %s, want %s", tt.format, gotType, tt.wantType)
			}
		})
	}
}

func TestValidFormats(t *testing.T) {
	formats := ValidFormats()

	expected := []string{"text", "json", "compact", "checkstyle", "junit", "sarif"}
	if len(formats) != len(expected) {
		t.Errorf("ValidFormats() returned %d formats, want %d", len(formats), len(expected))
	}

	for _, exp := range expected {
		found := false
		for _, f := range formats {
			if f == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidFormats() missing %q", exp)
		}
	}
}

func TestIsValidFormat(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{"text", true},
		{"json", true},
		{"compact", true},
		{"checkstyle", true},
		{"junit", true},
		{"sarif", true},
		{"unknown", false},
		{"", false},
		{"TEXT", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			if got := IsValidFormat(tt.format); got != tt.valid {
				t.Errorf("IsValidFormat(%q) = %v, want %v", tt.format, got, tt.valid)
			}
		})
	}
}

func getTypeName(r Renderer) string {
	switch r.(type) {
	case *TextRenderer:
		return "*output.TextRenderer"
	case *JSONRenderer:
		return "*output.JSONRenderer"
	case *CompactRenderer:
		return "*output.CompactRenderer"
	case *CheckstyleRenderer:
		return "*output.CheckstyleRenderer"
	case *JUnitRenderer:
		return "*output.JUnitRenderer"
	case *SARIFRenderer:
		return "*output.SARIFRenderer"
	default:
		return "unknown"
	}
}
