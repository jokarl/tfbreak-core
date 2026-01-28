package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC102 detects invalid moved blocks
type BC102 struct{}

func init() {
	Register(&BC102{})
}

func (r *BC102) ID() string {
	return "BC102"
}

func (r *BC102) Name() string {
	return "invalid-moved-block"
}

func (r *BC102) Description() string {
	return "A moved block has invalid syntax or type mismatch between from/to addresses"
}

func (r *BC102) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC102) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for _, moved := range new.MovedBlocks {
		// Check for type mismatch (resource to module or vice versa)
		fromIsResource := types.IsResourceAddress(moved.From)
		fromIsModule := types.IsModuleAddress(moved.From)
		toIsResource := types.IsResourceAddress(moved.To)
		toIsModule := types.IsModuleAddress(moved.To)

		if fromIsResource && toIsModule {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Moved block has type mismatch: cannot move resource %q to module %q", moved.From, moved.To),
			).WithNewLocation(&moved.DeclRange)

			findings = append(findings, finding)
			continue
		}

		if fromIsModule && toIsResource {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Moved block has type mismatch: cannot move module %q to resource %q", moved.From, moved.To),
			).WithNewLocation(&moved.DeclRange)

			findings = append(findings, finding)
			continue
		}

		// If neither is a valid resource or module address, that's also invalid
		if !fromIsResource && !fromIsModule {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Moved block has invalid 'from' address: %q", moved.From),
			).WithNewLocation(&moved.DeclRange)

			findings = append(findings, finding)
			continue
		}

		if !toIsResource && !toIsModule {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Moved block has invalid 'to' address: %q", moved.To),
			).WithNewLocation(&moved.DeclRange)

			findings = append(findings, finding)
			continue
		}
	}

	return findings
}
