package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// RC003 detects when an optional variable is renamed (removed + similar optional variable added)
type RC003 struct{}

func init() {
	Register(&RC003{})
}

func (r *RC003) ID() string {
	return "RC003"
}

func (r *RC003) Name() string {
	return "input-renamed-optional"
}

func (r *RC003) Description() string {
	return "An optional variable was renamed, which may break callers that explicitly set the old name"
}

func (r *RC003) DefaultSeverity() types.Severity {
	return types.SeverityWarning
}

func (r *RC003) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `variable "timeout" {
  type    = string
  default = "30s"
}`,
		ExampleNew: `variable "timeout_seconds" {
  type    = string
  default = "30s"
}`,
		Remediation: `To fix this issue, either:
1. Keep the old variable name for backward compatibility
2. Add the old variable as a deprecated alias
3. Coordinate with callers who explicitly set this variable
4. Use an annotation if the rename is intentional:
   # tfbreak:ignore input-renamed-optional reason="coordinated rename"

Note: This rule fires for optional variables where we cannot statically determine
if callers are passing an explicit value. If no callers set this variable explicitly,
the rename has no impact.`,
	}
}

func (r *RC003) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	// Only run if rename detection is enabled
	if !IsRenameDetectionEnabled() {
		return nil
	}

	var findings []*types.Finding
	threshold := GetSimilarityThreshold()

	// Collect removed variables (exist in old, not in new)
	removedVars := make(map[string]*types.VariableSignature)
	for name, v := range old.Variables {
		if _, exists := new.Variables[name]; !exists {
			removedVars[name] = v
		}
	}

	// Collect added optional variables (exist in new, not in old, has default)
	addedOptionalVars := make([]string, 0)
	for name, v := range new.Variables {
		if _, exists := old.Variables[name]; !exists {
			// Check if it's optional (has default)
			if v.Default != nil {
				addedOptionalVars = append(addedOptionalVars, name)
			}
		}
	}

	// For each removed variable, try to find a matching added optional variable
	for oldName, oldVar := range removedVars {
		match, similarity, found := FindBestMatch(oldName, addedOptionalVars, threshold)
		if !found {
			continue
		}

		newVar := new.Variables[match]

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Variable %q was renamed to %q", oldName, match),
		).WithOldLocation(&oldVar.DeclRange).
			WithNewLocation(&newVar.DeclRange).
			WithDetail(fmt.Sprintf("Similarity: %.2f (threshold: %.2f). Callers explicitly setting %q will have their value ignored.", similarity, threshold, oldName)).
			WithMetadata("old_name", oldName).
			WithMetadata("new_name", match)

		findings = append(findings, finding)

		// Remove the matched variable from candidates so it can't match again
		for i, name := range addedOptionalVars {
			if name == match {
				addedOptionalVars = append(addedOptionalVars[:i], addedOptionalVars[i+1:]...)
				break
			}
		}
	}

	return findings
}
