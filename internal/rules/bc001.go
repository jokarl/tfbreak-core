package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC001 detects when a new required variable is added
type BC001 struct{}

func init() {
	Register(&BC001{})
}

func (r *BC001) ID() string {
	return "BC001"
}

func (r *BC001) Name() string {
	return "required-input-added"
}

func (r *BC001) Description() string {
	return "A new variable was added without a default value, which will break existing callers"
}

func (r *BC001) DefaultSeverity() types.Severity {
	return types.SeverityError
}

func (r *BC001) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `# No variable defined`,
		ExampleNew: `variable "cluster_name" {
  type        = string
  description = "Name of the cluster"
  # No default - this breaks existing callers!
}`,
		Remediation: `To fix this issue, either:
1. Add a default value to make the variable optional:
   variable "cluster_name" {
     type    = string
     default = "my-cluster"
   }

2. Or ensure all callers are updated to provide the new variable

3. Or use an annotation to suppress if callers are updated in the same change:
   # tfbreak:ignore required-input-added # callers updated
   variable "cluster_name" { ... }`,
	}
}

func (r *BC001) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, newVar := range new.Variables {
		// Skip if variable existed in old
		if _, exists := old.Variables[name]; exists {
			continue
		}

		// Skip if variable has a default (is optional)
		if !newVar.Required {
			continue
		}

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("New required variable %q has no default", name),
		).WithNewLocation(&newVar.DeclRange)

		findings = append(findings, finding)
	}

	return findings
}
