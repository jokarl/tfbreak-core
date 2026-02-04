package output

import (
	"io"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// Renderer defines the interface for output renderers
type Renderer interface {
	// Render writes the check result to the writer
	Render(w io.Writer, result *types.CheckResult) error
}

// Format represents an output format
type Format string

const (
	FormatText       Format = "text"
	FormatJSON       Format = "json"
	FormatCompact    Format = "compact"
	FormatCheckstyle Format = "checkstyle"
	FormatJUnit      Format = "junit"
	FormatSARIF      Format = "sarif"
)

// ValidFormats returns all valid output format names
func ValidFormats() []string {
	return []string{
		string(FormatText),
		string(FormatJSON),
		string(FormatCompact),
		string(FormatCheckstyle),
		string(FormatJUnit),
		string(FormatSARIF),
	}
}

// IsValidFormat returns true if the format is valid
func IsValidFormat(format string) bool {
	for _, f := range ValidFormats() {
		if f == format {
			return true
		}
	}
	return false
}

// NewRenderer creates a renderer for the given format
func NewRenderer(format Format, colorEnabled bool) Renderer {
	switch format {
	case FormatJSON:
		return &JSONRenderer{}
	case FormatCompact:
		return &CompactRenderer{}
	case FormatCheckstyle:
		return &CheckstyleRenderer{}
	case FormatJUnit:
		return &JUnitRenderer{}
	case FormatSARIF:
		return &SARIFRenderer{}
	default:
		return &TextRenderer{ColorEnabled: colorEnabled}
	}
}
