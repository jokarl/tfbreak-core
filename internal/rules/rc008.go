package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RC008 detects when a variable's sensitive attribute changes
type RC008 struct{}

func init() {
	Register(&RC008{})
}

func (r *RC008) ID() string {
	return "RC008"
}

func (r *RC008) Name() string {
	return "input-sensitive-changed"
}

func (r *RC008) Description() string {
	return "A variable's sensitive attribute changed, which may affect downstream outputs and logging"
}

func (r *RC008) DefaultSeverity() types.Severity {
	return types.SeverityWarning
}

func (r *RC008) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `variable "api_key" {
  type      = string
  sensitive = false
}`,
		ExampleNew: `variable "api_key" {
  type      = string
  sensitive = true  # Now marked as sensitive
}`,
		Remediation: `This is a RISKY change because it affects how the value appears in plans and logs.

If sensitive was added (false -> true):
- Values will be redacted in terraform plan output
- Dependent outputs must also be marked sensitive

If sensitive was removed (true -> false):
- Values may now appear in plain text in logs
- Review security implications before removing

Use an annotation if this change is intentional:
   # tfbreak:ignore input-sensitive-changed # security classification updated`,
	}
}

func (r *RC008) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldVar := range old.Variables {
		newVar, exists := new.Variables[name]
		if !exists {
			// Variable was removed - handled by BC002
			continue
		}

		// Check if sensitive actually changed
		if oldVar.Sensitive == newVar.Sensitive {
			continue
		}

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Variable %q sensitive changed: %v -> %v", name, oldVar.Sensitive, newVar.Sensitive),
		).WithOldLocation(&oldVar.DeclRange).
			WithNewLocation(&newVar.DeclRange)

		findings = append(findings, finding)
	}

	return findings
}
