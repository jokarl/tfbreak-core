package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC005 detects when a variable's default value is removed
type BC005 struct{}

func init() {
	Register(&BC005{})
}

func (r *BC005) ID() string {
	return "BC005"
}

func (r *BC005) Name() string {
	return "input-default-removed"
}

func (r *BC005) Description() string {
	return "A variable's default value was removed, making it required"
}

func (r *BC005) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC005) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldVar := range old.Variables {
		newVar, exists := new.Variables[name]
		if !exists {
			// Variable was removed entirely - handled by BC002
			continue
		}

		// Check if default was removed (was optional, now required)
		if !oldVar.Required && newVar.Required {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Variable %q default was removed, now required", name),
			).WithOldLocation(&oldVar.DeclRange).
				WithNewLocation(&newVar.DeclRange)

			findings = append(findings, finding)
		}
	}

	return findings
}
