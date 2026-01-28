package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC009 detects when an output is removed
type BC009 struct{}

func init() {
	Register(&BC009{})
}

func (r *BC009) ID() string {
	return "BC009"
}

func (r *BC009) Name() string {
	return "output-removed"
}

func (r *BC009) Description() string {
	return "An output was removed, which will break callers that reference this output"
}

func (r *BC009) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC009) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldOutput := range old.Outputs {
		// Check if output was removed
		if _, exists := new.Outputs[name]; exists {
			continue
		}

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Output %q was removed", name),
		).WithOldLocation(&oldOutput.DeclRange)

		findings = append(findings, finding)
	}

	return findings
}
