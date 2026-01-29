package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC010 detects when an output is renamed (removed + similar output added)
type BC010 struct{}

func init() {
	Register(&BC010{})
}

func (r *BC010) ID() string {
	return "BC010"
}

func (r *BC010) Name() string {
	return "output-renamed"
}

func (r *BC010) Description() string {
	return "An output was renamed, which will break callers referencing the old name"
}

func (r *BC010) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC010) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `output "vpc_id" {
  value = aws_vpc.main.id
}`,
		ExampleNew: `output "main_vpc_id" {
  value = aws_vpc.main.id
}`,
		Remediation: `To fix this issue, either:
1. Keep the old output name for backward compatibility
2. Add the old output as an alias pointing to the same value
3. Coordinate with all callers to update to the new output name
4. Use an annotation if the rename is intentional and coordinated:
   # tfbreak:ignore output-renamed reason="coordinated rename"`,
	}
}

func (r *BC010) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	// Only run if rename detection is enabled
	if !IsRenameDetectionEnabled() {
		return nil
	}

	var findings []*types.Finding
	threshold := GetSimilarityThreshold()

	// Collect removed outputs (exist in old, not in new)
	removedOutputs := make(map[string]*types.OutputSignature)
	for name, o := range old.Outputs {
		if _, exists := new.Outputs[name]; !exists {
			removedOutputs[name] = o
		}
	}

	// Collect added outputs (exist in new, not in old)
	addedOutputs := make([]string, 0)
	for name := range new.Outputs {
		if _, exists := old.Outputs[name]; !exists {
			addedOutputs = append(addedOutputs, name)
		}
	}

	// For each removed output, try to find a matching added output
	for oldName, oldOutput := range removedOutputs {
		match, similarity, found := FindBestMatch(oldName, addedOutputs, threshold)
		if !found {
			continue
		}

		newOutput := new.Outputs[match]

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Output %q was renamed to %q", oldName, match),
		).WithOldLocation(&oldOutput.DeclRange).
			WithNewLocation(&newOutput.DeclRange).
			WithDetail(fmt.Sprintf("Similarity: %.2f (threshold: %.2f)", similarity, threshold)).
			WithMetadata("old_name", oldName).
			WithMetadata("new_name", match)

		findings = append(findings, finding)

		// Remove the matched output from candidates so it can't match again
		for i, name := range addedOutputs {
			if name == match {
				addedOutputs = append(addedOutputs[:i], addedOutputs[i+1:]...)
				break
			}
		}
	}

	return findings
}
