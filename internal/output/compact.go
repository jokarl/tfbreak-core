package output

import (
	"fmt"
	"io"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// CompactRenderer renders output in a condensed single-line-per-issue format
// This format is useful for logs and quick scanning
type CompactRenderer struct{}

// Render writes the check result in compact format
// Format: filename:line:column: severity: [rule_id] message
func (r *CompactRenderer) Render(w io.Writer, result *types.CheckResult) error {
	for _, f := range result.Findings {
		if f.Ignored {
			continue
		}

		// Determine the location to use (prefer new, fallback to old)
		filename := "<unknown>"
		line := 0
		col := 0

		if f.NewLocation != nil {
			filename = f.NewLocation.Filename
			line = f.NewLocation.Line
			col = f.NewLocation.Column
		} else if f.OldLocation != nil {
			filename = f.OldLocation.Filename
			line = f.OldLocation.Line
			col = f.OldLocation.Column
		}

		severity := f.Severity.String()

		fmt.Fprintf(w, "%s:%d:%d: %s: [%s] %s\n",
			filename, line, col, severity, f.RuleID, f.Message)
	}

	return nil
}
