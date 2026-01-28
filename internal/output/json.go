package output

import (
	"encoding/json"
	"io"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// JSONRenderer renders output in JSON format
type JSONRenderer struct{}

// jsonOutput is the structure for JSON output
type jsonOutput struct {
	Version  string          `json:"version"`
	OldPath  string          `json:"old_path"`
	NewPath  string          `json:"new_path"`
	Findings []*types.Finding `json:"findings"`
	Summary  types.Summary   `json:"summary"`
	Result   string          `json:"result"`
	FailOn   string          `json:"fail_on"`
}

// Render writes the check result in JSON format
func (r *JSONRenderer) Render(w io.Writer, result *types.CheckResult) error {
	output := jsonOutput{
		Version:  "1.0",
		OldPath:  result.OldPath,
		NewPath:  result.NewPath,
		Findings: result.Findings,
		Summary:  result.Summary,
		Result:   result.Result,
		FailOn:   result.FailOn.String(),
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
