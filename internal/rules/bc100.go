package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC100 detects when a resource is removed without a corresponding moved block
type BC100 struct{}

func init() {
	Register(&BC100{})
}

func (r *BC100) ID() string {
	return "BC100"
}

func (r *BC100) Name() string {
	return "resource-removed-no-moved"
}

func (r *BC100) Description() string {
	return "A resource was removed without a moved block, which will destroy the resource"
}

func (r *BC100) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC100) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	// Build set of moved "from" addresses
	movedFrom := make(map[string]bool)
	for _, moved := range new.MovedBlocks {
		movedFrom[moved.From] = true
	}

	for addr, oldResource := range old.Resources {
		// Check if resource still exists
		if _, exists := new.Resources[addr]; exists {
			continue
		}

		// Check if there's a moved block for this resource
		if movedFrom[addr] {
			continue
		}

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Resource %q removed without moved block", addr),
		).WithOldLocation(&oldResource.DeclRange)

		findings = append(findings, finding)
	}

	return findings
}
