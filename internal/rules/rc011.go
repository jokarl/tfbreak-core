package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RC011 detects when an output's sensitive attribute changes
type RC011 struct{}

func init() {
	Register(&RC011{})
}

func (r *RC011) ID() string {
	return "RC011"
}

func (r *RC011) Name() string {
	return "output-sensitive-changed"
}

func (r *RC011) Description() string {
	return "An output's sensitive attribute changed, which affects plan visibility and downstream consumers"
}

func (r *RC011) DefaultSeverity() types.Severity {
	return types.SeverityRisky
}

func (r *RC011) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `output "connection_string" {
  value     = "postgres://${var.host}:${var.port}"
  sensitive = false
}`,
		ExampleNew: `output "connection_string" {
  value     = "postgres://${var.host}:${var.port}"
  sensitive = true  # Now marked as sensitive
}`,
		Remediation: `This is a RISKY change because it affects how the output appears in plans and logs.

If sensitive was added (false -> true):
- Output values will be redacted in terraform plan/apply output
- Downstream modules consuming this output must handle sensitive values
- State file will mark the value as sensitive

If sensitive was removed (true -> false):
- Output values may now appear in plain text in logs and plans
- Review security implications before removing sensitive marking

Use an annotation if this change is intentional:
   # tfbreak:ignore output-sensitive-changed # security classification updated`,
	}
}

func (r *RC011) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldOutput := range old.Outputs {
		newOutput, exists := new.Outputs[name]
		if !exists {
			// Output was removed - handled by BC009
			continue
		}

		// Check if sensitive actually changed
		if oldOutput.Sensitive == newOutput.Sensitive {
			continue
		}

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Output %q sensitive changed: %v -> %v", name, oldOutput.Sensitive, newOutput.Sensitive),
		).WithOldLocation(&oldOutput.DeclRange).
			WithNewLocation(&newOutput.DeclRange)

		findings = append(findings, finding)
	}

	return findings
}
