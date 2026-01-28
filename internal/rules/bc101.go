package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC101 detects when a module is removed without a corresponding moved block
type BC101 struct{}

func init() {
	Register(&BC101{})
}

func (r *BC101) ID() string {
	return "BC101"
}

func (r *BC101) Name() string {
	return "module-removed-no-moved"
}

func (r *BC101) Description() string {
	return "A module was removed without a moved block, which will destroy the module's resources"
}

func (r *BC101) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC101) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	// Build set of moved "from" addresses
	movedFrom := make(map[string]bool)
	for _, moved := range new.MovedBlocks {
		movedFrom[moved.From] = true
	}

	for name, oldModule := range old.Modules {
		// Check if module still exists
		if _, exists := new.Modules[name]; exists {
			continue
		}

		// Check if there's a moved block for this module
		addr := oldModule.Address // e.g., "module.vpc"
		if movedFrom[addr] {
			continue
		}

		finding := types.NewFinding(
			r.ID(),
			r.Name(),
			r.DefaultSeverity(),
			fmt.Sprintf("Module %q removed without moved block", addr),
		).WithOldLocation(&oldModule.DeclRange)

		findings = append(findings, finding)
	}

	return findings
}
