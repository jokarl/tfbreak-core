package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RC007 detects when a variable's nullable attribute changes
type RC007 struct{}

func init() {
	Register(&RC007{})
}

func (r *RC007) ID() string {
	return "RC007"
}

func (r *RC007) Name() string {
	return "input-nullable-changed"
}

func (r *RC007) Description() string {
	return "A variable's nullable attribute changed, which may cause callers passing null to fail"
}

func (r *RC007) DefaultSeverity() types.Severity {
	return types.SeverityRisky
}

func (r *RC007) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `variable "optional_config" {
  type     = string
  default  = null
  nullable = true  # Accepts null values
}`,
		ExampleNew: `variable "optional_config" {
  type     = string
  default  = ""
  nullable = false  # No longer accepts null!
}`,
		Remediation: `This is a RISKY change because callers explicitly passing null will fail.
Consider:
1. Keep nullable = true if callers may pass null
2. Provide a meaningful default value for callers passing null
3. Coordinate with callers to stop passing null
4. Use an annotation if this is intentional:
   # tfbreak:ignore input-nullable-changed # null no longer valid input`,
	}
}

func (r *RC007) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for name, oldVar := range old.Variables {
		newVar, exists := new.Variables[name]
		if !exists {
			// Variable was removed - handled by BC002
			continue
		}

		// Get effective nullable values (nil defaults to true)
		oldNullable := oldVar.IsNullable()
		newNullable := newVar.IsNullable()

		// Check if nullable actually changed
		if oldNullable == newNullable {
			continue
		}

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Variable %q nullable changed: %v -> %v", name, oldNullable, newNullable),
		).WithOldLocation(&oldVar.DeclRange).
			WithNewLocation(&newVar.DeclRange)

		findings = append(findings, finding)
	}

	return findings
}
