package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC103 detects conflicting moved blocks
type BC103 struct{}

func init() {
	Register(&BC103{})
}

func (r *BC103) ID() string {
	return "BC103"
}

func (r *BC103) Name() string {
	return "conflicting-moved"
}

func (r *BC103) Description() string {
	return "Moved blocks have conflicts: duplicate from addresses, cycles, or non-existent to targets"
}

func (r *BC103) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC103) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	// Build maps for analysis
	fromToMap := make(map[string]string)    // from -> to
	fromLocs := make(map[string]*types.FileRange) // from -> location
	seenFrom := make(map[string]bool)

	// Check for duplicate "from" addresses
	for _, moved := range new.MovedBlocks {
		if seenFrom[moved.From] {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Duplicate moved block 'from' address: %q", moved.From),
			).WithNewLocation(&moved.DeclRange)

			findings = append(findings, finding)
		} else {
			seenFrom[moved.From] = true
			fromToMap[moved.From] = moved.To
			loc := moved.DeclRange // copy to avoid pointer issues
			fromLocs[moved.From] = &loc
		}
	}

	// Check for cycles
	cycleFindings := r.detectCycles(new.MovedBlocks, fromToMap, fromLocs)
	findings = append(findings, cycleFindings...)

	// Check for non-existent "to" targets
	targetFindings := r.checkTargetsExist(new)
	findings = append(findings, targetFindings...)

	return findings
}

// detectCycles checks for cycles in moved block chains
func (r *BC103) detectCycles(movedBlocks []*types.MovedBlock, fromToMap map[string]string, fromLocs map[string]*types.FileRange) []*types.Finding {
	var findings []*types.Finding
	reported := make(map[string]bool)

	for _, moved := range movedBlocks {
		// Follow the chain from this moved block
		visited := make(map[string]bool)
		current := moved.From

		for {
			if visited[current] {
				// Found a cycle
				if !reported[current] {
					finding := types.NewFinding(
						r.ID(),
						r.Name(),
						r.DefaultSeverity(),
						fmt.Sprintf("Moved blocks form a cycle involving %q", current),
					).WithNewLocation(fromLocs[current])

					findings = append(findings, finding)
					reported[current] = true
				}
				break
			}

			visited[current] = true
			next, exists := fromToMap[current]
			if !exists {
				// No more chain
				break
			}

			// Check if the "to" address is also a "from" address (chain continues)
			if _, hasNext := fromToMap[next]; hasNext {
				current = next
			} else {
				break
			}
		}
	}

	return findings
}

// checkTargetsExist verifies that all "to" addresses exist in the new config
func (r *BC103) checkTargetsExist(new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	// Build set of existing addresses in new config
	existingResources := make(map[string]bool)
	for addr := range new.Resources {
		existingResources[addr] = true
	}

	existingModules := make(map[string]bool)
	for name := range new.Modules {
		existingModules[fmt.Sprintf("module.%s", name)] = true
	}

	for _, moved := range new.MovedBlocks {
		// Check if "to" target exists
		toIsResource := types.IsResourceAddress(moved.To)
		toIsModule := types.IsModuleAddress(moved.To)

		if toIsResource {
			if !existingResources[moved.To] {
				finding := types.NewFinding(
					r.ID(),
					r.Name(),
					r.DefaultSeverity(),
					fmt.Sprintf("Moved block 'to' target does not exist: %q", moved.To),
				).WithNewLocation(&moved.DeclRange)

				findings = append(findings, finding)
			}
		} else if toIsModule {
			if !existingModules[moved.To] {
				finding := types.NewFinding(
					r.ID(),
					r.Name(),
					r.DefaultSeverity(),
					fmt.Sprintf("Moved block 'to' target does not exist: %q", moved.To),
				).WithNewLocation(&moved.DeclRange)

				findings = append(findings, finding)
			}
		}
	}

	return findings
}
