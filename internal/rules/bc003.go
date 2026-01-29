package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC003 detects when a required variable is renamed (removed + similar required variable added)
type BC003 struct{}

func init() {
	Register(&BC003{})
}

func (r *BC003) ID() string {
	return "BC003"
}

func (r *BC003) Name() string {
	return "input-renamed"
}

func (r *BC003) Description() string {
	return "A required variable was renamed, which will break callers using the old name"
}

func (r *BC003) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC003) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `variable "api_key" {
  type = string
}`,
		ExampleNew: `variable "api_key_v2" {
  type = string
}`,
		Remediation: `To fix this issue, either:
1. Keep the old variable name for backward compatibility
2. Add the old variable as an alias that passes through to the new one
3. Coordinate with all callers to update to the new variable name
4. Use an annotation if the rename is intentional and coordinated:
   # tfbreak:ignore input-renamed reason="coordinated rename"`,
	}
}

func (r *BC003) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
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

	// Collect added required variables (exist in new, not in old, no default)
	addedRequiredVars := make([]string, 0)
	for name, v := range new.Variables {
		if _, exists := old.Variables[name]; !exists {
			// Check if it's required (no default)
			if v.Default == nil {
				addedRequiredVars = append(addedRequiredVars, name)
			}
		}
	}

	// For each removed variable, try to find a matching added required variable
	for oldName, oldVar := range removedVars {
		match, similarity, found := FindBestMatch(oldName, addedRequiredVars, threshold)
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
			WithDetail(fmt.Sprintf("Similarity: %.2f (threshold: %.2f)", similarity, threshold)).
			WithMetadata("old_name", oldName).
			WithMetadata("new_name", match)

		findings = append(findings, finding)

		// Remove the matched variable from candidates so it can't match again
		for i, name := range addedRequiredVars {
			if name == match {
				addedRequiredVars = append(addedRequiredVars[:i], addedRequiredVars[i+1:]...)
				break
			}
		}
	}

	return findings
}
