package rules

import (
	"fmt"
	"strings"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RC013 detects when allowed values are removed from a contains() validation pattern
type RC013 struct{}

func init() {
	Register(&RC013{})
}

// ID returns the unique identifier for this rule.
func (r *RC013) ID() string {
	return "RC013"
}

// Name returns the human-readable name for this rule.
func (r *RC013) Name() string {
	return "validation-value-removed"
}

// Description returns a description of what this rule detects.
func (r *RC013) Description() string {
	return "Allowed values were removed from a contains() validation, which may break consumers using those values"
}

// DefaultSeverity returns the default severity level for this rule.
func (r *RC013) DefaultSeverity() types.Severity {
	return types.SeverityWarning
}

// Documentation returns the documentation for this rule.
func (r *RC013) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `variable "environment" {
  type = string

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}`,
		ExampleNew: `variable "environment" {
  type = string

  validation {
    condition     = contains(["dev", "staging"], var.environment)  # "prod" removed!
    error_message = "Environment must be dev or staging."
  }
}`,
		Remediation: `This is a RISKY change because allowed values were removed from a validation.

Consumers using the removed values will experience deployment failures when
they upgrade to this version.

Before proceeding:
1. Ensure no consumers are using the removed values
2. Document the deprecation of the removed values
3. Consider a deprecation period before removing values

Use an annotation if this change is intentional:
   # tfbreak:ignore validation-value-removed # deprecating prod environment`,
	}
}

// Evaluate checks for removed values in contains() validation patterns.
func (r *RC013) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldVar := range old.Variables {
		newVar, exists := new.Variables[name]
		if !exists {
			// Variable was removed - handled by BC002
			continue
		}

		// Compare contains() patterns in validations
		removed := findRemovedContainsValues(oldVar.Validations, newVar.Validations)
		if len(removed) > 0 {
			message := fmt.Sprintf("Variable %q: allowed values removed from validation: %s",
				name, formatRemovedValues(removed))

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

// findRemovedContainsValues compares old and new validations to find removed contains() values
func findRemovedContainsValues(oldValidations, newValidations []types.ValidationBlock) []string {
	// Build map of old contains patterns by variable name
	oldPatterns := make(map[string]*ContainsPattern)
	for _, v := range oldValidations {
		pattern := ParseContainsPattern(v.Condition)
		if pattern != nil {
			oldPatterns[pattern.VarName] = pattern
		}
	}

	// Build map of new contains patterns by variable name
	newPatterns := make(map[string]*ContainsPattern)
	for _, v := range newValidations {
		pattern := ParseContainsPattern(v.Condition)
		if pattern != nil {
			newPatterns[pattern.VarName] = pattern
		}
	}

	// Find removed values across all patterns
	var allRemoved []string
	for varName, oldPattern := range oldPatterns {
		newPattern, exists := newPatterns[varName]
		if !exists {
			// The contains() validation was removed entirely
			// This could mean validation was loosened, not a risky change
			continue
		}

		removed := FindRemovedValues(oldPattern, newPattern)
		allRemoved = append(allRemoved, removed...)
	}

	return allRemoved
}

// formatRemovedValues formats a list of removed values for display
func formatRemovedValues(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(quoted, ", ")
}
