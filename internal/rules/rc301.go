package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RC301 detects when a module call's version constraint changes
type RC301 struct{}

func init() {
	Register(&RC301{})
}

// ID returns the unique identifier for this rule.
func (r *RC301) ID() string {
	return "RC301"
}

// Name returns the human-readable name for this rule.
func (r *RC301) Name() string {
	return "module-version-changed"
}

// Description returns a description of what this rule detects.
func (r *RC301) Description() string {
	return "A module call's version constraint changed, which may pull in different module behavior"
}

// DefaultSeverity returns the default severity level for this rule.
func (r *RC301) DefaultSeverity() types.Severity {
	return types.SeverityWarning
}

// Documentation returns the documentation for this rule.
func (r *RC301) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `module "vpc" {
  source  = "registry.terraform.io/org/vpc/aws"
  version = "~> 3.0"
}`,
		ExampleNew: `module "vpc" {
  source  = "registry.terraform.io/org/vpc/aws"
  version = "~> 4.0"  # Version constraint changed!
}`,
		Remediation: `This is a RISKY change because the module version constraint changed.

Common scenarios:
- Major version bump (e.g., ~> 3.0 to ~> 4.0) - may include breaking changes
- Constraint tightened (e.g., >= 1.0 to >= 2.0) - excludes older versions
- Constraint removed - module version becomes unpinned
- Constraint added - module version becomes pinned

Before proceeding:
1. Review the module's changelog for breaking changes
2. Verify compatibility with the new version constraint
3. Test the change in a non-production environment

Use an annotation if this change is intentional:
   # tfbreak:ignore module-version-changed # upgrading to v4`,
	}
}

// Evaluate checks for module version constraint changes between old and new snapshots.
func (r *RC301) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldModule := range old.Modules {
		newModule, exists := new.Modules[name]
		if !exists {
			// Module was removed - handled by BC101
			continue
		}

		// Check if version constraint changed
		if oldModule.Version != newModule.Version {
			message := formatVersionChangeMessage(name, oldModule.Version, newModule.Version)

			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				message,
			).WithOldLocation(&oldModule.DeclRange).
				WithNewLocation(&newModule.DeclRange)

			findings = append(findings, finding)
		}
	}

	return findings
}

// formatVersionChangeMessage creates a descriptive message for version changes.
func formatVersionChangeMessage(moduleName, oldVersion, newVersion string) string {
	switch {
	case oldVersion == "" && newVersion != "":
		return fmt.Sprintf("Module %q version constraint added: %q", moduleName, newVersion)
	case oldVersion != "" && newVersion == "":
		return fmt.Sprintf("Module %q version constraint removed (was %q)", moduleName, oldVersion)
	default:
		return fmt.Sprintf("Module %q version constraint changed: %q -> %q",
			moduleName, oldVersion, newVersion)
	}
}
