package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC002 detects when an existing variable is removed
type BC002 struct{}

func init() {
	Register(&BC002{})
}

func (r *BC002) ID() string {
	return "BC002"
}

func (r *BC002) Name() string {
	return "input-removed"
}

func (r *BC002) Description() string {
	return "A variable was removed, which will break callers that provide this variable"
}

func (r *BC002) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC002) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldVar := range old.Variables {
		// Check if variable was removed
		if _, exists := new.Variables[name]; exists {
			continue
		}

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Variable %q was removed", name),
		).WithOldLocation(&oldVar.DeclRange)

		findings = append(findings, finding)
	}

	return findings
}
