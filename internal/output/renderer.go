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
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// NewRenderer creates a renderer for the given format
func NewRenderer(format Format, colorEnabled bool) Renderer {
	switch format {
	case FormatJSON:
		return &JSONRenderer{}
	default:
		return &TextRenderer{ColorEnabled: colorEnabled}
	}
}
