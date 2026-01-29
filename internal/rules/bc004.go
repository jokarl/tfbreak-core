package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC004 detects when a variable's type constraint changes
type BC004 struct{}

func init() {
	Register(&BC004{})
}

func (r *BC004) ID() string {
	return "BC004"
}

func (r *BC004) Name() string {
	return "input-type-changed"
}

func (r *BC004) Description() string {
	return "A variable's type constraint changed, which may break callers passing values of the old type"
}

func (r *BC004) DefaultSeverity() types.Severity {
	return types.SeverityError
}

func (r *BC004) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `variable "instance_count" {
  type    = string
  default = "1"
}`,
		ExampleNew: `variable "instance_count" {
  type    = number  # Changed from string!
  default = 1
}`,
		Remediation: `This is a BREAKING change because callers passing string values will fail.
Consider:
1. Keep the original type and convert internally
2. Create a new variable with the new type and deprecate the old one
3. Coordinate with all callers to update their values
4. Use an annotation if all callers are updated in the same change:
   # tfbreak:ignore input-type-changed # coordinated type migration`,
	}
}

func (r *BC004) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldVar := range old.Variables {
		newVar, exists := new.Variables[name]
		if !exists {
			// Variable was removed - handled by BC002
			continue
		}

		// Normalize types for comparison
		oldType := normalizeType(oldVar.Type)
		newType := normalizeType(newVar.Type)

		// Check if type actually changed
		if oldType == newType {
			continue
		}

		// Check if this is a non-breaking change (any -> specific)
		if isAnyType(oldType) && !isAnyType(newType) {
			// Narrowing from any to specific type is safe
			continue
		}

		// All other type changes are breaking:
		// - specific -> any (widening)
		// - typeA -> typeB (incompatible)
		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Variable %q type changed: %s -> %s", name,
				formatType(oldType), formatType(newType)),
		).WithOldLocation(&oldVar.DeclRange).
			WithNewLocation(&newVar.DeclRange)

		findings = append(findings, finding)
	}

	return findings
}

// normalizeType normalizes a type expression for comparison.
// Empty string is treated as "any" (unspecified type).
func normalizeType(t string) string {
	if t == "" {
		return "any"
	}
	return t
}

// isAnyType checks if a type is "any" (accepts any value).
func isAnyType(t string) bool {
	return t == "any" || t == ""
}

// formatType formats a type for display.
func formatType(t string) string {
	if t == "" || t == "any" {
		return "any"
	}
	return t
}
