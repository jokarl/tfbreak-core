package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC200 detects when terraform required_version constraint is added or changed
type BC200 struct{}

func init() {
	Register(&BC200{})
}

func (r *BC200) ID() string {
	return "BC200"
}

func (r *BC200) Name() string {
	return "terraform-version-constrained"
}

func (r *BC200) Description() string {
	return "Terraform required_version constraint was added or changed, which may break CI pipelines using older versions"
}

func (r *BC200) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC200) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `terraform {
  # No version constraint
}`,
		ExampleNew: `terraform {
  required_version = ">= 1.5.0"  # New constraint added
}`,
		Remediation: `This is a BREAKING change because CI pipelines or users with older Terraform versions will fail.

Common scenarios:
- Adding a constraint where none existed forces version upgrades
- Tightening a constraint (e.g., ">= 1.0" to ">= 1.5") excludes older versions

Before making this change:
1. Verify all CI pipelines support the required version
2. Communicate the requirement change to module consumers
3. Consider using a range constraint (e.g., ">= 1.0, < 2.0") for flexibility

Use an annotation if this is intentional:
   # tfbreak:ignore terraform-version-constrained # minimum version bump for new features`,
	}
}

func (r *BC200) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	oldVersion := old.RequiredVersion
	newVersion := new.RequiredVersion

	// No finding if constraint was removed (loosening is safe)
	if oldVersion != "" && newVersion == "" {
		return findings
	}

	// No finding if unchanged
	if oldVersion == newVersion {
		return findings
	}

	// Constraint was added or changed
	var message string
	if oldVersion == "" {
		message = fmt.Sprintf("Terraform required_version constraint added: %q", newVersion)
	} else {
		message = fmt.Sprintf("Terraform required_version changed: %q -> %q", oldVersion, newVersion)
	}

	finding := types.NewFinding(
		r.ID(),
		r.Name(),
		r.DefaultSeverity(),
		message,
	)

	// No specific file location for terraform block in current data model
	// Future enhancement could track terraform block location

	findings = append(findings, finding)

	return findings
}
