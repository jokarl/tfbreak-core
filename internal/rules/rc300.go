package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RC300 detects when a module call's source URL changes
type RC300 struct{}

func init() {
	Register(&RC300{})
}

// ID returns the unique identifier for this rule.
func (r *RC300) ID() string {
	return "RC300"
}

// Name returns the human-readable name for this rule.
func (r *RC300) Name() string {
	return "module-source-changed"
}

// Description returns a description of what this rule detects.
func (r *RC300) Description() string {
	return "A module call's source URL changed, which may point to a different module implementation"
}

// DefaultSeverity returns the default severity level for this rule.
func (r *RC300) DefaultSeverity() types.Severity {
	return types.SeverityRisky
}

// Documentation returns the documentation for this rule.
func (r *RC300) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `module "vpc" {
  source  = "git::https://github.com/org/terraform-aws-vpc.git"
  version = "~> 3.0"
}`,
		ExampleNew: `module "vpc" {
  source  = "registry.terraform.io/org/vpc/aws"  # Source changed!
  version = "~> 3.0"
}`,
		Remediation: `This is a RISKY change because the module source URL changed.

Common scenarios:
- Migrating from Git to Terraform Registry (same module, different delivery)
- Reorganizing module paths (same content, different location)
- Pointing to a different module entirely (breaking change)

Before proceeding:
1. Verify the new source points to the same or compatible module
2. Check that the module interface (variables/outputs) is unchanged
3. Test the change in a non-production environment

Use an annotation if this change is intentional:
   # tfbreak:ignore module-source-changed # migrating to registry`,
	}
}

// Evaluate checks for module source changes between old and new snapshots.
func (r *RC300) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldModule := range old.Modules {
		newModule, exists := new.Modules[name]
		if !exists {
			// Module was removed - handled by BC101
			continue
		}

		// Check if source changed
		if oldModule.Source != newModule.Source {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Module %q source changed: %q -> %q",
					name, oldModule.Source, newModule.Source),
			).WithOldLocation(&oldModule.DeclRange).
				WithNewLocation(&newModule.DeclRange)

			findings = append(findings, finding)
		}
	}

	return findings
}
