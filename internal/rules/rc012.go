package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RC012 detects when validation blocks are added to a variable
type RC012 struct{}

func init() {
	Register(&RC012{})
}

// ID returns the unique identifier for this rule.
func (r *RC012) ID() string {
	return "RC012"
}

// Name returns the human-readable name for this rule.
func (r *RC012) Name() string {
	return "validation-added"
}

// Description returns a description of what this rule detects.
func (r *RC012) Description() string {
	return "Validation blocks were added to a variable, which may cause deployment failures for consumers"
}

// DefaultSeverity returns the default severity level for this rule.
func (r *RC012) DefaultSeverity() types.Severity {
	return types.SeverityWarning
}

// Documentation returns the documentation for this rule.
func (r *RC012) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `variable "environment" {
  type = string
}`,
		ExampleNew: `variable "environment" {
  type = string

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}`,
		Remediation: `This is a RISKY change because validation blocks were added to the variable.

Consumers passing values that don't meet the new validation criteria will
experience deployment failures when they upgrade to this version.

Before proceeding:
1. Ensure the validation criteria are not too restrictive
2. Document the new requirements in your changelog
3. Consider a deprecation period before enforcing strict validation

Use an annotation if this change is intentional:
   # tfbreak:ignore validation-added # adding input validation`,
	}
}

// Evaluate checks for validation blocks being added to variables.
func (r *RC012) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldVar := range old.Variables {
		newVar, exists := new.Variables[name]
		if !exists {
			// Variable was removed - handled by BC002
			continue
		}

		// Check if validation count increased
		if newVar.ValidationCount > oldVar.ValidationCount {
			var message string
			if oldVar.ValidationCount == 0 {
				message = fmt.Sprintf("Variable %q: validation block added (now has %d)",
					name, newVar.ValidationCount)
			} else {
				message = fmt.Sprintf("Variable %q: validation blocks increased from %d to %d",
					name, oldVar.ValidationCount, newVar.ValidationCount)
			}

			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				message,
			).WithOldLocation(&oldVar.DeclRange).
				WithNewLocation(&newVar.DeclRange)

			findings = append(findings, finding)
		}
	}

	return findings
}
