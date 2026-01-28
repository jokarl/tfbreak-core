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

func (r *BC002) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `variable "instance_type" {
  type    = string
  default = "t3.micro"
}`,
		ExampleNew: `# Variable was removed`,
		Remediation: `To fix this issue, either:
1. Keep the variable (even if unused) for backward compatibility
2. Deprecate the variable first (add description noting deprecation)
3. Coordinate with all callers to remove the variable usage
4. Use an annotation if removal is intentional:
   # tfbreak:ignore BC002 reason="deprecated variable removed"`,
	}
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
